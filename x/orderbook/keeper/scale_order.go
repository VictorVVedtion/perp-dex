package keeper

import (
	"fmt"
	"time"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/openalpha/perp-dex/x/orderbook/types"
)

// ScaleDistribution represents the distribution type for scale orders
type ScaleDistribution string

const (
	// DistributionLinear distributes orders evenly across price range
	DistributionLinear ScaleDistribution = "linear"

	// DistributionExponential distributes more orders near current price
	DistributionExponential ScaleDistribution = "exponential"

	// DistributionDescending distributes more quantity at better prices
	DistributionDescending ScaleDistribution = "descending"
)

// ScaleOrderStatus represents the status of a scale order
type ScaleOrderStatus int

const (
	ScaleOrderStatusPending ScaleOrderStatus = iota
	ScaleOrderStatusActive
	ScaleOrderStatusPartiallyFilled
	ScaleOrderStatusFilled
	ScaleOrderStatusCancelled
)

// String returns the string representation of ScaleOrderStatus
func (s ScaleOrderStatus) String() string {
	switch s {
	case ScaleOrderStatusPending:
		return "pending"
	case ScaleOrderStatusActive:
		return "active"
	case ScaleOrderStatusPartiallyFilled:
		return "partially_filled"
	case ScaleOrderStatusFilled:
		return "filled"
	case ScaleOrderStatusCancelled:
		return "cancelled"
	default:
		return "unknown"
	}
}

// ScaleOrder represents a scale order that creates multiple limit orders across a price range
// Aligned with Hyperliquid's scale order functionality
type ScaleOrder struct {
	ScaleOrderID  string            // Unique identifier
	Trader        string            // Trader address
	MarketID      string            // Market identifier
	Side          types.Side        // Buy or Sell
	TotalQuantity math.LegacyDec    // Total quantity across all sub-orders
	FilledQty     math.LegacyDec    // Total filled quantity
	PriceStart    math.LegacyDec    // Start of price range
	PriceEnd      math.LegacyDec    // End of price range
	OrderCount    int               // Number of sub-orders (typically 5-20)
	Distribution  ScaleDistribution // How to distribute orders
	ReduceOnly    bool              // Only reduce position
	PostOnly      bool              // Post-only orders
	SubOrders     []*types.Order    // Generated sub-orders
	Status        ScaleOrderStatus  // Current status
	CreatedAt     time.Time         // Creation time
	UpdatedAt     time.Time         // Last update time
}

// NewScaleOrder creates a new scale order
func NewScaleOrder(
	scaleOrderID, trader, marketID string,
	side types.Side,
	totalQuantity, priceStart, priceEnd math.LegacyDec,
	orderCount int,
	distribution ScaleDistribution,
	reduceOnly, postOnly bool,
) (*ScaleOrder, error) {
	// Validate inputs
	if orderCount < 2 || orderCount > 50 {
		return nil, fmt.Errorf("order count must be between 2 and 50, got %d", orderCount)
	}

	if totalQuantity.LTE(math.LegacyZeroDec()) {
		return nil, fmt.Errorf("total quantity must be positive")
	}

	if priceStart.LTE(math.LegacyZeroDec()) || priceEnd.LTE(math.LegacyZeroDec()) {
		return nil, fmt.Errorf("prices must be positive")
	}

	if priceStart.Equal(priceEnd) {
		return nil, fmt.Errorf("price start and end must be different")
	}

	// For buy orders, start should be lower than end (buying from low to high)
	// For sell orders, start should be higher than end (selling from high to low)
	if side == types.SideBuy && priceStart.GT(priceEnd) {
		priceStart, priceEnd = priceEnd, priceStart
	} else if side == types.SideSell && priceStart.LT(priceEnd) {
		priceStart, priceEnd = priceEnd, priceStart
	}

	now := time.Now()
	return &ScaleOrder{
		ScaleOrderID:  scaleOrderID,
		Trader:        trader,
		MarketID:      marketID,
		Side:          side,
		TotalQuantity: totalQuantity,
		FilledQty:     math.LegacyZeroDec(),
		PriceStart:    priceStart,
		PriceEnd:      priceEnd,
		OrderCount:    orderCount,
		Distribution:  distribution,
		ReduceOnly:    reduceOnly,
		PostOnly:      postOnly,
		SubOrders:     make([]*types.Order, 0, orderCount),
		Status:        ScaleOrderStatusPending,
		CreatedAt:     now,
		UpdatedAt:     now,
	}, nil
}

