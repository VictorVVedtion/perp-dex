/**
 * PerpDEX SDK Client
 * Main entry point for interacting with the PerpDEX protocol
 */

import axios, { AxiosInstance } from 'axios';
import BigNumber from 'bignumber.js';
import { WebSocketClient } from './websocket';
import type {
  ClientConfig,
  Market,
  Position,
  Order,
  OrderRequest,
  Trade,
  Account,
  Ticker,
  Orderbook,
  FundingRate,
  KlineData,
} from './types';

// Default configuration
const DEFAULT_CONFIG: Partial<ClientConfig> = {
  timeout: 30000,
  retryAttempts: 3,
  retryDelay: 1000,
};

/**
 * PerpDEX Client
 */
export class PerpDEXClient {
  private _config: ClientConfig;
  private _http: AxiosInstance;
  private _ws: WebSocketClient | null = null;

  constructor(config: ClientConfig) {
    this._config = { ...DEFAULT_CONFIG, ...config } as ClientConfig;

    // Setup HTTP client
    this._http = axios.create({
      baseURL: this._config.restUrl,
      timeout: this._config.timeout,
      headers: {
        'Content-Type': 'application/json',
      },
    });

    // Add request interceptor for auth
    this._http.interceptors.request.use((config) => {
      if (this._config.apiKey) {
        config.headers['X-API-Key'] = this._config.apiKey;
      }
      return config;
    });

    // Add retry interceptor
    this._http.interceptors.response.use(
      (response) => response,
      async (error) => {
        const config = error.config;
        if (!config || !config.__retryCount) {
          config.__retryCount = 0;
        }

        if (config.__retryCount >= this._config.retryAttempts!) {
          throw error;
        }

        config.__retryCount++;
        await new Promise((resolve) =>
          setTimeout(resolve, this._config.retryDelay)
        );
        return this._http(config);
      }
    );
  }

  // ============ Market Data ============

  /**
   * Get all available markets
   */
  async getMarkets(): Promise<Market[]> {
    const response = await this._http.get('/v1/markets');
    return response.data.markets;
  }

  /**
   * Get a specific market
   */
  async getMarket(marketId: string): Promise<Market> {
    const response = await this._http.get(`/v1/markets/${marketId}`);
    return response.data;
  }

  /**
   * Get ticker for a market
   */
  async getTicker(marketId: string): Promise<Ticker> {
    const response = await this._http.get(`/v1/markets/${marketId}/ticker`);
    return response.data;
  }

  /**
   * Get all tickers
   */
  async getAllTickers(): Promise<Ticker[]> {
    const response = await this._http.get('/v1/tickers');
    return response.data.tickers;
  }

  /**
   * Get orderbook for a market
   */
  async getOrderbook(marketId: string, depth: number = 20): Promise<Orderbook> {
    const response = await this._http.get(`/v1/markets/${marketId}/orderbook`, {
      params: { depth },
    });
    return response.data;
  }

  /**
   * Get recent trades for a market
   */
  async getTrades(marketId: string, limit: number = 100): Promise<Trade[]> {
    const response = await this._http.get(`/v1/markets/${marketId}/trades`, {
      params: { limit },
    });
    return response.data.trades;
  }

  /**
   * Get funding rate for a market
   */
  async getFundingRate(marketId: string): Promise<FundingRate> {
    const response = await this._http.get(`/v1/markets/${marketId}/funding`);
    return response.data;
  }

  /**
   * Get funding rate history
   */
  async getFundingHistory(
    marketId: string,
    limit: number = 100
  ): Promise<FundingRate[]> {
    const response = await this._http.get(
      `/v1/markets/${marketId}/funding/history`,
      { params: { limit } }
    );
    return response.data.history;
  }

  /**
   * Get K-line (candlestick) data
   * @param marketId - Market identifier (e.g., 'BTC-USDC')
   * @param interval - Time interval ('1m', '5m', '15m', '30m', '1h', '4h', '1d')
   * @param options - Optional parameters: from, to, limit
   */
  async getKlines(
    marketId: string,
    interval: string = '1m',
    options?: {
      from?: number;
      to?: number;
      limit?: number;
    }
  ): Promise<KlineData[]> {
    const response = await this._http.get(`/v1/markets/${marketId}/klines`, {
      params: {
        interval,
        from: options?.from,
        to: options?.to,
        limit: options?.limit || 200,
      },
    });
    return response.data.klines;
  }

