package middleware

import (
	"github.com/gin-gonic/gin"
)

func CORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Allow requests from any origin but without credentials.
		// This is safe for Bearer-token-based auth (tokens are sent explicitly,
		// not via cookies) and avoids the reflective-origin + credentials CSRF risk.
		// Native apps and server-to-server clients ignore CORS entirely.
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE,OPTIONS,PATCH")
		c.Header("Access-Control-Allow-Headers", "Content-Type,Authorization,X-Requested-With")
		c.Header("Access-Control-Expose-Headers", "Content-Length,Content-Type")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	}
}
