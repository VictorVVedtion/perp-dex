/**
 * Trade Form Component
 * Handles order creation with Keplr wallet integration
 */

import { useState, useCallback } from 'react';
import { useTradingStore } from '@/stores/tradingStore';
import { useWallet } from '@/hooks/useWallet';
import { useToast } from '@/contexts/ToastContext';
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

  const { connected, address, isMockMode, signAndBroadcast } = useWallet();
  const { showToast } = useToast();

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

  // Use account from store or default for demo
  const { account: storeAccount } = useTradingStore();
  const account = storeAccount || { balance: connected ? '10000.00' : '0.00', trader: '', lockedMargin: '0', totalEquity: '0' };
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

  // Handle percent click
  const handlePercentClick = (percent: number) => {
    const priceVal = parseFloat(price);
    const balanceVal = parseFloat(account.balance);
    if (balanceVal <= 0) return;

    // Use price if available for calculation
    const refPrice = priceVal > 0 ? priceVal : 0;
    if (refPrice <= 0) return;

    const maxMargin = balanceVal * 0.99; // 99% usage
    const targetNotional = maxMargin * percent * parseFloat(leverage);
    const targetQty = targetNotional / refPrice;

    setQuantity(targetQty.toFixed(4));
  };

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
        const txHashShort = result.transactionHash.slice(0, 16);
        setSuccess(`订单已提交! 交易哈希: ${txHashShort}...`);
        setShowConfirmModal(false);

        // Show success toast
        showToast({
          type: 'success',
          title: isMockMode ? '模拟订单已提交' : '订单已提交',
          message: `TxHash: ${txHashShort}...`,
        });

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
      const errorMsg = err.message || '订单提交失败';
      setError(errorMsg);

      // Show error toast
      showToast({
        type: 'error',
        title: '订单提交失败',
        message: errorMsg,
      });
    } finally {
      setIsSubmitting(false);
    }
  };

  const isBuy = orderSide === 'buy';

  return (
    <div className="relative overflow-hidden backdrop-blur-xl bg-dark-900/80 rounded-xl border border-dark-700/50 shadow-2xl transition-all duration-300 hover:shadow-glow-sm">
      {/* Background decoration */}
      <div className={`absolute top-0 right-0 -mt-10 -mr-10 w-32 h-32 rounded-full blur-3xl opacity-10 transition-colors duration-500 ${isBuy ? 'bg-primary-500' : 'bg-danger-500'}`}></div>

      {/* Header */}
      <div className="px-5 py-4 border-b border-dark-700/50 flex items-center justify-between relative z-10">
        <h3 className="text-base font-semibold text-white tracking-wide flex items-center gap-2">
           Place Order
           {isMockMode && <span className="text-[10px] px-1.5 py-0.5 rounded bg-blue-500/20 text-blue-400 border border-blue-500/30">MOCK</span>}
        </h3>
        <div className="flex items-center space-x-2 text-xs text-dark-400">
             <span>Bal:</span>
             <span className="text-white font-mono">{parseFloat(account.balance).toLocaleString()}</span>
        </div>
      </div>

      <form onSubmit={handleSubmit} className="p-5 space-y-6 relative z-10">
        {/* Buy/Sell Toggle */}
        <div className="relative flex bg-dark-950/50 p-1 rounded-lg border border-dark-700/50">
           <div
             className={`absolute inset-y-1 rounded-md transition-all duration-300 ease-out shadow-lg ${
               isBuy
                 ? 'bg-primary-500 left-1 right-1/2 mr-0.5'
                 : 'bg-danger-500 left-1/2 right-1 ml-0.5'
             }`}
           />
          <button
            type="button"
            onClick={() => setOrderSide('buy')}
            className={`flex-1 relative z-10 py-2.5 text-sm font-bold transition-colors duration-200 ${
              isBuy ? 'text-white' : 'text-dark-400 hover:text-white'
            }`}
          >
            Long
          </button>
          <button
            type="button"
            onClick={() => setOrderSide('sell')}
            className={`flex-1 relative z-10 py-2.5 text-sm font-bold transition-colors duration-200 ${
              !isBuy ? 'text-white' : 'text-dark-400 hover:text-white'
            }`}
          >
            Short
          </button>
        </div>

        {/* Order Type Tabs */}
        <div className="flex border-b border-dark-700/50 pb-1">
          {['limit', 'market', 'trailing_stop'].map((type) => (
             <button
                key={type}
                type="button"
                onClick={() => setOrderType(type as any)}
                className={`flex-1 pb-2 text-xs font-medium transition-all duration-200 relative ${
                    orderType === type ? 'text-white' : 'text-dark-400 hover:text-dark-200'
                }`}
             >
                {type === 'trailing_stop' ? 'Trailing' : type.charAt(0).toUpperCase() + type.slice(1)}
                {/* Active Indicator */}
                <span className={`absolute bottom-[-5px] left-0 w-full h-[2px] transform transition-transform duration-300 ${
                    orderType === type ? `scale-x-100 ${isBuy ? 'bg-primary-500' : 'bg-danger-500'}` : 'scale-x-0'
                }`} />
             </button>
          ))}
        </div>

        {/* Inputs Section */}
        <div className="space-y-4">
             {/* Price Input */}
             {orderType === 'limit' && (
                <div className="group">
                    <div className="flex justify-between text-xs mb-1.5 px-1">
                        <span className="text-dark-400">Price</span>
                        <span className="text-dark-500">USDC</span>
                    </div>
                    <div className="relative">
                        <input
                            type="number"
                            value={price}
                            onChange={(e) => setPrice(e.target.value)}
                            placeholder="0.00"
                            step="0.01"
                            className={`w-full bg-dark-800/50 border border-dark-600 rounded-lg px-4 py-3 text-white text-sm font-mono transition-all duration-200 focus:outline-none focus:ring-2 focus:ring-opacity-50 ${isBuy ? 'focus:border-primary-500 focus:ring-primary-500/20' : 'focus:border-danger-500 focus:ring-danger-500/20'}`}
                        />
                    </div>
                </div>
             )}

            {/* Trailing Config */}
            {orderType === 'trailing_stop' && (
               <div className="space-y-3 p-4 bg-dark-800/30 rounded-lg border border-dark-700/30">
                  <div className="grid grid-cols-2 gap-2">
                       <button type="button" onClick={() => setTrailingConfig({ ...trailingConfig, trailType: 'amount' })} className={`py-1.5 text-xs rounded border transition-colors ${trailingConfig.trailType === 'amount' ? 'bg-primary-500/10 border-primary-500 text-primary-500' : 'border-dark-600 text-dark-400 hover:border-dark-500'}`}>Amount</button>
                       <button type="button" onClick={() => setTrailingConfig({ ...trailingConfig, trailType: 'percent' })} className={`py-1.5 text-xs rounded border transition-colors ${trailingConfig.trailType === 'percent' ? 'bg-primary-500/10 border-primary-500 text-primary-500' : 'border-dark-600 text-dark-400 hover:border-dark-500'}`}>Percent</button>
                  </div>
                  <input
                     type="number"
                     placeholder={trailingConfig.trailType === 'amount' ? "Distance ($)" : "Rate (%)"}
                     value={trailingConfig.trailValue}
                     onChange={(e) => setTrailingConfig({ ...trailingConfig, trailValue: e.target.value })}
                     className="w-full bg-dark-900/50 border border-dark-600 rounded px-3 py-2 text-white text-sm focus:border-primary-500 focus:outline-none font-mono"
                  />
                  <input
                     type="number"
                     placeholder="Activation Price (Optional)"
                     value={trailingConfig.activationPrice}
                     onChange={(e) => setTrailingConfig({ ...trailingConfig, activationPrice: e.target.value })}
                     className="w-full bg-dark-900/50 border border-dark-600 rounded px-3 py-2 text-white text-sm focus:border-primary-500 focus:outline-none font-mono"
                  />
               </div>
            )}

             {/* Quantity Input */}
             <div className="group">
                 <div className="flex justify-between text-xs mb-1.5 px-1">
                     <span className="text-dark-400">Size</span>
                     <span className="text-dark-500">BTC</span>
                 </div>
                 <div className="relative">
                     <input
                         type="number"
                         value={quantity}
                         onChange={(e) => setQuantity(e.target.value)}
                         placeholder="0.00"
                         step="0.0001"
                         className={`w-full bg-dark-800/50 border border-dark-600 rounded-lg px-4 py-3 text-white text-sm font-mono transition-all duration-200 focus:outline-none focus:ring-2 focus:ring-opacity-50 ${isBuy ? 'focus:border-primary-500 focus:ring-primary-500/20' : 'focus:border-danger-500 focus:ring-danger-500/20'}`}
                     />
                 </div>
                 {/* Percentage Buttons */}
                 <div className="flex space-x-2 mt-2">
                    {[0.25, 0.50, 0.75, 1.0].map((pct) => (
                        <button
                           key={pct}
                           type="button"
                           onClick={() => handlePercentClick(pct)}
                           className="flex-1 py-1 text-[10px] font-medium bg-dark-800 border border-dark-700 hover:bg-dark-700 text-dark-400 hover:text-white rounded transition-colors"
                        >
                           {pct * 100}%
                        </button>
                    ))}
                 </div>
             </div>

             {/* Leverage Slider */}
             <div className="space-y-3 pt-2">
                 <div className="flex justify-between items-center">
                     <span className="text-xs text-dark-400">Leverage</span>
                     <span className={`text-xs font-bold px-2 py-0.5 rounded ${isBuy ? 'bg-primary-500/10 text-primary-500' : 'bg-danger-500/10 text-danger-500'}`}>
                         {leverage}x
                     </span>
                 </div>
                 <div className="relative h-6 flex items-center">
                     <input
                         type="range"
                         min="1"
                         max="50"
                         value={leverage}
                         onChange={(e) => setLeverage(e.target.value)}
                         className="w-full h-1.5 rounded-lg appearance-none cursor-pointer z-10"
                         style={{
                             background: `linear-gradient(to right, ${isBuy ? '#00C896' : '#FF4D4D'} 0%, ${isBuy ? '#00C896' : '#FF4D4D'} ${(parseInt(leverage)/50)*100}%, #334155 ${(parseInt(leverage)/50)*100}%, #334155 100%)`
                         }}
                     />
                 </div>
                 <div className="flex justify-between text-[10px] text-dark-500 font-mono">
                     <span>1x</span>
                     <span>10x</span>
                     <span>25x</span>
                     <span>50x</span>
                 </div>
             </div>
        </div>

        {/* Order Summary */}
        <div className="bg-dark-800/40 backdrop-blur-sm rounded-lg p-4 space-y-3 border border-dark-700/30 text-xs shadow-inner">
            <div className="flex justify-between items-center">
                <span className="text-dark-400">Notional</span>
                <span className="text-white font-mono">${notional}</span>
            </div>
            <div className="flex justify-between items-center">
                <span className="text-dark-400">Required Margin</span>
                <span className="text-white font-mono border-b border-dashed border-dark-600 pb-0.5">${margin}</span>
            </div>
            <div className="flex justify-between items-center">
                 <span className="text-dark-400">Available Balance</span>
                 <span className="text-white font-mono">${parseFloat(account.balance).toLocaleString()}</span>
            </div>
        </div>

        {/* Messages */}
        <div className="min-h-[20px]">
           {error && (
             <div className="animate-slide-up bg-danger-500/10 border border-danger-500/20 rounded-lg p-3 flex items-center gap-2">
               <div className="w-1.5 h-1.5 rounded-full bg-danger-500"></div>
               <p className="text-xs text-danger-400">{error}</p>
             </div>
           )}
           {success && (
             <div className="animate-slide-up bg-primary-500/10 border border-primary-500/20 rounded-lg p-3 flex items-center gap-2">
               <div className="w-1.5 h-1.5 rounded-full bg-primary-500"></div>
               <p className="text-xs text-primary-400 truncate">{success}</p>
             </div>
           )}
        </div>

        {/* Submit Button */}
        <button
          type="submit"
          disabled={!connected || isSubmitting}
          className={`w-full py-3.5 rounded-lg text-sm font-bold text-white transition-all duration-300 shadow-lg transform active:scale-[0.98] disabled:opacity-50 disabled:cursor-not-allowed disabled:transform-none flex justify-center items-center gap-2 ${
             isBuy
               ? 'bg-gradient-to-r from-primary-600 to-primary-500 hover:from-primary-500 hover:to-primary-400 shadow-glow-primary'
               : 'bg-gradient-to-r from-danger-600 to-danger-500 hover:from-danger-500 hover:to-danger-400 shadow-glow-danger'
          }`}
        >
          {!connected ? (
             'Connect Wallet to Trade'
          ) : isSubmitting ? (
             <>
               <svg className="animate-spin h-4 w-4 text-white" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
                 <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4"></circle>
                 <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
               </svg>
               <span>Processing...</span>
             </>
          ) : (
             <span>{isBuy ? 'Place Long Order' : 'Place Short Order'}</span>
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
