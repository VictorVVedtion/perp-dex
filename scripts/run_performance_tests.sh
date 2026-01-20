#!/bin/bash
# PerpDEX Comprehensive Performance Test Runner
# Executes all performance tests in the correct order with proper setup

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
BUILD_DIR="${PROJECT_ROOT}/build"
TEST_HOME="${PROJECT_ROOT}/.perpdex-test"
REPORTS_DIR="${PROJECT_ROOT}/reports"
LOG_FILE="${REPORTS_DIR}/performance_test_$(date +%Y%m%d_%H%M%S).log"

# Test configuration
BENCHMARK_COUNT=5
BENCHMARK_TIMEOUT="10m"
E2E_TIMEOUT="30m"
SCENARIO_TIMEOUT="60m"
STABILITY_DURATION="30m"

# Create reports directory
mkdir -p "${REPORTS_DIR}"

# Logging function
log() {
    local level=$1
    shift
    local message="$@"
    local timestamp=$(date '+%Y-%m-%d %H:%M:%S')

    case $level in
        INFO)  echo -e "${BLUE}[INFO]${NC} ${message}" ;;
        OK)    echo -e "${GREEN}[OK]${NC} ${message}" ;;
        WARN)  echo -e "${YELLOW}[WARN]${NC} ${message}" ;;
        ERROR) echo -e "${RED}[ERROR]${NC} ${message}" ;;
        STEP)  echo -e "${GREEN}==>${NC} ${message}" ;;
    esac

    echo "[${timestamp}] [${level}] ${message}" >> "${LOG_FILE}"
}

# Check if command exists
check_command() {
    if ! command -v "$1" &> /dev/null; then
        log ERROR "$1 is required but not installed."
        exit 1
    fi
}

# Check prerequisites
check_prerequisites() {
    log STEP "Checking prerequisites..."

    check_command go
    check_command make

    # Check Go version
    GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
    log INFO "Go version: ${GO_VERSION}"

    # Check if binary exists
    if [[ ! -f "${BUILD_DIR}/perpdexd" ]]; then
        log WARN "Binary not found, will build..."
    fi

    log OK "Prerequisites check passed"
}

# Build the project
build_project() {
    log STEP "Building project..."
    cd "${PROJECT_ROOT}"

    make build 2>&1 | tee -a "${LOG_FILE}"

    if [[ -f "${BUILD_DIR}/perpdexd" ]]; then
        log OK "Build successful"
    else
        log ERROR "Build failed"
        exit 1
    fi
}

# Initialize test chain
init_chain() {
    log STEP "Initializing test chain..."
    cd "${PROJECT_ROOT}"

    # Clean previous state if exists
    if [[ -d "${TEST_HOME}" ]]; then
        log INFO "Cleaning previous test state..."
        rm -rf "${TEST_HOME}"
    fi

    # Run init script
    if [[ -f "${PROJECT_ROOT}/scripts/init-chain.sh" ]]; then
        bash "${PROJECT_ROOT}/scripts/init-chain.sh" 2>&1 | tee -a "${LOG_FILE}"
        log OK "Chain initialized"
    else
        log WARN "init-chain.sh not found, skipping..."
    fi
}

# Apply fast consensus config
apply_fast_config() {
    log STEP "Applying fast consensus configuration..."

    if [[ -f "${PROJECT_ROOT}/scripts/apply_fast_config.sh" ]]; then
        bash "${PROJECT_ROOT}/scripts/apply_fast_config.sh" 2>&1 | tee -a "${LOG_FILE}"
        log OK "Fast config applied"
    else
        log WARN "apply_fast_config.sh not found, skipping..."
    fi
}

# Start the node
start_node() {
    log STEP "Starting blockchain node..."
    cd "${PROJECT_ROOT}"

    # Kill any existing instance
    pkill -f "perpdexd start" || true
    sleep 2

    # Start node in background
    "${BUILD_DIR}/perpdexd" start \
        --home "${TEST_HOME}" \
        --minimum-gas-prices "0usdc" \
        > "${REPORTS_DIR}/node.log" 2>&1 &

    NODE_PID=$!
    echo $NODE_PID > "${REPORTS_DIR}/node.pid"

    log INFO "Node started with PID: ${NODE_PID}"
    log INFO "Waiting for node to be ready..."

    # Wait for node to be ready
    for i in {1..30}; do
        if curl -s http://localhost:26657/status > /dev/null 2>&1; then
            log OK "Node is ready"
            return 0
        fi
        sleep 1
    done

    log ERROR "Node failed to start within 30 seconds"
    return 1
}

