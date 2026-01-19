package keeper

import (
	"encoding/binary"
	"encoding/json"
	"fmt"

	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/openalpha/perp-dex/x/orderbook/types"
)

// Store key prefixes for trailing stop orders
var (
	TrailingStopKeyPrefix        = []byte{0x20}
	TrailingStopByMarketPrefix   = []byte{0x21}
	TrailingStopCounterKeyPrefix = []byte{0x22}
)

// ============ Trailing Stop Order Storage ============

// generateTrailingStopID generates a unique trailing stop order ID
func (k *Keeper) generateTrailingStopID(ctx sdk.Context) string {
	store := k.GetStore(ctx)
	bz := store.Get(TrailingStopCounterKeyPrefix)
	var counter uint64
	if bz != nil {
		counter = binary.BigEndian.Uint64(bz)
	}
	counter++

	newBz := make([]byte, 8)
	binary.BigEndian.PutUint64(newBz, counter)
	store.Set(TrailingStopCounterKeyPrefix, newBz)

	return fmt.Sprintf("trail-%d", counter)
}

// SetTrailingStopOrder saves a trailing stop order
func (k *Keeper) SetTrailingStopOrder(ctx sdk.Context, order *types.TrailingStopOrder) {
	store := k.GetStore(ctx)

	// Primary key: prefix + orderID
	key := append(TrailingStopKeyPrefix, []byte(order.OrderID)...)
	bz, _ := json.Marshal(order)
	store.Set(key, bz)

	// Index by market: prefix + marketID + orderID
	marketKey := append(TrailingStopByMarketPrefix, []byte(order.MarketID+":"+order.OrderID)...)
	store.Set(marketKey, []byte(order.OrderID))
}

// GetTrailingStopOrder retrieves a trailing stop order by ID
func (k *Keeper) GetTrailingStopOrder(ctx sdk.Context, orderID string) *types.TrailingStopOrder {
	store := k.GetStore(ctx)
	key := append(TrailingStopKeyPrefix, []byte(orderID)...)
	bz := store.Get(key)
	if bz == nil {
		return nil
	}

	var order types.TrailingStopOrder
	if err := json.Unmarshal(bz, &order); err != nil {
		return nil
	}
	return &order
}

// DeleteTrailingStopOrder deletes a trailing stop order
func (k *Keeper) DeleteTrailingStopOrder(ctx sdk.Context, order *types.TrailingStopOrder) {
	store := k.GetStore(ctx)

	// Delete primary key
	key := append(TrailingStopKeyPrefix, []byte(order.OrderID)...)
	store.Delete(key)

	// Delete market index
	marketKey := append(TrailingStopByMarketPrefix, []byte(order.MarketID+":"+order.OrderID)...)
	store.Delete(marketKey)
}

// GetActiveTrailingStops returns all active trailing stop orders for a market
func (k *Keeper) GetActiveTrailingStops(ctx sdk.Context, marketID string) []*types.TrailingStopOrder {
	store := k.GetStore(ctx)
	prefix := append(TrailingStopByMarketPrefix, []byte(marketID+":")...)

	iterator := storetypes.KVStorePrefixIterator(store, prefix)
	defer iterator.Close()

	var orders []*types.TrailingStopOrder
	for ; iterator.Valid(); iterator.Next() {
		orderID := string(iterator.Value())
		order := k.GetTrailingStopOrder(ctx, orderID)
		if order != nil && order.IsActive() {
			orders = append(orders, order)
		}
	}

	return orders
}

// GetTrailingStopsByTrader returns trailing stop orders for a trader
func (k *Keeper) GetTrailingStopsByTrader(ctx sdk.Context, trader string) []*types.TrailingStopOrder {
	store := k.GetStore(ctx)

	iterator := storetypes.KVStorePrefixIterator(store, TrailingStopKeyPrefix)
	defer iterator.Close()

	var orders []*types.TrailingStopOrder
	for ; iterator.Valid(); iterator.Next() {
		var order types.TrailingStopOrder
		if err := json.Unmarshal(iterator.Value(), &order); err != nil {
			continue
		}
		if order.Trader == trader {
			orders = append(orders, &order)
		}
	}

	return orders
}

// ============ Trailing Stop Order Operations ============

