/**
 * Community Pool List Component
 * Displays filterable and sortable list of community pools
 */

import { useState } from 'react';
import { useRouter } from 'next/router';
import BigNumber from 'bignumber.js';
import { useRiverpoolStore, Pool } from '@/stores/riverpoolStore';

interface PoolListProps {
  onDeposit: (pool: Pool) => void;
  onWithdraw: (pool: Pool) => void;
  onViewDetails: (pool: Pool) => void;
}

const SORT_OPTIONS = [
  { value: 'tvl', label: 'TVL' },
  { value: 'nav', label: 'NAV' },
  { value: 'holders', label: 'Holders' },
  { value: 'created', label: 'Newest' },
];

const POPULAR_TAGS = ['BTC', 'ETH', 'Trend', 'Grid', 'Arbitrage', 'DeFi', 'High-Risk', 'Conservative'];

export default function CommunityPoolList({ onDeposit, onWithdraw, onViewDetails }: PoolListProps) {
  const router = useRouter();
  const {
    getFilteredCommunityPools,
    communityPoolFilter,
    setCommunityPoolFilter,
    isLoading,
  } = useRiverpoolStore();

  const [searchQuery, setSearchQuery] = useState('');

  const pools = getFilteredCommunityPools();

  // Apply search filter
  const filteredPools = searchQuery
    ? pools.filter(
        (p) =>
          p.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
          p.description?.toLowerCase().includes(searchQuery.toLowerCase()) ||
          p.owner?.toLowerCase().includes(searchQuery.toLowerCase())
      )
    : pools;

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

  const toggleTag = (tag: string) => {
    const currentTags = communityPoolFilter.tags;
    const newTags = currentTags.includes(tag)
      ? currentTags.filter((t) => t !== tag)
      : [...currentTags, tag];
    setCommunityPoolFilter({ tags: newTags });
  };

  if (isLoading) {
    return (
      <div className="flex justify-center items-center h-64">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-500" />
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Filters */}
      <div className="bg-gray-800/50 rounded-xl p-4 space-y-4">
        {/* Search and Sort Row */}
        <div className="flex flex-col md:flex-row gap-4">
          {/* Search */}
          <div className="flex-1 relative">
            <input
              type="text"
              placeholder="Search pools by name, description, or owner..."
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              className="w-full bg-gray-700/50 border border-gray-600 rounded-lg px-4 py-3 pl-10 text-white placeholder-gray-400 focus:outline-none focus:border-blue-500"
            />
            <svg
              className="absolute left-3 top-1/2 -translate-y-1/2 w-5 h-5 text-gray-400"
              fill="none"
              stroke="currentColor"
              viewBox="0 0 24 24"
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"
              />
            </svg>
          </div>

          {/* Sort */}
          <div className="flex items-center gap-2">
            <span className="text-gray-400 text-sm">Sort by:</span>
            <select
              value={communityPoolFilter.sortBy}
              onChange={(e) =>
                setCommunityPoolFilter({ sortBy: e.target.value as any })
              }
              className="bg-gray-700 border border-gray-600 rounded-lg px-3 py-2 text-white focus:outline-none focus:border-blue-500"
            >
              {SORT_OPTIONS.map((opt) => (
                <option key={opt.value} value={opt.value}>
                  {opt.label}
                </option>
              ))}
            </select>
            <button
              onClick={() =>
                setCommunityPoolFilter({
                  sortOrder: communityPoolFilter.sortOrder === 'asc' ? 'desc' : 'asc',
                })
              }
              className="p-2 bg-gray-700 border border-gray-600 rounded-lg text-gray-300 hover:text-white"
            >
              {communityPoolFilter.sortOrder === 'asc' ? '↑' : '↓'}
            </button>
          </div>
        </div>

        {/* Tags */}
        <div className="flex flex-wrap gap-2">
          {POPULAR_TAGS.map((tag) => (
            <button
              key={tag}
              onClick={() => toggleTag(tag)}
              className={`px-3 py-1 rounded-full text-sm font-medium transition-colors ${
                communityPoolFilter.tags.includes(tag)
                  ? 'bg-blue-600 text-white'
                  : 'bg-gray-700 text-gray-300 hover:bg-gray-600'
              }`}
            >
              {tag}
            </button>
          ))}
        </div>

        {/* Additional Filters */}
        <div className="flex items-center gap-4">
          <label className="flex items-center gap-2 text-sm text-gray-300 cursor-pointer">
            <input
              type="checkbox"
              checked={communityPoolFilter.showPrivate}
              onChange={(e) => setCommunityPoolFilter({ showPrivate: e.target.checked })}
              className="w-4 h-4 rounded bg-gray-700 border-gray-600 text-blue-600 focus:ring-blue-500"
            />
            Show private pools
          </label>
          <div className="flex items-center gap-2">
            <span className="text-sm text-gray-400">Min TVL:</span>
            <select
              value={communityPoolFilter.minTvl}
              onChange={(e) => setCommunityPoolFilter({ minTvl: e.target.value })}
              className="bg-gray-700 border border-gray-600 rounded px-2 py-1 text-sm text-white"
            >
              <option value="0">All</option>
              <option value="1000">$1K+</option>
              <option value="10000">$10K+</option>
              <option value="100000">$100K+</option>
              <option value="1000000">$1M+</option>
            </select>
          </div>
        </div>
      </div>

      {/* Results Count */}
      <div className="flex items-center justify-between">
        <p className="text-gray-400 text-sm">
          {filteredPools.length} pool{filteredPools.length !== 1 ? 's' : ''} found
        </p>
      </div>

      {/* Pool Cards */}
      {filteredPools.length === 0 ? (
        <div className="text-center py-12 bg-gray-800/30 rounded-xl">
          <div className="text-5xl mb-4">👥</div>
          <h3 className="text-xl font-medium text-white mb-2">No Community Pools Found</h3>
          <p className="text-gray-400 max-w-md mx-auto mb-6">
            {searchQuery || communityPoolFilter.tags.length > 0
              ? 'Try adjusting your filters to see more results.'
              : 'Be the first to create a community pool and start earning!'}
          </p>
          <button
            onClick={() => router.push('/riverpool/community/create')}
            className="px-6 py-3 bg-blue-600 hover:bg-blue-700 text-white font-medium rounded-lg transition-colors"
          >
            Create Pool
          </button>
        </div>
      ) : (
        <div className="grid grid-cols-1 lg:grid-cols-2 xl:grid-cols-3 gap-4">
          {filteredPools.map((pool) => (
            <CommunityPoolCard
              key={pool.poolId}
              pool={pool}
              onDeposit={() => onDeposit(pool)}
              onWithdraw={() => onWithdraw(pool)}
              onViewDetails={() => onViewDetails(pool)}
              formatNumber={formatNumber}
              formatPercent={formatPercent}
              shortenAddress={shortenAddress}
            />
          ))}
        </div>
      )}
    </div>
  );
}

