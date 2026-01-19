package keeper

import (
	"testing"

	"cosmossdk.io/math"
	"github.com/openalpha/perp-dex/x/orderbook/types"
)

// TestOCOOrder_NewOCOOrder tests creating a new OCO order
func TestOCOOrder_NewOCOOrder(t *testing.T) {
	ocoID := "oco-1"
	trader := "cosmos1abc..."
	marketID := "BTC-USDC"

	// Create stop loss order
	stopOrder := types.NewConditionalOrder(
		"oco-1-stop", trader, marketID,
		types.SideSell,
		types.OrderTypeStopLoss,
		math.LegacyNewDec(49000), // Stop trigger at 49000
		math.LegacyZeroDec(),     // Market execution
		math.LegacyNewDec(1),
		types.OrderFlags{ReduceOnly: true},
	)

	// Create take profit limit order
	limitOrder := types.NewOrder(
		"oco-1-limit", trader, marketID,
		types.SideSell,
		types.OrderTypeLimit,
		math.LegacyNewDec(52000), // Take profit at 52000
		math.LegacyNewDec(1),
	)

	oco := types.NewOCOOrder(ocoID, trader, marketID, stopOrder, limitOrder)

	if oco.OCOID != ocoID {
		t.Errorf("expected OCO ID %s, got %s", ocoID, oco.OCOID)
	}
	if oco.Trader != trader {
		t.Errorf("expected trader %s, got %s", trader, oco.Trader)
	}
	if oco.MarketID != marketID {
		t.Errorf("expected market ID %s, got %s", marketID, oco.MarketID)
	}
	if oco.StopOrder == nil {
		t.Error("stop order should not be nil")
	}
	if oco.LimitOrder == nil {
		t.Error("limit order should not be nil")
	}
	if oco.Status != types.OCOStatusPending {
		t.Errorf("expected status pending, got %v", oco.Status)
	}
}

// TestOCOOrder_IsActive tests the IsActive method
func TestOCOOrder_IsActive(t *testing.T) {
	stopOrder := types.NewConditionalOrder(
		"stop-1", "trader", "BTC-USDC",
		types.SideSell, types.OrderTypeStopLoss,
		math.LegacyNewDec(49000), math.LegacyZeroDec(),
		math.LegacyNewDec(1),
		types.OrderFlags{ReduceOnly: true},
	)

	limitOrder := types.NewOrder(
		"limit-1", "trader", "BTC-USDC",
		types.SideSell, types.OrderTypeLimit,
		math.LegacyNewDec(52000),
		math.LegacyNewDec(1),
	)

	oco := types.NewOCOOrder("oco-1", "trader", "BTC-USDC", stopOrder, limitOrder)

	if !oco.IsActive() {
		t.Error("new OCO should be active")
	}

	oco.Cancel()
	if oco.IsActive() {
		t.Error("cancelled OCO should not be active")
	}
}

// TestOCOOrder_Cancel tests the Cancel method
func TestOCOOrder_Cancel(t *testing.T) {
	stopOrder := types.NewConditionalOrder(
		"stop-1", "trader", "BTC-USDC",
		types.SideSell, types.OrderTypeStopLoss,
		math.LegacyNewDec(49000), math.LegacyZeroDec(),
		math.LegacyNewDec(1),
		types.OrderFlags{ReduceOnly: true},
	)

	limitOrder := types.NewOrder(
		"limit-1", "trader", "BTC-USDC",
		types.SideSell, types.OrderTypeLimit,
		math.LegacyNewDec(52000),
		math.LegacyNewDec(1),
	)

	oco := types.NewOCOOrder("oco-1", "trader", "BTC-USDC", stopOrder, limitOrder)
	oco.Cancel()

	if oco.Status != types.OCOStatusCancelled {
		t.Errorf("expected status cancelled, got %v", oco.Status)
	}
	if oco.StopOrder.Status != types.OrderStatusCancelled {
		t.Errorf("stop order should be cancelled")
	}
	if oco.LimitOrder.Status != types.OrderStatusCancelled {
		t.Errorf("limit order should be cancelled")
	}
}

