package keeper

import (
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/openalpha/perp-dex/x/perpetual/types"
)

// MarginChecker handles all margin-related calculations and validations
type MarginChecker struct {
	keeper *Keeper
}

// NewMarginChecker creates a new margin checker
func NewMarginChecker(keeper *Keeper) *MarginChecker {
	return &MarginChecker{keeper: keeper}
}

// CalculateInitialMargin calculates the initial margin requirement
// InitialMargin = Size × Price × InitialMarginRate (5%)
// Updated from 10% to 5% to align with Hyperliquid
func (mc *MarginChecker) CalculateInitialMargin(size, price math.LegacyDec) math.LegacyDec {
	initialMarginRate := math.LegacyNewDecWithPrec(5, 2) // 5% (updated from 10%)
	return size.Mul(price).Mul(initialMarginRate)
}

// CalculateMaintenanceMargin calculates the maintenance margin requirement
// MaintenanceMargin = Size × MarkPrice × MaintenanceMarginRate (2.5%)
// Updated from 5% to 2.5% to align with Hyperliquid
func (mc *MarginChecker) CalculateMaintenanceMargin(size, markPrice math.LegacyDec) math.LegacyDec {
	maintenanceMarginRate := math.LegacyNewDecWithPrec(25, 3) // 2.5% (updated from 5%)
	return size.Mul(markPrice).Mul(maintenanceMarginRate)
}

// CalculateLiquidationPrice calculates the liquidation price for a position
// For Long: LiquidationPrice = EntryPrice × (1 - MaintenanceMarginRate)
// For Short: LiquidationPrice = EntryPrice × (1 + MaintenanceMarginRate)
// MaintenanceMarginRate: 2.5% (updated from 5%)
func (mc *MarginChecker) CalculateLiquidationPrice(entryPrice math.LegacyDec, side types.PositionSide) math.LegacyDec {
	maintenanceMarginRate := math.LegacyNewDecWithPrec(25, 3) // 2.5% (updated from 5%)
	if side == types.PositionSideLong {
		return entryPrice.Mul(math.LegacyOneDec().Sub(maintenanceMarginRate))
	}
	return entryPrice.Mul(math.LegacyOneDec().Add(maintenanceMarginRate))
}

// CalculateUnrealizedPnL calculates the unrealized PnL for a position
func (mc *MarginChecker) CalculateUnrealizedPnL(position *types.Position, markPrice math.LegacyDec) math.LegacyDec {
	return position.CalculateUnrealizedPnL(markPrice)
}

// CalculateAccountEquity calculates the total equity of an account
// AccountEquity = Balance + UnrealizedPnL (from all positions)
func (mc *MarginChecker) CalculateAccountEquity(ctx sdk.Context, trader string) math.LegacyDec {
	account := mc.keeper.GetAccount(ctx, trader)
	if account == nil {
		return math.LegacyZeroDec()
	}

	totalUnrealizedPnL := math.LegacyZeroDec()
	positions := mc.keeper.GetPositionsByTrader(ctx, trader)
	for _, position := range positions {
		priceInfo := mc.keeper.GetPrice(ctx, position.MarketID)
		if priceInfo != nil {
			unrealizedPnL := position.CalculateUnrealizedPnL(priceInfo.MarkPrice)
			totalUnrealizedPnL = totalUnrealizedPnL.Add(unrealizedPnL)
		}
	}

	return account.Balance.Add(totalUnrealizedPnL)
}

// CalculateMarginRatio calculates the margin ratio for a position
// MarginRatio = (Margin + UnrealizedPnL) / (Size × MarkPrice)
func (mc *MarginChecker) CalculateMarginRatio(position *types.Position, markPrice math.LegacyDec) math.LegacyDec {
	return position.CalculateMarginRatio(markPrice)
}

// CheckInitialMarginRequirement verifies if a trader has sufficient margin for a new order
func (mc *MarginChecker) CheckInitialMarginRequirement(ctx sdk.Context, trader, marketID string, size, price math.LegacyDec) error {
	account := mc.keeper.GetAccount(ctx, trader)
	if account == nil {
		return types.ErrAccountNotFound
	}

	requiredMargin := mc.CalculateInitialMargin(size, price)
	if !account.CanAfford(requiredMargin) {
		return types.ErrInsufficientMargin
	}

	return nil
}

