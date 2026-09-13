package service

import (
	"crypto/tls"
	"fmt"
	"log"
	"os"
	"strconv"

	"gopkg.in/gomail.v2"
)

// EmailService handles sending emails via SMTP.
type EmailService struct {
	dialer *gomail.Dialer
	from   string
}

// NewEmailService initializes a new EmailService instance with SMTP configurations.
func NewEmailService() *EmailService {
	host := os.Getenv("SMTP_HOST")
	port, _ := strconv.Atoi(os.Getenv("SMTP_PORT"))
	username := os.Getenv("SMTP_USERNAME")
	password := os.Getenv("SMTP_PASSWORD")
	from := os.Getenv("SMTP_FROM")
	log.Printf("SMTP CONFIG: host=%s port=%d user=%s from=%s", host, port, username, from)

	// Initialize the SMTP dialer
	dialer := gomail.NewDialer(host, port, username, password)

	// Configure TLS to ensure compatibility with modern SMTP servers like Gmail
	dialer.TLSConfig = &tls.Config{
		ServerName: host,
	}

	return &EmailService{
		dialer: dialer,
		from:   from,
	}
}

// SendVerificationEmail sends an HTML verification email to the specified user.
func (s *EmailService) SendVerificationEmail(to, fullName, code string) error {
	// Create a new email message
	m := gomail.NewMessage()

	// Set email headers
	m.SetHeader("From", s.from)
	m.SetHeader("To", to)
	m.SetHeader("Subject", "✅ Email Verification - Prompt Market")

	// Define the HTML body template
	body := fmt.Sprintf(`
		<div style="font-family: Arial, sans-serif; max-width: 600px; margin: 0 auto; padding: 20px; border: 1px solid #e0e0e0; border-radius: 8px;">
			<h2 style="color: #333;">✅ Email Verification</h2>
			<p>Hello <strong>%s</strong>,</p>
			<p>Your verification code is:</p>
			<h1 style="color: #2563eb; letter-spacing: 3px; background: #f3f4f6; padding: 10px; text-align: center; border-radius: 4px;">%s</h1>
			<p style="color: #666; font-size: 14px;">This code expires in 10 minutes.</p>
			<p style="color: #999; font-size: 12px;">If you didn't request this, please ignore this email.</p>
		</div>
	`, fullName, code)

	// Set the email body in HTML format
	m.SetBody("text/html", body)

	// Send the email and return any errors
	if err := s.dialer.DialAndSend(m); err != nil {
		return fmt.Errorf("failed to send email via SMTP: %w", err)
	}

	return nil
}
