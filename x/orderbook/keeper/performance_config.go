// Package keeper provides high-performance orderbook management
package keeper

import (
	"sync"
	"time"

	"cosmossdk.io/math"
	"github.com/openalpha/perp-dex/x/orderbook/types"
)

// PerformanceConfig contains tuning parameters for high TPS
type PerformanceConfig struct {
	// Parallel matching configuration
	Parallel ParallelConfig

	// Memory pool configuration
	Pool PoolConfig

	// Batch processing configuration
	Batch BatchConfig

	// Cache configuration
	Cache CacheConfig
}

// PoolConfig holds object pool settings
type PoolConfig struct {
	// OrderPoolSize is the initial size of the order pool
	OrderPoolSize int
	// TradePoolSize is the initial size of the trade pool
	TradePoolSize int
	// ResultPoolSize is the initial size of the match result pool
	ResultPoolSize int
}

// BatchConfig holds batch processing settings
type BatchConfig struct {
	// MaxBatchSize is the maximum orders processed per batch
	MaxBatchSize int
	// FlushInterval is the interval for cache flush
	FlushInterval time.Duration
	// ParallelFlush enables parallel flushing to store
	ParallelFlush bool
}

// CacheConfig holds cache settings
type CacheConfig struct {
	// OrderBookCacheSize is the max order books in memory
	OrderBookCacheSize int
	// OrderCacheSize is the max orders in memory
	OrderCacheSize int
	// PreloadMarkets preloads these markets on startup
	PreloadMarkets []string
}

// DefaultPerformanceConfig returns optimized defaults for high TPS
func DefaultPerformanceConfig() PerformanceConfig {
	return PerformanceConfig{
		Parallel: ParallelConfig{
			Enabled:   true,
			Workers:   16, // Increased from 4 for 2000 TPS target
			BatchSize: 500, // Increased from 100
			Timeout:   10 * time.Second,
		},
		Pool: PoolConfig{
			OrderPoolSize:  10000,
			TradePoolSize:  5000,
			ResultPoolSize: 1000,
		},
		Batch: BatchConfig{
			MaxBatchSize:  1000,
			FlushInterval: 100 * time.Millisecond,
			ParallelFlush: true,
		},
		Cache: CacheConfig{
			OrderBookCacheSize: 100,
			OrderCacheSize:     100000,
			PreloadMarkets:     []string{"BTC-USDC", "ETH-USDC"},
		},
	}
}

// HighTPSConfig returns configuration optimized for 2000+ TPS
func HighTPSConfig() PerformanceConfig {
	return PerformanceConfig{
		Parallel: ParallelConfig{
			Enabled:   true,
			Workers:   32, // Maximum parallelism
			BatchSize: 1000,
			Timeout:   15 * time.Second,
		},
		Pool: PoolConfig{
			OrderPoolSize:  50000,
			TradePoolSize:  25000,
			ResultPoolSize: 5000,
		},
		Batch: BatchConfig{
			MaxBatchSize:  2000,
			FlushInterval: 50 * time.Millisecond,
			ParallelFlush: true,
		},
		Cache: CacheConfig{
			OrderBookCacheSize: 200,
			OrderCacheSize:     500000,
			PreloadMarkets:     []string{"BTC-USDC", "ETH-USDC", "SOL-USDC"},
		},
	}
}

// ObjectPools provides memory pools for frequently allocated objects
type ObjectPools struct {
	orders     *sync.Pool
	trades     *sync.Pool
	results    *sync.Pool
	priceLevels *sync.Pool
	decPools   *sync.Pool
}

// GlobalPools is the singleton instance of object pools
var GlobalPools = NewObjectPools()

// NewObjectPools creates initialized object pools
func NewObjectPools() *ObjectPools {
	return &ObjectPools{
		orders: &sync.Pool{
			New: func() interface{} {
				return &types.Order{}
			},
		},
		trades: &sync.Pool{
			New: func() interface{} {
				return &types.Trade{}
			},
		},
		results: &sync.Pool{
			New: func() interface{} {
				return &MatchResultV2{
					Trades: make([]*types.Trade, 0, 16),
				}
			},
		},
		priceLevels: &sync.Pool{
			New: func() interface{} {
				return &PriceLevelV2{
					Orders: make([]*types.Order, 0, 64),
				}
			},
		},
		decPools: &sync.Pool{
			New: func() interface{} {
				return math.LegacyZeroDec()
			},
		},
	}
}

// GetOrder retrieves an order from the pool
func (p *ObjectPools) GetOrder() *types.Order {
	return p.orders.Get().(*types.Order)
}

// PutOrder returns an order to the pool
func (p *ObjectPools) PutOrder(o *types.Order) {
	if o == nil {
		return
	}
	// Reset order before returning to pool
	*o = types.Order{}
	p.orders.Put(o)
}

// GetTrade retrieves a trade from the pool
func (p *ObjectPools) GetTrade() *types.Trade {
	return p.trades.Get().(*types.Trade)
}

// PutTrade returns a trade to the pool
func (p *ObjectPools) PutTrade(t *types.Trade) {
	if t == nil {
		return
	}
	*t = types.Trade{}
	p.trades.Put(t)
}

