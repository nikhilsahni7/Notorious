package handlers

// admin_gin.go — AdminGinHandler struct, constructor, and user CRUD operations.
//
// Related admin methods live in:
//   admin_requests.go  — access requests, search history, dashboard, online users, EOD
//   admin_passwords.go — password changes and password change requests
//   admin_sessions.go  — session management and user details

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"notorious-backend/internal/auth"
	"notorious-backend/internal/models"
	"notorious-backend/internal/repository"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type AdminGinHandler struct {
	userRepo           *repository.UserRepository
	userRequestRepo    *repository.UserRequestRepository
	searchHistoryRepo  *repository.SearchHistoryRepository
	passwordChangeRepo *repository.PasswordChangeRepository
	metadataRepo       *repository.MetadataRepository
	adminSessionRepo   *repository.AdminSessionRepository
	sessionRepo        *repository.SessionRepository
}

func NewAdminGinHandler(
	userRepo *repository.UserRepository,
	userRequestRepo *repository.UserRequestRepository,
	searchHistoryRepo *repository.SearchHistoryRepository,
	passwordChangeRepo *repository.PasswordChangeRepository,
	metadataRepo *repository.MetadataRepository,
	adminSessionRepo *repository.AdminSessionRepository,
	sessionRepo *repository.SessionRepository,
) *AdminGinHandler {
	return &AdminGinHandler{
		userRepo:           userRepo,
		userRequestRepo:    userRequestRepo,
		searchHistoryRepo:  searchHistoryRepo,
		passwordChangeRepo: passwordChangeRepo,
		metadataRepo:       metadataRepo,
		adminSessionRepo:   adminSessionRepo,
		sessionRepo:        sessionRepo,
	}
}

func (h *AdminGinHandler) CreateUser(c *gin.Context) {
	// Maximum allowed daily search limit (security measure)
	const maxDailySearchLimit = 10000

	var req struct {
		Email            string `json:"email" binding:"required,email"`
		Password         string `json:"password" binding:"required,min=6"`
		Name             string `json:"name" binding:"required"`
		Phone            string `json:"phone"`
		Region           string `json:"region"` // "pan-india" or "delhi-ncr"
		DailySearchLimit int    `json:"daily_search_limit" binding:"required,min=1"`
		DeviceLimit      int    `json:"device_limit"`
		IsActive         bool   `json:"is_active"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	req.Phone = strings.TrimSpace(req.Phone)

	// SECURITY: Validate daily search limit has a maximum
	if req.DailySearchLimit > maxDailySearchLimit {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":     fmt.Sprintf("daily_search_limit cannot exceed %d", maxDailySearchLimit),
			"max_limit": maxDailySearchLimit,
		})
		return
	}

	// Validate region
	if req.Region == "" {
		req.Region = "pan-india" // Default to pan-india
	}
	if req.Region != "pan-india" && req.Region != "delhi-ncr" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "region must be either 'pan-india' or 'delhi-ncr'"})
		return
	}

	passwordHash, err := auth.HashPassword(req.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to hash password"})
		return
	}

	user := &models.User{
		Email:            req.Email,
		PasswordHash:     passwordHash,
		Name:             req.Name,
		Phone:            req.Phone,
		Role:             models.RoleUser,
		Region:           req.Region,
		DailySearchLimit: req.DailySearchLimit,
		DeviceLimit:      req.DeviceLimit,
		IsActive:         req.IsActive,
	}

	if user.DeviceLimit == 0 {
		user.DeviceLimit = 1
	}

	if err := h.userRepo.Create(c.Request.Context(), user); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create user"})
		return
	}

	c.JSON(http.StatusCreated, user)
}

func (h *AdminGinHandler) ListUsers(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	role := c.Query("role")

	if limit > 100 {
		limit = 100
	}

	users, err := h.userRepo.List(c.Request.Context(), role, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch users"})
		return
	}

	// Enrich users with total search count
	type UserWithStats struct {
		*models.User
		TotalSearches int `json:"total_searches"`
	}

	if len(users) == 0 {
		c.JSON(http.StatusOK, []UserWithStats{})
		return
	}

	userIDs := make([]uuid.UUID, len(users))
	for i, user := range users {
		userIDs[i] = user.ID
	}

	searchCounts, err := h.searchHistoryRepo.CountByUserIDs(c.Request.Context(), userIDs)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch search counts"})
		return
	}

	usersWithStats := make([]UserWithStats, len(users))
	for i, user := range users {
		usersWithStats[i] = UserWithStats{
			User:          user,
			TotalSearches: searchCounts[user.ID],
		}
	}

	c.JSON(http.StatusOK, usersWithStats)
}

func (h *AdminGinHandler) GetUser(c *gin.Context) {
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

	c.JSON(http.StatusOK, user)
}

func (h *AdminGinHandler) UpdateUser(c *gin.Context) {
	// Maximum allowed daily search limit (security measure)
	const maxDailySearchLimit = 10000

	userID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user ID"})
		return
	}

	var req struct {
		Name             string `json:"name" binding:"required"`
		Phone            string `json:"phone"`
		Region           string `json:"region"` // "pan-india" or "delhi-ncr"
		DailySearchLimit int    `json:"daily_search_limit" binding:"required,min=1"`
		DeviceLimit      int    `json:"device_limit"`
		IsActive         bool   `json:"is_active"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// SECURITY: Validate daily search limit has a maximum
	if req.DailySearchLimit > maxDailySearchLimit {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":     fmt.Sprintf("daily_search_limit cannot exceed %d", maxDailySearchLimit),
			"max_limit": maxDailySearchLimit,
		})
		return
	}

	// Validate region if provided
	if req.Region != "" && req.Region != "pan-india" && req.Region != "delhi-ncr" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "region must be either 'pan-india' or 'delhi-ncr'"})
		return
	}

	user, err := h.userRepo.GetByID(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	user.Name = req.Name
	user.Phone = req.Phone
	if req.Region != "" {
		user.Region = req.Region
	}
	user.DailySearchLimit = req.DailySearchLimit
	if req.DeviceLimit > 0 {
		user.DeviceLimit = req.DeviceLimit
	}
	user.IsActive = req.IsActive

	if err := h.userRepo.Update(c.Request.Context(), user); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update user"})
		return
	}

	c.JSON(http.StatusOK, user)
}

func (h *AdminGinHandler) DeleteUser(c *gin.Context) {
	userID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user ID"})
		return
	}

	if err := h.userRepo.Delete(c.Request.Context(), userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete user"})
		return
	}

	c.JSON(http.StatusNoContent, nil)
}

// BulkUpdateUsers updates multiple users' active status at once
func (h *AdminGinHandler) BulkUpdateUsers(c *gin.Context) {
	var req struct {
		UserIDs  []string `json:"user_ids" binding:"required,min=1"`
		IsActive bool     `json:"is_active"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Convert string IDs to UUIDs
	userIDs := make([]uuid.UUID, 0, len(req.UserIDs))
	for _, idStr := range req.UserIDs {
		id, err := uuid.Parse(idStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("invalid user ID: %s", idStr)})
			return
		}
		userIDs = append(userIDs, id)
	}

	// Perform bulk update
	updated, err := h.userRepo.BulkUpdateStatus(c.Request.Context(), userIDs, req.IsActive)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update users"})
		return
	}

	action := "deactivated"
	if req.IsActive {
		action = "activated"
	}

	c.JSON(http.StatusOK, gin.H{
		"message": fmt.Sprintf("Successfully %s %d users", action, updated),
		"updated": updated,
	})
}
