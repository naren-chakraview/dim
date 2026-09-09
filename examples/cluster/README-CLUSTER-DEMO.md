# M3.1 Cluster Demo: Multi-Instance Deduplication

This directory contains a working example of the **M3.1 Distributed Clustering** feature, demonstrating how two `dimd` instances can coordinate via a shared PostgreSQL backend to prevent duplicate message processing.

## What This Demonstrates

The cluster demo shows:
- ✅ **Multi-instance coordination** — Two dimd instances working together
- ✅ **Deduplication across instances** — Same message only processed once
- ✅ **Shared PostgreSQL backend** — Unified dedup and lineage storage
- ✅ **No single point of failure** — Active-active clustering (no leader)
- ✅ **Unified lineage** — All instances write to same audit trail

## Quick Start

### 1. Start the cluster
```bash
docker-compose -f docker-compose.cluster.yml up -d
```

### 2. Wait for instances to be ready
```bash
docker logs dim-instance-1 | grep "listening"
```

### 3. Run the test
```bash
bash test-cluster-dedup.sh
```

Expected output: `✓ SUCCESS: Order was processed ONLY ONCE`

## How It Works

**Message arrives at Instance 1** → Checks PostgreSQL dedup table → Not found → Process and store

**Same message arrives at Instance 2** → Checks PostgreSQL dedup table → Found → Skip (already processed)

Result: Message processed exactly once, coordinated across instances.

## Test Files

- `docker-compose.cluster.yml` — Orchestrates 2 instances + PostgreSQL
- `routes-cluster.yaml` — Route config with idempotent dedup step
- `init-cluster-db.sql` — PostgreSQL schema for dedup + lineage
- `test-cluster-dedup.sh` — Automated test showing dedup working

## Cleanup

```bash
docker-compose -f docker-compose.cluster.yml down -v
```

## Debugging

View PostgreSQL dedup table:
```bash
psql -h localhost -U dim -d dim_cluster
# SELECT * FROM dedup_store;
```

View instance logs:
```bash
docker logs dim-instance-1  # First instance
docker logs dim-instance-2  # Second instance
```
