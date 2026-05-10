package entity

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

func TestTenant_Entity(t *testing.T) {
	now := time.Now()
	tenant := Tenant{
		ID:                uuid.New(),
		Name:              "Test Tenant",
		Slug:              "test-tenant",
		Status:            "active",
		Plan:              "pro",
		RateLimitRPS:      100,
		RateLimitBurst:    200,
		MonthlyRequestLimit: 10000,
		BillingEmail:      "billing@example.com",
		Balance:           decimal.NewFromInt(1000),
		Currency:          "USD",
		Metadata:          "{}",
		CreatedAt:         now,
		UpdatedAt:         now,
	}

	// Test field values
	if tenant.Name != "Test Tenant" {
		t.Errorf("Name = %v, want Test Tenant", tenant.Name)
	}
	if tenant.Slug != "test-tenant" {
		t.Errorf("Slug = %v, want test-tenant", tenant.Slug)
	}
	if tenant.Status != "active" {
		t.Errorf("Status = %v, want active", tenant.Status)
	}
	if tenant.Balance.IntPart() != 1000 {
		t.Errorf("Balance = %v, want 1000", tenant.Balance)
	}
}

func TestTenant_StatusValues(t *testing.T) {
	statuses := []string{"active", "suspended", "deleted"}

	for _, status := range statuses {
		tenant := Tenant{Status: status}
		if tenant.Status != status {
			t.Errorf("Status = %v, want %v", tenant.Status, status)
		}
	}
}

func TestTenant_PlanValues(t *testing.T) {
	plans := []string{"free", "basic", "pro", "enterprise"}

	for _, plan := range plans {
		tenant := Tenant{Plan: plan}
		if tenant.Plan != plan {
			t.Errorf("Plan = %v, want %v", tenant.Plan, plan)
		}
	}
}

