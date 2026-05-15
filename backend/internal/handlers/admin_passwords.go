package handlers

// admin_passwords.go — Password management: admin change, approve/reject requests.

import (
	"net/http"
	"strconv"

	"notorious-backend/internal/auth"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func (h *AdminGinHandler) ChangeUserPassword(c *gin.Context) {
	userID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user ID"})
		return
	}

	var req struct {
		NewPassword string `json:"new_password" binding:"required,min=6"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	passwordHash, err := auth.HashPassword(req.NewPassword)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to hash password"})
		return
	}

	if err := h.userRepo.UpdatePassword(c.Request.Context(), userID, passwordHash); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update password"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "password updated successfully"})
}

func (h *AdminGinHandler) ListPasswordChangeRequests(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	status := c.DefaultQuery("status", "pending")

	if limit > 100 {
		limit = 100
	}

	requests, err := h.passwordChangeRepo.ListByStatus(c.Request.Context(), status, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch requests"})
		return
	}

	c.JSON(http.StatusOK, requests)
}

func (h *AdminGinHandler) ApprovePasswordChangeRequest(c *gin.Context) {
	requestID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request ID"})
		return
	}

	adminID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "admin authentication required"})
		return
	}

	var req struct {
		NewPassword string  `json:"new_password" binding:"required,min=6"`
		AdminNotes  *string `json:"admin_notes"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	passwordRequest, err := h.passwordChangeRepo.GetByID(c.Request.Context(), requestID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "request not found"})
		return
	}

	passwordHash, err := auth.HashPassword(req.NewPassword)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to hash password"})
		return
	}

	// Update user password
	if err := h.userRepo.UpdatePassword(c.Request.Context(), passwordRequest.UserID, passwordHash); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update password"})
		return
	}

	// Update request status
	if err := h.passwordChangeRepo.UpdateStatus(c.Request.Context(), requestID, "approved", req.AdminNotes, &passwordHash, adminID.(uuid.UUID)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update request status"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "password change approved"})
}

func (h *AdminGinHandler) RejectPasswordChangeRequest(c *gin.Context) {
	requestID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request ID"})
		return
	}

	adminID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "admin authentication required"})
		return
	}

	var req struct {
		AdminNotes string `json:"admin_notes"`
	}
	c.ShouldBindJSON(&req)

	var notes *string
	if req.AdminNotes != "" {
		notes = &req.AdminNotes
	}

	if err := h.passwordChangeRepo.UpdateStatus(c.Request.Context(), requestID, "rejected", notes, nil, adminID.(uuid.UUID)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update request status"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "password change rejected"})
}
