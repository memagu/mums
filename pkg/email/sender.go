package email

import (
	"fmt"
	"log"
	"net"
	"net/mail"
	"net/smtp"
	"strconv"
	"strings"
)

type SMTPConfig struct {
	From     string
	Host     string
	Password string
	Port     int
	Username string
}

type Sender interface {
	Send(to, subject, body string) error
}

func NewSender(cfg SMTPConfig) Sender {
	if cfg.Host == "" {
		return logSender{}
	}
	if cfg.From == "" {
		cfg.From = cfg.Username
	}
	return &smtpSender{config: cfg}
}

type smtpSender struct {
	config SMTPConfig
}

// Send delivers a plain-text message over STARTTLS (port 587).
func (s *smtpSender) Send(to, subject, body string) error {
	addr := net.JoinHostPort(s.config.Host, strconv.Itoa(s.config.Port))

	auth := smtp.PlainAuth("", s.config.Username, s.config.Password, s.config.Host)

	fromAddress, err := mail.ParseAddress(s.config.From)
	if err != nil {
		return fmt.Errorf("invalid from address %q: %w", s.config.From, err)
	}

	message := "From: " + s.config.From + "\r\n" +
		"To: " + to + "\r\n" +
		"Subject: " + subject + "\r\n" +
		"\r\n" +
		normalizeCRLF(body)

	if err := smtp.SendMail(addr, auth, fromAddress.Address, []string{to}, []byte(message)); err != nil {
		return fmt.Errorf("smtp send: %w", err)
	}

	return nil
}

func normalizeCRLF(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	return strings.ReplaceAll(s, "\n", "\r\n")
}

type logSender struct{}

// Send prints the message to the server log so a dev can copy the link.
func (logSender) Send(to, subject, body string) error {
	log.Printf("password reset email (SMTP not configured): to=%s subject=%q\n%s", to, subject, body)
	return nil
}
