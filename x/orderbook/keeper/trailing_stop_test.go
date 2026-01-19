package keeper

import (
	"testing"

	"cosmossdk.io/math"
	"github.com/openalpha/perp-dex/x/orderbook/types"
)

// TestTrailingStopOrder_NewTrailingStopOrder tests creating a new trailing stop order
func TestTrailingStopOrder_NewTrailingStopOrder(t *testing.T) {
	orderID := "trail-1"
	trader := "cosmos1abc..."
	marketID := "BTC-USDC"
	side := types.SideSell
	quantity := math.LegacyNewDec(1)
	trailAmount := math.LegacyNewDec(100) // $100 trail distance
	activationPrice := math.LegacyNewDec(50000)

	order := types.NewTrailingStopOrder(orderID, trader, marketID, side, quantity, trailAmount, activationPrice)

	if order.OrderID != orderID {
		t.Errorf("expected order ID %s, got %s", orderID, order.OrderID)
	}
	if order.Trader != trader {
		t.Errorf("expected trader %s, got %s", trader, order.Trader)
	}
	if order.Side != side {
		t.Errorf("expected side %v, got %v", side, order.Side)
	}
	if !order.Quantity.Equal(quantity) {
		t.Errorf("expected quantity %s, got %s", quantity.String(), order.Quantity.String())
	}
	if !order.TrailAmount.Equal(trailAmount) {
		t.Errorf("expected trail amount %s, got %s", trailAmount.String(), order.TrailAmount.String())
	}
	if order.Status != types.OrderStatusOpen {
		t.Errorf("expected status open, got %s", order.Status.String())
	}
}

// TestTrailingStopOrder_NewTrailingStopOrderPercent tests creating a percentage-based trailing stop
func TestTrailingStopOrder_NewTrailingStopOrderPercent(t *testing.T) {
	orderID := "trail-2"
	trader := "cosmos1abc..."
	marketID := "BTC-USDC"
	side := types.SideSell
	quantity := math.LegacyNewDec(1)
	trailPercent := math.LegacyNewDecWithPrec(2, 2) // 2%
	activationPrice := math.LegacyNewDec(50000)

	order := types.NewTrailingStopOrderPercent(orderID, trader, marketID, side, quantity, trailPercent, activationPrice)

	if !order.TrailPercent.Equal(trailPercent) {
		t.Errorf("expected trail percent %s, got %s", trailPercent.String(), order.TrailPercent.String())
	}
}

// TestTrailingStopOrder_GetTrailDistance tests trail distance calculation
func TestTrailingStopOrder_GetTrailDistance(t *testing.T) {
	tests := []struct {
		name          string
		trailAmount   math.LegacyDec
		trailPercent  math.LegacyDec
		currentPrice  math.LegacyDec
		expectedDist  math.LegacyDec
	}{
		{
			name:         "fixed amount",
			trailAmount:  math.LegacyNewDec(100),
			trailPercent: math.LegacyZeroDec(),
			currentPrice: math.LegacyNewDec(50000),
			expectedDist: math.LegacyNewDec(100),
		},
		{
			name:         "percentage 2%",
			trailAmount:  math.LegacyZeroDec(),
			trailPercent: math.LegacyNewDec(2), // 2% (function divides by 100)
			currentPrice: math.LegacyNewDec(50000),
			expectedDist: math.LegacyNewDec(1000), // 50000 * 2 / 100 = 1000
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			order := &types.TrailingStopOrder{
				TrailAmount:  tt.trailAmount,
				TrailPercent: tt.trailPercent,
			}

			dist := order.GetTrailDistance(tt.currentPrice)
			if !dist.Equal(tt.expectedDist) {
				t.Errorf("expected distance %s, got %s", tt.expectedDist.String(), dist.String())
			}
		})
	}
}

// TestTrailingStopOrder_Update_LongStop tests updating a trailing stop for a long position
func TestTrailingStopOrder_Update_LongStop(t *testing.T) {
	// Long position uses SideSell stop (sells to close)
	order := types.NewTrailingStopOrder(
		"trail-1", "trader", "BTC-USDC",
		types.SideSell,
		math.LegacyNewDec(1),
		math.LegacyNewDec(100), // $100 trail
		math.LegacyZeroDec(),   // No activation price
	)

	// Initialize with starting price of 50000
	startPrice := math.LegacyNewDec(50000)
	order.HighWaterMark = startPrice
	order.CurrentStopPrice = startPrice.Sub(math.LegacyNewDec(100)) // 49900
	order.IsActivated = true

	// Price goes up to 51000 - stop should trail up
	triggered := order.Update(math.LegacyNewDec(51000))
	if triggered {
		t.Error("should not trigger when price goes up")
	}
	if !order.HighWaterMark.Equal(math.LegacyNewDec(51000)) {
		t.Errorf("expected high water mark 51000, got %s", order.HighWaterMark.String())
	}
	if !order.CurrentStopPrice.Equal(math.LegacyNewDec(50900)) {
		t.Errorf("expected stop price 50900, got %s", order.CurrentStopPrice.String())
	}

	// Price goes down to 50950 - stop should NOT update
	triggered = order.Update(math.LegacyNewDec(50950))
	if triggered {
		t.Error("should not trigger yet")
	}
	if !order.HighWaterMark.Equal(math.LegacyNewDec(51000)) {
		t.Error("high water mark should not change when price goes down")
	}

	// Price crashes to 50800 - below stop price, should trigger
	triggered = order.Update(math.LegacyNewDec(50800))
	if !triggered {
		t.Error("should trigger when price breaks stop")
	}
}

