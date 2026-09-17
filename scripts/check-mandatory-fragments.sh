#!/bin/bash
# Lint: Ensure all routes in domains/ have the governance fragment actually resolved into their steps.
# Verifies compiled route output (not raw source), so that commented-out or missing fragment references
# are properly caught.
# Exit 0 if all routes contain the governance baseline in compiled output, 1 otherwise.

set -euo pipefail

ROUTE_FILES=$(find domains -name "*.yaml" -not -name "*.route_test.yaml" -not -name "DOMAIN.yaml" -not -path "*/governance/*" 2>/dev/null || true)

if [ -z "$ROUTE_FILES" ]; then
  echo "No route files found; fragment check skipped."
  exit 0
fi

echo "Checking mandatory governance fragment resolution (compiled routes):"

# Verify dimctl can resolve routes
if ! go run ./cmd/dimctl resolve --help >/dev/null 2>&1; then
  echo "FAIL: dimctl resolve command not available"
  exit 1
fi

FAILED=0
for FILE in $ROUTE_FILES; do
  # Skip templates and error-handling placeholders
  if [[ "$FILE" == *.template.yaml ]] || [[ "$FILE" == *error-path* ]]; then
    continue
  fi

  # Use dimctl resolve to check the COMPILED route (not raw source)
  RESOLVED=$(go run ./cmd/dimctl resolve "$FILE" 2>&1) || {
    echo "  FAIL: $FILE failed to resolve"
    FAILED=1
    continue
  }

  # Check if the resolved output contains the authorize step from governance-baseline
  # The governance-baseline fragment injects: authorize: mode: rbac
  if echo "$RESOLVED" | grep -q "mode: rbac"; then
    echo "  OK: $FILE (governance-baseline resolved)"
  else
    echo "  FAIL: $FILE does not contain governance-baseline in resolved output"
    FAILED=1
  fi
done

exit $FAILED