// PlaceTrailingStop places a new trailing stop order
func (k *Keeper) PlaceTrailingStop(
	ctx sdk.Context,
	trader, marketID string,
	side types.Side,
	quantity, trailAmount, trailPercent, activationPrice math.LegacyDec,
) (*types.TrailingStopOrder, error) {
	// Validate inputs
	if quantity.IsNil() || !quantity.IsPositive() {
		return nil, types.ErrInvalidQuantity
	}

	if trailAmount.IsNil() && trailPercent.IsNil() {
		return nil, types.ErrInvalidQuantity
	}

	// Generate order ID
	orderID := k.generateTrailingStopID(ctx)

	var order *types.TrailingStopOrder
	if trailAmount.IsPositive() {
		order = types.NewTrailingStopOrder(
			orderID, trader, marketID,
			side,
			quantity, trailAmount, activationPrice,
		)
	} else {
		order = types.NewTrailingStopOrderPercent(
			orderID, trader, marketID,
			side,
			quantity, trailPercent, activationPrice,
		)
	}

	// Save order (price initialization will be handled by ProcessTrailingStopsForMarket)
	k.SetTrailingStopOrder(ctx, order)

	// Emit event
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"trailing_stop_placed",
			sdk.NewAttribute("order_id", orderID),
			sdk.NewAttribute("trader", trader),
			sdk.NewAttribute("market_id", marketID),
			sdk.NewAttribute("side", side.String()),
			sdk.NewAttribute("quantity", quantity.String()),
			sdk.NewAttribute("trail_amount", trailAmount.String()),
			sdk.NewAttribute("trail_percent", trailPercent.String()),
		),
	)

	k.Logger().Info("Trailing stop placed",
		"order_id", orderID,
		"trader", trader,
		"market", marketID,
	)

	return order, nil
}

// CancelTrailingStop cancels a trailing stop order
func (k *Keeper) CancelTrailingStop(ctx sdk.Context, trader, orderID string) error {
	order := k.GetTrailingStopOrder(ctx, orderID)
	if order == nil {
		return types.ErrOrderNotFound
	}

	if order.Trader != trader {
		return types.ErrUnauthorized
	}

	if !order.IsActive() {
		return types.ErrOrderNotActive
	}

	order.Cancel()
	k.SetTrailingStopOrder(ctx, order)

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"trailing_stop_cancelled",
			sdk.NewAttribute("order_id", orderID),
			sdk.NewAttribute("trader", trader),
		),
	)

	return nil
}

// UpdateTrailingStops updates all active trailing stops for a market based on current price
// This should be called on each price update
func (k *Keeper) UpdateTrailingStops(ctx sdk.Context, marketID string, markPrice math.LegacyDec) {
	orders := k.GetActiveTrailingStops(ctx, marketID)

	for _, order := range orders {
		// Initialize water marks if not set
		if order.HighWaterMark.IsZero() && order.Side == types.SideSell {
			order.HighWaterMark = markPrice
			trailDist := order.GetTrailDistance(markPrice)
			order.CurrentStopPrice = markPrice.Sub(trailDist)
		}
		if order.LowWaterMark.IsZero() && order.Side == types.SideBuy {
			order.LowWaterMark = markPrice
			trailDist := order.GetTrailDistance(markPrice)
			order.CurrentStopPrice = markPrice.Add(trailDist)
		}

		// Update the trailing stop
		triggered := order.Update(markPrice)

		if triggered {
			// Trigger the order
			k.TriggerTrailingStop(ctx, order)
		} else {
			// Save updated order
			k.SetTrailingStopOrder(ctx, order)
		}
	}
}

// TriggerTrailingStop triggers a trailing stop order
func (k *Keeper) TriggerTrailingStop(ctx sdk.Context, order *types.TrailingStopOrder) {
	// Create execution order
	execOrder := order.Trigger()

	// Update trailing stop status
	k.SetTrailingStopOrder(ctx, order)

	// Place the execution order
	// In production, this would go through the matching engine
	k.Logger().Info("Trailing stop triggered",
		"order_id", order.OrderID,
		"trader", order.Trader,
		"market", order.MarketID,
		"stop_price", order.CurrentStopPrice.String(),
	)

	// Emit event
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"trailing_stop_triggered",
			sdk.NewAttribute("order_id", order.OrderID),
			sdk.NewAttribute("trader", order.Trader),
			sdk.NewAttribute("market_id", order.MarketID),
			sdk.NewAttribute("stop_price", order.CurrentStopPrice.String()),
			sdk.NewAttribute("high_water_mark", order.HighWaterMark.String()),
			sdk.NewAttribute("low_water_mark", order.LowWaterMark.String()),
		),
	)

	// Submit execution order to matching engine
	// This would be handled by the order placement logic
	_ = execOrder // Placeholder for actual execution
}

// TrailingStopEndBlocker processes trailing stops at end of block
// Note: This requires integration with perpetual module during app wiring
func (k *Keeper) TrailingStopEndBlocker(ctx sdk.Context) {
	// TODO: Integrate with perpetual module to get active markets and prices
	// For now, this is a placeholder that will be called from the app module
	// with the proper market list and price info

	// Example usage when integrated:
	// markets := perpetualKeeper.ListActiveMarkets(ctx)
	// for _, market := range markets {
	//     priceInfo := perpetualKeeper.GetPrice(ctx, market.MarketID)
	//     k.UpdateTrailingStops(ctx, market.MarketID, priceInfo.MarkPrice)
	// }
}

// ProcessTrailingStopsForMarket processes trailing stops for a specific market
// This is the integration point called from perpetual module's end blocker
func (k *Keeper) ProcessTrailingStopsForMarket(ctx sdk.Context, marketID string, markPrice math.LegacyDec) {
	k.UpdateTrailingStops(ctx, marketID, markPrice)
}
