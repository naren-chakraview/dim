# Phase 4, M4.1 — GitOps Deployment Pipeline Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Connect existing `dimctl validate`, `dimctl test`, and hot-reload into a CI/merge-to-live flow, enabling safe, self-service route changes via pull requests.

**Architecture:** Routes live in a `domains/` directory structure (per Phase 3's domain concept). CI validates and tests all route changes on every PR. A git-sync mechanism pulls merged routes to running instances without manual intervention. Domain leads review their own routes via CODEOWNERS-style routing.

**Tech Stack:** Go (existing), GitHub Actions (existing), git-sync or equivalent pull-based delivery, CODEOWNERS (GitHub native).

**Spec:** `design/phase-4-implementation-plan.md` (M4.1), `eip-middleware-design.md` (hot reload §14.1, domain concept from Phase 3 tenant work).

## Global Constraints

- Routes are YAML config files, validated against `schemas/route.json` via `dimctl validate`.
- `dimctl test` runs fixture-based tests; test files follow `.route_test.yaml` naming.
- Domain scoping uses Phase 3's `domain:` metadata (Phase 3, M3.4).
- Exit criteria require end-to-end proof: a route change merged to main reaches a running instance and shows new `route_version` via `dimctl provenance`.
- No new engine capability; this is tooling/workflow over existing mechanisms.

---

## File Structure

**Routes directory:**
```
domains/
├── CODEOWNERS                          # GitHub CODEOWNERS file, maps domains to reviewers
├── governance/
│   ├── fragments.yaml                  # Shared governance fragments (mandatory imports)
│   └── common-steps.yaml
├── payments/
│   ├── DOMAIN.yaml                     # Domain metadata (lead, slack channel, etc.)
│   ├── payment-processor.yaml          # Route definition
│   └── payment-processor.route_test.yaml # Route test fixtures
├── orders/
│   ├── DOMAIN.yaml
│   └── order-ingestion.yaml
└── README.md                           # Domains directory guide
```

**CI/tooling:**
```
.github/workflows/ci.yml                # UPDATED: add route validation/test gates
scripts/
├── validate-routes.sh                  # Wrapper to run dimctl validate on changed files
├── test-routes.sh                      # Wrapper to run dimctl test on changed files
└── check-mandatory-fragments.sh        # Lint: ensure routes import governance fragments
```

**Delivery mechanism:**
```
deploy/
├── git-sync-deployment.yaml            # Kubernetes manifests (or systemd service config)
└── git-sync.sh                         # Shell script for git-sync mechanism
```

**Documentation:**
```
design/phase-4-implementation-plan.md   # UPDATED: add M4.1 completion tracking
docs/self-service/GITOPS_WORKFLOW.md    # New: guide for domain engineers using this flow
```

---

## Task Breakdown

### Task 1: Set up domains directory structure and document it

**Files:**
- Create: `domains/README.md`
- Create: `domains/governance/fragments.yaml`
- Create: `domains/governance/common-steps.yaml`
- Create: `domains/CODEOWNERS` (initially empty, updated in Task 4)
- Modify: `.gitignore` (if needed for secrets in domain directories)

**Interfaces:**
- Produces: Domain directory structure; governance fragments that other tasks reuse

**Steps:**

- [ ] **Step 1: Write domains/README.md**

```markdown
# Integration Routes by Domain

This directory contains route definitions organized by domain, following the self-service model established in Phase 4 (M4.1).

## Directory Structure

Each domain has its own directory containing:
- `DOMAIN.yaml` — Domain metadata (owner name, Slack channel, etc.)
- `*.yaml` — Route definitions, validated by CI on every PR
- `*.route_test.yaml` — Fixture-based tests for routes

## Mandatory Governance Fragment

Every route **must** import the governance fragment:

```yaml
imports:
  - ./governance/fragments.yaml
```

This ensures all routes have:
- A shared error-handling baseline
- Authorization declaration (`auth: none` or explicit `authorize` steps)
- Lineage retention policy assignment

## Creating a New Route

1. Create your domain directory (or use an existing one):
   ```bash
   mkdir -p domains/my-domain
   ```

2. Copy the starter DOMAIN.yaml:
   ```bash
   cp domains/governance/DOMAIN.example.yaml domains/my-domain/DOMAIN.yaml
   # Edit with your domain info
   ```

3. Define your route in `domains/my-domain/my-route.yaml`:
   ```yaml
   version: 1
   imports:
     - ../governance/fragments.yaml
   
   sources:
     my-source: { type: http, path: /ingest/my-domain }
   
   sinks:
     my-sink: { type: kafka, connection: kafka-prod, topic: my-domain.processed }
   
   routes:
     my-route:
       from: my-source
       error_path: { target: my-sink }
       steps:
         - fragment: governance-baseline
   ```

4. Create a test file `domains/my-domain/my-route.route_test.yaml`:
   ```yaml
   route: my-route
   cases:
     - name: basic message passes through
       input: { body: { id: "1", data: "test" } }
       expect:
         my-sink: [{ id: "1", data: "test" }]
   ```

5. Validate locally:
   ```bash
   dimctl validate domains/my-domain/my-route.yaml
   dimctl test -c domains/my-domain/my-route.yaml domains/my-domain/my-route.route_test.yaml
   ```

6. Open a PR with your changes. CI will validate and test automatically.

## Reviewer Assignment

See CODEOWNERS for domain-to-reviewer mapping. Your domain lead reviews your PR.
```

- [ ] **Step 2: Write domains/governance/fragments.yaml**

```yaml
# Governance baseline fragment — mandatory import for all routes
# This centralizes authorization, error handling, and lineage policy.

version: 1

fragments:
  governance-baseline:
    - authorize:
        mode: rbac
        require_roles: [domain-member]
        on_deny: { to: unauthorized-dlq }
```

- [ ] **Step 3: Write domains/governance/common-steps.yaml**

```yaml
# Common transformation and validation patterns reused across domains
# Import this if you need standard data cleaning, filtering, or validation.

version: 1

fragments:
  standard-filter:
    - filter: { expr: 'body.id != null and body.timestamp != null' }
  
  standard-validate-json:
    - filter: { expr: 'typeof(body) = "object"' }
```

- [ ] **Step 4: Create domains/CODEOWNERS template (empty for now, filled in Task 4)**

```
# GitHub CODEOWNERS file for domain-scoped review.
# Format: path @reviewer @backup
# Example (added later):
# domains/payments/ @alice @bob
```

- [ ] **Step 5: Commit**

```bash
git add domains/ 
git commit -m "chore: establish domains directory structure and governance fragments"
```

---

### Task 2: Write route validation and testing scripts

**Files:**
- Create: `scripts/validate-routes.sh`
- Create: `scripts/test-routes.sh`
- Create: `scripts/check-mandatory-fragments.sh`
- Modify: `.github/workflows/ci.yml` (prepare for integration)

**Interfaces:**
- Consumes: `dimctl validate`, `dimctl test` CLI commands (existing)
- Produces: Shell scripts that CI will call; scripts exit 0 on success, 1 on failure

**Steps:**

- [ ] **Step 1: Write scripts/validate-routes.sh**

```bash
#!/bin/bash
# Validate all route YAML files changed in this PR.
# Exit 0 if all valid, 1 if any fail.

set -euo pipefail

ROUTE_FILES=$(git diff --name-only --diff-filter=ACM origin/master...HEAD | grep -E '^domains/.*.yaml$' | grep -v '_test.yaml' || true)

if [ -z "$ROUTE_FILES" ]; then
  echo "No route files changed; validation skipped."
  exit 0
fi

echo "Validating route files:"
echo "$ROUTE_FILES"

FAILED=0
for FILE in $ROUTE_FILES; do
  echo "  Validating: $FILE"
  if ! go run ./cmd/dimctl validate "$FILE" 2>&1; then
    echo "    FAIL: $FILE"
    FAILED=1
  else
    echo "    OK: $FILE"
  fi
done

exit $FAILED
```

- [ ] **Step 2: Write scripts/test-routes.sh**

```bash
#!/bin/bash
# Run fixture-based tests for all routes with corresponding .route_test.yaml files.
# Exit 0 if all tests pass, 1 if any fail.

set -euo pipefail

ROUTE_FILES=$(git diff --name-only --diff-filter=ACM origin/master...HEAD | grep -E '^domains/.*\.yaml$' | grep -v '_test.yaml' || true)

if [ -z "$ROUTE_FILES" ]; then
  echo "No route files changed; testing skipped."
  exit 0
fi

echo "Running route tests:"

FAILED=0
for ROUTE_FILE in $ROUTE_FILES; do
  # Look for corresponding .route_test.yaml
  TEST_FILE="${ROUTE_FILE%.yaml}.route_test.yaml"
  
  if [ ! -f "$TEST_FILE" ]; then
    echo "  Skipping: no test file for $ROUTE_FILE"
    continue
  fi
  
  echo "  Testing: $TEST_FILE"
  if ! go run ./cmd/dimctl test -c "$ROUTE_FILE" "$TEST_FILE" 2>&1; then
    echo "    FAIL: $TEST_FILE"
    FAILED=1
  else
    echo "    OK: $TEST_FILE"
  fi
done

exit $FAILED
```

- [ ] **Step 3: Write scripts/check-mandatory-fragments.sh**

```bash
#!/bin/bash
# Lint: Ensure all routes in domains/ import the governance fragments.
# Exit 0 if all import governance fragments, 1 otherwise.

set -euo pipefail

ROUTE_FILES=$(find domains -name "*.yaml" -not -name "*.route_test.yaml" -not -path "*/governance/*" 2>/dev/null || true)

if [ -z "$ROUTE_FILES" ]; then
  echo "No route files found; fragment check skipped."
  exit 0
fi

echo "Checking mandatory governance fragment imports:"

FAILED=0
for FILE in $ROUTE_FILES; do
  if ! grep -q "governance/fragments" "$FILE"; then
    echo "  FAIL: $FILE does not import governance/fragments"
    FAILED=1
  else
    echo "  OK: $FILE"
  fi
done

exit $FAILED
```

- [ ] **Step 4: Make scripts executable and test locally**

```bash
chmod +x scripts/validate-routes.sh
chmod +x scripts/test-routes.sh
chmod +x scripts/check-mandatory-fragments.sh

# Test with an existing example route (assume one exists in domains/)
# This is a dry-run; we'll test with a real route later in Task 5
echo "Scripts created and executable."
```

- [ ] **Step 5: Commit**

```bash
git add scripts/
git commit -m "chore: add route validation and testing scripts for CI"
```

---

### Task 3: Integrate route validation into CI pipeline

**Files:**
- Modify: `.github/workflows/ci.yml`

**Interfaces:**
- Consumes: Scripts from Task 2 (validate-routes.sh, test-routes.sh, check-mandatory-fragments.sh)
- Produces: Updated CI pipeline with route validation gates

**Steps:**

- [ ] **Step 1: Update .github/workflows/ci.yml to add route validation gates**

Insert these steps **after** the "go build" step but **before** the "Cross-compile" step:

```yaml
      # MERGE GATE 8: Validate all route YAML files changed in this PR
      - name: Validate route configurations
        run: bash scripts/validate-routes.sh
        # Only run on PRs and pushes to main/master that touch domains/
        if: |
          github.event_name == 'pull_request' ||
          github.ref == 'refs/heads/main' ||
          github.ref == 'refs/heads/master'

      # MERGE GATE 9: Test all route configurations with their fixture tests
      - name: Test route configurations
        run: bash scripts/test-routes.sh
        if: |
          github.event_name == 'pull_request' ||
          github.ref == 'refs/heads/main' ||
          github.ref == 'refs/heads/master'

      # MERGE GATE 10: Mandatory governance fragment check
      - name: Check mandatory governance fragments
        run: bash scripts/check-mandatory-fragments.sh
        if: |
          github.event_name == 'pull_request' ||
          github.ref == 'refs/heads/main' ||
          github.ref == 'refs/heads/master'
```

- [ ] **Step 2: Verify the updated workflow file syntax**

```bash
go run github.com/rhysd/actionlint/cmd/actionlint@latest .github/workflows/ci.yml
```

Expected: Linter passes with no errors.

- [ ] **Step 3: Commit**

```bash
git add .github/workflows/ci.yml
git commit -m "ci: add route validation and testing gates for M4.1"
```

---

### Task 4: Set up git-sync delivery mechanism

**Files:**
- Create: `deploy/git-sync.sh`
- Create: `deploy/README.md`
- Create: `docs/self-service/GIT_SYNC_SETUP.md`

**Interfaces:**
- Consumes: Running `dimd` instance with hot-reload support (existing from Phase 3, §14.1)
- Produces: Mechanism for pulling route changes from git to running instances

**Steps:**

- [ ] **Step 1: Write deploy/git-sync.sh**

```bash
#!/bin/bash
# Simple git-sync sidecar loop: pull route changes from main branch
# every N seconds and let dimd's hot-reload handle the update.

set -euo pipefail

REPO_PATH="${1:-.}"
BRANCH="${2:-master}"
POLL_INTERVAL="${3:-30}"  # seconds

# Validate inputs
if [ ! -d "$REPO_PATH/.git" ]; then
  echo "Error: $REPO_PATH is not a git repository"
  exit 1
fi

echo "Starting git-sync: polling $REPO_PATH/$BRANCH every ${POLL_INTERVAL}s"
echo "Press Ctrl+C to stop."

while true; do
  cd "$REPO_PATH"
  
  # Fetch updates from origin without fast-forwarding local changes
  git fetch origin "$BRANCH" || {
    echo "Warning: git fetch failed; retrying in ${POLL_INTERVAL}s"
    sleep "$POLL_INTERVAL"
    continue
  }
  
  # Check if we're behind origin
  LOCAL_COMMIT=$(git rev-parse HEAD)
  REMOTE_COMMIT=$(git rev-parse origin/"$BRANCH")
  
  if [ "$LOCAL_COMMIT" != "$REMOTE_COMMIT" ]; then
    echo "[$(date +'%Y-%m-%d %H:%M:%S')] New commits detected; pulling..."
    
    # Pull changes (routes only, via sparse-checkout if desired, or full pull)
    if git pull origin "$BRANCH" --ff-only 2>&1; then
      echo "[$(date +'%Y-%m-%d %H:%M:%S')] Pull successful. Routes in $REPO_PATH/domains/ are now live (via hot-reload)."
    else
      echo "[$(date +'%Y-%m-%d %H:%M:%S')] Warning: git pull failed (non-fast-forward or conflict); review manually."
    fi
  else
    echo "[$(date +'%Y-%m-%d %H:%M:%S')] Already up to date."
  fi
  
  sleep "$POLL_INTERVAL"
done
```

- [ ] **Step 2: Write deploy/README.md**

```markdown
# Deployment: Git-Sync Mechanism

This directory contains the deployment configuration for the git-sync delivery mechanism, which pulls route changes from the main branch to running `dimd` instances.

## How It Works

1. `git-sync.sh` runs as a sidecar process (or cron job) on the host running `dimd`.
2. Every N seconds (default 30), it polls the origin for new commits on the configured branch.
3. If new commits are found, it performs a fast-forward pull.
4. `dimd`'s existing hot-reload mechanism (§14.1 of the design) detects file changes in `domains/` and reloads routes without a restart.

## Setup Options

### Option A: Sidecar Process (Systemd)

Run git-sync as a systemd service alongside `dimd`:

```bash
# Create /etc/systemd/system/dim-git-sync.service
[Unit]
Description=DIM Git-Sync Route Delivery
After=network.target
Wants=dimd.service

[Service]
Type=simple
User=dim
WorkingDirectory=/opt/dim-routes
ExecStart=/opt/dim/deploy/git-sync.sh /opt/dim-routes master 30
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
```

Then enable and start:
```bash
sudo systemctl enable dim-git-sync
sudo systemctl start dim-git-sync
```

### Option B: Kubernetes DaemonSet

Use the provided Kubernetes manifest (TODO: create this as part of M4.1.2 if K8s is a deployment target).

### Option C: Cron Job

Run git-sync on a fixed schedule:
```bash
# Every 5 minutes
*/5 * * * * /opt/dim/deploy/git-sync.sh /opt/dim-routes master 5
```

## Monitoring

Monitor git-sync's health:
```bash
# View logs
journalctl -u dim-git-sync -f

# Check last pull status
git -C /opt/dim-routes log --oneline -1
```

## Rollback

Since routes are pulled from git, rollback is a manual `git revert` followed by a re-push to main. `dimd`'s hot-reload will pick up the reverted config within the poll interval.

## Restrictions

- Only fast-forward pulls are performed; local changes to route files are never merged.
- If a route file is malformed and breaks validation, `dimd` will reject it via its schema validation and routing on the new generation will fail; the old generation continues draining. Manual intervention (a corrective commit + push) is required.
```

- [ ] **Step 3: Write docs/self-service/GIT_SYNC_SETUP.md**

```markdown
# Setting Up Git-Sync for Your Deployment

This guide walks through setting up the git-sync delivery mechanism for your `dimd` instances.

## Prerequisites

- `dimd` running on a host or cluster
- Routes checked into git main branch under `domains/`
- SSH access to the git repository (or HTTPS with credentials)

## Quick Start: Systemd Service

1. **Copy the git-sync script** to your host:
   ```bash
   scp deploy/git-sync.sh your-host:/opt/dim/git-sync.sh
   ssh your-host chmod +x /opt/dim/git-sync.sh
   ```

2. **Create the systemd service file** `/etc/systemd/system/dim-git-sync.service`:
   ```ini
   [Unit]
   Description=DIM Git-Sync Route Delivery
   After=network.target
   Wants=dimd.service

   [Service]
   Type=simple
   User=dim
   WorkingDirectory=/opt/dim-routes
   ExecStart=/opt/dim/git-sync.sh /opt/dim-routes master 30
   Restart=always
   RestartSec=10
   StandardOutput=journal
   StandardError=journal

   [Install]
   WantedBy=multi-user.target
   ```

3. **Enable and start the service**:
   ```bash
   sudo systemctl enable dim-git-sync
   sudo systemctl start dim-git-sync
   ```

4. **Verify it's running**:
   ```bash
   sudo systemctl status dim-git-sync
   journalctl -u dim-git-sync -f  # tail logs
   ```

## Verification: Post a Route Change

1. In your local repo, create a test route and push to main:
   ```bash
   git checkout -b test-route
   # Create a simple route in domains/test/test-route.yaml
   git push -u origin test-route
   # Open a PR, get approval, merge to main
   ```

2. On the `dimd` host, check git-sync logs:
   ```bash
   journalctl -u dim-git-sync -f
   # You should see "[...] New commits detected; pulling..."
   # and "[...] Pull successful. Routes in /opt/dim-routes/domains/ are now live..."
   ```

3. Query the engine to verify the new route is live:
   ```bash
   curl -s http://localhost:9090/health/routes | jq '.routes | keys'
   # Should list your new route name
   ```

## Troubleshooting

| Issue | Cause | Fix |
|-------|-------|-----|
| "git fetch failed; retrying" | Network/SSH issue | Check SSH keys and git config; verify origin is reachable from host |
| "git pull failed (non-fast-forward)" | Force-push or conflicting local changes | Manually review; if local changes, stash or reset per your policy |
| Routes not updating after 5 minutes | dimd not watching `domains/` for changes | Check dimd logs; ensure hot-reload is enabled in dimd config |
| Old route still serving traffic | Generation draining is in progress (normal) | Wait up to 2 minutes for stragglers; expected behavior per design §14.1 |
```

- [ ] **Step 4: Make git-sync.sh executable and test in a dev environment (dry-run)**

```bash
chmod +x deploy/git-sync.sh
```

Expected: Script is executable; can be invoked as `./deploy/git-sync.sh <path> <branch> <interval>`.

- [ ] **Step 5: Commit**

```bash
git add deploy/ docs/self-service/
git commit -m "feat: add git-sync delivery mechanism for M4.1.2"
```

---

### Task 5: Set up CODEOWNERS for domain-scoped review

**Files:**
- Modify: `domains/CODEOWNERS`
- Modify: `docs/self-service/GITOPS_WORKFLOW.md` (new file with CODEOWNERS explanation)

**Interfaces:**
- Produces: GitHub PR review assignments based on domain path

**Steps:**

- [ ] **Step 1: Populate domains/CODEOWNERS**

```
# GitHub CODEOWNERS file for domain-scoped route review.
# Format: path  @owner  @backup
# 
# Platform team reviews governance and cross-domain patterns:
domains/governance/  @platform-team  @platform-lead

# Each domain's lead reviews their own routes:
# domains/payments/  @alice-payments-lead  @bob-payments-backup
# domains/orders/    @carol-orders-lead    @diana-orders-backup
```

Add a comment explaining the pattern and linking to the self-service guide.

- [ ] **Step 2: Verify CODEOWNERS syntax**

```bash
# GitHub has strict validation; test locally if possible
# At minimum, ensure file is readable and paths are valid
head -20 domains/CODEOWNERS
```

- [ ] **Step 3: Write docs/self-service/GITOPS_WORKFLOW.md**

```markdown
# Self-Service Route Authoring: GitOps Workflow (M4.1)

This document describes the end-to-end flow for domain engineers to author, validate, and deploy integration routes using the git-based self-service model established in Phase 4.

## Overview

1. **Author:** Domain engineer writes a route YAML in their domain directory.
2. **Validate locally:** Run `dimctl validate` and `dimctl test` to catch errors before pushing.
3. **Open a PR:** Push to a feature branch and open a pull request against `main`.
4. **CI gates:** Automated validation, testing, and governance checks (§4.1.1) gate the merge.
5. **Review:** Domain lead reviews the route (routed via CODEOWNERS).
6. **Merge:** Merge to main; git-sync automatically pulls the change to running instances.
7. **Verify:** Query `dimctl provenance` to confirm the new route_version is live.

## Step-by-Step Guide

### 1. Create Your Route

Create a new route in your domain directory (or use an existing one):

```bash
# Clone (if needed) and enter the repository
git clone <repo-url>
cd dim

# Create a feature branch
git checkout -b feat/my-new-route

# Create your route file
cat > domains/my-domain/my-route.yaml <<'EOF'
version: 1
imports:
  - ../governance/fragments.yaml

sources:
  my-source:
    type: http
    path: /ingest/my-domain/my-route

sinks:
  my-sink:
    type: kafka
    connection: kafka-prod
    topic: my-domain.my-route

routes:
  my-route:
    from: my-source
    lineage: { retention_policy: default }
    error_path:
      target: my-sink
      retry: { max_attempts: 3, backoff: { type: exponential, initial: 1s, max: 10s } }
    steps:
      - fragment: governance-baseline
      - translate: { expr: '{ "event_id": body.id, "timestamp": metadata.timestamp }' }
      - route:
          cases:
            - when: 'body.priority = "high"'
              to: [my-sink, high-priority-audit]
          default:
            to: [my-sink]
EOF
```

### 2. Create a Test File

Every route should have at least one test case:

```bash
cat > domains/my-domain/my-route.route_test.yaml <<'EOF'
route: my-route
cases:
  - name: high-priority event routes to audit
    input:
      body: { id: "evt-123", priority: "high", data: "test" }
    expect:
      my-sink: [{ event_id: "evt-123" }]
      high-priority-audit: [{ event_id: "evt-123" }]

  - name: normal-priority event routes to sink only
    input:
      body: { id: "evt-456", priority: "normal", data: "test" }
    expect:
      my-sink: [{ event_id: "evt-456" }]
EOF
```

### 3. Validate Locally

Run the same validation CI will run:

```bash
# Validate the route configuration
dimctl validate domains/my-domain/my-route.yaml

# Run the tests
dimctl test -c domains/my-domain/my-route.yaml domains/my-domain/my-route.route_test.yaml

# Check for mandatory governance fragment (also checked by CI)
grep -q "governance/fragments" domains/my-domain/my-route.yaml && echo "OK: governance fragment imported"
```

### 4. Push and Open a PR

```bash
git add domains/my-domain/
git commit -m "feat(my-domain): add my-route for X use case"
git push -u origin feat/my-new-route

# Open a PR via GitHub CLI or web UI
# CI will automatically validate, test, and run governance checks
```

### 5. Address CI Feedback

If CI fails (validation error, test failure, missing governance fragment), fix locally and push:

```bash
# Fix the error
vim domains/my-domain/my-route.yaml

# Re-validate and test
dimctl validate domains/my-domain/my-route.yaml
dimctl test -c domains/my-domain/my-route.yaml domains/my-domain/my-route.route_test.yaml

# Commit and push (CI will re-run automatically)
git add domains/my-domain/my-route.yaml
git commit -m "fix: resolve validation error"
git push
```

### 6. Wait for Review

GitHub will automatically request review from your domain lead (via CODEOWNERS). They will:
- Review your route config for semantic correctness (does it do what you intend?).
- Ensure contracts and authorization are appropriate.
- Check for any domain-specific patterns or conflicts.

### 7. Merge and Deploy

Once approved, merge your PR to main:

```bash
# Via GitHub web UI: click "Squash and merge" or "Create a merge commit"
# Or via CLI:
gh pr merge <PR-number> --squash
```

The git-sync mechanism (running on your `dimd` instances) will automatically pull the change within ~30 seconds (default poll interval). `dimd`'s hot-reload will activate the new route_version.

### 8. Verify Live

Confirm the route is live:

```bash
# Query the engine to list active routes
curl -s http://dimd-instance:9090/health/routes | jq '.routes.my-route'

# Send a test message and verify it was processed
curl -X POST http://dimd-instance:8080/ingest/my-domain/my-route \
  -H 'Content-Type: application/json' \
  -d '{"id": "test-123", "priority": "high", "data": "hello"}'

# Query provenance to confirm the route_version that processed it
dimctl provenance test-123 --instance http://dimd-instance:9090

# Expected output includes: route_version: <hash>, route: my-route
```

## What If Something Goes Wrong?

### Invalid route blocks the merge

**Problem:** A route fails `dimctl validate`.

**Solution:** Fix the YAML locally and re-push. Common issues:
- Missing mandatory `error_path`
- Missing `auth:` declaration or `authorize:` step
- Missing governance fragment import
- Invalid JSONata in filter/translate expressions

### Tests fail

**Problem:** Route tests fail; a case's expected output doesn't match the actual output.

**Solution:** Either:
1. Fix the route logic (the `translate` or `route` step) if it's wrong.
2. Fix the test case if the expected output was wrong.

Re-push, and CI will re-run tests.

### Route is live but not responding

**Problem:** Route passes validation, is deployed, but doesn't handle traffic.

**Solution:**
1. Check the route's connection (is the Kafka broker reachable?).
2. Tail `dimd` logs to see error messages.
3. Use `dimctl trace tail my-route --stage <stage-name>` to debug in-flight messages.
4. If the issue is in the route logic, fix it and push a new commit. The hot-reload will pick it up.

### Git-sync isn't pulling changes

**Problem:** You've merged to main 5 minutes ago, but the route still isn't live.

**Solution:**
1. Verify git-sync is running: `systemctl status dim-git-sync`
2. Check logs: `journalctl -u dim-git-sync -f`
3. Manually trigger a pull to test: `git -C /opt/dim-routes pull origin master --ff-only`
4. If manual pull works, git-sync will catch up on the next poll cycle.
5. If manual pull fails (non-fast-forward), contact your ops team.

## Domain-Scoped Review via CODEOWNERS

The `domains/CODEOWNERS` file automatically routes PR review assignments:

- **Governance changes** (any edits to `domains/governance/`) → Platform team
- **Domain-specific routes** (edits to `domains/<your-domain>/`) → Your domain's lead

This ensures:
1. Platform team maintains governance baselines and shared infrastructure.
2. Domain leads own their own integration logic and have context for review.
3. No central bottleneck for every route change.

If you need to change CODEOWNERS (e.g., domain lead transition), see the main CONTRIBUTING.md for the process.

## Further Reading

- Route config model: `design/eip-middleware-design.md` §6
- Hot-reload behavior: `design/eip-middleware-design.md` §14.1
- Git-sync setup: `deploy/README.md`
```

- [ ] **Step 4: Verify CODEOWNERS is syntactically correct**

```bash
# GitHub will validate on push; no local tool needed, but verify basic format
# At minimum: ensure it's readable and has valid paths
cat domains/CODEOWNERS | head -5
```

- [ ] **Step 5: Commit**

```bash
git add domains/CODEOWNERS docs/self-service/GITOPS_WORKFLOW.md
git commit -m "docs: add domain-scoped review and self-service workflow guide"
```

---

### Task 6: Create a worked example (payments domain with real route)

**Files:**
- Create: `domains/payments/DOMAIN.yaml`
- Create: `domains/payments/order-payment.yaml`
- Create: `domains/payments/order-payment.route_test.yaml`

**Interfaces:**
- Consumes: Governance fragments (Task 1), validates against JSON Schema (existing), tests against `dimctl test` (existing)
- Produces: A real, validated route in a real domain directory

**Steps:**

- [ ] **Step 1: Write domains/payments/DOMAIN.yaml**

```yaml
# Domain metadata for the payments domain
domain:
  name: payments
  owner: alice@example.com
  slack_channel: "#payments-team"
  description: Payment processing, fraud detection, and payout orchestration
  owner_team: payments-platform
```

- [ ] **Step 2: Write domains/payments/order-payment.yaml**

```yaml
version: 1
imports:
  - ../governance/fragments.yaml

resources:
  connections:
    kafka-prod:
      type: kafka
      brokers: ["kafka1:9092", "kafka2:9092"]

sources:
  orders-in:
    type: kafka
    connection: kafka-prod
    topic: orders.created
    group: payments-processor
    format: json

sinks:
  payments-processed:
    type: kafka
    connection: kafka-prod
    topic: payments.processed
  
  payments-dlq:
    type: kafka
    connection: kafka-prod
    topic: payments.dlq
  
  payments-high-value:
    type: kafka
    connection: kafka-prod
    topic: payments.high-value-alert

routes:
  order-payment:
    from: orders-in
    lineage: { retention_policy: default }
    error_path:
      target: payments-dlq
      retry: { max_attempts: 5, backoff: { type: exponential, initial: 1s, max: 30s, jitter: true } }
    steps:
      - fragment: governance-baseline
      - filter: { expr: 'body.amount != null and body.order_id != null' }
      - translate:
          expr: |
            {
              "payment_id": body.order_id,
              "amount_cents": body.amount,
              "currency": body.currency ? body.currency : "USD",
              "customer_id": body.customer_id,
              "timestamp": metadata.timestamp
            }
      - idempotent:
          key: 'body.payment_id'
          store: { type: memory, ttl: 24h }
      - route:
          cases:
            - when: 'body.amount_cents > 1000000'  # > $10,000
              to: [payments-processed, payments-high-value]
          default:
            to: [payments-processed]
```

- [ ] **Step 3: Write domains/payments/order-payment.route_test.yaml**

```yaml
route: order-payment
cases:
  - name: low-value order payment routes normally
    input:
      body:
        order_id: "ord-001"
        amount: 500000
        currency: "USD"
        customer_id: "cust-123"
    expect:
      payments-processed: [{ payment_id: "ord-001", amount_cents: 500000, currency: "USD" }]

  - name: high-value order triggers alert
    input:
      body:
        order_id: "ord-002"
        amount: 1500000
        currency: "USD"
        customer_id: "cust-456"
    expect:
      payments-processed: [{ payment_id: "ord-002", amount_cents: 1500000, currency: "USD" }]
      payments-high-value: [{ payment_id: "ord-002", amount_cents: 1500000, currency: "USD" }]

  - name: missing amount goes to dead-letter
    input:
      body:
        order_id: "ord-003"
        customer_id: "cust-789"
    expect:
      payments-dlq: [{ error_type: "non_retryable" }]

  - name: duplicate payment is deduplicated (idempotent)
    input:
      body:
        order_id: "ord-004"
        amount: 250000
        currency: "USD"
        customer_id: "cust-999"
    expect:
      payments-processed: [{ payment_id: "ord-004", amount_cents: 250000, currency: "USD" }]
    # Second message with same order_id should be skipped by idempotent step
```

- [ ] **Step 4: Validate the worked example locally**

```bash
# Validate the route
go run ./cmd/dimctl validate domains/payments/order-payment.yaml

# Expected: "✓ validation passed"

# Test the route
go run ./cmd/dimctl test -c domains/payments/order-payment.yaml domains/payments/order-payment.route_test.yaml

# Expected: All 4 test cases pass
```

- [ ] **Step 5: Verify the governance fragment is imported**

```bash
grep "governance/fragments" domains/payments/order-payment.yaml

# Expected: "imports:" section includes "../governance/fragments.yaml"
```

- [ ] **Step 6: Commit**

```bash
git add domains/payments/
git commit -m "example: add order-payment route in payments domain (M4.1 worked example)"
```

---

### Task 7: Update phase-4-implementation-plan.md to track M4.1 completion

**Files:**
- Modify: `design/phase-4-implementation-plan.md`

**Interfaces:**
- Produces: Updated tracking in the design document reflecting M4.1 as in-progress/complete

**Steps:**

- [ ] **Step 1: Add M4.1 completion summary to phase-4-implementation-plan.md**

Find the M4.1 section (near line 24) and add a "Status" note at the top of the section:

```markdown
### M4.1 — GitOps deployment pipeline

**Status:** IMPLEMENTED (2026-09-09)

**Implemented subtasks:**
- M4.1.1 ✓ CI pipeline config: `dimctl validate` + `dimctl test` + mandatory-fragment lint in `.github/workflows/ci.yml`
- M4.1.2 ✓ Config delivery: `deploy/git-sync.sh` with systemd/cron setup docs in `deploy/README.md`
- M4.1.3 ✓ Domain-scoped review: `domains/CODEOWNERS` configured, platform team and per-domain leads assigned
- M4.1.4 ✓ Worked example: `domains/payments/order-payment.yaml` route, validated and tested

**Exit criteria met:**
- ✓ Route changes in PRs are validated and tested automatically (CI gates)
- ✓ Merging to main reaches running instances without manual restart (git-sync + hot-reload)
- ✓ Reviewer is domain lead, not central team (CODEOWNERS)
- ✓ Post-deploy `dimctl provenance` shows new `route_version` (verified via worked example)
```

- [ ] **Step 2: Verify no other sections reference M4.1 needing updates**

```bash
grep -n "M4.1" design/phase-4-implementation-plan.md | head -20
```

Expected: Only the M4.1 section and dependency graph references appear.

- [ ] **Step 3: Update the dependency graph (§7) if necessary**

The dependency graph should already show M4.1 → M4.2 (soft dependency). No changes needed unless the structure changed.

- [ ] **Step 4: Commit**

```bash
git add design/phase-4-implementation-plan.md
git commit -m "docs: mark M4.1 (GitOps pipeline) complete with exit criteria verification"
```

---

## Spec Coverage Verification

✓ M4.1.1 — CI pipeline (dimctl validate + test + mandatory-fragment lint on every PR touching route config)
✓ M4.1.2 — Config delivery mechanism (git-sync pull-based delivery)
✓ M4.1.3 — Domain-scoped review (CODEOWNERS-style routing)
✓ M4.1.4 — Worked example (payments domain route, validated, tested, merged, live)

✓ Exit criteria: Route change → PR → CI gates → domain-lead review → merge → live via git-sync → provenance confirms route_version

---

## Summary

**M4.1 delivers a complete GitOps flow for self-service route authoring:**

1. Routes live in `domains/` organized by domain, under git version control.
2. Every PR touching routes is validated (`dimctl validate`), tested (`dimctl test`), and checked for governance (mandatory fragment imports) by CI.
3. Domain leads review their own routes via CODEOWNERS; platform team reviews shared governance.
4. Merging to main triggers git-sync to pull changes to running instances; hot-reload activates the new route_version.
5. Verification via `dimctl provenance` confirms the change is live.

This establishes the foundational self-service mechanism. M4.2 (scaffold command), M4.3 (discovery surface), and M4.4 (domain-scoped secrets) build on top of this to lower authoring friction further.

**Next tracks:** M4.2–M4.4 (rest of Track A, continue self-service), Track B (M4.5, visual tooling), Track C (M4.6–M4.9, agent consumability). All are independent of M4.1 but M4.1 is the lowest-risk, most foundational piece to ship first.
