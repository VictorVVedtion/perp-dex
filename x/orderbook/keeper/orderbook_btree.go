package keeper

import (
	"sync"

	"cosmossdk.io/math"
	"github.com/google/btree"
	"github.com/openalpha/perp-dex/x/orderbook/types"
)

// ============================================================================
// B+Tree OrderBook (CEX Style - Bybit, Binance, etc.)
// ============================================================================
// Key characteristics:
// - O(log n) for all operations (insert, delete, lookup)
// - Efficient range queries O(log n + k)
// - Cache-friendly due to B-tree node structure
// - Similar to traditional CEX implementations
// ============================================================================

const btreeDegree = 32 // B-tree degree, affects node size and cache efficiency

// priceLevelItem wraps a price level for use in btree
// Implements btree.Item interface
type priceLevelItem struct {
	price math.LegacyDec
	level *PriceLevelV2
}

// Less implements btree.Item interface - ascending order by price
func (a *priceLevelItem) Less(b btree.Item) bool {
	return a.price.LT(b.(*priceLevelItem).price)
}

// ============================================================================
// BTree Side - one side of the order book (bids or asks)
// ============================================================================

type btreeSide struct {
	tree *btree.BTree
	desc bool // true for bids (iterate descending), false for asks (iterate ascending)
}

func newBTreeSide(desc bool) *btreeSide {
	return &btreeSide{
		tree: btree.New(btreeDegree),
		desc: desc,
	}
}

// Get returns the price level at the given price, or nil if not found
func (s *btreeSide) Get(price math.LegacyDec) *PriceLevelV2 {
	item := s.tree.Get(&priceLevelItem{price: price})
	if item == nil {
		return nil
	}
	return item.(*priceLevelItem).level
}

// Set adds or updates a price level
func (s *btreeSide) Set(price math.LegacyDec, level *PriceLevelV2) {
	s.tree.ReplaceOrInsert(&priceLevelItem{
		price: price,
		level: level,
	})
}

// GetOrCreate returns the existing price level or creates a new one
func (s *btreeSide) GetOrCreate(price math.LegacyDec) *PriceLevelV2 {
	level := s.Get(price)
	if level == nil {
		level = NewPriceLevelV2(price)
		s.Set(price, level)
	}
	return level
}

// Remove removes a price level
func (s *btreeSide) Remove(price math.LegacyDec) {
	s.tree.Delete(&priceLevelItem{price: price})
}

// Best returns the best price level
// For bids (desc=true): returns the highest price (Max)
// For asks (desc=false): returns the lowest price (Min)
func (s *btreeSide) Best() *PriceLevelV2 {
	var item btree.Item
	if s.desc {
		item = s.tree.Max()
	} else {
		item = s.tree.Min()
	}
	if item == nil {
		return nil
	}
	return item.(*priceLevelItem).level
}

// Len returns the number of price levels
func (s *btreeSide) Len() int {
	return s.tree.Len()
}

// Iterate iterates over all price levels in sorted order
// For bids: descending (highest to lowest)
// For asks: ascending (lowest to highest)
func (s *btreeSide) Iterate(fn func(*PriceLevelV2) bool) {
	if s.desc {
		// Descend for bids (highest first)
		s.tree.Descend(func(item btree.Item) bool {
			return fn(item.(*priceLevelItem).level)
		})
	} else {
		// Ascend for asks (lowest first)
		s.tree.Ascend(func(item btree.Item) bool {
			return fn(item.(*priceLevelItem).level)
		})
	}
}

// IterateRange iterates over price levels within a range
func (s *btreeSide) IterateRange(minPrice, maxPrice math.LegacyDec, fn func(*PriceLevelV2) bool) {
	minItem := &priceLevelItem{price: minPrice}
	maxItem := &priceLevelItem{price: maxPrice}

	if s.desc {
		// For bids, iterate from max to min (descending)
		s.tree.DescendRange(maxItem, minItem, func(item btree.Item) bool {
			return fn(item.(*priceLevelItem).level)
		})
	} else {
		// For asks, iterate from min to max (ascending)
		s.tree.AscendRange(minItem, maxItem, func(item btree.Item) bool {
			return fn(item.(*priceLevelItem).level)
		})
	}
}

// ============================================================================
// OrderBookBTree - complete order book using B+Tree
// ============================================================================

// OrderBookBTree is an order book implementation using B+Tree for O(log n) operations
// and efficient range queries. This is similar to traditional CEX implementations.
type OrderBookBTree struct {
	MarketID string
	Bids     *btreeSide
	Asks     *btreeSide
	mu       sync.RWMutex
}

