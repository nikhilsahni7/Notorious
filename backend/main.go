package main

import (
	"context"
	"log"
	"os"
	"time"

	"notorious-backend/internal/config"
	"notorious-backend/internal/handlers"
	"notorious-backend/internal/services"

	"notorious-backend/internal/auth"
	"notorious-backend/internal/database"
	"notorious-backend/internal/middleware"
	"notorious-backend/internal/repository"
	"notorious-backend/internal/scheduler"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using system environment variables")
	}

	cfg := config.Load()

	databaseURL := os.Getenv("DATABASE_URL")
	jwtSecret := os.Getenv("JWT_SECRET")

	var db *database.DB
	var authMiddleware *middleware.GinAuthMiddleware
	var authHandler *handlers.AuthGinHandler
	var adminHandler *handlers.AdminGinHandler
	var userHandler *handlers.UserGinHandler
	var searchHandler *handlers.SearchHandler

	if databaseURL != "" && jwtSecret != "" {
		var err error
		db, err = database.NewPostgresDB(databaseURL)
		if err != nil {
			log.Printf("Warning: Failed to connect to database: %v", err)
		} else {
			log.Println("Successfully connected to PostgreSQL database")

			userRepo := repository.NewUserRepository(db)
			userRequestRepo := repository.NewUserRequestRepository(db)
			searchHistoryRepo := repository.NewSearchHistoryRepository(db)

			jwtManager := auth.NewJWTManager(jwtSecret, 24*time.Hour)
			authMiddleware = middleware.NewGinAuthMiddleware(jwtManager)

			authHandler = handlers.NewAuthGinHandler(userRepo, userRequestRepo, jwtManager)
			adminHandler = handlers.NewAdminGinHandler(userRepo, userRequestRepo, searchHistoryRepo)
			userHandler = handlers.NewUserGinHandler(searchHistoryRepo)
			searchHandler = handlers.NewSearchHandler(services.NewOpenSearchService(cfg), userRepo, searchHistoryRepo)

			resetter := scheduler.NewSearchLimitResetter(userRepo)
			ctx := context.Background()
			resetter.Start(ctx)
		}
	}

	uploadService := services.NewUploadService(cfg)
	uploadHandler := handlers.NewUploadHandler(uploadService)

	r := gin.Default()

	r.Use(cors.New(cors.Config{
		AllowOrigins: []string{"http://localhost:3000", "http://localhost:3001",
			"https://www.knotorious.co.in", "https://notorious.nikhilsahni.xyz"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	if authHandler != nil {
		r.POST("/auth/login", authHandler.Login)
		r.POST("/auth/request-access", authHandler.RequestAccess)
	}

	if authMiddleware != nil && userHandler != nil {
		userRoutes := r.Group("/api/user")
		userRoutes.Use(authMiddleware.AuthRequired())
		{
			userRoutes.GET("/search-history", userHandler.GetSearchHistory)
		}
	}

	if authMiddleware != nil && adminHandler != nil {
		adminRoutes := r.Group("/api/admin")
		adminRoutes.Use(authMiddleware.AuthRequired(), authMiddleware.RequireRole("admin"))
		{
			adminRoutes.GET("/users", adminHandler.ListUsers)
			adminRoutes.POST("/users", adminHandler.CreateUser)
			adminRoutes.GET("/users/:id", adminHandler.GetUser)
			adminRoutes.PUT("/users/:id", adminHandler.UpdateUser)
			adminRoutes.DELETE("/users/:id", adminHandler.DeleteUser)
			adminRoutes.GET("/user-requests", adminHandler.ListUserRequests)
			adminRoutes.POST("/user-requests/:id/approve", adminHandler.ApproveUserRequest)
			adminRoutes.POST("/user-requests/:id/reject", adminHandler.RejectUserRequest)
			adminRoutes.GET("/search-history", adminHandler.GetSearchHistory)
			adminRoutes.GET("/users/:id/search-history", adminHandler.GetUserSearchHistory)
		}
	}

	uploadGroup := r.Group("/upload")
	uploadGroup.POST("/init", uploadHandler.InitUpload)
	uploadGroup.POST("/presign", uploadHandler.PresignPart)
	uploadGroup.POST("/complete", uploadHandler.CompleteUpload)
	uploadGroup.POST("/abort", uploadHandler.AbortUpload)

	if authMiddleware != nil && searchHandler != nil {
		searchRoutes := r.Group("/search")
		searchRoutes.Use(authMiddleware.AuthRequired())
		{
			searchRoutes.GET("", searchHandler.Search)
			searchRoutes.POST("", searchHandler.Search)
			searchRoutes.GET("/suggest", searchHandler.Suggest)
		}
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server starting on port %s", port)
	r.Run(":" + port)
}
