# M2.2 Splitter Step — Design

**Status:** Design phase (M2.2.1)  
**Date:** 2026-09-05

## Overview

The Splitter step takes one message and emits many, the mirror image of M2.1 (Aggregator). It evaluates a JSONata expression that produces an array, and emits one output message per array element. Each output carries lineage back to the single input.

## Split Expression

### Expression Surface
- **Type:** JSONata expression
- **Input:** Single message body
- **Output:** Array of elements (one per output message)
- **Example:**
  ```yaml
  - split:
      expr: body.line_items
  ```

### Semantics
- Evaluates the expression against message body
- Expression must produce an array
- If expression produces null or non-array, routes to error_path (configurable)
- One output message per array element
- Empty array produces no output messages
- Output messages preserve input headers and principal

### Examples

**Simple array split:**
```yaml
- split:
    expr: body.items
```
Input: `{"items": [{"id": 1}, {"id": 2}]}`  
Output: Two messages with bodies `{"id": 1}` and `{"id": 2}`

**Computed split (flatten):**
```yaml
- split:
    expr: |
      $map(body.orders, $ ->
        $map($.line_items, item -> {
          "order_id": $.order_id,
          "item": item
        })
      )
```
Input: Order with multiple line items  
Output: One message per line item (flattened)

**Conditional split:**
```yaml
- split:
    expr: |
      body.status = "active" ?
        body.documents :
        []
```
Routes inactive orders to error_path (produces non-array)

## Output Messages

### Structure
```json
{
  "value": <array_element>,
  "index": 0,
  "total_elements": 3,
  "array_source": <original_expr>
}
```

### Metadata
- **CorrelationID:** Inherited from input (all outputs share correlation ID)
- **Principal:** Inherited from input
- **Route:** Inherited from input
- **RouteVersion:** Inherited from input

## Lineage Handling

### SplitterFacet (inverse of AggregatorFacet)
```go
type SplitterFacet struct {
  SplitExpr      string // The JSONata expression used
  TotalElements  int    // Total array elements
  ElementIndex   int    // Index of this output (0-based)
  SplitTrigger   string // "array_produced" or "empty_array"
}
```

### Parent Relationship
- All output messages have ONE parent: the input message
- Lineage facet records: split expression, array size, element index
- Enables tracing each output back to original input
- Correlation ID preserved (all outputs share same correlation ID)

## Configuration

### SplitSpec
```yaml
routes:
  - name: order-split
    from: orders
    steps:
      - split:
          expr: body.line_items          # JSONata expression (required)
          output_expr: |                 # optional: transform output
            {
              "order_id": $root.body.order_id,
              "item": $
            }
          on_non_array: error_path       # or "skip" (skip empty arrays)
          on_error: error_path
```

### Parameters
- **expr** (required): JSONata expression producing array
- **output_expr** (optional): Transform each array element before output
- **on_non_array** (optional): "error_path" (default) or "skip"
- **on_error** (optional): Error handling for expression failures

## Ordering Guarantees

**Output order:** Messages emitted in array order (index 0, 1, 2, ...)  
**Correlation:** All outputs share same correlation ID (for tracing)

## Hot Reload Behavior

The Splitter step has no in-flight state (each input immediately produces outputs). Hot reload is safe — no draining needed.

## Split-Then-Aggregate Round Trip

Demonstrates M2.1 and M2.2 working together:

1. **Split phase:** One order message splits into N line-item messages
   - Each inherits order_id in body for re-aggregation
   - Each carries parent lineage via SplitterFacet

2. **Aggregate phase:** Line items re-aggregate by order_id
   - Groups collect line items from multiple splits
   - Output has AggregatorFacet recording aggregation
   - Final lineage path: original order → split → aggregate → final order

Example flow:
```
Order message
  ↓
Split by line_items
  ├→ LineItem 1 (SplitterFacet: index=0, total=10)
  ├→ LineItem 2 (SplitterFacet: index=1, total=10)
  └→ LineItem 10 (SplitterFacet: index=9, total=10)
  
All LineItems
  ↓
Aggregate by order_id (count=10)
  ↓
Order message (AggregatorFacet: inputCount=10)

Lineage chain:
  Order → Split:0, Split:1, ... Split:9 → Aggregate → Order
```

## Testing Strategy

- **Unit tests (M2.2.2):**
  - Basic array split
  - Non-array/null handling
  - Empty array handling
  - Output transformation
  - Lineage facet generation

- **Integration tests (M2.2.3):**
  - End-to-end split
  - Split-then-aggregate round trip
  - Lineage parent-child verification
  - Complex transformations (map, filter, flatten)

## Exit Criteria

✓ Route can split one message into N  
✓ Each output carries lineage back to single input  
✓ Split-then-aggregate round trip demonstrated  
✓ Configuration validated (expr required)  
✓ Output message structure matches spec  
✓ Tests: 10+ unit + integration, all passing  
✓ Worked example with step-by-step documentation
