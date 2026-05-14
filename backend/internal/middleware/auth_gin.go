package middleware

import (
	"context"
	"net/http"
	"strings"
	"sync"
	"time"

	"notorious-backend/internal/auth"
	"notorious-backend/internal/repository"

	"github.com/gin-gonic/gin"
)

// sessionCacheEntry holds a cached session existence check result.
type sessionCacheEntry struct {
	expiresAt time.Time
}

// sessionCache is an in-memory TTL cache for session token existence checks.
// This avoids hitting PostgreSQL on every authenticated request.
type sessionCache struct {
	mu      sync.RWMutex
	entries map[string]sessionCacheEntry
}

func newSessionCache() *sessionCache {
	sc := &sessionCache{entries: make(map[string]sessionCacheEntry)}
	go sc.cleanupLoop()
	return sc
}

func (sc *sessionCache) exists(tokenHash string) bool {
	sc.mu.RLock()
	defer sc.mu.RUnlock()
	e, ok := sc.entries[tokenHash]
	return ok && time.Now().Before(e.expiresAt)
}

func (sc *sessionCache) set(tokenHash string, ttl time.Duration) {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	sc.entries[tokenHash] = sessionCacheEntry{expiresAt: time.Now().Add(ttl)}
}

func (sc *sessionCache) remove(tokenHash string) {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	delete(sc.entries, tokenHash)
}

func (sc *sessionCache) cleanupLoop() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		sc.mu.Lock()
		now := time.Now()
		for hash, entry := range sc.entries {
			if now.After(entry.expiresAt) {
				delete(sc.entries, hash)
			}
		}
		sc.mu.Unlock()
	}
}

type GinAuthMiddleware struct {
	jwtManager  *auth.JWTManager
	sessionRepo *repository.SessionRepository
	cache       *sessionCache
}

func NewGinAuthMiddleware(jwtManager *auth.JWTManager, sessionRepo *repository.SessionRepository) *GinAuthMiddleware {
	return &GinAuthMiddleware{
		jwtManager:  jwtManager,
		sessionRepo: sessionRepo,
		cache:       newSessionCache(),
	}
}

func (m *GinAuthMiddleware) AuthRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		var tokenString string

		// SECURITY: First try to get token from httpOnly cookie
		if cookie, err := c.Cookie("auth_token"); err == nil && cookie != "" {
			tokenString = cookie
		} else {
			// Fallback: Check Authorization header for backward compatibility
			authHeader := c.GetHeader("Authorization")
			if authHeader == "" {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required", "code": "TOKEN_MISSING"})
				c.Abort()
				return
			}

			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || parts[0] != "Bearer" {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid authorization header format"})
				c.Abort()
				return
			}
			tokenString = parts[1]
		}

		claims, err := m.jwtManager.Verify(tokenString)
		if err != nil {
			// Return specific error for expired vs invalid
			c.JSON(http.StatusUnauthorized, gin.H{"error": "token expired or invalid", "code": "TOKEN_EXPIRED"})
			c.Abort()
			return
		}

		// Check if session exists in DB (enforce revocation)
		// Only for regular users, admin sessions are handled differently (or we can add admin session check too, but let's stick to user sessions for now)
		// Actually, admin sessions are in a different table. If role is admin, we might skip this or check admin_sessions.
		// The user request was about "device limiting" which applies to users.
		// Admin exemption logic in Login suggests admins are special.
		// However, if an admin logs out, their token should also be invalidated?
		// Admin sessions are in `admin_sessions` table.
		// Let's check user role.

		if claims.Role == "user" && m.sessionRepo != nil {
			tokenHash := auth.HashToken(tokenString)

			// Check in-memory cache first to avoid DB hit on every request.
			// Cache TTL of 30s means a revoked session takes at most 30s to take effect.
			if !m.cache.exists(tokenHash) {
				exists, err := m.sessionRepo.ExistsByTokenHash(c.Request.Context(), tokenHash)
				if err != nil {
					// If DB error, fail safe? Or allow?
					// Fail safe: deny access.
					c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to verify session"})
					c.Abort()
					return
				}
				if !exists {
					c.JSON(http.StatusUnauthorized, gin.H{"error": "session revoked or invalid"})
					c.Abort()
					return
				}
				// Session verified — cache for 30 seconds
				m.cache.set(tokenHash, 30*time.Second)
			}

			// Store token hash for heartbeat functionality
			c.Set("token_hash", tokenHash)

			// Update last_active on every authenticated request for reliable presence tracking.
			// Uses context.Background() because c.Request.Context() is cancelled when
			// the HTTP response is sent, which would cause this DB update to fail silently.
			go func() {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				m.sessionRepo.UpdateLastActive(ctx, tokenHash)
			}()
		}

		c.Set("user_id", claims.UserID)
		c.Set("user_email", claims.Email)
		c.Set("user_role", claims.Role)
		c.Next()
	}
}

func (m *GinAuthMiddleware) RequireRole(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userRole, exists := c.Get("user_role")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			c.Abort()
			return
		}

		role := userRole.(string)
		hasRole := false
		for _, r := range roles {
			if role == r {
				hasRole = true
				break
			}
		}

		if !hasRole {
			c.JSON(http.StatusForbidden, gin.H{"error": "insufficient permissions"})
			c.Abort()
			return
		}

		c.Next()
	}
}
