package handlers

// admin_requests.go — Access request approval/rejection workflows.

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"notorious-backend/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

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

// GetDashboardStats returns counts of pending requests and overall stats for admin dashboard
func (h *AdminGinHandler) GetDashboardStats(c *gin.Context) {
	ctx := c.Request.Context()

	// Count pending requests
	userRequests, _ := h.userRequestRepo.ListByStatus(ctx, "pending", 1000, 0)
	passwordRequests, _ := h.passwordChangeRepo.ListByStatus(ctx, "pending", 1000, 0)

	totalUsers, _ := h.userRepo.CountTotal(ctx)
	activeUsers, _ := h.userRepo.CountActive(ctx)
	totalSearches, _ := h.searchHistoryRepo.CountAll(ctx)

	c.JSON(http.StatusOK, gin.H{
		"pending_user_requests":     len(userRequests),
		"pending_password_requests": len(passwordRequests),
		"total_users":               totalUsers,
		"active_users":              activeUsers,
		"total_searches":            totalSearches,
	})
}

// GetOnlineUsers returns list of user IDs currently online (active in last 60 seconds)
func (h *AdminGinHandler) GetOnlineUsers(c *gin.Context) {
	// Users are considered "online" if their last_active is within the last 60 seconds
	const onlineThresholdSeconds = 60

	if h.sessionRepo == nil {
		c.JSON(http.StatusOK, gin.H{"online_user_ids": []string{}})
		return
	}

	userIDs, err := h.sessionRepo.GetOnlineUserIDs(c.Request.Context(), onlineThresholdSeconds)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch online users"})
		return
	}

	// Convert UUIDs to strings
	stringIDs := make([]string, len(userIDs))
	for i, id := range userIDs {
		stringIDs[i] = id.String()
	}

	c.JSON(http.StatusOK, gin.H{"online_user_ids": stringIDs})
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
		if !strings.ContainsAny(value, ",\"\n\r") {
			return value
		}

		// Escape existing quotes by doubling them, then wrap in quotes
		return `"` + strings.ReplaceAll(value, `"`, `""`) + `"`
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
