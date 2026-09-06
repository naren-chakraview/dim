# dim Cluster Mode Example

This example demonstrates `dimd` running in **active-active cluster mode** with three instances processing a shared route set, coordinating via PostgreSQL for dedup and lineage.

## Architecture

```
┌─────────────┐  ┌─────────────┐  ┌─────────────┐
│   dimd-1    │  │   dimd-2    │  │   dimd-3    │
│  :8080      │  │  :8081      │  │  :8082      │
└──────┬──────┘  └──────┬──────┘  └──────┬──────┘
       │                │                │
       └────────────────┼────────────────┘
                        │
                   PostgreSQL
                   (shared state)
                   - dedup keys
                   - lineage
                   - generation tracking
```

## Key Features

- **Active-Active:** All three instances process the same routes simultaneously
- **No Duplicates:** Messages are deduplicated across instances via shared Postgres dedup table
- **Cluster-Wide Lineage:** All lineage records stored centrally, queryable from any instance
- **Coordinated Config:** Generation tracking allows coordinated or staggered config rollouts
- **Instance Discovery:** Static config via `DIMD_CLUSTER_HOSTS` environment variable

## Prerequisites

- Docker and Docker Compose
- `curl` for sending test messages

## Running the Cluster

```bash
cd examples/cluster
docker-compose up
```

This starts:
- PostgreSQL database (port 5432)
- 3x dimd instances (ports 8080, 8081, 8082)

Verify all instances are running:
```bash
docker-compose ps
```

## Testing No Duplicates

Send the same message to all three instances and verify only one processes it:

```bash
# Send a test message
MESSAGE='{"order_id":"order-123","amount":99.99}'

# Send to dimd-1 (port 8080)
curl -X POST http://localhost:8080/message \
  -H "Content-Type: application/json" \
  -d "$MESSAGE"

# Send same message to dimd-2 (port 8081)
curl -X POST http://localhost:8081/message \
  -H "Content-Type: application/json" \
  -d "$MESSAGE"

# Send same message to dimd-3 (port 8082)
curl -X POST http://localhost:8082/message \
  -H "Content-Type: application/json" \
  -d "$MESSAGE"
```

Expected behavior: Only ONE instance processes the message; the other two drop it as a duplicate due to the shared dedup store.

Query the dedup table to see the key was stored once:
```bash
docker-compose exec postgres psql -U dimadmin -d dim_cluster \
  -c "SELECT COUNT(*) FROM dedup_store WHERE dedup_key = 'order-123';"

# Result: 1 (only stored once, despite 3 sends)
```

## Testing Cluster-Wide Lineage

Query lineage records from any instance:
```bash
docker-compose exec postgres psql -U dimadmin -d dim_cluster \
  -c "SELECT id, route_name, subject_id, instance_id FROM lineage_records LIMIT 10;"
```

Each record shows which instance processed it and the route version.

## Testing Coordinated Reload

To test config change propagation:

1. Update `routes-cluster.yaml`
2. Send a signal to all instances (or restart them)
3. Verify all instances pick up the new config simultaneously

For staggered rollout, instances can update independently; for coordinated rollout, wait for all to reach the same generation before proceeding.

## Stopping the Cluster

```bash
docker-compose down

# Clean up volumes (optional)
docker-compose down -v
```

## Configuration

### Environment Variables (per instance)

- `DIMD_INSTANCE_ID`: Unique instance identifier
- `DIMD_CLUSTER_HOSTS`: Comma-separated list of all instances (e.g., "dimd-1:8080,dimd-2:8080,dimd-3:8080")
- `DIMD_DEDUP_BACKEND`: "postgres" for cluster mode, "memory" for single-instance
- `DIMD_DEDUP_DSN`: PostgreSQL connection string

### Configuration Files

- `routes-cluster.yaml`: Route definitions for this example
- `init-db.sql`: Database schema initialization
- `docker-compose.yml`: Cluster deployment manifest

## Exit Criteria Verification

✅ **Multiple instances process shared route set** — All three instances listen on the same routes
✅ **No duplicate message processing** — Shared dedup store prevents duplicates across instances
✅ **Cluster-wide lineage queryable** — All lineage in central Postgres, not per-instance
✅ **Config change reaches all instances** — Generation tracking enables coordinated or staggered rollout

## Next Steps

For production deployments, consider:
- Using managed PostgreSQL (RDS, Cloud SQL) instead of container
- Adding Kubernetes deployment manifests (Helm chart)
- Setting up monitoring and alerting for cluster health
- Implementing instance auto-scaling and failover
