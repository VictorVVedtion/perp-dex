package api

import (
	"github.com/openalpha/perp-dex/api/types"
)

// Re-export types for convenience
type (
	Order                = types.Order
	MatchResult          = types.MatchResult
	TradeInfo            = types.TradeInfo
	Position             = types.Position
	Account              = types.Account
	PlaceOrderRequest    = types.PlaceOrderRequest
	PlaceOrderResponse   = types.PlaceOrderResponse
	CancelOrderResponse  = types.CancelOrderResponse
	ModifyOrderRequest   = types.ModifyOrderRequest
	ModifyOrderResponse  = types.ModifyOrderResponse
	ListOrdersRequest    = types.ListOrdersRequest
	ListOrdersResponse   = types.ListOrdersResponse
	ClosePositionRequest = types.ClosePositionRequest
	ClosePositionResponse = types.ClosePositionResponse
	DepositRequest       = types.DepositRequest
	WithdrawRequest      = types.WithdrawRequest
	AccountResponse      = types.AccountResponse
	OrderService         = types.OrderService
	PositionService      = types.PositionService
	AccountService       = types.AccountService
)

// nowMillis returns current timestamp in milliseconds
func nowMillis() int64 {
	return types.NowMillis()
}
