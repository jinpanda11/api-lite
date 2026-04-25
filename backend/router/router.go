package router

import (
	"io/fs"
	"net/http"
	"strings"

	"new-api-lite/handler"
	"new-api-lite/middleware"

	"github.com/gin-gonic/gin"
)

func Setup(r *gin.Engine, webFS fs.FS) {
	r.Use(middleware.CORS())

	// Serve embedded frontend (SPA) for non-API routes
	r.NoRoute(spaHandler(webFS))

	// ── Public routes ─────────────────────────────────────────────────────────
	api := r.Group("/api")
	{
		// Email verification code (two compatible paths)
		api.GET("/verification", handler.SendVerificationCode)
		api.POST("/user/email/code", handler.SendVerificationCode)

		// Auth
		api.POST("/user/register", handler.Register)
		api.POST("/user/login", handler.Login)

		// Settings (public)
		api.GET("/settings/email-verification", handler.GetEmailVerificationStatus)
	}

	// ── Authenticated routes ──────────────────────────────────────────────────
	auth := api.Group("", middleware.AuthRequired())
	{
		// User
		auth.GET("/user/info", handler.GetUserInfo)
		auth.POST("/user/update-password", handler.UpdatePassword)

		// Dashboard
		auth.GET("/dashboard", handler.GetDashboard)

		// Tokens (user's own)
		auth.GET("/token", handler.ListTokens)
		auth.POST("/token", handler.CreateToken)
		auth.PUT("/token/:id", handler.UpdateToken)
		auth.DELETE("/token/:id", handler.DeleteToken)

		// Models
		auth.GET("/models", handler.ListModels)

		// Notices (active, user-facing)
		auth.GET("/notice", handler.GetActiveNotices)

		// Logs
		auth.GET("/log", handler.GetLogs)

		// Wallet
		auth.GET("/balance", handler.GetBalance)
		auth.POST("/redeem", handler.Redeem)
		auth.GET("/topup/logs", handler.GetTopupLogs)
	}

	// ── Admin-only routes ─────────────────────────────────────────────────────
	admin := api.Group("", middleware.AuthRequired(), middleware.AdminRequired())
	{
		// Channels
		admin.GET("/channel", handler.ListChannels)
		admin.POST("/channel", handler.CreateChannel)
		admin.PUT("/channel/:id", handler.UpdateChannel)
		admin.DELETE("/channel/:id", handler.DeleteChannel)
		admin.POST("/channel/test", handler.TestChannel)

		// Redeem code management
		admin.GET("/admin/redeem", handler.ListRedeemCodes)
		admin.POST("/admin/redeem", handler.CreateRedeemCodes)
		admin.DELETE("/admin/redeem/:id", handler.DeleteRedeemCode)

		// User management
		admin.GET("/admin/user", handler.ListUsers)
		admin.PUT("/admin/user/:id", handler.UpdateUserStatus)

		// Model pricing
		admin.GET("/admin/model-pricing", handler.ListModelPricing)
		admin.PUT("/admin/model-pricing/:modelName", handler.UpdateModelPricing)

		// Notices (admin CRUD)
		admin.GET("/admin/notice", handler.ListNotices)
		admin.POST("/admin/notice", handler.CreateNotice)
		admin.PUT("/admin/notice/:id", handler.UpdateNotice)
		admin.DELETE("/admin/notice/:id", handler.DeleteNotice)

		// System settings
		admin.GET("/admin/settings", handler.GetSettings)
		admin.PUT("/admin/settings", handler.UpdateSettings)
	}

	// ── Relay: forward all /v1/* to upstream ─────────────────────────────────
	r.Any("/v1/*path", handler.Relay)
}

// spaHandler serves the embedded frontend for SPA routing.
// If the requested file exists, serve it directly; otherwise serve index.html
// so React Router handles client-side routes.
func spaHandler(webFS fs.FS) gin.HandlerFunc {
	fileServer := http.FileServer(http.FS(webFS))
	return func(c *gin.Context) {
		p := c.Request.URL.Path
		if strings.HasPrefix(p, "/api") || strings.HasPrefix(p, "/v1") {
			c.Status(http.StatusNotFound)
			return
		}
		c.Request.URL.Path = strings.TrimPrefix(p, "/")
		_, err := webFS.Open(c.Request.URL.Path)
		if err != nil {
			c.Request.URL.Path = "/"
		}
		fileServer.ServeHTTP(c.Writer, c.Request)
	}
}
