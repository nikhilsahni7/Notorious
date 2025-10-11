package repository

import (
	"context"

	"notorious-backend/internal/database"
	"notorious-backend/internal/models"

	"github.com/google/uuid"
)

type MetadataRepository struct {
	db *database.DB
}

func NewMetadataRepository(db *database.DB) *MetadataRepository {
	return &MetadataRepository{db: db}
}

// CreateUserMetadata stores user signup metadata
func (r *MetadataRepository) CreateUserMetadata(ctx context.Context, metadata *models.UserMetadata) error {
	query := `
		INSERT INTO user_metadata (
			user_id, ip_address, country, country_code, city, latitude, longitude, timezone,
			device_type, browser, browser_version, os, os_version, user_agent
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
		RETURNING id, created_at
	`
	return r.db.Pool.QueryRow(ctx, query,
		metadata.UserID, metadata.IPAddress, metadata.Country, metadata.CountryCode,
		metadata.City, metadata.Latitude, metadata.Longitude, metadata.Timezone,
		metadata.DeviceType, metadata.Browser, metadata.BrowserVersion,
		metadata.OS, metadata.OSVersion, metadata.UserAgent,
	).Scan(&metadata.ID, &metadata.CreatedAt)
}

// GetUserMetadata retrieves metadata for a user
func (r *MetadataRepository) GetUserMetadata(ctx context.Context, userID uuid.UUID) (*models.UserMetadata, error) {
	var metadata models.UserMetadata
	query := `
		SELECT id, user_id, ip_address, country, country_code, city, latitude, longitude, timezone,
		       device_type, browser, browser_version, os, os_version, user_agent, created_at
		FROM user_metadata
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT 1
	`
	err := r.db.Pool.QueryRow(ctx, query, userID).Scan(
		&metadata.ID, &metadata.UserID, &metadata.IPAddress, &metadata.Country,
		&metadata.CountryCode, &metadata.City, &metadata.Latitude, &metadata.Longitude,
		&metadata.Timezone, &metadata.DeviceType, &metadata.Browser, &metadata.BrowserVersion,
		&metadata.OS, &metadata.OSVersion, &metadata.UserAgent, &metadata.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &metadata, nil
}
