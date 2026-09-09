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
