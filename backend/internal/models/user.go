package models

import (
	"time"

	"github.com/google/uuid"
)

type Role string

const (
	RoleAdmin Role = "admin"
	RoleUser  Role = "user"
)

type User struct {
	ID                uuid.UUID `json:"id" db:"id"`
	Email             string    `json:"email" db:"email"`
	PasswordHash      string    `json:"-" db:"password_hash"`
	Name              string    `json:"name" db:"name"`
	Phone             string    `json:"phone" db:"phone"`
	Role              Role      `json:"role" db:"role"`
	DailySearchLimit  int       `json:"daily_search_limit" db:"daily_search_limit"`
	SearchesUsedToday int       `json:"searches_used_today" db:"searches_used_today"`
	IsActive          bool      `json:"is_active" db:"is_active"`
	CreatedAt         time.Time `json:"created_at" db:"created_at"`
	UpdatedAt         time.Time `json:"updated_at" db:"updated_at"`
	LastResetDate     time.Time `json:"last_reset_date" db:"last_reset_date"`
	LastSearchQuery   string    `json:"last_search_query" db:"last_search_query"`
}

type UserRequest struct {
	ID                      uuid.UUID `json:"id" db:"id"`
	Email                   string    `json:"email" db:"email"`
	Name                    string    `json:"name" db:"name"`
	Phone                   string    `json:"phone" db:"phone"`
	RequestedSearchesPerDay int       `json:"requested_searches_per_day" db:"requested_searches_per_day"`
	Status                  string    `json:"status" db:"status"`
	CreatedAt               time.Time `json:"created_at" db:"created_at"`
	AdminNotes              *string   `json:"admin_notes,omitempty" db:"admin_notes"`
	// Metadata fields for tracking signup requests
	IPAddress   *string `json:"ip_address,omitempty" db:"ip_address"`
	Country     *string `json:"country,omitempty" db:"country"`
	City        *string `json:"city,omitempty" db:"city"`
	DeviceType  *string `json:"device_type,omitempty" db:"device_type"`
	Browser     *string `json:"browser,omitempty" db:"browser"`
	OS          *string `json:"os,omitempty" db:"os"`
	UserAgent   *string `json:"user_agent,omitempty" db:"user_agent"`
}

type SearchHistory struct {
	ID           uuid.UUID   `json:"id" db:"id"`
	UserID       uuid.UUID   `json:"user_id" db:"user_id"`
	Query        string      `json:"query" db:"query"`
	TotalResults int         `json:"total_results" db:"total_results"`
	TopResults   interface{} `json:"top_results" db:"top_results"`
	SearchedAt   time.Time   `json:"searched_at" db:"searched_at"`
}

type SearchHistoryWithUser struct {
	SearchHistory
	UserEmail string `json:"user_email" db:"user_email"`
	UserName  string `json:"user_name" db:"user_name"`
}

type PasswordChangeRequest struct {
	ID              uuid.UUID  `json:"id" db:"id"`
	UserID          uuid.UUID  `json:"user_id" db:"user_id"`
	Reason          string     `json:"reason" db:"reason"`
	Status          string     `json:"status" db:"status"`
	AdminNotes      *string    `json:"admin_notes,omitempty" db:"admin_notes"`
	NewPasswordHash *string    `json:"-" db:"new_password_hash"`
	CreatedAt       time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at" db:"updated_at"`
	ProcessedBy     *uuid.UUID `json:"processed_by,omitempty" db:"processed_by"`
}

type PasswordChangeRequestWithUser struct {
	PasswordChangeRequest
	UserEmail string `json:"user_email" db:"user_email"`
	UserName  string `json:"user_name" db:"user_name"`
}
