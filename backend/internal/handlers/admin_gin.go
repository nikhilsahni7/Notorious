package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

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
}

func NewAdminGinHandler(
	userRepo *repository.UserRepository,
	userRequestRepo *repository.UserRequestRepository,
	searchHistoryRepo *repository.SearchHistoryRepository,
	passwordChangeRepo *repository.PasswordChangeRepository,
	metadataRepo *repository.MetadataRepository,
	adminSessionRepo *repository.AdminSessionRepository,
) *AdminGinHandler {
	return &AdminGinHandler{
		userRepo:           userRepo,
		userRequestRepo:    userRequestRepo,
		searchHistoryRepo:  searchHistoryRepo,
		passwordChangeRepo: passwordChangeRepo,
		metadataRepo:       metadataRepo,
		adminSessionRepo:   adminSessionRepo,
	}
}

func (h *AdminGinHandler) CreateUser(c *gin.Context) {
	var req struct {
		Email            string `json:"email" binding:"required,email"`
		Password         string `json:"password" binding:"required,min=6"`
		Name             string `json:"name" binding:"required"`
		Phone            string `json:"phone"`
		Region           string `json:"region"` // "pan-india" or "delhi-ncr"
		DailySearchLimit int    `json:"daily_search_limit" binding:"required,min=1"`
		IsActive         bool   `json:"is_active"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
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
		IsActive:         req.IsActive,
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

	usersWithStats := make([]UserWithStats, len(users))
	for i, user := range users {
		totalSearches, _ := h.searchHistoryRepo.CountByUserID(c.Request.Context(), user.ID)
		usersWithStats[i] = UserWithStats{
			User:          user,
			TotalSearches: totalSearches,
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
		IsActive         bool   `json:"is_active"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
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

func (h *AdminGinHandler) ListUserRequests(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	status := c.DefaultQuery("status", "pending")

	if limit > 100 {
		limit = 100
	}

	requests, err := h.userRequestRepo.ListByStatus(c.Request.Context(), status, limit, offset)
	if err != nil {
		println("Error fetching requests:", err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch requests", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, requests)
}

func (h *AdminGinHandler) ApproveUserRequest(c *gin.Context) {
	requestID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request ID"})
		return
	}

	var req struct {
		AdminNote string `json:"admin_note"` // Optional note explaining approval
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get admin user ID from context
	adminID, _ := c.Get("user_id")
	adminUUID := adminID.(uuid.UUID)

	userRequest, err := h.userRequestRepo.GetByID(c.Request.Context(), requestID)
	if err != nil || userRequest == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "request not found"})
		return
	}

	if userRequest.Status != "pending" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "request is not pending"})
		return
	}

	// Update request with admin note and reviewer
	adminNote := req.AdminNote
	if adminNote == "" {
		adminNote = "Request approved - awaiting user creation"
	}
	now := time.Now()

	if err := h.userRequestRepo.UpdateStatus(c.Request.Context(), requestID, "approved", &adminNote, &adminUUID, &now); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update request status"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Request approved successfully",
		"request": gin.H{
			"id":          userRequest.ID,
			"email":       userRequest.Email,
			"name":        userRequest.Name,
			"status":      "approved",
			"admin_note":  adminNote,
			"reviewed_by": adminUUID,
			"reviewed_at": now,
		},
	})
}