// NewOrderBookBTree creates a new B+Tree-based order book
func NewOrderBookBTree(marketID string) *OrderBookBTree {
	return &OrderBookBTree{
		MarketID: marketID,
		Bids:     newBTreeSide(true),  // descending for bids
		Asks:     newBTreeSide(false), // ascending for asks
	}
}

// GetMarketID returns the market ID
func (ob *OrderBookBTree) GetMarketID() string {
	return ob.MarketID
}

// getSide returns the appropriate side based on order side
func (ob *OrderBookBTree) getSide(side types.Side) *btreeSide {
	if side == types.SideBuy {
		return ob.Bids
	}
	return ob.Asks
}

// AddOrder adds an order to the order book - O(log n)
func (ob *OrderBookBTree) AddOrder(order *types.Order) {
	ob.mu.Lock()
	defer ob.mu.Unlock()

	side := ob.getSide(order.Side)
	level := side.GetOrCreate(order.Price)
	level.AddOrder(order)
}

// RemoveOrder removes an order from the order book - O(log n) for tree + O(n) within level
func (ob *OrderBookBTree) RemoveOrder(order *types.Order) *types.Order {
	ob.mu.Lock()
	defer ob.mu.Unlock()

	side := ob.getSide(order.Side)
	level := side.Get(order.Price)
	if level == nil {
		return nil
	}

	removed := level.RemoveOrder(order.OrderID)
	if level.IsEmpty() {
		side.Remove(order.Price)
	}
	return removed
}

// RemoveOrderByID removes an order by ID - O(log n) for tree + O(n) within level
func (ob *OrderBookBTree) RemoveOrderByID(orderID string, side types.Side, price math.LegacyDec) *types.Order {
	ob.mu.Lock()
	defer ob.mu.Unlock()

	bookSide := ob.getSide(side)
	level := bookSide.Get(price)
	if level == nil {
		return nil
	}

	removed := level.RemoveOrder(orderID)
	if level.IsEmpty() {
		bookSide.Remove(price)
	}
	return removed
}

// GetBestBid returns the best (highest) bid level - O(log n)
func (ob *OrderBookBTree) GetBestBid() *PriceLevelV2 {
	ob.mu.RLock()
	defer ob.mu.RUnlock()
	return ob.Bids.Best()
}

// GetBestAsk returns the best (lowest) ask level - O(log n)
func (ob *OrderBookBTree) GetBestAsk() *PriceLevelV2 {
	ob.mu.RLock()
	defer ob.mu.RUnlock()
	return ob.Asks.Best()
}

// GetBestLevels returns the best bid and ask levels
func (ob *OrderBookBTree) GetBestLevels() (bestBid, bestAsk *PriceLevelV2) {
	ob.mu.RLock()
	defer ob.mu.RUnlock()
	return ob.Bids.Best(), ob.Asks.Best()
}

// GetSpread returns the spread between best bid and ask
func (ob *OrderBookBTree) GetSpread() math.LegacyDec {
	bestBid, bestAsk := ob.GetBestLevels()
	if bestBid == nil || bestAsk == nil {
		return math.LegacyZeroDec()
	}
	return bestAsk.Price.Sub(bestBid.Price)
}

// GetMidPrice returns the mid price
func (ob *OrderBookBTree) GetMidPrice() math.LegacyDec {
	bestBid, bestAsk := ob.GetBestLevels()
	if bestBid == nil || bestAsk == nil {
		return math.LegacyZeroDec()
	}
	return bestBid.Price.Add(bestAsk.Price).QuoInt64(2)
}

// GetDepth returns the order book depth (number of price levels)
func (ob *OrderBookBTree) GetDepth() (bidLevels, askLevels int) {
	ob.mu.RLock()
	defer ob.mu.RUnlock()
	return ob.Bids.Len(), ob.Asks.Len()
}

// GetBidLevels returns n best bid levels - O(log n + k)
func (ob *OrderBookBTree) GetBidLevels(n int) []*PriceLevelV2 {
	ob.mu.RLock()
	defer ob.mu.RUnlock()

	levels := make([]*PriceLevelV2, 0, n)
	count := 0
	ob.Bids.Iterate(func(level *PriceLevelV2) bool {
		if count >= n {
			return false
		}
		levels = append(levels, level)
		count++
		return true
	})
	return levels
}

