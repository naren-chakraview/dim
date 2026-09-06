# End-to-End Testing Guide

This document describes the DIM e2e testing infrastructure, including how to run tests locally and how releases are gated on e2e test success.

## Overview

The e2e test suite verifies DIM's integration with external dependencies:
- **Kafka** (message broker)
- **PostgreSQL** (relational database)
- **MinIO** (S3-compatible object storage)
- **Prometheus** (metrics collection)

Each test validates a key workflow or feature in a production-like environment.

## Running E2E Tests Locally

### Quick Start

```bash
# Run all e2e tests (starts Docker services automatically)
scripts/e2e-test.sh

# This will:
# 1. Start all required services via docker-compose
# 2. Wait for services to be healthy
# 3. Run the full e2e test suite
# 4. Clean up services
```

### Manual Testing

If you prefer to manage services manually:

```bash
# Start services
cd deploy
docker-compose -f docker-compose.e2e.yml -p dim-e2e up -d

# Wait for services to be ready (watch health status)
docker-compose -f docker-compose.e2e.yml -p dim-e2e ps

# Run tests with the e2e build tag
go test -tags e2e -v ./internal/integration/... -run "E2E"

# Stop services
docker-compose -f docker-compose.e2e.yml -p dim-e2e down -v
```

### Test Output

When tests pass:
```
=== RUN   TestE2EKafkaIntegration
--- PASS: TestE2EKafkaIntegration (1.23s)
✓ Kafka integration test passed

=== RUN   TestE2EDatabaseIntegration
--- PASS: TestE2EDatabaseIntegration (0.45s)
✓ Database integration test passed

...
PASS
ok  	github.com/naren-chakraview/dim/internal/integration	12.34s
```

## E2E Test Suite

### TestE2EKafkaIntegration
Verifies Kafka message broker connectivity and basic publish/subscribe:
- Creates a test topic
- Writes a message
- Reads the message back
- Validates message integrity

**Coverage:** Message queue reliability, topic management

### TestE2EDatabaseIntegration
Verifies PostgreSQL connectivity and basic operations:
- Creates a test table
- Inserts records
- Queries data back
- Validates JSONB support

**Coverage:** Data persistence, SQL queries, JSON storage

### TestE2ES3Integration
Verifies S3-compatible object storage (MinIO):
- Creates a bucket
- Uploads an object
- Downloads the object
- Validates data integrity

**Coverage:** Large payload storage, object retrieval

### TestE2EClaimCheckPattern
Verifies the Claim-Check EIP pattern with S3 backend:
- Stores a large payload in S3
- Retrieves it using a ticket reference
- Validates that the payload is intact

**Coverage:** Claim-check pattern, large message handling

### TestE2EMessageFlow
Verifies end-to-end message processing through the engine:
- Creates an engine with executor
- Sends test messages through channels
- Receives processed messages
- Validates message integrity

**Coverage:** Engine architecture, message processing pipeline

### TestE2EMultiTenantIsolation
Verifies multi-tenant resource isolation with database backend:
- Creates tenant-aware tables
- Inserts messages for multiple tenants
- Verifies each tenant only sees their own data
- Validates isolation guarantees

**Coverage:** Multi-tenancy, data isolation

## Docker Services

The e2e environment (`docker-compose.e2e.yml`) includes:

| Service | Port | Purpose |
|---------|------|---------|
| Kafka | 9092 | Message broker |
| Zookeeper | 2181 | Kafka coordination |
| PostgreSQL | 5432 | Relational database |
| MinIO | 9000 | S3-compatible storage |
| Prometheus | 9090 | Metrics collection |

All services include health checks to ensure readiness before tests start.

### Service Health Checks

Services are considered ready when:
- **Kafka:** `kafka-broker-api-versions` succeeds
- **PostgreSQL:** `pg_isready` succeeds
- **MinIO:** HTTP health endpoint responds
- **Zookeeper:** TCP port is open and responsive
- **Prometheus:** HTTP health endpoint responds

## CI/CD Integration

### GitHub Actions Workflows

#### E2E Tests on Every Push
**File:** `.github/workflows/e2e.yml`
- Runs on: Push to master/main, Pull Requests
- Uses: GitHub-hosted runners with Docker services
- Duration: ~5-10 minutes

#### Release Gate: E2E Tests Required
**File:** `.github/workflows/release.yml`

Releases are only created if:
1. ✓ Tag format is valid (semantic version: v1.2.3)
2. ✓ All CI checks pass on the tagged commit
3. ✓ **E2E tests pass on the release commit**
4. ✓ Build succeeds for all platforms

