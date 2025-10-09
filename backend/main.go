package main

import (
	"log"
	"os"

	"notorious-backend/internal/config"
	"notorious-backend/internal/handlers"
	"notorious-backend/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using system environment variables")
	}

	// Initialize configuration
	cfg := config.Load()

	// Initialize services
	opensearchService := services.NewOpenSearchService(cfg)
	uploadService := services.NewUploadService(cfg)

	// Initialize handlers
	uploadHandler := handlers.NewUploadHandler(uploadService)
	searchHandler := handlers.NewSearchHandler(opensearchService)

	// Setup routes
	r := gin.Default()

	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// Upload routes
	uploadGroup := r.Group("/upload")
	uploadGroup.POST("/init", uploadHandler.InitUpload)
	uploadGroup.POST("/presign", uploadHandler.PresignPart)
	uploadGroup.POST("/complete", uploadHandler.CompleteUpload)
	uploadGroup.POST("/abort", uploadHandler.AbortUpload)

	// Search routes
	r.POST("/search", searchHandler.Search)
	r.GET("/search/suggest", searchHandler.Suggest)

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server starting on port %s", port)
	r.Run(":" + port)
}
