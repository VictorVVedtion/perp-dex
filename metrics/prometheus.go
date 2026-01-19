package metrics

import (
	"net/http"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// PerpDEX Metrics Collector
// Provides comprehensive metrics for monitoring

var (
	// Singleton collector
	collector     *Collector
	collectorOnce sync.Once
)

// Collector holds all PerpDEX metrics
type Collector struct {
	// Order metrics
	OrdersTotal          *prometheus.CounterVec
	OrdersActive         *prometheus.GaugeVec
	OrderFillRate        *prometheus.HistogramVec
	OrderLatency         *prometheus.HistogramVec

	// Matching engine metrics
	MatchingLatency      *prometheus.HistogramVec
	MatchingThroughput   *prometheus.GaugeVec
	OrderbookDepth       *prometheus.GaugeVec
	SpreadBps            *prometheus.GaugeVec

	// Trade metrics
	TradesTotal          *prometheus.CounterVec
	TradeVolume          *prometheus.CounterVec
	TradeValue           *prometheus.CounterVec

	// Position metrics
	PositionsOpen        *prometheus.GaugeVec
	PositionValue        *prometheus.GaugeVec
	UnrealizedPnL        *prometheus.GaugeVec
	Leverage             *prometheus.HistogramVec

	// Liquidation metrics
	LiquidationsTotal    *prometheus.CounterVec
	LiquidationValue     *prometheus.CounterVec
	LiquidationDeficit   *prometheus.CounterVec

	// Insurance fund metrics
	InsuranceFundBalance *prometheus.GaugeVec
	InsuranceFundInflow  *prometheus.CounterVec
	InsuranceFundOutflow *prometheus.CounterVec

	// ADL metrics
	ADLEventsTotal       *prometheus.CounterVec
	ADLPositionsAffected *prometheus.CounterVec
	ADLValueDeleveraged  *prometheus.CounterVec

	// Funding rate metrics
	FundingRate          *prometheus.GaugeVec
	FundingPayments      *prometheus.CounterVec

	// Oracle metrics
	OraclePrice          *prometheus.GaugeVec
	OracleDeviation      *prometheus.GaugeVec
	OracleSourceCount    *prometheus.GaugeVec
	OracleLatency        *prometheus.HistogramVec

	// WebSocket metrics
	WSConnectionsActive  *prometheus.GaugeVec
	WSMessagesTotal      *prometheus.CounterVec
	WSMessageLatency     *prometheus.HistogramVec
	WSSubscriptions      *prometheus.GaugeVec

	// API metrics
	APIRequestsTotal     *prometheus.CounterVec
	APIRequestLatency    *prometheus.HistogramVec
	APIErrorsTotal       *prometheus.CounterVec
	RateLimitHits        *prometheus.CounterVec

	// System metrics
	ActiveUsers          prometheus.Gauge
	BlockHeight prometheus.Gauge
	BlockTime   *prometheus.HistogramVec
	TxPoolSize  prometheus.Gauge
	PeerCount   prometheus.Gauge
}

// GetCollector returns the singleton metrics collector
func GetCollector() *Collector {
	collectorOnce.Do(func() {
		collector = newCollector()
	})
	return collector
}

// newCollector creates a new metrics collector
func newCollector() *Collector {
	c := &Collector{}

	// Order metrics
	c.OrdersTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "perpdex",
			Subsystem: "orders",
			Name:      "total",
			Help:      "Total number of orders submitted",
		},
		[]string{"market_id", "side", "type", "status"},
	)

	c.OrdersActive = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "perpdex",
			Subsystem: "orders",
			Name:      "active",
			Help:      "Number of active orders",
		},
		[]string{"market_id", "side"},
	)

	c.OrderFillRate = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "perpdex",
			Subsystem: "orders",
			Name:      "fill_rate",
			Help:      "Order fill rate (0-1)",
			Buckets:   []float64{0.1, 0.25, 0.5, 0.75, 0.9, 0.95, 1.0},
		},
		[]string{"market_id", "type"},
	)

	c.OrderLatency = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "perpdex",
			Subsystem: "orders",
			Name:      "latency_ms",
			Help:      "Order processing latency in milliseconds",
			Buckets:   []float64{1, 5, 10, 25, 50, 100, 250, 500, 1000},
		},
		[]string{"market_id", "type"},
	)

	// Matching engine metrics
	c.MatchingLatency = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "perpdex",
			Subsystem: "matching",
			Name:      "latency_ms",
			Help:      "Matching engine latency in milliseconds",
			Buckets:   []float64{0.1, 0.5, 1, 2, 5, 10, 25, 50},
		},
		[]string{"market_id"},
	)

	c.MatchingThroughput = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "perpdex",
			Subsystem: "matching",
			Name:      "throughput_ops",
			Help:      "Matching engine throughput (operations per second)",
		},
		[]string{"market_id"},
	)

	c.OrderbookDepth = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "perpdex",
			Subsystem: "orderbook",
			Name:      "depth",
			Help:      "Orderbook depth (number of price levels)",
		},
		[]string{"market_id", "side"},
	)

	c.SpreadBps = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "perpdex",
			Subsystem: "orderbook",
			Name:      "spread_bps",
			Help:      "Bid-ask spread in basis points",
		},
		[]string{"market_id"},
	)

	// Trade metrics
	c.TradesTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "perpdex",
			Subsystem: "trades",
			Name:      "total",
			Help:      "Total number of trades executed",
		},
		[]string{"market_id"},
	)

	c.TradeVolume = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "perpdex",
			Subsystem: "trades",
			Name:      "volume",
			Help:      "Total trading volume (in base asset)",
		},
		[]string{"market_id"},
	)

	c.TradeValue = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "perpdex",
			Subsystem: "trades",
			Name:      "value_usdc",
			Help:      "Total trading value in USDC",
		},
		[]string{"market_id"},
	)

	// Position metrics
	c.PositionsOpen = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "perpdex",
			Subsystem: "positions",
			Name:      "open",
			Help:      "Number of open positions",
		},
		[]string{"market_id", "side"},
	)

	c.PositionValue = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "perpdex",
			Subsystem: "positions",
			Name:      "value_usdc",
			Help:      "Total position value in USDC",
		},
		[]string{"market_id", "side"},
	)

	c.UnrealizedPnL = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "perpdex",
			Subsystem: "positions",
			Name:      "unrealized_pnl_usdc",
			Help:      "Total unrealized PnL in USDC",
		},
		[]string{"market_id"},
	)

	c.Leverage = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "perpdex",
			Subsystem: "positions",
			Name:      "leverage",
			Help:      "Position leverage distribution",
			Buckets:   []float64{1, 2, 5, 10, 20, 50, 100, 125},
		},
		[]string{"market_id"},
	)

	// Liquidation metrics
	c.LiquidationsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "perpdex",
			Subsystem: "liquidations",
			Name:      "total",
			Help:      "Total number of liquidations",
		},
		[]string{"market_id", "type"},
	)

	c.LiquidationValue = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "perpdex",
			Subsystem: "liquidations",
			Name:      "value_usdc",
			Help:      "Total liquidation value in USDC",
		},
		[]string{"market_id"},
	)

	c.LiquidationDeficit = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "perpdex",
			Subsystem: "liquidations",
			Name:      "deficit_usdc",
			Help:      "Total liquidation deficit in USDC",
		},
		[]string{"market_id"},
	)

	// Insurance fund metrics
	c.InsuranceFundBalance = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "perpdex",
			Subsystem: "insurance_fund",
			Name:      "balance_usdc",
			Help:      "Insurance fund balance in USDC",
		},
		[]string{"fund_id"},
	)

	c.InsuranceFundInflow = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "perpdex",
			Subsystem: "insurance_fund",
			Name:      "inflow_usdc",
			Help:      "Total inflow to insurance fund in USDC",
		},
		[]string{"fund_id", "source"},
	)

	c.InsuranceFundOutflow = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "perpdex",
			Subsystem: "insurance_fund",
			Name:      "outflow_usdc",
			Help:      "Total outflow from insurance fund in USDC",
		},
		[]string{"fund_id", "reason"},
	)

	// ADL metrics
	c.ADLEventsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "perpdex",
			Subsystem: "adl",
			Name:      "events_total",
			Help:      "Total ADL events",
		},
		[]string{"market_id", "reason"},
	)

	c.ADLPositionsAffected = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "perpdex",
			Subsystem: "adl",
			Name:      "positions_affected",
			Help:      "Total positions affected by ADL",
		},
		[]string{"market_id"},
	)

	c.ADLValueDeleveraged = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "perpdex",
			Subsystem: "adl",
			Name:      "value_deleveraged_usdc",
			Help:      "Total value deleveraged in USDC",
		},
		[]string{"market_id"},
	)

	// Funding rate metrics
	c.FundingRate = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "perpdex",
			Subsystem: "funding",
			Name:      "rate",
			Help:      "Current funding rate",
		},
		[]string{"market_id"},
	)

	c.FundingPayments = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "perpdex",
			Subsystem: "funding",
			Name:      "payments_usdc",
			Help:      "Total funding payments in USDC",
		},
		[]string{"market_id", "direction"},
	)

	// Oracle metrics
	c.OraclePrice = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "perpdex",
			Subsystem: "oracle",
			Name:      "price",
			Help:      "Current oracle price",
		},
		[]string{"market_id", "price_type"},
	)

	c.OracleDeviation = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "perpdex",
			Subsystem: "oracle",
			Name:      "deviation",
			Help:      "Price deviation between sources",
		},
		[]string{"market_id"},
	)

	c.OracleSourceCount = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "perpdex",
			Subsystem: "oracle",
			Name:      "source_count",
			Help:      "Number of active oracle sources",
		},
		[]string{"market_id"},
	)

	c.OracleLatency = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "perpdex",
			Subsystem: "oracle",
			Name:      "latency_ms",
			Help:      "Oracle update latency in milliseconds",
			Buckets:   []float64{10, 50, 100, 250, 500, 1000, 2000},
		},
		[]string{"source"},
	)

	// WebSocket metrics
	c.WSConnectionsActive = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "perpdex",
			Subsystem: "websocket",
			Name:      "connections_active",
			Help:      "Number of active WebSocket connections",
		},
		[]string{},
	)

	c.WSMessagesTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "perpdex",
			Subsystem: "websocket",
			Name:      "messages_total",
			Help:      "Total WebSocket messages sent",
		},
		[]string{"channel"},
	)

	c.WSMessageLatency = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "perpdex",
			Subsystem: "websocket",
			Name:      "message_latency_ms",
			Help:      "WebSocket message latency in milliseconds",
			Buckets:   []float64{1, 5, 10, 25, 50, 100},
		},
		[]string{"channel"},
	)

	c.WSSubscriptions = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "perpdex",
			Subsystem: "websocket",
			Name:      "subscriptions",
			Help:      "Number of active subscriptions per channel",
		},
		[]string{"channel"},
	)

	// API metrics
	c.APIRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "perpdex",
			Subsystem: "api",
			Name:      "requests_total",
			Help:      "Total API requests",
		},
		[]string{"method", "path", "status"},
	)

	c.APIRequestLatency = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "perpdex",
			Subsystem: "api",
			Name:      "request_latency_ms",
			Help:      "API request latency in milliseconds",
			Buckets:   []float64{1, 5, 10, 25, 50, 100, 250, 500, 1000},
		},
		[]string{"method", "path"},
	)

	c.APIErrorsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "perpdex",
			Subsystem: "api",
			Name:      "errors_total",
			Help:      "Total API errors",
		},
		[]string{"method", "path", "error_type"},
	)

	c.RateLimitHits = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "perpdex",
			Subsystem: "api",
			Name:      "rate_limit_hits",
			Help:      "Total rate limit hits",
		},
		[]string{"limit_type"},
	)

	// System metrics
	c.ActiveUsers = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: "perpdex",
			Subsystem: "system",
			Name:      "active_users",
			Help:      "Number of active users",
		},
	)

	c.BlockHeight = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: "perpdex",
			Subsystem: "system",
			Name:      "block_height",
			Help:      "Current block height",
		},
	)

	c.BlockTime = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "perpdex",
			Subsystem: "system",
			Name:      "block_time_ms",
			Help:      "Block time in milliseconds",
			Buckets:   []float64{100, 250, 500, 1000, 2000, 5000},
		},
		[]string{},
	)

	c.TxPoolSize = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: "perpdex",
			Subsystem: "system",
			Name:      "tx_pool_size",
			Help:      "Transaction pool size",
		},
	)

	c.PeerCount = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: "perpdex",
			Subsystem: "system",
			Name:      "peer_count",
			Help:      "Number of connected peers",
		},
	)

	// Register all metrics
	c.registerAll()

	return c
}

