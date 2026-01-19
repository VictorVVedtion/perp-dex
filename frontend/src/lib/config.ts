/**
 * Application Configuration
 * Centralized configuration management
 */

export const config = {
  // API Configuration
  api: {
    baseUrl: process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080',
    wsUrl: process.env.NEXT_PUBLIC_WS_URL || 'ws://localhost:8080/ws',
  },

  // Chain Configuration
  chain: {
    rpcUrl: process.env.NEXT_PUBLIC_RPC_URL || 'http://localhost:26657',
    restUrl: process.env.NEXT_PUBLIC_REST_URL || 'http://localhost:1317',
    chainId: process.env.NEXT_PUBLIC_CHAIN_ID || 'perpdex-local-1',
  },

  // Feature Flags
  features: {
    mockMode: process.env.NEXT_PUBLIC_MOCK_MODE === 'true',
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
    },
  },
};

export default config;