// GetAskLevels returns n best ask levels - O(log n + k)
func (ob *OrderBookBTree) GetAskLevels(n int) []*PriceLevelV2 {
	ob.mu.RLock()
	defer ob.mu.RUnlock()

	levels := make([]*PriceLevelV2, 0, n)
	count := 0
	ob.Asks.Iterate(func(level *PriceLevelV2) bool {
		if count >= n {
			return false
		}
		levels = append(levels, level)
		count++
		return true
	})
	return levels
}

// IterateBids iterates over all bid levels in price order (highest first)
func (ob *OrderBookBTree) IterateBids(fn func(level *PriceLevelV2) bool) {
	ob.mu.RLock()
	defer ob.mu.RUnlock()
	ob.Bids.Iterate(fn)
}

// IterateAsks iterates over all ask levels in price order (lowest first)
func (ob *OrderBookBTree) IterateAsks(fn func(level *PriceLevelV2) bool) {
	ob.mu.RLock()
	defer ob.mu.RUnlock()
	ob.Asks.Iterate(fn)
}

// IterateBidsRange iterates over bid levels within a price range
func (ob *OrderBookBTree) IterateBidsRange(minPrice, maxPrice math.LegacyDec, fn func(level *PriceLevelV2) bool) {
	ob.mu.RLock()
	defer ob.mu.RUnlock()
	ob.Bids.IterateRange(minPrice, maxPrice, fn)
}

// IterateAsksRange iterates over ask levels within a price range
func (ob *OrderBookBTree) IterateAsksRange(minPrice, maxPrice math.LegacyDec, fn func(level *PriceLevelV2) bool) {
	ob.mu.RLock()
	defer ob.mu.RUnlock()
	ob.Asks.IterateRange(minPrice, maxPrice, fn)
}

// ToOrderBook converts to the standard types.OrderBook for compatibility
func (ob *OrderBookBTree) ToOrderBook() *types.OrderBook {
	ob.mu.RLock()
	defer ob.mu.RUnlock()

	result := types.NewOrderBook(ob.MarketID)

	// Convert bids
	ob.Bids.Iterate(func(level *PriceLevelV2) bool {
		result.Bids = append(result.Bids, level.ToPriceLevel())
		return true
	})

	// Convert asks
	ob.Asks.Iterate(func(level *PriceLevelV2) bool {
		result.Asks = append(result.Asks, level.ToPriceLevel())
		return true
	})

	return result
}

// FromOrderBookToBTree creates an OrderBookBTree from a standard OrderBook
func FromOrderBookToBTree(ob *types.OrderBook, orders map[string]*types.Order) *OrderBookBTree {
	result := NewOrderBookBTree(ob.MarketID)

	// Convert bids
	for _, pl := range ob.Bids {
		level := NewPriceLevelV2(pl.Price)
		for _, orderID := range pl.OrderIDs {
			if order, ok := orders[orderID]; ok {
				level.Orders = append(level.Orders, order)
			}
		}
		level.Quantity = pl.Quantity
		if !level.IsEmpty() {
			result.Bids.Set(pl.Price, level)
		}
	}

	// Convert asks
	for _, pl := range ob.Asks {
		level := NewPriceLevelV2(pl.Price)
		for _, orderID := range pl.OrderIDs {
			if order, ok := orders[orderID]; ok {
				level.Orders = append(level.Orders, order)
			}
		}
		level.Quantity = pl.Quantity
		if !level.IsEmpty() {
			result.Asks.Set(pl.Price, level)
		}
	}

	return result
}

// Clear removes all orders from the order book
func (ob *OrderBookBTree) Clear() {
	ob.mu.Lock()
	defer ob.mu.Unlock()

	ob.Bids = newBTreeSide(true)
	ob.Asks = newBTreeSide(false)
}

// GetPriceLevel returns the price level at a specific price - O(log n)
func (ob *OrderBookBTree) GetPriceLevel(price math.LegacyDec, side types.Side) *PriceLevelV2 {
	ob.mu.RLock()
	defer ob.mu.RUnlock()
	return ob.getSide(side).Get(price)
}

// UpdateOrderQuantity updates an order's quantity after partial fill
func (ob *OrderBookBTree) UpdateOrderQuantity(order *types.Order) {
	ob.mu.Lock()
	defer ob.mu.Unlock()

	side := ob.getSide(order.Side)
	level := side.Get(order.Price)
	if level != nil {
		level.UpdateQuantity()
	}
}
