#!/bin/bash
# Verify the committed capability manifest matches the route schema.
#
# Regenerates the manifest from schemas/route.schema.json with a freshly built
# dimctl (a stale binary would hide schema changes) and compares the full
# capability entries: name, type, description and config_schema.
set -euo pipefail

SCHEMA_PATH="schemas/route.schema.json"
COMMITTED_MANIFEST="docs/schemas/capability-manifest.json"

WORK_DIR="$(mktemp -d)"
trap 'rm -rf "$WORK_DIR"' EXIT
DIMCTL="$WORK_DIR/dimctl"
GENERATED="$WORK_DIR/generated-manifest.json"

go build -o "$DIMCTL" ./cmd/dimctl
"$DIMCTL" manifest generate "$SCHEMA_PATH" "$GENERATED" >/dev/null

if ! diff <(jq -S '.capabilities' "$COMMITTED_MANIFEST") \
          <(jq -S '.capabilities' "$GENERATED") > "$WORK_DIR/diff.txt"; then
    echo "❌ ERROR: Capability manifest has drifted from $SCHEMA_PATH"
    echo ""
    echo "Differences (committed < > generated), first 200 lines:"
    head -n 200 "$WORK_DIR/diff.txt"
    echo ""
    echo "To fix, run: go run ./cmd/dimctl manifest generate $SCHEMA_PATH $COMMITTED_MANIFEST"
    exit 1
fi

echo "✅ Capability manifest is in sync with $SCHEMA_PATH"
