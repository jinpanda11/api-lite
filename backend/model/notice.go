package model

import "time"

type Notice struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	Title     string    `gorm:"size:256;not null" json:"title"`
	Content   string    `gorm:"size:4096;not null" json:"content"` // supports HTML
	Priority  int       `gorm:"default:0" json:"priority"`         // higher = show first
	Status    int       `gorm:"default:1" json:"status"`           // 1=active, 0=disabled
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
