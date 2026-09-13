# M4.9 — Impact Analysis Query Examples

## Example 1: Contract Version Bump

### Query
An agent asks: "What routes/sinks would be affected if we bump the payment contract?"

```json
{
  "change_type": "contract",
  "change_name": "payment-contract",
  "change_version": "2.0"
}
```

### Response
```json
{
  "change_type": "contract",
  "change_name": "payment-contract",
  "affected_routes": ["process-payment", "validate-refund"],
  "affected_sinks": ["kafka-payments", "dlq"],
  "references": [
    {
      "source_type": "sink",
      "source_name": "kafka-payments",
      "target_type": "contract",
      "target_name": "payment-contract",
      "is_static": true,
      "config_path": "domains/payments/payment-routes.yaml"
    }
  ],
  "uncertain_refs": [],
  "summary": "Changing contract 'payment-contract' would affect: 2 routes, 2 sinks (Confidence: 100%)",
  "impact_level": "high",
  "confidence": 1.0
}
```

**Insight:** Version bump requires testing these 2 routes + validating 2 sinks. Complete picture (100% confidence).

## Example 2: Sink Configuration Change

### Query
Kafka team migrates the payments topic. What routes are affected?

```json
{
  "change_type": "sink",
  "change_name": "kafka-payments"
}
```

### Response
```json
{
  "change_type": "sink",
  "change_name": "kafka-payments",
  "affected_routes": ["process-payment", "batch-export"],
  "affected_sinks": [],
  "references": [...],
  "uncertain_refs": [],
  "summary": "Changing sink 'kafka-payments' would affect: 2 routes (Confidence: 100%)",
  "impact_level": "medium",
  "confidence": 1.0
}
```

**Insight:** 2 routes use this sink. Migration needs testing these routes + endpoint updates.

## Example 3: With Uncertain References

### Query
Impact of changing a dynamically-resolved connection?

```json
{
  "change_type": "connection",
  "change_name": "message-broker"
}
```

### Response
```json
{
  "change_type": "connection",
  "change_name": "message-broker",
  "affected_routes": ["known-route"],
  "references": [
    {
      "source_type": "route",
      "source_name": "known-route",
      "target_type": "connection",
      "is_static": true
    }
  ],
  "uncertain_refs": [
    {
      "source_type": "step",
      "source_path": "$.routes.dynamic-route.steps[0].translate",
      "target_type": "connection",
      "is_static": false,
      "uncertainty": "connection name computed from JSONata expression; cannot determine statically"
    }
  ],
  "summary": "Changing connection 'message-broker' would affect: 1 route. Additionally, 1 uncertain reference (dynamic names) not included in this analysis (Confidence: 50%)",
  "impact_level": "medium",
  "confidence": 0.5
}
```

**Insight:** 50% confidence - 1 route definitely affected, 1 more might be (dynamic route uses JSONata-computed connection names). Agent should flag this to human: "Some routes use dynamic connection names - manual review needed."

## How Agents Use This

1. **During design review:** Agent asks impact of proposed contract version bump
2. **During migration planning:** Agent identifies all affected routes before making changes
3. **During incident response:** Agent quickly maps "if we disable this sink, what breaks?"
4. **Confidence-aware:** Agent warns human when analysis is incomplete (dynamic references)

## What This Enables

- Agents propose changes without manual YAML grepping
- Humans trust agent impact analysis because uncertainty is explicit
- No false confidence - "unknown" cases are clearly marked
- Reduces risk of missed dependencies
