# dim — Language Reference

Complete reference for the dim configuration language.

---

## Configuration Structure

Every dim route file follows this structure:

```yaml
version: 1                    # Config format version

imports:                      # Optional: load fragments from other files
  - fragments.yaml

fragments:                    # Optional: reusable configuration blocks
  kafka-config:
    type: kafka
    # ...

functions:                    # Optional: register custom functions
  myFunction: { type: plugin, runtime: go, ref: plugins/my_function }

sources:                      # Required: message inputs
  <name>:
    type: http|file|kafka|amqp|s3|sftp|database
    # type-specific config

sinks:                        # Required: message outputs
  <name>:
    type: file|http|kafka|amqp|s3|sftp|database
    # type-specific config

routes:                       # Required: pipelines
  <name>:
    from: <source-name>
    auth: <auth-spec>
    error_path: <error-config>
    steps: [<step>, ...]
    sinks: [<sink-name>, ...]
    lineage: <lineage-config>  # Optional
```

---

## Top-Level Fields

### `version: integer`
Config format version. Currently `1`.

```yaml
version: 1
```

### `imports: [string]`
Load reusable fragments from other YAML files (relative to current file).

```yaml
imports:
  - ./fragments.yaml
  - ./adapters/kafka.yaml
  - ../shared/retention-policies.yaml
```

### `fragments: {string: object}`
Define reusable configuration blocks (used with YAML anchors & aliases).

```yaml
fragments:
  standard-retry:
    max_attempts: 3
    backoff_ms: 1000
  
  pii-sink:
    type: file
    path: ./output/pii.jsonl
```

Use in routes:
```yaml
routes:
  my-route:
    error_path:
      retry:
        <<: $fragments.standard-retry  # Merge fragment
    sinks:
      - $fragments.pii-sink            # Reference fragment
```

### `functions: {string: function-def}`
Register custom functions for use in expressions.

```yaml
functions:
  normalizePhone:
    type: plugin
    runtime: go
    ref: plugins/normalize_phone
  
  scoreRisk:
    type: wasm
    ref: plugins/risk_score.wasm
```

Use in expressions:
```yaml
- translate:
    expr: "{ phone: $normalizePhone(body.phone), risk: $scoreRisk(body) }"
```

---

## Sources

### HTTP Source

```yaml
sources:
  api:
    type: http
    port: 8080                    # default: 8080
    path: /message                # default: /message
    health_check_path: /health    # default: /health
```

Receives POST requests with JSON body.

### File Source

```yaml
sources:
  raw-data:
    type: file
    directory: ./input/           # Required: poll this directory
    pattern: "*.json"             # default: *
    poll_interval_seconds: 5      # default: 5
    deduplicate: true             # default: true (use LastModified)
```

Polls directory for new files, reads each as a message.

### Kafka Source

```yaml
sources:
  events:
    type: kafka
    brokers:                      # Required
      - kafka-1.example.com:9092
      - kafka-2.example.com:9092
    topic: my-topic               # Required
    group: my-consumer-group      # Required
    from_beginning: false         # default: false
    security:
      protocol: sasl_ssl          # optional: none, ssl, sasl_ssl
      sasl_mechanism: PLAIN       # optional: PLAIN, SCRAM-SHA-256
      sasl_username: $SECRET:KAFKA_USER
      sasl_password: $SECRET:KAFKA_PASS
```

### AMQP Source

```yaml
sources:
  queue-events:
    type: amqp
    url: amqp://guest:guest@localhost:5672/   # Required
    queue: my-queue               # Required
    auto_ack: true                # default: true (auto-acknowledge)
```

### S3 Source

```yaml
sources:
  s3-data:
    type: s3
    region: us-east-1             # Required
    bucket: my-bucket             # Required
    prefix: data/                 # optional: key prefix filter
    poll_interval_seconds: 60     # default: 60
    delete_after_read: false      # default: false
```

### SFTP Source

```yaml
sources:
  sftp-upload:
    type: sftp
    host: sftp.example.com        # Required
    port: 22                      # default: 22
    username: $SECRET:SFTP_USER
    password: $SECRET:SFTP_PASS
    # OR use key authentication:
    private_key_path: ~/.ssh/id_rsa
    directory: /uploads/          # default: /
    pattern: "*.csv"              # default: *
    poll_interval_seconds: 10     # default: 10
```

### Database Source

