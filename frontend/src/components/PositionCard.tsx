import { Position } from '@/stores/tradingStore'
import BigNumber from 'bignumber.js'

interface PositionCardProps {
  position: Position
  markPrice: string
  onClose?: () => void
}

export function PositionCard({ position, markPrice, onClose }: PositionCardProps) {
  // Calculate unrealized PnL
  const priceDiff = new BigNumber(markPrice).minus(position.entryPrice)
  const unrealizedPnL = position.side === 'long'
    ? new BigNumber(position.size).times(priceDiff)
    : new BigNumber(position.size).times(priceDiff).negated()

  const pnlPercent = unrealizedPnL.div(position.margin).times(100)
  const isProfit = unrealizedPnL.isPositive()

  // Calculate margin ratio
  const notional = new BigNumber(position.size).times(markPrice)
  const equity = new BigNumber(position.margin).plus(unrealizedPnL)
  const marginRatio = equity.div(notional).times(100)

  return (
    <div className="glass-panel group bg-dark-900/80 backdrop-blur-md rounded-xl border border-dark-700/50 p-5 hover:shadow-[0_8px_30px_rgb(0,0,0,0.12)] hover:border-primary-500/30 transition-all duration-300">
      <div className="flex items-center justify-between mb-5">
        <div className="flex items-center space-x-3">
          <div className={`px-2.5 py-1 rounded-md text-xs font-bold tracking-wide shadow-sm ${
            position.side === 'long'
              ? 'bg-gradient-to-r from-primary-500/20 to-primary-500/5 text-primary-500 border border-primary-500/20'
              : 'bg-gradient-to-r from-danger-500/20 to-danger-500/5 text-danger-500 border border-danger-500/20'
          }`}>
            {position.side.toUpperCase()}
          </div>
          <span className="text-white font-bold text-lg tracking-tight">{position.marketId}</span>
          <span className="text-dark-400 text-sm bg-dark-800/50 px-2 py-0.5 rounded border border-dark-700/50">{position.leverage}x</span>
        </div>
        <button
          onClick={onClose}
          className="px-4 py-1.5 bg-dark-700/50 hover:bg-danger-500/20 hover:text-danger-400 hover:border-danger-500/30 hover:scale-105 active:scale-95 rounded-lg text-sm text-dark-300 font-medium transition-all shadow-sm border border-dark-600"
        >
          Close
        </button>
      </div>

      <div className="grid grid-cols-2 md:grid-cols-4 gap-6">
        {/* Size */}
        <div>
          <div className="text-xs text-dark-400 mb-1">Size</div>
          <div className="text-white font-mono">
            {parseFloat(position.size).toFixed(4)} BTC
          </div>
        </div>

        {/* Entry Price */}
        <div>
          <div className="text-xs text-dark-400 mb-1">Entry Price</div>
          <div className="text-white font-mono">
            ${parseFloat(position.entryPrice).toLocaleString()}
          </div>
        </div>

        {/* Mark Price */}
        <div>
          <div className="text-xs text-dark-400 mb-1">Mark Price</div>
          <div className="text-white font-mono">
            ${parseFloat(markPrice).toLocaleString()}
          </div>
        </div>

        {/* Unrealized PnL */}
        <div>
          <div className="text-xs text-dark-400 mb-1">Unrealized PnL</div>
          <div className={`font-mono font-bold text-lg tracking-tight ${
            isProfit
              ? 'bg-gradient-to-r from-primary-400 to-primary-300 bg-clip-text text-transparent filter drop-shadow-[0_0_8px_rgba(var(--primary-500),0.3)]'
              : 'bg-gradient-to-r from-danger-400 to-danger-300 bg-clip-text text-transparent'
          }`}>
            {isProfit ? '+' : ''}{unrealizedPnL.toFixed(2)} USDC
            <span className={`text-xs ml-1.5 font-medium ${isProfit ? 'text-primary-400' : 'text-danger-400'}`}>
              ({isProfit ? '+' : ''}{pnlPercent.toFixed(2)}%)
            </span>
          </div>
        </div>
      </div>

      {/* Additional Info */}
      <div className="mt-5 pt-4 border-t border-dark-700/50 grid grid-cols-3 gap-4 bg-dark-900/30 -mx-5 px-5 -mb-5 pb-5 rounded-b-xl">
        <div>
          <div className="text-xs text-dark-400 mb-1">Margin</div>
          <div className="text-white font-mono text-sm">
            ${parseFloat(position.margin).toLocaleString()}
          </div>
        </div>
        <div>
          <div className="text-xs text-dark-400 mb-1">Liq. Price</div>
          <div className="text-danger-400 font-mono text-sm">
            ${parseFloat(position.liquidationPrice).toLocaleString()}
          </div>
        </div>
        <div>
          <div className="text-xs text-dark-400 mb-1">Margin Ratio</div>
          <div className={`font-mono text-sm ${
            marginRatio.lt(10) ? 'text-danger-400' :
            marginRatio.lt(20) ? 'text-yellow-400' : 'text-primary-400'
          }`}>
            {marginRatio.toFixed(2)}%
          </div>
        </div>
      </div>
    </div>
  )
}
