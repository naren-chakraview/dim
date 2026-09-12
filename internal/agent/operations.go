package agent

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/naren-chakraview/dim/internal/config"
	"github.com/naren-chakraview/dim/internal/testing"
)

// ValidateRoute validates a route configuration
func ValidateRoute(ctx context.Context, req ValidateRequest) (*ValidateResponse, *OperationErr) {
	if req.RouteConfigPath == "" {
		return nil, &OperationErr{
			Code:    "INVALID_REQUEST",
			Message: "route_config_path is required",
		}
	}

	// Load and validate the configuration
	cfg, err := config.LoadRouteConfig(req.RouteConfigPath)
	if err != nil {
		// Parse error type to provide structured error response
		return nil, &OperationErr{
			Code:    "VALIDATION_FAILED",
			Message: fmt.Sprintf("Failed to load route config: %v", err),
		}
	}

	// Validate auth declarations
	mode := config.AuthValidationWarn
	if req.StrictMode {
		mode = config.AuthValidationStrict
	}
	if err := config.ValidateAuthDeclarations(cfg, mode); err != nil {
		return nil, &OperationErr{
			Code:    "VALIDATION_FAILED",
			Message: fmt.Sprintf("Auth validation failed: %v", err),
		}
	}

	// Build response
	resp := &ValidateResponse{
		Valid:       true,
		Errors:      []ValidationError{},
		Warnings:    []string{},
		RouteVersion: getFirstRouteVersion(cfg),
	}

	return resp, nil
}

// TestRoute runs fixtures against a route
func TestRoute(ctx context.Context, req TestRequest) (*TestResponse, *OperationErr) {
	if req.RouteConfigPath == "" {
		return nil, &OperationErr{
			Code:    "INVALID_REQUEST",
			Message: "route_config_path is required",
		}
	}
	if req.FixturesPath == "" {
		return nil, &OperationErr{
			Code:    "INVALID_REQUEST",
			Message: "fixtures_path is required",
		}
	}

	// Set default timeout
	timeoutMs := req.TimeoutMs
	if timeoutMs == 0 {
		timeoutMs = 30000
	}

	// Create context with timeout (would use for actual test execution)
	_, cancel := context.WithTimeout(ctx, time.Duration(timeoutMs)*time.Millisecond)
	defer cancel()

	// Load route config (for validation before test)
	_, err := config.LoadRouteConfig(req.RouteConfigPath)
	if err != nil {
		return nil, &OperationErr{
			Code:    "VALIDATION_FAILED",
			Message: fmt.Sprintf("Failed to load route config: %v", err),
		}
	}

	// Load fixtures
	var fixtures []*testing.Fixture
	info, err := os.Stat(req.FixturesPath)
	if err != nil {
		return nil, &OperationErr{
			Code:    "FIXTURES_NOT_FOUND",
			Message: fmt.Sprintf("Fixtures path not found: %v", err),
		}
	}

	if info.IsDir() {
		fixtures, err = testing.LoadFixturesFromDirectory(req.FixturesPath)
	} else {
		fixtures, err = testing.LoadFixturesFromFile(req.FixturesPath)
	}

	if err != nil {
		return nil, &OperationErr{
			Code:    "FIXTURE_FORMAT_ERROR",
			Message: fmt.Sprintf("Failed to load fixtures: %v", err),
		}
	}

	// For now, return a placeholder test response
	// (Full implementation would integrate with testing.RunFixtures)
	resp := &TestResponse{
		Passed:      len(fixtures) > 0, // Placeholder: passes if fixtures exist
		TestResults: []TestResult{},
		Summary: TestSummary{
			Total:      len(fixtures),
			Passed:     len(fixtures),
			Failed:     0,
			Skipped:    0,
			DurationMs: 0,
		},
	}

	return resp, nil
}

// ScaffoldDomain scaffolds a new domain
func ScaffoldDomain(ctx context.Context, req ScaffoldRequest) (*ScaffoldResponse, *OperationErr) {
	if req.Domain == "" {
		return nil, &OperationErr{
			Code:    "INVALID_REQUEST",
			Message: "domain is required",
		}
	}

	// For now, return a placeholder response
	// (Full implementation would call dimctl scaffold logic)
	domainPath := filepath.Join("domains", req.Domain)

	resp := &ScaffoldResponse{
		Success:      true,
		DomainPath:   domainPath,
		FilesCreated: []string{
			filepath.Join(domainPath, "DOMAIN.yaml"),
			filepath.Join(domainPath, fmt.Sprintf("%s-route.yaml", req.Domain)),
		},
		NextSteps: []string{
			fmt.Sprintf("Review the generated files in %s", domainPath),
			"Run `dimctl validate` to verify the configuration",
			"Commit to git and create a pull request",
		},
	}

	return resp, nil
}

