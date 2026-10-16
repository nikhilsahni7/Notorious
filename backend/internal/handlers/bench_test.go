package handlers

// Benchmarks for the hot handler and repository paths.
//
// DB-backed benches use benchRouter (no rate limiters) and skip when
// DATABASE_URL/JWT_SECRET are not configured. They perform read-only queries.
//
// Baseline command (from backend/):
//
//	go test -run='^$' -bench=. -benchmem -count=10 ./... > old.txt
//
// Compare after each optimization with:
//
//	benchstat old.txt new.txt

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"notorious-backend/internal/auth"
	"notorious-backend/internal/database"
	"notorious-backend/internal/repository"

	"github.com/google/uuid"
)

type ginLikeRouter interface {
	ServeHTTP(http.ResponseWriter, *http.Request)
}

// benchOnce warms up the path outside the timed loop and fails fast on non-200.
func benchOnce(b *testing.B, router ginLikeRouter, method, path, token string) {
	req := httptest.NewRequest(method, path, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		b.Fatalf("%s %s: expected 200, got %d: %s", method, path, w.Code, w.Body.String())
	}
}

func benchHandler(b *testing.B, method, pathFormat string) {
	e := setupEnv(b)
	path := strings.ReplaceAll(pathFormat, "%s", e.adminID.String())

	benchOnce(b, e.benchRouter, method, path, e.token)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(method, path, nil)
		req.Header.Set("Authorization", "Bearer "+e.token)
		w := httptest.NewRecorder()
		e.benchRouter.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			b.Fatalf("%s %s: expected 200, got %d: %s", method, path, w.Code, w.Body.String())
		}
	}
}

// ── Handler-level benchmarks (through the real router) ────────────────────

// 10 sequential queries per request — consolidation target #1.
func BenchmarkHandler_GetUserStats(b *testing.B) {
	benchHandler(b, http.MethodGet, "/api/admin/stats/user/%s")
}

// ~15 sequential queries with duplicated full-table COUNT(*) — target #2.
func BenchmarkHandler_GetSystemStats(b *testing.B) {
	benchHandler(b, http.MethodGet, "/api/admin/stats/system")
}

// List + CountByRegion + fully-batched search counts — target #7.
func BenchmarkHandler_ListUsers(b *testing.B) {
	benchHandler(b, http.MethodGet, "/api/admin/users?limit=50&offset=0")
}

func BenchmarkHandler_GetDashboardStats(b *testing.B) {
	benchHandler(b, http.MethodGet, "/api/admin/dashboard-stats")
}

func BenchmarkHandler_GetUser(b *testing.B) {
	benchHandler(b, http.MethodGet, "/api/admin/users/%s")
}

func BenchmarkHandler_GetUserDetails(b *testing.B) {
	benchHandler(b, http.MethodGet, "/api/admin/users/%s/details")
}

// Correlated COUNT(*) subquery per conversation row — target #5.
func BenchmarkHandler_GetConversations(b *testing.B) {
	benchHandler(b, http.MethodGet, "/api/chat/conversations")
}

func BenchmarkHandler_GetAdminSearchHistory(b *testing.B) {
	benchHandler(b, http.MethodGet, "/api/admin/search-history?limit=50&offset=0")
}

func BenchmarkHandler_GetOnlineUsers(b *testing.B) {
	benchHandler(b, http.MethodGet, "/api/admin/users/online")
}

// ── Repository-level benchmarks ───────────────────────────────────────────

func benchRepos(b *testing.B) (*repository.UserRepository, *repository.SearchHistoryRepository, uuid.UUID, *database.DB) {
	e := setupEnv(b)
	db := &database.DB{Pool: e.pool}
	return repository.NewUserRepository(db),
		repository.NewSearchHistoryRepository(db),
		e.adminID,
		db
}

func BenchmarkRepo_UserList(b *testing.B) {
	userRepo, _, _, _ := benchRepos(b)
	ctx := context.Background()
	if _, _, err := userRepo.List(ctx, "", "", "", 50, 0); err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, _, err := userRepo.List(ctx, "", "", "", 50, 0); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkRepo_UserGetByEmail(b *testing.B) {
	userRepo, _, adminID, db := benchRepos(b)
	var email string
	if err := db.Pool.QueryRow(context.Background(), `SELECT email FROM users WHERE id = $1`, adminID).Scan(&email); err != nil {
		b.Fatal(err)
	}
	ctx := context.Background()
	if _, err := userRepo.GetByEmail(ctx, email); err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := userRepo.GetByEmail(ctx, email); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkRepo_SearchHistoryGetAllWithUsers(b *testing.B) {
	_, searchRepo, _, _ := benchRepos(b)
	ctx := context.Background()
	if _, _, err := searchRepo.GetAllWithUsers(ctx, 50, 0, ""); err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, _, err := searchRepo.GetAllWithUsers(ctx, 50, 0, ""); err != nil {
			b.Fatal(err)
		}
	}
}

// ── DB-free micro-benchmarks (run even without database env) ───────────────

var hashTokenSink string

func BenchmarkAuth_HashToken(b *testing.B) {
	token := "benchmark-fixed-input-token-for-hashing"
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		hashTokenSink = auth.HashToken(token)
	}
}

var jwtVerifySink interface{}

func BenchmarkAuth_JWTVerify(b *testing.B) {
	mgr := auth.NewJWTManager("benchmark-secret", time.Hour)
	token, err := mgr.Generate(uuid.New(), "bench@example.com", "admin")
	if err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		claims, err := mgr.Verify(token)
		if err != nil {
			b.Fatal(err)
		}
		jwtVerifySink = claims
	}
}

func BenchmarkFormatHour(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		formatHour(i % 24)
	}
}
