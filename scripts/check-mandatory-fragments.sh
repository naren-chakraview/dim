#!/bin/bash
# Lint: Ensure all routes in domains/ import the governance fragments.
# Exit 0 if all import governance fragments, 1 otherwise.

set -euo pipefail

ROUTE_FILES=$(find domains -name "*.yaml" -not -name "*.route_test.yaml" -not -name "DOMAIN.yaml" -not -path "*/governance/*" 2>/dev/null || true)

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
