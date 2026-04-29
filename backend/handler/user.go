package handler

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"new-api-lite/config"
	"new-api-lite/middleware"
	"new-api-lite/model"
	"new-api-lite/service"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// rateLimiter provides multi-window rate limiting with automatic cleanup.
type rateLimiter struct {
	mu      sync.Mutex
	entries map[string]*limiterEntry
}

type limiterEntry struct {
	counts      map[time.Duration]int
	windowStart time.Time
}

type rateWindow struct {
	window time.Duration
	max    int
}

func newRateLimiter() *rateLimiter {
	rl := &rateLimiter{entries: make(map[string]*limiterEntry)}
	go rl.cleanup()
	return rl
}

func (rl *rateLimiter) check(key string, windows []rateWindow) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	e, ok := rl.entries[key]
	if !ok {
		e = &limiterEntry{counts: make(map[time.Duration]int), windowStart: now}
		rl.entries[key] = e
	}

	// Check all windows
	for _, w := range windows {
		if now.Sub(e.windowStart) > w.window {
			continue
		}
		if e.counts[w.window] >= w.max {
			return false
		}
	}

	// Increment all windows
	for _, w := range windows {
		e.counts[w.window]++
	}
	return true
}

func (rl *rateLimiter) cleanup() {
	for {
		time.Sleep(10 * time.Minute)
		rl.mu.Lock()
		now := time.Now()
		for k, e := range rl.entries {
			if now.Sub(e.windowStart) > 25*time.Hour {
				delete(rl.entries, k)
			}
		}
		rl.mu.Unlock()
	}
}

var (
	ipLimiter    = newRateLimiter()
	ipWindows    = []rateWindow{
		{1 * time.Minute, 5},
		{1 * time.Hour, 20},
	}
	emailRL      = newRateLimiter()
	emailWindows = []rateWindow{
		{1 * time.Minute, 1},
		{1 * time.Hour, 5},
		{24 * time.Hour, 10},
	}
)

