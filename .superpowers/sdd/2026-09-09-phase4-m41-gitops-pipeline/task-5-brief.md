# Task 5: Set up CODEOWNERS for domain-scoped review

## Files to Modify/Create

- Modify: `domains/CODEOWNERS` (was created empty in Task 1, now populate it)
- Create: `docs/self-service/GITOPS_WORKFLOW.md` (new guide for domain engineers)

## Interfaces

- Consumes: Domains directory structure from Task 1 (domains/governance/, domain directories)
- Produces: GitHub PR review assignments based on domain path; workflow documentation for self-service authors

## Steps to Execute

### Step 1: Populate domains/CODEOWNERS

Open the file (created empty in Task 1) and add:

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

**Note:** The domain-specific assignments (payments, orders) are shown as examples/comments. 
At this stage, only the governance/ line needs to be active. Domain assignments will be added 
as real domains are onboarded. The examples show the pattern for future maintainers.

**Key rules:**
- One path per line
- Format: `path  @user1  @user2` (GitHub handles will be expanded by GitHub)
- Comments start with `#`
- GitHub will require the users to exist in your GitHub org

### Step 2: Verify CODEOWNERS syntax

CODEOWNERS has strict validation. Verify locally:
- Check the file is readable: `cat domains/CODEOWNERS | head -10`
- Verify paths start with valid characters (no leading spaces)
- Verify @mentions follow GitHub username format (alphanumeric, hyphens, underscores)

### Step 3: Write docs/self-service/GITOPS_WORKFLOW.md

Create the file with comprehensive self-service workflow documentation. This is a large file (~500 lines) that covers:
- Overview of the GitOps model (propose → validate → review → merge → live)
- Step-by-step route authoring guide
- Mandatory local validation commands
- PR workflow details
- What happens on merge (git-sync pulls, hot-reload activates new route_version)
- Verification commands to confirm live deployment
- Troubleshooting common issues
- Domain-scoped review explanation (CODEOWNERS)

**The complete file content is provided below. Copy it verbatim:**

