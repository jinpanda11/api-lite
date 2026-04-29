package main

import (
	"embed"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"new-api-lite/config"
	"new-api-lite/handler"
	"new-api-lite/model"
	"new-api-lite/router"
	"os"
	"time"

	"github.com/gin-gonic/gin"
)

//go:embed web
var webFiles embed.FS

func main() {
	// Load configuration
	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = "config.yaml"
	}
	config.Load(configPath)

	if config.C.JWT.Secret == "change-me-in-production" || config.C.JWT.Secret == "change-me-in-production-please" {
		log.Fatalf("[FATAL] JWT secret is set to the default example value. Change it in config.yaml before running in production.")
	}
	if len(config.C.JWT.Secret) < 32 {
		log.Printf("[WARN] JWT secret is only %d characters. Use at least 32 random characters for production.", len(config.C.JWT.Secret))
	}

	// Init database and auto-migrate
	model.Init()

	// Seed default admin user if no users exist
	seedAdmin()

	// Start daily database backup
	handler.StartAutoBackup()

	// Start channel connectivity monitor
	handler.StartMonitor()

	// Setup Gin
	if !config.C.Server.Debug {
		gin.SetMode(gin.ReleaseMode)
	}
	r := gin.Default()
	r.MaxMultipartMemory = 8 << 20 // 8 MB

	webFS, err := fs.Sub(webFiles, "web")
	if err != nil {
		log.Fatalf("failed to get web filesystem: %v", err)
	}
	router.Setup(r, webFS)

	addr := fmt.Sprintf(":%d", config.C.Server.Port)
	log.Printf("[SERVER] Listening on http://0.0.0.0%s", addr)
	srv := &http.Server{
		Addr:         addr,
		Handler:      r,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 120 * time.Second,
		IdleTimeout:  120 * time.Second,
	}
	if err := srv.ListenAndServe(); err != nil {
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
	log.Printf("[SEED] Created admin user: %s (password from config)", cfg.Username)
}
