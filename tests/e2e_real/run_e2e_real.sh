#!/bin/bash

# E2E Real Tests Runner Script (NO MOCK MODE)
# Runs end-to-end tests against a REAL API server with in-memory orderbook engine
# This tests the actual matching engine without any mock data

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
API_PID=""
REPORT_DIR="$SCRIPT_DIR/logs"
REPORT_FILE="$REPORT_DIR/e2e_real_$(date +%Y%m%d_%H%M%S).txt"
API_LOG="$REPORT_DIR/api_server.log"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

print_header() {
    echo ""
    echo -e "${CYAN}════════════════════════════════════════════════════════════${NC}"
    echo -e "${CYAN}  $1${NC}"
    echo -e "${CYAN}════════════════════════════════════════════════════════════${NC}"
    echo ""
}

print_success() {
    echo -e "${GREEN}✓ $1${NC}"
}

print_error() {
    echo -e "${RED}✗ $1${NC}"
}

print_warning() {
    echo -e "${YELLOW}⚠ $1${NC}"
}

print_info() {
    echo -e "${BLUE}ℹ $1${NC}"
}

usage() {
    echo "Usage: $0 [OPTIONS] [TEST_FILTER]"
    echo ""
    echo "Options:"
    echo "  --start-server    Start API server in REAL mode (no mock)"
    echo "  --stop-server     Stop API server after tests"
    echo "  --report          Generate detailed report"
    echo "  --verbose, -v     Verbose output"
    echo "  --parallel        Run tests in parallel"
    echo "  --benchmark       Run benchmark tests"
    echo "  --help, -h        Show this help"
    echo ""
    echo "Test Filters:"
    echo "  trading           Run trading flow tests only"
    echo "  riverpool         Run RiverPool tests only"
    echo "  websocket         Run WebSocket tests only"
    echo "  integration       Run integration tests only"
    echo "  comprehensive     Run comprehensive API tests only"
    echo "  benchmark         Run latency benchmark tests"
    echo "  all               Run all tests (default)"
    echo ""
    echo "Examples:"
    echo "  $0 --start-server --report all"
    echo "  $0 --verbose riverpool"
    echo "  $0 --benchmark"
}

check_server() {
    curl -s http://localhost:8080/health > /dev/null 2>&1
    return $?
}

check_server_mode() {
    local response=$(curl -s http://localhost:8080/health 2>/dev/null)
    if echo "$response" | grep -q '"mode":"real"'; then
        echo "real"
    elif echo "$response" | grep -q '"mode":"mock"'; then
        echo "mock"
    else
        echo "unknown"
    fi
}

start_server() {
    print_header "Starting API Server (REAL MODE)"

    mkdir -p "$REPORT_DIR"

    if check_server; then
        local mode=$(check_server_mode)
        if [ "$mode" = "real" ]; then
            print_success "API server already running in REAL mode"
            return 0
        elif [ "$mode" = "mock" ]; then
            print_warning "API server running in MOCK mode"
            print_info "Stopping mock server and starting real server..."
            # Try to stop existing server
            pkill -f "perpdex-api" 2>/dev/null || true
            sleep 2
        fi
    fi

    cd "$PROJECT_ROOT"

    print_info "Building API server..."
    go build -o /tmp/perpdex-api ./cmd/api 2>&1 || {
        print_error "Failed to build API server"
        return 1
    }
    print_success "API server built"

    print_info "Starting API server in REAL mode (no mock data, no rate limit)..."
    # Start WITHOUT --mock flag for real mode, with --no-rate-limit for E2E testing
    /tmp/perpdex-api --port 8080 --no-rate-limit > "$API_LOG" 2>&1 &
    API_PID=$!

    print_info "Waiting for server to start..."
    for i in {1..30}; do
        if check_server; then
            local mode=$(check_server_mode)
            print_success "API server started (PID: $API_PID, Mode: $mode)"

            if [ "$mode" != "real" ]; then
                print_warning "Server started but not in real mode!"
            fi
            return 0
        fi
        sleep 1
        echo -n "."
    done
    echo ""

    print_error "Failed to start API server"
    print_info "Check logs at: $API_LOG"
    return 1
}

stop_server() {
    if [ -n "$API_PID" ]; then
        print_header "Stopping API Server"
        kill $API_PID 2>/dev/null || true
        print_success "API server stopped (PID: $API_PID)"
    fi
}

run_tests() {
    local verbose=""
    local parallel=""
    local filter="$1"
    local run_pattern=""

    if [ "$VERBOSE" = "1" ]; then
        verbose="-v"
    fi

    if [ "$PARALLEL" = "1" ]; then
        parallel="-parallel 4"
    fi

    # Set test pattern based on filter
    case $filter in
        trading)
            run_pattern="-run TestTradingFlow"
            print_info "Running trading flow tests..."
            ;;
        riverpool)
            run_pattern="-run TestRiverpool"
            print_info "Running RiverPool tests..."
            ;;
        websocket)
            run_pattern="-run TestWebSocket"
            print_info "Running WebSocket tests..."
            ;;
        integration)
            run_pattern="-run TestIntegration"
            print_info "Running integration tests..."
            ;;
        comprehensive)
            run_pattern="-run TestAPI"
            print_info "Running comprehensive API tests..."
            ;;
        benchmark)
            run_pattern="-run Benchmark"
            print_info "Running benchmark tests..."
            ;;
        all|"")
            run_pattern=""
            print_info "Running ALL tests..."
            ;;
        *)
            print_error "Unknown filter: $filter"
            usage
            exit 1
            ;;
    esac

    print_header "Running E2E Tests (REAL MODE - No Mock Data)"

    cd "$SCRIPT_DIR"

    echo "Test directory: $SCRIPT_DIR"
    echo "Report directory: $REPORT_DIR"
    echo ""

    local test_cmd="go test $verbose $parallel -count=1 -timeout=600s $run_pattern ./..."

    print_info "Command: $test_cmd"
    echo ""

    mkdir -p "$REPORT_DIR"

    # Run tests
    if [ "$GENERATE_REPORT" = "1" ]; then
        $test_cmd 2>&1 | tee "$REPORT_FILE"
        echo ""
        print_success "Report saved to: $REPORT_FILE"
    else
        $test_cmd
    fi
}