```yaml
sources:
  db-events:
    type: database
    connection: postgresql://user:pass@localhost/mydb  # Required
    
    # Option 1: Polling query
    query: |
      SELECT id, event_type, payload
      FROM events
      WHERE created_at > $last_watermark
      ORDER BY created_at ASC
    watermark_column: created_at  # Track progress with this column
    poll_interval_seconds: 5
    
    # Option 2: Log-based CDC (Kafka)
    # kafka_brokers: [kafka:9092]
    # cdc_topic: mysql.events
    # cdc_format: debezium    # or: maxwell
    
    # Option 3: Trigger-based CDC (with watermark)
    # trigger_table: events_changelog
    # watermark_column: id
    # poll_interval_seconds: 1
```

---

## Sinks

### File Sink

```yaml
sinks:
  output:
    type: file
    path: ./output/messages.jsonl  # Required: file path
    format: jsonl                  # default: jsonl (also: json, csv)
```

Writes messages as newline-delimited JSON (JSONL) by default.

### HTTP Sink

```yaml
sinks:
  webhook:
    type: http
    url: https://api.example.com/events  # Required
    method: POST                   # default: POST
    headers:
      Authorization: "Bearer $SECRET:API_TOKEN"
      X-Custom-Header: "value"
    timeout_seconds: 30            # default: 30
    retry_on_status: [408, 429, 500, 502, 503]  # default: [500, 502, 503]
```

### Kafka Sink

```yaml
sinks:
  events:
    type: kafka
    brokers:
      - kafka-1.example.com:9092
      - kafka-2.example.com:9092
    topic: processed-events        # Required
    key_expr: "$.customer_id"      # optional: JSONata expr for partition key
    compression: snappy            # optional: none, gzip, snappy, lz4
    acks: all                       # optional: 0, 1, all (default: 1)
    security:
      protocol: sasl_ssl
      sasl_mechanism: PLAIN
      sasl_username: $SECRET:KAFKA_USER
      sasl_password: $SECRET:KAFKA_PASS
```

### AMQP Sink

```yaml
sinks:
  events-queue:
    type: amqp
    url: amqp://guest:guest@localhost:5672/
    exchange: events               # optional: exchange to publish to
    routing_key: process.*         # optional: routing key template
    queue: my-queue                # optional: queue name (auto-create if needed)
    durable: true                  # optional: durable queue
```

### S3 Sink

```yaml
sinks:
  archive:
    type: s3
    region: us-east-1              # Required
    bucket: my-bucket              # Required
    prefix: archive/               # optional: key prefix
    key_expr: "$.timestamp + '-' + $.id"  # optional: JSONata for key
    format: jsonl                  # default: jsonl (also: json, csv)
    storage_class: STANDARD_IA     # optional: S3 storage class
```

### SFTP Sink

```yaml
sinks:
  remote-archive:
    type: sftp
    host: sftp.example.com         # Required
    port: 22
    username: $SECRET:SFTP_USER
    password: $SECRET:SFTP_PASS
    private_key_path: ~/.ssh/id_rsa
    directory: /archive/           # default: /
    filename_expr: "$.date + '-' + $.id"  # JSONata for filename
```

### Database Sink

```yaml
sinks:
  postgres-db:
    type: database
    connection: postgresql://user:pass@localhost/mydb  # Required
    table: events                  # Required
    mode: upsert                   # default: insert (or: upsert)
    id_column: id                  # for upsert: column that identifies row
    on_conflict: update            # for upsert: update or ignore
    mapping:                        # optional: column mappings
      id: $.id
      event_type: $.type
      payload: $string(body)
      timestamp: $now()
```

---

## Routes

### Route Structure

```yaml
routes:
  <route-name>:
    from: <source-name>           # Required
    auth: <auth-spec>             # Required
    error_path:                   # Required
      target: <sink-name>
      retry: <retry-config>
    steps: [<step>, ...]          # Required
    sinks: [<sink-name>, ...]     # Required
    lineage: <lineage-config>     # Optional
```

### Authentication

#### No Auth
```yaml
auth: none
```

#### Role-Based Access Control (RBAC)

```yaml
auth:
  mode: rbac
  allowRoles:
    - admin
    - processor
  denyRoles:
    - guest
```

The principal's roles come from JWT claims (decoded from `Authorization: Bearer <token>` header).

#### Attribute-Based Access Control (ABAC)

