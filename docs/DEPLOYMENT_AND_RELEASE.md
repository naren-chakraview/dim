# dim — Deployment and Release Guide

This guide covers how to test dim locally, create releases, and deploy to production.

---

## Overview

There are three main activities in the dim lifecycle:

1. **Local Development & Testing** — Test your routes with real external services (Kafka, PostgreSQL)
2. **Release Management** — Create semantic version releases that are gated by service health checks
3. **Production Deployment** — Run dim in your environment (Docker, Kubernetes, systemd, etc.)

---

## Part 1: Local Development & Integration Testing

### Testing Your Routes Locally

Before committing code, test your routes with production-like external services.

#### Quick Start: Full Integration Environment

```bash
# Start Kafka, PostgreSQL, Zookeeper with one command
scripts/e2e-test.sh
```

This script:
1. Starts all services via Docker Compose
2. Waits for services to become healthy
3. Verifies services are ready for your routes
4. Cleans up when done

See [E2E_TESTING.md](E2E_TESTING.md) for complete integration testing guide.

#### Manual Service Management

If you prefer to manage services yourself:

```bash
# Start services
cd deploy
docker-compose -f docker-compose.e2e.yml -p dim-e2e up -d

# Verify services are healthy
docker-compose -f docker-compose.e2e.yml -p dim-e2e ps

# Run your routes
./dimctl run my-route.yaml

# Stop services
docker-compose -f docker-compose.e2e.yml -p dim-e2e down -v
```

#### Testing Kafka Connectivity

```bash
# Produce a message
docker exec dim-e2e-kafka kafka-console-producer \
  --broker-list localhost:9092 \
  --topic test-topic \
  <<< '{"test": "message"}'

# Consume the message
docker exec dim-e2e-kafka kafka-console-consumer \
  --bootstrap-server localhost:9092 \
  --topic test-topic \
  --from-beginning \
  --max-messages 1
```

#### Testing Database Connectivity

```bash
# Connect to PostgreSQL
docker exec dim-e2e-postgres psql -U dim_test -d dim_e2e

# Or test via SQL
docker exec dim-e2e-postgres psql -U dim_test -d dim_e2e \
  -c "SELECT version();"
```

---

## Part 2: Release Management

### Creating a Release

Releases follow semantic versioning and are gated by service health checks.

#### Prerequisites

Before creating a release, verify everything is ready:

```bash
scripts/check-release-readiness.sh
```

This checks:
- ✓ Working directory is clean
- ✓ On main/master branch
- ✓ Up to date with remote
- ✓ All tests pass
- ✓ go vet passes
- ✓ Build succeeds for all platforms

#### Release Workflow

```
1. Create semantic version tag (v1.2.3)
   ↓
2. Push to GitHub
   ↓
3. GitHub Actions automatically runs:
   [Stage 1] Verify CI Status
   [Stage 2] Service Health Checks (Kafka, PostgreSQL, Zookeeper)
   [Stage 3] Build & Release
   ↓
4. Release published to GitHub Releases
```

#### Step-by-Step: Create a Release

```bash
# 1. Verify readiness
./scripts/check-release-readiness.sh

# 2. Create semantic version tag
git tag v0.8.0

# 3. Push tag to GitHub
git push origin v0.8.0

# 4. Watch release workflow
# Go to: https://github.com/naren-chakraview/dim/actions

# 5. Verify release artifacts
# Go to: https://github.com/naren-chakraview/dim/releases
```

#### Understanding Service Health Checks

The release workflow gates on service health checks:

- **Kafka:** `kafka-broker-api-versions --bootstrap-server localhost:9092`
  - Returns version info if Kafka is healthy
  - Health check every 5 seconds, timeout 30s, 20 retries

- **PostgreSQL:** `pg_isready -U dim_test -d dim_e2e`
  - Returns 0 if database is ready
  - Health check every 5 seconds, timeout 30s, 15 retries

- **Zookeeper:** TCP port 2181 connectivity
  - Returns 0 if port is responsive
  - Health check every 5 seconds, timeout 30s, 15 retries

If any service doesn't become healthy, the release workflow fails and blocks release creation.

### Semantic Versioning

