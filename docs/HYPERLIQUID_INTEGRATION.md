# Hyperliquid API Integration

## Overview

This document describes the integration of Hyperliquid's public API as the data source for real market data in PerpDEX.

## Configuration

### Environment Variables

The integration is controlled by the following environment variables in `frontend/.env.local`:

```env
# Feature flags
NEXT_PUBLIC_MOCK_MODE=false        # Set to 'true' to use mock data
NEXT_PUBLIC_USE_HYPERLIQUID=true   # Set to 'false' to disable Hyperliquid

# Hyperliquid API endpoints
NEXT_PUBLIC_HL_API_URL=https://api.hyperliquid.xyz/info
NEXT_PUBLIC_HL_WS_URL=wss://api.hyperliquid.xyz/ws
```

### Switching Modes

**Real Mode (Hyperliquid):**
```env
NEXT_PUBLIC_MOCK_MODE=false
NEXT_PUBLIC_USE_HYPERLIQUID=true
```

**Local Backend Only:**
```env
NEXT_PUBLIC_MOCK_MODE=false
NEXT_PUBLIC_USE_HYPERLIQUID=false
```

**Mock Mode (Development):**
```env
NEXT_PUBLIC_MOCK_MODE=true
NEXT_PUBLIC_USE_HYPERLIQUID=false
```

## Architecture

### Data Flow

```
┌─────────────────┐     ┌──────────────────┐     ┌─────────────┐
│  Hyperliquid    │────▶│  Frontend        │────▶│  UI         │
│  API/WebSocket  │     │  Trading Store   │     │  Components │
└─────────────────┘     └──────────────────┘     └─────────────┘
                               │
                               ▼
                        ┌──────────────────┐
                        │  Local Backend   │
                        │  (Order Mgmt)    │
                        └──────────────────┘
```

### Key Components

1. **`lib/api/hyperliquid.ts`** - REST API client for Hyperliquid
   - Price data (tickers)
   - Orderbook data
   - Recent trades
   - Candlestick/K-line data

2. **`lib/websocket/hyperliquid.ts`** - WebSocket client for real-time updates
   - Ticker subscriptions
   - Orderbook updates
   - Trade stream

3. **`stores/tradingStore.ts`** - State management
   - `initHyperliquid()` - Initialize Hyperliquid connection
   - `closeHyperliquid()` - Close connection

4. **`pages/_app.tsx`** - Application initialization
   - Auto-connects based on configuration

## Supported Markets

The integration supports the following market pairs:

| Market ID   | Hyperliquid Coin |
|-------------|------------------|
| BTC-USDC    | BTC              |
| ETH-USDC    | ETH              |
| SOL-USDC    | SOL              |
| DOGE-USDC   | DOGE             |
| ARB-USDC    | ARB              |
| OP-USDC     | OP               |
| AVAX-USDC   | AVAX             |
| MATIC-USDC  | MATIC            |
| LINK-USDC   | LINK             |
| UNI-USDC    | UNI              |

## API Endpoints Used

### REST API (`https://api.hyperliquid.xyz/info`)

| Type              | Description                    |
|-------------------|--------------------------------|
| meta              | Market metadata                |
| metaAndAssetCtxs  | Market data with asset context |
| l2Book            | Order book depth               |
| recentTrades      | Recent trades                  |
| candleSnapshot    | Historical candles             |

### WebSocket (`wss://api.hyperliquid.xyz/ws`)

| Subscription | Description              |
|--------------|--------------------------|
| allMids      | Real-time mid prices     |
| l2Book       | Order book updates       |
| trades       | Trade stream             |

## Testing

### Verify API Connectivity

```bash
./scripts/verify-hyperliquid.sh
```

### Start Development Server

```bash
cd frontend
npm run dev
```

### Build for Production

```bash
cd frontend
npm run build
npm start
```

## Component Indicators

When Hyperliquid data is active, components display a "HL" badge:

- **Chart Component** - Shows "HL" tag next to market name
- **OrderBook Component** - Shows "HL" tag in header
- **RecentTrades Component** - Shows "HL" tag next to title

## Error Handling

All components have fallback mechanisms:

1. **API Failure** → Falls back to local API
2. **Local API Failure** → Falls back to mock data
3. **WebSocket Disconnection** → Auto-reconnect with exponential backoff

## Performance Considerations

- WebSocket connections use heartbeat (30s interval)
- Auto-reconnect with exponential backoff (max 10 attempts)
- Initial data fetched via REST before WebSocket subscription
- React components use memoization for performance

## Limitations

1. **Read-only data** - Only market data is sourced from Hyperliquid
2. **No trading** - Order execution uses local backend
3. **Market mapping** - Only mapped markets are supported

## Troubleshooting

### No data displayed
1. Check browser console for errors
2. Verify network connectivity to Hyperliquid API
3. Check environment variable configuration

### WebSocket disconnecting
1. Check network stability
2. Verify WebSocket URL is correct
3. Check console for reconnection messages

### Stale data
1. Verify WebSocket connection is active (look for "Live" indicator)
2. Refresh the page to reinitialize connections