// GenerateSubOrders generates the individual limit orders based on distribution
func (so *ScaleOrder) GenerateSubOrders(orderIDPrefix string) []*types.Order {
	so.SubOrders = make([]*types.Order, 0, so.OrderCount)

	// Calculate prices based on distribution
	prices := so.calculatePrices()

	// Calculate quantities based on distribution
	quantities := so.calculateQuantities()

	for i := 0; i < so.OrderCount; i++ {
		orderID := fmt.Sprintf("%s-sub-%d", orderIDPrefix, i)

		// Create time-in-force flags
		timeInForce := types.TimeInForceGTC
		if so.PostOnly {
			timeInForce = types.TimeInForceGTX // Post-only
		}

		order := types.NewExtendedOrder(
			orderID,
			so.Trader,
			so.MarketID,
			so.Side,
			types.OrderTypeLimit,
			prices[i],
			quantities[i],
			timeInForce,
			types.OrderFlags{
				ReduceOnly: so.ReduceOnly,
				PostOnly:   so.PostOnly,
			},
		)

		so.SubOrders = append(so.SubOrders, order.ToOrder())
	}

	so.Status = ScaleOrderStatusActive
	so.UpdatedAt = time.Now()

	return so.SubOrders
}

// calculatePrices calculates the prices for each sub-order based on distribution
func (so *ScaleOrder) calculatePrices() []math.LegacyDec {
	prices := make([]math.LegacyDec, so.OrderCount)
	priceRange := so.PriceEnd.Sub(so.PriceStart)

	switch so.Distribution {
	case DistributionExponential:
		// Exponential distribution - more orders near the start price
		for i := 0; i < so.OrderCount; i++ {
			// Use exponential spacing
			ratio := math.LegacyNewDec(int64(i)).Quo(math.LegacyNewDec(int64(so.OrderCount - 1)))
			expRatio := ratio.Mul(ratio) // Square for exponential effect
			prices[i] = so.PriceStart.Add(priceRange.Mul(expRatio))
		}

	case DistributionDescending:
		// More quantity at better prices (linear prices, descending quantities handled separately)
		for i := 0; i < so.OrderCount; i++ {
			ratio := math.LegacyNewDec(int64(i)).Quo(math.LegacyNewDec(int64(so.OrderCount - 1)))
			prices[i] = so.PriceStart.Add(priceRange.Mul(ratio))
		}

	default: // Linear
		// Linear distribution - evenly spaced
		for i := 0; i < so.OrderCount; i++ {
			ratio := math.LegacyNewDec(int64(i)).Quo(math.LegacyNewDec(int64(so.OrderCount - 1)))
			prices[i] = so.PriceStart.Add(priceRange.Mul(ratio))
		}
	}

	return prices
}

