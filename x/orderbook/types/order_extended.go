package types

import (
	"time"

	"cosmossdk.io/math"
)

// Extended OrderType constants
const (
	OrderTypeStopLoss        OrderType = 3 // Stop loss order
	OrderTypeTakeProfit      OrderType = 4 // Take profit order
	OrderTypeStopLimit       OrderType = 5 // Stop limit order
	OrderTypeTakeProfitLimit OrderType = 6 // Take profit limit order
	OrderTypeTrailingStop    OrderType = 7 // Trailing stop order
)

// StringToOrderType returns the extended string representation
func (t OrderType) ExtendedString() string {
	switch t {
	case OrderTypeLimit:
		return "limit"
	case OrderTypeMarket:
		return "market"
	case OrderTypeStopLoss:
		return "stop_loss"
	case OrderTypeTakeProfit:
		return "take_profit"
	case OrderTypeStopLimit:
		return "stop_limit"
	case OrderTypeTakeProfitLimit:
		return "take_profit_limit"
	case OrderTypeTrailingStop:
		return "trailing_stop"
	default:
		return "unspecified"
	}
}

// IsConditional returns true if the order type is conditional
func (t OrderType) IsConditional() bool {
	return t == OrderTypeStopLoss || t == OrderTypeTakeProfit ||
		t == OrderTypeStopLimit || t == OrderTypeTakeProfitLimit ||
		t == OrderTypeTrailingStop
}

// TimeInForce represents order time in force
type TimeInForce int

const (
	TimeInForceGTC TimeInForce = iota // Good Till Cancel (default)
	TimeInForceIOC                    // Immediate Or Cancel
	TimeInForceFOK                    // Fill Or Kill
	TimeInForceGTX                    // Post Only (Good Till Crossing)
)

// String returns the string representation of TimeInForce
func (t TimeInForce) String() string {
	switch t {
	case TimeInForceGTC:
		return "GTC"
	case TimeInForceIOC:
		return "IOC"
	case TimeInForceFOK:
		return "FOK"
	case TimeInForceGTX:
		return "GTX"
	default:
		return "GTC"
	}
}

// OrderFlags contains additional order flags
type OrderFlags struct {
	ReduceOnly bool // Only reduce existing position, never increase
	PostOnly   bool // Only add liquidity, never take
	Hidden     bool // Hidden order (not shown in order book)
}

// ExtendedOrder extends the base Order with additional fields
type ExtendedOrder struct {
	// Base fields from Order
	OrderID   string
	Trader    string
	MarketID  string
	Side      Side
	OrderType OrderType
	Price     math.LegacyDec
	Quantity  math.LegacyDec
	FilledQty math.LegacyDec
	Status    OrderStatus
	CreatedAt time.Time
	UpdatedAt time.Time

	// Extended fields
	TimeInForce   TimeInForce    // Order time in force
	TriggerPrice  math.LegacyDec // Trigger price for conditional orders
	Flags         OrderFlags     // Order flags
	ClientOrderID string         // Client-provided order ID
	TriggeredAt   *time.Time     // When the conditional order was triggered
}

// NewExtendedOrder creates a new extended order
func NewExtendedOrder(
	orderID, trader, marketID string,
	side Side,
	orderType OrderType,
	price, quantity math.LegacyDec,
	timeInForce TimeInForce,
	flags OrderFlags,
) *ExtendedOrder {
	now := time.Now()
	return &ExtendedOrder{
		OrderID:     orderID,
		Trader:      trader,
		MarketID:    marketID,
		Side:        side,
		OrderType:   orderType,
		Price:       price,
		Quantity:    quantity,
		FilledQty:   math.LegacyZeroDec(),
		Status:      OrderStatusOpen,
		CreatedAt:   now,
		UpdatedAt:   now,
		TimeInForce: timeInForce,
		Flags:       flags,
	}
}

// ToOrder converts ExtendedOrder to base Order
func (o *ExtendedOrder) ToOrder() *Order {
	return &Order{
		OrderID:   o.OrderID,
		Trader:    o.Trader,
		MarketID:  o.MarketID,
		Side:      o.Side,
		OrderType: o.OrderType,
		Price:     o.Price,
		Quantity:  o.Quantity,
		FilledQty: o.FilledQty,
		Status:    o.Status,
		CreatedAt: o.CreatedAt,
		UpdatedAt: o.UpdatedAt,
	}
}

