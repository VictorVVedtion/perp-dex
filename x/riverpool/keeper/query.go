package keeper

import (
	"context"
	"strconv"
	"time"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/openalpha/perp-dex/x/riverpool/types"
)

// QueryServer defines the riverpool QueryServer
type QueryServer struct {
	keeper *Keeper
}

// NewQueryServerImpl creates a new QueryServer instance
func NewQueryServerImpl(keeper *Keeper) *QueryServer {
	return &QueryServer{keeper: keeper}
}

// Pool returns a pool by ID
func (q *QueryServer) Pool(ctx context.Context, poolID string) (*types.Pool, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	pool := q.keeper.GetPool(sdkCtx, poolID)
	if pool == nil {
		return nil, types.ErrPoolNotFound
	}
	return pool, nil
}

// Pools returns all pools
func (q *QueryServer) Pools(ctx context.Context, offset, limit uint64) ([]*types.Pool, uint64, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	allPools := q.keeper.GetAllPools(sdkCtx)

	total := uint64(len(allPools))

	// Apply pagination
	if offset >= total {
		return []*types.Pool{}, total, nil
	}

	end := offset + limit
	if end > total || limit == 0 {
		end = total
	}

	return allPools[offset:end], total, nil
}

// PoolsByType returns pools filtered by type
func (q *QueryServer) PoolsByType(ctx context.Context, poolType string) ([]*types.Pool, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	pools := q.keeper.GetPoolsByType(sdkCtx, poolType)
	return pools, nil
}

// UserDeposits returns all deposits for a user
func (q *QueryServer) UserDeposits(ctx context.Context, user string) ([]*types.Deposit, math.LegacyDec, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	deposits := q.keeper.GetUserDeposits(sdkCtx, user)

	totalValue := math.LegacyZeroDec()
	for _, deposit := range deposits {
		pool := q.keeper.GetPool(sdkCtx, deposit.PoolID)
		if pool != nil {
			value := pool.CalculateValueForShares(deposit.Shares)
			totalValue = totalValue.Add(value)
		}
	}

	return deposits, totalValue, nil
}

// PoolDeposits returns all deposits in a pool
func (q *QueryServer) PoolDeposits(ctx context.Context, poolID string, offset, limit uint64) ([]*types.Deposit, uint64, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	allDeposits := q.keeper.GetPoolDeposits(sdkCtx, poolID)

	total := uint64(len(allDeposits))

	// Apply pagination
	if offset >= total {
		return []*types.Deposit{}, total, nil
	}

	end := offset + limit
	if end > total || limit == 0 {
		end = total
	}

	return allDeposits[offset:end], total, nil
}

// UserWithdrawals returns all withdrawals for a user
func (q *QueryServer) UserWithdrawals(ctx context.Context, user string) ([]*types.Withdrawal, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	withdrawals := q.keeper.GetUserWithdrawals(sdkCtx, user)
	return withdrawals, nil
}

// PendingWithdrawals returns all pending withdrawals for a pool
func (q *QueryServer) PendingWithdrawals(ctx context.Context, poolID string) ([]*types.Withdrawal, math.LegacyDec, math.LegacyDec, math.LegacyDec, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	withdrawals := q.keeper.GetPendingWithdrawals(sdkCtx, poolID)

	pool := q.keeper.GetPool(sdkCtx, poolID)
	if pool == nil {
		return withdrawals, math.LegacyZeroDec(), math.LegacyZeroDec(), math.LegacyZeroDec(), nil
	}

	// Calculate totals
	totalPendingShares := math.LegacyZeroDec()
	for _, w := range withdrawals {
		totalPendingShares = totalPendingShares.Add(w.SharesRequested.Sub(w.SharesRedeemed))
	}
	totalPendingValue := pool.CalculateValueForShares(totalPendingShares)

	// Calculate daily limit remaining
	dailyLimit := pool.TotalDeposits.Mul(pool.DailyRedemptionLimit)
	// For simplicity, assume daily limit remaining equals daily limit
	// In production, track daily redemptions
	dailyLimitRemaining := dailyLimit

	return withdrawals, totalPendingShares, totalPendingValue, dailyLimitRemaining, nil
}

// PoolStats returns statistics for a pool
func (q *QueryServer) PoolStats(ctx context.Context, poolID string) (*types.PoolStats, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	stats := q.keeper.GetPoolStats(sdkCtx, poolID)
	return stats, nil
}

