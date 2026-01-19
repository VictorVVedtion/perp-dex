package keeper

import (
	"container/heap"
	"sync"

	"cosmossdk.io/math"
	"github.com/openalpha/perp-dex/x/orderbook/types"
)

// ============================================================================
// HashMap + Queue OrderBook (dYdX Style)
// ============================================================================
// Key characteristics:
// - O(1) price level lookup via HashMap
// - O(log n) best price query via heap
// - O(n) order removal within price level (FIFO queue)
// - Similar to dYdX v4 orderbook implementation
// ============================================================================

// priceKey converts a decimal price to a string key for map indexing
func priceKey(p math.LegacyDec) string {
	return p.String()
}

// ============================================================================
// Price Heap - maintains sorted price levels
// ============================================================================

// priceHeapItem represents an item in the price heap
type priceHeapItem struct {
	price math.LegacyDec
	index int // index in heap for O(log n) removal
}

// priceHeap implements heap.Interface for sorted prices
type priceHeap struct {
	items   []*priceHeapItem
	keyToIndex map[string]int // priceKey -> heap index
	desc    bool             // true for max-heap (bids), false for min-heap (asks)
}

func newPriceHeap(desc bool) *priceHeap {
	return &priceHeap{
		items:      make([]*priceHeapItem, 0),
		keyToIndex: make(map[string]int),
		desc:       desc,
	}
}

func (h *priceHeap) Len() int { return len(h.items) }

func (h *priceHeap) Less(i, j int) bool {
	if h.desc {
		// Max-heap for bids (highest price first)
		return h.items[i].price.GT(h.items[j].price)
	}
	// Min-heap for asks (lowest price first)
	return h.items[i].price.LT(h.items[j].price)
}

func (h *priceHeap) Swap(i, j int) {
	h.items[i], h.items[j] = h.items[j], h.items[i]
	h.items[i].index = i
	h.items[j].index = j
	h.keyToIndex[priceKey(h.items[i].price)] = i
	h.keyToIndex[priceKey(h.items[j].price)] = j
}

func (h *priceHeap) Push(x interface{}) {
	item := x.(*priceHeapItem)
	item.index = len(h.items)
	h.items = append(h.items, item)
	h.keyToIndex[priceKey(item.price)] = item.index
}

func (h *priceHeap) Pop() interface{} {
	n := len(h.items)
	item := h.items[n-1]
	h.items[n-1] = nil // avoid memory leak
	h.items = h.items[:n-1]
	delete(h.keyToIndex, priceKey(item.price))
	item.index = -1
	return item
}

// Peek returns the top price without removing it
func (h *priceHeap) Peek() (math.LegacyDec, bool) {
	if len(h.items) == 0 {
		return math.LegacyDec{}, false
	}
	return h.items[0].price, true
}

// Contains checks if a price exists in the heap
func (h *priceHeap) Contains(price math.LegacyDec) bool {
	_, ok := h.keyToIndex[priceKey(price)]
	return ok
}

// RemoveByPrice removes a specific price from the heap
func (h *priceHeap) RemoveByPrice(price math.LegacyDec) {
	key := priceKey(price)
	if idx, ok := h.keyToIndex[key]; ok {
		heap.Remove(h, idx)
	}
}

// Clone creates a copy of the heap for iteration
func (h *priceHeap) Clone() *priceHeap {
	clone := &priceHeap{
		items:      make([]*priceHeapItem, len(h.items)),
		keyToIndex: make(map[string]int, len(h.keyToIndex)),
		desc:       h.desc,
	}
	for i, item := range h.items {
		clone.items[i] = &priceHeapItem{
			price: item.price,
			index: i,
		}
		clone.keyToIndex[priceKey(item.price)] = i
	}
	return clone
}

// ============================================================================
// Hash Book Side - one side of the order book (bids or asks)
// ============================================================================

type hashBookSide struct {
	levels map[string]*PriceLevelV2 // priceKey -> price level
	heap   *priceHeap               // sorted prices
}

