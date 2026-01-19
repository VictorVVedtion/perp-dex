/**
 * Trade History Component
 * Displays user's trade history with filtering and pagination
 */

import { useState, useEffect } from 'react';
import { useTradingStore } from '@/stores/tradingStore';

interface HistoricalTrade {
  id: string;
  orderId: string;
  marketId: string;
  side: 'long' | 'short';
  orderType: 'market' | 'limit' | 'stop_loss' | 'take_profit';
  price: string;
  quantity: string;
  value: string;
  fee: string;
  realizedPnl: string;
  timestamp: number;
  status: 'filled' | 'partial';
}

interface TradeHistoryProps {
  trader?: string;
}

// Filter options
type TimeFilter = 'all' | '1d' | '7d' | '30d' | '90d';
type SideFilter = 'all' | 'long' | 'short';

// Generate mock trade history
const generateMockHistory = (count: number): HistoricalTrade[] => {
  const trades: HistoricalTrade[] = [];
  const markets = ['BTC-USDC', 'ETH-USDC', 'SOL-USDC', 'ARB-USDC'];
  const orderTypes: HistoricalTrade['orderType'][] = ['market', 'limit', 'stop_loss', 'take_profit'];
  const basePrices: Record<string, number> = {
    'BTC-USDC': 50000,
    'ETH-USDC': 3000,
    'SOL-USDC': 100,
    'ARB-USDC': 1.5,
  };

  for (let i = 0; i < count; i++) {
    const marketId = markets[Math.floor(Math.random() * markets.length)];
    const side = Math.random() > 0.5 ? 'long' : 'short';
    const orderType = orderTypes[Math.floor(Math.random() * orderTypes.length)];
    const basePrice = basePrices[marketId];
    const priceVariation = basePrice * (0.95 + Math.random() * 0.1);
    const quantity = (Math.random() * 2 + 0.1).toFixed(4);
    const value = (priceVariation * parseFloat(quantity)).toFixed(2);
    const fee = (parseFloat(value) * 0.0005).toFixed(4);
    const pnl = ((Math.random() - 0.4) * parseFloat(value) * 0.1).toFixed(2);

    trades.push({
      id: `trade-${i}`,
      orderId: `order-${i}-${Math.random().toString(36).substr(2, 6)}`,
      marketId,
      side,
      orderType,
      price: priceVariation.toFixed(2),
      quantity,
      value,
      fee,
      realizedPnl: pnl,
      timestamp: Date.now() - i * 3600000 * (1 + Math.random() * 5),
      status: Math.random() > 0.1 ? 'filled' : 'partial',
    });
  }

  return trades.sort((a, b) => b.timestamp - a.timestamp);
};

