package handlers

import (
	"net/http"
	"strconv"

	"notorious-backend/internal/models"
	"notorious-backend/internal/repository"
	chatws "notorious-backend/internal/websocket"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type ChatHandler struct {
	chatRepo *repository.ChatRepository
	hub      *chatws.Hub
}

func NewChatHandler(chatRepo *repository.ChatRepository, hub *chatws.Hub) *ChatHandler {
	return &ChatHandler{
		chatRepo: chatRepo,
		hub:      hub,
	}
}

// GetConversations returns all conversations for the authenticated user.
func (h *ChatHandler) GetConversations(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)

	conversations, err := h.chatRepo.GetConversations(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get conversations"})
		return
	}

	// Enrich with online status
	if conversations != nil {
		for i := range conversations {
			conversations[i].IsOnline = h.hub.IsOnline(conversations[i].OtherUserID)
		}
	}

	if conversations == nil {
		conversations = make([]models.ConversationWithDetails, 0)
	}

	c.JSON(http.StatusOK, conversations)
}

// GetMessages returns paginated message history with a specific user.
func (h *ChatHandler) GetMessages(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)

	otherUserIDStr := c.Param("userId")
	otherUserID, err := uuid.Parse(otherUserIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user ID"})
		return
	}

	// Parse pagination params
	var beforeSeq int64
	if s := c.Query("before_seq"); s != "" {
		beforeSeq, _ = strconv.ParseInt(s, 10, 64)
	}
	limit := 50
	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}

	// Find the conversation
	conv, err := h.chatRepo.GetConversationByUsers(c.Request.Context(), userID, otherUserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to find conversation"})
		return
	}
	if conv == nil {
		c.JSON(http.StatusOK, gin.H{"messages": []interface{}{}, "has_more": false})
		return
	}

	messages, err := h.chatRepo.GetMessages(c.Request.Context(), conv.ID, beforeSeq, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get messages"})
		return
	}

	if messages == nil {
		messages = make([]models.ChatMessage, 0)
	}

	c.JSON(http.StatusOK, gin.H{
		"messages": messages,
		"has_more": len(messages) == limit,
	})
}

// GetUnreadCount returns the total unread count for the authenticated user.
func (h *ChatHandler) GetUnreadCount(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)

	dmCount, broadcastCount, err := h.chatRepo.GetUnreadCount(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get unread count"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"dm_unread":        dmCount,
		"broadcast_unread": broadcastCount,
		"total_unread":     dmCount + broadcastCount,
	})
}

// MarkAsRead marks all messages from a specific user as read.
func (h *ChatHandler) MarkAsRead(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)

	senderIDStr := c.Param("userId")
	senderID, err := uuid.Parse(senderIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user ID"})
		return
	}

	affected, err := h.chatRepo.MarkAsRead(c.Request.Context(), userID, senderID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to mark as read"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"marked_read": affected})
}

// GetBroadcasts returns broadcast messages with read status.
func (h *ChatHandler) GetBroadcasts(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)

	limit := 50
	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}

	broadcasts, err := h.chatRepo.GetBroadcasts(c.Request.Context(), userID, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get broadcasts"})
		return
	}

	if broadcasts == nil {
		broadcasts = make([]models.ChatBroadcastWithStatus, 0)
	}

	c.JSON(http.StatusOK, broadcasts)
}

// MarkBroadcastRead marks a broadcast as read for the authenticated user.
func (h *ChatHandler) MarkBroadcastRead(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)

	broadcastIDStr := c.Param("broadcastId")
	broadcastID, err := uuid.Parse(broadcastIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid broadcast ID"})
		return
	}

	if err := h.chatRepo.MarkBroadcastRead(c.Request.Context(), broadcastID, userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to mark broadcast as read"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// GetOnlineUsers returns list of currently online users via WebSocket.
func (h *ChatHandler) GetOnlineUsers(c *gin.Context) {
	onlineIDs := h.hub.GetOnlineUserIDs()
	stringIDs := make([]string, len(onlineIDs))
	for i, id := range onlineIDs {
		stringIDs[i] = id.String()
	}
	c.JSON(http.StatusOK, gin.H{"online_user_ids": stringIDs})
}

// GetUnreadPerUser returns unread counts per sender for the authenticated user.
func (h *ChatHandler) GetUnreadPerUser(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)

	counts, err := h.chatRepo.GetUnreadCountPerUser(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get unread counts"})
		return
	}

	// Convert to string keys for JSON
	result := make(map[string]int, len(counts))
	for id, count := range counts {
		result[id.String()] = count
	}

	c.JSON(http.StatusOK, gin.H{"unread_per_user": result})
}
