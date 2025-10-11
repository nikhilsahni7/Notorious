package models

import (
	"time"

	"github.com/google/uuid"
)

type UserMetadata struct {
	ID             uuid.UUID `json:"id" db:"id"`
	UserID         uuid.UUID `json:"user_id" db:"user_id"`
	IPAddress      *string   `json:"ip_address" db:"ip_address"`
	Country        *string   `json:"country" db:"country"`
	CountryCode    *string   `json:"country_code" db:"country_code"`
	City           *string   `json:"city" db:"city"`
	Latitude       *float64  `json:"latitude,omitempty" db:"latitude"`
	Longitude      *float64  `json:"longitude,omitempty" db:"longitude"`
	Timezone       *string   `json:"timezone,omitempty" db:"timezone"`
	DeviceType     *string   `json:"device_type" db:"device_type"`
	Browser        *string   `json:"browser" db:"browser"`
	BrowserVersion *string   `json:"browser_version,omitempty" db:"browser_version"`
	OS             *string   `json:"os" db:"os"`
	OSVersion      *string   `json:"os_version,omitempty" db:"os_version"`
	UserAgent      *string   `json:"user_agent" db:"user_agent"`
	CreatedAt      time.Time `json:"created_at" db:"created_at"`
}

type AdminSession struct {
	ID             uuid.UUID `json:"id" db:"id"`
	AdminID        uuid.UUID `json:"admin_id" db:"admin_id"`
	TokenHash      string    `json:"-" db:"token_hash"`
	IPAddress      *string   `json:"ip_address" db:"ip_address"`
	Country        *string   `json:"country" db:"country"`
	CountryCode    *string   `json:"country_code" db:"country_code"`
	City           *string   `json:"city" db:"city"`
	Latitude       *float64  `json:"latitude,omitempty" db:"latitude"`
	Longitude      *float64  `json:"longitude,omitempty" db:"longitude"`
	Timezone       *string   `json:"timezone,omitempty" db:"timezone"`
	DeviceType     *string   `json:"device_type" db:"device_type"`
	Browser        *string   `json:"browser" db:"browser"`
	BrowserVersion *string   `json:"browser_version,omitempty" db:"browser_version"`
	OS             *string   `json:"os" db:"os"`
	OSVersion      *string   `json:"os_version,omitempty" db:"os_version"`
	UserAgent      *string   `json:"user_agent" db:"user_agent"`
	IsActive       bool      `json:"is_active" db:"is_active"`
	CreatedAt      time.Time `json:"created_at" db:"created_at"`
	LastUsedAt     time.Time `json:"last_used_at" db:"last_used_at"`
	ExpiresAt      time.Time `json:"expires_at" db:"expires_at"`
}

type AdminSessionWithUser struct {
	AdminSession
	AdminEmail string `json:"admin_email" db:"admin_email"`
	AdminName  string `json:"admin_name" db:"admin_name"`
}

type UserWithMetadata struct {
	User
	Metadata *UserMetadata `json:"metadata,omitempty"`
}
