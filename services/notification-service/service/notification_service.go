package service

import (
	"fmt"
	"net/smtp"

	"github.com/auron/notification-service/config"
)

type NotificationService struct {
	cfg *config.Config
}

func NewNotificationService(cfg *config.Config) *NotificationService {
	return &NotificationService{cfg: cfg}
}

func (s *NotificationService) SendEmail(to, subject, body string) error {
	fmt.Printf("Sending email to %s: %s\n", to, subject)

	// SMTP configuration
	host := s.cfg.SMTPHost
	port := s.cfg.SMTPPort
	from := s.cfg.SMTPFrom
	username := s.cfg.SMTPUser
	password := s.cfg.SMTPPass

	// Build email message
	msg := fmt.Sprintf("From: %s\r\n", from)
	msg += fmt.Sprintf("To: %s\r\n", to)
	msg += fmt.Sprintf("Subject: %s\r\n", subject)
	msg += "\r\n" + body

	// Connect to SMTP server
	addr := host + ":" + port

	var auth smtp.Auth
	if username != "" {
		auth = smtp.PlainAuth("", username, password, host)
	}

	// Send email
	err := smtp.SendMail(addr, auth, from, []string{to}, []byte(msg))
	if err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	fmt.Printf("Email sent successfully to %s\n", to)
	return nil
}

func (s *NotificationService) SendSMS(to, message string) error {
	fmt.Printf("Sending SMS to %s: %s\n", to, message)
	// SMS placeholder - can integrate with Twilio or other providers
	return nil
}

func (s *NotificationService) StartConsumers() {
	fmt.Println("Starting notification consumers...")
	// Kafka consumers placeholder
}

func (s *NotificationService) Stop() {
	fmt.Println("Stopping notification service...")
}
