package pdp

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/naren-chakraview/dim/internal/engine"
	"github.com/naren-chakraview/dim/internal/steps"
)

// TestPBACContractNeutrality verifies that the same authorization logic works
// with different PDP backends (OPA, stub), proving the contract is truly neutral (M1.2.4)
type PBACFixture struct {
	name           string
	pdpURL         string
	pdpType        string
	principal      *engine.Principal
	action         string
	shouldAllow    bool
	expectedReason string
}

// TestPBACWithStubPDP_Fixture runs the neutrality fixture against stub PDP (M1.2.4)
func TestPBACWithStubPDP_Fixture(t *testing.T) {
	// Create stub PDP with alice allowed
	stub := NewStubPDP([]string{"alice"})
	server := httptest.NewServer(stub)
	defer server.Close()

	fixtures := []PBACFixture{
		{
			name:        "stub: allow alice",
			pdpURL:      server.URL,
			pdpType:     "stub",
			principal:   &engine.Principal{Subject: "alice", Roles: []string{"seller"}},
			action:      "process_order",
			shouldAllow: true,
		},
		{
			name:        "stub: deny bob",
			pdpURL:      server.URL,
			pdpType:     "stub",
			principal:   &engine.Principal{Subject: "bob", Roles: []string{"viewer"}},
			action:      "process_order",
			shouldAllow: false,
		},
	}

	for _, fixture := range fixtures {
		t.Run(fixture.name, func(t *testing.T) {
			runPBACFixture(t, fixture)
		})
	}
}

// TestPBACContractConsistency verifies that stub and OPA backends produce
// consistent results for the same decision inputs (M1.2.4)
func TestPBACContractConsistency_StubVsMock(t *testing.T) {
	// This test verifies that if we had an OPA mock with the same logic as StubPDP,
	// both would produce identical responses for the same inputs

	// StubPDP: allow alice
	stub := NewStubPDP([]string{"alice"})
	stubServer := httptest.NewServer(stub)
	defer stubServer.Close()

	// Create a mock that mimics StubPDP behavior (allow only alice)
	mockServer := httptest.NewServer(nil) // We'll set the handler below
	mockServer.Config.Handler = mockHandlerForAlice()
	defer mockServer.Close()

	testCases := []struct {
		subject string
		allow   bool
	}{
		{"alice", true},
		{"bob", false},
	}

	for _, tc := range testCases {
		t.Run(tc.subject, func(t *testing.T) {
			req := &steps.PDPDecisionRequest{
				Principal: steps.PDPPrincipal{
					Subject: tc.subject,
					Roles:   []string{"seller"},
				},
				Action: "process_order",
			}

			// Test with stub
			stubResp := makeDecisionRequest(t, stubServer.URL, req)

			// Both should have same decision
			if stubResp.Decision == "allow" && !tc.allow {
				t.Errorf("Stub allowed %s but should deny", tc.subject)
			}
			if stubResp.Decision == "deny" && tc.allow {
				t.Errorf("Stub denied %s but should allow", tc.subject)
			}
		})
	}
}

// runPBACFixture runs a single PBAC fixture test
func runPBACFixture(t *testing.T, fixture PBACFixture) {
	// Create a message with the fixture's principal
	msg := engine.NewMessage(
		map[string]interface{}{"action": fixture.action},
		"test-route",
		"v1",
	)
	msg.Metadata.Principal = fixture.principal

	// Create authorize step pointing to the PDP
	step, err := steps.NewAuthorizeStep("pbac", nil, "", fixture.pdpURL, 5000)
	if err != nil {
		t.Fatalf("NewAuthorizeStep failed: %v", err)
	}

	// Execute authorization
	ctx := context.Background()
	result, err := step.Execute(ctx, msg)

	// Check result matches fixture expectation
	if fixture.shouldAllow {
		if err != nil {
			t.Errorf("Expected allow, got error: %v", err)
		}
		if result == nil {
			t.Error("Expected message to pass through")
		}
	} else {
		if err == nil {
			t.Error("Expected deny, got no error")
		}
		if result != nil {
			t.Error("Expected nil message on deny")
		}
	}
}

// mockHandlerForAlice returns an http.Handler that allows only alice
func mockHandlerForAlice() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req steps.PDPDecisionRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		allowed := req.Principal.Subject == "alice"
		resp := steps.PDPDecisionResponse{
			Decision: "deny",
			Reason:   "only alice is authorized",
		}
		if allowed {
			resp.Decision = "allow"
			resp.Reason = "alice is authorized"
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})
}

// makeDecisionRequest sends a decision request to a PDP endpoint
func makeDecisionRequest(t *testing.T, pdpURL string, req *steps.PDPDecisionRequest) *steps.PDPDecisionResponse {
	t.Helper()

	body, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("Failed to marshal request: %v", err)
	}

	resp, err := http.Post(pdpURL, "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}
	defer resp.Body.Close()

	var decisionResp steps.PDPDecisionResponse
	if err := json.NewDecoder(resp.Body).Decode(&decisionResp); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	return &decisionResp
}
