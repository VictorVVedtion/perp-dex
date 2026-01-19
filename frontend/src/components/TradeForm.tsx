/**
 * Trade Form Component
 * Handles order creation with Keplr wallet integration
 */

import { useState, useCallback } from 'react';
import { useTradingStore, mockAccount } from '@/stores/tradingStore';
import { useWallet } from '@/hooks/useWallet';
import { OrderConfirmModal } from './OrderConfirmModal';
import BigNumber from 'bignumber.js';

// Order type definitions
type OrderSide = 'buy' | 'sell';
type OrderType = 'limit' | 'market' | 'trailing_stop';

interface TrailingStopConfig {
  trailType: 'amount' | 'percent';
  trailValue: string;
  activationPrice: string;
}

export function TradeForm() {
  const {
    orderSide,
    orderType,
    price,
    quantity,
    leverage,
    setOrderSide,
    setOrderType,
    setPrice,
    setQuantity,
    setLeverage,
    calculateMargin,
  } = useTradingStore();

  const { connected, address, signAndBroadcast } = useWallet();

  // Local state
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [showConfirmModal, setShowConfirmModal] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState<string | null>(null);

  // Trailing stop configuration
  const [trailingConfig, setTrailingConfig] = useState<TrailingStopConfig>({
    trailType: 'amount',
    trailValue: '',
    activationPrice: '',
  });

  const account = connected ? { balance: '10000.00' } : mockAccount;
  const marketId = 'BTC-USDC';

  // Calculate derived values
  const margin = calculateMargin();
  const notional = price && quantity
    ? new BigNumber(price).times(quantity).toFixed(2)
    : '0.00';

  // Validate order
  const validateOrder = useCallback((): string | null => {
    if (!connected) {
      return '请先连接钱包';
    }

    if (!quantity || parseFloat(quantity) <= 0) {
      return '请输入有效的数量';
    }

    if (orderType === 'limit' && (!price || parseFloat(price) <= 0)) {
      return '请输入有效的价格';
    }

    const marginRequired = parseFloat(margin);
    const availableBalance = parseFloat(account.balance);

    if (marginRequired > availableBalance) {
      return '保证金不足';
    }

    if (orderType === 'trailing_stop') {
      if (!trailingConfig.trailValue || parseFloat(trailingConfig.trailValue) <= 0) {
        return '请设置跟踪距离';
      }
    }

    return null;
  }, [connected, quantity, orderType, price, margin, account.balance, trailingConfig]);

  // Handle form submission
  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);
    setSuccess(null);

    const validationError = validateOrder();
    if (validationError) {
      setError(validationError);
      return;
    }

    // Show confirmation modal
    setShowConfirmModal(true);
  };

  // Confirm and submit order
  const handleConfirmOrder = async () => {
    setIsSubmitting(true);
    setError(null);

    try {
      // Build order message
      const orderMsg = {
        typeUrl: '/perpdex.orderbook.MsgPlaceOrder',
        value: {
          trader: address,
          marketId: marketId,
          side: orderSide === 'buy' ? 1 : 2,
          orderType: orderType === 'limit' ? 1 : 2,
          price: orderType === 'limit' ? price : '0',
          size: quantity,
          leverage: leverage,
          reduceOnly: false,
          postOnly: false,
          timeInForce: 1, // GTC
          triggerPrice: '0',
        },
      };

      // Add trailing stop params if applicable
      if (orderType === 'trailing_stop') {
        orderMsg.value.orderType = 7; // Trailing stop type
        Object.assign(orderMsg.value, {
          trailAmount: trailingConfig.trailType === 'amount' ? trailingConfig.trailValue : '0',
          trailPercent: trailingConfig.trailType === 'percent' ? trailingConfig.trailValue : '0',
          activationPrice: trailingConfig.activationPrice || '0',
        });
      }

      // Sign and broadcast
      const result = await signAndBroadcast([orderMsg], 'PerpDEX Order');

      // Check result
      if (result.code === 0) {
        setSuccess(`订单已提交! 交易哈希: ${result.transactionHash.slice(0, 16)}...`);
        setShowConfirmModal(false);

        // Reset form
        setQuantity('');
        if (orderType === 'limit') {
          setPrice('');
        }
      } else {
        throw new Error(result.rawLog || '交易失败');
      }
    } catch (err: any) {
      console.error('Order submission error:', err);
      setError(err.message || '订单提交失败');
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <div className="bg-dark-900 rounded-lg border border-dark-700">
      {/* Header */}
      <div className="px-4 py-3 border-b border-dark-700">
        <h3 className="text-sm font-medium text-white">Place Order</h3>
      </div>

      <form onSubmit={handleSubmit} className="p-4 space-y-4">
        {/* Buy/Sell Toggle */}
        <div className="flex rounded-lg overflow-hidden">
          <button
            type="button"
            onClick={() => setOrderSide('buy')}
            className={`flex-1 py-2 text-sm font-medium transition-colors ${
              orderSide === 'buy'
                ? 'bg-primary-600 text-white'
                : 'bg-dark-800 text-dark-300 hover:bg-dark-700'
            }`}
          >
            Long
          </button>
          <button
            type="button"
            onClick={() => setOrderSide('sell')}
            className={`flex-1 py-2 text-sm font-medium transition-colors ${
              orderSide === 'sell'
                ? 'bg-danger-600 text-white'
                : 'bg-dark-800 text-dark-300 hover:bg-dark-700'
            }`}
          >
            Short
          </button>
        </div>

        {/* Order Type */}
        <div className="flex space-x-2">
          <button
            type="button"
            onClick={() => setOrderType('limit')}
            className={`flex-1 py-1.5 text-xs font-medium rounded transition-colors ${
              orderType === 'limit'
                ? 'bg-dark-700 text-white'
                : 'text-dark-400 hover:text-white'
            }`}
          >
            Limit
          </button>
          <button
            type="button"
            onClick={() => setOrderType('market')}
            className={`flex-1 py-1.5 text-xs font-medium rounded transition-colors ${
              orderType === 'market'
                ? 'bg-dark-700 text-white'
                : 'text-dark-400 hover:text-white'
            }`}
          >
            Market
          </button>
          <button
            type="button"
            onClick={() => setOrderType('trailing_stop' as any)}
            className={`flex-1 py-1.5 text-xs font-medium rounded transition-colors ${
              orderType === 'trailing_stop'
                ? 'bg-dark-700 text-white'
                : 'text-dark-400 hover:text-white'
            }`}
          >
            追踪止损
          </button>
        </div>

        {/* Price Input (for limit orders) */}
        {orderType === 'limit' && (
          <div>
            <label className="block text-xs text-dark-400 mb-1">Price (USDC)</label>
            <div className="relative">
              <input
                type="number"
                value={price}
                onChange={(e) => setPrice(e.target.value)}
                placeholder="0.00"
                step="0.01"
                className="w-full bg-dark-800 border border-dark-600 rounded-lg px-3 py-2 text-white text-sm font-mono focus:border-primary-500 focus:outline-none"
              />
            </div>
          </div>
        )}

        {/* Trailing Stop Config */}
        {orderType === 'trailing_stop' && (
          <div className="space-y-3 p-3 bg-dark-800 rounded-lg">
            <div>
              <label className="block text-xs text-dark-400 mb-1">跟踪类型</label>
              <div className="flex space-x-2">
                <button
                  type="button"
                  onClick={() => setTrailingConfig({ ...trailingConfig, trailType: 'amount' })}
                  className={`flex-1 py-1.5 text-xs rounded ${
                    trailingConfig.trailType === 'amount'
                      ? 'bg-primary-600 text-white'
                      : 'bg-dark-700 text-dark-400'
                  }`}
                >
                  固定金额
                </button>
                <button
                  type="button"
                  onClick={() => setTrailingConfig({ ...trailingConfig, trailType: 'percent' })}
                  className={`flex-1 py-1.5 text-xs rounded ${
                    trailingConfig.trailType === 'percent'
                      ? 'bg-primary-600 text-white'
                      : 'bg-dark-700 text-dark-400'
                  }`}
                >
                  百分比
                </button>
              </div>
            </div>
            <div>
              <label className="block text-xs text-dark-400 mb-1">
                {trailingConfig.trailType === 'amount' ? '跟踪距离 ($)' : '跟踪比例 (%)'}
              </label>
              <input
                type="number"
                value={trailingConfig.trailValue}
                onChange={(e) => setTrailingConfig({ ...trailingConfig, trailValue: e.target.value })}
                placeholder={trailingConfig.trailType === 'amount' ? '100' : '5'}
                step={trailingConfig.trailType === 'amount' ? '1' : '0.1'}
                className="w-full bg-dark-700 border border-dark-600 rounded px-3 py-2 text-white text-sm font-mono focus:border-primary-500 focus:outline-none"
              />
            </div>
            <div>
              <label className="block text-xs text-dark-400 mb-1">激活价格 (可选)</label>
              <input
                type="number"
                value={trailingConfig.activationPrice}
                onChange={(e) => setTrailingConfig({ ...trailingConfig, activationPrice: e.target.value })}
                placeholder="留空则立即激活"
                className="w-full bg-dark-700 border border-dark-600 rounded px-3 py-2 text-white text-sm font-mono focus:border-primary-500 focus:outline-none"
              />
            </div>
          </div>
        )}

        {/* Size Input */}
        <div>
          <label className="block text-xs text-dark-400 mb-1">Size (BTC)</label>
          <div className="relative">
            <input
              type="number"
              value={quantity}
              onChange={(e) => setQuantity(e.target.value)}
              placeholder="0.00"
              step="0.0001"
              className="w-full bg-dark-800 border border-dark-600 rounded-lg px-3 py-2 text-white text-sm font-mono focus:border-primary-500 focus:outline-none"
            />
          </div>
        </div>

        {/* Leverage Slider */}
        <div>
          <div className="flex justify-between text-xs mb-2">
            <span className="text-dark-400">Leverage</span>
            <span className="text-white font-medium">{leverage}x</span>
          </div>
          <input
            type="range"
            min="1"
            max="50"
            value={leverage}
            onChange={(e) => setLeverage(e.target.value)}
            className="w-full h-2 bg-dark-700 rounded-lg appearance-none cursor-pointer accent-primary-500"
          />
          <div className="flex justify-between text-xs text-dark-500 mt-1">
            <span>1x</span>
            <span>10x</span>
            <span>25x</span>
            <span>50x</span>
          </div>
        </div>

        {/* Order Summary */}
        <div className="bg-dark-800 rounded-lg p-3 space-y-2 text-xs">
          <div className="flex justify-between">
            <span className="text-dark-400">Notional Value</span>
            <span className="text-white font-mono">${notional}</span>
          </div>
          <div className="flex justify-between">
            <span className="text-dark-400">Required Margin</span>
            <span className="text-white font-mono">${margin}</span>
          </div>
          <div className="flex justify-between">
            <span className="text-dark-400">Available Balance</span>
            <span className="text-white font-mono">${parseFloat(account.balance).toLocaleString()}</span>
          </div>
        </div>

        {/* Error/Success Messages */}
        {error && (
          <div className="bg-danger-900/20 border border-danger-700/50 rounded-lg p-3">
            <p className="text-xs text-danger-400">{error}</p>
          </div>
        )}
        {success && (
          <div className="bg-primary-900/20 border border-primary-700/50 rounded-lg p-3">
            <p className="text-xs text-primary-400">{success}</p>
          </div>
        )}

        {/* Submit Button */}
        <button
          type="submit"
          disabled={!connected || isSubmitting}
          className={`btn-trade w-full py-3 rounded-lg text-sm font-medium text-white transition-colors disabled:opacity-50 disabled:cursor-not-allowed ${
            orderSide === 'buy'
              ? 'bg-primary-600 hover:bg-primary-500'
              : 'bg-danger-600 hover:bg-danger-500'
          }`}
        >
          {!connected ? (
            'Connect Wallet to Trade'
          ) : isSubmitting ? (
            'Submitting...'
          ) : (
            `${orderSide === 'buy' ? 'Long' : 'Short'} BTC`
          )}
        </button>
      </form>

      {/* Confirmation Modal */}
      <OrderConfirmModal
        isOpen={showConfirmModal}
        onClose={() => setShowConfirmModal(false)}
        onConfirm={handleConfirmOrder}
        order={{
          side: orderSide as 'buy' | 'sell',
          type: orderType as 'limit' | 'market',
          price: price || '0',
          quantity: quantity || '0',
          leverage: leverage,
          marketId: marketId,
        }}
        isLoading={isSubmitting}
      />
    </div>
  );
}

export default TradeForm;
