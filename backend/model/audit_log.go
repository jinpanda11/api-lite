package model

import "time"

type AuditLog struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	AdminName string    `gorm:"size:64;index" json:"admin_name"`
	AdminID   uint      `gorm:"index" json:"admin_id"`
	Action    string    `gorm:"size:64;index" json:"action"`
	Detail    string    `gorm:"size:1024" json:"detail"`
	CreatedAt time.Time `json:"created_at"`
}
