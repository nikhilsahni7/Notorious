package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"notorious-backend/internal/services"
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
	var req services.SearchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Set default values
	if req.Size == 0 {
		req.Size = 10
	}
	if req.AndOr == "" {
		req.AndOr = "OR"
	}
	if len(req.Fields) == 0 {
		req.Fields = []string{"name", "address", "mobile", "alt", "id"}
	}

	response, err := h.openSearchService.Search(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, response)
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

	c.JSON(http.StatusOK, response)
}
