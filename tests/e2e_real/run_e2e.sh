#!/bin/bash

# ============================================================================
# PerpDEX Real E2E Test Runner
# ============================================================================
# This script runs the complete E2E test suite
# Usage: ./run_e2e.sh [options]
# Options:
#   --start-server    Start the API server before tests
#   --stop-server     Stop the API server after tests
#   --verbose         Show detailed output
#   --report          Generate JSON report
#   --quick           Run only quick tests
#   --full            Run all tests including stability tests
# ============================================================================

set -e

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
API_PORT=8080
API_PID_FILE="/tmp/perpdex_e2e_api.pid"
REPORT_DIR="$PROJECT_ROOT/reports"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Default options
START_SERVER=false
STOP_SERVER=false
VERBOSE=false
GENERATE_REPORT=false
QUICK_MODE=false
FULL_MODE=false

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --start-server)
            START_SERVER=true
            shift
            ;;
        --stop-server)
            STOP_SERVER=true
            shift
            ;;
        --verbose|-v)
            VERBOSE=true
            shift
            ;;
        --report)
            GENERATE_REPORT=true
            shift
            ;;
        --quick)
            QUICK_MODE=true
            shift
            ;;
        --full)
            FULL_MODE=true
            shift
            ;;
        --help|-h)
            echo "PerpDEX E2E Test Runner"
            echo ""
            echo "Usage: $0 [options]"
            echo ""
            echo "Options:"
            echo "  --start-server    Start the API server before tests"
            echo "  --stop-server     Stop the API server after tests"
            echo "  --verbose, -v     Show detailed output"
            echo "  --report          Generate JSON report"
            echo "  --quick           Run only quick tests (skip stability)"
            echo "  --full            Run all tests including long stability tests"
            echo "  --help, -h        Show this help message"
            exit 0
            ;;
        *)
            echo "Unknown option: $1"
            exit 1
            ;;
    esac
done

# Functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

check_server() {
    curl -s "http://localhost:$API_PORT/health" > /dev/null 2>&1
    return $?
}

start_api_server() {
    log_info "Starting API server on port $API_PORT..."

    # Check if already running
    if check_server; then
        log_info "API server already running"
        return 0
    fi

    # Build API server if needed
    if [[ ! -f "$PROJECT_ROOT/build/perpdexd" ]]; then
        log_info "Building API server..."
        cd "$PROJECT_ROOT"
        go build -o ./build/perpdexd ./cmd/perpdexd
    fi

    # Start API server
    cd "$PROJECT_ROOT"
    go run ./cmd/api/main.go --port $API_PORT --mock &
    API_PID=$!
    echo $API_PID > "$API_PID_FILE"

    # Wait for server to be ready
    log_info "Waiting for server to start..."
    for i in {1..30}; do
        if check_server; then
            log_success "API server started (PID: $API_PID)"
            return 0
        fi
        sleep 1
    done

    log_error "Failed to start API server"
    return 1
}

stop_api_server() {
    if [[ -f "$API_PID_FILE" ]]; then
        PID=$(cat "$API_PID_FILE")
        if kill -0 "$PID" 2>/dev/null; then
            log_info "Stopping API server (PID: $PID)..."
            kill "$PID"
            rm -f "$API_PID_FILE"
            log_success "API server stopped"
        fi
    fi
}

run_tests() {
    local test_pattern=$1
    local test_name=$2

    log_info "Running $test_name..."

    cd "$SCRIPT_DIR"

    local test_args="-v"
    if [[ "$QUICK_MODE" == "true" ]]; then
        test_args="$test_args -short"
    fi

    if [[ -n "$test_pattern" ]]; then
        test_args="$test_args -run $test_pattern"
    fi

    if go test $test_args ./... 2>&1 | tee /tmp/e2e_test_output.txt; then
        log_success "$test_name completed"
        return 0
    else
        log_error "$test_name failed"
        return 1
    fi
}

generate_report() {
    log_info "Generating test report..."

    mkdir -p "$REPORT_DIR"

    local report_file="$REPORT_DIR/e2e_test_report_$(date +%Y%m%d_%H%M%S).json"

    # Create JSON report
    cat > "$report_file" << EOF
{
    "timestamp": "$(date -Iseconds)",
    "test_suite": "PerpDEX E2E Tests",
    "mode": "$([ "$QUICK_MODE" == "true" ] && echo "quick" || echo "full")",
    "results": {
        "output_file": "/tmp/e2e_test_output.txt"
    }
}
EOF

    log_success "Report saved to: $report_file"
}

# Main execution
main() {
    echo ""
    echo "╔══════════════════════════════════════════════════════════════╗"
    echo "║           PerpDEX E2E Test Suite                             ║"
    echo "╚══════════════════════════════════════════════════════════════╝"
    echo ""

    # Start server if requested
    if [[ "$START_SERVER" == "true" ]]; then
        start_api_server
    fi

    # Check if server is available
    if ! check_server; then
        log_error "API server not available at http://localhost:$API_PORT"
        log_info "Start the server manually or use --start-server option"
        echo ""
        echo "To start the server manually:"
        echo "  cd $PROJECT_ROOT"
        echo "  go run ./cmd/api/main.go --port $API_PORT --mock"
        exit 1
    fi

    log_success "API server is available"
    echo ""

    # Run tests
    local exit_code=0

    # Basic tests
    log_info "════════════════════════════════════════════════════════════"
    if ! run_tests "TestHealthCheck" "Health Check"; then
        exit_code=1
    fi

    # Trading flow tests
    log_info "════════════════════════════════════════════════════════════"
    if ! run_tests "TestCompleteTradingFlow|TestOrderTypes|TestOrderMatching|TestMarketData" "Trading Flow Tests"; then
        exit_code=1
    fi

    # WebSocket tests
    log_info "════════════════════════════════════════════════════════════"
    if ! run_tests "TestWebSocket" "WebSocket Tests"; then
        exit_code=1
    fi

    # Concurrent tests (shorter)
    log_info "════════════════════════════════════════════════════════════"
    if ! run_tests "TestConcurrentOrderPlacement|TestConcurrentMatching" "Concurrent Tests"; then
        exit_code=1
    fi

    # Full mode: run stability and benchmark tests
    if [[ "$FULL_MODE" == "true" ]]; then
        log_info "════════════════════════════════════════════════════════════"
        log_info "Running full test suite (this may take several minutes)..."
        if ! run_tests "TestSystemStability" "Stability Tests"; then
            exit_code=1
        fi
    fi

    # Generate report if requested
    if [[ "$GENERATE_REPORT" == "true" ]]; then
        generate_report
    fi

    # Stop server if requested
    if [[ "$STOP_SERVER" == "true" ]]; then
        stop_api_server
    fi

    # Summary
    echo ""
    echo "════════════════════════════════════════════════════════════════"
    if [[ $exit_code -eq 0 ]]; then
        log_success "All E2E tests completed successfully!"
    else
        log_error "Some E2E tests failed"
    fi
    echo "════════════════════════════════════════════════════════════════"

    exit $exit_code
}

# Run main
main