// CheckMaintenanceMarginRequirement verifies if a position meets maintenance margin
func (mc *MarginChecker) CheckMaintenanceMarginRequirement(ctx sdk.Context, position *types.Position) (bool, math.LegacyDec) {
	priceInfo := mc.keeper.GetPrice(ctx, position.MarketID)
	if priceInfo == nil {
		return true, math.LegacyZeroDec() // No price, assume healthy
	}

	markPrice := priceInfo.MarkPrice
	maintenanceMargin := mc.CalculateMaintenanceMargin(position.Size, markPrice)
	unrealizedPnL := position.CalculateUnrealizedPnL(markPrice)
	equity := position.Margin.Add(unrealizedPnL)

	isHealthy := equity.GTE(maintenanceMargin)
	deficit := maintenanceMargin.Sub(equity)
	if deficit.IsNegative() {
		deficit = math.LegacyZeroDec()
	}

	return isHealthy, deficit
}

// GetPositionHealth returns detailed health information for a position
type PositionHealth struct {
	Trader             string
	MarketID           string
	MarginRatio        math.LegacyDec
	MaintenanceMargin  math.LegacyDec
	AccountEquity      math.LegacyDec
	UnrealizedPnL      math.LegacyDec
	LiquidationPrice   math.LegacyDec
	IsHealthy          bool
	AtRisk             bool // true if margin ratio < 150% of maintenance
}

// GetPositionHealth returns detailed health information
func (mc *MarginChecker) GetPositionHealth(ctx sdk.Context, position *types.Position) *PositionHealth {
	priceInfo := mc.keeper.GetPrice(ctx, position.MarketID)
	if priceInfo == nil {
		return nil
	}

	markPrice := priceInfo.MarkPrice
	unrealizedPnL := position.CalculateUnrealizedPnL(markPrice)
	equity := position.Margin.Add(unrealizedPnL)
	maintenanceMargin := mc.CalculateMaintenanceMargin(position.Size, markPrice)
	marginRatio := position.CalculateMarginRatio(markPrice)

	maintenanceRate := math.LegacyNewDecWithPrec(25, 3) // 2.5% (updated from 5%)
	isHealthy := marginRatio.GTE(maintenanceRate)
	atRiskThreshold := maintenanceRate.Mul(math.LegacyNewDecWithPrec(15, 1)) // 150% of maintenance = 3.75%
	atRisk := marginRatio.LT(atRiskThreshold)

	return &PositionHealth{
		Trader:            position.Trader,
		MarketID:          position.MarketID,
		MarginRatio:       marginRatio,
		MaintenanceMargin: maintenanceMargin,
		AccountEquity:     equity,
		UnrealizedPnL:     unrealizedPnL,
		LiquidationPrice:  position.LiquidationPrice,
		IsHealthy:         isHealthy,
		AtRisk:            atRisk,
	}
}

// GetUnhealthyPositions returns all positions below maintenance margin
func (mc *MarginChecker) GetUnhealthyPositions(ctx sdk.Context) []*PositionHealth {
	positions := mc.keeper.GetAllPositions(ctx)
	var unhealthy []*PositionHealth

	for _, position := range positions {
		health := mc.GetPositionHealth(ctx, position)
		if health != nil && !health.IsHealthy {
			unhealthy = append(unhealthy, health)
		}
	}

	return unhealthy
}

// GetAtRiskPositions returns all positions close to liquidation
func (mc *MarginChecker) GetAtRiskPositions(ctx sdk.Context) []*PositionHealth {
	positions := mc.keeper.GetAllPositions(ctx)
	var atRisk []*PositionHealth

	for _, position := range positions {
		health := mc.GetPositionHealth(ctx, position)
		if health != nil && health.AtRisk {
			atRisk = append(atRisk, health)
		}
	}

	return atRisk
}
