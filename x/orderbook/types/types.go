package types

import (
	"fmt"
	"sort"
	"time"

	"cosmossdk.io/math"
	proto "github.com/cosmos/gogoproto/proto"
)

func init() {
	proto.RegisterEnum("perpdex.orderbook.v1.Side", Side_name, Side_value)
	proto.RegisterEnum("perpdex.orderbook.v1.OrderType", OrderType_name, OrderType_value)
	proto.RegisterEnum("perpdex.orderbook.v1.OrderStatus", OrderStatus_name, OrderStatus_value)
}

// Side represents order side (int32 for proto compatibility)
type Side int32

const (
	SideUnspecified Side = iota
	SideBuy
	SideSell
)

// Proto-compatible aliases for Side enum
const (
	Side_SIDE_UNSPECIFIED = SideUnspecified
	Side_SIDE_BUY         = SideBuy
	Side_SIDE_SELL        = SideSell
)

// Proto-compatible maps for Side enum
var Side_name = map[int32]string{
	0: "SIDE_UNSPECIFIED",
	1: "SIDE_BUY",
	2: "SIDE_SELL",
}

var Side_value = map[string]int32{
	"SIDE_UNSPECIFIED": 0,
	"SIDE_BUY":         1,
	"SIDE_SELL":        2,
}

func (s Side) String() string {
	switch s {
	case SideBuy:
		return "SIDE_BUY"
	case SideSell:
		return "SIDE_SELL"
	default:
		return "SIDE_UNSPECIFIED"
	}
}

func (s Side) Opposite() Side {
	if s == SideBuy {
		return SideSell
	}
	return SideBuy
}

// OrderType represents order type (int32 for proto compatibility)
type OrderType int32

const (
	OrderTypeUnspecified OrderType = iota
	OrderTypeLimit
	OrderTypeMarket
)

// Proto-compatible aliases for OrderType enum
const (
	OrderType_ORDER_TYPE_UNSPECIFIED = OrderTypeUnspecified
	OrderType_ORDER_TYPE_LIMIT       = OrderTypeLimit
	OrderType_ORDER_TYPE_MARKET      = OrderTypeMarket
)

// Proto-compatible maps for OrderType enum
var OrderType_name = map[int32]string{
	0: "ORDER_TYPE_UNSPECIFIED",
	1: "ORDER_TYPE_LIMIT",
	2: "ORDER_TYPE_MARKET",
}

var OrderType_value = map[string]int32{
	"ORDER_TYPE_UNSPECIFIED": 0,
	"ORDER_TYPE_LIMIT":       1,
	"ORDER_TYPE_MARKET":      2,
}

func (t OrderType) String() string {
	switch t {
	case OrderTypeLimit:
		return "ORDER_TYPE_LIMIT"
	case OrderTypeMarket:
		return "ORDER_TYPE_MARKET"
	default:
		return "ORDER_TYPE_UNSPECIFIED"
	}
}

// OrderStatus represents order status (int32 for proto compatibility)
type OrderStatus int32

const (
	OrderStatusUnspecified OrderStatus = iota
	OrderStatusOpen
	OrderStatusFilled
	OrderStatusPartiallyFilled
	OrderStatusCancelled
)

// Proto-compatible aliases for OrderStatus enum
const (
	OrderStatus_ORDER_STATUS_UNSPECIFIED      = OrderStatusUnspecified
	OrderStatus_ORDER_STATUS_OPEN             = OrderStatusOpen
	OrderStatus_ORDER_STATUS_FILLED           = OrderStatusFilled
	OrderStatus_ORDER_STATUS_PARTIALLY_FILLED = OrderStatusPartiallyFilled
	OrderStatus_ORDER_STATUS_CANCELLED        = OrderStatusCancelled
)

// Proto-compatible maps for OrderStatus enum
var OrderStatus_name = map[int32]string{
	0: "ORDER_STATUS_UNSPECIFIED",
	1: "ORDER_STATUS_OPEN",
	2: "ORDER_STATUS_FILLED",
	3: "ORDER_STATUS_PARTIALLY_FILLED",
	4: "ORDER_STATUS_CANCELLED",
}

var OrderStatus_value = map[string]int32{
	"ORDER_STATUS_UNSPECIFIED":      0,
	"ORDER_STATUS_OPEN":             1,
	"ORDER_STATUS_FILLED":           2,
	"ORDER_STATUS_PARTIALLY_FILLED": 3,
	"ORDER_STATUS_CANCELLED":        4,
}

func (s OrderStatus) String() string {
	switch s {
	case OrderStatusOpen:
		return "ORDER_STATUS_OPEN"
	case OrderStatusFilled:
		return "ORDER_STATUS_FILLED"
	case OrderStatusPartiallyFilled:
		return "ORDER_STATUS_PARTIALLY_FILLED"
	case OrderStatusCancelled:
		return "ORDER_STATUS_CANCELLED"
	default:
		return "ORDER_STATUS_UNSPECIFIED"
	}
}

