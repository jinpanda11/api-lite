package middleware

import (
	"github.com/gin-gonic/gin"
)

func CORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		host := c.GetHeader("Host")

		if origin != "" {
			if isSameOrigin(origin, host) {
				c.Header("Access-Control-Allow-Origin", origin)
				c.Header("Vary", "Origin")
			}
			// Cross-origin browser request: don't set Allow-Origin (blocked by browser)
		} else {
			// No Origin header (curl, server-to-server, native apps): allow
			c.Header("Access-Control-Allow-Origin", "*")
		}
		c.Header("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE,OPTIONS,PATCH")
		c.Header("Access-Control-Allow-Headers", "Content-Type,Authorization")
		c.Header("Access-Control-Expose-Headers", "Content-Length,Content-Type")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	}
}

func isSameOrigin(origin, host string) bool {
	// Strip protocol prefix from origin for comparison
	o := origin
	if len(o) > 8 && o[:8] == "https://" {
		o = o[8:]
	} else if len(o) > 7 && o[:7] == "http://" {
		o = o[7:]
	}
	// Strip port from both for flexible matching
	if i := lastPort(o); i != -1 {
		o = o[:i]
	}
	h := host
	if i := lastPort(h); i != -1 {
		h = h[:i]
	}
	return o == h
}

func lastPort(s string) int {
	// Only strip port after host, not after IPv6 ]
	colon := -1
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] == ']' {
			break
		}
		if s[i] == ':' {
			colon = i
		}
	}
	return colon
}
