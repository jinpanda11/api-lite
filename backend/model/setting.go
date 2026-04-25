package model

// Setting stores key-value system settings (SMTP, feature flags, etc.)
type Setting struct {
	Key   string `gorm:"primarykey;size:100" json:"key"`
	Value string `gorm:"size:500" json:"value"`
}

// GetSetting retrieves a setting value by key.
func GetSetting(key string) (string, error) {
	var s Setting
	err := DB.First(&s, "key = ?", key).Error
	return s.Value, err
}

// SetSetting creates or updates a setting.
func SetSetting(key, value string) error {
	return DB.Save(&Setting{Key: key, Value: value}).Error
}

// GetAllSettings returns all settings as a map.
func GetAllSettings() (map[string]string, error) {
	var settings []Setting
	if err := DB.Find(&settings).Error; err != nil {
		return nil, err
	}
	result := make(map[string]string, len(settings))
	for _, s := range settings {
		result[s.Key] = s.Value
	}
	return result, nil
}