// calculateQuantities calculates the quantities for each sub-order based on distribution
func (so *ScaleOrder) calculateQuantities() []math.LegacyDec {
	quantities := make([]math.LegacyDec, so.OrderCount)
	orderCountDec := math.LegacyNewDec(int64(so.OrderCount))

	switch so.Distribution {
	case DistributionDescending:
		// Descending distribution - more quantity at better prices
		// Weight: n, n-1, n-2, ..., 1
		totalWeight := int64(0)
		for i := 0; i < so.OrderCount; i++ {
			totalWeight += int64(so.OrderCount - i)
		}
		totalWeightDec := math.LegacyNewDec(totalWeight)

		for i := 0; i < so.OrderCount; i++ {
			weight := math.LegacyNewDec(int64(so.OrderCount - i))
			quantities[i] = so.TotalQuantity.Mul(weight).Quo(totalWeightDec)
		}

	case DistributionExponential:
		// Exponential distribution for quantities - more at start
		totalWeight := math.LegacyZeroDec()
		weights := make([]math.LegacyDec, so.OrderCount)
		for i := 0; i < so.OrderCount; i++ {
			// Weight decreases exponentially
			ratio := math.LegacyOneDec().Sub(
				math.LegacyNewDec(int64(i)).Quo(orderCountDec),
			)
			weights[i] = ratio.Mul(ratio).Add(math.LegacyNewDecWithPrec(1, 1)) // Square + 0.1 minimum
			totalWeight = totalWeight.Add(weights[i])
		}

		for i := 0; i < so.OrderCount; i++ {
			quantities[i] = so.TotalQuantity.Mul(weights[i]).Quo(totalWeight)
		}

	default: // Linear
		// Equal distribution
		qtyPerOrder := so.TotalQuantity.Quo(orderCountDec)
		for i := 0; i < so.OrderCount; i++ {
			quantities[i] = qtyPerOrder
		}
	}

	return quantities
}

// UpdateFillStatus updates the scale order's fill status based on sub-order fills
func (so *ScaleOrder) UpdateFillStatus() {
	totalFilled := math.LegacyZeroDec()
	activeOrders := 0
	filledOrders := 0

	for _, order := range so.SubOrders {
		totalFilled = totalFilled.Add(order.FilledQty)
		if order.Status == types.OrderStatusFilled {
			filledOrders++
		} else if order.IsActive() {
			activeOrders++
		}
	}

	so.FilledQty = totalFilled
	so.UpdatedAt = time.Now()

	// Update status
	if filledOrders == len(so.SubOrders) {
		so.Status = ScaleOrderStatusFilled
	} else if totalFilled.IsPositive() {
		so.Status = ScaleOrderStatusPartiallyFilled
	} else if activeOrders == 0 {
		so.Status = ScaleOrderStatusCancelled
	}
}

// Cancel cancels the scale order and all its sub-orders
func (so *ScaleOrder) Cancel() {
	for _, order := range so.SubOrders {
		if order.IsActive() {
			order.Cancel()
		}
	}
	so.Status = ScaleOrderStatusCancelled
	so.UpdatedAt = time.Now()
}

// GetFillPercentage returns the fill percentage
func (so *ScaleOrder) GetFillPercentage() math.LegacyDec {
	if so.TotalQuantity.IsZero() {
		return math.LegacyZeroDec()
	}
	return so.FilledQty.Quo(so.TotalQuantity).Mul(math.LegacyNewDec(100))
}

// IsActive returns true if the scale order is still active
func (so *ScaleOrder) IsActive() bool {
	return so.Status == ScaleOrderStatusActive || so.Status == ScaleOrderStatusPartiallyFilled
}

// ScaleOrderManager manages scale orders
type ScaleOrderManager struct {
	keeper      *Keeper
	scaleOrders map[string]*ScaleOrder // scaleOrderID -> ScaleOrder
}

// NewScaleOrderManager creates a new scale order manager
func NewScaleOrderManager(keeper *Keeper) *ScaleOrderManager {
	return &ScaleOrderManager{
		keeper:      keeper,
		scaleOrders: make(map[string]*ScaleOrder),
	}
}

