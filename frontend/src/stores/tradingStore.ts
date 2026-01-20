/**
 * Trading Store with WebSocket Integration
 * Manages trading state and real-time data updates
 * Supports both local WebSocket and Hyperliquid API
 */

import { create } from 'zustand';
import BigNumber from 'bignumber.js';
import { WSClient, Ticker, Orderbook, Trade } from '@/lib/websocket/client';
import { HyperliquidWSClient, getHyperliquidWSClient } from '@/lib/websocket/hyperliquid';
import { getHyperliquidClient, NormalizedTicker, NormalizedOrderbook, NormalizedTrade } from '@/lib/api/hyperliquid';
import { config } from '@/lib/config';

// Types
export interface Order {
  orderId: string;
  trader: string;
  marketId: string;
  side: 'buy' | 'sell';
  orderType: 'limit' | 'market';
  price: string;
  quantity: string;
  filledQty: string;
  status: 'open' | 'filled' | 'partially_filled' | 'cancelled';
  createdAt: number;
}

export interface Position {
  trader: string;
  marketId: string;
  side: 'long' | 'short';
  size: string;
  entryPrice: string;
  margin: string;
  leverage: string;
  unrealizedPnl: string;
  liquidationPrice: string;
}

export interface Account {
  trader: string;
  balance: string;
  lockedMargin: string;
  totalEquity: string;
}

export interface PriceLevel {
  price: string;
  quantity: string;
}

export interface OrderBookData {
  bids: PriceLevel[];
  asks: PriceLevel[];
  bestBid: string;
  bestAsk: string;
  spread: string;
}

export interface PriceInfo {
  marketId: string;
  markPrice: string;
  indexPrice: string;
  lastPrice: string;
  change24h: string;
  high24h: string;
  low24h: string;
  volume24h: string;
}

interface TradingState {
  // Market data
  currentMarket: string;
  priceInfo: PriceInfo | null;
  orderBook: OrderBookData | null;
  ticker: Ticker | null;
  recentTrades: Trade[];

  // User data
  account: Account | null;
  positions: Position[];
  openOrders: Order[];

  // Trading form
  orderSide: 'buy' | 'sell';
  orderType: 'limit' | 'market' | 'trailing_stop';
  price: string;
  quantity: string;
  leverage: string;

  // WebSocket state
  wsClient: WSClient | null;
  hlWsClient: HyperliquidWSClient | null;
  wsConnected: boolean;
  wsError: string | null;

  // UI state
  isConnected: boolean;
  isLoading: boolean;
  error: string | null;

  // WebSocket actions
  initWebSocket: (url?: string) => void;
  closeWebSocket: () => void;
  initHyperliquid: () => void;
  closeHyperliquid: () => void;

  // Real-time data actions
  setTicker: (ticker: Ticker) => void;
  updateOrderbook: (orderbook: Orderbook) => void;
  addTrade: (trade: Trade) => void;

  // Actions
  setCurrentMarket: (market: string) => void;
  setPriceInfo: (info: PriceInfo) => void;
  setOrderBook: (data: OrderBookData) => void;
  setAccount: (account: Account) => void;
  setPositions: (positions: Position[]) => void;
  setOpenOrders: (orders: Order[]) => void;
  setOrderSide: (side: 'buy' | 'sell') => void;
  setOrderType: (type: 'limit' | 'market' | 'trailing_stop') => void;
  setPrice: (price: string) => void;
  setQuantity: (quantity: string) => void;
  setLeverage: (leverage: string) => void;
  setConnected: (connected: boolean) => void;
  setLoading: (loading: boolean) => void;
  setError: (error: string | null) => void;

  // Computed
  calculateMargin: () => string;
  calculatePnL: (position: Position) => string;
}

// WebSocket URL (default to local dev)
const DEFAULT_WS_URL = process.env.NEXT_PUBLIC_WS_URL || 'wss://ws.perpdex.io';

