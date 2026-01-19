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

// Note: This file uses k.GetStore(ctx) pattern consistent with keeper.go

// Store key prefixes for OCO orders
var (
	OCOKeyPrefix          = []byte{0x30}
	OCOByMarketPrefix     = []byte{0x31}
	OCOByOrderPrefix      = []byte{0x32} // Index to find OCO by component order ID
	OCOCounterKeyPrefix   = []byte{0x33}
)

// ============ OCO Order Storage ============

// generateOCOID generates a unique OCO order ID
func (k *Keeper) generateOCOID(ctx sdk.Context) string {
	store := k.GetStore(ctx)
	bz := store.Get(OCOCounterKeyPrefix)
	var counter uint64
	if bz != nil {
		counter = binary.BigEndian.Uint64(bz)
	}
	counter++

	newBz := make([]byte, 8)
	binary.BigEndian.PutUint64(newBz, counter)
	store.Set(OCOCounterKeyPrefix, newBz)

	return fmt.Sprintf("oco-%d", counter)
}

// SetOCO saves an OCO order
func (k *Keeper) SetOCO(ctx sdk.Context, oco *types.OCOOrder) {
	store := k.GetStore(ctx)

	// Primary key: prefix + ocoID
	key := append(OCOKeyPrefix, []byte(oco.OCOID)...)
	bz, _ := json.Marshal(oco)
	store.Set(key, bz)

	// Index by market: prefix + marketID + ocoID
	marketKey := append(OCOByMarketPrefix, []byte(oco.MarketID+":"+oco.OCOID)...)
	store.Set(marketKey, []byte(oco.OCOID))

	// Index by component order IDs
	if oco.StopOrder != nil {
		stopKey := append(OCOByOrderPrefix, []byte(oco.StopOrder.OrderID)...)
		store.Set(stopKey, []byte(oco.OCOID))
	}
	if oco.LimitOrder != nil {
		limitKey := append(OCOByOrderPrefix, []byte(oco.LimitOrder.OrderID)...)
		store.Set(limitKey, []byte(oco.OCOID))
	}
}

// GetOCO retrieves an OCO order by ID
func (k *Keeper) GetOCO(ctx sdk.Context, ocoID string) *types.OCOOrder {
	store := k.GetStore(ctx)
	key := append(OCOKeyPrefix, []byte(ocoID)...)
	bz := store.Get(key)
	if bz == nil {
		return nil
	}

	var oco types.OCOOrder
	if err := json.Unmarshal(bz, &oco); err != nil {
		return nil
	}
	return &oco
}

// GetOCOByOrderID retrieves an OCO by one of its component order IDs
func (k *Keeper) GetOCOByOrderID(ctx sdk.Context, orderID string) *types.OCOOrder {
	store := k.GetStore(ctx)
	key := append(OCOByOrderPrefix, []byte(orderID)...)
	bz := store.Get(key)
	if bz == nil {
		return nil
	}

	ocoID := string(bz)
	return k.GetOCO(ctx, ocoID)
}

// DeleteOCO deletes an OCO order
func (k *Keeper) DeleteOCO(ctx sdk.Context, oco *types.OCOOrder) {
	store := k.GetStore(ctx)

	// Delete primary key
	key := append(OCOKeyPrefix, []byte(oco.OCOID)...)
	store.Delete(key)

	// Delete market index
	marketKey := append(OCOByMarketPrefix, []byte(oco.MarketID+":"+oco.OCOID)...)
	store.Delete(marketKey)

	// Delete order ID indices
	if oco.StopOrder != nil {
		stopKey := append(OCOByOrderPrefix, []byte(oco.StopOrder.OrderID)...)
		store.Delete(stopKey)
	}
	if oco.LimitOrder != nil {
		limitKey := append(OCOByOrderPrefix, []byte(oco.LimitOrder.OrderID)...)
		store.Delete(limitKey)
	}
}

