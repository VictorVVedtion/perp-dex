# Real Chain E2E Tests

This directory contains **real chain** end-to-end tests that run against an actual CometBFT chain, not mock/memory implementations.

## Why Real Chain Tests?

| Test Type | Storage | Use Case |
|-----------|---------|----------|
| `tests/e2e_real/` | In-memory API | Fast API testing, development |
| `tests/e2e_chain/` | **Real CometBFT chain** | Production verification, consensus testing |

Real chain tests verify:
- Transaction signing and broadcast
- Consensus finality
- State persistence
- Multi-block workflows
- Gas estimation and fees

## Quick Start

### Run All Tests
```bash
./tests/e2e_chain/run_chain_e2e.sh
```

### Run Specific Tests
```bash
# Only RiverPool tests
./tests/e2e_chain/run_chain_e2e.sh --filter "TestSuite_RiverPool"

# Only connectivity tests
./tests/e2e_chain/run_chain_e2e.sh --filter "TestChain_Connectivity"
```

### Options
```bash
./tests/e2e_chain/run_chain_e2e.sh --help

Options:
  --skip-build      Skip building the binary
  --skip-init       Skip chain initialization (use existing)
  --no-auto-start   Don't auto-start chain (assume already running)
  --cleanup         Clean up chain data after tests
  --verbose, -v     Enable verbose output
  --filter, -f      Filter tests by pattern
```

## Manual Testing

### 1. Build
```bash
make build
```

### 2. Initialize Chain
```bash
./build/perpdexd init test-node --chain-id perpdex-test-1 --home .perpdex-test
./build/perpdexd keys add validator --keyring-backend test --home .perpdex-test
# ... (see run_chain_e2e.sh for full setup)
```

### 3. Start Chain
```bash
./build/perpdexd start --home .perpdex-test --minimum-gas-prices "0usdc"
```

### 4. Run Tests
```bash
export PERPDEX_AUTO_START=false
go test -v ./tests/e2e_chain/... -count=1
```

## Test Structure

```
tests/e2e_chain/
├── framework/               # Test framework
│   ├── config.go           # Configuration
│   ├── chain_manager.go    # Chain lifecycle (start/stop/reset)
│   └── suite.go            # Test suite and client
├── riverpool_chain_test.go # RiverPool real chain tests
├── msg_server_test.go      # MsgServer tests
├── engine_direct_test.go   # Direct engine tests
├── chain_client.go         # Legacy chain client
└── run_chain_e2e.sh        # Test runner script
```

## Framework Usage

### Basic Test Suite
```go
func TestSuite_MyFeature(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping chain E2E tests in short mode")
    }
    suite.Run(t, new(MyFeatureTestSuite))
}

type MyFeatureTestSuite struct {
    suite.Suite
    fw *framework.ChainTestSuite
}

func (s *MyFeatureTestSuite) SetupSuite() {
    s.fw = framework.NewChainTestSuite(s.T())
    if err := s.fw.Setup(); err != nil {
        s.T().Skipf("Failed to setup: %v", err)
    }
}

func (s *MyFeatureTestSuite) TearDownSuite() {
    s.fw.Teardown()
}

func (s *MyFeatureTestSuite) TestMyFeature() {
    ctx := s.fw.Context()

    // Submit transaction
    result, err := s.fw.Client.PlaceOrder(ctx, "trader1", "BTC-USDC", "buy", "limit", "50000", "0.1")
    require.NoError(s.T(), err)
    require.True(s.T(), result.Success)

    // Wait for confirmation
    s.fw.WaitForBlocks(2)

    // Verify state
    orderBook, err := s.fw.Client.QueryOrderBook(ctx, "BTC-USDC")
    require.NoError(s.T(), err)
    // ... assertions
}
```

### Available Client Methods
```go
// Trading
client.PlaceOrder(ctx, trader, market, side, orderType, price, quantity)
client.CancelOrder(ctx, trader, orderID)

// RiverPool
client.DepositToRiverPool(ctx, depositor, poolID, amount)
client.RequestWithdrawal(ctx, withdrawer, poolID, shares)
client.CreateCommunityPool(ctx, owner, name, strategy, params)

// Queries
client.QueryOrderBook(ctx, market)
client.QueryPool(ctx, poolID)
client.QueryUserDeposit(ctx, poolID, user)
client.QueryBalance(ctx, address, denom)

// Utilities
client.SendTokens(ctx, from, to, amount)
client.GetAccountAddress(ctx, keyName)
```

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `PERPDEX_RPC_URL` | `http://localhost:26657` | CometBFT RPC endpoint |
| `PERPDEX_API_URL` | `http://localhost:1317` | REST API endpoint |
| `PERPDEX_CHAIN_ID` | `perpdex-test-1` | Chain ID |
| `PERPDEX_AUTO_START` | `true` | Auto-start chain if not running |
| `PERPDEX_CLEANUP` | `false` | Clean up data after tests |
| `PERPDEX_VERBOSE` | `false` | Verbose logging |

## CI/CD Integration

The GitHub Actions workflow (`.github/workflows/e2e-chain-tests.yml`) runs automatically on:
- Push to `main` or `develop`
- Pull requests to `main` or `develop`
- Manual trigger via `workflow_dispatch`

### Manual Trigger
```bash
gh workflow run e2e-chain-tests.yml \
  -f test_filter="TestSuite_RiverPool" \
  -f verbose="true"
```

## Troubleshooting

### Chain Won't Start
```bash
# Check logs
cat .perpdex-test/chain.log | tail -50

# Reset chain
rm -rf .perpdex-test
./tests/e2e_chain/run_chain_e2e.sh
```

### Tests Skip with "Chain not running"
```bash
# Ensure chain is running
curl http://localhost:26657/status

# Or let the test framework start it
export PERPDEX_AUTO_START=true
```

### Transaction Fails with "key not found"
```bash
# Ensure test accounts exist
./build/perpdexd keys list --keyring-backend test --home .perpdex-test
```

### "Module not found" Errors
Some modules (like `x/riverpool`) may not be implemented on chain yet. Tests will skip gracefully with informative messages.

## Adding New Tests

1. Create a new test file: `myfeature_chain_test.go`
2. Use the framework:
   ```go
   import "github.com/openalpha/perp-dex/tests/e2e_chain/framework"
   ```
3. Implement `SetupSuite`, `TearDownSuite`, and test methods
4. Add new client methods to `framework/suite.go` if needed

## Performance Considerations

- Block time: ~500ms (configurable)
- Transaction confirmation: ~1-2 blocks
- Recommended timeout: 30-60s per test

For load testing, see `TestRiverPool_Throughput` as an example.
