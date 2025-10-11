package handlers

import (
	"net/http"
	"strconv"

	"notorious-backend/internal/models"
	"notorious-backend/internal/repository"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type UserPasswordGinHandler struct {
	passwordChangeRepo *repository.PasswordChangeRepository
}

func NewUserPasswordGinHandler(
	passwordChangeRepo *repository.PasswordChangeRepository,
) *UserPasswordGinHandler {
	return &UserPasswordGinHandler{
		passwordChangeRepo: passwordChangeRepo,
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

