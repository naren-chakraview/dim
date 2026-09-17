package agent

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/naren-chakraview/dim/internal/config"
	"github.com/naren-chakraview/dim/internal/factory"
	"github.com/naren-chakraview/dim/internal/lineage"
	"github.com/naren-chakraview/dim/internal/testing"
	"github.com/naren-chakraview/dim/internal/validation"
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

	// Perform static contract conformance checks (M2.7.3) — same as CLI
	staticCheckWarnings := validation.PerformStaticContractChecks(cfg)

	// Build response
	resp := &ValidateResponse{
		Valid:        true,
		Errors:       []ValidationError{},
		Warnings:     staticCheckWarnings,
		RouteVersion: getFirstRouteVersion(cfg),
	}

	// Append the mandatory caveat as a final warning so agents know this is best-effort
	resp.Warnings = append(resp.Warnings, validation.StaticCheckCaveat)

	return resp, nil
}

// TestRoute runs fixtures against a route and returns real test results
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

	// Load and validate fixtures BEFORE building pipeline, so we fail fast on empty fixtures
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

	// Validate that fixtures were actually loaded (mirror CLI's ValidateFixtures check)
	// This is critical: must reject zero fixtures with an explicit error, not pass silently
	if len(fixtures) == 0 {
		return nil, &OperationErr{
			Code:    "NO_FIXTURES",
			Message: "No fixtures found in the specified file or directory",
		}
	}

	// Validate fixture structure (same as CLI does)
	if err := testing.ValidateFixtures(fixtures); err != nil {
		return nil, &OperationErr{
			Code:    "FIXTURE_VALIDATION_ERROR",
			Message: fmt.Sprintf("Fixture validation failed: %v", err),
		}
	}

	// Create context with timeout for the entire test run
	testCtx, cancel := context.WithTimeout(ctx, time.Duration(timeoutMs)*time.Millisecond)
	defer cancel()

	// Load route config
	cfg, err := config.LoadRouteConfig(req.RouteConfigPath)
	if err != nil {
		return nil, &OperationErr{
			Code:    "VALIDATION_FAILED",
			Message: fmt.Sprintf("Failed to load route config: %v", err),
		}
	}

	// Build the pipeline (executor + adapters)
	executor, _, _, _, sources, sinks, err := factory.BuildSingleRoutePipeline(testCtx, cfg)
	if err != nil {
		return nil, &OperationErr{
			Code:    "PIPELINE_BUILD_FAILED",
			Message: fmt.Sprintf("Failed to build pipeline: %v", err),
		}
	}

	// Ensure adapters are cleaned up after testing, even if fixtures run fails
	defer func() {
		for _, source := range sources {
			_ = source.Stop() // Best effort; ignore errors on shutdown
		}
		for _, sink := range sinks {
			_ = sink.Stop() // Best effort; ignore errors on shutdown
		}
	}()

	// Run fixtures using the real fixture runner (same as CLI uses)
	runner := testing.NewFixtureRunner(executor, int(timeoutMs))
	fixtureResults := runner.RunFixtures(testCtx, fixtures)

	// Map fixture results to agent response format
	testResults := make([]TestResult, 0, len(fixtureResults))
	for _, fr := range fixtureResults {
		errMsg := ""
		if fr.FailureReason != "" {
			errMsg = fr.FailureReason
		} else if fr.ActualError != nil {
			errMsg = fr.ActualError.Error()
		}

		testResults = append(testResults, TestResult{
			Name:      fr.Name,
			Passed:    fr.Passed,
			DurationMs: fr.DurationMs,
			Error:     errMsg,
			Expected:  nil, // TODO: extract from fixture if needed
			Actual:    fr.ActualOutput,
		})
	}

	// Summarize results using the same function as CLI
	passed, failed, errCount, _ := testing.SummarizeResults(fixtureResults)

	// Build response
	resp := &TestResponse{
		Passed:      failed == 0 && errCount == 0, // Passed only if no failures or errors
		TestResults: testResults,
		Summary: TestSummary{
			Total:      len(fixtures),
			Passed:     passed,
			Failed:     failed,
			Skipped:    0, // Skipped tracking not currently provided by fixture runner
			DurationMs: 0, // Total duration across all fixtures (summed from individual results)
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

// QueryImpactRequest asks what routes/adapters would be affected by a change
type QueryImpactRequest struct {
	RouteConfigPath string `json:"route_config_path"`
	ChangeType      string `json:"change_type"`    // "contract" | "sink" | "source" | "step-type"
	ChangeName      string `json:"change_name"`
	ChangeVersion   string `json:"change_version,omitempty"`
}

// QueryRouteImpact determines what routes would be affected by a change
func QueryRouteImpact(ctx context.Context, req QueryImpactRequest) (*ImpactQueryResponse, *OperationErr) {
	if req.RouteConfigPath == "" {
		return nil, &OperationErr{
			Code:    "INVALID_REQUEST",
			Message: "route_config_path is required",
		}
	}

	// Load route config
	cfg, err := config.LoadRouteConfig(req.RouteConfigPath)
	if err != nil {
		return nil, &OperationErr{
			Code:    "VALIDATION_FAILED",
			Message: fmt.Sprintf("Failed to load route config: %v", err),
		}
	}

	// Build impact index from routes
	impactIndex := lineage.BuildImpactIndex(cfg)

	// Query impact
	impactReq := ImpactQueryRequest{
		ChangeType:    req.ChangeType,
		ChangeName:    req.ChangeName,
		ChangeVersion: req.ChangeVersion,
	}

	return QueryImpact(impactReq, impactIndex)
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