// Order represents a trading order
type Order struct {
	OrderID   string
	Trader    string
	MarketID  string
	Side      Side
	OrderType OrderType
	Price     math.LegacyDec // limit price (ignored for market orders)
	Quantity  math.LegacyDec // order quantity
	FilledQty math.LegacyDec // filled quantity
	Status    OrderStatus
	CreatedAt time.Time
	UpdatedAt time.Time
}

// NewOrder creates a new order
func NewOrder(orderID, trader, marketID string, side Side, orderType OrderType, price, quantity math.LegacyDec) *Order {
	now := time.Now()
	return &Order{
		OrderID:   orderID,
		Trader:    trader,
		MarketID:  marketID,
		Side:      side,
		OrderType: orderType,
		Price:     price,
		Quantity:  quantity,
		FilledQty: math.LegacyZeroDec(),
		Status:    OrderStatusOpen,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// RemainingQty returns the remaining unfilled quantity
func (o *Order) RemainingQty() math.LegacyDec {
	return o.Quantity.Sub(o.FilledQty)
}

// IsFilled returns true if the order is completely filled
func (o *Order) IsFilled() bool {
	return o.FilledQty.GTE(o.Quantity)
}

// IsActive returns true if the order can still be matched
func (o *Order) IsActive() bool {
	return o.Status == OrderStatusOpen || o.Status == OrderStatusPartiallyFilled
}

// Fill fills the order with the given quantity
func (o *Order) Fill(qty math.LegacyDec) error {
	if qty.GT(o.RemainingQty()) {
		return fmt.Errorf("fill quantity %s exceeds remaining %s", qty, o.RemainingQty())
	}
	o.FilledQty = o.FilledQty.Add(qty)
	o.UpdatedAt = time.Now()
	if o.IsFilled() {
		o.Status = OrderStatusFilled
	} else if o.FilledQty.IsPositive() {
		o.Status = OrderStatusPartiallyFilled
	}
	return nil
}

// Cancel cancels the order
func (o *Order) Cancel() {
	o.Status = OrderStatusCancelled
	o.UpdatedAt = time.Now()
}

// PriceLevel represents a price level in the order book
type PriceLevel struct {
	Price    math.LegacyDec
	Quantity math.LegacyDec // total quantity at this price
	OrderIDs []string       // order IDs in FIFO order
}

// NewPriceLevel creates a new price level
func NewPriceLevel(price math.LegacyDec) *PriceLevel {
	return &PriceLevel{
		Price:    price,
		Quantity: math.LegacyZeroDec(),
		OrderIDs: make([]string, 0),
	}
}

// AddOrder adds an order to the price level
func (pl *PriceLevel) AddOrder(orderID string, qty math.LegacyDec) {
	pl.OrderIDs = append(pl.OrderIDs, orderID)
	pl.Quantity = pl.Quantity.Add(qty)
}

// RemoveOrder removes an order from the price level
func (pl *PriceLevel) RemoveOrder(orderID string, qty math.LegacyDec) {
	for i, id := range pl.OrderIDs {
		if id == orderID {
			pl.OrderIDs = append(pl.OrderIDs[:i], pl.OrderIDs[i+1:]...)
			pl.Quantity = pl.Quantity.Sub(qty)
			break
		}
	}
}

// IsEmpty returns true if the price level has no orders
func (pl *PriceLevel) IsEmpty() bool {
	return len(pl.OrderIDs) == 0
}

// OrderBook represents the order book for a market
type OrderBook struct {
	MarketID string
	Bids     []*PriceLevel // sorted descending by price (highest first)
	Asks     []*PriceLevel // sorted ascending by price (lowest first)
}

// NewOrderBook creates a new order book
func NewOrderBook(marketID string) *OrderBook {
	return &OrderBook{
		MarketID: marketID,
		Bids:     make([]*PriceLevel, 0),
		Asks:     make([]*PriceLevel, 0),
	}
}

// AddOrder adds an order to the order book
func (ob *OrderBook) AddOrder(order *Order) {
	var levels *[]*PriceLevel
	if order.Side == SideBuy {
		levels = &ob.Bids
	} else {
		levels = &ob.Asks
	}

	// Find or create price level
	var level *PriceLevel
	for _, pl := range *levels {
		if pl.Price.Equal(order.Price) {
			level = pl
			break
		}
	}

	if level == nil {
		level = NewPriceLevel(order.Price)
		*levels = append(*levels, level)
		ob.sortLevels()
	}

	level.AddOrder(order.OrderID, order.RemainingQty())
}

// RemoveOrder removes an order from the order book
func (ob *OrderBook) RemoveOrder(order *Order) {
	var levels *[]*PriceLevel
	if order.Side == SideBuy {
		levels = &ob.Bids
	} else {
		levels = &ob.Asks
	}

	for i, pl := range *levels {
		if pl.Price.Equal(order.Price) {
			pl.RemoveOrder(order.OrderID, order.RemainingQty())
			if pl.IsEmpty() {
				*levels = append((*levels)[:i], (*levels)[i+1:]...)
			}
			break
		}
	}
}

// sortLevels sorts bids descending and asks ascending
func (ob *OrderBook) sortLevels() {
	// Sort bids descending (highest price first)
	sort.Slice(ob.Bids, func(i, j int) bool {
		return ob.Bids[i].Price.GT(ob.Bids[j].Price)
	})
	// Sort asks ascending (lowest price first)
	sort.Slice(ob.Asks, func(i, j int) bool {
		return ob.Asks[i].Price.LT(ob.Asks[j].Price)
	})
}

// BestBid returns the best (highest) bid price, or nil if empty
func (ob *OrderBook) BestBid() *PriceLevel {
	if len(ob.Bids) == 0 {
		return nil
	}
	return ob.Bids[0]
}

// BestAsk returns the best (lowest) ask price, or nil if empty
func (ob *OrderBook) BestAsk() *PriceLevel {
	if len(ob.Asks) == 0 {
		return nil
	}
	return ob.Asks[0]
}

// Spread returns the spread between best bid and best ask
func (ob *OrderBook) Spread() math.LegacyDec {
	bid := ob.BestBid()
	ask := ob.BestAsk()
	if bid == nil || ask == nil {
		return math.LegacyZeroDec()
	}
	return ask.Price.Sub(bid.Price)
}

// Trade represents an executed trade
type Trade struct {
	TradeID      string
	MarketID     string
	TakerOrderID string
	MakerOrderID string
	Taker        string
	Maker        string
	TakerSide    Side
	Price        math.LegacyDec
	Quantity     math.LegacyDec
	TakerFee     math.LegacyDec
	MakerFee     math.LegacyDec
	Timestamp    time.Time
}

// TradeWithSettlement contains trade data plus settlement fields.
type TradeWithSettlement struct {
	TradeID      string
	MarketID     string
	TakerOrderID string
	MakerOrderID string
	Taker        string
	Maker        string
	TakerSide    Side
	Price        math.LegacyDec
	Quantity     math.LegacyDec
	TakerFee     math.LegacyDec
	MakerFee     math.LegacyDec
	Timestamp    time.Time

	// Settlement fields (per-side)
	TakerRealizedPnL  math.LegacyDec
	MakerRealizedPnL  math.LegacyDec
	TakerMarginChange math.LegacyDec
	MakerMarginChange math.LegacyDec
}

// SettlementRequest bundles trades for settlement processing.
type SettlementRequest struct {
	Trades []*TradeWithSettlement
}

// NewSettlementRequest creates a settlement request from trades.
func NewSettlementRequest(trades []*TradeWithSettlement) *SettlementRequest {
	return &SettlementRequest{Trades: trades}
}

// NewTrade creates a new trade
func NewTrade(tradeID, marketID string, takerOrder, makerOrder *Order, price, qty, takerFee, makerFee math.LegacyDec) *Trade {
	return &Trade{
		TradeID:      tradeID,
		MarketID:     marketID,
		TakerOrderID: takerOrder.OrderID,
		MakerOrderID: makerOrder.OrderID,
		Taker:        takerOrder.Trader,
		Maker:        makerOrder.Trader,
		TakerSide:    takerOrder.Side,
		Price:        price,
		Quantity:     qty,
		TakerFee:     takerFee,
		MakerFee:     makerFee,
		Timestamp:    time.Now(),
	}
}

// NewTradeWithSettlement creates a settlement-ready trade from a trade.
func NewTradeWithSettlement(trade *Trade) *TradeWithSettlement {
	if trade == nil {
		return nil
	}
	return &TradeWithSettlement{
		TradeID:           trade.TradeID,
		MarketID:          trade.MarketID,
		TakerOrderID:      trade.TakerOrderID,
		MakerOrderID:      trade.MakerOrderID,
		Taker:             trade.Taker,
		Maker:             trade.Maker,
		TakerSide:         trade.TakerSide,
		Price:             trade.Price,
		Quantity:          trade.Quantity,
		TakerFee:          trade.TakerFee,
		MakerFee:          trade.MakerFee,
		Timestamp:         trade.Timestamp,
		TakerRealizedPnL:  math.LegacyZeroDec(),
		MakerRealizedPnL:  math.LegacyZeroDec(),
		TakerMarginChange: math.LegacyZeroDec(),
		MakerMarginChange: math.LegacyZeroDec(),
	}
}
