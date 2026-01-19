package matcher

import (
	"context"
	"fmt"
	"log"
	"sort"
	"sync"
	"time"

	"cosmossdk.io/math"
	"github.com/openalpha/perp-dex/x/orderbook/types"
)

// Config holds the matcher configuration
type Config struct {
	BatchSize     int           // Maximum trades per batch submission
	BatchInterval time.Duration // Time interval for batch submission
	WebSocketURL  string        // WebSocket URL for event listening
	ChainRPCURL   string        // Chain RPC URL for submission
}

// DefaultConfig returns the default matcher configuration
func DefaultConfig() *Config {
	return &Config{
		BatchSize:     100,
		BatchInterval: 500 * time.Millisecond,
		WebSocketURL:  "ws://localhost:26657/websocket",
		ChainRPCURL:   "http://localhost:26657",
	}
}

// OffchainMatcher is the main offchain matching engine
type OffchainMatcher struct {
	config     *Config
	cache      *OrderCache
	tradeBuffer *TradeBuffer
	submitter  TxSubmitter

	// Internal state
	orderBooks map[string]*types.OrderBook // marketID -> orderBook
	orders     map[string]*types.Order     // orderID -> order
	mu         sync.RWMutex

	// Event channel for simulated WebSocket events
	eventCh chan Event

	// Control channels
	stopCh chan struct{}
	wg     sync.WaitGroup
}

// Event represents an incoming event from the chain
type Event struct {
	Type      EventType
	Order     *types.Order
	MarketID  string
	Timestamp time.Time
}

// EventType represents the type of chain event
type EventType int

const (
	EventTypeNewOrder EventType = iota
	EventTypeCancelOrder
	EventTypeMarketUpdate
)

func (e EventType) String() string {
	switch e {
	case EventTypeNewOrder:
		return "new_order"
	case EventTypeCancelOrder:
		return "cancel_order"
	case EventTypeMarketUpdate:
		return "market_update"
	default:
		return "unknown"
	}
}

// NewOffchainMatcher creates a new offchain matcher instance
func NewOffchainMatcher(config *Config, submitter TxSubmitter) *OffchainMatcher {
	if config == nil {
		config = DefaultConfig()
	}
	if submitter == nil {
		submitter = NewMockSubmitter()
	}

	return &OffchainMatcher{
		config:      config,
		cache:       NewOrderCache(),
		tradeBuffer: NewTradeBuffer(config.BatchSize),
		submitter:   submitter,
		orderBooks:  make(map[string]*types.OrderBook),
		orders:      make(map[string]*types.Order),
		eventCh:     make(chan Event, 1000),
		stopCh:      make(chan struct{}),
	}
}

// Start starts the offchain matcher
func (m *OffchainMatcher) Start(ctx context.Context) error {
	log.Println("Starting offchain matcher...")

	// Start event listener
	m.wg.Add(1)
	go m.eventLoop(ctx)

	// Start batch submission loop
	m.wg.Add(1)
	go m.batchLoop(ctx)

	log.Println("Offchain matcher started")
	return nil
}

// Stop stops the offchain matcher
func (m *OffchainMatcher) Stop() error {
	log.Println("Stopping offchain matcher...")
	close(m.stopCh)
	m.wg.Wait()
	log.Println("Offchain matcher stopped")
	return nil
}

// eventLoop processes incoming events
func (m *OffchainMatcher) eventLoop(ctx context.Context) {
	defer m.wg.Done()

	for {
		select {
		case <-ctx.Done():
			return
		case <-m.stopCh:
			return
		case event := <-m.eventCh:
			if err := m.handleEvent(event); err != nil {
				log.Printf("Error handling event: %v", err)
			}
		}
	}
}

// batchLoop periodically submits trade batches to the chain
func (m *OffchainMatcher) batchLoop(ctx context.Context) {
	defer m.wg.Done()

	ticker := time.NewTicker(m.config.BatchInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			// Submit any remaining trades before stopping
			m.submitPendingTrades(ctx)
			return
		case <-m.stopCh:
			m.submitPendingTrades(ctx)
			return
		case <-ticker.C:
			m.submitPendingTrades(ctx)
		}
	}
}

// submitPendingTrades submits pending trades to the chain
func (m *OffchainMatcher) submitPendingTrades(ctx context.Context) {
	trades := m.tradeBuffer.Flush()
	if len(trades) == 0 {
		return
	}

	log.Printf("Submitting %d trades to chain...", len(trades))
	if err := m.submitter.SubmitTrades(ctx, trades); err != nil {
		log.Printf("Error submitting trades: %v", err)
		// Re-add trades to buffer for retry
		for _, trade := range trades {
			m.tradeBuffer.Add(trade)
		}
	}
}

