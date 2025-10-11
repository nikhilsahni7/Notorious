package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"notorious-backend/internal/database"
	"notorious-backend/internal/models"
)

type UserRequestRepository struct {
	db *database.DB
}

func NewUserRequestRepository(db *database.DB) *UserRequestRepository {
	return &UserRequestRepository{db: db}
}

func (r *UserRequestRepository) Create(ctx context.Context, req *models.UserRequest) error {
	query := `
		INSERT INTO user_requests (email, name, phone, requested_searches_per_day)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at, status
	`
	
	return r.db.Pool.QueryRow(ctx, query,
		req.Email,
		req.Name,
		req.Phone,
		req.RequestedSearchesPerDay,
	).Scan(&req.ID, &req.CreatedAt, &req.Status)
}

func (r *UserRequestRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.UserRequest, error) {
	var req models.UserRequest
	query := `
		SELECT id, email, name, phone, requested_searches_per_day, status, created_at, admin_notes
		FROM user_requests
		WHERE id = $1
	`
	
	err := r.db.Pool.QueryRow(ctx, query, id).Scan(
		&req.ID,
		&req.Email,
		&req.Name,
		&req.Phone,
		&req.RequestedSearchesPerDay,
		&req.Status,
		&req.CreatedAt,
		&req.AdminNotes,
	)
	
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	
	return &req, err
}

func (r *UserRequestRepository) ListByStatus(ctx context.Context, status string, limit, offset int) ([]*models.UserRequest, error) {
	requests := make([]*models.UserRequest, 0)
	query := `
		SELECT id, email, name, phone, requested_searches_per_day, status, created_at, admin_notes
		FROM user_requests
		WHERE status = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`
	
	rows, err := r.db.Pool.Query(ctx, query, status, limit, offset)
	if err != nil {
		return requests, err
	}
	defer rows.Close()
	
	for rows.Next() {
		var req models.UserRequest
		if err := rows.Scan(
			&req.ID,
			&req.Email,
			&req.Name,
			&req.Phone,
			&req.RequestedSearchesPerDay,
			&req.Status,
			&req.CreatedAt,
			&req.AdminNotes,
		); err != nil {
			return requests, err
		}
		requests = append(requests, &req)
	}
	
	return requests, rows.Err()
}

func (r *UserRequestRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status string, adminNotes *string) error {
	query := `
		UPDATE user_requests
		SET status = $1, admin_notes = $2
		WHERE id = $3
	`
	_, err := r.db.Pool.Exec(ctx, query, status, adminNotes, id)
	return err
}

func (r *UserRequestRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM user_requests WHERE id = $1`
	_, err := r.db.Pool.Exec(ctx, query, id)
	return err
}

