package handlers

import (
	"fmt"
	"log"
	"net/http"
	"regexp"
	"time"

	"notorious-backend/internal/models"
	"notorious-backend/internal/repository"
	"notorious-backend/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

var mobileRegex = regexp.MustCompile(`^\d{10,}$`)

// isMobileNumber checks if the query string looks like a mobile number
func isMobileNumber(query string) bool {
	return mobileRegex.MatchString(query)
}

// extractMobileNumber extracts mobile number from various query formats
// Handles: "9876543210", "mobile:9876543210", "alt:9876543210"
// Returns (mobileNumber, isMobileSearch)
func extractMobileNumber(query string) (string, bool) {
	query = trimSpace(query)

	// Case 1: Direct mobile number (e.g., "9876543210")
	if isMobileNumber(query) {
		return query, true
	}

	// Case 2: Field syntax with mobile or alt (e.g., "mobile:9876543210" or "alt:9876543210")
	if colonIdx := findChar(query, ':'); colonIdx != -1 {
		field := trimSpace(query[:colonIdx])
		value := trimSpace(query[colonIdx+1:])

		// Check if it's a mobile or alt field with a valid mobile number
		if (toLower(field) == "mobile" || toLower(field) == "alt") && isMobileNumber(value) {
			return value, true
		}
	}

	return "", false
}

// Helper function to find character index
func findChar(s string, ch rune) int {
	for i, c := range s {
		if c == ch {
			return i
		}
	}
	return -1
}

// Helper function to convert to lowercase
func toLower(s string) string {
	result := ""
	for _, ch := range s {
		if ch >= 'A' && ch <= 'Z' {
			result += string(ch + 32)
		} else {
			result += string(ch)
		}
	}
	return result
}

type SearchHandler struct {
	openSearchService *services.OpenSearchService
	userRepo          *repository.UserRepository
	searchHistoryRepo *repository.SearchHistoryRepository
	istLocation       *time.Location
}

func NewSearchHandler(
	openSearchService *services.OpenSearchService,
	userRepo *repository.UserRepository,
	searchHistoryRepo *repository.SearchHistoryRepository,
) *SearchHandler {
	ist, _ := time.LoadLocation("Asia/Kolkata")
	return &SearchHandler{
		openSearchService: openSearchService,
		userRepo:          userRepo,
		searchHistoryRepo: searchHistoryRepo,
		istLocation:       ist,
	}
}

func (h *SearchHandler) Search(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}
	uid := userID.(uuid.UUID)

	user, err := h.userRepo.CheckAndResetDailyLimit(c.Request.Context(), uid, h.istLocation)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to check user limits"})
		return
	}

	if !user.IsActive {
		c.JSON(http.StatusForbidden, gin.H{"error": "account is inactive"})
		return
	}

	if user.SearchesUsedToday >= user.DailySearchLimit {
		c.JSON(http.StatusTooManyRequests, gin.H{
			"error":               "daily search limit exceeded",
			"searches_used_today": user.SearchesUsedToday,
			"daily_search_limit":  user.DailySearchLimit,
			"searches_remaining":  0,
		})
		return
	}

	var req services.SearchRequest

	if c.Request.Method == "POST" {
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
	} else {
		req.Query = c.Query("q")
		if req.Query == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "query parameter 'q' is required"})
			return
		}

		if sizeStr := c.Query("size"); sizeStr != "" {
			var size int
			if _, err := fmt.Sscanf(sizeStr, "%d", &size); err == nil {
				req.Size = size
			}
		}

		if operator := c.Query("operator"); operator != "" {
			req.AndOr = operator
		}

		if fields := c.Query("fields"); fields != "" {
			req.Fields = []string{}
			for _, field := range c.QueryArray("fields[]") {
				if field != "" {
					req.Fields = append(req.Fields, field)
				}
			}
			if len(req.Fields) == 0 {
				for _, field := range splitAndTrim(fields, ",") {
					if field != "" {
						req.Fields = append(req.Fields, field)
					}
				}
			}
		}
	}

	if req.Size == 0 {
		req.Size = 50
	}
	if req.AndOr == "" {
		req.AndOr = "OR"
	}
	if len(req.Fields) == 0 {
		req.Fields = []string{"name", "fname", "address", "mobile", "alt", "id", "oid", "email"}
	}

	// Check if this is a mobile number search
	// Supports both raw numbers (9876543210) and field syntax (mobile:9876543210)
	mobileNumber, isMobileSearch := extractMobileNumber(req.Query)

	var response *services.SearchResponse
	var searchErr error

	if isMobileSearch {
		// Use comprehensive mobile search for better results
		log.Printf("Using comprehensive mobile search for number: %s (original query: %s)", mobileNumber, req.Query)
		response, searchErr = h.openSearchService.ComprehensiveMobileSearch(mobileNumber, req.Size)
		if searchErr != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": searchErr.Error()})
			return
		}
	} else {
		// Use regular search for non-mobile queries
		log.Printf("Using regular search for query: %s", req.Query)
		response, searchErr = h.openSearchService.Search(req)
		if searchErr != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": searchErr.Error()})
			return
		}
	}

	totalResults := response.Hits.Total.Value

	// Check if this is a duplicate search (same query as last search)
	isDuplicate := user.LastSearchQuery == req.Query

	if totalResults > 0 && !isDuplicate {
		h.userRepo.IncrementSearchUsage(c.Request.Context(), user.ID)

		topResults := make([]map[string]interface{}, 0)
		limit := 5
		if len(response.Hits.Hits) < limit {
			limit = len(response.Hits.Hits)
		}

		for i := 0; i < limit; i++ {
			hit := response.Hits.Hits[i]
			topResults = append(topResults, map[string]interface{}{
				"mobile":               hit.Source.Mobile,
				"name":                 hit.Source.Name,
				"fname":                hit.Source.Fname,
				"id":                   hit.Source.ID,
				"email":                hit.Source.Email,
				"year_of_registration": hit.Source.YearOfRegistration,
			})
		}

		history := &models.SearchHistory{
			UserID:       user.ID,
			Query:        req.Query,
			TotalResults: totalResults,
			TopResults:   topResults,
		}
		h.searchHistoryRepo.Create(c.Request.Context(), history)

		// Update user's searches_used_today counter if not duplicate
		user.SearchesUsedToday++
	}

	// Always update last search query
	h.userRepo.UpdateLastSearchQuery(c.Request.Context(), user.ID, req.Query)

	results := make([]map[string]interface{}, 0, len(response.Hits.Hits))
	for _, hit := range response.Hits.Hits {
		results = append(results, map[string]interface{}{
			"mobile":               hit.Source.Mobile,
			"name":                 hit.Source.Name,
			"fname":                hit.Source.Fname,
			"address":              hit.Source.Address,
			"alt_address":          hit.Source.AltAddress,
			"alt":                  hit.Source.Alt,
			"id":                   hit.Source.ID,
			"oid":                  hit.Source.OID,
			"email":                hit.Source.Email,
			"year_of_registration": hit.Source.YearOfRegistration,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"total":               totalResults,
		"results":             results,
		"took_ms":             response.Took,
		"searches_used_today": user.SearchesUsedToday,
		"daily_search_limit":  user.DailySearchLimit,
		"searches_remaining":  user.DailySearchLimit - user.SearchesUsedToday,
		"is_duplicate":        isDuplicate && totalResults > 0,
	})
}

