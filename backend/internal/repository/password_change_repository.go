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
		INSERT INTO password_change_requests (user_id, reason, status, admin_notes, new_password_hash, processed_by)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at, updated_at
	`
	return r.db.Pool.QueryRow(ctx, query,
		req.UserID,
		req.Reason,
		req.Status,
		req.AdminNotes,
		req.NewPasswordHash,
		req.ProcessedBy,
	).Scan(
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
	var query string
	var args []any

	if status == "self-changed" {
		query = `
			SELECT 
				pcr.id, pcr.user_id, pcr.reason, pcr.status, pcr.admin_notes,
				pcr.created_at, pcr.updated_at, pcr.processed_by,
				u.email, u.name
			FROM password_change_requests pcr
			JOIN users u ON pcr.user_id = u.id
			WHERE pcr.reason = 'Self-service password change'
			ORDER BY pcr.created_at DESC
			LIMIT $1 OFFSET $2
		`
		args = []any{limit, offset}
	} else {
		query = `
			SELECT 
				pcr.id, pcr.user_id, pcr.reason, pcr.status, pcr.admin_notes,
				pcr.created_at, pcr.updated_at, pcr.processed_by,
				u.email, u.name
			FROM password_change_requests pcr
			JOIN users u ON pcr.user_id = u.id
			WHERE pcr.status = $1 AND pcr.reason != 'Self-service password change'
			ORDER BY pcr.created_at DESC
			LIMIT $2 OFFSET $3
		`
		args = []any{status, limit, offset}
	}

	rows, err := r.db.Pool.Query(ctx, query, args...)
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

func (r *PasswordChangeRepository) CountByStatus(ctx context.Context, status string) (int, error) {
	var count int
	var query string
	var err error

	if status == "self-changed" {
		query = `SELECT COUNT(*) FROM password_change_requests WHERE reason = 'Self-service password change'`
		err = r.db.Pool.QueryRow(ctx, query).Scan(&count)
	} else {
		query = `SELECT COUNT(*) FROM password_change_requests WHERE status = $1 AND reason != 'Self-service password change'`
		err = r.db.Pool.QueryRow(ctx, query, status).Scan(&count)
	}
	return count, err
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
