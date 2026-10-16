package repository

import (
	"context"
	"fmt"

	"notorious-backend/internal/database"
	"notorious-backend/internal/models"

	"github.com/google/uuid"
)

type PushSubscriptionRepository struct {
	db *database.DB
}

func NewPushSubscriptionRepository(db *database.DB) *PushSubscriptionRepository {
	return &PushSubscriptionRepository{db: db}
}

// Upsert registers a browser endpoint for a user. Re-subscribing with the same
// endpoint refreshes keys and device type instead of violating the UNIQUE constraint.
func (r *PushSubscriptionRepository) Upsert(ctx context.Context, sub *models.PushSubscription) error {
	query := `
		INSERT INTO push_subscriptions (user_id, endpoint, p256dh, auth, device_type)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (endpoint) DO UPDATE
		SET user_id = EXCLUDED.user_id,
		    p256dh = EXCLUDED.p256dh,
		    auth = EXCLUDED.auth,
		    device_type = EXCLUDED.device_type,
		    created_at = NOW()
		RETURNING id
	`
	return r.db.Pool.QueryRow(ctx, query,
		sub.UserID, sub.Endpoint, sub.P256dh, sub.Auth, sub.DeviceType,
	).Scan(&sub.ID)
}

// ListByUser returns all of a user's push subscriptions.
func (r *PushSubscriptionRepository) ListByUser(ctx context.Context, userID uuid.UUID) ([]models.PushSubscription, error) {
	query := `SELECT id, user_id, endpoint, p256dh, auth, COALESCE(device_type, ''), created_at FROM push_subscriptions WHERE user_id = $1`
	rows, err := r.db.Pool.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list push subscriptions: %w", err)
	}
	defer rows.Close()

	subs := []models.PushSubscription{}
	for rows.Next() {
		var s models.PushSubscription
		if err := rows.Scan(&s.ID, &s.UserID, &s.Endpoint, &s.P256dh, &s.Auth, &s.DeviceType, &s.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan push subscription: %w", err)
		}
		subs = append(subs, s)
	}
	return subs, nil
}

// DeleteByEndpoint removes a subscription. Ownership is enforced when userID is non-zero.
func (r *PushSubscriptionRepository) DeleteByEndpoint(ctx context.Context, endpoint string, userID *uuid.UUID) error {
	if userID != nil {
		_, err := r.db.Pool.Exec(ctx, `DELETE FROM push_subscriptions WHERE endpoint = $1 AND user_id = $2`, endpoint, *userID)
		return err
	}
	_, err := r.db.Pool.Exec(ctx, `DELETE FROM push_subscriptions WHERE endpoint = $1`, endpoint)
	return err
}
