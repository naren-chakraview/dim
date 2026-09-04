# M1.7: OpenLineage/Marquez Integration — Data Lineage Export

**Phase:** Phase 1  
**Milestone:** M1.7 (OpenLineage/Marquez)  
**Status:** Design & Implementation Plan  
**Date:** 2026-09-04

## Overview

M1.7 integrates OpenLineage standard for data lineage tracking with Marquez as the metadata backend. This enables:
- **Standards-based lineage export:** OpenLineage events (JSON-LD format)
- **Automatic lineage capture:** On each message flow through route
- **Dataset facets:** SchemaDatasetFacet tied to contract validation results
- **Continuous export mode:** Stream lineage events to Marquez in real-time

## Subtasks

### M1.7.1: Stand Up Local Marquez

**Objective:** Enable local Marquez development environment

**Deployment:**
```yaml
# docker-compose.marquez.yml
version: '3.8'
services:
  postgres:
    image: postgres:14-alpine
    environment:
      POSTGRES_DB: marquez
      POSTGRES_USER: marquez
      POSTGRES_PASSWORD: marquez
    ports:
      - "5432:5432"
    volumes:
      - postgres_data:/var/lib/postgresql/data

  marquez:
    image: marquezproject/marquez:latest
    environment:
      MARQUEZ_DB_HOST: postgres
      MARQUEZ_DB_NAME: marquez
      MARQUEZ_DB_USER: marquez
      MARQUEZ_DB_PASSWORD: marquez
    ports:
      - "5000:5000"  # API
      - "3000:3000"  # Web UI
    depends_on:
      - postgres

  # Optional: Marquez CLI for local testing
  marquez-cli:
    image: marquezproject/marquez-cli:latest
    environment:
      MARQUEZ_URL: http://marquez:5000
```

**Setup:**
```bash
docker-compose -f docker-compose.marquez.yml up -d
# Marquez API at http://localhost:5000
# Marquez Web UI at http://localhost:3000
```

**Health Check:**
```bash
curl http://localhost:5000/api/v1/health
# Expected: {"status":"ok"}
```

**Exit Criteria:**
✅ Marquez container running  
✅ Postgres initialized  
✅ Web UI accessible  
✅ API health check passes  

### M1.7.2: Wire OpenLineage Event Emission

**Objective:** Emit OpenLineage events as messages flow through routes

**OpenLineage Event Structure:**
```json
{
  "eventType": "COMPLETE",
  "eventTime": "2026-09-04T12:34:56.000Z",
  "run": {
    "runId": "urn:uuid:...",
    "facets": {
      "parent": {
        "_producer": "https://github.com/naren-chakraview/dim",
        "parentRunId": null,
        "job": {
          "namespace": "dim",
          "name": "process-orders.v1"
        }
      },
      "nominalTime": {
        "nominalStartTime": "2026-09-04T12:34:00.000Z",
        "nominalEndTime": "2026-09-04T12:35:00.000Z"
      }
    }
  },
  "job": {
    "namespace": "dim",
    "name": "process-orders.v1",
    "facets": {
      "sourceCodeLocation": {
        "repoUrl": "https://github.com/naren-chakraview/dim",
        "path": "examples/order-processing/order-processing.yaml",
        "branch": "master",
        "tag": ""
      },
      "ownership": {
        "owners": [
          {
            "name": "Data Platform Team",
            "type": "TEAM"
          }
        ]
      }
    }
  },
  "inputs": [
    {
      "namespace": "dim://webhook",
      "name": "order-events",
      "facets": {
        "schema": {
          "fields": [
            {"name": "order_id", "type": "string"},
            {"name": "amount", "type": "decimal"}
          ]
        }
      }
    }
  ],
  "outputs": [
    {
      "namespace": "dim://amqp",
      "name": "orders-processed",
      "facets": {
        "schema": {
          "fields": [...]
        }
      }
    }
  ]
}
```

**Implementation:**
```go
type OpenLineageEmitter struct {
    marquezURL string
    httpClient *http.Client
}

func (ole *OpenLineageEmitter) EmitEvent(ctx context.Context, event *OpenLineageEvent) error {
    // Marshal to JSON-LD
    // POST to Marquez /api/v1/lineage
    // Handle errors (retry-able vs permanent)
}

type OpenLineageEvent struct {
    EventType   string `json:"eventType"` // START, COMPLETE, FAIL, ABORT
    EventTime   string `json:"eventTime"`
    Run         *Run   `json:"run"`
    Job         *Job   `json:"job"`
    Inputs      []Dataset `json:"inputs"`
    Outputs     []Dataset `json:"outputs"`
}
```

**Hook Points:**
- Route ingestion (START event)
- Step completion (COMPLETE event)
- Step failure (FAIL event)
- Route completion (COMPLETE event with output datasets)

