package handler

import (
	"fmt"
	"net/http"
	"new-api-lite/middleware"
	"new-api-lite/model"
	"regexp"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// htmlSanitize strips dangerous HTML elements and attributes.
// Guards against stored XSS when admin-written notice content is rendered in browsers.
var (
	reScript   = regexp.MustCompile(`(?is)<script[^>]*>.*?</script>`)
	reStyle    = regexp.MustCompile(`(?is)<style[^>]*>.*?</style>`)
	reDanger   = regexp.MustCompile(`(?is)</?(?:iframe|object|embed|link|meta|base)[^>]*>`)
	reOnAttr   = regexp.MustCompile(`(?i)\s+on\w+\s*=\s*("[^"]*"|'[^']*'|[^\s>]+)`)
	reJSUri    = regexp.MustCompile(`(?i)(href|src)\s*=\s*"(?:javascript|data)\s*:`)
	reJSUriSq  = regexp.MustCompile(`(?i)(href|src)\s*=\s*'(?:javascript|data)\s*:`)
)

func sanitizeHTML(s string) string {
	s = reScript.ReplaceAllString(s, "")
	s = reStyle.ReplaceAllString(s, "")
	s = reDanger.ReplaceAllString(s, "")
	s = reOnAttr.ReplaceAllString(s, "")
	s = reJSUri.ReplaceAllString(s, `$1="#"`)
	s = reJSUriSq.ReplaceAllString(s, `$1='#'`)
	return s
}


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
		Title:    sanitizeHTML(req.Title),
		Content:  sanitizeHTML(req.Content),
		Priority: req.Priority,
		Status:   1,
	}
	if err := model.DB.Create(&n).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create notice"})
		return
	}
	audit(c, "create_notice", fmt.Sprintf("title=%s", n.Title))
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
		"title":    sanitizeHTML(req.Title),
		"content":  sanitizeHTML(req.Content),
		"priority": req.Priority,
	}
	if err := model.DB.Model(&notice).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update notice"})
		return
	}
	audit(c, "update_notice", fmt.Sprintf("id=%d title=%s", id, req.Title))
	c.JSON(http.StatusOK, gin.H{"data": notice})
}

// DeleteNotice deletes a notice (admin).
// DELETE /api/admin/notice/:id
func DeleteNotice(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	model.DB.Delete(&model.Notice{}, id)
	audit(c, "delete_notice", fmt.Sprintf("id=%d", id))
	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}

// GetActiveNotices returns active notices for the current user (auth).
// GET /api/notice
func GetActiveNotices(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	var list []model.Notice
	model.DB.Where("status = 1 AND (user_id IS NULL OR user_id = ?)", user.ID).Order("priority desc, id desc").Find(&list)
	if list == nil {
		list = []model.Notice{}
	}
	c.JSON(http.StatusOK, gin.H{"data": list})
}

// checkUsageThresholds creates per-user notifications when usage or balance thresholds are exceeded.
func checkUsageThresholds(user *model.User) {
	today := time.Now().Truncate(24 * time.Hour)

	// Check daily cost threshold
	dailyWarn := 0.50
	if v, err := model.GetSetting("usage_notify_daily_cost"); err == nil {
		if d, err := strconv.ParseFloat(v, 64); err == nil {
			dailyWarn = d
		}
	}
	if dailyWarn > 0 {
		var todayCost float64
		model.DB.Model(&model.Log{}).
			Where("user_id = ? AND created_at >= ?", user.ID, today).
			Select("COALESCE(SUM(cost),0)").Scan(&todayCost)
		if todayCost >= dailyWarn {
			var existing int64
			model.DB.Model(&model.Notice{}).
				Where("user_id = ? AND type = ? AND created_at >= ?", user.ID, "usage", today).Count(&existing)
			if existing == 0 {
				model.DB.Create(&model.Notice{
					Title:    "用量提醒",
					Content:  fmt.Sprintf("您今日的 API 消费已达到 $%.4f，请注意控制用量。", todayCost),
					Priority: 5,
					Status:   1,
					UserID:   &user.ID,
					Type:     "usage",
				})
			}
		}
	}

	// Check low balance threshold
	balanceWarn := 1.0
	if v, err := model.GetSetting("usage_notify_balance_dollars"); err == nil {
		if d, err := strconv.ParseFloat(v, 64); err == nil {
			balanceWarn = d
		}
	}
	if balanceWarn > 0 && user.Balance <= balanceWarn {
		var existing int64
		model.DB.Model(&model.Notice{}).
			Where("user_id = ? AND type = ? AND created_at >= ?", user.ID, "balance", today).Count(&existing)
		if existing == 0 {
			model.DB.Create(&model.Notice{
				Title:    "余额不足提醒",
				Content:  fmt.Sprintf("您的账户余额仅剩 $%.4f，请及时充值以免影响使用。", user.Balance),
				Priority: 10,
				Status:   1,
				UserID:   &user.ID,
				Type:     "balance",
			})
		}
	}
}

