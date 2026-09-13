package agent

import (
	"context"
	"encoding/json"
	"testing"
)

func TestMCPServerCreation(t *testing.T) {
	server := NewMCPServer()
	if server == nil {
		t.Fatal("failed to create MCP server")
	}

	if server.version != AgentInterfaceVersion {
		t.Fatalf("version mismatch: got %s, want %s", server.version, AgentInterfaceVersion)
	}

	// Verify all operations are registered (validate, test, scaffold, lineage, provenance, capabilities, scaffold_from_intent, critique_route)
	if len(server.operations) != 8 {
		t.Fatalf("expected 8 operations, got %d", len(server.operations))
	}
}

func TestValidateOperation(t *testing.T) {
	server := NewMCPServer()
	ctx := context.Background()

	// Test with invalid request (missing required field)
	invalidReq := json.RawMessage(`{}`)
	result := server.CallOperation(ctx, "validate_route", invalidReq)

	if result.InterfaceVersion != AgentInterfaceVersion {
		t.Fatalf("version mismatch in response")
	}

	if result.Error == nil {
		t.Fatal("expected error for invalid request")
	}

	if result.Error.Code != "INVALID_REQUEST" {
		t.Fatalf("expected INVALID_REQUEST, got %s", result.Error.Code)
	}
}

func TestOperationNotFound(t *testing.T) {
	server := NewMCPServer()
	ctx := context.Background()

	result := server.CallOperation(ctx, "nonexistent_operation", json.RawMessage(`{}`))

	if result.Error == nil {
		t.Fatal("expected error for nonexistent operation")
	}

	if result.Error.Code != "NOT_IMPLEMENTED" {
		t.Fatalf("expected NOT_IMPLEMENTED, got %s", result.Error.Code)
	}
}

func TestListOperations(t *testing.T) {
	server := NewMCPServer()
	operations := server.ListOperations()

	if len(operations) != 8 {
		t.Fatalf("expected 8 operations, got %d", len(operations))
	}

	// Verify known operations exist
	opNames := make(map[string]bool)
	for _, op := range operations {
		opNames[op.Name] = true
	}

	expectedOps := []string{
		"validate_route",
		"test_route",
		"scaffold_domain",
		"get_lineage",
		"get_provenance",
		"get_capabilities",
		"scaffold_from_intent",
		"critique_route",
	}

	for _, expected := range expectedOps {
		if !opNames[expected] {
			t.Fatalf("operation %s not found", expected)
		}
	}
}

func TestResponseEnvelopeStructure(t *testing.T) {
	server := NewMCPServer()
	ctx := context.Background()

	// Make a valid request to test response structure
	req := json.RawMessage(`{"route_config_path": "nonexistent.yaml"}`)
	result := server.CallOperation(ctx, "validate_route", req)

	// Verify envelope fields
	if result.InterfaceVersion == "" {
		t.Fatal("InterfaceVersion should not be empty")
	}

	if result.Timestamp.IsZero() {
		t.Fatal("Timestamp should not be zero")
	}

	// Should have error (file not found)
	if result.Error == nil {
		t.Fatal("expected error in response")
	}
}

func TestTestConnection(t *testing.T) {
	server := NewMCPServer()
	ctx := context.Background()

	result := server.TestConnection(ctx)

	if result.InterfaceVersion != AgentInterfaceVersion {
		t.Fatalf("version mismatch")
	}

	if result.Error != nil {
		t.Fatalf("TestConnection should not error: %v", result.Error)
	}

	// Verify result contains status
	if result.Result == nil {
		t.Fatal("result should not be nil")
	}

	resultMap, ok := result.Result.(map[string]interface{})
	if !ok {
		t.Fatal("result should be a map")
	}

	if status, exists := resultMap["status"]; !exists || status != "ok" {
		t.Fatalf("expected status='ok' in result")
	}
}