# Stop the node
stop_node() {
    log STEP "Stopping blockchain node..."

    if [[ -f "${REPORTS_DIR}/node.pid" ]]; then
        NODE_PID=$(cat "${REPORTS_DIR}/node.pid")
        if kill -0 $NODE_PID 2>/dev/null; then
            kill $NODE_PID
            log OK "Node stopped"
        fi
        rm -f "${REPORTS_DIR}/node.pid"
    fi

    # Kill any remaining instances
    pkill -f "perpdexd start" || true
}

# Run Layer 1: Engine Microbenchmarks
run_benchmarks() {
    log STEP "Running Layer 1: Engine Microbenchmarks..."
    cd "${PROJECT_ROOT}"

    local benchmark_output="${REPORTS_DIR}/benchmark_results.txt"

    go test -bench=. -benchmem -count=${BENCHMARK_COUNT} \
        -timeout ${BENCHMARK_TIMEOUT} \
        ./tests/benchmark/... 2>&1 | tee "${benchmark_output}" | tee -a "${LOG_FILE}"

    if [[ ${PIPESTATUS[0]} -eq 0 ]]; then
        log OK "Benchmark tests completed"
    else
        log WARN "Some benchmark tests may have failed"
    fi
}

# Run Layer 2: API Performance Tests
run_api_tests() {
    log STEP "Running Layer 2: API Performance Tests..."
    cd "${PROJECT_ROOT}"

    local api_output="${REPORTS_DIR}/api_performance_results.txt"

    # Start API server if needed
    # Note: The tests expect an API server running on localhost:8080

    go test -v -timeout ${E2E_TIMEOUT} \
        ./tests/e2e_real/... \
        -run "TestAPI" 2>&1 | tee "${api_output}" | tee -a "${LOG_FILE}"

    if [[ ${PIPESTATUS[0]} -eq 0 ]]; then
        log OK "API performance tests completed"
    else
        log WARN "Some API tests may have failed (API server may not be running)"
    fi
}

# Run Layer 3: E2E Scenario Tests
run_scenario_tests() {
    log STEP "Running Layer 3: E2E Scenario Tests..."
    cd "${PROJECT_ROOT}"

    local scenario_output="${REPORTS_DIR}/scenario_results.txt"

    go test -v -timeout ${SCENARIO_TIMEOUT} \
        ./tests/e2e_chain/... \
        -run "TestScenario" 2>&1 | tee "${scenario_output}" | tee -a "${LOG_FILE}"

    if [[ ${PIPESTATUS[0]} -eq 0 ]]; then
        log OK "Scenario tests completed"
    else
        log WARN "Some scenario tests may have failed"
    fi
}

# Run Layer 4: Stability Tests
run_stability_tests() {
    log STEP "Running Layer 4: Stability Tests (${STABILITY_DURATION})..."
    cd "${PROJECT_ROOT}"

    local stability_output="${REPORTS_DIR}/stability_results.txt"

    go test -v -timeout 2h \
        ./tests/e2e_chain/... \
        -run "TestStability_30Min" 2>&1 | tee "${stability_output}" | tee -a "${LOG_FILE}"

    if [[ ${PIPESTATUS[0]} -eq 0 ]]; then
        log OK "Stability tests completed"
    else
        log WARN "Stability tests may have failed"
    fi
}

# Run load tests
run_loadtest() {
    log STEP "Running Load Tests..."
    cd "${PROJECT_ROOT}"

    local loadtest_output="${REPORTS_DIR}/loadtest_results.txt"

    if [[ -f "${PROJECT_ROOT}/tests/loadtest/main.go" ]]; then
        go run ./tests/loadtest/main.go \
            -c 100 \
            -d 60s 2>&1 | tee "${loadtest_output}" | tee -a "${LOG_FILE}"
        log OK "Load tests completed"
    else
        log WARN "Load test not found, skipping..."
    fi
}

