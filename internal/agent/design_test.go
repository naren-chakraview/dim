package agent

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestScaffoldFromIntentPassthrough(t *testing.T) {
	ctx := context.Background()
	req := ScaffoldFromIntentRequest{
		Intent: "connect webhook to file output",
		Domain: "data-pipeline",
	}

	resp, err := ScaffoldFromIntent(ctx, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !resp.Success {
		t.Fatal("scaffold should produce valid route")
	}

	if !strings.Contains(resp.RouteYAML, "sources:") {
		t.Fatal("route YAML should contain sources")
	}

	if !strings.Contains(resp.RouteYAML, "sinks:") {
		t.Fatal("route YAML should contain sinks")
	}

	if resp.TemplateUsed == "" {
		t.Fatal("should report which template was used")
	}

	if len(resp.NextSteps) == 0 {
		t.Fatal("should provide next steps")
	}
}

func TestScaffoldFromIntentTransform(t *testing.T) {
	ctx := context.Background()
	req := ScaffoldFromIntentRequest{
		Intent: "webhook to Kafka with PII redaction",
		Domain: "payments",
	}

	resp, err := ScaffoldFromIntent(ctx, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.TemplateUsed != "transform" {
		t.Fatalf("should detect transform template, got %s", resp.TemplateUsed)
	}

	if !strings.Contains(resp.RouteYAML, "translate:") {
		t.Fatal("transform template should include translate step")
	}

	if len(resp.Placeholders) == 0 {
		t.Fatal("should identify placeholders for human review")
	}
}

func TestScaffoldFromIntentContractEnforced(t *testing.T) {
	ctx := context.Background()
	req := ScaffoldFromIntentRequest{
		Intent: "validate messages against contract before publishing",
		Domain: "orders",
	}

	resp, err := ScaffoldFromIntent(ctx, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.TemplateUsed != "contract-enforced" {
		t.Fatalf("should detect contract-enforced template, got %s", resp.TemplateUsed)
	}

	if !strings.Contains(resp.RouteYAML, "enforce: true") {
		t.Fatal("contract-enforced template should have enforce: true")
	}
}

func TestScaffoldValidationPasses(t *testing.T) {
	ctx := context.Background()
	req := ScaffoldFromIntentRequest{
		Intent: "webhook to file",
		Domain: "test",
	}

	resp, err := ScaffoldFromIntent(ctx, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.ValidationResult == nil {
		t.Fatal("should include validation result")
	}

	if !resp.ValidationResult.Valid {
		t.Fatalf("generated route should be valid, got errors: %v", resp.ValidationResult.Errors)
	}
}

func TestScaffoldRequiredFields(t *testing.T) {
	ctx := context.Background()

	testCases := []struct {
		name string
		req  ScaffoldFromIntentRequest
	}{
		{"missing intent", ScaffoldFromIntentRequest{Domain: "test"}},
		{"missing domain", ScaffoldFromIntentRequest{Intent: "test"}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := ScaffoldFromIntent(ctx, tc.req)
			if err == nil {
				t.Errorf("%s: should error, got response: %v", tc.name, resp)
			}
			if err.Code != "INVALID_REQUEST" {
				t.Errorf("%s: expected INVALID_REQUEST, got %s", tc.name, err.Code)
			}
		})
	}
}

func TestScaffoldInvalidTemplateShape(t *testing.T) {
	ctx := context.Background()
	req := ScaffoldFromIntentRequest{
		Intent:        "webhook to file",
		Domain:        "test",
		TemplateShape: "invalid-shape",
	}

	resp, err := ScaffoldFromIntent(ctx, req)
	if err == nil {
		t.Errorf("should error for invalid template shape, got response: %v", resp)
	}
	if err.Code != "INVALID_REQUEST" {
		t.Errorf("expected INVALID_REQUEST, got %s", err.Code)
	}
}

func TestScaffoldReasoningProvided(t *testing.T) {
	ctx := context.Background()
	req := ScaffoldFromIntentRequest{
		Intent: "transform payment events to Kafka with redaction",
		Domain: "payments",
	}

	resp, err := ScaffoldFromIntent(ctx, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Reasoning == "" {
		t.Fatal("should provide reasoning for template choice")
	}

	if !strings.Contains(resp.Reasoning, "transform") {
		t.Fatal("reasoning should explain template selection")
	}
}

func TestScaffoldPlaceholdersForIntentMissingInfo(t *testing.T) {
	ctx := context.Background()
	req := ScaffoldFromIntentRequest{
		Intent: "do something",  // Very vague, missing source/sink info
		Domain: "test",
	}

	resp, err := ScaffoldFromIntent(ctx, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should identify missing source and/or sink
	if len(resp.Placeholders) == 0 {
		t.Fatal("should identify placeholders for missing source/sink")
	}

	foundSource := false
	foundSink := false
	for _, p := range resp.Placeholders {
		if strings.Contains(p.Path, "source") {
			foundSource = true
		}
		if strings.Contains(p.Path, "sink") {
			foundSink = true
		}
	}

	if !foundSource && !foundSink {
		t.Fatal("should have placeholders for source or sink")
	}
}

func TestMCPScaffoldFromIntent(t *testing.T) {
	server := NewMCPServer()
	ctx := context.Background()

	req := json.RawMessage(`{
		"intent": "webhook to file output",
		"domain": "test-domain"
	}`)

	result := server.CallOperation(ctx, "scaffold_from_intent", req)

	if result.Error != nil {
		t.Fatalf("unexpected error: %v", result.Error)
	}

	// Verify response structure
	var resp ScaffoldFromIntentResponse
	respJSON, _ := json.Marshal(result.Result)
	if err := json.Unmarshal(respJSON, &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if !resp.Success {
		t.Fatal("scaffold should produce valid route")
	}

	if resp.TemplateUsed == "" {
		t.Fatal("should report which template was used")
	}

	if resp.RouteYAML == "" {
		t.Fatal("should include generated YAML")
	}
}

func TestMCPScaffoldFromIntentVersioning(t *testing.T) {
	server := NewMCPServer()
	ctx := context.Background()

	req := json.RawMessage(`{
		"intent": "webhook to file",
		"domain": "test"
	}`)

	result := server.CallOperation(ctx, "scaffold_from_intent", req)

	if result.InterfaceVersion != AgentInterfaceVersion {
		t.Fatalf("version mismatch: got %s, want %s", result.InterfaceVersion, AgentInterfaceVersion)
	}
}

func TestScaffoldFromIntentOperationRegistered(t *testing.T) {
	server := NewMCPServer()
	operations := server.ListOperations()

	found := false
	for _, op := range operations {
		if op.Name == "scaffold_from_intent" {
			found = true
			if op.DisplayName == "" {
				t.Fatal("scaffold_from_intent should have display name")
			}
			if op.Description == "" {
				t.Fatal("scaffold_from_intent should have description")
			}
		}
	}

	if !found {
		t.Fatal("scaffold_from_intent operation not registered")
	}
}

func TestCritiqueRouteOperationRegistered(t *testing.T) {
	server := NewMCPServer()
	operations := server.ListOperations()

	found := false
	for _, op := range operations {
		if op.Name == "critique_route" {
			found = true
		}
	}

	if !found {
		t.Fatal("critique_route operation not registered")
	}
}
