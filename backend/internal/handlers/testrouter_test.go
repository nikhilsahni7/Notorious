package handlers

// Test infrastructure for golden tests and benchmarks.
//
// Builds the real Gin router from main.go (same middlewares, same routes)
// against the database specified by DATABASE_URL. Read-only endpoints only:
// all requests use an admin-role JWT, and the AuthRequired middleware only
// performs a DB lookup (and a background UPDATE) for role == "user", so no
// test request writes to the database.
//
// Env required: DATABASE_URL, JWT_SECRET (falls back to ../../.env upstream).
// Tests and benchmarks skip silently when the env is not configured.

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"notorious-backend/internal/auth"
	"notorious-backend/internal/database"
	"notorious-backend/internal/middleware"
	"notorious-backend/internal/repository"
	chatws "notorious-backend/internal/websocket"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

var errEnvMissing = errors.New("DATABASE_URL or JWT_SECRET not set; skipping DB-backed test")

type testEnv struct {
	pool        *pgxpool.Pool
	router      *gin.Engine // for golden tests: rate limiters on, exactly as in main.go
	benchRouter *gin.Engine // for benchmarks: no rate limiters (they would 429 at 120 req/min)
	token       string
	adminID     uuid.UUID
}

var (
	envOnce sync.Once
	env     testEnv
	envErr  error
)

func setupEnv(t testing.TB) *testEnv {
	envOnce.Do(func() {
		gin.SetMode(gin.TestMode)

		godotenv.Load(filepath.Join("..", "..", ".env")) // best effort (backend/.env)

		dbURL := os.Getenv("DATABASE_URL")
		jwtSecret := os.Getenv("JWT_SECRET")
		if dbURL == "" || jwtSecret == "" {
			envErr = errEnvMissing
			return
		}

		db, err := database.NewPostgresDB(dbURL)
		if err != nil {
			envErr = errors.New("database not reachable: " + err.Error())
			return
		}

		// Pick an admin user to act as. Read-only lookup.
		var adminEmail string
		err = db.Pool.QueryRow(context.Background(),
			`SELECT id, email FROM users WHERE role = 'admin' ORDER BY created_at LIMIT 1`,
		).Scan(&env.adminID, &adminEmail)
		if err != nil {
			envErr = errors.New("no admin user found in database: " + err.Error())
			return
		}

		jwtManager := auth.NewJWTManager(jwtSecret, 6*time.Hour)
		token, err := jwtManager.Generate(env.adminID, adminEmail, "admin")
		if err != nil {
			envErr = errors.New("failed to generate test JWT: " + err.Error())
			return
		}
		env.token = token

		userRepo := repository.NewUserRepository(db)
		userRequestRepo := repository.NewUserRequestRepository(db)
		searchHistoryRepo := repository.NewSearchHistoryRepository(db)
		passwordChangeRepo := repository.NewPasswordChangeRepository(db)
		metadataRepo := repository.NewMetadataRepository(db)
		adminSessionRepo := repository.NewAdminSessionRepository(db)
		sessionRepo := repository.NewSessionRepository(db)
		chatRepo := repository.NewChatRepository(db)

		adminHandler := NewAdminGinHandler(userRepo, userRequestRepo, searchHistoryRepo, passwordChangeRepo, metadataRepo, adminSessionRepo, sessionRepo)
		userHandler := NewUserGinHandler(searchHistoryRepo, metadataRepo, sessionRepo)
		userPasswordHandler := NewUserPasswordGinHandler(passwordChangeRepo, userRepo)
		statsHandler := NewStatsGinHandler(db)
		chatHandler := NewChatHandler(chatRepo, chatws.NewHub(chatRepo, jwtManager))

		authMiddleware := middleware.NewGinAuthMiddleware(jwtManager, sessionRepo)

		env.router = buildRouter(ginRouterSpec{
			authMiddleware: authMiddleware,
			adminHandler:   adminHandler,
			userHandler:    userHandler,
			pwHandler:      userPasswordHandler,
			statsHandler:   statsHandler,
			chatHandler:    chatHandler,
			withLimiters:   true,
		})
		env.benchRouter = buildRouter(ginRouterSpec{
			authMiddleware: authMiddleware,
			adminHandler:   adminHandler,
			userHandler:    userHandler,
			pwHandler:      userPasswordHandler,
			statsHandler:   statsHandler,
			chatHandler:    chatHandler,
			withLimiters:   false,
		})
		env.pool = db.Pool
	})
	if envErr != nil {
		t.Skipf("test env not available: %v", envErr)
	}
	return &env
}

