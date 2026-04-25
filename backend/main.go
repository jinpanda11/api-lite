package main

import (
	"embed"
	"fmt"
	"io/fs"
	"log"
	"new-api-lite/config"
	"new-api-lite/model"
	"new-api-lite/router"

	"github.com/gin-gonic/gin"
)

//go:embed web/*
var webFiles embed.FS

func main() {
	// Load configuration
	config.Load("config.yaml")

	// Init database and auto-migrate
	model.Init()

	// Seed default admin user if no users exist
	seedAdmin()

	// Setup Gin
	if !config.C.Server.Debug {
		gin.SetMode(gin.ReleaseMode)
	}
	r := gin.Default()

	webFS, err := fs.Sub(webFiles, "web")
	if err != nil {
		log.Fatalf("failed to get web filesystem: %v", err)
	}
	router.Setup(r, webFS)

	addr := fmt.Sprintf(":%d", config.C.Server.Port)
	log.Printf("[SERVER] Listening on http://0.0.0.0%s", addr)
	if err := r.Run(addr); err != nil {
		log.Fatalf("server error: %v", err)
	}
}

func seedAdmin() {
	var count int64
	model.DB.Model(&model.User{}).Count(&count)
	if count > 0 {
		return
	}

	cfg := config.C.Admin
	admin := &model.User{
		Username: cfg.Username,
		Email:    cfg.Email,
		Role:     model.RoleAdmin,
		Status:   model.StatusEnabled,
	}
	if err := admin.SetPassword(cfg.Password); err != nil {
		log.Printf("[SEED] failed to hash admin password: %v", err)
		return
	}
	if err := model.DB.Create(admin).Error; err != nil {
		log.Printf("[SEED] failed to create admin user: %v", err)
		return
	}
	log.Printf("[SEED] Created admin user: %s / %s", cfg.Username, cfg.Password)
}
