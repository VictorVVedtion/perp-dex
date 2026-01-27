/**
 * Application Configuration
 * Centralized configuration management
 *
 * Architecture:
 * - Market data (prices, orderbook, klines): Hyperliquid API
 * - Trading operations (orders, positions, accounts): Local API server
 * - Wallet: MetaMask (primary) with EIP-712 signing
 *
 * Environment Variables:
 * - NEXT_PUBLIC_API_URL: Backend API URL
 * - NEXT_PUBLIC_WS_URL: WebSocket URL
 * - NEXT_PUBLIC_MOCK_MODE: Enable mock wallet (development only)
 * - NEXT_PUBLIC_USE_REAL_ENGINE: Use real matching engine
 */

// Environment detection
const isDevelopment = process.env.NODE_ENV === 'development';
const isProduction = process.env.NODE_ENV === 'production';

export const config = {
  // Environment
  env: {
    isDevelopment,
    isProduction,
  },

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

  // Chain Configuration (for Cosmos SDK backend)
  chain: {
    rpcUrl: process.env.NEXT_PUBLIC_RPC_URL || 'http://localhost:26657',
    restUrl: process.env.NEXT_PUBLIC_REST_URL || 'http://localhost:1317',
    chainId: process.env.NEXT_PUBLIC_CHAIN_ID || 'perpdex-local-1',
  },

  // Wallet Configuration
  wallet: {
    // Preferred wallet provider (metamask, keplr, mock)
    preferredProvider: (process.env.NEXT_PUBLIC_WALLET_PROVIDER || 'metamask') as 'metamask' | 'keplr' | 'mock',
    // EIP-712 domain for MetaMask signing
    eip712Domain: {
      name: 'PerpDEX',
      version: '1',
      // chainId will be set dynamically from MetaMask
    },
    // Auto-connect timeout (ms)
    autoConnectTimeout: 5000,
  },

  // Feature Flags
  features: {
    // Mock mode: Use mock wallet for development (NOT recommended for production)
    // Set NEXT_PUBLIC_MOCK_MODE=true only in development
    mockMode: process.env.NEXT_PUBLIC_MOCK_MODE === 'true' && isDevelopment,

    // Use Hyperliquid for market data (prices, orderbook display, klines)
    useHyperliquidForMarketData: process.env.NEXT_PUBLIC_USE_HYPERLIQUID !== 'false',

    // Use real engine for trading (orders go to MatchingEngineV2)
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
