package keeper

import (
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/openalpha/perp-dex/x/perpetual/types"
)

// ============ Margin Mode Operations ============

// SetMarginMode sets the margin mode for a trader
// Note: Cannot change margin mode when there are open positions
func (k *Keeper) SetMarginMode(ctx sdk.Context, trader string, mode types.MarginMode) error {
	account := k.GetOrCreateAccount(ctx, trader)

	// Check if trader has open positions
	positions := k.GetPositionsByTrader(ctx, trader)
	if len(positions) > 0 {
		return types.ErrCannotChangeMarginModeWithPositions
	}

	account.MarginMode = mode
	account.UpdatedAt = ctx.BlockTime()
	k.SetAccount(ctx, account)

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"margin_mode_changed",
			sdk.NewAttribute("trader", trader),
			sdk.NewAttribute("mode", mode.String()),
		),
	)

	return nil
}

// GetMarginMode returns the margin mode for a trader
func (k *Keeper) GetMarginMode(ctx sdk.Context, trader string) types.MarginMode {
	account := k.GetAccount(ctx, trader)
	if account == nil {
		return types.MarginModeIsolated // Default
	}
	return account.MarginMode
}

// ============ Isolated Margin Calculations ============

// CalculateIsolatedMargin calculates margin info for an isolated position
func (k *Keeper) CalculateIsolatedMargin(ctx sdk.Context, position *types.Position) *types.MarginInfo {
	if position == nil {
		return nil
	}

	priceInfo := k.GetPrice(ctx, position.MarketID)
	if priceInfo == nil {
		return nil
	}

	market := k.GetMarket(ctx, position.MarketID)
	if market == nil {
		return nil
	}

	markPrice := priceInfo.MarkPrice
	notional := position.Size.Mul(markPrice)
	unrealizedPnL := position.CalculateUnrealizedPnL(markPrice)
	equity := position.Margin.Add(unrealizedPnL)
	maintenanceMargin := notional.Mul(market.MaintenanceMarginRate)

	var marginRatio math.LegacyDec
	if notional.IsPositive() {
		marginRatio = equity.Quo(notional)
	} else {
		marginRatio = math.LegacyNewDec(1)
	}

	return &types.MarginInfo{
		Equity:            equity,
		MaintenanceMargin: maintenanceMargin,
		MarginRatio:       marginRatio,
		IsHealthy:         marginRatio.GTE(market.MaintenanceMarginRate),
		AvailableMargin:   equity.Sub(maintenanceMargin),
	}
}

// ============ Cross Margin Calculations ============

// CalculateCrossMargin calculates margin info for cross margin mode
func (k *Keeper) CalculateCrossMargin(ctx sdk.Context, trader string) *types.CrossMarginInfo {
	account := k.GetAccount(ctx, trader)
	if account == nil {
		return nil
	}

	positions := k.GetPositionsByTrader(ctx, trader)

	info := &types.CrossMarginInfo{
		Equity:                 account.Balance,
		TotalNotional:          math.LegacyZeroDec(),
		TotalUnrealizedPnL:     math.LegacyZeroDec(),
		TotalMaintenanceMargin: math.LegacyZeroDec(),
	}

	for _, pos := range positions {
		priceInfo := k.GetPrice(ctx, pos.MarketID)
		market := k.GetMarket(ctx, pos.MarketID)

		if priceInfo == nil || market == nil {
			continue
		}

		markPrice := priceInfo.MarkPrice
		notional := pos.Size.Mul(markPrice)
		pnl := pos.CalculateUnrealizedPnL(markPrice)
		maintenance := notional.Mul(market.MaintenanceMarginRate)

		info.TotalNotional = info.TotalNotional.Add(notional)
		info.TotalUnrealizedPnL = info.TotalUnrealizedPnL.Add(pnl)
		info.TotalMaintenanceMargin = info.TotalMaintenanceMargin.Add(maintenance)
	}

	// Cross equity = balance + all unrealized PnL
	info.Equity = account.Balance.Add(info.TotalUnrealizedPnL)

	// Calculate margin ratio
	if info.TotalNotional.IsPositive() {
		info.MarginRatio = info.Equity.Quo(info.TotalNotional)
	} else {
		info.MarginRatio = math.LegacyNewDec(1)
	}

	// Check health (2.5% minimum margin ratio - updated from 5%)
	minMarginRatio := math.LegacyNewDecWithPrec(25, 3) // 2.5%
	info.IsHealthy = info.MarginRatio.GTE(minMarginRatio)

	// Available margin = equity - total maintenance margin
	info.AvailableMargin = info.Equity.Sub(info.TotalMaintenanceMargin)
	if info.AvailableMargin.IsNegative() {
		info.AvailableMargin = math.LegacyZeroDec()
	}

	return info
}