func TestUser_Entity(t *testing.T) {
	tenantID := uuid.New()
	now := time.Now()
	lastLogin := now.Add(-1 * time.Hour)
	rps := 10
	burst := 20

	user := User{
		ID:            uuid.New(),
		TenantID:      tenantID,
		Email:         "user@example.com",
		PasswordHash:  "hashedpassword",
		Name:          "Test User",
		Role:          "admin",
		RateLimitRPS:  &rps,
		RateLimitBurst: &burst,
		Status:        "active",
		LastLoginAt:   &lastLogin,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	if user.Email != "user@example.com" {
		t.Errorf("Email = %v, want user@example.com", user.Email)
	}
	if user.TenantID != tenantID {
		t.Errorf("TenantID mismatch")
	}
	if user.Role != "admin" {
		t.Errorf("Role = %v, want admin", user.Role)
	}
	if *user.RateLimitRPS != 10 {
		t.Errorf("RateLimitRPS = %v, want 10", *user.RateLimitRPS)
	}
}

func TestUser_RoleValues(t *testing.T) {
	roles := []string{"admin", "member", "viewer"}

	for _, role := range roles {
		user := User{Role: role}
		if user.Role != role {
			t.Errorf("Role = %v, want %v", user.Role, role)
		}
	}
}

func TestUser_StatusValues(t *testing.T) {
	statuses := []string{"active", "inactive"}

	for _, status := range statuses {
		user := User{Status: status}
		if user.Status != status {
			t.Errorf("Status = %v, want %v", user.Status, status)
		}
	}
}

func TestAPIKey_Entity(t *testing.T) {
	userID := uuid.New()
	tenantID := uuid.New()
	now := time.Now()
	expiry := now.Add(30 * 24 * time.Hour)
	rps := 15
	burst := 30
	tokenLimit := int64(100000)

	apiKey := APIKey{
		ID:                  uuid.New(),
		UserID:              userID,
		TenantID:            tenantID,
		KeyHash:             "hash123",
		KeyPrefix:           "sk-test",
		Name:                "Test Key",
		Permissions:         `["read","write"]`,
		AllowedModels:       `["gpt-4","gpt-3.5-turbo"]`,
		AllowedProviders:    `["openai","anthropic"]`,
		RateLimitRPS:        &rps,
		RateLimitBurst:      &burst,
		MonthlyTokenLimit:   &tokenLimit,
		UsedTokensThisMonth: 5000,
		Status:              "active",
		ExpiresAt:           &expiry,
		LastUsedAt:          &now,
		CreatedAt:           now,
		UpdatedAt:           now,
	}

	if apiKey.KeyHash != "hash123" {
		t.Errorf("KeyHash = %v, want hash123", apiKey.KeyHash)
	}
	if apiKey.KeyPrefix != "sk-test" {
		t.Errorf("KeyPrefix = %v, want sk-test", apiKey.KeyPrefix)
	}
	if apiKey.Status != "active" {
		t.Errorf("Status = %v, want active", apiKey.Status)
	}
	if apiKey.UsedTokensThisMonth != 5000 {
		t.Errorf("UsedTokensThisMonth = %v, want 5000", apiKey.UsedTokensThisMonth)
	}
}

func TestAPIKey_StatusValues(t *testing.T) {
	statuses := []string{"active", "revoked", "expired"}

	for _, status := range statuses {
		key := APIKey{Status: status}
		if key.Status != status {
			t.Errorf("Status = %v, want %v", key.Status, status)
		}
	}
}

func TestModel_Entity(t *testing.T) {
	maxTokens := 4096
	contextWindow := 8192

	model := Model{
		ID:               uuid.New(),
		Provider:         "openai",
		ModelID:          "gpt-4",
		DisplayName:      "GPT-4",
		PromptPrice:      decimal.NewFromFloat(0.03),
		CompletionPrice:  decimal.NewFromFloat(0.06),
		Currency:         "USD",
		MaxTokens:        &maxTokens,
		ContextWindow:    &contextWindow,
		Capabilities:     `{"streaming":true,"function_calling":true}`,
		Status:           "active",
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}

	if model.Provider != "openai" {
		t.Errorf("Provider = %v, want openai", model.Provider)
	}
	if model.ModelID != "gpt-4" {
		t.Errorf("ModelID = %v, want gpt-4", model.ModelID)
	}
	if model.PromptPrice.Cmp(decimal.NewFromFloat(0.03)) != 0 {
		t.Errorf("PromptPrice = %v, want 0.03", model.PromptPrice)
	}
}

func TestModel_StatusValues(t *testing.T) {
	statuses := []string{"active", "deprecated"}

	for _, status := range statuses {
		model := Model{Status: status}
		if model.Status != status {
			t.Errorf("Status = %v, want %v", model.Status, status)
		}
	}
}

func TestUsageRecord_Entity(t *testing.T) {
	latency := 100
	statusCode := 200
	errorMsg := "timeout"

	record := UsageRecord{
		ID:              uuid.New(),
		TenantID:        uuid.New(),
		UserID:          uuid.New(),
		APIKeyID:        uuid.New(),
		RequestID:       "req-123456",
		Provider:        "openai",
		ModelID:         "gpt-4",
		PromptTokens:    100,
		CompletionTokens: 50,
		TotalTokens:     150,
		Cost:            decimal.NewFromFloat(0.01),
		Currency:        "USD",
		LatencyMs:       &latency,
		StatusCode:      &statusCode,
		ErrorMessage:    &errorMsg,
		CreatedAt:       time.Now(),
	}

	if record.RequestID != "req-123456" {
		t.Errorf("RequestID = %v, want req-123456", record.RequestID)
	}
	if record.TotalTokens != 150 {
		t.Errorf("TotalTokens = %v, want 150", record.TotalTokens)
	}
}

func TestBill_Entity(t *testing.T) {
	now := time.Now()
	paidAt := now.Add(1 * time.Hour)

	bill := Bill{
		ID:          uuid.New(),
		TenantID:    uuid.New(),
		BillNumber:  "BILL-2024-001",
		PeriodStart: now,
		PeriodEnd:   now.Add(30 * 24 * time.Hour),
		TotalTokens: 50000,
		TotalCost:   decimal.NewFromInt(100),
		Currency:    "USD",
		Status:      "paid",
		PaidAt:      &paidAt,
		Items:       `[]`,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if bill.BillNumber != "BILL-2024-001" {
		t.Errorf("BillNumber = %v, want BILL-2024-001", bill.BillNumber)
	}
	if bill.Status != "paid" {
		t.Errorf("Status = %v, want paid", bill.Status)
	}
}

func TestBill_StatusValues(t *testing.T) {
	statuses := []string{"pending", "paid", "overdue", "cancelled"}

	for _, status := range statuses {
		bill := Bill{Status: status}
		if bill.Status != status {
			t.Errorf("Status = %v, want %v", bill.Status, status)
		}
	}
}

func TestRechargeRecord_Entity(t *testing.T) {
	now := time.Now()
	completed := now.Add(1 * time.Minute)

	record := RechargeRecord{
		ID:           uuid.New(),
		TenantID:     uuid.New(),
		Amount:       decimal.NewFromInt(500),
		Currency:     "USD",
		PaymentMethod: "credit_card",
		PaymentID:    "payment-123",
		Status:       "completed",
		CompletedAt:  &completed,
		Notes:        "Monthly recharge",
		CreatedAt:    now,
	}

	if record.Amount.IntPart() != 500 {
		t.Errorf("Amount = %v, want 500", record.Amount)
	}
	if record.Status != "completed" {
		t.Errorf("Status = %v, want completed", record.Status)
	}
}

func TestRechargeRecord_StatusValues(t *testing.T) {
	statuses := []string{"pending", "completed", "failed"}

	for _, status := range statuses {
		record := RechargeRecord{Status: status}
		if record.Status != status {
			t.Errorf("Status = %v, want %v", record.Status, status)
		}
	}
}

func TestAuditLog_Entity(t *testing.T) {
	log := AuditLog{
		ID:           uuid.New(),
		TenantID:     uuid.New(),
		UserID:       uuid.New(),
		Action:       "api_key_created",
		ResourceType: "api_key",
		ResourceID:   uuid.New(),
		OldValues:    "",
		NewValues:    `{"name":"Test Key"}`,
		IPAddress:    "192.168.1.1",
		UserAgent:    "Mozilla/5.0",
		CreatedAt:    time.Now(),
	}

	if log.Action != "api_key_created" {
		t.Errorf("Action = %v, want api_key_created", log.Action)
	}
}

func TestProviderAccount_Entity(t *testing.T) {
	now := time.Now()
	lastError := "rate limit exceeded"
	lastErrorAt := now.Add(-1 * time.Hour)
	lastUsed := now
	disabled := now.Add(-24 * time.Hour)
	monthlyLimit := decimal.NewFromFloat(1000.0)

	account := ProviderAccount{
		ID:              uuid.New(),
		Provider:        "openai",
		Name:            "OpenAI Main",
		APIKey:          "sk-key123",
		BaseURL:         "https://api.openai.com/v1",
		Priority:        0,
		Status:          "active",
		IsDefault:       true,
		MonthlyLimit:    &monthlyLimit,
		UsedThisMonth:   decimal.NewFromFloat(50.0),
		RequestCount:    1000,
		SuccessCount:    950,
		ErrorCount:      5,
		LastError:       &lastError,
		LastErrorAt:     &lastErrorAt,
		TotalRequests:   10000,
		TotalSuccess:    9500,
		TotalErrors:     500,
		TotalCost:       decimal.NewFromFloat(500.0),
		Timeout:         120,
		RetryCount:      3,
		RateLimitRPS:    100,
		EnabledModels:   `["gpt-4","gpt-3.5-turbo"]`,
		CreatedAt:       now,
		UpdatedAt:       now,
		LastUsedAt:      &lastUsed,
		DisabledAt:      &disabled,
	}

	if account.Provider != "openai" {
		t.Errorf("Provider = %v, want openai", account.Provider)
	}
	if account.Status != "active" {
		t.Errorf("Status = %v, want active", account.Status)
	}
	if !account.IsDefault {
		t.Error("IsDefault should be true")
	}
}

func TestProviderAccount_StatusValues(t *testing.T) {
	statuses := []string{"active", "disabled", "limited", "exhausted"}

	for _, status := range statuses {
		account := ProviderAccount{Status: status}
		if account.Status != status {
			t.Errorf("Status = %v, want %v", account.Status, status)
		}
	}
}

func TestProviderAccount_ProviderValues(t *testing.T) {
	providers := []string{"openai", "anthropic", "gemini", "deepseek", "glm"}

	for _, provider := range providers {
		account := ProviderAccount{Provider: provider}
		if account.Provider != provider {
			t.Errorf("Provider = %v, want %v", account.Provider, provider)
		}
	}
}

func TestBillItem(t *testing.T) {
	item := BillItem{
		Provider: "openai",
		Model:    "gpt-4",
		Tokens:   10000,
		Cost:     decimal.NewFromFloat(50.0),
	}

	if item.Provider != "openai" {
		t.Errorf("Provider = %v, want openai", item.Provider)
	}
	if item.Model != "gpt-4" {
		t.Errorf("Model = %v, want gpt-4", item.Model)
	}
}

func TestTenant_DeletedAt(t *testing.T) {
	now := time.Now()
	deletedAt := gorm.DeletedAt{Time: now, Valid: true}

	tenant := Tenant{
		DeletedAt: deletedAt,
	}

	if !tenant.DeletedAt.Valid {
		t.Error("DeletedAt should be valid")
	}
}

func TestUser_DeletedAt(t *testing.T) {
	now := time.Now()
	deletedAt := gorm.DeletedAt{Time: now, Valid: true}

	user := User{
		DeletedAt: deletedAt,
	}

	if !user.DeletedAt.Valid {
		t.Error("DeletedAt should be valid")
	}
}

// APIKey does not have DeletedAt field - test removed

func TestNilPointerFields(t *testing.T) {
	// Test entities with nil pointer fields
	user := User{
		RateLimitRPS:    nil,
		RateLimitBurst:  nil,
		LastLoginAt:     nil,
	}

	if user.RateLimitRPS != nil {
		t.Error("RateLimitRPS should be nil")
	}

	apiKey := APIKey{
		RateLimitRPS:         nil,
		RateLimitBurst:       nil,
		MonthlyTokenLimit:    nil,
		ExpiresAt:            nil,
		LastUsedAt:           nil,
		RevokedAt:            nil,
	}

	if apiKey.ExpiresAt != nil {
		t.Error("ExpiresAt should be nil")
	}

	tenant := Tenant{
		Balance: decimal.Zero,
	}

	if !tenant.Balance.IsZero() {
		t.Error("Balance should be zero")
	}
}

func TestDecimalPrecision(t *testing.T) {
	// Test decimal field precision
	tenant := Tenant{
		Balance: decimal.NewFromFloat(1234.5678),
	}

	expected := decimal.NewFromFloat(1234.5678)
	if !tenant.Balance.Equals(expected) {
		t.Errorf("Balance = %v, want %v", tenant.Balance, expected)
	}

	model := Model{
		PromptPrice:     decimal.NewFromFloat(0.000123456),
		CompletionPrice: decimal.NewFromFloat(0.000654321),
	}

	if model.PromptPrice.Cmp(decimal.NewFromFloat(0.000123456)) != 0 {
		t.Errorf("PromptPrice precision error")
	}
}

func TestUUIDGeneration(t *testing.T) {
	// Test that UUID fields can be generated
	tenant := Tenant{
		ID: uuid.New(),
	}

	if tenant.ID == uuid.Nil {
		t.Error("ID should not be Nil UUID")
	}

	// Test multiple UUIDs are different
	id1 := uuid.New()
	id2 := uuid.New()

	if id1 == id2 {
		t.Error("UUIDs should be unique")
	}
}