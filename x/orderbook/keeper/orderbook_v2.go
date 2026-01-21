package keeper

import (
	"sync"

	"cosmossdk.io/math"
	"github.com/huandu/skiplist"
	"github.com/openalpha/perp-dex/x/orderbook/types"
)

// PriceLevelV2 represents a price level with orders in FIFO queue
type PriceLevelV2 struct {
	Price    math.LegacyDec
	Quantity math.LegacyDec
	Orders   []*types.Order // Orders in FIFO order
}

// NewPriceLevelV2 creates a new price level
func NewPriceLevelV2(price math.LegacyDec) *PriceLevelV2 {
	return &PriceLevelV2{
		Price:    price,
		Quantity: math.LegacyZeroDec(),
		Orders:   make([]*types.Order, 0),
	}
}

// AddOrder adds an order to the price level (FIFO)
func (pl *PriceLevelV2) AddOrder(order *types.Order) {
	pl.Orders = append(pl.Orders, order)
	pl.Quantity = pl.Quantity.Add(order.RemainingQty())
}

// RemoveOrder removes an order from the price level
func (pl *PriceLevelV2) RemoveOrder(orderID string) *types.Order {
	for i, o := range pl.Orders {
		if o.OrderID == orderID {
			pl.Orders = append(pl.Orders[:i], pl.Orders[i+1:]...)
			pl.Quantity = pl.Quantity.Sub(o.RemainingQty())
			return o
		}
	}
	return nil
}

// UpdateQuantity recalculates the total quantity
func (pl *PriceLevelV2) UpdateQuantity() {
	total := math.LegacyZeroDec()
	for _, o := range pl.Orders {
		total = total.Add(o.RemainingQty())
	}
	pl.Quantity = total
}

// IsEmpty returns true if no orders at this level
func (pl *PriceLevelV2) IsEmpty() bool {
	return len(pl.Orders) == 0
}

// FirstOrder returns the first order (oldest) at this level
func (pl *PriceLevelV2) FirstOrder() *types.Order {
	if len(pl.Orders) == 0 {
		return nil
	}
	return pl.Orders[0]
}

// ToPriceLevel converts to standard PriceLevel for compatibility
func (pl *PriceLevelV2) ToPriceLevel() *types.PriceLevel {
	orderIDs := make([]string, len(pl.Orders))
	for i, o := range pl.Orders {
		orderIDs[i] = o.OrderID
	}
	return &types.PriceLevel{
		Price:    pl.Price,
		Quantity: pl.Quantity,
		OrderIDs: orderIDs,
	}
}

// priceKeyAsc is a comparator for ascending price order (asks)
type priceKeyAsc struct{}

func (k priceKeyAsc) Compare(lhs, rhs interface{}) int {
	l := lhs.(math.LegacyDec)
	r := rhs.(math.LegacyDec)
	if l.LT(r) {
		return -1
	}
	if l.GT(r) {
		return 1
	}
	return 0
}

func (k priceKeyAsc) CalcScore(key interface{}) float64 {
	dec := key.(math.LegacyDec)
	f, _ := dec.Float64()
	return f
}

// priceKeyDesc is a comparator for descending price order (bids)
type priceKeyDesc struct{}

func (k priceKeyDesc) Compare(lhs, rhs interface{}) int {
	l := lhs.(math.LegacyDec)
	r := rhs.(math.LegacyDec)
	// Reverse order for descending
	if l.GT(r) {
		return -1
	}
	if l.LT(r) {
		return 1
	}
	return 0
}

func (k priceKeyDesc) CalcScore(key interface{}) float64 {
	dec := key.(math.LegacyDec)
	f, _ := dec.Float64()
	return -f // Negative for descending
}

// OrderBookV2 is an optimized order book using skip lists
// Provides O(log n) insertion and deletion
type OrderBookV2 struct {
	MarketID string
	Bids     *skiplist.SkipList // Descending by price (highest first)
	Asks     *skiplist.SkipList // Ascending by price (lowest first)
	mu       sync.RWMutex
}

// NewOrderBookV2 creates a new optimized order book
func NewOrderBookV2(marketID string) *OrderBookV2 {
	return &OrderBookV2{
		MarketID: marketID,
		Bids:     skiplist.New(priceKeyDesc{}),
		Asks:     skiplist.New(priceKeyAsc{}),
	}
}

// AddOrder adds an order to the order book - O(log n)
func (ob *OrderBookV2) AddOrder(order *types.Order) {
	ob.mu.Lock()
	defer ob.mu.Unlock()

	var list *skiplist.SkipList
	if order.Side == types.SideBuy {
		list = ob.Bids
	} else {
		list = ob.Asks
	}

	// Find or create price level
	elem := list.Get(order.Price)
	var level *PriceLevelV2
	if elem != nil {
		level = elem.Value.(*PriceLevelV2)
	} else {
		level = NewPriceLevelV2(order.Price)
		list.Set(order.Price, level)
	}

	level.AddOrder(order)
}

