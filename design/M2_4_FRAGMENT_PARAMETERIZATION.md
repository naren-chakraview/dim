# M2.4 Fragment Parameterization — Design Document

**Status:** M2.4.1 Design  
**Date:** 2026-09-05

## Overview

Fragments enable reuse of configuration patterns (retry policies, authorization checks, transformations) across routes. Currently, fragments are static — a fragment like `retry.yaml` can be imported, but if different routes need different retry parameters, separate copies must be created.

Fragment parameterization adds **substitutable parameters** so one fragment can be imported with different values per route, eliminating copy-paste variants and centralizing policy definitions.

## Design Principles

1. **Distinct from secrets syntax** — Use `${PARAM:name}` (not `${name}` or `$param(name)`) to avoid collision with existing `${SECRET:name}` (design §12).
2. **Late binding** — Parameters are resolved at route composition time, after all imports and merges are complete.
3. **Type-agnostic** — Parameters can substitute any YAML value: scalar, list, or object.
4. **Scoped defaults** — A fragment can declare default parameter values; importers can override.
5. **Validation at composition time** — Missing or unused parameters surface as errors in `dimctl validate`, not silently.

## Parameter Syntax

### Fragment Definition (with defaults)

```yaml
# fragments/retry-policy.yaml
$params:
  maxRetries: 3            # Default: 3 retries
  backoffMultiplier: 2.0   # Default: exponential backoff (2x)
  maxBackoffSec: 300       # Default: max 5 minutes
  retryableErrors:         # Default: common transient errors
    - "connection_error"
    - "timeout"

steps:
  - name: retry
    type: idempotent
    config:
      max_retries: ${PARAM:maxRetries}
      backoff_multiplier: ${PARAM:backoffMultiplier}
      max_backoff_sec: ${PARAM:maxBackoffSec}
      retryable_error_codes: ${PARAM:retryableErrors}
```

### Route Usage (override parameters)

```yaml
# routes/aggressive-retry.yaml
$import: fragments/retry-policy.yaml

$params:
  maxRetries: 10          # Override: 10 retries (more aggressive)
  backoffMultiplier: 1.5  # Override: slower backoff

routes:
  critical-orders:
    from: kafka-orders
    steps:
      # Retry step now uses overridden params:
      # max_retries: 10, backoff_multiplier: 1.5, etc.
      - $ref: retry  # References fragment's retry step with overridden params
    sinks:
      - sink-db
```

### Scoping Rules

1. **Fragment defaults** are declared in `$params:` at fragment root
2. **Route overrides** declared in `$params:` at route root
3. **Overrides cascade**: route `$params` override fragment `$params`
4. **Resolution order**:
   - Look for `${PARAM:name}` in the composed route
   - If not found in `$params`, check fragment defaults
   - If still not found, error: "undefined parameter 'name' in route X"

## Exit Criteria (M2.4.1)

✅ Parameter syntax defined: `${PARAM:name}` (distinct from `${SECRET:name}`)  
✅ Scoping rules documented: defaults + overrides + resolution order  
✅ Fragment structure extended: `$params:` section for defaults  
✅ Type handling specified: scalars, lists, objects allowed  
✅ Composition semantics specified: when/where parameters are resolved  

## Implementation Notes (for M2.4.2)

- Parameter substitution occurs **after** fragment import but **before** final validation
- Use string replacement (template-like) for scalar values; for complex types, use YAML merge
- Validate at composition time: all `${PARAM:*}` references must resolve to a value
- Error messages should include: parameter name, expected type (if specified), where it's used

## Example: Transformation Fragment with Parameters

```yaml
# fragments/order-transform.yaml
$params:
  includeMetadata: true    # Whether to include timestamps
  currencyCode: "USD"      # Currency for formatting
  roundPrecision: 2        # Decimal places for amounts

steps:
  - name: transform
    type: translate
    config:
      expr: |
        $ | {
          id: $.order_id,
          amount: ($.total round ${PARAM:roundPrecision}) & " ${PARAM:currencyCode}",
          metadata: ${PARAM:includeMetadata} ? {timestamp: $now(), version: "v2"} : null
        }
```

Usage with different settings:

```yaml
# Route A: full details
$import: fragments/order-transform.yaml
$params:
  includeMetadata: true
  currencyCode: "EUR"

# Route B: compact (no metadata)
$import: fragments/order-transform.yaml
$params:
  includeMetadata: false
  currencyCode: "GBP"
```

Both routes use the same fragment definition; parameters customize the behavior.

## Relationship to Existing Features

- **Secrets** (`${SECRET:name}`) — orthogonal; secrets are credentials/runtime values; params are configuration-time composition values
- **Fragment imports** (`$import`) — parameters are resolved after imports
- **Composition/merge** — parameters affect how fragments merge into routes
- **Validation** (`dimctl validate`) — should report undefined or unused parameters as errors

