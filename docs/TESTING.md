# PerpDEX Testing Documentation

## Overview

This document provides comprehensive testing guidelines and results for the PerpDEX perpetual futures DEX built on Cosmos SDK.

---

## Test Categories

### 1. Unit Tests

Location: `x/*/keeper/*_test.go`

```bash
# Run all unit tests
go test -v ./...

# Run specific module tests
go test -v ./x/orderbook/keeper/...
go test -v ./x/perpetual/keeper/...
go test -v ./x/clearinghouse/keeper/...
```

### 2. Integration Tests

Location: `tests/e2e/`

```bash
# Run all E2E tests
go test -v ./tests/e2e/ -timeout 300s

# Run specific test categories
go test -v -run "TestPlaceOrderAPI" ./tests/e2e/
go test -v -run "TestOrderLifecycle" ./tests/e2e/
go test -v -run "TestHighThroughputOrders" ./tests/e2e/
```

### 3. Benchmark Tests

Location: `x/orderbook/keeper/benchmark_*.go`

```bash
# Run all benchmarks
go test -bench="." -benchmem ./x/orderbook/keeper/

# Run specific benchmark categories
go test -bench="BenchmarkAddOrder" -benchmem ./x/orderbook/keeper/
go test -bench="BenchmarkGetBest" -benchmem ./x/orderbook/keeper/
go test -bench="BenchmarkMixedOps" -benchmem ./x/orderbook/keeper/
```

### 4. Stress Tests

Location: `x/orderbook/keeper/e2e_stress_test.go`

```bash
# Run stress tests
go test -v -run "TestE2E" ./x/orderbook/keeper/ -timeout 600s

# Run specific stress test
go test -v -run "TestE2EStressAllImplementations" ./x/orderbook/keeper/
go test -v -run "TestE2EConcurrentStress" ./x/orderbook/keeper/
go test -v -run "TestE2EHighReadRatio" ./x/orderbook/keeper/
```

---

## Test Coverage

### API Tests

| Test | Description | Status |
|------|-------------|--------|
| `TestPlaceOrderAPI` | Tests order placement via Keeper | ✅ PASS |
| `TestCancelOrderAPI` | Tests order cancellation | ✅ PASS |
| `TestQueryOrderBookAPI` | Tests order book queries | ✅ PASS |
| `TestQueryTradesAPI` | Tests trade history queries | ✅ PASS |

### Order Lifecycle Tests

| Test | Description | Status |
|------|-------------|--------|
| `TestOrderLifecycle` | Full order lifecycle (place→match→cancel) | ✅ PASS |
| `TestOrderMatchingPriority` | Price-time priority verification | ✅ PASS |

### Data Structure Tests

| Test | Description | Status |
|------|-------------|--------|
| `TestAllImplementationsCorrectness` | Correctness verification for all implementations | ✅ PASS |
| `TestOrderBookInterface` | Interface compliance test | ✅ PASS |
| `TestAllDataStructures` | Performance comparison | ✅ PASS |

### Stress Tests

| Test | Description | Status |
|------|-------------|--------|
| `TestHighThroughputOrders` | 5000+ orders throughput | ✅ PASS |
| `TestConcurrentOrders` | Multi-threaded access | ✅ PASS |
| `TestE2EStressAllImplementations` | 50K orders stress test | ✅ PASS |
| `TestEndBlockerPerformance` | EndBlocker performance | ✅ PASS |

---

## Benchmark Results

### Platform Information

- **OS**: macOS (Darwin)
- **Architecture**: ARM64 (Apple M4 Pro)
- **CPUs**: 14 cores
- **Go Version**: 1.23+

### Order Book Data Structures

| Implementation | Add Order | Remove Order | Get Best | Get Top 10 | Throughput |
|----------------|-----------|--------------|----------|------------|------------|
| **B+ Tree** | 543 ns/op | 414 ns/op | 6 ns/op | 106 ns/op | 4.3M ops/s |
| **Skip List** | 814 ns/op | 674 ns/op | 4 ns/op | 30 ns/op | 2.6M ops/s |
| **HashMap** | 924 ns/op | 1,261 ns/op | 241 ns/op | 727μs/op | 1.4M ops/s |
| **ART** | 1,047 ns/op | 855 ns/op | 3.5ms/op | 1.8ms/op | 2.9K ops/s |

