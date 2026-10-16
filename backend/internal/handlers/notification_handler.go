package handlers

import (
	"errors"
	"net/http"
	"strconv"

	"notorious-backend/internal/repository"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type NotificationHandler struct {
	notifRepo *repository.NotificationRepository
	userRepo  *repository.UserRepository
}

func NewNotificationHandler(notifRepo *repository.NotificationRepository, userRepo *repository.UserRepository) *NotificationHandler {
	return &NotificationHandler{
		notifRepo: notifRepo,
		userRepo:  userRepo,
	}
}

// List returns the user's notifications, newest first.
// Query params: ?limit=50&offset=0&unread=true
func (h *NotificationHandler) List(c *gin.Context) {
	userID, ok := userIDFromContext(c)
	if !ok {
		return
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	if limit < 1 || limit > 100 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	unreadOnly := c.Query("unread") == "true"

	notifications, total, err := h.notifRepo.List(c.Request.Context(), userID, limit, offset, unreadOnly)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch notifications"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"notifications": notifications,
		"total":         total,
		"limit":         limit,
		"offset":        offset,
	})
}

// MarkRead marks a single notification read for the authenticated user.
func (h *NotificationHandler) MarkRead(c *gin.Context) {
	userID, ok := userIDFromContext(c)
	if !ok {
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid notification id"})
		return
	}

	affected, err := h.notifRepo.MarkRead(c.Request.Context(), id, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to mark notification read"})
		return
	}
	if affected == 0 {
		// Rows affected 0: not found, not owned, or already read (idempotent
		// success). Only report not-found when the row genuinely does not
		// exist for this user — transient DB errors surface as 500.
		if _, getErr := h.notifRepo.GetByID(c.Request.Context(), id, userID); getErr != nil {
			if errors.Is(getErr, pgx.ErrNoRows) {
				c.JSON(http.StatusNotFound, gin.H{"error": "notification not found"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to verify notification"})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// MarkAllRead marks every notification of the user read.
func (h *NotificationHandler) MarkAllRead(c *gin.Context) {
	userID, ok := userIDFromContext(c)
	if !ok {
		return
	}
	if err := h.notifRepo.MarkAllRead(c.Request.Context(), userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to mark notifications read"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// UnreadCount returns the number of unread notifications.
func (h *NotificationHandler) UnreadCount(c *gin.Context) {
	userID, ok := userIDFromContext(c)
	if !ok {
		return
	}
	count, err := h.notifRepo.CountUnread(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to count unread notifications"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"unread": count})
}

// GetPrefs returns the user's notification preferences.
func (h *NotificationHandler) GetPrefs(c *gin.Context) {
	userID, ok := userIDFromContext(c)
	if !ok {
		return
	}
	prefs, err := h.userRepo.GetNotificationPrefs(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load notification preferences"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"prefs": prefs})
}

// UpdatePrefs merges a partial preference update, e.g. {"quota_reset": true}.
func (h *NotificationHandler) UpdatePrefs(c *gin.Context) {
	userID, ok := userIDFromContext(c)
	if !ok {
		return
	}

	var req struct {
		Prefs map[string]bool `json:"prefs" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	prefs, err := h.userRepo.UpdateNotificationPrefs(c.Request.Context(), userID, req.Prefs)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update notification preferences"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"prefs": prefs})
}

// userIDFromContext extracts the authenticated user id or writes the error response.
func userIDFromContext(c *gin.Context) (uuid.UUID, bool) {
	raw, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return uuid.UUID{}, false
	}
	uid, ok := raw.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return uuid.UUID{}, false
	}
	return uid, true
}
