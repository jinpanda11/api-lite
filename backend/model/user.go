package model

import (
	"strconv"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

const (
	RoleUser  = "user"
	RoleAdmin = "admin"

	StatusEnabled  = 1
)

type User struct {
	ID           uint           `gorm:"primarykey" json:"id"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"-"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
	Username     string         `gorm:"uniqueIndex;size:64;not null" json:"username"`
	Email        string         `gorm:"uniqueIndex;size:128" json:"email"`
	PasswordHash string         `gorm:"size:128;not null" json:"-"`
	Role         string         `gorm:"size:16;default:user" json:"role"`
	Balance         float64        `gorm:"default:0" json:"balance"`
	Status          int            `gorm:"default:1" json:"status"`
	PriceMultiplier float64        `gorm:"default:1.0" json:"price_multiplier"`
	TokenVersion    int            `gorm:"default:0" json:"-"` // incremented to invalidate JWTs
	DailyDrawQuota    int    `gorm:"default:10" json:"daily_draw_quota"`
	LastDrawResetDate string `gorm:"size:10" json:"-"`
}

const DefaultDailyDrawQuota = 10

func (u *User) SetPassword(password string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	u.PasswordHash = string(hash)
	return nil
}

func (u *User) CheckPassword(password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)) == nil
}

func GetUserByID(id uint) (*User, error) {
	var user User
	if err := DB.First(&user, id).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func GetUserByUsername(username string) (*User, error) {
	var user User
	if err := DB.Where("username = ?", username).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func GetUserByEmail(email string) (*User, error) {
	var user User
	if err := DB.Where("email = ?", email).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

// DeductBalance atomically subtracts amount from balance, only if sufficient.
// Returns true if the deduction succeeded.
func (u *User) DeductBalance(amount float64) bool {
	result := DB.Model(u).Where("balance >= ?", amount).UpdateColumn("balance", gorm.Expr("balance - ?", amount))
	return result.RowsAffected > 0
}

func (u *User) AddBalance(amount float64) error {
	return DB.Model(u).UpdateColumn("balance", gorm.Expr("balance + ?", amount)).Error
}

func (u *User) GetDrawQuota() int {
	today := time.Now().Format("2006-01-02")
	if u.LastDrawResetDate != today {
		quota := DefaultDailyDrawQuota
		if v, err := GetSetting("daily_draw_quota"); err == nil && v != "" {
			if n, err2 := strconv.Atoi(v); err2 == nil && n > 0 {
				quota = n
			}
		}
		DB.Model(u).Updates(map[string]interface{}{
			"daily_draw_quota":     quota,
			"last_draw_reset_date": today,
		})
		u.DailyDrawQuota = quota
		u.LastDrawResetDate = today
		return quota
	}
	return u.DailyDrawQuota
}

func (u *User) DeductDrawQuota() bool {
	// Atomic: only decrement if quota > 0, avoids TOCTOU race
	result := DB.Model(u).Where("daily_draw_quota > 0").UpdateColumn("daily_draw_quota", gorm.Expr("daily_draw_quota - 1"))
	if result.RowsAffected > 0 {
		u.DailyDrawQuota--
		return true
	}
	return false
}
