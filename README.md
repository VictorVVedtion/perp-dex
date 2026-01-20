# PerpDEX - High-Performance Decentralized Perpetual Exchange

<div align="center">

**A production-grade perpetual futures DEX built on Cosmos SDK**

[![Go Version](https://img.shields.io/badge/Go-1.22+-00ADD8?style=flat&logo=go)](https://go.dev/)
[![Cosmos SDK](https://img.shields.io/badge/Cosmos%20SDK-0.50.11-blue?style=flat)](https://cosmos.network/)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)
[![Tests](https://img.shields.io/badge/Tests-100%25%20E2E%20Pass-success)](COMPREHENSIVE_TEST_REPORT.md)
[![API Coverage](https://img.shields.io/badge/API%20Coverage-19%2F20-blue)](tests/e2e_comprehensive/)

</div>

---

## Performance Highlights

| Metric | V2 Engine | V1 Engine | Improvement |
|--------|-----------|-----------|-------------|
| **10K Orders Matching** | 5.88 ms | 1,965 ms | **334x faster** |
| **Add Order** | 780 ns | 47,709 ns | **61x faster** |
| **Remove Order** | 872 ns | 23,226 ns | **27x faster** |
| **Memory per Match** | 8.2 MB | 1,247 MB | **152x less** |

*Benchmarked on Apple M4 Pro, darwin/arm64*

---

## Table of Contents

- [Overview](#overview)
- [Architecture](#architecture)
- [Features](#features)
- [Quick Start](#quick-start)
- [Performance](#performance)
- [API Reference](#api-reference)
- [CLI Commands](#cli-commands)
- [Configuration](#configuration)
- [Testing](#testing)
- [Deployment](#deployment)
- [Project Structure](#project-structure)
- [Contributing](#contributing)
- [License](#license)

---

## Overview

PerpDEX is a high-performance decentralized perpetual futures exchange built on the Cosmos SDK. It provides institutional-grade trading infrastructure with sub-millisecond order execution, advanced risk management, and real-time funding rate calculations.

### Key Metrics

- **Throughput**: 170,000+ orders/second
- **Latency**: < 1ms order placement
- **Memory Efficiency**: 152x improvement over baseline
- **Test Coverage**: 50 unit tests, 100% passing

---

## Architecture

```
┌─────────────────────────────────────────────────────────────────────────┐
│                              Frontend (Next.js 14)                       │
│  ┌───────────┐  ┌───────────┐  ┌───────────┐  ┌───────────────────────┐ │
│  │   Trade   │  │  Account  │  │ Positions │  │   WebSocket Client    │ │
│  │   Page    │  │   Page    │  │   Page    │  │  (Real-time Updates)  │ │
│  └─────┬─────┘  └─────┬─────┘  └─────┬─────┘  └───────────┬───────────┘ │
└────────┼──────────────┼──────────────┼────────────────────┼─────────────┘
         │              │              │                    │
         ▼              ▼              ▼                    ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                          API Gateway (REST + WebSocket)                  │
│  ┌─────────────────────────────────────────────────────────────────────┐│
│  │  REST API: /api/v1/*           WebSocket: /ws (real-time streams)   ││
│  └─────────────────────────────────────────────────────────────────────┘│
└──────────────────────────────────┬──────────────────────────────────────┘
                                   │
                                   ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                       Cosmos SDK Application Layer                       │
│  ┌───────────────┐  ┌───────────────┐  ┌───────────────────────────────┐│
│  │   Orderbook   │  │   Perpetual   │  │       Clearinghouse          ││
│  │    Module     │  │    Module     │  │          Module              ││
│  │               │  │               │  │                               ││
│  │ • SkipList    │  │ • Markets     │  │ • Liquidation Engine V2      ││
│  │   OrderBook   │  │ • Positions   │  │ • Insurance Fund             ││
│  │ • Parallel    │  │ • Funding     │  │ • ADL Mechanism              ││
│  │   Matching    │  │   Rate        │  │ • 3-Tier Liquidation         ││
│  │ • OCO Orders  │  │ • K-Lines     │  │                               ││
│  │ • TWAP        │  │               │  │                               ││
│  │ • Trailing    │  │               │  │                               ││
│  │   Stop        │  │               │  │                               ││
│  └───────┬───────┘  └───────┬───────┘  └───────────────┬───────────────┘│
│          │                  │                          │                 │
│          └──────────────────┼──────────────────────────┘                 │
│                             │                                            │
│                    ┌────────▼────────┐                                   │
│                    │   EndBlocker    │                                   │
│                    │  (Per-block     │                                   │
│                    │   processing)   │                                   │
│                    └────────┬────────┘                                   │
└─────────────────────────────┼───────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                         CometBFT Consensus Layer                         │
│  ┌─────────────────────────────────────────────────────────────────────┐│
│  │  Block Production → Validation → Finality (~2s block time)          ││
│  └─────────────────────────────────────────────────────────────────────┘│
└─────────────────────────────────────────────────────────────────────────┘
```

---

## Features

### Trading Engine

| Feature | Description |
|---------|-------------|
| **SkipList OrderBook** | O(log n) insert/delete with price-time priority |
| **Parallel Matching** | Multi-core optimized matching engine |
| **OCO Orders** | One-Cancels-Other for automated risk management |
| **TWAP Orders** | Time-Weighted Average Price execution |
| **Trailing Stop** | Dynamic stop-loss that follows price movement |
| **Conditional Orders** | Trigger-based order execution |

### Risk Management

| Feature | Description |
|---------|-------------|
| **3-Tier Liquidation** | Gradual liquidation: 25% → 50% → 100% |
| **Insurance Fund** | Socialized loss protection |
| **ADL (Auto-Deleveraging)** | Backstop mechanism when insurance depleted |
| **Position Health V2** | Real-time margin ratio monitoring |
| **Cooldown Mechanism** | Anti-manipulation protection |

### Funding Rate System

| Feature | Description |
|---------|-------------|
| **Dynamic Funding** | 8-hour funding intervals |
| **Rate Clamping** | ±0.05% max funding rate |
| **TWAP Premium** | Time-weighted premium index |
| **Auto Settlement** | Block-level funding distribution |

### Real-Time System

| Feature | Description |
|---------|-------------|
| **WebSocket Streams** | Live orderbook, trades, positions |
| **K-Line Data** | OHLCV candlestick aggregation |
| **Depth Updates** | Incremental orderbook snapshots |
| **Trade Notifications** | Instant fill notifications |

---

## Quick Start

### Prerequisites

- Go 1.22+
- Node.js 18+
- Make (optional)

### 1. Clone & Build

```bash
# Clone repository
git clone https://github.com/openalpha/perp-dex.git
cd perp-dex

# Build backend
go build -o ./build/perpdexd ./cmd/perpdexd

# Build frontend
cd frontend && npm install && npm run build
```

### 2. Initialize Chain

```bash
# Initialize chain with test accounts
./scripts/init-chain.sh

# This creates:
# - Chain ID: perpdex-1
# - Validator with 1,000,000,000,000 usdc
# - 3 test traders with 10,000,000,000 usdc each
```

### 3. Start Node

```bash
# Start the node
./build/perpdexd start --home ~/.perpdex --api.enable --minimum-gas-prices "0usdc"

# Expected output:
# INF committed state height=1
# INF EndBlocker performance matching_ms=0 liquidation_ms=0 funding_ms=0
```

### 4. Start Frontend

```bash
cd frontend
npm run dev
# Open http://localhost:3000
```

---

## Performance

### Benchmark Results

```
goos: darwin
goarch: arm64
cpu: Apple M4 Pro

BenchmarkNewMatching-14        204     5,880,207 ns/op    8,208,438 B/op   107,420 allocs/op
BenchmarkOldMatching-14          1 1,965,217,875 ns/op 1,247,621,120 B/op 32,546,405 allocs/op

BenchmarkNewAddOrder-14    1,477,113       780.4 ns/op       271 B/op         8 allocs/op
BenchmarkOldAddOrder-14       39,596    47,709 ns/op       236 B/op         8 allocs/op

BenchmarkNewRemoveOrder-14 1,432,076       871.8 ns/op       239 B/op         8 allocs/op
BenchmarkOldRemoveOrder-14   250,983    23,226 ns/op       109 B/op         4 allocs/op

BenchmarkNewGetBest-14   311,721,440       3.861 ns/op         0 B/op         0 allocs/op
BenchmarkOldGetBest-14 1,000,000,000       0.2542 ns/op        0 B/op         0 allocs/op
```

### Performance Summary

| Operation | V2 (New) | V1 (Old) | Speedup |
|-----------|----------|----------|---------|
| Match 10K Orders | 5.88 ms | 1,965 ms | **334x** |
| Add Order | 780 ns | 47,709 ns | **61x** |
| Remove Order | 872 ns | 23,226 ns | **27x** |
| Get Best Price | 3.86 ns | 0.25 ns | 0.07x* |
| Mixed Operations | 161 ms | 109 ms | 0.68x* |

*Note: GetBest is slower due to SkipList traversal vs direct access, but this is acceptable given the massive improvements in other operations.

### Throughput Analysis

- **Add Order**: 1,281,230 ops/sec
- **Remove Order**: 1,147,068 ops/sec
- **Combined Throughput**: ~2.4M operations/sec

---

## API Reference

### REST Endpoints

#### Markets

```
GET  /api/v1/markets              # List all markets
GET  /api/v1/markets/{id}         # Get market details
GET  /api/v1/markets/{id}/klines  # Get K-line data
```

#### Trading

```
POST /api/v1/orders               # Place order
GET  /api/v1/orders/{id}          # Get order status
DELETE /api/v1/orders/{id}        # Cancel order
GET  /api/v1/orders?address=...   # List user orders
```

#### Account

```
GET  /api/v1/account/{address}           # Get account info
GET  /api/v1/positions/{address}         # Get positions
GET  /api/v1/positions/{address}/{market} # Get specific position
```

### WebSocket Streams

```javascript
// Connect
const ws = new WebSocket('ws://localhost:26657/ws');

// Subscribe to orderbook
ws.send(JSON.stringify({
  type: 'subscribe',
  channel: 'orderbook',
  market: 'BTC-USDC'
}));

// Subscribe to trades
ws.send(JSON.stringify({
  type: 'subscribe',
  channel: 'trades',
  market: 'BTC-USDC'
}));

// Subscribe to positions
ws.send(JSON.stringify({
  type: 'subscribe',
  channel: 'positions',
  address: 'cosmos1...'
}));
```

---

## CLI Commands

### Chain Management

```bash
# Initialize new chain
perpdexd init <moniker> --chain-id perpdex-1

# Start node
perpdexd start --home ~/.perpdex

# Query node status
perpdexd status
```

### Key Management

```bash
# Create new key
perpdexd keys add <name>

# List keys
perpdexd keys list

# Export key
perpdexd keys export <name>
```

### Trading Commands

```bash
# Place limit order
perpdexd tx orderbook place-order \
  --market BTC-USDC \
  --side buy \
  --price 50000 \
  --size 1.0 \
  --from trader1

# Cancel order
perpdexd tx orderbook cancel-order \
  --order-id <order-id> \
  --from trader1

# Query orderbook
perpdexd query orderbook book BTC-USDC
```

### Position Commands

```bash
# Query positions
perpdexd query perpetual positions <address>

# Add margin
perpdexd tx perpetual add-margin \
  --market BTC-USDC \
  --amount 1000usdc \
  --from trader1
```

---

## Configuration

### Chain Configuration

```toml
# ~/.perpdex/config/config.toml

[consensus]
timeout_commit = "2s"

[mempool]
size = 10000
max_txs_bytes = 1073741824

[p2p]
max_num_inbound_peers = 40
max_num_outbound_peers = 10
```

### App Configuration

```toml
# ~/.perpdex/config/app.toml

[api]
enable = true
swagger = true
address = "tcp://0.0.0.0:1317"

[grpc]
enable = true
address = "0.0.0.0:9090"

[state-sync]
snapshot-interval = 1000
snapshot-keep-recent = 2
```

### Module Parameters

```yaml
# Orderbook Module
orderbook:
  max_orders_per_market: 100000
  matching_interval_blocks: 1
  parallel_workers: 8

# Perpetual Module
perpetual:
  funding_interval: 28800  # 8 hours
  max_funding_rate: 0.0005  # 0.05%
  maintenance_margin: 0.05  # 5%

# Clearinghouse Module
clearinghouse:
  liquidation_tier1_threshold: 0.0625  # 6.25%
  liquidation_tier2_threshold: 0.05    # 5%
  liquidation_tier3_threshold: 0.03    # 3%
  insurance_fund_fee: 0.0005           # 0.05%
```

---

## Testing

### Run All Tests

```bash
# Run all unit tests
go test -v ./...

# Run E2E integration tests
go test -v ./tests/e2e/ -timeout 300s

# Run with coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### Run Benchmarks

```bash
# Run all benchmarks
go test -bench=. -benchmem ./x/orderbook/keeper

# Run specific benchmark
go test -bench=BenchmarkNewMatching -benchmem ./x/orderbook/keeper

# Run data structure comparison
go test -bench="BenchmarkAddOrder|BenchmarkGetBest" -benchmem ./x/orderbook/keeper/
```

### Run Stress Tests

```bash
# Run E2E stress tests
go test -v -run "TestE2EStressAllImplementations" ./x/orderbook/keeper/ -timeout 600s

# Run concurrent stress tests
go test -v -run "TestE2EConcurrentStress" ./x/orderbook/keeper/

# Run high read ratio tests
go test -v -run "TestE2EHighReadRatio" ./x/orderbook/keeper/
```

### Test Categories

| Category | Tests | Description |
|----------|-------|-------------|
| Unit Tests | 50+ | Core functionality |
| E2E Tests | 15+ | API integration |
| Benchmark | 20+ | Performance |
| Stress Tests | 5 | High load scenarios |

### Order Book Data Structures Comparison

| Implementation | Throughput | P99 Latency | Memory | Status |
|----------------|------------|-------------|--------|--------|
| **B+ Tree** | 4.3M ops/s | 542 ns | 6.1 MB | **Recommended** |
| Skip List | 2.6M ops/s | 1.7 μs | 9.2 MB | Current Default |
| HashMap | 1.4M ops/s | 42 μs | 20 MB | Not Recommended |
| ART | 2.9K ops/s | 70 μs | 13.5 MB | Not Recommended |

> See [docs/TESTING.md](docs/TESTING.md) for detailed test documentation.
> See [docs/E2E_STRESS_TEST_REPORT.md](docs/E2E_STRESS_TEST_REPORT.md) for stress test results.

---

## Deployment

### Docker

```bash
# Build image
docker build -t perpdex:latest .

# Run container
docker run -d \
  --name perpdex-node \
  -p 26656:26656 \
  -p 26657:26657 \
  -p 1317:1317 \
  -p 9090:9090 \
  -v perpdex-data:/root/.perpdex \
  perpdex:latest

# View logs
docker logs -f perpdex-node
```

### Docker Compose

```yaml
version: '3.8'
services:
  perpdex:
    build: .
    ports:
      - "26656:26656"
      - "26657:26657"
      - "1317:1317"
      - "9090:9090"
    volumes:
      - perpdex-data:/root/.perpdex
    environment:
      - CHAIN_ID=perpdex-1
      - MONIKER=my-node

  frontend:
    build: ./frontend
    ports:
      - "3000:3000"
    depends_on:
      - perpdex

volumes:
  perpdex-data:
```

---

## Project Structure

```
perp-dex/
├── app/                    # Cosmos SDK application
│   ├── app.go             # Main application setup
│   └── encoding.go        # Codec configuration
├── cmd/
│   └── perpdexd/          # CLI binary
│       └── main.go
├── proto/                  # Protobuf definitions
│   └── perpdex/
│       ├── orderbook/
│       ├── perpetual/
│       └── clearinghouse/
├── x/                      # Cosmos modules
│   ├── orderbook/         # Order management & matching
│   │   ├── keeper/
│   │   │   ├── keeper.go
│   │   │   ├── msg_server.go
│   │   │   ├── matching_v2.go
│   │   │   ├── orderbook_skiplist.go
│   │   │   ├── parallel_matcher.go
│   │   │   ├── oco_order.go
│   │   │   ├── twap_order.go
│   │   │   └── trailing_stop.go
│   │   └── types/
│   ├── perpetual/         # Position & funding
│   │   ├── keeper/
│   │   │   ├── keeper.go
│   │   │   ├── funding.go
│   │   │   ├── market.go
│   │   │   └── position.go
│   │   └── types/
│   └── clearinghouse/     # Risk & liquidation
│       ├── keeper/
│       │   ├── keeper.go
│       │   ├── liquidation_v2.go
│       │   ├── insurance_fund.go
│       │   └── adl.go
│       └── types/
├── frontend/              # Next.js frontend
│   ├── src/
│   │   ├── app/          # App router pages
│   │   ├── components/   # React components
│   │   └── hooks/        # Custom hooks
│   └── package.json
├── scripts/
│   └── init-chain.sh     # Chain initialization
├── build/                 # Build artifacts
├── docs/                  # Documentation
└── README.md
```

---

## Margin Calculations

### Initial Margin
```
InitialMargin = Size × Price × 10%
```

### Maintenance Margin
```
MaintenanceMargin = Size × MarkPrice × 5%
```

### Liquidation Price
```
Long:  LiquidationPrice = EntryPrice × (1 - InitialMarginRatio + MaintenanceMarginRatio)
Short: LiquidationPrice = EntryPrice × (1 + InitialMarginRatio - MaintenanceMarginRatio)
```

### Unrealized PnL
```
Long:  PnL = Size × (MarkPrice - EntryPrice)
Short: PnL = Size × (EntryPrice - MarkPrice)
```

---

## Trading Parameters

| Parameter | Value |
|-----------|-------|
| Max Leverage | 10x |
| Initial Margin | 10% |
| Maintenance Margin | 5% |
| Taker Fee | 0.05% |
| Maker Fee | 0.02% |
| Tick Size | 0.01 |
| Lot Size | 0.0001 |
| Funding Interval | 8 hours |
| Max Funding Rate | ±0.05% |

---

## Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

### Code Style

- Go: Follow [Effective Go](https://golang.org/doc/effective_go)
- TypeScript: Use ESLint + Prettier
- Commits: Follow [Conventional Commits](https://www.conventionalcommits.org/)

---

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

---

## Acknowledgments

- [Cosmos SDK](https://cosmos.network/) - Blockchain framework
- [CometBFT](https://cometbft.com/) - Consensus engine
- [Hyperliquid](https://hyperliquid.xyz/) - Inspiration for perpetual exchange design
- [Lightweight Charts](https://tradingview.github.io/lightweight-charts/) - Trading charts
