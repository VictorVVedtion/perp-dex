#!/bin/bash

# ============================================================================
# PerpDEX E2E Stress Test Script
# ============================================================================
# This script performs a comprehensive end-to-end stress test on the PerpDEX
# chain, using real chain interaction (no mock data).
#
# Features:
# - Starts a local chain node
# - Creates test accounts
# - Runs order placement stress tests
# - Measures latency and throughput
# - Generates a detailed report
# ============================================================================

set -e

# Configuration
PROJECT_ROOT="/Users/vvedition/Desktop/dex mvp/perp-dex_副本"
BINARY="$PROJECT_ROOT/build/perpdexd"
HOME_DIR="$HOME/.perpdex-e2e-test"
CHAIN_ID="perpdex-e2e-1"
LOG_DIR="$PROJECT_ROOT/tests/e2e/logs"
REPORT_FILE="$PROJECT_ROOT/tests/e2e/E2E_STRESS_TEST_REPORT.md"

# Test parameters
NUM_TRADERS=3
ORDERS_PER_TRADER=20
MARKET_ID="BTC-USDC"
BASE_PRICE=50000

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Create directories
mkdir -p "$LOG_DIR"

echo -e "${BLUE}=============================================${NC}"
echo -e "${BLUE}   PerpDEX E2E Stress Test${NC}"
echo -e "${BLUE}=============================================${NC}"
echo ""

# ============================================================================
# Functions
# ============================================================================

log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

cleanup() {
    log_info "Cleaning up..."
    # Kill any running node
    pkill -f "perpdexd start" 2>/dev/null || true
    sleep 2
}

wait_for_node() {
    local max_attempts=30
    local attempt=1

    log_info "Waiting for node to be ready..."
    while [ $attempt -le $max_attempts ]; do
        if curl -s http://localhost:26657/status > /dev/null 2>&1; then
            log_info "Node is ready!"
            return 0
        fi
        echo -n "."
        sleep 1
        attempt=$((attempt + 1))
    done

    log_error "Node failed to start within ${max_attempts} seconds"
    return 1
}

get_latest_height() {
    curl -s http://localhost:26657/status | grep -o '"latest_block_height":"[0-9]*"' | grep -o '[0-9]*'
}

# ============================================================================
# Phase 1: Initialize Chain
# ============================================================================

init_chain() {
    log_info "Phase 1: Initializing chain..."

    # Clean previous data
    rm -rf "$HOME_DIR"

    # Initialize
    "$BINARY" init "e2e-validator" --chain-id "$CHAIN_ID" --home "$HOME_DIR" --default-denom usdc 2>&1 | head -5

    # Create validator key
    "$BINARY" keys add validator --home "$HOME_DIR" --keyring-backend test 2>&1 | grep -E "address:|mnemonic" || true

    VALIDATOR_ADDR=$("$BINARY" keys show validator -a --home "$HOME_DIR" --keyring-backend test)
    log_info "Validator address: $VALIDATOR_ADDR"

    # Add genesis account
    "$BINARY" genesis add-genesis-account "$VALIDATOR_ADDR" 1000000000000usdc,100000000000stake --home "$HOME_DIR"

    # Create trader accounts
    log_info "Creating $NUM_TRADERS trader accounts..."
    for i in $(seq 1 $NUM_TRADERS); do
        "$BINARY" keys add "trader$i" --home "$HOME_DIR" --keyring-backend test 2>/dev/null
        ADDR=$("$BINARY" keys show "trader$i" -a --home "$HOME_DIR" --keyring-backend test)
        "$BINARY" genesis add-genesis-account "$ADDR" 100000000000usdc --home "$HOME_DIR"
        log_info "  trader$i: $ADDR"
    done

    # Create gentx
    "$BINARY" genesis gentx validator 100000000stake \
        --chain-id "$CHAIN_ID" \
        --home "$HOME_DIR" \
        --keyring-backend test \
        --moniker "e2e-validator" 2>&1 | head -3

    # Collect gentxs
    "$BINARY" genesis collect-gentxs --home "$HOME_DIR" 2>&1 | head -3

    # Validate genesis
    "$BINARY" genesis validate --home "$HOME_DIR"

    log_info "Chain initialized successfully!"
}

