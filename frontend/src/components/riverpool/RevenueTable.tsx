/**
 * RevenueTable Component
 * Displays revenue records and breakdown for a pool
 */

import { useState, useEffect } from 'react';
import { useRiverpoolStore } from '@/stores/riverpoolStore';
import BigNumber from 'bignumber.js';

interface RevenueTableProps {
  poolId: string;
}

type RevenueSource = 'spread' | 'funding' | 'liquidation' | 'trading' | 'fees';

interface RevenueRecord {
  recordId: string;
  poolId: string;
  source: RevenueSource;
  amount: string;
  navImpact: string;
  timestamp: number;
  blockHeight: number;
  marketId: string;
  details: string;
}

interface RevenueBreakdown {
  spread: string;
  funding: string;
  liquidation: string;
  trading: string;
  fees: string;
  total: string;
}

const sourceConfig: Record<RevenueSource, { label: string; icon: string; color: string }> = {
  spread: { label: 'Spread', icon: '📊', color: 'text-blue-400' },
  funding: { label: 'Funding', icon: '💰', color: 'text-green-400' },
  liquidation: { label: 'Liquidation', icon: '⚡', color: 'text-orange-400' },
  trading: { label: 'Trading', icon: '📈', color: 'text-purple-400' },
  fees: { label: 'Fee Rebates', icon: '🎁', color: 'text-cyan-400' },
};

type TimePeriod = '24h' | '7d' | '30d' | 'all';

