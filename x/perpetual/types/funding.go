package types

import (
	"time"

	"cosmossdk.io/math"
)

// FundingRate represents a funding rate snapshot
type FundingRate struct {
	MarketID   string         // Market identifier
	Rate       math.LegacyDec // Funding rate (can be positive or negative)
	MarkPrice  math.LegacyDec // Mark price at settlement
	IndexPrice math.LegacyDec // Index price at settlement
	Timestamp  time.Time      // Settlement timestamp
}

// NewFundingRate creates a new FundingRate
func NewFundingRate(marketID string, rate, markPrice, indexPrice math.LegacyDec) *FundingRate {
	return &FundingRate{
		MarketID:   marketID,
		Rate:       rate,
		MarkPrice:  markPrice,
		IndexPrice: indexPrice,
		Timestamp:  time.Now(),
	}
}

// FundingPayment represents a funding payment record
type FundingPayment struct {
	PaymentID string         // Unique payment identifier
	Trader    string         // Trader address
	MarketID  string         // Market identifier
	Amount    math.LegacyDec // Payment amount (positive = received, negative = paid)
	Rate      math.LegacyDec // Funding rate at settlement
	Timestamp time.Time      // Payment timestamp
}

// NewFundingPayment creates a new FundingPayment
func NewFundingPayment(paymentID, trader, marketID string, amount, rate math.LegacyDec) *FundingPayment {
	return &FundingPayment{
		PaymentID: paymentID,
		Trader:    trader,
		MarketID:  marketID,
		Amount:    amount,
		Rate:      rate,
		Timestamp: time.Now(),
	}
}

// FundingConfig contains funding rate configuration
// Updated parameters aligned with settlement schedule:
// - Interval: 8 hours
// - MaxRate: ±0.5%
type FundingConfig struct {
	Interval      int64          // Settlement interval in seconds (default: 28800 = 8 hours)
	MaxRate       math.LegacyDec // Maximum funding rate per interval
	MinRate       math.LegacyDec // Minimum funding rate per interval
	DampingFactor math.LegacyDec // Damping factor for rate calculation (default: 0.05)
}

// DefaultFundingConfig returns the default funding configuration
// Updated parameters aligned with settlement schedule:
// - Interval: 8 hours (28800 seconds)
// - MaxRate: 0.5% (0.005)
// - MinRate: -0.5% (-0.005)
// - DampingFactor: 0.05
func DefaultFundingConfig() FundingConfig {
	return FundingConfig{
		Interval:      28800,                            // 8 hours
		MaxRate:       math.LegacyNewDecWithPrec(5, 3),  // 0.005 = 0.5% (updated from 0.1%)
		MinRate:       math.LegacyNewDecWithPrec(-5, 3), // -0.005 = -0.5% (updated from -0.1%)
		DampingFactor: math.LegacyNewDecWithPrec(5, 2),  // 0.05 (updated from 0.03)
	}
}

// FundingInfo contains current funding information for a market
type FundingInfo struct {
	MarketID         string         // Market identifier
	CurrentRate      math.LegacyDec // Current calculated funding rate
	NextSettlement   time.Time      // Next settlement time
	LastSettlement   time.Time      // Last settlement time
	TotalLongSize    math.LegacyDec // Total long position size
	TotalShortSize   math.LegacyDec // Total short position size
	PredictedPayment math.LegacyDec // Predicted payment for 1 unit position
}

// MarginInfo contains margin information for a position
type MarginInfo struct {
	Equity            math.LegacyDec // Current equity (margin + unrealized PnL)
	MaintenanceMargin math.LegacyDec // Required maintenance margin
	MarginRatio       math.LegacyDec // Current margin ratio
	IsHealthy         bool           // Whether position is above maintenance margin
	AvailableMargin   math.LegacyDec // Available margin for new positions
}

// CrossMarginInfo contains cross margin information for an account
type CrossMarginInfo struct {
	Equity                 math.LegacyDec // Total account equity
	TotalNotional          math.LegacyDec // Total notional value of positions
	TotalUnrealizedPnL     math.LegacyDec // Total unrealized PnL
	TotalMaintenanceMargin math.LegacyDec // Total required maintenance margin
	MarginRatio            math.LegacyDec // Account margin ratio
	IsHealthy              bool           // Whether account is above maintenance margin
	AvailableMargin        math.LegacyDec // Available margin for new positions
}
