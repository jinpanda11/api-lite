package handler

import (
	"fmt"
	"io"
	"net/http"
	"new-api-lite/config"
	"new-api-lite/middleware"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// BackupNow triggers a manual database backup.
// POST /api/admin/backup
func BackupNow(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	path, err := doBackup()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"message": "backup created",
		"path":    path,
		"by":      user.Username,
	})
}

// StartAutoBackup runs a daily backup at 3:07 AM local time.
func StartAutoBackup() {
	go func() {
		for {
			next := nextBackupTime()
			time.Sleep(time.Until(next))
			path, err := doBackup()
			if err != nil {
				fmt.Printf("[BACKUP] failed: %v\n", err)
			} else {
				fmt.Printf("[BACKUP] saved: %s\n", path)
			}
			// Clean up old backups (>30 days)
			cleanOldBackups(30)
		}
	}()
}

func nextBackupTime() time.Time {
	now := time.Now()
	t := time.Date(now.Year(), now.Month(), now.Day(), 3, 7, 0, 0, now.Location())
	if !t.After(now) {
		t = t.Add(24 * time.Hour)
	}
	return t
}

func doBackup() (string, error) {
	dsn := config.C.Database.DSN
	if dsn == "" {
		return "", fmt.Errorf("database DSN not configured")
	}

	// Only SQLite is supported for file-based backup
	if !strings.HasSuffix(dsn, ".db") {
		return "", fmt.Errorf("backup only supported for SQLite")
	}

	src, err := os.Open(dsn)
	if err != nil {
		return "", fmt.Errorf("failed to open database: %w", err)
	}
	defer src.Close()

	backupDir := filepath.Join(filepath.Dir(dsn), "backups")
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create backup dir: %w", err)
	}

	name := fmt.Sprintf("backup-%s.db", time.Now().Format("2006-01-02T15-04-05"))
	dstPath := filepath.Join(backupDir, name)
	dst, err := os.Create(dstPath)
	if err != nil {
		return "", fmt.Errorf("failed to create backup file: %w", err)
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		return "", fmt.Errorf("failed to copy database: %w", err)
	}

	return dstPath, nil
}

func cleanOldBackups(maxDays int) {
	dsn := config.C.Database.DSN
	if dsn == "" {
		return
	}
	backupDir := filepath.Join(filepath.Dir(dsn), "backups")
	entries, err := os.ReadDir(backupDir)
	if err != nil {
		return
	}
	cutoff := time.Now().AddDate(0, 0, -maxDays)
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), "backup-") && strings.HasSuffix(e.Name(), ".db") {
			info, err := e.Info()
			if err == nil && info.ModTime().Before(cutoff) {
				os.Remove(filepath.Join(backupDir, e.Name()))
			}
		}
	}
}