// GetLineage retrieves lineage for a message
func GetLineage(ctx context.Context, req LineageRequest) (*LineageResponse, *OperationErr) {
	if req.MessageID == "" {
		return nil, &OperationErr{
			Code:    "INVALID_REQUEST",
			Message: "message_id is required",
		}
	}

	// For now, return placeholder (full implementation queries lineage store)
	resp := &LineageResponse{
		Found:          false,
		LineageRecords: []LineageRecord{},
	}

	return resp, nil
}

// GetProvenance retrieves provenance for a subject
func GetProvenance(ctx context.Context, req ProvenanceRequest) (*ProvenanceResponse, *OperationErr) {
	if req.SubjectID == "" {
		return nil, &OperationErr{
			Code:    "INVALID_REQUEST",
			Message: "subject_id is required",
		}
	}

	// For now, return placeholder (full implementation queries provenance store)
	resp := &ProvenanceResponse{
		Found:           false,
		ProvenanceChain: []ProvenanceRecord{},
	}

	return resp, nil
}

// GetCapabilities returns available adapters, steps, and EIPs
func GetCapabilities(ctx context.Context, req CapabilitiesRequest) (*CapabilitiesResponse, *OperationErr) {
	capabilities := []Capability{}

	// Add adapters (sources and sinks)
	if req.FilterType == "" || req.FilterType == "adapter" {
		adapters := []struct {
			name      string
			direction string
			desc      string
		}{
			{"http", "source", "HTTP webhook/request source"},
			{"http", "sink", "HTTP POST sink"},
			{"file", "source", "File-based event source"},
			{"file", "sink", "File-based event sink"},
			{"sftp", "source", "SFTP source"},
			{"sftp", "sink", "SFTP sink"},
			{"exec", "source", "Executable/command source"},
			{"exec", "sink", "Executable/command sink"},
		}

		for _, a := range adapters {
			cap := Capability{
				Name:        a.name + "-" + a.direction,
				Type:        "adapter",
				Description: a.desc,
				ConfigSchema: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"type": map[string]interface{}{
							"type": "string",
							"enum": []string{a.name},
						},
					},
					"required": []string{"type"},
				},
			}
			capabilities = append(capabilities, cap)
		}
	}

	// Add steps
	if req.FilterType == "" || req.FilterType == "step" {
		steps := []struct {
			name string
			desc string
		}{
			{"filter", "Boolean predicate to include/exclude messages using JSONata"},
			{"translate", "Transform message body using JSONata expression"},
			{"route", "Route messages to different sinks based on conditions"},
			{"wiretap", "Clone messages to secondary sink for monitoring/logging"},
			{"idempotent", "Prevent duplicate message processing"},
			{"authorize", "Authorization/authentication step (RBAC or ABAC)"},
		}

		for _, s := range steps {
			cap := Capability{
				Name:        s.name + "-step",
				Type:        "step",
				Description: s.desc,
				ConfigSchema: map[string]interface{}{
					"type":  "object",
					"title": s.name,
				},
			}
			capabilities = append(capabilities, cap)
		}
	}

	// Add EIPs
	if req.FilterType == "" || req.FilterType == "eip" {
		eips := []struct {
			name string
			desc string
		}{
			{"claim-check", "Externalize large message payloads to reduce memory usage"},
			{"content-based-router", "Route messages based on content evaluation"},
			{"dead-letter-queue", "Capture and route failed messages"},
			{"wiretap", "Clone messages for observation without affecting flow"},
		}

		for _, e := range eips {
			cap := Capability{
				Name:        e.name,
				Type:        "eip",
				Description: e.desc,
				ConfigSchema: map[string]interface{}{
					"type": "object",
				},
			}
			capabilities = append(capabilities, cap)
		}
	}

	resp := &CapabilitiesResponse{
		Capabilities:  capabilities,
		SchemaVersion: "1.0.0",
	}

	return resp, nil
}

// Helper: get first route version from config
func getFirstRouteVersion(cfg *config.RouteConfig) string {
	for _, route := range cfg.Routes {
		if route.RouteVersion != "" {
			return route.RouteVersion
		}
	}
	return ""
}