export function TradeHistory({ trader }: TradeHistoryProps) {
  const [trades, setTrades] = useState<HistoricalTrade[]>([]);
  const [filteredTrades, setFilteredTrades] = useState<HistoricalTrade[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [timeFilter, setTimeFilter] = useState<TimeFilter>('all');
  const [sideFilter, setSideFilter] = useState<SideFilter>('all');
  const [marketFilter, setMarketFilter] = useState<string>('all');
  const [currentPage, setCurrentPage] = useState(1);
  const tradesPerPage = 10;

  const { wsConnected } = useTradingStore();

  // Load mock data
  useEffect(() => {
    setIsLoading(true);
    // Simulate API call
    setTimeout(() => {
      const mockTrades = generateMockHistory(50);
      setTrades(mockTrades);
      setIsLoading(false);
    }, 500);
  }, [trader]);

  // Apply filters
  useEffect(() => {
    let filtered = [...trades];

    // Time filter
    if (timeFilter !== 'all') {
      const now = Date.now();
      const days = parseInt(timeFilter.replace('d', ''));
      const cutoff = now - days * 24 * 60 * 60 * 1000;
      filtered = filtered.filter((t) => t.timestamp >= cutoff);
    }

    // Side filter
    if (sideFilter !== 'all') {
      filtered = filtered.filter((t) => t.side === sideFilter);
    }

    // Market filter
    if (marketFilter !== 'all') {
      filtered = filtered.filter((t) => t.marketId === marketFilter);
    }

    setFilteredTrades(filtered);
    setCurrentPage(1);
  }, [trades, timeFilter, sideFilter, marketFilter]);

  // Pagination
  const totalPages = Math.ceil(filteredTrades.length / tradesPerPage);
  const paginatedTrades = filteredTrades.slice(
    (currentPage - 1) * tradesPerPage,
    currentPage * tradesPerPage
  );

  // Format timestamp
  const formatDate = (timestamp: number): string => {
    return new Date(timestamp).toLocaleString('en-US', {
      month: 'short',
      day: 'numeric',
      hour: '2-digit',
      minute: '2-digit',
    });
  };

  // Format price with commas
  const formatPrice = (price: string): string => {
    return parseFloat(price).toLocaleString(undefined, {
      minimumFractionDigits: 2,
      maximumFractionDigits: 2,
    });
  };

  // Get unique markets
  const markets = ['all', ...Array.from(new Set(trades.map((t) => t.marketId)))];

  // Order type display
  const orderTypeLabels: Record<HistoricalTrade['orderType'], string> = {
    market: 'Market',
    limit: 'Limit',
    stop_loss: 'Stop Loss',
    take_profit: 'Take Profit',
  };

  return (
    <div className="bg-dark-900 rounded-lg border border-dark-700">
      {/* Header */}
      <div className="flex items-center justify-between px-4 py-3 border-b border-dark-700">
        <div className="flex items-center space-x-2">
          <h3 className="text-sm font-medium text-white">Trade History</h3>
          {wsConnected && (
            <span className="flex items-center space-x-1 text-xs text-primary-400">
              <span className="w-1.5 h-1.5 bg-primary-400 rounded-full animate-pulse" />
            </span>
          )}
        </div>

        {/* Filters */}
        <div className="flex items-center space-x-3">
          {/* Time Filter */}
          <select
            value={timeFilter}
            onChange={(e) => setTimeFilter(e.target.value as TimeFilter)}
            className="bg-dark-800 border border-dark-600 rounded px-2 py-1 text-xs text-white focus:outline-none focus:border-primary-500"
          >
            <option value="all">All Time</option>
            <option value="1d">Last 24h</option>
            <option value="7d">Last 7d</option>
            <option value="30d">Last 30d</option>
            <option value="90d">Last 90d</option>
          </select>

          {/* Side Filter */}
          <select
            value={sideFilter}
            onChange={(e) => setSideFilter(e.target.value as SideFilter)}
            className="bg-dark-800 border border-dark-600 rounded px-2 py-1 text-xs text-white focus:outline-none focus:border-primary-500"
          >
            <option value="all">All Sides</option>
            <option value="long">Long</option>
            <option value="short">Short</option>
          </select>

          {/* Market Filter */}
          <select
            value={marketFilter}
            onChange={(e) => setMarketFilter(e.target.value)}
            className="bg-dark-800 border border-dark-600 rounded px-2 py-1 text-xs text-white focus:outline-none focus:border-primary-500"
          >
            {markets.map((m) => (
              <option key={m} value={m}>
                {m === 'all' ? 'All Markets' : m}
              </option>
            ))}
          </select>
        </div>
      </div>

      {/* Table */}
      <div className="overflow-x-auto">
        <table className="w-full">
          <thead>
            <tr className="text-xs text-dark-400 border-b border-dark-700">
              <th className="text-left px-4 py-3">Time</th>
              <th className="text-left px-4 py-3">Market</th>
              <th className="text-left px-4 py-3">Type</th>
              <th className="text-left px-4 py-3">Side</th>
              <th className="text-right px-4 py-3">Price</th>
              <th className="text-right px-4 py-3">Size</th>
              <th className="text-right px-4 py-3">Value</th>
              <th className="text-right px-4 py-3">Fee</th>
              <th className="text-right px-4 py-3">Realized PnL</th>
            </tr>
          </thead>
          <tbody>
            {isLoading ? (
              <tr>
                <td colSpan={9} className="text-center py-8">
                  <div className="flex items-center justify-center space-x-2">
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
                    <span className="text-dark-400 text-sm">Loading history...</span>
                  </div>
                </td>
              </tr>
            ) : paginatedTrades.length === 0 ? (
              <tr>
                <td colSpan={9} className="text-center py-8 text-dark-400 text-sm">
                  No trades found
                </td>
              </tr>
            ) : (
              paginatedTrades.map((trade) => (
                <tr
                  key={trade.id}
                  className="text-xs border-b border-dark-800 hover:bg-dark-800 transition-colors"
                >
                  <td className="px-4 py-3 text-dark-300">{formatDate(trade.timestamp)}</td>
                  <td className="px-4 py-3 text-white font-medium">{trade.marketId}</td>
                  <td className="px-4 py-3 text-dark-300">{orderTypeLabels[trade.orderType]}</td>
                  <td className="px-4 py-3">
                    <span
                      className={`px-1.5 py-0.5 rounded text-xs font-medium ${
                        trade.side === 'long'
                          ? 'bg-primary-500/20 text-primary-400'
                          : 'bg-danger-500/20 text-danger-400'
                      }`}
                    >
                      {trade.side.toUpperCase()}
                    </span>
                  </td>
                  <td className="px-4 py-3 text-right text-white font-mono">
                    ${formatPrice(trade.price)}
                  </td>
                  <td className="px-4 py-3 text-right text-white font-mono">{trade.quantity}</td>
                  <td className="px-4 py-3 text-right text-dark-300 font-mono">
                    ${formatPrice(trade.value)}
                  </td>
                  <td className="px-4 py-3 text-right text-dark-400 font-mono">${trade.fee}</td>
                  <td
                    className={`px-4 py-3 text-right font-mono ${
                      parseFloat(trade.realizedPnl) >= 0 ? 'text-primary-400' : 'text-danger-400'
                    }`}
                  >
                    {parseFloat(trade.realizedPnl) >= 0 ? '+' : ''}${trade.realizedPnl}
                  </td>
                </tr>
              ))
            )}
          </tbody>
        </table>
      </div>

      {/* Pagination */}
      {totalPages > 1 && (
        <div className="flex items-center justify-between px-4 py-3 border-t border-dark-700">
          <div className="text-xs text-dark-400">
            Showing {(currentPage - 1) * tradesPerPage + 1}-
            {Math.min(currentPage * tradesPerPage, filteredTrades.length)} of{' '}
            {filteredTrades.length} trades
          </div>
          <div className="flex items-center space-x-2">
            <button
              onClick={() => setCurrentPage((p) => Math.max(1, p - 1))}
              disabled={currentPage === 1}
              className="px-2 py-1 text-xs rounded bg-dark-800 text-white disabled:opacity-50 disabled:cursor-not-allowed hover:bg-dark-700"
            >
              Previous
            </button>
            <span className="text-xs text-dark-400">
              Page {currentPage} of {totalPages}
            </span>
            <button
              onClick={() => setCurrentPage((p) => Math.min(totalPages, p + 1))}
              disabled={currentPage === totalPages}
              className="px-2 py-1 text-xs rounded bg-dark-800 text-white disabled:opacity-50 disabled:cursor-not-allowed hover:bg-dark-700"
            >
              Next
            </button>
          </div>
        </div>
      )}

      {/* Export Button */}
      <div className="px-4 py-3 border-t border-dark-700">
        <button
          className="text-xs text-primary-400 hover:text-primary-300 transition-colors"
          onClick={() => {
            // Export functionality placeholder
            console.log('Export trades:', filteredTrades);
          }}
        >
          Export to CSV
        </button>
      </div>
    </div>
  );
}

export default TradeHistory;
