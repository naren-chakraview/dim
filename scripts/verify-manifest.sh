#!/bin/bash
# Verify that the capability manifest doesn't drift from the current implementation
# This script is run in CI to ensure the published manifest is always in sync

set -e

SCHEMA_PATH="schemas/route.schema.json"
COMMITTED_MANIFEST="docs/schemas/capability-manifest.json"
TMP_MANIFEST="/tmp/manifest-generated.json"

# Build dimctl if not already built
if [ ! -f "./dimctl" ]; then
    go build -o ./dimctl ./cmd/dimctl
fi

# Generate fresh manifest from the current schema
./dimctl manifest generate "$SCHEMA_PATH" "$TMP_MANIFEST" 2>&1 | grep -v "^✅"

# Compare capabilities structure (ignore generated_at timestamp and interface_version which may vary)
# The key check is that the same adapters and steps are present
COMMITTED_CAPS=$(jq '.capabilities | map({name: .name, type: .type}) | sort_by(.name)' "$COMMITTED_MANIFEST")
GENERATED_CAPS=$(jq '.capabilities | map({name: .name, type: .type}) | sort_by(.name)' "$TMP_MANIFEST")

if [ "$COMMITTED_CAPS" != "$GENERATED_CAPS" ]; then
    echo ""
    echo "❌ ERROR: Capability manifest has drifted from implementation"
    echo ""
    echo "Committed capabilities:"
    jq '.capabilities | map(.name) | sort' "$COMMITTED_MANIFEST"
    echo ""
    echo "Generated capabilities:"
    jq '.capabilities | map(.name) | sort' "$TMP_MANIFEST"
    echo ""
    echo "To fix, run: ./dimctl manifest generate $SCHEMA_PATH $COMMITTED_MANIFEST"
    exit 1
fi

echo "✅ Capability manifest is in sync with implementation"
rm -f "$TMP_MANIFEST"
