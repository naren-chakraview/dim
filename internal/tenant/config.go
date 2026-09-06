package tenant

import (
	"fmt"
	"sync"
	"time"
)

// Priority represents the priority level of a tenant for future QoS features
type Priority string

const (
	PriorityHigh   Priority = "high"
	PriorityNormal Priority = "normal"
	PriorityLow    Priority = "low"
)

// RateLimitStrategy determines how to handle rate limit breaches
type RateLimitStrategy string

const (
	RateLimitReject RateLimitStrategy = "reject"  // Return 429
	RateLimitQueue  RateLimitStrategy = "queue"   // Queue and wait
)

// LineageQuotaStrategy determines how to handle lineage quota exhaustion
type LineageQuotaStrategy string

const (
	LineageQuotaReject LineageQuotaStrategy = "reject"         // Skip write
	LineageQuotaExpire LineageQuotaStrategy = "expire_oldest"  // Delete oldest
)

// Config represents resource quotas for a single tenant
type Config struct {
	// Name is the domain identifier
	Name string

	// Description of this tenant (for logging/observability)
	Description string

	// MessageRateLimit is the maximum messages per second for this domain
	// 0 = unlimited
	MessageRateLimit int

	// MessageRateStrategy determines how to handle rate limit breaches
	MessageRateStrategy RateLimitStrategy

	// WorkerSlots is the maximum concurrent messages for this domain
	// 0 = unlimited
	WorkerSlots int

	// LineageQuotaRecords is the maximum lineage records for this domain
	// 0 = unlimited
	LineageQuotaRecords int64

	// LineageQuotaStrategy determines how to handle quota exhaustion
	LineageQuotaStrategy LineageQuotaStrategy

	// LineageRetentionDays is the retention period if using expire_oldest
	LineageRetentionDays int

	// Priority is the SLA tier (for future prioritization)
	Priority Priority
}

// Manager manages tenant configurations for an instance
type Manager struct {
	mu       sync.RWMutex
	tenants  map[string]*Config
	defaults *Config
}

// NewManager creates a new tenant manager with a default tenant
func NewManager() *Manager {
	return &Manager{
		tenants: make(map[string]*Config),
		defaults: &Config{
			Name:                 "default",
			Description:          "Default tenant for unmapped domains",
			MessageRateLimit:     100, // Conservative default
			MessageRateStrategy:  RateLimitReject,
			WorkerSlots:          5,
			LineageQuotaRecords:  10000,
			LineageQuotaStrategy: LineageQuotaReject,
			Priority:             PriorityLow,
		},
	}
}

// RegisterTenant adds or updates a tenant configuration
func (m *Manager) RegisterTenant(cfg *Config) error {
	if cfg == nil {
		return fmt.Errorf("tenant config cannot be nil")
	}
	if cfg.Name == "" {
		return fmt.Errorf("tenant name cannot be empty")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.tenants[cfg.Name] = cfg
	return nil
}

// GetTenant retrieves a tenant's configuration by name
// Returns default config if tenant not found
func (m *Manager) GetTenant(name string) *Config {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if cfg, exists := m.tenants[name]; exists {
		return cfg
	}
	return m.defaults
}

// SetDefaultTenant updates the default configuration
func (m *Manager) SetDefaultTenant(cfg *Config) error {
	if cfg == nil {
		return fmt.Errorf("default tenant config cannot be nil")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.defaults = cfg
	return nil
}

// ListTenants returns all registered tenant names
func (m *Manager) ListTenants() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	names := make([]string, 0, len(m.tenants))
	for name := range m.tenants {
		names = append(names, name)
	}
	return names
}

// Count returns the number of registered tenants (not including default)
func (m *Manager) Count() int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return len(m.tenants)
}

// LoadFromYAML loads tenant configurations from YAML structure
// Expected structure: map[string]map[string]interface{} where keys are tenant names
func (m *Manager) LoadFromYAML(data map[string]interface{}) error {
	if data == nil {
		return nil // No tenants configured, use defaults
	}

	for name, tenantData := range data {
		cfg, err := parseTenanConfig(name, tenantData)
		if err != nil {
			return fmt.Errorf("failed to parse tenant %q: %w", name, err)
		}

		if err := m.RegisterTenant(cfg); err != nil {
			return err
		}
	}

	return nil
}

// parseTenanConfig converts a YAML-parsed map into a Config struct
func parseTenanConfig(name string, data interface{}) (*Config, error) {
	cfg := &Config{
		Name:                name,
		MessageRateLimit:    100,
		MessageRateStrategy: RateLimitReject,
		WorkerSlots:         5,
		LineageQuotaRecords: 10000,
		LineageQuotaStrategy: LineageQuotaReject,
		Priority:            PriorityNormal,
	}

	m, ok := data.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("expected map, got %T", data)
	}

	// Parse fields
	if v, ok := m["description"].(string); ok {
		cfg.Description = v
	}

	if v, ok := m["message_rate_limit"].(float64); ok {
		cfg.MessageRateLimit = int(v)
	}

	if v, ok := m["message_rate_strategy"].(string); ok {
		cfg.MessageRateStrategy = RateLimitStrategy(v)
	}

	if v, ok := m["worker_slots"].(float64); ok {
		cfg.WorkerSlots = int(v)
	}

	if v, ok := m["lineage_quota_records"].(float64); ok {
		cfg.LineageQuotaRecords = int64(v)
	}

	if v, ok := m["lineage_quota_strategy"].(string); ok {
		cfg.LineageQuotaStrategy = LineageQuotaStrategy(v)
	}

	if v, ok := m["lineage_quota_retention_days"].(float64); ok {
		cfg.LineageRetentionDays = int(v)
	}

	if v, ok := m["priority"].(string); ok {
		cfg.Priority = Priority(v)
	}

	return cfg, nil
}

// Usage tracks current resource consumption for a tenant
type Usage struct {
	// MessageRateCurrent is the current message rate (messages per second)
	MessageRateCurrent float64

	// WorkerSlotsCurrent is the current number of occupied slots
	WorkerSlotsCurrent int

	// LineageRecordsCurrent is the current number of lineage records
	LineageRecordsCurrent int64

	// LastUpdated is when this usage snapshot was taken
	LastUpdated time.Time
}

// Utilization returns a percentage of quota used (0-100)
func (u *Usage) MessageRateUtilization(quota int) float64 {
	if quota == 0 {
		return 0 // Unlimited
	}
	pct := (u.MessageRateCurrent / float64(quota)) * 100
	if pct > 100 {
		pct = 100
	}
	return pct
}

// WorkerSlotsUtilization returns a percentage of worker slots used (0-100)
func (u *Usage) WorkerSlotsUtilization(quota int) float64 {
	if quota == 0 {
		return 0 // Unlimited
	}
	pct := (float64(u.WorkerSlotsCurrent) / float64(quota)) * 100
	if pct > 100 {
		pct = 100
	}
	return pct
}

// LineageUtilization returns a percentage of lineage quota used (0-100)
func (u *Usage) LineageUtilization(quota int64) float64 {
	if quota == 0 {
		return 0 // Unlimited
	}
	pct := (float64(u.LineageRecordsCurrent) / float64(quota)) * 100
	if pct > 100 {
		pct = 100
	}
	return pct
}
