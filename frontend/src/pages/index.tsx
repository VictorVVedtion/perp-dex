import { OrderBook } from '@/components/OrderBook'
import { TradeForm } from '@/components/TradeForm'
import { PositionCard } from '@/components/PositionCard'
import { Chart } from '@/components/Chart'
import { RecentTrades } from '@/components/RecentTrades'
import { TradeHistory } from '@/components/TradeHistory'
import { useTradingStore } from '@/stores/tradingStore'
import { config } from '@/lib/config'
import { useEffect, useState } from 'react'
import {
  ChartBarIcon,
  ClockIcon,
  QueueListIcon,
  ArrowsUpDownIcon,
  InformationCircleIcon
} from '@heroicons/react/24/outline'

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
    <div className="max-w-[1600px] mx-auto px-4 sm:px-6 lg:px-8 py-4 animate-fade-in">
      {/* Market Stats Card */}
      <div className="mb-6 relative group">
        <div className="absolute -inset-[1px] bg-gradient-to-r from-primary-500/40 via-dark-700/50 to-danger-500/40 rounded-xl blur-[2px] group-hover:blur-[3px] transition-all duration-500" />
        <div className="relative bg-dark-900/80 backdrop-blur-xl rounded-xl border border-white/5 p-4 overflow-hidden shadow-2xl">
          {/* Decorative background element */}
          <div className="absolute top-0 right-0 w-64 h-64 bg-primary-500/5 blur-[80px] -mr-32 -mt-32 pointer-events-none" />

          <div className="flex items-center justify-between">
            <div className="flex items-center space-x-8">
              <div className="flex items-center space-x-4">
                <div className="w-10 h-10 bg-primary-500/10 rounded-lg flex items-center justify-center border border-primary-500/20">
                  <ChartBarIcon className="w-6 h-6 text-primary-500" />
                </div>
                <div>
                  <div className="text-2xl font-bold bg-gradient-to-br from-white to-dark-400 bg-clip-text text-transparent font-mono tracking-tight">
                    {currentPrice === '--' ? '--' : `$${parseFloat(currentPrice).toLocaleString()}`}
                  </div>
                  <div className="flex items-center space-x-2 mt-0.5">
                    <span className="text-xs font-bold text-dark-400 tracking-wider">BTC-USDC</span>
                    {wsConnected && (
                      <span className="flex items-center space-x-1.5 px-1.5 py-0.5 bg-primary-500/10 rounded-full border border-primary-500/20">
                        <span className="relative flex h-1.5 w-1.5">
                          <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-primary-400 opacity-75"></span>
                          <span className="relative inline-flex rounded-full h-1.5 w-1.5 bg-primary-500"></span>
                        </span>
                        <span className="text-[10px] uppercase font-bold text-primary-500 tracking-tighter">Live</span>
                      </span>
                    )}
                  </div>
                </div>
              </div>

              <div className="h-10 w-px bg-white/5" />

              <div className="grid grid-cols-4 gap-8">
                <div>
                  <div className="text-[10px] uppercase font-bold text-dark-500 tracking-widest mb-1">24h Change</div>
                  <div className={`font-mono text-sm font-semibold ${changeFormatted.positive ? 'text-primary-500' : 'text-danger-500'}`}>
                    {changeFormatted.text}
                  </div>
                </div>
                <div>
                  <div className="text-[10px] uppercase font-bold text-dark-500 tracking-widest mb-1">24h High</div>
                  <div className="text-white font-mono text-sm font-semibold">
                    {high24h === '--' ? '--' : `$${parseFloat(high24h).toLocaleString(undefined, { minimumFractionDigits: 2 })}`}
                  </div>
                </div>
                <div>
                  <div className="text-[10px] uppercase font-bold text-dark-500 tracking-widest mb-1">24h Low</div>
                  <div className="text-white font-mono text-sm font-semibold">
                    {low24h === '--' ? '--' : `$${parseFloat(low24h).toLocaleString(undefined, { minimumFractionDigits: 2 })}`}
                  </div>
                </div>
                <div>
                  <div className="text-[10px] uppercase font-bold text-dark-500 tracking-widest mb-1">24h Volume</div>
                  <div className="text-white font-mono text-sm font-semibold">{formatVolume(volume24h)}</div>
                </div>
              </div>
            </div>

            <div className="flex items-center space-x-3">
              <div className="px-3 py-1.5 bg-gradient-to-r from-primary-500/10 to-primary-500/5 text-primary-500 text-[10px] font-bold uppercase tracking-widest rounded-lg border border-primary-500/20 shadow-glow-sm">
                50x Max Leverage
              </div>
            </div>
          </div>
        </div>
      </div>

      {/* Main Trading Layout */}
      <div className="grid grid-cols-12 gap-5">
        {/* Order Book - Left */}
        <div className="col-span-2">
          <div className="h-[650px] bg-dark-900/50 backdrop-blur-sm rounded-xl border border-white/5 overflow-hidden shadow-lg">
            <OrderBook />
          </div>
        </div>

        {/* TradingView Chart - Center */}
        <div className="col-span-5">
          <div className="h-[650px] bg-dark-900/50 backdrop-blur-sm rounded-xl border border-white/5 overflow-hidden shadow-lg">
            <Chart marketId="BTC-USDC" height={650} />
          </div>
        </div>

        {/* Recent Trades */}
        <div className="col-span-2">
          <div className="h-[650px] bg-dark-900/50 backdrop-blur-sm rounded-xl border border-white/5 overflow-hidden shadow-lg">
            <RecentTrades marketId="BTC-USDC" maxTrades={50} />
          </div>
        </div>

        {/* Trade Form - Right */}
        <div className="col-span-3">
          <div className="h-[650px] bg-dark-900/50 backdrop-blur-sm rounded-xl border border-white/5 overflow-hidden shadow-lg">
            <TradeForm />
          </div>
        </div>
      </div>

      {/* Bottom Panel with Tabs */}
      <div className="mt-8 bg-dark-900/40 backdrop-blur-md rounded-2xl border border-white/5 p-1 shadow-2xl">
        <div className="flex items-center justify-between p-4 pb-0">
          <div className="flex items-center space-x-1 relative">
            <button
              onClick={() => setActiveTab('positions')}
              className={`flex items-center space-x-2 px-6 py-3 text-sm font-bold transition-all duration-300 relative z-10 ${
                activeTab === 'positions' ? 'text-primary-500' : 'text-dark-400 hover:text-white'
              }`}
            >
              <ChartBarIcon className="w-4 h-4" />
              <span>Positions</span>
              {displayPositions.length > 0 && (
                <span className="ml-1 px-1.5 py-0.5 bg-primary-500/20 text-primary-500 text-[10px] rounded-md">
                  {displayPositions.length}
                </span>
              )}
            </button>
            <button
              onClick={() => setActiveTab('orders')}
              className={`flex items-center space-x-2 px-6 py-3 text-sm font-bold transition-all duration-300 relative z-10 ${
                activeTab === 'orders' ? 'text-primary-500' : 'text-dark-400 hover:text-white'
              }`}
            >
              <QueueListIcon className="w-4 h-4" />
              <span>Open Orders</span>
            </button>
            <button
              onClick={() => setActiveTab('history')}
              className={`flex items-center space-x-2 px-6 py-3 text-sm font-bold transition-all duration-300 relative z-10 ${
                activeTab === 'history' ? 'text-primary-500' : 'text-dark-400 hover:text-white'
              }`}
            >
              <ClockIcon className="w-4 h-4" />
              <span>Trade History</span>
            </button>

            {/* Sliding Underline Effect */}
            <div
              className="absolute bottom-0 h-0.5 bg-primary-500 transition-all duration-300 ease-in-out"
              style={{
                left: activeTab === 'positions' ? '0' : activeTab === 'orders' ? '120px' : '240px',
                width: activeTab === 'positions' ? '120px' : activeTab === 'orders' ? '120px' : '120px'
              }}
            />
          </div>

          <div className="px-4 text-xs text-dark-500 flex items-center space-x-2">
            <InformationCircleIcon className="w-4 h-4 text-dark-600" />
            <span>Market execution is currently active</span>
          </div>
        </div>

        <div className="p-4 pt-6 min-h-[300px]">
          {/* Tab Content */}
          {activeTab === 'positions' && (
            displayPositions.length > 0 ? (
              <div className="space-y-4 animate-fade-in">
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
              <div className="flex flex-col items-center justify-center py-20 animate-fade-in">
                <div className="w-16 h-16 bg-dark-800 rounded-full flex items-center justify-center mb-4 border border-white/5">
                  <ArrowsUpDownIcon className="w-8 h-8 text-dark-600" />
                </div>
                <h3 className="text-white font-bold mb-1">No open positions</h3>
                <p className="text-dark-500 text-sm">Your active positions will appear here</p>
              </div>
            )
          )}

          {activeTab === 'orders' && (
            <div className="overflow-hidden rounded-xl border border-white/5 animate-fade-in">
              <table className="w-full">
                <thead>
                  <tr className="bg-dark-950/50 text-[10px] uppercase tracking-widest font-bold text-dark-500">
                    <th className="text-left px-6 py-4">Time</th>
                    <th className="text-left px-6 py-4">Type</th>
                    <th className="text-left px-6 py-4">Side</th>
                    <th className="text-right px-6 py-4">Price</th>
                    <th className="text-right px-6 py-4">Size</th>
                    <th className="text-right px-6 py-4">Filled</th>
                    <th className="text-right px-6 py-4">Action</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-white/5">
                  <tr>
                    <td colSpan={7} className="text-center py-20">
                      <div className="flex flex-col items-center justify-center">
                        <div className="w-16 h-16 bg-dark-800 rounded-full flex items-center justify-center mb-4 border border-white/5">
                          <QueueListIcon className="w-8 h-8 text-dark-600" />
                        </div>
                        <h3 className="text-white font-bold mb-1">No open orders</h3>
                        <p className="text-dark-500 text-sm">Your limit orders will appear here</p>
                      </div>
                    </td>
                  </tr>
                </tbody>
              </table>
            </div>
          )}

          {activeTab === 'history' && (
            <div className="animate-fade-in">
              <TradeHistory />
            </div>
          )}
        </div>
      </div>
    </div>
  )
}
