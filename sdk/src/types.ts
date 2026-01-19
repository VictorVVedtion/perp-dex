/**
 * PerpDEX SDK Types
 */

// Client configuration
export interface ClientConfig {
  restUrl: string;
  wsUrl: string;
  chainId?: string;
  apiKey?: string;
  timeout?: number;
  retryAttempts?: number;
  retryDelay?: number;
}

// Market
export interface Market {
  marketId: string;
  baseAsset: string;
  quoteAsset: string;
  tickSize: string;
  stepSize: string;
  minOrderSize: string;
  maxOrderSize: string;
  maxLeverage: string;
  maintenanceMarginRatio: string;
  initialMarginRatio: string;
  makerFee: string;
  takerFee: string;
  status: MarketStatus;
  fundingInterval: number;
}

export type MarketStatus = 'active' | 'inactive' | 'settling' | 'paused';

// Ticker
export interface Ticker {
  marketId: string;
  markPrice: string;
  indexPrice: string;
  lastPrice: string;
  high24h: string;
  low24h: string;
  volume24h: string;
  turnover24h: string;
  change24h: string;
  changePercent24h: string;
  fundingRate: string;
  nextFundingTime: number;
  openInterest: string;
  timestamp: number;
}

// Orderbook
export interface Orderbook {
  marketId: string;
  bids: OrderbookLevel[];
  asks: OrderbookLevel[];
  timestamp: number;
}

export interface OrderbookLevel {
  price: string;
  size: string;
  orderCount?: number;
}

// Trade
export interface Trade {
  tradeId: string;
  marketId: string;
  price: string;
  size: string;
  side: 'buy' | 'sell';
  timestamp: number;
  makerOrderId?: string;
  takerOrderId?: string;
}

// Funding rate
export interface FundingRate {
  marketId: string;
  fundingRate: string;
  markPrice: string;
  indexPrice: string;
  nextFundingTime: number;
  timestamp: number;
}

// K-line (candlestick) data
export interface KlineData {
  time: number;      // Unix timestamp
  open: number;
  high: number;
  low: number;
  close: number;
  volume?: number;
  turnover?: number;
}

// K-line interval types
export type KlineInterval = '1m' | '5m' | '15m' | '30m' | '1h' | '4h' | '1d';

// Account
export interface Account {
  address: string;
  balance: string;
  availableBalance: string;
  lockedMargin: string;
  unrealizedPnl: string;
  realizedPnl: string;
  marginMode: MarginMode;
  positionCount: number;
  openOrderCount: number;
}

export type MarginMode = 'isolated' | 'cross';

// Position
export interface Position {
  trader: string;
  marketId: string;
  side: PositionSide;
  size: string;
  entryPrice: string;
  markPrice: string;
  liquidationPrice: string;
  unrealizedPnl: string;
  realizedPnl: string;
  margin: string;
  leverage: string;
  marginMode: MarginMode;
  createdAt: number;
  updatedAt: number;
}

export type PositionSide = 'long' | 'short';

// Order
export interface Order {
  orderId: string;
  marketId: string;
  trader: string;
  side: OrderSide;
  type: OrderType;
  price: string;
  size: string;
  filledSize: string;
  remainingSize: string;
  avgFillPrice: string;
  status: OrderStatus;
  reduceOnly: boolean;
  postOnly: boolean;
  timeInForce: TimeInForce;
  triggerPrice?: string;
  createdAt: number;
  updatedAt: number;
}

export type OrderSide = 'buy' | 'sell';
export type OrderType = 'limit' | 'market' | 'stop_limit' | 'stop_market';
export type OrderStatus = 'open' | 'partial' | 'filled' | 'cancelled' | 'expired' | 'triggered';
export type TimeInForce = 'gtc' | 'ioc' | 'fok';

// Order request
export interface OrderRequest {
  marketId: string;
  side: OrderSide;
  type: OrderType;
  price?: string;
  size: string;
  leverage?: string;
  reduceOnly?: boolean;
  postOnly?: boolean;
  timeInForce?: TimeInForce;
  triggerPrice?: string;
  clientOrderId?: string;
}

// Cancel order request
export interface CancelOrderRequest {
  orderId: string;
  marketId: string;
}

// Transaction result
export interface TxResult {
  hash: string;
  height: number;
  code: number;
  rawLog: string;
  gasUsed: number;
  gasWanted: number;
  events: TxEvent[];
}

export interface TxEvent {
  type: string;
  attributes: { key: string; value: string }[];
}

// WebSocket message types
export interface WSTickerMessage {
  marketId: string;
  markPrice: string;
  indexPrice: string;
  lastPrice: string;
  high24h: string;
  low24h: string;
  volume24h: string;
  change24h: string;
  fundingRate: string;
  nextFunding: number;
  timestamp: number;
}

export interface WSDepthMessage {
  marketId: string;
  bids: [string, string][];
  asks: [string, string][];
  timestamp: number;
}

export interface WSTradeMessage {
  tradeId: string;
  marketId: string;
  price: string;
  size: string;
  side: OrderSide;
  timestamp: number;
}

export interface WSPositionMessage {
  trader: string;
  marketId: string;
  side: PositionSide;
  size: string;
  entryPrice: string;
  markPrice: string;
  unrealizedPnl: string;
  margin: string;
  leverage: string;
  liquidationPrice: string;
  timestamp: number;
}

export interface WSOrderMessage {
  orderId: string;
  marketId: string;
  trader: string;
  side: OrderSide;
  type: OrderType;
  price: string;
  size: string;
  filledSize: string;
  status: OrderStatus;
  timestamp: number;
}

// Error types
export class PerpDEXError extends Error {
  code: string;
  details?: any;

  constructor(code: string, message: string, details?: any) {
    super(message);
    this.code = code;
    this.details = details;
    this.name = 'PerpDEXError';
  }
}

export class InsufficientMarginError extends PerpDEXError {
  constructor(required: string, available: string) {
    super(
      'INSUFFICIENT_MARGIN',
      `Insufficient margin: required ${required}, available ${available}`,
      { required, available }
    );
  }
}

export class OrderNotFoundError extends PerpDEXError {
  constructor(orderId: string) {
    super('ORDER_NOT_FOUND', `Order not found: ${orderId}`, { orderId });
  }
}

export class PositionNotFoundError extends PerpDEXError {
  constructor(marketId: string) {
    super('POSITION_NOT_FOUND', `Position not found for market: ${marketId}`, {
      marketId,
    });
  }
}

export class MarketNotFoundError extends PerpDEXError {
  constructor(marketId: string) {
    super('MARKET_NOT_FOUND', `Market not found: ${marketId}`, { marketId });
  }
}

export class RateLimitError extends PerpDEXError {
  retryAfter: number;

  constructor(retryAfter: number) {
    super('RATE_LIMIT', `Rate limit exceeded. Retry after ${retryAfter}s`, {
      retryAfter,
    });
    this.retryAfter = retryAfter;
  }
}
