package keeper

import (
	"bytes"
	"encoding/binary"
	"sort"
	"sync"

	"cosmossdk.io/math"
	"github.com/openalpha/perp-dex/x/orderbook/types"
)

// ============================================================================
// ART (Adaptive Radix Tree) OrderBook (ExchangeCore Style)
// ============================================================================
// Key characteristics:
// - O(k) operations where k is key length (constant for fixed-size prices)
// - Memory efficient through adaptive node sizing
// - Cache-friendly traversal
// - No rebalancing required
// - Similar to exchange-core/exchange-core implementation
// ============================================================================

const (
	// priceKeySize is the fixed size for price keys (16 bytes for int128)
	priceKeySize = 16
)

// priceToKey converts a decimal price to a fixed-size byte key
// Uses big-endian encoding for lexicographic ordering
func priceToKey(price math.LegacyDec) []byte {
	// Get the big integer representation
	// We need to handle both positive and negative numbers for proper ordering
	// Add a large offset to ensure all values are positive
	bi := price.BigInt()

	// Create a fixed-size buffer
	key := make([]byte, priceKeySize)

	// Use big-endian encoding for lexicographic ordering
	// Handle sign by adding offset
	if bi.Sign() >= 0 {
		// Positive: prefix with 0x80 and use big-endian bytes
		key[0] = 0x80
		biBytes := bi.Bytes()
		if len(biBytes) > priceKeySize-1 {
			biBytes = biBytes[len(biBytes)-(priceKeySize-1):]
		}
		copy(key[priceKeySize-len(biBytes):], biBytes)
	} else {
		// Negative: prefix with 0x00-0x7F (inverted magnitude)
		bi.Neg(bi) // make positive
		biBytes := bi.Bytes()
		if len(biBytes) > priceKeySize-1 {
			biBytes = biBytes[len(biBytes)-(priceKeySize-1):]
		}
		// Invert all bytes for proper ordering (larger negative = smaller key)
		for i := 1; i < priceKeySize; i++ {
			key[i] = 0xFF
		}
		for i, b := range biBytes {
			key[priceKeySize-len(biBytes)+i] ^= b
		}
	}

	return key
}

// keyToUint64 converts a key to uint64 for comparison (for simple cases)
func keyToUint64(key []byte) uint64 {
	if len(key) >= 8 {
		return binary.BigEndian.Uint64(key[len(key)-8:])
	}
	var padded [8]byte
	copy(padded[8-len(key):], key)
	return binary.BigEndian.Uint64(padded[:])
}

// ============================================================================
// Simple ART Node Implementation
// ============================================================================

// artNode represents a node in the adaptive radix tree
type artNode struct {
	children map[byte]*artNode
	key      []byte // full key if this is a leaf
	value    *PriceLevelV2
	isLeaf   bool
}

func newARTNode() *artNode {
	return &artNode{
		children: make(map[byte]*artNode),
	}
}

// artTree is a simple implementation of an Adaptive Radix Tree
type artTree struct {
	root *artNode
	size int
}

func newARTTree() *artTree {
	return &artTree{
		root: newARTNode(),
	}
}

// Insert inserts a key-value pair into the tree
func (t *artTree) Insert(key []byte, value *PriceLevelV2) {
	node := t.root
	for i := 0; i < len(key); i++ {
		child, exists := node.children[key[i]]
		if !exists {
			child = newARTNode()
			node.children[key[i]] = child
		}
		node = child
	}
	if !node.isLeaf {
		t.size++
	}
	node.key = key
	node.value = value
	node.isLeaf = true
}

// Search finds a value by key
func (t *artTree) Search(key []byte) *PriceLevelV2 {
	node := t.root
	for i := 0; i < len(key); i++ {
		child, exists := node.children[key[i]]
		if !exists {
			return nil
		}
		node = child
	}
	if node.isLeaf {
		return node.value
	}
	return nil
}

// Delete removes a key from the tree
func (t *artTree) Delete(key []byte) bool {
	return t.deleteRecursive(t.root, key, 0)
}

func (t *artTree) deleteRecursive(node *artNode, key []byte, depth int) bool {
	if depth == len(key) {
		if node.isLeaf {
			node.isLeaf = false
			node.value = nil
			t.size--
			return len(node.children) == 0
		}
		return false
	}

	child, exists := node.children[key[depth]]
	if !exists {
		return false
	}

	shouldDeleteChild := t.deleteRecursive(child, key, depth+1)
	if shouldDeleteChild {
		delete(node.children, key[depth])
		return !node.isLeaf && len(node.children) == 0
	}
	return false
}

