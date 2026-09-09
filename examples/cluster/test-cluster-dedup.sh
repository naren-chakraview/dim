#!/bin/bash

# M3.1 Cluster Demo Test Script
# Demonstrates multi-instance deduplication with shared PostgreSQL backend
#
# Preconditions:
#   docker-compose -f docker-compose.cluster.yml up -d
#   Wait for instances to be healthy (docker logs dim-instance-1)
#
# Usage:
#   bash test-cluster-dedup.sh

set -e

echo "═══════════════════════════════════════════════════════════════"
echo "M3.1 Cluster Demo: Multi-Instance Deduplication Test"
echo "═══════════════════════════════════════════════════════════════"
echo ""

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
INSTANCE_1_HOST="localhost:8080"
INSTANCE_2_HOST="localhost:8082"
POSTGRES_HOST="localhost"
POSTGRES_USER="dim"
POSTGRES_PASSWORD="dim_password"
POSTGRES_DB="dim_cluster"

# Test order ID (same for both instances = tests dedup)
ORDER_ID="ORD-CLUSTER-$(date +%s)"
CUSTOMER="Test Customer"
AMOUNT="1500.00"

echo -e "${BLUE}Test Setup:${NC}"
echo "  Instance 1: http://${INSTANCE_1_HOST}"
echo "  Instance 2: http://${INSTANCE_2_HOST}"
echo "  Test Order ID: ${ORDER_ID}"
echo "  Customer: ${CUSTOMER}"
echo "  Amount: ${AMOUNT}"
echo ""

# Step 1: Send message to Instance 1
echo -e "${BLUE}Step 1: Sending order to Instance 1...${NC}"
curl -s -X POST "http://${INSTANCE_1_HOST}/ingest" \
  -H "Content-Type: application/json" \
  -d "{\"order_id\": \"${ORDER_ID}\", \"customer\": \"${CUSTOMER}\", \"amount\": ${AMOUNT}}" \
  || echo -e "${RED}Failed to send to Instance 1${NC}"
echo "✓ Message sent to Instance 1"
sleep 1

# Step 2: Send SAME order to Instance 2
echo -e "${BLUE}Step 2: Sending SAME order to Instance 2 (should be deduplicated)...${NC}"
curl -s -X POST "http://${INSTANCE_2_HOST}/ingest" \
  -H "Content-Type: application/json" \
  -d "{\"order_id\": \"${ORDER_ID}\", \"customer\": \"${CUSTOMER}\", \"amount\": ${AMOUNT}}" \
  || echo -e "${RED}Failed to send to Instance 2${NC}"
echo "✓ Message sent to Instance 2"
sleep 2

# Step 3: Check output files
echo ""
echo -e "${BLUE}Step 3: Checking output files...${NC}"
echo ""

OUTPUT_1="/tmp/dim/output/output-1/orders.jsonl"
OUTPUT_2="/tmp/dim/output/output-2/orders.jsonl"

if [ -f "$OUTPUT_1" ]; then
    COUNT_1=$(grep -c "$ORDER_ID" "$OUTPUT_1" 2>/dev/null || echo "0")
    echo -e "  Instance 1 output: ${GREEN}✓${NC} ($COUNT_1 matching orders)"
else
    COUNT_1="0"
    echo -e "  Instance 1 output: ${YELLOW}?${NC} (file not found yet)"
fi

if [ -f "$OUTPUT_2" ]; then
    COUNT_2=$(grep -c "$ORDER_ID" "$OUTPUT_2" 2>/dev/null || echo "0")
    echo -e "  Instance 2 output: ${GREEN}✓${NC} ($COUNT_2 matching orders)"
else
    COUNT_2="0"
    echo -e "  Instance 2 output: ${YELLOW}?${NC} (file not found yet)"
fi

TOTAL=$((COUNT_1 + COUNT_2))
echo ""
echo -e "${BLUE}Step 4: Verification${NC}"
echo "  Order processed by Instance 1: $COUNT_1 time(s)"
echo "  Order processed by Instance 2: $COUNT_2 time(s)"
echo "  Total (across both instances): $TOTAL time(s)"
echo ""

if [ "$TOTAL" -eq 1 ]; then
    echo -e "${GREEN}✓ SUCCESS: Order was processed ONLY ONCE (deduplication working!)${NC}"
    echo ""
    echo "  This proves that:"
    echo "  - Both instances received the same message"
    echo "  - PostgreSQL dedup prevented duplicate processing"
    echo "  - Multi-instance coordination is working correctly"
    exit 0
elif [ "$TOTAL" -eq 0 ]; then
    echo -e "${YELLOW}⚠ NO RESULTS: Order not found in outputs yet${NC}"
    echo ""
    echo "  Possible causes:"
    echo "  - Instances still starting up (wait and retry)"
    echo "  - PostgreSQL not healthy (check docker logs)"
    echo "  - Routes not configured correctly"
    echo ""
    echo "  Debug steps:"
    echo "  1. Check instance logs: docker logs dim-instance-1"
    echo "  2. Check PostgreSQL: psql -h $POSTGRES_HOST -U $POSTGRES_USER -d $POSTGRES_DB"
    echo "  3. Query dedup table: SELECT * FROM dedup_store WHERE message_id LIKE '%$ORDER_ID%';"
    exit 1
else
    echo -e "${RED}✗ FAILURE: Order was processed $TOTAL times (duplicate processing detected!)${NC}"
    echo ""
    echo "  This indicates:"
    echo "  - PostgreSQL dedup is NOT preventing duplicates"
    echo "  - Cluster coordination is not working"
    echo ""
    echo "  Debug steps:"
    echo "  1. Verify PostgreSQL is running: docker ps | grep postgres"
    echo "  2. Check dedup table: SELECT * FROM dedup_store WHERE message_id LIKE '%$ORDER_ID%';"
    echo "  3. Check instance logs for errors: docker logs dim-instance-1 | grep -i error"
    exit 1
fi
