package keeper

import (
	"context"
	"sort"
	"strconv"
	"time"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/openalpha/perp-dex/x/riverpool/types"
)

// RequestWithdrawal initiates a withdrawal request
func (k *Keeper) RequestWithdrawal(ctx context.Context, withdrawer, poolID string, shares math.LegacyDec) (*types.Withdrawal, error) {
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

	// Check user's available shares
	availableShares := k.GetUserAvailableShares(sdkCtx, poolID, withdrawer)
	if shares.GT(availableShares) {
		return nil, types.ErrInsufficientShares
	}

	// Create withdrawal request
	withdrawal := types.NewWithdrawal(poolID, withdrawer, shares, pool.NAV, pool.RedemptionDelayDays)

	// Calculate estimated amount
	estimatedAmount := pool.CalculateValueForShares(shares)

	// Save withdrawal
	k.SetWithdrawal(sdkCtx, withdrawal)

	// Update pool stats
	stats := k.GetPoolStats(sdkCtx, poolID)
	stats.TotalPendingWithdrawals = stats.TotalPendingWithdrawals.Add(estimatedAmount)
	stats.UpdatedAt = time.Now().Unix()
	k.SetPoolStats(sdkCtx, stats)

	// Emit event
	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent(
			"riverpool_withdrawal_request",
			sdk.NewAttribute("pool_id", poolID),
			sdk.NewAttribute("withdrawer", withdrawer),
			sdk.NewAttribute("shares", shares.String()),
			sdk.NewAttribute("estimated_amount", estimatedAmount.String()),
			sdk.NewAttribute("available_at", strconv.FormatInt(withdrawal.AvailableAt, 10)),
		),
	)

	k.logger.Info("Withdrawal requested",
		"pool_id", poolID,
		"withdrawer", withdrawer,
		"shares", shares.String(),
		"estimated_amount", estimatedAmount.String(),
	)

	return withdrawal, nil
}

// ClaimWithdrawal processes a withdrawal claim
func (k *Keeper) ClaimWithdrawal(ctx context.Context, withdrawer, withdrawalID string) (*types.Withdrawal, math.LegacyDec, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// Get withdrawal
	withdrawal := k.GetWithdrawal(sdkCtx, withdrawalID)
	if withdrawal == nil {
		return nil, math.LegacyZeroDec(), types.ErrWithdrawalNotFound
	}

	// Verify owner
	if withdrawal.Withdrawer != withdrawer {
		return nil, math.LegacyZeroDec(), types.ErrUnauthorized
	}

	// Check if ready
	if !withdrawal.IsReady() {
		return nil, math.LegacyZeroDec(), types.ErrWithdrawalNotReady
	}

	// Check if already claimed
	if withdrawal.Status == types.WithdrawalStatusCompleted {
		return nil, math.LegacyZeroDec(), types.ErrWithdrawalNotFound
	}

	// Get pool
	pool := k.GetPool(sdkCtx, withdrawal.PoolID)
	if pool == nil {
		return nil, math.LegacyZeroDec(), types.ErrPoolNotFound
	}

	// Calculate redemption with pro-rata if needed
	sharesToRedeem, amountToReceive := k.calculateProRataRedemption(sdkCtx, pool, withdrawal)

	// Update withdrawal
	withdrawal.SharesRedeemed = withdrawal.SharesRedeemed.Add(sharesToRedeem)
	withdrawal.AmountReceived = withdrawal.AmountReceived.Add(amountToReceive)
	withdrawal.CompletedAt = time.Now().Unix()

	// Check if fully redeemed
	if withdrawal.SharesRedeemed.GTE(withdrawal.SharesRequested) {
		withdrawal.Status = types.WithdrawalStatusCompleted
	} else {
		withdrawal.Status = types.WithdrawalStatusProcessing
	}

	// Update pool
	pool.TotalDeposits = pool.TotalDeposits.Sub(amountToReceive)
	pool.TotalShares = pool.TotalShares.Sub(sharesToRedeem)
	pool.UpdatedAt = time.Now().Unix()

	// Reduce user's shares from deposits (FIFO)
	k.reduceUserShares(sdkCtx, withdrawal.Withdrawer, withdrawal.PoolID, sharesToRedeem)

	// Save changes
	k.SetWithdrawal(sdkCtx, withdrawal)
	k.SetPool(sdkCtx, pool)

	// Update pool stats
	stats := k.GetPoolStats(sdkCtx, withdrawal.PoolID)
	stats.TotalValueLocked = pool.TotalDeposits
	stats.TotalPendingWithdrawals = stats.TotalPendingWithdrawals.Sub(amountToReceive)
	stats.UpdatedAt = time.Now().Unix()
	k.SetPoolStats(sdkCtx, stats)

	// Emit event
	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent(
			"riverpool_withdrawal_claim",
			sdk.NewAttribute("pool_id", withdrawal.PoolID),
			sdk.NewAttribute("withdrawer", withdrawer),
			sdk.NewAttribute("shares_redeemed", sharesToRedeem.String()),
			sdk.NewAttribute("amount_received", amountToReceive.String()),
			sdk.NewAttribute("status", withdrawal.Status),
		),
	)

	k.logger.Info("Withdrawal claimed",
		"pool_id", withdrawal.PoolID,
		"withdrawer", withdrawer,
		"shares_redeemed", sharesToRedeem.String(),
		"amount_received", amountToReceive.String(),
	)

	return withdrawal, amountToReceive, nil
}

