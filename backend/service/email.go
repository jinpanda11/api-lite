package service

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"new-api-lite/config"
	"new-api-lite/model"
	"strconv"

	gomail "gopkg.in/gomail.v2"
)

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

// SendVerificationEmail sends a 6-digit code via SMTP.
func SendVerificationEmail(to, code string) error {
	host, username, password, from, port, ssl := getSMTPConfig()
	if host == "" || host == "smtp.example.com" {
		// SMTP not configured: print to console for development
		fmt.Printf("[EMAIL] To: %s | Code: %s\n", to, code)
		return nil
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
	return d.DialAndSend(m)
}