// TestOCOOrder_TriggerStop tests triggering the stop order
func TestOCOOrder_TriggerStop(t *testing.T) {
	stopOrder := types.NewConditionalOrder(
		"stop-1", "trader", "BTC-USDC",
		types.SideSell, types.OrderTypeStopLoss,
		math.LegacyNewDec(49000), math.LegacyZeroDec(),
		math.LegacyNewDec(1),
		types.OrderFlags{ReduceOnly: true},
	)

	limitOrder := types.NewOrder(
		"limit-1", "trader", "BTC-USDC",
		types.SideSell, types.OrderTypeLimit,
		math.LegacyNewDec(52000),
		math.LegacyNewDec(1),
	)

	oco := types.NewOCOOrder("oco-1", "trader", "BTC-USDC", stopOrder, limitOrder)

	execOrder := oco.TriggerStop()

	if oco.Status != types.OCOStatusTriggered {
		t.Errorf("expected status triggered, got %v", oco.Status)
	}
	if oco.LimitOrder.Status != types.OrderStatusCancelled {
		t.Error("limit order should be cancelled when stop triggers")
	}
	if execOrder == nil {
		t.Error("execution order should be returned")
	}
}

// TestOCOOrder_TriggerLimit tests triggering the limit order
func TestOCOOrder_TriggerLimit(t *testing.T) {
	stopOrder := types.NewConditionalOrder(
		"stop-1", "trader", "BTC-USDC",
		types.SideSell, types.OrderTypeStopLoss,
		math.LegacyNewDec(49000), math.LegacyZeroDec(),
		math.LegacyNewDec(1),
		types.OrderFlags{ReduceOnly: true},
	)

	limitOrder := types.NewOrder(
		"limit-1", "trader", "BTC-USDC",
		types.SideSell, types.OrderTypeLimit,
		math.LegacyNewDec(52000),
		math.LegacyNewDec(1),
	)

	oco := types.NewOCOOrder("oco-1", "trader", "BTC-USDC", stopOrder, limitOrder)

	oco.TriggerLimit()

	if oco.Status != types.OCOStatusTriggered {
		t.Errorf("expected status triggered, got %v", oco.Status)
	}
	if oco.StopOrder.Status != types.OrderStatusCancelled {
		t.Error("stop order should be cancelled when limit triggers")
	}
}

// TestOCOOrder_CheckTrigger tests the trigger check logic
// Note: CheckTrigger only checks stop order triggers - limit orders are handled by the matching engine
func TestOCOOrder_CheckTrigger(t *testing.T) {
	tests := []struct {
		name          string
		stopTrigger   math.LegacyDec // Stop loss trigger price
		limitPrice    math.LegacyDec // Take profit price
		currentPrice  math.LegacyDec // Current mark price
		expectedType  string         // "stop" or "" (limit is handled by matching engine)
	}{
		{
			name:         "price above limit - no trigger (limit handled by matching engine)",
			stopTrigger:  math.LegacyNewDec(49000),
			limitPrice:   math.LegacyNewDec(52000),
			currentPrice: math.LegacyNewDec(52100),
			expectedType: "", // Limit orders filled through matching, not CheckTrigger
		},
		{
			name:         "price below stop - should trigger stop",
			stopTrigger:  math.LegacyNewDec(49000),
			limitPrice:   math.LegacyNewDec(52000),
			currentPrice: math.LegacyNewDec(48900),
			expectedType: "stop",
		},
		{
			name:         "price in middle - no trigger",
			stopTrigger:  math.LegacyNewDec(49000),
			limitPrice:   math.LegacyNewDec(52000),
			currentPrice: math.LegacyNewDec(50000),
			expectedType: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stopOrder := types.NewConditionalOrder(
				"stop-1", "trader", "BTC-USDC",
				types.SideSell, types.OrderTypeStopLoss,
				tt.stopTrigger, math.LegacyZeroDec(),
				math.LegacyNewDec(1),
				types.OrderFlags{ReduceOnly: true},
			)

			limitOrder := types.NewOrder(
				"limit-1", "trader", "BTC-USDC",
				types.SideSell, types.OrderTypeLimit,
				tt.limitPrice,
				math.LegacyNewDec(1),
			)

			oco := types.NewOCOOrder("oco-1", "trader", "BTC-USDC", stopOrder, limitOrder)
			result := oco.CheckTrigger(tt.currentPrice)

			if result != tt.expectedType {
				t.Errorf("expected trigger type %s, got %s", tt.expectedType, result)
			}
		})
	}
}

