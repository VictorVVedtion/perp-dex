/**
 * Recent Trades Component
 * Displays real-time trade stream with animations
 */

import { useEffect, useState, useRef } from 'react';
import { useTradingStore } from '@/stores/tradingStore';

interface Trade {
  id: string;
  price: string;
  quantity: string;
  side: 'buy' | 'sell';
  timestamp: number;
}

interface RecentTradesProps {
  marketId?: string;
  maxTrades?: number;
}

// Generate mock trades for development
const generateMockTrade = (basePrice: number): Trade => {
  const side = Math.random() > 0.5 ? 'buy' : 'sell';
  const priceChange = (Math.random() - 0.5) * 20;
  const price = basePrice + priceChange;
  const quantity = (Math.random() * 0.5 + 0.01).toFixed(4);

  return {
    id: `trade-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`,
    price: price.toFixed(2),
    quantity,
    side,
    timestamp: Date.now(),
  };
};

export function RecentTrades({ marketId = 'BTC-USDC', maxTrades = 50 }: RecentTradesProps) {
  const [trades, setTrades] = useState<Trade[]>([]);
  const [newTradeId, setNewTradeId] = useState<string | null>(null);
  const containerRef = useRef<HTMLDivElement>(null);
  const { ticker, wsConnected } = useTradingStore();

  // Initialize with mock trades
  useEffect(() => {
    const basePrice = parseFloat(ticker?.lastPrice || '50000');
    const initialTrades: Trade[] = [];

    for (let i = 0; i < 20; i++) {
      const trade = generateMockTrade(basePrice);
      trade.timestamp = Date.now() - i * 1000;
      trade.id = `init-${i}`;
      initialTrades.push(trade);
    }

    setTrades(initialTrades);
  }, []);

  // Simulate real-time trades
  useEffect(() => {
    const basePrice = parseFloat(ticker?.lastPrice || '50000');

    const interval = setInterval(() => {
      const newTrade = generateMockTrade(basePrice);
      setNewTradeId(newTrade.id);

      setTrades((prev) => {
        const updated = [newTrade, ...prev];
        return updated.slice(0, maxTrades);
      });

      // Clear animation highlight after 500ms
      setTimeout(() => {
        setNewTradeId(null);
      }, 500);
    }, 1500 + Math.random() * 1000); // Random interval 1.5-2.5s

    return () => clearInterval(interval);
  }, [ticker?.lastPrice, maxTrades]);

  // Format time
  const formatTime = (timestamp: number): string => {
    const date = new Date(timestamp);
    return date.toLocaleTimeString('en-US', {
      hour12: false,
      hour: '2-digit',
      minute: '2-digit',
      second: '2-digit',
    });
  };

  // Format price with commas
  const formatPrice = (price: string): string => {
    return parseFloat(price).toLocaleString(undefined, {
      minimumFractionDigits: 2,
      maximumFractionDigits: 2,
    });
  };

  return (
    <div className="bg-dark-900 rounded-lg border border-dark-700 h-full flex flex-col">
      {/* Header */}
      <div className="flex items-center justify-between px-4 py-3 border-b border-dark-700">
        <div className="flex items-center space-x-2">
          <h3 className="text-sm font-medium text-white">Recent Trades</h3>
          {wsConnected && (
            <span className="flex items-center space-x-1 text-xs text-primary-400">
              <span className="w-1.5 h-1.5 bg-primary-400 rounded-full animate-pulse" />
              <span>Live</span>
            </span>
          )}
        </div>
        <span className="text-xs text-dark-400">{marketId}</span>
      </div>

      {/* Column Headers */}
      <div className="grid grid-cols-3 px-4 py-2 text-xs text-dark-400 border-b border-dark-700">
        <span>Price (USDC)</span>
        <span className="text-right">Size (BTC)</span>
        <span className="text-right">Time</span>
      </div>

      {/* Trades List */}
      <div ref={containerRef} className="flex-1 overflow-y-auto scrollbar-thin">
        <div className="divide-y divide-dark-800">
          {trades.map((trade) => (
            <div
              key={trade.id}
              className={`grid grid-cols-3 px-4 py-1.5 text-xs transition-colors duration-300 ${
                newTradeId === trade.id
                  ? trade.side === 'buy'
                    ? 'bg-primary-500/20'
                    : 'bg-danger-500/20'
                  : 'hover:bg-dark-800'
              }`}
            >
              <span
                className={`font-mono ${
                  trade.side === 'buy' ? 'text-primary-400' : 'text-danger-400'
                }`}
              >
                {formatPrice(trade.price)}
              </span>
              <span className="text-right text-white font-mono">{trade.quantity}</span>
              <span className="text-right text-dark-400">{formatTime(trade.timestamp)}</span>
            </div>
          ))}
        </div>
      </div>

      {/* Footer Stats */}
      <div className="px-4 py-2 border-t border-dark-700 text-xs text-dark-400">
        <div className="flex items-center justify-between">
          <span>Trades: {trades.length}</span>
          <span>
            Last: {trades[0] ? formatTime(trades[0].timestamp) : '--:--:--'}
          </span>
        </div>
      </div>
    </div>
  );
}

export default RecentTrades;
