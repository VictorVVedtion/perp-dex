/**
 * Hyperliquid API Client
 * Provides access to Hyperliquid's public API for market data
 *
 * API Documentation: https://hyperliquid.gitbook.io/hyperliquid-docs/for-developers/api
 */

// API Endpoints
const HL_API_URL = process.env.NEXT_PUBLIC_HL_API_URL || 'https://api.hyperliquid.xyz/info';

// Types
export interface HLTicker {
  coin: string;
  markPx: string;
  midPx: string;
  prevDayPx: string;
  dayNtlVlm: string;
  premium: string;
  oraclePx: string;
  impactPxs: [string, string]; // [bid impact, ask impact]
  funding: string;
  openInterest: string;
}

export interface HLOrderbookLevel {
  px: string;
  sz: string;
  n: number;
}

export interface HLOrderbook {
  coin: string;
  levels: [HLOrderbookLevel[], HLOrderbookLevel[]]; // [bids, asks]
  time: number;
}

export interface HLTrade {
  coin: string;
  side: 'A' | 'B'; // A = Ask taker (sell), B = Bid taker (buy)
  px: string;
  sz: string;
  hash: string;
  time: number;
  tid: number;
}

export interface HLCandle {
  t: number; // timestamp (ms)
  T: number; // close time
  s: string; // symbol
  i: string; // interval
  o: string; // open
  c: string; // close
  h: string; // high
  l: string; // low
  v: string; // volume (base)
  n: number; // number of trades
}

export interface HLMeta {
  universe: {
    name: string;
    szDecimals: number;
    maxLeverage: number;
    onlyIsolated: boolean;
  }[];
}

// Normalized types for our application
export interface NormalizedTicker {
  marketId: string;
  markPrice: string;
  indexPrice: string;
  lastPrice: string;
  high24h: string;
  low24h: string;
  volume24h: string;
  change24h: string;
  fundingRate: string;
  openInterest: string;
}

export interface NormalizedOrderbook {
  marketId: string;
  bids: { price: string; quantity: string }[];
  asks: { price: string; quantity: string }[];
  timestamp: number;
}

export interface NormalizedTrade {
  id: string;
  marketId: string;
  price: string;
  quantity: string;
  side: 'buy' | 'sell';
  timestamp: number;
}

export interface NormalizedCandle {
  time: number;
  open: number;
  high: number;
  low: number;
  close: number;
  volume: number;
}

/**
 * Map from our market IDs to Hyperliquid coins
 */
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

/**
 * Map interval to Hyperliquid format
 */
const INTERVAL_MAP: Record<string, string> = {
  '1m': '1m',
  '5m': '5m',
  '15m': '15m',
  '30m': '30m',
  '1h': '1h',
  '4h': '4h',
  '1d': '1d',
};

/**
 * Hyperliquid API Client
 */
export class HyperliquidClient {
  private baseUrl: string;
  private defaultTimeout: number;

  constructor(baseUrl: string = HL_API_URL, timeout: number = 10000) {
    this.baseUrl = baseUrl;
    this.defaultTimeout = timeout;
  }

