package types

import (
	"cosmossdk.io/errors"
)

// Module error codes
var (
	ErrPositionHealthy       = errors.Register("clearinghouse", 1, "position is healthy, cannot liquidate")
	ErrPositionNotFound      = errors.Register("clearinghouse", 2, "position not found")
	ErrLiquidationFailed     = errors.Register("clearinghouse", 3, "liquidation failed")
	ErrLiquidationNotFound   = errors.Register("clearinghouse", 4, "liquidation not found")
	ErrInvalidLiquidator     = errors.Register("clearinghouse", 5, "invalid liquidator")
)