export default function RevenueTable({ poolId }: RevenueTableProps) {
  const { poolStats, fetchPoolStats } = useRiverpoolStore();
  const [records, setRecords] = useState<RevenueRecord[]>([]);
  const [breakdown, setBreakdown] = useState<RevenueBreakdown | null>(null);
  const [period, setPeriod] = useState<TimePeriod>('7d');
  const [isLoading, setIsLoading] = useState(false);

  useEffect(() => {
    fetchPoolStats(poolId);
    fetchRevenueData();
  }, [poolId, period]);

  const fetchRevenueData = async () => {
    setIsLoading(true);
    try {
      // Calculate time range
      const now = Math.floor(Date.now() / 1000);
      let fromTime = 0;
      switch (period) {
        case '24h':
          fromTime = now - 24 * 60 * 60;
          break;
        case '7d':
          fromTime = now - 7 * 24 * 60 * 60;
          break;
        case '30d':
          fromTime = now - 30 * 24 * 60 * 60;
          break;
        case 'all':
          fromTime = 0;
          break;
      }

      // In a real implementation, these would be actual API calls
      // For now, using mock data structure
      const mockRecords: RevenueRecord[] = [
        {
          recordId: '1',
          poolId,
          source: 'funding',
          amount: '125.50',
          navImpact: '0.00012',
          timestamp: now - 3600,
          blockHeight: 1000000,
          marketId: 'BTC-USDC',
          details: 'Funding payment received',
        },
        {
          recordId: '2',
          poolId,
          source: 'spread',
          amount: '89.75',
          navImpact: '0.00009',
          timestamp: now - 7200,
          blockHeight: 999950,
          marketId: 'ETH-USDC',
          details: 'Market making spread earned',
        },
        {
          recordId: '3',
          poolId,
          source: 'liquidation',
          amount: '250.00',
          navImpact: '0.00025',
          timestamp: now - 10800,
          blockHeight: 999900,
          marketId: 'BTC-USDC',
          details: 'Liquidation profit',
        },
      ];

      const mockBreakdown: RevenueBreakdown = {
        spread: '1250.00',
        funding: '3500.00',
        liquidation: '2100.00',
        trading: '890.00',
        fees: '150.00',
        total: '7890.00',
      };

      setRecords(mockRecords.filter((r) => r.timestamp >= fromTime));
      setBreakdown(mockBreakdown);
    } catch (error) {
      console.error('Failed to fetch revenue data:', error);
    } finally {
      setIsLoading(false);
    }
  };

  const formatAmount = (value: string, showSign = true) => {
    const bn = new BigNumber(value);
    const formatted = bn.abs().toFormat(2);
    if (!showSign) return `$${formatted}`;
    return bn.gte(0) ? `+$${formatted}` : `-$${formatted}`;
  };

  const formatDate = (timestamp: number) => {
    return new Date(timestamp * 1000).toLocaleString('en-US', {
      month: 'short',
      day: 'numeric',
      hour: '2-digit',
      minute: '2-digit',
    });
  };

  const periods: TimePeriod[] = ['24h', '7d', '30d', 'all'];

  // Calculate breakdown percentages for chart
  const calculateBreakdownPercentages = () => {
    if (!breakdown) return [];

    const total = new BigNumber(breakdown.total);
    if (total.isZero()) return [];

    return [
      { source: 'funding' as RevenueSource, value: breakdown.funding, percent: new BigNumber(breakdown.funding).div(total).times(100).toNumber() },
      { source: 'spread' as RevenueSource, value: breakdown.spread, percent: new BigNumber(breakdown.spread).div(total).times(100).toNumber() },
      { source: 'liquidation' as RevenueSource, value: breakdown.liquidation, percent: new BigNumber(breakdown.liquidation).div(total).times(100).toNumber() },
      { source: 'trading' as RevenueSource, value: breakdown.trading, percent: new BigNumber(breakdown.trading).div(total).times(100).toNumber() },
      { source: 'fees' as RevenueSource, value: breakdown.fees, percent: new BigNumber(breakdown.fees).div(total).times(100).toNumber() },
    ].filter((item) => new BigNumber(item.value).gt(0));
  };

  const breakdownItems = calculateBreakdownPercentages();

  return (
    <div className="space-y-4">
      {/* Header with Period Selector */}
      <div className="flex items-center justify-between">
        <h3 className="text-lg font-semibold text-white">Revenue Breakdown</h3>
        <div className="flex gap-1">
          {periods.map((p) => (
            <button
              key={p}
              onClick={() => setPeriod(p)}
              className={`px-3 py-1 text-sm font-medium rounded transition-colors ${
                period === p
                  ? 'bg-blue-600 text-white'
                  : 'bg-gray-700 text-gray-400 hover:bg-gray-600 hover:text-white'
              }`}
            >
              {p}
            </button>
          ))}
        </div>
      </div>

      {/* Revenue Summary Cards */}
      {breakdown && (
        <div className="grid grid-cols-2 md:grid-cols-3 gap-3">
          <div className="bg-gray-800/50 rounded-lg p-4 col-span-2 md:col-span-1">
            <div className="text-sm text-gray-400">Total Revenue</div>
            <div className="text-2xl font-bold text-green-400">
              {formatAmount(breakdown.total, false)}
            </div>
            <div className="text-xs text-gray-500 mt-1">
              {period === 'all' ? 'All Time' : `Last ${period}`}
            </div>
          </div>

          {/* Mini breakdown cards */}
          {breakdownItems.slice(0, 2).map((item) => {
            const config = sourceConfig[item.source];
            return (
              <div key={item.source} className="bg-gray-800/50 rounded-lg p-4">
                <div className="flex items-center gap-2 text-sm text-gray-400">
                  <span>{config.icon}</span>
                  <span>{config.label}</span>
                </div>
                <div className={`text-lg font-semibold ${config.color}`}>
                  {formatAmount(item.value, false)}
                </div>
                <div className="text-xs text-gray-500">{item.percent.toFixed(1)}% of total</div>
              </div>
            );
          })}
        </div>
      )}

      {/* Revenue Breakdown Chart */}
      {breakdown && breakdownItems.length > 0 && (
        <div className="bg-gray-800/50 rounded-xl border border-gray-700/50 p-5">
          <h4 className="text-sm font-medium text-gray-400 mb-3">By Source</h4>

          {/* Horizontal stacked bar */}
          <div className="h-4 rounded-full overflow-hidden flex mb-3">
            {breakdownItems.map((item, index) => {
              const colors = ['bg-green-500', 'bg-blue-500', 'bg-orange-500', 'bg-purple-500', 'bg-cyan-500'];
              return (
                <div
                  key={item.source}
                  className={`${colors[index % colors.length]} transition-all duration-500`}
                  style={{ width: `${item.percent}%` }}
                />
              );
            })}
          </div>

          {/* Legend */}
          <div className="grid grid-cols-2 md:grid-cols-5 gap-2">
            {breakdownItems.map((item) => {
              const config = sourceConfig[item.source];
              return (
                <div key={item.source} className="flex items-center gap-2 text-sm">
                  <span>{config.icon}</span>
                  <div>
                    <div className={config.color}>{config.label}</div>
                    <div className="text-gray-500 text-xs">
                      {formatAmount(item.value, false)} ({item.percent.toFixed(1)}%)
                    </div>
                  </div>
                </div>
              );
            })}
          </div>
        </div>
      )}

      {/* Recent Revenue Records */}
      <div className="bg-gray-800/50 rounded-xl border border-gray-700/50 overflow-hidden">
        <div className="p-4 border-b border-gray-700">
          <h4 className="text-sm font-medium text-white">Recent Activity</h4>
        </div>

        {isLoading ? (
          <div className="p-8 text-center text-gray-500">Loading...</div>
        ) : records.length === 0 ? (
          <div className="p-8 text-center text-gray-500">No revenue records for this period</div>
        ) : (
          <div className="divide-y divide-gray-700/50">
            {records.map((record) => {
              const config = sourceConfig[record.source];
              const isPositive = new BigNumber(record.amount).gte(0);
              return (
                <div
                  key={record.recordId}
                  className="p-4 hover:bg-gray-700/30 transition-colors"
                >
                  <div className="flex items-center justify-between">
                    <div className="flex items-center gap-3">
                      <span className="text-xl">{config.icon}</span>
                      <div>
                        <div className="flex items-center gap-2">
                          <span className={`font-medium ${config.color}`}>
                            {config.label}
                          </span>
                          {record.marketId && (
                            <span className="text-xs bg-gray-700 px-2 py-0.5 rounded text-gray-400">
                              {record.marketId}
                            </span>
                          )}
                        </div>
                        <div className="text-xs text-gray-500 mt-0.5">
                          {record.details}
                        </div>
                      </div>
                    </div>
                    <div className="text-right">
                      <div
                        className={`font-semibold ${
                          isPositive ? 'text-green-400' : 'text-red-400'
                        }`}
                      >
                        {formatAmount(record.amount)}
                      </div>
                      <div className="text-xs text-gray-500">
                        NAV Impact: {new BigNumber(record.navImpact).times(100).toFixed(4)}%
                      </div>
                    </div>
                  </div>
                  <div className="flex items-center gap-4 mt-2 text-xs text-gray-500">
                    <span>{formatDate(record.timestamp)}</span>
                    <span>Block #{record.blockHeight.toLocaleString()}</span>
                  </div>
                </div>
              );
            })}
          </div>
        )}
      </div>

      {/* Performance Stats from Pool Stats */}
      {poolStats && (
        <div className="bg-gray-800/50 rounded-xl border border-gray-700/50 p-5">
          <h4 className="text-sm font-medium text-gray-400 mb-3">Return Performance</h4>
          <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
            <div>
              <div className="text-sm text-gray-400">1D Return</div>
              <div
                className={`text-lg font-semibold ${
                  new BigNumber(poolStats.return1d).gte(0) ? 'text-green-400' : 'text-red-400'
                }`}
              >
                {new BigNumber(poolStats.return1d).times(100).toFixed(2)}%
              </div>
            </div>
            <div>
              <div className="text-sm text-gray-400">7D Return</div>
              <div
                className={`text-lg font-semibold ${
                  new BigNumber(poolStats.return7d).gte(0) ? 'text-green-400' : 'text-red-400'
                }`}
              >
                {new BigNumber(poolStats.return7d).times(100).toFixed(2)}%
              </div>
            </div>
            <div>
              <div className="text-sm text-gray-400">30D Return</div>
              <div
                className={`text-lg font-semibold ${
                  new BigNumber(poolStats.return30d).gte(0) ? 'text-green-400' : 'text-red-400'
                }`}
              >
                {new BigNumber(poolStats.return30d).times(100).toFixed(2)}%
              </div>
            </div>
            <div>
              <div className="text-sm text-gray-400">All-Time Return</div>
              <div
                className={`text-lg font-semibold ${
                  new BigNumber(poolStats.returnAllTime).gte(0) ? 'text-green-400' : 'text-red-400'
                }`}
              >
                {new BigNumber(poolStats.returnAllTime).times(100).toFixed(2)}%
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
