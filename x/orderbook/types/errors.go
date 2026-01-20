package types

import (
	"cosmossdk.io/errors"
)

// Module error codes
var (
	ErrOrderNotFound         = errors.Register("orderbook", 1, "order not found")
	ErrInvalidPrice          = errors.Register("orderbook", 2, "invalid price")
	ErrInvalidQuantity       = errors.Register("orderbook", 3, "invalid quantity")
	ErrInvalidSide           = errors.Register("orderbook", 4, "invalid order side")
	ErrInvalidOrderType      = errors.Register("orderbook", 5, "invalid order type")
	ErrInvalidMarketID       = errors.Register("orderbook", 6, "invalid market ID")
	ErrInvalidTrader         = errors.Register("orderbook", 7, "invalid trader address")
	ErrOrderAlreadyFilled    = errors.Register("orderbook", 8, "order already filled")
	ErrOrderAlreadyCancelled = errors.Register("orderbook", 9, "order already cancelled")
	ErrUnauthorized          = errors.Register("orderbook", 10, "unauthorized")
	ErrInsufficientMargin    = errors.Register("orderbook", 11, "insufficient margin")

	// Conditional order errors
	ErrInvalidTriggerPrice       = errors.Register("orderbook", 20, "invalid trigger price")
	ErrConditionalOrderNotFound  = errors.Register("orderbook", 21, "conditional order not found")
	ErrConditionalOrderTriggered = errors.Register("orderbook", 22, "conditional order already triggered")
	ErrConditionalOrderCancelled = errors.Register("orderbook", 23, "conditional order already cancelled")

	// Time in force errors
	ErrFOKNotFilled      = errors.Register("orderbook", 30, "FOK order could not be fully filled")
	ErrPostOnlyWouldTake = errors.Register("orderbook", 31, "post-only order would take liquidity")
	ErrIOCNoFill         = errors.Register("orderbook", 32, "IOC order had no fills")

	// Order flag errors
	ErrReduceOnlyIncrease  = errors.Register("orderbook", 40, "reduce-only order would increase position")
	ErrOrderWouldExceedMax = errors.Register("orderbook", 41, "order would exceed maximum position size")

	// Order state errors
	ErrOrderNotActive = errors.Register("orderbook", 50, "order is not active")

	// Batch operation errors
	ErrInvalidOrder  = errors.Register("orderbook", 60, "invalid order")
	ErrBatchTooLarge = errors.Register("orderbook", 61, "batch size exceeds maximum (100)")
)