// CancelWithdrawal cancels a pending withdrawal
func (k *Keeper) CancelWithdrawal(ctx context.Context, withdrawer, withdrawalID string) (*types.Withdrawal, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// Get withdrawal
	withdrawal := k.GetWithdrawal(sdkCtx, withdrawalID)
	if withdrawal == nil {
		return nil, types.ErrWithdrawalNotFound
	}

	// Verify owner
	if withdrawal.Withdrawer != withdrawer {
		return nil, types.ErrUnauthorized
	}

	// Can only cancel pending withdrawals
	if withdrawal.Status != types.WithdrawalStatusPending {
		return nil, types.ErrWithdrawalNotFound
	}

	// Get pool for stats update
	pool := k.GetPool(sdkCtx, withdrawal.PoolID)
	estimatedAmount := math.LegacyZeroDec()
	if pool != nil {
		estimatedAmount = pool.CalculateValueForShares(withdrawal.SharesRequested.Sub(withdrawal.SharesRedeemed))
	}

	// Update withdrawal status
	withdrawal.Status = types.WithdrawalStatusCancelled
	withdrawal.CompletedAt = time.Now().Unix()

	// Save changes
	k.SetWithdrawal(sdkCtx, withdrawal)

	// Update pool stats
	if pool != nil {
		stats := k.GetPoolStats(sdkCtx, withdrawal.PoolID)
		stats.TotalPendingWithdrawals = stats.TotalPendingWithdrawals.Sub(estimatedAmount)
		stats.UpdatedAt = time.Now().Unix()
		k.SetPoolStats(sdkCtx, stats)
	}

	// Emit event
	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent(
			"riverpool_withdrawal_cancel",
			sdk.NewAttribute("pool_id", withdrawal.PoolID),
			sdk.NewAttribute("withdrawer", withdrawer),
			sdk.NewAttribute("withdrawal_id", withdrawalID),
			sdk.NewAttribute("shares_returned", withdrawal.SharesRequested.Sub(withdrawal.SharesRedeemed).String()),
		),
	)

	k.logger.Info("Withdrawal cancelled",
		"pool_id", withdrawal.PoolID,
		"withdrawer", withdrawer,
		"withdrawal_id", withdrawalID,
	)

	return withdrawal, nil
}

// calculateProRataRedemption calculates the pro-rata redemption amount
func (k *Keeper) calculateProRataRedemption(ctx sdk.Context, pool *types.Pool, withdrawal *types.Withdrawal) (math.LegacyDec, math.LegacyDec) {
	// Get remaining shares to redeem
	remainingShares := withdrawal.SharesRequested.Sub(withdrawal.SharesRedeemed)
	if remainingShares.IsZero() || remainingShares.IsNegative() {
		return math.LegacyZeroDec(), math.LegacyZeroDec()
	}

	// Calculate daily limit
	dailyLimit := pool.TotalDeposits.Mul(pool.DailyRedemptionLimit)
	if dailyLimit.IsZero() {
		// No limit, redeem all
		amountToReceive := pool.CalculateValueForShares(remainingShares)
		return remainingShares, amountToReceive
	}

	// Get total pending withdrawals for today
	pendingWithdrawals := k.GetPendingWithdrawals(ctx, pool.PoolID)
	totalPendingShares := math.LegacyZeroDec()
	for _, w := range pendingWithdrawals {
		if w.IsReady() {
			totalPendingShares = totalPendingShares.Add(w.SharesRequested.Sub(w.SharesRedeemed))
		}
	}

	// Calculate available capacity
	availableCapacity := dailyLimit

	// If total pending exceeds capacity, pro-rata
	totalPendingValue := pool.CalculateValueForShares(totalPendingShares)
	if totalPendingValue.GT(availableCapacity) {
		// Pro-rata allocation
		ratio := availableCapacity.Quo(totalPendingValue)
		sharesToRedeem := remainingShares.Mul(ratio)
		amountToReceive := pool.CalculateValueForShares(sharesToRedeem)
		return sharesToRedeem, amountToReceive
	}

	// Full redemption
	amountToReceive := pool.CalculateValueForShares(remainingShares)
	return remainingShares, amountToReceive
}

// reduceUserShares reduces user's shares from deposits (FIFO order)
func (k *Keeper) reduceUserShares(ctx sdk.Context, user, poolID string, sharesToReduce math.LegacyDec) {
	deposits := k.GetUserDeposits(ctx, user)

	// Filter and sort by deposit time (FIFO)
	var poolDeposits []*types.Deposit
	for _, d := range deposits {
		if d.PoolID == poolID && !d.IsLocked() && d.Shares.IsPositive() {
			poolDeposits = append(poolDeposits, d)
		}
	}
	sort.Slice(poolDeposits, func(i, j int) bool {
		return poolDeposits[i].DepositedAt < poolDeposits[j].DepositedAt
	})

	remaining := sharesToReduce
	for _, deposit := range poolDeposits {
		if remaining.IsZero() || remaining.IsNegative() {
			break
		}

		if deposit.Shares.LTE(remaining) {
			// Use entire deposit
			remaining = remaining.Sub(deposit.Shares)
			deposit.Shares = math.LegacyZeroDec()
		} else {
			// Partial use
			deposit.Shares = deposit.Shares.Sub(remaining)
			remaining = math.LegacyZeroDec()
		}
		k.SetDeposit(ctx, deposit)
	}
}

// GetQueuePosition returns the position in the withdrawal queue
func (k *Keeper) GetQueuePosition(ctx sdk.Context, poolID, withdrawalID string) int {
	pendingWithdrawals := k.GetPendingWithdrawals(ctx, poolID)

	// Sort by request time
	sort.Slice(pendingWithdrawals, func(i, j int) bool {
		return pendingWithdrawals[i].RequestedAt < pendingWithdrawals[j].RequestedAt
	})

	for i, w := range pendingWithdrawals {
		if w.WithdrawalID == withdrawalID {
			return i + 1
		}
	}
	return 0
}