```yaml
auth:
  mode: abac
  policy: 'principal.department = "payments" AND body.amount < 1000'
```

Expression evaluated with access to:
- `principal.subject` — user ID
- `principal.roles` — array of roles
- `principal.attributes` — custom attributes from JWT
- `body` — message body
- `$now()` — current timestamp

#### Policy Decision Point (PDP)

```yaml
auth:
  mode: pdp
  endpoint: http://opa.example.com/v1/data/policy/allow
  request_template: |
    {
      "principal": principal,
      "action": "process",
      "resource": body,
      "obligations": true
    }
```

---

## Steps

### filter

Keep or drop messages based on a condition.

```yaml
- filter:
    expr: "amount > 0 AND status != 'cancelled'"
```

If expression evaluates to `false`, message is dropped (sent to error_path).

### translate

Transform message shape using JSONata.

```yaml
- translate:
    expr: "{
      id: $.message_id,
      value: $.amount * 1.1,
      timestamp: $now(),
      tags: $.labels ? $.labels : []
    }"
```

### route

Content-based routing — send to different sinks based on message content.

```yaml
- route:
    cases:
      - when: 'type = "payment"'
        target: payments-sink
      - when: 'type = "refund"'
        target: refunds-sink
      - when: 'type = "dispute"'
        target: disputes-sink
    default: other-sink
```

### authorize

Enforce authorization and optionally apply obligations (field redaction, etc.).

```yaml
- authorize:
    mode: rbac
    allowRoles: [admin, analyst]
    pdp_endpoint: http://opa.example.com/v1/data/policy/allow  # optional
```

If authorized, messages continue. If denied, sent to error_path.

Obligations from PDP (e.g., `redact_fields`) are applied to messages.

### idempotent

Deduplicate messages based on an ID.

```yaml
- idempotent:
    id_expr: "$.transaction_id"   # JSONata to extract unique ID
    ttl_minutes: 1440             # How long to track (24 hours)
```

Duplicate messages (same ID within TTL) are dropped.

### aggregate

Group and collect messages (Phase 2: M2.1).

```yaml
- aggregate:
    correlation_key: "$.customer_id"  # Group by this expression
    aggregation_fn: count             # or: sum, avg, min, max, collect
    aggregation_field: "$.amount"     # For sum/avg/min/max
    timeout_minutes: 5                # Emit group after timeout
    max_concurrent_groups: 1000       # Concurrent group limit
```

Emits a single message per group with aggregated value.

### split

Emit multiple messages from an array (Phase 2: M2.2).

```yaml
- split:
    path: "$.items"               # Array to expand
    preserve_original: false      # Include parent fields?
```

Takes 1 message with an array, emits N messages (one per array element).

---

## Error Path

Where to send failed/invalid messages.

```yaml
error_path:
  target: dead-letter-sink      # Required: sink for errors
  retry:                         # Optional: retry config
    max_attempts: 3
    backoff_ms: 1000
```

### Retry Configuration

```yaml
retry:
  max_attempts: 3               # Total attempts (including first)
  backoff_ms: 1000              # Initial backoff, doubled each retry
  max_backoff_ms: 30000         # Cap on backoff
```

Retry sequence: attempt 1 (fails) → wait 1s → attempt 2 (fails) → wait 2s → attempt 3 (fails) → deadletter

---

## Lineage & Retention

Optional lineage configuration (audit trail and retention policies).

```yaml
lineage:
  retention_policy: pci         # Static: always use this policy
  # OR:
  retention_policy: default
  retention_policy_expr: 'body.card_number ? "pci" : "default"'  # Dynamic
  
  subject_id_expr: 'principal.subject'  # Who was this message for?
  
  export:                       # Phase 2 M2.6: auto-export at purge
    format: jsonl               # default: jsonl
    s3_bucket: audit-archive
    s3_prefix: pii-purge/
```

Retention policies are defined separately (see `design/M2_6_PURGE_LOG_AUTO_EXPORT.md`).

---

## Expressions (JSONata)