// ConditionalOrder represents a stop loss or take profit order
type ConditionalOrder struct {
	OrderID        string         // Unique order identifier
	Trader         string         // Trader address
	MarketID       string         // Market identifier
	Side           Side           // Order side (buy/sell)
	OrderType      OrderType      // Order type (stop_loss, take_profit, etc.)
	TriggerPrice   math.LegacyDec // Price at which to trigger
	ExecutionPrice math.LegacyDec // Limit price for execution (for limit types)
	Quantity       math.LegacyDec // Order quantity
	Flags          OrderFlags     // Order flags
	Status         OrderStatus    // Order status
	CreatedAt      time.Time      // Creation time
	TriggeredAt    *time.Time     // Trigger time (nil if not triggered)
	ClientOrderID  string         // Client-provided order ID
}

// NewConditionalOrder creates a new conditional order
func NewConditionalOrder(
	orderID, trader, marketID string,
	side Side,
	orderType OrderType,
	triggerPrice, executionPrice, quantity math.LegacyDec,
	flags OrderFlags,
) *ConditionalOrder {
	return &ConditionalOrder{
		OrderID:        orderID,
		Trader:         trader,
		MarketID:       marketID,
		Side:           side,
		OrderType:      orderType,
		TriggerPrice:   triggerPrice,
		ExecutionPrice: executionPrice,
		Quantity:       quantity,
		Flags:          flags,
		Status:         OrderStatusOpen,
		CreatedAt:      time.Now(),
	}
}

// IsActive returns true if the conditional order is still active
func (o *ConditionalOrder) IsActive() bool {
	return o.Status == OrderStatusOpen
}

// ShouldTrigger checks if the order should trigger at the given price
func (o *ConditionalOrder) ShouldTrigger(markPrice math.LegacyDec) bool {
	if !o.IsActive() {
		return false
	}

	switch o.OrderType {
	case OrderTypeStopLoss, OrderTypeStopLimit:
		// Stop loss for long: triggers when price falls to trigger price
		// Stop loss for short: triggers when price rises to trigger price
		if o.Side == SideSell {
			return markPrice.LTE(o.TriggerPrice)
		}
		return markPrice.GTE(o.TriggerPrice)

	case OrderTypeTakeProfit, OrderTypeTakeProfitLimit:
		// Take profit for long: triggers when price rises to trigger price
		// Take profit for short: triggers when price falls to trigger price
		if o.Side == SideSell {
			return markPrice.GTE(o.TriggerPrice)
		}
		return markPrice.LTE(o.TriggerPrice)
	}

	return false
}

