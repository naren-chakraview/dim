# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.5.0] - 2026-09-03

### Added - Phase 0 Complete

#### Core Pipeline (M0.1)
- HTTP message ingestion endpoint (`POST /message`)
- Single-worker executor with configurable pipeline steps
- Step types: `filter` (JSONata), `translate` (JSONata), `route`, `idempotent`
- Error handling with dead-letter queue (DLQ)
- Message correlation and tracing IDs
- YAML configuration validation with JSON Schema
- CLI: `midctl validate` and `midctl run` commands
- 120+ unit tests with 99.2% pass rate

#### Reliability & Concurrency (M0.2)
- Worker pool architecture with configurable concurrency
- Exponential backoff retry logic for transient failures
- Configuration composition with fragment merging
- Route versioning with deterministic hashing
- Hot reload (SIGHUP) with graceful message draining
- Idempotent deduplication (in-memory, 60-second TTL)
- Ordered message processing by correlation ID

#### Governance & Security (M0.3)
- JWT principal propagation from `Authorization: Bearer` headers
- Role-Based Access Control (RBAC) with configurable allowed roles
- Attribute-Based Access Control (ABAC) with JSONata expressions
- Authorization step in pipeline with policy enforcement
- Data contracts with JSON Schema validation
- Contract violation handling with `on_violation` routing
- Strict mode for mandatory authorization
- Contract version tracking per route

#### Lineage & Audit (M0.4)
- SQLite-backed message lineage store with WAL mode
- Message provenance tracking (subject_id, principal, route, timestamp)
- Static and dynamic retention policies
- Automatic reaper daemon for policy-based purging
- Manual purge with evidence logging (tamper-proof)
- CSV and NDJSON export for audit compliance
- Lineage queries by message ID or subject ID
- Purge logs separate from lineage (immutable audit trail)

#### Observability & Monitoring (M0.5)
- OpenTelemetry distributed tracing with OTLP exporters
- Prometheus metrics collection (throughput, latency, errors)
- Built-in live viewer endpoint (`/debug/routes`)
- Grafana dashboard examples for visualization
- `midctl trace tail` for live span streaming
- Configurable sampling rate for tracing

#### Hardening & Release (M0.6)
- Comprehensive security audit (no critical vulnerabilities)
- Performance benchmarks (1,245+ msg/sec, p99: 45ms)
- Security review document with compliance mapping
- Performance results with detailed metrics
- Release notes and deployment checklist
- Updated README with quick start guide

### Security

- Input validation for JWT headers, YAML configs, JSON messages
- No hardcoded secrets (all via environment variables)
- Parameterized SQL queries (SQLi prevention)
- JSON-only processing (XXE prevention)
- No sensitive data leakage in logs or errors
- Dependency audit with `go mod verify` (no CVEs)
- Authentication/authorization flow verification
- Audit trail integrity with tamper-evident logging

### Performance

- **Throughput:** 1,245+ msg/sec on single route (3-step pipeline)
- **Latency (p99):** 45ms through full pipeline with tracing
- **Contract validation:** 0.62-1.42ms per validation
- **Tracing overhead:** <1% latency impact
- **Authorization (RBAC):** 0.18ms
- **Authorization (ABAC):** 0.65ms
- **Channel operations:** <0.02ms

### Testing

- 440+ unit and integration tests
- All tests passing with `-race` flag (no race conditions)
- Full end-to-end coverage with fixtures
- Performance benchmarks with detailed metrics
- Stress tested at 2× sustained load (2000+ msg/sec)

### Compatibility

- **Go version:** 1.26.7+
- **Platforms:** Linux, macOS, Windows (WSL2)
- **API stability:** Stable for Phase 0 (no breaking changes expected)

### Documentation

- User Guide (installation, quick start, project structure)
- Development Guide (patterns, testing, contributing)
- Architecture documentation (EIP mapping, design principles)
- Security review with compliance framework analysis
- Performance results with production recommendations
- Configuration examples (basic, contracts, lineage, observability)
- Deployment checklist for production

### Known Limitations

- Single-process deployment (horizontal scale via load balancer)
- In-memory idempotent store (60-second TTL, not persistent)
- No built-in TLS (deploy behind reverse proxy)
- File sinks only (Kafka, AMQP coming in Phase 1)
- Hot reload cannot change step types
- Deterministic ordering only (single instance)

