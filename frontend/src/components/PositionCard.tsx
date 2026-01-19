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
    <div className="bg-dark-900 rounded-lg border border-dark-700 p-4">
      <div className="flex items-center justify-between mb-4">
        <div className="flex items-center space-x-3">
          <div className={`px-2 py-1 rounded text-xs font-medium ${
            position.side === 'long'
              ? 'bg-primary-500/20 text-primary-400'
              : 'bg-danger-500/20 text-danger-400'
          }`}>
            {position.side.toUpperCase()}
          </div>
          <span className="text-white font-medium">{position.marketId}</span>
          <span className="text-dark-400 text-sm">{position.leverage}x</span>
        </div>
        <button
          onClick={onClose}
          className="px-3 py-1 bg-dark-700 hover:bg-dark-600 rounded text-sm text-white transition-colors"
        >
          Close
        </button>
      </div>

      <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
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
          <div className={`font-mono ${isProfit ? 'text-primary-400' : 'text-danger-400'}`}>
            {isProfit ? '+' : ''}{unrealizedPnL.toFixed(2)} USDC
            <span className="text-xs ml-1">
              ({isProfit ? '+' : ''}{pnlPercent.toFixed(2)}%)
            </span>
          </div>
        </div>
      </div>

      {/* Additional Info */}
      <div className="mt-4 pt-4 border-t border-dark-700 grid grid-cols-3 gap-4">
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
