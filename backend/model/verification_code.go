package model

import (
	"time"
)

type VerificationCode struct {
	Email     string    `gorm:"primarykey;size:128" json:"email"`
	Code      string    `gorm:"size:8;not null" json:"code"`
	ExpiredAt time.Time `json:"expired_at"`
	CreatedAt time.Time `json:"created_at"`
}

func SaveVerificationCode(email, code string, ttl time.Duration) error {
	vc := VerificationCode{
		Email:     email,
		Code:      code,
		ExpiredAt: time.Now().Add(ttl),
		CreatedAt: time.Now(),
	}
	// Upsert: replace if exists
	return DB.Save(&vc).Error
}

func VerifyCode(email, code string) bool {
	var vc VerificationCode
	if err := DB.First(&vc, "email = ?", email).Error; err != nil {
		return false
	}
	if time.Now().After(vc.ExpiredAt) {
		return false
	}
	return vc.Code == code
}

func DeleteVerificationCode(email string) {
	DB.Delete(&VerificationCode{}, "email = ?", email)
}
