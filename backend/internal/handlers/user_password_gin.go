package handlers

import (
	"log"
	"net/http"
	"strconv"

	"notorious-backend/internal/auth"
	"notorious-backend/internal/models"
	"notorious-backend/internal/repository"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type UserPasswordGinHandler struct {
	passwordChangeRepo *repository.PasswordChangeRepository
	userRepo           *repository.UserRepository
}

func NewUserPasswordGinHandler(
	passwordChangeRepo *repository.PasswordChangeRepository,
	userRepo *repository.UserRepository,
) *UserPasswordGinHandler {
	return &UserPasswordGinHandler{
		passwordChangeRepo: passwordChangeRepo,
		userRepo:           userRepo,
	}
}

func (h *UserPasswordGinHandler) RequestPasswordChange(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}

	var req struct {
		Reason string `json:"reason" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	passwordRequest := &models.PasswordChangeRequest{
		UserID: userID.(uuid.UUID),
		Reason: req.Reason,
		Status: "pending",
	}

	if err := h.passwordChangeRepo.Create(c.Request.Context(), passwordRequest); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create request"})
		return
	}

	c.JSON(http.StatusCreated, passwordRequest)
}

func (h *UserPasswordGinHandler) GetPasswordChangeRequests(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	if limit > 100 {
		limit = 100
	}

	requests, err := h.passwordChangeRepo.GetByUserID(c.Request.Context(), userID.(uuid.UUID), limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch requests"})
		return
	}

	c.JSON(http.StatusOK, requests)
}

func (h *UserPasswordGinHandler) ChangePassword(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}
	uid := userID.(uuid.UUID)

	var req struct {
		CurrentPassword string `json:"current_password" binding:"required"`
		NewPassword     string `json:"new_password" binding:"required,min=6"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 1. Get user details to verify current password
	user, err := h.userRepo.GetByID(c.Request.Context(), uid)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	// 2. Check current password
	if err := auth.CheckPassword(user.PasswordHash, req.CurrentPassword); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "incorrect current password"})
		return
	}

	// 3. Hash new password
	newHash, err := auth.HashPassword(req.NewPassword)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to hash new password"})
		return
	}

	// 4. Update password in DB
	if err := h.userRepo.UpdatePassword(c.Request.Context(), uid, newHash); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update password"})
		return
	}

	// 5. Log the self-service change by creating an approved request record
	adminNotes := "Changed by user directly"
	passwordRequest := &models.PasswordChangeRequest{
		UserID:          uid,
		Reason:          "Self-service password change",
		Status:          "approved",
		AdminNotes:      &adminNotes,
		NewPasswordHash: &newHash,
		ProcessedBy:     &uid,
	}

	if err := h.passwordChangeRepo.Create(c.Request.Context(), passwordRequest); err != nil {
		// Log the error but don't fail the password change response since DB update succeeded
		log.Printf("ERROR: failed to log self-service password change: %v", err)
	}

	c.JSON(http.StatusOK, gin.H{"message": "password updated successfully"})
}

