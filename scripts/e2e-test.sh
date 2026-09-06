#!/bin/bash
# E2E Test Runner
# Starts docker-compose services, runs e2e tests, and cleans up
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
DEPLOY_DIR="$PROJECT_ROOT/deploy"

# Configuration
COMPOSE_FILE="$DEPLOY_DIR/docker-compose.e2e.yml"
COMPOSE_PROJECT="dim-e2e"
COMPOSE_WAIT_TIMEOUT=60
TEST_TIMEOUT=300

echo "🚀 DIM E2E Test Suite"
echo "===================="
echo ""

# Cleanup function
cleanup() {
	local exit_code=$?
	echo ""
	echo "🧹 Cleaning up..."
	docker-compose -f "$COMPOSE_FILE" -p "$COMPOSE_PROJECT" down -v 2>/dev/null || true
	exit $exit_code
}

trap cleanup EXIT

# Start services
echo "📦 Starting services (docker-compose)..."
cd "$DEPLOY_DIR"
docker-compose -f docker-compose.e2e.yml -p "$COMPOSE_PROJECT" up -d

# Wait for services to be healthy
echo "⏳ Waiting for services to be ready (timeout: ${COMPOSE_WAIT_TIMEOUT}s)..."
WAITED=0
HEALTHY=0
REQUIRED_HEALTHY=5  # Kafka, Zookeeper, Postgres, MinIO, Prometheus

while [ $WAITED -lt $COMPOSE_WAIT_TIMEOUT ] && [ $HEALTHY -lt $REQUIRED_HEALTHY ]; do
	HEALTHY=$(docker-compose -f docker-compose.e2e.yml -p "$COMPOSE_PROJECT" ps | grep -c "healthy" || echo 0)
	if [ $HEALTHY -lt $REQUIRED_HEALTHY ]; then
		echo "  Status: $HEALTHY/$REQUIRED_HEALTHY services healthy..."
		sleep 2
		WAITED=$((WAITED + 2))
	fi
done

if [ $HEALTHY -lt $REQUIRED_HEALTHY ]; then
	echo "❌ Services did not become healthy in time"
	docker-compose -f docker-compose.e2e.yml -p "$COMPOSE_PROJECT" ps
	exit 1
fi

echo "✅ All services healthy!"
echo ""

# Run e2e tests
echo "🧪 Running e2e tests..."
cd "$PROJECT_ROOT"

# Set timeout for tests
timeout $TEST_TIMEOUT go test -tags e2e -v -race ./internal/integration/... -run "E2E" 2>&1 | tee /tmp/e2e-test-output.log

TEST_EXIT_CODE=${PIPESTATUS[0]}

echo ""
if [ $TEST_EXIT_CODE -eq 0 ]; then
	echo "✅ All e2e tests passed!"
else
	echo "❌ E2E tests failed (exit code: $TEST_EXIT_CODE)"
	echo ""
	echo "📋 Test output:"
	tail -50 /tmp/e2e-test-output.log
fi

exit $TEST_EXIT_CODE
