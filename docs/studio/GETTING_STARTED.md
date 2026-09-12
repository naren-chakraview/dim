# DIM Visual Route Editor — Getting Started

The **DIM Visual Route Editor** (`dimctl studio`) provides a local web-based interface for authoring, editing, and validating routes visually while preserving hand-edited YAML comments and structure.

## Installation

### From Source

Build the `dimctl` binary:

```bash
cd /home/gundu/portfolio/dim
go build -o /tmp/dimctl ./cmd/dimctl
```

### Prerequisites

- Go 1.26 or later
- Node.js 18+ (for development; not required for end users)

## Starting the Studio

Open a route file for editing:

```bash
/tmp/dimctl studio domains/payments/order-payment.yaml
```

Or open the current directory (scans all `.yaml` files in `domains/`):

```bash
/tmp/dimctl studio
```

This will:
1. Start a local HTTP server on `http://localhost:7070`
2. Automatically open your browser (if `--open=true`, the default)
3. Display all discovered routes in the domains directory

**Flags:**
- `--port` — Port to serve on (default: 7070)
- `--open` — Automatically open browser (default: true)

## Basic Workflow

### 1. Open a Route

When the studio loads, you'll see:
- **Canvas (left)** — Interactive node graph showing your route's step pipeline
- **Configuration Panel (right)** — Auto-generated form for editing the selected step
- **Validation Panel (bottom)** — Real-time feedback on route validity

### 2. Select a Step

Click any step in the canvas to select it. The configuration panel updates to show that step's options.

**Step Types:**
- **Source** (green) — Kafka, HTTP, S3, or other input
- **Steps** (yellow) — Translate, filter, log, delay, enrich, etc.
- **Sink** (blue) — File, S3, HTTP, or other output

### 3. Edit Configuration

The form auto-generates based on the route's JSON schema. No hand-coded form maintenance — edit the schema, see the UI update automatically.

**Field Types:**
- **Text** — String values (names, patterns, etc.)
- **Number** — Numeric config (timeouts, batch sizes, etc.)
- **Select** — Dropdown for enums (http, kafka, file, etc.)
- **Checkbox** — Boolean flags
- **Text Area** — Multi-line text
- **JSONata** — Expression editor (with syntax highlighting)

### 4. Write JSONata Expressions

For steps with expression fields (e.g., translate `expr`), the **JSONata Editor** opens automatically:

```
// Examples:
$payload.id
$payload.amount * 1.1
$payload.status = "active"
```

The editor provides:
- Syntax highlighting
- Auto-closing brackets
- Validation feedback (✓ valid, ⚠️ invalid)

### 5. Save Changes

**Manual Save:**
```
Click the "Save" button (or press Ctrl+S when implemented)
```

**Auto-Save:**
Every 5 seconds (when implemented), changes are automatically validated and saved.

**Validation:**
Before writing to disk, every save:
1. Validates against the route schema
2. Checks auth declarations
3. Runs `dimctl validate` (when implemented)
4. Shows errors or confirms success

## Validation Panel

Shows real-time feedback as you edit:

- **✓ Valid** — Route passes all checks
- **⚠️ Error** — Schema or validation failure
- **⟳ Validating** — Validation in progress

Expandable error items show:
- Error code
- Path in the route
- "How to fix" suggestion

Example errors:
- `REQUIRED_FIELD_MISSING` — Add the missing field
- `INVALID_SCHEMA` — Check YAML syntax
- `VALIDATION_FAILED` — Review error details

## Limitations

### Preserved on Round-Trip
✅ Hand-written YAML comments  
✅ Key ordering  
✅ Blank lines between sections  
✅ Most formatting (with exceptions noted below)  

### Not Preserved
❌ Comments on same line as values (e.g., `field: value # comment`)  
❌ Multi-line string formatting (will be normalized)  
❌ Arbitrary comment positioning between keys  

**Workaround:** For complex hand-formatted YAML, export to raw YAML editor (future feature), make changes, then reload the visual editor.

## Troubleshooting

### "Route file not found"
Ensure the file exists in the `domains/` directory and has a `.yaml` extension.

### "Validation failed"
Check the Validation Panel for specific errors. Most common issues:
- Missing required fields (marked with `*`)
- Invalid enum values (use the dropdown)
- Malformed JSONata expressions

### "Can't edit JSONata"
Some fields are auto-detected as JSONata (names containing "expr", "expression", or "condition"). If a field should show the Monaco editor but doesn't, the field name may not match the detection pattern.

### "Changes not saving"
Check the browser console (F12) for network errors. Ensure the backend server is running and responding to `/api/save`.

## Next Steps

- **Read the full User Guide:** `docs/studio/USER_GUIDE.md`
- **Modify the route schema:** `schemas/route.schema.json`
- **Run validation independently:** `dimctl validate domains/payments/order-payment.yaml`
- **Test your route:** `dimctl test domains/payments/order-payment.yaml`

## Architecture

- **Frontend:** React 18 + TypeScript (Vite build)
- **Backend:** Go HTTP server (embedded in `dimctl` binary)
- **DAG Visualization:** React Flow
- **Code Editor:** Monaco Editor (VS Code engine)
- **YAML Round-Trip:** gopkg.in/yaml.v3 (Node API for comment preservation)

## Support

For issues or feature requests, see `README.md` or the project's issue tracker.
