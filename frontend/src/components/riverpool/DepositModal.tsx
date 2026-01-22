/**
 * DepositModal Component
 * Handles deposits into liquidity pools
 */

import { useState, useEffect } from 'react';
import { Pool, useRiverpoolStore } from '@/stores/riverpoolStore';
import BigNumber from 'bignumber.js';

interface DepositModalProps {
  pool: Pool;
  onClose: () => void;
}

export default function DepositModal({ pool, onClose }: DepositModalProps) {
  const {
    deposit,
    estimateDeposit,
    depositAmount,
    setDepositAmount,
    isLoading,
    error,
  } = useRiverpoolStore();

  const [estimatedShares, setEstimatedShares] = useState('0');
  const [localError, setLocalError] = useState<string | null>(null);

  // For Foundation LP, amount is fixed at $100K
  const isFoundation = pool.poolType === 'foundation';
  const minDeposit = new BigNumber(pool.minDeposit);
  const maxDeposit = pool.maxDeposit !== '0' ? new BigNumber(pool.maxDeposit) : null;

  useEffect(() => {
    if (isFoundation) {
      setDepositAmount('100000');
    }
  }, [isFoundation, setDepositAmount]);

  useEffect(() => {
    const fetchEstimate = async () => {
      if (!depositAmount || new BigNumber(depositAmount).isZero()) {
        setEstimatedShares('0');
        return;
      }

      try {
        const estimate = await estimateDeposit(pool.poolId, depositAmount);
        setEstimatedShares(estimate.shares);
      } catch (err) {
        console.error('Failed to estimate deposit:', err);
      }
    };

    const debounce = setTimeout(fetchEstimate, 300);
    return () => clearTimeout(debounce);
  }, [depositAmount, pool.poolId, estimateDeposit]);

  const handleDeposit = async () => {
    setLocalError(null);

    const amount = new BigNumber(depositAmount);

    // Validation
    if (amount.lt(minDeposit)) {
      setLocalError(`Minimum deposit is $${minDeposit.toFixed(2)}`);
      return;
    }

    if (maxDeposit && amount.gt(maxDeposit)) {
      setLocalError(`Maximum deposit is $${maxDeposit.toFixed(2)}`);
      return;
    }

    if (isFoundation && pool.seatsAvailable === 0) {
      setLocalError('No seats available');
      return;
    }

    try {
      // In a real implementation, this would use the connected wallet address
      const depositor = 'cosmos1...'; // Placeholder
      await deposit(depositor, pool.poolId, depositAmount);
      onClose();
    } catch (err) {
      setLocalError((err as Error).message);
    }
  };

  const formatNumber = (value: string) => {
    const num = new BigNumber(value);
    return num.toFormat(4);
  };

  return (
    <div className="fixed inset-0 bg-black/60 backdrop-blur-sm flex items-center justify-center z-50">
      <div className="bg-gray-800 rounded-xl border border-gray-700 w-full max-w-md mx-4 overflow-hidden">
        {/* Header */}
        <div className="flex items-center justify-between p-5 border-b border-gray-700">
          <h2 className="text-lg font-semibold text-white">
            Deposit to {pool.name}
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
          {/* Pool Info */}
          <div className="bg-gray-700/50 rounded-lg p-4 space-y-2">
            <div className="flex justify-between text-sm">
              <span className="text-gray-400">Current NAV</span>
              <span className="text-white">${formatNumber(pool.nav)}</span>
            </div>
            <div className="flex justify-between text-sm">
              <span className="text-gray-400">Total TVL</span>
              <span className="text-white">
                ${new BigNumber(pool.totalDeposits).toFormat(2)}
              </span>
            </div>
            {isFoundation && (
              <div className="flex justify-between text-sm">
                <span className="text-gray-400">Available Seats</span>
                <span className="text-white">{pool.seatsAvailable} / 100</span>
              </div>
            )}
          </div>

          {/* Amount Input */}
          <div>
            <label htmlFor="deposit-amount" className="block text-sm text-gray-400 mb-2">
              Amount (USDC)
            </label>
            <div className="relative">
              <span className="absolute left-4 top-1/2 -translate-y-1/2 text-gray-400">
                $
              </span>
              <input
                id="deposit-amount"
                type="number"
                value={depositAmount}
                onChange={(e) => setDepositAmount(e.target.value)}
                disabled={isFoundation}
                placeholder={minDeposit.toFixed(0)}
                aria-label="Amount"
                className="w-full bg-gray-700 border border-gray-600 rounded-lg py-3 pl-8 pr-4 text-white placeholder-gray-500 focus:outline-none focus:ring-2 focus:ring-blue-500 disabled:opacity-50 disabled:cursor-not-allowed"
              />
            </div>
            {isFoundation && (
              <p className="text-xs text-gray-500 mt-1">
                Foundation LP requires a fixed $100,000 deposit per seat
              </p>
            )}
          </div>

          {/* Estimated Shares */}
          <div className="bg-gray-700/50 rounded-lg p-4">
            <div className="flex justify-between items-center">
              <span className="text-sm text-gray-400">You will receive</span>
              <span className="text-lg font-semibold text-white">
                {formatNumber(estimatedShares)} shares
              </span>
            </div>
          </div>

          {/* Lock Period Warning */}
          {pool.lockPeriodDays > 0 && (
            <div className="bg-yellow-500/10 border border-yellow-500/20 rounded-lg p-4">
              <div className="flex items-start gap-3">
                <span className="text-yellow-400">⚠️</span>
                <div>
                  <div className="text-sm font-medium text-yellow-400">
                    Lock Period
                  </div>
                  <div className="text-xs text-yellow-400/80 mt-1">
                    Your deposit will be locked for {pool.lockPeriodDays} days.
                    You will not be able to withdraw during this period.
                  </div>
                </div>
              </div>
            </div>
          )}

          {/* Foundation LP Points */}
          {isFoundation && (
            <div className="bg-purple-500/10 border border-purple-500/20 rounded-lg p-4">
              <div className="flex items-center gap-3">
                <span className="text-2xl">🎁</span>
                <div>
                  <div className="text-sm font-medium text-purple-400">
                    5,000,000 Points
                  </div>
                  <div className="text-xs text-purple-400/80 mt-1">
                    You will earn 5M points for this deposit
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
            onClick={handleDeposit}
            disabled={
              isLoading ||
              !depositAmount ||
              new BigNumber(depositAmount).isZero() ||
              (isFoundation && pool.seatsAvailable === 0)
            }
            className="flex-1 py-3 px-4 bg-blue-600 hover:bg-blue-700 disabled:bg-gray-600 disabled:cursor-not-allowed text-white font-medium rounded-lg transition-colors"
          >
            {isLoading ? (
              <span className="flex items-center justify-center gap-2">
                <div className="w-4 h-4 border-2 border-white/30 border-t-white rounded-full animate-spin" />
                Processing...
              </span>
            ) : (
              'Confirm'
            )}
          </button>
        </div>
      </div>
    </div>
  );
}
