#!/bin/bash

# E2E Real Tests Runner Script
# Runs end-to-end tests against a real API server

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
API_PID=""
REPORT_FILE="$SCRIPT_DIR/e2e_report_$(date +%Y%m%d_%H%M%S).txt"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

print_header() {
    echo ""
    echo "========================================"
    echo "$1"
    echo "========================================"
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

usage() {
    echo "Usage: $0 [OPTIONS]"
    echo ""
    echo "Options:"
    echo "  --start-server    Start API server before tests"
    echo "  --stop-server     Stop API server after tests"
    echo "  --report          Generate detailed report"
    echo "  --verbose         Verbose output"
    echo "  --help            Show this help"
    echo ""
    echo "Examples:"
    echo "  $0 --start-server --report"
    echo "  $0 --verbose"
}

check_server() {
    curl -s http://localhost:8080/v1/markets > /dev/null 2>&1
    return $?
}

start_server() {
    print_header "Starting API Server"

    if check_server; then
        print_warning "API server already running on port 8080"
        return 0
    fi

    cd "$PROJECT_ROOT"

    echo "Building API server..."
    go build -o /tmp/perpdex-api ./cmd/api 2>&1 || {
        print_error "Failed to build API server"
        return 1
    }

    echo "Starting API server (mock mode)..."
    /tmp/perpdex-api --port 8080 --mock > /tmp/perpdex-api.log 2>&1 &
    API_PID=$!

    echo "Waiting for server to start..."
    for i in {1..30}; do
        if check_server; then
            print_success "API server started (PID: $API_PID)"
            return 0
        fi
        sleep 1
    done

    print_error "Failed to start API server"
    return 1
}

stop_server() {
    if [ -n "$API_PID" ]; then
        print_header "Stopping API Server"
        kill $API_PID 2>/dev/null || true
        print_success "API server stopped"
    fi
}

run_tests() {
    local verbose=""
    if [ "$VERBOSE" = "1" ]; then
        verbose="-v"
    fi

    print_header "Running E2E Tests"

    cd "$SCRIPT_DIR"

    echo "Test directory: $SCRIPT_DIR"
    echo ""

    # Run tests with go test
    if [ "$GENERATE_REPORT" = "1" ]; then
        echo "Running tests with report generation..."
        go test $verbose -count=1 -timeout=300s ./... 2>&1 | tee "$REPORT_FILE"

        echo ""
        print_success "Report saved to: $REPORT_FILE"
    else
        go test $verbose -count=1 -timeout=300s ./...
    fi
}

generate_summary() {
    print_header "Test Summary"

    if [ -f "$REPORT_FILE" ]; then
        echo "Report file: $REPORT_FILE"
        echo ""

        # Count passed/failed
        passed=$(grep -c "PASS:" "$REPORT_FILE" 2>/dev/null || echo "0")
        failed=$(grep -c "FAIL:" "$REPORT_FILE" 2>/dev/null || echo "0")
        skipped=$(grep -c "SKIP:" "$REPORT_FILE" 2>/dev/null || echo "0")

        echo "Results:"
        echo "  Passed:  $passed"
        echo "  Failed:  $failed"
        echo "  Skipped: $skipped"
    fi
}

# Parse arguments
START_SERVER=0
STOP_SERVER=0
GENERATE_REPORT=0
VERBOSE=0

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
        --help|-h)
            usage
            exit 0
            ;;
        *)
            echo "Unknown option: $1"
            usage
            exit 1
            ;;
    esac
done

# Main execution
trap stop_server EXIT

print_header "PerpDEX E2E Real Tests"
echo "Date: $(date)"
echo "Project: $PROJECT_ROOT"
echo ""

if [ "$START_SERVER" = "1" ]; then
    start_server || exit 1
else
    if ! check_server; then
        print_warning "API server not running. Use --start-server to start it."
        print_warning "Or start it manually: go run ./cmd/api --port 8080 --mock"
        echo ""
    fi
fi

run_tests

if [ "$GENERATE_REPORT" = "1" ]; then
    generate_summary
fi

print_header "Done"
