package handlers

// Golden response tests: capture today's JSON for the top read endpoints into
// testdata/golden/*.json and compare byte-for-byte (canonical re-marshal).
// Volatile keys (all *_at timestamps, *_date, date, last_login) are masked so
// the comparison targets structure, field names, types, null-vs-[] and values
// of everything else.
//
// Run with UPDATE_GOLDEN=1 to (re)write the golden files, e.g. right after a
// data change or migration, before starting an optimization.
// Requires DATABASE_URL + JWT_SECRET (or backend/.env), read-only.

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"net/http"
	"net/http/httptest"
)

type goldenCase struct {
	name string // file name in testdata/golden (without .json)
	path string // request path (format with %s for admin ID)
}

// %s is replaced with the admin user's ID picked by setupEnv.
func goldenCases() []goldenCase {
	return []goldenCase{
		{"users", "/api/admin/users?limit=50&offset=0"},
		{"user", "/api/admin/users/%s"},
		{"user_details", "/api/admin/users/%s/details"},
		{"user_stats", "/api/admin/stats/user/%s"},
		{"system_stats", "/api/admin/stats/system"},
		{"dashboard_stats", "/api/admin/dashboard-stats"},
		{"user_requests", "/api/admin/user-requests?limit=50&offset=0&status=pending"},
		{"password_change_requests", "/api/admin/password-change-requests?limit=50&offset=0&status=pending"},
		{"admin_search_history", "/api/admin/search-history?limit=50&offset=0"},
		{"admin_sessions", "/api/admin/sessions?limit=100&offset=0"},
		{"chat_conversations", "/api/chat/conversations"},
		{"chat_unread", "/api/chat/unread"},
		{"chat_unread_per_user", "/api/chat/unread-per-user"},
		{"chat_broadcasts", "/api/chat/broadcasts"},
		{"user_search_history", "/api/user/search-history?limit=50&offset=0"},
		{"password_change_requests_mine", "/api/user/password-change/requests?limit=50&offset=0"},
	}
}

// volatileKeys masks fields that legitimately change between capture and
// compare: timestamps, dates and the "last_login" proxy value.
func isVolatileKey(key string) bool {
	if strings.HasSuffix(key, "_at") || strings.HasSuffix(key, "_date") {
		return true
	}
	switch key {
	case "date", "last_login":
		return true
	}
	return false
}

func maskJSON(v interface{}) {
	switch t := v.(type) {
	case map[string]interface{}:
		for k, val := range t {
			if isVolatileKey(k) {
				if val != nil {
					t[k] = "<MASKED>"
				}
				continue
			}
			maskJSON(val)
		}
	case []interface{}:
		for _, e := range t {
			maskJSON(e)
		}
	}
}

func pathFor(path, adminID string) string { return strings.ReplaceAll(path, "%s", adminID) }

func TestGoldenResponses(t *testing.T) {
	e := setupEnv(t)
	update := os.Getenv("UPDATE_GOLDEN") == "1"

	dir := "testdata/golden"
	if update {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("cannot create %s: %v", dir, err)
		}
	}

	for _, tc := range goldenCases() {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, pathFor(tc.path, e.adminID.String()), nil)
			req.Header.Set("Authorization", "Bearer "+e.token)
			w := httptest.NewRecorder()
			e.router.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
			}

			var payload interface{}
			if err := json.Unmarshal(w.Body.Bytes(), &payload); err != nil {
				t.Fatalf("response is not valid JSON: %v", err)
			}
			maskJSON(payload)
			actual, err := json.MarshalIndent(payload, "", "  ")
			if err != nil {
				t.Fatalf("cannot re-marshal response: %v", err)
			}
			actual = append(actual, '\n')

			goldenPath := filepath.Join(dir, tc.name+".json")
			if update {
				if err := os.WriteFile(goldenPath, actual, 0o644); err != nil {
					t.Fatalf("cannot write golden %s: %v", goldenPath, err)
				}
				t.Logf("golden written: %s", goldenPath)
				return
			}

			raw, err := os.ReadFile(goldenPath)
			if os.IsNotExist(err) {
				// Seed the file so CI stays green until the user seeds goldens.
				if err := os.MkdirAll(dir, 0o755); err != nil {
					t.Skipf("golden not found and cannot seed: %v", err)
				}
				if werr := os.WriteFile(goldenPath, actual, 0o644); werr != nil {
					t.Skipf("golden not found and cannot seed: %v", werr)
				}
				t.Skipf("golden %s seeded from current response; re-run to compare", goldenPath)
			}
			if err != nil {
				t.Fatalf("cannot read golden %s: %v", goldenPath, err)
			}

			if string(raw) != string(actual) {
				t.Errorf("golden mismatch for %s\nrun UPDATE_GOLDEN=1 only if the change is intended\n--- golden ---\n%s\n--- actual ---\n%s",
					goldenPath, raw, actual)
			}
		})
	}
}
