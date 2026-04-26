package model

import (
	"time"
)

type RedeemCode struct {
	ID        uint       `gorm:"primarykey" json:"id"`
	CreatedAt time.Time  `json:"created_at"`
	Code      string     `gorm:"uniqueIndex;size:64;not null" json:"code"`
	Value     float64    `gorm:"not null" json:"value"` // USD amount
	UsedBy    *uint      `json:"used_by"`
	UsedAt    *time.Time `json:"used_at"`
	Status    int        `gorm:"default:1" json:"status"` // 1=available, 0=used
}

func GetRedeemCodeByCode(code string) (*RedeemCode, error) {
	var rc RedeemCode
	if err := DB.Where("code = ? AND status = 1", code).First(&rc).Error; err != nil {
		return nil, err
	}
	return &rc, nil
}

func (rc *RedeemCode) MarkUsed(userID uint) bool {
	now := time.Now()
	// Atomic: only mark used if still available (prevents double-spending)
	result := DB.Model(rc).Where("status = 1").Updates(map[string]interface{}{
		"status":  0,
		"used_by": userID,
		"used_at": now,
	})
	return result.RowsAffected > 0
}

// TopupLog records balance additions for display in wallet history.
type TopupLog struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UserID    uint      `gorm:"index;not null" json:"user_id"`
	Amount    float64   `json:"amount"`
	Code      string    `gorm:"size:64" json:"code"`
	Remark    string    `gorm:"size:128" json:"remark"`
}
