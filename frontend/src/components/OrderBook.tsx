import { useMemo, useEffect, useState } from 'react'
import BigNumber from 'bignumber.js'
import { useTradingStore, PriceLevel, OrderBookData } from '@/stores/tradingStore'
import { config } from '@/lib/config'
import { getHyperliquidClient } from '@/lib/api/hyperliquid'

// Empty orderbook for initial state (not mock data)
const emptyOrderBook: OrderBookData = {
  bids: [],
  asks: [],
  bestBid: '0',
  bestAsk: '0',
  spread: '0',
}

interface OrderBookRowProps {
  price: string
  quantity: string
  total: string
  side: 'bid' | 'ask'
  maxTotal: string
  onClick: () => void
}

function OrderBookRow({ price, quantity, total, side, maxTotal, onClick }: OrderBookRowProps) {
  const percentage = new BigNumber(total).div(maxTotal).times(100).toNumber()

  return (
    <div
      onClick={onClick}
      className="orderbook-row relative grid grid-cols-3 text-xs py-1 px-2 cursor-pointer"
    >
      {/* Background bar */}
      <div
        className={`absolute inset-y-0 right-0 ${
          side === 'bid' ? 'bg-primary-500/10' : 'bg-danger-500/10'
        }`}
        style={{ width: `${percentage}%` }}
      />

      {/* Content */}
      <span className={`relative z-10 text-right ${
        side === 'bid' ? 'text-primary-400' : 'text-danger-400'
      }`}>
        {parseFloat(price).toLocaleString(undefined, { minimumFractionDigits: 2 })}
      </span>
      <span className="relative z-10 text-right text-dark-200">
        {parseFloat(quantity).toFixed(4)}
      </span>
      <span className="relative z-10 text-right text-dark-300">
        {parseFloat(total).toFixed(4)}
      </span>
    </div>
  )
}

interface OrderBookProps {
  marketId?: string
}

export function OrderBook({ marketId = 'BTC-USDC' }: OrderBookProps) {
  const { orderBook, setPrice, wsConnected } = useTradingStore()
  const [isLoading, setIsLoading] = useState(true)
  const [localOrderBook, setLocalOrderBook] = useState(emptyOrderBook)

  const useHyperliquid = config.features.useHyperliquid && !config.features.mockMode

  // Load initial orderbook from Hyperliquid API
  useEffect(() => {
    if (useHyperliquid) {
      const loadOrderbook = async () => {
        setIsLoading(true)
        try {
          const hlClient = getHyperliquidClient()
          const hlOrderbook = await hlClient.getOrderbook(marketId)

          if (hlOrderbook) {
            const bestBid = hlOrderbook.bids[0]?.price || '0'
            const bestAsk = hlOrderbook.asks[0]?.price || '0'
            const spread = new BigNumber(bestAsk).minus(bestBid).toString()

            setLocalOrderBook({
              bids: hlOrderbook.bids,
              asks: hlOrderbook.asks,
              bestBid,
              bestAsk,
              spread,
            })
          }
        } catch (error) {
          console.error('Failed to load orderbook:', error)
        } finally {
          setIsLoading(false)
        }
      }

      loadOrderbook()
    } else {
      setIsLoading(false)
    }
  }, [marketId, useHyperliquid])

  // Use store orderbook if available (from WebSocket), otherwise use local state
  const data = orderBook || localOrderBook

  // Calculate running totals
  const asksWithTotals = useMemo(() => {
    let runningTotal = new BigNumber(0)
    return [...data.asks].reverse().map((level) => {
      runningTotal = runningTotal.plus(level.quantity)
      return { ...level, total: runningTotal.toString() }
    }).reverse()
  }, [data.asks])

  const bidsWithTotals = useMemo(() => {
    let runningTotal = new BigNumber(0)
    return data.bids.map((level) => {
      runningTotal = runningTotal.plus(level.quantity)
      return { ...level, total: runningTotal.toString() }
    })
  }, [data.bids])

  const maxTotal = useMemo(() => {
    const maxAsk = asksWithTotals[asksWithTotals.length - 1]?.total || '0'
    const maxBid = bidsWithTotals[bidsWithTotals.length - 1]?.total || '0'
    return BigNumber.max(maxAsk, maxBid).toString()
  }, [asksWithTotals, bidsWithTotals])

  return (
    <div className="bg-dark-900 rounded-lg border border-dark-700 h-full flex flex-col">
      {/* Header */}
      <div className="flex items-center justify-between px-4 py-3 border-b border-dark-700">
        <div className="flex items-center space-x-2">
          <h3 className="text-sm font-medium text-white">Order Book</h3>
          {wsConnected && (
            <span className="flex items-center space-x-1 text-xs text-primary-400">
              <span className="w-1.5 h-1.5 bg-primary-400 rounded-full animate-pulse" />
              <span>Live</span>
            </span>
          )}
          {useHyperliquid && (
            <span className="text-xs text-dark-500 bg-dark-800 px-1.5 py-0.5 rounded">HL</span>
          )}
        </div>
      </div>

      {/* Column Headers */}
      <div className="grid grid-cols-3 text-xs text-dark-400 px-2 py-2 border-b border-dark-800">
        <span className="text-right">Price (USDC)</span>
        <span className="text-right">Size (BTC)</span>
        <span className="text-right">Total</span>
      </div>

      {/* Asks */}
      <div className="flex-1 overflow-y-auto">
        <div className="flex flex-col-reverse">
          {asksWithTotals.map((level, i) => (
            <OrderBookRow
              key={`ask-${i}`}
              price={level.price}
              quantity={level.quantity}
              total={level.total}
              side="ask"
              maxTotal={maxTotal}
              onClick={() => setPrice(level.price)}
            />
          ))}
        </div>
      </div>

      {/* Spread */}
      <div className="px-2 py-2 bg-dark-800 border-y border-dark-700">
        <div className="flex items-center justify-between text-sm">
          <span className="text-primary-400 font-mono">
            {parseFloat(data.bestBid).toLocaleString(undefined, { minimumFractionDigits: 2 })}
          </span>
          <span className="text-dark-400 text-xs">
            Spread: ${parseFloat(data.spread).toFixed(2)}
          </span>
          <span className="text-danger-400 font-mono">
            {parseFloat(data.bestAsk).toLocaleString(undefined, { minimumFractionDigits: 2 })}
          </span>
        </div>
      </div>

      {/* Bids */}
      <div className="flex-1 overflow-y-auto">
        {bidsWithTotals.map((level, i) => (
          <OrderBookRow
            key={`bid-${i}`}
            price={level.price}
            quantity={level.quantity}
            total={level.total}
            side="bid"
            maxTotal={maxTotal}
            onClick={() => setPrice(level.price)}
          />
        ))}
      </div>
    </div>
  )
}