// GetMatchResult retrieves a match result from the pool
func (p *ObjectPools) GetMatchResult() *MatchResultV2 {
	r := p.results.Get().(*MatchResultV2)
	r.Trades = r.Trades[:0]
	r.FilledQty = math.LegacyZeroDec()
	r.AvgPrice = math.LegacyZeroDec()
	r.RemainingQty = math.LegacyZeroDec()
	return r
}

// PutMatchResult returns a match result to the pool
func (p *ObjectPools) PutMatchResult(r *MatchResultV2) {
	if r == nil {
		return
	}
	// Clear trades slice but keep capacity
	r.Trades = r.Trades[:0]
	p.results.Put(r)
}

// GetPriceLevel retrieves a price level from the pool
func (p *ObjectPools) GetPriceLevel() *PriceLevelV2 {
	pl := p.priceLevels.Get().(*PriceLevelV2)
	pl.Orders = pl.Orders[:0]
	return pl
}

// PutPriceLevel returns a price level to the pool
func (p *ObjectPools) PutPriceLevel(pl *PriceLevelV2) {
	if pl == nil {
		return
	}
	pl.Orders = pl.Orders[:0]
	p.priceLevels.Put(pl)
}

// PreallocatedSlices provides pre-allocated slice buffers
type PreallocatedSlices struct {
	mu      sync.Mutex
	orders  [][]*types.Order
	trades  [][]*types.Trade
	results [][]*MatchResultV2
}

// NewPreallocatedSlices creates slice buffers
func NewPreallocatedSlices(count, orderCap, tradeCap, resultCap int) *PreallocatedSlices {
	p := &PreallocatedSlices{
		orders:  make([][]*types.Order, count),
		trades:  make([][]*types.Trade, count),
		results: make([][]*MatchResultV2, count),
	}
	for i := 0; i < count; i++ {
		p.orders[i] = make([]*types.Order, 0, orderCap)
		p.trades[i] = make([]*types.Trade, 0, tradeCap)
		p.results[i] = make([]*MatchResultV2, 0, resultCap)
	}
	return p
}

// GetOrderSlice retrieves a pre-allocated order slice
func (p *PreallocatedSlices) GetOrderSlice(idx int) []*types.Order {
	p.mu.Lock()
	defer p.mu.Unlock()
	if idx < 0 || idx >= len(p.orders) {
		return make([]*types.Order, 0, 100)
	}
	s := p.orders[idx]
	p.orders[idx] = p.orders[idx][:0]
	return s
}

// PerformanceMetrics tracks engine performance
type PerformanceMetrics struct {
	mu sync.RWMutex

	// Order metrics
	TotalOrders   uint64
	MatchedOrders uint64
	CancelledOrders uint64

	// Trade metrics
	TotalTrades   uint64
	TotalVolume   math.LegacyDec

	// Latency metrics
	TotalLatencyNs  int64
	OrderCount      int64
	MinLatencyNs    int64
	MaxLatencyNs    int64

	// Throughput metrics
	LastSecondOrders  uint64
	PeakOrdersPerSec  uint64

	// Memory metrics
	PoolHits   uint64
	PoolMisses uint64
}

// NewPerformanceMetrics creates initialized metrics
func NewPerformanceMetrics() *PerformanceMetrics {
	return &PerformanceMetrics{
		TotalVolume:  math.LegacyZeroDec(),
		MinLatencyNs: int64(^uint64(0) >> 1), // Max int64
	}
}

// RecordOrder records an order processing event
func (m *PerformanceMetrics) RecordOrder(latencyNs int64, matched bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.TotalOrders++
	if matched {
		m.MatchedOrders++
	}

	m.TotalLatencyNs += latencyNs
	m.OrderCount++

	if latencyNs < m.MinLatencyNs {
		m.MinLatencyNs = latencyNs
	}
	if latencyNs > m.MaxLatencyNs {
		m.MaxLatencyNs = latencyNs
	}
}

// RecordTrade records a trade event
func (m *PerformanceMetrics) RecordTrade(volume math.LegacyDec) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.TotalTrades++
	m.TotalVolume = m.TotalVolume.Add(volume)
}

// GetAverageLatency returns average order latency in nanoseconds
func (m *PerformanceMetrics) GetAverageLatency() int64 {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.OrderCount == 0 {
		return 0
	}
	return m.TotalLatencyNs / m.OrderCount
}

// GetStats returns a snapshot of all metrics
func (m *PerformanceMetrics) GetStats() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return map[string]interface{}{
		"total_orders":      m.TotalOrders,
		"matched_orders":    m.MatchedOrders,
		"cancelled_orders":  m.CancelledOrders,
		"total_trades":      m.TotalTrades,
		"total_volume":      m.TotalVolume.String(),
		"avg_latency_ns":    m.GetAverageLatency(),
		"min_latency_ns":    m.MinLatencyNs,
		"max_latency_ns":    m.MaxLatencyNs,
		"peak_orders_sec":   m.PeakOrdersPerSec,
	}
}

// Reset resets all metrics
func (m *PerformanceMetrics) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.TotalOrders = 0
	m.MatchedOrders = 0
	m.CancelledOrders = 0
	m.TotalTrades = 0
	m.TotalVolume = math.LegacyZeroDec()
	m.TotalLatencyNs = 0
	m.OrderCount = 0
	m.MinLatencyNs = int64(^uint64(0) >> 1)
	m.MaxLatencyNs = 0
	m.LastSecondOrders = 0
	m.PeakOrdersPerSec = 0
	m.PoolHits = 0
	m.PoolMisses = 0
}
