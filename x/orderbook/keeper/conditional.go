package keeper

import (
	"encoding/json"
	"fmt"

	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/openalpha/perp-dex/x/orderbook/types"
)

// Store key prefix for conditional orders
var ConditionalOrderKeyPrefix = []byte{0x06}

// ============ Conditional Order Storage ============

// SetConditionalOrder saves a conditional order to the store
func (k *Keeper) SetConditionalOrder(ctx sdk.Context, order *types.ConditionalOrder) {
	store := k.GetStore(ctx)
	key := append(ConditionalOrderKeyPrefix, []byte(order.OrderID)...)
	bz, _ := json.Marshal(order)
	store.Set(key, bz)
}

// GetConditionalOrder retrieves a conditional order from the store
func (k *Keeper) GetConditionalOrder(ctx sdk.Context, orderID string) *types.ConditionalOrder {
	store := k.GetStore(ctx)
	key := append(ConditionalOrderKeyPrefix, []byte(orderID)...)
	bz := store.Get(key)
	if bz == nil {
		return nil
	}
	var order types.ConditionalOrder
	if err := json.Unmarshal(bz, &order); err != nil {
		return nil
	}
	return &order
}

// DeleteConditionalOrder removes a conditional order from the store
func (k *Keeper) DeleteConditionalOrder(ctx sdk.Context, orderID string) {
	store := k.GetStore(ctx)
	key := append(ConditionalOrderKeyPrefix, []byte(orderID)...)
	store.Delete(key)
}

// GetActiveConditionalOrders returns all active conditional orders for a market
func (k *Keeper) GetActiveConditionalOrders(ctx sdk.Context, marketID string) []*types.ConditionalOrder {
	store := k.GetStore(ctx)
	iterator := storetypes.KVStorePrefixIterator(store, ConditionalOrderKeyPrefix)
	defer iterator.Close()

	var orders []*types.ConditionalOrder
	for ; iterator.Valid(); iterator.Next() {
		var order types.ConditionalOrder
		if err := json.Unmarshal(iterator.Value(), &order); err != nil {
			continue
		}
		if order.MarketID == marketID && order.IsActive() {
			orders = append(orders, &order)
		}
	}
	return orders
}

// GetConditionalOrdersByTrader returns all conditional orders for a trader
func (k *Keeper) GetConditionalOrdersByTrader(ctx sdk.Context, trader string) []*types.ConditionalOrder {
	store := k.GetStore(ctx)
	iterator := storetypes.KVStorePrefixIterator(store, ConditionalOrderKeyPrefix)
	defer iterator.Close()

	var orders []*types.ConditionalOrder
	for ; iterator.Valid(); iterator.Next() {
		var order types.ConditionalOrder
		if err := json.Unmarshal(iterator.Value(), &order); err != nil {
			continue
		}
		if order.Trader == trader {
			orders = append(orders, &order)
		}
	}
	return orders
}

// ============ Conditional Order Operations ============

// PlaceConditionalOrder creates a new conditional order
func (k *Keeper) PlaceConditionalOrder(ctx sdk.Context, order *types.ConditionalOrder) error {
	// Validate trigger price
	if order.TriggerPrice.IsNil() || order.TriggerPrice.IsZero() {
		return types.ErrInvalidTriggerPrice
	}

	// Validate order type
	if !order.OrderType.IsConditional() {
		return types.ErrInvalidOrderType
	}

	// Generate order ID if not provided
	if order.OrderID == "" {
		order.OrderID = k.generateOrderID(ctx)
	}

	// Save conditional order
	k.SetConditionalOrder(ctx, order)

	// Emit event
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"conditional_order_placed",
			sdk.NewAttribute("order_id", order.OrderID),
			sdk.NewAttribute("trader", order.Trader),
			sdk.NewAttribute("market_id", order.MarketID),
			sdk.NewAttribute("order_type", order.OrderType.ExtendedString()),
			sdk.NewAttribute("trigger_price", order.TriggerPrice.String()),
			sdk.NewAttribute("quantity", order.Quantity.String()),
		),
	)

	k.Logger().Info("conditional order placed",
		"order_id", order.OrderID,
		"trader", order.Trader,
		"market_id", order.MarketID,
		"order_type", order.OrderType.ExtendedString(),
		"trigger_price", order.TriggerPrice.String(),
	)

	return nil
}

