package handler

import (
	"net/http"
	"new-api-lite/model"
	"strconv"

	"github.com/gin-gonic/gin"
)

// ListNotices returns all notices (admin).
// GET /api/admin/notice
func ListNotices(c *gin.Context) {
	var list []model.Notice
	model.DB.Order("priority desc, id desc").Find(&list)
	if list == nil {
		list = []model.Notice{}
	}
	c.JSON(http.StatusOK, gin.H{"data": list})
}

// CreateNotice creates a notice (admin).
// POST /api/admin/notice
func CreateNotice(c *gin.Context) {
	var req model.Notice
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	req.Status = 1
	if err := model.DB.Create(&req).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": req})
}

// UpdateNotice updates a notice (admin).
// PUT /api/admin/notice/:id
func UpdateNotice(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var notice model.Notice
	if err := model.DB.First(&notice, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "notice not found"})
		return
	}
	var req model.Notice
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	req.ID = notice.ID
	if err := model.DB.Save(&req).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": req})
}

// DeleteNotice deletes a notice (admin).
// DELETE /api/admin/notice/:id
func DeleteNotice(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	model.DB.Delete(&model.Notice{}, id)
	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}

// GetActiveNotices returns active notices for the current user (auth).
// GET /api/notice
func GetActiveNotices(c *gin.Context) {
	var list []model.Notice
	model.DB.Where("status = 1").Order("priority desc, id desc").Find(&list)
	if list == nil {
		list = []model.Notice{}
	}
	c.JSON(http.StatusOK, gin.H{"data": list})
}
