# Task 1: Set up domains directory structure and document it

## Files to Create/Modify

- Create: `domains/README.md`
- Create: `domains/governance/fragments.yaml`
- Create: `domains/governance/common-steps.yaml`
- Create: `domains/CODEOWNERS` (initially empty, updated in Task 4)
- Modify: `.gitignore` (if needed for secrets in domain directories)

## Interfaces

- Produces: Domain directory structure; governance fragments that other tasks reuse

## Steps to Execute

### Step 1: Write domains/README.md

Create the file with the following content:

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

### Step 2: Write domains/governance/fragments.yaml

Create the file with the following content:

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

### Step 3: Write domains/governance/common-steps.yaml

Create the file with the following content:

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

### Step 4: Create domains/CODEOWNERS template (empty for now)

Create the file with the following content:

```
# GitHub CODEOWNERS file for domain-scoped review.
# Format: path @reviewer @backup
# Example (added later):
# domains/payments/ @alice @bob
```

### Step 5: Check if .gitignore needs updates

Review `.gitignore` to ensure it properly ignores secrets files in domain directories (if any). Add a line like this if needed:

```
domains/**/*.secret.yaml
```

But do NOT add this now unless `.gitignore` is being actively used for domain-level secrets (which it shouldn't be — secrets are handled via ${SECRET:name} resolution).

### Step 6: Verify structure and commit

After creating all files, verify they exist:

```bash
ls -la domains/README.md
ls -la domains/governance/fragments.yaml
ls -la domains/governance/common-steps.yaml
ls -la domains/CODEOWNERS
```

Then commit:

```bash
git add domains/
git commit -m "chore: establish domains directory structure and governance fragments"
```

## Success Criteria

- All four files created (`domains/README.md`, `domains/governance/fragments.yaml`, `domains/governance/common-steps.yaml`, `domains/CODEOWNERS`)
- `domains/governance/` directory structure exists
- Files are valid YAML (governance fragments parse correctly)
- Commit is present in git log