---

## [0.1.0] - 2026-08-01

### Added - Initial Release (Walking Skeleton)

#### Core Architecture
- Message envelope with headers, body, and metadata
- Pipeline-based step execution model
- Channel-based communication between components
- Bounded channels with configurable buffer sizes
- Error handling with dead-letter queue pattern

#### Step Types
- `filter` step with JSONata boolean expressions
- `translate` step with JSONata object transformation
- Error path with automatic retry

#### Adapters
- HTTP source adapter (port 8080)
- File sink adapter (JSONL output)
- DLQ sink adapter

#### Configuration
- YAML-based route configuration
- JSON Schema validation for routes
- Configuration loader with error handling

#### Expression Evaluation
- JSONata integration (https://github.com/blues/jsonata-go)
- Support for 90% of JSONata specification
- Pluggable function registry (for future extensions)

#### CLI
- `midctl validate` — validate route YAML
- `midctl run` — start middleware with route configuration

#### Testing
- 120+ unit tests
- Message processing fixtures
- Configuration validation tests
- Expression evaluation tests

### Documentation
- README with architecture overview
- Project structure guide
- Development setup instructions

---

## [0.6.0-beta] - 2026-09-03 (Phase 1 Track A + Track B Adapters)

### Added - Phase 1 Infrastructure & Ecosystem Adapters

#### Track A: Governance & Infrastructure (Complete)
- **R6:** File source adapter with polling, modification-time deduplication, optional archive
- **R7:** Real OpenTelemetry SDK integration with OTLP gRPC export, sampler config
- **R8:** Prometheus metrics with proper summary type (histogram → summary)
- **R9:** dimd daemon engine with graceful shutdown, observability init, lifecycle
- **R10:** CLI tool renamed from midctl to dimctl with full reference updates
- **R11:** ORDERING_FEATURE.md documentation (keeping internal/ordering)
- **R12:** CODE_REVIEW_WORKFLOW.md (3-tier review), REVIEWERS.md (expertise map)
- **R13:** Build health verification and Phase 1 readiness testing
- **R14:** dimctl CLI finalization with validate, run, test, lineage, trace, provenance

#### Track B: Ecosystem Adapters (In Progress)
- **Kafka adapter (PR #14, merged):**
  - Consumer groups with offset tracking per partition
  - Configurable compression and ACKs, metadata headers
  - Integration tests with broker detection

- **AMQP adapter (PR #15, merged):**
  - Queue-based consumption with manual ack
  - Configurable exchange types, metadata headers
  - Integration tests with RabbitMQ

- **S3 adapter (PR #16, merged):**
  - Bucket polling with LastModified tracking
  - JSON serialization, metadata headers
  - Configurable bucket/region/prefix

- **Adapter interface spec (PR #17, merged):**
  - Source/Sink interfaces matching design spec §8
  - HealthCheck(), Checkpoint(), Write() → []Result
  - ADAPTER_SPEC.md with patterns and contract

### Documentation
- ADAPTER_SPEC.md: interface contract and implementation patterns
- CODE_REVIEW_WORKFLOW.md: 3-tier governance process
- REVIEWERS.md: subsystem expertise mapping
- OKF.md: Phase 1 progress tracking
- Updated CHANGELOG with Phase 0.5 and 0.6-beta entries

## Unreleased (Phase 1+)

### Planned Features
- Database adapters (CDC, JDBC)
- Replay tooling
- PBAC/OPA reference implementation
- OBO token exchange
- OpenLineage export with Marquez
- Schema registry integration
- Event sourcing patterns
- Persistent idempotent store (RocksDB)
- Built-in TLS listener
- Distributed consensus for ordering
- WASM expression sandbox
- GraphQL API over lineage
- Kubernetes operator

---

## Versioning

### Version Format
`v<major>.<minor>.<patch>[-<prerelease>][+<build>]`

### Release Schedule
- **Phase 0 (M0.1–M0.6):** v0.1–v0.5 (2026-08 to 2026-09)
- **Phase 1:** v0.6–v0.9 (2026-10 to 2026-12)
- **v1.0:** 2027-Q1 (production hardened)

### Support Policy
- Latest minor version receives security patches
- Critical bugs fixed in previous minor versions
- Go version upgrades supported for latest 2 minor versions
