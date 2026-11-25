package handlers

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestTelegramSearchRejectsInvalidRequests(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	h := &SearchHandler{}
	router.POST("/search", h.TelegramSearch(strings.Repeat("x", 32)))
	for _, tc := range []struct {
		name, auth, body string
		want             int
	}{
		{"missing secret", "", `{}`, 401},
		{"bad phone", strings.Repeat("x", 32), `{"type":"phone","value":"abc","region":"pan-india"}`, 400},
		{"bad email", strings.Repeat("x", 32), `{"type":"email","value":"bad@@address","region":"pan-india"}`, 400},
		{"bad master id", strings.Repeat("x", 32), `{"type":"master_id","value":"*","region":"pan-india"}`, 400},
		{"bad type", strings.Repeat("x", 32), `{"type":"name","value":"someone","region":"pan-india"}`, 400},
		{"old region", strings.Repeat("x", 32), `{"type":"phone","value":"9876543210","region":"delhi-ncr"}`, 400},
		{"bad region", strings.Repeat("x", 32), `{"type":"phone","value":"9876543210","region":"world"}`, 400},
		{"malformed json", strings.Repeat("x", 32), `{`, 400},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodPost, "/search", bytes.NewBufferString(tc.body))
			r.Header.Set("Content-Type", "application/json")
			if tc.auth != "" {
				r.Header.Set("Authorization", "Bearer "+tc.auth)
			}
			w := httptest.NewRecorder()
			router.ServeHTTP(w, r)
			if w.Code != tc.want {
				t.Fatalf("got %d wanted %d", w.Code, tc.want)
			}
		})
	}
}
