package models

import (
	"time"

	"github.com/google/uuid"
)

type UserSession struct {
	ID         uuid.UUID `json:"id"`
	UserID     uuid.UUID `json:"user_id"`
	TokenHash  string    `json:"-"`
	DeviceName string    `json:"device_name"`
	DeviceOS   string    `json:"device_os"`
	DeviceType string    `json:"device_type"`
	IPAddress  string    `json:"ip_address"`
	Location   string    `json:"location"`
	LastActive time.Time `json:"last_active"`
	CreatedAt  time.Time `json:"created_at"`
}
