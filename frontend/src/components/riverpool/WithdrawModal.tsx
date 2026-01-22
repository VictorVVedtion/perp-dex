/**
 * WithdrawModal Component
 * Handles withdrawal requests from liquidity pools
 */

import { useState, useEffect } from 'react';
import { Pool, useRiverpoolStore } from '@/stores/riverpoolStore';
import BigNumber from 'bignumber.js';

interface WithdrawModalProps {
  pool: Pool;
  onClose: () => void;
}

export default function WithdrawModal({ pool, onClose }: WithdrawModalProps) {
  const {
    requestWithdrawal,
    estimateWithdrawal,
    userPoolBalance,
    fetchUserPoolBalance,
    withdrawShares,
    setWithdrawShares,
    isLoading,
    error,
  } = useRiverpoolStore();

  const [estimation, setEstimation] = useState<{
    amount: string;
    nav: string;
    availableAt: number;
    queuePosition: string;
    mayBeProrated: boolean;
  } | null>(null);
  const [localError, setLocalError] = useState<string | null>(null);
  const [percentageSelected, setPercentageSelected] = useState<number | null>(null);

  // Placeholder user address
  const userAddress = 'cosmos1...';

  useEffect(() => {
    fetchUserPoolBalance(pool.poolId, userAddress);
  }, [pool.poolId, fetchUserPoolBalance]);

  useEffect(() => {
    const fetchEstimate = async () => {
      if (!withdrawShares || new BigNumber(withdrawShares).isZero()) {
        setEstimation(null);
        return;
      }

      try {
        const estimate = await estimateWithdrawal(pool.poolId, withdrawShares);
        setEstimation(estimate);
      } catch (err) {
        console.error('Failed to estimate withdrawal:', err);
      }
    };

    const debounce = setTimeout(fetchEstimate, 300);
    return () => clearTimeout(debounce);
  }, [withdrawShares, pool.poolId, estimateWithdrawal]);

  const handlePercentageClick = (percent: number) => {
    setPercentageSelected(percent);
    if (userPoolBalance) {
      const shares = new BigNumber(userPoolBalance.shares)
        .times(percent / 100)
        .toFixed(8);
      setWithdrawShares(shares);
    }
  };

  const handleWithdraw = async () => {
    setLocalError(null);

    if (!userPoolBalance || !userPoolBalance.canWithdraw) {
      setLocalError('Your deposit is still locked');
      return;
    }

    const shares = new BigNumber(withdrawShares);
    const available = new BigNumber(userPoolBalance.shares);

    if (shares.gt(available)) {
      setLocalError('Insufficient shares');
      return;
    }

    try {
      await requestWithdrawal(userAddress, pool.poolId, withdrawShares);
      onClose();
    } catch (err) {
      setLocalError((err as Error).message);
    }
  };

  const formatNumber = (value: string, decimals = 4) => {
    return new BigNumber(value).toFormat(decimals);
  };

  const formatDate = (timestamp: number) => {
    return new Date(timestamp * 1000).toLocaleDateString('en-US', {
      month: 'short',
      day: 'numeric',
      year: 'numeric',
    });
  };

  const percentages = [25, 50, 75, 100];

  return (
    <div className="fixed inset-0 bg-black/60 backdrop-blur-sm flex items-center justify-center z-50">
      <div className="bg-gray-800 rounded-xl border border-gray-700 w-full max-w-md mx-4 overflow-hidden">
        {/* Header */}
        <div className="flex items-center justify-between p-5 border-b border-gray-700">
          <h2 className="text-lg font-semibold text-white">
            Withdraw from {pool.name}
          </h2>
          <button
            onClick={onClose}
            className="text-gray-400 hover:text-white transition-colors"
          >
            <svg
              className="w-5 h-5"
              fill="none"
              stroke="currentColor"
              viewBox="0 0 24 24"
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M6 18L18 6M6 6l12 12"
              />
            </svg>
          </button>
        </div>

        {/* Content */}
        <div className="p-5 space-y-4">
          {/* Your Balance */}
          <div className="bg-gray-700/50 rounded-lg p-4 space-y-2">
            <div className="text-sm text-gray-400">Your Balance</div>
            {userPoolBalance ? (
              <>
                <div className="flex justify-between">
                  <span className="text-white font-semibold">
                    {formatNumber(userPoolBalance.shares)} shares
                  </span>
                  <span className="text-gray-400">
                    ≈ ${formatNumber(userPoolBalance.value, 2)}
                  </span>
                </div>
                <div className="flex justify-between text-sm">
                  <span className="text-gray-400">Unrealized P&L</span>
                  <span
                    className={
                      new BigNumber(userPoolBalance.unrealizedPnl).gte(0)
                        ? 'text-green-400'
                        : 'text-red-400'
                    }
                  >
                    ${formatNumber(userPoolBalance.unrealizedPnl, 2)} (
                    {formatNumber(userPoolBalance.pnlPercent, 2)}%)
                  </span>
                </div>
                {!userPoolBalance.canWithdraw && (
                  <div className="text-xs text-yellow-400 mt-2">
                    🔒 Locked until {formatDate(userPoolBalance.unlockAt)}
                  </div>
                )}
              </>
            ) : (
              <div className="text-gray-500">Loading...</div>
            )}
          </div>

          {/* Shares Input */}
          <div>
            <label className="block text-sm text-gray-400 mb-2">
              Shares to Withdraw
            </label>
            <input
              type="number"
              value={withdrawShares}
              onChange={(e) => {
                setWithdrawShares(e.target.value);
                setPercentageSelected(null);
              }}
              placeholder="0.0"
              className="w-full bg-gray-700 border border-gray-600 rounded-lg py-3 px-4 text-white placeholder-gray-500 focus:outline-none focus:ring-2 focus:ring-blue-500"
            />

            {/* Percentage Buttons */}
            <div className="flex gap-2 mt-2">
              {percentages.map((percent) => (
                <button
                  key={percent}
                  onClick={() => handlePercentageClick(percent)}
                  className={`flex-1 py-2 text-sm font-medium rounded-lg transition-colors ${
                    percentageSelected === percent
                      ? 'bg-blue-600 text-white'
                      : 'bg-gray-700 text-gray-400 hover:bg-gray-600 hover:text-white'
                  }`}
                >
                  {percent}%
                </button>
              ))}
            </div>
          </div>

          {/* Estimation */}
          {estimation && (
            <div className="bg-gray-700/50 rounded-lg p-4 space-y-2">
              <div className="flex justify-between text-sm">
                <span className="text-gray-400">Estimated Amount</span>
                <span className="text-white font-semibold">
                  ${formatNumber(estimation.amount, 2)}
                </span>
              </div>
              <div className="flex justify-between text-sm">
                <span className="text-gray-400">Available On</span>
                <span className="text-white">
                  {formatDate(estimation.availableAt)}
                </span>
              </div>
              <div className="flex justify-between text-sm">
                <span className="text-gray-400">Queue Position</span>
                <span className="text-white">#{estimation.queuePosition}</span>
              </div>
            </div>
          )}

          {/* Pro-rata Warning */}
          {estimation?.mayBeProrated && (
            <div className="bg-yellow-500/10 border border-yellow-500/20 rounded-lg p-4">
              <div className="flex items-start gap-3">
                <span className="text-yellow-400">⚠️</span>
                <div>
                  <div className="text-sm font-medium text-yellow-400">
                    May Be Pro-rated
                  </div>
                  <div className="text-xs text-yellow-400/80 mt-1">
                    Daily withdrawal limit ({new BigNumber(pool.dailyRedemptionLimit).times(100).toFixed(0)}%) may apply.
                    Your withdrawal might be partially filled.
                  </div>
                </div>
              </div>
            </div>
          )}

          {/* T+N Notice */}
          {pool.redemptionDelayDays > 0 && (
            <div className="bg-blue-500/10 border border-blue-500/20 rounded-lg p-4">
              <div className="flex items-start gap-3">
                <span className="text-blue-400">ℹ️</span>
                <div>
                  <div className="text-sm font-medium text-blue-400">
                    T+{pool.redemptionDelayDays} Redemption
                  </div>
                  <div className="text-xs text-blue-400/80 mt-1">
                    Your withdrawal will be available {pool.redemptionDelayDays} days after request.
                  </div>
                </div>
              </div>
            </div>
          )}

          {/* Error Message */}
          {(error || localError) && (
            <div className="bg-red-500/10 border border-red-500/20 rounded-lg p-3 text-sm text-red-400">
              {localError || error}
            </div>
          )}
        </div>

        {/* Footer */}
        <div className="p-5 pt-0 flex gap-3">
          <button
            onClick={onClose}
            className="flex-1 py-3 px-4 bg-gray-700 hover:bg-gray-600 text-white font-medium rounded-lg transition-colors"
          >
            Cancel
          </button>
          <button
            onClick={handleWithdraw}
            disabled={
              isLoading ||
              !withdrawShares ||
              new BigNumber(withdrawShares).isZero() ||
              Boolean(userPoolBalance && !userPoolBalance.canWithdraw)
            }
            className="flex-1 py-3 px-4 bg-red-600 hover:bg-red-700 disabled:bg-gray-600 disabled:cursor-not-allowed text-white font-medium rounded-lg transition-colors"
          >
            {isLoading ? (
              <span className="flex items-center justify-center gap-2">
                <div className="w-4 h-4 border-2 border-white/30 border-t-white rounded-full animate-spin" />
                Processing...
              </span>
            ) : (
              'Request Withdrawal'
            )}
          </button>
        </div>
      </div>
    </div>
  );
}
