package handler

import (
	"net/http"
	"new-api-lite/middleware"
	"new-api-lite/model"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// ListTokens godoc
// GET /api/token
func ListTokens(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	tokens, err := model.GetTokensByUserID(user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": tokens})
}

// GetOrCreateChatToken returns the user's first enabled token, or creates one.
// GET /api/token/chat-key
func GetOrCreateChatToken(c *gin.Context) {
	user := middleware.GetCurrentUser(c)

	tokens, err := model.GetTokensByUserID(user.ID)
	if err == nil {
		for _, t := range tokens {
			if t.Status == model.StatusEnabled {
				c.JSON(http.StatusOK, gin.H{"status": "Success", "data": gin.H{"key": t.Key}})
				return
			}
		}
	}

	// No enabled token — auto-create one
	key, err := model.GenerateTokenKey()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate token"})
		return
	}

	token := model.Token{
		UserID: user.ID,
		Key:    key,
		Name:   "chat",
		Status: model.StatusEnabled,
	}
	if err := model.DB.Create(&token).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create token"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "Success", "data": gin.H{"key": key}})
}

// CreateToken godoc
// POST /api/token
func CreateToken(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	var req struct {
		Name      string     `json:"name" binding:"required,max=64"`
		Remark    string     `json:"remark"`
		ExpiredAt *time.Time `json:"expired_at"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "internal error"})
		return
	}

	key, err := model.GenerateTokenKey()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate key"})
		return
	}

	token := model.Token{
		UserID:    user.ID,
		Key:       key,
		Name:      req.Name,
		Remark:    req.Remark,
		ExpiredAt: req.ExpiredAt,
		Status:    model.StatusEnabled,
	}
	if err := model.DB.Create(&token).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": token})
}

// UpdateToken godoc
// PUT /api/token/:id
func UpdateToken(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	id, _ := strconv.Atoi(c.Param("id"))

	var token model.Token
	if err := model.DB.Where("id = ? AND user_id = ?", id, user.ID).First(&token).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "token not found"})
		return
	}

	var req struct {
		Name      string     `json:"name"`
		Remark    string     `json:"remark"`
		Status    *int       `json:"status"`
		ExpiredAt *time.Time `json:"expired_at"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "internal error"})
		return
	}

	updates := map[string]interface{}{}
	if req.Name != "" {
		updates["name"] = req.Name
	}
	if req.Remark != "" {
		updates["remark"] = req.Remark
	}
	if req.Status != nil {
		if *req.Status != 0 && *req.Status != 1 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "status must be 0 or 1"})
			return
		}
		updates["status"] = *req.Status
	}
	if req.ExpiredAt != nil {
		updates["expired_at"] = req.ExpiredAt
	}

	if err := model.DB.Model(&token).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}
	model.CacheInvalidateToken(token.Key)
	c.JSON(http.StatusOK, gin.H{"data": token})
}

// DeleteToken godoc
// DELETE /api/token/:id
func DeleteToken(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	id, _ := strconv.Atoi(c.Param("id"))

	var token model.Token
	if err := model.DB.Where("id = ? AND user_id = ?", id, user.ID).First(&token).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "token not found"})
		return
	}
	if err := model.DB.Delete(&token).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}
	model.CacheInvalidateToken(token.Key)
	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}
