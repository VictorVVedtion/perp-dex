package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/openalpha/perp-dex/x/orderbook/types"
)

// ProcessTimeInForce processes an order according to its time in force setting
func (k *Keeper) ProcessTimeInForce(ctx sdk.Context, order *types.ExtendedOrder, result *MatchResult) error {
	switch order.TimeInForce {
	case types.TimeInForceIOC:
		return k.processIOC(ctx, order, result)
	case types.TimeInForceFOK:
		return k.processFOK(ctx, order, result)
	case types.TimeInForceGTX:
		return k.processGTX(ctx, order, result)
	default:
		// GTC - no special processing needed
		return nil
	}
}

// processIOC handles Immediate Or Cancel orders
// Any unfilled portion is cancelled immediately
func (k *Keeper) processIOC(ctx sdk.Context, order *types.ExtendedOrder, result *MatchResult) error {
	// If nothing was filled, return error
	if result == nil || result.FilledQty.IsZero() {
		order.Status = types.OrderStatusCancelled
		k.SetOrder(ctx, order.ToOrder())

		ctx.EventManager().EmitEvent(
			sdk.NewEvent(
				"ioc_cancelled",
				sdk.NewAttribute("order_id", order.OrderID),
				sdk.NewAttribute("reason", "no_fill"),
			),
		)

		return types.ErrIOCNoFill
	}

	// If partially filled, cancel the remainder
	if !order.FilledQty.GTE(order.Quantity) {
		order.Status = types.OrderStatusCancelled
		k.SetOrder(ctx, order.ToOrder())

		ctx.EventManager().EmitEvent(
			sdk.NewEvent(
				"ioc_partial_cancel",
				sdk.NewAttribute("order_id", order.OrderID),
				sdk.NewAttribute("filled_qty", order.FilledQty.String()),
				sdk.NewAttribute("cancelled_qty", order.Quantity.Sub(order.FilledQty).String()),
			),
		)
	}

	return nil
}

// processFOK handles Fill Or Kill orders
// The entire order must be filled or nothing
func (k *Keeper) processFOK(ctx sdk.Context, order *types.ExtendedOrder, result *MatchResult) error {
	// Check if order would be completely filled
	if result == nil || !result.FilledQty.GTE(order.Quantity) {
		// Cancel the order and reject any partial fills
		order.Status = types.OrderStatusCancelled
		k.SetOrder(ctx, order.ToOrder())

		ctx.EventManager().EmitEvent(
			sdk.NewEvent(
				"fok_rejected",
				sdk.NewAttribute("order_id", order.OrderID),
				sdk.NewAttribute("requested_qty", order.Quantity.String()),
				sdk.NewAttribute("available_qty", result.FilledQty.String()),
			),
		)

		return types.ErrFOKNotFilled
	}

	return nil
}

// processGTX handles Post Only (Good Till Crossing) orders
// These orders should only add liquidity, never take
func (k *Keeper) processGTX(ctx sdk.Context, order *types.ExtendedOrder, result *MatchResult) error {
	// If any trades occurred, the order would have taken liquidity
	if result != nil && len(result.Trades) > 0 {
		order.Status = types.OrderStatusCancelled
		k.SetOrder(ctx, order.ToOrder())

		ctx.EventManager().EmitEvent(
			sdk.NewEvent(
				"gtx_rejected",
				sdk.NewAttribute("order_id", order.OrderID),
				sdk.NewAttribute("reason", "would_take_liquidity"),
			),
		)

		return types.ErrPostOnlyWouldTake
	}

	return nil
}

// CheckPostOnly checks if an order with post-only flag would take liquidity
func (k *Keeper) CheckPostOnly(ctx sdk.Context, order *types.Order) bool {
	orderBook := k.GetOrderBook(ctx, order.MarketID)
	if orderBook == nil {
		return false // No order book, won't take liquidity
	}

	if order.Side == types.SideBuy {
		// Check if buy order would match with asks
		bestAsk := orderBook.BestAsk()
		if bestAsk != nil && order.Price.GTE(bestAsk.Price) {
			return true // Would take liquidity
		}
	} else {
		// Check if sell order would match with bids
		bestBid := orderBook.BestBid()
		if bestBid != nil && order.Price.LTE(bestBid.Price) {
			return true // Would take liquidity
		}
	}

	return false
}

// ValidateReduceOnly validates that a reduce-only order would actually reduce the position
func (k *Keeper) ValidateReduceOnly(ctx sdk.Context, trader, marketID string, side types.Side, quantity interface{}) error {
	// This would need to check with the perpetual keeper to get the current position
	// For now, return nil as this requires integration with perpetual module

	// In production:
	// 1. Get current position from perpetual keeper
	// 2. Check if the order side would reduce the position
	// 3. Check if the quantity is <= current position size

	return nil
}

// TimeInForceStats tracks statistics for time in force processing
type TimeInForceStats struct {
	GTCOrders      int
	IOCOrders      int
	IOCCancelled   int
	FOKOrders      int
	FOKRejected    int
	GTXOrders      int
	GTXRejected    int
}

// GetTimeInForceStats returns statistics for time in force processing
// This could be used for monitoring and analytics
func (k *Keeper) GetTimeInForceStats(ctx sdk.Context) *TimeInForceStats {
	// In production, this would read from accumulated counters
	return &TimeInForceStats{}
}
