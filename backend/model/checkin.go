package model

import "time"

type CheckInRecord struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	UserID    uint      `gorm:"uniqueIndex:idx_user_date;not null" json:"user_id"`
	Date      string    `gorm:"uniqueIndex:idx_user_date;size:10;not null" json:"date"`
	Reward    float64   `json:"reward"`
	CreatedAt time.Time `json:"created_at"`
}
