package router

import (
	"io/fs"
	"net/http"
	"strings"

	"new-api-lite/handler"
	"new-api-lite/middleware"

	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
)

func Setup(r *gin.Engine, webFS fs.FS) {
	r.Use(gzip.Gzip(gzip.DefaultCompression))
	r.Use(middleware.SecurityHeaders())
	r.Use(middleware.CORS())

	// Serve embedded frontend (SPA) for non-API routes
	r.NoRoute(spaHandler(webFS))

	// -- Public routes --
	api := r.Group("/api")
	{
		// Email verification code (two compatible paths)
		api.GET("/verification", handler.SendVerificationCode)
		api.POST("/user/email/code", handler.SendVerificationCode)

		// Auth
		api.POST("/user/register", handler.Register)
		api.POST("/user/login", handler.Login)

		// Settings (public)
		api.GET("/settings/branding", handler.GetBranding)
		api.GET("/settings/email-verification", handler.GetEmailVerificationStatus)
	}

	// -- Authenticated routes --
	auth := api.Group("", middleware.AuthRequired())
	{
		// User
		auth.GET("/user/info", handler.GetUserInfo)
		auth.POST("/user/update-password", handler.UpdatePassword)
		auth.POST("/user/logout", handler.Logout)

		// Dashboard
		auth.GET("/dashboard", handler.GetDashboard)

		// Tokens (user's own)
		auth.GET("/token", handler.ListTokens)
		auth.POST("/token", handler.CreateToken)
		auth.PUT("/token/:id", handler.UpdateToken)
		auth.DELETE("/token/:id", handler.DeleteToken)
			auth.GET("/token/chat-key", handler.GetOrCreateChatToken)

		// Models
		auth.GET("/models", handler.ListModels)

		// Notices (active, user-facing)
		auth.GET("/notice", handler.GetActiveNotices)

		// Logs
		auth.GET("/log", handler.GetLogs)

		// Status
		auth.GET("/status", handler.GetStatus)

		// Wallet
		auth.GET("/balance", handler.GetBalance)
		auth.POST("/redeem", handler.Redeem)
		auth.GET("/topup/logs", handler.GetTopupLogs)

		// Check-in
		auth.POST("/checkin", handler.CheckIn)
		auth.GET("/checkin/status", handler.GetCheckInStatus)
	}

	// -- Admin-only routes --
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
		admin.DELETE("/admin/redeem/used", handler.DeleteUsedRedeemCodes)
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

		// Admin stats
		admin.GET("/admin/stats", handler.AdminStats)
		admin.GET("/admin/daily-costs", handler.GetDailyCosts)

		// Backup
		admin.POST("/admin/backup", handler.BackupNow)

		// Audit log
		admin.GET("/admin/audit", handler.GetAuditLogs)

		// Channel monitor toggle
		admin.PUT("/channel/:id/monitor", handler.ToggleChannelMonitor)
		admin.GET("/admin/monitor-config", handler.GetMonitorConfig)
		admin.PUT("/admin/monitor-config", handler.UpdateMonitorConfig)
	}

	// -- Chat API routes (auth required) --
	chat := api.Group("", middleware.AuthRequired())
	chat.POST("/chat-process", handler.ChatProcess)
	chat.POST("/session", handler.ChatSession)
	chat.POST("/config", handler.ChatConfig)

	// -- Chat SPA --
	r.GET("/chat", serveChatIndex(webFS))
	r.GET("/chat/*filepath", serveChatSPA(webFS))

	// -- Relay: forward all /v1/* to upstream --
	r.Any("/v1/*path", handler.Relay)
}

// -- Chat SPA static file handlers --

func serveChatIndex(webFS fs.FS) gin.HandlerFunc {
	return func(c *gin.Context) {
		data, err := fs.ReadFile(webFS, "chat/index.html")
		if err != nil {
			c.Status(http.StatusNotFound)
			return
		}
		c.Header("Content-Type", "text/html; charset=utf-8")
		c.Header("Cache-Control", "no-cache")
		c.Data(http.StatusOK, "text/html; charset=utf-8", data)
	}
}

func serveChatSPA(webFS fs.FS) gin.HandlerFunc {
	return func(c *gin.Context) {
		filePath := "chat/" + strings.TrimPrefix(c.Param("filepath"), "/")
		data, err := fs.ReadFile(webFS, filePath)
		if err != nil {
			c.Status(http.StatusNotFound)
			return
		}
		serveFile(c, data, filePath)
	}
}

// spaHandler serves the embedded frontend for SPA routing.
// Static assets (JS/CSS/fonts) get long-term cache headers because they have
// hashed filenames. index.html gets no-cache to allow seamless deployments.
func spaHandler(webFS fs.FS) gin.HandlerFunc {
	return func(c *gin.Context) {
		p := c.Request.URL.Path
		if strings.HasPrefix(p, "/api") || strings.HasPrefix(p, "/v1") {
			c.Status(http.StatusNotFound)
			return
		}

		// Normalize path: strip leading / and default to index.html
		filePath := strings.TrimPrefix(p, "/")
		if filePath == "" {
			filePath = "index.html"
		}

		// Try to serve the requested file; fall back to index.html (SPA routing)
		data, err := fs.ReadFile(webFS, filePath)
		if err != nil {
			data, err = fs.ReadFile(webFS, "index.html")
			if err != nil {
				c.Status(http.StatusNotFound)
				return
			}
			serveFile(c, data, "index.html")
			return
		}
		serveFile(c, data, filePath)
	}
}

// serveFile writes file data with appropriate Content-Type and Cache-Control.
func serveFile(c *gin.Context, data []byte, name string) {
	ext := name[strings.LastIndexByte(name, '.')+1:]
	ct := contentType(ext)
	if ct != "" {
		c.Header("Content-Type", ct)
	}

	// Hashed filenames -> cache aggressively; index.html -> never cache
	if name == "index.html" {
		c.Header("Cache-Control", "no-cache")
	} else if isHashedAsset(name) {
		c.Header("Cache-Control", "public, max-age=31536000, immutable")
	}

	c.Data(http.StatusOK, ct, data)
}

func isHashedAsset(name string) bool {
	return strings.Contains(name, "-") &&
		(strings.HasSuffix(name, ".js") || strings.HasSuffix(name, ".css") ||
			strings.HasSuffix(name, ".woff") || strings.HasSuffix(name, ".woff2") ||
			strings.HasSuffix(name, ".ttf") || strings.HasSuffix(name, ".svg") ||
			strings.HasSuffix(name, ".png") || strings.HasSuffix(name, ".jpg") ||
			strings.HasSuffix(name, ".webp") || strings.HasSuffix(name, ".ico"))
}

func contentType(ext string) string {
	switch ext {
	case "html":
		return "text/html; charset=utf-8"
	case "js":
		return "text/javascript; charset=utf-8"
	case "css":
		return "text/css; charset=utf-8"
	case "json":
		return "application/json; charset=utf-8"
	case "svg":
		return "image/svg+xml"
	case "png":
		return "image/png"
	case "jpg", "jpeg":
		return "image/jpeg"
	case "webp":
		return "image/webp"
	case "ico":
		return "image/x-icon"
	case "woff":
		return "font/woff"
	case "woff2":
		return "font/woff2"
	case "ttf":
		return "font/ttf"
	default:
		return ""
	}
}
