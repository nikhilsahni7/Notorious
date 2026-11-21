package handlers

import (
	"notorious-backend/internal/models"
	"testing"
)

func TestHideAdminEmails(t *testing.T) {
	original := []models.ConversationWithDetails{
		{OtherUserRole: "admin", OtherUserEmail: "admin@example.invalid"},
		{OtherUserRole: "user", OtherUserEmail: "user@example.invalid"},
	}
	for _, tc := range []struct{ viewer, adminEmail string }{{"user", ""}, {"", ""}, {"admin", "admin@example.invalid"}} {
		rows := append([]models.ConversationWithDetails(nil), original...)
		hideAdminEmails(rows, tc.viewer)
		if rows[0].OtherUserEmail != tc.adminEmail || rows[1].OtherUserEmail != "user@example.invalid" {
			t.Fatalf("role %q got emails %q, %q", tc.viewer, rows[0].OtherUserEmail, rows[1].OtherUserEmail)
		}
	}
}
