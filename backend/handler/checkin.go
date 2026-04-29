package handler

import (
	"net/http"
	"new-api-lite/middleware"
	"new-api-lite/model"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

func CheckIn(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	today := time.Now().Format("2006-01-02")
	reward := getCheckInReward()

	// Create the check-in record FIRST — unique constraint on (user_id, date)
	// prevents concurrent duplicate check-ins. Balance is added only after the
	// record is committed, so a DB conflict never leaves orphaned balance.
	record := model.CheckInRecord{
		UserID: user.ID,
		Date:   today,
		Reward: reward,
	}
	if err := model.DB.Create(&record).Error; err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "already checked in today"})
		return
	}

	if err := user.AddBalance(reward); err != nil {
		// Compensate: remove the record so the user can retry
		model.DB.Delete(&record)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to add balance"})
		return
	}

	model.DB.Create(&model.TopupLog{
		UserID: user.ID,
		Amount: reward,
		Remark: "daily check-in",
	})

	streak := getCheckInStreak(user.ID)

	c.JSON(http.StatusOK, gin.H{
		"message": "check-in successful",
		"reward":  reward,
		"streak":  streak,
	})
}

func GetCheckInStatus(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	today := time.Now().Format("2006-01-02")

	var count int64
	model.DB.Model(&model.CheckInRecord{}).Where("user_id = ? AND date = ?", user.ID, today).Count(&count)

	streak := getCheckInStreak(user.ID)
	reward := getCheckInReward()

	c.JSON(http.StatusOK, gin.H{
		"checked_in_today": count > 0,
		"streak":           streak,
		"today_reward":     reward,
	})
}

func getCheckInStreak(userID uint) int {
	streak := 0
	for i := 0; i < 365; i++ {
		date := time.Now().AddDate(0, 0, -i).Format("2006-01-02")
		var count int64
		model.DB.Model(&model.CheckInRecord{}).Where("user_id = ? AND date = ?", userID, date).Count(&count)
		if count == 0 {
			break
		}
		streak++
	}
	return streak
}

func getCheckInReward() float64 {
	if v, err := model.GetSetting("checkin_reward"); err == nil {
		if r, err := strconv.ParseFloat(v, 64); err == nil && r > 0 {
			return r
		}
	}
	return 0.01
}
