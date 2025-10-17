package repository

import (
	"context"
	"encoding/json"

	"notorious-backend/internal/database"
	"notorious-backend/internal/models"

	"github.com/google/uuid"
)

type SearchHistoryRepository struct {
	db *database.DB
}

func NewSearchHistoryRepository(db *database.DB) *SearchHistoryRepository {
	return &SearchHistoryRepository{db: db}
}

func (r *SearchHistoryRepository) Create(ctx context.Context, history *models.SearchHistory) error {
	topResultsJSON, err := json.Marshal(history.TopResults)
	if err != nil {
		return err
	}

	query := `
		INSERT INTO search_history (user_id, query, total_results, top_results)
		VALUES ($1, $2, $3, $4)
		RETURNING id, searched_at
	`

	return r.db.Pool.QueryRow(ctx, query,
		history.UserID,
		history.Query,
		history.TotalResults,
		topResultsJSON,
	).Scan(&history.ID, &history.SearchedAt)
}

func (r *SearchHistoryRepository) GetByUserID(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*models.SearchHistory, error) {
	histories := make([]*models.SearchHistory, 0)
	query := `
		SELECT id, user_id, query, total_results, top_results, searched_at
		FROM search_history
		WHERE user_id = $1
		ORDER BY searched_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.Pool.Query(ctx, query, userID, limit, offset)
	if err != nil {
		return histories, err
	}
	defer rows.Close()

	for rows.Next() {
		var history models.SearchHistory
		var topResultsJSON []byte

		if err := rows.Scan(
			&history.ID,
			&history.UserID,
			&history.Query,
			&history.TotalResults,
			&topResultsJSON,
			&history.SearchedAt,
		); err != nil {
			return histories, err
		}

		if err := json.Unmarshal(topResultsJSON, &history.TopResults); err != nil {
			return histories, err
		}

		histories = append(histories, &history)
	}

	return histories, rows.Err()
}

func (r *SearchHistoryRepository) GetAllWithUsers(ctx context.Context, limit, offset int) ([]*models.SearchHistoryWithUser, error) {
	histories := make([]*models.SearchHistoryWithUser, 0)
	query := `
		SELECT
			sh.id, sh.user_id, sh.query, sh.total_results, sh.top_results, sh.searched_at,
			u.email, u.name
		FROM search_history sh
		JOIN users u ON sh.user_id = u.id
		ORDER BY sh.searched_at DESC
		LIMIT $1 OFFSET $2
	`

	rows, err := r.db.Pool.Query(ctx, query, limit, offset)
	if err != nil {
		return histories, err
	}
	defer rows.Close()

	for rows.Next() {
		var history models.SearchHistoryWithUser
		var topResultsJSON []byte

		if err := rows.Scan(
			&history.ID,
			&history.UserID,
			&history.Query,
			&history.TotalResults,
			&topResultsJSON,
			&history.SearchedAt,
			&history.UserEmail,
			&history.UserName,
		); err != nil {
			return histories, err
		}

		if err := json.Unmarshal(topResultsJSON, &history.TopResults); err != nil {
			return histories, err
		}

		histories = append(histories, &history)
	}

	return histories, rows.Err()
}

func (r *SearchHistoryRepository) CountByUserID(ctx context.Context, userID uuid.UUID) (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM search_history WHERE user_id = $1`
	err := r.db.Pool.QueryRow(ctx, query, userID).Scan(&count)
	return count, err
}

// GetTodaySearches retrieves all searches from midnight to now in IST
func (r *SearchHistoryRepository) GetTodaySearches(ctx context.Context) ([]*models.SearchHistory, error) {
	histories := make([]*models.SearchHistory, 0)

	// IST is UTC+5:30
	query := `
		SELECT id, user_id, query, total_results, top_results, searched_at
		FROM search_history
		WHERE searched_at >= (CURRENT_DATE AT TIME ZONE 'Asia/Kolkata')
		  AND searched_at < (CURRENT_DATE AT TIME ZONE 'Asia/Kolkata' + INTERVAL '1 day')
		ORDER BY searched_at ASC
	`

	rows, err := r.db.Pool.Query(ctx, query)
	if err != nil {
		return histories, err
	}
	defer rows.Close()

	for rows.Next() {
		var history models.SearchHistory
		var topResultsJSON []byte

		if err := rows.Scan(
			&history.ID,
			&history.UserID,
			&history.Query,
			&history.TotalResults,
			&topResultsJSON,
			&history.SearchedAt,
		); err != nil {
			return histories, err
		}

		if err := json.Unmarshal(topResultsJSON, &history.TopResults); err != nil {
			return histories, err
		}

		histories = append(histories, &history)
	}

	return histories, rows.Err()
}