// SendVerificationCode godoc
// GET /api/verification?email=xxx  OR  POST /api/user/email/code
func SendVerificationCode(c *gin.Context) {
	email := c.Query("email")
	if email == "" {
		var req struct {
			Email string `json:"email" binding:"required,email"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid email"})
			return
		}
		email = req.Email
	}
	email = strings.ToLower(strings.TrimSpace(email))

	// Rate limit: IP-based (5/min, 20/hour) and email-based (1/min, 5/hour, 10/day)
	clientIP := c.ClientIP()
	if !ipLimiter.check(clientIP, ipWindows) {
		c.JSON(http.StatusTooManyRequests, gin.H{"error": "too many requests from this IP, please try later"})
		return
	}
	if !emailRL.check(email, emailWindows) {
		c.JSON(http.StatusTooManyRequests, gin.H{"error": "too many codes sent to this email, please try later"})
		return
	}

	code := service.GenerateCode()

	// Send email BEFORE saving the code. If SMTP fails, no stale code is left in DB.
	if err := service.SendVerificationEmail(email, code); err != nil {
		if errors.Is(err, service.ErrSMTPNotConfigured) && gin.Mode() == gin.DebugMode {
			// Dev convenience: log code to console so it can be used for testing
			fmt.Printf("[EMAIL] To: %s | Code: %s\n", email, code)
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to send verification email"})
			return
		}
	}

	if err := model.SaveVerificationCode(email, code, 10*time.Minute); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save verification code"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "verification code sent"})
}

// GetEmailVerificationStatus godoc
// GET /api/settings/email-verification
func GetEmailVerificationStatus(c *gin.Context) {
	enabled := true
	if v, err := model.GetSetting("email_verification_enabled"); err == nil {
		enabled = v == "true"
	}
	c.JSON(http.StatusOK, gin.H{"enabled": enabled})
}

// login rate limiter helpers

var (
	loginLimiter = newRateLimiter()
	loginWindows = []rateWindow{
		{1 * time.Minute, 5},     // max 5 failed attempts per minute
		{15 * time.Minute, 15},   // max 15 failed attempts per 15 minutes
		{1 * time.Hour, 30},      // max 30 failed attempts per hour
	}
)

func loginRateLimit(ip string) bool {
	return loginLimiter.check(ip, loginWindows)
}

var (
	registerLimiter = newRateLimiter()
	registerWindows = []rateWindow{
		{10 * time.Minute, 5},   // 5 registration attempts per 10 minutes
		{1 * time.Hour, 15},     // 15 per hour
	}
)

// Register godoc
// POST /api/user/register
func Register(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required,min=3,max=32"`
		Email    string `json:"email" binding:"required,email"`
		Password string `json:"password" binding:"required,min=8"`
		Code     string `json:"code"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	email := strings.ToLower(strings.TrimSpace(req.Email))

	// Verify email code (if enabled)
	emailVerificationEnabled := true
	if v, err := model.GetSetting("email_verification_enabled"); err == nil {
		emailVerificationEnabled = v == "true"
	}
	if emailVerificationEnabled {
		if req.Code == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "verification code is required"})
			return
		}
		if !registerLimiter.check(c.ClientIP(), registerWindows) {
			c.JSON(http.StatusTooManyRequests, gin.H{"error": "too many registration attempts, please try later"})
			return
		}
		if !model.VerifyCode(email, req.Code) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid or expired verification code"})
			return
		}
	}

	// Check uniqueness
	if _, err := model.GetUserByUsername(req.Username); err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "username already exists"})
		return
	}
	if _, err := model.GetUserByEmail(email); err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "email already registered"})
		return
	}

	user := &model.User{
		Username: req.Username,
		Email:    email,
		Role:     model.RoleUser,
		Status:   model.StatusEnabled,
	}
	if err := user.SetPassword(req.Password); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to hash password"})
		return
	}

		// Give register bonus if configured
		if v, err := model.GetSetting("register_bonus_balance"); err == nil {
			if bonus, err := strconv.ParseFloat(v, 64); err == nil && bonus > 0 {
				user.Balance = bonus
			}
		}
	if err := model.DB.Create(user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create user"})
		return
	}

	if emailVerificationEnabled {
		model.DeleteVerificationCode(email)
	}

	c.JSON(http.StatusOK, gin.H{"message": "registration successful"})
}

// Login godoc
// POST /api/user/login
func Login(c *gin.Context) {
	ip, _, _ := net.SplitHostPort(c.Request.RemoteAddr)
	if ip == "" {
		ip = c.ClientIP()
	}
	if !loginRateLimit(ip) {
		c.JSON(http.StatusTooManyRequests, gin.H{"error": "too many login attempts, try again later"})
		return
	}

	var req struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, err := model.GetUserByUsername(req.Username)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid username or password"})
		return
	}
	if !user.CheckPassword(req.Password) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid username or password"})
		return
	}
	if user.Status != model.StatusEnabled {
		c.JSON(http.StatusForbidden, gin.H{"error": "account is disabled"})
		return
	}

	token, err := middleware.GenerateToken(user.ID, user.Role, user.TokenVersion)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate token"})
		return
	}

	// Set HttpOnly cookie for SPA (XSS can't steal it)
	expireHours := config.C.JWT.ExpireHours
	if expireHours <= 0 {
		expireHours = 168
	}
	// Mark cookie Secure if TLS is present OR behind a trusted reverse proxy
	secure := c.Request.TLS != nil || c.GetHeader("X-Forwarded-Proto") == "https"
	http.SetCookie(c.Writer, &http.Cookie{
			Name:     "auth_token",
			Value:    token,
			Path:     "/",
			MaxAge:   expireHours * 3600,
			HttpOnly: true,
			Secure:   secure,
			SameSite: http.SameSiteStrictMode,
		})

	c.JSON(http.StatusOK, gin.H{
		"token": token, // kept for API clients that can't use cookies
		"user": gin.H{
			"id":       user.ID,
			"username": user.Username,
			"email":    user.Email,
			"role":     user.Role,
			"balance":  user.Balance,
		},
	})
}

// Logout clears the auth cookie.
// POST /api/user/logout
func Logout(c *gin.Context) {
	http.SetCookie(c.Writer, &http.Cookie{
			Name:     "auth_token",
			Value:    "",
			Path:     "/",
			MaxAge:   -1,
			HttpOnly: true,
			SameSite: http.SameSiteStrictMode,
		})
	c.JSON(http.StatusOK, gin.H{"message": "logged out"})
}

// GetUserInfo godoc
// GET /api/user/info
func GetUserInfo(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	stats := model.GetDashboardStats(user.ID)
	tokenCount := int64(0)
	model.DB.Model(&model.Token{}).Where("user_id = ? AND status = 1", user.ID).Count(&tokenCount)

	c.JSON(http.StatusOK, gin.H{
		"id":           user.ID,
		"username":     user.Username,
		"email":        user.Email,
		"role":         user.Role,
		"balance":      user.Balance,
		"token_count":  tokenCount,
		"stats":        stats,
	})
}

// UpdatePassword godoc
// POST /api/user/update-password
func UpdatePassword(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	var req struct {
		OldPassword string `json:"old_password" binding:"required"`
		NewPassword string `json:"new_password" binding:"required,min=8"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if !user.CheckPassword(req.OldPassword) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "incorrect current password"})
		return
	}
	if err := user.SetPassword(req.NewPassword); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to hash password"})
		return
	}
	// Increment token version to invalidate all existing JWTs
	user.TokenVersion++
	if err := model.DB.Save(user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update password"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "password updated, all other sessions logged out"})
}
