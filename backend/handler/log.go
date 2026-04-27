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

	resp := gin.H{
		"stats":       stats,
		"trend":       trend,
		"token_count": tokenCount,
		"balance":     user.Balance,
	}

	// Admin sees system-wide data
	if user.Role == model.RoleAdmin {
		var totalUsers, activeChannels int64
		model.DB.Model(&model.User{}).Count(&totalUsers)
		model.DB.Model(&model.Channel{}).Where("status = 1").Count(&activeChannels)
		resp["total_users"] = totalUsers
		resp["active_channels"] = activeChannels

		// System-wide stats
		sysStats := model.GetDashboardStats(0)
		resp["sys_stats"] = sysStats

		// Top 5 models (exclude empty model from /v1/models requests)
		type modelRank struct {
			Model string `json:"model"`
			Count int64  `json:"count"`
		}
		var topModels []modelRank
		model.DB.Model(&model.Log{}).
			Where("model != ''").
			Select("model, COUNT(*) as count").
			Group("model").Order("count desc").Limit(5).
			Scan(&topModels)
		resp["top_models"] = topModels
	}

	c.JSON(http.StatusOK, resp)
}

// DailyCost represents aggregated cost for a single day.
type DailyCost struct {
	Date         string  `json:"date"`
	TotalCost    float64 `json:"total_cost"`
	RequestCount int64   `json:"request_count"`
}

// GetDailyCosts godoc
// GET /api/admin/daily-costs — daily aggregated consumption
func GetDailyCosts(c *gin.Context) {
	var rows []DailyCost
	model.DB.Model(&model.Log{}).
		Select("DATE(created_at) as date, COALESCE(SUM(cost),0) as total_cost, COUNT(*) as request_count").
		Group("DATE(created_at)").Order("date desc").Limit(30).
		Scan(&rows)
	c.JSON(http.StatusOK, gin.H{"data": rows})
}

// AdminStats godoc
// GET /api/admin/stats — system-wide usage statistics
func AdminStats(c *gin.Context) {
	type dailyTrend struct {
		Date  string  `json:"date"`
		Calls int64   `json:"calls"`
		Cost  float64 `json:"cost"`
	}

	today := time.Now().Truncate(24 * time.Hour)

	buildTrend := func(days int) []dailyTrend {
		result := make([]dailyTrend, days)
		for i := days - 1; i >= 0; i-- {
			d := today.AddDate(0, 0, -i)
			var cnt int64
			var cost float64
			model.DB.Model(&model.Log{}).
				Where("created_at >= ? AND created_at < ?", d, d.Add(24*time.Hour)).
				Count(&cnt)
			model.DB.Model(&model.Log{}).
				Where("created_at >= ? AND created_at < ?", d, d.Add(24*time.Hour)).
				Select("COALESCE(SUM(cost),0)").Scan(&cost)
			result[days-1-i] = dailyTrend{Date: d.Format("01-02"), Calls: cnt, Cost: cost}
		}
		return result
	}

	var totalUsers, activeChannels, todayCalls, totalCalls int64
	var totalRevenue float64

	model.DB.Model(&model.User{}).Count(&totalUsers)
	model.DB.Model(&model.Channel{}).Where("status = 1").Count(&activeChannels)
	model.DB.Model(&model.Log{}).Where("created_at >= ?", today).Count(&todayCalls)
	model.DB.Model(&model.Log{}).Count(&totalCalls)
	model.DB.Model(&model.Log{}).Select("COALESCE(SUM(cost),0)").Scan(&totalRevenue)

	// Top models (exclude empty model from /v1/models requests)
	type modelRank struct {
		Model string `json:"model"`
		Count int64  `json:"count"`
	}
	var topModels []modelRank
	model.DB.Model(&model.Log{}).
		Where("model != ''").
		Select("model, COUNT(*) as count").
		Group("model").Order("count desc").Limit(10).
		Scan(&topModels)

	c.JSON(http.StatusOK, gin.H{
		"trend_7d":        buildTrend(7),
		"trend_30d":       buildTrend(30),
		"total_users":     totalUsers,
		"active_channels": activeChannels,
		"today_calls":     todayCalls,
		"total_calls":     totalCalls,
		"total_revenue":   totalRevenue,
		"top_models":      topModels,
	})
}
