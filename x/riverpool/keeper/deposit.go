package keeper

import (
	"context"
	"time"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/openalpha/perp-dex/x/riverpool/types"
)

// Deposit handles deposit into a pool
func (k *Keeper) Deposit(ctx context.Context, depositor, poolID string, amount math.LegacyDec, inviteCode string) (*types.Deposit, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// Get pool
	pool := k.GetPool(sdkCtx, poolID)
	if pool == nil {
		return nil, types.ErrPoolNotFound
	}

	// Validate pool status
	if pool.Status != types.PoolStatusActive {
		return nil, types.ErrPoolNotActive
	}

	// Validate deposit amount
	if amount.LT(pool.MinDeposit) {
		return nil, types.ErrDepositTooSmall
	}
	if !pool.MaxDeposit.IsZero() && amount.GT(pool.MaxDeposit) {
		return nil, types.ErrDepositTooLarge
	}

	// Foundation LP specific checks
	if pool.PoolType == types.PoolTypeFoundation {
		if !pool.HasAvailableSeats() {
			return nil, types.ErrFoundationPoolFull
		}
		// Foundation LP requires exact seat size
		if !amount.Equal(types.FoundationSeatSize) {
			return nil, types.ErrDepositTooSmall
		}
	}

	// Private pool check
	if pool.IsPrivate && pool.InviteCode != inviteCode {
		return nil, types.ErrInvalidInviteCode
	}

	// Calculate shares
	shares := pool.CalculateSharesForDeposit(amount)

	// Create deposit record
	deposit := types.NewDeposit(poolID, depositor, amount, shares, pool.NAV, pool.LockPeriodDays)

	// Foundation LP points
	if pool.PoolType == types.PoolTypeFoundation {
		deposit.PointsEarned = types.FoundationPointsPerSeat
	}

	// Update pool
	pool.TotalDeposits = pool.TotalDeposits.Add(amount)
	pool.TotalShares = pool.TotalShares.Add(shares)
	pool.UpdatedAt = time.Now().Unix()

	// Save to store
	k.SetDeposit(sdkCtx, deposit)
	k.SetPool(sdkCtx, pool)

	// Update pool stats
	stats := k.GetPoolStats(sdkCtx, poolID)
	stats.TotalValueLocked = pool.TotalDeposits
	stats.TotalDepositors++
	stats.UpdatedAt = time.Now().Unix()
	k.SetPoolStats(sdkCtx, stats)

	// Emit event
	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent(
			"riverpool_deposit",
			sdk.NewAttribute("pool_id", poolID),
			sdk.NewAttribute("depositor", depositor),
			sdk.NewAttribute("amount", amount.String()),
			sdk.NewAttribute("shares", shares.String()),
			sdk.NewAttribute("nav", pool.NAV.String()),
		),
	)

	k.logger.Info("Deposit processed",
		"pool_id", poolID,
		"depositor", depositor,
		"amount", amount.String(),
		"shares", shares.String(),
	)

	return deposit, nil
}

// GetUserTotalShares returns total shares for a user in a pool
func (k *Keeper) GetUserTotalShares(ctx sdk.Context, poolID, user string) math.LegacyDec {
	deposits := k.GetUserDeposits(ctx, user)
	total := math.LegacyZeroDec()
	for _, deposit := range deposits {
		if deposit.PoolID == poolID {
			total = total.Add(deposit.Shares)
		}
	}
	return total
}

// GetUserAvailableShares returns unlocked shares for a user in a pool
func (k *Keeper) GetUserAvailableShares(ctx sdk.Context, poolID, user string) math.LegacyDec {
	deposits := k.GetUserDeposits(ctx, user)
	available := math.LegacyZeroDec()
	for _, deposit := range deposits {
		if deposit.PoolID == poolID && !deposit.IsLocked() {
			available = available.Add(deposit.Shares)
		}
	}
	return available
}