// GetActiveOCOs returns all active OCO orders for a market
func (k *Keeper) GetActiveOCOs(ctx sdk.Context, marketID string) []*types.OCOOrder {
	store := k.GetStore(ctx)
	prefix := append(OCOByMarketPrefix, []byte(marketID+":")...)

	iterator := storetypes.KVStorePrefixIterator(store, prefix)
	defer iterator.Close()

	var ocos []*types.OCOOrder
	for ; iterator.Valid(); iterator.Next() {
		ocoID := string(iterator.Value())
		oco := k.GetOCO(ctx, ocoID)
		if oco != nil && oco.IsActive() {
			ocos = append(ocos, oco)
		}
	}

	return ocos
}

// GetOCOsByTrader returns OCO orders for a trader
func (k *Keeper) GetOCOsByTrader(ctx sdk.Context, trader string) []*types.OCOOrder {
	store := k.GetStore(ctx)

	iterator := storetypes.KVStorePrefixIterator(store, OCOKeyPrefix)
	defer iterator.Close()

	var ocos []*types.OCOOrder
	for ; iterator.Valid(); iterator.Next() {
		var oco types.OCOOrder
		if err := json.Unmarshal(iterator.Value(), &oco); err != nil {
			continue
		}
		if oco.Trader == trader {
			ocos = append(ocos, &oco)
		}
	}

	return ocos
}

// ============ OCO Order Operations ============

// PlaceOCO places a new OCO order
func (k *Keeper) PlaceOCO(
	ctx sdk.Context,
	trader, marketID string,
	// Stop order parameters
	stopSide types.Side,
	stopTriggerPrice, stopQuantity math.LegacyDec,
	// Limit order parameters
	limitSide types.Side,
	limitPrice, limitQuantity math.LegacyDec,
) (*types.OCOOrder, error) {
	// Validate inputs
	if stopQuantity.IsNil() || !stopQuantity.IsPositive() {
		return nil, types.ErrInvalidQuantity
	}
	if limitQuantity.IsNil() || !limitQuantity.IsPositive() {
		return nil, types.ErrInvalidQuantity
	}
	if stopTriggerPrice.IsNil() || !stopTriggerPrice.IsPositive() {
		return nil, types.ErrInvalidPrice
	}
	if limitPrice.IsNil() || !limitPrice.IsPositive() {
		return nil, types.ErrInvalidPrice
	}

	// Generate IDs
	ocoID := k.generateOCOID(ctx)
	stopOrderID := ocoID + "-stop"
	limitOrderID := ocoID + "-limit"

	// Create stop order
	stopOrder := types.NewConditionalOrder(
		stopOrderID, trader, marketID,
		stopSide,
		types.OrderTypeStopLoss,
		stopTriggerPrice,
		math.LegacyZeroDec(), // Market execution
		stopQuantity,
		types.OrderFlags{ReduceOnly: true},
	)

	// Create limit order
	limitOrder := types.NewOrder(
		limitOrderID, trader, marketID,
		limitSide,
		types.OrderTypeLimit,
		limitPrice,
		limitQuantity,
	)

	// Create OCO
	oco := types.NewOCOOrder(ocoID, trader, marketID, stopOrder, limitOrder)

	// Save OCO
	k.SetOCO(ctx, oco)

	// Save component orders
	k.SetConditionalOrder(ctx, stopOrder)
	k.SetOrder(ctx, limitOrder)

	// Emit event
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"oco_placed",
			sdk.NewAttribute("oco_id", ocoID),
			sdk.NewAttribute("trader", trader),
			sdk.NewAttribute("market_id", marketID),
			sdk.NewAttribute("stop_order_id", stopOrderID),
			sdk.NewAttribute("limit_order_id", limitOrderID),
			sdk.NewAttribute("stop_trigger_price", stopTriggerPrice.String()),
			sdk.NewAttribute("limit_price", limitPrice.String()),
		),
	)

	k.Logger().Info("OCO order placed",
		"oco_id", ocoID,
		"trader", trader,
		"market", marketID,
	)

	return oco, nil
}

