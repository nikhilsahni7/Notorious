package repository

import (
	"context"
	"crypto/sha256"
	"encoding/hex"

	"github.com/google/uuid"
	"notorious-backend/internal/database"
	"notorious-backend/internal/models"
)

type AdminSessionRepository struct {
	db *database.DB
}

func NewAdminSessionRepository(db *database.DB) *AdminSessionRepository {
	return &AdminSessionRepository{db: db}
}

// hashToken creates a SHA-256 hash of the token for storage
func hashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}

// CreateSession stores a new admin session
func (r *AdminSessionRepository) CreateSession(ctx context.Context, session *models.AdminSession, token string) error {
	session.TokenHash = hashToken(token)
	query := `
		INSERT INTO admin_sessions (
			admin_id, token_hash, ip_address, country, country_code, city,
			latitude, longitude, timezone, device_type, browser, browser_version,
			os, os_version, user_agent, expires_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
		RETURNING id, created_at, last_used_at
	`
	return r.db.Pool.QueryRow(ctx, query,
		session.AdminID, session.TokenHash, session.IPAddress, session.Country,
		session.CountryCode, session.City, session.Latitude, session.Longitude,
		session.Timezone, session.DeviceType, session.Browser, session.BrowserVersion,
		session.OS, session.OSVersion, session.UserAgent, session.ExpiresAt,
	).Scan(&session.ID, &session.CreatedAt, &session.LastUsedAt)
}

// GetActiveSessions retrieves all active sessions with admin details
func (r *AdminSessionRepository) GetActiveSessions(ctx context.Context, limit, offset int) ([]*models.AdminSessionWithUser, error) {
	sessions := make([]*models.AdminSessionWithUser, 0)
	query := `
		SELECT 
			s.id, s.admin_id, s.ip_address, s.country, s.country_code, s.city,
			s.latitude, s.longitude, s.timezone, s.device_type, s.browser,
			s.browser_version, s.os, s.os_version, s.user_agent,
			s.is_active, s.created_at, s.last_used_at, s.expires_at,
			u.email, u.name
		FROM admin_sessions s
		JOIN users u ON s.admin_id = u.id
		WHERE s.is_active = true AND s.expires_at > NOW()
		ORDER BY s.last_used_at DESC
		LIMIT $1 OFFSET $2
	`
	rows, err := r.db.Pool.Query(ctx, query, limit, offset)
	if err != nil {
		return sessions, err
	}
	defer rows.Close()

	for rows.Next() {
		var session models.AdminSessionWithUser
		if err := rows.Scan(
			&session.ID, &session.AdminID, &session.IPAddress, &session.Country,
			&session.CountryCode, &session.City, &session.Latitude, &session.Longitude,
			&session.Timezone, &session.DeviceType, &session.Browser, &session.BrowserVersion,
			&session.OS, &session.OSVersion, &session.UserAgent, &session.IsActive,
			&session.CreatedAt, &session.LastUsedAt, &session.ExpiresAt,
			&session.AdminEmail, &session.AdminName,
		); err != nil {
			return sessions, err
		}
		sessions = append(sessions, &session)
	}
	return sessions, rows.Err()
}

// InvalidateSession marks a session as inactive
func (r *AdminSessionRepository) InvalidateSession(ctx context.Context, sessionID uuid.UUID) error {
	query := `
		UPDATE admin_sessions
		SET is_active = false
		WHERE id = $1
	`
	_, err := r.db.Pool.Exec(ctx, query, sessionID)
	return err
}

// InvalidateSessionByToken marks a session as inactive by token
func (r *AdminSessionRepository) InvalidateSessionByToken(ctx context.Context, token string) error {
	tokenHash := hashToken(token)
	query := `
		UPDATE admin_sessions
		SET is_active = false
		WHERE token_hash = $1
	`
	_, err := r.db.Pool.Exec(ctx, query, tokenHash)
	return err
}

// UpdateLastUsed updates the last_used_at timestamp
func (r *AdminSessionRepository) UpdateLastUsed(ctx context.Context, token string) error {
	tokenHash := hashToken(token)
	query := `
		UPDATE admin_sessions
		SET last_used_at = NOW()
		WHERE token_hash = $1 AND is_active = true AND expires_at > NOW()
	`
	_, err := r.db.Pool.Exec(ctx, query, tokenHash)
	return err
}

// IsSessionValid checks if a session is valid
func (r *AdminSessionRepository) IsSessionValid(ctx context.Context, token string) (bool, error) {
	tokenHash := hashToken(token)
	var count int
	query := `
		SELECT COUNT(*) FROM admin_sessions
		WHERE token_hash = $1 AND is_active = true AND expires_at > NOW()
	`
	err := r.db.Pool.QueryRow(ctx, query, tokenHash).Scan(&count)
	return count > 0, err
}

// CleanupExpiredSessions removes expired sessions
func (r *AdminSessionRepository) CleanupExpiredSessions(ctx context.Context) error {
	query := `
		UPDATE admin_sessions
		SET is_active = false
		WHERE expires_at < NOW() AND is_active = true
	`
	_, err := r.db.Pool.Exec(ctx, query)
	return err
}

