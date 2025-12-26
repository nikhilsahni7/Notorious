package middleware

import (
	"net/http"
	"strings"

	"notorious-backend/internal/auth"
	"notorious-backend/internal/repository"

	"github.com/gin-gonic/gin"
)

type GinAuthMiddleware struct {
	jwtManager  *auth.JWTManager
	sessionRepo *repository.SessionRepository
}

func NewGinAuthMiddleware(jwtManager *auth.JWTManager, sessionRepo *repository.SessionRepository) *GinAuthMiddleware {
	return &GinAuthMiddleware{
		jwtManager:  jwtManager,
		sessionRepo: sessionRepo,
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
