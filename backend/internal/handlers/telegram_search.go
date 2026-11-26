package handlers

import (
	"context"
	"crypto/subtle"
	"errors"
	"net/http"
	"net/mail"
	"regexp"
	"strings"
	"time"

	"notorious-backend/internal/services"

	"github.com/gin-gonic/gin"
)

var telegramPhone = regexp.MustCompile(`^[0-9]{10,15}$`)
var telegramMaster = regexp.MustCompile(`^[a-zA-Z0-9_-]{8,64}$`)

func (h *SearchHandler) TelegramSearch(secret string) gin.HandlerFunc {
	return h.telegramSearch(secret, false)
}

// TelegramAdminSearch is mounted only on the loopback-only service listener.
// The bot selects it exclusively for configured administrator Telegram IDs.
func (h *SearchHandler) TelegramAdminSearch(secret string) gin.HandlerFunc {
	return h.telegramSearch(secret, true)
}

func (h *SearchHandler) telegramSearch(secret string, admin bool) gin.HandlerFunc {
	// Restrict expensive searches independently from ordinary web searches.
	slots := make(chan struct{}, 4)
	return func(c *gin.Context) {
		token := strings.TrimPrefix(c.GetHeader("Authorization"), "Bearer ")
		if subtle.ConstantTimeCompare([]byte(token), []byte(secret)) != 1 {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		var req struct {
			Type   string `json:"type"`
			Value  string `json:"value"`
			Region string `json:"region"`
		}
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 4096)
		if c.ShouldBindJSON(&req) != nil {
			c.JSON(400, gin.H{"error": "invalid request"})
			return
		}
		req.Value = strings.TrimSpace(req.Value)
		valid := false
		switch req.Type {
		case "phone", "auto":
			valid = telegramPhone.MatchString(req.Value)
		case "master_id":
			valid = telegramMaster.MatchString(req.Value)
		case "email":
			addr, err := mail.ParseAddress(req.Value)
			valid = err == nil && addr.Address == req.Value && len(req.Value) <= 254
		}
		if !valid {
			c.JSON(400, gin.H{"error": "invalid search value"})
			return
		}
		if !admin {
			select {
			case slots <- struct{}{}:
				defer func() { <-slots }()
			default:
				c.JSON(503, gin.H{"error": "search busy"})
				return
			}
		}
		ctx, cancel := context.WithTimeout(c.Request.Context(), 12*time.Second)
		defer cancel()
		// The bot integration always uses the fixed search scope.
		if req.Region != "pan-india" {
			c.JSON(400, gin.H{"error": "invalid region"})
			return
		}
		region := req.Region
		result, err := h.openSearchService.TelegramSearch(ctx, req.Type, req.Value, region)
		if err != nil {
			code := "upstream_error"
			switch {
			case errors.Is(err, services.ErrTelegramSearchTooManyMatches):
				code = "too_many_matches"
			case errors.Is(err, services.ErrTelegramSearchIncomplete):
				code = "incomplete_search"
			case errors.Is(err, services.ErrTelegramSearchInvalidRecord):
				code = "invalid_record"
			}
			c.JSON(503, gin.H{"error": "search unavailable", "code": code})
			return
		}
		c.JSON(200, result)
	}
}
