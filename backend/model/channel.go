package model

import (
	"time"

	"gorm.io/gorm"
)

type Channel struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
	Name      string         `gorm:"size:64;not null" json:"name"`
	Type      string         `gorm:"size:32;default:openai" json:"type"` // openai, azure, etc.
	BaseURL   string         `gorm:"size:256;not null" json:"base_url"`
	APIKey    string         `gorm:"size:256;not null" json:"api_key"`
	Models    string         `gorm:"size:1024" json:"models"`   // comma-separated model names
	Priority  int            `gorm:"default:0" json:"priority"` // higher = preferred
	Status    int            `gorm:"default:1" json:"status"`
	FixedPath      string         `gorm:"size:128" json:"fixed_path"`
	MonitorEnabled bool           `gorm:"default:1" json:"monitor_enabled"`
}

// escapeLike escapes SQL LIKE wildcard characters so they match literally.
func escapeLike(s string) string {
	result := make([]byte, 0, len(s)+4)
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '%', '_', '\\':
			result = append(result, '\\', s[i])
		default:
			result = append(result, s[i])
		}
	}
	return string(result)
}

func GetAvailableChannels() ([]Channel, error) {
	var channels []Channel
	if err := DB.Where("status = 1").Order("priority desc").Find(&channels).Error; err != nil {
		return nil, err
	}
	return channels, nil
}

// SelectChannel picks the highest-priority available channel.
// Optionally filter by model name.
func SelectChannel(model string) (*Channel, error) {
	var channel Channel
	query := DB.Where("status = 1")
	if model != "" {
		// Escape LIKE wildcards in the model name to prevent unintended matches
		escaped := escapeLike(model)
		query = query.Where("models = '' OR models LIKE ?", "%"+escaped+"%")
	}
	if err := query.Order("priority desc").First(&channel).Error; err != nil {
		return nil, err
	}
	return &channel, nil
}
