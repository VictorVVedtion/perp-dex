/**
 * Hyperliquid WebSocket Client
 * Handles real-time market data streaming from Hyperliquid
 *
 * WebSocket API: wss://api.hyperliquid.xyz/ws
 */

import type { NormalizedTicker, NormalizedOrderbook, NormalizedTrade } from '../api/hyperliquid';

// WebSocket URL
const HL_WS_URL = process.env.NEXT_PUBLIC_HL_WS_URL || 'wss://api.hyperliquid.xyz/ws';

// Market mapping
const MARKET_TO_COIN: Record<string, string> = {
  'BTC-USDC': 'BTC',
  'ETH-USDC': 'ETH',
  'SOL-USDC': 'SOL',
  'DOGE-USDC': 'DOGE',
  'ARB-USDC': 'ARB',
  'OP-USDC': 'OP',
  'AVAX-USDC': 'AVAX',
  'MATIC-USDC': 'MATIC',
  'LINK-USDC': 'LINK',
  'UNI-USDC': 'UNI',
};

const COIN_TO_MARKET: Record<string, string> = Object.fromEntries(
  Object.entries(MARKET_TO_COIN).map(([k, v]) => [v, k])
);

// Callback types
type TickerCallback = (ticker: NormalizedTicker) => void;
type OrderbookCallback = (orderbook: NormalizedOrderbook) => void;
type TradeCallback = (trade: NormalizedTrade) => void;

// Hyperliquid WebSocket message types
interface HLWsMessage {
  channel: string;
  data: any;
}

interface HLWsTrades {
  coin: string;
  side: 'A' | 'B';
  px: string;
  sz: string;
  hash: string;
  time: number;
  tid: number;
}

interface HLWsOrderbook {
  coin: string;
  levels: [[{ px: string; sz: string; n: number }], [{ px: string; sz: string; n: number }]];
  time: number;
}

interface HLWsAllMids {
  mids: Record<string, string>;
}

/**
 * Hyperliquid WebSocket Client
 */
export class HyperliquidWSClient {
  private ws: WebSocket | null = null;
  private url: string;
  private reconnectAttempts = 0;
  private maxReconnects = 10;
  private reconnectDelay = 1000;
  private isConnecting = false;
  private subscribedCoins = new Set<string>();

  // Callbacks
  private tickerCallbacks = new Map<string, Set<TickerCallback>>();
  private orderbookCallbacks = new Map<string, Set<OrderbookCallback>>();
  private tradeCallbacks = new Map<string, Set<TradeCallback>>();
  private onConnectCallbacks: (() => void)[] = [];
  private onDisconnectCallbacks: (() => void)[] = [];
  private onErrorCallbacks: ((error: Event) => void)[] = [];

  // Heartbeat
  private heartbeatInterval: ReturnType<typeof setInterval> | null = null;

  constructor(url: string = HL_WS_URL) {
    this.url = url;
  }

  /**
   * Connect to WebSocket server
   */
  connect(): void {
    if (this.ws?.readyState === WebSocket.OPEN || this.isConnecting) {
      return;
    }

    this.isConnecting = true;

    try {
      this.ws = new WebSocket(this.url);
      this.ws.onopen = this.handleOpen.bind(this);
      this.ws.onmessage = this.handleMessage.bind(this);
      this.ws.onclose = this.handleClose.bind(this);
      this.ws.onerror = this.handleError.bind(this);
    } catch (error) {
      console.error('Hyperliquid WebSocket connection failed:', error);
      this.isConnecting = false;
      this.scheduleReconnect();
    }
  }

  /**
   * Disconnect from WebSocket server
   */
  disconnect(): void {
    this.stopHeartbeat();
    this.reconnectAttempts = this.maxReconnects;

    if (this.ws) {
      this.ws.close(1000, 'Client disconnect');
      this.ws = null;
    }

    this.onDisconnectCallbacks.forEach((cb) => cb());
  }

  /**
   * Subscribe to ticker updates for a market
   */
  subscribeTicker(marketId: string, callback: TickerCallback): void {
    const coin = MARKET_TO_COIN[marketId];
    if (!coin) {
      console.warn(`Unknown market: ${marketId}`);
      return;
    }

    if (!this.tickerCallbacks.has(marketId)) {
      this.tickerCallbacks.set(marketId, new Set());
    }
    this.tickerCallbacks.get(marketId)!.add(callback);

    // Subscribe to allMids for ticker data
    this.subscribeAllMids();
  }

  /**
   * Subscribe to orderbook updates for a market
   */
  subscribeOrderbook(marketId: string, callback: OrderbookCallback): void {
    const coin = MARKET_TO_COIN[marketId];
    if (!coin) {
      console.warn(`Unknown market: ${marketId}`);
      return;
    }

    if (!this.orderbookCallbacks.has(marketId)) {
      this.orderbookCallbacks.set(marketId, new Set());
    }
    this.orderbookCallbacks.get(marketId)!.add(callback);

    this.subscribeL2Book(coin);
  }