// TestOCOOrder_TypicalUseCase tests a typical stop-loss/take-profit scenario
func TestOCOOrder_TypicalUseCase(t *testing.T) {
	// Scenario: Long position at 50000
	// Stop loss at 49000 (2% loss)
	// Take profit at 52000 (4% gain)

	stopOrder := types.NewConditionalOrder(
		"stop-1", "trader", "BTC-USDC",
		types.SideSell, types.OrderTypeStopLoss,
		math.LegacyNewDec(49000),
		math.LegacyZeroDec(),
		math.LegacyNewDec(1),
		types.OrderFlags{ReduceOnly: true},
	)

	limitOrder := types.NewOrder(
		"limit-1", "trader", "BTC-USDC",
		types.SideSell, types.OrderTypeLimit,
		math.LegacyNewDec(52000),
		math.LegacyNewDec(1),
	)

	oco := types.NewOCOOrder("oco-1", "trader", "BTC-USDC", stopOrder, limitOrder)

	// Price hovers at 50500 - no trigger
	trigger := oco.CheckTrigger(math.LegacyNewDec(50500))
	if trigger != "" {
		t.Error("should not trigger at 50500")
	}

	// Price crashes to 48900 - stop loss triggers
	trigger = oco.CheckTrigger(math.LegacyNewDec(48900))
	if trigger != "stop" {
		t.Errorf("expected stop trigger, got %s", trigger)
	}

	// Execute the stop trigger
	oco.TriggerStop()

	// Verify final state
	if oco.IsActive() {
		t.Error("OCO should no longer be active after trigger")
	}
	if oco.Status != types.OCOStatusTriggered {
		t.Error("OCO status should be triggered")
	}
	// Limit order should be cancelled when stop triggers
	if oco.LimitOrder.Status != types.OrderStatusCancelled {
		t.Error("limit order should be cancelled")
	}
}

// TestOCOOrder_ShortPosition tests OCO for a short position
func TestOCOOrder_ShortPosition(t *testing.T) {
	// Scenario: Short position at 50000
	// Stop loss at 51000 (2% loss if price rises)
	// Take profit at 48000 (4% gain if price falls)

	stopOrder := types.NewConditionalOrder(
		"stop-1", "trader", "BTC-USDC",
		types.SideBuy, // Buy to close short
		types.OrderTypeStopLoss,
		math.LegacyNewDec(51000), // Trigger if price rises above
		math.LegacyZeroDec(),
		math.LegacyNewDec(1),
		types.OrderFlags{ReduceOnly: true},
	)

	limitOrder := types.NewOrder(
		"limit-1", "trader", "BTC-USDC",
		types.SideBuy, // Buy to close short
		types.OrderTypeLimit,
		math.LegacyNewDec(48000), // Take profit at lower price
		math.LegacyNewDec(1),
	)

	oco := types.NewOCOOrder("oco-1", "trader", "BTC-USDC", stopOrder, limitOrder)

	// Both orders should be on the buy side for closing a short
	if oco.StopOrder.Side != types.SideBuy {
		t.Error("stop order should be buy side for short")
	}
	if oco.LimitOrder.Side != types.SideBuy {
		t.Error("limit order should be buy side for short")
	}
}
