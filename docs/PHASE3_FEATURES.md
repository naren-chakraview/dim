# dim — Phase 3 Features Guide

Phase 3 introduces powerful new capabilities for scalability, extensibility, and multi-tenancy.

---

## Quick Overview

| Feature | Problem Solved | When to Use |
|---------|---|---|
| **Claim Check Pattern** | Large message payloads slow down processing | Handling attachments, documents, media files |
| **Plugin SDK** | Need custom logic without modifying dim | Domain-specific validation, transformation, enrichment |
| **Multi-Tenant Resource Isolation** | One overloaded tenant affects others | SaaS platforms, shared internal infrastructure |
| **Distributed Clustering** | Single instance is a bottleneck | Horizontal scaling, high availability, geographic distribution |

---

## Feature 1: Claim Check Pattern (M3.2)

### Problem

Processing large message payloads (5MB+ PDFs, videos, archives) through a pipeline is expensive:
- Network overhead: Payload travels across all steps
- Memory pressure: Message buffers hold entire payload
- Processing delay: Downstream steps must deserialize large data

### Solution: Claim Check

Store large payloads externally and pass lightweight "tickets" through the pipeline instead.

```
Original Message (5MB PDF)
        ↓
   [claim-check step]
   • Store payload in S3/external storage
   • Return lightweight ticket (100 bytes)
        ↓
Lightweight Message (with ticket)
   • Travels through pipeline (fast)
   • Only steps needing payload retrieve it
        ↓
   [claim-resolve step] (when needed)
   • Retrieve payload using ticket
   • Continue processing
```

### Configuration

#### Step 1: Define claim-check storage in route

```yaml
routes:
  order-with-attachments:
    from: api
    auth: rbac
    steps:
      - claim-check:
          store_type: s3           # or 'memory' for testing
          store_config:
            bucket: dim-payloads
            region: us-east-1
          payload_field: attachments   # Field to store
          remove_after: true           # Remove from message after storing
    sinks:
      - kafka-orders
```

#### Step 2: Later, retrieve payload when needed

```yaml
routes:
  compliance-check:
    from: kafka-orders
    auth: rbac
    steps:
      - claim-resolve:
          store_type: s3
          store_config:
            bucket: dim-payloads
            region: us-east-1
          ticket_field: _claim_check    # Where ticket is stored
          restore_field: attachments     # Restore to this field
      - translate:
          expr: '{ validated: validateAttachments(attachments), order_id: order_id }'
    sinks:
      - compliance-db
```

### Message Flow Example

```
Input message:
{
  "order_id": "ORD-2026-001",
  "customer": "ACME Corp",
  "invoice_pdf": <5 MB binary data>,
  "attachment_count": 3
}

↓ [claim-check step]

After claim-check (lightweight):
{
  "order_id": "ORD-2026-001",
  "customer": "ACME Corp",
  "attachment_count": 3,
  "_claim_check": {
    "ticket": "s3://dim-payloads/2026-09-06/xyz123",
    "size": 5242880,
    "hash": "sha256:abc123...",
    "content_type": "application/octet-stream"
  }
}

↓ [pipeline: routing, filtering, authorization] (no 5MB overhead)

↓ [compliance-check route]

↓ [claim-resolve step]

Full message restored (payload retrieved from S3)
```

### Use Cases

1. **E-commerce with Attachments**
   - Orders with invoices, shipping labels, signatures
   - Pipeline stays fast; compliance only retrieves when auditing

2. **Document Processing**
   - Incoming scans, PDFs, archives
   - Routed immediately; expensive OCR only when needed

3. **Media/Video Workflows**
   - Video uploads with metadata
   - Metadata routes instantly; video transcoding on-demand

### Storage Backends

| Backend | Use Case | When |
|---------|----------|------|
| **Memory** | Testing, development | Local testing only |
| **S3 / S3-compatible** | Production | Standard choice (MinIO, AWS S3, DigitalOcean) |
| **PostgreSQL** | Central persistence | Future (planned) |

---

## Feature 2: Plugin SDK (M3.3)

### Problem

Standard JSON transformations aren't enough. You need:
- Domain-specific validation (credit card, PII checks)
- External API enrichment (customer data lookups)
- Complex transformations (encryption, compression)
- Custom business logic

Writing dim core code for each use case doesn't scale.

### Solution: Pluggable Functions

Register custom functions (Go binaries or WebAssembly modules) and call them from JSONata expressions.

### Configuration

#### Register plugins

