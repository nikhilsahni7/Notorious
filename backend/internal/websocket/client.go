package websocket

import (
	"context"
	"encoding/json"
	"log"
	"sync"
	"time"

	"notorious-backend/internal/models"

	"github.com/google/uuid"
	ws "github.com/gorilla/websocket"
)

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = 54 * time.Second

	// Maximum message size allowed from peer (8KB).
	maxMessageSize = 8192

	// Size of the send channel buffer.
	sendBufferSize = 256
)

// Client represents a single WebSocket connection.
type Client struct {
	hub       *Hub
	conn      *ws.Conn
	userID    uuid.UUID
	userRole  string
	send      chan []byte
	done      chan struct{}
	closeOnce sync.Once
}

// NewClient creates a new Client instance.
func NewClient(hub *Hub, conn *ws.Conn, userID uuid.UUID, userRole string) *Client {
	return &Client{
		hub:      hub,
		conn:     conn,
		userID:   userID,
		userRole: userRole,
		send:     make(chan []byte, sendBufferSize),
		done:     make(chan struct{}),
	}
}

// ReadPump reads messages from the WebSocket connection.
// Runs in its own goroutine for each connection.
func (c *Client) ReadPump() {
	defer func() {
		c.hub.Unregister(c)
		c.closeOnce.Do(func() {
			close(c.done)
			c.conn.Close()
		})
	}()

	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if ws.IsUnexpectedCloseError(err, ws.CloseGoingAway, ws.CloseNormalClosure) {
				log.Printf("[Client] Read error for user %s: %v", c.userID, err)
			}
			return
		}

		c.handleMessage(message)
	}
}

// WritePump writes messages to the WebSocket connection.
// Runs in its own goroutine for each connection.
func (c *Client) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.closeOnce.Do(func() {
			close(c.done)
			c.conn.Close()
		})
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// Channel closed
				c.conn.WriteMessage(ws.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(ws.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// Drain queued messages into the same write for efficiency
			n := len(c.send)
			for i := 0; i < n; i++ {
				w.Write([]byte("\n"))
				w.Write(<-c.send)
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(ws.PingMessage, nil); err != nil {
				return
			}

		case <-c.done:
			return
		}
	}
}

// handleMessage routes incoming WebSocket messages to the appropriate handler.
func (c *Client) handleMessage(data []byte) {
	var envelope models.WSEnvelope
	if err := json.Unmarshal(data, &envelope); err != nil {
		c.sendError("invalid message format", "PARSE_ERROR")
		return
	}

	switch envelope.Type {
	case models.WSTypeChatMessage:
		c.handleChatMessage(envelope.Payload)
	case models.WSTypeTypingStart, models.WSTypeTypingStop:
		c.handleTyping(envelope.Type, envelope.Payload)
	case models.WSTypeReadReceipt:
		c.handleReadReceipt(envelope.Payload)
	case models.WSTypeBroadcast:
		c.handleBroadcast(envelope.Payload)
	case models.WSTypeBroadcastRead:
		c.handleBroadcastRead(envelope.Payload)
	default:
		c.sendError("unknown message type: "+envelope.Type, "UNKNOWN_TYPE")
	}
}