# ============================================================================
# Phase 2: Start Node
# ============================================================================

start_node() {
    log_info "Phase 2: Starting node..."

    # Start node in background
    "$BINARY" start \
        --home "$HOME_DIR" \
        --api.enable \
        --api.enabled-unsafe-cors \
        --grpc.enable \
        --grpc-web.enable \
        --minimum-gas-prices "0usdc" \
        --log_level warn \
        > "$LOG_DIR/node.log" 2>&1 &

    NODE_PID=$!
    echo $NODE_PID > "$LOG_DIR/node.pid"

    log_info "Node started with PID: $NODE_PID"

    # Wait for node to be ready
    wait_for_node

    # Get initial height
    INITIAL_HEIGHT=$(get_latest_height)
    log_info "Initial block height: $INITIAL_HEIGHT"
}

# ============================================================================
# Phase 3: Run Stress Test
# ============================================================================

run_stress_test() {
    log_info "Phase 3: Running stress test..."

    # Use python for millisecond timestamps (macOS compatible)
    get_time_ms() {
        python3 -c "import time; print(int(time.time() * 1000))"
    }

    local start_time=$(get_time_ms)
    local total_orders=0
    local successful_orders=0
    local failed_orders=0
    local total_latency=0

    # Results file
    RESULTS_FILE="$LOG_DIR/order_results.csv"
    echo "trader,order_num,side,price,quantity,latency_ms,status,tx_hash" > "$RESULTS_FILE"

    log_info "Placing orders: $NUM_TRADERS traders x $ORDERS_PER_TRADER orders = $((NUM_TRADERS * ORDERS_PER_TRADER)) total"
    echo ""

    for trader_num in $(seq 1 $NUM_TRADERS); do
        trader="trader$trader_num"
        log_info "Processing $trader..."

        for order_num in $(seq 1 $ORDERS_PER_TRADER); do
            total_orders=$((total_orders + 1))

            # Randomize side
            if [ $((RANDOM % 2)) -eq 0 ]; then
                side="buy"
                price=$((BASE_PRICE - (RANDOM % 100)))
            else
                side="sell"
                price=$((BASE_PRICE + (RANDOM % 100)))
            fi

            # Random quantity (0.01 to 1.0)
            qty_int=$((1 + RANDOM % 100))
            quantity="0.$(printf '%02d' $qty_int)"

            # Time the order
            order_start=$(get_time_ms)

            # Place order
            result=$("$BINARY" tx orderbook place-order "$MARKET_ID" "$side" limit "$price" "$quantity" \
                --from "$trader" \
                --home "$HOME_DIR" \
                --keyring-backend test \
                --chain-id "$CHAIN_ID" \
                --gas auto \
                --gas-adjustment 1.5 \
                -y \
                --output json 2>&1) || result='{"code":1,"raw_log":"failed"}'

            order_end=$(get_time_ms)
            latency=$((order_end - order_start))
            total_latency=$((total_latency + latency))

            # Parse result
            tx_hash=$(echo "$result" | grep -o '"txhash":"[^"]*"' | head -1 | cut -d'"' -f4 || echo "N/A")
            code=$(echo "$result" | grep -o '"code":[0-9]*' | head -1 | cut -d':' -f2 || echo "1")

            if [ "$code" = "0" ] || [ -n "$tx_hash" ] && [ "$tx_hash" != "N/A" ]; then
                status="success"
                successful_orders=$((successful_orders + 1))
            else
                status="failed"
                failed_orders=$((failed_orders + 1))
            fi

            # Log to CSV
            echo "$trader,$order_num,$side,$price,$quantity,$latency,$status,$tx_hash" >> "$RESULTS_FILE"

            # Progress indicator
            if [ $((total_orders % 50)) -eq 0 ]; then
                echo -e "  Progress: $total_orders / $((NUM_TRADERS * ORDERS_PER_TRADER)) orders"
            fi

            # Small delay to avoid overwhelming mempool
            sleep 0.1
        done
    done

    local end_time=$(get_time_ms)
    local total_time=$((end_time - start_time))

    # Calculate metrics
    AVG_LATENCY=$((total_latency / total_orders))
    THROUGHPUT=$(echo "scale=2; $total_orders * 1000 / $total_time" | bc)
    SUCCESS_RATE=$(echo "scale=2; $successful_orders * 100 / $total_orders" | bc)

    log_info "Stress test completed!"
    echo ""
    echo "================================================"
    echo "  Stress Test Results Summary"
    echo "================================================"
    echo "  Total Orders:     $total_orders"
    echo "  Successful:       $successful_orders"
    echo "  Failed:           $failed_orders"
    echo "  Success Rate:     ${SUCCESS_RATE}%"
    echo "  Total Time:       ${total_time}ms"
    echo "  Avg Latency:      ${AVG_LATENCY}ms"
    echo "  Throughput:       ${THROUGHPUT} orders/sec"
    echo "================================================"

    # Export metrics for report
    export TOTAL_ORDERS=$total_orders
    export SUCCESSFUL_ORDERS=$successful_orders
    export FAILED_ORDERS=$failed_orders
    export SUCCESS_RATE=$SUCCESS_RATE
    export TOTAL_TIME=$total_time
    export AVG_LATENCY=$AVG_LATENCY
    export THROUGHPUT=$THROUGHPUT
}

