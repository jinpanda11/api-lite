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
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": tokens})
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
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
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
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
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
		updates["status"] = *req.Status
	}
	if req.ExpiredAt != nil {
		updates["expired_at"] = req.ExpiredAt
	}

	if err := model.DB.Model(&token).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": token})
}

// DeleteToken godoc
// DELETE /api/token/:id
func DeleteToken(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	id, _ := strconv.Atoi(c.Param("id"))

	if err := model.DB.Where("id = ? AND user_id = ?", id, user.ID).Delete(&model.Token{}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}
