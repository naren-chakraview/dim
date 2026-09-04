# M1.2: PBAC + OPA Reference Implementation — Complete Guide

**Status:** ✅ COMPLETE (M1.2.1-M1.2.4)  
**Date:** 2026-09-04  
**Scope:** Full PBAC implementation with OPA adapter and BYO-PDP proof

## What Was Built

### M1.2.1: PBAC Mode Schema ✅
- Extended `authorize` step to support `mode: pbac`
- Added `PDPConfig` (endpoint, timeout) and `OnDenySpec` (denial routing)
- Implemented `executePBAC()` method
- Wired PDP configuration from YAML through factory
- Added configuration validation tests

**Files:**
- `internal/config/schema.go` — PDPConfig, OnDenySpec, updated AuthorizeSpec
- `internal/steps/authorize.go` — PBAC execution logic with HTTP PDP calls
- `internal/steps/factory.go` — Config wiring

### M1.2.2: Mock PDP + Fixture Tests ✅
- Mock HTTP server implementing M1.1 contract exactly
- 6 fixture tests covering allow, deny, timeout, error handling, malformed responses, obligations
- Proves PBAC evaluator works against any HTTP-based PDP

**Files:**
- `internal/steps/authorize_test.go` — Mock PDP fixtures (TestPBACWithMockPDP_*)

**Test Coverage:**
- ✅ Allow flow
- ✅ Deny flow
- ✅ Timeout handling (100ms timeout, slow PDP)
- ✅ PDP connection failure
- ✅ Malformed PDP response
- ✅ Obligations processing

### M1.2.3: OPA Reference Adapter ✅
- Translation layer: neutral contract ↔ OPA input/output
- Sends decisions to OPA's `/v1/data/dim/authorize` endpoint
- Extracts obligations from OPA result
- Example Rego policies (order-processing.rego)
- Docker Compose for local OPA testing

**Files:**
- `internal/pdp/opa_adapter.go` — OPA integration logic
- `internal/pdp/opa_adapter_test.go` — 5 adapter tests
- `examples/opa/order-processing.rego` — Reference policies
- `deploy/docker-compose.opa.yml` — Local OPA setup

**OPA Policies Include:**
- Role-based allow rules (seller, admin, viewer, compliance)
- Obligations for PII redaction (non-admins)
- Obligations for rate limiting (non-premium sellers)
- Obligations for audit logging (admin actions)
- Obligations for data classification tagging (PCI)

### M1.2.4: BYO-PDP Conformance Proof ✅
- Stub PDP: trivial non-OPA implementation (allows/denies based on subject)
- Same fixture test suite (M1.2.2) passes against stub unchanged
- Integration tests proving contract neutrality

**Files:**
- `internal/pdp/stub_pdp.go` — Stub PDP implementation
- `internal/pdp/stub_pdp_test.go` — Conformance tests
- `internal/pdp/pbac_integration_test.go` — Contract consistency tests

## How It Works

### Route Configuration (YAML)

```yaml
routes:
  order-processing:
    from: webhook-in
    steps:
      - authorize:
          mode: pbac
          pdp:
            endpoint: http://opa:8181/v1/data/dim/authorize
            timeout: 2000
          on_deny:
            target: unauthorized-dlq
```

### Decision Flow

1. **Message arrives** at authorize step with authenticated principal
2. **Decision request** is built (principal, action, resource, context)
3. **HTTP POST** to PDP endpoint with neutral DecisionRequest
4. **PDP evaluates** (OPA or custom implementation)
5. **Decision response** returned with decision + obligations + reason
6. **Route handling:**
   - `allow` → message passes through to next step
   - `deny` → message routed to on_deny sink (e.g., unauthorized-dlq)

### Example Decision Flows

**Seller processing order:**
```
Request: {principal.subject: "alice", roles: ["seller"], action: "process_order"}
        ↓ OPA policy
Allow: true (seller role matches)
Obligations: [redact_fields: {fields: ["ssn"]}]
        ↓
Message passes through with redaction obligation
```

**Viewer attempting admin action:**
```
Request: {principal.subject: "bob", roles: ["viewer"], action: "modify_route"}
        ↓ OPA policy
Allow: false (no matching rule)
Reason: "insufficient roles for this action"
        ↓
Message routed to unauthorized-dlq
```

## Testing Strategy

### Unit Tests (Mock PDP)
- Configuration validation
- Request/response marshaling
- Timeout handling
- Error scenarios

**Test count:** 6 mock PDP + 3 PBAC config tests

### Integration Tests (OPA Adapter)
- Real OPA request/response translation
- Obligation extraction
- Graceful degradation (missing result)

**Test count:** 5 OPA adapter tests

### Conformance Tests (Stub PDP)
- Stub PDP implements neutral contract
- Same fixtures pass against stub (proving neutrality)
- Response schema validation

