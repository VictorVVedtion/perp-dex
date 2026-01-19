package keeper

import (
	"fmt"
	"time"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/openalpha/perp-dex/x/orderbook/types"
)

// TWAPStatus represents the status of a TWAP order
type TWAPStatus int

const (
	TWAPStatusPending TWAPStatus = iota
	TWAPStatusActive
	TWAPStatusCompleted
	TWAPStatusCancelled
	TWAPStatusFailed
)

// String returns the string representation of TWAPStatus
func (s TWAPStatus) String() string {
	switch s {
	case TWAPStatusPending:
		return "pending"
	case TWAPStatusActive:
		return "active"
	case TWAPStatusCompleted:
		return "completed"
	case TWAPStatusCancelled:
		return "cancelled"
	case TWAPStatusFailed:
		return "failed"
	default:
		return "unknown"
	}
}

// Default TWAP configuration aligned with Hyperliquid
const (
	// DefaultTWAPInterval is the interval between sub-orders (30 seconds)
	DefaultTWAPInterval = 30 * time.Second

	// DefaultMaxSlippage is the maximum allowed slippage (3%)
	DefaultMaxSlippage = 3

	// MaxCatchUpMultiplier is the maximum multiplier for catch-up orders
	MaxCatchUpMultiplier = 3
)

// TWAPOrder represents a Time-Weighted Average Price order
// Divides a large order into smaller chunks executed over time
// Aligned with Hyperliquid's TWAP implementation
type TWAPOrder struct {
	TWAPOrderID     string         // Unique identifier
	Trader          string         // Trader address
	MarketID        string         // Market identifier
	Side            types.Side     // Buy or Sell
	TotalQuantity   math.LegacyDec // Total quantity to execute
	ExecutedQty     math.LegacyDec // Quantity already executed
	AvgExecutedPrice math.LegacyDec // Volume-weighted average execution price
	Duration        time.Duration  // Total execution duration
	Interval        time.Duration  // Interval between sub-orders (default 30s)
	MaxSlippage     math.LegacyDec // Maximum slippage tolerance (default 3%)
	ReduceOnly      bool           // Only reduce position

	// Execution tracking
	SubOrdersTotal    int       // Total number of planned sub-orders
	SubOrdersExecuted int       // Number of sub-orders executed
	SubOrdersPending  int       // Number of sub-orders pending
	CurrentSubOrderID string    // Current active sub-order ID
	CatchUpQuantity   math.LegacyDec // Accumulated quantity to catch up

	// Timing
	StartTime       time.Time // When TWAP execution started
	EndTime         time.Time // When TWAP should complete
	NextExecutionAt time.Time // Next sub-order execution time
	LastExecutionAt time.Time // Last sub-order execution time

	Status    TWAPStatus // Current status
	CreatedAt time.Time  // Creation time
	UpdatedAt time.Time  // Last update time

	// Error tracking
	LastError     string // Last error message
	FailedRetries int    // Number of failed retries
}

// NewTWAPOrder creates a new TWAP order
func NewTWAPOrder(
	twapOrderID, trader, marketID string,
	side types.Side,
	totalQuantity math.LegacyDec,
	duration time.Duration,
	maxSlippage math.LegacyDec,
	reduceOnly bool,
) (*TWAPOrder, error) {
	// Validate inputs
	if totalQuantity.LTE(math.LegacyZeroDec()) {
		return nil, fmt.Errorf("total quantity must be positive")
	}

	if duration < time.Minute {
		return nil, fmt.Errorf("duration must be at least 1 minute")
	}

	if maxSlippage.LTE(math.LegacyZeroDec()) || maxSlippage.GT(math.LegacyNewDec(10)) {
		return nil, fmt.Errorf("max slippage must be between 0 and 10 percent")
	}

	// Calculate number of sub-orders
	subOrdersTotal := int(duration / DefaultTWAPInterval)
	if subOrdersTotal < 2 {
		subOrdersTotal = 2
	}

	now := time.Now()
	return &TWAPOrder{
		TWAPOrderID:       twapOrderID,
		Trader:            trader,
		MarketID:          marketID,
		Side:              side,
		TotalQuantity:     totalQuantity,
		ExecutedQty:       math.LegacyZeroDec(),
		AvgExecutedPrice:  math.LegacyZeroDec(),
		Duration:          duration,
		Interval:          DefaultTWAPInterval,
		MaxSlippage:       maxSlippage,
		ReduceOnly:        reduceOnly,
		SubOrdersTotal:    subOrdersTotal,
		SubOrdersExecuted: 0,
		SubOrdersPending:  subOrdersTotal,
		CatchUpQuantity:   math.LegacyZeroDec(),
		Status:            TWAPStatusPending,
		CreatedAt:         now,
		UpdatedAt:         now,
	}, nil
}

