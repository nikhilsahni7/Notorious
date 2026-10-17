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
	Region            string    `json:"region" db:"region"` // "pan-india" or "delhi-ncr"
	DeviceLimit       int       `json:"device_limit" db:"device_limit"`
	DailySearchLimit  int       `json:"daily_search_limit" db:"daily_search_limit"`
	SearchesUsedToday int       `json:"searches_used_today" db:"searches_used_today"`
	IsActive          bool      `json:"is_active" db:"is_active"`
	CreatedAt         time.Time `json:"created_at" db:"created_at"`
	UpdatedAt         time.Time `json:"updated_at" db:"updated_at"`
	LastResetDate     time.Time `json:"last_reset_date" db:"last_reset_date"`
	LastSearchQuery   string    `json:"last_search_query" db:"last_search_query"`
	// NotificationPrefs stored as JSONB (migration 015), e.g. {"quota_reset": true}
	NotificationPrefs map[string]bool `json:"notification_prefs,omitempty" db:"notification_prefs"`
}

// EffectiveSearchesUsedToday reports the searches_used_today that should be
// shown for the current day in loc. A counter written on an earlier day reads
// as 0 because it will be reset by the next search or the midnight scheduler.
//
// It is read-only: callers that must not mutate quota (login, suggestions) use
// it instead of the previous write-on-read reset, which persisted a UTC date
// and re-reset the counter on every call between 00:00 and 05:30 IST.
func (u *User) EffectiveSearchesUsedToday(now time.Time, loc *time.Location) int {
	if u.LastResetDate.In(loc).Format("2006-01-02") != now.In(loc).Format("2006-01-02") {
		return 0
	}
	return u.SearchesUsedToday
}

type UserRequest struct {
	ID                      uuid.UUID  `json:"id" db:"id"`
	Email                   string     `json:"email" db:"email"`
	Name                    string     `json:"name" db:"name"`
	Phone                   string     `json:"phone" db:"phone"`
	RequestedSearchesPerDay int        `json:"requested_searches_per_day" db:"requested_searches_per_day"`
	Status                  string     `json:"status" db:"status"`
	CreatedAt               time.Time  `json:"created_at" db:"created_at"`
	AdminNote               *string    `json:"admin_note,omitempty" db:"admin_note"`
	ReviewedBy              *uuid.UUID `json:"reviewed_by,omitempty" db:"reviewed_by"`
	ReviewedAt              *time.Time `json:"reviewed_at,omitempty" db:"reviewed_at"`
	AdminNotes              *string    `json:"admin_notes,omitempty" db:"admin_notes"` // Deprecated, use AdminNote
	// Metadata fields for tracking signup requests
	IPAddress  *string `json:"ip_address,omitempty" db:"ip_address"`
	Country    *string `json:"country,omitempty" db:"country"`
	City       *string `json:"city,omitempty" db:"city"`
	DeviceType *string `json:"device_type,omitempty" db:"device_type"`
	Browser    *string `json:"browser,omitempty" db:"browser"`
	OS         *string `json:"os,omitempty" db:"os"`
	UserAgent  *string `json:"user_agent,omitempty" db:"user_agent"`
}

type SearchHistory struct {
	ID           uuid.UUID   `json:"id" db:"id"`
	UserID       uuid.UUID   `json:"user_id" db:"user_id"`
	Query        string      `json:"query" db:"query"`
	TotalResults int         `json:"total_results" db:"total_results"`
	TopResults   interface{} `json:"top_results" db:"top_results"`
	SearchedAt   time.Time   `json:"searched_at" db:"searched_at"`
	IsRefinement bool        `json:"is_refinement" db:"is_refinement"`
	BaseSearchID *uuid.UUID  `json:"base_search_id,omitempty" db:"base_search_id"`
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
