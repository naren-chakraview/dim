# Task 6: Create a worked example (payments domain with real route)

## Files to Create

- Create: `domains/payments/DOMAIN.yaml`
- Create: `domains/payments/order-payment.yaml`
- Create: `domains/payments/order-payment.route_test.yaml`

## Interfaces

- Consumes: Governance fragments from Task 1 (`domains/governance/fragments.yaml`), validates against JSON Schema via `dimctl validate`, tests via `dimctl test`
- Produces: A real, validated domain directory with a route and tests; demonstrates the full self-service workflow

## Steps to Execute

### Step 1: Create domains/payments/ directory

```bash
mkdir -p domains/payments
```

### Step 2: Write domains/payments/DOMAIN.yaml

Create this file with domain metadata:

```yaml
# Domain metadata for the payments domain
domain:
  name: payments
  owner: alice@example.com
  slack_channel: "#payments-team"
  description: Payment processing, fraud detection, and payout orchestration
  owner_team: payments-platform
```

### Step 3: Write domains/payments/order-payment.yaml

Create the route file with this exact content:

```yaml
version: 1
imports:
  - ../governance/fragments.yaml

resources:
  connections:
    kafka-prod:
      type: kafka
      brokers: ["kafka1:9092", "kafka2:9092"]

sources:
  orders-in:
    type: kafka
    connection: kafka-prod
    topic: orders.created
    group: payments-processor
    format: json

sinks:
  payments-processed:
    type: kafka
    connection: kafka-prod
    topic: payments.processed
  
  payments-dlq:
    type: kafka
    connection: kafka-prod
    topic: payments.dlq
  
  payments-high-value:
    type: kafka
    connection: kafka-prod
    topic: payments.high-value-alert

routes:
  order-payment:
    from: orders-in
    lineage: { retention_policy: default }
    error_path:
      target: payments-dlq
      retry: { max_attempts: 5, backoff: { type: exponential, initial: 1s, max: 30s, jitter: true } }
    steps:
      - fragment: governance-baseline
      - filter: { expr: 'body.amount != null and body.order_id != null' }
      - translate:
          expr: |
            {
              "payment_id": body.order_id,
              "amount_cents": body.amount,
              "currency": body.currency ? body.currency : "USD",
              "customer_id": body.customer_id,
              "timestamp": metadata.timestamp
            }
      - idempotent:
          key: 'body.payment_id'
          store: { type: memory, ttl: 24h }
      - route:
          cases:
            - when: 'body.amount_cents > 1000000'  # > $10,000
              to: [payments-processed, payments-high-value]
          default:
            to: [payments-processed]
```

### Step 4: Write domains/payments/order-payment.route_test.yaml

Create the test file with these test cases:

```yaml
route: order-payment
cases:
  - name: low-value order payment routes normally
    input:
      body:
        order_id: "ord-001"
        amount: 500000
        currency: "USD"
        customer_id: "cust-123"
    expect:
      payments-processed: [{ payment_id: "ord-001", amount_cents: 500000, currency: "USD" }]

  - name: high-value order triggers alert
    input:
      body:
        order_id: "ord-002"
        amount: 1500000
        currency: "USD"
        customer_id: "cust-456"
    expect:
      payments-processed: [{ payment_id: "ord-002", amount_cents: 1500000, currency: "USD" }]
      payments-high-value: [{ payment_id: "ord-002", amount_cents: 1500000, currency: "USD" }]

  - name: missing amount goes to dead-letter
    input:
      body:
        order_id: "ord-003"
        customer_id: "cust-789"
    expect:
      payments-dlq: [{ error_type: "non_retryable" }]

  - name: duplicate payment is deduplicated (idempotent)
    input:
      body:
        order_id: "ord-004"
        amount: 250000
        currency: "USD"
        customer_id: "cust-999"
    expect:
      payments-processed: [{ payment_id: "ord-004", amount_cents: 250000, currency: "USD" }]
```

### Step 5: Validate the worked example locally

Run validation and testing:

```bash
# Validate the route
go run ./cmd/dimctl validate domains/payments/order-payment.yaml

# Expected output: should pass validation (no errors)

# Test the route
go run ./cmd/dimctl test -c domains/payments/order-payment.yaml domains/payments/order-payment.route_test.yaml

# Expected output: all 4 test cases should pass
```

### Step 6: Verify governance fragment import

```bash
# Check that the route imports governance fragments
grep "governance/fragments" domains/payments/order-payment.yaml

# Expected: should show "- ../governance/fragments.yaml" in the imports section
```

### Step 7: Commit all three files

```bash
git add domains/payments/
git commit -m "example: add order-payment route in payments domain (M4.1 worked example)"
```

## Success Criteria

- All three files created in `domains/payments/`: DOMAIN.yaml, order-payment.yaml, order-payment.route_test.yaml
- `dimctl validate domains/payments/order-payment.yaml` passes with no errors
- `dimctl test -c domains/payments/order-payment.yaml domains/payments/order-payment.route_test.yaml` runs all 4 test cases and they all pass
- Route imports governance fragments (grep confirms)
- Commit present in git log with message: "example: add order-payment route in payments domain (M4.1 worked example)"

## Notes

- This is a worked example demonstrating a realistic payment-processing domain
- The route shows:
  - Mandatory governance fragment import
  - Authorization via governance-baseline fragment
  - Filtering, translation, idempotency, and routing patterns
  - Multiple sinks (normal and high-value paths)
  - Proper error path with retry configuration
  - Lineage retention policy
- All test cases are realistic and show both happy path and edge cases
- This route will be used in later tasks to demonstrate the full M4.1 flow (PR→CI→merge→live)