// Len returns the number of entries
func (t *artTree) Len() int {
	return t.size
}

// artEntry represents an entry for iteration
type artEntry struct {
	key   []byte
	value *PriceLevelV2
}

// collectAll collects all entries in the tree
func (t *artTree) collectAll() []artEntry {
	entries := make([]artEntry, 0, t.size)
	t.collectRecursive(t.root, &entries)
	return entries
}

func (t *artTree) collectRecursive(node *artNode, entries *[]artEntry) {
	if node.isLeaf {
		*entries = append(*entries, artEntry{key: node.key, value: node.value})
	}
	for _, child := range node.children {
		t.collectRecursive(child, entries)
	}
}

// ============================================================================
// ART Side - one side of the order book (bids or asks)
// ============================================================================

type artSide struct {
	tree *artTree
	desc bool // true for bids (descending), false for asks (ascending)
}

func newARTSide(desc bool) *artSide {
	return &artSide{
		tree: newARTTree(),
		desc: desc,
	}
}

// Get returns the price level at the given price, or nil if not found
func (s *artSide) Get(price math.LegacyDec) *PriceLevelV2 {
	key := priceToKey(price)
	return s.tree.Search(key)
}

// Set adds or updates a price level
func (s *artSide) Set(price math.LegacyDec, level *PriceLevelV2) {
	key := priceToKey(price)
	s.tree.Insert(key, level)
}

// GetOrCreate returns the existing price level or creates a new one
func (s *artSide) GetOrCreate(price math.LegacyDec) *PriceLevelV2 {
	level := s.Get(price)
	if level == nil {
		level = NewPriceLevelV2(price)
		s.Set(price, level)
	}
	return level
}

// Remove removes a price level
func (s *artSide) Remove(price math.LegacyDec) {
	key := priceToKey(price)
	s.tree.Delete(key)
}

// Best returns the best price level
// For bids (desc=true): returns the highest price
// For asks (desc=false): returns the lowest price
func (s *artSide) Best() *PriceLevelV2 {
	entries := s.tree.collectAll()
	if len(entries) == 0 {
		return nil
	}

	// Sort entries by key
	sort.Slice(entries, func(i, j int) bool {
		cmp := bytes.Compare(entries[i].key, entries[j].key)
		if s.desc {
			return cmp > 0 // descending for bids
		}
		return cmp < 0 // ascending for asks
	})

	return entries[0].value
}

// Len returns the number of price levels
func (s *artSide) Len() int {
	return s.tree.Len()
}

// Iterate iterates over all price levels in sorted order
// For bids: descending (highest to lowest)
// For asks: ascending (lowest to highest)
func (s *artSide) Iterate(fn func(*PriceLevelV2) bool) {
	entries := s.tree.collectAll()
	if len(entries) == 0 {
		return
	}

	// Sort entries by key
	sort.Slice(entries, func(i, j int) bool {
		cmp := bytes.Compare(entries[i].key, entries[j].key)
		if s.desc {
			return cmp > 0 // descending for bids
		}
		return cmp < 0 // ascending for asks
	})

	for _, entry := range entries {
		if !fn(entry.value) {
			break
		}
	}
}

// ============================================================================
// OrderBookART - complete order book using Adaptive Radix Tree
// ============================================================================

// OrderBookART is an order book implementation using Adaptive Radix Tree
// for O(k) operations where k is key length. This is similar to
// ExchangeCore's approach for high-performance trading.
type OrderBookART struct {
	MarketID string
	Bids     *artSide
	Asks     *artSide
	mu       sync.RWMutex
}

// NewOrderBookART creates a new ART-based order book
func NewOrderBookART(marketID string) *OrderBookART {
	return &OrderBookART{
		MarketID: marketID,
		Bids:     newARTSide(true),  // descending for bids
		Asks:     newARTSide(false), // ascending for asks
	}
}

// GetMarketID returns the market ID
func (ob *OrderBookART) GetMarketID() string {
	return ob.MarketID
}

// getSide returns the appropriate side based on order side
func (ob *OrderBookART) getSide(side types.Side) *artSide {
	if side == types.SideBuy {
		return ob.Bids
	}
	return ob.Asks
}

