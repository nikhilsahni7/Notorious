package websocket

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	ws "github.com/gorilla/websocket"
	"notorious-backend/internal/auth"
	"notorious-backend/internal/database"
	"notorious-backend/internal/repository"
)

// Opt in only with a dedicated local database containing migrations 001, 010 and 019.
func TestBroadcastExclusionsEndToEnd(t *testing.T) {
	url := os.Getenv("CHAT_TEST_DATABASE_URL")
	if url == "" {
		t.Skip("set CHAT_TEST_DATABASE_URL to a dedicated local database")
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
	for _, u := range []struct {
		id   uuid.UUID
		role string
	}{{sender, "admin"}, {included, "user"}, {excluded, "user"}} {
		_, err := db.Pool.Exec(ctx, `INSERT INTO users (id,email,password_hash,name,role) VALUES ($1,$2,'test','test',$3)`, u.id, u.id.String()+"@example.invalid", u.role)
		if err != nil {
			t.Fatal(err)
		}
	}
	defer db.Pool.Exec(ctx, `DELETE FROM users WHERE id = ANY($1::uuid[])`, []uuid.UUID{sender, included, excluded})
	repo := repository.NewChatRepository(db)
	manager := auth.NewJWTManager("test-secret", time.Hour)
	hub := NewHub(repo, manager)
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.GET("/ws", hub.HandleUpgrade)
	server := httptest.NewServer(router)
	defer server.Close()
	connect := func(id uuid.UUID, role string) *ws.Conn {
		token, err := manager.Generate(id, "test@example.invalid", role)
		if err != nil {
			t.Fatal(err)
		}
		c, _, err := ws.DefaultDialer.Dial("ws"+strings.TrimPrefix(server.URL, "http")+"/ws?token="+token, nil)
		if err != nil {
			t.Fatal(err)
		}
		return c
	}
	admin := connect(sender, "admin")
	defer admin.Close()
	recipient := connect(included, "user")
	defer recipient.Close()
	excludedClient := connect(excluded, "user")
	defer excludedClient.Close()
	if err := admin.WriteJSON(map[string]any{"type": "broadcast", "payload": map[string]any{"content": "test announcement", "temp_id": "test-broadcast", "excluded_user_ids": []string{excluded.String()}}}); err != nil {
		t.Fatal(err)
	}
	wait := func(c *ws.Conn, want string, timeout time.Duration) map[string]any {
		t.Helper()
		_ = c.SetReadDeadline(time.Now().Add(timeout))
		for {
			_, raw, err := c.ReadMessage()
			if err != nil {
				if want == "none" {
					return nil
				}
				t.Fatalf("waiting for %s: %v", want, err)
			}
			for _, line := range strings.Split(string(raw), "\n") {
				var envelope struct {
					Type    string         `json:"type"`
					Payload map[string]any `json:"payload"`
				}
				if json.Unmarshal([]byte(line), &envelope) == nil && envelope.Type == "broadcast" && want == "none" {
					t.Fatal("excluded user received broadcast")
				}
				if envelope.Type == want {
					return envelope.Payload
				}
			}
		}
	}
	ack := wait(admin, "message_ack", 3*time.Second)
	if ack["temp_id"] != "test-broadcast" {
		t.Fatalf("unexpected ack: %v", ack)
	}
	wait(recipient, "broadcast", 3*time.Second)
	wait(excludedClient, "none", 300*time.Millisecond)
	_, unread, err := repo.GetUnreadCount(ctx, excluded)
	if err != nil || unread != 0 {
		t.Fatalf("excluded unread=%d: %v", unread, err)
	}
	history, err := repo.GetBroadcasts(ctx, excluded, 100)
	if err != nil || len(history) != 0 {
		t.Fatalf("excluded history=%d: %v", len(history), err)
	}
	_, unread, err = repo.GetUnreadCount(ctx, included)
	if err != nil || unread != 1 {
		t.Fatalf("included unread=%d: %v", unread, err)
	}
}
