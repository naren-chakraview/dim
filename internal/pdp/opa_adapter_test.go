package pdp

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/naren-chakraview/dim/internal/steps"
)

// TestOPAAdapterAllow verifies OPA adapter translates allow decision correctly (M1.2.3)
func TestOPAAdapterAllow(t *testing.T) {
	// Mock OPA server
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/data/dim/authorize", func(w http.ResponseWriter, r *http.Request) {
		// OPA returns result.allow = true
		resp := map[string]interface{}{
			"result": map[string]interface{}{
				"allow":  true,
				"reason": "seller role matched",
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	adapter, err := NewOPAAdapter(server.URL+"/v1/data/dim/authorize", 5000)
	if err != nil {
		t.Fatalf("NewOPAAdapter failed: %v", err)
	}

	ctx := context.Background()
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

	resp, err := adapter.Evaluate(ctx, req)
	if err != nil {
		t.Fatalf("Evaluate failed: %v", err)
	}

	if resp.Decision != "allow" {
		t.Errorf("Expected allow, got %q", resp.Decision)
	}
	if resp.Reason != "seller role matched" {
		t.Errorf("Expected reason 'seller role matched', got %q", resp.Reason)
	}
}

// TestOPAAdapterDeny verifies OPA adapter translates deny decision correctly (M1.2.3)
func TestOPAAdapterDeny(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/data/dim/authorize", func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"result": map[string]interface{}{
				"allow":  false,
				"reason": "viewer role insufficient",
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	adapter, err := NewOPAAdapter(server.URL+"/v1/data/dim/authorize", 5000)
	if err != nil {
		t.Fatalf("NewOPAAdapter failed: %v", err)
	}

	ctx := context.Background()
	req := &steps.PDPDecisionRequest{
		Principal: steps.PDPPrincipal{
			Subject: "bob",
			Roles:   []string{"viewer"},
		},
		Action: "process_order",
	}

	resp, err := adapter.Evaluate(ctx, req)
	if err != nil {
		t.Fatalf("Evaluate failed: %v", err)
	}

	if resp.Decision != "deny" {
		t.Errorf("Expected deny, got %q", resp.Decision)
	}
}

// TestOPAAdapterObligations verifies OPA adapter extracts obligations (M1.2.3)
func TestOPAAdapterObligations(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/data/dim/authorize", func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"result": map[string]interface{}{
				"allow":  true,
				"reason": "allowed with redaction",
				"obligations": []map[string]interface{}{
					{
						"type": "redact_fields",
						"parameters": map[string]interface{}{
							"fields": []string{"ssn", "credit_card"},
						},
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	adapter, err := NewOPAAdapter(server.URL+"/v1/data/dim/authorize", 5000)
	if err != nil {
		t.Fatalf("NewOPAAdapter failed: %v", err)
	}

	ctx := context.Background()
	req := &steps.PDPDecisionRequest{
		Principal: steps.PDPPrincipal{Subject: "alice"},
		Action:    "read_pii",
	}

	resp, err := adapter.Evaluate(ctx, req)
	if err != nil {
		t.Fatalf("Evaluate failed: %v", err)
	}

	if len(resp.Obligations) != 1 {
		t.Errorf("Expected 1 obligation, got %d", len(resp.Obligations))
	}
	if len(resp.Obligations) > 0 && resp.Obligations[0].Type != "redact_fields" {
		t.Errorf("Expected redact_fields obligation, got %q", resp.Obligations[0].Type)
	}
}

// TestOPAAdapterRequestTranslation verifies neutral request is correctly translated to OPA format (M1.2.3)
func TestOPAAdapterRequestTranslation(t *testing.T) {
	receivedInput := map[string]interface{}{}

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/data/dim/authorize", func(w http.ResponseWriter, r *http.Request) {
		var req map[string]interface{}
		json.NewDecoder(r.Body).Decode(&req)

		// Extract input for verification
		if input, ok := req["input"].(map[string]interface{}); ok {
			receivedInput = input
		}

		resp := map[string]interface{}{
			"result": map[string]interface{}{"allow": true},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	adapter, err := NewOPAAdapter(server.URL+"/v1/data/dim/authorize", 5000)
	if err != nil {
		t.Fatalf("NewOPAAdapter failed: %v", err)
	}

	ctx := context.Background()
	req := &steps.PDPDecisionRequest{
		Principal: steps.PDPPrincipal{
			Subject: "alice",
			Roles:   []string{"seller", "admin"},
		},
		Action: "process_order",
		Resource: steps.PDPResource{
			Type:  "message",
			Route: "order-processing",
		},
		Context: steps.PDPContext{
			Timestamp:     "2026-09-04T12:34:56Z",
			CorrelationID: "corr-12345",
		},
	}

	_, err = adapter.Evaluate(ctx, req)
	if err != nil {
		t.Fatalf("Evaluate failed: %v", err)
	}

	// Verify input structure
	if principal, ok := receivedInput["principal"].(map[string]interface{}); ok {
		if subject, ok := principal["subject"].(string); !ok || subject != "alice" {
			t.Errorf("Principal.subject not translated correctly")
		}
	}

	if action, ok := receivedInput["action"].(string); !ok || action != "process_order" {
		t.Errorf("Action not translated correctly")
	}

	if resource, ok := receivedInput["resource"].(map[string]interface{}); ok {
		if route, ok := resource["route"].(string); !ok || route != "order-processing" {
			t.Errorf("Resource.route not translated correctly")
		}
	}
}

// TestOPAAdapterMissingResult verifies graceful handling of missing OPA result (M1.2.3)
func TestOPAAdapterMissingResult(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/data/dim/authorize", func(w http.ResponseWriter, r *http.Request) {
		// OPA returns empty response (no result)
		resp := map[string]interface{}{}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	adapter, err := NewOPAAdapter(server.URL+"/v1/data/dim/authorize", 5000)
	if err != nil {
		t.Fatalf("NewOPAAdapter failed: %v", err)
	}

	ctx := context.Background()
	req := &steps.PDPDecisionRequest{
		Principal: steps.PDPPrincipal{Subject: "alice"},
		Action:    "test",
	}

	resp, err := adapter.Evaluate(ctx, req)
	if err != nil {
		t.Fatalf("Evaluate should handle missing result gracefully, got: %v", err)
	}

	// Should default to deny when result is missing
	if resp.Decision != "deny" {
		t.Errorf("Expected default deny on missing result, got %q", resp.Decision)
	}
}