// RemoveOrder removes an order from the order book - O(log n)
func (ob *OrderBookV2) RemoveOrder(order *types.Order) *types.Order {
	ob.mu.Lock()
	defer ob.mu.Unlock()

	var list *skiplist.SkipList
	if order.Side == types.SideBuy {
		list = ob.Bids
	} else {
		list = ob.Asks
	}

	elem := list.Get(order.Price)
	if elem == nil {
		return nil
	}

	level := elem.Value.(*PriceLevelV2)
	removed := level.RemoveOrder(order.OrderID)

	// Remove empty price level
	if level.IsEmpty() {
		list.Remove(order.Price)
	}

	return removed
}

// RemoveOrderByID removes an order by ID and side - O(n) worst case
// Use RemoveOrder when you have the order object for O(log n)
func (ob *OrderBookV2) RemoveOrderByID(orderID string, side types.Side, price math.LegacyDec) *types.Order {
	ob.mu.Lock()
	defer ob.mu.Unlock()

	var list *skiplist.SkipList
	if side == types.SideBuy {
		list = ob.Bids
	} else {
		list = ob.Asks
	}

	elem := list.Get(price)
	if elem == nil {
		return nil
	}

	level := elem.Value.(*PriceLevelV2)
	removed := level.RemoveOrder(orderID)

	if level.IsEmpty() {
		list.Remove(price)
	}

	return removed
}

// GetBestBid returns the best (highest) bid level - O(1)
func (ob *OrderBookV2) GetBestBid() *PriceLevelV2 {
	ob.mu.RLock()
	defer ob.mu.RUnlock()

	front := ob.Bids.Front()
	if front == nil {
		return nil
	}
	return front.Value.(*PriceLevelV2)
}

// GetBestAsk returns the best (lowest) ask level - O(1)
func (ob *OrderBookV2) GetBestAsk() *PriceLevelV2 {
	ob.mu.RLock()
	defer ob.mu.RUnlock()

	front := ob.Asks.Front()
	if front == nil {
		return nil
	}
	return front.Value.(*PriceLevelV2)
}

// GetBestLevels returns best bid and ask levels
func (ob *OrderBookV2) GetBestLevels() (bestBid, bestAsk *PriceLevelV2) {
	ob.mu.RLock()
	defer ob.mu.RUnlock()

	if front := ob.Bids.Front(); front != nil {
		bestBid = front.Value.(*PriceLevelV2)
	}
	if front := ob.Asks.Front(); front != nil {
		bestAsk = front.Value.(*PriceLevelV2)
	}
	return
}

// GetSpread returns the spread between best bid and ask
func (ob *OrderBookV2) GetSpread() math.LegacyDec {
	bestBid, bestAsk := ob.GetBestLevels()
	if bestBid == nil || bestAsk == nil {
		return math.LegacyZeroDec()
	}
	return bestAsk.Price.Sub(bestBid.Price)
}

// GetMidPrice returns the mid price
func (ob *OrderBookV2) GetMidPrice() math.LegacyDec {
	bestBid, bestAsk := ob.GetBestLevels()
	if bestBid == nil || bestAsk == nil {
		return math.LegacyZeroDec()
	}
	return bestBid.Price.Add(bestAsk.Price).QuoInt64(2)
}

// GetDepth returns the order book depth (number of price levels)
func (ob *OrderBookV2) GetDepth() (bidLevels, askLevels int) {
	ob.mu.RLock()
	defer ob.mu.RUnlock()
	return ob.Bids.Len(), ob.Asks.Len()
}

// GetBidLevels returns n best bid levels
func (ob *OrderBookV2) GetBidLevels(n int) []*PriceLevelV2 {
	ob.mu.RLock()
	defer ob.mu.RUnlock()

	levels := make([]*PriceLevelV2, 0, n)
	elem := ob.Bids.Front()
	for i := 0; i < n && elem != nil; i++ {
		levels = append(levels, elem.Value.(*PriceLevelV2))
		elem = elem.Next()
	}
	return levels
}

// GetAskLevels returns n best ask levels
func (ob *OrderBookV2) GetAskLevels(n int) []*PriceLevelV2 {
	ob.mu.RLock()
	defer ob.mu.RUnlock()

	levels := make([]*PriceLevelV2, 0, n)
	elem := ob.Asks.Front()
	for i := 0; i < n && elem != nil; i++ {
		levels = append(levels, elem.Value.(*PriceLevelV2))
		elem = elem.Next()
	}
	return levels
}