**Test count:** 4 stub PDP + 2 integration tests

**Total:** 20+ tests across all components

## Local Development

### Running with Local OPA

```bash
# Start OPA with example policies
docker-compose -f deploy/docker-compose.opa.yml up

# Build dim
go build ./cmd/dimctl -o dimctl

# Create a route with PBAC
cat > order-processing.yaml << EOF
routes:
  orders:
    from: webhook
    steps:
      - authorize:
          mode: pbac
          pdp:
            endpoint: http://localhost:8181/v1/data/dim/authorize
            timeout: 2000
          on_deny:
            target: denied
EOF

# Run the route
./dimctl run order-processing.yaml

# Send a request
curl -X POST http://localhost:8080/orders \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <JWT with seller role>" \
  -d '{"order_id": "12345"}'
```

### Testing with Mock PDP

All fixture tests run without external dependencies using Go's `httptest` package.

```bash
go test ./internal/steps -run TestPBACWithMockPDP
go test ./internal/pdp -run TestOPAAdapter
go test ./internal/pdp -run TestStubPDP
```

## Design Principles

### 1. Engine-Agnostic Contract First
- M1.1 contract published before OPA adapter
- Contract is language-agnostic (JSON/HTTP)
- OPA is reference implementation, not the only option

### 2. Neutral Translation Layer
- OPA adapter doesn't leak Rego-specific concepts into routes
- Routes reference PDP by endpoint, not by policy language
- Same route works against OPA, stub, or custom PDP

### 3. Obligations Are First-Class
- PDP can impose requirements beyond allow/deny
- Obligations (redact_fields, rate_limit, audit_log, data_classification_tag) are extracted and processed
- Future: dim enforces obligations in M1.x

### 4. Clear Error Handling
- PDP connection failures → clear error messages
- Timeouts → explicit "deadline exceeded" error
- Malformed responses → validation errors
- Authorization denials → permanent errors (don't retry)

## Security Considerations

### PDP Endpoint Trust
- Configured in route YAML (trusted config)
- HTTPS recommended for production (future enhancement: TLS config in PDPConfig)
- No credentials in DecisionRequest (only data)

### Response Validation
- Response schema validated against M1.1 DecisionResponse
- Decision must be "allow" or "deny" (case-sensitive)
- Malformed responses rejected (not allow-by-default)

### Obligation Enforcement
- Currently: extracted and logged
- Future (M1.x): actively enforced by dim
  - `redact_fields` → remove fields from message body
  - `rate_limit` → throttle by principal/resource
  - `audit_log` → write audit record with tags
  - `data_classification_tag` → tag message with sensitivity

## Files Changed/Added

**Modified:**
- `internal/config/schema.go` — Added PDPConfig, OnDenySpec
- `internal/steps/authorize.go` — Added PBAC mode, HTTP PDP client
- `internal/steps/factory.go` — Wire PDP config from YAML
- `internal/steps/authorize_test.go` — Updated all tests, added M1.2.2 fixtures

**Created:**
- `internal/pdp/opa_adapter.go` — OPA integration
- `internal/pdp/opa_adapter_test.go` — OPA adapter tests
- `internal/pdp/stub_pdp.go` — Stub PDP for conformance
- `internal/pdp/stub_pdp_test.go` — Stub tests
- `internal/pdp/pbac_integration_test.go` — Integration tests
- `examples/opa/order-processing.rego` — Reference policies
- `deploy/docker-compose.opa.yml` — Local OPA setup

## Exit Criteria Met

### M1.2.1 ✅
- Routes with `mode: pbac` validate against schema
- PDP endpoint and timeout configurable
- PBAC step instantiation succeeds with valid config

### M1.2.2 ✅
- Mock PDP accepts neutral DecisionRequest
- Fixture tests verify allow/deny decisions
- Timeout and error handling tested
- Obligations extracted and processed

### M1.2.3 ✅
- Real OPA instance evaluates sample policy
- Decision response follows M1.1 contract
- Translation layer proves contract neutrality
- Docker Compose for local development

### M1.2.4 ✅
- Stub PDP implements neutral contract
- Same fixture suite passes against stub unchanged
- Proves contract is truly engine-agnostic

## Next Steps

**Immediate (M1.x implementations):**
- M1.3: OBO (on-behalf-of) token exchange
- M1.4: Kafka reliability
- M1.5: AMQP reliability

**Future (enforcement):**
- Obligation enforcement in routes (redact_fields, rate_limit, audit_log, data_classification_tag)
- PBAC-driven lineage enrichment (policy decision recorded in lineage)
- PBAC audit trail export to compliance systems

## References

- **M1.1:** `design/PDP_CONTRACT_SPEC.md` — Neutral contract
- **OPA:** https://www.openpolicyagent.org/
- **Rego:** https://www.openpolicyagent.org/docs/latest/policy-language/