```yaml
version: 1

functions:
  # Native Go plugin (RPC subprocess)
  validate_credit_card:
    type: plugin
    runtime: native
    ref: ./plugins/validate_cc

  # WebAssembly plugin (in-process sandbox)
  encrypt_pii:
    type: plugin
    runtime: wasm
    ref: ./plugins/encrypt_pii.wasm

  # Built-in function (no registration needed)
  # e.g. $length(), $substring(), etc.

sources:
  api:
    type: http
    port: 8080

routes:
  process-payment:
    from: api
    steps:
      - translate:
          expr: '{
            validation: $validate_credit_card(cc_number),
            encrypted: $encrypt_pii(ssn),
            amount: amount * 1.1
          }'
    sinks:
      - payment-sink
```

### Writing a Native Plugin (Go)

```go
// plugins/validate_cc/main.go
package main

import (
  "github.com/hashicorp/go-plugin"
  "github.com/naren-chakraview/dim/pkg/sdk"
)

type ValidateCCPlugin struct{}

func (p *ValidateCCPlugin) Call(input interface{}) (interface{}, error) {
  cc, ok := input.(string)
  if !ok {
    return nil, errors.New("expected string")
  }

  // Validate using Luhn algorithm or external API
  valid := luhnCheck(cc)
  return map[string]interface{}{
    "valid": valid,
    "last_4": cc[len(cc)-4:],
  }, nil
}

func main() {
  plugin.Serve(&plugin.ServeConfig{
    HandshakeConfig: plugin.HandshakeConfig{
      ProtocolVersion:  1,
      MagicCookieKey:   "DIM_PLUGIN",
      MagicCookieValue: "dim",
    },
    Plugins: map[string]plugin.Plugin{
      "validate_credit_card": &sdk.FunctionPlugin{Impl: &ValidateCCPlugin{}},
    },
  })
}
```

Build and place in `./plugins/validate_cc`.

### Writing a WASM Plugin (Rust)

```rust
// plugins/encrypt_pii/src/lib.rs
#[no_mangle]
pub extern "C" fn encrypt_pii(input: *const u8, len: usize) -> *const u8 {
    // Read input
    let input_str = std::str::from_utf8(unsafe {
        std::slice::from_raw_parts(input, len)
    }).unwrap();

    // Encrypt using AES-256
    let encrypted = encrypt_aes256(input_str, KEY);

    // Return as JSON
    let result = format!(r#"{{"encrypted": "{}"}}"#, encrypted);
    result.as_ptr()
}
```

Compile to WASM:
```bash
cargo build --target wasm32-unknown-unknown --release
cp target/wasm32-unknown-unknown/release/encrypt_pii.wasm ./plugins/
```

### Use Cases

1. **Validation Plugins**
   - Credit card, email, phone validation
   - Industry-specific rules (healthcare, financial)
   - Custom PII detection

2. **Enrichment Plugins**
   - Real-time customer/product lookups
   - Fraud scoring
   - Risk assessment

3. **Transformation Plugins**
   - Field encryption/decryption
   - Data formatting (phone, address normalization)
   - Compression/decompression

4. **Custom Business Logic**
   - Domain-specific calculations
   - API integrations
   - Legacy system compatibility

### Plugin Runtimes

| Runtime | Language | Performance | Sandbox | When |
|---------|----------|---|---|---|
| **Native** | Go (or any; compiled to binary) | Fastest | Process isolation | Trust & performance critical |
| **WASM** | Any (Rust, C, AssemblyScript) | Fast | Full sandbox | Security critical; untrusted code |

---

## Feature 3: Multi-Tenant Resource Isolation (M3.4)

### Problem

When running routes for multiple domains/tenants on shared infrastructure:

```
One overloaded domain → exhausts worker pool → all domains slow down
```

You need isolation without running separate instances (expensive).

### Solution: Per-Domain Resource Quotas

Specify resource limits per tenant/domain and let dim enforce them.

```yaml
version: 1

# Multi-tenant routing with resource quotas
routes:
  acme-orders:
    domain: acme-corp           # Tenant identifier
    from: kafka
    resource_quota:             # NEW in M3.4
      message_rate: 1000        # msg/sec max
      worker_slots: 10          # Parallel execution capacity
      lineage_quota: 100000     # Max lineage records
    steps:
      - filter: { expr: "status != 'cancelled'" }
      - translate: { expr: "{ order_id: id, total: amount * 1.1 }" }
    sinks:
      - acme-db

  globex-orders:
    domain: globex-inc          # Different tenant
    from: kafka
    resource_quota:
      message_rate: 5000        # Different limits
      worker_slots: 20
      lineage_quota: 500000
    steps:
      - filter: { expr: "status != 'cancelled'" }
      - translate: { expr: "{ order_id: id, total: amount * 1.05 }" }
    sinks:
      - globex-db
```