export const useTradingStore = create<TradingState>((set, get) => ({
  // Initial state
  currentMarket: 'BTC-USDC',
  priceInfo: null,
  orderBook: null,
  ticker: null,
  recentTrades: [],
  account: null,
  positions: [],
  openOrders: [],
  orderSide: 'buy',
  orderType: 'limit',
  price: '',
  quantity: '',
  leverage: '10',
  wsClient: null,
  hlWsClient: null,
  wsConnected: false,
  wsError: null,
  isConnected: false,
  isLoading: false,
  error: null,

  // WebSocket initialization
  initWebSocket: (url?: string) => {
    const { wsClient, currentMarket } = get();

    // Close existing connection
    if (wsClient) {
      wsClient.disconnect();
    }

    // Create new client
    const client = new WSClient(url || DEFAULT_WS_URL);

    // Set up callbacks
    client.onConnect(() => {
      set({ wsConnected: true, wsError: null });

      // Subscribe to market data
      client.subscribe(`ticker:${currentMarket}`, (data: Ticker) => {
        get().setTicker(data);
      });

      client.subscribe(`depth:${currentMarket}`, (data: Orderbook) => {
        get().updateOrderbook(data);
      });

      client.subscribe(`trades:${currentMarket}`, (data: Trade) => {
        get().addTrade(data);
      });
    });

    client.onDisconnect(() => {
      set({ wsConnected: false });
    });

    client.onError(() => {
      set({ wsError: 'WebSocket connection error' });
    });

    // Connect
    client.connect();

    set({ wsClient: client });
  },

  closeWebSocket: () => {
    const { wsClient } = get();
    if (wsClient) {
      wsClient.disconnect();
      set({ wsClient: null, wsConnected: false });
    }
  },

  // Hyperliquid WebSocket initialization
  initHyperliquid: () => {
    const { hlWsClient, currentMarket } = get();

    // Close existing connection
    if (hlWsClient) {
      hlWsClient.disconnect();
    }

    // Create new client
    const client = getHyperliquidWSClient();

    // Set up callbacks
    client.onConnect(() => {
      set({ wsConnected: true, wsError: null });

      // Subscribe to market data
      client.subscribeTicker(currentMarket, (ticker: NormalizedTicker) => {
        set({
          ticker: {
            marketId: ticker.marketId,
            markPrice: ticker.markPrice,
            indexPrice: ticker.indexPrice,
            lastPrice: ticker.lastPrice,
            high24h: ticker.high24h,
            low24h: ticker.low24h,
            volume24h: ticker.volume24h,
            change24h: ticker.change24h,
            fundingRate: ticker.fundingRate,
            nextFunding: 0,
            timestamp: Date.now(),
          },
          priceInfo: {
            marketId: ticker.marketId,
            markPrice: ticker.markPrice,
            indexPrice: ticker.indexPrice,
            lastPrice: ticker.lastPrice,
            change24h: ticker.change24h,
            high24h: ticker.high24h,
            low24h: ticker.low24h,
            volume24h: ticker.volume24h,
          },
        });
      });

      client.subscribeOrderbook(currentMarket, (orderbook: NormalizedOrderbook) => {
        const bids = orderbook.bids;
        const asks = orderbook.asks;
        const bestBid = bids[0]?.price || '0';
        const bestAsk = asks[0]?.price || '0';
        const spread = new BigNumber(bestAsk).minus(bestBid).toString();

        set({
          orderBook: {
            bids,
            asks,
            bestBid,
            bestAsk,
            spread,
          },
        });
      });

      client.subscribeTrades(currentMarket, (trade: NormalizedTrade) => {
        set((state) => ({
          recentTrades: [
            {
              tradeId: trade.id,
              marketId: trade.marketId,
              price: trade.price,
              quantity: trade.quantity,
              side: trade.side,
              timestamp: trade.timestamp,
            },
            ...state.recentTrades.slice(0, 99),
          ],
        }));
      });
    });

    client.onDisconnect(() => {
      set({ wsConnected: false });
    });

    client.onError(() => {
      set({ wsError: 'Hyperliquid WebSocket connection error' });
    });

    // Connect
    client.connect();

    // Also fetch initial data via REST API
    const fetchInitialData = async () => {
      try {
        const hlClient = getHyperliquidClient();

        // Fetch ticker
        const ticker = await hlClient.getTicker(currentMarket);
        if (ticker) {
          set({
            ticker: {
              marketId: ticker.marketId,
              markPrice: ticker.markPrice,
              indexPrice: ticker.indexPrice,
              lastPrice: ticker.lastPrice,
              high24h: ticker.high24h,
              low24h: ticker.low24h,
              volume24h: ticker.volume24h,
              change24h: ticker.change24h,
              fundingRate: ticker.fundingRate,
              nextFunding: 0,
              timestamp: Date.now(),
            },
            priceInfo: {
              marketId: ticker.marketId,
              markPrice: ticker.markPrice,
              indexPrice: ticker.indexPrice,
              lastPrice: ticker.lastPrice,
              change24h: ticker.change24h,
              high24h: ticker.high24h,
              low24h: ticker.low24h,
              volume24h: ticker.volume24h,
            },
          });
        }

        // Fetch orderbook
        const orderbook = await hlClient.getOrderbook(currentMarket);
        if (orderbook) {
          const bestBid = orderbook.bids[0]?.price || '0';
          const bestAsk = orderbook.asks[0]?.price || '0';
          const spread = new BigNumber(bestAsk).minus(bestBid).toString();

          set({
            orderBook: {
              bids: orderbook.bids,
              asks: orderbook.asks,
              bestBid,
              bestAsk,
              spread,
            },
          });
        }

        // Fetch recent trades
        const trades = await hlClient.getRecentTrades(currentMarket);
        if (trades.length > 0) {
          set({
            recentTrades: trades.map((trade) => ({
              tradeId: trade.id,
              marketId: trade.marketId,
              price: trade.price,
              quantity: trade.quantity,
              side: trade.side,
              timestamp: trade.timestamp,
            })),
          });
        }
      } catch (error) {
        console.error('Failed to fetch initial Hyperliquid data:', error);
      }
    };

    fetchInitialData();
    set({ hlWsClient: client });
  },

  closeHyperliquid: () => {
    const { hlWsClient } = get();
    if (hlWsClient) {
      hlWsClient.disconnect();
      set({ hlWsClient: null, wsConnected: false });
    }
  },

  // Real-time data handlers
  setTicker: (ticker: Ticker) => {
    set({ ticker });

    // Also update priceInfo for compatibility
    set({
      priceInfo: {
        marketId: ticker.marketId,
        markPrice: ticker.markPrice,
        indexPrice: ticker.indexPrice,
        lastPrice: ticker.lastPrice,
        change24h: ticker.change24h,
        high24h: ticker.high24h,
        low24h: ticker.low24h,
        volume24h: ticker.volume24h,
      },
    });
  },

  updateOrderbook: (orderbook: Orderbook) => {
    const bids = orderbook.bids.map((b) => ({
      price: b.price,
      quantity: b.quantity,
    }));
    const asks = orderbook.asks.map((a) => ({
      price: a.price,
      quantity: a.quantity,
    }));

    const bestBid = bids[0]?.price || '0';
    const bestAsk = asks[0]?.price || '0';
    const spread = new BigNumber(bestAsk).minus(bestBid).toString();

    set({
      orderBook: {
        bids,
        asks,
        bestBid,
        bestAsk,
        spread,
      },
    });
  },

  addTrade: (trade: Trade) => {
    set((state) => ({
      recentTrades: [trade, ...state.recentTrades.slice(0, 99)], // Keep last 100 trades
    }));
  },

  // Actions
  setCurrentMarket: (market) => {
    const { wsClient, currentMarket } = get();

    // Unsubscribe from old market
    if (wsClient && wsClient.isConnected()) {
      wsClient.unsubscribe(`ticker:${currentMarket}`);
      wsClient.unsubscribe(`depth:${currentMarket}`);
      wsClient.unsubscribe(`trades:${currentMarket}`);

      // Subscribe to new market
      wsClient.subscribe(`ticker:${market}`, (data: Ticker) => {
        get().setTicker(data);
      });
      wsClient.subscribe(`depth:${market}`, (data: Orderbook) => {
        get().updateOrderbook(data);
      });
      wsClient.subscribe(`trades:${market}`, (data: Trade) => {
        get().addTrade(data);
      });
    }

    set({ currentMarket: market, recentTrades: [] });
  },
  setPriceInfo: (info) => set({ priceInfo: info }),
  setOrderBook: (data) => set({ orderBook: data }),
  setAccount: (account) => set({ account }),
  setPositions: (positions) => set({ positions }),
  setOpenOrders: (orders) => set({ openOrders: orders }),
  setOrderSide: (side) => set({ orderSide: side }),
  setOrderType: (type) => set({ orderType: type }),
  setPrice: (price) => set({ price }),
  setQuantity: (quantity) => set({ quantity }),
  setLeverage: (leverage) => set({ leverage }),
  setConnected: (connected) => set({ isConnected: connected }),
  setLoading: (loading) => set({ isLoading: loading }),
  setError: (error) => set({ error }),

  // Computed - Updated with new 5% margin rate
  calculateMargin: () => {
    const { price, quantity, leverage } = get();
    if (!price || !quantity || !leverage) return '0';

    const notional = new BigNumber(price).times(quantity);
    const margin = notional.times(0.05); // 5% initial margin (updated from 10%)
    return margin.toFixed(2);
  },

  calculatePnL: (position) => {
    const { priceInfo } = get();
    if (!priceInfo) return '0';

    const markPrice = new BigNumber(priceInfo.markPrice);
    const entryPrice = new BigNumber(position.entryPrice);
    const size = new BigNumber(position.size);

    let priceDiff = markPrice.minus(entryPrice);
    if (position.side === 'short') {
      priceDiff = priceDiff.negated();
    }

    return size.times(priceDiff).toFixed(2);
  },
}));

// Note: Mock data removed - using real Hyperliquid API only
// All components should fetch real data from the trading store or Hyperliquid API
