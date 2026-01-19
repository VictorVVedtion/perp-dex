package types

import (
	"time"

	"cosmossdk.io/math"
)

// PositionSide represents position direction
type PositionSide int

const (
	PositionSideUnspecified PositionSide = iota
	PositionSideLong
	PositionSideShort
)

func (s PositionSide) String() string {
	switch s {
	case PositionSideLong:
		return "long"
	case PositionSideShort:
		return "short"
	default:
		return "unspecified"
	}
}

// Market defines a perpetual trading market
type Market struct {
	MarketID              string
	BaseAsset             string         // e.g., "BTC"
	QuoteAsset            string         // e.g., "USDC"
	MaxLeverage           math.LegacyDec // e.g., 10x
	InitialMarginRate     math.LegacyDec // e.g., 10% (0.1)
	MaintenanceMarginRate math.LegacyDec // e.g., 5% (0.05)
	TakerFeeRate          math.LegacyDec // e.g., 0.05% (0.0005)
	MakerFeeRate          math.LegacyDec // e.g., 0.02% (0.0002)
	TickSize              math.LegacyDec // minimum price increment
	LotSize               math.LegacyDec // minimum quantity increment
	IsActive              bool

	// Extended fields for production
	Status          MarketStatus   // Market status
	MinOrderSize    math.LegacyDec // Minimum order size
	MaxOrderSize    math.LegacyDec // Maximum order size
	MaxPositionSize math.LegacyDec // Maximum position size per trader
	FundingInterval int64          // Funding rate interval in seconds (default: 28800 = 8h)
	InsuranceFundID string         // Insurance fund identifier
	CreatedAt       time.Time      // Market creation time
	UpdatedAt       time.Time      // Last update time
}

// NewMarket creates a new market with default values for MVP
// Updated parameters aligned with Hyperliquid:
// - MaxLeverage: 50x (from 10x)
// - InitialMarginRate: 5% (from 10%)
// - MaintenanceMarginRate: 2.5% (from 5%)
// - FundingInterval: 1 hour (from 8 hours)
func NewMarket(marketID, baseAsset, quoteAsset string) *Market {
	now := time.Now()
	return &Market{
		MarketID:              marketID,
		BaseAsset:             baseAsset,
		QuoteAsset:            quoteAsset,
		MaxLeverage:           math.LegacyNewDec(50),            // 50x (updated)
		InitialMarginRate:     math.LegacyNewDecWithPrec(5, 2),  // 5% (updated from 10%)
		MaintenanceMarginRate: math.LegacyNewDecWithPrec(25, 3), // 2.5% (updated from 5%)
		TakerFeeRate:          math.LegacyNewDecWithPrec(5, 4),  // 0.05%
		MakerFeeRate:          math.LegacyNewDecWithPrec(2, 4),  // 0.02%
		TickSize:              math.LegacyNewDecWithPrec(1, 2),  // 0.01
		LotSize:               math.LegacyNewDecWithPrec(1, 4),  // 0.0001
		IsActive:              true,
		// Extended fields defaults
		Status:          MarketStatusActive,
		MinOrderSize:    math.LegacyNewDecWithPrec(1, 4), // 0.0001
		MaxOrderSize:    math.LegacyNewDec(1000),         // 1000
		MaxPositionSize: math.LegacyNewDec(10000),        // 10000
		FundingInterval: 3600,                            // 1 hour (updated from 8 hours)
		InsuranceFundID: "",
		CreatedAt:       now,
		UpdatedAt:       now,
	}
}

// NewMarketWithConfig creates a new market with custom configuration
func NewMarketWithConfig(config MarketConfig) *Market {
	now := time.Now()
	return &Market{
		MarketID:              config.MarketID,
		BaseAsset:             config.BaseAsset,
		QuoteAsset:            config.QuoteAsset,
		MaxLeverage:           config.MaxLeverage,
		InitialMarginRate:     config.InitialMarginRate,
		MaintenanceMarginRate: config.MaintenanceMarginRate,
		TakerFeeRate:          config.TakerFeeRate,
		MakerFeeRate:          config.MakerFeeRate,
		TickSize:              config.TickSize,
		LotSize:               config.LotSize,
		IsActive:              true,
		Status:                MarketStatusActive,
		MinOrderSize:          config.MinOrderSize,
		MaxOrderSize:          config.MaxOrderSize,
		MaxPositionSize:       config.MaxPositionSize,
		FundingInterval:       config.FundingInterval,
		InsuranceFundID:       config.InsuranceFundID,
		CreatedAt:             now,
		UpdatedAt:             now,
	}
}

