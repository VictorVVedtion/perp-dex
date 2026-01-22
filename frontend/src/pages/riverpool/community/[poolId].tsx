/**
 * Community Pool Detail Page
 * Displays full pool information with holders, positions, and trades
 */

import { useEffect, useState } from 'react';
import { useRouter } from 'next/router';
import Head from 'next/head';
import BigNumber from 'bignumber.js';
import NAVChart from '@/components/riverpool/NAVChart';
import DDGuardIndicator from '@/components/riverpool/DDGuardIndicator';
import RevenueTable from '@/components/riverpool/RevenueTable';
import DepositModal from '@/components/riverpool/DepositModal';
import WithdrawModal from '@/components/riverpool/WithdrawModal';
import { useRiverpoolStore } from '@/stores/riverpoolStore';

type TabType = 'overview' | 'holders' | 'positions' | 'trades' | 'revenue';

export default function PoolDetailPage() {
  const router = useRouter();
  const { poolId } = router.query;

  const {
    selectedPool,
    poolStats,
    ddGuardState,
    poolHolders,
    poolPositions,
    poolTrades,
    isLoading,
    error,
    fetchPool,
    fetchPoolStats,
    fetchDDGuardState,
    fetchPoolHolders,
    fetchPoolPositions,
    fetchPoolTrades,
  } = useRiverpoolStore();

  const [activeTab, setActiveTab] = useState<TabType>('overview');
  const [showDepositModal, setShowDepositModal] = useState(false);
  const [showWithdrawModal, setShowWithdrawModal] = useState(false);

  useEffect(() => {
    if (poolId && typeof poolId === 'string') {
      fetchPool(poolId);
      fetchPoolStats(poolId);
      fetchDDGuardState(poolId);
      fetchPoolHolders(poolId);
      fetchPoolPositions(poolId);
      fetchPoolTrades(poolId);
    }
  }, [poolId, fetchPool, fetchPoolStats, fetchDDGuardState, fetchPoolHolders, fetchPoolPositions, fetchPoolTrades]);

  const formatNumber = (value: string, decimals = 2) => {
    const num = new BigNumber(value);
    if (num.gte(1000000)) return `$${num.div(1000000).toFixed(2)}M`;
    if (num.gte(1000)) return `$${num.div(1000).toFixed(2)}K`;
    return `$${num.toFixed(decimals)}`;
  };

  const formatPercent = (value: string) => {
    const num = new BigNumber(value).times(100);
    const prefix = num.gte(0) ? '+' : '';
    return `${prefix}${num.toFixed(2)}%`;
  };

  const shortenAddress = (address: string) => {
    if (!address) return '';
    return `${address.slice(0, 8)}...${address.slice(-6)}`;
  };

  const formatDate = (timestamp: number) => {
    return new Date(timestamp * 1000).toLocaleString();
  };

  const tabs: { id: TabType; label: string }[] = [
    { id: 'overview', label: 'Overview' },
    { id: 'holders', label: 'Holders' },
    { id: 'positions', label: 'Positions' },
    { id: 'trades', label: 'Trade History' },
    { id: 'revenue', label: 'Revenue' },
  ];

  if (isLoading && !selectedPool) {
    return (
      <div className="min-h-screen bg-gray-900 flex items-center justify-center">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-500" />
      </div>
    );
  }

  if (error || !selectedPool) {
    return (
      <div className="min-h-screen bg-gray-900 flex flex-col items-center justify-center">
        <h2 className="text-xl text-white mb-4">Pool not found</h2>
        <button
          onClick={() => router.back()}
          className="px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded-lg"
        >
          Go Back
        </button>
      </div>
    );
  }

  const isOwner = false; // In real app, compare with connected wallet

  return (
    <>
      <Head>
        <title>{selectedPool.name} | Community Pool</title>
        <meta name="description" content={selectedPool.description} />
      </Head>

      <div className="min-h-screen bg-gray-900 text-white">
        <div className="max-w-7xl mx-auto px-4 py-8">
          {/* Header */}
          <div className="flex flex-col lg:flex-row lg:items-start lg:justify-between gap-4 mb-8">
            <div className="flex items-start gap-4">
              <button
                onClick={() => router.back()}
                className="p-2 bg-gray-800 hover:bg-gray-700 rounded-lg"
              >
                ←
              </button>
              <div>
                <div className="flex items-center gap-3">
                  <h1 className="text-2xl font-bold">{selectedPool.name}</h1>
                  {selectedPool.isPrivate && (
                    <span className="px-2 py-1 bg-purple-500/20 text-purple-400 text-xs rounded">
                      🔒 Private
                    </span>
                  )}
                  <span
                    className={`px-2 py-1 rounded text-xs font-medium ${
                      selectedPool.status === 'active'
                        ? 'bg-green-500/20 text-green-400'
                        : selectedPool.status === 'paused'
                        ? 'bg-yellow-500/20 text-yellow-400'
                        : 'bg-red-500/20 text-red-400'
                    }`}
                  >
                    {selectedPool.status.toUpperCase()}
                  </span>
                </div>
                <p className="text-gray-400 mt-1">{selectedPool.description}</p>
                <div className="flex items-center gap-2 mt-2 text-sm text-gray-400">
                  <span>Owner:</span>
                  <span className="text-blue-400 font-mono">
                    {shortenAddress(selectedPool.owner || '')}
                  </span>
                </div>
              </div>
            </div>

            <div className="flex gap-3">
              <button
                onClick={() => setShowDepositModal(true)}
                disabled={selectedPool.status !== 'active'}
                className="px-6 py-3 bg-blue-600 hover:bg-blue-700 disabled:bg-gray-600 disabled:cursor-not-allowed text-white font-medium rounded-lg"
              >
                Deposit
              </button>
              <button
                onClick={() => setShowWithdrawModal(true)}
                className="px-6 py-3 bg-gray-700 hover:bg-gray-600 text-white font-medium rounded-lg"
              >
                Withdraw
              </button>
              {isOwner && (
                <button
                  onClick={() => router.push(`/riverpool/community/${poolId}/manage`)}
                  className="px-6 py-3 bg-purple-600 hover:bg-purple-700 text-white font-medium rounded-lg"
                >
                  Manage
                </button>
              )}
            </div>
          </div>

          {/* Stats Cards */}
          <div className="grid grid-cols-2 md:grid-cols-4 lg:grid-cols-6 gap-4 mb-8">
            <div className="bg-gray-800/50 rounded-xl p-4">
              <div className="text-sm text-gray-400">TVL</div>
              <div className="text-xl font-bold">{formatNumber(selectedPool.totalDeposits)}</div>
            </div>
            <div className="bg-gray-800/50 rounded-xl p-4">
              <div className="text-sm text-gray-400">NAV</div>
              <div className="text-xl font-bold">${new BigNumber(selectedPool.nav).toFixed(4)}</div>
            </div>
            <div className="bg-gray-800/50 rounded-xl p-4">
              <div className="text-sm text-gray-400">Holders</div>
              <div className="text-xl font-bold">{selectedPool.totalHolders || poolHolders.length}</div>
            </div>
            <div className="bg-gray-800/50 rounded-xl p-4">
              <div className="text-sm text-gray-400">Drawdown</div>
              <div className={`text-xl font-bold ${
                new BigNumber(selectedPool.currentDrawdown).gt(0.1) ? 'text-red-400' : 'text-green-400'
              }`}>
                {formatPercent(selectedPool.currentDrawdown)}
              </div>
            </div>
            <div className="bg-gray-800/50 rounded-xl p-4">
              <div className="text-sm text-gray-400">Management Fee</div>
              <div className="text-xl font-bold">
                {formatPercent(selectedPool.managementFee || '0')}/yr
              </div>
            </div>
            <div className="bg-gray-800/50 rounded-xl p-4">
              <div className="text-sm text-gray-400">Performance Fee</div>
              <div className="text-xl font-bold">
                {formatPercent(selectedPool.performanceFee || '0')}
              </div>
            </div>
          </div>

          {/* Tabs */}
          <div className="flex gap-1 bg-gray-800/50 p-1 rounded-lg mb-6 overflow-x-auto">
            {tabs.map((tab) => (
              <button
                key={tab.id}
                onClick={() => setActiveTab(tab.id)}
                className={`px-4 py-2 rounded-md text-sm font-medium whitespace-nowrap transition-colors ${
                  activeTab === tab.id
                    ? 'bg-blue-600 text-white'
                    : 'text-gray-400 hover:text-white hover:bg-gray-700/50'
                }`}
              >
                {tab.label}
              </button>
            ))}
          </div>

          {/* Tab Content */}
          <div className="bg-gray-800/30 rounded-xl p-6">
            {activeTab === 'overview' && (
              <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
                {/* NAV Chart */}
                <div>
                  <h3 className="text-lg font-semibold mb-4">NAV History</h3>
                  <NAVChart poolId={selectedPool.poolId} />
                </div>

                {/* DDGuard */}
                <div>
                  <h3 className="text-lg font-semibold mb-4">Risk Status</h3>
                  {ddGuardState ? (
                    <DDGuardIndicator state={ddGuardState} />
                  ) : (
                    <div className="text-gray-400">Loading...</div>
                  )}
                </div>

                {/* Pool Info */}
                <div className="lg:col-span-2">
                  <h3 className="text-lg font-semibold mb-4">Pool Details</h3>
                  <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
                    <div className="bg-gray-700/30 rounded-lg p-3">
                      <div className="text-sm text-gray-400">Min Deposit</div>
                      <div className="font-semibold">{formatNumber(selectedPool.minDeposit)}</div>
                    </div>
                    <div className="bg-gray-700/30 rounded-lg p-3">
                      <div className="text-sm text-gray-400">Max Deposit</div>
                      <div className="font-semibold">{formatNumber(selectedPool.maxDeposit)}</div>
                    </div>
                    <div className="bg-gray-700/30 rounded-lg p-3">
                      <div className="text-sm text-gray-400">Lock Period</div>
                      <div className="font-semibold">{selectedPool.lockPeriodDays} days</div>
                    </div>
                    <div className="bg-gray-700/30 rounded-lg p-3">
                      <div className="text-sm text-gray-400">Redemption Delay</div>
                      <div className="font-semibold">T+{selectedPool.redemptionDelayDays}</div>
                    </div>
                    <div className="bg-gray-700/30 rounded-lg p-3">
                      <div className="text-sm text-gray-400">Max Leverage</div>
                      <div className="font-semibold">{selectedPool.maxLeverage || '10'}x</div>
                    </div>
                    <div className="bg-gray-700/30 rounded-lg p-3">
                      <div className="text-sm text-gray-400">Owner Stake</div>
                      <div className="font-semibold">
                        {formatNumber(selectedPool.ownerCurrentStake || '0')}
                      </div>
                    </div>
                    <div className="bg-gray-700/30 rounded-lg p-3 col-span-2">
                      <div className="text-sm text-gray-400">Allowed Markets</div>
                      <div className="flex flex-wrap gap-1 mt-1">
                        {(selectedPool.allowedMarkets || ['BTC-USDC', 'ETH-USDC']).map((m) => (
                          <span key={m} className="px-2 py-0.5 bg-gray-600 rounded text-xs">
                            {m}
                          </span>
                        ))}
                      </div>
                    </div>
                  </div>
                </div>

                {/* Tags */}
                {selectedPool.tags && selectedPool.tags.length > 0 && (
                  <div className="lg:col-span-2">
                    <h3 className="text-lg font-semibold mb-4">Tags</h3>
                    <div className="flex flex-wrap gap-2">
                      {selectedPool.tags.map((tag) => (
                        <span
                          key={tag}
                          className="px-3 py-1 bg-gray-700/50 text-gray-300 rounded-full text-sm"
                        >
                          {tag}
                        </span>
                      ))}
                    </div>
                  </div>
                )}
              </div>
            )}

            {activeTab === 'holders' && (
              <div>
                <h3 className="text-lg font-semibold mb-4">Pool Holders</h3>
                {poolHolders.length === 0 ? (
                  <div className="text-center py-8 text-gray-400">No holders yet</div>
                ) : (
                  <div className="overflow-x-auto">
                    <table className="w-full">
                      <thead>
                        <tr className="text-left text-sm text-gray-400 border-b border-gray-700">
                          <th className="pb-3 pr-4">Address</th>
                          <th className="pb-3 pr-4">Shares</th>
                          <th className="pb-3 pr-4">Value</th>
                          <th className="pb-3 pr-4">Deposited</th>
                          <th className="pb-3">Role</th>
                        </tr>
                      </thead>
                      <tbody>
                        {poolHolders.map((holder, index) => (
                          <tr key={index} className="border-b border-gray-700/50">
                            <td className="py-3 pr-4 font-mono text-sm text-blue-400">
                              {shortenAddress(holder.address)}
                            </td>
                            <td className="py-3 pr-4">
                              {new BigNumber(holder.shares).toFixed(4)}
                            </td>
                            <td className="py-3 pr-4">{formatNumber(holder.value)}</td>
                            <td className="py-3 pr-4 text-sm text-gray-400">
                              {formatDate(holder.depositedAt)}
                            </td>
                            <td className="py-3">
                              {holder.isOwner && (
                                <span className="px-2 py-0.5 bg-purple-500/20 text-purple-400 text-xs rounded">
                                  Owner
                                </span>
                              )}
                            </td>
                          </tr>
                        ))}
                      </tbody>
                    </table>
                  </div>
                )}
              </div>
            )}

            {activeTab === 'positions' && (
              <div>
                <h3 className="text-lg font-semibold mb-4">Active Positions</h3>
                {poolPositions.length === 0 ? (
                  <div className="text-center py-8 text-gray-400">No open positions</div>
                ) : (
                  <div className="overflow-x-auto">
                    <table className="w-full">
                      <thead>
                        <tr className="text-left text-sm text-gray-400 border-b border-gray-700">
                          <th className="pb-3 pr-4">Market</th>
                          <th className="pb-3 pr-4">Side</th>
                          <th className="pb-3 pr-4">Size</th>
                          <th className="pb-3 pr-4">Entry</th>
                          <th className="pb-3 pr-4">Mark</th>
                          <th className="pb-3 pr-4">PnL</th>
                          <th className="pb-3">Leverage</th>
                        </tr>
                      </thead>
                      <tbody>
                        {poolPositions.map((pos) => (
                          <tr key={pos.positionId} className="border-b border-gray-700/50">
                            <td className="py-3 pr-4 font-medium">{pos.marketId}</td>
                            <td className="py-3 pr-4">
                              <span
                                className={`px-2 py-0.5 rounded text-xs ${
                                  pos.side === 'long'
                                    ? 'bg-green-500/20 text-green-400'
                                    : 'bg-red-500/20 text-red-400'
                                }`}
                              >
                                {pos.side.toUpperCase()}
                              </span>
                            </td>
                            <td className="py-3 pr-4">{pos.size}</td>
                            <td className="py-3 pr-4">${pos.entryPrice}</td>
                            <td className="py-3 pr-4">${pos.markPrice}</td>
                            <td
                              className={`py-3 pr-4 ${
                                parseFloat(pos.pnl) >= 0 ? 'text-green-400' : 'text-red-400'
                              }`}
                            >
                              {formatNumber(pos.pnl)} ({formatPercent(pos.pnlPercent)})
                            </td>
                            <td className="py-3">{pos.leverage}x</td>
                          </tr>
                        ))}
                      </tbody>
                    </table>
                  </div>
                )}
              </div>
            )}

            {activeTab === 'trades' && (
              <div>
                <h3 className="text-lg font-semibold mb-4">Trade History</h3>
                {poolTrades.length === 0 ? (
                  <div className="text-center py-8 text-gray-400">No trades yet</div>
                ) : (
                  <div className="overflow-x-auto">
                    <table className="w-full">
                      <thead>
                        <tr className="text-left text-sm text-gray-400 border-b border-gray-700">
                          <th className="pb-3 pr-4">Time</th>
                          <th className="pb-3 pr-4">Market</th>
                          <th className="pb-3 pr-4">Side</th>
                          <th className="pb-3 pr-4">Price</th>
                          <th className="pb-3 pr-4">Size</th>
                          <th className="pb-3 pr-4">Fee</th>
                          <th className="pb-3">PnL</th>
                        </tr>
                      </thead>
                      <tbody>
                        {poolTrades.map((trade) => (
                          <tr key={trade.tradeId} className="border-b border-gray-700/50">
                            <td className="py-3 pr-4 text-sm text-gray-400">
                              {formatDate(trade.timestamp)}
                            </td>
                            <td className="py-3 pr-4 font-medium">{trade.marketId}</td>
                            <td className="py-3 pr-4">
                              <span
                                className={`px-2 py-0.5 rounded text-xs ${
                                  trade.side === 'buy'
                                    ? 'bg-green-500/20 text-green-400'
                                    : 'bg-red-500/20 text-red-400'
                                }`}
                              >
                                {trade.side.toUpperCase()}
                              </span>
                            </td>
                            <td className="py-3 pr-4">${trade.price}</td>
                            <td className="py-3 pr-4">{trade.size}</td>
                            <td className="py-3 pr-4 text-gray-400">${trade.fee}</td>
                            <td
                              className={`py-3 ${
                                parseFloat(trade.pnl) >= 0 ? 'text-green-400' : 'text-red-400'
                              }`}
                            >
                              {formatNumber(trade.pnl)}
                            </td>
                          </tr>
                        ))}
                      </tbody>
                    </table>
                  </div>
                )}
              </div>
            )}

            {activeTab === 'revenue' && <RevenueTable poolId={selectedPool.poolId} />}
          </div>
        </div>

        {/* Modals */}
        {showDepositModal && selectedPool && (
          <DepositModal pool={selectedPool} onClose={() => setShowDepositModal(false)} />
        )}
        {showWithdrawModal && selectedPool && (
          <WithdrawModal pool={selectedPool} onClose={() => setShowWithdrawModal(false)} />
        )}
      </div>
    </>
  );
}