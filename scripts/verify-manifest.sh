#!/bin/bash
# Verify that the capability manifest doesn't drift from the current implementation
# This script is run in CI to ensure the published manifest is always in sync

set -e

MANIFEST_PATH="docs/schemas/capability-manifest.json"
TMP_MANIFEST="/tmp/manifest.json"

# Generate fresh manifest
go run ./cmd/dimctl capabilities > "$TMP_MANIFEST"

# Compare capabilities and schema_version (ignore generated_at timestamp)
if ! diff -u \
    <(jq '{capabilities: .capabilities, schema_version: .schema_version, interface_version: .interface_version}' "$MANIFEST_PATH") \
    <(jq '{capabilities: .capabilities, schema_version: .schema_version, interface_version: .interface_version}' "$TMP_MANIFEST") \
    > /dev/null 2>&1; then
    echo "ERROR: Capability manifest has drifted from implementation"
    echo "Expected (committed):"
    jq '.capabilities | length' "$MANIFEST_PATH"
    echo ""
    echo "Got (current):"
    jq '.capabilities | length' "$TMP_MANIFEST"
    echo ""
    echo "Run: go run ./cmd/dimctl capabilities > $MANIFEST_PATH"
    exit 1
fi

echo "✓ Capability manifest is in sync with implementation"
rm -f "$TMP_MANIFEST"