func TestMain(m *testing.M) {
	code := m.Run()
	// Close the pool only if a DB-backed test built it.
	if env.pool != nil {
		env.pool.Close()
	}
	os.Exit(code)
}

type ginRouterSpec struct {
	authMiddleware *middleware.GinAuthMiddleware
	adminHandler   *AdminGinHandler
	userHandler    *UserGinHandler
	pwHandler      *UserPasswordGinHandler
	statsHandler   *StatsGinHandler
	chatHandler    *ChatHandler
	withLimiters   bool
}

// buildRouter mirrors the route registration in main.go for the read-only
// groups under test, in the same order. withLimiters=false is used by
// benchmarks because the package-level in-process rate limiters (120/min for
// admin routes) would artificially cap benchmark throughput with 429s.
func buildRouter(s ginRouterSpec) *gin.Engine {
	r := gin.New()

	if s.adminHandler != nil && s.statsHandler != nil {
		adminRoutes := r.Group("/api/admin")
		adminRoutes.Use(s.authMiddleware.AuthRequired())
		if s.withLimiters {
			adminRoutes.Use(middleware.AdminRateLimiter())
		}
		adminRoutes.Use(s.authMiddleware.RequireRole("admin"))
		{
			adminRoutes.GET("/users", s.adminHandler.ListUsers)
			adminRoutes.GET("/users/:id", s.adminHandler.GetUser)
			adminRoutes.GET("/users/:id/details", s.adminHandler.GetUserDetails)
			adminRoutes.GET("/users/:id/eod-report", s.adminHandler.GenerateUserEOD)
			adminRoutes.GET("/user-requests", s.adminHandler.ListUserRequests)
			adminRoutes.GET("/password-change-requests", s.adminHandler.ListPasswordChangeRequests)
			adminRoutes.GET("/search-history", s.adminHandler.GetSearchHistory)
			adminRoutes.GET("/sessions", s.adminHandler.GetAdminSessions)
			adminRoutes.GET("/dashboard-stats", s.adminHandler.GetDashboardStats)
			adminRoutes.GET("/users/online", s.adminHandler.GetOnlineUsers)
			adminRoutes.GET("/stats/user/:userId", s.statsHandler.GetUserStats)
			adminRoutes.GET("/stats/system", s.statsHandler.GetSystemStats)
		}
	}

	if s.userHandler != nil {
		userRoutes := r.Group("/api/user")
		userRoutes.Use(s.authMiddleware.AuthRequired())
		{
			userRoutes.GET("/search-history", s.userHandler.GetSearchHistory)
			userRoutes.GET("/metadata", s.userHandler.GetMetadata)
		}
	}

	if s.pwHandler != nil {
		passwordRoutes := r.Group("/api/user/password-change")
		passwordRoutes.Use(s.authMiddleware.AuthRequired())
		{
			passwordRoutes.GET("/requests", s.pwHandler.GetPasswordChangeRequests)
		}
	}

	if s.chatHandler != nil {
		chatRoutes := r.Group("/api/chat")
		chatRoutes.Use(s.authMiddleware.AuthRequired())
		{
			chatRoutes.GET("/conversations", s.chatHandler.GetConversations)
			chatRoutes.GET("/unread", s.chatHandler.GetUnreadCount)
			chatRoutes.GET("/unread-per-user", s.chatHandler.GetUnreadPerUser)
			chatRoutes.GET("/broadcasts", s.chatHandler.GetBroadcasts)
		}
	}

	return r
}
