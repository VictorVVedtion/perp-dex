import { useMemo } from 'react'
import { PositionCard } from '@/components/PositionCard'
import { useTradingStore } from '@/stores/tradingStore'

export default function PositionsPage() {
  const { positions, priceInfo, ticker } = useTradingStore()

  // Use real mark price from ticker or priceInfo
  const markPrice = ticker?.markPrice || priceInfo?.markPrice || '0'

  // Calculate total PnL from real positions
  const { totalUnrealizedPnL, totalMargin } = useMemo(() => {
    const unrealized = positions.reduce((sum, pos) => {
      const pnl = parseFloat(pos.unrealizedPnl)
      return sum + pnl
    }, 0)

    const margin = positions.reduce((sum, pos) => {
      return sum + parseFloat(pos.margin)
    }, 0)

    return { totalUnrealizedPnL: unrealized, totalMargin: margin }
  }, [positions])

  return (
    <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-6">
      <h1 className="text-2xl font-bold text-white mb-6">Positions</h1>

      {/* Summary Cards */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4 mb-6">
        <div className="bg-dark-900 rounded-lg border border-dark-700 p-4">
          <div className="text-xs text-dark-400 mb-1">Open Positions</div>
          <div className="text-2xl font-bold text-white">{positions.length}</div>
        </div>
        <div className="bg-dark-900 rounded-lg border border-dark-700 p-4">
          <div className="text-xs text-dark-400 mb-1">Total Margin Used</div>
          <div className="text-2xl font-bold text-white font-mono">
            ${totalMargin.toLocaleString()}
          </div>
        </div>
        <div className="bg-dark-900 rounded-lg border border-dark-700 p-4">
          <div className="text-xs text-dark-400 mb-1">Total Unrealized PnL</div>
          <div className={`text-2xl font-bold font-mono ${
            totalUnrealizedPnL >= 0 ? 'text-primary-400' : 'text-danger-400'
          }`}>
            {totalUnrealizedPnL >= 0 ? '+' : ''}${totalUnrealizedPnL.toLocaleString()}
          </div>
        </div>
        <div className="bg-dark-900 rounded-lg border border-dark-700 p-4">
          <div className="text-xs text-dark-400 mb-1">ROE</div>
          <div className={`text-2xl font-bold font-mono ${
            totalUnrealizedPnL >= 0 ? 'text-primary-400' : 'text-danger-400'
          }`}>
            {totalMargin > 0
              ? `${totalUnrealizedPnL >= 0 ? '+' : ''}${((totalUnrealizedPnL / totalMargin) * 100).toFixed(2)}%`
              : '0.00%'}
          </div>
        </div>
      </div>

      {/* Positions List */}
      {positions.length > 0 ? (
        <div className="space-y-4">
          {positions.map((position, i) => (
            <PositionCard
              key={i}
              position={position}
              markPrice={markPrice}
              onClose={() => console.log('Close position:', position)}
            />
          ))}
        </div>
      ) : (
        <div className="bg-dark-900 rounded-lg border border-dark-700 p-12 text-center">
          <svg className="w-16 h-16 mx-auto mb-4 text-dark-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1} d="M9 17v-2m3 2v-4m3 4v-6m2 10H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
          </svg>
          <p className="text-dark-400 text-lg">No open positions</p>
          <p className="text-dark-500 text-sm mt-2">Open a position on the Trade page to get started</p>
        </div>
      )}

      {/* Trade History */}
      <div className="mt-8">
        <h2 className="text-lg font-medium text-white mb-4">Trade History</h2>
        <div className="bg-dark-900 rounded-lg border border-dark-700">
          <table className="w-full">
            <thead>
              <tr className="text-xs text-dark-400 border-b border-dark-700">
                <th className="text-left px-4 py-3">Time</th>
                <th className="text-left px-4 py-3">Market</th>
                <th className="text-left px-4 py-3">Side</th>
                <th className="text-right px-4 py-3">Size</th>
                <th className="text-right px-4 py-3">Price</th>
                <th className="text-right px-4 py-3">PnL</th>
                <th className="text-right px-4 py-3">Fee</th>
              </tr>
            </thead>
            <tbody>
              <tr>
                <td colSpan={7} className="text-center py-8 text-dark-400 text-sm">
                  No trade history
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
    </div>
  )
}
