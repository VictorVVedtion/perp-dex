#!/bin/bash
# Hyperliquid E2E Latency Test Runner

set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(dirname "$(dirname "$SCRIPT_DIR")")"
REPORT_DIR="$PROJECT_DIR/reports"

mkdir -p "$REPORT_DIR"

echo "╔══════════════════════════════════════════════════════════════╗"
echo "║     Hyperliquid E2E Latency Test Suite                       ║"
echo "╚══════════════════════════════════════════════════════════════╝"
echo ""

# Step 1: Check connectivity
echo "=== Step 1: Checking Hyperliquid API Connectivity ==="
echo -n "Testing REST API... "
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" -X POST https://api.hyperliquid.xyz/info \
  -H "Content-Type: application/json" \
  -d '{"type": "meta"}')

if [ "$HTTP_CODE" == "200" ]; then
    echo "✅ OK (HTTP $HTTP_CODE)"
else
    echo "❌ Failed (HTTP $HTTP_CODE)"
    exit 1
fi

echo -n "Testing WebSocket... "
WS_TEST=$(timeout 5 websocat -t wss://api.hyperliquid.xyz/ws 2>&1 || true)
if [ -n "$WS_TEST" ] || [ $? -eq 0 ]; then
    echo "✅ OK"
else
    echo "⚠️  Could not verify (continuing anyway)"
fi
echo ""

# Step 2: Build the test tool
echo "=== Step 2: Building Test Tool ==="
cd "$SCRIPT_DIR"

# Check for gorilla/websocket dependency
if ! go list -m github.com/gorilla/websocket > /dev/null 2>&1; then
    echo "Installing gorilla/websocket..."
    go get github.com/gorilla/websocket
fi

go build -o "$PROJECT_DIR/build/e2e_latency" ./main.go
echo "✅ Build successful"
echo ""

# Step 3: Run the tests
echo "=== Step 3: Running Latency Tests ==="
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
OUTPUT_FILE="$REPORT_DIR/e2e_latency_$TIMESTAMP.json"

"$PROJECT_DIR/build/e2e_latency" -n 50 -ws-duration 30 -o "$OUTPUT_FILE"

echo ""
echo "═══════════════════════════════════════════════════════════════"
echo "                     TEST COMPLETED                             "
echo "═══════════════════════════════════════════════════════════════"
echo "Report saved to: $OUTPUT_FILE"
echo ""
