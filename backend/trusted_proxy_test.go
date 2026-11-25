package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestTrustedProxyDoesNotTrustArbitraryForwarding(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	if err := r.SetTrustedProxies([]string{"127.0.0.1", "::1"}); err != nil {
		t.Fatal(err)
	}
	r.GET("/ip", func(c *gin.Context) { c.String(http.StatusOK, c.ClientIP()) })
	for _, tc := range []struct{ remote, forwarded, want string }{{"198.51.100.10:1234", "203.0.113.99", "198.51.100.10"}, {"127.0.0.1:1234", "203.0.113.99, 198.51.100.10", "198.51.100.10"}} {
		req := httptest.NewRequest("GET", "/ip", nil)
		req.RemoteAddr = tc.remote
		req.Header.Set("X-Forwarded-For", tc.forwarded)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if got := w.Body.String(); got != tc.want {
			t.Errorf("remote %s: got %s want %s", tc.remote, got, tc.want)
		}
	}
}