// registerAll registers all metrics with Prometheus
func (c *Collector) registerAll() {
	// Order metrics
	prometheus.MustRegister(c.OrdersTotal)
	prometheus.MustRegister(c.OrdersActive)
	prometheus.MustRegister(c.OrderFillRate)
	prometheus.MustRegister(c.OrderLatency)

	// Matching engine metrics
	prometheus.MustRegister(c.MatchingLatency)
	prometheus.MustRegister(c.MatchingThroughput)
	prometheus.MustRegister(c.OrderbookDepth)
	prometheus.MustRegister(c.SpreadBps)

	// Trade metrics
	prometheus.MustRegister(c.TradesTotal)
	prometheus.MustRegister(c.TradeVolume)
	prometheus.MustRegister(c.TradeValue)

	// Position metrics
	prometheus.MustRegister(c.PositionsOpen)
	prometheus.MustRegister(c.PositionValue)
	prometheus.MustRegister(c.UnrealizedPnL)
	prometheus.MustRegister(c.Leverage)

	// Liquidation metrics
	prometheus.MustRegister(c.LiquidationsTotal)
	prometheus.MustRegister(c.LiquidationValue)
	prometheus.MustRegister(c.LiquidationDeficit)

	// Insurance fund metrics
	prometheus.MustRegister(c.InsuranceFundBalance)
	prometheus.MustRegister(c.InsuranceFundInflow)
	prometheus.MustRegister(c.InsuranceFundOutflow)

	// ADL metrics
	prometheus.MustRegister(c.ADLEventsTotal)
	prometheus.MustRegister(c.ADLPositionsAffected)
	prometheus.MustRegister(c.ADLValueDeleveraged)

	// Funding rate metrics
	prometheus.MustRegister(c.FundingRate)
	prometheus.MustRegister(c.FundingPayments)

	// Oracle metrics
	prometheus.MustRegister(c.OraclePrice)
	prometheus.MustRegister(c.OracleDeviation)
	prometheus.MustRegister(c.OracleSourceCount)
	prometheus.MustRegister(c.OracleLatency)

	// WebSocket metrics
	prometheus.MustRegister(c.WSConnectionsActive)
	prometheus.MustRegister(c.WSMessagesTotal)
	prometheus.MustRegister(c.WSMessageLatency)
	prometheus.MustRegister(c.WSSubscriptions)

	// API metrics
	prometheus.MustRegister(c.APIRequestsTotal)
	prometheus.MustRegister(c.APIRequestLatency)
	prometheus.MustRegister(c.APIErrorsTotal)
	prometheus.MustRegister(c.RateLimitHits)

	// System metrics
	prometheus.MustRegister(c.ActiveUsers)
	prometheus.MustRegister(c.BlockHeight)
	prometheus.MustRegister(c.BlockTime)
	prometheus.MustRegister(c.TxPoolSize)
	prometheus.MustRegister(c.PeerCount)
}

