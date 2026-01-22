#!/bin/bash

# ============================================
# Real Chain E2E Test Runner
# ============================================
# This script runs E2E tests against a REAL chain
# It handles chain startup, test execution, and cleanup
# ============================================

set -e

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
BINARY="$PROJECT_ROOT/build/perpdexd"
HOME_DIR="$PROJECT_ROOT/.perpdex-test"
CHAIN_ID="perpdex-test-1"
LOG_FILE="$HOME_DIR/chain.log"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Logging functions
log_info() { echo -e "${BLUE}[INFO]${NC} $1"; }
log_success() { echo -e "${GREEN}[SUCCESS]${NC} $1"; }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }

# Parse arguments
SKIP_BUILD=false
SKIP_INIT=false
AUTO_START=true
CLEANUP=false
VERBOSE=false
TEST_FILTER=""

while [[ $# -gt 0 ]]; do
    case $1 in
        --skip-build)
            SKIP_BUILD=true
            shift
            ;;
        --skip-init)
            SKIP_INIT=true
            shift
            ;;
        --no-auto-start)
            AUTO_START=false
            shift
            ;;
        --cleanup)
            CLEANUP=true
            shift
            ;;
        --verbose|-v)
            VERBOSE=true
            shift
            ;;
        --filter|-f)
            TEST_FILTER="$2"
            shift 2
            ;;
        --help|-h)
            echo "Usage: $0 [options]"
            echo ""
            echo "Options:"
            echo "  --skip-build      Skip building the binary"
            echo "  --skip-init       Skip chain initialization (use existing)"
            echo "  --no-auto-start   Don't auto-start chain (assume already running)"
            echo "  --cleanup         Clean up chain data after tests"
            echo "  --verbose, -v     Enable verbose output"
            echo "  --filter, -f      Filter tests by pattern"
            echo "  --help, -h        Show this help"
            exit 0
            ;;
        *)
            log_error "Unknown option: $1"
            exit 1
            ;;
    esac
done

# ============================================
# Functions
# ============================================

build_binary() {
    if [ "$SKIP_BUILD" = true ]; then
        log_info "Skipping build (--skip-build)"
        return 0
    fi

    log_info "Building perpdexd..."
    cd "$PROJECT_ROOT"
    make build

    if [ ! -f "$BINARY" ]; then
        log_error "Binary not found at $BINARY"
        exit 1
    fi

    log_success "Binary built successfully"
}

init_chain() {
    if [ "$SKIP_INIT" = true ]; then
        log_info "Skipping chain initialization (--skip-init)"
        return 0
    fi

    log_info "Initializing test chain..."

    # Clean previous data
    rm -rf "$HOME_DIR"

    # Initialize chain
    $BINARY init test-node --chain-id "$CHAIN_ID" --home "$HOME_DIR" > /dev/null 2>&1

    # Create test accounts
    log_info "Creating test accounts..."
    $BINARY keys add validator --keyring-backend test --home "$HOME_DIR" > /dev/null 2>&1
    $BINARY keys add trader1 --keyring-backend test --home "$HOME_DIR" > /dev/null 2>&1
    $BINARY keys add trader2 --keyring-backend test --home "$HOME_DIR" > /dev/null 2>&1
    $BINARY keys add trader3 --keyring-backend test --home "$HOME_DIR" > /dev/null 2>&1

    # Get addresses
    VALIDATOR_ADDR=$($BINARY keys show validator --keyring-backend test --home "$HOME_DIR" -a)
    TRADER1_ADDR=$($BINARY keys show trader1 --keyring-backend test --home "$HOME_DIR" -a)
    TRADER2_ADDR=$($BINARY keys show trader2 --keyring-backend test --home "$HOME_DIR" -a)
    TRADER3_ADDR=$($BINARY keys show trader3 --keyring-backend test --home "$HOME_DIR" -a)

    log_info "Validator: $VALIDATOR_ADDR"
    log_info "Trader1: $TRADER1_ADDR"
    log_info "Trader2: $TRADER2_ADDR"
    log_info "Trader3: $TRADER3_ADDR"

    # Add genesis accounts with initial balance
    INITIAL_BALANCE="10000000000000usdc,10000000000ubtc,10000000000ueth"
    $BINARY add-genesis-account "$VALIDATOR_ADDR" "$INITIAL_BALANCE" --home "$HOME_DIR" --keyring-backend test
    $BINARY add-genesis-account "$TRADER1_ADDR" "$INITIAL_BALANCE" --home "$HOME_DIR" --keyring-backend test
    $BINARY add-genesis-account "$TRADER2_ADDR" "$INITIAL_BALANCE" --home "$HOME_DIR" --keyring-backend test
    $BINARY add-genesis-account "$TRADER3_ADDR" "$INITIAL_BALANCE" --home "$HOME_DIR" --keyring-backend test

    # Create validator gentx
    $BINARY gentx validator 1000000000usdc --chain-id "$CHAIN_ID" --home "$HOME_DIR" --keyring-backend test > /dev/null 2>&1

    # Collect gentxs
    $BINARY collect-gentxs --home "$HOME_DIR" > /dev/null 2>&1

    # Validate genesis
    $BINARY validate-genesis --home "$HOME_DIR" > /dev/null 2>&1

    log_success "Chain initialized successfully"
}

