package keeper

import (
	"context"
	"fmt"
	"sync"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/openalpha/perp-dex/x/orderbook/types"
)

// PlaceOrderBatch handles batch order placement for high throughput
// Processes up to 100 orders in a single transaction, significantly reducing overhead
func (m *msgServer) PlaceOrderBatch(ctx context.Context, msg *types.MsgPlaceOrderBatch) (*types.MsgPlaceOrderBatchResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// Validate the batch
	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}

	results := make([]*types.OrderResult, len(msg.Orders))
	var wg sync.WaitGroup
	var mu sync.Mutex

	// Parallel validation (CPU bound)
	for i, order := range msg.Orders {
		wg.Add(1)
		go func(idx int, orderItem *types.BatchOrderItem) {
			defer wg.Done()

			result := &types.OrderResult{
				Success: false,
			}

			// Validate price and quantity
			_, err := math.LegacyNewDecFromStr(orderItem.Price)
			if err != nil {
				result.Error = fmt.Sprintf("invalid price: %v", err)
				mu.Lock()
				results[idx] = result
				mu.Unlock()
				return
			}

			_, err = math.LegacyNewDecFromStr(orderItem.Quantity)
			if err != nil {
				result.Error = fmt.Sprintf("invalid quantity: %v", err)
				mu.Lock()
				results[idx] = result
				mu.Unlock()
				return
			}

			result.Success = true
			mu.Lock()
			results[idx] = result
			mu.Unlock()
		}(i, order)
	}

	wg.Wait()

	// Sequential state writes for correctness
	successCount := 0
	for i, order := range msg.Orders {
		if results[i].Success {
			// Parse values
			price, _ := math.LegacyNewDecFromStr(order.Price)
			quantity, _ := math.LegacyNewDecFromStr(order.Quantity)

			// Convert string side to types.Side
			var side types.Side
			switch order.Side {
			case "buy":
				side = types.SideBuy
			case "sell":
				side = types.SideSell
			default:
				results[i].Success = false
				results[i].Error = "invalid side"
				continue
			}

			// Convert string order type to types.OrderType
			var orderType types.OrderType
			switch order.OrderType {
			case "limit":
				orderType = types.OrderTypeLimit
			case "market":
				orderType = types.OrderTypeMarket
			default:
				results[i].Success = false
				results[i].Error = "invalid order type"
				continue
			}

			// Process the order through the matching engine
			placedOrder, matchResult, err := m.Keeper.PlaceOrder(
				sdkCtx,
				msg.Trader,
				order.MarketId,
				side,
				orderType,
				price,
				quantity,
			)

			if err != nil {
				results[i].Success = false
				results[i].Error = err.Error()
			} else {
				results[i].OrderId = placedOrder.OrderID
				successCount++
				if matchResult != nil {
					results[i].FilledQty = matchResult.FilledQty.String()
					results[i].AvgPrice = matchResult.AvgPrice.String()
				}
			}
		}
	}

	// Emit batch event
	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent(
			"place_order_batch",
			sdk.NewAttribute("trader", msg.Trader),
			sdk.NewAttribute("count", fmt.Sprintf("%d", len(msg.Orders))),
			sdk.NewAttribute("success_count", fmt.Sprintf("%d", successCount)),
		),
	)

	return &types.MsgPlaceOrderBatchResponse{
		Results: results,
	}, nil
}

// CancelOrderBatch handles batch order cancellation
func (m *msgServer) CancelOrderBatch(ctx context.Context, msg *types.MsgCancelOrderBatch) (*types.MsgCancelOrderBatchResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}

	results := make([]*types.CancelResult, len(msg.OrderIds))

	for i, orderID := range msg.OrderIds {
		result := &types.CancelResult{
			OrderId: orderID,
			Success: false,
		}

		// Try to cancel the order
		_, err := m.Keeper.CancelOrder(sdkCtx, msg.Trader, orderID)
		if err != nil {
			result.Error = err.Error()
		} else {
			result.Success = true
			result.Cancelled = true
		}

		results[i] = result
	}

	// Emit batch cancel event
	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent(
			"cancel_order_batch",
			sdk.NewAttribute("trader", msg.Trader),
			sdk.NewAttribute("count", fmt.Sprintf("%d", len(msg.OrderIds))),
		),
	)

	return &types.MsgCancelOrderBatchResponse{
		Results: results,
	}, nil
}