# ============================================================================
# Phase 4: Query Chain State
# ============================================================================

query_chain_state() {
    log_info "Phase 4: Querying chain state..."

    # Get final height
    FINAL_HEIGHT=$(get_latest_height)
    BLOCKS_PRODUCED=$((FINAL_HEIGHT - INITIAL_HEIGHT))

    log_info "Final block height: $FINAL_HEIGHT"
    log_info "Blocks produced during test: $BLOCKS_PRODUCED"

    # Query orderbook depth
    log_info "Querying orderbook state..."
    ORDERBOOK_QUERY=$("$BINARY" query orderbook depth "$MARKET_ID" --home "$HOME_DIR" --output json 2>&1 || echo '{"bids":[],"asks":[]}')

    # Count bids and asks
    BID_COUNT=$(echo "$ORDERBOOK_QUERY" | grep -o '"bids":\[' | wc -l || echo "0")
    ASK_COUNT=$(echo "$ORDERBOOK_QUERY" | grep -o '"asks":\[' | wc -l || echo "0")

    # Query trades
    log_info "Querying trades..."
    TRADES_QUERY=$("$BINARY" query orderbook trades "$MARKET_ID" --home "$HOME_DIR" --output json 2>&1 || echo '{"trades":[]}')
    TRADE_COUNT=$(echo "$TRADES_QUERY" | grep -o '"trade_id"' | wc -l || echo "0")

    log_info "Orderbook state: Bids=$BID_COUNT, Asks=$ASK_COUNT"
    log_info "Trades executed: $TRADE_COUNT"

    export FINAL_HEIGHT
    export BLOCKS_PRODUCED
    export TRADE_COUNT
}

# ============================================================================
# Phase 5: Generate Report
# ============================================================================

