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
