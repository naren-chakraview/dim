# M1.2: PBAC (Policy-Based Access Control) + OPA Reference Implementation

**Phase:** Phase 1  
**Milestone:** M1.2 (PBAC + OPA reference adapter)  
**Status:** M1.2.1 Complete (PBAC schema); M1.2.2–M1.2.4 Design & Implementation Plan  
**Date:** 2026-09-04

## Overview

M1.2 implements Policy-Based Access Control (PBAC) by extending the `authorize` step to support `mode: pbac`, backed by an external Policy Decision Point (PDP) following the neutral contract defined in M1.1 (design/PDP_CONTRACT_SPEC.md). OPA is the reference implementation, but the design intentionally decouples the PDP contract from OPA-specific details.

### Design Philosophy

**Engine-agnostic contract first:** The contract (M1.1) is published independent of OPA. The OPA adapter (M1.2.3) then implements that contract. This prevents OPA-specific details from leaking into routes and enables bring-your-own-PDP.

**Neutral flow:**
```
Route YAML (mode: pbac)
  ↓
dim (authorize step)
  ↓
Neutral PDP contract (M1.1)
  ↓
PDP implementation (OPA in this case)
  ↓
Decision: allow/deny + obligations
```

## Subtasks

### M1.2.1: Add `mode: pbac` to config schema ✅ COMPLETE

**Status:** DONE

**What was delivered:**
- Extended `AuthorizeSpec` to accept `mode: pbac`
- Added `PDPConfig` struct for PDP endpoint and timeout
- Added `OnDenySpec` for routing denied messages
- Updated `AuthorizeStep` constructor and factory
- Added PBAC configuration validation tests

**Exit criteria met:**
✅ Routes with `mode: pbac` validate against schema  
✅ PDP endpoint and timeout are configurable  
✅ PBAC step instantiation succeeds with valid config  

### M1.2.2: Implement the PBAC evaluator with mock PDP

**Objective:** PBAC decision flow works with a mock PDP; fixture tests confirm allow/deny behavior.

**Implementation:**

The PBAC evaluator (`executePBAC()` in authorize.go) is already stubbed in M1.2.1. M1.2.2 fills in the fixture test with a mock PDP.

```go
// In authorize_test.go (NEW)

// Mock HTTP server that implements the neutral PDP contract
type MockPDP struct {
    server *httptest.Server
    decisions map[string]bool // key: subject, value: allowed
}

// NewMockPDP creates a mock PDP server
func NewMockPDP(decisions map[string]bool) *MockPDP {
    decisions := make(map[string]bool)
    mux := http.NewServeMux()
    
    mux.HandleFunc("/v1/data/dim/authorize", func(w http.ResponseWriter, r *http.Request) {
        // Parse M1.1 DecisionRequest
        // Lookup subject in decisions map
        // Return DecisionResponse matching M1.1 contract
    })
    
    return &MockPDP{
        server: httptest.NewServer(mux),
        decisions: decisions,
    }
}

// Fixture tests
func TestPBACWithMockPDP_Allow(t *testing.T) {
    // Create mock PDP that allows certain subjects
    pdp := NewMockPDP(map[string]bool{
        "alice": true,
        "bob": false,
    })
    defer pdp.Close()
    
    // Create authorize step with mock PDP endpoint
    step, err := NewAuthorizeStep("pbac", nil, "", pdp.URL(), 5000)
    
    // Test: alice is allowed
    msg := engine.NewMessage(..., "test-route", "v1")
    msg.Metadata.Principal = &engine.Principal{Subject: "alice"}
    
    result, err := step.Execute(ctx, msg)
    assert(err == nil, "alice should be allowed")
    assert(result != nil, "message should pass through")
}

func TestPBACWithMockPDP_Deny(t *testing.T) {
    // Test: bob is denied
    msg.Metadata.Principal = &engine.Principal{Subject: "bob"}
    
    result, err := step.Execute(ctx, msg)
    assert(err != nil, "bob should be denied")
    assert(result == nil, "message should be rejected")
    assert(strings.Contains(err.Error(), "authorization_denied"), "error should indicate denial")
}
```