// MarketConfig contains market configuration parameters
type MarketConfig struct {
	MarketID              string
	BaseAsset             string
	QuoteAsset            string
	MaxLeverage           math.LegacyDec
	InitialMarginRate     math.LegacyDec
	MaintenanceMarginRate math.LegacyDec
	TakerFeeRate          math.LegacyDec
	MakerFeeRate          math.LegacyDec
	TickSize              math.LegacyDec
	LotSize               math.LegacyDec
	MinOrderSize          math.LegacyDec
	MaxOrderSize          math.LegacyDec
	MaxPositionSize       math.LegacyDec
	FundingInterval       int64
	InsuranceFundID       string
}

// DefaultMarketConfigs returns default configurations for initial markets
// Updated parameters aligned with Hyperliquid:
// - MaxLeverage: 50x
// - InitialMarginRate: 5%
// - MaintenanceMarginRate: 2.5%
// - FundingInterval: 1 hour (3600 seconds)
func DefaultMarketConfigs() map[string]MarketConfig {
	return map[string]MarketConfig{
		"BTC-USDC": {
			MarketID:              "BTC-USDC",
			BaseAsset:             "BTC",
			QuoteAsset:            "USDC",
			MaxLeverage:           math.LegacyNewDec(50),            // 50x
			InitialMarginRate:     math.LegacyNewDecWithPrec(5, 2),  // 5%
			MaintenanceMarginRate: math.LegacyNewDecWithPrec(25, 3), // 2.5%
			TakerFeeRate:          math.LegacyNewDecWithPrec(5, 4),  // 0.05%
			MakerFeeRate:          math.LegacyNewDecWithPrec(2, 4),  // 0.02%
			TickSize:              math.LegacyNewDecWithPrec(1, 1),  // 0.1
			LotSize:               math.LegacyNewDecWithPrec(1, 4),  // 0.0001
			MinOrderSize:          math.LegacyNewDecWithPrec(1, 4),  // 0.0001
			MaxOrderSize:          math.LegacyNewDec(100),           // 100 BTC
			MaxPositionSize:       math.LegacyNewDec(1000),          // 1000 BTC
			FundingInterval:       3600,                             // 1 hour
		},
		"ETH-USDC": {
			MarketID:              "ETH-USDC",
			BaseAsset:             "ETH",
			QuoteAsset:            "USDC",
			MaxLeverage:           math.LegacyNewDec(50),            // 50x
			InitialMarginRate:     math.LegacyNewDecWithPrec(5, 2),  // 5%
			MaintenanceMarginRate: math.LegacyNewDecWithPrec(25, 3), // 2.5%
			TakerFeeRate:          math.LegacyNewDecWithPrec(5, 4),
			MakerFeeRate:          math.LegacyNewDecWithPrec(2, 4),
			TickSize:              math.LegacyNewDecWithPrec(1, 2), // 0.01
			LotSize:               math.LegacyNewDecWithPrec(1, 3), // 0.001
			MinOrderSize:          math.LegacyNewDecWithPrec(1, 3),
			MaxOrderSize:          math.LegacyNewDec(1000),
			MaxPositionSize:       math.LegacyNewDec(10000),
			FundingInterval:       3600, // 1 hour
		},
		"SOL-USDC": {
			MarketID:              "SOL-USDC",
			BaseAsset:             "SOL",
			QuoteAsset:            "USDC",
			MaxLeverage:           math.LegacyNewDec(50),            // 50x
			InitialMarginRate:     math.LegacyNewDecWithPrec(5, 2),  // 5%
			MaintenanceMarginRate: math.LegacyNewDecWithPrec(25, 3), // 2.5%
			TakerFeeRate:          math.LegacyNewDecWithPrec(5, 4),
			MakerFeeRate:          math.LegacyNewDecWithPrec(2, 4),
			TickSize:              math.LegacyNewDecWithPrec(1, 3), // 0.001
			LotSize:               math.LegacyNewDecWithPrec(1, 2), // 0.01
			MinOrderSize:          math.LegacyNewDecWithPrec(1, 2),
			MaxOrderSize:          math.LegacyNewDec(10000),
			MaxPositionSize:       math.LegacyNewDec(100000),
			FundingInterval:       3600, // 1 hour
		},
		"ARB-USDC": {
			MarketID:              "ARB-USDC",
			BaseAsset:             "ARB",
			QuoteAsset:            "USDC",
			MaxLeverage:           math.LegacyNewDec(50),            // 50x
			InitialMarginRate:     math.LegacyNewDecWithPrec(5, 2),  // 5%
			MaintenanceMarginRate: math.LegacyNewDecWithPrec(25, 3), // 2.5%
			TakerFeeRate:          math.LegacyNewDecWithPrec(5, 4),
			MakerFeeRate:          math.LegacyNewDecWithPrec(2, 4),
			TickSize:              math.LegacyNewDecWithPrec(1, 4), // 0.0001
			LotSize:               math.LegacyNewDecWithPrec(1, 1), // 0.1
			MinOrderSize:          math.LegacyNewDecWithPrec(1, 1),
			MaxOrderSize:          math.LegacyNewDec(100000),
			MaxPositionSize:       math.LegacyNewDec(1000000),
			FundingInterval:       3600, // 1 hour
		},
	}
}

