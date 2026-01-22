/**
 * PoolCard Component
 * Displays pool information with deposit/withdraw actions
 */

import { Pool } from '@/stores/riverpoolStore';
import BigNumber from 'bignumber.js';

interface PoolCardProps {
  pool: Pool;
  onDeposit: () => void;
  onWithdraw: () => void;
}

export default function PoolCard({ pool, onDeposit, onWithdraw }: PoolCardProps) {
  const formatNumber = (value: string, decimals = 2) => {
    const num = new BigNumber(value);
    if (num.gte(1000000)) {
      return `$${num.div(1000000).toFixed(2)}M`;
    } else if (num.gte(1000)) {
      return `$${num.div(1000).toFixed(2)}K`;
    }
    return `$${num.toFixed(decimals)}`;
  };

  const formatPercent = (value: string) => {
    const num = new BigNumber(value).times(100);
    return `${num.toFixed(2)}%`;
  };

  const getDDGuardColor = (level: string) => {
    switch (level) {
      case 'normal':
        return 'text-green-400 bg-green-400/10';
      case 'warning':
        return 'text-yellow-400 bg-yellow-400/10';
      case 'reduce':
        return 'text-orange-400 bg-orange-400/10';
      case 'halt':
        return 'text-red-400 bg-red-400/10';
      default:
        return 'text-gray-400 bg-gray-400/10';
    }
  };

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'active':
        return 'text-green-400';
      case 'paused':
        return 'text-yellow-400';
      case 'closed':
        return 'text-red-400';
      default:
        return 'text-gray-400';
    }
  };

  const isFoundation = pool.poolType === 'foundation';
  const tvl = new BigNumber(pool.totalDeposits);

  return (
    <div className="bg-gray-800/50 rounded-xl border border-gray-700/50 overflow-hidden hover:border-blue-500/30 transition-colors">
      {/* Header */}
      <div className="p-5 border-b border-gray-700/50">
        <div className="flex justify-between items-start">
          <div>
            <h3 className="text-lg font-semibold text-white">{pool.name}</h3>
            <p className="text-sm text-gray-400 mt-1">{pool.description}</p>
          </div>
          <div className="flex items-center gap-2">
            <span className={`text-xs font-medium ${getStatusColor(pool.status)}`}>
              {pool.status.toUpperCase()}
            </span>
            <span
              className={`px-2 py-1 rounded text-xs font-medium ${getDDGuardColor(
                pool.ddGuardLevel
              )}`}
            >
              DDGuard: {pool.ddGuardLevel}
            </span>
          </div>
        </div>
      </div>

      {/* Stats Grid */}
      <div className="p-5 grid grid-cols-2 gap-4">
        <div>
          <div className="text-sm text-gray-400">Total Value Locked</div>
          <div className="text-xl font-bold text-white mt-1">
            {formatNumber(pool.totalDeposits)}
          </div>
        </div>
        <div>
          <div className="text-sm text-gray-400">NAV</div>
          <div className="text-xl font-bold text-white mt-1">
            ${new BigNumber(pool.nav).toFixed(4)}
          </div>
        </div>
        <div>
          <div className="text-sm text-gray-400">Current Drawdown</div>
          <div
            className={`text-lg font-semibold mt-1 ${
              new BigNumber(pool.currentDrawdown).gt(0.1)
                ? 'text-red-400'
                : 'text-green-400'
            }`}
          >
            {formatPercent(pool.currentDrawdown)}
          </div>
        </div>
        <div>
          <div className="text-sm text-gray-400">High Water Mark</div>
          <div className="text-lg font-semibold text-white mt-1">
            ${new BigNumber(pool.highWaterMark).toFixed(4)}
          </div>
        </div>

        {/* Foundation LP specific: Seats */}
        {isFoundation && pool.seatsAvailable !== undefined && (
          <>
            <div>
              <div className="text-sm text-gray-400">Available Seats</div>
              <div className="text-lg font-semibold text-white mt-1">
                {pool.seatsAvailable} of 100 seats
              </div>
            </div>
            <div>
              <div className="text-sm text-gray-400">Lock Period</div>
              <div className="text-lg font-semibold text-white mt-1">
                {pool.lockPeriodDays} days lock
              </div>
            </div>
            <div>
              <div className="text-sm text-gray-400">Seat Price</div>
              <div className="text-lg font-semibold text-white mt-1">
                $100K per seat
              </div>
            </div>
          </>
        )}

        {/* Main LP specific: Redemption info */}
        {pool.poolType === 'main' && (
          <>
            <div>
              <div className="text-sm text-gray-400">Minimum Deposit</div>
              <div className="text-lg font-semibold text-white mt-1">
                ${new BigNumber(pool.minDeposit).toFixed(0)} minimum
              </div>
            </div>
            <div>
              <div className="text-sm text-gray-400">Redemption Delay</div>
              <div className="text-lg font-semibold text-white mt-1">
                T+{pool.redemptionDelayDays} ({pool.redemptionDelayDays} days)
              </div>
            </div>
          </>
        )}
      </div>

      {/* Progress bar for Foundation LP seats */}
      {isFoundation && pool.seatsAvailable !== undefined && (
        <div className="px-5 pb-4">
          <div className="h-2 bg-gray-700 rounded-full overflow-hidden">
            <div
              className="h-full bg-blue-500 rounded-full transition-all"
              style={{ width: `${((100 - pool.seatsAvailable) / 100) * 100}%` }}
            />
          </div>
          <div className="flex justify-between text-xs text-gray-400 mt-1">
            <span>{100 - pool.seatsAvailable} seats filled</span>
            <span>{pool.seatsAvailable} remaining</span>
          </div>
        </div>
      )}

      {/* Actions */}
      <div className="p-5 pt-0 flex gap-3">
        <button
          onClick={onDeposit}
          disabled={pool.status !== 'active'}
          className="flex-1 py-3 px-4 bg-blue-600 hover:bg-blue-700 disabled:bg-gray-600 disabled:cursor-not-allowed text-white font-medium rounded-lg transition-colors"
        >
          Deposit
        </button>
        <button
          onClick={onWithdraw}
          disabled={pool.status === 'closed'}
          className="flex-1 py-3 px-4 bg-gray-700 hover:bg-gray-600 disabled:bg-gray-800 disabled:cursor-not-allowed text-white font-medium rounded-lg transition-colors"
        >
          Withdraw
        </button>
      </div>

      {/* Foundation LP Points Badge */}
      {isFoundation && (
        <div className="px-5 pb-5">
          <div className="bg-gradient-to-r from-purple-500/20 to-blue-500/20 border border-purple-500/30 rounded-lg p-3 flex items-center gap-3">
            <div className="text-2xl">🎁</div>
            <div>
              <div className="text-sm font-medium text-white">5M Points per Seat</div>
              <div className="text-xs text-gray-400">Earn rewards for early liquidity</div>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
