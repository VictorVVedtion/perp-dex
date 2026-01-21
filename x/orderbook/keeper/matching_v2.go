package keeper

import (
	"fmt"
	"sync"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/openalpha/perp-dex/x/orderbook/types"
)

// OrderBookCache caches order books and orders in memory
type OrderBookCache struct {
	orderBooks map[string]*OrderBookV2 // marketID -> OrderBookV2
	orders     map[string]*types.Order // orderID -> Order
	dirtyOBs   map[string]bool         // dirty order books
	dirtyOrds  map[string]bool         // dirty orders
	newTrades  []*types.Trade          // new trades to persist
	mu         sync.RWMutex
}

// NewOrderBookCache creates a new order book cache
func NewOrderBookCache() *OrderBookCache {
	return &OrderBookCache{
		orderBooks: make(map[string]*OrderBookV2),
		orders:     make(map[string]*types.Order),
		dirtyOBs:   make(map[string]bool),
		dirtyOrds:  make(map[string]bool),
		newTrades:  make([]*types.Trade, 0),
	}
}

// GetOrderBook gets an order book from cache, loading from store if needed
func (c *OrderBookCache) GetOrderBook(ctx sdk.Context, keeper *Keeper, marketID string) *OrderBookV2 {
	c.mu.Lock()
	defer c.mu.Unlock()

	if ob, ok := c.orderBooks[marketID]; ok {
		return ob
	}

	// Load from store
	storedOB := keeper.GetOrderBook(ctx, marketID)
	if storedOB == nil {
		ob := NewOrderBookV2(marketID)
		c.orderBooks[marketID] = ob
		return ob
	}

	// Convert to V2 with order lookup
	orders := make(map[string]*types.Order)
	for _, pl := range storedOB.Bids {
		for _, orderID := range pl.OrderIDs {
			if order := keeper.GetOrder(ctx, orderID); order != nil {
				orders[orderID] = order
				c.orders[orderID] = order
			}
		}
	}
	for _, pl := range storedOB.Asks {
		for _, orderID := range pl.OrderIDs {
			if order := keeper.GetOrder(ctx, orderID); order != nil {
				orders[orderID] = order
				c.orders[orderID] = order
			}
		}
	}

	ob := FromOrderBook(storedOB, orders)
	c.orderBooks[marketID] = ob
	return ob
}

// GetOrder gets an order from cache, loading from store if needed
func (c *OrderBookCache) GetOrder(ctx sdk.Context, keeper *Keeper, orderID string) *types.Order {
	c.mu.Lock()
	defer c.mu.Unlock()

	if order, ok := c.orders[orderID]; ok {
		return order
	}

	order := keeper.GetOrder(ctx, orderID)
	if order != nil {
		c.orders[orderID] = order
	}
	return order
}

// SetOrder updates an order in cache
func (c *OrderBookCache) SetOrder(order *types.Order) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.orders[order.OrderID] = order
	c.dirtyOrds[order.OrderID] = true
}

// MarkOrderBookDirty marks an order book as needing persistence
func (c *OrderBookCache) MarkOrderBookDirty(marketID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.dirtyOBs[marketID] = true
}

// AddTrade adds a trade to be persisted
func (c *OrderBookCache) AddTrade(trade *types.Trade) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.newTrades = append(c.newTrades, trade)
}

// Flush writes all dirty data to the store
func (c *OrderBookCache) Flush(ctx sdk.Context, keeper *Keeper) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Flush dirty order books
	for marketID := range c.dirtyOBs {
		if ob, ok := c.orderBooks[marketID]; ok {
			keeper.SetOrderBook(ctx, ob.ToOrderBook())
		}
	}
	c.dirtyOBs = make(map[string]bool)

	// Flush dirty orders
	for orderID := range c.dirtyOrds {
		if order, ok := c.orders[orderID]; ok {
			keeper.SetOrder(ctx, order)
		}
	}
	c.dirtyOrds = make(map[string]bool)

	// Flush trades
	for _, trade := range c.newTrades {
		keeper.SetTrade(ctx, trade)
	}
	c.newTrades = make([]*types.Trade, 0)

	return nil
}

// Clear clears the cache
func (c *OrderBookCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.orderBooks = make(map[string]*OrderBookV2)
	c.orders = make(map[string]*types.Order)
	c.dirtyOBs = make(map[string]bool)
	c.dirtyOrds = make(map[string]bool)
	c.newTrades = make([]*types.Trade, 0)
}

// MatchingEngineV2 is an optimized matching engine with memory caching
type MatchingEngineV2 struct {
	keeper *Keeper
	cache  *OrderBookCache
}