**Mock PDP behavior:**
- Accepts POST requests to `/v1/data/dim/authorize`
- Reads `principal.subject` from DecisionRequest
- Returns `decision: allow` or `deny` per mock configuration
- Returns `reason` string for audit trail

**Test coverage:**
- ✅ PBAC allow flow with mock PDP
- ✅ PBAC deny flow with mock PDP
- ✅ PDP timeout handling (short timeout, PDP takes too long)
- ✅ PDP connection error handling
- ✅ PDP malformed response handling

**Exit criteria:**
✅ Fixture test with mock PDP allows/denies per response  
✅ Error handling for PDP connection failures  
✅ Request/response marshaling correct per M1.1 contract  

### M1.2.3: OPA reference adapter + translation layer

**Objective:** Routes evaluate policies via embedded or standalone OPA, proving the neutral contract works with a real PDP implementation.

**Architecture:**

```
dim authorize step (PBAC mode)
  ↓ (neutral DecisionRequest per M1.1)
OPA HTTP interface (port 8181 by default)
  ↓ (Rego policies)
OPA evaluation
  ↓ (neutral DecisionResponse per M1.1)
dim receives decision
```

**Implementation:**

1. **OPA translation layer** (`internal/pdp/opa_adapter.go`):
   ```go
   type OPAAdapter struct {
       endpoint string // e.g., http://localhost:8181
       policies string // Rego policies to load
   }
   
   // TranslateRequest converts neutral DecisionRequest to OPA input
   func (oa *OPAAdapter) TranslateRequest(req *steps.PDPDecisionRequest) (OPAInput, error) {
       // Map neutral principal/action/resource to Rego data structures
   }
   
   // TranslateResponse converts OPA output to neutral DecisionResponse
   func (oa *OPAAdapter) TranslateResponse(opaResult map[string]interface{}) (*steps.PDPDecisionResponse, error) {
       // Extract decision, obligations, reason from OPA output
   }
   ```

2. **Rego policy example** (`examples/opa/order-processing.rego`):
   ```rego
   package dim.authorize
   
   # Default: deny
   default allow = false
   
   # Rule: sellers can process orders
   allow = true {
       input.principal.roles[_] = "seller"
       input.action = "process_order"
   }
   
   # Rule: admins can do anything
   allow = true {
       input.principal.roles[_] = "admin"
   }
   
   # Reason for deny
   reason = "seller role required to process orders" {
       not allow
   }
   ```

3. **Docker Compose for local OPA** (`deploy/docker-compose.opa.yml`):
   ```yaml
   version: '3.8'
   services:
     opa:
       image: openpolicyagent/opa:latest
       ports:
         - "8181:8181"
       command: run --server
   ```

**Test coverage:**
- ✅ Embedded OPA instance responds to DecisionRequest
- ✅ Rego policy evaluated correctly (seller role allows, viewer role denies)
- ✅ Response translated to neutral contract correctly
- ✅ Decision applied to route (allow passes message, deny routes to on_deny sink)

**Exit criteria:**
✅ Real embedded OPA instance evaluates sample policy  
✅ Decision response follows M1.1 contract exactly  
✅ Translation layer proves contract neutrality (same contract, OPA backend)

### M1.2.4: BYO-PDP conformance test

**Objective:** Prove the contract is truly neutral by implementing a non-OPA PDP stub that passes the same fixtures.

**Implementation:**

A trivial non-OPA PDP (`internal/pdp/stub_adapter.go`) that implements the neutral contract without OPA:

```go
type StubPDP struct {
    allowedSubjects []string
}

// HTTP handler that returns decisions per M1.1 contract
func (sp *StubPDP) HandleAuthorize(w http.ResponseWriter, r *http.Request) {
    // Read DecisionRequest from body
    // Check if principal.subject is in allowedSubjects
    // Return DecisionResponse (allow or deny)
}

// Fixture test: same test suite, different PDP backend
func TestPBACWithStubPDP_AllowFlow(t *testing.T) {
    // This is the SAME TEST as M1.2.2's mock PDP test
    // (same setup, same assertions)
    // but running against the stub PDP instead of mock
}
```

**Why this proves neutrality:**
- M1.2.2 (with mock): tests dim's PBAC evaluator
- M1.2.3 (with OPA): tests OPA as reference implementation
- M1.2.4 (with stub): same test suite, different PDP → proves contract is actually neutral

