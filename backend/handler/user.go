package handler

import (
	"net/http"
	"new-api-lite/middleware"
	"new-api-lite/model"
	"new-api-lite/service"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// emailLimiter is a simple in-memory rate limiter (email -> last send time).
var (
	emailLimiter   = map[string]time.Time{}
	emailLimiterMu sync.Mutex
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

	// Rate limit: 60 seconds per email
	emailLimiterMu.Lock()
	if last, ok := emailLimiter[email]; ok && time.Since(last) < 60*time.Second {
		emailLimiterMu.Unlock()
		c.JSON(http.StatusTooManyRequests, gin.H{"error": "please wait 60 seconds before requesting another code"})
		return
	}
	emailLimiter[email] = time.Now()
	emailLimiterMu.Unlock()

	code := service.GenerateCode()
	if err := model.SaveVerificationCode(email, code, 10*time.Minute); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save verification code"})
		return
	}
	if err := service.SendVerificationEmail(email, code); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to send email"})
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

	token, err := middleware.GenerateToken(user.ID, user.Role)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"token": token,
		"user": gin.H{
			"id":       user.ID,
			"username": user.Username,
			"email":    user.Email,
			"role":     user.Role,
			"balance":  user.Balance,
		},
	})
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
	if err := model.DB.Save(user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update password"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "password updated"})
}
