package handler

import (
	"net/http"
	"new-api-lite/middleware"
	"new-api-lite/model"

	"github.com/gin-gonic/gin"
)

// GetBalance godoc
// GET /api/balance
func GetBalance(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	c.JSON(http.StatusOK, gin.H{"balance": user.Balance})
}

// Redeem godoc
// POST /api/redeem
func Redeem(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	var req struct {
		Code string `json:"code" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	rc, err := model.GetRedeemCodeByCode(req.Code)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid or already used redeem code"})
		return
	}

	// Mark code as used and add balance atomically
	if err := rc.MarkUsed(user.ID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to redeem code"})
		return
	}
	if err := user.AddBalance(rc.Value); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to add balance"})
		return
	}

	// Record topup log
	model.DB.Create(&model.TopupLog{
		UserID: user.ID,
		Amount: rc.Value,
		Code:   req.Code,
		Remark: "redeem code",
	})

	c.JSON(http.StatusOK, gin.H{
		"message": "redeemed successfully",
		"amount":  rc.Value,
		"balance": user.Balance + rc.Value,
	})
}

// GetTopupLogs godoc
// GET /api/topup/logs
func GetTopupLogs(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	var logs []model.TopupLog
	model.DB.Where("user_id = ?", user.ID).Order("created_at desc").Limit(50).Find(&logs)
	c.JSON(http.StatusOK, gin.H{"data": logs})
}
