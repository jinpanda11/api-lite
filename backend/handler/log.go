package handler

import (
	"net/http"
	"new-api-lite/middleware"
	"new-api-lite/model"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// GetLogs godoc
// GET /api/log
func GetLogs(c *gin.Context) {
	user := middleware.GetCurrentUser(c)

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	modelFilter := c.Query("model")

	var startTime, endTime time.Time
	if s := c.Query("start_time"); s != "" {
		startTime, _ = time.Parse(time.RFC3339, s)
	}
	if e := c.Query("end_time"); e != "" {
		endTime, _ = time.Parse(time.RFC3339, e)
	}

	// Admin can query all logs; regular users see only their own
	uid := user.ID
	if user.Role == model.RoleAdmin && c.Query("all") == "1" {
		uid = 0
	}

	logs, total, err := model.GetLogs(model.LogQuery{
		UserID:    uid,
		Model:     modelFilter,
		StartTime: startTime,
		EndTime:   endTime,
		Page:      page,
		PageSize:  pageSize,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"data":      logs,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// GetDashboard godoc
// GET /api/dashboard
func GetDashboard(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	stats := model.GetDashboardStats(user.ID)
	trend := model.GetDailyRequestCounts(user.ID, 7)

	var tokenCount int64
	model.DB.Model(&model.Token{}).Where("user_id = ? AND status = 1", user.ID).Count(&tokenCount)

	c.JSON(http.StatusOK, gin.H{
		"stats":       stats,
		"trend":       trend,
		"token_count": tokenCount,
		"balance":     user.Balance,
	})
}