// Start begins the TWAP execution
func (t *TWAPOrder) Start() {
	t.StartTime = time.Now()
	t.EndTime = t.StartTime.Add(t.Duration)
	t.NextExecutionAt = t.StartTime
	t.Status = TWAPStatusActive
	t.UpdatedAt = time.Now()
}

// GetTargetQuantityForInterval calculates the target quantity for the current interval
// Implements catch-up logic: if previous sub-orders didn't fully fill,
// subsequent orders increase up to 3x to catch up
func (t *TWAPOrder) GetTargetQuantityForInterval() math.LegacyDec {
	if t.SubOrdersPending <= 0 {
		return math.LegacyZeroDec()
	}

	// Base quantity per interval
	remainingQty := t.TotalQuantity.Sub(t.ExecutedQty)
	baseQty := remainingQty.Quo(math.LegacyNewDec(int64(t.SubOrdersPending)))

	// Add catch-up quantity (max 3x normal size)
	targetQty := baseQty.Add(t.CatchUpQuantity)
	maxQty := baseQty.Mul(math.LegacyNewDec(MaxCatchUpMultiplier))

	if targetQty.GT(maxQty) {
		targetQty = maxQty
	}

	// Don't exceed remaining quantity
	if targetQty.GT(remainingQty) {
		targetQty = remainingQty
	}

	return targetQty
}

// GetElapsedRatio returns the ratio of elapsed time to total duration
func (t *TWAPOrder) GetElapsedRatio() math.LegacyDec {
	if t.Status != TWAPStatusActive {
		return math.LegacyZeroDec()
	}

	elapsed := time.Since(t.StartTime)
	elapsedMs := math.LegacyNewDec(elapsed.Milliseconds())
	durationMs := math.LegacyNewDec(t.Duration.Milliseconds())

	if durationMs.IsZero() {
		return math.LegacyOneDec()
	}

	ratio := elapsedMs.Quo(durationMs)
	if ratio.GT(math.LegacyOneDec()) {
		return math.LegacyOneDec()
	}
	return ratio
}

// GetTargetExecutedRatio returns the target execution ratio based on elapsed time
func (t *TWAPOrder) GetTargetExecutedRatio() math.LegacyDec {
	return t.GetElapsedRatio()
}

// GetActualExecutedRatio returns the actual execution ratio
func (t *TWAPOrder) GetActualExecutedRatio() math.LegacyDec {
	if t.TotalQuantity.IsZero() {
		return math.LegacyZeroDec()
	}
	return t.ExecutedQty.Quo(t.TotalQuantity)
}

// IsOnTrack checks if execution is on track
func (t *TWAPOrder) IsOnTrack() bool {
	targetRatio := t.GetTargetExecutedRatio()
	actualRatio := t.GetActualExecutedRatio()

	// Allow 20% deviation
	deviation := targetRatio.Sub(actualRatio).Abs()
	maxDeviation := math.LegacyNewDecWithPrec(2, 1) // 20%

	return deviation.LTE(maxDeviation)
}

// ShouldExecute checks if it's time to execute the next sub-order
func (t *TWAPOrder) ShouldExecute(currentTime time.Time) bool {
	if t.Status != TWAPStatusActive {
		return false
	}

	// Check if we've passed the end time
	if currentTime.After(t.EndTime) {
		return true // Execute remaining quantity
	}

	// Check if it's time for next execution
	return currentTime.After(t.NextExecutionAt) || currentTime.Equal(t.NextExecutionAt)
}

// RecordExecution records a sub-order execution
func (t *TWAPOrder) RecordExecution(executedQty, executedPrice math.LegacyDec, subOrderID string) {
	// Update volume-weighted average price
	if t.ExecutedQty.IsZero() {
		t.AvgExecutedPrice = executedPrice
	} else {
		totalValue := t.AvgExecutedPrice.Mul(t.ExecutedQty).Add(executedPrice.Mul(executedQty))
		newTotalQty := t.ExecutedQty.Add(executedQty)
		if newTotalQty.IsPositive() {
			t.AvgExecutedPrice = totalValue.Quo(newTotalQty)
		}
	}

	t.ExecutedQty = t.ExecutedQty.Add(executedQty)
	t.SubOrdersExecuted++
	t.SubOrdersPending--
	t.LastExecutionAt = time.Now()
	t.NextExecutionAt = t.LastExecutionAt.Add(t.Interval)
	t.CurrentSubOrderID = ""
	t.UpdatedAt = time.Now()

	// Check if complete
	if t.ExecutedQty.GTE(t.TotalQuantity) {
		t.Status = TWAPStatusCompleted
	}
}

// RecordPartialFill records a partial fill for the current sub-order
func (t *TWAPOrder) RecordPartialFill(filledQty, targetQty math.LegacyDec) {
	// Calculate unfilled quantity to add to catch-up
	unfilledQty := targetQty.Sub(filledQty)
	if unfilledQty.IsPositive() {
		t.CatchUpQuantity = t.CatchUpQuantity.Add(unfilledQty)
	}
}