// CancelConditionalOrder cancels a conditional order
func (k *Keeper) CancelConditionalOrder(ctx sdk.Context, trader, orderID string) error {
	order := k.GetConditionalOrder(ctx, orderID)
	if order == nil {
		return types.ErrConditionalOrderNotFound
	}

	// Verify ownership
	if order.Trader != trader {
		return types.ErrUnauthorized
	}

	// Check if already triggered
	if order.TriggeredAt != nil {
		return types.ErrConditionalOrderTriggered
	}

	// Check if already cancelled
	if order.Status == types.OrderStatusCancelled {
		return types.ErrConditionalOrderCancelled
	}

	// Cancel the order
	order.Cancel()
	k.SetConditionalOrder(ctx, order)

	// Emit event
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"conditional_order_cancelled",
			sdk.NewAttribute("order_id", orderID),
			sdk.NewAttribute("trader", trader),
		),
	)

	return nil
}

// CheckAndTriggerConditionalOrders checks all conditional orders for a market
// and triggers those that meet their conditions
func (k *Keeper) CheckAndTriggerConditionalOrders(ctx sdk.Context, marketID string, markPrice math.LegacyDec) []*types.Order {
	conditionalOrders := k.GetActiveConditionalOrders(ctx, marketID)
	triggeredOrders := make([]*types.Order, 0)

	for _, condOrder := range conditionalOrders {
		if condOrder.ShouldTrigger(markPrice) {
			// Trigger the order
			execOrder := condOrder.Trigger()

			// Update conditional order in store
			k.SetConditionalOrder(ctx, condOrder)

			// Add to triggered orders
			triggeredOrders = append(triggeredOrders, execOrder)

			// Emit event
			ctx.EventManager().EmitEvent(
				sdk.NewEvent(
					"conditional_order_triggered",
					sdk.NewAttribute("conditional_order_id", condOrder.OrderID),
					sdk.NewAttribute("execution_order_id", execOrder.OrderID),
					sdk.NewAttribute("trigger_price", condOrder.TriggerPrice.String()),
					sdk.NewAttribute("mark_price", markPrice.String()),
				),
			)

			k.Logger().Info("conditional order triggered",
				"conditional_order_id", condOrder.OrderID,
				"execution_order_id", execOrder.OrderID,
				"trigger_price", condOrder.TriggerPrice.String(),
				"mark_price", markPrice.String(),
			)
		}
	}

	return triggeredOrders
}

// ProcessTriggeredOrder processes an order that was triggered from a conditional order
func (k *Keeper) ProcessTriggeredOrder(ctx sdk.Context, order *types.Order) (*MatchResult, error) {
	// Save the order
	k.SetOrder(ctx, order)

	// Process through matching engine
	engine := NewMatchingEngine(k)
	result, err := engine.ProcessOrder(ctx, order)
	if err != nil {
		return nil, fmt.Errorf("failed to process triggered order: %w", err)
	}

	return result, nil
}

// ConditionalOrderEndBlocker checks and triggers conditional orders at end of block
func (k *Keeper) ConditionalOrderEndBlocker(ctx sdk.Context) error {
	// This would be called from the app's EndBlocker
	// Get all active markets and check their conditional orders

	// Note: In a real implementation, you would get the list of markets
	// from the perpetual keeper. For now, we'll process all conditional orders.

	store := k.GetStore(ctx)
	iterator := storetypes.KVStorePrefixIterator(store, ConditionalOrderKeyPrefix)
	defer iterator.Close()

	// Group orders by market
	ordersByMarket := make(map[string][]*types.ConditionalOrder)
	for ; iterator.Valid(); iterator.Next() {
		var order types.ConditionalOrder
		if err := json.Unmarshal(iterator.Value(), &order); err != nil {
			continue
		}
		if order.IsActive() {
			ordersByMarket[order.MarketID] = append(ordersByMarket[order.MarketID], &order)
		}
	}

	// Process each market's orders
	// Note: In production, we would get the mark price from the perpetual keeper
	for marketID, orders := range ordersByMarket {
		if len(orders) == 0 {
			continue
		}

		// Get mark price from perpetual keeper (would need interface)
		// For now, skip price-dependent logic here as it's handled elsewhere
		k.Logger().Debug("conditional orders pending",
			"market_id", marketID,
			"count", len(orders),
		)
	}

	return nil
}