// NewMatchingEngineV2 creates a new optimized matching engine
func NewMatchingEngineV2(keeper *Keeper) *MatchingEngineV2 {
	return &MatchingEngineV2{
		keeper: keeper,
		cache:  NewOrderBookCache(),
	}
}

// NewMatchingEngineV2WithCache creates a matching engine with provided cache
func NewMatchingEngineV2WithCache(keeper *Keeper, cache *OrderBookCache) *MatchingEngineV2 {
	return &MatchingEngineV2{
		keeper: keeper,
		cache:  cache,
	}
}

// MatchResultV2 contains the result of order matching
type MatchResultV2 struct {
	Trades               []*types.Trade
	TradesWithSettlement []*types.TradeWithSettlement
	FilledQty            math.LegacyDec
	AvgPrice             math.LegacyDec
	RemainingQty         math.LegacyDec
}

// ToMatchResult converts to standard MatchResult
func (r *MatchResultV2) ToMatchResult() *MatchResult {
	return &MatchResult{
		Trades:       r.Trades,
		FilledQty:    r.FilledQty,
		AvgPrice:     r.AvgPrice,
		RemainingQty: r.RemainingQty,
	}
}

// Match attempts to match an incoming order against the order book
// CRITICAL FIX: Uses write lock to prevent concurrent modification during matching
func (me *MatchingEngineV2) Match(ctx sdk.Context, order *types.Order) (*MatchResultV2, error) {
	orderBook := me.cache.GetOrderBook(ctx, me.keeper, order.MarketID)

	result := &MatchResultV2{
		Trades:               make([]*types.Trade, 0),
		TradesWithSettlement: make([]*types.TradeWithSettlement, 0),
		FilledQty:            math.LegacyZeroDec(),
		AvgPrice:             math.LegacyZeroDec(),
		RemainingQty:         order.RemainingQty(),
	}

	// Track total value for average price calculation
	totalValue := math.LegacyZeroDec()

	// CRITICAL: Acquire write lock for the entire matching operation
	// This prevents concurrent modification during iteration
	orderBook.Lock()
	defer orderBook.Unlock()

	// Determine which side to match against (use unsafe iterators since we hold the lock)
	var iterateFunc func(fn func(level *PriceLevelV2) bool)
	if order.Side == types.SideBuy {
		iterateFunc = orderBook.IterateAsksUnsafe
	} else {
		iterateFunc = orderBook.IterateBidsUnsafe
	}

	// Levels to update after matching
	levelsToRemove := make([]*PriceLevelV2, 0)

	// Match against price levels
	iterateFunc(func(level *PriceLevelV2) bool {
		if result.RemainingQty.IsZero() {
			return false // Stop iteration
		}

		// Check price compatibility
		if !me.isPriceCompatible(order, level.Price) {
			return false // Stop - no more compatible prices
		}

		// Match against orders at this level (FIFO)
		ordersToRemove := make([]string, 0)

		for _, makerOrder := range level.Orders {
			if result.RemainingQty.IsZero() {
				break
			}

			if !makerOrder.IsActive() {
				ordersToRemove = append(ordersToRemove, makerOrder.OrderID)
				continue
			}

			// Calculate match quantity
			matchQty := math.LegacyMinDec(result.RemainingQty, makerOrder.RemainingQty())
			matchPrice := level.Price

			// Calculate fees
			market := me.keeper.perpetualKeeper.GetMarket(ctx, order.MarketID)
			takerFee := me.calculateFee(matchQty, matchPrice, market.TakerFeeRate)
			makerFee := me.calculateFee(matchQty, matchPrice, market.MakerFeeRate)

			// Create trade
			tradeID := me.keeper.generateTradeID(ctx)
			trade := types.NewTrade(tradeID, order.MarketID, order, makerOrder, matchPrice, matchQty, takerFee, makerFee)
			result.Trades = append(result.Trades, trade)
			result.TradesWithSettlement = append(result.TradesWithSettlement, types.NewTradeWithSettlement(trade))
			me.cache.AddTrade(trade)

			// Update quantities
			if err := order.Fill(matchQty); err != nil {
				return false
			}
			if err := makerOrder.Fill(matchQty); err != nil {
				return false
			}

			// Update tracking
			result.FilledQty = result.FilledQty.Add(matchQty)
			result.RemainingQty = result.RemainingQty.Sub(matchQty)
			totalValue = totalValue.Add(matchQty.Mul(matchPrice))

			// Mark order as dirty
			me.cache.SetOrder(makerOrder)

			// Track filled orders for removal
			if makerOrder.IsFilled() {
				ordersToRemove = append(ordersToRemove, makerOrder.OrderID)
			}

			// Emit trade event
			me.keeper.emitTradeEvent(ctx, trade)
		}

		// Remove filled orders from level
		for _, orderID := range ordersToRemove {
			level.RemoveOrder(orderID)
		}

		// Track empty levels for removal
		if level.IsEmpty() {
			levelsToRemove = append(levelsToRemove, level)
		} else {
			level.UpdateQuantity()
		}

		return true // Continue iteration
	})

	// Remove empty levels (use unsafe since we hold the lock)
	for _, level := range levelsToRemove {
		if order.Side == types.SideBuy {
			orderBook.RemoveUnsafe(level.Price, types.SideSell)
		} else {
			orderBook.RemoveUnsafe(level.Price, types.SideBuy)
		}
	}

	// Calculate average price
	if result.FilledQty.IsPositive() {
		result.AvgPrice = totalValue.Quo(result.FilledQty)
	}

	// Mark order book as dirty
	me.cache.MarkOrderBookDirty(order.MarketID)

	return result, nil
}