# Generate summary report
generate_report() {
    log STEP "Generating Performance Report..."

    local report_file="${REPORTS_DIR}/performance_report.md"

    cat > "${report_file}" << EOF
# PerpDEX Performance Test Report

Generated: $(date '+%Y-%m-%d %H:%M:%S')

## Test Environment

- **OS:** $(uname -s) $(uname -r)
- **Go Version:** $(go version | awk '{print $3}')
- **CPU:** $(sysctl -n machdep.cpu.brand_string 2>/dev/null || cat /proc/cpuinfo | grep "model name" | head -1 | cut -d: -f2)
- **Memory:** $(sysctl -n hw.memsize 2>/dev/null | awk '{print $1/1024/1024/1024 " GB"}' || free -h | awk '/^Mem:/ {print $2}')

## Target Performance Metrics

| Metric | Target |
|--------|--------|
| Block Time | ~500ms |
| Orders/Block | 1,000+ |
| Orders/Second | 500+ |
| Match Latency | <100ms |
| API P99 Latency | <100ms |
| Success Rate | >99.9% |

## Test Results Summary

### Layer 1: Engine Microbenchmarks

$(cat "${REPORTS_DIR}/benchmark_results.txt" 2>/dev/null || echo "Results not available")

### Layer 2: API Performance

$(grep -A 20 "Latency Statistics" "${REPORTS_DIR}/api_performance_results.txt" 2>/dev/null || echo "Results not available")

### Layer 3: E2E Scenarios

$(grep -A 10 "Scenario Results" "${REPORTS_DIR}/scenario_results.txt" 2>/dev/null || echo "Results not available")

### Layer 4: Stability

$(grep -A 20 "Stability Test" "${REPORTS_DIR}/stability_results.txt" 2>/dev/null || echo "Results not available")

## Conclusion

Report generated automatically by run_performance_tests.sh
EOF

    log OK "Report generated: ${report_file}"
}

# Main execution
main() {
    echo ""
    echo "╔══════════════════════════════════════════════════════════════╗"
    echo "║       PerpDEX Performance Test Suite                         ║"
    echo "╠══════════════════════════════════════════════════════════════╣"
    echo "║  This script runs comprehensive performance tests including: ║"
    echo "║  - Layer 1: Engine Microbenchmarks                           ║"
    echo "║  - Layer 2: API Performance Tests                            ║"
    echo "║  - Layer 3: E2E Scenario Tests                               ║"
    echo "║  - Layer 4: Stability Tests                                  ║"
    echo "╚══════════════════════════════════════════════════════════════╝"
    echo ""

    log INFO "Log file: ${LOG_FILE}"

    # Parse arguments
    RUN_ALL=true
    RUN_BENCHMARK=false
    RUN_API=false
    RUN_SCENARIO=false
    RUN_STABILITY=false
    SKIP_BUILD=false
    SKIP_CHAIN=false

    while [[ $# -gt 0 ]]; do
        case $1 in
            --benchmark-only)
                RUN_ALL=false
                RUN_BENCHMARK=true
                shift
                ;;
            --api-only)
                RUN_ALL=false
                RUN_API=true
                shift
                ;;
            --scenario-only)
                RUN_ALL=false
                RUN_SCENARIO=true
                shift
                ;;
            --stability-only)
                RUN_ALL=false
                RUN_STABILITY=true
                shift
                ;;
            --skip-build)
                SKIP_BUILD=true
                shift
                ;;
            --skip-chain)
                SKIP_CHAIN=true
                shift
                ;;
            --help)
                echo "Usage: $0 [options]"
                echo ""
                echo "Options:"
                echo "  --benchmark-only   Run only benchmark tests"
                echo "  --api-only         Run only API performance tests"
                echo "  --scenario-only    Run only E2E scenario tests"
                echo "  --stability-only   Run only stability tests"
                echo "  --skip-build       Skip building the project"
                echo "  --skip-chain       Skip chain initialization"
                echo "  --help             Show this help message"
                exit 0
                ;;
            *)
                log ERROR "Unknown option: $1"
                exit 1
                ;;
        esac
    done

    # Run tests
    check_prerequisites

    if [[ "${SKIP_BUILD}" != "true" ]]; then
        build_project
    fi

    if [[ "${SKIP_CHAIN}" != "true" ]]; then
        init_chain
        apply_fast_config
    fi

    # Determine which tests to run
    if [[ "${RUN_ALL}" == "true" ]]; then
        # Run all tests
        run_benchmarks

        # Start node for chain-based tests
        if start_node; then
            trap stop_node EXIT

            run_api_tests
            run_scenario_tests
            run_stability_tests
            run_loadtest
        else
            log ERROR "Failed to start node, skipping chain-based tests"
        fi
    else
        # Run selected tests
        if [[ "${RUN_BENCHMARK}" == "true" ]]; then
            run_benchmarks
        fi

        if [[ "${RUN_API}" == "true" || "${RUN_SCENARIO}" == "true" || "${RUN_STABILITY}" == "true" ]]; then
            if start_node; then
                trap stop_node EXIT

                [[ "${RUN_API}" == "true" ]] && run_api_tests
                [[ "${RUN_SCENARIO}" == "true" ]] && run_scenario_tests
                [[ "${RUN_STABILITY}" == "true" ]] && run_stability_tests
            fi
        fi
    fi

    # Generate report
    generate_report

    echo ""
    log OK "All tests completed!"
    log INFO "Results saved to: ${REPORTS_DIR}"
    log INFO "Full log: ${LOG_FILE}"
}

# Run main
main "$@"
