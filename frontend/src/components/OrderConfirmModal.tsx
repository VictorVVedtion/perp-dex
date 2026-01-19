/**
 * Order Confirmation Modal
 * Displays order details and requests user confirmation before submission
 */

import { useEffect, useCallback } from 'react';
import BigNumber from 'bignumber.js';

interface OrderDetails {
  side: 'buy' | 'sell';
  type: 'limit' | 'market';
  price: string;
  quantity: string;
  leverage: string;
  marketId: string;
}

interface OrderConfirmModalProps {
  isOpen: boolean;
  onClose: () => void;
  onConfirm: () => void;
  order: OrderDetails;
  isLoading: boolean;
}

export function OrderConfirmModal({
  isOpen,
  onClose,
  onConfirm,
  order,
  isLoading,
}: OrderConfirmModalProps) {
  // Handle escape key
  const handleEscape = useCallback(
    (e: KeyboardEvent) => {
      if (e.key === 'Escape' && !isLoading) {
        onClose();
      }
    },
    [onClose, isLoading]
  );

  useEffect(() => {
    if (isOpen) {
      document.addEventListener('keydown', handleEscape);
      document.body.style.overflow = 'hidden';
    }

    return () => {
      document.removeEventListener('keydown', handleEscape);
      document.body.style.overflow = 'unset';
    };
  }, [isOpen, handleEscape]);

  if (!isOpen) return null;

  const notional = new BigNumber(order.price || '0')
    .times(order.quantity || '0')
    .toFixed(2);

  const margin = new BigNumber(notional)
    .div(order.leverage || '1')
    .toFixed(2);

  const isBuy = order.side === 'buy';

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      {/* Backdrop */}
      <div
        className="absolute inset-0 bg-black/60 backdrop-blur-sm"
        onClick={!isLoading ? onClose : undefined}
      />

      {/* Modal */}
      <div className="relative bg-dark-800 border border-dark-600 rounded-xl shadow-2xl w-full max-w-md mx-4 animate-in fade-in zoom-in-95 duration-200">
        {/* Header */}
        <div className="flex items-center justify-between p-4 border-b border-dark-600">
          <h3 className="text-lg font-semibold text-white">确认订单</h3>
          {!isLoading && (
            <button
              onClick={onClose}
              className="text-dark-400 hover:text-white transition-colors"
            >
              <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
              </svg>
            </button>
          )}
        </div>

        {/* Content */}
        <div className="p-4 space-y-4">
          {/* Order Direction */}
          <div className="flex items-center justify-center py-2">
            <div
              className={`px-6 py-2 rounded-lg text-lg font-bold ${
                isBuy
                  ? 'bg-primary-600/20 text-primary-400'
                  : 'bg-danger-600/20 text-danger-400'
              }`}
            >
              {isBuy ? '做多 Long' : '做空 Short'} {order.marketId.split('-')[0]}
            </div>
          </div>

          {/* Order Details */}
          <div className="bg-dark-900 rounded-lg p-4 space-y-3">
            <div className="flex justify-between text-sm">
              <span className="text-dark-400">订单类型</span>
              <span className="text-white capitalize">
                {order.type === 'limit' ? '限价单' : '市价单'}
              </span>
            </div>

            {order.type === 'limit' && (
              <div className="flex justify-between text-sm">
                <span className="text-dark-400">价格</span>
                <span className="text-white font-mono">${order.price}</span>
              </div>
            )}

            <div className="flex justify-between text-sm">
              <span className="text-dark-400">数量</span>
              <span className="text-white font-mono">
                {order.quantity} {order.marketId.split('-')[0]}
              </span>
            </div>

            <div className="flex justify-between text-sm">
              <span className="text-dark-400">杠杆</span>
              <span className="text-white font-mono">{order.leverage}x</span>
            </div>

            <div className="border-t border-dark-700 pt-3 mt-3">
              <div className="flex justify-between text-sm">
                <span className="text-dark-400">名义价值</span>
                <span className="text-white font-mono">${notional}</span>
              </div>
              <div className="flex justify-between text-sm mt-2">
                <span className="text-dark-400">所需保证金</span>
                <span className="text-primary-400 font-mono font-medium">${margin}</span>
              </div>
            </div>
          </div>

          {/* Warning */}
          <div className="bg-warning-900/20 border border-warning-700/50 rounded-lg p-3">
            <div className="flex items-start space-x-2">
              <svg className="w-5 h-5 text-warning-400 flex-shrink-0 mt-0.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
              </svg>
              <p className="text-xs text-warning-300">
                永续合约交易具有高风险。请确保您了解杠杆交易的风险，并只使用您能承受损失的资金。
              </p>
            </div>
          </div>
        </div>

        {/* Footer */}
        <div className="flex space-x-3 p-4 border-t border-dark-600">
          <button
            onClick={onClose}
            disabled={isLoading}
            className="flex-1 py-3 rounded-lg text-sm font-medium text-dark-300 bg-dark-700 hover:bg-dark-600 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
          >
            取消
          </button>
          <button
            onClick={onConfirm}
            disabled={isLoading}
            className={`flex-1 py-3 rounded-lg text-sm font-medium text-white transition-colors disabled:opacity-50 disabled:cursor-not-allowed flex items-center justify-center space-x-2 ${
              isBuy
                ? 'bg-primary-600 hover:bg-primary-500'
                : 'bg-danger-600 hover:bg-danger-500'
            }`}
          >
            {isLoading ? (
              <>
                <svg
                  className="animate-spin h-4 w-4"
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
                    d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
                  />
                </svg>
                <span>提交中...</span>
              </>
            ) : (
              <span>确认 {isBuy ? '做多' : '做空'}</span>
            )}
          </button>
        </div>
      </div>
    </div>
  );
}

export default OrderConfirmModal;