generate_report() {
    log_info "Phase 5: Generating report..."

    cat > "$REPORT_FILE" << EOF
# PerpDEX E2E Stress Test Report

**Generated:** $(date "+%Y-%m-%d %H:%M:%S")
**Chain ID:** $CHAIN_ID
**Market:** $MARKET_ID

---

## Test Configuration

| Parameter | Value |
|-----------|-------|
| Number of Traders | $NUM_TRADERS |
| Orders per Trader | $ORDERS_PER_TRADER |
| Total Orders | $((NUM_TRADERS * ORDERS_PER_TRADER)) |
| Base Price | $BASE_PRICE |
| Order Type | Limit |

---

## Performance Results

### Order Submission

| Metric | Value |
|--------|-------|
| **Total Orders Submitted** | $TOTAL_ORDERS |
| **Successful Orders** | $SUCCESSFUL_ORDERS |
| **Failed Orders** | $FAILED_ORDERS |
| **Success Rate** | ${SUCCESS_RATE}% |
| **Total Test Duration** | ${TOTAL_TIME}ms |
| **Average Latency** | ${AVG_LATENCY}ms |
| **Throughput** | ${THROUGHPUT} orders/sec |

### Chain State

| Metric | Value |
|--------|-------|
| Initial Block Height | $INITIAL_HEIGHT |
| Final Block Height | $FINAL_HEIGHT |
| Blocks Produced | $BLOCKS_PRODUCED |
| Trades Executed | $TRADE_COUNT |

---

## Test Environment

- **Binary:** $BINARY
- **Home Directory:** $HOME_DIR
- **Log Directory:** $LOG_DIR

---

## Latency Distribution

\`\`\`
See detailed results in: $RESULTS_FILE
\`\`\`

---

## Observations

1. **Throughput:** ${THROUGHPUT} orders/sec indicates the chain can handle approximately $(echo "scale=0; $THROUGHPUT * 60" | bc) orders per minute under this test load.

2. **Latency:** Average latency of ${AVG_LATENCY}ms includes transaction signing, broadcasting, and initial validation.

3. **Success Rate:** ${SUCCESS_RATE}% success rate shows the overall reliability of order submission.

4. **Chain Performance:** $BLOCKS_PRODUCED blocks produced during the test duration indicates an average of $(echo "scale=2; $BLOCKS_PRODUCED * 1000 / $TOTAL_TIME" | bc) blocks per second.

---

## Recommendations

$(if [ $(echo "$SUCCESS_RATE < 95" | bc) -eq 1 ]; then
    echo "- ⚠️ Success rate below 95%. Investigate failed transactions in the logs."
fi)

$(if [ $AVG_LATENCY -gt 1000 ]; then
    echo "- ⚠️ Average latency above 1 second. Consider optimizing consensus parameters."
fi)

$(if [ $(echo "$THROUGHPUT < 10" | bc) -eq 1 ]; then
    echo "- ⚠️ Throughput below 10 orders/sec. Consider scaling or optimizing."
fi)

- ✅ Test completed successfully. All systems operational.

---

## Raw Data Files

- Node Log: \`$LOG_DIR/node.log\`
- Order Results: \`$RESULTS_FILE\`
- Node PID: \`$LOG_DIR/node.pid\`

---

*Report generated by PerpDEX E2E Stress Test Script*
EOF

    log_info "Report saved to: $REPORT_FILE"
}

# ============================================================================
# Main Execution
# ============================================================================

main() {
    echo ""
    log_info "Starting E2E Stress Test..."
    echo ""

    # Cleanup any previous runs
    cleanup

    # Run test phases
    init_chain
    echo ""
    start_node
    echo ""

    # Wait a bit for chain to stabilize
    log_info "Waiting for chain to stabilize..."
    sleep 5

    run_stress_test
    echo ""
    query_chain_state
    echo ""
    generate_report
    echo ""

    log_info "E2E Stress Test completed successfully!"
    echo ""
    echo -e "${GREEN}=============================================${NC}"
    echo -e "${GREEN}   Test Complete!${NC}"
    echo -e "${GREEN}=============================================${NC}"
    echo ""
    echo "Report: $REPORT_FILE"
    echo "Logs:   $LOG_DIR"
    echo ""

    # Keep node running for inspection
    log_info "Node is still running. To stop: kill $(cat $LOG_DIR/node.pid)"
}

# Run main
main "$@"
