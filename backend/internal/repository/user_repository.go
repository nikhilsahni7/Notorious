package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"notorious-backend/internal/database"
	"notorious-backend/internal/models"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// ErrUserNotFound is returned when a user lookup by ID or email finds no matching row.
var ErrUserNotFound = errors.New("user not found")

type UserRepository struct {
	db *database.DB
}

func NewUserRepository(db *database.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) Create(ctx context.Context, user *models.User) error {
	normalizedEmail := strings.ToLower(strings.TrimSpace(user.Email))
	normalizedPhone := strings.TrimSpace(user.Phone)

	query := `
		INSERT INTO users (email, password_hash, name, phone, role, region, daily_search_limit, is_active, device_limit)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, created_at, updated_at, searches_used_today, last_reset_date
	`

	return r.db.Pool.QueryRow(ctx, query,
		normalizedEmail,
		user.PasswordHash,
		user.Name,
		normalizedPhone,
		user.Role,
		user.Region,
		user.DailySearchLimit,
		user.IsActive,
		user.DeviceLimit,
	).Scan(&user.ID, &user.CreatedAt, &user.UpdatedAt, &user.SearchesUsedToday, &user.LastResetDate)
}

func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	var user models.User
	query := `
		SELECT id, email, password_hash, name, COALESCE(phone, '') as phone, role, daily_search_limit,
		       searches_used_today, is_active, created_at, updated_at, last_reset_date,
		       COALESCE(last_search_query, '') as last_search_query,
		       COALESCE(region, 'pan-india') as region, device_limit
		FROM users
		WHERE lower(trim(email)) = lower(trim($1))
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
		&user.DeviceLimit,
	)

	if err == pgx.ErrNoRows {
		return nil, ErrUserNotFound
	}

	return &user, err
}

func (r *UserRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	var user models.User
	query := `
		SELECT id, email, password_hash, name, COALESCE(phone, '') as phone, role, daily_search_limit,
		       searches_used_today, is_active, created_at, updated_at, last_reset_date,
		       COALESCE(last_search_query, '') as last_search_query,
		       COALESCE(region, 'pan-india') as region, device_limit
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
		&user.DeviceLimit,
	)

	if err == pgx.ErrNoRows {
		return nil, ErrUserNotFound
	}

	return &user, err
}

func (r *UserRepository) Update(ctx context.Context, user *models.User) error {
	query := `
		UPDATE users
		SET name = $1, phone = $2, region = $3, daily_search_limit = $4, is_active = $5, updated_at = $6, device_limit = $7
		WHERE id = $8
	`

	user.UpdatedAt = time.Now()
	_, err := r.db.Pool.Exec(ctx, query,
		user.Name,
		user.Phone,
		user.Region,
		user.DailySearchLimit,
		user.IsActive,
		user.UpdatedAt,
		user.DeviceLimit,
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

func (r *UserRepository) buildListFilters(role, region, search string) (string, []interface{}) {
	conditions := make([]string, 0)
	args := make([]interface{}, 0)
	argIdx := 1

	if role != "" {
		conditions = append(conditions, fmt.Sprintf("role = $%d", argIdx))
		args = append(args, role)
		argIdx++
	}

	if region != "" && region != "all" {
		conditions = append(conditions, fmt.Sprintf("COALESCE(region, 'pan-india') = $%d", argIdx))
		args = append(args, region)
		argIdx++
	}

	if search != "" {
		pattern := "%" + search + "%"
		conditions = append(conditions, fmt.Sprintf(
			"(name ILIKE $%d OR email ILIKE $%d OR COALESCE(phone, '') ILIKE $%d)",
			argIdx, argIdx, argIdx,
		))
		args = append(args, pattern)
		argIdx++
	}

	where := ""
	if len(conditions) > 0 {
		where = "WHERE " + strings.Join(conditions, " AND ")
	}

	return where, args
}

func (r *UserRepository) List(ctx context.Context, role, region, search string, limit, offset int) ([]*models.User, int, error) {
	users := make([]*models.User, 0)
	where, args := r.buildListFilters(role, region, search)

	countQuery := "SELECT COUNT(*) FROM users " + where
	var total int
	if err := r.db.Pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return users, 0, err
	}

	argIdx := len(args) + 1
	query := fmt.Sprintf(`
		SELECT id, email, password_hash, name, COALESCE(phone, '') as phone, role, daily_search_limit,
		       searches_used_today, is_active, created_at, updated_at, last_reset_date,
		       COALESCE(last_search_query, '') as last_search_query,
		       COALESCE(region, 'pan-india') as region,
		       device_limit
		FROM users
		%s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d
	`, where, argIdx, argIdx+1)

	listArgs := append(append([]interface{}{}, args...), limit, offset)
	rows, err := r.db.Pool.Query(ctx, query, listArgs...)
	if err != nil {
		return users, 0, err
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
			&user.Region,
			&user.DeviceLimit,
		); err != nil {
			return users, 0, err
		}
		users = append(users, &user)
	}

	return users, total, rows.Err()
}

// CountByRegion returns user counts grouped by region, optionally filtered by search/role.
func (r *UserRepository) CountByRegion(ctx context.Context, role, search string) (map[string]int, error) {
	where, args := r.buildListFilters(role, "", search)

	query := `
		SELECT COALESCE(region, 'pan-india') as region, COUNT(*)
		FROM users
		` + where + `
		GROUP BY COALESCE(region, 'pan-india')
	`

	rows, err := r.db.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	counts := map[string]int{
		"all":       0,
		"pan-india": 0,
		"delhi-ncr": 0,
	}

	for rows.Next() {
		var region string
		var count int
		if err := rows.Scan(&region, &count); err != nil {
			return nil, err
		}
		counts[region] = count
		counts["all"] += count
	}

	return counts, rows.Err()
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

// IncrementSearchAndSetLastQuery combines IncrementSearchUsage and UpdateLastSearchQuery
// into a single SQL statement, eliminating one DB roundtrip per search request.
func (r *UserRepository) IncrementSearchAndSetLastQuery(ctx context.Context, userID uuid.UUID, query string) error {
	sql := `
		UPDATE users
		SET searches_used_today = searches_used_today + 1,
		    last_search_query = $1
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

// BulkUpdateStatus updates the is_active status for multiple users at once
func (r *UserRepository) BulkUpdateStatus(ctx context.Context, userIDs []uuid.UUID, isActive bool) (int64, error) {
	query := `
		UPDATE users
		SET is_active = $1, updated_at = $2
		WHERE id = ANY($3)
	`
	result, err := r.db.Pool.Exec(ctx, query, isActive, time.Now(), userIDs)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected(), nil
}

func (r *UserRepository) CountTotal(ctx context.Context) (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM users`
	err := r.db.Pool.QueryRow(ctx, query).Scan(&count)
	return count, err
}

func (r *UserRepository) CountActive(ctx context.Context) (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM users WHERE is_active = true`
	err := r.db.Pool.QueryRow(ctx, query).Scan(&count)
	return count, err
}
