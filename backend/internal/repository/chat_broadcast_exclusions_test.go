package repository

import (
	"context"
	"os"
	"testing"

	"github.com/google/uuid"
	"notorious-backend/internal/database"
)

// This opt-in test uses a dedicated database initialized with migrations 001, 010 and 019.
func TestBroadcastExclusionsPersistAndFilterUnread(t *testing.T) {
	url := os.Getenv("CHAT_TEST_DATABASE_URL")
	if url == "" {
		t.Skip("set CHAT_TEST_DATABASE_URL to a dedicated local test database")
	}
	ctx := context.Background()
	db, err := database.NewPostgresDB(url)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	lock, err := db.Pool.Acquire(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer lock.Release()
	if _, err := lock.Exec(ctx, `SELECT pg_advisory_lock(84484311)`); err != nil {
		t.Fatal(err)
	}
	defer lock.Exec(ctx, `SELECT pg_advisory_unlock(84484311)`)
	sender, included, excluded := uuid.New(), uuid.New(), uuid.New()
	for _, entry := range []struct {
		id   uuid.UUID
		role string
	}{{sender, "admin"}, {included, "user"}, {excluded, "user"}} {
		_, err := db.Pool.Exec(ctx, `INSERT INTO users (id,email,password_hash,name,role) VALUES ($1,$2,'test','test',$3)`, entry.id, entry.id.String()+"@example.invalid", entry.role)
		if err != nil {
			t.Fatal(err)
		}
	}
	defer db.Pool.Exec(ctx, `DELETE FROM users WHERE id = ANY($1::uuid[])`, []uuid.UUID{sender, included, excluded})
	repo := NewChatRepository(db)
	sent, err := repo.SaveBroadcast(ctx, sender, "selected audience", []uuid.UUID{excluded})
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		id                  uuid.UUID
		wantCount, wantList int
	}{{included, 1, 1}, {excluded, 0, 0}, {sender, 0, 1}} {
		_, count, err := repo.GetUnreadCount(ctx, tc.id)
		if err != nil || count != tc.wantCount {
			t.Fatalf("user %s: unread=%d, want %d: %v", tc.id, count, tc.wantCount, err)
		}
		items, err := repo.GetBroadcasts(ctx, tc.id, 100)
		if err != nil || len(items) != tc.wantList {
			t.Fatalf("user %s: list length=%d, want %d: %v", tc.id, len(items), tc.wantList, err)
		}
		if len(items) > 0 && items[0].ID != sent.ID {
			t.Fatal("unexpected broadcast")
		}
	}
	if err := repo.MarkBroadcastRead(ctx, sent.ID, included); err != nil {
		t.Fatal(err)
	}
	_, count, err := repo.GetUnreadCount(ctx, included)
	if err != nil || count != 0 {
		t.Fatalf("read count=%d: %v", count, err)
	}
	// A broadcast without exclusions remains visible to everyone.
	if _, err := repo.SaveBroadcast(ctx, sender, "everyone", nil); err != nil {
		t.Fatal(err)
	}
	items, err := repo.GetBroadcasts(ctx, excluded, 100)
	if err != nil || len(items) != 1 || items[0].Content != "everyone" {
		t.Fatalf("excluded user's later broadcast list: %+v: %v", items, err)
	}
	_, count, err = repo.GetUnreadCount(ctx, excluded)
	if err != nil || count != 1 {
		t.Fatalf("excluded user's later unread count=%d: %v", count, err)
	}
}
