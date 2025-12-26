package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// BotBlocker blocks requests from known automated tools
func BotBlocker() gin.HandlerFunc {
	blockedAgents := []string{
		"python-requests",
		"python",
		"postman",
		"curl",
		"wget",
		"httpie",
		"insomnia",
		"axios",
		"got",
		"node-fetch",
		"scrapy",
		"bot",
		"spider",
		"crawler",
	}

	return func(c *gin.Context) {
		userAgent := strings.ToLower(c.GetHeader("User-Agent"))

		// Allow empty user agent to pass (some legitimate requests)
		if userAgent == "" {
			c.Next()
			return
		}

		// Check for blocked agents
		for _, blocked := range blockedAgents {
			if strings.Contains(userAgent, blocked) {
				c.JSON(http.StatusForbidden, gin.H{
					"error": "automated access is not permitted",
				})
				c.Abort()
				return
			}
		}

		c.Next()
	}
}