// IterateBids iterates over all bid levels in price order (highest first)
func (ob *OrderBookV2) IterateBids(fn func(level *PriceLevelV2) bool) {
	ob.mu.RLock()
	defer ob.mu.RUnlock()

	elem := ob.Bids.Front()
	for elem != nil {
		if !fn(elem.Value.(*PriceLevelV2)) {
			break
		}
		elem = elem.Next()
	}
}

// IterateAsks iterates over all ask levels in price order (lowest first)
func (ob *OrderBookV2) IterateAsks(fn func(level *PriceLevelV2) bool) {
	ob.mu.RLock()
	defer ob.mu.RUnlock()

	elem := ob.Asks.Front()
	for elem != nil {
		if !fn(elem.Value.(*PriceLevelV2)) {
			break
		}
		elem = elem.Next()
	}
}

// ToOrderBook converts to the standard types.OrderBook for compatibility
func (ob *OrderBookV2) ToOrderBook() *types.OrderBook {
	ob.mu.RLock()
	defer ob.mu.RUnlock()

	result := types.NewOrderBook(ob.MarketID)

	// Convert bids
	elem := ob.Bids.Front()
	for elem != nil {
		level := elem.Value.(*PriceLevelV2)
		result.Bids = append(result.Bids, level.ToPriceLevel())
		elem = elem.Next()
	}

	// Convert asks
	elem = ob.Asks.Front()
	for elem != nil {
		level := elem.Value.(*PriceLevelV2)
		result.Asks = append(result.Asks, level.ToPriceLevel())
		elem = elem.Next()
	}

	return result
}

// FromOrderBook creates an OrderBookV2 from a standard OrderBook
func FromOrderBook(ob *types.OrderBook, orders map[string]*types.Order) *OrderBookV2 {
	result := NewOrderBookV2(ob.MarketID)

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
func (ob *OrderBookV2) Clear() {
	ob.mu.Lock()
	defer ob.mu.Unlock()

	ob.Bids = skiplist.New(priceKeyDesc{})
	ob.Asks = skiplist.New(priceKeyAsc{})
}

// Lock acquires write lock for exclusive access during matching operations
// CRITICAL: Must call Unlock() after use to prevent deadlocks
func (ob *OrderBookV2) Lock() {
	ob.mu.Lock()
}

// Unlock releases write lock
func (ob *OrderBookV2) Unlock() {
	ob.mu.Unlock()
}

// IterateBidsUnsafe iterates without acquiring lock (caller must hold lock)
// Use this when you need to modify the order book during iteration
func (ob *OrderBookV2) IterateBidsUnsafe(fn func(level *PriceLevelV2) bool) {
	elem := ob.Bids.Front()
	for elem != nil {
		if !fn(elem.Value.(*PriceLevelV2)) {
			break
		}
		elem = elem.Next()
	}
}

// IterateAsksUnsafe iterates without acquiring lock (caller must hold lock)
// Use this when you need to modify the order book during iteration
func (ob *OrderBookV2) IterateAsksUnsafe(fn func(level *PriceLevelV2) bool) {
	elem := ob.Asks.Front()
	for elem != nil {
		if !fn(elem.Value.(*PriceLevelV2)) {
			break
		}
		elem = elem.Next()
	}
}

// RemoveUnsafe removes a price level without acquiring lock (caller must hold lock)
func (ob *OrderBookV2) RemoveUnsafe(price math.LegacyDec, side types.Side) {
	if side == types.SideBuy {
		ob.Bids.Remove(price)
	} else {
		ob.Asks.Remove(price)
	}
}

// GetPriceLevel returns the price level at a specific price
func (ob *OrderBookV2) GetPriceLevel(price math.LegacyDec, side types.Side) *PriceLevelV2 {
	ob.mu.RLock()
	defer ob.mu.RUnlock()

	var list *skiplist.SkipList
	if side == types.SideBuy {
		list = ob.Bids
	} else {
		list = ob.Asks
	}

	elem := list.Get(price)
	if elem == nil {
		return nil
	}
	return elem.Value.(*PriceLevelV2)
}

// UpdateOrderQuantity updates an order's quantity after partial fill
func (ob *OrderBookV2) UpdateOrderQuantity(order *types.Order) {
	ob.mu.Lock()
	defer ob.mu.Unlock()

	var list *skiplist.SkipList
	if order.Side == types.SideBuy {
		list = ob.Bids
	} else {
		list = ob.Asks
	}

	elem := list.Get(order.Price)
	if elem == nil {
		return
	}

	level := elem.Value.(*PriceLevelV2)
	level.UpdateQuantity()
}
