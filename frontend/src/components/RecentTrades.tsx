/**
 * Recent Trades Component
 * Displays real-time trade stream with animations
 * Uses Hyperliquid API when enabled, otherwise uses mock data
 */

import { useEffect, useState, useRef } from 'react';
import { useTradingStore } from '@/stores/tradingStore';
import { config } from '@/lib/config';
import { getHyperliquidClient } from '@/lib/api/hyperliquid';

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

// Note: Mock trade generation removed - using real Hyperliquid API only

export function RecentTrades({ marketId = 'BTC-USDC', maxTrades = 50 }: RecentTradesProps) {
  const [trades, setTrades] = useState<Trade[]>([]);
  const [newTradeId, setNewTradeId] = useState<string | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const containerRef = useRef<HTMLDivElement>(null);
  const { ticker, wsConnected, recentTrades } = useTradingStore();

  const useHyperliquid = config.features.useHyperliquid && !config.features.mockMode;

  // Load initial trades from Hyperliquid API
  useEffect(() => {
    if (useHyperliquid) {
      const loadTrades = async () => {
        setIsLoading(true);
        try {
          const hlClient = getHyperliquidClient();
          const hlTrades = await hlClient.getRecentTrades(marketId, maxTrades);

          setTrades(
            hlTrades.map((trade) => ({
              id: trade.id,
              price: trade.price,
              quantity: trade.quantity,
              side: trade.side,
              timestamp: trade.timestamp,
            }))
          );
        } catch (error) {
          console.error('Failed to load trades from Hyperliquid:', error);
          // Keep trades empty on error - no mock fallback
          setTrades([]);
        } finally {
          setIsLoading(false);
        }
      };

      loadTrades();
    } else {
      // Mock mode disabled - show empty state
      setIsLoading(false);
      setTrades([]);
      console.warn('RecentTrades: Mock mode is disabled, but Hyperliquid is not enabled');
    }
  }, [marketId, useHyperliquid]);

  // Handle real-time trade updates from store
  useEffect(() => {
    if (useHyperliquid && recentTrades.length > 0) {
      const storeTrades: Trade[] = recentTrades.map((trade) => ({
        id: trade.tradeId,
        price: trade.price,
        quantity: trade.quantity,
        side: trade.side,
        timestamp: trade.timestamp,
      }));

      // Check for new trade
      if (storeTrades[0]?.id !== trades[0]?.id) {
        setNewTradeId(storeTrades[0]?.id || null);

        // Clear animation highlight after 500ms
        setTimeout(() => {
          setNewTradeId(null);
        }, 500);
      }

      setTrades(storeTrades.slice(0, maxTrades));
    }
  }, [recentTrades, useHyperliquid, maxTrades]);

  // Note: Mock mode trade simulation removed - only real data from Hyperliquid WebSocket

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
    <div className="glass-panel bg-dark-900/80 backdrop-blur-md rounded-lg border border-dark-700/50 h-full flex flex-col overflow-hidden shadow-xl ring-1 ring-white/5">
      {/* Header */}
      <div className="flex items-center justify-between px-4 py-3 border-b border-dark-700/50 bg-dark-900/50">
        <div className="flex items-center space-x-2">
          <h3 className="text-sm font-medium text-white">Recent Trades</h3>
          {wsConnected && (
            <div className="flex items-center px-2 py-0.5 rounded-full bg-primary-500/10 border border-primary-500/20 shadow-[0_0_8px_rgba(var(--primary-500),0.2)]">
              <span className="relative flex h-2 w-2 mr-1.5">
                <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-primary-400 opacity-75"></span>
                <span className="relative inline-flex rounded-full h-2 w-2 bg-primary-500"></span>
              </span>
              <span className="text-[10px] font-bold text-primary-400 uppercase tracking-wider">Live</span>
            </div>
          )}
          {useHyperliquid && (
            <span className="text-[10px] font-bold text-dark-400 bg-dark-800/80 border border-dark-700 px-1.5 py-0.5 rounded backdrop-blur-sm">HL</span>
          )}
        </div>
        <span className="text-xs text-dark-400 font-mono">{marketId}</span>
      </div>

      {/* Column Headers */}
      <div className="grid grid-cols-3 px-4 py-2 text-xs text-dark-400 border-b border-dark-700/50 bg-dark-900/30 font-medium tracking-wide">
        <span>Price (USDC)</span>
        <span className="text-right">Size (BTC)</span>
        <span className="text-right">Time</span>
      </div>

      {/* Loading State */}
      {isLoading && (
        <div className="flex-1 flex items-center justify-center">
          <div className="flex items-center space-x-2">
            <svg
              className="animate-spin h-4 w-4 text-primary-400"
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
            <span className="text-xs text-dark-400">Loading trades...</span>
          </div>
        </div>
      )}

      {/* Trades List */}
      {!isLoading && (
        <div ref={containerRef} className="flex-1 overflow-y-auto scrollbar-thin">
          <div className="divide-y divide-dark-800/50">
            {trades.map((trade) => {
              const isLarge = parseFloat(trade.quantity) > 0.5; // Highlight significant trades
              return (
                <div
                  key={trade.id}
                  className={`grid grid-cols-3 px-4 py-2 text-xs transition-all duration-300 animate-slide-up ${
                    newTradeId === trade.id
                      ? trade.side === 'buy'
                        ? 'bg-primary-500/20 shadow-[inset_0_0_10px_rgba(var(--primary-500),0.1)]'
                        : 'bg-danger-500/20 shadow-[inset_0_0_10px_rgba(var(--danger-500),0.1)]'
                      : isLarge
                        ? 'bg-dark-800/40 hover:bg-dark-800/60'
                        : 'hover:bg-dark-800/30'
                  }`}
                >
                  <span
                    className={`font-mono font-medium ${
                      trade.side === 'buy' ? 'text-primary-500' : 'text-danger-500'
                    }`}
                  >
                    {formatPrice(trade.price)}
                  </span>
                  <span className={`text-right font-mono ${isLarge ? 'text-white font-bold opacity-100' : 'text-dark-300 opacity-90'}`}>
                    {trade.quantity}
                  </span>
                  <span className="text-right text-dark-500">{formatTime(trade.timestamp)}</span>
                </div>
              );
            })}
          </div>
        </div>
      )}

      {/* Footer Stats */}
      <div className="px-4 py-2 border-t border-dark-700/50 text-xs text-dark-400 bg-dark-900/30">
        <div className="flex items-center justify-between">
          <span>Trades: {trades.length}</span>
          <span>Last: {trades[0] ? formatTime(trades[0].timestamp) : '--:--:--'}</span>
        </div>
      </div>
    </div>
  );
}

export default RecentTrades;