// ============ Recording Helpers ============

// RecordOrder records an order event
func (c *Collector) RecordOrder(marketID, side, orderType, status string) {
	c.OrdersTotal.WithLabelValues(marketID, side, orderType, status).Inc()
}

// RecordOrderLatency records order processing latency
func (c *Collector) RecordOrderLatency(marketID, orderType string, latencyMs float64) {
	c.OrderLatency.WithLabelValues(marketID, orderType).Observe(latencyMs)
}

// RecordTrade records a trade event
func (c *Collector) RecordTrade(marketID string, volume, value float64) {
	c.TradesTotal.WithLabelValues(marketID).Inc()
	c.TradeVolume.WithLabelValues(marketID).Add(volume)
	c.TradeValue.WithLabelValues(marketID).Add(value)
}

// RecordMatchingLatency records matching engine latency
func (c *Collector) RecordMatchingLatency(marketID string, latencyMs float64) {
	c.MatchingLatency.WithLabelValues(marketID).Observe(latencyMs)
}

// RecordLiquidation records a liquidation event
func (c *Collector) RecordLiquidation(marketID, liquidationType string, value, deficit float64) {
	c.LiquidationsTotal.WithLabelValues(marketID, liquidationType).Inc()
	c.LiquidationValue.WithLabelValues(marketID).Add(value)
	if deficit > 0 {
		c.LiquidationDeficit.WithLabelValues(marketID).Add(deficit)
	}
}