// CancelOCO cancels an OCO order
func (k *Keeper) CancelOCO(ctx sdk.Context, trader, ocoID string) error {
	oco := k.GetOCO(ctx, ocoID)
	if oco == nil {
		return types.ErrOrderNotFound
	}

	if oco.Trader != trader {
		return types.ErrUnauthorized
	}

	if !oco.IsActive() {
		return types.ErrOrderNotActive
	}

	// Cancel the OCO and both component orders
	oco.Cancel()
	k.SetOCO(ctx, oco)

	// Cancel component orders
	if oco.StopOrder != nil {
		k.SetConditionalOrder(ctx, oco.StopOrder)
	}
	if oco.LimitOrder != nil {
		k.SetOrder(ctx, oco.LimitOrder)
	}

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"oco_cancelled",
			sdk.NewAttribute("oco_id", ocoID),
			sdk.NewAttribute("trader", trader),
		),
	)

	return nil
}

// ProcessOCOTrigger processes when one of the OCO component orders is triggered
func (k *Keeper) ProcessOCOTrigger(ctx sdk.Context, triggeredOrderID string) {
	oco := k.GetOCOByOrderID(ctx, triggeredOrderID)
	if oco == nil || !oco.IsActive() {
		return
	}

	// Determine which order was triggered
	if oco.StopOrder != nil && triggeredOrderID == oco.StopOrder.OrderID {
		// Stop order triggered, cancel limit order
		execOrder := oco.TriggerStop()
		k.SetOCO(ctx, oco)

		// Cancel the limit order
		if oco.LimitOrder != nil {
			k.CancelOrder(ctx, oco.LimitOrder.Trader, oco.LimitOrder.OrderID)
		}

		k.Logger().Info("OCO stop triggered",
			"oco_id", oco.OCOID,
			"triggered_order", triggeredOrderID,
		)

		// Execute the stop order
		_ = execOrder // Placeholder for actual execution

	} else if oco.LimitOrder != nil && triggeredOrderID == oco.LimitOrder.OrderID {
		// Limit order triggered (filled), cancel stop order
		oco.TriggerLimit()
		k.SetOCO(ctx, oco)

		// Cancel the stop order
		if oco.StopOrder != nil {
			k.CancelConditionalOrder(ctx, oco.StopOrder.Trader, oco.StopOrder.OrderID)
		}

		k.Logger().Info("OCO limit triggered",
			"oco_id", oco.OCOID,
			"triggered_order", triggeredOrderID,
		)
	}

	// Emit event
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"oco_triggered",
			sdk.NewAttribute("oco_id", oco.OCOID),
			sdk.NewAttribute("triggered_order_id", triggeredOrderID),
		),
	)
}

// CheckOCOs checks all active OCOs for a market and processes triggers
func (k *Keeper) CheckOCOs(ctx sdk.Context, marketID string, markPrice math.LegacyDec) {
	ocos := k.GetActiveOCOs(ctx, marketID)

	for _, oco := range ocos {
		triggerType := oco.CheckTrigger(markPrice)

		switch triggerType {
		case "stop":
			k.ProcessOCOTrigger(ctx, oco.StopOrder.OrderID)
		case "limit":
			// Limit order fill is handled by matching engine
			// This path shouldn't normally be taken here
		}
	}
}

// OCOEndBlocker processes OCOs at end of block
// Note: This requires integration with perpetual module during app wiring
func (k *Keeper) OCOEndBlocker(ctx sdk.Context) {
	// TODO: Integrate with perpetual module to get active markets and prices
	// For now, this is a placeholder that will be called from the app module
	// with the proper market list and price info

	// Example usage when integrated:
	// markets := perpetualKeeper.ListActiveMarkets(ctx)
	// for _, market := range markets {
	//     priceInfo := perpetualKeeper.GetPrice(ctx, market.MarketID)
	//     k.CheckOCOs(ctx, market.MarketID, priceInfo.MarkPrice)
	// }
}

// ProcessOCOsForMarket processes OCOs for a specific market with given mark price
// This is the integration point called from perpetual module's end blocker
func (k *Keeper) ProcessOCOsForMarket(ctx sdk.Context, marketID string, markPrice math.LegacyDec) {
	k.CheckOCOs(ctx, marketID, markPrice)
}
