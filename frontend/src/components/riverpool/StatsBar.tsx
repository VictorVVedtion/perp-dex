/**
 * StatsBar Component
 * Displays aggregate statistics across all pools
 */

import { useEffect, useState } from 'react';
import { useRiverpoolStore } from '@/stores/riverpoolStore';
import BigNumber from 'bignumber.js';

export default function StatsBar() {
  const { pools, fetchPools } = useRiverpoolStore();

  useEffect(() => {
    if (pools.length === 0) {
      fetchPools();
    }
  }, [pools.length, fetchPools]);

  // Calculate aggregate stats
  const totalTVL = pools.reduce(
    (sum, pool) => sum.plus(new BigNumber(pool.totalDeposits)),
    new BigNumber(0)
  );

  const totalDepositors = pools.length > 0 ? pools.length * 10 : 0; // Placeholder

  const avgNAV =
    pools.length > 0
      ? pools
          .reduce((sum, pool) => sum.plus(new BigNumber(pool.nav)), new BigNumber(0))
          .div(pools.length)
      : new BigNumber(1);

  const formatNumber = (value: BigNumber, decimals = 2) => {
    if (value.gte(1000000)) {
      return `$${value.div(1000000).toFixed(decimals)}M`;
    } else if (value.gte(1000)) {
      return `$${value.div(1000).toFixed(decimals)}K`;
    }
    return `$${value.toFixed(decimals)}`;
  };

  const stats = [
    {
      label: 'Total Value Locked',
      value: formatNumber(totalTVL),
      change: '+12.5%',
      changePositive: true,
    },
    {
      label: 'Active Pools',
      value: pools.filter((p) => p.status === 'active').length.toString(),
      change: null,
      changePositive: null,
    },
    {
      label: 'Average NAV',
      value: `$${avgNAV.toFixed(4)}`,
      change: '+0.02%',
      changePositive: true,
    },
    {
      label: 'Foundation Seats',
      value: (() => {
        const foundationPool = pools.find((p) => p.poolType === 'foundation');
        if (foundationPool && foundationPool.seatsAvailable !== undefined) {
          return `${100 - foundationPool.seatsAvailable}/100`;
        }
        return '0/100';
      })(),
      change: null,
      changePositive: null,
    },
  ];

  return (
    <div className="bg-gray-800/30 border-b border-gray-800">
      <div className="max-w-7xl mx-auto px-4 py-4">
        <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
          {stats.map((stat, index) => (
            <div key={index} className="text-center md:text-left">
              <div className="text-sm text-gray-400">{stat.label}</div>
              <div className="flex items-baseline justify-center md:justify-start gap-2 mt-1">
                <span className="text-xl font-bold text-white">{stat.value}</span>
                {stat.change && (
                  <span
                    className={`text-xs font-medium ${
                      stat.changePositive ? 'text-green-400' : 'text-red-400'
                    }`}
                  >
                    {stat.change}
                  </span>
                )}
              </div>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}
