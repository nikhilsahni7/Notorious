package handlers

// Read-only endpoint tests for the notifications API, following the
// DB-backed pattern of the golden tests. Skips silently when DATABASE_URL /
// JWT_SECRET are not configured.
import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestNotificationsEndpoints hits the read-only notification endpoints as the
// admin test user (who may have zero rows; the shape of the response is what
// matters).
func TestNotificationsEndpoints(t *testing.T) {
	env := setupEnv(t)

	cases := []struct {
		name     string
		path     string
		method   string
		wantCode int
		check    func(t *testing.T, body map[string]interface{})
	}{
		{
			name:     "list",
			path:     "/api/notifications?limit=50&offset=0",
			wantCode: http.StatusOK,
			check: func(t *testing.T, body map[string]interface{}) {
				if _, ok := body["total"]; !ok {
					t.Errorf("`total` missing from list response")
				}
				if _, ok := body["notifications"]; !ok {
					t.Errorf("`notifications` missing from list response")
				}
			},
		},
		{
			name:     "unread-filtered list",
			path:     "/api/notifications?limit=10&unread=true",
			wantCode: http.StatusOK,
			check: func(t *testing.T, body map[string]interface{}) {
				// every returned row must be unread (omitted read_at)
				rows, _ := body["notifications"].([]interface{})
				for i, r := range rows {
					row, ok := r.(map[string]interface{})
					if !ok {
						t.Fatalf("row %d is not an object", i)
					}
					if _, present := row["read_at"]; present {
						t.Errorf("unread filter returned row %d with read_at set", i)
					}
				}
			},
		},
		{
			name:     "unread-count",
			path:     "/api/notifications/unread-count",
			wantCode: http.StatusOK,
			check: func(t *testing.T, body map[string]interface{}) {
				if _, ok := body["unread"]; !ok {
					t.Errorf("`unread` missing from unread-count response")
				}
			},
		},
		{
			name:     "prefs",
			path:     "/api/notifications/prefs",
			wantCode: http.StatusOK,
			check: func(t *testing.T, body map[string]interface{}) {
				if _, ok := body["prefs"]; !ok {
					t.Errorf("`prefs` missing from prefs response")
				}
			},
		},
		{
			name:     "heartbeat carries unread count",
			path:     "/api/user/heartbeat",
			method:   http.MethodPost,
			wantCode: http.StatusOK,
			check: func(t *testing.T, body map[string]interface{}) {
				if _, ok := body["unread_notifications"]; !ok {
					t.Errorf("`unread_notifications` missing from heartbeat response")
				}
			},
		},
		{
			name:     "mark read requires uuid",
			path:     "/api/notifications/not-a-uuid/read",
			method:   http.MethodPost,
			wantCode: http.StatusBadRequest,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			method := tc.method
			if method == "" {
				method = http.MethodGet
			}
			req := httptest.NewRequest(method, tc.path, nil)
			req.Header.Set("Authorization", "Bearer "+env.token)
			w := httptest.NewRecorder()
			env.router.ServeHTTP(w, req)

			if w.Code != tc.wantCode {
				t.Fatalf("expected %d, got %d: %s", tc.wantCode, w.Code, w.Body.String())
			}

			if tc.check == nil {
				return
			}
			var body map[string]interface{}
			if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
				t.Fatalf("failed to decode response: %v", err)
			}
			tc.check(t, body)
		})
	}
}