  /**
   * Subscribe to trade updates for a market
   */
  subscribeTrades(marketId: string, callback: TradeCallback): void {
    const coin = MARKET_TO_COIN[marketId];
    if (!coin) {
      console.warn(`Unknown market: ${marketId}`);
      return;
    }

    if (!this.tradeCallbacks.has(marketId)) {
      this.tradeCallbacks.set(marketId, new Set());
    }
    this.tradeCallbacks.get(marketId)!.add(callback);

    this.subscribeCoinTrades(coin);
  }

  /**
   * Unsubscribe from ticker updates
   */
  unsubscribeTicker(marketId: string, callback?: TickerCallback): void {
    const callbacks = this.tickerCallbacks.get(marketId);
    if (callbacks) {
      if (callback) {
        callbacks.delete(callback);
      } else {
        callbacks.clear();
      }

      if (callbacks.size === 0) {
        this.tickerCallbacks.delete(marketId);
      }
    }
  }

  /**
   * Unsubscribe from orderbook updates
   */
  unsubscribeOrderbook(marketId: string, callback?: OrderbookCallback): void {
    const callbacks = this.orderbookCallbacks.get(marketId);
    if (callbacks) {
      if (callback) {
        callbacks.delete(callback);
      } else {
        callbacks.clear();
      }

      if (callbacks.size === 0) {
        this.orderbookCallbacks.delete(marketId);
        const coin = MARKET_TO_COIN[marketId];
        if (coin) {
          this.unsubscribeL2Book(coin);
        }
      }
    }
  }

  /**
   * Unsubscribe from trade updates
   */
  unsubscribeTrades(marketId: string, callback?: TradeCallback): void {
    const callbacks = this.tradeCallbacks.get(marketId);
    if (callbacks) {
      if (callback) {
        callbacks.delete(callback);
      } else {
        callbacks.clear();
      }

      if (callbacks.size === 0) {
        this.tradeCallbacks.delete(marketId);
        const coin = MARKET_TO_COIN[marketId];
        if (coin) {
          this.unsubscribeCoinTrades(coin);
        }
      }
    }
  }

  /**
   * Check if connected
   */
  isConnected(): boolean {
    return this.ws?.readyState === WebSocket.OPEN;
  }

  /**
   * Add connection callback
   */
  onConnect(callback: () => void): void {
    this.onConnectCallbacks.push(callback);
  }

  /**
   * Add disconnection callback
   */
  onDisconnect(callback: () => void): void {
    this.onDisconnectCallbacks.push(callback);
  }

  /**
   * Add error callback
   */
  onError(callback: (error: Event) => void): void {
    this.onErrorCallbacks.push(callback);
  }

  // Private methods

