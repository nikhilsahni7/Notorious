package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"notorious-backend/internal/database"
	"notorious-backend/internal/models"
)

type PasswordChangeRepository struct {
	db *database.DB
}

func NewPasswordChangeRepository(db *database.DB) *PasswordChangeRepository {
	return &PasswordChangeRepository{db: db}
}

func (r *PasswordChangeRepository) Create(ctx context.Context, req *models.PasswordChangeRequest) error {
	query := `
		INSERT INTO password_change_requests (user_id, reason, status)
		VALUES ($1, $2, $3)
		RETURNING id, created_at, updated_at
	`
	return r.db.Pool.QueryRow(ctx, query, req.UserID, req.Reason, "pending").Scan(
		&req.ID, &req.CreatedAt, &req.UpdatedAt,
	)
}

func (r *PasswordChangeRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.PasswordChangeRequest, error) {
	var req models.PasswordChangeRequest
	query := `
		SELECT id, user_id, reason, status, admin_notes, new_password_hash, 
		       created_at, updated_at, processed_by
		FROM password_change_requests
		WHERE id = $1
	`
	err := r.db.Pool.QueryRow(ctx, query, id).Scan(
		&req.ID, &req.UserID, &req.Reason, &req.Status, &req.AdminNotes,
		&req.NewPasswordHash, &req.CreatedAt, &req.UpdatedAt, &req.ProcessedBy,
	)
	return &req, err
}

func (r *PasswordChangeRepository) GetByUserID(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*models.PasswordChangeRequest, error) {
	requests := make([]*models.PasswordChangeRequest, 0)
	query := `
		SELECT id, user_id, reason, status, admin_notes, created_at, updated_at, processed_by
		FROM password_change_requests
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`
	rows, err := r.db.Pool.Query(ctx, query, userID, limit, offset)
	if err != nil {
		return requests, err
	}
	defer rows.Close()

	for rows.Next() {
		var req models.PasswordChangeRequest
		if err := rows.Scan(
			&req.ID, &req.UserID, &req.Reason, &req.Status, &req.AdminNotes,
			&req.CreatedAt, &req.UpdatedAt, &req.ProcessedBy,
		); err != nil {
			return requests, err
		}
		requests = append(requests, &req)
	}
	return requests, rows.Err()
}

func (r *PasswordChangeRepository) ListByStatus(ctx context.Context, status string, limit, offset int) ([]*models.PasswordChangeRequestWithUser, error) {
	requests := make([]*models.PasswordChangeRequestWithUser, 0)
	query := `
		SELECT 
			pcr.id, pcr.user_id, pcr.reason, pcr.status, pcr.admin_notes,
			pcr.created_at, pcr.updated_at, pcr.processed_by,
			u.email, u.name
		FROM password_change_requests pcr
		JOIN users u ON pcr.user_id = u.id
		WHERE pcr.status = $1
		ORDER BY pcr.created_at DESC
		LIMIT $2 OFFSET $3
	`
	rows, err := r.db.Pool.Query(ctx, query, status, limit, offset)
	if err != nil {
		return requests, err
	}
	defer rows.Close()

	for rows.Next() {
		var req models.PasswordChangeRequestWithUser
		if err := rows.Scan(
			&req.ID, &req.UserID, &req.Reason, &req.Status, &req.AdminNotes,
			&req.CreatedAt, &req.UpdatedAt, &req.ProcessedBy,
			&req.UserEmail, &req.UserName,
		); err != nil {
			return requests, err
		}
		requests = append(requests, &req)
	}
	return requests, rows.Err()
}

func (r *PasswordChangeRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status string, adminNotes *string, newPasswordHash *string, processedBy uuid.UUID) error {
	query := `
		UPDATE password_change_requests
		SET status = $1, admin_notes = $2, new_password_hash = $3, processed_by = $4, updated_at = $5
		WHERE id = $6
	`
	_, err := r.db.Pool.Exec(ctx, query, status, adminNotes, newPasswordHash, processedBy, time.Now(), id)
	return err
}

