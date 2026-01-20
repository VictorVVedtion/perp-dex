package keeper

import (
	"context"
	"fmt"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/openalpha/perp-dex/x/orderbook/types"
)

var _ types.MsgServer = (*msgServer)(nil)

type msgServer struct {
	Keeper *Keeper
}

// NewMsgServerImpl returns an implementation of the MsgServer interface
func NewMsgServerImpl(keeper *Keeper) types.MsgServer {
	return &msgServer{Keeper: keeper}
}

// PlaceOrder handles the MsgPlaceOrder message
func (m *msgServer) PlaceOrder(ctx context.Context, msg *types.MsgPlaceOrder) (*types.MsgPlaceOrderResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// Validate message
	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}

	// Parse price and quantity
	price, err := math.LegacyNewDecFromStr(msg.Price)
	if err != nil {
		return nil, fmt.Errorf("invalid price: %w", err)
	}

	quantity, err := math.LegacyNewDecFromStr(msg.Quantity)
	if err != nil {
		return nil, fmt.Errorf("invalid quantity: %w", err)
	}

	// Use side and order type from proto message
	side := msg.Side
	orderType := msg.OrderType

	// Place order through keeper
	order, result, err := m.Keeper.PlaceOrder(sdkCtx, msg.Trader, msg.MarketId, side, orderType, price, quantity)
	if err != nil {
		return nil, err
	}

	// Emit event
	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent(
			"place_order",
			sdk.NewAttribute("order_id", order.OrderID),
			sdk.NewAttribute("trader", order.Trader),
			sdk.NewAttribute("market_id", order.MarketID),
			sdk.NewAttribute("side", side.String()),
			sdk.NewAttribute("order_type", orderType.String()),
			sdk.NewAttribute("price", price.String()),
			sdk.NewAttribute("quantity", quantity.String()),
		),
	)

	// Calculate filled quantity
	filledQty := math.LegacyZeroDec()
	if result != nil {
		filledQty = result.FilledQty
	}

	return &types.MsgPlaceOrderResponse{
		OrderId:   order.OrderID,
		FilledQty: filledQty.String(),
	}, nil
}

// CancelOrder handles the MsgCancelOrder message
func (m *msgServer) CancelOrder(ctx context.Context, msg *types.MsgCancelOrder) (*types.MsgCancelOrderResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// Validate message
	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}

	// Cancel order through keeper
	order, err := m.Keeper.CancelOrder(sdkCtx, msg.Trader, msg.OrderId)
	if err != nil {
		return nil, err
	}

	// Emit event
	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent(
			"cancel_order",
			sdk.NewAttribute("order_id", msg.OrderId),
			sdk.NewAttribute("trader", msg.Trader),
		),
	)

	// Calculate cancelled quantity
	cancelledQty := math.LegacyZeroDec()
	if order != nil {
		cancelledQty = order.RemainingQty()
	}

	return &types.MsgCancelOrderResponse{
		CancelledQty: cancelledQty.String(),
	}, nil
}