  private send(message: object): void {
    if (this.ws?.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify(message));
    }
  }

  private subscribeAllMids(): void {
    this.send({
      method: 'subscribe',
      subscription: { type: 'allMids' },
    });
  }

  private subscribeL2Book(coin: string): void {
    if (this.subscribedCoins.has(`l2:${coin}`)) return;

    this.send({
      method: 'subscribe',
      subscription: { type: 'l2Book', coin },
    });

    this.subscribedCoins.add(`l2:${coin}`);
  }

  private unsubscribeL2Book(coin: string): void {
    this.send({
      method: 'unsubscribe',
      subscription: { type: 'l2Book', coin },
    });

    this.subscribedCoins.delete(`l2:${coin}`);
  }

  private subscribeCoinTrades(coin: string): void {
    if (this.subscribedCoins.has(`trades:${coin}`)) return;

    this.send({
      method: 'subscribe',
      subscription: { type: 'trades', coin },
    });

    this.subscribedCoins.add(`trades:${coin}`);
  }

  private unsubscribeCoinTrades(coin: string): void {
    this.send({
      method: 'unsubscribe',
      subscription: { type: 'trades', coin },
    });

    this.subscribedCoins.delete(`trades:${coin}`);
  }

  private handleOpen(): void {
    console.log('Hyperliquid WebSocket connected');
    this.isConnecting = false;
    this.reconnectAttempts = 0;

    // Start heartbeat
    this.startHeartbeat();

    // Resubscribe to channels
    this.resubscribeAll();

    // Notify callbacks
    this.onConnectCallbacks.forEach((cb) => cb());
  }

  private handleMessage(event: MessageEvent): void {
    try {
      const message = JSON.parse(event.data);

      // Handle different message types
      if (message.channel === 'allMids') {
        this.handleAllMids(message.data);
      } else if (message.channel === 'l2Book') {
        this.handleL2Book(message.data);
      } else if (message.channel === 'trades') {
        this.handleTrades(message.data);
      } else if (message.channel === 'subscriptionResponse') {
        console.log('Subscription confirmed:', message.data);
      } else if (message.channel === 'pong') {
        // Heartbeat response
      }
    } catch (error) {
      console.error('Failed to parse Hyperliquid WebSocket message:', error);
    }
  }

  private handleAllMids(data: HLWsAllMids): void {
    // Update tickers for all subscribed markets
    this.tickerCallbacks.forEach((callbacks, marketId) => {
      const coin = MARKET_TO_COIN[marketId];
      if (coin && data.mids[coin]) {
        const midPrice = data.mids[coin];
        const ticker: NormalizedTicker = {
          marketId,
          markPrice: midPrice,
          indexPrice: midPrice,
          lastPrice: midPrice,
          high24h: midPrice,
          low24h: midPrice,
          volume24h: '0',
          change24h: '0%',
          fundingRate: '0',
          openInterest: '0',
        };

        callbacks.forEach((cb) => cb(ticker));
      }
    });
  }

  private handleL2Book(data: HLWsOrderbook): void {
    const marketId = COIN_TO_MARKET[data.coin];
    if (!marketId) return;

    const callbacks = this.orderbookCallbacks.get(marketId);
    if (!callbacks) return;

    const [bids, asks] = data.levels;

    const orderbook: NormalizedOrderbook = {
      marketId,
      bids: bids.map((level) => ({
        price: level.px,
        quantity: level.sz,
      })),
      asks: asks.map((level) => ({
        price: level.px,
        quantity: level.sz,
      })),
      timestamp: data.time,
    };

    callbacks.forEach((cb) => cb(orderbook));
  }

  private handleTrades(data: HLWsTrades[]): void {
    if (!Array.isArray(data) || data.length === 0) return;

    for (const trade of data) {
      const marketId = COIN_TO_MARKET[trade.coin];
      if (!marketId) continue;

      const callbacks = this.tradeCallbacks.get(marketId);
      if (!callbacks) continue;

      const normalizedTrade: NormalizedTrade = {
        id: `${trade.tid}-${trade.hash}`,
        marketId,
        price: trade.px,
        quantity: trade.sz,
        side: trade.side === 'B' ? 'buy' : 'sell',
        timestamp: trade.time,
      };

      callbacks.forEach((cb) => cb(normalizedTrade));
    }
  }

  private handleClose(event: CloseEvent): void {
    console.log('Hyperliquid WebSocket disconnected:', event.code, event.reason);
    this.isConnecting = false;
    this.stopHeartbeat();

    this.onDisconnectCallbacks.forEach((cb) => cb());

    if (event.code !== 1000) {
      this.scheduleReconnect();
    }
  }

  private handleError(error: Event): void {
    console.error('Hyperliquid WebSocket error:', error);
    this.isConnecting = false;

    this.onErrorCallbacks.forEach((cb) => cb(error));
  }

  private scheduleReconnect(): void {
    if (this.reconnectAttempts >= this.maxReconnects) {
      console.error('Max reconnection attempts reached for Hyperliquid WebSocket');
      return;
    }

    const delay = this.reconnectDelay * Math.pow(2, this.reconnectAttempts);
    console.log(
      `Reconnecting to Hyperliquid in ${delay}ms (attempt ${this.reconnectAttempts + 1}/${this.maxReconnects})`
    );

    setTimeout(() => {
      this.reconnectAttempts++;
      this.connect();
    }, delay);
  }

  private resubscribeAll(): void {
    // Resubscribe to allMids if we have ticker subscriptions
    if (this.tickerCallbacks.size > 0) {
      this.subscribeAllMids();
    }

    // Resubscribe to L2 books
    this.subscribedCoins.forEach((key) => {
      if (key.startsWith('l2:')) {
        const coin = key.slice(3);
        this.send({
          method: 'subscribe',
          subscription: { type: 'l2Book', coin },
        });
      } else if (key.startsWith('trades:')) {
        const coin = key.slice(7);
        this.send({
          method: 'subscribe',
          subscription: { type: 'trades', coin },
        });
      }
    });
  }

  private startHeartbeat(): void {
    this.stopHeartbeat();

    // Send ping every 30 seconds
    this.heartbeatInterval = setInterval(() => {
      if (this.ws?.readyState === WebSocket.OPEN) {
        this.send({ method: 'ping' });
      }
    }, 30000);
  }

  private stopHeartbeat(): void {
    if (this.heartbeatInterval) {
      clearInterval(this.heartbeatInterval);
      this.heartbeatInterval = null;
    }
  }
}

// Singleton instance
let wsClientInstance: HyperliquidWSClient | null = null;

export function getHyperliquidWSClient(): HyperliquidWSClient {
  if (!wsClientInstance) {
    wsClientInstance = new HyperliquidWSClient();
  }
  return wsClientInstance;
}

export default HyperliquidWSClient;
