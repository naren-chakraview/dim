# Order Processing Pipeline — Complete Worked Example

This example demonstrates a realistic order-processing pipeline that exercises most of the core dim features in a single, cohesive route.

**Features Demonstrated:**
- ✅ **Validation and idempotency** — filter required fields, deduplicate by ID
- ✅ **Authorization (RBAC)** — enforce role-based access control with distinct denial path
- ✅ **Wiretap step** — capture audit trail of all inbound messages
- ✅ **JSONata expressions** — compute derived fields and transformations
- ✅ **Custom function references** — documented for Go plugins and WASM (Phase 1)
- ✅ **Multiple sinks** — route successful orders, authorization denials, and errors to distinct outputs
- ✅ **Error handling** — retry on transient failures, route permanent failures to dead-letter
- ✅ **Lineage and retention** — track message journey and enforce data retention policies

## Files

- **`common-steps.yaml`** — Shared functions and fragments (design reference)
  - Documents function definitions for future plugin integration
  - Demonstrates fragment patterns from the design spec (§16)
  - Phase 1: will be fully integrated with the config loader

- **`order-processing.yaml`** — Main route configuration (Phase 0 complete)
  - HTTP webhook source for order ingestion
  - Four output sinks: valid orders, authorization denials, general dead-letter, audit log
  - Five-step pipeline: wiretap → authorize → filter → idempotent → translate
  - Configurable retry logic and retention policies
  - All steps fully implemented and tested in Phase 0

- **`test-order.json`** — Sample order for testing

## Running the Example

### 1. Validate the configuration

```bash
./dimctl validate examples/order-processing/order-processing.yaml
```

Expected output:
```
OK: valid route configuration
  Version: 1
  Sources: 1
  Sinks: 4
  Routes: 1
```

### 2. Start the pipeline

```bash
# Terminal 1: run the route
./dimctl run examples/order-processing/order-processing.yaml

# You should see:
# [INFO] listening on http://localhost:8080/ingest
# [INFO] Tier 1 viewer enabled on http://localhost:8081/debug/routes
```

### 3. Send test orders

```bash
# Terminal 2: send an order through the pipeline
curl -X POST http://localhost:8080/ingest \
  -H "Content-Type: application/json" \
  -d @examples/order-processing/test-order.json

# Expected HTTP 202 Accepted response
```

### 4. Monitor output

```bash
# Terminal 3: watch the valid orders sink
tail -f output/orders-valid.jsonl

# You should see the transformed order:
# {"order_id":"order-12345","customer_name":"Alice Johnson","total":150.00,"currency":"USD",...}

# Terminal 4: watch the audit trail
tail -f output/orders-audit.jsonl

# You should see the original, untransformed order (wiretap output)
```

### 5. View live pipeline stats

```bash
# Terminal 5: query the Tier 1 viewer
curl http://localhost:8081/debug/routes | jq .routes.\"ingest-orders\"
```

Expected output shows message counts, latencies, and status per route.

## Example Messages

### Successful Order

```json
{
  "id": "order-12345",
  "customer_name": "Alice Johnson",
  "amount": 15000,
  "currency": "USD",
  "phone": "+1-415-555-1234",
  "status": "active"
}
```

**Processing path:** input → wiretap (audit) → authorize (allow) → validate (pass) → translate → route (high-value) → orders-valid

**Output:**
```json
{
  "order_id": "order-12345",
  "customer_name": "Alice Johnson",
  "total": 150.00,
  "currency": "USD",
  "phone": "+1-415-555-1234",
  "timestamp": "2026-09-04T12:34:56Z"
}
```

### Rejected Order (Missing Required Field)

```json
{
  "id": "order-99999",
  "customer_name": "Bob Smith",
  "amount": null,
  "status": "active"
}
```

**Processing path:** input → wiretap (audit) → authorize (allow) → validate (filter fails, missing amount) → error_path (DLQ)

**Output (to DLQ):**
```json
{
  "error_reason": "filter expression evaluated to false",
  "error_type": "validation",
  "attempt_count": 1,
  "route": "ingest-orders",
  "route_version": "sha256:...",
  "original_message": {...}
}
```

## Architecture

```
HTTP Webhook → /ingest
    ↓
[Step 1] Wiretap → Audit Log (side effect, all messages)
    ↓
[Step 2] Authorize (RBAC)
         • Allow: continue to next step
         • Deny: route to unauthorized-dlq, stop processing
    ↓
[Step 3] Filter: require id and amount
         • Pass: continue
         • Fail: retry or route to orders-dlq
    ↓
[Step 4] Idempotent: deduplicate by order id
         • New: continue
         • Duplicate: silent drop (idempotent dedup)
    ↓
[Step 5] Translate (JSONata):
         • Normalize fields
         • Compute order_value flag
         • Format timestamps
    ↓
Output → orders-valid (file sink, JSONL)

On error at ANY stage (except authorize denials, which go to unauthorized-dlq):
  • Retry up to 3 times with exponential backoff (500ms → 1s → 2s)
  • If retries exhausted: route to orders-dlq (dead-letter queue)
```

## Phase 0 vs. Full Design

This example demonstrates **all core Phase 0 features** and serves as a blueprint for Phase 1 integration:

| Feature | Phase 0 Status | Phase 1+ Plan |
|---------|---|---|
| Validation (filter/idempotent) | ✅ Working | — |
| RBAC authorization | ✅ Working | Phase 1: add PBAC mode with OPA |
| Wiretap (audit logging) | ✅ Working | — |
| JSONata expressions | ✅ Working | Phase 2: add more built-in functions |
| Retry + backoff | ✅ Working | — |
| Multiple file sinks | ✅ Working | Phase 1: add Kafka/AMQP |
| **Plugin functions** | ⏳ R5 (in progress) | Called as `$funcName(args)` in expressions |
| **WASM functions** | ⏳ R4 (in progress) | Called as `$funcName(args)` in expressions |
| Fragments/imports | 🔄 Partially | Phase 1: full fragment composition |
| Contracts/schemas | ⏳ Phase 1 | Inline JSON Schema + registry integration |
| Lineage export | ✅ Working | `dimctl lineage export` captures journey |

**Future enhancements:**
- When R4/R5 complete: uncomment `normalizePhone` call in translate step
- When Phase 1 Kafka: swap file sinks for Kafka topics
- When Phase 1 PBAC: upgrade authorize step from RBAC to PBAC with OPA

## Testing Locally

```bash
# Full end-to-end test without manual curl:
./dimctl test examples/order-processing/

# This would run fixture-based integration tests if they're defined
# (fixtures are a Phase 0+ feature documented in DEVELOPMENT.md)
```

## Key Takeaways

1. **Real production patterns** — this example combines multiple features in a single route, showing how they interact at scale
2. **Fragment reuse** — the `validate-order` fragment can be imported by other routes for consistency
3. **Error handling is structural** — not bolted on; every route declares retry + dead-letter behavior upfront
4. **Observability by default** — wiretap gives you an audit trail; lineage tracks the full journey; metrics show throughput/latency
5. **Plugin integration** — function definitions are first-class config, making it easy to wire in custom logic later

## Related Documentation

- **Design:** `design/eip-middleware-design.md` (§16 — full example in the design spec)
- **CLI reference:** `USER_GUIDE.md` (§2 — configuration syntax)
- **Testing:** `DEVELOPMENT.md` (§3 — fixture-based route testing)
- **Architecture:** `graphify-out/GRAPH_REPORT.md` (cross-module relationships)