// Position represents a trader's position in a market
type Position struct {
	Trader           string
	MarketID         string
	Side             PositionSide
	Size             math.LegacyDec // position size in base asset
	EntryPrice       math.LegacyDec // average entry price
	Margin           math.LegacyDec // deposited margin
	Leverage         math.LegacyDec // effective leverage (fixed 10x for MVP)
	LiquidationPrice math.LegacyDec
	OpenedAt         time.Time
	UpdatedAt        time.Time
}

// NewPosition creates a new position
func NewPosition(trader, marketID string, side PositionSide, size, entryPrice, margin math.LegacyDec) *Position {
	now := time.Now()
	p := &Position{
		Trader:     trader,
		MarketID:   marketID,
		Side:       side,
		Size:       size,
		EntryPrice: entryPrice,
		Margin:     margin,
		Leverage:   math.LegacyNewDec(50), // 50x max leverage (updated from 10x)
		OpenedAt:   now,
		UpdatedAt:  now,
	}
	p.LiquidationPrice = p.CalculateLiquidationPrice()
	return p
}

// CalculateLiquidationPrice calculates the liquidation price
// For Long: EntryPrice × (1 - MaintenanceMarginRate) = EntryPrice × 0.975
// For Short: EntryPrice × (1 + MaintenanceMarginRate) = EntryPrice × 1.025
// MaintenanceMarginRate: 2.5% (updated from 5%)
func (p *Position) CalculateLiquidationPrice() math.LegacyDec {
	maintenanceRate := math.LegacyNewDecWithPrec(25, 3) // 2.5% (updated from 5%)
	if p.Side == PositionSideLong {
		return p.EntryPrice.Mul(math.LegacyOneDec().Sub(maintenanceRate))
	}
	return p.EntryPrice.Mul(math.LegacyOneDec().Add(maintenanceRate))
}

// CalculateUnrealizedPnL calculates unrealized PnL at the given mark price
func (p *Position) CalculateUnrealizedPnL(markPrice math.LegacyDec) math.LegacyDec {
	priceDiff := markPrice.Sub(p.EntryPrice)
	if p.Side == PositionSideShort {
		priceDiff = priceDiff.Neg()
	}
	return p.Size.Mul(priceDiff)
}

// CalculateMarginRatio calculates the current margin ratio
// MarginRatio = (Margin + UnrealizedPnL) / NotionalValue
func (p *Position) CalculateMarginRatio(markPrice math.LegacyDec) math.LegacyDec {
	unrealizedPnL := p.CalculateUnrealizedPnL(markPrice)
	equity := p.Margin.Add(unrealizedPnL)
	notional := p.Size.Mul(markPrice)
	if notional.IsZero() {
		return math.LegacyZeroDec()
	}
	return equity.Quo(notional)
}