**Exit Criteria:**
✅ OpenLineageEmitter created  
✅ Events marshaled to JSON-LD  
✅ Events posted to Marquez API  
✅ Error handling (retry, deadletter)  
✅ Test coverage: 5+ test cases  

### M1.7.3: SchemaDatasetFacet Publication

**Objective:** Publish schema information tied to contract validation

**Contract-Derived Schema:**
```go
type SchemaDatasetFacet struct {
    Fields []SchemaField `json:"fields"`
}

type SchemaField struct {
    Name    string `json:"name"`
    Type    string `json:"type"`    // JSON schema type
    Desc    string `json:"description"`
    Nullable bool   `json:"nullable"`
}
```

**Flow:**
1. Contract validation runs on message
2. If schema defined in contract → extract fields
3. Create SchemaDatasetFacet
4. Include in OpenLineage output dataset
5. Publish to Marquez

**Example with Order Contract:**
```yaml
routes:
  process-orders:
    from: webhook-in
    contract:
      name: OrderEvent
      version: v1
      schema:
        type: object
        properties:
          order_id:
            type: string
            description: "Unique order identifier"
          amount:
            type: number
            description: "Order amount in USD"
          customer_id:
            type: string
    
    steps:
      - transform:
          expr: "body.amount > 0"
```

**Lineage Output with Schema:**
```json
{
  "outputs": [
    {
      "namespace": "dim://process-orders-output",
      "name": "orders-processed",
      "facets": {
        "schema": {
          "fields": [
            {
              "name": "order_id",
              "type": "string",
              "description": "Unique order identifier"
            },
            {
              "name": "amount",
              "type": "number",
              "description": "Order amount in USD"
            },
            {
              "name": "customer_id",
              "type": "string"
            }
          ]
        }
      }
    }
  ]
}
```

**Implementation:**
```go
type LineageBuilder struct {
    route   *Route
    contract *Contract
}

func (lb *LineageBuilder) BuildSchemaFacet() *SchemaDatasetFacet {
    if lb.contract == nil || lb.contract.Schema == nil {
        return nil
    }
    
    facet := &SchemaDatasetFacet{}
    for propName, propSchema := range lb.contract.Schema.Properties {
        facet.Fields = append(facet.Fields, SchemaField{
            Name: propName,
            Type: propSchema.Type,
            Desc: propSchema.Description,
            Nullable: !contains(lb.contract.Schema.Required, propName),
        })
    }
    return facet
}
```

**Exit Criteria:**
✅ SchemaDatasetFacet extracted from contracts  
✅ Published with output datasets  
✅ Marquez displays schema in UI  
✅ Field descriptions visible  
✅ Test coverage: contract → facet conversion  

### M1.7.4: Continuous Export Mode

**Objective:** Stream lineage events to Marquez in real-time

**Configuration:**
```yaml
lineage:
  mode: continuous  # or "batch", "off"
  marquez_url: http://localhost:5000
  batch_size: 100
  flush_interval_ms: 5000
  retry_policy:
    max_retries: 3
    backoff_ms: 500
```

**Implementation:**
```go
type LineageExporter struct {
    marquezURL string
    eventCh    chan *OpenLineageEvent
    batchSize  int
    flushTicker *time.Ticker
}

func (le *LineageExporter) Start(ctx context.Context) {
    go le.batchExportLoop(ctx)
}

func (le *LineageExporter) batchExportLoop(ctx context.Context) {
    batch := make([]*OpenLineageEvent, 0, le.batchSize)
    
    for {
        select {
        case event := <-le.eventCh:
            batch = append(batch, event)
            if len(batch) >= le.batchSize {
                le.flushBatch(ctx, batch)
                batch = make([]*OpenLineageEvent, 0, le.batchSize)
            }
        case <-le.flushTicker.C:
            if len(batch) > 0 {
                le.flushBatch(ctx, batch)
                batch = make([]*OpenLineageEvent, 0, le.batchSize)
            }
        case <-ctx.Done():
            if len(batch) > 0 {
                le.flushBatch(ctx, batch)
            }
            return
        }
    }
}

func (le *LineageExporter) flushBatch(ctx context.Context, events []*OpenLineageEvent) error {
    // POST batch to Marquez with retry policy
    // On failure: log and deadletter to lineage-export-dlq
}
```

**Batching Strategy:**
- Accumulate events in memory
- Flush on: batch size reached, timer expires, shutdown
- Retry failed batches with exponential backoff
- Deadletter permanently failed events

**Performance:**
- Non-blocking (channel-based)
- Configurable batch size and flush interval
- Thread-safe event queueing
- Memory bounded (batch size limit)

**Exit Criteria:**
✅ LineageExporter implementation  
✅ Batching and flushing logic  
✅ Retry policy with backoff  
✅ Deadlettering for failed exports  
✅ Startup/shutdown gracefully  
✅ Performance: <1ms overhead per message  
✅ Test coverage: batching, retries, shutdown  

## Configuration

### Route YAML with Lineage

