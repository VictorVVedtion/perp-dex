/**
 * TradingView-style Chart Component
 * Uses lightweight-charts library for professional trading charts
 * Supports both real API data and mock data for development
 */

import { useEffect, useRef, useState, useCallback } from 'react';
import { createChart, CandlestickData, Time, CandlestickSeries } from 'lightweight-charts';
import type { IChartApi, ISeriesApi } from 'lightweight-charts';
import { useTradingStore } from '@/stores/tradingStore';
import { config } from '@/lib/config';

// K-line intervals
type Interval = '1m' | '5m' | '15m' | '30m' | '1h' | '4h' | '1d';

interface ChartProps {
  marketId?: string;
  height?: number;
}

interface KlineApiResponse {
  market_id: string;
  interval: string;
  klines: {
    time: number;
    open: number;
    high: number;
    low: number;
    close: number;
    volume: number;
    turnover: number;
  }[];
}

// Simulated K-line data for development (fallback)
const generateMockKlines = (count: number = 200, basePrice: number = 50000): CandlestickData[] => {
  const klines: CandlestickData[] = [];
  let price = basePrice;
  const now = Math.floor(Date.now() / 1000);
  const interval = 60; // 1 minute

  for (let i = count; i >= 0; i--) {
    const time = (now - i * interval) as Time;
    const volatility = 0.002; // 0.2%
    const change = (Math.random() - 0.5) * 2 * volatility;
    const open = price;
    const close = price * (1 + change);
    const high = Math.max(open, close) * (1 + Math.random() * volatility);
    const low = Math.min(open, close) * (1 - Math.random() * volatility);

    klines.push({
      time,
      open,
      high,
      low,
      close,
    });

    price = close;
  }

  return klines;
};

// Fetch K-lines from API
async function fetchKlines(
  marketId: string,
  interval: Interval,
  limit: number = 200
): Promise<CandlestickData[]> {
  try {
    const response = await fetch(
      `${config.api.baseUrl}/v1/markets/${marketId}/klines/latest?interval=${interval}&limit=${limit}`
    );

    if (!response.ok) {
      throw new Error(`API error: ${response.status}`);
    }

    const data: KlineApiResponse = await response.json();

    return data.klines.map((k) => ({
      time: k.time as Time,
      open: k.open,
      high: k.high,
      low: k.low,
      close: k.close,
    }));
  } catch (error) {
    console.warn('Failed to fetch klines from API, using mock data:', error);
    return generateMockKlines(limit);
  }
}

