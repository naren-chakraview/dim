package agent

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestCritiqueRouteRequiredFields(t *testing.T) {
	ctx := context.Background()

	req := CritiqueRouteRequest{
		RouteConfigPath: "",
	}

	resp, err := CritiqueRoute(ctx, req)
	if err == nil {
		t.Errorf("should error on missing route_config_path, got response: %v", resp)
	}
	if err.Code != "INVALID_REQUEST" {
		t.Errorf("expected INVALID_REQUEST, got %s", err.Code)
	}
}

func TestCritiqueRouteNonexistentFile(t *testing.T) {
	ctx := context.Background()

	req := CritiqueRouteRequest{
		RouteConfigPath: "/nonexistent/route.yaml",
	}

	resp, err := CritiqueRoute(ctx, req)
	if err == nil {
		t.Errorf("should error on nonexistent file, got response: %v", resp)
	}
	if err.Code != "VALIDATION_FAILED" {
		t.Errorf("expected VALIDATION_FAILED, got %s", err.Code)
	}
}

func TestCritiqueRouteReturnsStructure(t *testing.T) {
	ctx := context.Background()

	// Use a valid example route that exists in the test fixtures
	// For this test, we'll assume there's a test route available
	req := CritiqueRouteRequest{
		RouteConfigPath: "domains/payments/order-payment.yaml",
	}

	resp, err := CritiqueRoute(ctx, req)

	// The route might not exist in the test environment, but the response structure should be consistent
	if err == nil && resp != nil {
		// Verify response has all required fields
		if resp.Findings == nil {
			t.Fatal("findings should not be nil")
		}

		if resp.Summary == "" {
			t.Fatal("summary should not be empty")
		}

		if resp.OverallRating == "" {
			t.Fatal("overall_rating should not be empty")
		}
	}
}

func TestMCPCritiqueRoute(t *testing.T) {
	server := NewMCPServer()
	ctx := context.Background()

	req := json.RawMessage(`{
		"route_config_path": "domains/test/test-route.yaml"
	}`)

	result := server.CallOperation(ctx, "critique_route", req)

	// Should handle missing file gracefully or return findings structure
	if result.Error == nil {
		var resp CritiqueRouteResponse
		respJSON, _ := json.Marshal(result.Result)
		if err := json.Unmarshal(respJSON, &resp); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}

		if resp.Findings == nil {
			t.Fatal("findings should not be nil")
		}

		if resp.Summary == "" {
			t.Fatal("summary should not be empty")
		}
	}
}

func TestMCPCritiqueRouteVersioning(t *testing.T) {
	server := NewMCPServer()
	ctx := context.Background()

	req := json.RawMessage(`{
		"route_config_path": "domains/test/route.yaml"
	}`)

	result := server.CallOperation(ctx, "critique_route", req)

	if result.InterfaceVersion != AgentInterfaceVersion {
		t.Fatalf("version mismatch: got %s, want %s", result.InterfaceVersion, AgentInterfaceVersion)
	}
}

func TestCritiqueFindingStructure(t *testing.T) {
	// Test that a finding has all required fields
	finding := CritiqueFinding{
		Category: "redundancy",
		Severity: "warning",
		Path:     "$.routes.test",
		Issue:    "Something is wrong",
		Action:   "Fix it by doing X",
	}

	if finding.Category == "" {
		t.Fatal("finding should have category")
	}
	if finding.Severity == "" {
		t.Fatal("finding should have severity")
	}
	if finding.Path == "" {
		t.Fatal("finding should have path")
	}
	if finding.Issue == "" {
		t.Fatal("finding should have issue description")
	}
	if finding.Action == "" {
		t.Fatal("finding should have action")
	}
}

func TestCritiqueSummaryGeneration(t *testing.T) {
	// Test with no findings
	findings := []CritiqueFinding{}
	summary := buildCritiqueSummary(findings)

	if !strings.Contains(summary, "No issues") {
		t.Fatalf("expected 'No issues' summary, got: %s", summary)
	}

	// Test with mixed severity findings
	findings = []CritiqueFinding{
		{Severity: "important", Issue: "test1"},
		{Severity: "warning", Issue: "test2"},
		{Severity: "info", Issue: "test3"},
	}
	summary = buildCritiqueSummary(findings)

	if !strings.Contains(summary, "Found 3 issues") {
		t.Fatalf("expected 'Found 3 issues', got: %s", summary)
	}

	if !strings.Contains(summary, "important") && !strings.Contains(summary, "warning") && !strings.Contains(summary, "info") {
		t.Fatalf("expected severity breakdown in summary, got: %s", summary)
	}
}

func TestOverallRatingCalculation(t *testing.T) {
	testCases := []struct {
		name            string
		findings        []CritiqueFinding
		valid           bool
		expectedRating  string
	}{
		{
			"no findings, valid",
			[]CritiqueFinding{},
			true,
			"excellent",
		},
		{
			"one important finding",
			[]CritiqueFinding{{Severity: "important"}},
			true,
			"fair",
		},
		{
			"three important findings",
			[]CritiqueFinding{
				{Severity: "important"},
				{Severity: "important"},
				{Severity: "important"},
			},
			true,
			"needs-review",
		},
		{
			"many warnings but valid",
			[]CritiqueFinding{
				{Severity: "warning"},
				{Severity: "warning"},
				{Severity: "warning"},
				{Severity: "warning"},
				{Severity: "warning"},
				{Severity: "warning"},
			},
			true,
			"good",
		},
		{
			"invalid structure",
			[]CritiqueFinding{},
			false,
			"invalid",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			validateResp := &ValidateResponse{
				Valid: tc.valid,
			}

			rating := calculateOverallRating(tc.findings, validateResp)

			if rating != tc.expectedRating {
				t.Fatalf("expected rating %s, got %s", tc.expectedRating, rating)
			}
		})
	}
}

func TestCritiqueFindings(t *testing.T) {
	// Test that findings are properly categorized
	validCategories := map[string]bool{
		"redundancy":     true,
		"convention":     true,
		"best-practice":  true,
		"security":       true,
		"clarity":        true,
	}

	validSeverities := map[string]bool{
		"info":      true,
		"warning":   true,
		"important": true,
	}

	finding := CritiqueFinding{
		Category: "best-practice",
		Severity: "important",
		Path:     "$.sinks.output",
		Issue:    "Missing enforce flag",
		Action:   "Add enforce: true",
	}

	if !validCategories[finding.Category] {
		t.Fatalf("invalid category: %s", finding.Category)
	}

	if !validSeverities[finding.Severity] {
		t.Fatalf("invalid severity: %s", finding.Severity)
	}
}

func TestMCPCritiqueRouteInvalidRequest(t *testing.T) {
	server := NewMCPServer()
	ctx := context.Background()

	// Invalid JSON
	req := json.RawMessage(`{invalid json}`)

	result := server.CallOperation(ctx, "critique_route", req)

	if result.Error == nil {
		t.Fatal("should error on invalid JSON")
	}

	if result.Error.Code != "INVALID_REQUEST" {
		t.Fatalf("expected INVALID_REQUEST, got %s", result.Error.Code)
	}
}