```yaml
routes:
  process-orders:
    from: webhook-in
    contract:
      name: OrderEvent
      version: v1
      schema:
        type: object
        properties:
          order_id:
            type: string
          amount:
            type: number
        required: [order_id, amount]
    
    steps:
      - authorize:
          mode: rbac
          require_roles: [seller]
      
      - transform:
          expr: |
            {
              "order_id": body.order_id,
              "amount": body.amount,
              "timestamp": $now()
            }
    
    to: amqp-sink

lineage:
  mode: continuous
  marquez_url: http://localhost:5000
  batch_size: 100
  flush_interval_ms: 5000
  retry_policy:
    max_retries: 3
    backoff_ms: 500
```

### Docker Compose for E2E Testing

```yaml
version: '3.8'
services:
  postgres:
    image: postgres:14-alpine
    environment:
      POSTGRES_DB: marquez
      POSTGRES_USER: marquez
      POSTGRES_PASSWORD: marquez
    ports:
      - "5432:5432"

  marquez:
    image: marquezproject/marquez:latest
    environment:
      MARQUEZ_DB_HOST: postgres
      MARQUEZ_DB_NAME: marquez
      MARQUEZ_DB_USER: marquez
      MARQUEZ_DB_PASSWORD: marquez
    ports:
      - "5000:5000"
      - "3000:3000"
    depends_on:
      - postgres

  rabbitmq:
    image: rabbitmq:3.12-management
    environment:
      RABBITMQ_DEFAULT_USER: guest
      RABBITMQ_DEFAULT_PASS: guest
    ports:
      - "5672:5672"
      - "15672:15672"

  dim:
    build: .
    environment:
      MARQUEZ_URL: http://marquez:5000
      RABBITMQ_URL: amqp://guest:guest@rabbitmq:5672/
    ports:
      - "8080:8080"
    volumes:
      - ./examples/order-processing:/config
    depends_on:
      - marquez
      - rabbitmq
```

## Testing Strategy

### Unit Tests
- OpenLineage event construction
- Schema extraction from contracts
- Event marshaling to JSON-LD
- Batch accumulation logic
- Retry backoff calculation

### Integration Tests
- Emit event to local Marquez
- Verify event stored in database
- Check schema facet in Marquez
- End-to-end lineage: source → steps → sink
- Replay lineage (M1.8 integration)

### End-to-End Test

```bash
# 1. Start Marquez
docker-compose -f docker-compose.marquez.yml up -d

# 2. Run dim with continuous export
dimctl run --lineage continuous examples/order-processing/order-processing.yaml

# 3. Send test message
curl -X POST http://localhost:8080/orders \
  -H "Content-Type: application/json" \
  -d '{
    "order_id": "ord-123",
    "amount": 99.99,
    "customer_id": "cust-456"
  }'

# 4. Verify in Marquez
curl http://localhost:5000/api/v1/lineage?namespace=dim&name=process-orders.v1

# Output:
# {
#   "jobs": [...],
#   "runs": [{
#     "id": "urn:uuid:...",
#     "job": {
#       "namespace": "dim",
#       "name": "process-orders.v1"
#     },
#     "inputs": [{
#       "namespace": "dim://webhook",
#       "name": "order-events"
#     }],
#     "outputs": [{
#       "namespace": "dim://amqp",
#       "name": "orders-processed",
#       "facets": {
#         "schema": {
#           "fields": [...]
#         }
#       }
#     }]
#   }]
# }

# 5. View in Web UI
# Visit http://localhost:3000
# Search for "process-orders.v1"
```

## Exit Criteria

### M1.7.1 ✅
- ✅ Marquez container running
- ✅ Postgres initialized
- ✅ Web UI accessible at :3000
- ✅ API health check passes

### M1.7.2 ✅
- ✅ OpenLineageEmitter created
- ✅ Events marshaled to JSON-LD
- ✅ Events posted to Marquez
- ✅ Error handling (retry, deadletter)
- ✅ Test coverage: 5+ tests

### M1.7.3 ✅
- ✅ SchemaDatasetFacet extracted from contracts
- ✅ Published with output datasets
- ✅ Marquez displays schema in UI
- ✅ Field descriptions visible
- ✅ Test coverage: contract → facet

### M1.7.4 ✅
- ✅ LineageExporter with batching
- ✅ Configurable batch size and flush interval
- ✅ Retry policy with exponential backoff
- ✅ Deadlettering for failed exports
- ✅ Graceful startup/shutdown
- ✅ Performance: <1ms overhead per message
- ✅ Test coverage: 7+ tests

## References

- **OpenLineage Spec:** https://openlineage.io/
- **OpenLineage Python Client:** https://github.com/OpenLineage/OpenLineage/tree/main/client/python
- **Marquez Docs:** https://marquezproject.github.io/
- **Marquez API:** https://marquezproject.github.io/api.html
- **Dataset Facets:** https://openlineage.io/docs/spec/facets/dataset-facets/schema
