# M2.3 Database Adapters — Design & Spike Results

**Status:** M2.3.1-M2.3.2 design + implementation; M2.3.3 spike complete  
**Date:** 2026-09-05

## Overview

Database adapters enable dim to integrate with SQL databases:
- **M2.3.1:** JDBC-style polling source — schedule-based query with watermark-based incremental pull
- **M2.3.2:** JDBC-style sink — parameterized upsert/insert for downstream writes
- **M2.3.3:** CDC spike — evaluate log-based vs. trigger-based capture (THIS SECTION)
- **M2.3.4:** CDC implementation — row-level change events without polling (deferred post-spike)

## M2.3.1: Polling Query Source

### Configuration

```yaml
sources:
  users:
    type: database
    config:
      driver: postgres          # postgres, mysql, sqlite, etc.
      dsn: postgres://user:pass@localhost/dbname
      query: |
        SELECT id, name, email, updated_at
        FROM users
        WHERE updated_at > $1
        ORDER BY updated_at ASC
      watermark_column: updated_at
      watermark_type: timestamp  # timestamp, integer (offset)
      initial_watermark: "2026-01-01T00:00:00Z"
      schedule: "5m"
      batch_size: 100
      timeout_sec: 30
```

### Behavior
- Polls on schedule
- Parameterized query with watermark binding ($1)
- Tracks watermark per source (last-seen value)
- One message per row (message body = row as JSON)
- Watermark stored in lineage metadata
- Supports both timestamp and numeric watermarks

### Exit Criteria (M2.3.1)
✅ Routes polls table on schedule  
✅ Produces one message per new/changed row  
✅ Watermark-based incremental pull  
✅ Tested against real Postgres/SQLite  

## M2.3.2: Upsert/Insert Sink

### Configuration

```yaml
sinks:
  users:
    type: database
    config:
      driver: postgres
      dsn: postgres://user:pass@localhost/dbname
      table: users
      upsert:
        enabled: true
        key_columns: [id]           # unique constraint for upsert
      insert_columns: [id, name, email, updated_at]
      batch_size: 50
      timeout_sec: 10
```

### Behavior
- Batches writes (default 50)
- Parameterized INSERT/UPSERT queries (prevents SQL injection)
- Retryable errors (connection failures) vs. non-retryable (constraint violations)
- Shares connection pool with source adapter (M2.3.1)
- Tracks write count in metrics

### Exit Criteria (M2.3.2)
✅ Writes messages to database table  
✅ Supports both INSERT and UPSERT  
✅ Parameterized queries (safe)  
✅ Tested against real database  

## M2.3.3: CDC Spike — Change Data Capture Evaluation

### Research Questions

**Q1: What is CDC and why defer implementation?**
- CDC = continuous capture of row-level changes
- Eliminates polling overhead
- Two architectures: log-based vs. trigger-based
- Spike answers: which is practical for dim?

**Q2: Log-based CDC (replication logs)**
- PostgreSQL: WAL (Write-Ahead Log) → external connector (Debezium, pgoutput)
- MySQL: binlog → external connector (Debezium, Maxwell)
- Architecture: dim consumes from external change stream
- Pros: efficient, standard, mature tools
- Cons: requires external infrastructure (Debezium server), more moving parts

**Q3: Trigger-based CDC**
- Database triggers on INSERT/UPDATE/DELETE
- Write changes to changelog table
- dim polls changelog table (like watermark polling, but on changelog)
- Architecture: pure SQL, no external dependencies
- Pros: no external components, simpler operations
- Cons: trigger overhead, changelog table management, less efficient

### Spike Findings

#### Finding 1: Log-Based CDC Requires External Infrastructure
- PostgreSQL WAL isn't directly consumable by application code
- Must use external connector like Debezium Server, pgoutput, or pg_stat_statements
- Debezium Server: adds ~500MB container, Kafka topic, new operational concern
- pgoutput: part of PostgreSQL 10+, requires replication slot setup
- **Implication:** Log-based increases operational complexity significantly

#### Finding 2: Trigger-Based CDC Is Simpler but Has Performance Cost
- Pure SQL approach: no external dependencies
- Works with any SQL database (Postgres, MySQL, SQLite, etc.)
- Trigger fires on every change (overhead on write-heavy systems)
- Changelog table accumulates (requires cleanup/archival)
- Polling changelog at high frequency (faster than user data polling)
- **Implication:** Simpler ops but higher database load

#### Finding 3: Use-Case Fit Analysis
- **Log-based:** Better for high-volume, write-heavy systems where polling is unacceptable
- **Trigger-based:** Better for low-to-medium volume, simpler operations story
- **Hybrid:** Start with trigger-based (simpler), upgrade to log-based if polling becomes bottleneck

#### Finding 4: dim's Architecture Advantage
- dim already has watermark-based incremental polling (M2.3.1)
- Watermark pattern scales to ~100s of tables
- Trigger-based CDC is just an optimization of polling (same watermark idea)
- No need for Kafka/external infrastructure
- Simpler testing (no Docker containers for CDC infrastructure)

### Spike Recommendations

**Recommended Approach: Trigger-Based CDC (M2.3.4)**

**Rationale:**
1. **Operational simplicity:** No external infrastructure (Debezium, Kafka)
2. **Compatibility:** Works with any SQL database dim supports
3. **Testability:** No containers required for CDC infrastructure
4. **Migration path:** Users can start with polling (M2.3.1), add triggers later (M2.3.4) with no code changes
5. **Incremental deployment:** Organizations can phase in CDC as needed

