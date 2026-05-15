package handlers

// admin_sessions.go — Session management: admin sessions, user sessions, user details.

import (
	"net/http"
	"strconv"

	"notorious-backend/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// GetUserDetails retrieves user with metadata (IP, location, device info)
func (h *AdminGinHandler) GetUserDetails(c *gin.Context) {
	userID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user ID"})
		return
	}

	user, err := h.userRepo.GetByID(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	var metadata *models.UserMetadata
	if h.metadataRepo != nil {
		metadata, _ = h.metadataRepo.GetUserMetadata(c.Request.Context(), userID)
	}

	var sessions []models.UserSession
	if h.sessionRepo != nil {
		sessions, _ = h.sessionRepo.GetActiveSessions(c.Request.Context(), userID)
	}

	c.JSON(http.StatusOK, gin.H{
		"user":     user,
		"metadata": metadata,
		"sessions": sessions,
	})
}

// GetAdminSessions retrieves all active admin sessions
func (h *AdminGinHandler) GetAdminSessions(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "100"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	if limit > 500 {
		limit = 500
	}

	sessions, err := h.adminSessionRepo.GetActiveSessions(c.Request.Context(), limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch sessions"})
		return
	}

	c.JSON(http.StatusOK, sessions)
}

// InvalidateSession invalidates/deletes an admin session
func (h *AdminGinHandler) InvalidateSession(c *gin.Context) {
	sessionID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid session ID"})
		return
	}

	if err := h.adminSessionRepo.InvalidateSession(c.Request.Context(), sessionID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to invalidate session"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "session invalidated successfully"})
}

// DeleteUserSession deletes a user session (for admin use)
func (h *AdminGinHandler) DeleteUserSession(c *gin.Context) {
	sessionID, err := uuid.Parse(c.Param("sessionId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid session ID"})
		return
	}

	// Admin can delete any user session
	if h.sessionRepo != nil {
		if err := h.sessionRepo.DeleteByID(c.Request.Context(), sessionID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete user session"})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"message": "user session deleted successfully"})
}
