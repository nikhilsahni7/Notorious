package handlers

import (
	"net/http"
	"strconv"

	"notorious-backend/internal/repository"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type UserGinHandler struct {
	searchHistoryRepo *repository.SearchHistoryRepository
	metadataRepo      *repository.MetadataRepository
}

func NewUserGinHandler(searchHistoryRepo *repository.SearchHistoryRepository, metadataRepo *repository.MetadataRepository) *UserGinHandler {
	return &UserGinHandler{
		searchHistoryRepo: searchHistoryRepo,
		metadataRepo:      metadataRepo,
	}
}

func (h *UserGinHandler) GetSearchHistory(c *gin.Context) {
	userIDStr, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}

	userID := userIDStr.(uuid.UUID)
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	if limit > 100 {
		limit = 100
	}

	history, err := h.searchHistoryRepo.GetByUserID(c.Request.Context(), userID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch history"})
		return
	}

	c.JSON(http.StatusOK, history)
}

// GetMetadata returns the user's signup metadata (IP, location, device info)
func (h *UserGinHandler) GetMetadata(c *gin.Context) {
	userIDStr, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}

	userID := userIDStr.(uuid.UUID)

	metadata, err := h.metadataRepo.GetUserMetadata(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "metadata not found"})
		return
	}

	c.JSON(http.StatusOK, metadata)
}

