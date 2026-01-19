package types

import (
	"time"

	"cosmossdk.io/math"
)

// InsuranceFund represents the insurance fund for a market
type InsuranceFund struct {
	FundID       string         // Unique fund identifier
	MarketID     string         // Associated market (empty for global fund)
	Balance      math.LegacyDec // Current balance in USDC
	TotalDeposits math.LegacyDec // Total deposits received
	TotalPayouts  math.LegacyDec // Total payouts made
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// NewInsuranceFund creates a new insurance fund
func NewInsuranceFund(fundID, marketID string) *InsuranceFund {
	now := time.Now()
	return &InsuranceFund{
		FundID:        fundID,
		MarketID:      marketID,
		Balance:       math.LegacyZeroDec(),
		TotalDeposits: math.LegacyZeroDec(),
		TotalPayouts:  math.LegacyZeroDec(),
		CreatedAt:     now,
		UpdatedAt:     now,
	}
}

// Deposit adds funds to the insurance fund
func (f *InsuranceFund) Deposit(amount math.LegacyDec) {
	f.Balance = f.Balance.Add(amount)
	f.TotalDeposits = f.TotalDeposits.Add(amount)
	f.UpdatedAt = time.Now()
}

// Withdraw removes funds from the insurance fund
func (f *InsuranceFund) Withdraw(amount math.LegacyDec) bool {
	if f.Balance.LT(amount) {
		return false
	}
	f.Balance = f.Balance.Sub(amount)
	f.TotalPayouts = f.TotalPayouts.Add(amount)
	f.UpdatedAt = time.Now()
	return true
}

// CanCover checks if the fund can cover a deficit
func (f *InsuranceFund) CanCover(amount math.LegacyDec) bool {
	return f.Balance.GTE(amount)
}

// InsuranceFundConfig contains insurance fund configuration
type InsuranceFundConfig struct {
	LiquidationPenaltyRate math.LegacyDec // Percentage of liquidation going to fund (e.g., 0.2 = 20%)
	TradingFeeRate         math.LegacyDec // Percentage of trading fees going to fund (e.g., 0.1 = 10%)
	ADLThreshold           math.LegacyDec // Fund balance threshold to trigger ADL (e.g., 0.1 = 10% of open interest)
	MinFundBalance         math.LegacyDec // Minimum fund balance before ADL
}

// DefaultInsuranceFundConfig returns default configuration
func DefaultInsuranceFundConfig() InsuranceFundConfig {
	return InsuranceFundConfig{
		LiquidationPenaltyRate: math.LegacyNewDecWithPrec(2, 1),  // 20%
		TradingFeeRate:         math.LegacyNewDecWithPrec(1, 1),  // 10%
		ADLThreshold:           math.LegacyNewDecWithPrec(1, 2),  // 1% of OI
		MinFundBalance:         math.LegacyNewDec(10000),          // 10,000 USDC
	}
}

// InsuranceEvent represents an insurance fund event
type InsuranceEvent struct {
	EventID      string
	FundID       string
	EventType    InsuranceEventType
	Amount       math.LegacyDec
	RelatedID    string // Liquidation ID, Trade ID, etc.
	Description  string
	Timestamp    time.Time
}

// InsuranceEventType represents the type of insurance event
type InsuranceEventType int

const (
	InsuranceEventDeposit InsuranceEventType = iota
	InsuranceEventLiquidationPenalty
	InsuranceEventTradingFee
	InsuranceEventDeficitCover
	InsuranceEventADLTrigger
)

func (t InsuranceEventType) String() string {
	switch t {
	case InsuranceEventDeposit:
		return "deposit"
	case InsuranceEventLiquidationPenalty:
		return "liquidation_penalty"
	case InsuranceEventTradingFee:
		return "trading_fee"
	case InsuranceEventDeficitCover:
		return "deficit_cover"
	case InsuranceEventADLTrigger:
		return "adl_trigger"
	default:
		return "unknown"
	}
}
