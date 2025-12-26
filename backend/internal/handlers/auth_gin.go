package handlers

import (
	"log"
	"net/http"
	"strings"
	"time"

	"notorious-backend/internal/auth"
	"notorious-backend/internal/models"
	"notorious-backend/internal/repository"
	"notorious-backend/internal/utils"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type AuthGinHandler struct {
	userRepo         *repository.UserRepository
	userRequestRepo  *repository.UserRequestRepository
	metadataRepo     *repository.MetadataRepository
	adminSessionRepo *repository.AdminSessionRepository
	sessionRepo      *repository.SessionRepository
	jwtManager       *auth.JWTManager
}

func NewAuthGinHandler(
	userRepo *repository.UserRepository,
	userRequestRepo *repository.UserRequestRepository,
	metadataRepo *repository.MetadataRepository,
	adminSessionRepo *repository.AdminSessionRepository,
	sessionRepo *repository.SessionRepository,
	jwtManager *auth.JWTManager,
) *AuthGinHandler {
	return &AuthGinHandler{
		userRepo:         userRepo,
		userRequestRepo:  userRequestRepo,
		metadataRepo:     metadataRepo,
		adminSessionRepo: adminSessionRepo,
		sessionRepo:      sessionRepo,
		jwtManager:       jwtManager,
	}
}

func (h *AuthGinHandler) Login(c *gin.Context) {
	var req struct {
		Email    string `json:"email" binding:"required"`
		Password string `json:"password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "email and password are required"})
		return
	}

	user, err := h.userRepo.GetByEmail(c.Request.Context(), req.Email)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}

	if !user.IsActive {
		c.JSON(http.StatusForbidden, gin.H{"error": "account is inactive"})
		return
	}

	if err := auth.CheckPassword(user.PasswordHash, req.Password); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}

	// Device Limit Check (Skip for Admins)
	if user.Role != models.RoleAdmin && h.sessionRepo != nil {
		activeSessions, err := h.sessionRepo.CountActiveSessions(c.Request.Context(), user.ID)
		if err != nil {
			// SECURITY: Fail closed - if we can't check, deny access
			log.Printf("ERROR: Failed to check device limit for %s: %v", req.Email, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to verify device limit"})
			return
		}

		if activeSessions >= user.DeviceLimit {
			sessions, _ := h.sessionRepo.GetActiveSessions(c.Request.Context(), user.ID)
			log.Printf("SECURITY: Device limit exceeded for %s (%d >= %d)", req.Email, activeSessions, user.DeviceLimit)
			c.JSON(http.StatusConflict, gin.H{
				"error":          "device_limit_exceeded",
				"message":        "You have reached your device limit.",
				"limit":          user.DeviceLimit,
				"active_devices": sessions,
			})
			return
		}
	}

	token, err := h.jwtManager.Generate(user.ID, user.Email, string(user.Role))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate token"})
		return
	}

	istLocation, _ := time.LoadLocation("Asia/Kolkata")
	user, _ = h.userRepo.CheckAndResetDailyLimit(c.Request.Context(), user.ID, istLocation)

	// Create User Session
	if h.sessionRepo != nil {
		userAgent := c.Request.UserAgent()
		ip := utils.GetClientIP(c.Request)
		deviceInfo := utils.ParseUserAgent(userAgent)
		location, _ := utils.GetIPLocation(ip)

		session := &models.UserSession{
			UserID:     user.ID,
			TokenHash:  auth.HashToken(token), // Hash the token for security
			DeviceName: deviceInfo.DeviceName, // Use the friendly name we constructed
			DeviceOS:   deviceInfo.OS,
			DeviceType: deviceInfo.DeviceType,
			IPAddress:  ip,
			LastActive: time.Now(),
			CreatedAt:  time.Now(),
		}

		if location != nil {
			session.Location = location.GetLocationString()
		}

		_ = h.sessionRepo.Create(c.Request.Context(), session)
	}

	// Track admin session if user is admin (keep existing logic)
	if user.Role == models.RoleAdmin && h.adminSessionRepo != nil {
		ip := utils.GetClientIP(c.Request)
		userAgent := c.Request.UserAgent()
		deviceInfo := utils.ParseUserAgent(userAgent)

		location, _ := utils.GetIPLocation(ip)

		session := &models.AdminSession{
			AdminID:        user.ID,
			IPAddress:      &ip,
			DeviceType:     &deviceInfo.DeviceType,
			Browser:        &deviceInfo.Browser,
			BrowserVersion: &deviceInfo.BrowserVersion,
			OS:             &deviceInfo.OS,
			OSVersion:      &deviceInfo.OSVersion,
			UserAgent:      &userAgent,
			ExpiresAt:      time.Now().Add(24 * time.Hour),
		}

		if location != nil {
			session.Country = &location.Country
			session.CountryCode = &location.CountryCode
			session.City = &location.City
			if location.Latitude != 0 {
				session.Latitude = &location.Latitude
				session.Longitude = &location.Longitude
			}
			if location.Timezone != "" {
				session.Timezone = &location.Timezone
			}
		}

		_ = h.adminSessionRepo.CreateSession(c.Request.Context(), session, token)
	}

	// SECURITY: Set httpOnly cookie with secure settings
	// Cookie settings: HttpOnly, Secure (HTTPS), SameSite=Strict
	isProduction := c.Request.TLS != nil || c.GetHeader("X-Forwarded-Proto") == "https"
	c.SetSameSite(http.SameSiteStrictMode)
	c.SetCookie(
		"auth_token", // name
		token,        // value
		6*60*60,      // maxAge (6 hours in seconds)
		"/",          // path
		"",           // domain (empty = current domain)
		isProduction, // secure (true in production with HTTPS)
		true,         // httpOnly (cannot be accessed by JavaScript)
	)

	c.JSON(http.StatusOK, gin.H{
		"token": token,
		"user":  user,
	})
}

func (h *AuthGinHandler) RequestAccess(c *gin.Context) {
	var req struct {
		Email                   string `json:"email" binding:"required,email"`
		Name                    string `json:"name" binding:"required"`
		Phone                   string `json:"phone" binding:"required"`
		RequestedSearchesPerDay int    `json:"requested_searches_per_day" binding:"required,min=1,max=10000"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userRequest := &models.UserRequest{
		Email:                   req.Email,
		Name:                    req.Name,
		Phone:                   req.Phone,
		RequestedSearchesPerDay: req.RequestedSearchesPerDay,
	}

	// Capture user request metadata for admin review
	if h.metadataRepo != nil {
		ip := utils.GetClientIP(c.Request)
		userAgent := c.Request.UserAgent()
		deviceInfo := utils.ParseUserAgent(userAgent)

		userRequest.IPAddress = &ip
		userRequest.DeviceType = &deviceInfo.DeviceType
		userRequest.Browser = &deviceInfo.Browser
		userRequest.OS = &deviceInfo.OS
		userRequest.UserAgent = &userAgent

		// Get location info
		if location, err := utils.GetIPLocation(ip); err == nil && location != nil {
			userRequest.Country = &location.Country
			userRequest.City = &location.City
		}
	}

	if err := h.userRequestRepo.Create(c.Request.Context(), userRequest); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create request"})
		return
	}

	c.JSON(http.StatusCreated, userRequest)
}

func (h *AuthGinHandler) RemoteLogout(c *gin.Context) {
	sessionIDStr := c.Param("id")
	sessionID, err := uuid.Parse(sessionIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid session id"})
		return
	}

	userID := c.MustGet("userID").(uuid.UUID)

	if err := h.sessionRepo.Delete(c.Request.Context(), sessionID, userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to logout device"})
		return
	}

	c.Status(http.StatusNoContent)
}

func (h *AuthGinHandler) RevokeSession(c *gin.Context) {
	var req struct {
		Email     string `json:"email" binding:"required,email"`
		Password  string `json:"password" binding:"required"`
		SessionID string `json:"session_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "email, password, and session_id are required"})
		return
	}

	user, err := h.userRepo.GetByEmail(c.Request.Context(), req.Email)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}

	if err := auth.CheckPassword(user.PasswordHash, req.Password); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}

	sessionID, err := uuid.Parse(req.SessionID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid session id"})
		return
	}

	if err := h.sessionRepo.Delete(c.Request.Context(), sessionID, user.ID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to revoke session"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "session revoked successfully"})
}

func (h *AuthGinHandler) Logout(c *gin.Context) {
	var tokenString string

	// SECURITY: First try to get token from httpOnly cookie
	if cookie, err := c.Cookie("auth_token"); err == nil && cookie != "" {
		tokenString = cookie
	} else {
		// Fallback to Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "not authenticated"})
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid authorization header format"})
			return
		}
		tokenString = parts[1]
	}

	tokenHash := auth.HashToken(tokenString)

	if err := h.sessionRepo.InvalidateSessionByTokenHash(c.Request.Context(), tokenHash); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to logout"})
		return
	}

	// SECURITY: Clear the httpOnly cookie
	c.SetSameSite(http.SameSiteStrictMode)
	c.SetCookie(
		"auth_token", // name
		"",           // value (empty)
		-1,           // maxAge (-1 = delete immediately)
		"/",          // path
		"",           // domain
		true,         // secure
		true,         // httpOnly
	)

	c.JSON(http.StatusOK, gin.H{"message": "logged out successfully"})
}
