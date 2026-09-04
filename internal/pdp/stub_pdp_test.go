package pdp

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/naren-chakraview/dim/internal/steps"
)

// TestStubPDPConformance_Allow verifies stub PDP returns allow decisions (M1.2.4)
// This proves the neutral contract works with non-OPA backends
func TestStubPDPConformance_Allow(t *testing.T) {
	stub := NewStubPDP([]string{"alice", "bob"})
	server := httptest.NewServer(stub)
	defer server.Close()

	// Create a decision request
	req := &steps.PDPDecisionRequest{
		Principal: steps.PDPPrincipal{
			Subject: "alice",
			Roles:   []string{"seller"},
		},
		Action: "process_order",
		Resource: steps.PDPResource{
			Type: "message",
		},
	}

	// Marshal request
	body, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("Failed to marshal request: %v", err)
	}

	// Send request to stub PDP
	resp, err := http.Post(server.URL, "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}
	defer resp.Body.Close()

	// Parse response
	var decisionResp steps.PDPDecisionResponse
	if err := json.NewDecoder(resp.Body).Decode(&decisionResp); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if decisionResp.Decision != "allow" {
		t.Errorf("Expected allow, got %q", decisionResp.Decision)
	}
}

// TestStubPDPConformance_Deny verifies stub PDP returns deny decisions (M1.2.4)
// This is the same test pattern as M1.2.2 mock PDP tests, proving neutrality
func TestStubPDPConformance_Deny(t *testing.T) {
	stub := NewStubPDP([]string{"alice"}) // Only alice is allowed
	server := httptest.NewServer(stub)
	defer server.Close()

	req := &steps.PDPDecisionRequest{
		Principal: steps.PDPPrincipal{
			Subject: "bob", // bob is not in allowed list
			Roles:   []string{"viewer"},
		},
		Action: "process_order",
	}

	body, _ := json.Marshal(req)
	resp, _ := http.Post(server.URL, "application/json", bytes.NewReader(body))
	defer resp.Body.Close()

	var decisionResp steps.PDPDecisionResponse
	json.NewDecoder(resp.Body).Decode(&decisionResp)

	if decisionResp.Decision != "deny" {
		t.Errorf("Expected deny for unauthorized subject, got %q", decisionResp.Decision)
	}
}

// TestStubPDPConformance_ResponseSchema verifies response matches M1.1 contract (M1.2.4)
func TestStubPDPConformance_ResponseSchema(t *testing.T) {
	stub := NewStubPDP([]string{"alice"})
	server := httptest.NewServer(stub)
	defer server.Close()

	req := &steps.PDPDecisionRequest{
		Principal: steps.PDPPrincipal{Subject: "alice"},
		Action:    "test",
	}

	body, _ := json.Marshal(req)
	resp, _ := http.Post(server.URL, "application/json", bytes.NewReader(body))
	defer resp.Body.Close()

	var decisionResp steps.PDPDecisionResponse
	json.NewDecoder(resp.Body).Decode(&decisionResp)

	// Verify response has required fields per M1.1 contract
	if decisionResp.Decision == "" {
		t.Error("Response missing decision field")
	}
	if decisionResp.Decision != "allow" && decisionResp.Decision != "deny" {
		t.Errorf("Decision must be 'allow' or 'deny', got %q", decisionResp.Decision)
	}
	if decisionResp.Reason == "" {
		t.Error("Response missing reason field")
	}
	if decisionResp.Metadata == nil {
		t.Error("Response missing metadata field")
	}
}

// TestStubPDPConformance_MalformedRequest verifies error handling (M1.2.4)
func TestStubPDPConformance_MalformedRequest(t *testing.T) {
	stub := NewStubPDP([]string{"alice"})
	server := httptest.NewServer(stub)
	defer server.Close()

	// Send malformed JSON
	resp, err := http.Post(server.URL, "application/json", bytes.NewReader([]byte("not json")))
	if err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected 400 for malformed request, got %d", resp.StatusCode)
	}
}
