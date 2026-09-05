# dim — Getting Started

Welcome to **dim**, a configuration-driven integration middleware. This guide will help you build your first route in 5 minutes.

---

## What is dim?

dim routes, transforms, and audits messages between systems. You define pipelines in YAML—no code required.

**Use dim to:**
- Route messages between APIs, queues, and databases
- Transform message payloads with expressions
- Enforce access control on every message
- Keep an audit trail of who accessed what
- Monitor message flow in real-time

---

## Installation

### 1. Clone the repository
```bash
git clone https://github.com/naren-chakraview/dim.git
cd dim
```

### 2. Build the CLI tool
```bash
go build ./cmd/dimctl -o dimctl
```

### 3. Verify the build
```bash
./dimctl --help
```

---

## Your First Route (5 minutes)

### Step 1: Create a route file

Create `my-first-route.yaml`:

```yaml
version: 1

sources:
  api:
    type: http
    port: 8080

sinks:
  output:
    type: file
    path: ./output/messages.jsonl
  errors:
    type: file
    path: ./output/errors.jsonl

routes:
  process-orders:
    from: api
    auth: none
    error_path:
      target: errors
    steps:
      - filter:
          expr: "amount > 0"
      - translate:
          expr: "{ order_id: id, total: amount * 1.1, status: 'pending' }"
    sinks:
      - output
```

### Step 2: Validate the route

```bash
./dimctl validate my-first-route.yaml
```

You should see: `✓ Route is valid`

### Step 3: Run the route

```bash
./dimctl run my-first-route.yaml
```

You'll see: `Starting route: process-orders on http://localhost:8080`

### Step 4: Send a message

In another terminal:

```bash
curl -X POST http://localhost:8080/message \
  -H "Content-Type: application/json" \
  -d '{"id":"order-123","amount":100,"customer":"alice"}'
```

### Step 5: Check the output

```bash
cat output/messages.jsonl
```

You should see:
```json
{"order_id":"order-123","total":110,"status":"pending"}
```

---

## Understanding the Configuration

Every dim route has three parts:

### 1. **Sources** — Where messages come from
```yaml
sources:
  api:
    type: http      # or: file, kafka, amqp, s3, sftp, database
    port: 8080
```

### 2. **Sinks** — Where messages go
```yaml
sinks:
  database:
    type: database  # or: file, kafka, amqp, s3, http, sftp
    connection: postgresql://localhost/mydb
    table: events
```

### 3. **Routes** — The pipeline connecting source → steps → sink(s)
```yaml
routes:
  my-route:
    from: api                    # source
    auth: none                   # authorization (required)
    error_path:
      target: errors             # where failed messages go
    steps:                        # ordered transformations
      - filter: ...
      - translate: ...
      - authorize: ...
    sinks:
      - database
      - monitoring
```

---

## Common Patterns

### Pattern 1: Filter and Transform

```yaml
routes:
  clean-data:
    from: raw-stream
    auth: none
    error_path:
      target: rejected
    steps:
      # Keep only messages where amount > 0
      - filter:
          expr: "amount > 0"
      
      # Transform to output shape
      - translate:
          expr: "{
            transaction_id: $.id,
            amount_cents: $.amount * 100,
            currency: $.currency,
            timestamp: $now()
          }"
    sinks:
      - clean-output
```

### Pattern 2: Conditional Routing (Content-Based)

```yaml
routes:
  smart-router:
    from: events
    auth: none
    error_path:
      target: errors
    steps:
      - route:
          cases:
            - when: 'type = "payment"'
              target: payments
            - when: 'type = "shipment"'
              target: shipping
            - when: 'type = "refund"'
              target: refunds
          default: other
    sinks:
      - payments
      - shipping
      - refunds
      - other
```

### Pattern 3: Deduplication

```yaml
routes:
  deduplicate:
    from: messy-source
    auth: none
    error_path:
      target: errors
    steps:
      - idempotent:
          id_expr: "$.transaction_id"
          ttl_minutes: 1440    # 24 hours
    sinks:
      - clean-output
```

### Pattern 4: Access Control

```yaml
routes:
  sensitive-data:
    from: api
    auth:
      mode: rbac
      allowRoles: [admin, analyst]
    steps:
      - translate:
          expr: "{ data: body }"
    sinks:
      - secure-output
```

See [docs/USE_CASES.md](USE_CASES.md) for more real-world examples.

---

## Available Sources and Sinks

