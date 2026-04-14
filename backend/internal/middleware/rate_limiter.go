package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// RateLimiter provides IP-based rate limiting
type RateLimiter struct {
	visitors map[string]*visitor
	mu       sync.RWMutex
	limit    int           // requests per window
	window   time.Duration // time window
}

type visitor struct {
	count    int
	lastSeen time.Time
	windowAt time.Time
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	rl := &RateLimiter{
		visitors: make(map[string]*visitor),
		limit:    limit,
		window:   window,
	}

	// Cleanup old entries every minute
	go rl.cleanupLoop()

	return rl
}

func (rl *RateLimiter) cleanupLoop() {
	for {
		time.Sleep(time.Minute)
		rl.mu.Lock()
		for ip, v := range rl.visitors {
			if time.Since(v.lastSeen) > rl.window*2 {
				delete(rl.visitors, ip)
			}
		}
		rl.mu.Unlock()
	}
}

// getVisitor retrieves or creates a visitor record
func (rl *RateLimiter) getVisitor(ip string) *visitor {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	now := time.Now()

	v, exists := rl.visitors[ip]
	if !exists {
		rl.visitors[ip] = &visitor{count: 0, lastSeen: now, windowAt: now}
		return rl.visitors[ip]
	}

	// Reset the counter when the active rate-limit window elapses.
	if now.Sub(v.windowAt) >= rl.window {
		v.count = 0
		v.windowAt = now
	}

	v.lastSeen = now

	return v
}

// Limit returns a Gin middleware that rate limits requests
func (rl *RateLimiter) Limit() gin.HandlerFunc {
	return rl.LimitWithKey(func(c *gin.Context) string {
		return c.ClientIP()
	})
}

// LimitWithKey returns a Gin middleware that rate limits requests by a custom key.
func (rl *RateLimiter) LimitWithKey(keyFn func(*gin.Context) string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip rate limiting for OPTIONS preflight requests (CORS)
		if c.Request.Method == "OPTIONS" {
			c.Next()
			return
		}

		key := keyFn(c)
		if key == "" {
			key = c.ClientIP()
		}

		v := rl.getVisitor(key)

		rl.mu.Lock()
		v.count++
		v.lastSeen = time.Now()
		count := v.count
		rl.mu.Unlock()

		if count > rl.limit {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":       "too many requests",
				"retry_after": int(rl.window.Seconds()),
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// AdminRateLimiter - limits for admin endpoints (120 requests per minute)
// Increased from 60 to accommodate online users polling every 30 seconds
func AdminRateLimiter() gin.HandlerFunc {
	limiter := NewRateLimiter(120, time.Minute)
	return limiter.LimitWithKey(func(c *gin.Context) string {
		if userID, exists := c.Get("user_id"); exists {
			if id, ok := userID.(string); ok && id != "" {
				return "admin:" + id
			}
		}

		return "ip:" + c.ClientIP()
	})
}

// AuthRateLimiter - strict limits for login attempts (15 per minute per IP)
// Prevents brute force attacks while allowing a few password typos
func AuthRateLimiter() gin.HandlerFunc {
	limiter := NewRateLimiter(15, time.Minute)
	return limiter.Limit()
}

// AccessRequestRateLimiter - very strict limits for public access requests (5 per hour per IP)
// Prevents spam registration requests
func AccessRequestRateLimiter() gin.HandlerFunc {
	limiter := NewRateLimiter(5, time.Hour)
	return limiter.Limit()
}

// SearchRateLimiter - allows up to 100 searches per minute per IP
// Increased to accommodate heartbeat polling + active user searching
func SearchRateLimiter() gin.HandlerFunc {
	limiter := NewRateLimiter(100, time.Minute)
	return limiter.Limit()
}
