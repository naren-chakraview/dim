# M2.6 Purge-Log Auto-Export on Expiry Warning — Design Document

**Status:** M2.6.1 Design  
**Date:** 2026-09-05

## Overview

The purge-log (design §10.4) currently emits alerts when entries approach their retention boundary. M2.6 extends this by automatically exporting expiring entries to S3 before deletion, keeping records available for audit/recovery.

## Design (M2.6.1)

### Export Target & Format

**Use existing S3 sink adapter** (Phase 1 standard):
- Same connection pooling, error handling, retry logic as regular S3 sink
- No new transport layer required

**Export format: JSONL** (JSON Lines):
```jsonl
{"id":"abc123","route":"orders","ingested_at":"2026-01-01T10:30:00Z","expired_at":"2026-09-05T10:30:00Z","data":{...}}
{"id":"abc124","route":"orders","ingested_at":"2026-01-02T11:30:00Z","expired_at":"2026-09-05T11:30:00Z","data":{...}}
```

**S3 path structure:**
```
s3://bucket/dim-purge-log-exports/
  2026/09/05/  # date partition
    orders-2026-09-05-001.jsonl  # route + batch
    orders-2026-09-05-002.jsonl
    payments-2026-09-05-001.jsonl
```

**Record schema:**
- `id`: purge-log entry ID
- `route`: route name
- `ingested_at`: message ingest time (ISO 8601)
- `expired_at`: expiration time
- `expiry_warning_triggered_at`: timestamp when warning triggered
- `data`: the original purge-log entry JSON

### Configuration

```yaml
purge_log:
  retention_days: 90
  expiry_warning_lead_days: 7
  auto_export:
    enabled: true
    target: s3
    s3:
      bucket: my-bucket
      path_prefix: dim-purge-log-exports/
      batch_size: 1000
      flush_interval_sec: 300
```

## Implementation (M2.6.2)

**Integration with reaper mechanism:**
1. Reaper thread checks expiry-warning threshold (existing)
2. For entries crossing threshold:
   - Emit alert (existing behavior)
   - Queue for export (new behavior)
3. Export goroutine batches and writes to S3
4. Mark as exported in purge-log

**Flow:**
```
Reaper checks expiry threshold
    ↓
Entry crossing expiry_warning_lead?
    ├─ NO → continue
    └─ YES ↓
        Emit alert (existing)
        Queue for S3 export (new)
        Background export goroutine
            Batch entries
            Write to S3 (JSONL)
            Mark exported in purge-log
```

## Testing (M2.6.3)

- Entry approaching expiration triggers alert AND export
- Exported JSONL is valid JSON (one object per line)
- Exported record matches original purge-log entry
- S3 path structure correct with date partitions
- Retry on S3 failure (exponential backoff)

## Exit Criteria

✅ Purge-log entries crossing expiry-warning threshold exported to S3  
✅ Export format JSONL, partitioned by date  
✅ Existing alert mechanism still works (both alert AND export)  
✅ Exported record independently verifiable against purge-log  
✅ S3 sink connection reused (no new infrastructure)  

