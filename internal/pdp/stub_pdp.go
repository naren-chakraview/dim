package pdp

import (
	"encoding/json"
	"net/http"

	"github.com/naren-chakraview/dim/internal/steps"
)

// StubPDP is a trivial non-OPA implementation of the neutral PDP contract (M1.2.4)
// Used to prove the contract is engine-agnostic (same tests, different PDP backend)
type StubPDP struct {
	allowedSubjects map[string]bool
}

// NewStubPDP creates a new stub PDP with a set of allowed subjects
func NewStubPDP(allowedSubjects []string) *StubPDP {
	allowed := make(map[string]bool)
	for _, subj := range allowedSubjects {
		allowed[subj] = true
	}
	return &StubPDP{
		allowedSubjects: allowed,
	}
}

// ServeHTTP implements http.Handler for the stub PDP
// Accepts DecisionRequests on any path and returns DecisionResponses
func (sp *StubPDP) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	// Parse decision request
	var req steps.PDPDecisionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "malformed request"})
		return
	}

	// Make decision based on allowed subjects
	resp := &steps.PDPDecisionResponse{
		Decision: "deny",
		Reason:   "subject not authorized",
		Metadata: make(map[string]interface{}),
	}

	if allowed, ok := sp.allowedSubjects[req.Principal.Subject]; ok && allowed {
		resp.Decision = "allow"
		resp.Reason = "subject is authorized"
	}

	// Add stub metadata
	resp.Metadata["stub_pdp"] = true
	resp.Metadata["backend"] = "stub"

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}
