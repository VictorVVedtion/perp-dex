#!/bin/bash

# PerpDEX Matching Benchmark Script
# Usage: ./run_benchmark.sh [orders_per_side] [concurrency]

ORDERS=${1:-10000}       # Default: 10000 orders per side
CONCURRENCY=${2:-200}    # Default: 200 concurrent connections
MARKET=${3:-BTC-USDC}
PRICE=${4:-50000}
QTY=${5:-0.01}
API_URL="http://localhost:8080"

echo "=========================================="
echo "PerpDEX Matching Benchmark"
echo "=========================================="
echo "Orders per side: $ORDERS (total: $((ORDERS * 2)))"
echo "Concurrency:     $CONCURRENCY"
echo "Market:          $MARKET"
echo "Price:           $PRICE"
echo "Quantity:        $QTY"
echo "=========================================="
echo ""

# Check if API is running
echo -n "Checking API server... "
if ! curl -s "$API_URL/health" > /dev/null 2>&1; then
    echo "FAILED"
    echo ""
    echo "API server is not running. Start it with:"
    echo "  go run ./cmd/api --real --no-rate-limit"
    exit 1
fi
echo "OK"
echo ""

# Create reports directory if not exists
mkdir -p reports

# Generate report filename
REPORT_FILE="reports/benchmark_$(date +%Y%m%d_%H%M%S).json"

# Run benchmark
echo "Starting benchmark..."
echo ""

go run ./scripts/benchmark_matching.go \
    -n "$ORDERS" \
    -c "$CONCURRENCY" \
    -market "$MARKET" \
    -price "$PRICE" \
    -qty "$QTY" \
    -o "$REPORT_FILE"

echo ""
echo "Report saved to: $REPORT_FILE"