Follow [semver.org](https://semver.org/):

- **MAJOR (X.0.0)** — Breaking changes to config, API, or behavior
  - Example: v1.0.0 → v2.0.0 (incompatible config format change)

- **MINOR (1.Y.0)** — New features, backward compatible
  - Example: v1.0.0 → v1.1.0 (new step type added)

- **PATCH (1.0.Z)** — Bug fixes, backward compatible
  - Example: v1.0.0 → v1.0.1 (fixed memory leak)

**Release Examples:**
```bash
git tag v0.7.0   # New features, backward compatible
git tag v0.7.1   # Bug fix, backward compatible
git tag v1.0.0   # Breaking change
```

### Troubleshooting Release Failures

**Problem:** "Tag format not valid"
```bash
# Must be v{MAJOR}.{MINOR}.{PATCH}
git tag v1.2.3    # ✓ Correct
git tag release-1.2.3  # ✗ Wrong (no 'v' prefix)
git tag v1.2      # ✗ Wrong (missing PATCH)
```

**Problem:** "Service health check timed out"
- Services take 30-60 seconds to start
- If services are timing out, there may be an infrastructure issue
- Retry the release (delete tag and push again)
- Check GitHub Actions logs for Docker errors

**Problem:** "Build failed for platform X"
- Check the GoReleaser logs in GitHub Actions
- Common issues: missing dependencies, platform-specific code
- Fix the issue and try again with a new tag

### Release Rollback

If a released version has critical issues:

1. Fix the issue in a new commit
2. Create a new release with incremented version
3. **Never re-release or modify an existing tag** (releases are immutable)

```bash
# Example: v1.0.0 had critical bug
git tag v1.0.1     # New release with fix
git push origin v1.0.1
```

---

## Part 3: Production Deployment

### Running dim as a Daemon

#### With systemd (Linux)

```bash
# 1. Build the binary
go build ./cmd/dimd -o dimd

# 2. Copy to bin directory
sudo cp dimd /usr/local/bin/

# 3. Create systemd service file
sudo tee /etc/systemd/system/dim.service > /dev/null <<EOF
[Unit]
Description=dim Integration Middleware
After=network.target

[Service]
Type=simple
User=dim
ExecStart=/usr/local/bin/dimd -config /etc/dim/config.yaml
Restart=on-failure
RestartSec=10

[Install]
WantedBy=multi-user.target
EOF

# 4. Start the service
sudo systemctl daemon-reload
sudo systemctl start dim
sudo systemctl enable dim

# 5. Check status
sudo systemctl status dim
```

#### With Docker

```bash
# 1. Build image
docker build -t dim:v0.8.0 -f Dockerfile .

# 2. Run container
docker run -d \
  --name dim-prod \
  -v /etc/dim/config.yaml:/etc/dim/config.yaml \
  -p 8080:8080 \
  -p 8081:8081 \
  dim:v0.8.0

# 3. Check logs
docker logs dim-prod

# 4. Stop container
docker stop dim-prod
```

#### With Kubernetes

```bash
# 1. Create ConfigMap with your route
kubectl create configmap dim-route --from-file=config.yaml

# 2. Create deployment
kubectl apply -f deployment.yaml

# 3. Expose service
kubectl expose deployment dim --type=LoadBalancer --port=8080
```

See [CLI_REFERENCE.md](CLI_REFERENCE.md#dimd-daemon) for complete daemon documentation.

### Configuration Management

#### Directory Structure

```
/etc/dim/
├── config.yaml           # Main route configuration
├── fragments/            # Reusable configuration fragments
│   ├── auth-rules.yaml
│   └── retry-policy.yaml
└── secrets/              # Secret provider configuration
    └── vault.env         # Environment variables for secret resolution
```

#### Environment Variables

```bash
# Secret provider
export SECRET_PROVIDER=env          # or 'vault', 'file'
export VAULT_ADDR=https://vault.example.com
export VAULT_TOKEN=s.xxxxx

# Observability
export OTEL_EXPORTER_OTLP_ENDPOINT=localhost:4317
export PROMETHEUS_METRICS_PORT=9090

# Logging
export LOG_LEVEL=info               # debug, info, warn, error
export LOG_FORMAT=json              # json or text
```

#### Secrets Management

Never hardcode secrets in configuration:

```yaml
# ✗ Wrong: Secret in config file
sinks:
  database:
    connection_string: "postgres://user:password@host:5432/db"

# ✓ Correct: Secret resolved at runtime
sinks:
  database:
    connection_string: "${SECRET:DATABASE_URL}"
```

Secrets are resolved at startup from your secret provider:

```bash
# Environment variables
export DATABASE_URL="postgres://user:password@host:5432/db"

# Or HashiCorp Vault
vault kv put secret/dim/database url="..."
```

### Monitoring & Observability

#### Prometheus Metrics

Metrics available at `http://localhost:9090/metrics`:

```
dim_messages_processed_total           # Messages by route/outcome
dim_message_latency_seconds            # Latency histogram
dim_sink_writes_total                  # Writes by sink type
dim_authorization_decisions_total      # Auth decisions (allow/deny)
dim_lineage_records_stored_total       # Lineage records
```

#### OpenTelemetry Tracing

Traces exported to OTLP endpoint:

```bash
# Point to your tracing backend
./dimd -config config.yaml \
  --tracing-endpoint localhost:4317
```

#### Live Viewer

Real-time route statistics:

```bash
# Open browser to:
http://localhost:8081/debug/routes

# Or query via API
curl http://localhost:8081/debug/routes | jq
```

### Performance Tuning

#### Worker Pool Configuration

```yaml
routes:
  my-route:
    # Number of parallel workers per route
    workers: 10
    
    # Max messages in flight
    max_inflight: 1000
    
    # Buffer overflow policy
    overflow_policy: block  # or 'drop'
```

#### Message Channel Sizing

```yaml
routes:
  my-route:
    # Channel buffer size between pipeline stages
    channel_buffer: 100
```

#### Retention & Purge

```yaml
routes:
  my-route:
    lineage:
      retention_policy: pci  # Named policy
      subject_id_expr: body.customer_id
      
# Policies configured per instance
purge_policies:
  pci:
    ttl: 7d        # Keep records 7 days
    batch_size: 10000
    flush_interval: 5m
```

---

## Complete Workflow Example

Here's a typical complete workflow:

```bash
# 1. Develop locally
./dimctl validate my-route.yaml
./dimctl run my-route.yaml

# 2. Test with external services
scripts/e2e-test.sh
# ... verify route works with Kafka, PostgreSQL ...

# 3. Commit code
git add my-route.yaml
git commit -m "Add order processing route"
git push origin feature/order-processing

# 4. Create PR, get review, merge to master

# 5. Prepare release
scripts/check-release-readiness.sh

# 6. Create semantic version tag
git tag v0.8.0

# 7. Push tag to GitHub
git push origin v0.8.0

# 8. Watch release workflow at GitHub Actions
# Verify:
# - ✓ CI validation passes
# - ✓ Service health checks pass
# - ✓ Build succeeds for all platforms

# 9. Deploy to production
docker run -d \
  -v /etc/dim/config.yaml:/etc/dim/config.yaml \
  dim:v0.8.0

# 10. Verify production is healthy
curl http://localhost:8081/debug/routes
```

---

## Related Documentation

- **[E2E_TESTING.md](E2E_TESTING.md)** — Integration testing with Docker services
- **[CLI_REFERENCE.md](CLI_REFERENCE.md)** — dimctl and dimd command reference
- **[LANGUAGE_REFERENCE.md](LANGUAGE_REFERENCE.md)** — Route configuration reference
- **[../RELEASE.md](../RELEASE.md)** — Complete release process documentation (for maintainers)
- **[../OKF.md](../OKF.md)** — Architectural decisions and patterns

---

## Questions or Issues?

- **Release workflow failing?** See troubleshooting section above or check GitHub Actions logs
- **Service won't start?** See [E2E_TESTING.md](E2E_TESTING.md) troubleshooting section
- **Production issue?** Check logs and use `dimctl provenance` to trace message journey
- **Questions?** File an issue on GitHub or start a discussion

---

Happy deploying! 🚀