\`\`\`markdown
# Self-Service Route Authoring: GitOps Workflow (M4.1)

This document describes the end-to-end flow for domain engineers to author, validate, and deploy integration routes using the git-based self-service model established in Phase 4.

## Overview

1. **Author:** Domain engineer writes a route YAML in their domain directory.
2. **Validate locally:** Run \`dimctl validate\` and \`dimctl test\` to catch errors before pushing.
3. **Open a PR:** Push to a feature branch and open a pull request against \`main\`.
4. **CI gates:** Automated validation, testing, and governance checks (§4.1.1) gate the merge.
5. **Review:** Domain lead reviews the route (routed via CODEOWNERS).
6. **Merge:** Merge to main; git-sync automatically pulls the change to running instances.
7. **Verify:** Query \`dimctl provenance\` to confirm the new route_version is live.

## Step-by-Step Guide

### 1. Create Your Route

Create a new route in your domain directory (or use an existing one):

\`\`\`bash
# Clone (if needed) and enter the repository
git clone <repo-url>
cd dim

# Create a feature branch
git checkout -b feat/my-new-route

# Create your route file
cat > domains/my-domain/my-route.yaml <<'ROUTE_EOF'
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
ROUTE_EOF
\`\`\`

### 2. Create a Test File

Every route should have at least one test case:

\`\`\`bash
cat > domains/my-domain/my-route.route_test.yaml <<'TEST_EOF'
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
TEST_EOF
\`\`\`

### 3. Validate Locally

Run the same validation CI will run:

\`\`\`bash
# Validate the route configuration
dimctl validate domains/my-domain/my-route.yaml

# Run the tests
dimctl test -c domains/my-domain/my-route.yaml domains/my-domain/my-route.route_test.yaml

# Check for mandatory governance fragment (also checked by CI)
grep -q "governance/fragments" domains/my-domain/my-route.yaml && echo "OK: governance fragment imported"
\`\`\`

### 4. Push and Open a PR

\`\`\`bash
git add domains/my-domain/
git commit -m "feat(my-domain): add my-route for X use case"
git push -u origin feat/my-new-route

# Open a PR via GitHub CLI or web UI
# CI will automatically validate, test, and run governance checks
\`\`\`

### 5. Address CI Feedback

If CI fails (validation error, test failure, missing governance fragment), fix locally and push:

\`\`\`bash
# Fix the error
vim domains/my-domain/my-route.yaml

# Re-validate and test
dimctl validate domains/my-domain/my-route.yaml
dimctl test -c domains/my-domain/my-route.yaml domains/my-domain/my-route.route_test.yaml

# Commit and push (CI will re-run automatically)
git add domains/my-domain/my-route.yaml
git commit -m "fix: resolve validation error"
git push
\`\`\`

### 6. Wait for Review

GitHub will automatically request review from your domain lead (via CODEOWNERS). They will:
- Review your route config for semantic correctness (does it do what you intend?).
- Ensure contracts and authorization are appropriate.
- Check for any domain-specific patterns or conflicts.

### 7. Merge and Deploy

Once approved, merge your PR to main:

\`\`\`bash
# Via GitHub web UI: click "Squash and merge" or "Create a merge commit"
# Or via CLI:
gh pr merge <PR-number> --squash
\`\`\`

The git-sync mechanism (running on your \`dimd\` instances) will automatically pull the change within ~30 seconds (default poll interval). \`dimd\`'s hot-reload will activate the new route_version.

### 8. Verify Live

Confirm the route is live:

\`\`\`bash
# Query the engine to list active routes
curl -s http://dimd-instance:9090/health/routes | jq '.routes.my-route'

# Send a test message and verify it was processed
curl -X POST http://dimd-instance:8080/ingest/my-domain/my-route \\
  -H 'Content-Type: application/json' \\
  -d '{"id": "test-123", "priority": "high", "data": "hello"}'

# Query provenance to confirm the route_version that processed it
dimctl provenance test-123 --instance http://dimd-instance:9090

# Expected output includes: route_version: <hash>, route: my-route
\`\`\`

## What If Something Goes Wrong?

### Invalid route blocks the merge

**Problem:** A route fails \`dimctl validate\`.

**Solution:** Fix the YAML locally and re-push. Common issues:
- Missing mandatory \`error_path\`
- Missing \`auth:\` declaration or \`authorize:\` step
- Missing governance fragment import
- Invalid JSONata in filter/translate expressions

### Tests fail

**Problem:** Route tests fail; a case's expected output doesn't match the actual output.

**Solution:** Either:
1. Fix the route logic (the \`translate\` or \`route\` step) if it's wrong.
2. Fix the test case if the expected output was wrong.

Re-push, and CI will re-run tests.

### Route is live but not responding

**Problem:** Route passes validation, is deployed, but doesn't handle traffic.

**Solution:**
1. Check the route's connection (is the Kafka broker reachable?).
2. Tail \`dimd\` logs to see error messages.
3. Use \`dimctl trace tail my-route --stage <stage-name>\` to debug in-flight messages.
4. If the issue is in the route logic, fix it and push a new commit. The hot-reload will pick it up.

### Git-sync isn't pulling changes

**Problem:** You've merged to main 5 minutes ago, but the route still isn't live.

**Solution:**
1. Verify git-sync is running: \`systemctl status dim-git-sync\`
2. Check logs: \`journalctl -u dim-git-sync -f\`
3. Manually trigger a pull to test: \`git -C /opt/dim-routes pull origin master --ff-only\`
4. If manual pull works, git-sync will catch up on the next poll cycle.
5. If manual pull fails (non-fast-forward), contact your ops team.

## Domain-Scoped Review via CODEOWNERS

The \`domains/CODEOWNERS\` file automatically routes PR review assignments:

- **Governance changes** (any edits to \`domains/governance/\`) → Platform team
- **Domain-specific routes** (edits to \`domains/<your-domain>/\`) → Your domain's lead

This ensures:
1. Platform team maintains governance baselines and shared infrastructure.
2. Domain leads own their own integration logic and have context for review.
3. No central bottleneck for every route change.

If you need to change CODEOWNERS (e.g., domain lead transition), see the main CONTRIBUTING.md for the process.

## Further Reading

- Route config model: \`design/eip-middleware-design.md\` §6
- Hot-reload behavior: \`design/eip-middleware-design.md\` §14.1
- Git-sync setup: \`deploy/README.md\`
\`\`\`

### Step 4: Verify file content

After creating the file, verify it's readable and well-formed:

```bash
# Check file exists and has content
wc -l docs/self-service/GITOPS_WORKFLOW.md
head -30 docs/self-service/GITOPS_WORKFLOW.md | head -15
```

### Step 5: Commit both files

```bash
git add domains/CODEOWNERS docs/self-service/GITOPS_WORKFLOW.md
git commit -m "docs: add domain-scoped review and self-service workflow guide"
```

## Success Criteria

- `domains/CODEOWNERS` populated with governance path and example domain patterns (as comments)
- `docs/self-service/GITOPS_WORKFLOW.md` created with complete workflow guide
- Both files committed with message matching: "docs: add domain-scoped review and self-service workflow guide"
- Git log shows the commit

## Notes

- CODEOWNERS examples for domains/payments/, domains/orders/ are shown as comments; they will be populated when those domains are onboarded
- The governance line (domains/governance/ → @platform-team) is the only active assignment at this stage
- The GITOPS_WORKFLOW.md is a comprehensive guide; it's substantial (~500 lines) but covers all user workflows