// ============ Margin Requirement Checks ============

// CheckMarginRequirement checks if a trader has sufficient margin for a new position
func (k *Keeper) CheckMarginRequirement(ctx sdk.Context, trader, marketID string, side types.PositionSide, quantity, price math.LegacyDec) error {
	account := k.GetAccount(ctx, trader)
	if account == nil {
		return types.ErrAccountNotFound
	}

	market := k.GetMarket(ctx, marketID)
	if market == nil {
		return types.ErrMarketNotFound
	}

	// Calculate required margin
	notional := quantity.Mul(price)
	requiredMargin := notional.Mul(market.InitialMarginRate)

	if account.MarginMode.IsCross() {
		// Cross margin mode - check total available margin
		crossInfo := k.CalculateCrossMargin(ctx, trader)
		if crossInfo == nil || crossInfo.AvailableMargin.LT(requiredMargin) {
			return types.ErrInsufficientMargin
		}
	} else {
		// Isolated margin mode - check available balance
		if account.AvailableBalance().LT(requiredMargin) {
			return types.ErrInsufficientBalance
		}
	}

	return nil
}

// GetEffectiveMargin returns the effective margin for a position based on mode
func (k *Keeper) GetEffectiveMargin(ctx sdk.Context, trader, marketID string) math.LegacyDec {
	account := k.GetAccount(ctx, trader)
	if account == nil {
		return math.LegacyZeroDec()
	}

	if account.MarginMode.IsCross() {
		crossInfo := k.CalculateCrossMargin(ctx, trader)
		if crossInfo != nil {
			return crossInfo.AvailableMargin
		}
	}

	// Isolated mode - return position-specific margin
	position := k.GetPosition(ctx, trader, marketID)
	if position != nil {
		return position.Margin
	}

	return account.AvailableBalance()
}

// ============ Cross Margin PnL Tracking ============

// UpdateCrossMarginPnL updates the cross margin PnL for an account
func (k *Keeper) UpdateCrossMarginPnL(ctx sdk.Context, trader string) error {
	account := k.GetAccount(ctx, trader)
	if account == nil || !account.MarginMode.IsCross() {
		return nil
	}

	positions := k.GetPositionsByTrader(ctx, trader)
	totalPnL := math.LegacyZeroDec()

	for _, pos := range positions {
		priceInfo := k.GetPrice(ctx, pos.MarketID)
		if priceInfo == nil {
			continue
		}
		pnl := pos.CalculateUnrealizedPnL(priceInfo.MarkPrice)
		totalPnL = totalPnL.Add(pnl)
	}

	account.CrossMarginPnL = totalPnL
	account.UpdatedAt = ctx.BlockTime()
	k.SetAccount(ctx, account)

	return nil
}

// ============ Liquidation Checks by Margin Mode ============

// CheckLiquidation checks if a position/account should be liquidated
func (k *Keeper) CheckLiquidation(ctx sdk.Context, trader, marketID string) (bool, *types.Position) {
	account := k.GetAccount(ctx, trader)
	if account == nil {
		return false, nil
	}

	if account.MarginMode.IsCross() {
		// Cross margin - check entire account
		return k.checkCrossMarginLiquidation(ctx, trader)
	}

	// Isolated margin - check specific position
	return k.checkIsolatedMarginLiquidation(ctx, trader, marketID)
}

// checkIsolatedMarginLiquidation checks isolated margin position for liquidation
func (k *Keeper) checkIsolatedMarginLiquidation(ctx sdk.Context, trader, marketID string) (bool, *types.Position) {
	position := k.GetPosition(ctx, trader, marketID)
	if position == nil {
		return false, nil
	}

	marginInfo := k.CalculateIsolatedMargin(ctx, position)
	if marginInfo == nil {
		return false, nil
	}

	return !marginInfo.IsHealthy, position
}

