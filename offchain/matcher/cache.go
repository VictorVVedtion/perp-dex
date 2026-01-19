package matcher

import (
	"sync"

	"github.com/openalpha/perp-dex/x/orderbook/types"
)

// OrderCache is a thread-safe cache for orders
type OrderCache struct {
	orders map[string]*types.Order
	mu     sync.RWMutex
}

// NewOrderCache creates a new order cache
func NewOrderCache() *OrderCache {
	return &OrderCache{
		orders: make(map[string]*types.Order),
	}
}

// Get retrieves an order from the cache
func (c *OrderCache) Get(orderID string) (*types.Order, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	order, exists := c.orders[orderID]
	return order, exists
}

// Set stores an order in the cache
func (c *OrderCache) Set(order *types.Order) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.orders[order.OrderID] = order
}

// Delete removes an order from the cache
func (c *OrderCache) Delete(orderID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.orders, orderID)
}

// Len returns the number of orders in the cache
func (c *OrderCache) Len() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.orders)
}

// Clear removes all orders from the cache
func (c *OrderCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.orders = make(map[string]*types.Order)
}

// GetAll returns all orders in the cache
func (c *OrderCache) GetAll() []*types.Order {
	c.mu.RLock()
	defer c.mu.RUnlock()
	orders := make([]*types.Order, 0, len(c.orders))
	for _, order := range c.orders {
		orders = append(orders, order)
	}
	return orders
}

// GetByMarket returns all orders for a specific market
func (c *OrderCache) GetByMarket(marketID string) []*types.Order {
	c.mu.RLock()
	defer c.mu.RUnlock()
	orders := make([]*types.Order, 0)
	for _, order := range c.orders {
		if order.MarketID == marketID {
			orders = append(orders, order)
		}
	}
	return orders
}

// GetByTrader returns all orders for a specific trader
func (c *OrderCache) GetByTrader(trader string) []*types.Order {
	c.mu.RLock()
	defer c.mu.RUnlock()
	orders := make([]*types.Order, 0)
	for _, order := range c.orders {
		if order.Trader == trader {
			orders = append(orders, order)
		}
	}
	return orders
}

// GetActiveOrders returns all active orders
func (c *OrderCache) GetActiveOrders() []*types.Order {
	c.mu.RLock()
	defer c.mu.RUnlock()
	orders := make([]*types.Order, 0)
	for _, order := range c.orders {
		if order.IsActive() {
			orders = append(orders, order)
		}
	}
	return orders
}

// TradeBuffer is a thread-safe buffer for trades pending submission
type TradeBuffer struct {
	trades   []*types.Trade
	maxSize  int
	mu       sync.Mutex
}

// NewTradeBuffer creates a new trade buffer with the given max size
func NewTradeBuffer(maxSize int) *TradeBuffer {
	if maxSize <= 0 {
		maxSize = 100
	}
	return &TradeBuffer{
		trades:  make([]*types.Trade, 0, maxSize),
		maxSize: maxSize,
	}
}

// Add adds a trade to the buffer
func (b *TradeBuffer) Add(trade *types.Trade) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.trades = append(b.trades, trade)
}

// AddBatch adds multiple trades to the buffer
func (b *TradeBuffer) AddBatch(trades []*types.Trade) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.trades = append(b.trades, trades...)
}

// Flush returns all trades and clears the buffer
func (b *TradeBuffer) Flush() []*types.Trade {
	b.mu.Lock()
	defer b.mu.Unlock()
	trades := b.trades
	b.trades = make([]*types.Trade, 0, b.maxSize)
	return trades
}

// FlushBatch returns up to maxSize trades and removes them from the buffer
func (b *TradeBuffer) FlushBatch() []*types.Trade {
	b.mu.Lock()
	defer b.mu.Unlock()

	if len(b.trades) == 0 {
		return nil
	}

	count := b.maxSize
	if len(b.trades) < count {
		count = len(b.trades)
	}

	batch := b.trades[:count]
	b.trades = b.trades[count:]
	return batch
}

// Len returns the number of trades in the buffer
func (b *TradeBuffer) Len() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return len(b.trades)
}

// IsFull returns true if the buffer is at or above max size
func (b *TradeBuffer) IsFull() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	return len(b.trades) >= b.maxSize
}

// Clear removes all trades from the buffer
func (b *TradeBuffer) Clear() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.trades = make([]*types.Trade, 0, b.maxSize)
}

// Peek returns the trades without removing them (for inspection)
func (b *TradeBuffer) Peek() []*types.Trade {
	b.mu.Lock()
	defer b.mu.Unlock()
	result := make([]*types.Trade, len(b.trades))
	copy(result, b.trades)
	return result
}

// MarketCache is a thread-safe cache for market data
type MarketCache struct {
	markets map[string]*MarketInfo
	mu      sync.RWMutex
}

// MarketInfo holds cached market information
type MarketInfo struct {
	MarketID     string
	BaseDenom    string
	QuoteDenom   string
	TakerFeeRate string
	MakerFeeRate string
	MinOrderSize string
	TickSize     string
}

// NewMarketCache creates a new market cache
func NewMarketCache() *MarketCache {
	return &MarketCache{
		markets: make(map[string]*MarketInfo),
	}
}

// Get retrieves market info from the cache
func (c *MarketCache) Get(marketID string) (*MarketInfo, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	info, exists := c.markets[marketID]
	return info, exists
}

// Set stores market info in the cache
func (c *MarketCache) Set(info *MarketInfo) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.markets[info.MarketID] = info
}

// Delete removes market info from the cache
func (c *MarketCache) Delete(marketID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.markets, marketID)
}

// GetAll returns all market info in the cache
func (c *MarketCache) GetAll() []*MarketInfo {
	c.mu.RLock()
	defer c.mu.RUnlock()
	markets := make([]*MarketInfo, 0, len(c.markets))
	for _, info := range c.markets {
		markets = append(markets, info)
	}
	return markets
}

// Len returns the number of markets in the cache
func (c *MarketCache) Len() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.markets)
}