### Resource Dimensions

#### 1. Message Rate Limit

Controls inbound message throughput per domain.

```yaml
resource_quota:
  message_rate: 1000  # Max 1000 messages/second
```

When exceeded: New messages are queued and delayed (backpressure).

**Use case:** Prevent one tenant's burst from starving others.

#### 2. Worker Slot Allocation

Reserves processing concurrency per domain.

```yaml
resource_quota:
  worker_slots: 10  # Reserve 10 parallel workers
```

When exceeded: Messages wait in domain-specific queue.

**Use case:** Guarantee processing capacity per tenant.

**Example:**
```
Instance with 30 total workers:
├─ ACME Corp: 10 workers (order processing)
├─ Globex Inc: 12 workers (data sync)
└─ Initech: 8 workers (log aggregation)

If ACME's 10 workers are busy, Globex and Initech continue unaffected.
```

#### 3. Lineage Storage Quota

Caps audit trail records per domain.

```yaml
resource_quota:
  lineage_quota: 100000  # Keep max 100k records
```

When exceeded: Oldest records are purged per retention policy.

**Use case:** Prevent one tenant from filling lineage database.

### Multi-Tenant SaaS Example

```yaml
version: 1

sources:
  api:
    type: http
    port: 8080
    # Extract tenant from Authorization header
    tenant_from_header: X-Customer-ID

sinks:
  tenant-db:
    type: database
    connection_string: "${SECRET:DATABASE_URL}"

routes:
  # Template: each tenant gets identical route
  message-processing:
    from: api
    domain: "${tenant}"           # Variable from header
    auth: rbac                     # Verify tenant can access
    resource_quota:
      message_rate: 100          # Per-tenant rate limit
      worker_slots: 5            # Per-tenant workers
      lineage_quota: 10000       # Per-tenant lineage
    steps:
      - authorize:
          rule: "principal.tenant == domain"
      - translate:
          expr: '{ customer_id: principal.customer_id, payload: body }'
    sinks:
      - tenant-db
```

### Monitoring Multi-Tenant Usage

Query per-domain resource utilization:

```bash
# View current resource usage per domain
dimctl stats --by-domain

# Example output:
# Domain: acme-corp
#   Messages/sec: 450/1000 (45% of quota)
#   Active workers: 7/10 (70% of quota)
#   Lineage records: 45,200/100,000 (45% of quota)

# Domain: globex-inc
#   Messages/sec: 3,200/5000 (64% of quota)
#   Active workers: 18/20 (90% of quota)
#   Lineage records: 425,000/500,000 (85% of quota)
```

### Use Cases

1. **SaaS Platform**
   - Multiple customer accounts on single instance
   - Prevent noisy neighbors
   - Predictable per-customer performance

2. **Internal Multi-Domain**
   - Shared infrastructure across business units
   - Finance, Marketing, Operations on same instance
   - Different resource needs per domain

3. **Regulated Industries**
   - PCI/HIPAA compliance: data isolation per customer
   - Guaranteed processing capacity per customer
   - Audit trail quotas per compliance requirement

---

## Feature 4: Distributed Clustering (M3.1)

### Problem

Single instance limits:
- Throughput ceiling: One machine's worth of processing
- No high availability: Instance failure = service down
- Geographic distribution: All processing in one location

### Solution: Active-Active Cluster

Run multiple `dimd` instances sharing state, with:
- **No duplicate processing:** Shared idempotent dedup
- **Unified lineage:** Cross-instance audit trail queries
- **Coordinated hot-reload:** Config changes reach all instances
- **Horizontal scaling:** Add instances to increase throughput

### Configuration

#### Single Instance (Default)

```yaml
# dimd daemon starts in single-instance mode
./dimd --config routes.yaml
```

#### Cluster with Static Membership

```bash
# Environment variable or config file
export DIMD_CLUSTER_HOSTS="dimd-1:8080,dimd-2:8080,dimd-3:8080"
./dimd --config routes.yaml
```

Or in config file:

```yaml
version: 1

cluster:
  mode: active-active
  discovery: static
  hosts:
    - host: dimd-1
      port: 8080
    - host: dimd-2
      port: 8080
    - host: dimd-3
      port: 8080

  # Shared state backend
  state_backend: postgres
  state_connection: "postgres://cluster-db:5432/dimd_state"

routes:
  # Routes defined once, run on all instances
  order-processing:
    from: kafka
    steps:
      - filter: { expr: "amount > 0" }
    sinks:
      - order-db
```

#### Docker Compose: 3-Instance Cluster