// RecordFailure records a failed sub-order execution
func (t *TWAPOrder) RecordFailure(errorMsg string) {
	t.LastError = errorMsg
	t.FailedRetries++
	t.UpdatedAt = time.Now()

	// Fail after too many retries
	if t.FailedRetries >= 5 {
		t.Status = TWAPStatusFailed
	}
}

// Cancel cancels the TWAP order
func (t *TWAPOrder) Cancel() {
	t.Status = TWAPStatusCancelled
	t.UpdatedAt = time.Now()
}

// IsActive returns true if the TWAP order is still active
func (t *TWAPOrder) IsActive() bool {
	return t.Status == TWAPStatusActive
}

// GetProgress returns progress information
func (t *TWAPOrder) GetProgress() (executedPct, targetPct math.LegacyDec, onTrack bool) {
	executedPct = t.GetActualExecutedRatio().Mul(math.LegacyNewDec(100))
	targetPct = t.GetTargetExecutedRatio().Mul(math.LegacyNewDec(100))
	onTrack = t.IsOnTrack()
	return
}

// TWAPManager manages TWAP orders
type TWAPManager struct {
	keeper     *Keeper
	twapOrders map[string]*TWAPOrder // twapOrderID -> TWAPOrder
}

// NewTWAPManager creates a new TWAP manager
func NewTWAPManager(keeper *Keeper) *TWAPManager {
	return &TWAPManager{
		keeper:     keeper,
		twapOrders: make(map[string]*TWAPOrder),
	}
}

// CreateTWAPOrder creates a new TWAP order
func (m *TWAPManager) CreateTWAPOrder(
	ctx sdk.Context,
	trader, marketID string,
	side types.Side,
	totalQuantity math.LegacyDec,
	duration time.Duration,
	maxSlippage math.LegacyDec,
	reduceOnly bool,
) (*TWAPOrder, error) {
	// Generate TWAP order ID
	twapOrderID := fmt.Sprintf("twap-%s-%d", trader[:8], ctx.BlockHeight())

	// Create TWAP order
	twapOrder, err := NewTWAPOrder(
		twapOrderID, trader, marketID,
		side, totalQuantity, duration,
		maxSlippage, reduceOnly,
	)
	if err != nil {
		return nil, err
	}

	// Start execution
	twapOrder.Start()

	// Store order
	m.twapOrders[twapOrderID] = twapOrder

	// Emit event
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"twap_order_created",
			sdk.NewAttribute("twap_order_id", twapOrderID),
			sdk.NewAttribute("trader", trader),
			sdk.NewAttribute("market_id", marketID),
			sdk.NewAttribute("side", side.String()),
			sdk.NewAttribute("total_quantity", totalQuantity.String()),
			sdk.NewAttribute("duration_seconds", fmt.Sprintf("%d", int(duration.Seconds()))),
			sdk.NewAttribute("max_slippage_pct", maxSlippage.String()),
			sdk.NewAttribute("sub_orders_total", fmt.Sprintf("%d", twapOrder.SubOrdersTotal)),
		),
	)

	return twapOrder, nil
}

// ProcessTWAPOrders processes all active TWAP orders
// Should be called at the end of each block or on a timer
func (m *TWAPManager) ProcessTWAPOrders(ctx sdk.Context) {
	currentTime := ctx.BlockTime()

	for _, twapOrder := range m.twapOrders {
		if !twapOrder.IsActive() {
			continue
		}

		// Check if should execute
		if !twapOrder.ShouldExecute(currentTime) {
			continue
		}

		// Execute sub-order
		m.executeSubOrder(ctx, twapOrder)
	}
}

// executeSubOrder executes a single sub-order for a TWAP order
func (m *TWAPManager) executeSubOrder(ctx sdk.Context, twapOrder *TWAPOrder) {
	// Calculate target quantity for this interval
	targetQty := twapOrder.GetTargetQuantityForInterval()
	if targetQty.LTE(math.LegacyZeroDec()) {
		return
	}

	// Get current market price for slippage check
	// In production, this would come from the oracle
	// For now, we'll create the order and let the matching engine handle it

	// Generate sub-order ID
	subOrderID := fmt.Sprintf("%s-sub-%d", twapOrder.TWAPOrderID, twapOrder.SubOrdersExecuted)
	twapOrder.CurrentSubOrderID = subOrderID

	// Create market order (TWAP uses market orders by default)
	order := types.NewOrder(
		subOrderID,
		twapOrder.Trader,
		twapOrder.MarketID,
		twapOrder.Side,
		types.OrderTypeMarket,
		math.LegacyZeroDec(), // Market order - no limit price
		targetQty,
	)

	// Place order
	m.keeper.SetOrder(ctx, order)

	// For MVP, simulate immediate execution at market price
	// In production, this would go through the matching engine
	// and we'd wait for fills

	// Emit sub-order event
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"twap_sub_order_created",
			sdk.NewAttribute("twap_order_id", twapOrder.TWAPOrderID),
			sdk.NewAttribute("sub_order_id", subOrderID),
			sdk.NewAttribute("target_quantity", targetQty.String()),
			sdk.NewAttribute("sub_order_number", fmt.Sprintf("%d/%d", twapOrder.SubOrdersExecuted+1, twapOrder.SubOrdersTotal)),
		),
	)

	// Record execution (in production, wait for actual fills)
	// For MVP, assume immediate fill at current mark price
	// This would be updated by the matching engine callback
}