func newHashBookSide(desc bool) *hashBookSide {
	return &hashBookSide{
		levels: make(map[string]*PriceLevelV2),
		heap:   newPriceHeap(desc),
	}
}

// Get returns the price level at the given price, or nil if not found
func (s *hashBookSide) Get(price math.LegacyDec) *PriceLevelV2 {
	return s.levels[priceKey(price)]
}

// Set adds or updates a price level
func (s *hashBookSide) Set(price math.LegacyDec, level *PriceLevelV2) {
	key := priceKey(price)
	if _, exists := s.levels[key]; !exists {
		// New price level - add to heap
		heap.Push(s.heap, &priceHeapItem{price: price})
	}
	s.levels[key] = level
}

// GetOrCreate returns the existing price level or creates a new one
func (s *hashBookSide) GetOrCreate(price math.LegacyDec) *PriceLevelV2 {
	level := s.Get(price)
	if level == nil {
		level = NewPriceLevelV2(price)
		s.Set(price, level)
	}
	return level
}

// Remove removes a price level
func (s *hashBookSide) Remove(price math.LegacyDec) {
	key := priceKey(price)
	if _, exists := s.levels[key]; exists {
		delete(s.levels, key)
		s.heap.RemoveByPrice(price)
	}
}

// Best returns the best (top) price level
func (s *hashBookSide) Best() *PriceLevelV2 {
	if price, ok := s.heap.Peek(); ok {
		return s.levels[priceKey(price)]
	}
	return nil
}

// Len returns the number of price levels
func (s *hashBookSide) Len() int {
	return len(s.levels)
}

// Iterate iterates over all price levels in sorted order
func (s *hashBookSide) Iterate(fn func(*PriceLevelV2) bool) {
	if s.Len() == 0 {
		return
	}

	// Clone heap for iteration without modifying original
	clone := s.heap.Clone()
	for clone.Len() > 0 {
		item := heap.Pop(clone).(*priceHeapItem)
		level := s.levels[priceKey(item.price)]
		if level != nil && !fn(level) {
			break
		}
	}
}

// ============================================================================
// OrderBookHashMap - complete order book using HashMap + Heap
// ============================================================================

// OrderBookHashMap is an order book implementation using HashMap for O(1) price lookup
// and a heap for O(1) best price access. This is similar to dYdX's approach.
type OrderBookHashMap struct {
	MarketID string
	Bids     *hashBookSide
	Asks     *hashBookSide
	mu       sync.RWMutex
}

// NewOrderBookHashMap creates a new HashMap-based order book
func NewOrderBookHashMap(marketID string) *OrderBookHashMap {
	return &OrderBookHashMap{
		MarketID: marketID,
		Bids:     newHashBookSide(true),  // max-heap for bids
		Asks:     newHashBookSide(false), // min-heap for asks
	}
}

// GetMarketID returns the market ID
func (ob *OrderBookHashMap) GetMarketID() string {
	return ob.MarketID
}

// getSide returns the appropriate side based on order side
func (ob *OrderBookHashMap) getSide(side types.Side) *hashBookSide {
	if side == types.SideBuy {
		return ob.Bids
	}
	return ob.Asks
}

// AddOrder adds an order to the order book - O(1) average, O(log n) for new price level
func (ob *OrderBookHashMap) AddOrder(order *types.Order) {
	ob.mu.Lock()
	defer ob.mu.Unlock()

	side := ob.getSide(order.Side)
	level := side.GetOrCreate(order.Price)
	level.AddOrder(order)
}

