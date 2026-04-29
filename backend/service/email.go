package service

import (
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"new-api-lite/config"
	"new-api-lite/model"
	"strconv"
	"time"

	gomail "gopkg.in/gomail.v2"
)

// ErrSMTPNotConfigured is returned when SMTP has not been set up.
var ErrSMTPNotConfigured = errors.New("SMTP not configured")

// GenerateCode creates a 6-digit numeric code.
func GenerateCode() string {
	n, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		return "000000"
	}
	return fmt.Sprintf("%06d", n.Int64())
}

func getSMTPConfig() (host, username, password, from string, port int, ssl bool) {
	host = config.C.SMTP.Host
	username = config.C.SMTP.Username
	password = config.C.SMTP.Password
	from = config.C.SMTP.From
	port = config.C.SMTP.Port
	ssl = config.C.SMTP.SSL

	if v, err := model.GetSetting("smtp_host"); err == nil && v != "" {
		host = v
	}
	if v, err := model.GetSetting("smtp_port"); err == nil && v != "" {
		if p, e := strconv.Atoi(v); e == nil {
			port = p
		}
	}
	if v, err := model.GetSetting("smtp_username"); err == nil && v != "" {
		username = v
	}
	if v, err := model.GetSetting("smtp_password"); err == nil && v != "" {
		password = v
	}
	if v, err := model.GetSetting("smtp_from"); err == nil && v != "" {
		from = v
	}
	if v, err := model.GetSetting("smtp_ssl"); err == nil && v != "" {
		ssl = v == "true"
	}
	return
}

// IsSMTPConfigured returns true if SMTP settings are present and not default.
func IsSMTPConfigured() bool {
	host, _, _, _, _, _ := getSMTPConfig()
	return host != "" && host != "smtp.example.com"
}

// SendVerificationEmail sends a 6-digit code via SMTP.
// Returns ErrSMTPNotConfigured if SMTP is not set up.
func SendVerificationEmail(to, code string) error {
	host, username, password, from, port, ssl := getSMTPConfig()
	if host == "" || host == "smtp.example.com" {
		return ErrSMTPNotConfigured
	}

	m := gomail.NewMessage()
	m.SetHeader("From", from)
	m.SetHeader("To", to)
	m.SetHeader("Subject", "Your verification code")
	m.SetBody("text/html", fmt.Sprintf(`
			<div style="font-family:Arial,sans-serif;max-width:480px;margin:0 auto">
				<h2>Verification Code</h2>
				<p>Your verification code is:</p>
				<div style="font-size:36px;font-weight:bold;letter-spacing:8px;color:#4F46E5">%s</div>
				<p style="color:#888;font-size:12px">This code expires in 10 minutes. Do not share it.</p>
			</div>
		`, code))

	d := gomail.NewDialer(host, port, username, password)
	d.SSL = ssl

	done := make(chan error, 1)
	go func() {
		done <- d.DialAndSend(m)
	}()
	select {
	case err := <-done:
		return err
	case <-time.After(30 * time.Second):
		return fmt.Errorf("SMTP send timed out after 30s")
	}
}
