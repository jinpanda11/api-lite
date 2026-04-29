package handler

import (
	"log"
	"net/http"
	"new-api-lite/middleware"
	"new-api-lite/model"
	"strconv"

	"github.com/gin-gonic/gin"
)

// audit writes an admin action to both the database and server log.
func audit(c *gin.Context, action, detail string) {
	username := "unknown"
	userID := uint(0)
	if u := middleware.GetCurrentUser(c); u != nil {
		username = u.Username
		userID = u.ID
	}

	entry := model.AuditLog{
		AdminName: username,
		AdminID:   userID,
		Action:    action,
		Detail:    detail,
	}
	if err := model.DB.Create(&entry).Error; err != nil {
		log.Printf("[AUDIT] DB write failed: %v", err)
	}

	log.Printf("[AUDIT] admin=%s(id=%d) action=%s detail=%s", username, userID, action, detail)
}

// GetAuditLogs returns paginated audit log entries (admin only).
func GetAuditLogs(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	action := c.Query("action")

	var logs []model.AuditLog
	var total int64

	q := model.DB.Model(&model.AuditLog{})
	if action != "" {
		q = q.Where("action = ?", action)
	}
	q.Count(&total)

	q.Order("created_at desc").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&logs)

	if logs == nil {
		logs = []model.AuditLog{}
	}

	c.JSON(http.StatusOK, gin.H{
		"data":      logs,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}