// handleEvent handles an incoming event
func (m *OffchainMatcher) handleEvent(event Event) error {
	switch event.Type {
	case EventTypeNewOrder:
		return m.handleNewOrder(event.Order)
	case EventTypeCancelOrder:
		return m.handleCancelOrder(event.Order.OrderID)
	default:
		return fmt.Errorf("unknown event type: %v", event.Type)
	}
}

// handleNewOrder processes a new order
func (m *OffchainMatcher) handleNewOrder(order *types.Order) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Store order in cache
	m.cache.Set(order)
	m.orders[order.OrderID] = order

	// Get or create order book
	orderBook := m.getOrCreateOrderBook(order.MarketID)

	// Match the order
	trades, remainingQty := m.matchOrder(order, orderBook)

	// Add trades to buffer
	for _, trade := range trades {
		m.tradeBuffer.Add(trade)
	}

	// If remaining quantity, add to order book (limit orders only)
	if remainingQty.IsPositive() && order.OrderType == types.OrderTypeLimit {
		orderBook.AddOrder(order)
	}

	return nil
}

// handleCancelOrder cancels an order
func (m *OffchainMatcher) handleCancelOrder(orderID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	order, exists := m.orders[orderID]
	if !exists {
		return fmt.Errorf("order not found: %s", orderID)
	}

	if !order.IsActive() {
		return fmt.Errorf("order is not active: %s", orderID)
	}

	// Remove from order book
	orderBook := m.orderBooks[order.MarketID]
	if orderBook != nil {
		orderBook.RemoveOrder(order)
	}

	// Update order status
	order.Cancel()
	m.cache.Delete(orderID)
	delete(m.orders, orderID)

	return nil
}

// getOrCreateOrderBook gets or creates an order book for a market
func (m *OffchainMatcher) getOrCreateOrderBook(marketID string) *types.OrderBook {
	orderBook, exists := m.orderBooks[marketID]
	if !exists {
		orderBook = types.NewOrderBook(marketID)
		m.orderBooks[marketID] = orderBook
	}
	return orderBook
}

// matchOrder matches an incoming order against the order book
// Uses Price-Time Priority algorithm
func (m *OffchainMatcher) matchOrder(order *types.Order, orderBook *types.OrderBook) ([]*types.Trade, math.LegacyDec) {
	trades := make([]*types.Trade, 0)
	remainingQty := order.RemainingQty()

	// Get opposite side levels
	var oppositeLevels []*types.PriceLevel
	if order.Side == types.SideBuy {
		oppositeLevels = orderBook.Asks
	} else {
		oppositeLevels = orderBook.Bids
	}

	// Sort levels by price priority
	// For buy orders: ascending (lowest ask first)
	// For sell orders: descending (highest bid first)
	sortedLevels := make([]*types.PriceLevel, len(oppositeLevels))
	copy(sortedLevels, oppositeLevels)
	sort.Slice(sortedLevels, func(i, j int) bool {
		if order.Side == types.SideBuy {
			return sortedLevels[i].Price.LT(sortedLevels[j].Price)
		}
		return sortedLevels[i].Price.GT(sortedLevels[j].Price)
	})

	// Match against each price level
	for _, level := range sortedLevels {
		if remainingQty.IsZero() {
			break
		}

		// Check price compatibility
		if !m.isPriceCompatible(order, level.Price) {
			break
		}

		// Match against orders at this level (FIFO - time priority)
		orderIDsToRemove := make([]string, 0)
		for _, makerOrderID := range level.OrderIDs {
			if remainingQty.IsZero() {
				break
			}

			makerOrder, exists := m.orders[makerOrderID]
			if !exists || !makerOrder.IsActive() {
				orderIDsToRemove = append(orderIDsToRemove, makerOrderID)
				continue
			}

			// Calculate match quantity
			matchQty := math.LegacyMinDec(remainingQty, makerOrder.RemainingQty())
			matchPrice := level.Price // Maker's price

			// Calculate fees (using default rates for now)
			takerFee := m.calculateFee(matchQty, matchPrice, math.LegacyNewDecWithPrec(5, 4))  // 0.05%
			makerFee := m.calculateFee(matchQty, matchPrice, math.LegacyNewDecWithPrec(2, 4))  // 0.02%

			// Create trade
			tradeID := m.generateTradeID()
			trade := types.NewTrade(tradeID, order.MarketID, order, makerOrder, matchPrice, matchQty, takerFee, makerFee)
			trades = append(trades, trade)

			// Update order quantities
			if err := order.Fill(matchQty); err != nil {
				log.Printf("Error filling taker order: %v", err)
				continue
			}
			if err := makerOrder.Fill(matchQty); err != nil {
				log.Printf("Error filling maker order: %v", err)
				continue
			}

			// Update remaining quantity
			remainingQty = remainingQty.Sub(matchQty)

			// Update level quantity
			level.Quantity = level.Quantity.Sub(matchQty)

			// Mark filled maker orders for removal
			if makerOrder.IsFilled() {
				orderIDsToRemove = append(orderIDsToRemove, makerOrderID)
				m.cache.Delete(makerOrderID)
				delete(m.orders, makerOrderID)
			}
		}

		// Remove filled orders from level
		for _, id := range orderIDsToRemove {
			level.RemoveOrder(id, math.LegacyZeroDec())
		}
	}

	// Cleanup empty levels
	m.cleanupOrderBook(orderBook)

	return trades, remainingQty
}

