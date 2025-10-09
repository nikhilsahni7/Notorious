package handlers

import (
	"fmt"
	"net/http"

	"notorious-backend/internal/services"

	"github.com/gin-gonic/gin"
)

type SearchHandler struct {
	openSearchService *services.OpenSearchService
}

func NewSearchHandler(openSearchService *services.OpenSearchService) *SearchHandler {
	return &SearchHandler{
		openSearchService: openSearchService,
	}
}

func (h *SearchHandler) Search(c *gin.Context) {
	// Support both GET (query params) and POST (JSON body)
	var req services.SearchRequest

	if c.Request.Method == "POST" {
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
	} else {
		// GET request - parse query params
		req.Query = c.Query("q")
		if req.Query == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "query parameter 'q' is required"})
			return
		}

		// Parse size
		if sizeStr := c.Query("size"); sizeStr != "" {
			var size int
			if _, err := fmt.Sscanf(sizeStr, "%d", &size); err == nil {
				req.Size = size
			}
		}

		// Parse operator (and_or)
		if operator := c.Query("operator"); operator != "" {
			req.AndOr = operator
		}

		// Parse fields (comma-separated)
		if fields := c.Query("fields"); fields != "" {
			req.Fields = []string{}
			for _, field := range c.QueryArray("fields[]") {
				if field != "" {
					req.Fields = append(req.Fields, field)
				}
			}
			// If no fields[] array, try comma-separated
			if len(req.Fields) == 0 {
				for _, field := range splitAndTrim(fields, ",") {
					if field != "" {
						req.Fields = append(req.Fields, field)
					}
				}
			}
		}
	}

	// Set default values
	if req.Size == 0 {
		req.Size = 10
	}
	if req.AndOr == "" {
		req.AndOr = "OR"
	}
	if len(req.Fields) == 0 {
		req.Fields = []string{"name", "fname", "address", "mobile", "alt", "id"}
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
