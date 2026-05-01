package handler

import (
	"fmt"
	"net/http"
	"new-api-lite/model"
	"strconv"

	"github.com/gin-gonic/gin"
)

// numericBounds defines min/max for settings that hold numeric values.
type numericBounds struct {
	min, max float64
}

var numericSettings = map[string]numericBounds{
	"checkin_reward":               {0, 100},     // daily check-in (dollars)
	"register_bonus_balance":       {0, 10000},   // registration bonus (dollars)
	"usage_notify_daily_cost":      {0, 100000},  // daily cost warning threshold
	"usage_notify_balance_dollars": {0, 100000},  // low balance warning threshold
	"monitor_interval_seconds":     {30, 86400},  // 30s to 24h
	"monitor_alert_threshold":      {1, 100},     // consecutive failures before alert
	"site_logo_size":               {12, 200},    // px
	"site_name_size":               {12, 48},     // px
}

var boolSettings = map[string]bool{
	"email_verification_enabled": true,
	"monitor_enabled":            true,
}

func validateSettingValue(key, value string) error {
	if v, ok := numericSettings[key]; ok {
		f, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return fmt.Errorf("%s must be a number", key)
		}
		if f < v.min || f > v.max {
			return fmt.Errorf("%s must be between %g and %g", key, v.min, v.max)
		}
	}
	if _, ok := boolSettings[key]; ok {
		if value != "true" && value != "false" {
			return fmt.Errorf("%s must be 'true' or 'false'", key)
		}
	}
	return nil
}

// GetBranding returns public site branding info.
// GET /api/settings/branding
func GetBranding(c *gin.Context) {
	settings, err := model.GetAllSettings()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load settings"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"site_name":           settings["site_name"],
		"site_logo":           settings["site_logo"],
		"site_title":          settings["site_title"],
		"site_favicon":        settings["site_favicon"],
		"site_logo_size":      settings["site_logo_size"],
		"site_name_size":      settings["site_name_size"],
		"redeem_purchase_url": settings["redeem_purchase_url"],
		"draw_ad_code":        settings["draw_ad_code"],
		"analytics_code":      settings["analytics_code"],
	})
}

// sensitiveKeys holds setting keys that must not be returned in full.
var sensitiveKeys = map[string]bool{
	"smtp_password": true,
}

// GetSettings returns all system settings (admin).
// GET /api/admin/settings
func GetSettings(c *gin.Context) {
	settings, err := model.GetAllSettings()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load settings"})
		return
	}
	if _, ok := settings["email_verification_enabled"]; !ok {
		settings["email_verification_enabled"] = "true"
	}
	// Mask sensitive values
	for k := range settings {
		if sensitiveKeys[k] && settings[k] != "" {
			settings[k] = "****"
		}
	}
	c.JSON(http.StatusOK, gin.H{"data": settings})
}

// UpdateSettings saves system settings (admin).
// PUT /api/admin/settings
func UpdateSettings(c *gin.Context) {
	var req struct {
		Settings map[string]string `json:"settings" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "internal error"})
		return
	}
	keysList := make([]string, 0, len(req.Settings))
	for key, value := range req.Settings {
		if err := validateSettingValue(key, value); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "internal error"})
			return
		}
		if err := model.SetSetting(key, value); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save setting: " + key})
			return
		}
		keysList = append(keysList, key)
	}
	audit(c, "update_settings", fmt.Sprintf("keys=%v", keysList))
	c.JSON(http.StatusOK, gin.H{"message": "settings updated"})
}