// handleChatMessage processes an incoming chat message.
func (c *Client) handleChatMessage(payload json.RawMessage) {
	var msg models.WSChatMessage
	if err := json.Unmarshal(payload, &msg); err != nil {
		c.sendError("invalid chat message payload", "PARSE_ERROR")
		return
	}

	// Validate
	if msg.Content == "" {
		c.sendError("message content cannot be empty", "VALIDATION_ERROR")
		return
	}
	if len(msg.Content) > 4000 {
		c.sendError("message too long (max 4000 characters)", "VALIDATION_ERROR")
		return
	}

	receiverID, err := uuid.Parse(msg.ReceiverID)
	if err != nil {
		c.sendError("invalid receiver_id", "VALIDATION_ERROR")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Get or create conversation
	conv, err := c.hub.chatRepo.GetOrCreateConversation(ctx, c.userID, receiverID)
	if err != nil {
		log.Printf("[Client] Failed to get/create conversation: %v", err)
		c.sendError("failed to create conversation", "SERVER_ERROR")
		return
	}

	// Save message
	savedMsg, err := c.hub.chatRepo.SaveMessage(ctx, conv.ID, c.userID, receiverID, msg.Content)
	if err != nil {
		log.Printf("[Client] Failed to save message: %v", err)
		c.sendError("failed to save message", "SERVER_ERROR")
		return
	}

	// Send ACK to sender
	ack := models.WSMessageAck{
		TempID:      msg.TempID,
		MessageID:   savedMsg.ID,
		SequenceNum: savedMsg.SequenceNum,
		SentAt:      savedMsg.SentAt,
	}
	ackPayload, _ := json.Marshal(ack)
	c.sendEnvelope(models.WSTypeMessageAck, ackPayload)

	// Deliver to receiver if online
	newMsgPayload, _ := json.Marshal(models.WSNewMessage{Message: *savedMsg})
	delivered := c.hub.SendToUser(receiverID, &models.WSEnvelope{
		Type:    models.WSTypeNewMessage,
		Payload: newMsgPayload,
	})

	// If delivered, update status
	if delivered {
		go func() {
			dCtx, dCancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer dCancel()
			c.hub.chatRepo.MarkAsDelivered(dCtx, receiverID, c.userID)

			// Notify sender about delivery
			deliveryPayload, _ := json.Marshal(models.WSDeliveryUpdate{
				ReceiverID: receiverID,
			})
			c.sendEnvelope(models.WSTypeDeliveryUpdate, deliveryPayload)
		}()
	}
}

// handleTyping forwards typing indicators to the other user.
func (c *Client) handleTyping(eventType string, payload json.RawMessage) {
	var indicator models.WSTypingIndicator
	if err := json.Unmarshal(payload, &indicator); err != nil {
		return // Silently ignore malformed typing
	}

	// Forward to the target user with the sender's ID
	outPayload, _ := json.Marshal(models.WSTypingIndicator{
		UserID: c.userID,
	})

	c.hub.SendToUser(indicator.UserID, &models.WSEnvelope{
		Type:    eventType,
		Payload: outPayload,
	})
}

// handleReadReceipt processes a read receipt from the client.
func (c *Client) handleReadReceipt(payload json.RawMessage) {
	var receipt models.WSReadReceipt
	if err := json.Unmarshal(payload, &receipt); err != nil {
		c.sendError("invalid read receipt payload", "PARSE_ERROR")
		return
	}

	senderID, err := uuid.Parse(receipt.SenderID)
	if err != nil {
		c.sendError("invalid sender_id", "VALIDATION_ERROR")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	affected, err := c.hub.chatRepo.MarkAsRead(ctx, c.userID, senderID)
	if err != nil {
		log.Printf("[Client] Failed to mark as read: %v", err)
		return
	}

	if affected > 0 {
		// Notify the sender their messages were read
		readUpdate, _ := json.Marshal(models.WSReadUpdate{
			ReaderID: c.userID,
			ReadAt:   time.Now(),
		})
		c.hub.SendToUser(senderID, &models.WSEnvelope{
			Type:    models.WSTypeReadUpdate,
			Payload: readUpdate,
		})
	}
}

// handleBroadcast processes a broadcast message from admin.
func (c *Client) handleBroadcast(payload json.RawMessage) {
	// Only admins can broadcast
	if c.userRole != "admin" {
		c.sendError("only admins can send broadcasts", "FORBIDDEN")
		return
	}

	var msg models.WSBroadcastMessage
	if err := json.Unmarshal(payload, &msg); err != nil {
		c.sendError("invalid broadcast payload", "PARSE_ERROR")
		return
	}

	if msg.Content == "" {
		c.sendError("broadcast content cannot be empty", "VALIDATION_ERROR")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	broadcast, err := c.hub.chatRepo.SaveBroadcast(ctx, c.userID, msg.Content)
	if err != nil {
		log.Printf("[Client] Failed to save broadcast: %v", err)
		c.sendError("failed to save broadcast", "SERVER_ERROR")
		return
	}

	// Send to all connected users (except sender)
	notification := models.WSBroadcastNotification{
		Broadcast: models.ChatBroadcastWithStatus{
			ID:         broadcast.ID,
			SenderID:   broadcast.SenderID,
			SenderName: "Admin", // We know it's admin
			Content:    broadcast.Content,
			SentAt:     broadcast.SentAt,
			IsRead:     false,
		},
	}
	notifPayload, _ := json.Marshal(notification)
	c.hub.BroadcastToAll(&models.WSEnvelope{
		Type:    models.WSTypeBroadcast,
		Payload: notifPayload,
	}, &c.userID)

	// ACK to sender
	ackPayload, _ := json.Marshal(map[string]interface{}{
		"broadcast_id": broadcast.ID,
		"sent_at":      broadcast.SentAt,
	})
	c.sendEnvelope(models.WSTypeMessageAck, ackPayload)
}

// handleBroadcastRead processes a broadcast read receipt.
func (c *Client) handleBroadcastRead(payload json.RawMessage) {
	var receipt models.WSBroadcastReadReceipt
	if err := json.Unmarshal(payload, &receipt); err != nil {
		c.sendError("invalid broadcast read payload", "PARSE_ERROR")
		return
	}

	broadcastID, err := uuid.Parse(receipt.BroadcastID)
	if err != nil {
		c.sendError("invalid broadcast_id", "VALIDATION_ERROR")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := c.hub.chatRepo.MarkBroadcastRead(ctx, broadcastID, c.userID); err != nil {
		log.Printf("[Client] Failed to mark broadcast as read: %v", err)
	}
}

// sendEnvelope sends a typed WebSocket message to this client.
func (c *Client) sendEnvelope(msgType string, payload json.RawMessage) {
	envelope := models.WSEnvelope{
		Type:    msgType,
		Payload: payload,
	}
	data, err := json.Marshal(envelope)
	if err != nil {
		log.Printf("[Client] Failed to marshal envelope: %v", err)
		return
	}

	select {
	case c.send <- data:
	default:
		log.Printf("[Client] Send buffer full for user %s", c.userID)
	}
}

// sendError sends an error message to this client.
func (c *Client) sendError(message, code string) {
	errPayload, _ := json.Marshal(models.WSError{
		Message: message,
		Code:    code,
	})
	c.sendEnvelope(models.WSTypeError, errPayload)
}
