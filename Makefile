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
