package model

import (
	"time"
)

type Log struct {
	ID           uint      `gorm:"primarykey" json:"id"`
	CreatedAt    time.Time `json:"created_at"`
	UserID       uint      `gorm:"index" json:"user_id"`
	TokenID      uint      `json:"token_id"`
	TokenName    string    `gorm:"size:64" json:"token_name"`
	ChannelID    uint      `json:"channel_id"`
	ChannelName  string    `gorm:"size:64" json:"channel_name"`
	Model        string    `gorm:"size:64;index" json:"model"`
	InputTokens  int       `json:"input_tokens"`
	OutputTokens int       `json:"output_tokens"`
	Cost         float64   `json:"cost"`
	Status       int       `gorm:"default:1" json:"status"` // 1=success, 2=error
	RequestPath  string    `gorm:"size:128" json:"request_path"`
}

func CreateLog(l *Log) error {
	return DB.Create(l).Error
}

type LogQuery struct {
	UserID    uint
	Model     string
	StartTime time.Time
	EndTime   time.Time
	Page      int
	PageSize  int
}

func GetLogs(q LogQuery) ([]Log, int64, error) {
	var logs []Log
	var total int64

	db := DB.Model(&Log{})
	if q.UserID > 0 {
		db = db.Where("user_id = ?", q.UserID)
	}
	if q.Model != "" {
		db = db.Where("model = ?", q.Model)
	}
	if !q.StartTime.IsZero() {
		db = db.Where("created_at >= ?", q.StartTime)
	}
	if !q.EndTime.IsZero() {
		db = db.Where("created_at <= ?", q.EndTime)
	}

	db.Count(&total)

	if q.Page < 1 {
		q.Page = 1
	}
	if q.PageSize < 1 {
		q.PageSize = 20
	}
	offset := (q.Page - 1) * q.PageSize
	err := db.Order("created_at desc").Offset(offset).Limit(q.PageSize).Find(&logs).Error
	return logs, total, err
}

// DashboardStats returns summary stats for a user (or all users if userID=0 for admin).
type DashboardStats struct {
	TodayRequests int64   `json:"today_requests"`
	TotalRequests int64   `json:"total_requests"`
	TodayCost     float64 `json:"today_cost"`
	TotalCost     float64 `json:"total_cost"`
}

func GetDashboardStats(userID uint) DashboardStats {
	var stats DashboardStats
	today := time.Now().Truncate(24 * time.Hour)

	q := DB.Model(&Log{})
	if userID > 0 {
		q = q.Where("user_id = ?", userID)
	}

	q.Count(&stats.TotalRequests)
	q.Where("created_at >= ?", today).Count(&stats.TodayRequests)

	type costResult struct{ Total float64 }
	var cr costResult
	q2 := DB.Model(&Log{})
	if userID > 0 {
		q2 = q2.Where("user_id = ?", userID)
	}
	q2.Select("COALESCE(SUM(cost),0) as total").Scan(&cr)
	stats.TotalCost = cr.Total

	var cr2 costResult
	q3 := DB.Model(&Log{})
	if userID > 0 {
		q3 = q3.Where("user_id = ?", userID)
	}
	q3.Where("created_at >= ?", today).Select("COALESCE(SUM(cost),0) as total").Scan(&cr2)
	stats.TodayCost = cr2.Total

	return stats
}

// DailyCount is used for trend charts.
type DailyCount struct {
	Date  string `json:"date"`
	Count int64  `json:"count"`
}

func GetDailyRequestCounts(userID uint, days int) []DailyCount {
	result := make([]DailyCount, days)
	for i := days - 1; i >= 0; i-- {
		d := time.Now().Truncate(24 * time.Hour).AddDate(0, 0, -i)
		var cnt int64
		q := DB.Model(&Log{}).Where("created_at >= ? AND created_at < ?", d, d.Add(24*time.Hour))
		if userID > 0 {
			q = q.Where("user_id = ?", userID)
		}
		q.Count(&cnt)
		result[days-1-i] = DailyCount{Date: d.Format("01-02"), Count: cnt}
	}
	return result
}
