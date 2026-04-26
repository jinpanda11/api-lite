package handler

import (
	"net/http"
	"new-api-lite/model"
	"strconv"

	"github.com/gin-gonic/gin"
)


// noticeRequest limits which fields can be set during create/update.
type noticeRequest struct {
	Title    string `json:"title" binding:"required"`
	Content  string `json:"content" binding:"required"`
	Priority int    `json:"priority"`
}

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
	var req noticeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	n := model.Notice{
		Title:    req.Title,
		Content:  req.Content,
		Priority: req.Priority,
		Status:   1,
	}
	if err := model.DB.Create(&n).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create notice"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": n})
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
	var req noticeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	updates := map[string]interface{}{
		"title":    req.Title,
		"content":  req.Content,
		"priority": req.Priority,
	}
	if err := model.DB.Model(&notice).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update notice"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": notice})
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
