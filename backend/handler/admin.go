package handler

import (
	"crypto/rand"
	"math/big"
	"net/http"
	"new-api-lite/model"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// ListRedeemCodes godoc
// GET /api/admin/redeem
func ListRedeemCodes(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize := 20
	var codes []model.RedeemCode
	model.DB.Order("created_at desc").Offset((page - 1) * pageSize).Limit(pageSize).Find(&codes)
	var total int64
	model.DB.Model(&model.RedeemCode{}).Count(&total)
	c.JSON(http.StatusOK, gin.H{"data": codes, "total": total})
}

// CreateRedeemCodes godoc
// POST /api/admin/redeem  { count: 5, value: 5.00 }
func CreateRedeemCodes(c *gin.Context) {
	var req struct {
		Count int     `json:"count" binding:"required,min=1,max=100"`
		Value float64 `json:"value" binding:"required,gt=0"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	codes := make([]model.RedeemCode, req.Count)
	for i := range codes {
		codes[i] = model.RedeemCode{
			Code:   generateCode(16),
			Value:  req.Value,
			Status: model.StatusEnabled,
		}
	}
	if err := model.DB.Create(&codes).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": codes, "message": "created"})
}

// DeleteRedeemCode godoc
// DELETE /api/admin/redeem/:id
func DeleteRedeemCode(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	model.DB.Delete(&model.RedeemCode{}, id)
	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}

// DeleteUsedRedeemCodes godoc
// DELETE /api/admin/redeem/used
func DeleteUsedRedeemCodes(c *gin.Context) {
	result := model.DB.Where("status != 1").Delete(&model.RedeemCode{})
	c.JSON(http.StatusOK, gin.H{"message": "deleted", "count": result.RowsAffected})
}

// ListUsers godoc
// GET /api/admin/user
func ListUsers(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize := 20
	var users []model.User
	model.DB.Select("id,username,email,role,balance,status,price_multiplier,created_at").
		Order("created_at desc").Offset((page-1)*pageSize).Limit(pageSize).Find(&users)
	var total int64
	model.DB.Model(&model.User{}).Count(&total)
	c.JSON(http.StatusOK, gin.H{"data": users, "total": total})
}

// UpdateUserStatus godoc
// PUT /api/admin/user/:id
func UpdateUserStatus(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var req struct {
		Status          *int     `json:"status"`
		Balance         *float64 `json:"balance"`
		Role            string   `json:"role"`
		PriceMultiplier *float64 `json:"price_multiplier"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Verify user exists
	var target model.User
	if err := model.DB.First(&target, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	updates := map[string]interface{}{}
	if req.Status != nil {
		updates["status"] = *req.Status
	}
	if req.Balance != nil {
		updates["balance"] = *req.Balance
	}
	if req.Role != "" {
		if req.Role != "user" && req.Role != "admin" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "role must be 'user' or 'admin'"})
			return
		}
		updates["role"] = req.Role
	}
	if req.PriceMultiplier != nil {
		if *req.PriceMultiplier <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "price_multiplier must be positive"})
			return
		}
		updates["price_multiplier"] = *req.PriceMultiplier
	}
	// If disabling the user, invalidate all their JWTs
	if req.Status != nil && *req.Status == 0 {
		updates["token_version"] = gorm.Expr("token_version + 1")
	}
	if err := model.DB.Model(&model.User{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "updated"})
}

var codeChars = []rune("ABCDEFGHJKLMNPQRSTUVWXYZ23456789")

func generateCode(n int) string {
	b := make([]rune, n)
	for i := range b {
		idx, err := rand.Int(rand.Reader, big.NewInt(int64(len(codeChars))))
		if err != nil {
			idx = big.NewInt(0)
		}
		b[i] = codeChars[idx.Int64()]
	}
	return string(b)
}