// RemoveOrder removes an order from the order book - O(n) within price level
func (ob *OrderBookHashMap) RemoveOrder(order *types.Order) *types.Order {
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

// RemoveOrderByID removes an order by ID - O(n) within price level
func (ob *OrderBookHashMap) RemoveOrderByID(orderID string, side types.Side, price math.LegacyDec) *types.Order {
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

// GetBestBid returns the best (highest) bid level - O(1)
func (ob *OrderBookHashMap) GetBestBid() *PriceLevelV2 {
	ob.mu.RLock()
	defer ob.mu.RUnlock()
	return ob.Bids.Best()
}

// GetBestAsk returns the best (lowest) ask level - O(1)
func (ob *OrderBookHashMap) GetBestAsk() *PriceLevelV2 {
	ob.mu.RLock()
	defer ob.mu.RUnlock()
	return ob.Asks.Best()
}

// GetBestLevels returns the best bid and ask levels - O(1)
func (ob *OrderBookHashMap) GetBestLevels() (bestBid, bestAsk *PriceLevelV2) {
	ob.mu.RLock()
	defer ob.mu.RUnlock()
	return ob.Bids.Best(), ob.Asks.Best()
}

// GetSpread returns the spread between best bid and ask
func (ob *OrderBookHashMap) GetSpread() math.LegacyDec {
	bestBid, bestAsk := ob.GetBestLevels()
	if bestBid == nil || bestAsk == nil {
		return math.LegacyZeroDec()
	}
	return bestAsk.Price.Sub(bestBid.Price)
}

// GetMidPrice returns the mid price
func (ob *OrderBookHashMap) GetMidPrice() math.LegacyDec {
	bestBid, bestAsk := ob.GetBestLevels()
	if bestBid == nil || bestAsk == nil {
		return math.LegacyZeroDec()
	}
	return bestBid.Price.Add(bestAsk.Price).QuoInt64(2)
}

// GetDepth returns the order book depth (number of price levels)
func (ob *OrderBookHashMap) GetDepth() (bidLevels, askLevels int) {
	ob.mu.RLock()
	defer ob.mu.RUnlock()
	return ob.Bids.Len(), ob.Asks.Len()
}

// GetBidLevels returns n best bid levels
func (ob *OrderBookHashMap) GetBidLevels(n int) []*PriceLevelV2 {
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

// GetAskLevels returns n best ask levels
func (ob *OrderBookHashMap) GetAskLevels(n int) []*PriceLevelV2 {
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
func (ob *OrderBookHashMap) IterateBids(fn func(level *PriceLevelV2) bool) {
	ob.mu.RLock()
	defer ob.mu.RUnlock()
	ob.Bids.Iterate(fn)
}

// IterateAsks iterates over all ask levels in price order (lowest first)
func (ob *OrderBookHashMap) IterateAsks(fn func(level *PriceLevelV2) bool) {
	ob.mu.RLock()
	defer ob.mu.RUnlock()
	ob.Asks.Iterate(fn)
}

// ToOrderBook converts to the standard types.OrderBook for compatibility
func (ob *OrderBookHashMap) ToOrderBook() *types.OrderBook {
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

// FromOrderBookToHashMap creates an OrderBookHashMap from a standard OrderBook
func FromOrderBookToHashMap(ob *types.OrderBook, orders map[string]*types.Order) *OrderBookHashMap {
	result := NewOrderBookHashMap(ob.MarketID)

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
func (ob *OrderBookHashMap) Clear() {
	ob.mu.Lock()
	defer ob.mu.Unlock()

	ob.Bids = newHashBookSide(true)
	ob.Asks = newHashBookSide(false)
}

// GetPriceLevel returns the price level at a specific price
func (ob *OrderBookHashMap) GetPriceLevel(price math.LegacyDec, side types.Side) *PriceLevelV2 {
	ob.mu.RLock()
	defer ob.mu.RUnlock()
	return ob.getSide(side).Get(price)
}

// UpdateOrderQuantity updates an order's quantity after partial fill
func (ob *OrderBookHashMap) UpdateOrderQuantity(order *types.Order) {
	ob.mu.Lock()
	defer ob.mu.Unlock()

	side := ob.getSide(order.Side)
	level := side.Get(order.Price)
	if level != nil {
		level.UpdateQuantity()
	}
}
