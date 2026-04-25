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
		query = query.Where("models = '' OR models LIKE ?", "%"+model+"%")
	}
	if err := query.Order("priority desc").First(&channel).Error; err != nil {
		return nil, err
	}
	return &channel, nil
}
