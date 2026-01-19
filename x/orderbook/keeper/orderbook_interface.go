package keeper

import (
	"cosmossdk.io/math"
	"github.com/openalpha/perp-dex/x/orderbook/types"
)

// OrderBookEngine defines the unified interface for all order book implementations
// This allows swapping between different data structures (SkipList, HashMap, BTree, ART)
type OrderBookEngine interface {
	// Basic operations
	AddOrder(order *types.Order)
	RemoveOrder(order *types.Order) *types.Order
	RemoveOrderByID(orderID string, side types.Side, price math.LegacyDec) *types.Order

	// Query operations
	GetBestBid() *PriceLevelV2
	GetBestAsk() *PriceLevelV2
	GetBestLevels() (bestBid, bestAsk *PriceLevelV2)
	GetSpread() math.LegacyDec
	GetMidPrice() math.LegacyDec

	// Depth queries
	GetBidLevels(n int) []*PriceLevelV2
	GetAskLevels(n int) []*PriceLevelV2
	GetDepth() (bidLevels, askLevels int)
	GetPriceLevel(price math.LegacyDec, side types.Side) *PriceLevelV2

	// Iteration
	IterateBids(fn func(level *PriceLevelV2) bool)
	IterateAsks(fn func(level *PriceLevelV2) bool)

	// Update
	UpdateOrderQuantity(order *types.Order)

	// Conversion
	ToOrderBook() *types.OrderBook

	// Management
	Clear()
	GetMarketID() string
}

// Verify that all implementations satisfy the interface
var _ OrderBookEngine = (*OrderBookV2)(nil)
var _ OrderBookEngine = (*OrderBookHashMap)(nil)
var _ OrderBookEngine = (*OrderBookBTree)(nil)
var _ OrderBookEngine = (*OrderBookART)(nil)

// GetMarketID returns the market ID for OrderBookV2
func (ob *OrderBookV2) GetMarketID() string {
	return ob.MarketID
}