// NAVHistory returns historical NAV data for a pool
func (q *QueryServer) NAVHistory(ctx context.Context, poolID string, fromTime, toTime int64) ([]*types.NAVHistory, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	history := q.keeper.GetNAVHistory(sdkCtx, poolID, fromTime, toTime)
	return history, nil
}

// DDGuardState returns the DDGuard state for a pool
func (q *QueryServer) DDGuardState(ctx context.Context, poolID string) (*types.DDGuardState, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	state := q.keeper.GetDDGuardState(sdkCtx, poolID)
	if state == nil {
		// Return default state if not found
		pool := q.keeper.GetPool(sdkCtx, poolID)
		if pool == nil {
			return nil, types.ErrPoolNotFound
		}
		state = &types.DDGuardState{
			PoolID:           poolID,
			Level:            pool.DDGuardLevel,
			PeakNAV:          pool.HighWaterMark,
			CurrentNAV:       pool.NAV,
			DrawdownPercent:  pool.CurrentDrawdown,
			MaxExposureLimit: math.LegacyOneDec(),
			TriggeredAt:      0,
			LastCheckedAt:    time.Now().Unix(),
		}
	}
	return state, nil
}

// UserPoolBalance returns user's balance and shares in a pool
func (q *QueryServer) UserPoolBalance(ctx context.Context, poolID, user string) (
	shares, value, costBasis, unrealizedPnL, pnlPercent math.LegacyDec,
	unlockAt int64, canWithdraw bool, err error,
) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	shares, value, costBasis = q.keeper.GetUserPoolBalance(sdkCtx, poolID, user)

	// Calculate unrealized PnL
	unrealizedPnL = value.Sub(costBasis)

	// Calculate PnL percentage
	if costBasis.IsPositive() {
		pnlPercent = unrealizedPnL.Quo(costBasis).Mul(math.LegacyNewDec(100))
	} else {
		pnlPercent = math.LegacyZeroDec()
	}

	// Check unlock status
	deposits := q.keeper.GetUserDeposits(sdkCtx, user)
	canWithdraw = true
	unlockAt = 0
	for _, deposit := range deposits {
		if deposit.PoolID == poolID && deposit.IsLocked() {
			canWithdraw = false
			if deposit.UnlockAt > unlockAt {
				unlockAt = deposit.UnlockAt
			}
		}
	}

	return shares, value, costBasis, unrealizedPnL, pnlPercent, unlockAt, canWithdraw, nil
}

// EstimateDeposit estimates shares for a given deposit amount
func (q *QueryServer) EstimateDeposit(ctx context.Context, poolID string, amount math.LegacyDec) (shares, nav, sharePrice math.LegacyDec, err error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	pool := q.keeper.GetPool(sdkCtx, poolID)
	if pool == nil {
		return math.LegacyZeroDec(), math.LegacyZeroDec(), math.LegacyZeroDec(), types.ErrPoolNotFound
	}

	nav = pool.NAV
	shares = pool.CalculateSharesForDeposit(amount)
	sharePrice = nav // 1 share = NAV

	return shares, nav, sharePrice, nil
}

// EstimateWithdrawal estimates amount for a given share redemption
func (q *QueryServer) EstimateWithdrawal(ctx context.Context, poolID string, shares math.LegacyDec) (
	amount, nav math.LegacyDec, availableAt int64, queuePosition string, mayBeProrated bool, err error,
) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	pool := q.keeper.GetPool(sdkCtx, poolID)
	if pool == nil {
		return math.LegacyZeroDec(), math.LegacyZeroDec(), 0, "0", false, types.ErrPoolNotFound
	}

	nav = pool.NAV
	amount = pool.CalculateValueForShares(shares)
	availableAt = time.Now().Unix() + pool.RedemptionDelayDays*24*60*60

	// Estimate queue position
	pendingWithdrawals := q.keeper.GetPendingWithdrawals(sdkCtx, poolID)
	queuePosition = strconv.Itoa(len(pendingWithdrawals) + 1)

	// Check if pro-rata might apply
	dailyLimit := pool.TotalDeposits.Mul(pool.DailyRedemptionLimit)
	pendingTotal := math.LegacyZeroDec()
	for _, w := range pendingWithdrawals {
		pendingTotal = pendingTotal.Add(pool.CalculateValueForShares(w.SharesRequested.Sub(w.SharesRedeemed)))
	}
	mayBeProrated = pendingTotal.Add(amount).GT(dailyLimit)

	return amount, nav, availableAt, queuePosition, mayBeProrated, nil
}
