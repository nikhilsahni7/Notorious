package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"notorious-backend/internal/database"
	"notorious-backend/internal/models"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type NotificationRepository struct {
	db *database.DB
}

func NewNotificationRepository(db *database.DB) *NotificationRepository {
	return &NotificationRepository{db: db}
}

// Create inserts a notification row. Payload may be nil.
func (r *NotificationRepository) Create(ctx context.Context, n *models.Notification) error {
	query := `
		INSERT INTO notifications (user_id, type, title, body, payload)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at
	`
	var payload []byte
	if n.Payload != nil {
		var err error
		payload, err = json.Marshal(n.Payload)
		if err != nil {
			return fmt.Errorf("failed to marshal payload: %w", err)
		}
	}
	return r.db.Pool.QueryRow(ctx, query,
		n.UserID, n.Type, n.Title, n.Body, payload,
	).Scan(&n.ID, &n.CreatedAt)
}

// CreateOncePerDay inserts a notification row only if the user has no
// notification of the same type today (IST). Used by trigger checks that run
// on hot paths (e.g. near-limit quota warnings) — the conditional insert
// keeps the day-dedupe in one statement.
func (r *NotificationRepository) CreateOncePerDay(ctx context.Context, n *models.Notification, ist *time.Location) (bool, error) {
	now := time.Now().In(ist)
	dayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, ist)

	var payload []byte
	if n.Payload != nil {
		var err error
		payload, err = json.Marshal(n.Payload)
		if err != nil {
			return false, fmt.Errorf("failed to marshal payload: %w", err)
		}
	}

	query := `
		INSERT INTO notifications (user_id, type, title, body, payload)
		SELECT $1, $2, $3, $4, $5
		WHERE NOT EXISTS (
			SELECT 1 FROM notifications
			WHERE user_id = $1 AND type = $2
			AND created_at >= $6 AND created_at < $7
		)
		RETURNING id
	`
	err := r.db.Pool.QueryRow(ctx, query,
		n.UserID, n.Type, n.Title, n.Body, payload, dayStart, dayStart.Add(24*time.Hour),
	).Scan(&n.ID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil // already notified today
		}
		return false, fmt.Errorf("failed to create notification: %w", err)
	}
	return true, nil
}

// GetByID returns a single notification owned by userID.
func (r *NotificationRepository) GetByID(ctx context.Context, id, userID uuid.UUID) (*models.Notification, error) {
	query := `
		SELECT id, user_id, type, title, body, payload, created_at, read_at, pushed_at
		FROM notifications
		WHERE id = $1 AND user_id = $2
	`
	var n models.Notification
	err := r.db.Pool.QueryRow(ctx, query, id, userID).Scan(
		&n.ID, &n.UserID, &n.Type, &n.Title, &n.Body, &n.Payload, &n.CreatedAt, &n.ReadAt, &n.PushedAt,
	)
	if err != nil {
		return nil, err
	}
	return &n, nil
}

// List returns a user's notifications, newest first. unreadOnly filters to unreads.
func (r *NotificationRepository) List(ctx context.Context, userID uuid.UUID, limit, offset int, unreadOnly bool) ([]models.Notification, int, error) {
	where := "user_id = $1"
	if unreadOnly {
		where += " AND read_at IS NULL"
	}

	countQuery := fmt.Sprintf(`SELECT COUNT(*) FROM notifications WHERE %s`, where)
	total := 0
	if err := r.db.Pool.QueryRow(ctx, countQuery, userID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count notifications: %w", err)
	}

	query := fmt.Sprintf(`
		SELECT id, user_id, type, title, body, payload, created_at, read_at, pushed_at
		FROM notifications
		WHERE %s
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`, where)
	rows, err := r.db.Pool.Query(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list notifications: %w", err)
	}
	defer rows.Close()

	notifications := []models.Notification{}
	for rows.Next() {
		var n models.Notification
		if err := rows.Scan(
			&n.ID, &n.UserID, &n.Type, &n.Title, &n.Body, &n.Payload, &n.CreatedAt, &n.ReadAt, &n.PushedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("failed to scan notification: %w", err)
		}
		notifications = append(notifications, n)
	}
	return notifications, total, nil
}

func (r *NotificationRepository) CountUnread(ctx context.Context, userID uuid.UUID) (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM notifications WHERE user_id = $1 AND read_at IS NULL`
	err := r.db.Pool.QueryRow(ctx, query, userID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count unread notifications: %w", err)
	}
	return count, nil
}

// MarkRead marks a notification read for its owner. Returns rows affected (0 = not found).
func (r *NotificationRepository) MarkRead(ctx context.Context, id, userID uuid.UUID) (int64, error) {
	query := `UPDATE notifications SET read_at = NOW() WHERE id = $1 AND user_id = $2 AND read_at IS NULL`
	result, err := r.db.Pool.Exec(ctx, query, id, userID)
	if err != nil {
		return 0, fmt.Errorf("failed to mark notification read: %w", err)
	}
	return result.RowsAffected(), nil
}

func (r *NotificationRepository) MarkAllRead(ctx context.Context, userID uuid.UUID) error {
	query := `UPDATE notifications SET read_at = NOW() WHERE user_id = $1 AND read_at IS NULL`
	_, err := r.db.Pool.Exec(ctx, query, userID)
	return err
}

// HasTypeToday reports whether the user already has a notification of type
// created today (IST). Used by schedulers for idempotent daily delivery.
func (r *NotificationRepository) HasTypeToday(ctx context.Context, userID uuid.UUID, ntype string, ist *time.Location) (bool, error) {
	var exists bool
	query := `
		SELECT EXISTS (
			SELECT 1 FROM notifications
			WHERE user_id = $1 AND type = $2
			AND created_at >= $3 AND created_at < $4
		)
	`
	now := time.Now().In(ist)
	dayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, ist)
	err := r.db.Pool.QueryRow(ctx, query, userID, ntype, dayStart, dayStart.Add(24*time.Hour)).Scan(&exists)
	return exists, err
}

// CountPushedToday counts the user's notifications that carried an actual OS
// push today (IST) for push-capable trigger types. quota_reset is a single
// scheduled push (exempt); test rows (quota_warning) are diagnostic and
// excluded, so running tests never consumes the daily budget.
func (r *NotificationRepository) CountPushedToday(ctx context.Context, userID uuid.UUID, ist *time.Location) (int, error) {
	var count int
	query := `
		SELECT COUNT(*) FROM notifications
		WHERE user_id = $1 AND pushed_at IS NOT NULL
		AND type IN ('watchlist_match', 'export_ready')
		AND pushed_at >= $2
	`
	now := time.Now().In(ist)
	dayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, ist)
	err := r.db.Pool.QueryRow(ctx, query, userID, dayStart).Scan(&count)
	return count, err
}

// MarkPushed stamps the OS-push delivery time on a notification row.
func (r *NotificationRepository) MarkPushed(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE notifications SET pushed_at = NOW() WHERE id = $1`
	_, err := r.db.Pool.Exec(ctx, query, id)
	return err
}
