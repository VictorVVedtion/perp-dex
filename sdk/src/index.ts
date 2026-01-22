/**
 * @perpdex/sdk - PerpDEX TypeScript SDK
 *
 * Official SDK for interacting with the PerpDEX perpetual futures DEX.
 *
 * @example
 * ```typescript
 * import { PerpDEXClient } from '@perpdex/sdk';
 *
 * // Create client
 * const client = new PerpDEXClient({
 *   restUrl: 'https://api.perpdex.io',
 *   wsUrl: 'wss://ws.perpdex.io',
 * });
 *
 * // Get markets
 * const markets = await client.getMarkets();
 *
 * // Get ticker
 * const ticker = await client.getTicker('BTC-USDC');
 *
 * // Subscribe to real-time data
 * const ws = client.connectWebSocket();
 * ws.subscribeTicker('BTC-USDC', (data) => {
 *   console.log('Price:', data.markPrice);
 * });
 *
 * // Create order message (requires wallet signing)
 * const orderMsg = client.createOrderMessage('perpdex1...', {
 *   marketId: 'BTC-USDC',
 *   side: 'buy',
 *   type: 'limit',
 *   price: '50000',
 *   size: '0.1',
 *   leverage: '10',
 * });
 * ```
 *
 * @packageDocumentation
 */

// Main client
export { PerpDEXClient } from './client';
export { default } from './client';

// WebSocket client
export { WebSocketClient } from './websocket';
export type { WebSocketConfig } from './websocket';

// RiverPool client
export { RiverpoolClient, createRiverpoolClient } from './riverpool';
export type {
  // RiverPool types
  Pool,
  Deposit,
  Withdrawal,
  PoolStats,
  DDGuardState,
  NAVHistory,
  UserPoolBalance,
  DepositEstimate,
  WithdrawalEstimate,
  DepositResponse,
  WithdrawalRequestResponse,
  WithdrawalClaimResponse,
  WithdrawalCancelResponse,
  RiverpoolClientConfig,
  // Revenue types
  RevenueStats,
  RevenueRecord,
  RevenueBreakdown,
} from './riverpool';

// Types
export type {
  // Config
  ClientConfig,

  // Market
  Market,
  MarketStatus,
  Ticker,
  Orderbook,
  OrderbookLevel,
  Trade,
  FundingRate,

  // Account
  Account,
  MarginMode,

  // Position
  Position,
  PositionSide,

  // Order
  Order,
  OrderSide,
  OrderType,
  OrderStatus,
  TimeInForce,
  OrderRequest,
  CancelOrderRequest,

  // Transaction
  TxResult,
  TxEvent,

  // WebSocket messages
  WSTickerMessage,
  WSDepthMessage,
  WSTradeMessage,
  WSPositionMessage,
  WSOrderMessage,
} from './types';

// Error classes
export {
  PerpDEXError,
  InsufficientMarginError,
  OrderNotFoundError,
  PositionNotFoundError,
  MarketNotFoundError,
  RateLimitError,
} from './types';

// Constants
export const MAINNET_CONFIG = {
  restUrl: 'https://api.perpdex.io',
  wsUrl: 'wss://ws.perpdex.io',
  chainId: 'perpdex-mainnet-1',
};

export const TESTNET_CONFIG = {
  restUrl: 'https://testnet-api.perpdex.io',
  wsUrl: 'wss://testnet-ws.perpdex.io',
  chainId: 'perpdex-testnet-1',
};

// Version
export const VERSION = '1.0.0';
