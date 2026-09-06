# dim Documentation

Welcome! This folder contains user-facing guides for running dim.

---

## I'm New to dim

Start here:

1. **[GETTING_STARTED.md](GETTING_STARTED.md)** (5 min)
   - What is dim?
   - Build and install
   - Your first route
   - Common patterns

2. **[LANGUAGE_REFERENCE.md](LANGUAGE_REFERENCE.md)** (30 min)
   - Configuration structure
   - All source/sink types
   - All step types
   - Expression syntax (JSONata)
   - Secrets, parameters, fragments

3. **[USE_CASES.md](USE_CASES.md)** (20 min)
   - E-commerce order processing
   - Financial compliance
   - Multi-source data lake
   - Real-time notifications
   - CDC and database sync
   - Rate limiting
   - Log aggregation
   - Access control & redaction
   - Message replay
   - Schema validation

4. **[DEPLOYMENT_AND_RELEASE.md](DEPLOYMENT_AND_RELEASE.md)** (15 min)
   - Local integration testing
   - Creating releases
   - Production deployment
   - Monitoring & observability

5. **[PHASE3_FEATURES.md](PHASE3_FEATURES.md)** (30 min)
   - Claim Check Pattern (large payloads)
   - Plugin SDK (custom functions)
   - Multi-Tenant Resource Isolation (SaaS)
   - Distributed Clustering (high availability)

---

## I Want to...

### ...handle large message attachments

