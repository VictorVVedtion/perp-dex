/**
 * RiverPool Main Page
 * Liquidity pool management interface with tabbed navigation
 */

import { useEffect, useState } from 'react';
import Head from 'next/head';
import { useRouter } from 'next/router';
import { useRiverpoolStore, Pool } from '@/stores/riverpoolStore';
import PoolCard from '@/components/riverpool/PoolCard';
import StatsBar from '@/components/riverpool/StatsBar';
import DepositModal from '@/components/riverpool/DepositModal';
import WithdrawModal from '@/components/riverpool/WithdrawModal';
import NAVChart from '@/components/riverpool/NAVChart';
import CommunityPoolList from '@/components/riverpool/community/PoolList';

type TabType = 'foundation' | 'main' | 'community';

const tabs: { id: TabType; label: string; description: string; icon: string }[] = [
  {
    id: 'foundation',
    label: 'Foundation LP',
    description: '100 seats × $100K, 180-day lock, 5M Points/seat',
    icon: '🏛️',
  },
  {
    id: 'main',
    label: 'Main LP',
    description: '$100 min, no lock, T+4 redemption',
    icon: '🌊',
  },
  {
    id: 'community',
    label: 'Community Pools',
    description: 'User-created strategy pools',
    icon: '👥',
  },
];

export default function RiverpoolPage() {
  const router = useRouter();
  const {
    pools,
    activeTab,
    setActiveTab,
    fetchPools,
    isLoading,
    error,
    getPoolByType,
  } = useRiverpoolStore();

  const [selectedPool, setSelectedPool] = useState<Pool | null>(null);
  const [showDepositModal, setShowDepositModal] = useState(false);
  const [showWithdrawModal, setShowWithdrawModal] = useState(false);

  useEffect(() => {
    fetchPools();
  }, [fetchPools]);

  const filteredPools = getPoolByType(activeTab);

  const handleDeposit = (pool: Pool) => {
    setSelectedPool(pool);
    setShowDepositModal(true);
  };

  const handleWithdraw = (pool: Pool) => {
    setSelectedPool(pool);
    setShowWithdrawModal(true);
  };

  return (
    <>
      <Head>
        <title>RiverPool - Liquidity Pools | PerpDEX</title>
        <meta name="description" content="Provide liquidity and earn yields" />
      </Head>

      <div className="min-h-screen bg-gray-900 text-white">
        {/* Header */}
        <div className="border-b border-gray-800 bg-gray-900/95 backdrop-blur">
          <div className="max-w-7xl mx-auto px-4 py-6">
            <h1 className="text-2xl font-bold">RiverPool</h1>
            <p className="text-gray-400 mt-1">
              Provide liquidity to earn trading fees and rewards
            </p>
          </div>
        </div>

        {/* Stats Bar */}
        <StatsBar />

        {/* Tabs */}
        <div className="max-w-7xl mx-auto px-4 mt-6">
          <div className="flex items-center justify-between mb-4">
            <div
              role="tablist"
              aria-label="Pool types"
              className="flex space-x-1 bg-gray-800/50 rounded-lg p-1 flex-1"
            >
              {tabs.map((tab) => (
                <button
                  key={tab.id}
                  role="tab"
                  aria-selected={activeTab === tab.id}
                  aria-controls={`${tab.id}-panel`}
                  onClick={() => setActiveTab(tab.id)}
                  className={`flex-1 py-3 px-4 rounded-md text-sm font-medium transition-colors ${
                    activeTab === tab.id
                      ? 'bg-blue-600 text-white'
                      : 'text-gray-400 hover:text-white hover:bg-gray-700/50'
                  }`}
                >
                  <div className="flex items-center justify-center gap-2">
                    <span>{tab.icon}</span>
                    <span>{tab.label}</span>
                  </div>
                  <div className="text-xs opacity-75 mt-0.5">{tab.description}</div>
                </button>
              ))}
            </div>
            {activeTab === 'community' && (
              <button
                onClick={() => router.push('/riverpool/community/create')}
                className="ml-4 px-6 py-3 bg-gradient-to-r from-blue-600 to-purple-600 hover:from-blue-700 hover:to-purple-700 text-white font-medium rounded-lg transition-all flex items-center gap-2 whitespace-nowrap"
              >
                <span className="text-lg">+</span>
                <span>Create Pool</span>
              </button>
            )}
          </div>
        </div>

        {/* Content */}
        <div className="max-w-7xl mx-auto px-4 py-6">
          {isLoading ? (
            <div className="flex justify-center items-center h-64">
              <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-500"></div>
            </div>
          ) : error ? (
            <div className="bg-red-500/10 border border-red-500/20 rounded-lg p-4 text-red-400">
              {error}
            </div>
          ) : activeTab === 'community' ? (
            <CommunityPoolList
              onDeposit={handleDeposit}
              onWithdraw={handleWithdraw}
              onViewDetails={(pool) => router.push(`/riverpool/community/${pool.poolId}`)}
            />
          ) : filteredPools.length === 0 ? (
            <div className="text-center py-12">
              <div className="text-gray-400 mb-4">No pools available</div>
            </div>
          ) : (
            <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
              {filteredPools.map((pool) => (
                <PoolCard
                  key={pool.poolId}
                  pool={pool}
                  onDeposit={() => handleDeposit(pool)}
                  onWithdraw={() => handleWithdraw(pool)}
                />
              ))}
            </div>
          )}

          {/* NAV Chart Section */}
          {selectedPool && (
            <div className="mt-8">
              <h2 className="text-lg font-semibold mb-4">NAV History</h2>
              <NAVChart poolId={selectedPool.poolId} />
            </div>
          )}
        </div>

        {/* Modals */}
        {showDepositModal && selectedPool && (
          <DepositModal
            pool={selectedPool}
            onClose={() => {
              setShowDepositModal(false);
              setSelectedPool(null);
            }}
          />
        )}

        {showWithdrawModal && selectedPool && (
          <WithdrawModal
            pool={selectedPool}
            onClose={() => {
              setShowWithdrawModal(false);
              setSelectedPool(null);
            }}
          />
        )}
      </div>
    </>
  );
}