# M2.5 Authorization Obligations and Redaction — Design Document

**Status:** M2.5.1 Obligation Vocabulary Design  
**Date:** 2026-09-05

## Overview

Authorization decisions from a PDP (Policy Decision Point) may come with **obligations** — additional requirements that must be enforced beyond the allow/deny decision. M1.2 already parses obligations from PDP responses, but nothing consumes them. M2.5 closes this loop by defining an obligation vocabulary and implementing enforcement.

## Obligation Vocabulary

### Redaction/Masking Obligation

**Type:** `redact_fields`

Redacts sensitive fields from a message body before it continues downstream.

**Parameters:**
- `fields` (required, array of strings): Field paths to redact
  - Supports dot notation: `customer.ssn`, `personal_info.email`
  - Supports wildcard: `secret.*` (redact all fields under secret)
- `replacement` (optional, string): Value to replace with (default: `***REDACTED***`)
- `depth` (optional, string): Redaction scope
  - `shallow`: Only redact top-level or specified path (default)
  - `deep`: Recursively redact nested occurrences
  - `recursive`: Redact in all nested objects/arrays

**Example:**

```json
{
  "type": "redact_fields",
  "parameters": {
    "fields": ["ssn", "credit_card", "password"],
    "replacement": "[REDACTED]",
    "depth": "shallow"
  }
}
```

Transforms message:
```json
// Before
{"order_id": 123, "customer": {"ssn": "123-45-6789", "name": "Alice"}}

// After
{"order_id": 123, "customer": {"ssn": "[REDACTED]", "name": "Alice"}}
```

### Attribute Masking Obligation (Future)

**Type:** `mask_attributes`

Masks specific message attributes (metadata) rather than body fields.

**Parameters:**
- `attributes` (required, array): Metadata fields to mask
- `replacement` (optional, string): Replacement value

Not implemented in M2.5; reserved for future use.

### Field Transformation Obligation (Future)

**Type:** `transform_field`

Applies a transformation function to a field (e.g., tokenization, hashing).

**Parameters:**
- `field` (required, string): Field to transform
- `transformer` (required, string): Transformer type (e.g., "hash_sha256", "tokenize")

Not implemented in M2.5; reserved for future use.

## Design Principles

1. **Fail-safe default** — If an obligation is unknown, reject the message (don't silently skip)
2. **Early enforcement** — Apply obligations immediately after authorization, before any steps
3. **Lineage tracking** — Record which obligations were applied in message metadata/lineage
4. **Composable** — Multiple obligations can be applied in sequence
5. **Message-preserving** — Redaction modifies the message body, not the message structure
6. **Type preservation** — Redacted fields retain their type (string → string, int → int)

## Obligation Application Flow

```
PDP Decision
    ↓
Is Decision "allow"?
    ├─ NO  → Return authorization denied error
    └─ YES ↓
        Any Obligations?
            ├─ NO  → Return message unchanged
            └─ YES ↓
                Apply Each Obligation in Order
                    ├─ Unknown Type? → Error: "unknown obligation type"
                    ├─ Redact Fields → Remove/mask values
                    └─ Track in Lineage
                Return Modified Message
```

## Lineage Recording

When obligations are applied, record in message lineage/metadata:
- Obligation type(s) applied
- Fields affected
- Replacement value(s) used

Example facet:
```json
{
  "name": "obligation_facet",
  "obligation_type": "redact_fields",
  "fields_redacted": ["ssn", "credit_card"],
  "replacement": "[REDACTED]",
  "timestamp": "2026-09-05T10:30:00Z"
}
```

## Implementation Notes (for M2.5.2)

### Field Path Resolution

Support multiple field path syntaxes:
- `simple_field` — top-level field
- `nested.field.path` — dot notation for nested objects
- `array[0].field` — array element access (apply to all elements if no index)
- `parent.*` — wildcard for all children

### Redaction Strategy

For each field to redact:
1. Navigate to the field using the path
2. Replace value with `replacement` string (or `***REDACTED***` if not specified)
3. Preserve field existence (don't delete, just empty/mask)
4. Handle missing paths gracefully (skip, don't error)

### Error Handling

- Unknown obligation type → Permanent error (authorization failure)
- Malformed obligation (missing required params) → Permanent error
- Missing field to redact → Skip (no error, field may not exist in all messages)
- Invalid field path syntax → Permanent error

## Exit Criteria (M2.5.1)

✅ Obligation vocabulary defined with redaction as primary type  
✅ Parameter schema specified (fields, replacement, depth)  
✅ Application flow documented  
✅ Lineage recording specified  
✅ Future obligations listed (mask_attributes, transform_field)  
✅ Error handling strategy defined  

## Relationship to Existing Features

- **PDPObligation** (M1.2) — Already exists; Type and Parameters fields used
- **Authorization step** (M1.2) — Receives obligations in PDPDecisionResponse
- **Message metadata** (core) — Will carry obligation facets for lineage
- **Redaction in pipelines** — Orthogonal to filters/transforms; obligations enforce policy

