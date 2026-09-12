# DIM Visual Route Editor — User Guide

Comprehensive guide to authoring and editing routes visually in the DIM Studio.

## Table of Contents

1. [UI Overview](#ui-overview)
2. [Canvas (DAG Visualization)](#canvas-dag-visualization)
3. [Configuration Panel (Schema-Driven Forms)](#configuration-panel-schema-driven-forms)
4. [JSONata Code Editor](#jsonata-code-editor)
5. [Validation Panel](#validation-panel)
6. [Workflows](#workflows)
7. [YAML Round-Trip Guarantees](#yaml-round-trip-guarantees)
8. [Known Limitations](#known-limitations)
9. [FAQ](#faq)

---

## UI Overview

### Layout

```
┌─────────────────────────────────────────────────────────┐
│  DIM Visual Route Editor (order-payment)           ●    │  Header
├─────────────────────────┬──────────────────────────────┤
│                         │                              │
│     Canvas (DAG)        │  Configuration Panel         │  Main
│   (2/3 width)           │  (1/3 width)                 │
│                         │                              │
│     [Source]            │  ☑ Name: order-payment       │
│        ↓                │  ☑ Type: translate           │
│     [translate]         │  ○ Expression: $payload.id   │
│        ↓                │                              │
│     [filter]            │  [Monaco Editor for expr]    │
│        ↓                │                              │
│     [Sink]              │                              │
│                         │                              │
├─────────────────────────┴──────────────────────────────┤
│  ✓ Valid · Route is ready to save · sha256:abc123      │  Validation
│  Validated at 14:32:45                                 │
└─────────────────────────────────────────────────────────┘
```

### Header

- **Title** — Route name and domain
- **Dirty Indicator** (●) — Shows if unsaved changes exist
- **Domain Badge** — Displays route domain

---

## Canvas (DAG Visualization)

The Canvas visualizes your route as a directed acyclic graph (DAG) of steps.

### Node Types

| Type | Color | Meaning |
|------|-------|---------|
| **Source** | 🟢 Green | Input adapter (Kafka, HTTP, S3, etc.) |
| **Step** | 🟡 Yellow | Transformation (translate, filter, log, etc.) |
| **Sink** | 🔵 Blue | Output adapter (file, S3, HTTP, etc.) |
| **Error Sink** | 🔴 Red | Error handler (optional) |

### Interactions

**Select a Step:**
- Click any node → node highlights in blue, config panel updates
- Step name appears in the panel title

**Pan & Zoom:**
- **Drag** the canvas to pan
- **Scroll** to zoom in/out
- **Double-click** to fit to screen

**Hover Tooltips:**
- Hover over a node to see step name and type (when implemented)

---

## Configuration Panel (Schema-Driven Forms)

The right panel renders auto-generated forms based on the route schema.

### How It Works

1. Select a step in the canvas
2. The panel loads the schema for that step type
3. Form fields are generated based on property types:
   - `type: string` → Text input
   - `type: number` → Number input
   - `type: boolean` → Checkbox
   - `enum: [...]` → Select dropdown
   - Complex objects → Nested forms

### Field Types

#### Text Input
```
Name: ___________________
Required fields show an asterisk (*)
```
- Single-line text
- Supports maxLength constraint
- Validates on blur

#### Number Input
```
Timeout (ms): [  5000  ]
Constraints: min, max
```
- Integer or decimal
- Enforces min/max bounds

#### Select Dropdown
```
Adapter Type: [  http  ▼]
             • http
             • kafka
             • file
```
- Enums from schema
- First option is placeholder

#### Checkbox
```
☐ Enable Retry Logic
```
- Boolean flag
- Shows description text

#### JSONata Editor
```
Expression: ┌──────────────────────────────┐
           │ $payload.amount * 1.1        │ ← Syntax highlighting
           │ ✓ Valid JSONata expression   │
           └──────────────────────────────┘
```
- Embedded Monaco editor
- Auto-closes brackets/quotes
- Validation feedback
- See [JSONata Code Editor](#jsonata-code-editor)

#### Text Area
```
Description: ┌──────────────────────────────┐
            │ This step enriches the      │
            │ payload with...            │
            └──────────────────────────────┘
```
- Multi-line text
- Enforces maxLength

### Validation

Fields validate on **blur** (when you leave the field):
- **Red highlight** — Invalid value
- **Error message** — Explanation of the problem

Common validation rules:
- `required` — Field must not be empty
- `minLength` / `maxLength` — String length constraints
- `minimum` / `maximum` — Number range
- `pattern` — Regex validation (if specified in schema)
- `enum` — Value must be one of allowed values

### Nested Forms

For object-type fields, a nested form appears:

```
Config: [Nested Form Header]
  ├─ Host: ___________________
  ├─ Port: [  443  ]
  └─ TLS: ☐
```

Click the header to collapse/expand.

---

## JSONata Code Editor

The JSONata editor is a Monaco-based code editor with syntax highlighting for JSONata expressions.

### When It Appears

Fields detected as JSONata (names containing "expr", "expression", or "condition"):
- `expression` — Filter step
- `filter_expression` — Custom filter
- `transform_expr` — Translate step

### Features

- **Syntax Highlighting** — JavaScript-like highlighting (JSONata is similar)
- **Auto-Closing** — Brackets, quotes, braces auto-close
- **Validation** — Real-time syntax checking
- **Help Panel** — Common patterns and documentation link

### Examples

**Access payload fields:**
```
$payload.id
$payload.amount
$payload.status
```

**Arithmetic:**
```
$payload.amount * 1.1
($payload.price * $payload.quantity) - $payload.discount
```

**String operations:**
```
$payload.id & ":" & $payload.type    (* Concatenation *)
$lowercase($payload.status)
$substring($payload.email, 0, 5)
```

**Conditionals:**
```
$payload.status = "active"
$payload.amount > 1000
($payload.country = "US") and ($payload.age >= 18)
```

**Array operations:**
```
$payload.items[0]
$payload.items.price    (* Gets price of all items *)
```

**Functions:**
```
$sum($payload.items.price)
$count($payload.items)
$max($payload.amounts)
```

### Validation Status

```
✓ Valid JSONata expression        (Green checkmark)
⚠️ Check syntax                    (Red warning)
```

Validation checks:
- Balanced brackets `()`, `[]`, `{}`
- Balanced quotes `"..."`, `'...'`
- No incomplete expressions

### Help Panel

Click the "JSONata Syntax Help" section to expand:
- Common patterns
- Function examples
- Link to [JSONata documentation](http://docs.jsonata.org/)

---

## Validation Panel

Real-time feedback on route validity appears at the bottom.

### Status Indicators

| Icon | Status | Meaning |
|------|--------|---------|
| ✓ | Valid | Route passes all checks |
| ⚠️ | Invalid | Errors found (see details) |
| ⟳ | Validating | Check in progress |

### Error Display

**Header:**
```
⚠️ 2 errors, 1 warning
sha256:abc123def456...
```

**Error List:**
```
Errors (2)
  ✕ REQUIRED_FIELD_MISSING: routes.order-payment
    Field "from" is required
    
    How to fix:
    Add the missing required field to your route configuration.

  ✕ INVALID_SCHEMA: routes.order-payment.steps[0]
    Schema validation failed
    
    How to fix:
    Check that your YAML syntax is correct and matches the schema.

Warnings (1)
  ! MISSING_IMPORT: (no path)
    Governance fragment not imported
    
    How to fix:
    Add the missing governance fragment import to your route.
```

### Error Codes

Common error codes and fixes:

| Code | Meaning | Fix |
|------|---------|-----|
| `REQUIRED_FIELD_MISSING` | Required field not set | Add the field to your config |
| `INVALID_SCHEMA` | YAML syntax mismatch | Check YAML formatting |
| `VALIDATION_FAILED` | Custom validation rule failed | See error details |
| `INVALID_ADAPTER_TYPE` | Unsupported adapter | Use: http, kafka, file, s3, amqp, sftp |
| `DUPLICATE_STEP` | Step name already exists | Rename to unique name |
| `MISSING_IMPORT` | Governance fragment missing | Add import |
| `CROSS_DOMAIN_SECRET` | Secret not accessible | Use `${SECRET:domain/name}` |
| `UNRESOLVED_SECRET` | Secret not found | Check secret registration |

### Auto-Validation

The panel auto-validates your route:
- **On change** — Updates form after 500ms debounce
- **On save** — Full validation before writing file
- **Timestamp** — Shows when last validated

---

## Workflows

### Creating a New Route

1. **Use `dimctl scaffold`** to create initial route file:
   ```bash
   dimctl scaffold --domain payments --route order-payment
   ```
   This creates: `domains/payments/order-payment.yaml`

2. **Open in studio:**
   ```bash
   dimctl studio domains/payments/order-payment.yaml
   ```

3. **Configure each step:**
   - Click source node → configure input adapter
   - Click each step → configure transform
   - Click sink node → configure output adapter

4. **Add steps:**
   - Edit the YAML directly (or use raw editor when available)
   - Save → studio reloads and shows new steps

5. **Validate and save:**
   - Check validation panel
   - Click Save when valid
   - YAML is written back to disk

### Modifying an Existing Route

1. Open studio (changes auto-reload when files change on disk)
2. Click the step to edit
3. Update config in the panel
4. Changes auto-save (5 second debounce)
5. Validation feedback shows in panel

### Testing a Route

After saving:
```bash
dimctl test domains/payments/order-payment.yaml
```

### Viewing Route Metrics

(Optional Tier 1 viewer integration)
```
Click "Show Live View" to see live route state alongside definition
```

---

## YAML Round-Trip Guarantees

When you edit a route and save, the YAML is written back with maximum preservation of hand-edited structure.

### Preserved ✅

- **Comments** above keys and sections
- **Blank lines** between sections
- **Key ordering** in maps
- **Indentation** style (mostly)
- **Quote style** (single/double, where detectable)

### Not Preserved ❌

- **Inline comments** (e.g., `key: value # comment`) — moved to line above
- **Multi-line strings** — may be reformatted
- **Arbitrary comment positioning** between keys — normalized
- **YAML anchors/aliases** — resolved (if used)

### Trade-Off

The studio prioritizes **data correctness** over perfect formatting. A small amount of reformatting is acceptable to ensure YAML remains valid and comments don't get corrupted.

**Best Practice:** For complex hand-formatted YAML, avoid inline comments and let the visual editor manage formatting.

---

## Known Limitations

### Concurrent Editing

❌ **Not supported**  
Only one user can edit a route file at a time. No multi-user conflict resolution.

### Hosted/Shared Instance

❌ **Not supported**  
Studio is local-only. Running a shared studio requires additional infrastructure (not in scope).

### Visual JSONata Builder

❌ **Not supported**  
JSONata must be written as text expressions (no point-and-click builder). See [JSONata Code Editor](#jsonata-code-editor) for syntax help.

### Custom Step Types

❌ **Partially supported**  
Unknown step types render as text. Add schema entries for full support.

### Large Routes

⚠️ **Performance**  
Routes with 100+ steps may experience lag in canvas rendering. Workaround: split into multiple files (not recommended).

---

## FAQ

### Q: Can I edit the YAML directly?

A: Currently, the visual editor is the primary interface. A raw YAML editor is planned for future releases.

### Q: What if I make a mistake?

A: The validation panel shows errors before saving. If you save invalid YAML:
1. Check error messages in the panel
2. Fix the field or expression
3. Save again
4. No file changes are made until validation passes

### Q: How do I add a new step?

A: Currently, you must edit the YAML directly (or use a future raw YAML editor):
1. Stop the studio (or it reloads automatically)
2. Edit `domains/payments/order-payment.yaml` in your text editor
3. Add a new step object to the `routes.*.steps` array
4. Save the file → studio auto-reloads and shows the new step

### Q: Can I use the same step name twice?

A: No. Each step in a route must have a unique name. The validation panel catches this and shows: `DUPLICATE_STEP`.

### Q: What's the difference between Source and Sink?

A: **Source** (input) reads data into the route. **Sink** (output) writes data out. Each route has one source and one or more sinks.

### Q: Can I have multiple routes in one file?

A: Yes. The studio supports multi-route YAML files. Each route is listed in the dropdown menu (when implemented).

### Q: What if validation fails but I know the route is correct?

A: Check the error message carefully. Common causes:
- Missing required schema fields
- Typo in adapter type
- Misformatted JSONata
- Invalid secret reference

If the error is incorrect, file an issue with the validation logic.

### Q: Can I use environment variables in config?

A: No. DIM uses secrets (registered in the secrets store), not environment variables. Use syntax: `${SECRET:domain/name}`

### Q: How often does auto-save run?

A: Every 5 seconds (configurable). Manual save is immediate.

### Q: Can I undo/redo changes?

A: No undo yet (planned feature). Rely on version control and manual backups.

---

## Support & Feedback

- **Issues:** File a bug report or feature request in the project's issue tracker
- **Documentation:** See `docs/studio/` and `schemas/route.schema.json` for schema details
- **Community:** Reach out to the DIM team for questions

Happy editing! 🎉
