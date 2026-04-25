package model

import (
	"fmt"
	"log"
	"new-api-lite/config"

	"github.com/glebarez/sqlite"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

func Init() {
	cfg := config.C.Database
	var dialector gorm.Dialector

	switch cfg.Driver {
	case "mysql":
		dialector = mysql.Open(cfg.DSN)
	default:
		dialector = sqlite.Open(cfg.DSN)
	}

	logLevel := logger.Silent
	if config.C.Server.Debug {
		logLevel = logger.Info
	}

	var err error
	DB, err = gorm.Open(dialector, &gorm.Config{
		Logger: logger.Default.LogMode(logLevel),
	})
	if err != nil {
		log.Fatalf("failed to connect database: %v", err)
	}

	// Auto-migrate all tables
	if err := DB.AutoMigrate(
		&User{},
		&Token{},
		&Channel{},
		&Log{},
		&RedeemCode{},
		&TopupLog{},
		&VerificationCode{},
		&Setting{},
		&ModelPricing{},
		&Notice{},
	); err != nil {
		log.Fatalf("auto migrate failed: %v", err)
	}

	fmt.Printf("[DB] Connected to %s database\n", cfg.Driver)
}
