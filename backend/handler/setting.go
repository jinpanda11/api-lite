package handler

import (
	"net/http"
	"new-api-lite/model"

	"github.com/gin-gonic/gin"
)

// GetBranding returns public site branding info.
// GET /api/settings/branding
func GetBranding(c *gin.Context) {
	settings, err := model.GetAllSettings()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load settings"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"site_name":    settings["site_name"],
		"site_logo":    settings["site_logo"],
		"site_title":   settings["site_title"],
		"site_favicon": settings["site_favicon"],
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
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	for key, value := range req.Settings {
		if err := model.SetSetting(key, value); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save setting: " + key})
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{"message": "settings updated"})
}