```yaml
# docker-compose.yml
version: '3.8'

services:
  postgres:
    image: postgres:15
    environment:
      POSTGRES_DB: dimd_state
      POSTGRES_USER: dimd
      POSTGRES_PASSWORD: secret
    volumes:
      - ./init-db.sql:/docker-entrypoint-initdb.d/init.sql

  dimd-1:
    image: dim:v0.8.0
    depends_on:
      - postgres
    environment:
      DIMD_CLUSTER_HOSTS: "dimd-1:8080,dimd-2:8080,dimd-3:8080"
      DIMD_STATE_DB: "postgres://dimd:secret@postgres:5432/dimd_state"
    ports:
      - "8080:8080"
      - "8081:8081"
    volumes:
      - ./routes.yaml:/etc/dim/routes.yaml

  dimd-2:
    image: dim:v0.8.0
    depends_on:
      - postgres
    environment:
      DIMD_CLUSTER_HOSTS: "dimd-1:8080,dimd-2:8080,dimd-3:8080"
      DIMD_STATE_DB: "postgres://dimd:secret@postgres:5432/dimd_state"
    ports:
      - "8081:8080"
      - "8082:8081"
    volumes:
      - ./routes.yaml:/etc/dim/routes.yaml

  dimd-3:
    image: dim:v0.8.0
    depends_on:
      - postgres
    environment:
      DIMD_CLUSTER_HOSTS: "dimd-1:8080,dimd-2:8080,dimd-3:8080"
      DIMD_STATE_DB: "postgres://dimd:secret@postgres:5432/dimd_state"
    ports:
      - "8082:8080"
      - "8083:8081"
    volumes:
      - ./routes.yaml:/etc/dim/routes.yaml
```

### Shared State Backend

Cluster instances share state through a backend database:

| Feature | State | Backend |
|---------|-------|---------|
| **Idempotent Dedup** | Message IDs seen | Postgres/Redis |
| **Lineage** | Message audit trail | Postgres (queryable across cluster) |
| **Config Generation** | Which config version all instances use | Postgres |

#### PostgreSQL Schema (auto-initialized)

```sql
-- Dedup store: which messages have been processed
CREATE TABLE dedup_store (
  id TEXT PRIMARY KEY,
  instance_id TEXT,
  processed_at TIMESTAMP,
  expires_at TIMESTAMP
);

-- Lineage: message journey across all instances
CREATE TABLE lineage_records (
  id TEXT PRIMARY KEY,
  message_id TEXT,
  instance_id TEXT,
  route TEXT,
  timestamp TIMESTAMP,
  metadata JSONB
);

-- Cluster state: generation tracking, membership
CREATE TABLE cluster_generation (
  instance_id TEXT,
  generation_version INT,
  updated_at TIMESTAMP
);
```

### Cluster Guarantees

#### No Duplicate Processing

If a message is ingested by multiple instances:

```
Instance 1: Receives message, checks dedup store → "not seen yet"
            Processes message, writes to dedup store

Instance 2: Receives same message (retry, multi-source)
            Checks dedup store → "already processed"
            Skips processing (idempotent)
```

#### Unified Lineage

Query audit trail across all instances:

```bash
# Find all messages from customer "ACME" processed by any instance
dimctl lineage query --filter "customer_id='ACME'"

# Output shows:
# dimd-1: 450 messages processed
# dimd-2: 480 messages processed
# dimd-3: 470 messages processed
# Total: 1,400 messages (with combined latencies, errors, etc.)
```

#### Coordinated Hot-Reload

Update routes on one instance → all instances transition together:

```bash
# Update routes.yaml on shared filesystem or config management
vim routes.yaml
git push

# Each instance detects change, validates, switches over
# No downtime, no message loss, in-flight messages complete on old DAG
```

### Cluster Topologies

#### 1. Geographic Distribution (High Availability)

```
         API Load Balancer
              ↙   ↓   ↘
         dimd-1 dimd-2 dimd-3
         (US-E) (EU)   (AP)
             ↘    ↓    ↙
        Postgres (Replication)
              ↙    ↓    ↘
         S3-replica (regional)
```

Benefits: Geographic locality, disaster recovery, regulatory compliance.

#### 2. Scale-Out (Throughput)

```
              API Load Balancer
                  ↙   ↓   ↘
             dimd-1...dimd-N
          [N identical instances]
                     ↓
          Postgres State Backend
```

Benefits: Linear throughput scaling, simple horizontal addition.

#### 3. Specialized Workloads (Routing by Domain)

```
       dimd-1: Financial routes (PCI isolated)
       dimd-2: Marketing routes (high throughput)
       dimd-3: Operations routes (low latency)
                      ↓
           Postgres Shared State
           (unified lineage, dedup)
```