  /**
   * Get latest K-lines
   * @param marketId - Market identifier
   * @param interval - Time interval
   * @param limit - Number of candles to return
   */
  async getLatestKlines(
    marketId: string,
    interval: string = '1m',
    limit: number = 200
  ): Promise<KlineData[]> {
    const response = await this._http.get(
      `/v1/markets/${marketId}/klines/latest`,
      { params: { interval, limit } }
    );
    return response.data.klines;
  }

  // ============ Account ============

  /**
   * Get account information
   */
  async getAccount(address: string): Promise<Account> {
    const response = await this._http.get(`/v1/accounts/${address}`);
    return response.data;
  }

  /**
   * Get account positions
   */
  async getPositions(address: string): Promise<Position[]> {
    const response = await this._http.get(`/v1/accounts/${address}/positions`);
    return response.data.positions;
  }

  /**
   * Get a specific position
   */
  async getPosition(address: string, marketId: string): Promise<Position | null> {
    try {
      const response = await this._http.get(
        `/v1/accounts/${address}/positions/${marketId}`
      );
      return response.data;
    } catch (error: any) {
      if (error.response?.status === 404) {
        return null;
      }
      throw error;
    }
  }

  /**
   * Get open orders
   */
  async getOpenOrders(address: string, marketId?: string): Promise<Order[]> {
    const response = await this._http.get(`/v1/accounts/${address}/orders`, {
      params: { market_id: marketId, status: 'open' },
    });
    return response.data.orders;
  }

  /**
   * Get order history
   */
  async getOrderHistory(
    address: string,
    marketId?: string,
    limit: number = 100
  ): Promise<Order[]> {
    const response = await this._http.get(
      `/v1/accounts/${address}/orders/history`,
      { params: { market_id: marketId, limit } }
    );
    return response.data.orders;
  }

  /**
   * Get trade history
   */
  async getTradeHistory(
    address: string,
    marketId?: string,
    limit: number = 100
  ): Promise<Trade[]> {
    const response = await this._http.get(
      `/v1/accounts/${address}/trades`,
      { params: { market_id: marketId, limit } }
    );
    return response.data.trades;
  }

  // ============ Trading ============

  /**
   * Place an order
   * Note: This creates the order message but requires signing with a wallet
   */
  createOrderMessage(
    address: string,
    order: OrderRequest
  ): {
    typeUrl: string;
    value: any;
  } {
    return {
      typeUrl: '/perpdex.orderbook.MsgPlaceOrder',
      value: {
        trader: address,
        marketId: order.marketId,
        side: order.side === 'buy' ? 1 : 2,
        orderType: this._orderTypeToNumber(order.type),
        price: order.price || '0',
        size: order.size,
        leverage: order.leverage || '1',
        reduceOnly: order.reduceOnly || false,
        postOnly: order.postOnly || false,
        timeInForce: this._tifToNumber(order.timeInForce),
        triggerPrice: order.triggerPrice || '0',
      },
    };
  }

  /**
   * Create cancel order message
   */
  createCancelOrderMessage(
    address: string,
    orderId: string,
    marketId: string
  ): {
    typeUrl: string;
    value: any;
  } {
    return {
      typeUrl: '/perpdex.orderbook.MsgCancelOrder',
      value: {
        trader: address,
        orderId,
        marketId,
      },
    };
  }

  /**
   * Create cancel all orders message
   */
  createCancelAllOrdersMessage(
    address: string,
    marketId?: string
  ): {
    typeUrl: string;
    value: any;
  } {
    return {
      typeUrl: '/perpdex.orderbook.MsgCancelAllOrders',
      value: {
        trader: address,
        marketId: marketId || '',
      },
    };
  }

  /**
   * Create close position message
   */
  createClosePositionMessage(
    address: string,
    marketId: string
  ): {
    typeUrl: string;
    value: any;
  } {
    return {
      typeUrl: '/perpdex.perpetual.MsgClosePosition',
      value: {
        trader: address,
        marketId,
      },
    };
  }

