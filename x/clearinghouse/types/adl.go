package types

import (
	"time"

	"cosmossdk.io/math"
)

// ADLEvent represents an Auto-Deleveraging event
type ADLEvent struct {
	EventID           string
	MarketID          string
	TriggerReason     ADLTriggerReason
	InsuranceFundBalance math.LegacyDec
	TotalDeficit      math.LegacyDec
	PositionsAffected int
	TotalDeleveraged  math.LegacyDec
	Timestamp         time.Time
}

// ADLTriggerReason represents why ADL was triggered
type ADLTriggerReason int

const (
	ADLTriggerInsuranceDepleted ADLTriggerReason = iota
	ADLTriggerLargeDeficit
	ADLTriggerEmergency
)

func (r ADLTriggerReason) String() string {
	switch r {
	case ADLTriggerInsuranceDepleted:
		return "insurance_depleted"
	case ADLTriggerLargeDeficit:
		return "large_deficit"
	case ADLTriggerEmergency:
		return "emergency"
	default:
		return "unknown"
	}
}

// ADLPosition represents a position subject to ADL
type ADLPosition struct {
	Trader          string
	MarketID        string
	Side            string // "long" or "short"
	Size            math.LegacyDec
	EntryPrice      math.LegacyDec
	UnrealizedPnL   math.LegacyDec
	PnLPercent      math.LegacyDec // PnL as percentage of margin
	ADLRanking      int            // 1 = highest priority for ADL
	DeleverageQty   math.LegacyDec // Quantity to deleverage
}

// ADLQueue represents the queue of positions for ADL
type ADLQueue struct {
	MarketID    string
	Side        string // "long" or "short"
	Positions   []*ADLPosition
	TotalSize   math.LegacyDec
	LastUpdated time.Time
}

// NewADLQueue creates a new ADL queue
func NewADLQueue(marketID, side string) *ADLQueue {
	return &ADLQueue{
		MarketID:    marketID,
		Side:        side,
		Positions:   make([]*ADLPosition, 0),
		TotalSize:   math.LegacyZeroDec(),
		LastUpdated: time.Now(),
	}
}

// ADLConfig contains ADL configuration
type ADLConfig struct {
	Enabled             bool           // Whether ADL is enabled
	MaxDeleverageRatio  math.LegacyDec // Max percentage of position to deleverage at once
	MinPositionForADL   math.LegacyDec // Minimum position size for ADL
	PriorityByPnL       bool           // If true, prioritize by PnL; if false, by leverage
}

// DefaultADLConfig returns default ADL configuration
func DefaultADLConfig() ADLConfig {
	return ADLConfig{
		Enabled:            true,
		MaxDeleverageRatio: math.LegacyNewDecWithPrec(5, 1),  // 50%
		MinPositionForADL:  math.LegacyNewDecWithPrec(1, 2),  // 0.01
		PriorityByPnL:      true,
	}
}

// ADLResult represents the result of an ADL operation
type ADLResult struct {
	Success           bool
	EventID           string
	PositionsAffected int
	TotalDeleveraged  math.LegacyDec
	DeficitCovered    math.LegacyDec
	RemainingDeficit  math.LegacyDec
	Errors            []string
}
