package types

import (
	"fmt"
	"sort"
	"time"

	"cosmossdk.io/math"
)

// Side represents order side
type Side int

const (
	SideUnspecified Side = iota
	SideBuy
	SideSell
)

func (s Side) String() string {
	switch s {
	case SideBuy:
		return "buy"
	case SideSell:
		return "sell"
	default:
		return "unspecified"
	}
}

func (s Side) Opposite() Side {
	if s == SideBuy {
		return SideSell
	}
	return SideBuy
}

// OrderType represents order type
type OrderType int

const (
	OrderTypeUnspecified OrderType = iota
	OrderTypeLimit
	OrderTypeMarket
)

func (t OrderType) String() string {
	switch t {
	case OrderTypeLimit:
		return "limit"
	case OrderTypeMarket:
		return "market"
	default:
		return "unspecified"
	}
}

// OrderStatus represents order status
type OrderStatus int

const (
	OrderStatusUnspecified OrderStatus = iota
	OrderStatusOpen
	OrderStatusFilled
	OrderStatusPartiallyFilled
	OrderStatusCancelled
)

func (s OrderStatus) String() string {
	switch s {
	case OrderStatusOpen:
		return "open"
	case OrderStatusFilled:
		return "filled"
	case OrderStatusPartiallyFilled:
		return "partially_filled"
	case OrderStatusCancelled:
		return "cancelled"
	default:
		return "unspecified"
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