Benefits: Compliance isolation, workload-specific tuning, shared audit trail.

### Monitoring Cluster Health

```bash
# Check cluster membership and status
dimctl cluster status

# Output:
# Cluster: active-active
# Instances: 3
#
# dimd-1 (us-east-1):
#   Status: healthy
#   Generation: 5
#   Routes: order-processing, fulfillment
#
# dimd-2 (eu-west-1):
#   Status: healthy
#   Generation: 5
#   Routes: order-processing, fulfillment
#
# dimd-3 (ap-southeast-1):
#   Status: healthy
#   Generation: 5
#   Routes: order-processing, fulfillment
#
# Shared State:
#   Lineage records: 1,250,000
#   Dedup entries: 45,000 (TTL: 1h)
```

### Use Cases

1. **High-Availability Mission-Critical Routes**
   - Order processing, payment approval
   - Instance failure doesn't drop messages
   - Automatic failover via load balancer

2. **Geographic Distribution**
   - Process orders in user's timezone
   - Reduced latency, regulatory data residency
   - Disaster recovery across regions

3. **Horizontal Scaling for High Throughput**
   - Ingesting 100k+ messages/second
   - Add instances as needed
   - Unified lineage across all

4. **Multi-Tenant with Tenant-Specific Instances**
   - Premium customers get dedicated instance
   - Shared dedup/lineage for operational visibility
   - Compliance isolation per tenant

---

## Combining Features: Complete Example

Multi-tenant SaaS with large attachments, custom validation, and geographic clustering:

```yaml
version: 1

cluster:
  mode: active-active
  discovery: static
  hosts:
    - host: dimd-us
      port: 8080
    - host: dimd-eu
      port: 8080
  state_backend: postgres
  state_connection: "${SECRET:CLUSTER_DB}"

functions:
  validate_tax_id:
    type: plugin
    runtime: native
    ref: ./plugins/validate_tax_id
  
  encrypt_pii:
    type: plugin
    runtime: wasm
    ref: ./plugins/encrypt_pii.wasm

sources:
  api:
    type: http
    port: 8080
    tenant_from_header: X-Customer-ID

sinks:
  customer-db:
    type: database
    connection_string: "${SECRET:DATABASE_URL}"

routes:
  invoice-processing:
    domain: "${tenant}"
    from: api
    
    # Multi-tenant resource isolation
    resource_quota:
      message_rate: 500
      worker_slots: 5
      lineage_quota: 50000
    
    auth: rbac
    
    steps:
      # Extract and store large invoice PDF
      - claim-check:
          store_type: s3
          store_config:
            bucket: invoices
          payload_field: invoice_pdf
          remove_after: true
      
      # Custom validation using plugin
      - translate:
          expr: '{
            invoice_id: id,
            tax_id_valid: $validate_tax_id(tax_id),
            customer: $encrypt_pii(customer_name),
            amount: amount
          }'
      
      # Retrieve PDF only if validation fails (for compliance)
      - route:
          cases:
            - when: "!tax_id_valid"
              steps:
                - claim-resolve:
                    store_type: s3
                    store_config:
                      bucket: invoices
                    ticket_field: _claim_check
                    restore_field: invoice_pdf
              sinks:
                - customer-db  # Store failed invoice with PDF
            - default:
              sinks:
                - customer-db  # Success path (no PDF needed)
```

**Result:**
- ✅ Multiple tenants share infrastructure (M3.4)
- ✅ Large invoices stay lightweight in transit (M3.2)
- ✅ Custom validation logic pluggable (M3.3)
- ✅ Geographic instances, unified lineage (M3.1)
- ✅ One tenant overload doesn't affect others

---

## Related Documentation

- **[LANGUAGE_REFERENCE.md](LANGUAGE_REFERENCE.md)** — Complete configuration reference
- **[USE_CASES.md](USE_CASES.md)** — Additional examples
- **[CLI_REFERENCE.md](CLI_REFERENCE.md)** — dimctl command reference
- **[DEPLOYMENT_AND_RELEASE.md](DEPLOYMENT_AND_RELEASE.md)** — Deployment patterns
- **[../OKF.md](../OKF.md)** — Architectural decisions

---

## Next Steps

1. **New to Phase 3?** Start with the quickstart matrix at top of this guide
2. **Building SaaS?** See Multi-Tenant Resource Isolation section
3. **Handling attachments?** See Claim Check Pattern section
4. **Custom logic needed?** See Plugin SDK section
5. **Global scale?** See Distributed Clustering section

Happy building! 🚀
