package repository

import (
	"context"
	"fmt"
	"notorious-backend/internal/database"
	"notorious-backend/internal/models"

	"github.com/google/uuid"
)

type SessionRepository struct {
	db *database.DB
}

func NewSessionRepository(db *database.DB) *SessionRepository {
	return &SessionRepository{db: db}
}

func (r *SessionRepository) Create(ctx context.Context, session *models.UserSession) error {
	query := `
		INSERT INTO user_sessions (
			user_id, token_hash, device_name, device_os, device_type,
			ip_address, location, last_active, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id
	`
	return r.db.Pool.QueryRow(ctx, query,
		session.UserID,
		session.TokenHash,
		session.DeviceName,
		session.DeviceOS,
		session.DeviceType,
		session.IPAddress,
		session.Location,
		session.LastActive,
		session.CreatedAt,
	).Scan(&session.ID)
}

func (r *SessionRepository) CountActiveSessions(ctx context.Context, userID uuid.UUID) (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM user_sessions WHERE user_id = $1`
	err := r.db.Pool.QueryRow(ctx, query, userID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count sessions: %w", err)
	}
	return count, nil
}

func (r *SessionRepository) GetActiveSessions(ctx context.Context, userID uuid.UUID) ([]models.UserSession, error) {
	query := `
		SELECT id, user_id, device_name, device_os, device_type,
		       ip_address, location, last_active, created_at
		FROM user_sessions
		WHERE user_id = $1
		ORDER BY last_active DESC
	`
	rows, err := r.db.Pool.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query sessions: %w", err)
	}
	defer rows.Close()

	var sessions []models.UserSession
	for rows.Next() {
		var s models.UserSession
		if err := rows.Scan(
			&s.ID, &s.UserID, &s.DeviceName, &s.DeviceOS, &s.DeviceType,
			&s.IPAddress, &s.Location, &s.LastActive, &s.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan session: %w", err)
		}
		sessions = append(sessions, s)
	}
	return sessions, nil
}

func (r *SessionRepository) Delete(ctx context.Context, sessionID uuid.UUID, userID uuid.UUID) error {
	query := `DELETE FROM user_sessions WHERE id = $1 AND user_id = $2`
	_, err := r.db.Pool.Exec(ctx, query, sessionID, userID)
	if err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}
	return nil
}

// DeleteByID deletes a session by ID only (for admin use)
func (r *SessionRepository) DeleteByID(ctx context.Context, sessionID uuid.UUID) error {
	query := `DELETE FROM user_sessions WHERE id = $1`
	_, err := r.db.Pool.Exec(ctx, query, sessionID)
	if err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}
	return nil
}

func (r *SessionRepository) DeleteAllForUser(ctx context.Context, userID uuid.UUID) error {
	query := `DELETE FROM user_sessions WHERE user_id = $1`
	_, err := r.db.Pool.Exec(ctx, query, userID)
	if err != nil {
		return fmt.Errorf("failed to delete all sessions: %w", err)
	}
	return nil
}

// InvalidateSessionByTokenHash deletes a session by its token hash
func (r *SessionRepository) InvalidateSessionByTokenHash(ctx context.Context, tokenHash string) error {
	query := `DELETE FROM user_sessions WHERE token_hash = $1`
	_, err := r.db.Pool.Exec(ctx, query, tokenHash)
	return err
}

// ExistsByTokenHash checks if a session exists by its token hash
func (r *SessionRepository) ExistsByTokenHash(ctx context.Context, tokenHash string) (bool, error) {
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM user_sessions WHERE token_hash = $1)`
	err := r.db.Pool.QueryRow(ctx, query, tokenHash).Scan(&exists)
	return exists, err
}

// UpdateLastActive updates the last_active timestamp for a session by token hash
// Throttled: only updates if last_active is older than 30 seconds to reduce DB load
func (r *SessionRepository) UpdateLastActive(ctx context.Context, tokenHash string) error {
	query := `
		UPDATE user_sessions
		SET last_active = NOW()
		WHERE token_hash = $1
		AND last_active < NOW() - INTERVAL '30 seconds'
	`
	_, err := r.db.Pool.Exec(ctx, query, tokenHash)
	return err
}

// GetOnlineUserIDs returns user IDs that have been active within the threshold seconds
func (r *SessionRepository) GetOnlineUserIDs(ctx context.Context, thresholdSeconds int) ([]uuid.UUID, error) {
	query := `
		SELECT DISTINCT user_id
		FROM user_sessions
		WHERE last_active > NOW() - INTERVAL '1 second' * $1
	`
	rows, err := r.db.Pool.Query(ctx, query, thresholdSeconds)
	if err != nil {
		return nil, fmt.Errorf("failed to query online users: %w", err)
	}
	defer rows.Close()

	var userIDs []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("failed to scan user ID: %w", err)
		}
		userIDs = append(userIDs, id)
	}
	return userIDs, nil
}