// TestTrailingStopOrder_Update_ShortStop tests updating a trailing stop for a short position
func TestTrailingStopOrder_Update_ShortStop(t *testing.T) {
	// Short position uses SideBuy stop (buys to close)
	order := types.NewTrailingStopOrder(
		"trail-2", "trader", "BTC-USDC",
		types.SideBuy,
		math.LegacyNewDec(1),
		math.LegacyNewDec(100), // $100 trail
		math.LegacyZeroDec(),
	)

	// Initialize with starting price of 50000
	startPrice := math.LegacyNewDec(50000)
	order.LowWaterMark = startPrice
	order.CurrentStopPrice = startPrice.Add(math.LegacyNewDec(100)) // 50100
	order.IsActivated = true

	// Price goes down to 49000 - stop should trail down
	triggered := order.Update(math.LegacyNewDec(49000))
	if triggered {
		t.Error("should not trigger when price goes down")
	}
	if !order.LowWaterMark.Equal(math.LegacyNewDec(49000)) {
		t.Errorf("expected low water mark 49000, got %s", order.LowWaterMark.String())
	}
	if !order.CurrentStopPrice.Equal(math.LegacyNewDec(49100)) {
		t.Errorf("expected stop price 49100, got %s", order.CurrentStopPrice.String())
	}

	// Price goes up to 49050 - stop should NOT update
	triggered = order.Update(math.LegacyNewDec(49050))
	if triggered {
		t.Error("should not trigger yet")
	}

	// Price spikes to 49200 - above stop price, should trigger
	triggered = order.Update(math.LegacyNewDec(49200))
	if !triggered {
		t.Error("should trigger when price breaks stop")
	}
}

// TestTrailingStopOrder_IsActive tests the IsActive method
func TestTrailingStopOrder_IsActive(t *testing.T) {
	order := types.NewTrailingStopOrder(
		"trail-1", "trader", "BTC-USDC",
		types.SideSell,
		math.LegacyNewDec(1),
		math.LegacyNewDec(100),
		math.LegacyZeroDec(),
	)

	if !order.IsActive() {
		t.Error("new order should be active")
	}

	order.Cancel()
	if order.IsActive() {
		t.Error("cancelled order should not be active")
	}
}

// TestTrailingStopOrder_Trigger tests the Trigger method
func TestTrailingStopOrder_Trigger(t *testing.T) {
	order := types.NewTrailingStopOrder(
		"trail-1", "trader", "BTC-USDC",
		types.SideSell,
		math.LegacyNewDec(1),
		math.LegacyNewDec(100),
		math.LegacyZeroDec(),
	)

	order.CurrentStopPrice = math.LegacyNewDec(49900)

	execOrder := order.Trigger()

	if order.Status != types.OrderStatusFilled {
		t.Errorf("expected status filled, got %s", order.Status.String())
	}
	if execOrder == nil {
		t.Error("expected execution order to be created")
	}
	if execOrder.Side != types.SideSell {
		t.Error("execution order should have same side")
	}
	if !execOrder.Quantity.Equal(math.LegacyNewDec(1)) {
		t.Error("execution order should have same quantity")
	}
}

// TestTrailingStopOrder_ActivationPrice tests activation price logic
func TestTrailingStopOrder_ActivationPrice(t *testing.T) {
	// Create order with activation price
	order := types.NewTrailingStopOrder(
		"trail-1", "trader", "BTC-USDC",
		types.SideSell,
		math.LegacyNewDec(1),
		math.LegacyNewDec(100),
		math.LegacyNewDec(51000), // Activate only when price reaches 51000
	)

	// Price at 50000 - below activation, should not activate
	order.Update(math.LegacyNewDec(50000))
	if order.IsActivated {
		t.Error("should not be activated below activation price")
	}

	// Price reaches 51000 - should activate
	order.Update(math.LegacyNewDec(51000))
	if !order.IsActivated {
		t.Error("should be activated at activation price")
	}
	if !order.HighWaterMark.Equal(math.LegacyNewDec(51000)) {
		t.Errorf("high water mark should be set to 51000, got %s", order.HighWaterMark.String())
	}
}
