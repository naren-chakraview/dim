# M1.3: OBO (On-Behalf-Of) Token Exchange — Design & Implementation

**Phase:** Phase 1  
**Milestone:** M1.3 (OBO token exchange)  
**Status:** Design Phase — Ready for Implementation  
**Date:** 2026-09-04

## Overview

M1.3 implements delegated token exchange for downstream calls. When dim processes a message on behalf of an authenticated principal, downstream systems (APIs, databases, other services) should receive a scoped-down token derived from the original rather than the full principal token.

### Example Flow

```
Original request:
  Principal: alice@corp.com (token_id: JWT-alice-full-perms)
  Action: export_sales_data
         ↓
dim processes via export route
         ↓
Calls downstream API: GET /api/sales
  Authorization: Bearer <scoped token>
              (derived from JWT-alice-full-perms, restricted to read-only on sales)
         ↓
Downstream validates scoped token
  Subject: alice@corp.com
  Scope: read:sales
  Issued-by: dim (proxy)
  Original-token-id: JWT-alice-full-perms (for audit)
```

## Core Concepts

### Token Exchange (RFC 8693 - OAuth 2.0)
- **Subject token:** Original principal's token
- **Requested scope:** What the downstream operation needs
- **Issued token:** Scoped-down token for downstream use

### Delegation Model
- **Principal ownership:** Alice owns her token
- **Dim as proxy:** Dim exchanges on Alice's behalf
- **Scope restriction:** Downstream gets minimal permissions
- **Audit trail:** All exchanges recorded

## Subtasks

### M1.3.1: Design Token-Exchange Flow

**Design Document Location:** This file

**Components:**

1. **Token Exchange Service**
   - Endpoint: `POST /internal/token-exchange`
   - Input: Original token + requested scope
   - Output: Scoped token
   - No external dependency (dim issues tokens, not delegating to external STS)

2. **Token Scope Grammar**
   ```
   action:resource[:qualifier]
   
   Examples:
   - read:orders
   - write:customers:acme
   - admin:*
   - execute:stored_procedure:get_sales
   ```

3. **Token Issuance**
   - Uses HS256 (HMAC-SHA256) for simplicity
   - Shared secret: derived from route version + principal + scope (deterministic)
   - Claims:
     ```json
     {
       "sub": "alice@corp.com",
       "scope": "read:orders",
       "issued_at": 1725456000,
       "expires_in": 3600,
       "original_token_id": "JWT-alice-...",
       "issued_by": "dim",
       "route_version": "abc123def456..."
     }
     ```

### M1.3.2: Implement OBO Token Exchange

**Files to Create:**

```
internal/obo/
  ├── token_exchange.go        # Token exchange service
  ├── token_exchange_test.go   # Unit tests
  ├── scope.go                 # Scope parsing & validation
  └── scope_test.go
```

**Token Exchange Service:**

```go
type TokenExchanger struct {
    secret string  // Shared secret for HMAC signing
}

type ExchangeRequest struct {
    OriginalToken string  // Principal's JWT
    Scope         string  // Requested scope (e.g., "read:orders")
    Audience      string  // Intended recipient (e.g., "sales-api.internal")
    TTL           int     // Token lifetime in seconds (default: 3600)
}

type ExchangeResponse struct {
    AccessToken string // Scoped JWT
    TokenType   string // "Bearer"
    ExpiresIn   int    // Seconds
    Scope       string // Granted scope
}

func (te *TokenExchanger) Exchange(ctx context.Context, req *ExchangeRequest) (*ExchangeResponse, error) {
    // 1. Validate original token
    // 2. Parse requested scope
    // 3. Check scope is subset of original (no privilege escalation)
    // 4. Issue new token with reduced scope
    // 5. Return to caller
}

func (te *TokenExchanger) ValidateScope(original, requested string) error {
    // Ensure requested ⊆ original
    // Example: original="read:orders write:customers"
    //          requested="read:orders" ✓
    //          requested="admin:*" ✗
}
```

**Test Coverage:**

- ✅ Scope subset validation (no privilege escalation)
- ✅ Token expiration handling
- ✅ Invalid scope rejection
- ✅ Original token expiry respected
- ✅ Multiple scope items (read:orders read:customers)
- ✅ Wildcard scope handling (admin:*)

**Exit Criteria:**
✅ Token exchange service operational  
✅ Scopes correctly validated  
✅ Tokens issued with correct claims  
✅ Tokens time-bound to original + TTL  

### M1.3.3: Wire OBO into Adapters

**Scope:** HTTP sink (first implementation)

**HTTP Sink Changes:**

