# M2.7 Static Contract Conformance Checking — Design Document

**Status:** M2.7.1 Design  
**Date:** 2026-09-05

## Overview

Static validation at `dimctl validate` time can catch some contract mismatches before deployment for statically-analyzable JSONata transformations. This is explicitly a **best-effort, partial check** — the runtime `enforce: true` guarantee (design §11.2) remains the only complete check.

## Statically-Analyzable JSONata Subset (M2.7.1)

### Definition

A JSONata expression is **statically analyzable** if it falls into this subset:
1. **Object literals**: `{ "key": expr, "field": expr, ... }`
2. **Property access**: `body.field`, `headers.x`, `metadata.y` (chained okay: `body.nested.deep`)
3. **Literal values**: strings, numbers, booleans, null, arrays of literals
4. **Arithmetic/string operations**: `+`, `-`, `*`, `/`, `&` (concatenation), `||`, `?:` (ternary)
5. **Array construction**: `[expr, expr, ...]`
6. **No dynamic features**: No function calls, no loops (`for`/`foreach`), no variables (`$var`), no conditionals (`if`), no regex, no Date functions

### Examples

**Statically analyzable (can check):**
```jsonata
{ "order_id": body.id, "total": body.amount / 100 }
{ "email": body.email, "status": "processed" }
{ "fields": [body.a, body.b, body.c] }
{ "value": body.x ? body.y : 0 }  // Ternary OK
```

**NOT statically analyzable (cannot check):**
```jsonata
{ "fields": body.items.(name & id) }  // foreach syntax
{ "result": $custom_function(body.id) }  // function call
{ "value": body.items[0].nested.deep }  // dynamic array access
{ "mapped": body.items.($map(this, .) ) }  // map function
$count(body.items)  // function call
body.items[body.index]  // dynamic index
```

## Static Check Algorithm (M2.7.2)

### Input
- JSONata expression string from `translate` step
- Target sink's declared contract (field name → type mapping)

### Process
1. **Subset detection**: Parse expression; check if it fits the analyzable subset
   - If NO → Return "unable to check" (don't fail, just skip)
   - If YES → Continue
2. **Shape construction**: Build a map of output field names → types by:
   - Walking the object literal structure
   - Resolving each field's type from the input message context (body, headers, etc.)
   - For literals: infer type (string, number, boolean, null)
   - For property access: look up type in contract (or assume unknown)
   - For operations: apply type rules (e.g., two strings + arity = string; int/int = float)
3. **Contract comparison**: Check output shape against sink contract
   - Missing required fields → Error
   - Extra fields → Warn (may be okay if sink ignores them)
   - Type mismatch (string → int) → Error
4. **Return result**: "matches contract", "unable to check", or specific mismatch error

## Caveats & False Confidence (M2.7.3)

**The caveat is non-negotiable and must appear in CLI output:**

```
WARNING: Static contract conformance check is a best-effort, partial check.
  • Only simple JSONata transformations can be validated (no functions, loops, conditionals).
  • This check does NOT guarantee runtime behavior — use enforce: true on the sink for a complete guarantee.
  • Passing this check is NOT a substitute for the runtime enforce check.
  • See design documentation for details.
```

This caveat must appear in `dimctl validate` output whenever a static check is performed, whether it passes or fails.

## Implementation Notes (M2.7.2)

### Integration Point
- Wire into `dimctl validate` command after YAML parse + config resolution
- For each route with a `translate` step followed by a sink with a `contract`:
  - Extract the translate expression
  - Fetch the sink's contract
  - Run static check
  - Report result inline with other validation errors/warnings

### Subset Detection
Parse JSONata AST (or expression string pattern matching):
- Allowed AST node types: object, array, string, number, boolean, property, arithmetic, ternary
- Disallowed: function call, for/foreach, variable, regex, date

### Type Inference
- `body.field` → look up in source contract (if available, else unknown)
- String literal → string
- Number literal → number
- Arithmetic operations → apply rules (int+int=int, int/int=float, string&string=string)
- Unknown + anything → unknown (don't error)

## Exit Criteria (M2.7)

✅ Static checker identifies statically-analyzable subset correctly  
✅ For analyzable transforms, detects at least one class of real contract mismatch  
✅ Visibly declines to check non-analyzable expressions (doesn't false-pass)  
✅ CLI caveat is prominent and unavoidable  
✅ Runtime `enforce: true` check is unaffected by static check presence  

