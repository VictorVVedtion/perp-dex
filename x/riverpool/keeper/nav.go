package keeper

import (
	"time"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/openalpha/perp-dex/x/riverpool/types"
)

// CalculatePoolNAV calculates the NAV for a pool
// NAV = (Pool Cash + Position Market Value + Unrealized PnL) / Total Shares
func (k *Keeper) CalculatePoolNAV(ctx sdk.Context, poolID string) math.LegacyDec {
	pool := k.GetPool(ctx, poolID)
	if pool == nil || pool.TotalShares.IsZero() {
		return math.LegacyOneDec()
	}

	// For MVP, NAV is based on total deposits
	// In Phase 2+, this would include position market value and unrealized PnL
	totalValue := pool.TotalDeposits

	// TODO: Add position market value when community pools start trading
	// positionValue := k.calculatePoolPositionValue(ctx, poolID)
	// unrealizedPnL := k.calculatePoolUnrealizedPnL(ctx, poolID)
	// totalValue = pool.TotalDeposits.Add(positionValue).Add(unrealizedPnL)

	nav := totalValue.Quo(pool.TotalShares)
	return nav
}

// UpdatePoolNAV updates the NAV for a pool
func (k *Keeper) UpdatePoolNAV(ctx sdk.Context, poolID string) {
	pool := k.GetPool(ctx, poolID)
	if pool == nil {
		return
	}

	// Calculate new NAV
	totalValue := pool.TotalDeposits // MVP: just use deposits
	pool.UpdateNAV(totalValue)

	// Save updated pool
	k.SetPool(ctx, pool)

	// Record NAV history
	history := &types.NAVHistory{
		PoolID:     poolID,
		NAV:        pool.NAV,
		TotalValue: totalValue,
		Timestamp:  time.Now().Unix(),
	}
	k.AddNAVHistory(ctx, history)

	// Update DDGuard state
	k.updateDDGuardState(ctx, pool)

	k.logger.Debug("Pool NAV updated",
		"pool_id", poolID,
		"nav", pool.NAV.String(),
		"total_value", totalValue.String(),
		"drawdown", pool.CurrentDrawdown.String(),
		"dd_level", pool.DDGuardLevel,
	)
}

// UpdateAllPoolNAVs updates NAV for all pools (called in EndBlocker)
func (k *Keeper) UpdateAllPoolNAVs(ctx sdk.Context) {
	pools := k.GetAllPools(ctx)
	for _, pool := range pools {
		if pool.Status != types.PoolStatusClosed {
			k.UpdatePoolNAV(ctx, pool.PoolID)
		}
	}
}

// updateDDGuardState updates the DDGuard state for a pool
func (k *Keeper) updateDDGuardState(ctx sdk.Context, pool *types.Pool) {
	now := time.Now().Unix()

	state := k.GetDDGuardState(ctx, pool.PoolID)
	if state == nil {
		state = &types.DDGuardState{
			PoolID:           pool.PoolID,
			Level:            types.DDGuardLevelNormal,
			PeakNAV:          pool.HighWaterMark,
			CurrentNAV:       pool.NAV,
			DrawdownPercent:  math.LegacyZeroDec(),
			MaxExposureLimit: math.LegacyOneDec(), // 100% exposure by default
			TriggeredAt:      now,
			LastCheckedAt:    now,
		}
	}

	previousLevel := state.Level
	state.CurrentNAV = pool.NAV
	state.PeakNAV = pool.HighWaterMark
	state.DrawdownPercent = pool.CurrentDrawdown
	state.Level = pool.DDGuardLevel
	state.LastCheckedAt = now

	// Update max exposure limit based on DDGuard level
	switch state.Level {
	case types.DDGuardLevelNormal:
		state.MaxExposureLimit = math.LegacyOneDec() // 100%
	case types.DDGuardLevelWarning:
		state.MaxExposureLimit = math.LegacyMustNewDecFromStr("0.80") // 80%
	case types.DDGuardLevelReduce:
		state.MaxExposureLimit = math.LegacyMustNewDecFromStr("0.50") // 50%
	case types.DDGuardLevelHalt:
		state.MaxExposureLimit = math.LegacyZeroDec() // 0% - no new positions
	}

	// Record trigger time if level changed
	if state.Level != previousLevel {
		state.TriggeredAt = now

		// Emit event for level change
		ctx.EventManager().EmitEvent(
			sdk.NewEvent(
				"riverpool_ddguard_level_change",
				sdk.NewAttribute("pool_id", pool.PoolID),
				sdk.NewAttribute("previous_level", previousLevel),
				sdk.NewAttribute("new_level", state.Level),
				sdk.NewAttribute("drawdown_percent", state.DrawdownPercent.String()),
			),
		)

		k.logger.Warn("DDGuard level changed",
			"pool_id", pool.PoolID,
			"previous_level", previousLevel,
			"new_level", state.Level,
			"drawdown", state.DrawdownPercent.String(),
		)
	}

	k.SetDDGuardState(ctx, state)
}

// GetPoolValue returns the current total value of a pool
func (k *Keeper) GetPoolValue(ctx sdk.Context, poolID string) math.LegacyDec {
	pool := k.GetPool(ctx, poolID)
	if pool == nil {
		return math.LegacyZeroDec()
	}

	// MVP: total value = total deposits
	// Phase 2+: add position value and unrealized PnL
	return pool.TotalDeposits
}

// EstimateSharesForDeposit estimates shares for a deposit amount
func (k *Keeper) EstimateSharesForDeposit(ctx sdk.Context, poolID string, amount math.LegacyDec) (shares math.LegacyDec, nav math.LegacyDec) {
	pool := k.GetPool(ctx, poolID)
	if pool == nil {
		return math.LegacyZeroDec(), math.LegacyOneDec()
	}

	nav = pool.NAV
	shares = pool.CalculateSharesForDeposit(amount)
	return shares, nav
}

// EstimateAmountForWithdrawal estimates amount for a share withdrawal
func (k *Keeper) EstimateAmountForWithdrawal(ctx sdk.Context, poolID string, shares math.LegacyDec) (amount math.LegacyDec, nav math.LegacyDec) {
	pool := k.GetPool(ctx, poolID)
	if pool == nil {
		return math.LegacyZeroDec(), math.LegacyOneDec()
	}

	nav = pool.NAV
	amount = pool.CalculateValueForShares(shares)
	return amount, nav
}
