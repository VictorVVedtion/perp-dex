import { OrderBook } from '@/components/OrderBook'
import { TradeForm } from '@/components/TradeForm'
import { PositionCard } from '@/components/PositionCard'
import { Chart } from '@/components/Chart'
import { RecentTrades } from '@/components/RecentTrades'
import { TradeHistory } from '@/components/TradeHistory'
import { useTradingStore } from '@/stores/tradingStore'
import { config } from '@/lib/config'
import { useEffect, useState } from 'react'

export default function TradePage() {
  const {
    priceInfo,
    ticker,
    positions,
    wsConnected,
    initWebSocket,
    closeWebSocket,
    initHyperliquid,
    closeHyperliquid,
  } = useTradingStore()

  // Initialize WebSocket connection based on config
  useEffect(() => {
    const useHyperliquid = config.features.useHyperliquid && !config.features.mockMode

    if (useHyperliquid) {
      initHyperliquid()
      return () => closeHyperliquid()
    } else {
      initWebSocket()
      return () => closeWebSocket()
    }
  }, [initWebSocket, closeWebSocket, initHyperliquid, closeHyperliquid])

  // Use real-time data - no mock fallback, show loading state for missing data
  const currentPrice = ticker?.lastPrice || priceInfo?.markPrice || '--'
  const change24h = ticker?.change24h || priceInfo?.change24h || '--'
  const high24h = ticker?.high24h || priceInfo?.high24h || '--'
  const low24h = ticker?.low24h || priceInfo?.low24h || '--'
  const volume24h = ticker?.volume24h || priceInfo?.volume24h || '0'

  // Format volume for display
  const formatVolume = (vol: string): string => {
    const num = parseFloat(vol)
    if (num >= 1e9) return `$${(num / 1e9).toFixed(1)}B`
    if (num >= 1e6) return `$${(num / 1e6).toFixed(0)}M`
    if (num >= 1e3) return `$${(num / 1e3).toFixed(0)}K`
    return `$${num.toFixed(0)}`
  }

  // Format change percentage
  const formatChange = (change: string): { text: string; positive: boolean } => {
    if (change === '--') {
      return { text: '--', positive: true }
    }
    const num = parseFloat(change.replace('%', '').replace('+', ''))
    const positive = !change.startsWith('-')
    return {
      text: `${positive ? '+' : ''}${num.toFixed(2)}%`,
      positive,
    }
  }

  const changeFormatted = formatChange(change24h)
  // Use real positions only - no mock fallback
  const displayPositions = positions

  // Tab state for bottom panel
  const [activeTab, setActiveTab] = useState<'positions' | 'orders' | 'history'>('positions')

  return (
    <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-6">
      {/* Market Stats */}
      <div className="mb-6 bg-dark-900 rounded-lg border border-dark-700 p-4">
        <div className="flex items-center justify-between">
          <div className="flex items-center space-x-6">
            <div>
              <div className="text-2xl font-bold text-white font-mono">
                {currentPrice === '--' ? '--' : `$${parseFloat(currentPrice).toLocaleString()}`}
              </div>
              <div className="flex items-center space-x-2 mt-1">
                <span className="text-xs text-dark-400">BTC-USDC</span>
                {wsConnected && (
                  <span className="flex items-center space-x-1">
                    <span className="w-1.5 h-1.5 bg-primary-400 rounded-full animate-pulse" />
                    <span className="text-xs text-primary-400">Live</span>
                  </span>
                )}
              </div>
            </div>
            <div className="h-10 w-px bg-dark-700" />
            <div>
              <div className="text-xs text-dark-400">24h Change</div>
              <div className={`font-mono ${changeFormatted.positive ? 'text-primary-400' : 'text-danger-400'}`}>
                {changeFormatted.text}
              </div>
            </div>
            <div>
              <div className="text-xs text-dark-400">24h High</div>
              <div className="text-white font-mono">
                {high24h === '--' ? '--' : `$${parseFloat(high24h).toLocaleString(undefined, { minimumFractionDigits: 2 })}`}
              </div>
            </div>
            <div>
              <div className="text-xs text-dark-400">24h Low</div>
              <div className="text-white font-mono">
                {low24h === '--' ? '--' : `$${parseFloat(low24h).toLocaleString(undefined, { minimumFractionDigits: 2 })}`}
              </div>
            </div>
            <div>
              <div className="text-xs text-dark-400">24h Volume</div>
              <div className="text-white font-mono">{formatVolume(volume24h)}</div>
            </div>
          </div>
          <div className="flex items-center space-x-2">
            <div className="px-2 py-1 bg-primary-500/20 text-primary-400 text-xs rounded">
              50x Max Leverage
            </div>
          </div>
        </div>
      </div>

      {/* Main Trading Layout */}
      <div className="grid grid-cols-12 gap-4">
        {/* Order Book - Left */}
        <div className="col-span-2">
          <div className="h-[600px]">
            <OrderBook />
          </div>
        </div>

        {/* TradingView Chart - Center */}
        <div className="col-span-5">
          <div className="h-[600px]">
            <Chart marketId="BTC-USDC" height={600} />
          </div>
        </div>

        {/* Recent Trades */}
        <div className="col-span-2">
          <div className="h-[600px]">
            <RecentTrades marketId="BTC-USDC" maxTrades={50} />
          </div>
        </div>

        {/* Trade Form - Right */}
        <div className="col-span-3">
          <TradeForm />
        </div>
      </div>

      {/* Bottom Panel with Tabs */}
      <div className="mt-6">
        {/* Tab Headers */}
        <div className="flex items-center space-x-1 mb-4 border-b border-dark-700">
          <button
            onClick={() => setActiveTab('positions')}
            className={`px-4 py-2 text-sm font-medium transition-colors border-b-2 ${
              activeTab === 'positions'
                ? 'text-primary-400 border-primary-400'
                : 'text-dark-400 border-transparent hover:text-white'
            }`}
          >
            Positions {displayPositions.length > 0 && `(${displayPositions.length})`}
          </button>
          <button
            onClick={() => setActiveTab('orders')}
            className={`px-4 py-2 text-sm font-medium transition-colors border-b-2 ${
              activeTab === 'orders'
                ? 'text-primary-400 border-primary-400'
                : 'text-dark-400 border-transparent hover:text-white'
            }`}
          >
            Open Orders
          </button>
          <button
            onClick={() => setActiveTab('history')}
            className={`px-4 py-2 text-sm font-medium transition-colors border-b-2 ${
              activeTab === 'history'
                ? 'text-primary-400 border-primary-400'
                : 'text-dark-400 border-transparent hover:text-white'
            }`}
          >
            Trade History
          </button>
        </div>

        {/* Tab Content */}
        {activeTab === 'positions' && (
          displayPositions.length > 0 ? (
            <div className="space-y-4">
              {displayPositions.map((position, i) => (
                <PositionCard
                  key={position.marketId + position.side + i}
                  position={position}
                  markPrice={currentPrice}
                  onClose={() => console.log('Close position:', position)}
                />
              ))}
            </div>
          ) : (
            <div className="bg-dark-900 rounded-lg border border-dark-700 p-8 text-center">
              <p className="text-dark-400">No open positions</p>
            </div>
          )
        )}

        {activeTab === 'orders' && (
          <div className="bg-dark-900 rounded-lg border border-dark-700">
            <table className="w-full">
              <thead>
                <tr className="text-xs text-dark-400 border-b border-dark-700">
                  <th className="text-left px-4 py-3">Time</th>
                  <th className="text-left px-4 py-3">Type</th>
                  <th className="text-left px-4 py-3">Side</th>
                  <th className="text-right px-4 py-3">Price</th>
                  <th className="text-right px-4 py-3">Size</th>
                  <th className="text-right px-4 py-3">Filled</th>
                  <th className="text-right px-4 py-3">Action</th>
                </tr>
              </thead>
              <tbody>
                <tr>
                  <td colSpan={7} className="text-center py-8 text-dark-400 text-sm">
                    No open orders
                  </td>
                </tr>
              </tbody>
            </table>
          </div>
        )}

        {activeTab === 'history' && (
          <TradeHistory />
        )}
      </div>
    </div>
  )
}
