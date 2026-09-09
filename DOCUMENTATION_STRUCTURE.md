# dim Documentation Structure

This document explains how dim's documentation is organized and where to find information for different audiences.

---

## Documentation Organization

### 📚 `docs/` — End-User & Integrator Documentation

**For:** Application developers, architects, and operators integrating dim into their systems.

**Contents:**
- **[README.md](docs/README.md)** — Navigation hub for all user-facing docs; learning paths by role
- **[GETTING_STARTED.md](docs/GETTING_STARTED.md)** — Quick start: build, install, create first route
- **[LANGUAGE_REFERENCE.md](docs/LANGUAGE_REFERENCE.md)** — Complete configuration syntax reference
- **[USE_CASES.md](docs/USE_CASES.md)** — Real-world example routes (payments, compliance, SaaS, etc.)
- **[PHASE3_FEATURES.md](docs/PHASE3_FEATURES.md)** — Advanced features (claim-check, plugins, clustering, quotas)
- **[DEPLOYMENT_AND_RELEASE.md](docs/DEPLOYMENT_AND_RELEASE.md)** — Production deployment patterns and release management
- **[CLI_REFERENCE.md](docs/CLI_REFERENCE.md)** — Command reference for `dimctl` and `dimd` daemon
- **[E2E_TESTING.md](docs/E2E_TESTING.md)** — Integration testing with real services

**Key principle:** These docs assume the user is integrating dim into their architecture, writing YAML configs, and deploying to production. They do NOT include internal architecture, codebase structure, or how to extend dim's source code.

---

### 🏗️ `design/` — Architecture & Design Documentation

**For:** Maintainers, contributors, and architects designing new features.

**Contents:**
- **phase-*-implementation-plan.md** — Implementation roadmaps and exit criteria
- **eip-middleware-design.md** — Complete architecture and design patterns
- **M3.1.1_DISTRIBUTION_MODEL.md** — Clustering design details
- **M3.4.1_MULTITENANT_RESOURCE_ACCOUNTING.md** — Resource quota design
- **PHASE-3-COMPLETION-REPORT.md** — What was implemented, what remains, follow-up work
- **OKF.md** — [In main directory] Architectural decisions and rationale
- Individual milestone summaries (M1_*, M2_*, M3.*)

**Key principle:** These docs are internal-facing. They document what was built, why, and what work remains.

---

### 📖 Root-Level Documentation

**[DEVELOPER_GUIDE.md](DEVELOPER_GUIDE.md)**
- For developers maintaining or extending dim
- Project structure (by Phase)
- Building, testing, benchmarking
- How to add new steps, adapters
- Hot reload and observability patterns
- Phase 3 follow-up work (maintainers only)

**[README.md](README.md)**
- High-level project overview
- Feature matrix
- Quick links to docs and examples

**[OKF.md](OKF.md)**
- Architecture decisions and their rationale
- Key design patterns (configuration, hot-reload, concurrency, lineage)

**[CONTRIBUTING.md](CONTRIBUTING.md)**
- Contribution guidelines
- Code style
- PR process

---

## Phase 3 Documentation Status

### ✅ Complete & Production-Ready
- **YAML Configuration** — All Phase 3 features can be declared in routes
- **User-Facing Docs** — PHASE3_FEATURES.md, examples in USE_CASES.md
- **Test Suites** — Unit and integration tests for all implementations
- **Plugin System** — Native-Go and WASM plugins fully functional with reference implementations

### 🟡 In-Memory Implementations
- **Claim-Check Store** — In-memory store suitable for testing and single instances
- **Multi-Tenant Quotas** — Executor-level enforcement (factory wiring pending)

### 🔧 Follow-Up Work (See PHASE-3-COMPLETION-REPORT.md)
- M3.2 S3-backed claim-check store (enables production-scale payloads)
- M3.4 Factory wiring (enables end-to-end multi-tenant quotas)
- M3.1 Cluster demo (end-to-end verification)

---

## Reading Guides by Audience

### 👨‍💼 **Business/Product Owner**
1. [README.md](README.md) — Feature overview
2. [docs/USE_CASES.md](docs/USE_CASES.md) — See real examples
3. [PHASE-3-COMPLETION-REPORT.md](design/PHASE-3-COMPLETION-REPORT.md) — Status and roadmap

### 👨‍💻 **Application Developer (Building with dim)**
1. [docs/GETTING_STARTED.md](docs/GETTING_STARTED.md) — Get up and running
2. [docs/LANGUAGE_REFERENCE.md](docs/LANGUAGE_REFERENCE.md) — Learn YAML config syntax
3. [docs/PHASE3_FEATURES.md](docs/PHASE3_FEATURES.md) — Use advanced features
4. [docs/USE_CASES.md](docs/USE_CASES.md) — Copy patterns that match your use case
5. [docs/CLI_REFERENCE.md](docs/CLI_REFERENCE.md) — Debug with CLI tools

### 🏗️ **DevOps/Operations (Deploying dim)**
1. [docs/DEPLOYMENT_AND_RELEASE.md](docs/DEPLOYMENT_AND_RELEASE.md) — How to deploy
2. [docs/E2E_TESTING.md](docs/E2E_TESTING.md) — Set up test environment
3. [docs/CLI_REFERENCE.md](docs/CLI_REFERENCE.md) — Monitor and operate
4. [docs/PHASE3_FEATURES.md](docs/PHASE3_FEATURES.md#production-readiness) — Production readiness status

### 🛠️ **Contributor (Extending dim)**
1. [DEVELOPER_GUIDE.md](DEVELOPER_GUIDE.md) — Project structure and patterns
2. [OKF.md](OKF.md) — Architectural decisions
3. [design/phase-*-implementation-plan.md](design/) — Feature design
4. [design/eip-middleware-design.md](design/eip-middleware-design.md) — Complete architecture
5. [design/PHASE-3-COMPLETION-REPORT.md](design/PHASE-3-COMPLETION-REPORT.md) — What's next?

---

## Key Rules

1. **`docs/` is for users** — No internal implementation details, no "dim maintainer" content
2. **`design/` is for maintainers** — No restrictions; capture all architecture decisions, design iterations, follow-up work
3. **DEVELOPER_GUIDE.md bridges both** — Explains structure and patterns; links to user docs for end-users
4. **Status is explicit** — All documentation clearly marks what's production-ready vs. pending work

---

## How Documentation Updates Work

### After a Feature is Complete
1. Update `design/` docs with implementation details and lessons learned
2. Add/update examples in `docs/USE_CASES.md` if new use case emerges
3. Update `docs/PHASE3_FEATURES.md` (or equivalent) with configuration and status
4. Update `design/PHASE-*-COMPLETION-REPORT.md` with what's left
5. Update relevant section in `DEVELOPER_GUIDE.md`

### Before Release
1. Ensure `docs/` accurately reflects what users can do
2. Ensure status markers (✅, 🟡, 🔧) are correct in user docs
3. Ensure `design/` has complete context for maintainers
4. Update root `README.md` with Phase status

---

## Questions?

- **How do I use dim?** → Start in `docs/README.md`
- **How do I deploy dim?** → See `docs/DEPLOYMENT_AND_RELEASE.md`
- **How do I extend dim?** → See `DEVELOPER_GUIDE.md`
- **What's the architecture?** → See `OKF.md` then `design/eip-middleware-design.md`
- **What's left to build?** → See `design/PHASE-3-COMPLETION-REPORT.md`
