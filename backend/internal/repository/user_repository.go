package repository

import (
	"context"
	"fmt"
	"time"

	"notorious-backend/internal/database"
	"notorious-backend/internal/models"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type UserRepository struct {
	db *database.DB
}

func NewUserRepository(db *database.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) Create(ctx context.Context, user *models.User) error {
	query := `
		INSERT INTO users (email, password_hash, name, phone, role, region, daily_search_limit, is_active)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, created_at, updated_at, searches_used_today, last_reset_date
	`

	return r.db.Pool.QueryRow(ctx, query,
		user.Email,
		user.PasswordHash,
		user.Name,
		user.Phone,
		user.Role,
		user.Region,
		user.DailySearchLimit,
		user.IsActive,
	).Scan(&user.ID, &user.CreatedAt, &user.UpdatedAt, &user.SearchesUsedToday, &user.LastResetDate)
}

func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	var user models.User
	query := `
		SELECT id, email, password_hash, name, phone, role, daily_search_limit,
		       searches_used_today, is_active, created_at, updated_at, last_reset_date,
		       COALESCE(last_search_query, '') as last_search_query,
		       COALESCE(region, 'pan-india') as region
		FROM users
		WHERE email = $1
	`

	err := r.db.Pool.QueryRow(ctx, query, email).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.Name,
		&user.Phone,
		&user.Role,
		&user.DailySearchLimit,
		&user.SearchesUsedToday,
		&user.IsActive,
		&user.CreatedAt,
		&user.UpdatedAt,
		&user.LastResetDate,
		&user.LastSearchQuery,
		&user.Region,
	)

	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("user not found")
	}

	return &user, err
}

func (r *UserRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	var user models.User
	query := `
		SELECT id, email, password_hash, name, phone, role, daily_search_limit,
		       searches_used_today, is_active, created_at, updated_at, last_reset_date,
		       COALESCE(last_search_query, '') as last_search_query,
		       COALESCE(region, 'pan-india') as region
		FROM users
		WHERE id = $1
	`

	err := r.db.Pool.QueryRow(ctx, query, id).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.Name,
		&user.Phone,
		&user.Role,
		&user.DailySearchLimit,
		&user.SearchesUsedToday,
		&user.IsActive,
		&user.CreatedAt,
		&user.UpdatedAt,
		&user.LastResetDate,
		&user.LastSearchQuery,
		&user.Region,
	)

	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("user not found")
	}

	return &user, err
}

func (r *UserRepository) Update(ctx context.Context, user *models.User) error {
	query := `
		UPDATE users
		SET name = $1, phone = $2, region = $3, daily_search_limit = $4, is_active = $5, updated_at = $6
		WHERE id = $7
	`

	user.UpdatedAt = time.Now()
	_, err := r.db.Pool.Exec(ctx, query,
		user.Name,
		user.Phone,
		user.Region,
		user.DailySearchLimit,
		user.IsActive,
		user.UpdatedAt,
		user.ID,
	)

	return err
}

func (r *UserRepository) UpdatePassword(ctx context.Context, userID uuid.UUID, passwordHash string) error {
	query := `UPDATE users SET password_hash = $1, updated_at = $2 WHERE id = $3`
	_, err := r.db.Pool.Exec(ctx, query, passwordHash, time.Now(), userID)
	return err
}

func (r *UserRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM users WHERE id = $1`
	_, err := r.db.Pool.Exec(ctx, query, id)
	return err
}

func (r *UserRepository) List(ctx context.Context, role string, limit, offset int) ([]*models.User, error) {
	users := make([]*models.User, 0)
	var query string
	var args []interface{}

	if role != "" {
		query = `
			SELECT id, email, password_hash, name, phone, role, daily_search_limit,
			       searches_used_today, is_active, created_at, updated_at, last_reset_date,
			       COALESCE(last_search_query, '') as last_search_query
			FROM users
			WHERE role = $1
			ORDER BY created_at DESC
			LIMIT $2 OFFSET $3
		`
		args = []interface{}{role, limit, offset}
	} else {
		query = `
			SELECT id, email, password_hash, name, phone, role, daily_search_limit,
			       searches_used_today, is_active, created_at, updated_at, last_reset_date,
			       COALESCE(last_search_query, '') as last_search_query
			FROM users
			ORDER BY created_at DESC
			LIMIT $1 OFFSET $2
		`
		args = []interface{}{limit, offset}
	}

	rows, err := r.db.Pool.Query(ctx, query, args...)
	if err != nil {
		return users, err
	}
	defer rows.Close()

	for rows.Next() {
		var user models.User
		if err := rows.Scan(
			&user.ID,
			&user.Email,
			&user.PasswordHash,
			&user.Name,
			&user.Phone,
			&user.Role,
			&user.DailySearchLimit,
			&user.SearchesUsedToday,
			&user.IsActive,
			&user.CreatedAt,
			&user.UpdatedAt,
			&user.LastResetDate,
			&user.LastSearchQuery,
		); err != nil {
			return users, err
		}
		users = append(users, &user)
	}

	return users, rows.Err()
}

func (r *UserRepository) IncrementSearchUsage(ctx context.Context, userID uuid.UUID) error {
	query := `
		UPDATE users
		SET searches_used_today = searches_used_today + 1
		WHERE id = $1
	`
	_, err := r.db.Pool.Exec(ctx, query, userID)
	return err
}

func (r *UserRepository) UpdateLastSearchQuery(ctx context.Context, userID uuid.UUID, query string) error {
	sql := `
		UPDATE users
		SET last_search_query = $1
		WHERE id = $2
	`
	_, err := r.db.Pool.Exec(ctx, sql, query, userID)
	return err
}

func (r *UserRepository) CheckAndResetDailyLimit(ctx context.Context, userID uuid.UUID, istLocation *time.Location) (*models.User, error) {
	user, err := r.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	currentDateIST := time.Now().In(istLocation).Format("2006-01-02")
	lastResetDateIST := user.LastResetDate.In(istLocation).Format("2006-01-02")

	if currentDateIST != lastResetDateIST {
		query := `
			UPDATE users
			SET searches_used_today = 0, last_reset_date = $1
			WHERE id = $2
			RETURNING searches_used_today, last_reset_date
		`

		err := r.db.Pool.QueryRow(ctx, query, time.Now(), userID).Scan(
			&user.SearchesUsedToday,
			&user.LastResetDate,
		)
		if err != nil {
			return nil, err
		}
	}

	return user, nil
}

func (r *UserRepository) ResetAllDailyLimits(ctx context.Context) error {
	query := `
		UPDATE users
		SET searches_used_today = 0, last_reset_date = CURRENT_DATE
		WHERE last_reset_date < CURRENT_DATE
	`
	_, err := r.db.Pool.Exec(ctx, query)
	return err
}