### Latency Percentiles (50K orders)

| Implementation | P50 | P95 | P99 | Max |
|----------------|-----|-----|-----|-----|
| **B+ Tree** | 208 ns | 333 ns | 542 ns | 6.6 μs |
| **Skip List** | 417 ns | 625 ns | 1.7 μs | 13 μs |
| **HashMap** | 333 ns | 35 μs | 42 μs | 236 μs |
| **ART** | 792 ns | 64 μs | 70 μs | 208 μs |

### Memory Efficiency

| Implementation | Memory Alloc | Total Alloc | GC Pauses |
|----------------|--------------|-------------|-----------|
| **B+ Tree** | 6.1 MB | 6.1 MB | 0 |
| **Skip List** | 9.2 MB | 9.2 MB | 0 |
| **HashMap** | 20.0 MB | 188.6 MB | 8 |
| **ART** | 13.5 MB | 93.1 MB | 4 |

---

## Performance Analysis

### Recommendation: B+ Tree

The B+ Tree implementation is recommended for production due to:

1. **Highest Throughput**: 4.3M ops/sec (1.6x faster than Skip List)
2. **Lowest Latency**: P99 at 542ns
3. **Best Memory Efficiency**: Minimal allocations, zero GC pauses
4. **Balanced Performance**: Good at both writes and reads

### Use Case Analysis

| Use Case | Recommended | Reason |
|----------|-------------|--------|
| High Throughput | B+ Tree | Best overall performance |
| Read-Heavy | Skip List | Fastest GetBest (4 ns) |
| Memory Constrained | B+ Tree | Lowest memory footprint |
| Simple Implementation | HashMap | Conceptually simplest |

---

## Running Tests

### Quick Start

```bash
# Clone repository
git clone https://github.com/openalpha/perp-dex.git
cd perp-dex

# Install dependencies
go mod tidy

# Run all tests
make test

# Run benchmarks
go test -bench="." -benchmem ./x/orderbook/keeper/
```

### CI/CD Integration

```yaml
# .github/workflows/test.yml
name: Tests
on: [push, pull_request]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.23'
      - run: go test -v ./...
      - run: go test -bench="." -benchmem ./x/orderbook/keeper/
```

### Test Configuration

Environment variables:

```bash
# Parallel test configuration
PARALLEL_WORKERS=8        # Number of parallel workers
PARALLEL_BATCH_SIZE=100   # Orders per batch
PARALLEL_ENABLED=true     # Enable parallel matching

# Test timeouts
TEST_TIMEOUT=300s         # Default test timeout
STRESS_TEST_ORDERS=50000  # Orders for stress tests
```

---

## Troubleshooting

### Common Issues

1. **Test Timeout**
   ```bash
   # Increase timeout
   go test -v ./tests/e2e/ -timeout 600s
   ```

2. **Memory Issues**
   ```bash
   # Run with memory limit
   GOGC=50 go test -v ./...
   ```

3. **Flaky Tests**
   ```bash
   # Run test multiple times
   go test -v -count=3 ./tests/e2e/
   ```

### Debug Mode

```bash
# Enable verbose logging
go test -v -run "TestPlaceOrderAPI" ./tests/e2e/ 2>&1 | tee test.log

# Enable race detection
go test -race ./...
```

---

## Test Files Reference

| File | Description |
|------|-------------|
| `tests/e2e/orderbook_e2e_test.go` | E2E integration tests |
| `x/orderbook/keeper/benchmark_test.go` | Unit benchmarks |
| `x/orderbook/keeper/benchmark_comparison_test.go` | Data structure comparison |
| `x/orderbook/keeper/e2e_stress_test.go` | Stress tests |
| `x/orderbook/keeper/parallel_test.go` | Parallel matching tests |

---

## Contributing

When adding new tests:

1. Follow existing naming conventions
2. Add appropriate test documentation
3. Include benchmark tests for performance-critical code
4. Update this document with new test coverage

---

**Last Updated**: 2026-01-19