func (h *AdminGinHandler) RejectUserRequest(c *gin.Context) {
	requestID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request ID"})
		return
	}

	var req struct {
		Reason string `json:"reason" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Rejection reason is required"})
		return
	}

	// Get admin user ID from context
	adminID, _ := c.Get("user_id")
	adminUUID := adminID.(uuid.UUID)

	// Update with admin note and reviewer
	now := time.Now()
	userRequest, _ := h.userRequestRepo.GetByID(c.Request.Context(), requestID)
	if userRequest != nil {
		userRequest.AdminNote = &req.Reason
		userRequest.ReviewedBy = &adminUUID
		userRequest.ReviewedAt = &now
	}

	if err := h.userRequestRepo.UpdateStatus(c.Request.Context(), requestID, "rejected", &req.Reason, &adminUUID, &now); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update request status"})
		return
	}

	c.JSON(http.StatusNoContent, nil)
}

func (h *AdminGinHandler) GetSearchHistory(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	if limit > 100 {
		limit = 100
	}

	histories, err := h.searchHistoryRepo.GetAllWithUsers(c.Request.Context(), limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch search history"})
		return
	}

	c.JSON(http.StatusOK, histories)
}

func (h *AdminGinHandler) GetUserSearchHistory(c *gin.Context) {
	userID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user ID"})
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	if limit > 100 {
		limit = 100
	}

	histories, err := h.searchHistoryRepo.GetByUserID(c.Request.Context(), userID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch search history"})
		return
	}

	c.JSON(http.StatusOK, histories)
}

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

	c.JSON(http.StatusOK, gin.H{
		"user":     user,
		"metadata": metadata,
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

// GetRequestCounts returns counts of pending requests for admin dashboard
func (h *AdminGinHandler) GetRequestCounts(c *gin.Context) {
	ctx := c.Request.Context()

	// Count pending user requests
	userRequests, _ := h.userRequestRepo.ListByStatus(ctx, "pending", 1000, 0)

	// Count pending password change requests
	passwordRequests, _ := h.passwordChangeRepo.ListByStatus(ctx, "pending", 1000, 0)

	c.JSON(http.StatusOK, gin.H{
		"pending_user_requests":     len(userRequests),
		"pending_password_requests": len(passwordRequests),
	})
}

// GenerateUserEOD generates End of Day report for a specific user
func (h *AdminGinHandler) GenerateUserEOD(c *gin.Context) {
	userID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user ID"})
		return
	}

	// Get user details
	user, err := h.userRepo.GetByID(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	// Get today's search history for this user
	todaySearches, err := h.searchHistoryRepo.GetTodaySearches(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch search history"})
		return
	}

	// Filter searches for this user only
	userSearches := make([]*models.SearchHistory, 0)
	for _, search := range todaySearches {
		if search.UserID == userID {
			userSearches = append(userSearches, search)
		}
	}

	// Set headers for file download
	filename := user.Name + "_EOD_" + time.Now().Format("2006-01-02") + ".csv"
	c.Header("Content-Description", "File Transfer")
	c.Header("Content-Disposition", "attachment; filename="+filename)
	c.Header("Content-Type", "text/csv")

	// Write CSV header row
	c.Writer.Write([]byte("Search ID,Timestamp,Total Results,OID,Name,Father Name,Mobile,Alt Phone,Email,Address,Alt Address,Year of Registration\n"))

	// Helper function to escape CSV values
	escapeCSV := func(value string) string {
		if value == "" {
			return ""
		}
		// Replace ! with comma for addresses
		value = strings.ReplaceAll(value, "!", ",")

		// If contains special chars, wrap in quotes
		needsQuotes := false
		for _, ch := range value {
			if ch == ',' || ch == '"' || ch == '\n' || ch == '\r' {
				needsQuotes = true
				break
			}
		}

		if needsQuotes {
			// Escape existing quotes by doubling them
			escaped := ""
			for _, ch := range value {
				if ch == '"' {
					escaped += "\"\""
				} else {
					escaped += string(ch)
				}
			}
			return "\"" + escaped + "\""
		}
		return value
	}

	// Helper to safely get string values from result map
	getStringValue := func(result map[string]interface{}, key string) string {
		if val, ok := result[key]; ok && val != nil {
			return fmt.Sprintf("%v", val)
		}
		return ""
	}

	// Process each search history record
	for searchID, history := range userSearches {
		// Parse top results
		topResults, ok := history.TopResults.([]interface{})
		if !ok {
			continue
		}

		// Format timestamp
		timestamp := history.SearchedAt.Format("2006-01-02 15:04:05")
		totalResults := history.TotalResults

		// Limit to top 25 results
		maxResults := len(topResults)
		if maxResults > 25 {
			maxResults = 25
		}

		// Write each result as a CSV row
		for resultNum := 0; resultNum < maxResults; resultNum++ {
			result, ok := topResults[resultNum].(map[string]interface{})
			if !ok {
				continue
			}

			// Build CSV row
			row := fmt.Sprintf("%d,%s,%d,%s,%s,%s,%s,%s,%s,%s,%s,%s\n",
				searchID+1,                               // Search ID (1-indexed)
				timestamp,                                // Timestamp
				totalResults,                             // Total Results
				escapeCSV(getStringValue(result, "oid")), // OID
				escapeCSV(getStringValue(result, "name")),                 // Name
				escapeCSV(getStringValue(result, "fname")),                // Father Name
				escapeCSV(getStringValue(result, "mobile")),               // Mobile
				escapeCSV(getStringValue(result, "alt")),                  // Alt Phone
				escapeCSV(getStringValue(result, "email")),                // Email
				escapeCSV(getStringValue(result, "address")),              // Address
				escapeCSV(getStringValue(result, "alt_address")),          // Alt Address
				escapeCSV(getStringValue(result, "year_of_registration")), // Year of Registration
			)

			c.Writer.Write([]byte(row))
		}
	}
}
