package factory

import (
	"context"
	"net"
	"strconv"
	"testing"

	"github.com/naren-chakraview/dim/internal/config"
	"github.com/naren-chakraview/dim/internal/tenant"
)

// TestSingleRouteWithTenantManager verifies tenant isolation is wired in single-route pipeline (M3.4)
// This test may be skipped if port 8080 is in use (concurrent test execution)
func TestSingleRouteWithTenantManager(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg := &config.RouteConfig{
		Routes: map[string]config.RouteSpec{
			"customer-a-orders": {
				From:   "api",
				Domain: "customer-a", // Tenant domain (M3.4)
				Steps: []config.StepSpec{
					{
						Filter: &config.FilterSpec{Expr: "true"},
					},
				},
			},
		},
		Sinks: map[string]config.SinkSpec{
			"output": {
				Type: "file",
				Path: "./output/messages.jsonl",
			},
		},
	}

	// Create tenant manager with custom quotas
	tenantManager := tenant.NewManager()
	err := tenantManager.RegisterTenant(&tenant.Config{
		Name:                "customer-a",
		MessageRateLimit:    500,
		WorkerSlots:         10,
		LineageQuotaRecords: 50000,
	})
	if err != nil {
		t.Fatalf("RegisterTenant failed: %v", err)
	}

	// Build pipeline with tenant manager (M3.4+). The route's HTTP ingest port is
	// isolated: 8080 is often held by another process, and tests must not depend on it.
	t.Setenv("DIM_HTTP_PORT", freeTCPPort(t))
	executor, _, _, _, _, _, err := BuildSingleRoutePipelineWithTracing(ctx, cfg, nil, tenantManager)
	if err != nil {
		t.Fatalf("BuildSingleRoutePipelineWithTracing failed: %v", err)
	}

	if executor == nil {
		t.Fatal("Expected non-nil executor")
	}

	// Verify executor is tenant-aware (has rate limiter and slot manager)
	// The executor should have the domain and limiters set (M3.4+)
	if executor.GetDomain() != "customer-a" {
		t.Errorf("Expected domain 'customer-a', got %q", executor.GetDomain())
	}
}

// TestTenantManagerRegistration verifies tenant manager is correctly configured (M3.4)
func TestTenantManagerRegistration(t *testing.T) {
	tenantManager := tenant.NewManager()

	// Register multiple tenants with different quotas
	acmeCfg := &tenant.Config{
		Name:                "acme-corp",
		MessageRateLimit:    1000,
		WorkerSlots:         10,
		LineageQuotaRecords: 100000,
	}
	if err := tenantManager.RegisterTenant(acmeCfg); err != nil {
		t.Fatalf("RegisterTenant (acme) failed: %v", err)
	}

	globexCfg := &tenant.Config{
		Name:                "globex-inc",
		MessageRateLimit:    5000,
		WorkerSlots:         20,
		LineageQuotaRecords: 500000,
	}
	if err := tenantManager.RegisterTenant(globexCfg); err != nil {
		t.Fatalf("RegisterTenant (globex) failed: %v", err)
	}

	// Verify retrieval
	acmeRetrieved := tenantManager.GetTenant("acme-corp")
	if acmeRetrieved.MessageRateLimit != 1000 {
		t.Errorf("Expected rate limit 1000, got %d", acmeRetrieved.MessageRateLimit)
	}

	globexRetrieved := tenantManager.GetTenant("globex-inc")
	if globexRetrieved.WorkerSlots != 20 {
		t.Errorf("Expected worker slots 20, got %d", globexRetrieved.WorkerSlots)
	}

	// List tenants
	names := tenantManager.ListTenants()
	if len(names) != 2 {
		t.Errorf("Expected 2 tenants, got %d", len(names))
	}
}

// freeTCPPort returns a loopback port nothing is listening on right now.
func freeTCPPort(t *testing.T) string {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := l.Addr().(*net.TCPAddr).Port
	l.Close()
	return strconv.Itoa(port)
}
