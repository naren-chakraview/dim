# dim — CLI Reference

Complete reference for `dimctl` (and `dimd`) command-line tools.

---

## Table of Contents

1. [Installation & Building](#installation--building)
2. [dimctl Commands](#dimctl-commands)
3. [dimd Daemon](#dimd-daemon)
4. [Exit Codes](#exit-codes)
5. [Environment Variables](#environment-variables)
6. [Examples](#examples)

---

## Installation & Building

### Build the CLI tool
```bash
go build ./cmd/dimctl -o dimctl
```

### Build the daemon
```bash
go build ./cmd/dimd -o dimd
```

### Verify installation
```bash
./dimctl --version
./dimctl --help
```

---

## dimctl Commands

### `dimctl validate`

Validate a route configuration file.

```bash
dimctl validate <config-file>
```

**Options:**
- `<config-file>` — Path to YAML route file
- `--strict` — Fail on warnings (default: false)
- `--schema <path>` — Custom schema file (default: built-in)

**Example:**
```bash
./dimctl validate my-route.yaml
./dimctl validate config.yaml --strict
```

**Output:** ✓ Success or ✗ Error with details

---

### `dimctl run`

Run a route configuration.

```bash
dimctl run <config-file> [options]
```

**Options:**
- `<config-file>` — Path to YAML route file (required)
- `--daemon` — Run in background (use `dimd` instead for production)
- `--port <int>` — HTTP port for sources/admin (default: 8080)
- `--admin-port <int>` — Admin/metrics port (default: 8081)
- `--log-level <level>` — Log level: debug, info, warn, error (default: info)
- `--param <key>=<value>` — Set fragment parameter (repeatable)
- `--watch` — Auto-reload on config changes (default: true)
- `--tracing-endpoint <url>` — OTLP endpoint for traces (e.g., localhost:4317)
- `--metrics-endpoint <url>` — Prometheus metrics port (default: :9090)

**Example:**
```bash
# Basic run
./dimctl run my-route.yaml

# With custom parameters
./dimctl run config.yaml --param max_retries=5 --param timeout=30

# With tracing
./dimctl run config.yaml --tracing-endpoint localhost:4317

# With debugging
./dimctl run config.yaml --log-level debug
```

**Output:**
- Logs to stdout
- Metrics available at `http://localhost:8081/metrics`
- Routes available at `http://localhost:8081/debug/routes`
- Traces exported to configured endpoint

**Signals:**
- `SIGTERM` or `SIGINT` — Graceful shutdown (drain in-flight messages)
- `SIGHUP` — Hot reload (reload config, transition new generation)

---

### `dimctl test`

Run route fixture tests.

```bash
dimctl test <fixtures-dir> [options]
```

**Options:**
- `<fixtures-dir>` — Directory containing fixture YAML files
- `--verbose` — Show detailed test output
- `--stop-on-first-failure` — Exit after first failure

**Fixture Format:**

```yaml
name: test-filter-positive-amounts
route: process-orders.yaml
input:
  - amount: 100
  - amount: -50
  - amount: 200
expectations:
  - amount: 100
  - amount: 200
```

**Example:**
```bash
./dimctl test examples/fixtures/
./dimctl test test/fixtures/ --verbose
```

**Output:** Summary of passed/failed tests

---

### `dimctl validate-contract`

Validate a message against a JSON Schema contract.

```bash
dimctl validate-contract <contract-file> <message-file>
```

**Options:**
- `<contract-file>` — JSON Schema file
- `<message-file>` — JSONL file with messages to validate

**Example:**
```bash
./dimctl validate-contract schemas/order.schema.json orders.jsonl
```

**Output:** ✓ Valid or ✗ Violations

---

### `dimctl lineage`

Manage message lineage (audit trail).

#### `lineage query`

Query lineage records.

```bash
dimctl lineage query [options]
```

**Options:**
- `--route <name>` — Filter by route name
- `--message-id <id>` — Find specific message
- `--correlation-id <id>` — Find by correlation ID
- `--subject <id>` — Filter by subject (who sent/received it)
- `--since <date>` — Start date (RFC3339 or YYYY-MM-DD)
- `--until <date>` — End date
- `--limit <n>` — Max results (default: 1000)
- `--format <format>` — json, jsonl, csv, table (default: table)

**Example:**
```bash
# Find messages from a route
./dimctl lineage query --route payment-processing --since 2024-01-01 --limit 100

# Find a specific message
./dimctl lineage query --message-id msg-12345

# Export to CSV
./dimctl lineage query --route orders --format csv > orders.csv

# Find by correlation ID
./dimctl lineage query --correlation-id order-abc123
```

#### `lineage export`

Export lineage records.

```bash
dimctl lineage export [options]
```

**Options:**
- `--route <name>` — Export from this route (required)
- `--since <date>` — Start date
- `--until <date>` — End date
- `--format <format>` — json, jsonl, csv (default: csv)
- `--output <file>` — Output file (default: stdout)

**Example:**
```bash
./dimctl lineage export --route payment-processing --since 2024-01-01 --format csv --output payments.csv
```

#### `lineage purge`

Manually purge lineage records (beyond retention policy).

```bash
dimctl lineage purge [options]
```

**Options:**
- `--route <name>` — Purge from this route
- `--before <date>` — Delete records before this date (required)
- `--reason <text>` — Reason for purge (for audit trail)
- `--dry-run` — Show what would be deleted without deleting
- `--auto-export` — Export to S3 before deleting (if configured)

**Example:**
```bash
# Preview what would be deleted
./dimctl lineage purge --before 2023-01-01 --dry-run

# Purge with reason
./dimctl lineage purge --before 2023-01-01 --reason "GDPR right to be forgotten" --auto-export

# Verify purge evidence
./dimctl lineage query --route payment-processing --since 2023-01-01 --limit 10
```

---

### `dimctl provenance`

Reconstruct a message's journey through the system.

```bash
dimctl provenance <message-id> [options]
```

**Options:**
- `<message-id>` — Message ID to trace (required)
- `--correlation-id <id>` — Use correlation ID instead
- `--format <format>` — json, text, timeline (default: text)

**Example:**
```bash
# Trace message journey
./dimctl provenance msg-12345

# By correlation ID
./dimctl provenance --correlation-id order-abc123 --format json

# Timeline view
./dimctl provenance msg-12345 --format timeline
```

**Output:** Message flow, steps executed, errors, authorization decisions

---

### `dimctl explain`

Explain a route's structure and execution plan.

```bash
dimctl explain <config-file> [options]
```

**Options:**
- `<config-file>` — Route configuration file
- `--route <name>` — Explain specific route (if multiple defined)
- `--format <format>` — text, json, dot (dot for Graphviz)

**Example:**
```bash
# Text explanation
./dimctl explain my-route.yaml

# JSON output for tools
./dimctl explain config.yaml --format json > route.json

# Graphviz DAG visualization
./dimctl explain config.yaml --format dot | dot -Tpng > route.png
```

**Output:** DAG structure with source → steps → sink

---

### `dimctl trace`

Stream OpenTelemetry traces in real-time.

```bash
dimctl trace tail [options]
```

**Options:**
- `--route <name>` — Filter by route name
- `--service <name>` — Filter by service name
- `--min-duration <ms>` — Min span duration
- `--max-duration <ms>` — Max span duration
- `--status <status>` — Filter by span status (ok, error)
- `--follow` — Keep streaming (like `tail -f`)
- `--format <format>` — text, json, compact (default: text)
- `--limit <n>` — Max spans to show (default: 100)

**Example:**
```bash
# Stream traces for a route
./dimctl trace tail --route payment-processing

# Follow mode (continuous)
./dimctl trace tail --follow --route orders

# Find slow spans (>1s)
./dimctl trace tail --min-duration 1000

# JSON for parsing
./dimctl trace tail --format json | jq '.spans[] | select(.duration > 500)'

# Error spans only
./dimctl trace tail --status error
```

---

### `dimctl replay`

Replay messages from lineage store.

```bash
dimctl replay <message-file> [options]
```

**Options:**
- `<message-file>` — JSONL file with messages (from `lineage export`)
- `--route <name>` — Which route to replay through
- `--source <name>` — Which source to inject into
- `--dry-run` — Validate without sending
- `--rate-limit <msg/s>` — Messages per second (default: unlimited)
- `--parallel <n>` — Parallel workers (default: 1)

**Example:**
```bash
# Export messages from a date range
./dimctl lineage export --route payment-processing --since 2024-01-01 --until 2024-01-02 --format jsonl > daily-payments.jsonl

# Replay with dry-run
./dimctl replay daily-payments.jsonl --route payment-processing --dry-run

# Actual replay with rate limiting
./dimctl replay daily-payments.jsonl --route payment-processing --rate-limit 100
```

---

### `dimctl stats`

Show real-time statistics.

```bash
dimctl stats [options]
```

**Options:**
- `--endpoint <url>` — Admin endpoint (default: localhost:8081)
- `--route <name>` — Filter by route
- `--interval <s>` — Update interval (default: 1)
- `--follow` — Continuous mode

**Example:**
```bash
# One-time stats
./dimctl stats

# Continuous monitoring
./dimctl stats --follow --interval 2

# Specific route
./dimctl stats --route payment-processing --follow
```

**Output:** Messages processed, latency, error rate, throughput

---

## dimd Daemon

The `dimd` daemon runs routes in the background with process management.

### Build and run

```bash
go build ./cmd/dimd -o dimd
./dimd --config routes.yaml
```

### Options

- `--config <file>` — Route configuration (required)
- `--port <int>` — HTTP port (default: 8080)
- `--admin-port <int>` — Admin/metrics port (default: 8081)
- `--log-level <level>` — debug, info, warn, error (default: info)
- `--log-file <path>` — Log to file (default: stdout)
- `--pid-file <path>` — Write PID file for process management
- `--health-check-interval <s>` — Health check frequency (default: 30)

### Running as a service

**systemd:**

```ini
[Unit]
Description=dim Integration Middleware
After=network.target

[Service]
Type=simple
User=dim
WorkingDirectory=/opt/dim
ExecStart=/opt/dim/dimd --config /etc/dim/routes.yaml --log-file /var/log/dim/dimd.log
Restart=on-failure
RestartSec=10

[Install]
WantedBy=multi-user.target
```

Install and start:
```bash
sudo systemctl enable dimd
sudo systemctl start dimd
sudo systemctl status dimd
```

### Management

```bash
# Check status
./dimctl stats --endpoint localhost:8081

# Reload config (hot reload)
kill -HUP $(cat /var/run/dimd.pid)

# Graceful shutdown
kill -TERM $(cat /var/run/dimd.pid)
```

---

## Exit Codes

| Code | Meaning |
|---|---|
| 0 | Success |
| 1 | General error (validation, runtime) |
| 2 | Configuration error (invalid YAML, missing files) |
| 3 | Connection error (can't reach broker, database, etc.) |
| 4 | Permission error (auth failure, access denied) |
| 5 | Resource exhausted (memory, disk space) |

---

## Environment Variables

### Secrets

Secrets referenced as `$SECRET:NAME` are resolved from environment:

```bash
export KAFKA_USER=alice
export KAFKA_PASS=secret123
./dimctl run config.yaml
```

In config:
```yaml
sources:
  kafka:
    type: kafka
    security:
      sasl_username: $SECRET:KAFKA_USER
      sasl_password: $SECRET:KAFKA_PASS
```

### Configuration

- `DIM_LOG_LEVEL` — Override log level (debug, info, warn, error)
- `DIM_METRICS_PORT` — Prometheus metrics port (default: 9090)
- `DIM_TRACES_ENDPOINT` — OTLP endpoint for traces
- `DIM_LINEAGE_DB` — SQLite DB path for lineage (default: ./lineage.db)
- `DIM_WATCH_CONFIG` — Auto-reload on config change (default: true)

### Example

```bash
export DIM_LOG_LEVEL=debug
export DIM_TRACES_ENDPOINT=localhost:4317
./dimctl run my-route.yaml
```

---

## Examples

### Example 1: Validate and run a route

```bash
# Validate first
./dimctl validate payment-route.yaml

# Run the route
./dimctl run payment-route.yaml --log-level info

# In another terminal, send test message
curl -X POST http://localhost:8080/message \
  -H "Content-Type: application/json" \
  -d '{"order_id":"o123","amount":99.99}'

# View stats
./dimctl stats --route payment-processing

# View traces
./dimctl trace tail --route payment-processing
```

### Example 2: Export and replay

```bash
# Export messages from last week
./dimctl lineage export \
  --route order-processor \
  --since 2024-01-01 \
  --until 2024-01-07 \
  --format jsonl > week-orders.jsonl

# Replay at 10 msg/s with 5 workers
./dimctl replay week-orders.jsonl \
  --route order-processor \
  --rate-limit 10 \
  --parallel 5
```

### Example 3: Trace a message

```bash
# Send a test message and capture its ID
ID=$(curl -s -X POST http://localhost:8080/message \
  -H "Content-Type: application/json" \
  -d '{"customer":"alice"}' | jq -r '.message_id')

# Get full provenance
./dimctl provenance "$ID" --format timeline

# Query lineage
./dimctl lineage query --message-id "$ID" --format json
```

### Example 4: Monitor performance

```bash
# Continuous monitoring with 2s updates
./dimctl stats --follow --interval 2

# Or watch traces in real-time
./dimctl trace tail --follow --route payment-processing
```

---

## Tips & Tricks

### Scripting

Export lineage to CSV for analysis:
```bash
./dimctl lineage export --route payments --format csv > payments.csv
wc -l payments.csv  # Count messages
```

Extract error messages:
```bash
./dimctl lineage query --status error --format json | jq '.[] | .error_message'
```

### Debugging

Find slow messages:
```bash
./dimctl trace tail --min-duration 1000  # Spans taking >1s
```

Check authorization decisions:
```bash
./dimctl lineage query --format json | jq '.[] | select(.authorization_decision == "deny")'
```

Find messages by customer:
```bash
./dimctl lineage query --subject alice --format table
```

### Troubleshooting

Check if a config is valid:
```bash
./dimctl validate config.yaml || echo "Invalid!"
```

Explain the DAG for a route:
```bash
./dimctl explain config.yaml --format dot | dot -Tpng > route.png
open route.png
```

---

## See Also

- `./dimctl --help` — Full help text
- `./dimctl <command> --help` — Help for specific command
- [docs/LANGUAGE_REFERENCE.md](LANGUAGE_REFERENCE.md) — Configuration reference
- [docs/GETTING_STARTED.md](GETTING_STARTED.md) — Quick start guide
