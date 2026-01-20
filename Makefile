.PHONY: all build install test lint proto clean

BINARY_NAME := perpdexd
BUILD_DIR := ./build

all: proto build

build:
	@echo "Building $(BINARY_NAME)..."
	go build -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/perpdexd

install:
	@echo "Installing $(BINARY_NAME)..."
	go install ./cmd/perpdexd

test:
	@echo "Running tests..."
	go test -v ./...

lint:
	@echo "Running linter..."
	golangci-lint run

proto:
	@echo "Generating protobuf files..."
	@./scripts/protocgen.sh

clean:
	@echo "Cleaning..."
	rm -rf $(BUILD_DIR)

init-chain:
	@echo "Initializing chain..."
	./scripts/init-chain.sh

start:
	@echo "Starting node..."
	./scripts/start-node.sh

# Development helpers
dev-setup:
	@echo "Setting up development environment..."
	go mod tidy
	go mod download

run-local: build init-chain start

# ============ Real Engine Mode ============

# Start API server with real orderbook engine (no mock data)
api-real:
	@echo "Starting API server with REAL orderbook engine..."
	go run ./cmd/api --real

# Start API server with mock data (default)
api-mock:
	@echo "Starting API server with mock data..."
	go run ./cmd/api --mock

# ============ Performance Testing ============

# Run engine benchmarks
benchmark:
	@echo "Running orderbook engine benchmarks..."
	go test -bench=. -benchmem ./tests/benchmark/...

# Run specific benchmark
benchmark-10k:
	@echo "Running 10K order matching benchmark..."
	go test -bench=Match10K -benchmem ./tests/benchmark/...

# Run stress test
stress-test:
	@echo "Running stress test..."
	go test -v -run TestStress10K ./tests/benchmark/...

# Run all performance tests
perf-test: benchmark stress-test
	@echo "All performance tests completed"

# ============ Load Testing ============

# Run load test against API (requires API server running)
loadtest:
	@echo "Running load test (ensure API server is running)..."
	go run ./tests/loadtest/main.go -c 50 -d 30s

loadtest-real:
	@echo "Running load test against real engine API..."
	go run ./tests/loadtest/main.go -c 50 -d 60s -realistic

# ============ Help ============

help:
	@echo "PerpDEX Makefile Commands:"
	@echo ""
	@echo "Build & Run:"
	@echo "  make build          - Build the binary"
	@echo "  make api-real       - Start API with REAL orderbook engine"
	@echo "  make api-mock       - Start API with mock data"
	@echo ""
	@echo "Testing:"
	@echo "  make test           - Run all tests"
	@echo "  make benchmark      - Run engine benchmarks"
	@echo "  make benchmark-10k  - Run 10K order benchmark"
	@echo "  make stress-test    - Run stress test"
	@echo "  make loadtest       - Run HTTP load test"
	@echo ""
	@echo "Development:"
	@echo "  make dev-setup      - Setup development environment"
	@echo "  make clean          - Clean build artifacts"
