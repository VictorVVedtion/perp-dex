package types

import (
	"time"

	"cosmossdk.io/math"
)

// LiquidationTier represents the liquidation tier
type LiquidationTier int

const (
	// TierMarketOrder - Tier 1: Market order liquidation
	// Position is closed via market order on the order book
	TierMarketOrder LiquidationTier = iota + 1

	// TierPartialLiquidation - Tier 2: Partial liquidation for large positions
	// Only 20% of position is liquidated initially for positions > $100K
	TierPartialLiquidation

	// TierBackstopLiquidation - Tier 3: Backstop liquidation via Vault
	// When equity < 2/3 maintenance margin, Liquidator Vault takes over
	TierBackstopLiquidation
)

// String returns the string representation of LiquidationTier
func (t LiquidationTier) String() string {
	switch t {
	case TierMarketOrder:
		return "market_order"
	case TierPartialLiquidation:
		return "partial"
	case TierBackstopLiquidation:
		return "backstop"
	default:
		return "unknown"
	}
}

// LiquidationConfig holds the configuration for liquidation parameters
// Aligned with Hyperliquid's liquidation mechanism
type LiquidationConfig struct {
	// LargePositionThreshold - Positions above this value (in USDC) trigger partial liquidation
	// Default: 100,000 USDC (Hyperliquid standard)
	LargePositionThreshold math.LegacyDec

	// PartialLiquidationRate - Percentage of position to liquidate initially for large positions
	// Default: 20% (0.2)
	PartialLiquidationRate math.LegacyDec

	// CooldownPeriod - Time to wait after partial liquidation before next liquidation
	// Default: 30 seconds
	CooldownPeriod time.Duration

	// BackstopThreshold - Equity ratio below which backstop liquidation is triggered
	// Default: 2/3 of maintenance margin (0.6667)
	BackstopThreshold math.LegacyDec

	// LiquidationPenaltyRate - Penalty rate applied to liquidated positions
	// Default: 1%
	LiquidationPenaltyRate math.LegacyDec

	// LiquidatorRewardRate - Portion of penalty paid to liquidator
	// Default: 30%
	LiquidatorRewardRate math.LegacyDec

	// InsuranceFundRate - Portion of penalty going to insurance fund
	// Default: 70%
	InsuranceFundRate math.LegacyDec

	// MaxLiquidationsPerBlock - Maximum liquidations per block to prevent spam
	// Default: 100
	MaxLiquidationsPerBlock int

	// MinMaintenanceMarginRate - Minimum maintenance margin rate
	// Default: 2.5%
	MinMaintenanceMarginRate math.LegacyDec
}

// DefaultLiquidationConfig returns the default liquidation configuration
// aligned with Hyperliquid parameters
func DefaultLiquidationConfig() LiquidationConfig {
	return LiquidationConfig{
		LargePositionThreshold:   math.LegacyNewDec(100000),                     // $100,000
		PartialLiquidationRate:   math.LegacyNewDecWithPrec(20, 2),              // 20%
		CooldownPeriod:           30 * time.Second,                              // 30 seconds
		BackstopThreshold:        math.LegacyNewDecWithPrec(6667, 4),            // 2/3 = 66.67%
		LiquidationPenaltyRate:   math.LegacyNewDecWithPrec(1, 2),               // 1%
		LiquidatorRewardRate:     math.LegacyNewDecWithPrec(30, 2),              // 30%
		InsuranceFundRate:        math.LegacyNewDecWithPrec(70, 2),              // 70%
		MaxLiquidationsPerBlock:  100,
		MinMaintenanceMarginRate: math.LegacyNewDecWithPrec(25, 3),              // 2.5%
	}
}

// LiquidationState represents the current state of a position's liquidation
type LiquidationState struct {
	// PositionID is the unique identifier for the position
	PositionID string

	// Trader address
	Trader string

	// MarketID
	MarketID string

	// CurrentTier - Current liquidation tier being processed
	CurrentTier LiquidationTier

	// TotalLiquidated - Amount already liquidated
	TotalLiquidated math.LegacyDec

	// RemainingSize - Remaining position size to be liquidated
	RemainingSize math.LegacyDec

	// LastLiquidationTime - Time of last liquidation attempt
	LastLiquidationTime time.Time

	// LiquidationCount - Number of liquidation attempts
	LiquidationCount int

	// IsInCooldown - Whether the position is in cooldown period
	IsInCooldown bool

	// CooldownEndTime - When the cooldown period ends
	CooldownEndTime time.Time

	// TotalPenaltyPaid - Total penalty paid across all liquidations
	TotalPenaltyPaid math.LegacyDec

	// IsBackstopTriggered - Whether backstop liquidation was triggered
	IsBackstopTriggered bool
}

// NewLiquidationState creates a new liquidation state
func NewLiquidationState(positionID, trader, marketID string, positionSize math.LegacyDec) *LiquidationState {
	return &LiquidationState{
		PositionID:          positionID,
		Trader:              trader,
		MarketID:            marketID,
		CurrentTier:         TierMarketOrder,
		TotalLiquidated:     math.LegacyZeroDec(),
		RemainingSize:       positionSize,
		LastLiquidationTime: time.Time{},
		LiquidationCount:    0,
		IsInCooldown:        false,
		CooldownEndTime:     time.Time{},
		TotalPenaltyPaid:    math.LegacyZeroDec(),
		IsBackstopTriggered: false,
	}
}

// CanLiquidate checks if the position can be liquidated (not in cooldown)
func (s *LiquidationState) CanLiquidate(currentTime time.Time) bool {
	if !s.IsInCooldown {
		return true
	}
	return currentTime.After(s.CooldownEndTime)
}

