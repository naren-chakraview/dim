# M4.8.2 — Route Critique and Review Examples

## Example 1: Under-Specified Route (Fair Rating)

### Input Route
```yaml
version: 1
imports:
  - ../governance/fragments.yaml

sources:
  webhook:
    type: http

routes:
  transform_and_send:
    from: webhook
    steps:
      - translate:
          expr: $
      - translate:
          expr: $.payload.data

sinks:
  output:
    type: http
    url: https://downstream.example.com/ingest
```

### Agent's Critique Request
```json
{
  "route_config_path": "domains/orders/transform-route.yaml"
}
```

### Agent's Findings
```json
{
  "findings": [
    {
      "category": "redundancy",
      "severity": "warning",
      "path": "$.routes.transform_and_send.steps",
      "issue": "Found 2 translate steps; consecutive ones can be combined",
      "action": "Combine multiple translate steps into a single expression for clarity and performance",
      "example": "// Before: [translate($), translate($.payload.data)]\n// After: translate($.payload.data)"
    },
    {
      "category": "convention",
      "severity": "warning",
      "path": "$.routes.transform_and_send",
      "issue": "No error_path configured; failed messages have no dead-letter queue",
      "action": "Add error_path configuration to route pointing to a DLQ sink for failure handling",
      "example": "routes:\n  transform_and_send:\n    error_path:\n      target: dlq_sink\n      retry:\n        max_attempts: 3"
    },
    {
      "category": "security",
      "severity": "important",
      "path": "$.routes.transform_and_send.auth",
      "issue": "No explicit auth declaration; authorization intent is unclear",
      "action": "Add auth: 'none' if intentional, or specify RBAC/ABAC policy for route authorization",
      "example": "routes:\n  transform_and_send:\n    auth: none  # Webhook is pre-authenticated by network"
    },
    {
      "category": "best-practice",
      "severity": "important",
      "path": "$.routes.transform_and_send.contracts",
      "issue": "Route references contracts but enforcement is not confirmed on output sinks",
      "action": "Verify that output sinks have enforce: true to validate messages against contracts",
      "example": "sinks:\n  output:\n    type: http\n    enforce: true  # Validate against contract"
    }
  ],
  "summary": "Found 4 issues: 2 important, 2 warnings",
  "structural_ok": true,
  "overall_rating": "fair"
}
```

### How Human Would Fix It
1. **Combine translate steps**
   ```yaml
   - translate:
       expr: $.payload.data
   ```

2. **Add error_path with DLQ**
   ```yaml
   error_path:
     target: dlq
     retry:
       max_attempts: 3
   ```

3. **Add explicit auth**
   ```yaml
   auth: none  # Webhook is pre-authenticated
   ```

4. **Enable contract enforcement (if applicable)**
   ```yaml
   sinks:
     output:
       type: http
       url: https://downstream.example.com/ingest
       enforce: true
   ```

### Improved Route After Human Review
```yaml
version: 1
imports:
  - ../governance/fragments.yaml

sources:
  webhook:
    type: http

routes:
  transform_and_send:
    from: webhook
    auth: none  # Webhook is pre-authenticated by network
    error_path:
      target: dlq
      retry:
        max_attempts: 3
    steps:
      - translate:
          expr: $.payload.data

sinks:
  output:
    type: http
    url: https://downstream.example.com/ingest
    enforce: true

  dlq:
    type: file
    path: ./dlq/failed-messages.jsonl
```

### Re-run Critique on Improved Route
```json
{
  "findings": [],
  "summary": "No issues found. Route looks good!",
  "structural_ok": true,
  "overall_rating": "excellent"
}
```

---

## Example 2: Well-Designed Route (Excellent Rating)

### Input Route
```yaml
version: 1
imports:
  - ../governance/fragments.yaml

sources:
  orders:
    type: http

routes:
  order_processing:
    from: orders
    auth: none
    error_path:
      target: order_dlq
      retry:
        max_attempts: 3
    steps:
      - filter:
          expr: $exists($.order_id)
      - translate:
          expr: $.items | $sum($.price)

sinks:
  kafka_orders:
    type: file
    path: ./processed-orders.jsonl

  order_dlq:
    type: file
    path: ./dlq/orders.jsonl
```

### Agent's Critique
```json
{
  "findings": [],
  "summary": "No issues found. Route looks good!",
  "structural_ok": true,
  "overall_rating": "excellent"
}
```

### What This Route Does Right
✅ **Has auth declaration** - explicit `auth: none`
✅ **Has error handling** - error_path with retry policy
✅ **Single transform step** - efficient translation
✅ **DLQ configured** - failed messages captured
✅ **Clear intent** - filter ensures required fields, translate computes total

---

## Example 3: Security Issue Detection

### Input Route (Missing Auth Declaration)
```yaml
version: 1
imports:
  - ../governance/fragments.yaml

sources:
  sensitive_data:
    type: http

routes:
  process_sensitive:
    from: sensitive_data
    # ⚠️ No auth declaration!
    error_path:
      target: dlq
    steps:
      - filter:
          expr: "true"

sinks:
  secure_output:
    type: http
    url: https://secure-endpoint.example.com
```

### Critique Findings
```json
{
  "category": "security",
  "severity": "important",
  "path": "$.routes.process_sensitive.auth",
  "issue": "No explicit auth declaration; authorization intent is unclear",
  "action": "Add auth: 'none' if intentional, or specify RBAC/ABAC policy for route authorization",
  "example": "routes:\n  process_sensitive:\n    auth: none  # Explicit: no auth needed for this webhook"
}
```

### Fix
```yaml
auth: none  # Explicit: webhook receives pre-authenticated requests
# OR
auth:
  mode: rbac
  rules: [...]  # RBAC configuration
```

---

## Critique Categories and Meanings

| Category | Severity | Example | Action |
|----------|----------|---------|--------|
| **redundancy** | warning | Two translate steps do same work | Combine into one expression |
| **convention** | warning | No error_path configured | Add error handling |
| **best-practice** | important | Contract not enforced | Add `enforce: true` to sink |
| **security** | important | No auth declaration | Declare `auth: none` or RBAC/ABAC |
| **clarity** | info | Unused imports | Clean up imports list |

---

## Key Principles

### What Critique Can Catch
- Redundant/sequential steps that can be combined
- Missing best-practice configuration (error paths, contracts)
- Security gaps (missing auth declarations)
- Code quality issues (unused imports, confusing nesting)

### What Critique Cannot Catch
- Wrong JSONata expression (semantic error, not structural)
- Wrong business logic (filter expression that excludes needed messages)
- Performance issues not visible in config
- Compliance requirements (audit trails, encryption levels)

**These require human review** — critique is advisory, not authoritative.

---

## Summary: When Critique Helps

| Scenario | Critique Value | Human Review |
|----------|----------------|--------------|
| Generated by agent | High | Recommended |
| Hand-authored by expert | Low | May still catch conventions |
| Legacy route migration | High | Identify modernization needs |
| Code review | Medium | Quick sanity check |