// Trigger triggers the conditional order
func (o *ConditionalOrder) Trigger() *Order {
	now := time.Now()
	o.TriggeredAt = &now
	o.Status = OrderStatusFilled

	// Determine execution order type
	execOrderType := OrderTypeMarket
	execPrice := o.ExecutionPrice
	if o.OrderType == OrderTypeStopLimit || o.OrderType == OrderTypeTakeProfitLimit {
		execOrderType = OrderTypeLimit
	}

	return &Order{
		OrderID:   o.OrderID + "-exec",
		Trader:    o.Trader,
		MarketID:  o.MarketID,
		Side:      o.Side,
		OrderType: execOrderType,
		Price:     execPrice,
		Quantity:  o.Quantity,
		FilledQty: math.LegacyZeroDec(),
		Status:    OrderStatusOpen,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// Cancel cancels the conditional order
func (o *ConditionalOrder) Cancel() {
	o.Status = OrderStatusCancelled
}

// ============ Trailing Stop Order ============

// TrailingStopOrder represents a trailing stop order
// The stop price follows the market price at a specified distance
type TrailingStopOrder struct {
	OrderID          string         // Unique order identifier
	Trader           string         // Trader address
	MarketID         string         // Market identifier
	Side             Side           // Order side (sell for long stop, buy for short stop)
	Quantity         math.LegacyDec // Order quantity
	TrailAmount      math.LegacyDec // Fixed trailing distance in price units
	TrailPercent     math.LegacyDec // Percentage trailing distance (alternative to TrailAmount)
	ActivationPrice  math.LegacyDec // Price at which trailing starts (optional)
	CurrentStopPrice math.LegacyDec // Current calculated stop price
	HighWaterMark    math.LegacyDec // Highest price recorded (for sell/long stop)
	LowWaterMark     math.LegacyDec // Lowest price recorded (for buy/short stop)
	IsActivated      bool           // Whether the trailing has started
	Status           OrderStatus    // Order status
	CreatedAt        time.Time      // Creation time
	UpdatedAt        time.Time      // Last update time
}

// NewTrailingStopOrder creates a new trailing stop order with fixed amount
func NewTrailingStopOrder(
	orderID, trader, marketID string,
	side Side,
	quantity, trailAmount, activationPrice math.LegacyDec,
) *TrailingStopOrder {
	now := time.Now()
	return &TrailingStopOrder{
		OrderID:          orderID,
		Trader:           trader,
		MarketID:         marketID,
		Side:             side,
		Quantity:         quantity,
		TrailAmount:      trailAmount,
		TrailPercent:     math.LegacyZeroDec(),
		ActivationPrice:  activationPrice,
		CurrentStopPrice: math.LegacyZeroDec(),
		HighWaterMark:    math.LegacyZeroDec(),
		LowWaterMark:     math.LegacyZeroDec(),
		IsActivated:      activationPrice.IsZero(), // Activate immediately if no activation price
		Status:           OrderStatusOpen,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
}

// NewTrailingStopOrderPercent creates a new trailing stop order with percentage
func NewTrailingStopOrderPercent(
	orderID, trader, marketID string,
	side Side,
	quantity, trailPercent, activationPrice math.LegacyDec,
) *TrailingStopOrder {
	now := time.Now()
	return &TrailingStopOrder{
		OrderID:          orderID,
		Trader:           trader,
		MarketID:         marketID,
		Side:             side,
		Quantity:         quantity,
		TrailAmount:      math.LegacyZeroDec(),
		TrailPercent:     trailPercent,
		ActivationPrice:  activationPrice,
		CurrentStopPrice: math.LegacyZeroDec(),
		HighWaterMark:    math.LegacyZeroDec(),
		LowWaterMark:     math.LegacyZeroDec(),
		IsActivated:      activationPrice.IsZero(),
		Status:           OrderStatusOpen,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
}

// IsActive returns true if the trailing stop is still active
func (o *TrailingStopOrder) IsActive() bool {
	return o.Status == OrderStatusOpen
}

// GetTrailDistance calculates the trail distance based on current price
func (o *TrailingStopOrder) GetTrailDistance(currentPrice math.LegacyDec) math.LegacyDec {
	if o.TrailAmount.IsPositive() {
		return o.TrailAmount
	}
	// Calculate from percentage
	return currentPrice.Mul(o.TrailPercent).Quo(math.LegacyNewDec(100))
}

// Update updates the trailing stop based on current market price
// Returns true if the stop was triggered
func (o *TrailingStopOrder) Update(markPrice math.LegacyDec) bool {
	if !o.IsActive() {
		return false
	}

	// Check activation
	if !o.IsActivated {
		if o.Side == SideSell {
			// For sell (long stop): activate when price rises above activation price
			if markPrice.GTE(o.ActivationPrice) {
				o.IsActivated = true
				o.HighWaterMark = markPrice
				trailDist := o.GetTrailDistance(markPrice)
				o.CurrentStopPrice = markPrice.Sub(trailDist)
			}
		} else {
			// For buy (short stop): activate when price falls below activation price
			if markPrice.LTE(o.ActivationPrice) {
				o.IsActivated = true
				o.LowWaterMark = markPrice
				trailDist := o.GetTrailDistance(markPrice)
				o.CurrentStopPrice = markPrice.Add(trailDist)
			}
		}
		o.UpdatedAt = time.Now()
		return false
	}

	// Update high/low water mark and stop price
	if o.Side == SideSell {
		// For sell (long position stop)
		// Update high water mark if price goes higher
		if markPrice.GT(o.HighWaterMark) {
			o.HighWaterMark = markPrice
			trailDist := o.GetTrailDistance(markPrice)
			o.CurrentStopPrice = markPrice.Sub(trailDist)
			o.UpdatedAt = time.Now()
		}
		// Check if stop triggered
		if markPrice.LTE(o.CurrentStopPrice) {
			return true
		}
	} else {
		// For buy (short position stop)
		// Update low water mark if price goes lower
		if markPrice.LT(o.LowWaterMark) || o.LowWaterMark.IsZero() {
			o.LowWaterMark = markPrice
			trailDist := o.GetTrailDistance(markPrice)
			o.CurrentStopPrice = markPrice.Add(trailDist)
			o.UpdatedAt = time.Now()
		}
		// Check if stop triggered
		if markPrice.GTE(o.CurrentStopPrice) {
			return true
		}
	}

	return false
}

// Trigger triggers the trailing stop and returns an execution order
func (o *TrailingStopOrder) Trigger() *Order {
	now := time.Now()
	o.Status = OrderStatusFilled
	o.UpdatedAt = now

	return &Order{
		OrderID:   o.OrderID + "-exec",
		Trader:    o.Trader,
		MarketID:  o.MarketID,
		Side:      o.Side,
		OrderType: OrderTypeMarket,
		Price:     o.CurrentStopPrice,
		Quantity:  o.Quantity,
		FilledQty: math.LegacyZeroDec(),
		Status:    OrderStatusOpen,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// Cancel cancels the trailing stop order
func (o *TrailingStopOrder) Cancel() {
	o.Status = OrderStatusCancelled
	o.UpdatedAt = time.Now()
}

// ============ OCO (One-Cancels-Other) Order ============

// OCOStatus represents the status of an OCO order
type OCOStatus int

const (
	OCOStatusPending          OCOStatus = iota // Both orders are pending
	OCOStatusPartialTriggered                  // One order triggered, other cancelled
	OCOStatusTriggered                         // One order executed successfully
	OCOStatusCancelled                         // Both orders cancelled
)

// String returns the string representation of OCOStatus
func (s OCOStatus) String() string {
	switch s {
	case OCOStatusPending:
		return "pending"
	case OCOStatusPartialTriggered:
		return "partial_triggered"
	case OCOStatusTriggered:
		return "triggered"
	case OCOStatusCancelled:
		return "cancelled"
	default:
		return "unknown"
	}
}

// OCOOrder represents a One-Cancels-Other order pair
// Typically used for setting both stop-loss and take-profit simultaneously
type OCOOrder struct {
	OCOID       string            // Unique OCO identifier
	Trader      string            // Trader address
	MarketID    string            // Market identifier
	StopOrder   *ConditionalOrder // Stop loss order
	LimitOrder  *Order            // Take profit / limit order
	Status      OCOStatus         // OCO status
	TriggeredID string            // ID of the order that was triggered
	CreatedAt   time.Time         // Creation time
	UpdatedAt   time.Time         // Last update time
}

// NewOCOOrder creates a new OCO order
func NewOCOOrder(
	ocoID, trader, marketID string,
	stopOrder *ConditionalOrder,
	limitOrder *Order,
) *OCOOrder {
	now := time.Now()
	return &OCOOrder{
		OCOID:      ocoID,
		Trader:     trader,
		MarketID:   marketID,
		StopOrder:  stopOrder,
		LimitOrder: limitOrder,
		Status:     OCOStatusPending,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
}

// IsActive returns true if the OCO is still active
func (o *OCOOrder) IsActive() bool {
	return o.Status == OCOStatusPending
}

// TriggerStop triggers the stop order and cancels the limit order
func (o *OCOOrder) TriggerStop() *Order {
	o.Status = OCOStatusTriggered
	o.TriggeredID = o.StopOrder.OrderID
	o.UpdatedAt = time.Now()

	// Cancel the limit order
	o.LimitOrder.Status = OrderStatusCancelled

	// Trigger the stop order
	return o.StopOrder.Trigger()
}

// TriggerLimit triggers the limit order and cancels the stop order
func (o *OCOOrder) TriggerLimit() {
	o.Status = OCOStatusTriggered
	o.TriggeredID = o.LimitOrder.OrderID
	o.UpdatedAt = time.Now()

	// Cancel the stop order
	o.StopOrder.Cancel()

	// The limit order is already in the order book
}

// Cancel cancels both orders
func (o *OCOOrder) Cancel() {
	o.Status = OCOStatusCancelled
	o.UpdatedAt = time.Now()

	o.StopOrder.Cancel()
	o.LimitOrder.Status = OrderStatusCancelled
}

// CheckTrigger checks if either order should be triggered
// Returns: triggered order type ("stop", "limit", or "")
func (o *OCOOrder) CheckTrigger(markPrice math.LegacyDec) string {
	if !o.IsActive() {
		return ""
	}

	// Check if stop order should trigger
	if o.StopOrder.ShouldTrigger(markPrice) {
		return "stop"
	}

	// Check if limit order is filled (handled by matching engine)
	// Here we just return empty if no trigger
	return ""
}
