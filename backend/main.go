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
	"notorious-backend/internal/utils"

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
	var userPasswordHandler *handlers.UserPasswordGinHandler
	var searchHandler *handlers.SearchHandler

	if databaseURL != "" && jwtSecret != "" {
		var err error
		db, err = database.NewPostgresDB(databaseURL)
		if err != nil {
			log.Fatalf("CRITICAL: Failed to connect to database: %v. The server cannot start without a database.", err)
		} else {

			log.Println("Successfully connected to PostgreSQL database for KNotorious")

			// Run migrations
			if err := db.RunMigrations("./migrations"); err != nil {
				log.Fatalf("Failed to run migrations: %v", err)
			}

			userRepo := repository.NewUserRepository(db)
			userRequestRepo := repository.NewUserRequestRepository(db)
			searchHistoryRepo := repository.NewSearchHistoryRepository(db)
			passwordChangeRepo := repository.NewPasswordChangeRepository(db)
			metadataRepo := repository.NewMetadataRepository(db)
			adminSessionRepo := repository.NewAdminSessionRepository(db)
			sessionRepo := repository.NewSessionRepository(db)

			// Initialize GeoIP (optional - falls back to API if not available)
			geoipPath := os.Getenv("GEOIP_DB_PATH")
			if geoipPath == "" {
				geoipPath = "./GeoLite2-City.mmdb"
			}
			utils.InitGeoIP(geoipPath)

			jwtManager := auth.NewJWTManager(jwtSecret, 24*time.Hour)
			authMiddleware = middleware.NewGinAuthMiddleware(jwtManager, sessionRepo)

			authHandler = handlers.NewAuthGinHandler(userRepo, userRequestRepo, metadataRepo, adminSessionRepo, sessionRepo, jwtManager)
			adminHandler = handlers.NewAdminGinHandler(userRepo, userRequestRepo, searchHistoryRepo, passwordChangeRepo, metadataRepo, adminSessionRepo, sessionRepo)
			userHandler = handlers.NewUserGinHandler(searchHistoryRepo, metadataRepo)
			userPasswordHandler = handlers.NewUserPasswordGinHandler(passwordChangeRepo)
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
			"https://knotoriousworld.site", "https://notorious-sigma.vercel.app",
			"https://www.knotoriousworld.site"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	r.GET("/deploy", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "deploy successful"})
	})

	if authHandler != nil {
		// Rate limit login attempts (10 per minute per IP)
		r.POST("/auth/login", middleware.AuthRateLimiter(), authHandler.Login)
		// Rate limit access requests heavily (5 per hour per IP) - prevents spam
		r.POST("/auth/request-access", middleware.AccessRequestRateLimiter(), authHandler.RequestAccess)
		r.POST("/auth/revoke-session", middleware.AuthRateLimiter(), authHandler.RevokeSession)
		if authMiddleware != nil {
			r.POST("/auth/logout", authMiddleware.AuthRequired(), authHandler.Logout)
		}
	}

	if authMiddleware != nil && userHandler != nil {
		userRoutes := r.Group("/api/user")
		userRoutes.Use(authMiddleware.AuthRequired())
		{
			userRoutes.GET("/search-history", userHandler.GetSearchHistory)
			userRoutes.GET("/metadata", userHandler.GetMetadata)
		}
	}

	if authMiddleware != nil && userPasswordHandler != nil {
		passwordRoutes := r.Group("/api/user/password-change")
		passwordRoutes.Use(authMiddleware.AuthRequired())
		{
			passwordRoutes.POST("/request", userPasswordHandler.RequestPasswordChange)
			passwordRoutes.GET("/requests", userPasswordHandler.GetPasswordChangeRequests)
		}
	}

	if authMiddleware != nil && adminHandler != nil {
		adminRoutes := r.Group("/api/admin")
		// Rate limit admin routes (30 per minute per IP)
		adminRoutes.Use(authMiddleware.AuthRequired(), authMiddleware.RequireRole("admin"), middleware.AdminRateLimiter())
		{
			// User management
			adminRoutes.GET("/users", adminHandler.ListUsers)
			adminRoutes.POST("/users", adminHandler.CreateUser)
			adminRoutes.GET("/users/:id", adminHandler.GetUser)
			adminRoutes.GET("/users/:id/details", adminHandler.GetUserDetails) // NEW: Get user with metadata
			adminRoutes.PUT("/users/:id", adminHandler.UpdateUser)
			adminRoutes.DELETE("/users/:id", adminHandler.DeleteUser)
			adminRoutes.POST("/users/:id/change-password", adminHandler.ChangeUserPassword)
			adminRoutes.GET("/users/:id/eod-report", adminHandler.GenerateUserEOD) // NEW: Generate EOD for user

			// User requests
			adminRoutes.GET("/user-requests", adminHandler.ListUserRequests)
			adminRoutes.POST("/user-requests/:id/approve", adminHandler.ApproveUserRequest)
			adminRoutes.POST("/user-requests/:id/reject", adminHandler.RejectUserRequest)

			// Password change requests
			adminRoutes.GET("/password-change-requests", adminHandler.ListPasswordChangeRequests)
			adminRoutes.POST("/password-change-requests/:id/approve", adminHandler.ApprovePasswordChangeRequest)
			adminRoutes.POST("/password-change-requests/:id/reject", adminHandler.RejectPasswordChangeRequest)

			// Search history
			adminRoutes.GET("/search-history", adminHandler.GetSearchHistory)
			adminRoutes.GET("/users/:id/search-history", adminHandler.GetUserSearchHistory)

			// Session management
			adminRoutes.GET("/sessions", adminHandler.GetAdminSessions)         // NEW: Get all admin sessions
			adminRoutes.DELETE("/sessions/:id", adminHandler.InvalidateSession) // NEW: Invalidate session

			// Dashboard stats
			adminRoutes.GET("/request-counts", adminHandler.GetRequestCounts) // NEW: Get pending request counts
		}
	}

	uploadGroup := r.Group("/upload")
	uploadGroup.POST("/init", uploadHandler.InitUpload)
	uploadGroup.POST("/presign", uploadHandler.PresignPart)
	uploadGroup.POST("/complete", uploadHandler.CompleteUpload)
	uploadGroup.POST("/abort", uploadHandler.AbortUpload)

	if authMiddleware != nil && searchHandler != nil {
		searchRoutes := r.Group("/search")
		// Rate limit search (100 per minute per IP)
		searchRoutes.Use(authMiddleware.AuthRequired(), middleware.SearchRateLimiter())
		{
			searchRoutes.GET("", searchHandler.Search)
			searchRoutes.POST("", searchHandler.Search)
			searchRoutes.POST("/refine", searchHandler.RefineSearch)
			searchRoutes.GET("/suggest", searchHandler.Suggest)
			searchRoutes.GET("/export-eod", searchHandler.ExportEODReport)
		}
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server starting on port %s", port)
	r.Run(":" + port)
}
