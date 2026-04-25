package handler

import (
	"net/http"
	"new-api-lite/model"
	"strings"

	"github.com/gin-gonic/gin"
)

// ListModelPricing returns all global model pricing configs.
// GET /api/admin/model-pricing
func ListModelPricing(c *gin.Context) {
	var list []model.ModelPricing
	model.DB.Find(&list)
	if list == nil {
		list = []model.ModelPricing{}
	}
	c.JSON(http.StatusOK, gin.H{"data": list})
}

// UpdateModelPricing creates or updates pricing for a specific model.
// PUT /api/admin/model-pricing/:modelName
func UpdateModelPricing(c *gin.Context) {
	modelName := c.Param("modelName")
	modelName = strings.TrimSpace(modelName)
	if modelName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "model name is required"})
		return
	}

	var req model.ModelPricing
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	req.ModelName = modelName

	var existing model.ModelPricing
	if err := model.DB.Where("model_name = ?", modelName).First(&existing).Error; err == nil {
		req.ID = existing.ID
		model.DB.Save(&req)
	} else {
		model.DB.Create(&req)
	}

	c.JSON(http.StatusOK, gin.H{"data": req})
}