// Community Pool Card Component
interface CommunityPoolCardProps {
  pool: Pool;
  onDeposit: () => void;
  onWithdraw: () => void;
  onViewDetails: () => void;
  formatNumber: (value: string, decimals?: number) => string;
  formatPercent: (value: string) => string;
  shortenAddress: (address: string) => string;
}

function CommunityPoolCard({
  pool,
  onDeposit,
  onWithdraw,
  onViewDetails,
  formatNumber,
  formatPercent,
  shortenAddress,
}: CommunityPoolCardProps) {
  const getStatusColor = (status: string) => {
    switch (status) {
      case 'active':
        return 'text-green-400 bg-green-400/10';
      case 'paused':
        return 'text-yellow-400 bg-yellow-400/10';
      case 'closed':
        return 'text-red-400 bg-red-400/10';
      default:
        return 'text-gray-400 bg-gray-400/10';
    }
  };

  return (
    <div className="bg-gray-800/50 rounded-xl border border-gray-700/50 overflow-hidden hover:border-blue-500/30 transition-all group">
      {/* Header */}
      <div className="p-4 border-b border-gray-700/50">
        <div className="flex items-start justify-between">
          <div className="flex-1 min-w-0">
            <div className="flex items-center gap-2">
              <h3 className="text-lg font-semibold text-white truncate">{pool.name}</h3>
              {pool.isPrivate && (
                <span className="px-2 py-0.5 bg-purple-500/20 text-purple-400 text-xs rounded">
                  🔒 Private
                </span>
              )}
            </div>
            <p className="text-sm text-gray-400 mt-1 line-clamp-2">{pool.description}</p>
          </div>
          <span className={`px-2 py-1 rounded text-xs font-medium ${getStatusColor(pool.status)}`}>
            {pool.status}
          </span>
        </div>
      </div>

      {/* Stats */}
      <div className="p-4 grid grid-cols-2 gap-3">
        <div>
          <div className="text-xs text-gray-400">TVL</div>
          <div className="text-lg font-bold text-white">{formatNumber(pool.totalDeposits)}</div>
        </div>
        <div>
          <div className="text-xs text-gray-400">NAV</div>
          <div className="text-lg font-bold text-white">${new BigNumber(pool.nav).toFixed(4)}</div>
        </div>
        <div>
          <div className="text-xs text-gray-400">Holders</div>
          <div className="text-base font-semibold text-white">{pool.totalHolders || 0}</div>
        </div>
        <div>
          <div className="text-xs text-gray-400">Max Leverage</div>
          <div className="text-base font-semibold text-white">{pool.maxLeverage || '10'}x</div>
        </div>
      </div>

      {/* Fees & Owner */}
      <div className="px-4 pb-4 space-y-2">
        <div className="flex items-center justify-between text-sm">
          <span className="text-gray-400">Management Fee</span>
          <span className="text-white">{formatPercent(pool.managementFee || '0')}/year</span>
        </div>
        <div className="flex items-center justify-between text-sm">
          <span className="text-gray-400">Performance Fee</span>
          <span className="text-white">{formatPercent(pool.performanceFee || '0')}</span>
        </div>
        <div className="flex items-center justify-between text-sm">
          <span className="text-gray-400">Owner</span>
          <span className="text-blue-400 font-mono text-xs">
            {shortenAddress(pool.owner || '')}
          </span>
        </div>
      </div>

      {/* Tags */}
      {pool.tags && pool.tags.length > 0 && (
        <div className="px-4 pb-4 flex flex-wrap gap-1">
          {pool.tags.slice(0, 3).map((tag) => (
            <span
              key={tag}
              className="px-2 py-0.5 bg-gray-700/50 text-gray-300 text-xs rounded"
            >
              {tag}
            </span>
          ))}
          {pool.tags.length > 3 && (
            <span className="px-2 py-0.5 text-gray-400 text-xs">+{pool.tags.length - 3}</span>
          )}
        </div>
      )}

      {/* Actions */}
      <div className="p-4 pt-0 flex gap-2">
        <button
          onClick={(e) => {
            e.stopPropagation();
            onDeposit();
          }}
          disabled={pool.status !== 'active'}
          className="flex-1 py-2 px-3 bg-blue-600 hover:bg-blue-700 disabled:bg-gray-600 disabled:cursor-not-allowed text-white text-sm font-medium rounded-lg transition-colors"
        >
          Deposit
        </button>
        <button
          onClick={(e) => {
            e.stopPropagation();
            onWithdraw();
          }}
          className="flex-1 py-2 px-3 bg-gray-700 hover:bg-gray-600 text-white text-sm font-medium rounded-lg transition-colors"
        >
          Withdraw
        </button>
        <button
          onClick={(e) => {
            e.stopPropagation();
            onViewDetails();
          }}
          className="py-2 px-3 bg-gray-700 hover:bg-gray-600 text-white text-sm font-medium rounded-lg transition-colors"
        >
          →
        </button>
      </div>
    </div>
  );
}
