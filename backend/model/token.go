package model

import (
	"crypto/rand"
	"encoding/hex"
	"time"

	"gorm.io/gorm"
)

type Token struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"-"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
	UserID    uint           `gorm:"index;not null" json:"user_id"`
	Key       string         `gorm:"uniqueIndex;size:64;not null" json:"key"`
	Name      string         `gorm:"size:64" json:"name"`
	Remark    string         `gorm:"size:256" json:"remark"`
	ExpiredAt *time.Time     `json:"expired_at"`
	Status    int            `gorm:"default:1" json:"status"`
}

func GenerateTokenKey() (string, error) {
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return "sk-" + hex.EncodeToString(b), nil
}

func GetTokenByKey(key string) (*Token, error) {
	var token Token
	if err := DB.Where("key = ? AND status = 1", key).First(&token).Error; err != nil {
		return nil, err
	}
	if token.ExpiredAt != nil && token.ExpiredAt.Before(time.Now()) {
		return nil, gorm.ErrRecordNotFound
	}
	return &token, nil
}

func GetTokensByUserID(userID uint) ([]Token, error) {
	var tokens []Token
	if err := DB.Where("user_id = ?", userID).Order("created_at desc").Find(&tokens).Error; err != nil {
		return nil, err
	}
	return tokens, nil
}