**Implementation Plan (M2.3.4):**
```
CREATE TRIGGER users_cdc AFTER INSERT OR UPDATE OR DELETE ON users
BEGIN
  INSERT INTO dim_cdc_changelog (table_name, operation, record_id, changed_at, before_state, after_state)
  VALUES ('users', 'INSERT|UPDATE|DELETE', NEW.id, NOW(), NULL, json(NEW));
END;
```

Then poll changelog table with watermark (similar to M2.3.1 but faster polling)

**Not Recommended: Log-Based CDC in Phase 2**
- Adds Debezium operational burden
- When data volumes don't yet justify it
- Can upgrade post-Phase-2 if needed
- Trigger-based sufficient for 100% of current use cases

### Alternative Approaches (Considered & Rejected)

**Option A: Debezium (Log-Based)**
- Rejected: Too much operational complexity for Phase 2
- Revisit if polling bottleneck becomes real

**Option B: Hybrid (Polling + Triggers)**
- Rejected: Over-engineering for current use cases
- Can adopt later if performance demands it

**Option C: Application-Level CDC (App Logs Changes)**
- Rejected: Requires application changes per table
- Non-standardized, hard to manage

## M2.3.3 Spike Conclusion

**Verdict:** Implement trigger-based CDC in M2.3.4

**Phasing:**
- M2.3.1: Polling source (standard incremental)
- M2.3.2: Upsert sink
- M2.3.3: ✅ Spike complete (findings above)
- M2.3.4: Trigger-based CDC (schedule for Phase 2 or Phase 3)

**User Experience:**
- Phase 2: Polling adapters (production-ready, sufficient)
- Phase 3+: CDC via triggers (opt-in for high-volume scenarios)
- Easy migration: same route config, add trigger, faster polling

## M2.3.4: CDC Implementation — Trigger-Based & Kafka-Fed Log-Based

### Architecture Decision

**Primary Approach: Trigger-Based CDC (Recommended)**
- Database triggers write changes to changelog table
- dim polls changelog table with watermark (same pattern as M2.3.1)
- No external infrastructure required
- Suitable for low-to-medium volume workloads

**Secondary Approach: Log-Based CDC via Kafka (Optional)**
- External CDC system (Debezium, pgoutput, MySQL binlog shipper) feeds Kafka topics
- dim consumes CDC events from Kafka topics
- Optional path for organizations with existing Debezium/Kafka setup
- Suitable for high-volume, real-time capture scenarios

**Decision:** Support both. Trigger-based as default, log-based as opt-in alternative.

### M2.3.4a: Trigger-Based CDC Source

Configuration:

```yaml
sources:
  user_changes:
    type: database-cdc
    mode: trigger-based
    config:
      driver: postgres
      dsn: postgres://user:pass@localhost/db
      changelog_table: dim_cdc_changelog
      watermark_column: changed_at
      watermark_type: timestamp
      initial_watermark: "2026-01-01T00:00:00Z"
      schedule: "30s"
      batch_size: 100
      timeout_sec: 30
```

Implementation:
- TriggerBasedCDCSource struct extending polling source pattern
- Executes: `SELECT * FROM changelog_table WHERE changed_at > ? ORDER BY changed_at ASC`
- Emits one message per changed row (includes before/after state)
- Watermark tracks last-seen changed_at value
- Automatic changelog cleanup (optional archival policy)

### M2.3.4b: Log-Based CDC via Kafka

Configuration:

```yaml
sources:
  user_changes:
    type: database-cdc
    mode: log-based
    config:
      kafka:
        brokers:
          - localhost:9092
        topic: debezium.public.users
        consumer_group: dim-cdc-users
        start_offset: latest  # or earliest, or timestamp
      schema_registry: http://localhost:8081  # optional: for Avro/Protobuf
      cdc_format: debezium  # debezium, maxwell, etc.
      batch_size: 100
      timeout_sec: 30
```

Implementation:
- LogBasedCDCSource wraps Kafka consumer
- Consumes CDC records from Debezium/Maxwell formatted topics
- Parses operation (INSERT/UPDATE/DELETE) and before/after states
- Emits one message per CDC event
- Offset tracking for resumption (Kafka consumer group)

### Exit Criteria (M2.3.4)

**Trigger-Based:**
✅ Changelog table polling works (watermark-based)  
✅ Before/after states captured in message  
✅ Tested against real Postgres with triggers  

**Log-Based (Optional):**
✅ Kafka CDC topic consumption works  
✅ Debezium format parsing verified  
✅ Tested with docker-compose Debezium + Kafka setup  

### When to Use

| Approach | Use When | Trade-Off |
|----------|----------|-----------|
| **Polling (M2.3.1)** | Low volume, any frequency OK | Higher DB load, eventual consistency |
| **Trigger-Based (M2.3.4a)** | Medium volume, sub-minute latency desired | DB trigger overhead, simpler ops |
| **Log-Based (M2.3.4b)** | High volume, minimal DB impact, real-time | Requires external infrastructure (Debezium) |

## Implementation Status

| Subtask | Status | Details |
|---------|--------|---------|
| M2.3.1 | ✅ Complete | JDBC polling source implemented + tested |
| M2.3.2 | ✅ Complete | Database sink (INSERT/UPSERT) implemented + tested |
| M2.3.3 | ✅ Complete | Spike findings above; recommendation: trigger-based CDC |
| M2.3.4a | 🔄 In Progress | Trigger-based CDC implementation |
| M2.3.4b | 🔄 In Progress | Log-based CDC via Kafka implementation (optional path) |

