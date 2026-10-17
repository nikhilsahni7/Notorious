package models

import (
	"time"

	"github.com/google/uuid"
)

// Notification types (mirrors the CHECK constraint on notifications.type).
const (
	NotificationTypeQuotaReset     = "quota_reset"
	NotificationTypeWatchlistMatch = "watchlist_match"
	NotificationTypeExportReady    = "export_ready"
	NotificationTypeQuotaWarning   = "quota_warning"
)

type Notification struct {
	ID        uuid.UUID              `json:"id" db:"id"`
	UserID    uuid.UUID              `json:"user_id" db:"user_id"`
	Type      string                 `json:"type" db:"type"`
	Title     string                 `json:"title" db:"title"`
	Body      string                 `json:"body" db:"body"`
	Payload   map[string]interface{} `json:"payload,omitempty" db:"payload"`
	CreatedAt time.Time              `json:"created_at" db:"created_at"`
	ReadAt    *time.Time             `json:"read_at,omitempty" db:"read_at"`
	PushedAt  *time.Time             `json:"-" db:"pushed_at"`
}

type PushSubscription struct {
	ID         uuid.UUID `json:"-" db:"id"`
	UserID     uuid.UUID `json:"-" db:"user_id"`
	Endpoint   string    `json:"endpoint" db:"endpoint"`
	P256dh     string    `json:"p256dh" db:"p256dh"`
	Auth       string    `json:"auth" db:"auth"`
	DeviceType string    `json:"device_type,omitempty" db:"device_type"`
	CreatedAt  time.Time `json:"-" db:"created_at"`
}

// NotificationPrefs helpers on User — prefs is stored as JSONB on users
// (see migration 015), e.g. {"quota_reset": true}.
// quota_reset explicitly requires opt-in; every other type defaults to enabled.
func (u *User) QuotaResetOptIn() bool {
	return u.NotificationPrefs["quota_reset"] == true
}

// PrefEnabled resolves a preference key. def is used when the key is absent.
func (u *User) PrefEnabled(key string, def bool) bool {
	v, ok := u.NotificationPrefs[key]
	if !ok {
		return def
	}
	return v == true
}
