# PerpDEX Real E2E Test Suite

This directory contains **real end-to-end tests** that verify the complete system behavior from API requests through order execution to WebSocket notifications.

## Overview

Unlike the previous "E2E" tests that only tested internal components, these tests:

- ✅ Start/connect to a real API server
- ✅ Make real HTTP requests to REST endpoints
- ✅ Connect to real WebSocket endpoints
- ✅ Test complete trading flows (order → match → position)
- ✅ Verify WebSocket message delivery
- ✅ Test concurrent user scenarios
- ✅ Measure actual latency and throughput

## Test Categories

### 1. Trading Flow Tests (`trading_flow_test.go`)
- Complete order lifecycle
- Various order types (limit, market)
- Order matching scenarios
- Position creation verification
- Market data endpoints

### 2. WebSocket Tests (`websocket_test.go`)
- Connection establishment
- Channel subscription (ticker, depth, trades)
- Private notifications (orders, positions)
- Message delivery under load
- Reconnection handling

### 3. Liquidation Tests (`liquidation_test.go`)
- Position health monitoring
- Three-tier liquidation mechanism
- Insurance fund interactions
- ADL (Auto-Deleveraging) scenarios

### 4. Concurrent Tests (`concurrent_test.go`)
- Multi-user order placement
- Concurrent order matching
- Race condition detection
- System stability under sustained load

## Running the Tests

### Prerequisites

1. API server must be running:
```bash
cd /home/user/perp-dex
go run ./cmd/api/main.go --port 8080 --mock
```

### Quick Run

```bash
cd /home/user/perp-dex/tests/e2e_real

# Run all tests
go test -v ./...

# Run specific test file
go test -v -run TestCompleteTradingFlow

# Run with short mode (skip long tests)
go test -v -short ./...
```

### Using the Test Runner Script

```bash
chmod +x run_e2e.sh

# Auto-start server and run tests
./run_e2e.sh --start-server --stop-server

# Run quick tests only
./run_e2e.sh --quick

# Run full suite with stability tests
./run_e2e.sh --full --report

# Generate report
./run_e2e.sh --report
```

## Test Configuration

Default configuration can be modified in `framework.go`:

```go
const (
    DefaultAPIURL     = "http://localhost:8080"
    DefaultWSURL      = "ws://localhost:8080/ws"
    DefaultTimeout    = 30 * time.Second
)
```

## Expected Results

### Health Check
```
=== RUN   TestHealthCheck
    trading_flow_test.go:XX: Server health: map[mock_mode:true status:healthy timestamp:1705...]
--- PASS: TestHealthCheck (0.01s)
```

### Trading Flow
```
=== RUN   TestCompleteTradingFlow
    trading_flow_test.go:XX: Maker order placed: order-xxx at price 50000.00
    trading_flow_test.go:XX: Taker order placed: order-yyy, status: filled, filled: 0.1
--- PASS: TestCompleteTradingFlow (0.15s)
```

### Concurrent Tests
```
═══════════════════════════════════════════════════════════════
        CONCURRENT ORDER PLACEMENT TEST RESULTS
═══════════════════════════════════════════════════════════════
Users:              10
Orders per user:    50
Duration:           2.5s
Total orders:       500
Successful:         498 (99.60%)
Failed:             2 (0.40%)
Throughput:         200.00 orders/sec
═══════════════════════════════════════════════════════════════
```

## Comparison: Old vs New E2E Tests

| Aspect | Old "E2E" Tests | New Real E2E Tests |
|--------|-----------------|-------------------|
| Server | Mock in-memory | Real HTTP server |
| Database | Memory DB | Real/Mock service |
| HTTP | Not tested | Full REST API |
| WebSocket | Not tested | Full WS coverage |
| Concurrency | Limited | Multi-user load |
| Latency | Internal only | Real network |

## File Structure

```
tests/e2e_real/
├── framework.go          # Test infrastructure
├── trading_flow_test.go  # Trading lifecycle tests
├── websocket_test.go     # WebSocket integration tests
├── liquidation_test.go   # Liquidation flow tests
├── concurrent_test.go    # Multi-user concurrent tests
├── run_e2e.sh           # Automated test runner
└── README.md            # This file
```

## Extending Tests

To add new E2E tests:

1. Create a new `*_test.go` file
2. Use the `E2ETestSuite` framework:

```go
func TestMyNewFeature(t *testing.T) {
    suite := NewE2ETestSuite(t, nil)

    err := suite.WaitForServer(10 * time.Second)
    if err != nil {
        t.Skipf("Server not available: %v", err)
    }

    // Your test code here
    user := suite.NewTestUser("perpdex1testuser001")
    order, err := user.PlaceOrder(&PlaceOrderRequest{...})
    // ...
}
```

## Troubleshooting

### "Server not available"
- Ensure API server is running on port 8080
- Check firewall settings
- Verify with: `curl http://localhost:8080/health`

### "WebSocket connection failed"
- Check WebSocket endpoint is enabled
- Verify CORS settings
- Check for proxy issues

### "Test timeout"
- Increase timeout in test config
- Check server response times
- Monitor server logs

## Integration with CI/CD

```yaml
# Example GitHub Actions workflow
e2e-tests:
  runs-on: ubuntu-latest
  steps:
    - uses: actions/checkout@v3
    - uses: actions/setup-go@v4
      with:
        go-version: '1.22'

    - name: Start API Server
      run: |
        go run ./cmd/api/main.go --mock &
        sleep 5

    - name: Run E2E Tests
      run: |
        cd tests/e2e_real
        go test -v -short ./...
```