// StartCooldown starts the cooldown period
func (s *LiquidationState) StartCooldown(cooldownDuration time.Duration) {
	s.IsInCooldown = true
	s.CooldownEndTime = time.Now().Add(cooldownDuration)
}

// EndCooldown ends the cooldown period
func (s *LiquidationState) EndCooldown() {
	s.IsInCooldown = false
	s.CooldownEndTime = time.Time{}
}

// UpdateAfterLiquidation updates the state after a liquidation
func (s *LiquidationState) UpdateAfterLiquidation(
	liquidatedSize, penalty math.LegacyDec,
	tier LiquidationTier,
) {
	s.TotalLiquidated = s.TotalLiquidated.Add(liquidatedSize)
	s.RemainingSize = s.RemainingSize.Sub(liquidatedSize)
	s.TotalPenaltyPaid = s.TotalPenaltyPaid.Add(penalty)
	s.LastLiquidationTime = time.Now()
	s.LiquidationCount++
	s.CurrentTier = tier
}

// IsFullyLiquidated checks if the position is fully liquidated
func (s *LiquidationState) IsFullyLiquidated() bool {
	return s.RemainingSize.LTE(math.LegacyZeroDec())
}

// HealthStatus represents the health status of a position
type HealthStatus int

const (
	// HealthStatusHealthy - Position is healthy, no liquidation needed
	HealthStatusHealthy HealthStatus = iota

	// HealthStatusAtRisk - Position is at risk, approaching liquidation
	HealthStatusAtRisk

	// HealthStatusLiquidatable - Position can be liquidated (below maintenance margin)
	HealthStatusLiquidatable

	// HealthStatusBackstop - Position needs backstop liquidation (< 2/3 maintenance)
	HealthStatusBackstop
)

// String returns the string representation of HealthStatus
func (s HealthStatus) String() string {
	switch s {
	case HealthStatusHealthy:
		return "healthy"
	case HealthStatusAtRisk:
		return "at_risk"
	case HealthStatusLiquidatable:
		return "liquidatable"
	case HealthStatusBackstop:
		return "backstop"
	default:
		return "unknown"
	}
}

// PositionHealthV2 contains detailed health information for a position
// Extended version with Hyperliquid-aligned fields
type PositionHealthV2 struct {
	Trader               string
	MarketID             string
	PositionSize         math.LegacyDec
	EntryPrice           math.LegacyDec
	MarkPrice            math.LegacyDec
	NotionalValue        math.LegacyDec
	Margin               math.LegacyDec
	UnrealizedPnL        math.LegacyDec
	Equity               math.LegacyDec
	MaintenanceMargin    math.LegacyDec
	InitialMargin        math.LegacyDec
	MarginRatio          math.LegacyDec // Equity / Notional Value
	HealthRatio          math.LegacyDec // Equity / Maintenance Margin
	LiquidationPrice     math.LegacyDec
	Status               HealthStatus
	RecommendedTier      LiquidationTier
	IsLargePosition      bool
}

// NewPositionHealthV2 creates a new position health assessment
func NewPositionHealthV2(
	trader, marketID string,
	positionSize, entryPrice, markPrice, margin math.LegacyDec,
	maintenanceMarginRate math.LegacyDec,
	largePositionThreshold math.LegacyDec,
) *PositionHealthV2 {
	notionalValue := positionSize.Mul(markPrice)
	unrealizedPnL := positionSize.Mul(markPrice.Sub(entryPrice))
	equity := margin.Add(unrealizedPnL)
	maintenanceMargin := notionalValue.Mul(maintenanceMarginRate)
	initialMargin := notionalValue.Mul(maintenanceMarginRate.Mul(math.LegacyNewDec(2))) // 2x maintenance

	// Calculate margin ratio (equity / notional)
	marginRatio := math.LegacyZeroDec()
	if notionalValue.IsPositive() {
		marginRatio = equity.Quo(notionalValue)
	}

	// Calculate health ratio (equity / maintenance margin)
	healthRatio := math.LegacyZeroDec()
	if maintenanceMargin.IsPositive() {
		healthRatio = equity.Quo(maintenanceMargin)
	}

	// Determine health status
	status := HealthStatusHealthy
	recommendedTier := TierMarketOrder
	backstopThreshold := math.LegacyNewDecWithPrec(6667, 4) // 2/3

	if healthRatio.LT(backstopThreshold) {
		status = HealthStatusBackstop
		recommendedTier = TierBackstopLiquidation
	} else if equity.LT(maintenanceMargin) {
		status = HealthStatusLiquidatable
		if notionalValue.GT(largePositionThreshold) {
			recommendedTier = TierPartialLiquidation
		}
	} else if equity.LT(initialMargin) {
		status = HealthStatusAtRisk
	}

	isLargePosition := notionalValue.GT(largePositionThreshold)

	return &PositionHealthV2{
		Trader:            trader,
		MarketID:          marketID,
		PositionSize:      positionSize,
		EntryPrice:        entryPrice,
		MarkPrice:         markPrice,
		NotionalValue:     notionalValue,
		Margin:            margin,
		UnrealizedPnL:     unrealizedPnL,
		Equity:            equity,
		MaintenanceMargin: maintenanceMargin,
		InitialMargin:     initialMargin,
		MarginRatio:       marginRatio,
		HealthRatio:       healthRatio,
		Status:            status,
		RecommendedTier:   recommendedTier,
		IsLargePosition:   isLargePosition,
	}
}

// NeedsLiquidation returns true if the position needs liquidation
func (h *PositionHealthV2) NeedsLiquidation() bool {
	return h.Status == HealthStatusLiquidatable || h.Status == HealthStatusBackstop
}

// NeedsBackstop returns true if backstop liquidation is needed
func (h *PositionHealthV2) NeedsBackstop() bool {
	return h.Status == HealthStatusBackstop
}
