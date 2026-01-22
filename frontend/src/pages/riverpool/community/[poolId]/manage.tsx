/**
 * Pool Management Page
 * Owner dashboard for managing community pool
 */

import { useEffect, useState } from 'react';
import { useRouter } from 'next/router';
import Head from 'next/head';
import BigNumber from 'bignumber.js';
import { useRiverpoolStore, InviteCode } from '@/stores/riverpoolStore';

type ManageTab = 'overview' | 'trading' | 'invites' | 'settings';

export default function ManagePoolPage() {
  const router = useRouter();
  const { poolId } = router.query;

  const {
    selectedPool,
    poolStats,
    inviteCodes,
    poolPositions,
    isLoading,
    error,
    fetchPool,
    fetchPoolStats,
    fetchInviteCodes,
    fetchPoolPositions,
    generateInviteCode,
    depositOwnerStake,
    pausePool,
    resumePool,
    closePool,
  } = useRiverpoolStore();

  const [activeTab, setActiveTab] = useState<ManageTab>('overview');
  const [newInviteMaxUses, setNewInviteMaxUses] = useState('10');
  const [newInviteExpires, setNewInviteExpires] = useState('30');
  const [stakeAmount, setStakeAmount] = useState('');
  const [showConfirmClose, setShowConfirmClose] = useState(false);

  useEffect(() => {
    if (poolId && typeof poolId === 'string') {
      fetchPool(poolId);
      fetchPoolStats(poolId);
      fetchInviteCodes(poolId);
      fetchPoolPositions(poolId);
    }
  }, [poolId, fetchPool, fetchPoolStats, fetchInviteCodes, fetchPoolPositions]);

  const formatNumber = (value: string, decimals = 2) => {
    const num = new BigNumber(value);
    if (num.gte(1000000)) return `$${num.div(1000000).toFixed(2)}M`;
    if (num.gte(1000)) return `$${num.div(1000).toFixed(2)}K`;
    return `$${num.toFixed(decimals)}`;
  };

  const formatDate = (timestamp: number) => {
    return new Date(timestamp * 1000).toLocaleDateString();
  };

  const handleGenerateInvite = async () => {
    if (!poolId || typeof poolId !== 'string') return;
    const owner = 'perpdex1owner...'; // Get from wallet
    await generateInviteCode(owner, poolId, parseInt(newInviteMaxUses), parseInt(newInviteExpires));
  };

  const handleAddStake = async () => {
    if (!poolId || typeof poolId !== 'string' || !stakeAmount) return;
    const owner = 'perpdex1owner...'; // Get from wallet
    await depositOwnerStake(owner, poolId, stakeAmount);
    setStakeAmount('');
  };

  const handlePauseResume = async () => {
    if (!poolId || typeof poolId !== 'string' || !selectedPool) return;
    const owner = 'perpdex1owner...'; // Get from wallet
    if (selectedPool.status === 'paused') {
      await resumePool(owner, poolId);
    } else {
      await pausePool(owner, poolId);
    }
  };

  const handleClosePool = async () => {
    if (!poolId || typeof poolId !== 'string') return;
    const owner = 'perpdex1owner...'; // Get from wallet
    await closePool(owner, poolId);
    router.push('/riverpool?tab=community');
  };

  const copyToClipboard = (text: string) => {
    navigator.clipboard.writeText(text);
  };

  const tabs: { id: ManageTab; label: string; icon: string }[] = [
    { id: 'overview', label: 'Overview', icon: '📊' },
    { id: 'trading', label: 'Trading', icon: '📈' },
    { id: 'invites', label: 'Invites', icon: '🎟️' },
    { id: 'settings', label: 'Settings', icon: '⚙️' },
  ];

  if (isLoading && !selectedPool) {
    return (
      <div className="min-h-screen bg-gray-900 flex items-center justify-center">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-500" />
      </div>
    );
  }

  if (!selectedPool) {
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

  return (
    <>
      <Head>
        <title>Manage {selectedPool.name} | RiverPool</title>
      </Head>

      <div className="min-h-screen bg-gray-900 text-white">
        <div className="max-w-7xl mx-auto px-4 py-8">
          {/* Header */}
          <div className="flex items-center justify-between mb-8">
            <div className="flex items-center gap-4">
              <button
                onClick={() => router.back()}
                className="p-2 bg-gray-800 hover:bg-gray-700 rounded-lg"
              >
                ←
              </button>
              <div>
                <h1 className="text-2xl font-bold">Manage: {selectedPool.name}</h1>
                <p className="text-gray-400 text-sm">Owner Dashboard</p>
              </div>
            </div>
            <span
              className={`px-3 py-1 rounded-full text-sm font-medium ${
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

          {/* Tabs */}
          <div className="flex gap-2 mb-6 bg-gray-800/50 p-1 rounded-lg">
            {tabs.map((tab) => (
              <button
                key={tab.id}
                onClick={() => setActiveTab(tab.id)}
                className={`flex-1 py-3 px-4 rounded-lg font-medium transition-colors flex items-center justify-center gap-2 ${
                  activeTab === tab.id
                    ? 'bg-purple-600 text-white'
                    : 'text-gray-400 hover:text-white hover:bg-gray-700/50'
                }`}
              >
                <span>{tab.icon}</span>
                <span>{tab.label}</span>
              </button>
            ))}
          </div>

          {error && (
            <div className="bg-red-500/10 border border-red-500/30 rounded-lg p-4 mb-6 text-red-400">
              {error}
            </div>
          )}

          {/* Content */}
          <div className="bg-gray-800/30 rounded-xl p-6">
            {/* Overview Tab */}
            {activeTab === 'overview' && (
              <div className="space-y-6">
                {/* Quick Stats */}
                <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
                  <div className="bg-gray-700/30 rounded-lg p-4">
                    <div className="text-sm text-gray-400">TVL</div>
                    <div className="text-2xl font-bold">
                      {formatNumber(selectedPool.totalDeposits)}
                    </div>
                  </div>
                  <div className="bg-gray-700/30 rounded-lg p-4">
                    <div className="text-sm text-gray-400">NAV</div>
                    <div className="text-2xl font-bold">
                      ${new BigNumber(selectedPool.nav).toFixed(4)}
                    </div>
                  </div>
                  <div className="bg-gray-700/30 rounded-lg p-4">
                    <div className="text-sm text-gray-400">Your Stake</div>
                    <div className="text-2xl font-bold">
                      {formatNumber(selectedPool.ownerCurrentStake || '0')}
                    </div>
                  </div>
                  <div className="bg-gray-700/30 rounded-lg p-4">
                    <div className="text-sm text-gray-400">Holders</div>
                    <div className="text-2xl font-bold">{selectedPool.totalHolders || 0}</div>
                  </div>
                </div>

                {/* Fee Earnings */}
                {poolStats && (
                  <div className="bg-gray-700/30 rounded-lg p-4">
                    <h3 className="font-semibold mb-4">Fee Earnings</h3>
                    <div className="grid grid-cols-3 gap-4">
                      <div>
                        <div className="text-sm text-gray-400">Total Collected</div>
                        <div className="text-xl font-bold text-green-400">
                          {formatNumber(poolStats.totalFeesCollected)}
                        </div>
                      </div>
                      <div>
                        <div className="text-sm text-gray-400">Realized PnL</div>
                        <div
                          className={`text-xl font-bold ${
                            parseFloat(poolStats.realizedPnl) >= 0 ? 'text-green-400' : 'text-red-400'
                          }`}
                        >
                          {formatNumber(poolStats.realizedPnl)}
                        </div>
                      </div>
                      <div>
                        <div className="text-sm text-gray-400">Unrealized PnL</div>
                        <div
                          className={`text-xl font-bold ${
                            parseFloat(poolStats.unrealizedPnl) >= 0
                              ? 'text-green-400'
                              : 'text-red-400'
                          }`}
                        >
                          {formatNumber(poolStats.unrealizedPnl)}
                        </div>
                      </div>
                    </div>
                  </div>
                )}

                {/* Add Stake */}
                <div className="bg-gray-700/30 rounded-lg p-4">
                  <h3 className="font-semibold mb-4">Add Owner Stake</h3>
                  <p className="text-sm text-gray-400 mb-4">
                    Maintain at least{' '}
                    {(parseFloat(selectedPool.ownerMinStake || '0.05') * 100).toFixed(0)}% stake
                    in your pool.
                  </p>
                  <div className="flex gap-3">
                    <input
                      type="number"
                      value={stakeAmount}
                      onChange={(e) => setStakeAmount(e.target.value)}
                      placeholder="Amount (USDC)"
                      className="flex-1 bg-gray-700/50 border border-gray-600 rounded-lg px-4 py-2 text-white"
                    />
                    <button
                      onClick={handleAddStake}
                      disabled={!stakeAmount || isLoading}
                      className="px-6 py-2 bg-blue-600 hover:bg-blue-700 disabled:bg-gray-600 text-white rounded-lg"
                    >
                      Add Stake
                    </button>
                  </div>
                </div>
              </div>
            )}

            {/* Trading Tab */}
            {activeTab === 'trading' && (
              <div className="space-y-6">
                <div className="bg-yellow-500/10 border border-yellow-500/30 rounded-lg p-4">
                  <h4 className="text-yellow-400 font-medium mb-2">Trading Panel</h4>
                  <p className="text-sm text-gray-300">
                    Full trading functionality will be available in the production release. You
                    will be able to execute trades on behalf of the pool within the configured
                    limits.
                  </p>
                </div>

                {/* Current Positions */}
                <div>
                  <h3 className="font-semibold mb-4">Current Positions</h3>
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
                            <th className="pb-3 pr-4">PnL</th>
                            <th className="pb-3">Actions</th>
                          </tr>
                        </thead>
                        <tbody>
                          {poolPositions.map((pos) => (
                            <tr key={pos.positionId} className="border-b border-gray-700/50">
                              <td className="py-3 pr-4">{pos.marketId}</td>
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
                              <td
                                className={`py-3 pr-4 ${
                                  parseFloat(pos.pnl) >= 0 ? 'text-green-400' : 'text-red-400'
                                }`}
                              >
                                {formatNumber(pos.pnl)}
                              </td>
                              <td className="py-3">
                                <button className="px-3 py-1 bg-gray-700 hover:bg-gray-600 text-sm rounded">
                                  Close
                                </button>
                              </td>
                            </tr>
                          ))}
                        </tbody>
                      </table>
                    </div>
                  )}
                </div>

                {/* Trading Limits */}
                <div className="bg-gray-700/30 rounded-lg p-4">
                  <h3 className="font-semibold mb-4">Trading Limits</h3>
                  <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
                    <div>
                      <div className="text-sm text-gray-400">Max Leverage</div>
                      <div className="font-semibold">{selectedPool.maxLeverage || '10'}x</div>
                    </div>
                    <div>
                      <div className="text-sm text-gray-400">DDGuard Level</div>
                      <div className="font-semibold capitalize">{selectedPool.ddGuardLevel}</div>
                    </div>
                    <div>
                      <div className="text-sm text-gray-400">Current Drawdown</div>
                      <div className="font-semibold">
                        {(parseFloat(selectedPool.currentDrawdown) * 100).toFixed(2)}%
                      </div>
                    </div>
                    <div>
                      <div className="text-sm text-gray-400">Allowed Markets</div>
                      <div className="font-semibold">
                        {(selectedPool.allowedMarkets || []).length}
                      </div>
                    </div>
                  </div>
                </div>
              </div>
            )}

            {/* Invites Tab */}
            {activeTab === 'invites' && (
              <div className="space-y-6">
                {!selectedPool.isPrivate ? (
                  <div className="bg-blue-500/10 border border-blue-500/30 rounded-lg p-4">
                    <p className="text-blue-400">
                      This is a public pool. Invite codes are only used for private pools.
                    </p>
                  </div>
                ) : (
                  <>
                    {/* Generate New Code */}
                    <div className="bg-gray-700/30 rounded-lg p-4">
                      <h3 className="font-semibold mb-4">Generate Invite Code</h3>
                      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
                        <div>
                          <label className="block text-sm text-gray-400 mb-2">Max Uses</label>
                          <input
                            type="number"
                            value={newInviteMaxUses}
                            onChange={(e) => setNewInviteMaxUses(e.target.value)}
                            min="1"
                            max="1000"
                            className="w-full bg-gray-700/50 border border-gray-600 rounded-lg px-4 py-2 text-white"
                          />
                        </div>
                        <div>
                          <label className="block text-sm text-gray-400 mb-2">
                            Expires In (days)
                          </label>
                          <input
                            type="number"
                            value={newInviteExpires}
                            onChange={(e) => setNewInviteExpires(e.target.value)}
                            min="1"
                            max="365"
                            className="w-full bg-gray-700/50 border border-gray-600 rounded-lg px-4 py-2 text-white"
                          />
                        </div>
                        <div className="flex items-end">
                          <button
                            onClick={handleGenerateInvite}
                            disabled={isLoading}
                            className="w-full px-6 py-2 bg-purple-600 hover:bg-purple-700 disabled:bg-gray-600 text-white rounded-lg"
                          >
                            Generate
                          </button>
                        </div>
                      </div>
                    </div>

                    {/* Existing Codes */}
                    <div>
                      <h3 className="font-semibold mb-4">Active Invite Codes</h3>
                      {inviteCodes.length === 0 ? (
                        <div className="text-center py-8 text-gray-400">No invite codes yet</div>
                      ) : (
                        <div className="space-y-3">
                          {inviteCodes.map((code) => (
                            <div
                              key={code.code}
                              className="bg-gray-700/30 rounded-lg p-4 flex items-center justify-between"
                            >
                              <div>
                                <div className="font-mono text-lg">{code.code}</div>
                                <div className="text-sm text-gray-400">
                                  Used: {code.usedCount}/{code.maxUses} • Expires:{' '}
                                  {formatDate(code.expiresAt)}
                                </div>
                              </div>
                              <div className="flex items-center gap-2">
                                <span
                                  className={`px-2 py-1 rounded text-xs ${
                                    code.isActive
                                      ? 'bg-green-500/20 text-green-400'
                                      : 'bg-gray-500/20 text-gray-400'
                                  }`}
                                >
                                  {code.isActive ? 'Active' : 'Inactive'}
                                </span>
                                <button
                                  onClick={() => copyToClipboard(code.code)}
                                  className="px-3 py-1 bg-gray-700 hover:bg-gray-600 text-sm rounded"
                                  aria-label={`Copy invite code ${code.code}`}
                                >
                                  Copy
                                </button>
                              </div>
                            </div>
                          ))}
                        </div>
                      )}
                    </div>
                  </>
                )}
              </div>
            )}

            {/* Settings Tab */}
            {activeTab === 'settings' && (
              <div className="space-y-6">
                {/* Pool Status */}
                <div className="bg-gray-700/30 rounded-lg p-4">
                  <h3 className="font-semibold mb-4">Pool Status</h3>
                  <div className="flex items-center justify-between">
                    <div>
                      <p className="text-gray-400">
                        {selectedPool.status === 'active'
                          ? 'Pool is accepting deposits and trading normally.'
                          : selectedPool.status === 'paused'
                          ? 'Pool is paused. No new deposits or trading allowed.'
                          : 'Pool is closed permanently.'}
                      </p>
                    </div>
                    {selectedPool.status !== 'closed' && (
                      <button
                        onClick={handlePauseResume}
                        className={`px-6 py-2 rounded-lg font-medium ${
                          selectedPool.status === 'paused'
                            ? 'bg-green-600 hover:bg-green-700 text-white'
                            : 'bg-yellow-600 hover:bg-yellow-700 text-white'
                        }`}
                      >
                        {selectedPool.status === 'paused' ? 'Resume Pool' : 'Pause Pool'}
                      </button>
                    )}
                  </div>
                </div>

                {/* Pool Info (Read-only) */}
                <div className="bg-gray-700/30 rounded-lg p-4">
                  <h3 className="font-semibold mb-4">Pool Configuration</h3>
                  <p className="text-sm text-gray-400 mb-4">
                    These settings cannot be changed after pool creation.
                  </p>
                  <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
                    <div>
                      <div className="text-sm text-gray-400">Management Fee</div>
                      <div className="font-semibold">
                        {(parseFloat(selectedPool.managementFee || '0') * 100).toFixed(1)}%/yr
                      </div>
                    </div>
                    <div>
                      <div className="text-sm text-gray-400">Performance Fee</div>
                      <div className="font-semibold">
                        {(parseFloat(selectedPool.performanceFee || '0') * 100).toFixed(0)}%
                      </div>
                    </div>
                    <div>
                      <div className="text-sm text-gray-400">Lock Period</div>
                      <div className="font-semibold">{selectedPool.lockPeriodDays} days</div>
                    </div>
                    <div>
                      <div className="text-sm text-gray-400">Redemption Delay</div>
                      <div className="font-semibold">T+{selectedPool.redemptionDelayDays}</div>
                    </div>
                  </div>
                </div>

                {/* Danger Zone */}
                <div className="bg-red-500/10 border border-red-500/30 rounded-lg p-4">
                  <h3 className="font-semibold text-red-400 mb-4">Danger Zone</h3>
                  <p className="text-sm text-gray-400 mb-4">
                    Closing a pool is permanent. All positions will be closed and funds returned
                    to holders. This action cannot be undone.
                  </p>
                  {!showConfirmClose ? (
                    <button
                      onClick={() => setShowConfirmClose(true)}
                      disabled={selectedPool.status === 'closed'}
                      className="px-6 py-2 bg-red-600 hover:bg-red-700 disabled:bg-gray-600 text-white rounded-lg"
                    >
                      Close Pool Permanently
                    </button>
                  ) : (
                    <div className="flex items-center gap-3">
                      <span className="text-red-400">Are you sure?</span>
                      <button
                        onClick={handleClosePool}
                        className="px-6 py-2 bg-red-600 hover:bg-red-700 text-white rounded-lg"
                      >
                        Yes, Close Pool
                      </button>
                      <button
                        onClick={() => setShowConfirmClose(false)}
                        className="px-6 py-2 bg-gray-700 hover:bg-gray-600 text-white rounded-lg"
                      >
                        Cancel
                      </button>
                    </div>
                  )}
                </div>
              </div>
            )}
          </div>
        </div>
      </div>
    </>
  );
}