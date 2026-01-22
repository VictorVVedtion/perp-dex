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
  const [flash, setFlash] = useState(false)

  useEffect(() => {
    setFlash(true)
    const timer = setTimeout(() => setFlash(false), 200)
    return () => clearTimeout(timer)
  }, [price, quantity])

  return (
    <div
      onClick={onClick}
      className={`orderbook-row relative grid grid-cols-3 text-xs py-1 px-2 cursor-pointer transition-all duration-75 border-y border-transparent hover:bg-white/5 hover:border-white/5 ${
        flash ? (side === 'bid' ? 'bg-primary-500/20' : 'bg-danger-500/20') : ''
      }`}
    >
      {/* Background bar */}
      <div
        className={`absolute inset-y-0 right-0 transition-all duration-300 ease-out opacity-20 ${
          side === 'bid'
            ? 'bg-gradient-to-l from-primary-500 to-transparent'
            : 'bg-gradient-to-l from-danger-500 to-transparent'
        }`}
        style={{ width: `${percentage}%` }}
      />

      {/* Content */}
      <span className={`relative z-10 text-right font-mono ${
        side === 'bid' ? 'text-primary-500' : 'text-danger-500'
      }`}>
        {parseFloat(price).toLocaleString(undefined, { minimumFractionDigits: 2 })}
      </span>
      <span className={`relative z-10 text-right font-mono transition-colors ${
        flash ? 'text-white' : 'text-dark-200'
      }`}>
        {parseFloat(quantity).toFixed(4)}
      </span>
      <span className="relative z-10 text-right text-dark-300 font-mono">
        {parseFloat(total).toFixed(4)}
      </span>
    </div>
  )
}

function OrderBookSkeleton() {
  return (
    <div className="flex-1 overflow-hidden space-y-px p-2 animate-pulse">
      {[...Array(15)].map((_, i) => (
        <div key={i} className="grid grid-cols-3 gap-2 py-1">
          <div className="h-3 bg-dark-700/30 rounded col-span-1 ml-auto w-16" />
          <div className="h-3 bg-dark-700/20 rounded col-span-1 ml-auto w-12" />
          <div className="h-3 bg-dark-700/10 rounded col-span-1 ml-auto w-14" />
        </div>
      ))}
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

  if (isLoading) {
    return (
      <div className="bg-dark-900/80 backdrop-blur-xl rounded-xl border border-white/10 h-full flex flex-col shadow-2xl">
        <div className="flex items-center justify-between px-4 py-3 border-b border-white/10">
           <h3 className="text-sm font-medium text-white">Order Book</h3>
        </div>
        <div className="grid grid-cols-3 text-xs text-dark-400 px-2 py-2 border-b border-white/10">
          <span className="text-right">Price</span>
          <span className="text-right">Size</span>
          <span className="text-right">Total</span>
        </div>
        <OrderBookSkeleton />
      </div>
    )
  }

  return (
    <div className="bg-dark-900/80 backdrop-blur-xl rounded-xl border border-white/10 h-full flex flex-col shadow-2xl overflow-hidden">
      {/* Header */}
      <div className="flex items-center justify-between px-4 py-3 border-b border-white/10">
        <div className="flex items-center space-x-2">
          <h3 className="text-sm font-medium text-white">Order Book</h3>
          {wsConnected && (
            <div className="flex items-center space-x-1.5 px-2 py-0.5 rounded-full bg-primary-500/10 border border-primary-500/20">
              <span className="relative flex h-2 w-2">
                <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-primary-500 opacity-75"></span>
                <span className="relative inline-flex rounded-full h-2 w-2 bg-primary-500"></span>
              </span>
              <span className="text-[10px] font-medium text-primary-500 uppercase tracking-wider">Live</span>
            </div>
          )}
          {useHyperliquid && (
            <span className="text-[10px] text-dark-400 bg-white/5 px-1.5 py-0.5 rounded border border-white/10">HL</span>
          )}
        </div>
      </div>

      {/* Column Headers */}
      <div className="grid grid-cols-3 text-xs text-dark-400 px-2 py-2 border-b border-white/5 bg-white/2">
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
      <div className="px-2 py-3 backdrop-blur-md bg-white/5 border-y border-white/10 z-10 my-0.5">
        <div className="flex items-center justify-between text-sm">
          <span className="text-primary-500 font-mono font-medium">
            {parseFloat(data.bestBid).toLocaleString(undefined, { minimumFractionDigits: 2 })}
          </span>
          <span className="text-dark-400 text-xs font-medium">
            Spread: ${parseFloat(data.spread).toFixed(2)}
          </span>
          <span className="text-danger-500 font-mono font-medium">
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