  /**
   * Create update margin message
   */
  createUpdateMarginMessage(
    address: string,
    marketId: string,
    amount: string,
    action: 'add' | 'remove'
  ): {
    typeUrl: string;
    value: any;
  } {
    return {
      typeUrl: '/perpdex.perpetual.MsgUpdateMargin',
      value: {
        trader: address,
        marketId,
        amount,
        action: action === 'add' ? 1 : 2,
      },
    };
  }

  /**
   * Create set margin mode message
   */
  createSetMarginModeMessage(
    address: string,
    marketId: string,
    mode: 'isolated' | 'cross'
  ): {
    typeUrl: string;
    value: any;
  } {
    return {
      typeUrl: '/perpdex.perpetual.MsgSetMarginMode',
      value: {
        trader: address,
        marketId,
        mode: mode === 'isolated' ? 1 : 2,
      },
    };
  }

  // ============ WebSocket ============

  /**
   * Connect to WebSocket for real-time data
   */
  connectWebSocket(): WebSocketClient {
    if (this._ws) {
      return this._ws;
    }

    this._ws = new WebSocketClient(this._config.wsUrl);
    this._ws.connect();
    return this._ws;
  }

  /**
   * Get WebSocket client
   */
  getWebSocket(): WebSocketClient | null {
    return this._ws;
  }

  /**
   * Disconnect WebSocket
   */
  disconnectWebSocket(): void {
    if (this._ws) {
      this._ws.disconnect();
      this._ws = null;
    }
  }

  // ============ Helpers ============

  /**
   * Calculate position PnL
   */
  calculatePnL(
    position: Position,
    currentPrice: string
  ): {
    unrealizedPnl: BigNumber;
    unrealizedPnlPercent: BigNumber;
    roe: BigNumber;
  } {
    const size = new BigNumber(position.size);
    const entryPrice = new BigNumber(position.entryPrice);
    const markPrice = new BigNumber(currentPrice);
    const margin = new BigNumber(position.margin);

    let pnl: BigNumber;
    if (position.side === 'long') {
      pnl = size.times(markPrice.minus(entryPrice));
    } else {
      pnl = size.times(entryPrice.minus(markPrice));
    }

    const pnlPercent = pnl.div(entryPrice.times(size)).times(100);
    const roe = margin.isPositive() ? pnl.div(margin).times(100) : new BigNumber(0);

    return {
      unrealizedPnl: pnl,
      unrealizedPnlPercent: pnlPercent,
      roe,
    };
  }

  /**
   * Calculate liquidation price
   */
  calculateLiquidationPrice(
    position: Position,
    maintenanceMarginRatio: number = 0.005
  ): BigNumber {
    const size = new BigNumber(position.size);
    const entryPrice = new BigNumber(position.entryPrice);
    const margin = new BigNumber(position.margin);
    const mmr = new BigNumber(maintenanceMarginRatio);

    if (position.side === 'long') {
      // liquidation_price = entry_price - (margin - mmr * size * entry_price) / size
      return entryPrice.minus(
        margin.minus(mmr.times(size).times(entryPrice)).div(size)
      );
    } else {
      // liquidation_price = entry_price + (margin - mmr * size * entry_price) / size
      return entryPrice.plus(
        margin.minus(mmr.times(size).times(entryPrice)).div(size)
      );
    }
  }

  /**
   * Calculate required margin
   */
  calculateRequiredMargin(
    size: string,
    price: string,
    leverage: string
  ): BigNumber {
    const sizeNum = new BigNumber(size);
    const priceNum = new BigNumber(price);
    const leverageNum = new BigNumber(leverage);

    return sizeNum.times(priceNum).div(leverageNum);
  }

  // ============ Private Helpers ============

  private _orderTypeToNumber(
    type: OrderRequest['type']
  ): number {
    const typeMap: Record<OrderRequest['type'], number> = {
      limit: 1,
      market: 2,
      stop_limit: 3,
      stop_market: 4,
    };
    return typeMap[type] || 1;
  }

  private _tifToNumber(tif?: OrderRequest['timeInForce']): number {
    const tifMap: Record<NonNullable<OrderRequest['timeInForce']>, number> = {
      gtc: 1,
      ioc: 2,
      fok: 3,
    };
    return tifMap[tif || 'gtc'] || 1;
  }
}

export default PerpDEXClient;
