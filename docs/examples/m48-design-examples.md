# M4.8 — Agent-Assisted Route Design Examples

## M4.8.1 — Natural-Language-to-Route Scaffolding Example

### Scenario
An agent is asked to design a route that:
- Receives webhook events from a payment processor
- Filters and transforms payment data
- Outputs to file-based sink
- Includes error handling for failures

### User's Intent
"Connect payment webhook to file output with error handling"

### Agent's Process

#### Step 1: Call scaffold_from_intent
```json
{
  "intent": "Connect payment webhook to file output with error handling",
  "domain": "payments",
  "template_shape": ""
}
```

#### Step 2: Agent's Analysis
- **Intent keywords detected**: "webhook" → source type: http, "file" → sink type: file, "error" → need error handling
- **Template selected**: passthrough (simple connector, no transformation mentioned)
- **Inferred signals**:
  - Source: http (from "webhook")
  - Sink: file (explicitly mentioned)
  - Steps: none beyond default filter

#### Step 3: Agent's Response
```json
{
  "success": true,
  "template_used": "passthrough",
  "route_yaml": "version: 1\nimports:\n  - ../governance/fragments.yaml\n\nsources:\n  input:\n    type: http\n\nroutes:\n  passthrough:\n    from: input\n    error_path:\n      target: error\n    steps:\n      - filter:\n          expr: \"true\"\n\nsinks:\n  output:\n    type: file\n\n  error:\n    type: file\n    path: ./dlq/errors.jsonl",
  "placeholders": [],
  "validation_result": {
    "valid": true,
    "errors": [],
    "warnings": [],
    "route_version": "sha256:abc123..."
  },
  "reasoning": "Selected 'passthrough' template based on intent keywords. Detected source type: http. Detected sink type: file. Placeholders ({{ }} markers) indicate fields requiring human judgment.",
  "next_steps": [
    "1. Review the generated route YAML",
    "2. Fill in any placeholders marked with {{ }} comments",
    "3. Run `dimctl test` with fixtures to verify behavior",
    "4. Create a PR for domain team review",
    "5. After approval, merge to main via GitOps pipeline"
  ]
}
```

### Step 4: Human Review Process
Domain engineer reviews the generated route:
1. ✅ Checks structure (sources, routes, sinks, error_path all present)
2. ✅ Runs `dimctl validate` - passes
3. ✅ Creates test fixtures with sample webhook events
4. ✅ Runs `dimctl test` - all fixtures pass
5. Creates PR with agent-generated route
6. Domain team approves and merges
7. Route deployed via GitOps pipeline

### Key Takeaways
- Agent generated structurally valid route (passes `dimctl validate`)
- No placeholders needed (intent was clear enough)
- Human review still required before deployment
- Same validation as human-authored route
- Framework for future enhancements (transform intent → different template)

---

## M4.8.1b — Transformation Example

### User's Intent
"Connect webhook to file output with PII redaction"

### Agent's Analysis
- **Keywords detected**: "webhook" → http, "file" → file, "redaction" → transform
- **Template selected**: transform (transformation mentioned)
- **Placeholders identified**: JSONata redaction expression

### Agent's Response
Generated route includes:
```yaml
routes:
  transform:
    from: input
    steps:
      - translate:
          expr: '$'  # TODO: Implement transformation logic
sinks:
  output:
    type: file
```

With placeholder:
```json
{
  "path": "$.routes.transform.steps[0].translate.expr",
  "reason": "JSONata expression for transformation not inferred from intent",
  "suggestion": "Provide a JSONata expression to extract/transform fields (e.g., `$ | {card_number: \"***\", ssn: \"***\"}`)"
}
```

### Human Fills In
```yaml
- translate:
    expr: '$ | { card_number: "***", ssn: "***", ...other_fields: $ }'
```

---

## M4.8.1c — Contract-Enforced Example

### User's Intent
"Validate messages against contract before publishing to output"

### Agent's Analysis
- **Keywords detected**: "contract" → contract-enforced template
- **Template selected**: contract-enforced

### Generated Route
```yaml
routes:
  enforced:
    from: input
    error_path:
      target: error
    steps:
      - translate:
          expr: '$'

sinks:
  output:
    type: file
    enforce: true  # Enforce contract on output
```

### Key Point
Agent correctly inferred `enforce: true` from the "contract" intent keyword, matching M4.8.2's critique pattern for contract enforcement.

---

## Summary: Scaffolding Best Practices

**Agent succeeds when intent is clear about:**
- Source type (webhook/http, file, sftp, exec)
- Sink type (file, http, sftp, exec)
- Transformation needs (mentions "transform", "redact", "extract", etc.)
- Contract enforcement (mentions "contract", "validate", "enforce")
- Error handling (mentions "error", "dlq", "retry", etc.)

**Placeholders used when:**
- JSONata expressions needed (too specific to infer)
- Exact field names unclear
- Domain-specific business logic required

**Human review always includes:**
1. Structural validation (`dimctl validate`)
2. Test coverage (fixtures exercising the route)
3. Contract compliance (if applicable)
4. Security audit (auth declarations, encryption, etc.)
