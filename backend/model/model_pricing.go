package model

type ModelPricing struct {
	ID               uint    `gorm:"primarykey" json:"id"`
	ModelName        string  `gorm:"uniqueIndex;size:256;not null" json:"model_name"`
	BillingMode      string  `gorm:"size:20;not null;default:token" json:"billing_mode"` // "token" or "call"
	IconURL          string  `gorm:"size:512" json:"icon_url"`
	InputPrice       float64 `gorm:"default:0" json:"input_price"`
	OutputPrice      float64 `gorm:"default:0" json:"output_price"`
	CacheReadPrice   float64 `gorm:"default:0" json:"cache_read_price"`
	CacheCreatePrice float64 `gorm:"default:0" json:"cache_create_price"`
	CallPrice        float64 `gorm:"default:0" json:"call_price"`
}
