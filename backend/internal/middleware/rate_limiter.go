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

	v, exists := rl.visitors[ip]
	if !exists || time.Since(v.lastSeen) > rl.window {
		// New visitor or window expired
		rl.visitors[ip] = &visitor{count: 0, lastSeen: time.Now()}
		return rl.visitors[ip]
	}

	return v
}

// Limit returns a Gin middleware that rate limits requests
func (rl *RateLimiter) Limit() gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		v := rl.getVisitor(ip)

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

// AdminRateLimiter - limits for admin endpoints (60 requests per minute)
func AdminRateLimiter() gin.HandlerFunc {
	limiter := NewRateLimiter(60, time.Minute)
	return limiter.Limit()
}

// AuthRateLimiter - strict limits for login attempts (10 per minute per IP)
// Prevents brute force attacks
func AuthRateLimiter() gin.HandlerFunc {
	limiter := NewRateLimiter(10, time.Minute)
	return limiter.Limit()
}

// AccessRequestRateLimiter - very strict limits for public access requests (5 per hour per IP)
// Prevents spam registration requests
func AccessRequestRateLimiter() gin.HandlerFunc {
	limiter := NewRateLimiter(5, time.Hour)
	return limiter.Limit()
}

// SearchRateLimiter - allows up to 60 searches per minute per IP
// Users can do 10-15 searches/min comfortably with room to spare
func SearchRateLimiter() gin.HandlerFunc {
	limiter := NewRateLimiter(60, time.Minute)
	return limiter.Limit()
}