All `expr` fields use [JSONata](https://jsonata.org/).

### Basic Syntax

```jsonata
// Access fields
$.name                          // root.name
$.user.email                    // root.user.email
$.items[0]                      // first item in array

// Arithmetic
$.amount * 1.1
$.total + $.tax

// String operations
$uppercase($.name)
$lowercase($.email)
$substring($.text, 0, 10)
$join($.tags, ", ")

// Conditionals
$.status = "active" ? "yes" : "no"
$.amount > 100 ? "high" : "low"

// Arrays
$.items[*]                      // all items
$.items[$.quantity > 0]         # items where quantity > 0
$.items.($.price * $.qty)       # price * qty for each

// Comparisons
$status = "active"
$.amount > 100
$.tags[*] = "urgent"

// Logical operators
$.status = "active" AND $.amount > 0
$.type = "payment" OR $.type = "refund"
NOT $.cancelled

// Dates
$now()                          # current timestamp
$fromISO8601($.date_string)    # parse ISO 8601
```

### Function Calls

```jsonata
// Built-in functions
$length($.items)
$count($.items)
$sum($.items.price)
$avg($.items.amount)
$min($.items.amount)
$max($.items.amount)

// String functions
$uppercase($.name)
$lowercase($.email)
$contains($.email, "@")
$split($.csv, ",")
$join($.items.name, ", ")
$trim($.text)

// Date functions
$now()                          # ISO 8601 timestamp
$fromISO8601($.date)
$toISO8601($.timestamp)

// Type functions
$type($.value)                  # "string", "number", "array", etc.
$string($.value)                # convert to string
$number($.value)                # convert to number

// Custom functions (if registered)
$normalizePhone($.phone)
$scoreRisk($.customer)
```

### Variables

In expressions, you have access to:

- `$.field` — Message body fields
- `principal.subject` — User ID (from JWT)
- `principal.roles` — User roles array (from JWT)
- `principal.attributes` — Custom JWT claims
- `$now()` — Current timestamp
- `$SECRET:NAME` — Secret values (from environment or secure store)
- `$PARAM:NAME` — Fragment parameters (Phase 2: M2.4)

### Secrets and Parameters

```yaml
steps:
  - translate:
      expr: "{
        secret_key: $SECRET:API_KEY,
        default_param: $PARAM:retry_count
      }"
```

Secrets never appear in logs or traces.

---

## Secrets and Configuration

### Secret References

Use `$SECRET:NAME` in any string field:

```yaml
sources:
  kafka:
    type: kafka
    brokers: [kafka:9092]
    security:
      sasl_username: $SECRET:KAFKA_USER
      sasl_password: $SECRET:KAFKA_PASS
```

Secrets come from environment variables or a secure secrets store.

### Fragment Parameters (Phase 2)

Use `$PARAM:NAME` for configurable parameters:

```yaml
fragments:
  retry-policy:
    max_attempts: $PARAM:max_attempts
    backoff_ms: $PARAM:backoff_ms

routes:
  quick-retry:
    error_path:
      retry:
        <<: $fragments.retry-policy
        # params come from route-level or fragment-level defaults
```

Override at route level or with CLI:

```bash
./dimctl run config.yaml --param max_attempts=5 --param backoff_ms=500
```

---

## Data Types

dim supports these JSON types:

| Type | Example | Use |
|---|---|---|
| string | `"hello"` | Text, names, IDs |
| number | `123`, `45.67` | Amounts, counts |
| boolean | `true`, `false` | Flags, conditions |
| null | `null` | Absence of value |
| array | `[1, 2, 3]`, `["a", "b"]` | Lists, groups |
| object | `{"id": 123}` | Structured data |

### Null Handling

```yaml
# Optional fields
optional_field: null

# Default values in expressions
$.field ? $.field : "default"
```

---

## Comments

Use `#` for comments (YAML standard):

```yaml
sources:
  api:
    type: http
    # This endpoint handles order events
    port: 8080

routes:
  process-orders:
    from: api
    # Require admin role for sensitive operations
    auth:
      mode: rbac
      allowRoles: [admin]
    steps: [...]
```

---

## Validation

Validate a route file before running:

```bash
./dimctl validate my-route.yaml
```

This checks:
- YAML syntax
- Required fields
- Valid field names and types
- Source and sink existence
- Expression syntax (JSONata)
- Schema conformance

---

## See Also

- [docs/GETTING_STARTED.md](GETTING_STARTED.md) — Quick start guide
- [docs/USE_CASES.md](USE_CASES.md) — Real-world examples
- [docs/CLI_REFERENCE.md](CLI_REFERENCE.md) — Command reference
- [JSONata Documentation](https://docs.jsonata.org/)
- [OKF.md](../OKF.md) — Architectural decisions and patterns
