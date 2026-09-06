package tenant

import (
	"testing"
)

// TestManagerRegisterTenant verifies tenant registration (M3.4.2)
func TestManagerRegisterTenant(t *testing.T) {
	manager := NewManager()

	cfg := &Config{
		Name:               "acme-corp",
		Description:        "ACME Corporation",
		MessageRateLimit:   1000,
		WorkerSlots:        50,
		LineageQuotaRecords: 1000000,
		Priority:           PriorityHigh,
	}

	err := manager.RegisterTenant(cfg)
	if err != nil {
		t.Fatalf("RegisterTenant failed: %v", err)
	}

	retrieved := manager.GetTenant("acme-corp")
	if retrieved.Name != "acme-corp" {
		t.Errorf("Expected 'acme-corp', got %q", retrieved.Name)
	}

	if retrieved.MessageRateLimit != 1000 {
		t.Errorf("Expected rate limit 1000, got %d", retrieved.MessageRateLimit)
	}
}

// TestManagerDefaultTenant verifies default config is used for unknown domains (M3.4.2)
func TestManagerDefaultTenant(t *testing.T) {
	manager := NewManager()

	// Request unknown domain
	cfg := manager.GetTenant("unknown-domain")

	if cfg == nil {
		t.Fatal("Expected default config, got nil")
	}

	if cfg.Name != "default" {
		t.Errorf("Expected default tenant, got %q", cfg.Name)
	}

	if cfg.MessageRateLimit != 100 {
		t.Errorf("Expected default rate limit 100, got %d", cfg.MessageRateLimit)
	}
}

// TestManagerMultipleTenants verifies multiple tenants with different quotas (M3.4.2)
func TestManagerMultipleTenants(t *testing.T) {
	manager := NewManager()

	tenants := []*Config{
		{
			Name:                "acme-corp",
			MessageRateLimit:    5000,
			WorkerSlots:         100,
			LineageQuotaRecords: 5000000,
			Priority:            PriorityHigh,
		},
		{
			Name:                "partners",
			MessageRateLimit:    1000,
			WorkerSlots:         30,
			LineageQuotaRecords: 500000,
			Priority:            PriorityNormal,
		},
		{
			Name:                "staging",
			MessageRateLimit:    100,
			WorkerSlots:         10,
			LineageQuotaRecords: 50000,
			Priority:            PriorityLow,
		},
	}

	for _, cfg := range tenants {
		if err := manager.RegisterTenant(cfg); err != nil {
			t.Fatalf("Failed to register %q: %v", cfg.Name, err)
		}
	}

	if manager.Count() != 3 {
		t.Errorf("Expected 3 tenants, got %d", manager.Count())
	}

	// Verify each has correct limits
	acme := manager.GetTenant("acme-corp")
	if acme.WorkerSlots != 100 {
		t.Errorf("Expected ACME 100 slots, got %d", acme.WorkerSlots)
	}

	partners := manager.GetTenant("partners")
	if partners.WorkerSlots != 30 {
		t.Errorf("Expected Partners 30 slots, got %d", partners.WorkerSlots)
	}
}

// TestUsageUtilization verifies utilization percentage calculations (M3.4.2)
func TestUsageUtilization(t *testing.T) {
	usage := &Usage{
		MessageRateCurrent:    750,
		WorkerSlotsCurrent:    40,
		LineageRecordsCurrent: 900000,
	}

	messageUtil := usage.MessageRateUtilization(1000)
	if messageUtil != 75.0 {
		t.Errorf("Expected 75 percent message utilization, got %.1f percent", messageUtil)
	}

	slotUtil := usage.WorkerSlotsUtilization(50)
	if slotUtil != 80.0 {
		t.Errorf("Expected 80 percent slot utilization, got %.1f percent", slotUtil)
	}

	lineageUtil := usage.LineageUtilization(1000000)
	if lineageUtil != 90.0 {
		t.Errorf("Expected 90%% lineage utilization, got %.1f%%", lineageUtil)
	}
}

// TestUsageUnlimitedQuota verifies unlimited quotas show 0% (M3.4.2)
func TestUsageUnlimitedQuota(t *testing.T) {
	usage := &Usage{
		MessageRateCurrent:    1000000,
		WorkerSlotsCurrent:    1000,
		LineageRecordsCurrent: 10000000,
	}

	// Zero quota = unlimited
	if usage.MessageRateUtilization(0) != 0 {
		t.Error("Expected 0% for unlimited message rate")
	}

	if usage.WorkerSlotsUtilization(0) != 0 {
		t.Error("Expected 0% for unlimited worker slots")
	}

	if usage.LineageUtilization(0) != 0 {
		t.Error("Expected 0% for unlimited lineage")
	}
}

// TestParseTenanConfig verifies YAML parsing (M3.4.2)
func TestParseTenanConfig(t *testing.T) {
	data := map[string]interface{}{
		"description":              "Test Domain",
		"message_rate_limit":       float64(2000),
		"message_rate_strategy":    "queue",
		"worker_slots":             float64(75),
		"lineage_quota_records":    float64(2000000),
		"lineage_quota_strategy":   "expire_oldest",
		"lineage_quota_retention_days": float64(60),
		"priority":                 "high",
	}

	cfg, err := parseTenanConfig("test-domain", data)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if cfg.Name != "test-domain" {
		t.Errorf("Expected name 'test-domain', got %q", cfg.Name)
	}

	if cfg.MessageRateLimit != 2000 {
		t.Errorf("Expected rate 2000, got %d", cfg.MessageRateLimit)
	}

	if cfg.MessageRateStrategy != RateLimitQueue {
		t.Errorf("Expected queue strategy, got %s", cfg.MessageRateStrategy)
	}

	if cfg.WorkerSlots != 75 {
		t.Errorf("Expected 75 slots, got %d", cfg.WorkerSlots)
	}

	if cfg.Priority != PriorityHigh {
		t.Errorf("Expected high priority, got %s", cfg.Priority)
	}
}
