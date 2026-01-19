package types

import (
	"cosmossdk.io/errors"
)

// Module error codes
var (
	ErrInsufficientBalance     = errors.Register("perpetual", 1, "insufficient balance")
	ErrInsufficientMargin      = errors.Register("perpetual", 2, "insufficient margin")
	ErrPositionNotFound        = errors.Register("perpetual", 3, "position not found")
	ErrMarketNotFound          = errors.Register("perpetual", 4, "market not found")
	ErrMarketNotActive         = errors.Register("perpetual", 5, "market not active")
	ErrAccountNotFound         = errors.Register("perpetual", 6, "account not found")
	ErrInvalidQuantity         = errors.Register("perpetual", 7, "invalid quantity")
	ErrInvalidPrice            = errors.Register("perpetual", 8, "invalid price")
	ErrInvalidLeverage         = errors.Register("perpetual", 9, "invalid leverage")
	ErrPositionAlreadyExists   = errors.Register("perpetual", 10, "position already exists")
	ErrCannotReducePosition    = errors.Register("perpetual", 11, "cannot reduce position by more than current size")
	ErrUnauthorized            = errors.Register("perpetual", 12, "unauthorized")
	ErrWithdrawExceedsBalance  = errors.Register("perpetual", 13, "withdraw amount exceeds available balance")

	// Market errors
	ErrMarketExists                       = errors.Register("perpetual", 14, "market already exists")
	ErrMarketPaused                       = errors.Register("perpetual", 15, "market is paused")
	ErrInvalidMarketID                    = errors.Register("perpetual", 16, "invalid market ID")
	ErrInvalidBaseAsset                   = errors.Register("perpetual", 17, "invalid base asset")
	ErrInvalidQuoteAsset                  = errors.Register("perpetual", 18, "invalid quote asset")

	// Funding rate errors
	ErrFundingNotDue                      = errors.Register("perpetual", 20, "funding settlement not due")
	ErrFundingAlreadySettled              = errors.Register("perpetual", 21, "funding already settled for this period")
	ErrInvalidFundingConfig               = errors.Register("perpetual", 22, "invalid funding configuration")

	// Margin mode errors
	ErrCannotChangeMarginModeWithPositions = errors.Register("perpetual", 30, "cannot change margin mode with open positions")
	ErrInvalidMarginMode                   = errors.Register("perpetual", 31, "invalid margin mode")
	ErrCrossMarginLiquidation              = errors.Register("perpetual", 32, "cross margin account liquidation triggered")

	// Order validation errors
	ErrOrderSizeTooSmall                  = errors.Register("perpetual", 40, "order size below minimum")
	ErrOrderSizeTooLarge                  = errors.Register("perpetual", 41, "order size above maximum")
	ErrPositionSizeTooLarge               = errors.Register("perpetual", 42, "position size would exceed maximum")
)