generate_summary() {
    print_header "Test Summary"

    if [ -f "$REPORT_FILE" ]; then
        print_info "Report file: $REPORT_FILE"
        echo ""

        # Count results
        local total_tests=$(grep -c "^--- " "$REPORT_FILE" 2>/dev/null || echo "0")
        local passed=$(grep -c "^--- PASS:" "$REPORT_FILE" 2>/dev/null || echo "0")
        local failed=$(grep -c "^--- FAIL:" "$REPORT_FILE" 2>/dev/null || echo "0")
        local skipped=$(grep -c "^--- SKIP:" "$REPORT_FILE" 2>/dev/null || echo "0")

        echo "Results:"
        echo -e "  ${GREEN}Passed:${NC}  $passed"
        echo -e "  ${RED}Failed:${NC}  $failed"
        echo -e "  ${YELLOW}Skipped:${NC} $skipped"
        echo "  ─────────────"
        echo "  Total:   $total_tests"

        # Extract latency info if available
        echo ""
        if grep -q "Average Latency" "$REPORT_FILE"; then
            echo "Performance Metrics:"
            grep -E "Average Latency|P50 Latency|P95 Latency|P99 Latency" "$REPORT_FILE" | head -8
        fi
    fi
}

# Parse arguments
START_SERVER=0
STOP_SERVER=0
GENERATE_REPORT=0
VERBOSE=0
PARALLEL=0
BENCHMARK=0
TEST_FILTER=""

while [[ $# -gt 0 ]]; do
    case $1 in
        --start-server)
            START_SERVER=1
            shift
            ;;
        --stop-server)
            STOP_SERVER=1
            shift
            ;;
        --report)
            GENERATE_REPORT=1
            shift
            ;;
        --verbose|-v)
            VERBOSE=1
            shift
            ;;
        --parallel)
            PARALLEL=1
            shift
            ;;
        --benchmark)
            TEST_FILTER="benchmark"
            shift
            ;;
        --help|-h)
            usage
            exit 0
            ;;
        trading|riverpool|websocket|integration|comprehensive|benchmark|all)
            TEST_FILTER="$1"
            shift
            ;;
        *)
            echo "Unknown option: $1"
            usage
            exit 1
            ;;
    esac
done

# Default filter
if [ -z "$TEST_FILTER" ]; then
    TEST_FILTER="all"
fi

# Main execution
if [ "$STOP_SERVER" = "1" ]; then
    trap stop_server EXIT
fi

print_header "PerpDEX E2E Real Tests (NO MOCK DATA)"
echo "Date: $(date)"
echo "Project: $PROJECT_ROOT"
echo "Test Filter: $TEST_FILTER"
echo ""

if [ "$START_SERVER" = "1" ]; then
    start_server || exit 1
else
    if ! check_server; then
        print_warning "API server not running!"
        print_info "Use --start-server to start it automatically"
        print_info "Or start manually: go run ./cmd/api --port 8080"
        echo ""
        print_info "Note: Tests will skip if server is not available"
        echo ""
    else
        local mode=$(check_server_mode)
        print_info "API server is running in $mode mode"
        if [ "$mode" = "mock" ]; then
            print_warning "Server is in MOCK mode. For real testing, restart without --mock flag"
        fi
    fi
fi

run_tests "$TEST_FILTER"

if [ "$GENERATE_REPORT" = "1" ]; then
    generate_summary
fi

print_header "Done"