  /**
   * Make a POST request to Hyperliquid API
   * CRITICAL FIX: Added AbortController timeout to prevent indefinite hangs
   */
  private async request<T>(payload: object, timeout?: number): Promise<T> {
    const controller = new AbortController();
    const timeoutMs = timeout ?? this.defaultTimeout;
    const timeoutId = setTimeout(() => controller.abort(), timeoutMs);

    try {
      const response = await fetch(this.baseUrl, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(payload),
        signal: controller.signal,
      });

      if (!response.ok) {
        throw new Error(`Hyperliquid API error: ${response.status}`);
      }

      return response.json();
    } catch (error) {
      if (error instanceof Error && error.name === 'AbortError') {
        throw new Error(`Hyperliquid API timeout after ${timeoutMs}ms`);
      }
      throw error;
    } finally {
      clearTimeout(timeoutId);
    }
  }

  /**
   * Get market metadata
   */
  async getMeta(): Promise<HLMeta> {
    return this.request<HLMeta>({ type: 'meta' });
  }

  /**
   * Get all tickers
   */
  async getAllTickers(): Promise<HLTicker[]> {
    const response = await this.request<{ assetCtxs: HLTicker[] }>({
      type: 'allMids',
    });

    // allMids returns { mids: {...}, assetCtxs: [...] }
    // We need the full context from metaAndAssetCtxs
    const fullResponse = await this.request<[HLMeta, HLTicker[]]>({
      type: 'metaAndAssetCtxs',
    });

    return fullResponse[1];
  }

  /**
   * Get ticker for a specific market
   */
  async getTicker(marketId: string): Promise<NormalizedTicker | null> {
    const coin = MARKET_TO_COIN[marketId];
    if (!coin) {
      console.warn(`Unknown market: ${marketId}`);
      return null;
    }

    try {
      const [meta, assetCtxs] = await this.request<[HLMeta, HLTicker[]]>({
        type: 'metaAndAssetCtxs',
      });

      const index = meta.universe.findIndex((u) => u.name === coin);
      if (index === -1) {
        return null;
      }

      const ticker = assetCtxs[index];
      return this.normalizeTicker(ticker, marketId);
    } catch (error) {
      console.error('Failed to fetch ticker:', error);
      return null;
    }
  }

  /**
   * Get orderbook for a market
   */
  async getOrderbook(marketId: string, depth: number = 20): Promise<NormalizedOrderbook | null> {
    const coin = MARKET_TO_COIN[marketId];
    if (!coin) {
      console.warn(`Unknown market: ${marketId}`);
      return null;
    }

    try {
      const response = await this.request<HLOrderbook>({
        type: 'l2Book',
        coin,
        nSigFigs: 5,
        mantissa: null,
      });

      return this.normalizeOrderbook(response, marketId);
    } catch (error) {
      console.error('Failed to fetch orderbook:', error);
      return null;
    }
  }

  /**
   * Get recent trades
   */
  async getRecentTrades(marketId: string, limit: number = 50): Promise<NormalizedTrade[]> {
    const coin = MARKET_TO_COIN[marketId];
    if (!coin) {
      console.warn(`Unknown market: ${marketId}`);
      return [];
    }

    try {
      const trades = await this.request<HLTrade[]>({
        type: 'recentTrades',
        coin,
      });

      return trades.slice(0, limit).map((trade) => this.normalizeTrade(trade, marketId));
    } catch (error) {
      console.error('Failed to fetch recent trades:', error);
      return [];
    }
  }

  /**
   * Get candlestick data
   */
  async getCandles(
    marketId: string,
    interval: string = '1m',
    limit: number = 200
  ): Promise<NormalizedCandle[]> {
    const coin = MARKET_TO_COIN[marketId];
    if (!coin) {
      console.warn(`Unknown market: ${marketId}`);
      return [];
    }

    const hlInterval = INTERVAL_MAP[interval] || '1m';
    const endTime = Date.now();

    // Calculate start time based on interval and limit
    const intervalMs: Record<string, number> = {
      '1m': 60 * 1000,
      '5m': 5 * 60 * 1000,
      '15m': 15 * 60 * 1000,
      '30m': 30 * 60 * 1000,
      '1h': 60 * 60 * 1000,
      '4h': 4 * 60 * 60 * 1000,
      '1d': 24 * 60 * 60 * 1000,
    };

    const startTime = endTime - (intervalMs[interval] || intervalMs['1m']) * limit;

    try {
      const candles = await this.request<HLCandle[]>({
        type: 'candleSnapshot',
        req: {
          coin,
          interval: hlInterval,
          startTime,
          endTime,
        },
      });

      return candles.map((candle) => this.normalizeCandle(candle));
    } catch (error) {
      console.error('Failed to fetch candles:', error);
      return [];
    }
  }

  /**
   * Normalize ticker data
   */
  private normalizeTicker(ticker: HLTicker, marketId: string): NormalizedTicker {
    const markPrice = parseFloat(ticker.markPx);
    const prevDayPrice = parseFloat(ticker.prevDayPx);
    const changePercent = prevDayPrice > 0
      ? ((markPrice - prevDayPrice) / prevDayPrice * 100).toFixed(2)
      : '0';

    return {
      marketId,
      markPrice: ticker.markPx,
      indexPrice: ticker.oraclePx,
      lastPrice: ticker.markPx,
      high24h: ticker.markPx, // HL doesn't provide 24h high/low directly
      low24h: ticker.markPx,
      volume24h: ticker.dayNtlVlm,
      change24h: `${parseFloat(changePercent) >= 0 ? '+' : ''}${changePercent}%`,
      fundingRate: ticker.funding,
      openInterest: ticker.openInterest,
    };
  }

  /**
   * Normalize orderbook data
   */
  private normalizeOrderbook(orderbook: HLOrderbook, marketId: string): NormalizedOrderbook {
    const [bids, asks] = orderbook.levels;

    return {
      marketId,
      bids: bids.map((level) => ({
        price: level.px,
        quantity: level.sz,
      })),
      asks: asks.map((level) => ({
        price: level.px,
        quantity: level.sz,
      })),
      timestamp: orderbook.time,
    };
  }

  /**
   * Normalize trade data
   */
  private normalizeTrade(trade: HLTrade, marketId: string): NormalizedTrade {
    return {
      id: `${trade.tid}-${trade.hash}`,
      marketId,
      price: trade.px,
      quantity: trade.sz,
      side: trade.side === 'B' ? 'buy' : 'sell',
      timestamp: trade.time,
    };
  }

  /**
   * Normalize candle data
   */
  private normalizeCandle(candle: HLCandle): NormalizedCandle {
    return {
      time: Math.floor(candle.t / 1000), // Convert to seconds for lightweight-charts
      open: parseFloat(candle.o),
      high: parseFloat(candle.h),
      low: parseFloat(candle.l),
      close: parseFloat(candle.c),
      volume: parseFloat(candle.v),
    };
  }
}

// Singleton instance
let clientInstance: HyperliquidClient | null = null;

export function getHyperliquidClient(): HyperliquidClient {
  if (!clientInstance) {
    clientInstance = new HyperliquidClient();
  }
  return clientInstance;
}

export default HyperliquidClient;
