package handlers

import (
	"net/http"
	"time"

	"notorious-backend/internal/auth"
	"notorious-backend/internal/models"
	"notorious-backend/internal/repository"
	"notorious-backend/internal/utils"

	"github.com/gin-gonic/gin"
)

type AuthGinHandler struct {
	userRepo           *repository.UserRepository
	userRequestRepo    *repository.UserRequestRepository
	metadataRepo       *repository.MetadataRepository
	adminSessionRepo   *repository.AdminSessionRepository
	jwtManager         *auth.JWTManager
}

func NewAuthGinHandler(
	userRepo *repository.UserRepository,
	userRequestRepo *repository.UserRequestRepository,
	metadataRepo *repository.MetadataRepository,
	adminSessionRepo *repository.AdminSessionRepository,
	jwtManager *auth.JWTManager,
) *AuthGinHandler {
	return &AuthGinHandler{
		userRepo:         userRepo,
		userRequestRepo:  userRequestRepo,
		metadataRepo:     metadataRepo,
		adminSessionRepo: adminSessionRepo,
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

	token, err := h.jwtManager.Generate(user.ID, user.Email, string(user.Role))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate token"})
		return
	}

	istLocation, _ := time.LoadLocation("Asia/Kolkata")
	user, _ = h.userRepo.CheckAndResetDailyLimit(c.Request.Context(), user.ID, istLocation)

	// Track admin session if user is admin
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

