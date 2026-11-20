package websocket

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"notorious-backend/internal/auth"
	"notorious-backend/internal/models"
	"notorious-backend/internal/repository"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	ws "github.com/gorilla/websocket"
)

var upgrader = ws.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // CORS is handled at the proxy level
	},
}

// Hub manages all active WebSocket connections and routes messages.
type Hub struct {
	mu         sync.RWMutex
	clients    map[uuid.UUID]map[*Client]struct{} // userID -> active connections
	chatRepo   *repository.ChatRepository
	jwtManager *auth.JWTManager
}

// NewHub creates a new Hub instance.
func NewHub(chatRepo *repository.ChatRepository, jwtManager *auth.JWTManager) *Hub {
	return &Hub{
		clients:    make(map[uuid.UUID]map[*Client]struct{}),
		chatRepo:   chatRepo,
		jwtManager: jwtManager,
	}
}

// Run starts the hub's main loop. This should be called in a goroutine.
func (h *Hub) Run() {
	// Periodic cleanup of stale connections (safety net)
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		h.mu.RLock()
		count := len(h.clients)
		h.mu.RUnlock()
		log.Printf("[Hub] Active WebSocket connections: %d", count)
	}
}

// Register adds a connection without displacing other tabs or devices.
func (h *Hub) Register(client *Client) {
	h.mu.Lock()
	first := len(h.clients[client.userID]) == 0
	if first {
		h.clients[client.userID] = make(map[*Client]struct{})
	}
	h.clients[client.userID][client] = struct{}{}
	h.mu.Unlock()
	if first {
		h.broadcastOnlineStatus(client.userID, true)
	}
	go h.deliverPendingMessages(client)
}

func (h *Hub) Unregister(client *Client) {
	h.mu.Lock()
	connections, exists := h.clients[client.userID]
	if exists {
		delete(connections, client)
	}
	offline := exists && len(connections) == 0
	if offline {
		delete(h.clients, client.userID)
	}
	h.mu.Unlock()
	if offline {
		h.broadcastOnlineStatus(client.userID, false)
	}
}

// IsOnline checks if a user has an active WebSocket connection.
func (h *Hub) IsOnline(userID uuid.UUID) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	_, ok := h.clients[userID]
	return ok
}

// GetOnlineUserIDs returns all currently connected user IDs.
func (h *Hub) GetOnlineUserIDs() []uuid.UUID {
	h.mu.RLock()
	defer h.mu.RUnlock()
	ids := make([]uuid.UUID, 0, len(h.clients))
	for id := range h.clients {
		ids = append(ids, id)
	}
	return ids
}

// SendToUser sends a message to a specific user if they're online.
// Returns true if the message was sent, false if user is offline.
func (h *Hub) SendToUser(userID uuid.UUID, envelope *models.WSEnvelope) bool {
	data, err := json.Marshal(envelope)
	if err != nil {
		return false
	}
	h.mu.RLock()
	clients := make([]*Client, 0, len(h.clients[userID]))
	for client := range h.clients[userID] {
		clients = append(clients, client)
	}
	h.mu.RUnlock()
	sent := false
	for _, client := range clients {
		select {
		case client.send <- data:
			sent = true
		default:
			log.Printf("[Hub] Send buffer full for user %s", userID)
		}
	}
	return sent
}

// BroadcastToAll sends a message to all connected clients.
// Snapshots client list under lock, then sends outside lock to avoid blocking.
func (h *Hub) BroadcastToAll(envelope *models.WSEnvelope, excludeUserID *uuid.UUID) {
	data, err := json.Marshal(envelope)
	if err != nil {
		log.Printf("[Hub] Failed to marshal broadcast: %v", err)
		return
	}

	// Snapshot clients under lock
	h.mu.RLock()
	clients := make([]*Client, 0, len(h.clients))
	for userID, connections := range h.clients {
		if excludeUserID != nil && userID == *excludeUserID {
			continue
		}
		for client := range connections {
			clients = append(clients, client)
		}
	}
	h.mu.RUnlock()

	// Send outside lock
	for _, client := range clients {
		select {
		case client.send <- data:
		default:
			log.Printf("[Hub] Broadcast buffer full for user %s", client.userID)
		}
	}
}

// broadcastOnlineStatus notifies all connected clients about a user's status change.
func (h *Hub) broadcastOnlineStatus(userID uuid.UUID, isOnline bool) {
	status := models.WSOnlineStatus{
		UserID:   userID,
		IsOnline: isOnline,
	}
	payload, _ := json.Marshal(status)
	envelope := &models.WSEnvelope{
		Type:    models.WSTypeOnlineStatus,
		Payload: payload,
	}
	h.BroadcastToAll(envelope, &userID) // Don't send to the user themselves
}

// deliverPendingMessages marks sent messages as delivered when a user connects.
func (h *Hub) deliverPendingMessages(client *Client) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Get all conversations for this user
	conversations, err := h.chatRepo.GetConversations(ctx, client.userID)
	if err != nil {
		log.Printf("[Hub] Failed to get conversations for delivery: %v", err)
		return
	}

	for _, conv := range conversations {
		if conv.UnreadCount > 0 {
			affected, err := h.chatRepo.MarkAsDelivered(ctx, client.userID, conv.OtherUserID)
			if err != nil {
				log.Printf("[Hub] Failed to mark as delivered: %v", err)
				continue
			}
			if affected > 0 {
				// Notify the sender that their messages were delivered
				deliveryPayload, _ := json.Marshal(models.WSDeliveryUpdate{
					ReceiverID: client.userID,
				})
				h.SendToUser(conv.OtherUserID, &models.WSEnvelope{
					Type:    models.WSTypeDeliveryUpdate,
					Payload: deliveryPayload,
				})
			}
		}
	}
}

// new
// HandleUpgrade handles the WebSocket upgrade request with JWT authentication.
func (h *Hub) HandleUpgrade(c *gin.Context) {
	// Get token from query parameter (WebSocket can't set headers)
	tokenString := c.Query("token")
	if tokenString == "" {
		// Fallback: check Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader != "" {
			parts := strings.Split(authHeader, " ")
			if len(parts) == 2 && parts[0] == "Bearer" {
				tokenString = parts[1]
			}
		}
	}

	if tokenString == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}

	// Verify JWT
	claims, err := h.jwtManager.Verify(tokenString)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired token"})
		return
	}

	// Upgrade to WebSocket
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("[Hub] Failed to upgrade connection for user %s: %v", claims.UserID, err)
		return
	}

	// Create new client
	client := NewClient(h, conn, claims.UserID, claims.Role)

	// Register and start read/write pumps
	h.Register(client)
	go client.WritePump()
	go client.ReadPump()
}