→ **[PHASE3_FEATURES.md](PHASE3_FEATURES.md#feature-1-claim-check-pattern-m32)**

Features:
- Store large payloads (5MB+ files) externally
- Lightweight tickets travel through pipeline
- Retrieve payloads only when needed
- Production-ready with S3 backend

### ...write custom validation or transformation logic

→ **[PHASE3_FEATURES.md](PHASE3_FEATURES.md#feature-2-plugin-sdk-m33)**

Features:
- Native Go plugins (fast, subprocess-based)
- WebAssembly plugins (secure, sandboxed)
- Call from JSONata expressions
- Real-world examples included

### ...build multi-tenant SaaS with resource isolation

→ **[PHASE3_FEATURES.md](PHASE3_FEATURES.md#feature-3-multi-tenant-resource-isolation-m34)**

Features:
- Per-tenant message rate limits
- Worker pool partitioning
- Lineage storage quotas
- Prevent noisy neighbor problems

### ...scale horizontally across multiple instances

→ **[PHASE3_FEATURES.md](PHASE3_FEATURES.md#feature-4-distributed-clustering-m31)**

Features:
- Active-active clustering
- Shared state (Postgres/Redis)
- Unified lineage across cluster
- No duplicate processing

### ...deploy to production

→ **[DEPLOYMENT_AND_RELEASE.md](DEPLOYMENT_AND_RELEASE.md)**

Covers:
- Running dim as a daemon (systemd, Docker, Kubernetes)
- Configuration management and secrets
- Monitoring with Prometheus and OpenTelemetry
- Complete workflow from development to production

### ...understand the language and configuration

→ **[LANGUAGE_REFERENCE.md](LANGUAGE_REFERENCE.md)**

Covers:
- Config file structure
- Sources (http, file, kafka, amqp, s3, sftp, database)
- Sinks (file, http, kafka, amqp, s3, sftp, database)
- Steps (filter, translate, route, authorize, aggregate, split)
- Error handling and retry
- Lineage and retention
- JSONata expressions
- Secrets and parameters

### ...see working examples

→ **[USE_CASES.md](USE_CASES.md)**

Real-world solutions for:
- E-commerce workflows
- Payment processing
- Data lake ingestion
- Notifications
- Database synchronization
- Rate limiting
- Log aggregation
- Access control
- Replay patterns
- Schema validation

### ...learn CLI commands

→ **[CLI_REFERENCE.md](CLI_REFERENCE.md)**

Reference for:
- `dimctl validate` — Validate config
- `dimctl run` — Run a route
- `dimctl test` — Run fixture tests
- `dimctl lineage` — Query/export audit trail
- `dimctl provenance` — Trace message journey
- `dimctl explain` — Visualize DAG
- `dimctl trace` — Stream traces
- `dimctl replay` — Replay messages
- `dimd` daemon — Run as a service

### ...test integration with external services

→ **[E2E_TESTING.md](E2E_TESTING.md)**

Local integration testing:
- Start Kafka, PostgreSQL, Zookeeper via Docker Compose
- Verify connectivity to external services
- Test message flow end-to-end
- Health check validation
- Troubleshooting guide

### ...set up alerts and monitoring

→ **[CLI_REFERENCE.md](CLI_REFERENCE.md#dimctl-stats)** and **[CLI_REFERENCE.md](CLI_REFERENCE.md#dimctl-trace)**

Use:
- `dimctl stats` — Real-time metrics
- `dimctl trace tail` — Stream OpenTelemetry traces
- Prometheus metrics endpoint
- Built-in `/debug/routes` viewer

### ...understand access control

→ **[LANGUAGE_REFERENCE.md](LANGUAGE_REFERENCE.md#authentication)** and **[USE_CASES.md](USE_CASES.md#access-control-and-redaction)**

Features:
- RBAC (role-based access control)
- ABAC (attribute-based)
- PDP (policy decision point)
- Field redaction and obligations

### ...set up audit and compliance

→ **[LANGUAGE_REFERENCE.md](LANGUAGE_REFERENCE.md#lineage--retention)** and **[CLI_REFERENCE.md](CLI_REFERENCE.md#dimctl-lineage)**

Use:
- Lineage tracking (SQLite audit trail)
- Retention policies
- Purge evidence
- Export for audit
- GDPR compliance patterns

### ...troubleshoot or debug

→ **[CLI_REFERENCE.md](CLI_REFERENCE.md#debugging)**

Commands:
- `dimctl explain` — Visualize route DAG
- `dimctl trace tail` — See live execution
- `dimctl provenance` — Trace a message
- `dimctl lineage query` — Find messages
- `dimctl stats` — Monitor performance

---

## For Developers

If you're extending dim or contributing code:

→ **[../DEVELOPER_GUIDE.md](../DEVELOPER_GUIDE.md)**

Covers:
- Project structure
- Building and testing
- Adding new steps
- Adding new adapters
- JSONata expressions
- Debugging techniques
- Architecture decisions (OKF.md)

---

## For Operators/DevOps

Running dim in production and managing releases:

**Daemonization & Operations:**
→ **[CLI_REFERENCE.md](CLI_REFERENCE.md#dimd-daemon)** (daemonization and systemd)

**Release Management:**
→ **[../RELEASE.md](../RELEASE.md)**
- Creating releases with semantic versioning
- 3-stage release workflow (verify CI → service health checks → build & release)
- Release troubleshooting guide
- Pre-release validation checklist

**Integration Testing & Service Health:**
→ **[E2E_TESTING.md](E2E_TESTING.md)**
- Local integration environment setup
- Service health check configuration
- Verifying production-like external services

**Additional Resources:**
- [RELEASE_NOTES_v0.5.0.md](../RELEASE_NOTES_v0.5.0.md) — Deployment checklist
- [../ADAPTER_SPEC.md](../ADAPTER_SPEC.md) — Adapter patterns
- [../OKF.md](../OKF.md) — Architecture patterns

---

## How to Navigate

| I want to... | Start with... | Then read... |
|---|---|---|
| Get started quickly | GETTING_STARTED | LANGUAGE_REFERENCE |
| See real examples | USE_CASES | LANGUAGE_REFERENCE |
| Learn config syntax | LANGUAGE_REFERENCE | USE_CASES for patterns |
| Use CLI tools | CLI_REFERENCE | Examples for each command |
| Test with external services | E2E_TESTING | DEPLOYMENT_AND_RELEASE (local testing) |
| Debug a problem | CLI_REFERENCE (debugging) | Provenance, trace tail, explain |
| Set up compliance | LANGUAGE_REFERENCE (lineage) | USE_CASES (compliance example) |
| Monitor performance | CLI_REFERENCE (stats, trace) | DEPLOYMENT_AND_RELEASE (production) |
| Deploy to production | DEPLOYMENT_AND_RELEASE | CLI_REFERENCE (daemon) |
| Create a release | DEPLOYMENT_AND_RELEASE | E2E_TESTING (service health) |
| Contribute code | ../DEVELOPER_GUIDE | OKF.md for patterns |

---

## Common Questions

**Q: What's the difference between dim and Kafka/RabbitMQ?**

A: dim is an *integration middleware* — it routes and transforms messages *between* systems. Kafka and RabbitMQ are *brokers* — they store and distribute messages. dim can *use* either as a source or sink.

**Q: Do I need to write code?**

A: No. Routes are pure YAML configuration. Expressions use JSONata (not imperative code).

**Q: Can I run dim in Kubernetes?**

A: Yes. dim compiles to a single binary with no external dependencies (SQLite is embedded). Use the `dimd` daemon in a container.

**Q: What about schema evolution?**

A: dim validates messages against JSON Schema contracts at runtime. Contracts can be stored inline or in a registry (Apicurio, Confluent). See [LANGUAGE_REFERENCE.md](LANGUAGE_REFERENCE.md) (contract validation).

**Q: How do I audit who accessed what?**

A: Routes emit audit trail (lineage) to SQLite automatically. Every message carries principal (user), authorization decision, and lineage facets. Export to CSV for analysis. See [CLI_REFERENCE.md](CLI_REFERENCE.md#dimctl-lineage).

**Q: Can I replay messages?**

A: Yes. Export messages from lineage, then replay with `dimctl replay`. See [CLI_REFERENCE.md](CLI_REFERENCE.md#dimctl-replay) and [USE_CASES.md](USE_CASES.md#message-replay-and-recovery).

---

## Learning Path (Recommended)

1. **Day 1:** [GETTING_STARTED.md](GETTING_STARTED.md) (15 min)
   - Build the binary
   - Create a simple route
   - Send a test message

2. **Day 1-2:** [LANGUAGE_REFERENCE.md](LANGUAGE_REFERENCE.md) (1-2 hours)
   - Read about sources, sinks, steps
   - Skim JSONata basics
   - Try examples from [examples/fragments/](../examples/fragments/)

3. **Day 2:** [USE_CASES.md](USE_CASES.md) (1 hour)
   - Pick 2-3 use cases relevant to you
   - Copy-paste into your own config
   - Adapt to your data/systems

4. **Day 3+:** Build your route
   - Reference [LANGUAGE_REFERENCE.md](LANGUAGE_REFERENCE.md) as needed
   - Use `dimctl validate` and `dimctl explain`
   - Test with `dimctl test` or manual sends
   - View live stats with `dimctl stats`

5. **Ongoing:** [CLI_REFERENCE.md](CLI_REFERENCE.md)
   - Learn monitoring (`dimctl trace tail`, `dimctl stats`)
   - Learn debugging (`dimctl explain`, `dimctl provenance`)
   - Learn compliance (`dimctl lineage query`, `dimctl lineage export`)

---

## External Resources

- **[JSONata](https://jsonata.org/)** — Expression language used in all transformations
  - [Interactive Playground](https://try.jsonata.org/)
  - [Documentation](https://docs.jsonata.org/)
- **[JSON Schema](https://json-schema.org/)** — Schema validation for contracts
- **[OpenTelemetry](https://opentelemetry.io/)** — Tracing and metrics standard

---

## Questions or Feedback?

- **Bugs?** File an issue on GitHub
- **Questions?** Start a discussion
- **Contributing?** See [DEVELOPER_GUIDE.md](../DEVELOPER_GUIDE.md)

---

Happy routing! 🚀
