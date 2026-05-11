package auth

import (
	"fmt"
	"net/smtp"

	"go.uber.org/zap"
	"github.com/zhaojiewen/open-station/pkg/logger"
)

// SMTPEmailSender sends verification emails via SMTP
type SMTPEmailSender struct {
	host     string
	port     int
	username string
	password string
	from     string
}

// NewSMTPEmailSender creates a new SMTP email sender
func NewSMTPEmailSender(host string, port int, username, password, from string) *SMTPEmailSender {
	return &SMTPEmailSender{
		host:     host,
		port:     port,
		username: username,
		password: password,
		from:     from,
	}
}

// SendVerificationEmail sends a verification email to the user
func (s *SMTPEmailSender) SendVerificationEmail(to string, token string, userName string) error {
	if s.host == "" {
		logger.Warn("SMTP not configured, logging verification token instead",
			zap.String("email", to),
			zap.String("token", token),
		)
		return nil
	}

	subject := "Verify your email - Open Station"
	body := fmt.Sprintf(`From: %s
To: %s
Subject: %s
MIME-Version: 1.0
Content-Type: text/html; charset="UTF-8"

<html>
<body>
<h2>Welcome to Open Station, %s!</h2>
<p>Please verify your email address by clicking the link below (valid for 24 hours):</p>
<p><a href="https://open-station.local/verify-email?token=%s">Verify Email</a></p>
<p>Or use this token directly: <code>%s</code></p>
<p>If you did not create this account, please ignore this email.</p>
</body>
</html>`, s.from, to, subject, userName, token, token)

	addr := fmt.Sprintf("%s:%d", s.host, s.port)
	var auth smtp.Auth
	if s.username != "" {
		auth = smtp.PlainAuth("", s.username, s.password, s.host)
	}

	if err := smtp.SendMail(addr, auth, s.from, []string{to}, []byte(body)); err != nil {
		return fmt.Errorf("failed to send verification email: %w", err)
	}

	logger.Info("verification email sent", zap.String("email", to))
	return nil
}

// LogEmailSender logs verification tokens for development use
type LogEmailSender struct{}

// NewLogEmailSender creates a log-based email sender for development
func NewLogEmailSender() *LogEmailSender {
	return &LogEmailSender{}
}

// SendVerificationEmail logs the verification token
func (s *LogEmailSender) SendVerificationEmail(to string, token string, userName string) error {
	logger.Info("=== EMAIL VERIFICATION (DEV) ===",
		zap.String("to", to),
		zap.String("user", userName),
		zap.String("token", token),
		zap.String("verify_url", fmt.Sprintf("https://open-station.local/verify-email?token=%s", token)),
	)
	return nil
}