// isPriceCompatible checks if the order can match at the given price
func (me *MatchingEngineV2) isPriceCompatible(order *types.Order, levelPrice math.LegacyDec) bool {
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
func (me *MatchingEngineV2) calculateFee(qty, price, feeRate math.LegacyDec) math.LegacyDec {
	if feeRate.IsZero() {
		return math.LegacyZeroDec()
	}
	return qty.Mul(price).Mul(feeRate)
}

// ProcessOrderOptimized is the optimized entry point for order processing
func (me *MatchingEngineV2) ProcessOrderOptimized(ctx sdk.Context, order *types.Order) (*MatchResultV2, error) {
	// Try to match the order
	result, err := me.Match(ctx, order)
	if err != nil {
		return nil, err
	}

	// If there's remaining quantity and it's a limit order, add to book
	if result.RemainingQty.IsPositive() && order.OrderType == types.OrderTypeLimit {
		orderBook := me.cache.GetOrderBook(ctx, me.keeper, order.MarketID)
		orderBook.AddOrder(order)
		me.cache.MarkOrderBookDirty(order.MarketID)
	} else if order.IsActive() && order.OrderType == types.OrderTypeMarket {
		// Market order with unfilled quantity - cancel the rest
		order.Cancel()
	}

	// Save the taker order
	me.cache.SetOrder(order)

	return result, nil
}

// CancelOrderOptimized cancels an order with cache support
func (me *MatchingEngineV2) CancelOrderOptimized(ctx sdk.Context, orderID string) (*types.Order, error) {
	order := me.cache.GetOrder(ctx, me.keeper, orderID)
	if order == nil {
		return nil, fmt.Errorf("order not found: %s", orderID)
	}

	if !order.IsActive() {
		return nil, fmt.Errorf("order is not active: %s", orderID)
	}

	// Remove from order book
	orderBook := me.cache.GetOrderBook(ctx, me.keeper, order.MarketID)
	orderBook.RemoveOrder(order)
	me.cache.MarkOrderBookDirty(order.MarketID)

	// Cancel the order
	order.Cancel()
	me.cache.SetOrder(order)

	return order, nil
}

// Flush writes all cached data to the store
func (me *MatchingEngineV2) Flush(ctx sdk.Context) error {
	return me.cache.Flush(ctx, me.keeper)
}

// GetCache returns the underlying cache
func (me *MatchingEngineV2) GetCache() *OrderBookCache {
	return me.cache
}

// ProcessBatch processes a batch of orders with single flush at the end
func (me *MatchingEngineV2) ProcessBatch(ctx sdk.Context, orders []*types.Order) ([]*MatchResultV2, error) {
	results := make([]*MatchResultV2, 0, len(orders))

	for _, order := range orders {
		result, err := me.ProcessOrderOptimized(ctx, order)
		if err != nil {
			return results, fmt.Errorf("failed to process order %s: %w", order.OrderID, err)
		}
		results = append(results, result)
	}

	// Single flush at the end
	if err := me.Flush(ctx); err != nil {
		return results, fmt.Errorf("failed to flush cache: %w", err)
	}

	return results, nil
}

// GetOrderBookV2 returns the cached order book
func (me *MatchingEngineV2) GetOrderBookV2(ctx sdk.Context, marketID string) *OrderBookV2 {
	return me.cache.GetOrderBook(ctx, me.keeper, marketID)
}