export function Chart({ marketId = 'BTC-USDC', height = 400 }: ChartProps) {
  const chartContainerRef = useRef<HTMLDivElement>(null);
  const chartRef = useRef<IChartApi | null>(null);
  const candlestickSeriesRef = useRef<ISeriesApi<'Candlestick'> | null>(null);
  const [interval, setInterval] = useState<Interval>('1m');
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const { ticker, wsConnected, wsClient } = useTradingStore();

  // Load K-line data
  const loadKlines = useCallback(async (selectedInterval: Interval) => {
    if (!candlestickSeriesRef.current) return;

    setIsLoading(true);
    setError(null);

    try {
      const klines = await fetchKlines(marketId, selectedInterval);
      candlestickSeriesRef.current.setData(klines);
      chartRef.current?.timeScale().fitContent();
    } catch (err) {
      setError('Failed to load chart data');
      console.error(err);
    } finally {
      setIsLoading(false);
    }
  }, [marketId]);

  // Initialize chart
  useEffect(() => {
    if (!chartContainerRef.current) return;

    // Create chart
    const chart = createChart(chartContainerRef.current, {
      width: chartContainerRef.current.clientWidth,
      height: height,
      layout: {
        background: { color: '#0f0f1a' },
        textColor: '#9ca3af',
      },
      grid: {
        vertLines: { color: '#1f2937' },
        horzLines: { color: '#1f2937' },
      },
      crosshair: {
        mode: 1, // Magnet mode
        vertLine: {
          color: '#4f46e5',
          width: 1,
          style: 2, // Dashed
          labelBackgroundColor: '#4f46e5',
        },
        horzLine: {
          color: '#4f46e5',
          width: 1,
          style: 2,
          labelBackgroundColor: '#4f46e5',
        },
      },
      timeScale: {
        borderColor: '#374151',
        timeVisible: true,
        secondsVisible: false,
      },
      rightPriceScale: {
        borderColor: '#374151',
      },
    });

    chartRef.current = chart;

    // Add candlestick series (v4+ API)
    const candlestickSeries = chart.addSeries(CandlestickSeries, {
      upColor: '#22c55e',
      downColor: '#ef4444',
      borderUpColor: '#22c55e',
      borderDownColor: '#ef4444',
      wickUpColor: '#22c55e',
      wickDownColor: '#ef4444',
    });

    candlestickSeriesRef.current = candlestickSeries;

    // Load initial data
    loadKlines(interval);

    // Handle resize
    const handleResize = () => {
      if (chartContainerRef.current) {
        chart.applyOptions({
          width: chartContainerRef.current.clientWidth,
        });
      }
    };

    window.addEventListener('resize', handleResize);

    return () => {
      window.removeEventListener('resize', handleResize);
      chart.remove();
      chartRef.current = null;
      candlestickSeriesRef.current = null;
    };
  }, [height, loadKlines, interval]);

  // Subscribe to real-time trades for chart updates
  useEffect(() => {
    if (!wsClient || !wsConnected || !candlestickSeriesRef.current) return;

    const handleTrade = (trade: { price: string; timestamp: number }) => {
      if (!candlestickSeriesRef.current) return;

      const price = parseFloat(trade.price);
      const time = Math.floor(trade.timestamp / 1000) as Time;

      // Update the latest candle
      candlestickSeriesRef.current.update({
        time,
        open: price,
        high: price,
        low: price,
        close: price,
      });
    };

    wsClient.subscribe(`trades:${marketId}`, handleTrade);

    return () => {
      wsClient.unsubscribe(`trades:${marketId}`, handleTrade);
    };
  }, [wsClient, wsConnected, marketId]);

  // Update chart with ticker data
  useEffect(() => {
    if (!candlestickSeriesRef.current || !ticker?.lastPrice) return;

    const price = parseFloat(ticker.lastPrice);
    const now = Math.floor(Date.now() / 1000) as Time;

    // Update the last candle
    candlestickSeriesRef.current.update({
      time: now,
      open: price,
      high: price,
      low: price,
      close: price,
    });
  }, [ticker?.lastPrice]);

  // Interval buttons
  const intervals: { label: string; value: Interval }[] = [
    { label: '1m', value: '1m' },
    { label: '5m', value: '5m' },
    { label: '15m', value: '15m' },
    { label: '30m', value: '30m' },
    { label: '1H', value: '1h' },
    { label: '4H', value: '4h' },
    { label: '1D', value: '1d' },
  ];

  const handleIntervalChange = useCallback((newInterval: Interval) => {
    setInterval(newInterval);
    loadKlines(newInterval);
  }, [loadKlines]);

  return (
    <div className="bg-dark-900 rounded-lg border border-dark-700 h-full flex flex-col">
      {/* Header */}
      <div className="flex items-center justify-between px-4 py-3 border-b border-dark-700">
        <div className="flex items-center space-x-4">
          <h3 className="text-sm font-medium text-white">{marketId}</h3>
          {wsConnected && (
            <span className="flex items-center space-x-1 text-xs text-primary-400">
              <span className="w-1.5 h-1.5 bg-primary-400 rounded-full animate-pulse" />
              <span>Live</span>
            </span>
          )}
        </div>

        {/* Interval Selector */}
        <div className="flex items-center space-x-1">
          {intervals.map(({ label, value }) => (
            <button
              key={value}
              onClick={() => handleIntervalChange(value)}
              className={`px-2 py-1 text-xs font-medium rounded transition-colors ${
                interval === value
                  ? 'bg-primary-600 text-white'
                  : 'text-dark-400 hover:text-white hover:bg-dark-700'
              }`}
            >
              {label}
            </button>
          ))}
        </div>
      </div>

      {/* Chart Container */}
      <div className="flex-1 relative">
        {isLoading && (
          <div className="absolute inset-0 flex items-center justify-center bg-dark-900/80 z-10">
            <div className="flex items-center space-x-2">
              <svg
                className="animate-spin h-5 w-5 text-primary-400"
                xmlns="http://www.w3.org/2000/svg"
                fill="none"
                viewBox="0 0 24 24"
              >
                <circle
                  className="opacity-25"
                  cx="12"
                  cy="12"
                  r="10"
                  stroke="currentColor"
                  strokeWidth="4"
                />
                <path
                  className="opacity-75"
                  fill="currentColor"
                  d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"
                />
              </svg>
              <span className="text-sm text-dark-400">Loading chart...</span>
            </div>
          </div>
        )}
        {error && (
          <div className="absolute inset-0 flex items-center justify-center bg-dark-900/80 z-10">
            <div className="text-center">
              <p className="text-sm text-danger-400">{error}</p>
              <button
                onClick={() => loadKlines(interval)}
                className="mt-2 px-3 py-1 text-xs bg-primary-600 text-white rounded hover:bg-primary-500"
              >
                Retry
              </button>
            </div>
          </div>
        )}
        <div ref={chartContainerRef} className="w-full h-full" />
      </div>

      {/* Footer - Price Info */}
      {ticker && (
        <div className="flex items-center justify-between px-4 py-2 border-t border-dark-700 text-xs">
          <div className="flex items-center space-x-4">
            <div>
              <span className="text-dark-400">O: </span>
              <span className="text-white font-mono">
                {parseFloat(ticker.lastPrice).toLocaleString()}
              </span>
            </div>
            <div>
              <span className="text-dark-400">H: </span>
              <span className="text-primary-400 font-mono">
                {parseFloat(ticker.high24h).toLocaleString()}
              </span>
            </div>
            <div>
              <span className="text-dark-400">L: </span>
              <span className="text-danger-400 font-mono">
                {parseFloat(ticker.low24h).toLocaleString()}
              </span>
            </div>
            <div>
              <span className="text-dark-400">C: </span>
              <span className="text-white font-mono">
                {parseFloat(ticker.lastPrice).toLocaleString()}
              </span>
            </div>
          </div>
          <div className="text-dark-400">
            Vol: {parseFloat(ticker.volume24h).toLocaleString()}
          </div>
        </div>
      )}
    </div>
  );
}

export default Chart;