// RecordInsuranceFund records insurance fund changes
func (c *Collector) RecordInsuranceFund(fundID string, balance float64) {
	c.InsuranceFundBalance.WithLabelValues(fundID).Set(balance)
}

// RecordADL records an ADL event
func (c *Collector) RecordADL(marketID, reason string, positionsAffected int, valueDeleveraged float64) {
	c.ADLEventsTotal.WithLabelValues(marketID, reason).Inc()
	c.ADLPositionsAffected.WithLabelValues(marketID).Add(float64(positionsAffected))
	c.ADLValueDeleveraged.WithLabelValues(marketID).Add(valueDeleveraged)
}

// RecordFundingRate records the current funding rate
func (c *Collector) RecordFundingRate(marketID string, rate float64) {
	c.FundingRate.WithLabelValues(marketID).Set(rate)
}

// RecordAPIRequest records an API request
func (c *Collector) RecordAPIRequest(method, path, status string, latencyMs float64) {
	c.APIRequestsTotal.WithLabelValues(method, path, status).Inc()
	c.APIRequestLatency.WithLabelValues(method, path).Observe(latencyMs)
}

// RecordWSConnection records WebSocket connection changes
func (c *Collector) RecordWSConnection(delta int) {
	c.WSConnectionsActive.WithLabelValues().Add(float64(delta))
}

// RecordWSMessage records a WebSocket message
func (c *Collector) RecordWSMessage(channel string, latencyMs float64) {
	c.WSMessagesTotal.WithLabelValues(channel).Inc()
	c.WSMessageLatency.WithLabelValues(channel).Observe(latencyMs)
}

// UpdateSystemMetrics updates system-level metrics
func (c *Collector) UpdateSystemMetrics(blockHeight int64, txPoolSize int, peerCount int) {
	c.BlockHeight.Set(float64(blockHeight))
	c.TxPoolSize.Set(float64(txPoolSize))
	c.PeerCount.Set(float64(peerCount))
}

// ============ HTTP Handler ============

// Handler returns the Prometheus HTTP handler
func Handler() http.Handler {
	return promhttp.Handler()
}

// Timer is a helper for measuring latency
type Timer struct {
	start time.Time
}

// NewTimer creates a new timer
func NewTimer() *Timer {
	return &Timer{start: time.Now()}
}

// ElapsedMs returns the elapsed time in milliseconds
func (t *Timer) ElapsedMs() float64 {
	return float64(time.Since(t.start).Microseconds()) / 1000.0
}