// IsHealthy checks if the position is above maintenance margin
// MaintenanceMarginRate: 2.5% (updated from 5%)
func (p *Position) IsHealthy(markPrice math.LegacyDec) bool {
	maintenanceRate := math.LegacyNewDecWithPrec(25, 3) // 2.5% (updated from 5%)
	return p.CalculateMarginRatio(markPrice).GTE(maintenanceRate)
}

// ShouldLiquidate checks if the position should be liquidated
func (p *Position) ShouldLiquidate(markPrice math.LegacyDec) bool {
	return !p.IsHealthy(markPrice)
}

// AddSize adds to the position (for averaging in)
func (p *Position) AddSize(size, price math.LegacyDec) {
	// Calculate new average entry price
	totalValue := p.Size.Mul(p.EntryPrice).Add(size.Mul(price))
	newSize := p.Size.Add(size)
	p.EntryPrice = totalValue.Quo(newSize)
	p.Size = newSize
	p.LiquidationPrice = p.CalculateLiquidationPrice()
	p.UpdatedAt = time.Now()
}

// ReduceSize reduces the position size
func (p *Position) ReduceSize(size math.LegacyDec) {
	p.Size = p.Size.Sub(size)
	p.UpdatedAt = time.Now()
}

// Account represents a trader's margin account
type Account struct {
	Trader       string
	Balance      math.LegacyDec // available balance (USDC)
	LockedMargin math.LegacyDec // margin locked in positions

	// Extended fields for production
	MarginMode     MarginMode     // Margin mode (isolated/cross)
	CrossMarginPnL math.LegacyDec // Unrealized PnL for cross margin positions
	CreatedAt      time.Time      // Account creation time
	UpdatedAt      time.Time      // Last update time
}

// NewAccount creates a new account
func NewAccount(trader string) *Account {
	now := time.Now()
	return &Account{
		Trader:         trader,
		Balance:        math.LegacyZeroDec(),
		LockedMargin:   math.LegacyZeroDec(),
		MarginMode:     MarginModeIsolated, // Default to isolated
		CrossMarginPnL: math.LegacyZeroDec(),
		CreatedAt:      now,
		UpdatedAt:      now,
	}
}

// TotalEquity returns total equity (balance + unrealized PnL from positions)
func (a *Account) TotalEquity(unrealizedPnL math.LegacyDec) math.LegacyDec {
	return a.Balance.Add(unrealizedPnL)
}

// AvailableBalance returns the available balance for new positions
func (a *Account) AvailableBalance() math.LegacyDec {
	return a.Balance.Sub(a.LockedMargin)
}

// CanAfford checks if the account can afford a given margin requirement
func (a *Account) CanAfford(amount math.LegacyDec) bool {
	return a.AvailableBalance().GTE(amount)
}

// LockMargin locks margin for a position
func (a *Account) LockMargin(amount math.LegacyDec) {
	a.LockedMargin = a.LockedMargin.Add(amount)
}

// UnlockMargin unlocks margin from a closed position
func (a *Account) UnlockMargin(amount math.LegacyDec) {
	a.LockedMargin = a.LockedMargin.Sub(amount)
	if a.LockedMargin.IsNegative() {
		a.LockedMargin = math.LegacyZeroDec()
	}
}

// Deposit adds funds to the account
func (a *Account) Deposit(amount math.LegacyDec) {
	a.Balance = a.Balance.Add(amount)
}

// Withdraw removes funds from the account
func (a *Account) Withdraw(amount math.LegacyDec) error {
	if a.AvailableBalance().LT(amount) {
		return ErrInsufficientBalance
	}
	a.Balance = a.Balance.Sub(amount)
	return nil
}

// PriceInfo represents current price information
type PriceInfo struct {
	MarketID   string
	MarkPrice  math.LegacyDec // mark price for PnL calculation
	IndexPrice math.LegacyDec // external reference price
	LastPrice  math.LegacyDec // last traded price
	Timestamp  time.Time
}

// NewPriceInfo creates new price info
func NewPriceInfo(marketID string, price math.LegacyDec) *PriceInfo {
	return &PriceInfo{
		MarketID:   marketID,
		MarkPrice:  price,
		IndexPrice: price,
		LastPrice:  price,
		Timestamp:  time.Now(),
	}
}
