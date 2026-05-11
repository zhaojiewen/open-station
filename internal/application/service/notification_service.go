package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/smtp"
	"strings"
	"text/template"
	"time"
)

// NotificationService handles sending notifications via email, Slack, and webhook
type NotificationService struct {
	smtpHost     string
	smtpPort     int
	smtpUser     string
	smtpPassword string
	smtpFrom     string
}

// NewNotificationService creates a new notification service
func NewNotificationService(smtpHost, smtpUser, smtpPassword, smtpFrom string, smtpPort int) *NotificationService {
	return &NotificationService{
		smtpHost:     smtpHost,
		smtpPort:     smtpPort,
		smtpUser:     smtpUser,
		smtpPassword: smtpPassword,
		smtpFrom:     smtpFrom,
	}
}

// SendEmail sends an email notification
func (s *NotificationService) SendEmail(to []string, info AlertInfo) error {
	if s.smtpHost == "" || len(to) == 0 {
		return nil // Skip if SMTP not configured
	}

	// Build email content
	subject := fmt.Sprintf("[Budget Alert] %s usage at %d%%", info.ResourceName, info.PercentUsed)
	body := s.buildEmailBody(info)

	// Build message
	msg := fmt.Sprintf("From: %s\r\n", s.smtpFrom)
	msg += fmt.Sprintf("To: %s\r\n", strings.Join(to, ","))
	msg += fmt.Sprintf("Subject: %s\r\n", subject)
	msg += "MIME-version: 1.0;\r\nContent-Type: text/html; charset=\"UTF-8\";\r\n\r\n"
	msg += body

	// Send email
	addr := fmt.Sprintf("%s:%d", s.smtpHost, s.smtpPort)
	auth := smtp.PlainAuth("", s.smtpUser, s.smtpPassword, s.smtpHost)

	if err := smtp.SendMail(addr, auth, s.smtpFrom, to, []byte(msg)); err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	return nil
}

// SendSlack sends a Slack notification via webhook
func (s *NotificationService) SendSlack(webhookURL string, info AlertInfo) error {
	if webhookURL == "" {
		return nil
	}

	// Build Slack message
	message := map[string]interface{}{
		"text": fmt.Sprintf("Budget Alert: %s usage at %d%%", info.ResourceName, info.PercentUsed),
		"attachments": []map[string]interface{}{
			{
				"color": s.getSlackColor(info.PercentUsed),
				"fields": []map[string]interface{}{
					{
						"title": "Resource",
						"value": info.ResourceName,
						"short": true,
					},
					{
						"title": "Tenant",
						"value": info.TenantName,
						"short": true,
					},
					{
						"title": "Usage",
						"value": fmt.Sprintf("%d%% (%s / %s)", info.PercentUsed, info.UsedAmount.StringFixed(2), info.LimitAmount.StringFixed(2)),
						"short": false,
					},
					{
						"title": "Time",
						"value": info.Timestamp.Format(time.RFC3339),
						"short": true,
					},
				},
			},
		},
	}

	// Send to Slack
	body, _ := json.Marshal(message)
	resp, err := http.Post(webhookURL, "application/json", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to send Slack notification: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Slack returned status %d", resp.StatusCode)
	}

	return nil
}

// SendWebhook sends a notification to a custom webhook
func (s *NotificationService) SendWebhook(webhookURL string, info AlertInfo) error {
	if webhookURL == "" {
		return nil
	}

	// Build webhook payload
	payload := map[string]interface{}{
		"event":       "budget_alert",
		"scope":       info.Scope,
		"scope_id":    info.ScopeID.String(),
		"resource":    info.ResourceName,
		"tenant":      info.TenantName,
		"percent_used": info.PercentUsed,
		"used_amount": info.UsedAmount.StringFixed(4),
		"limit_amount": info.LimitAmount.StringFixed(4),
		"timestamp":   info.Timestamp.Format(time.RFC3339),
	}

	// Send to webhook
	body, _ := json.Marshal(payload)
	resp, err := http.Post(webhookURL, "application/json", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to send webhook notification: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("webhook returned status %d", resp.StatusCode)
	}

	return nil
}

// buildEmailBody builds HTML email body
func (s *NotificationService) buildEmailBody(info AlertInfo) string {
	tmpl := `
<!DOCTYPE html>
<html>
<head>
	<style>
		body { font-family: Arial, sans-serif; }
		.alert { padding: 20px; background-color: {{.Color}}; color: white; }
		.content { padding: 20px; }
		.field { margin-bottom: 10px; }
		.label { font-weight: bold; }
	</style>
</head>
<body>
	<div class="alert">
		<h2>Budget Alert: {{.ResourceName}}</h2>
		<p>Usage has reached {{.PercentUsed}}% of the limit</p>
	</div>
	<div class="content">
		<div class="field">
			<span class="label">Resource:</span> {{.ResourceName}}
		</div>
		<div class="field">
			<span class="label">Tenant:</span> {{.TenantName}}
		</div>
		<div class="field">
			<span class="label">Usage:</span> {{.UsedAmount}} / {{.LimitAmount}} ({{.PercentUsed}}%)
		</div>
		<div class="field">
			<span class="label">Time:</span> {{.Timestamp}}
		</div>
	</div>
</body>
</html>
`

	data := map[string]interface{}{
		"ResourceName":  info.ResourceName,
		"TenantName":    info.TenantName,
		"UsedAmount":    info.UsedAmount.StringFixed(2),
		"LimitAmount":   info.LimitAmount.StringFixed(2),
		"PercentUsed":   info.PercentUsed,
		"Timestamp":     info.Timestamp.Format(time.RFC3339),
		"Color":         s.getHTMLColor(info.PercentUsed),
	}

	t, _ := template.New("email").Parse(tmpl)
	var buf bytes.Buffer
	t.Execute(&buf, data)
	return buf.String()
}

// getSlackColor returns Slack attachment color based on percentage
func (s *NotificationService) getSlackColor(percent int) string {
	if percent >= 100 {
		return "danger"
	}
	if percent >= 90 {
		return "warning"
	}
	return "good"
}

// getHTMLColor returns HTML color based on percentage
func (s *NotificationService) getHTMLColor(percent int) string {
	if percent >= 100 {
		return "#dc3545" // Red
	}
	if percent >= 90 {
		return "#ffc107" // Yellow
	}
	return "#17a2b8" // Blue
}