**Exit criteria:**
✅ Stub PDP accepts neutral DecisionRequest  
✅ Stub PDP returns neutral DecisionResponse  
✅ Same fixture test suite (M1.2.2) passes against stub unchanged  

## Configuration Example

**Route YAML with PBAC mode:**

```yaml
version: 1

routes:
  order-processing:
    from: webhook-in
    steps:
      - authorize:
          mode: pbac
          pdp:
            endpoint: http://opa:8181/v1/data/dim/authorize
            timeout: 2000  # 2 second timeout
          on_deny:
            target: unauthorized-dlq  # DLQ for denied messages
```

**Rego policy example:**

```rego
package dim.authorize

# Allow sellers to process orders
allow = true {
    input.principal.roles[_] = "seller"
    input.action = "process_order"
    input.resource.classification != "pii"  # No PII for sellers
}

# Allow admins everything
allow = true {
    input.principal.roles[_] = "admin"
}

# Reason for logging/audit
reason = sprintf("denied: %s cannot %s", [
    input.principal.subject,
    input.action
]) { not allow }

# Obligations: redact PII if not admin
obligations[obligation] {
    allow
    input.principal.roles[_] != "admin"
    input.resource.classification = "pii"
    obligation := {
        "type": "redact_fields",
        "parameters": {
            "fields": ["ssn", "credit_card", "phone"]
        }
    }
}
```

## Design Decisions

### Why HTTP for PDP calls?

- **Simplicity**: PDP can be deployed separately (no SDK requirement)
- **Language-agnostic**: any language can implement the contract
- **Observability**: HTTP calls are easily traced/monitored
- **Scaling**: PDP can scale independently (load balancing, caching, etc.)

**Tradeoff:** Slight latency vs. in-process evaluation. Mitigated by configurable timeout and connection pooling.

### Why not stub PBAC in Phase 0?

- M1.1's contract needed to be designed and published first
- OPA integration is non-trivial; building it mid-Phase-0 would have delayed other features
- Phase 0 users don't need PBAC; RBAC and ABAC cover the common cases

### Why OPA as reference?

- Well-established OSS policy language (Rego)
- Can be embedded or deployed standalone
- Used widely in Kubernetes and cloud-native ecosystems
- Mature HTTP API matching our contract
- No vendor lock-in (contract is neutral, OPA is just one impl)

## Testing Strategy

**Unit tests (M1.2.2-2.4):**
- PBAC configuration validation
- Mock PDP request/response handling
- Timeout and error handling
- Decision flow (allow/deny)

**Integration tests:**
- Real OPA instance (local Docker container, M1.2.3)
- Rego policy evaluation
- End-to-end route execution with PBAC

**Conformance tests:**
- Stub PDP (non-OPA) uses same fixtures → proves neutrality (M1.2.4)

## Security Considerations

### PDP Endpoint Trust

- PDP endpoint configured in route YAML (trusted configuration)
- HTTPS recommended for production (out of scope for Phase 1, but TLS config in PDPConfig extensibility)
- No secrets in DecisionRequest (subject/action/resource are data, not credentials)

### PDP Response Validation

- Response schema validated against M1.1 DecisionResponse
- Malformed responses rejected (connection error, not allow-by-default)
- Timeout enforced (prevents slowloris-style attacks)

### Obligations Enforcement

- Obligations returned by PDP must be applied by dim
- Future M1.2.x: redact_fields obligation implementation
- Unrecognized obligation types logged as warning, not silently ignored

## Success Criteria

**M1.2 exit:**
✅ M1.2.1: Routes with `mode: pbac` validate (DONE)  
✅ M1.2.2: PBAC evaluator works with mock PDP  
✅ M1.2.3: OPA reference adapter evaluates real Rego policies  
✅ M1.2.4: Non-OPA stub PDP passes same tests (proves neutrality)  

## References

- **M1.1:** `design/PDP_CONTRACT_SPEC.md` — Neutral PDP contract
- **OPA:** https://www.openpolicyagent.org/
- **Rego language:** https://www.openpolicyagent.org/docs/latest/policy-language/
- **Design §13.3:** PBAC vs. RBAC/ABAC rationale