// checkCrossMarginLiquidation checks cross margin account for liquidation
func (k *Keeper) checkCrossMarginLiquidation(ctx sdk.Context, trader string) (bool, *types.Position) {
	crossInfo := k.CalculateCrossMargin(ctx, trader)
	if crossInfo == nil {
		return false, nil
	}

	if !crossInfo.IsHealthy {
		// Return the largest position for liquidation
		positions := k.GetPositionsByTrader(ctx, trader)
		var largestPosition *types.Position
		largestNotional := math.LegacyZeroDec()

		for _, pos := range positions {
			priceInfo := k.GetPrice(ctx, pos.MarketID)
			if priceInfo == nil {
				continue
			}
			notional := pos.Size.Mul(priceInfo.MarkPrice)
			if notional.GT(largestNotional) {
				largestNotional = notional
				largestPosition = pos
			}
		}

		return true, largestPosition
	}

	return false, nil
}

// GetMarginSummary returns a summary of margin status for a trader
type MarginSummary struct {
	Trader              string
	Mode                types.MarginMode
	TotalBalance        math.LegacyDec
	TotalLockedMargin   math.LegacyDec
	TotalUnrealizedPnL  math.LegacyDec
	TotalEquity         math.LegacyDec
	AvailableMargin     math.LegacyDec
	MarginRatio         math.LegacyDec
	IsHealthy           bool
	PositionCount       int
	IsolatedPositions   []*IsolatedPositionSummary
}

type IsolatedPositionSummary struct {
	MarketID      string
	Side          types.PositionSide
	Size          math.LegacyDec
	Margin        math.LegacyDec
	UnrealizedPnL math.LegacyDec
	MarginRatio   math.LegacyDec
	IsHealthy     bool
}

func (k *Keeper) GetMarginSummary(ctx sdk.Context, trader string) *MarginSummary {
	account := k.GetAccount(ctx, trader)
	if account == nil {
		return nil
	}

	positions := k.GetPositionsByTrader(ctx, trader)

	summary := &MarginSummary{
		Trader:            trader,
		Mode:              account.MarginMode,
		TotalBalance:      account.Balance,
		TotalLockedMargin: account.LockedMargin,
		PositionCount:     len(positions),
	}

	if account.MarginMode.IsCross() {
		crossInfo := k.CalculateCrossMargin(ctx, trader)
		if crossInfo != nil {
			summary.TotalUnrealizedPnL = crossInfo.TotalUnrealizedPnL
			summary.TotalEquity = crossInfo.Equity
			summary.AvailableMargin = crossInfo.AvailableMargin
			summary.MarginRatio = crossInfo.MarginRatio
			summary.IsHealthy = crossInfo.IsHealthy
		}
	} else {
		// Isolated mode - aggregate position info
		summary.IsolatedPositions = make([]*IsolatedPositionSummary, 0, len(positions))
		totalPnL := math.LegacyZeroDec()

		for _, pos := range positions {
			marginInfo := k.CalculateIsolatedMargin(ctx, pos)
			if marginInfo == nil {
				continue
			}

			priceInfo := k.GetPrice(ctx, pos.MarketID)
			pnl := math.LegacyZeroDec()
			if priceInfo != nil {
				pnl = pos.CalculateUnrealizedPnL(priceInfo.MarkPrice)
			}

			summary.IsolatedPositions = append(summary.IsolatedPositions, &IsolatedPositionSummary{
				MarketID:      pos.MarketID,
				Side:          pos.Side,
				Size:          pos.Size,
				Margin:        pos.Margin,
				UnrealizedPnL: pnl,
				MarginRatio:   marginInfo.MarginRatio,
				IsHealthy:     marginInfo.IsHealthy,
			})

			totalPnL = totalPnL.Add(pnl)
		}

		summary.TotalUnrealizedPnL = totalPnL
		summary.TotalEquity = account.Balance.Add(totalPnL)
		summary.AvailableMargin = account.AvailableBalance()
		summary.IsHealthy = true
		for _, pos := range summary.IsolatedPositions {
			if !pos.IsHealthy {
				summary.IsHealthy = false
				break
			}
		}
	}

	return summary
}