// CreateScaleOrder creates a new scale order and its sub-orders
func (m *ScaleOrderManager) CreateScaleOrder(
	ctx sdk.Context,
	trader, marketID string,
	side types.Side,
	totalQuantity, priceStart, priceEnd math.LegacyDec,
	orderCount int,
	distribution ScaleDistribution,
	reduceOnly, postOnly bool,
) (*ScaleOrder, error) {
	// Generate scale order ID
	scaleOrderID := fmt.Sprintf("scale-%s-%d", trader[:8], ctx.BlockHeight())

	// Create scale order
	scaleOrder, err := NewScaleOrder(
		scaleOrderID, trader, marketID,
		side, totalQuantity, priceStart, priceEnd,
		orderCount, distribution, reduceOnly, postOnly,
	)
	if err != nil {
		return nil, err
	}

	// Generate sub-orders
	subOrders := scaleOrder.GenerateSubOrders(scaleOrderID)

	// Place sub-orders on the order book
	for _, order := range subOrders {
		// Add to order book through keeper
		// In production, this would call the matching engine
		m.keeper.SetOrder(ctx, order)
	}

	// Store scale order
	m.scaleOrders[scaleOrderID] = scaleOrder

	// Emit event
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"scale_order_created",
			sdk.NewAttribute("scale_order_id", scaleOrderID),
			sdk.NewAttribute("trader", trader),
			sdk.NewAttribute("market_id", marketID),
			sdk.NewAttribute("side", side.String()),
			sdk.NewAttribute("total_quantity", totalQuantity.String()),
			sdk.NewAttribute("price_start", priceStart.String()),
			sdk.NewAttribute("price_end", priceEnd.String()),
			sdk.NewAttribute("order_count", fmt.Sprintf("%d", orderCount)),
			sdk.NewAttribute("distribution", string(distribution)),
		),
	)

	return scaleOrder, nil
}

// CancelScaleOrder cancels a scale order and all its sub-orders
func (m *ScaleOrderManager) CancelScaleOrder(ctx sdk.Context, scaleOrderID string) error {
	scaleOrder, exists := m.scaleOrders[scaleOrderID]
	if !exists {
		return fmt.Errorf("scale order not found: %s", scaleOrderID)
	}

	// Cancel all active sub-orders
	for _, order := range scaleOrder.SubOrders {
		if order.IsActive() {
			// Remove from order book
			m.keeper.DeleteOrder(ctx, order.OrderID)
		}
	}

	scaleOrder.Cancel()

	// Emit event
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"scale_order_cancelled",
			sdk.NewAttribute("scale_order_id", scaleOrderID),
			sdk.NewAttribute("filled_quantity", scaleOrder.FilledQty.String()),
		),
	)

	return nil
}

// GetScaleOrder returns a scale order by ID
func (m *ScaleOrderManager) GetScaleOrder(scaleOrderID string) *ScaleOrder {
	return m.scaleOrders[scaleOrderID]
}

// GetScaleOrdersByTrader returns all scale orders for a trader
func (m *ScaleOrderManager) GetScaleOrdersByTrader(trader string) []*ScaleOrder {
	var orders []*ScaleOrder
	for _, order := range m.scaleOrders {
		if order.Trader == trader {
			orders = append(orders, order)
		}
	}
	return orders
}

// GetActiveScaleOrders returns all active scale orders
func (m *ScaleOrderManager) GetActiveScaleOrders() []*ScaleOrder {
	var orders []*ScaleOrder
	for _, order := range m.scaleOrders {
		if order.IsActive() {
			orders = append(orders, order)
		}
	}
	return orders
}

// UpdateScaleOrderFills updates fill status for all scale orders
// Called after trades are executed
func (m *ScaleOrderManager) UpdateScaleOrderFills(ctx sdk.Context) {
	for _, scaleOrder := range m.scaleOrders {
		if scaleOrder.IsActive() {
			// Update sub-order statuses from order book
			for _, subOrder := range scaleOrder.SubOrders {
				storedOrder := m.keeper.GetOrder(ctx, subOrder.OrderID)
				if storedOrder != nil {
					subOrder.FilledQty = storedOrder.FilledQty
					subOrder.Status = storedOrder.Status
				}
			}
			scaleOrder.UpdateFillStatus()
		}
	}
}

// CleanupCompletedOrders removes completed scale orders from memory
func (m *ScaleOrderManager) CleanupCompletedOrders() {
	for id, order := range m.scaleOrders {
		if order.Status == ScaleOrderStatusFilled || order.Status == ScaleOrderStatusCancelled {
			delete(m.scaleOrders, id)
		}
	}
}
