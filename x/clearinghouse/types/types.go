package types

import (
	"time"

	"cosmossdk.io/math"
)

// LiquidationStatus represents the status of a liquidation
type LiquidationStatus int

const (
	LiquidationStatusUnspecified LiquidationStatus = iota
	LiquidationStatusPending
	LiquidationStatusExecuted
	LiquidationStatusFailed
)

func (s LiquidationStatus) String() string {
	switch s {
	case LiquidationStatusPending:
		return "pending"
	case LiquidationStatusExecuted:
		return "executed"
	case LiquidationStatusFailed:
		return "failed"
	default:
		return "unspecified"
	}
}

// Liquidation represents a liquidation event
type Liquidation struct {
	LiquidationID    string
	Trader           string
	MarketID         string
	PositionSize     math.LegacyDec
	EntryPrice       math.LegacyDec
	MarkPrice        math.LegacyDec
	LiquidationPrice math.LegacyDec
	MarginDeficit    math.LegacyDec // how much below maintenance margin
	Penalty          math.LegacyDec // liquidation penalty
	Status           LiquidationStatus
	Timestamp        time.Time
}

// NewLiquidation creates a new liquidation record
func NewLiquidation(
	liquidationID string,
	trader string,
	marketID string,
	positionSize math.LegacyDec,
	entryPrice math.LegacyDec,
	markPrice math.LegacyDec,
	liquidationPrice math.LegacyDec,
	marginDeficit math.LegacyDec,
	penalty math.LegacyDec,
) *Liquidation {
	return &Liquidation{
		LiquidationID:    liquidationID,
		Trader:           trader,
		MarketID:         marketID,
		PositionSize:     positionSize,
		EntryPrice:       entryPrice,
		MarkPrice:        markPrice,
		LiquidationPrice: liquidationPrice,
		MarginDeficit:    marginDeficit,
		Penalty:          penalty,
		Status:           LiquidationStatusPending,
		Timestamp:        time.Now(),
	}
}

// PositionHealth represents the health status of a position
type PositionHealth struct {
	Trader            string
	MarketID          string
	MarginRatio       math.LegacyDec // current margin / required margin
	MaintenanceMargin math.LegacyDec // required maintenance margin
	AccountEquity     math.LegacyDec // current account equity
	IsHealthy         bool           // true if above maintenance margin
	AtRisk            bool           // true if close to liquidation
}