### Sources (message inputs)
- **http** — HTTP POST endpoint
- **file** — Poll a directory for files
- **kafka** — Kafka topic consumer
- **amqp** — RabbitMQ / AMQP queue
- **s3** — Poll S3 bucket
- **sftp** — Poll SFTP directory
- **database** — Query or poll database (with CDC support)

### Sinks (message outputs)
- **file** — Write to file (JSONL format)
- **http** — POST to HTTP endpoint
- **kafka** — Kafka topic producer
- **amqp** — RabbitMQ / AMQP queue
- **s3** — Write to S3 bucket
- **sftp** — Write to SFTP server
- **database** — Insert/upsert rows

---

## Steps Available

### 1. **filter** — Keep or drop messages
```yaml
- filter:
    expr: "amount > 100 AND status = 'active'"
```

### 2. **translate** — Transform message shape
```yaml
- translate:
    expr: "{
      id: $.message_id,
      value: $.amount * 1.1,
      tags: $.labels
    }"
```

### 3. **route** — Content-based routing
```yaml
- route:
    cases:
      - when: 'type = "order"'
        target: orders-sink
      - when: 'type = "invoice"'
        target: invoices-sink
    default: other-sink
```

### 4. **authorize** — Enforce access control
```yaml
- authorize:
    mode: rbac
    allowRoles: [admin, processor]
```

### 5. **aggregate** — Group and collect messages (Phase 2)
```yaml
- aggregate:
    correlation_key: "$.customer_id"
    aggregation_fn: count
    timeout_minutes: 5
```

### 6. **split** — Emit multiple messages (Phase 2)
```yaml
- split:
    path: "$.items"  # expand array to N messages
```

See [docs/LANGUAGE_REFERENCE.md](LANGUAGE_REFERENCE.md) for complete reference.

---

## Expressions (JSONata)

All transformations use [JSONata](https://jsonata.org/), a JSON query and transformation language.

**Basic examples:**

```jsonata
// Access properties
$.customer_id

// Arithmetic
$.amount * 1.1

// String functions
$uppercase($.name)

// Conditionals
$.status = "active" ? "keep" : "drop"

// Array operations
$.items.($price * $.quantity)

// Object construction
{
  id: $.order_id,
  total: $.amount * 1.1,
  tax: $.amount * 0.08
}
```

**Learning resources:**
- [JSONata Interactive Playground](https://try.jsonata.org/)
- [JSONata Documentation](https://docs.jsonata.org/)

---

## Error Handling

Every route must declare an `error_path`:

```yaml
routes:
  my-route:
    from: api
    auth: none
    error_path:
      target: dead-letter        # Where errors go
      retry:                      # Optional: retry config
        max_attempts: 3
        backoff_ms: 1000
    steps: [...]
    sinks: [...]
```

Failed messages go to the `dead-letter` sink for inspection and replay.

### Retry behavior
- Default: 3 attempts with exponential backoff (1s → 2s → 4s)
- Customize with `retry.max_attempts` and `retry.backoff_ms`
- Deadletter after max attempts exceeded

---

## Configuration Composition (DRY)

Avoid repeating sink configurations with **fragments**:

```yaml
# fragments.yaml
fragments:
  kafka-config:
    type: kafka
    brokers:
      - kafka-1.example.com:9092
      - kafka-2.example.com:9092

# routes.yaml
imports:
  - fragments.yaml

sinks:
  events:
    <<: $fragments.kafka-config
    topic: events
  logs:
    <<: $fragments.kafka-config
    topic: logs
```

---

## Next Steps

1. **Try more examples:** See [docs/USE_CASES.md](USE_CASES.md) for real-world patterns
2. **Learn the full language:** [docs/LANGUAGE_REFERENCE.md](LANGUAGE_REFERENCE.md)
3. **CLI commands:** [docs/CLI_REFERENCE.md](CLI_REFERENCE.md)
4. **Run the tests:** `go test ./...`

---

## Getting Help

- **Questions about syntax?** → [docs/LANGUAGE_REFERENCE.md](LANGUAGE_REFERENCE.md)
- **Looking for examples?** → [docs/USE_CASES.md](USE_CASES.md)
- **CLI troubleshooting?** → `./dimctl --help` or [docs/CLI_REFERENCE.md](CLI_REFERENCE.md)
- **Architecture questions?** → [README.md](../README.md) and [OKF.md](../OKF.md)
- **Contributing?** → [DEVELOPER_GUIDE.md](../DEVELOPER_GUIDE.md)
