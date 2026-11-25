package handlers

import (
	"context"
	"crypto/subtle"
	"net/http"
	"net/mail"
	"regexp"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

var telegramPhone = regexp.MustCompile(`^[0-9]{10,15}$`)
var telegramMaster = regexp.MustCompile(`^[a-zA-Z0-9_-]{8,64}$`)

func (h *SearchHandler) TelegramSearch(secret string) gin.HandlerFunc {
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
		case "phone":
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
		select {
		case slots <- struct{}{}:
			defer func() { <-slots }()
		default:
			c.JSON(503, gin.H{"error": "search busy"})
			return
		}
		ctx, cancel := context.WithTimeout(c.Request.Context(), 12*time.Second)
		defer cancel()
		// Only the authenticated bot service may choose the admin-assigned region.
		if req.Region != "pan-india" && req.Region != "delhi-ncr" {
			c.JSON(400, gin.H{"error": "invalid region"})
			return
		}
		region := req.Region
		result, err := h.openSearchService.TelegramSearch(ctx, req.Type, req.Value, region)
		if err != nil {
			c.JSON(503, gin.H{"error": "search unavailable"})
			return
		}
		c.JSON(200, result)
	}
}
