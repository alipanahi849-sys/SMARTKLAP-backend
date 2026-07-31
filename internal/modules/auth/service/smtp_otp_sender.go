package service

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/smtp"
	"strings"
	"time"

	"clap/internal/shared/config"
	"clap/internal/shared/logger"
)

type smtpOTPSender struct {
	cfg config.SMTP
}

// NewSMTPOTPSender delivers OTP codes over SMTP.
func NewSMTPOTPSender(cfg config.SMTP) OTPSender {
	return &smtpOTPSender{cfg: cfg}
}

func (s *smtpOTPSender) SendOTP(ctx context.Context, email, code string) error {
	from := strings.TrimSpace(s.cfg.From)
	if from == "" {
		from = strings.TrimSpace(s.cfg.Username)
	}
	if from == "" {
		return fmt.Errorf("SMTP_FROM (or SMTP_USERNAME) is required when SMTP is enabled")
	}

	fromHeader := from
	if name := strings.TrimSpace(s.cfg.FromName); name != "" {
		fromHeader = fmt.Sprintf("%s <%s>", name, from)
	}

	subject := "Your Clap verification code"
	body := fmt.Sprintf(
		"Your verification code is: %s\n\nThis code expires in 5 minutes.\nIf you did not request this, you can ignore this email.\n",
		code,
	)
	msg := strings.Join([]string{
		"From: " + fromHeader,
		"To: " + email,
		"Subject: " + subject,
		"MIME-Version: 1.0",
		"Content-Type: text/plain; charset=UTF-8",
		"",
		body,
	}, "\r\n")

	addr := s.cfg.Address()
	host := s.cfg.Host
	port := strings.TrimSpace(s.cfg.Port)
	if port == "" {
		port = "587"
	}

	dialer := net.Dialer{Timeout: 10 * time.Second}
	var (
		conn net.Conn
		err  error
	)

	// Port 465 typically uses implicit TLS; 587 uses STARTTLS when UseTLS is true.
	if port == "465" {
		tlsCfg := &tls.Config{ServerName: host, MinVersion: tls.VersionTLS12}
		conn, err = tls.DialWithDialer(&dialer, "tcp", addr, tlsCfg)
	} else {
		conn, err = dialer.DialContext(ctx, "tcp", addr)
	}
	if err != nil {
		return fmt.Errorf("smtp dial: %w", err)
	}

	client, err := smtp.NewClient(conn, host)
	if err != nil {
		_ = conn.Close()
		return fmt.Errorf("smtp client: %w", err)
	}
	defer client.Close()

	if s.cfg.UseTLS && port != "465" {
		tlsCfg := &tls.Config{ServerName: host, MinVersion: tls.VersionTLS12}
		if ok, _ := client.Extension("STARTTLS"); ok {
			if err := client.StartTLS(tlsCfg); err != nil {
				return fmt.Errorf("smtp starttls: %w", err)
			}
		}
	}

	if s.cfg.Username != "" {
		auth := smtp.PlainAuth("", s.cfg.Username, s.cfg.Password, host)
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("smtp auth: %w", err)
		}
	}

	if err := client.Mail(from); err != nil {
		return fmt.Errorf("smtp mail from: %w", err)
	}
	if err := client.Rcpt(email); err != nil {
		return fmt.Errorf("smtp rcpt: %w", err)
	}
	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("smtp data: %w", err)
	}
	if _, err := w.Write([]byte(msg)); err != nil {
		_ = w.Close()
		return fmt.Errorf("smtp write: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("smtp close data: %w", err)
	}
	if err := client.Quit(); err != nil {
		logger.Warn().Err(err).Str("email", email).Msg("smtp quit warning")
	}

	logger.Info().Str("email", email).Str("smtp_host", host).Msg("otp_email_sent")
	return nil
}

// NewOTPSenderFromConfig picks SMTP when configured, otherwise logs OTPs.
func NewOTPSenderFromConfig(cfg *config.Config) OTPSender {
	if cfg != nil && cfg.SMTP.Enabled() {
		logger.Info().
			Str("smtp_host", cfg.SMTP.Host).
			Str("smtp_port", cfg.SMTP.Port).
			Msg("OTP delivery via SMTP")
		return NewSMTPOTPSender(cfg.SMTP)
	}
	reveal := cfg != nil && cfg.Environment == "development"
	logger.Info().Bool("reveal_code", reveal).Msg("OTP delivery via logs (SMTP_HOST not set)")
	return NewLogOTPSender(reveal)
}