```yaml
jobs:
  verify-ci-status:      # Step 1: Validate tag
  e2e-tests:             # Step 2: Run e2e suite (required for release)
  release:               # Step 3: Create release (only if above pass)
```

## Creating a Release

### Prerequisites

Before creating a release tag:

```bash
# Verify everything is ready
scripts/check-release-readiness.sh
```

This checks:
- ✓ Working directory is clean
- ✓ On master/main branch
- ✓ Up to date with remote
- ✓ All tests pass
- ✓ go vet passes
- ✓ Build succeeds for all platforms

### Release Process

```bash
# 1. Verify readiness
scripts/check-release-readiness.sh

# 2. Create semantic version tag
git tag v1.2.3

# 3. Push tag to remote
git push origin v1.2.3

# 4. Monitor the release workflow
# - GitHub Actions automatically triggers on tag push
# - Release workflow runs e2e tests as a gate
# - If tests pass, GoReleaser creates the release
# - Release appears on GitHub Releases page

# 5. Verify release artifacts
# - Check GitHub Releases for binaries
# - Verify checksums
# - Validate that you can install from release
```

### Release Workflow Diagram

```
Tag pushed (git push origin v1.2.3)
        ↓
[Verify CI Status] ← Tag format, branch validation
        ↓
[E2E Tests] ← Full test suite with Docker services
        ↓
[Create Release] ← GoReleaser generates binaries
        ↓
GitHub Releases with downloadable artifacts
```

## Troubleshooting

### Services Don't Start

```bash
# Check Docker daemon is running
docker ps

# Check images are available
docker images | grep -E "kafka|postgres|minio|prometheus"

# Pull missing images
docker pull confluentinc/cp-kafka:7.5.0
docker pull postgres:16-alpine
docker pull minio/minio:latest
docker pull prom/prometheus:v2.40.0
```

### Tests Timeout

```bash
# Increase test timeout in docker-compose
# Services take ~30-60s to become healthy depending on system load

# Or run with verbose logging
docker-compose -f deploy/docker-compose.e2e.yml -p dim-e2e logs -f

# Check individual service status
docker-compose -f deploy/docker-compose.e2e.yml -p dim-e2e ps
```

### Database Connection Failed

```bash
# Verify PostgreSQL is ready
docker exec dim-e2e-postgres pg_isready -U dim_test -d dim_e2e

# Check credentials (see docker-compose.e2e.yml)
# Default: user=dim_test, password=test_password, db=dim_e2e

# Connect directly to verify
psql -h localhost -U dim_test -d dim_e2e
```

### Kafka Connection Failed

```bash
# Verify Kafka is ready
docker exec dim-e2e-kafka kafka-broker-api-versions --bootstrap-server localhost:9092

# Check Kafka logs
docker logs dim-e2e-kafka | tail -50

# Verify Zookeeper is running
docker logs dim-e2e-zookeeper
```

### MinIO Connection Failed

```bash
# Verify MinIO is ready
curl -I http://localhost:9000/minio/health/live

# Check MinIO logs
docker logs dim-e2e-minio

# Access MinIO console (for debugging)
# Open http://localhost:9001 (user: minioadmin, pass: minioadmin)
```

## Performance Notes

- **First run:** ~60-90 seconds (includes pulling images)
- **Subsequent runs:** ~30-45 seconds (services already cached)
- **Test duration:** ~10-15 seconds per test
- **Total with services:** ~45-60 seconds

To speed up iteration during development:

```bash
# Keep services running between test runs
docker-compose -f deploy/docker-compose.e2e.yml -p dim-e2e up -d

# Run just your test
go test -tags e2e -v -run TestE2EYourTest ./internal/integration

# Services remain available for the next test
```

## Contributing

When adding new e2e tests:

1. Add test to `internal/integration/e2e_test.go`
2. Use `//go:build e2e` tag
3. Follow naming: `TestE2E*`
4. Include health checks before using services
5. Document what the test covers
6. Add to this guide

Example:

```go
//go:build e2e

func TestE2EYourFeature(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping e2e test in short mode")
	}
	
	// Test your feature
}
```

## See Also

- [Release Workflow](../.github/workflows/release.yml)
- [E2E Workflow](../.github/workflows/e2e.yml)
- [Docker Compose Config](../deploy/docker-compose.e2e.yml)
- [Pre-Release Checklist Script](../scripts/check-release-readiness.sh)
