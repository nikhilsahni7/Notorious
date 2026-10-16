package handlers

import (
	"net/http"
	"net/url"

	"notorious-backend/internal/models"
	"notorious-backend/internal/repository"
	"notorious-backend/internal/services"

	"github.com/gin-gonic/gin"
)

type PushHandler struct {
	subRepo    *repository.PushSubscriptionRepository
	pushSender *services.PushSender
}

func NewPushHandler(subRepo *repository.PushSubscriptionRepository, pushSender *services.PushSender) *PushHandler {
	return &PushHandler{
		subRepo:    subRepo,
		pushSender: pushSender,
	}
}

// PublicKey returns the VAPID public key for PushManager.subscribe().
// Returns 503 when web push is not configured so the frontend degrades
// silently to in-app only.
func (h *PushHandler) PublicKey(c *gin.Context) {
	if !h.pushSender.Enabled() {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "web push is not configured"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"public_key": h.pushSender.PublicKey()})
}

// Subscribe registers (or refreshes) a browser push endpoint for the user.
// Expected body: {"endpoint": "...", "keys": {"p256dh": "...", "auth": "..."},
// "device_type": "mobile"|"desktop"}.
func (h *PushHandler) Subscribe(c *gin.Context) {
	userID, ok := userIDFromContext(c)
	if !ok {
		return
	}
	if !h.pushSender.Enabled() {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "web push is not configured"})
		return
	}

	var req struct {
		Endpoint string `json:"endpoint"`
		Keys     struct {
			P256dh string `json:"p256dh"`
			Auth   string `json:"auth"`
		} `json:"keys"`
		DeviceType string `json:"device_type"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}
	if _, err := url.ParseRequestURI(req.Endpoint); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid endpoint URL"})
		return
	}
	if req.Keys.P256dh == "" || req.Keys.Auth == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "keys.p256dh and keys.auth are required"})
		return
	}

	sub := &models.PushSubscription{
		UserID:     userID,
		Endpoint:   req.Endpoint,
		P256dh:     req.Keys.P256dh,
		Auth:       req.Keys.Auth,
		DeviceType: req.DeviceType,
	}
	if err := h.subRepo.Upsert(c.Request.Context(), sub); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save push subscription"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// Unsubscribe removes a browser endpoint. Passing subscribe's endpoint back
// avoids needing the sub id client side.
func (h *PushHandler) Unsubscribe(c *gin.Context) {
	userID, ok := userIDFromContext(c)
	if !ok {
		return
	}

	var req struct {
		Endpoint string `json:"endpoint" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	if err := h.subRepo.DeleteByEndpoint(c.Request.Context(), req.Endpoint, &userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to remove push subscription"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