start_chain() {
    if [ "$AUTO_START" = false ]; then
        log_info "Skipping chain start (--no-auto-start)"
        return 0
    fi

    # Check if already running
    if curl -s http://localhost:26657/status > /dev/null 2>&1; then
        log_info "Chain already running"
        return 0
    fi

    log_info "Starting chain..."

    # Create log directory
    mkdir -p "$(dirname "$LOG_FILE")"

    # Start chain in background
    $BINARY start \
        --home "$HOME_DIR" \
        --minimum-gas-prices "0usdc" \
        --log_level "info" \
        > "$LOG_FILE" 2>&1 &

    CHAIN_PID=$!
    echo $CHAIN_PID > "$HOME_DIR/chain.pid"

    log_info "Chain started with PID $CHAIN_PID"

    # Wait for chain to be ready
    log_info "Waiting for chain to be ready..."
    for i in {1..60}; do
        if curl -s http://localhost:26657/status > /dev/null 2>&1; then
            # Check if blocks are being produced
            HEIGHT=$(curl -s http://localhost:26657/status | jq -r '.result.sync_info.latest_block_height')
            if [ "$HEIGHT" != "null" ] && [ "$HEIGHT" -gt "0" ]; then
                log_success "Chain ready at height $HEIGHT"
                return 0
            fi
        fi
        sleep 1
        if [ $((i % 10)) -eq 0 ]; then
            log_info "Still waiting... ($i/60)"
        fi
    done

    log_error "Chain failed to start within 60 seconds"
    cat "$LOG_FILE" | tail -50
    exit 1
}

stop_chain() {
    if [ -f "$HOME_DIR/chain.pid" ]; then
        PID=$(cat "$HOME_DIR/chain.pid")
        if kill -0 "$PID" 2>/dev/null; then
            log_info "Stopping chain (PID $PID)..."
            kill "$PID" 2>/dev/null || true
            sleep 2
            # Force kill if still running
            if kill -0 "$PID" 2>/dev/null; then
                kill -9 "$PID" 2>/dev/null || true
            fi
        fi
        rm -f "$HOME_DIR/chain.pid"
    fi
}

run_tests() {
    log_info "Running E2E tests..."

    cd "$PROJECT_ROOT"

    # Build test filter
    TEST_ARGS="-v"
    if [ -n "$TEST_FILTER" ]; then
        TEST_ARGS="$TEST_ARGS -run $TEST_FILTER"
    fi

    # Set environment variables
    export PERPDEX_RPC_URL="http://localhost:26657"
    export PERPDEX_API_URL="http://localhost:1317"
    export PERPDEX_CHAIN_ID="$CHAIN_ID"
    export PERPDEX_AUTO_START="false"  # We handle startup
    export PERPDEX_VERBOSE="$VERBOSE"

    # Run tests
    if go test $TEST_ARGS ./tests/e2e_chain/... -count=1; then
        log_success "All tests passed!"
        return 0
    else
        log_error "Some tests failed"
        return 1
    fi
}

cleanup() {
    if [ "$CLEANUP" = true ]; then
        log_info "Cleaning up..."
        stop_chain
        rm -rf "$HOME_DIR"
        log_success "Cleanup complete"
    else
        stop_chain
        log_info "Chain data preserved at $HOME_DIR"
        log_info "Use --cleanup to remove"
    fi
}

# ============================================
# Main
# ============================================

main() {
    echo "=========================================="
    echo "  PerpDEX Real Chain E2E Test Runner"
    echo "=========================================="

    # Trap cleanup on exit
    trap cleanup EXIT

    # Build
    build_binary

    # Initialize
    init_chain

    # Start chain
    start_chain

    # Run tests
    run_tests
    TEST_RESULT=$?

    echo "=========================================="
    if [ $TEST_RESULT -eq 0 ]; then
        log_success "E2E Tests Complete - All Passed"
    else
        log_error "E2E Tests Complete - Some Failed"
    fi
    echo "=========================================="

    exit $TEST_RESULT
}

main "$@"