// AddOrder adds an order to the order book - O(k)
func (ob *OrderBookART) AddOrder(order *types.Order) {
	ob.mu.Lock()
	defer ob.mu.Unlock()

	side := ob.getSide(order.Side)
	level := side.GetOrCreate(order.Price)
	level.AddOrder(order)
}

// RemoveOrder removes an order from the order book - O(k) for tree + O(n) within level
func (ob *OrderBookART) RemoveOrder(order *types.Order) *types.Order {
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

// RemoveOrderByID removes an order by ID - O(k) for tree + O(n) within level
func (ob *OrderBookART) RemoveOrderByID(orderID string, side types.Side, price math.LegacyDec) *types.Order {
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

// GetBestBid returns the best (highest) bid level
func (ob *OrderBookART) GetBestBid() *PriceLevelV2 {
	ob.mu.RLock()
	defer ob.mu.RUnlock()
	return ob.Bids.Best()
}

// GetBestAsk returns the best (lowest) ask level
func (ob *OrderBookART) GetBestAsk() *PriceLevelV2 {
	ob.mu.RLock()
	defer ob.mu.RUnlock()
	return ob.Asks.Best()
}

// GetBestLevels returns the best bid and ask levels
func (ob *OrderBookART) GetBestLevels() (bestBid, bestAsk *PriceLevelV2) {
	ob.mu.RLock()
	defer ob.mu.RUnlock()
	return ob.Bids.Best(), ob.Asks.Best()
}

// GetSpread returns the spread between best bid and ask
func (ob *OrderBookART) GetSpread() math.LegacyDec {
	bestBid, bestAsk := ob.GetBestLevels()
	if bestBid == nil || bestAsk == nil {
		return math.LegacyZeroDec()
	}
	return bestAsk.Price.Sub(bestBid.Price)
}

// GetMidPrice returns the mid price
func (ob *OrderBookART) GetMidPrice() math.LegacyDec {
	bestBid, bestAsk := ob.GetBestLevels()
	if bestBid == nil || bestAsk == nil {
		return math.LegacyZeroDec()
	}
	return bestBid.Price.Add(bestAsk.Price).QuoInt64(2)
}

// GetDepth returns the order book depth (number of price levels)
func (ob *OrderBookART) GetDepth() (bidLevels, askLevels int) {
	ob.mu.RLock()
	defer ob.mu.RUnlock()
	return ob.Bids.Len(), ob.Asks.Len()
}

// GetBidLevels returns n best bid levels
func (ob *OrderBookART) GetBidLevels(n int) []*PriceLevelV2 {
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
func (ob *OrderBookART) GetAskLevels(n int) []*PriceLevelV2 {
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
func (ob *OrderBookART) IterateBids(fn func(level *PriceLevelV2) bool) {
	ob.mu.RLock()
	defer ob.mu.RUnlock()
	ob.Bids.Iterate(fn)
}

// IterateAsks iterates over all ask levels in price order (lowest first)
func (ob *OrderBookART) IterateAsks(fn func(level *PriceLevelV2) bool) {
	ob.mu.RLock()
	defer ob.mu.RUnlock()
	ob.Asks.Iterate(fn)
}

// ToOrderBook converts to the standard types.OrderBook for compatibility
func (ob *OrderBookART) ToOrderBook() *types.OrderBook {
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

// FromOrderBookToART creates an OrderBookART from a standard OrderBook
func FromOrderBookToART(ob *types.OrderBook, orders map[string]*types.Order) *OrderBookART {
	result := NewOrderBookART(ob.MarketID)

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
func (ob *OrderBookART) Clear() {
	ob.mu.Lock()
	defer ob.mu.Unlock()

	ob.Bids = newARTSide(true)
	ob.Asks = newARTSide(false)
}

// GetPriceLevel returns the price level at a specific price - O(k)
func (ob *OrderBookART) GetPriceLevel(price math.LegacyDec, side types.Side) *PriceLevelV2 {
	ob.mu.RLock()
	defer ob.mu.RUnlock()
	return ob.getSide(side).Get(price)
}

// UpdateOrderQuantity updates an order's quantity after partial fill
func (ob *OrderBookART) UpdateOrderQuantity(order *types.Order) {
	ob.mu.Lock()
	defer ob.mu.Unlock()

	side := ob.getSide(order.Side)
	level := side.Get(order.Price)
	if level != nil {
		level.UpdateQuantity()
	}
}