```go
type HTTPSink struct {
    // ... existing fields ...
    oboExchanger *obo.TokenExchanger
    oboScope     string  // Config: requested scope for downstream
}

func (h *HTTPSink) Write(ctx context.Context, msgs []*engine.Message) []engine.Result {
    results := make([]engine.Result, len(msgs))
    
    for i, msg := range msgs {
        // Extract original token from principal
        originalToken := msg.Metadata.Principal.Token
        
        // Exchange for scoped token
        exchangeReq := &obo.ExchangeRequest{
            OriginalToken: originalToken,
            Scope:         h.oboScope,
            Audience:      h.oboAudience,
        }
        
        exchangeResp, err := h.oboExchanger.Exchange(ctx, exchangeReq)
        if err != nil {
            results[i] = engine.Result{Error: fmt.Errorf("token exchange failed: %w", err)}
            continue
        }
        
        // Make downstream call with scoped token
        client := &http.Client{}
        req, _ := http.NewRequestWithContext(ctx, "POST", h.url, ...)
        req.Header.Set("Authorization", "Bearer "+exchangeResp.AccessToken)
        
        resp, err := client.Do(req)
        // ... handle response ...
    }
    
    return results
}
```

**Configuration:**

```yaml
sinks:
  downstream-api:
    type: http
    url: https://api.internal/orders/process
    # OBO configuration
    obo:
      scope: "write:orders"        # Requested scope
      audience: "api.internal"     # Intended recipient
      ttl: 300                     # 5 minute token lifetime
```

**Test Coverage:**

- ✅ HTTP call includes scoped token in Authorization header
- ✅ Token exchange called before downstream request
- ✅ Failed exchange → error result (no retry)
- ✅ Multiple messages → each gets fresh token
- ✅ Principal without token → error result

**Exit Criteria:**
✅ HTTP sink exchanges tokens for downstream calls  
✅ Scoped tokens used in Authorization header  
✅ Error handling (failed exchange → error result)  
✅ Integration test proves flow works  

## Implementation Checklist

### Phase 1: Design ✅
- [x] Design token exchange flow (M1.3.1)
- [x] Scope grammar defined
- [x] Token claims specified

### Phase 2: Implementation (Ready to start)
- [ ] Token exchange service (M1.3.2)
- [ ] Scope validation logic
- [ ] Token signing (HS256)
- [ ] Unit tests
- [ ] HTTP sink integration (M1.3.3)
- [ ] Integration test (end-to-end flow)

## Security Considerations

### Token Issuance
- **Secret management:** Shared secret stored in config, never in logs
- **Signing algorithm:** HS256 (symmetric key, suitable for internal service)
- **No external STS:** Dim issues tokens directly (avoids network hops to auth service)

### Privilege Escalation Prevention
- **Scope validation:** Requested scope must be subset of original
- **Original scope enforcement:** New token lifetime ≤ original token lifetime
- **Signature verification:** Downstream validates HMAC before accepting

### Audit Trail
- **Original token ID:** Included in new token claims for traceability
- **Route version:** Stamped on token (ensures token only valid for issuing route)
- **Timestamp:** Issued-at and expires-in recorded

### Downstream Validation
Downstream services should:
1. Verify HMAC signature (using shared secret from dim)
2. Check expiration (exp claim)
3. Check route version (allows invalidation if route changes)
4. Log original_token_id for audit

## Configuration

### Route-Level OBO

```yaml
routes:
  export-sales:
    from: webhook-in
    auth: none
    
    steps:
      - authorize:
          mode: pbac
          # ... PBAC config ...
      
      - translate: |
          {
            "format": body.format,
            "filters": body.filters
          }
    
    sinks:
      - sales-export-api

sinks:
  sales-export-api:
    type: http
    url: https://sales-api.internal/exports/create
    obo:
      scope: "read:sales write:exports"
      audience: "sales-api.internal"
      ttl: 1800  # 30 minutes
```

### Service-Level OBO

```go
// In route initialization:
httpSink := NewHTTPSink(config)
httpSink.SetOBOExchanger(oboService, "read:sales write:exports")
```

## Testing

### Unit Tests (M1.3.2)
```bash
go test ./internal/obo -v
```

### Integration Tests (M1.3.3)
```bash
# Test with mock downstream API
go test ./internal/adapters/http -run TestHTTPSinkWithOBO -v
```

### End-to-End Test
```
1. Start route with HTTP sink + OBO
2. Send message with principal JWT (original full scope)
3. Verify downstream call includes scoped token
4. Verify scoped token validates correctly
5. Verify audit trail records original_token_id
```

## Migration Path

**Phase 0 / Current:** No OBO (all downstream calls use principal's full token)

**Phase 1 (M1.3):**
- Routes can opt-in to OBO via sink config
- Non-OBO sinks continue to work unchanged (backward compatible)

**Phase 2 (Future):**
- OBO becomes default for all sinks
- Explicit `obo: disabled` for cases that need full token

## Exit Criteria

### M1.3.1 ✅
- Flow designed and documented
- Token scope grammar specified
- Claims structure defined

### M1.3.2 (Ready to implement)
- Token exchange service operational
- Scope validation prevents privilege escalation
- Tokens properly signed and expiring
- Unit tests pass

### M1.3.3 (Ready to implement)
- HTTP sink configured for OBO
- Scoped token included in downstream Authorization header
- Failed exchange → error result
- Integration tests pass

## References

- **RFC 8693 (OAuth 2.0 Token Exchange):** https://tools.ietf.org/html/rfc8693
- **JWT/HS256:** https://tools.ietf.org/html/rfc7518
- **M1.2 (PBAC):** `design/M1_2_PBAC_OPA_REFERENCE.md`
- **Phase 0 JWT:** `internal/engine/principal.go`
