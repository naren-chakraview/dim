# Task 2: Write route validation and testing scripts

## Files to Create/Modify

- Create: `scripts/validate-routes.sh`
- Create: `scripts/test-routes.sh`
- Create: `scripts/check-mandatory-fragments.sh`
- Modify: `.github/workflows/ci.yml` (prepare for integration — no changes needed yet, just verify it exists)

## Interfaces

- Consumes: `dimctl validate`, `dimctl test` CLI commands (existing in the project)
- Produces: Three executable shell scripts that CI will call in Task 3; scripts exit 0 on success, 1 on failure

## Steps to Execute

### Step 1: Create scripts/ directory if it doesn't exist

```bash
mkdir -p scripts
```

### Step 2: Write scripts/validate-routes.sh

Create this file with executable permissions. This script:
- Finds all route YAML files changed in the current PR/commit range (not test files)
- Calls `dimctl validate` on each one
- Returns exit code 0 if all pass, 1 if any fail

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

### Step 3: Write scripts/test-routes.sh

Create this file with executable permissions. This script:
- Finds all route YAML files changed in the PR
- For each one, looks for a corresponding `.route_test.yaml` file
- Calls `dimctl test -c <route> <test-file>` for each pair
- Returns exit code 0 if all pass, 1 if any fail

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

### Step 4: Write scripts/check-mandatory-fragments.sh

Create this file with executable permissions. This script:
- Finds all route YAML files in the `domains/` directory (excluding governance and test files)
- Checks each one for an import of `governance/fragments`
- Returns exit code 0 if all have the import, 1 otherwise

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

### Step 5: Make all three scripts executable

```bash
chmod +x scripts/validate-routes.sh
chmod +x scripts/test-routes.sh
chmod +x scripts/check-mandatory-fragments.sh
```

Verify they're executable:
```bash
ls -l scripts/*.sh
# Should show -rwxr-xr-x for all three
```

### Step 6: Test scripts locally (dry run)

Before committing, verify the scripts work:

```bash
# All three should handle "no changes" gracefully (exit 0)
# Use origin/master...HEAD to detect diffs (will be empty in initial run)

bash scripts/validate-routes.sh && echo "✅ validate-routes.sh works"
bash scripts/test-routes.sh && echo "✅ test-routes.sh works"
bash scripts/check-mandatory-fragments.sh && echo "✅ check-mandatory-fragments.sh works"
```

Expected: All three exit with code 0 (or skip messages about no files found/changed)

### Step 7: Verify scripts are syntactically correct (shellcheck)

Run shellcheck on each script (if available):

```bash
shellcheck scripts/validate-routes.sh
shellcheck scripts/test-routes.sh
shellcheck scripts/check-mandatory-fragments.sh
```

If shellcheck is not available, that's OK — bash -n can do a basic check:

```bash
bash -n scripts/validate-routes.sh && echo "✅ syntax OK"
bash -n scripts/test-routes.sh && echo "✅ syntax OK"
bash -n scripts/check-mandatory-fragments.sh && echo "✅ syntax OK"
```

### Step 8: Commit

```bash
git add scripts/
git commit -m "chore: add route validation and testing scripts for CI"
```

## Success Criteria

- All three scripts created in `scripts/` directory
- Scripts are executable (chmod +x)
- Scripts parse as valid bash (shellcheck or bash -n passes)
- Scripts exit 0 when run locally (no route files changed, so they skip gracefully)
- Commit is in git log
- Commit message matches: "chore: add route validation and testing scripts for CI"