func splitAndTrim(s, sep string) []string {
	parts := []string{}
	for _, p := range splitString(s, sep) {
		trimmed := trimSpace(p)
		if trimmed != "" {
			parts = append(parts, trimmed)
		}
	}
	return parts
}

func splitString(s, sep string) []string {
	result := []string{}
	current := ""
	for _, ch := range s {
		if string(ch) == sep {
			result = append(result, current)
			current = ""
		} else {
			current += string(ch)
		}
	}
	if current != "" {
		result = append(result, current)
	}
	return result
}

func trimSpace(s string) string {
	start := 0
	end := len(s)

	for start < end && (s[start] == ' ' || s[start] == '\t' || s[start] == '\n' || s[start] == '\r') {
		start++
	}

	for end > start && (s[end-1] == ' ' || s[end-1] == '\t' || s[end-1] == '\n' || s[end-1] == '\r') {
		end--
	}

	return s[start:end]
}

func (h *SearchHandler) Suggest(c *gin.Context) {
	query := c.Query("q")
	if query == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "query parameter 'q' is required"})
		return
	}

	req := services.SearchRequest{
		Query:  query,
		Fields: []string{"name", "fname", "address", "mobile", "alt", "id"},
		AndOr:  "OR",
		Size:   5,
	}

	response, err := h.openSearchService.Search(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Transform OpenSearch response to frontend-friendly format
	results := make([]map[string]interface{}, 0, len(response.Hits.Hits))
	for _, hit := range response.Hits.Hits {
		results = append(results, map[string]interface{}{
			"mobile":               hit.Source.Mobile,
			"name":                 hit.Source.Name,
			"fname":                hit.Source.Fname,
			"address":              hit.Source.Address,
			"alt_address":          hit.Source.AltAddress,
			"alt":                  hit.Source.Alt,
			"id":                   hit.Source.ID,
			"oid":                  hit.Source.OID,
			"email":                hit.Source.Email,
			"year_of_registration": hit.Source.YearOfRegistration,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"total":   response.Hits.Total.Value,
		"results": results,
		"took_ms": response.Took,
	})
}