func TestScaffoldOperation(t *testing.T) {
	server := NewMCPServer()
	ctx := context.Background()

	req := json.RawMessage(`{
		"domain": "test-domain",
		"source_type": "http",
		"sink_type": "file"
	}`)

	result := server.CallOperation(ctx, "scaffold_domain", req)

	if result.Error != nil {
		t.Fatalf("unexpected error: %v", result.Error)
	}

	// Parse response
	var resp ScaffoldResponse
	resultJSON, _ := json.Marshal(result.Result)
	if err := json.Unmarshal(resultJSON, &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if !resp.Success {
		t.Fatal("scaffold should succeed")
	}

	if resp.DomainPath == "" {
		t.Fatal("domain_path should not be empty")
	}

	if len(resp.FilesCreated) == 0 {
		t.Fatal("files_created should not be empty")
	}

	if len(resp.NextSteps) == 0 {
		t.Fatal("next_steps should not be empty")
	}
}

func TestLineageOperation(t *testing.T) {
	server := NewMCPServer()
	ctx := context.Background()

	req := json.RawMessage(`{"message_id": "test-msg-123"}`)
	result := server.CallOperation(ctx, "get_lineage", req)

	if result.Error != nil {
		t.Fatalf("unexpected error: %v", result.Error)
	}

	// Verify structure
	var resp LineageResponse
	resultJSON, _ := json.Marshal(result.Result)
	if err := json.Unmarshal(resultJSON, &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	// Should work even with nonexistent message (returns empty list)
	if resp.LineageRecords == nil {
		t.Fatal("lineage_records should not be nil")
	}
}

func TestProvenanceOperation(t *testing.T) {
	server := NewMCPServer()
	ctx := context.Background()

	req := json.RawMessage(`{"subject_id": "test-subject-123"}`)
	result := server.CallOperation(ctx, "get_provenance", req)

	if result.Error != nil {
		t.Fatalf("unexpected error: %v", result.Error)
	}

	// Verify structure
	var resp ProvenanceResponse
	resultJSON, _ := json.Marshal(result.Result)
	if err := json.Unmarshal(resultJSON, &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if resp.ProvenanceChain == nil {
		t.Fatal("provenance_chain should not be nil")
	}
}

func TestCapabilitiesOperation(t *testing.T) {
	server := NewMCPServer()
	ctx := context.Background()

	req := json.RawMessage(`{}`)
	result := server.CallOperation(ctx, "get_capabilities", req)

	if result.Error != nil {
		t.Fatalf("unexpected error: %v", result.Error)
	}

	// Verify structure
	var resp CapabilitiesResponse
	resultJSON, _ := json.Marshal(result.Result)
	if err := json.Unmarshal(resultJSON, &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if resp.Capabilities == nil {
		t.Fatal("capabilities should not be nil")
	}

	if resp.SchemaVersion == "" {
		t.Fatal("schema_version should not be empty")
	}
}

func TestVersioningInResponse(t *testing.T) {
	server := NewMCPServer()
	ctx := context.Background()

	// Every operation should include version
	operations := []string{
		"validate_route",
		"test_route",
		"scaffold_domain",
		"get_lineage",
		"get_provenance",
		"get_capabilities",
	}

	for _, opName := range operations {
		result := server.CallOperation(ctx, opName, json.RawMessage(`{}`))

		if result.InterfaceVersion == "" {
			t.Fatalf("operation %s missing version", opName)
		}

		if result.InterfaceVersion != AgentInterfaceVersion {
			t.Fatalf("operation %s has wrong version: %s", opName, result.InterfaceVersion)
		}
	}
}

func TestOperationRequiredFields(t *testing.T) {
	server := NewMCPServer()
	ctx := context.Background()

	testCases := []struct {
		operation string
		request   string
		wantError bool
	}{
		{"validate_route", `{}`, true},
		{"validate_route", `{"route_config_path":""}`, true}, // Empty path
		{"test_route", `{}`, true},
		{"scaffold_domain", `{}`, true},
		{"get_lineage", `{}`, true},
		{"get_provenance", `{}`, true},
	}

	for _, tc := range testCases {
		result := server.CallOperation(ctx, tc.operation, json.RawMessage(tc.request))

		hasError := result.Error != nil
		if hasError != tc.wantError {
			t.Errorf("operation %s: expected error=%v, got error=%v", tc.operation, tc.wantError, hasError)
		}
	}
}

func TestGetCapabilitiesReturnsAdapters(t *testing.T) {
	ctx := context.Background()
	req := CapabilitiesRequest{FilterType: ""}

	resp, err := GetCapabilities(ctx, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	adapters := filterByType(resp.Capabilities, "adapter")
	if len(adapters) == 0 {
		t.Fatal("expected adapters in response")
	}

	// Verify we have both source and sink adapters
	expectedAdapters := map[string]bool{
		"http-source":  false,
		"http-sink":    false,
		"file-source":  false,
		"file-sink":    false,
		"sftp-source":  false,
		"sftp-sink":    false,
		"exec-source":  false,
		"exec-sink":    false,
	}

	for _, cap := range adapters {
		if _, exists := expectedAdapters[cap.Name]; exists {
			expectedAdapters[cap.Name] = true
		}
	}

	for name, found := range expectedAdapters {
		if !found {
			t.Errorf("expected adapter %s not found", name)
		}
	}
}

func TestGetCapabilitiesReturnsSteps(t *testing.T) {
	ctx := context.Background()
	req := CapabilitiesRequest{FilterType: ""}

	resp, err := GetCapabilities(ctx, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	steps := filterByType(resp.Capabilities, "step")
	if len(steps) == 0 {
		t.Fatal("expected steps in response")
	}

	expectedSteps := map[string]bool{
		"filter-step":      false,
		"translate-step":   false,
		"route-step":       false,
		"wiretap-step":     false,
		"idempotent-step":  false,
		"authorize-step":   false,
	}

	for _, cap := range steps {
		if _, exists := expectedSteps[cap.Name]; exists {
			expectedSteps[cap.Name] = true
		}
	}

	for name, found := range expectedSteps {
		if !found {
			t.Errorf("expected step %s not found", name)
		}
	}
}

func TestGetCapabilitiesReturnsEIPs(t *testing.T) {
	ctx := context.Background()
	req := CapabilitiesRequest{FilterType: ""}

	resp, err := GetCapabilities(ctx, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	eips := filterByType(resp.Capabilities, "eip")
	if len(eips) == 0 {
		t.Fatal("expected EIPs in response")
	}

	expectedEIPs := map[string]bool{
		"claim-check":             false,
		"content-based-router":    false,
		"dead-letter-queue":       false,
		"wiretap":                 false,
	}

	for _, cap := range eips {
		if _, exists := expectedEIPs[cap.Name]; exists {
			expectedEIPs[cap.Name] = true
		}
	}

	for name, found := range expectedEIPs {
		if !found {
			t.Errorf("expected EIP %s not found", name)
		}
	}
}

func TestGetCapabilitiesFiltering(t *testing.T) {
	ctx := context.Background()

	testCases := []struct {
		filterType string
		wantType   string
		minCount   int
	}{
		{"adapter", "adapter", 1},
		{"step", "step", 1},
		{"eip", "eip", 1},
		{"", "", 1}, // Empty means all types
	}

	for _, tc := range testCases {
		resp, err := GetCapabilities(ctx, CapabilitiesRequest{FilterType: tc.filterType})
		if err != nil {
			t.Fatalf("filter %s: unexpected error: %v", tc.filterType, err)
		}

		if len(resp.Capabilities) < tc.minCount {
			t.Errorf("filter %s: expected at least %d capabilities, got %d", tc.filterType, tc.minCount, len(resp.Capabilities))
		}

		if tc.filterType != "" {
			// Verify only matching type returned
			for _, cap := range resp.Capabilities {
				if cap.Type != tc.wantType {
					t.Errorf("filter %s: got unexpected type %s", tc.filterType, cap.Type)
				}
			}
		}
	}
}

func TestGetCapabilitiesSchemaVersion(t *testing.T) {
	ctx := context.Background()
	resp, err := GetCapabilities(ctx, CapabilitiesRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.SchemaVersion == "" {
		t.Fatal("schema_version should not be empty")
	}

	if resp.SchemaVersion != "1.0.0" {
		t.Fatalf("expected schema_version 1.0.0, got %s", resp.SchemaVersion)
	}
}

func TestGetCapabilitiesConfigSchema(t *testing.T) {
	ctx := context.Background()
	resp, err := GetCapabilities(ctx, CapabilitiesRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(resp.Capabilities) == 0 {
		t.Fatal("expected capabilities")
	}

	for _, cap := range resp.Capabilities {
		if cap.ConfigSchema == nil {
			t.Errorf("capability %s has nil config_schema", cap.Name)
		}

		if cap.Description == "" {
			t.Errorf("capability %s has empty description", cap.Name)
		}
	}
}

// Helper function to filter capabilities by type
func filterByType(caps []Capability, capType string) []Capability {
	var result []Capability
	for _, cap := range caps {
		if cap.Type == capType {
			result = append(result, cap)
		}
	}
	return result
}
