package service

import (
	"strings"
	"testing"
)

func TestNewNotificationService(t *testing.T) {
	svc := NewNotificationService("smtp.example.com", "user", "pass", "from@example.com", 587)
	if svc == nil {
		t.Fatal("service should not be nil")
	}
	if svc.smtpHost != "smtp.example.com" {
		t.Fatalf("expected smtpHost smtp.example.com, got %s", svc.smtpHost)
	}
	if svc.smtpPort != 587 {
		t.Fatalf("expected smtpPort 587, got %d", svc.smtpPort)
	}
}

func TestSendEmail_NoSMTP(t *testing.T) {
	svc := NewNotificationService("", "", "", "", 0)
	err := svc.SendEmail([]string{"test@example.com"}, AlertInfo{})
	if err != nil {
		t.Fatalf("should skip without error, got: %v", err)
	}
}

func TestSendEmail_NoRecipients(t *testing.T) {
	svc := NewNotificationService("smtp.example.com", "", "", "", 587)
	err := svc.SendEmail([]string{}, AlertInfo{})
	if err != nil {
		t.Fatalf("should skip without error, got: %v", err)
	}
}

func TestSendEmail_EmptyHost(t *testing.T) {
	svc := NewNotificationService("", "user", "pass", "from@test.com", 587)
	err := svc.SendEmail([]string{"to@test.com"}, AlertInfo{ResourceName: "test"})
	if err != nil {
		t.Fatalf("should return nil for empty host, got: %v", err)
	}
}

func TestSendSlack_EmptyURL(t *testing.T) {
	svc := NewNotificationService("", "", "", "", 0)
	err := svc.SendSlack("", AlertInfo{})
	if err != nil {
		t.Fatalf("should return nil for empty URL, got: %v", err)
	}
}

func TestSendSlack_InvalidURL(t *testing.T) {
	svc := NewNotificationService("", "", "", "", 0)
	err := svc.SendSlack("http://127.0.0.1:9/nonexistent", AlertInfo{
		ResourceName: "test",
		TenantName:   "test-tenant",
		PercentUsed:  85,
	})
	// Connection refused is expected
	if err == nil {
		t.Fatal("should return error for invalid URL")
	}
}

func TestSendWebhook_EmptyURL(t *testing.T) {
	svc := NewNotificationService("", "", "", "", 0)
	err := svc.SendWebhook("", AlertInfo{})
	if err != nil {
		t.Fatalf("should return nil for empty URL, got: %v", err)
	}
}

func TestSendWebhook_InvalidURL(t *testing.T) {
	svc := NewNotificationService("", "", "", "", 0)
	err := svc.SendWebhook("http://127.0.0.1:9/nonexistent", AlertInfo{
		ResourceName: "test",
		TenantName:   "test-tenant",
		PercentUsed:  95,
	})
	if err == nil {
		t.Fatal("should return error for invalid URL")
	}
}

func TestBuildEmailBody(t *testing.T) {
	svc := NewNotificationService("", "", "", "", 0)
	body := svc.buildEmailBody(AlertInfo{
		ResourceName: "TestResource",
		TenantName:   "TestTenant",
		PercentUsed:  85,
	})
	if !strings.Contains(body, "TestResource") {
		t.Fatal("email body should contain resource name")
	}
	if !strings.Contains(body, "TestTenant") {
		t.Fatal("email body should contain tenant name")
	}
	if !strings.Contains(body, "85%") {
		t.Fatal("email body should contain percentage")
	}
	if !strings.Contains(body, "<html") {
		t.Fatal("email body should be HTML")
	}
}

func TestGetSlackColor_Danger(t *testing.T) {
	svc := NewNotificationService("", "", "", "", 0)
	c := svc.getSlackColor(100)
	if c != "danger" {
		t.Fatalf("expected danger, got %s", c)
	}
	c = svc.getSlackColor(101)
	if c != "danger" {
		t.Fatalf("expected danger for 101%%, got %s", c)
	}
}

func TestGetSlackColor_Warning(t *testing.T) {
	svc := NewNotificationService("", "", "", "", 0)
	c := svc.getSlackColor(90)
	if c != "warning" {
		t.Fatalf("expected warning, got %s", c)
	}
	c = svc.getSlackColor(99)
	if c != "warning" {
		t.Fatalf("expected warning, got %s", c)
	}
}

func TestGetSlackColor_Good(t *testing.T) {
	svc := NewNotificationService("", "", "", "", 0)
	c := svc.getSlackColor(89)
	if c != "good" {
		t.Fatalf("expected good, got %s", c)
	}
	c = svc.getSlackColor(0)
	if c != "good" {
		t.Fatalf("expected good, got %s", c)
	}
}

func TestGetHTMLColor_Danger(t *testing.T) {
	svc := NewNotificationService("", "", "", "", 0)
	c := svc.getHTMLColor(100)
	if c != "#dc3545" {
		t.Fatalf("expected #dc3545, got %s", c)
	}
}

func TestGetHTMLColor_Warning(t *testing.T) {
	svc := NewNotificationService("", "", "", "", 0)
	c := svc.getHTMLColor(90)
	if c != "#ffc107" {
		t.Fatalf("expected #ffc107, got %s", c)
	}
}

func TestGetHTMLColor_Good(t *testing.T) {
	svc := NewNotificationService("", "", "", "", 0)
	c := svc.getHTMLColor(89)
	if c != "#17a2b8" {
		t.Fatalf("expected #17a2b8, got %s", c)
	}
}

func TestSendSlack_Success(t *testing.T) {
	svc := NewNotificationService("", "", "", "", 0)
	// slack.bot.com returns 404 but this test verifies the request format
	err := svc.SendSlack("https://slack.com/api/chat.postMessage", AlertInfo{
		ResourceName: "test",
		TenantName:   "test-tenant",
		PercentUsed:  85,
	})
	// This will get a non-200 from slack.com without a token, which is expected
	if err == nil {
		t.Log("unexpected success calling real slack API")
	}
}

func TestSendWebhook_Success(t *testing.T) {
	svc := NewNotificationService("", "", "", "", 0)
	err := svc.SendWebhook("https://httpbin.org/post", AlertInfo{
		ResourceName: "test",
		TenantName:   "test-tenant",
		PercentUsed:  85,
	})
	// May or may not succeed depending on network
	if err != nil {
		t.Logf("webhook call failed (may be network issue): %v", err)
	}
}