#!/bin/bash
# Run fixture-based tests for all routes with corresponding .route_test.yaml files.
# Exit 0 if all tests pass, 1 if any fail.

set -euo pipefail

ROUTE_FILES=$(git diff --name-only --diff-filter=ACM origin/master...HEAD | grep -E '^domains/.*\.yaml$' | grep -vE '(_test\.yaml|DOMAIN\.yaml)' || echo "")

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
