# PerpDEX - Perpetual Decentralized Exchange

A Hyperliquid-inspired perpetual contract exchange built on **Cosmos SDK + CometBFT**.

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                   Frontend (Next.js + TailwindCSS)               │
│            Trade Page  │  Positions  │  Account                  │
└────────────────────────────┬────────────────────────────────────┘
                             │ gRPC-gateway / REST
┌────────────────────────────▼────────────────────────────────────┐
│                     Cosmos SDK Application                       │
├──────────────────┬──────────────────┬───────────────────────────┤
│   x/orderbook    │   x/perpetual    │    x/clearinghouse        │
│   Order Book +   │   Positions +    │    Liquidation            │
│   Matching       │   Margin         │    Engine                 │
├──────────────────┴──────────────────┴───────────────────────────┤
│                        x/bank (Asset Management)                 │
├─────────────────────────────────────────────────────────────────┤
│                       CometBFT (Consensus)                       │
└─────────────────────────────────────────────────────────────────┘
```

## Features

### MVP Scope
- **Single Market**: BTC-USDC perpetual contract
- **Fixed Leverage**: 10x
- **Order Types**: Limit and Market orders
- **Margin Mode**: Isolated margin only
- **Price Oracle**: Simulated random price movement

### Core Modules

| Module | Description |
|--------|-------------|
| `x/orderbook` | Order book management, Price-Time Priority matching engine |
| `x/perpetual` | Position management, margin calculations, account balances |
| `x/clearinghouse` | Liquidation monitoring and execution |

## Project Structure

```
perp-dex/
├── app/                      # Cosmos SDK app configuration
│   ├── app.go
│   └── encoding.go
├── cmd/perpdexd/            # Node binary
│   ├── main.go
│   └── cmd/
│       ├── root.go
│       └── init.go
├── proto/perpdex/           # Protobuf definitions
│   ├── orderbook/v1/
│   ├── perpetual/v1/
│   └── clearinghouse/v1/
├── x/                       # Custom modules
│   ├── orderbook/
│   │   ├── keeper/
│   │   │   ├── keeper.go
│   │   │   └── matching.go   # ⭐ Matching engine
│   │   └── types/
│   │       └── types.go      # ⭐ Order types
│   ├── perpetual/
│   │   ├── keeper/
│   │   │   ├── keeper.go
│   │   │   ├── margin.go     # ⭐ Margin checker
│   │   │   ├── position.go   # Position manager
│   │   │   └── oracle.go     # Price simulator
│   │   └── types/
│   └── clearinghouse/
│       ├── keeper/
│       │   ├── keeper.go
│       │   └── liquidation.go # ⭐ Liquidation engine
│       └── types/
├── frontend/                # Next.js frontend
│   ├── src/
│   │   ├── pages/
│   │   │   ├── index.tsx     # Trade page
│   │   │   ├── positions.tsx
│   │   │   └── account.tsx
│   │   ├── components/
│   │   │   ├── OrderBook.tsx
│   │   │   ├── TradeForm.tsx
│   │   │   └── PositionCard.tsx
│   │   └── stores/
│   │       └── tradingStore.ts
│   └── package.json
├── scripts/
│   ├── init-chain.sh
│   └── start-node.sh
├── go.mod
└── Makefile
```

## Quick Start

### Prerequisites

- Go 1.21+
- Node.js 18+
- Make

### 1. Build the Chain

```bash
cd perp-dex

# Build the binary
make build

# Or install globally
make install
```

### 2. Initialize the Chain

```bash
./scripts/init-chain.sh

# Or manually:
perpdexd init validator --chain-id perpdex-1
perpdexd keys add validator --keyring-backend test
perpdexd genesis add-genesis-account validator 1000000000000usdc
perpdexd genesis gentx validator 100000000stake --chain-id perpdex-1
perpdexd genesis collect-gentxs
```

### 3. Start the Node

```bash
./scripts/start-node.sh

# Or:
perpdexd start --api.enable --grpc.enable
```

### 4. Start the Frontend

```bash
cd frontend
npm install
npm run dev
```

Open http://localhost:3000

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

## API Endpoints

### REST (localhost:1317)

```
POST   /perpdex/orderbook/v1/place_order
POST   /perpdex/orderbook/v1/cancel_order
GET    /perpdex/orderbook/v1/orderbook/{market_id}
GET    /perpdex/perpetual/v1/account/{address}
GET    /perpdex/perpetual/v1/position/{trader}/{market_id}
GET    /perpdex/perpetual/v1/price/{market_id}
GET    /perpdex/clearinghouse/v1/health/{trader}/{market_id}
```

### CLI Commands

```bash
# Place an order
perpdexd tx orderbook place-order BTC-USDC buy limit 50000 0.1 --from trader1

# Cancel an order
perpdexd tx orderbook cancel-order order-1 --from trader1

# Query order book
perpdexd q orderbook orderbook BTC-USDC

# Query position
perpdexd q perpetual position $(perpdexd keys show trader1 -a) BTC-USDC

# Deposit margin
perpdexd tx perpetual deposit 10000usdc --from trader1

# Withdraw margin
perpdexd tx perpetual withdraw 5000usdc --from trader1
```

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
Long:  LiquidationPrice = EntryPrice × 0.95
Short: LiquidationPrice = EntryPrice × 1.05
```

### Unrealized PnL
```
Long:  PnL = Size × (MarkPrice - EntryPrice)
Short: PnL = Size × (EntryPrice - MarkPrice)
```

## Development

```bash
# Run tests
make test

# Lint code
make lint

# Generate protobuf
make proto

# Clean build
make clean
```

## Next Steps (Post-MVP)

1. **Funding Rate**: Implement 8-hour funding rate settlement
2. **Multiple Markets**: Support ETH-USDC, SOL-USDC
3. **Advanced Orders**: Stop-loss, take-profit, trailing stop
4. **Cross Margin**: Add cross-margin mode
5. **Real Oracle**: Integrate Chainlink/Band Protocol
6. **Insurance Fund**: Add insurance fund for socialized losses
7. **ADL**: Auto-deleveraging for extreme market conditions

## License

MIT