// OnOrderFilled callback when a TWAP sub-order is filled
func (m *TWAPManager) OnOrderFilled(ctx sdk.Context, orderID string, filledQty, filledPrice math.LegacyDec) {
	// Find the TWAP order for this sub-order
	for _, twapOrder := range m.twapOrders {
		if twapOrder.CurrentSubOrderID == orderID {
			twapOrder.RecordExecution(filledQty, filledPrice, orderID)

			// Emit fill event
			executedPct, targetPct, onTrack := twapOrder.GetProgress()
			ctx.EventManager().EmitEvent(
				sdk.NewEvent(
					"twap_sub_order_filled",
					sdk.NewAttribute("twap_order_id", twapOrder.TWAPOrderID),
					sdk.NewAttribute("sub_order_id", orderID),
					sdk.NewAttribute("filled_quantity", filledQty.String()),
					sdk.NewAttribute("filled_price", filledPrice.String()),
					sdk.NewAttribute("total_executed", twapOrder.ExecutedQty.String()),
					sdk.NewAttribute("avg_price", twapOrder.AvgExecutedPrice.String()),
					sdk.NewAttribute("executed_pct", executedPct.String()),
					sdk.NewAttribute("target_pct", targetPct.String()),
					sdk.NewAttribute("on_track", fmt.Sprintf("%t", onTrack)),
				),
			)

			// Check if completed
			if twapOrder.Status == TWAPStatusCompleted {
				ctx.EventManager().EmitEvent(
					sdk.NewEvent(
						"twap_order_completed",
						sdk.NewAttribute("twap_order_id", twapOrder.TWAPOrderID),
						sdk.NewAttribute("total_executed", twapOrder.ExecutedQty.String()),
						sdk.NewAttribute("avg_executed_price", twapOrder.AvgExecutedPrice.String()),
					),
				)
			}
			return
		}
	}
}

// CancelTWAPOrder cancels a TWAP order
func (m *TWAPManager) CancelTWAPOrder(ctx sdk.Context, twapOrderID string) error {
	twapOrder, exists := m.twapOrders[twapOrderID]
	if !exists {
		return fmt.Errorf("TWAP order not found: %s", twapOrderID)
	}

	// Cancel current sub-order if active
	if twapOrder.CurrentSubOrderID != "" {
		m.keeper.DeleteOrder(ctx, twapOrder.CurrentSubOrderID)
	}

	twapOrder.Cancel()

	// Emit event
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"twap_order_cancelled",
			sdk.NewAttribute("twap_order_id", twapOrderID),
			sdk.NewAttribute("executed_quantity", twapOrder.ExecutedQty.String()),
			sdk.NewAttribute("remaining_quantity", twapOrder.TotalQuantity.Sub(twapOrder.ExecutedQty).String()),
		),
	)

	return nil
}

// GetTWAPOrder returns a TWAP order by ID
func (m *TWAPManager) GetTWAPOrder(twapOrderID string) *TWAPOrder {
	return m.twapOrders[twapOrderID]
}

// GetTWAPOrdersByTrader returns all TWAP orders for a trader
func (m *TWAPManager) GetTWAPOrdersByTrader(trader string) []*TWAPOrder {
	var orders []*TWAPOrder
	for _, order := range m.twapOrders {
		if order.Trader == trader {
			orders = append(orders, order)
		}
	}
	return orders
}

// GetActiveTWAPOrders returns all active TWAP orders
func (m *TWAPManager) GetActiveTWAPOrders() []*TWAPOrder {
	var orders []*TWAPOrder
	for _, order := range m.twapOrders {
		if order.IsActive() {
			orders = append(orders, order)
		}
	}
	return orders
}

// CleanupCompletedOrders removes completed TWAP orders from memory
func (m *TWAPManager) CleanupCompletedOrders() {
	for id, order := range m.twapOrders {
		if order.Status == TWAPStatusCompleted || order.Status == TWAPStatusCancelled || order.Status == TWAPStatusFailed {
			delete(m.twapOrders, id)
		}
	}
}
