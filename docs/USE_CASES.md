# dim — Use Cases & Examples

Real-world patterns and solutions using dim.

---

## Table of Contents

1. [E-commerce Order Processing](#e-commerce-order-processing)
2. [Financial Transaction Compliance](#financial-transaction-compliance)
3. [Multi-Source Data Lake Ingestion](#multi-source-data-lake-ingestion)
4. [Real-Time Notifications](#real-time-notifications)
5. [CDC and Database Synchronization](#cdc-and-database-synchronization)
6. [API Rate Limiting and Throttling](#api-rate-limiting-and-throttling)
7. [Log Aggregation and Filtering](#log-aggregation-and-filtering)
8. [Access Control and Redaction](#access-control-and-redaction)
9. [Message Replay and Recovery](#message-replay-and-recovery)
10. [Schema Validation and Cleanup](#schema-validation-and-cleanup)

---

## E-commerce Order Processing

**Problem:** Route customer orders to different fulfillment systems (digital vs. physical) based on product type, with notifications and audit trail.

**Solution:**

```yaml
version: 1

sources:
  checkout:
    type: http
    port: 8080

sinks:
  physical-fulfillment:
    type: http
    url: https://warehouse.internal/api/orders
    headers:
      Authorization: "Bearer $SECRET:WAREHOUSE_TOKEN"
  
  digital-fulfillment:
    type: kafka
    brokers: [kafka-1:9092, kafka-2:9092]
    topic: digital-orders
  
  notifications:
    type: http
    url: https://notifications.example.com/send
  
  audit:
    type: file
    path: ./audit/orders.jsonl
  
  errors:
    type: file
    path: ./errors/orders.jsonl

routes:
  process-orders:
    from: checkout
    auth:
      mode: rbac
      allowRoles: [checkout-service]
    error_path:
      target: errors
      retry:
        max_attempts: 3
        backoff_ms: 500
    
    steps:
      # Validate order has required fields
      - filter:
          expr: "order_id AND customer_id AND items[*].sku"
      
      # Enrich with calculated fields
      - translate:
          expr: "{
            order_id: $.order_id,
            customer_id: $.customer_id,
            items: $.items,
            total: $sum($.items.($.price * $.quantity)),
            tax: $sum($.items.($.price * $.quantity)) * 0.08,
            fulfillment_type: $.items[0].type = 'digital' ? 'digital' : 'physical',
            timestamp: $now()
          }"
    
    # Route to appropriate fulfillment system
    steps:
      - route:
          cases:
            - when: "fulfillment_type = 'digital'"
              target: digital-fulfillment
            - when: "fulfillment_type = 'physical'"
              target: physical-fulfillment
          default: errors
    
    # Also send to audit
    sinks:
      - audit
      - notifications
    
    lineage:
      retention_policy: commerce
      subject_id_expr: "$.customer_id"
```

---

## Financial Transaction Compliance

**Problem:** Process financial transactions with PCI compliance (PII redaction), role-based access control, and immutable audit trail.

**Solution:**

```yaml
version: 1

sources:
  payments:
    type: kafka
    brokers: [kafka-1:9092]
    topic: payment-requests
    group: compliance-processor

sinks:
  processor:
    type: http
    url: https://payment-processor.internal/api/charge
    headers:
      Authorization: "Bearer $SECRET:PROCESSOR_TOKEN"
  
  audit:
    type: s3
    region: us-east-1
    bucket: compliance-audit
    prefix: payments/
  
  errors:
    type: file
    path: ./audit/payment-errors.jsonl

routes:
  process-payments:
    from: payments
    
    # Strict authorization: only admins and compliance officers
    auth:
      mode: rbac
      allowRoles: [admin, compliance-officer]
      denyRoles: [guest, read-only]
    
    error_path:
      target: errors
      retry:
        max_attempts: 2
        backoff_ms: 1000
    
    steps:
      # Validate amount range
      - filter:
          expr: "amount > 0 AND amount < 1000000"
      
      # Check for suspicious patterns (fraud detection)
      - authorize:
          mode: pdp
          endpoint: http://fraud-check.internal/v1/evaluate
          request_template: |
            {
              "amount": $.amount,
              "customer_id": $.customer_id,
              "merchant_id": $.merchant_id,
              "timestamp": $.timestamp
            }
      
      # Transform for processor (ensure required fields)
      - translate:
          expr: "{
            transaction_id: $.payment_id,
            amount_cents: $.amount * 100,
            currency: $.currency,
            merchant_id: $.merchant_id,
            cardholder_name: $.cardholder_name,
            last_four: $substring($.card_number, -4),
            expiry: $.expiry
          }"
    
    sinks:
      - processor
      - audit
    
    # PCI compliance: keep records for 7 years, auto-redact after 5 years
    lineage:
      retention_policy: pci
      retention_policy_expr: 'body.amount > 10000 ? "pci-extended" : "pci"'
      subject_id_expr: "$.customer_id"
      export:
        s3_bucket: compliance-exports
        s3_prefix: payment-purge/
```

---

## Multi-Source Data Lake Ingestion

**Problem:** Ingest data from multiple sources (APIs, databases, S3) into a data lake with deduplication and schema validation.

**Solution:**

```yaml
version: 1

sources:
  api-events:
    type: http
    port: 8080
  
  database-poll:
    type: database
    connection: postgresql://user:pass@db.internal/analytics
    query: |
      SELECT id, event_type, payload, created_at
      FROM events
      WHERE created_at > $last_watermark
      ORDER BY created_at ASC
    watermark_column: created_at
    poll_interval_seconds: 10
  
  s3-raw:
    type: s3
    region: us-east-1
    bucket: raw-data
    prefix: events/
    poll_interval_seconds: 60

sinks:
  data-lake:
    type: database
    connection: postgresql://user:pass@db.internal/datalake
    table: events_dedup
    mode: upsert
    id_column: event_id
    on_conflict: update
  
  schema-violations:
    type: file
    path: ./schema-errors/violations.jsonl
  
  errors:
    type: file
    path: ./errors/ingestion.jsonl

routes:
  ingest-events:
    from: api-events
    auth: none
    error_path:
      target: errors
      retry:
        max_attempts: 2
    
    steps:
      # Deduplicate by event ID (24-hour TTL)
      - idempotent:
          id_expr: "$.event_id"
          ttl_minutes: 1440
      
      # Validate schema: event_id, event_type, payload must exist
      - filter:
          expr: "event_id AND event_type AND payload"
      
      # Normalize timestamp to ISO 8601
      - translate:
          expr: "{
            event_id: $.event_id,
            event_type: $.event_type,
            payload: $.payload,
            source: 'api',
            ingested_at: $now(),
            ingested_by: principal.subject
          }"
    
    sinks:
      - data-lake
    
    lineage:
      retention_policy: general
      subject_id_expr: "principal.subject"

  ingest-db-changes:
    from: database-poll
    auth: none
    error_path:
      target: errors
    
    steps:
      - idempotent:
          id_expr: "$.id"
          ttl_minutes: 1440
      
      - translate:
          expr: "{
            event_id: 'db-' + $.id,
            event_type: $.event_type,
            payload: $.payload,
            source: 'database',
            ingested_at: $now()
          }"
    
    sinks:
      - data-lake

  ingest-s3-files:
    from: s3-raw
    auth: none
    error_path:
      target: errors
    
    steps:
      - filter:
          expr: "$substring(source, -5) = '.json'"
      
      - translate:
          expr: "{
            event_id: 's3-' + $.id,
            event_type: $.type,
            payload: $,
            source: 's3',
            ingested_at: $now()
          }"
    
    sinks:
      - data-lake
```

---

## Real-Time Notifications

**Problem:** Send notifications to customers based on their order status, with preference management (email, SMS, push).

**Solution:**

```yaml
version: 1

sources:
  order-events:
    type: kafka
    brokers: [kafka:9092]
    topic: order-status-changes

sinks:
  email-service:
    type: http
    url: https://email.internal/send
  
  sms-service:
    type: http
    url: https://sms.internal/send
  
  push-service:
    type: http
    url: https://notifications.internal/send
  
  notification-log:
    type: file
    path: ./audit/notifications.jsonl

routes:
  notify-customers:
    from: order-events
    auth: none
    error_path:
      target: notification-log
      retry:
        max_attempts: 3
        backoff_ms: 2000
    
    steps:
      # Fetch customer preferences from cache/DB
      # (In real implementation, this would be a custom function)
      
      - translate:
          expr: "{
            order_id: $.order_id,
            customer_id: $.customer_id,
            status: $.status,
            timestamp: $now()
          }"
      
      # Route to notification channels based on preference
      - route:
          cases:
            - when: 'status = "shipped" AND preferences.notify_email'
              target: email-service
            - when: 'status = "shipped" AND preferences.notify_sms'
              target: sms-service
            - when: 'status = "shipped" AND preferences.notify_push'
              target: push-service
          default: notification-log
    
    sinks:
      - notification-log
    
    lineage:
      retention_policy: marketing
      subject_id_expr: "$.customer_id"
```

---

## CDC and Database Synchronization

**Problem:** Keep a read replica in sync using Change Data Capture (CDC) from PostgreSQL.

**Solution:**

```yaml
version: 1

sources:
  # Option 1: Log-based CDC via Debezium
  db-changes:
    type: database
    connection: postgresql://user:pass@db.internal/prod
    
    cdc_source: log-based
    kafka_brokers: [kafka:9092]
    cdc_topic: debezium.public.customers
    cdc_format: debezium
    
    # Filter: only track customers table
    include_tables: [public.customers]

sinks:
  replica-db:
    type: database
    connection: postgresql://user:pass@replica.internal/prod
    table: customers
    mode: upsert
    id_column: id
    on_conflict: update

routes:
  sync-customers:
    from: db-changes
    auth: none
    error_path:
      target: replica-db  # Can also send to DLQ topic
    
    steps:
      # Parse Debezium event
      - translate:
          expr: "{
            id: $.after.id,
            name: $.after.name,
            email: $.after.email,
            created_at: $.after.created_at,
            updated_at: $.ts_ms,
            operation: $.op  # c: create, u: update, d: delete
          }"
      
      # Skip delete operations (soft delete pattern)
      - filter:
          expr: "operation != 'd'"
    
    sinks:
      - replica-db
```

---

## API Rate Limiting and Throttling

**Problem:** Throttle inbound requests by customer to enforce rate limits.

**Solution:**

```yaml
version: 1

sources:
  api:
    type: http
    port: 8080

sinks:
  fast-lane:
    type: file
    path: ./output/fast.jsonl
  
  slow-lane:
    type: file
    path: ./output/slow.jsonl
  
  rejected:
    type: file
    path: ./output/rejected.jsonl

routes:
  rate-limit:
    from: api
    auth: none
    error_path:
      target: rejected
    
    steps:
      # Aggregate by customer to count requests
      - aggregate:
          correlation_key: "$.customer_id"
          aggregation_fn: count
          timeout_minutes: 1  # 1-minute window
          max_concurrent_groups: 10000
      
      # Route based on count
      - route:
          cases:
            - when: "aggregation_result < 100"  # Under limit
              target: fast-lane
            - when: "aggregation_result >= 100 AND aggregation_result < 200"
              target: slow-lane
          default: rejected  # Over hard limit
    
    sinks:
      - fast-lane
      - slow-lane
```

---

## Log Aggregation and Filtering

**Problem:** Aggregate logs from multiple services, filter by severity, and store in different buckets.

**Solution:**

```yaml
version: 1

sources:
  syslog:
    type: kafka
    brokers: [kafka:9092]
    topic: logs
    group: log-processor

sinks:
  errors:
    type: s3
    bucket: logs-archive
    prefix: errors/
  
  warnings:
    type: s3
    bucket: logs-archive
    prefix: warnings/
  
  info:
    type: s3
    bucket: logs-archive
    prefix: info/
  
  alerting:
    type: http
    url: https://alerts.example.com/webhook

routes:
  process-logs:
    from: syslog
    auth: none
    error_path:
      target: info
    
    steps:
      # Parse and normalize log format
      - translate:
          expr: "{
            service: $.service,
            level: $uppercase($.level),
            message: $.msg,
            timestamp: $.ts ? $.ts : $now(),
            tags: $.tags ? $.tags : []
          }"
      
      # Route by severity
      - route:
          cases:
            - when: "level = 'ERROR' OR level = 'CRITICAL'"
              target: errors
            - when: "level = 'WARN'"
              target: warnings
            - when: "level = 'INFO' OR level = 'DEBUG'"
              target: info
          default: info
      
      # Alert on critical errors
      - filter:
          expr: "level = 'CRITICAL'"  # Only alerts go to alerting
    
    sinks:
      - errors
      - warnings
      - info
      - alerting
```

---

## Access Control and Redaction

**Problem:** Enforce role-based access and automatically redact sensitive fields (PII, secrets) from data based on user role.

**Solution:**

```yaml
version: 1

sources:
  customer-data:
    type: kafka
    brokers: [kafka:9092]
    topic: customer-events
    group: data-processor

sinks:
  data-warehouse:
    type: database
    connection: postgresql://localhost/dw
    table: customer_events
  
  errors:
    type: file
    path: ./errors/access-denied.jsonl

routes:
  secure-customer-data:
    from: customer-data
    
    # Only analysts and admins can access
    auth:
      mode: rbac
      allowRoles: [admin, analyst]
    
    error_path:
      target: errors
    
    steps:
      # Check PDP for obligations (e.g., redact_fields)
      - authorize:
          mode: pdp
          endpoint: http://opa.internal/v1/data/policy/access
          request_template: |
            {
              "principal": principal,
              "resource": "customer_data",
              "action": "read"
            }
      
      # Translate and optionally redact based on role
      - translate:
          expr: "{
            customer_id: $.customer_id,
            email: principal.roles[*] = 'admin' ? $.email : '***@***.***',
            phone: principal.roles[*] = 'admin' ? $.phone : '***-****',
            ssn: principal.roles[*] = 'admin' ? $.ssn : '***-**-****',
            address: $.address,
            data_classification: 'PII',
            accessed_by: principal.subject,
            accessed_at: $now()
          }"
    
    sinks:
      - data-warehouse
    
    lineage:
      retention_policy: pii
      subject_id_expr: "$.customer_id"
      export:
        s3_bucket: pii-archive
        s3_prefix: redacted-access/
```

---

## Message Replay and Recovery

**Problem:** Replay messages from lineage store when downstream system recovers from outage.

**Solution:**

```yaml
version: 1

sources:
  api:
    type: http
    port: 8080

sinks:
  primary:
    type: http
    url: https://primary.internal/events
  
  replay-log:
    type: file
    path: ./logs/replay.jsonl

routes:
  process-events:
    from: api
    auth:
      mode: rbac
      allowRoles: [service]
    
    error_path:
      target: primary  # Deadletter as backup
      retry:
        max_attempts: 3
        backoff_ms: 5000
    
    steps:
      - translate:
          expr: "{
            id: $.event_id,
            type: $.type,
            data: $.data,
            timestamp: $now()
          }"
    
    sinks:
      - primary
      - replay-log
    
    lineage:
      retention_policy: default

# Replay via CLI when system recovers:
# ./dimctl lineage query --since 2024-01-01 --until 2024-01-02 --format jsonl | \
#   ./dimctl replay --route process-events
```

---

## Schema Validation and Cleanup

**Problem:** Validate incoming data against a JSON Schema and clean/transform invalid data.

**Solution:**

```yaml
version: 1

sources:
  raw:
    type: kafka
    brokers: [kafka:9092]
    topic: raw-events

sinks:
  valid:
    type: kafka
    brokers: [kafka:9092]
    topic: validated-events
  
  invalid:
    type: file
    path: ./errors/schema-violations.jsonl

routes:
  validate-schema:
    from: raw
    auth: none
    error_path:
      target: invalid
    
    steps:
      # Contract validation with JSON Schema
      - filter:
          expr: "id AND type AND timestamp"  # Required fields
      
      # Type validation (filter out wrong types)
      - filter:
          expr: "$type(id) = 'string' AND $type(amount) = 'number'"
      
      # Range validation
      - filter:
          expr: "amount >= 0 AND amount <= 999999"
      
      # Clean and standardize
      - translate:
          expr: "{
            id: $.id,
            type: $.type,
            amount: $.amount,
            currency: $.currency ? $.currency : 'USD',
            timestamp: $fromISO8601($.timestamp),
            normalized_at: $now()
          }"
    
    sinks:
      - valid
    
    lineage:
      retention_policy: compliance
```

---

## See Also

- [docs/GETTING_STARTED.md](GETTING_STARTED.md) — Quick start guide
- [docs/LANGUAGE_REFERENCE.md](LANGUAGE_REFERENCE.md) — Full configuration reference
- [examples/](../examples/) — Working YAML examples in the repository
- [design/eip-middleware-design.md](../design/eip-middleware-design.md) — Architecture and EIP patterns
