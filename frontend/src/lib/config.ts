/**
 * Application Configuration
 * Centralized configuration management
 *
 * Hybrid Mode Architecture:
 * - Market data (prices, orderbook display, klines): Hyperliquid API
 * - Trading operations (orders, positions, accounts): Local real engine
 *
 * To enable real engine mode, start API server with: --real flag
 */

export const config = {
  // API Configuration - Local server for trading operations
  api: {
    baseUrl: process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080',
    wsUrl: process.env.NEXT_PUBLIC_WS_URL || 'ws://localhost:8080/ws',
  },

  // Hyperliquid API Configuration - For market data only
  hyperliquid: {
    apiUrl: process.env.NEXT_PUBLIC_HL_API_URL || 'https://api.hyperliquid.xyz/info',
    wsUrl: process.env.NEXT_PUBLIC_HL_WS_URL || 'wss://api.hyperliquid.xyz/ws',
  },

  // Chain Configuration
  chain: {
    rpcUrl: process.env.NEXT_PUBLIC_RPC_URL || 'http://localhost:26657',
    restUrl: process.env.NEXT_PUBLIC_REST_URL || 'http://localhost:1317',
    chainId: process.env.NEXT_PUBLIC_CHAIN_ID || 'perpdex-local-1',
  },

  // Feature Flags - Hybrid Mode Configuration
  features: {
    // Legacy mock mode (all data is mocked)
    mockMode: process.env.NEXT_PUBLIC_MOCK_MODE === 'true',

    // Use Hyperliquid for market data (prices, orderbook display, klines)
    useHyperliquidForMarketData: process.env.NEXT_PUBLIC_USE_HYPERLIQUID !== 'false',

    // Use real engine for trading (orders go to MatchingEngineV2)
    // Set to true when API server is started with --real flag
    useRealEngineForTrading: process.env.NEXT_PUBLIC_USE_REAL_ENGINE === 'true',

    // Backward compatibility alias
    useHyperliquid: process.env.NEXT_PUBLIC_USE_HYPERLIQUID !== 'false',
  },

  // Trading Configuration
  trading: {
    defaultMarket: 'BTC-USDC',
    maxLeverage: 50,
    defaultLeverage: 10,
    minOrderSize: {
      'BTC-USDC': 0.001,
      'ETH-USDC': 0.01,
      'SOL-USDC': 0.1,
    } as Record<string, number>,
  },

  // Available Markets
  markets: [
    { id: 'BTC-USDC', name: 'BTC/USDC', baseAsset: 'BTC', quoteAsset: 'USDC' },
    { id: 'ETH-USDC', name: 'ETH/USDC', baseAsset: 'ETH', quoteAsset: 'USDC' },
    { id: 'SOL-USDC', name: 'SOL/USDC', baseAsset: 'SOL', quoteAsset: 'USDC' },
  ],
};

export default config;