// isPriceCompatible checks if the taker order can match at the given price
func (m *OffchainMatcher) isPriceCompatible(order *types.Order, levelPrice math.LegacyDec) bool {
	// Market orders match at any price
	if order.OrderType == types.OrderTypeMarket {
		return true
	}

	// Limit orders: buy must be >= ask, sell must be <= bid
	if order.Side == types.SideBuy {
		return order.Price.GTE(levelPrice)
	}
	return order.Price.LTE(levelPrice)
}

// calculateFee calculates the trading fee
func (m *OffchainMatcher) calculateFee(qty, price, feeRate math.LegacyDec) math.LegacyDec {
	if feeRate.IsZero() {
		return math.LegacyZeroDec()
	}
	return qty.Mul(price).Mul(feeRate)
}

// cleanupOrderBook removes empty price levels
func (m *OffchainMatcher) cleanupOrderBook(ob *types.OrderBook) {
	// Clean bids
	cleanBids := make([]*types.PriceLevel, 0, len(ob.Bids))
	for _, level := range ob.Bids {
		if !level.IsEmpty() {
			cleanBids = append(cleanBids, level)
		}
	}
	ob.Bids = cleanBids

	// Clean asks
	cleanAsks := make([]*types.PriceLevel, 0, len(ob.Asks))
	for _, level := range ob.Asks {
		if !level.IsEmpty() {
			cleanAsks = append(cleanAsks, level)
		}
	}
	ob.Asks = cleanAsks
}

// generateTradeID generates a unique trade ID
func (m *OffchainMatcher) generateTradeID() string {
	return fmt.Sprintf("trade_%d", time.Now().UnixNano())
}

// SubmitOrder submits an order to the matcher (simulated WebSocket)
func (m *OffchainMatcher) SubmitOrder(order *types.Order) {
	m.eventCh <- Event{
		Type:      EventTypeNewOrder,
		Order:     order,
		MarketID:  order.MarketID,
		Timestamp: time.Now(),
	}
}

// CancelOrder cancels an order in the matcher
func (m *OffchainMatcher) CancelOrder(orderID string) {
	order, exists := m.orders[orderID]
	if !exists {
		return
	}
	m.eventCh <- Event{
		Type:      EventTypeCancelOrder,
		Order:     order,
		MarketID:  order.MarketID,
		Timestamp: time.Now(),
	}
}

// GetOrderBook returns a copy of the order book for a market
func (m *OffchainMatcher) GetOrderBook(marketID string) *types.OrderBook {
	m.mu.RLock()
	defer m.mu.RUnlock()

	ob, exists := m.orderBooks[marketID]
	if !exists {
		return nil
	}

	// Return a copy to avoid race conditions
	copy := types.NewOrderBook(marketID)
	copy.Bids = make([]*types.PriceLevel, len(ob.Bids))
	copy.Asks = make([]*types.PriceLevel, len(ob.Asks))
	for i, level := range ob.Bids {
		copy.Bids[i] = &types.PriceLevel{
			Price:    level.Price,
			Quantity: level.Quantity,
			OrderIDs: append([]string{}, level.OrderIDs...),
		}
	}
	for i, level := range ob.Asks {
		copy.Asks[i] = &types.PriceLevel{
			Price:    level.Price,
			Quantity: level.Quantity,
			OrderIDs: append([]string{}, level.OrderIDs...),
		}
	}
	return copy
}

// GetOrder returns an order by ID
func (m *OffchainMatcher) GetOrder(orderID string) *types.Order {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.orders[orderID]
}

// Stats returns matcher statistics
type Stats struct {
	OrderCount      int
	OrderBookCount  int
	PendingTrades   int
	CacheSize       int
}

// GetStats returns current matcher statistics
func (m *OffchainMatcher) GetStats() Stats {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return Stats{
		OrderCount:     len(m.orders),
		OrderBookCount: len(m.orderBooks),
		PendingTrades:  m.tradeBuffer.Len(),
		CacheSize:      m.cache.Len(),
	}
}
