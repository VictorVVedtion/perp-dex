package keeper

import (
	"encoding/json"
	"time"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/openalpha/perp-dex/x/riverpool/types"
)

// EndBlocker is called at the end of each block to process pool operations
func (k *Keeper) EndBlocker(ctx sdk.Context) error {
	blockHeight := ctx.BlockHeight()
	start := time.Now()

	// Phase 1: Update NAVs for all active pools
	navStart := time.Now()
	k.UpdateAllPoolNAVs(ctx)
	navDuration := time.Since(navStart)

	// Phase 2: Process pending withdrawals that are now available
	processStart := time.Now()
	processedCount := k.ProcessReadyWithdrawals(ctx)
	processDuration := time.Since(processStart)

	// Phase 3: Check DDGuard levels and take action if needed
	ddStart := time.Now()
	k.CheckDDGuardActions(ctx)
	ddDuration := time.Since(ddStart)

	totalDuration := time.Since(start)

	// Log performance metrics
	k.logger.Debug("RiverPool EndBlocker completed",
		"block", blockHeight,
		"total_ms", totalDuration.Milliseconds(),
		"nav_update_ms", navDuration.Milliseconds(),
		"withdrawal_process_ms", processDuration.Milliseconds(),
		"ddguard_check_ms", ddDuration.Milliseconds(),
		"withdrawals_processed", processedCount,
	)

	// Emit telemetry event
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"riverpool_endblock",
			sdk.NewAttribute("block_height", math.NewInt(blockHeight).String()),
			sdk.NewAttribute("duration_ms", math.NewInt(totalDuration.Milliseconds()).String()),
			sdk.NewAttribute("withdrawals_processed", math.NewInt(int64(processedCount)).String()),
		),
	)

	return nil
}

// ProcessReadyWithdrawals processes all pending withdrawals that are ready
func (k *Keeper) ProcessReadyWithdrawals(ctx sdk.Context) int {
	now := time.Now().Unix()
	processedCount := 0

	// Get all pools
	pools := k.GetAllPools(ctx)
	for _, pool := range pools {
		if pool.Status == types.PoolStatusClosed {
			continue
		}

		// Get pending withdrawals for this pool
		pendingWithdrawals := k.GetPendingWithdrawals(ctx, pool.PoolID)
		if len(pendingWithdrawals) == 0 {
			continue
		}

		// Calculate daily available amount (15% TVL limit)
		dailyLimit := pool.TotalDeposits.Mul(pool.DailyRedemptionLimit)
		processedToday := k.GetDailyProcessedAmount(ctx, pool.PoolID)
		availableToday := dailyLimit.Sub(processedToday)

		if availableToday.LTE(math.LegacyZeroDec()) {
			k.logger.Debug("Daily limit reached for pool",
				"pool_id", pool.PoolID,
				"daily_limit", dailyLimit.String(),
				"processed_today", processedToday.String(),
			)
			continue
		}

		// Calculate total pending value for ready withdrawals
		var readyWithdrawals []*types.Withdrawal
		totalPendingValue := math.LegacyZeroDec()

		for _, w := range pendingWithdrawals {
			if w.AvailableAt <= now {
				readyWithdrawals = append(readyWithdrawals, w)
				// Calculate value for pending shares
				value := w.SharesRequested.Sub(w.SharesRedeemed).Mul(pool.NAV)
				totalPendingValue = totalPendingValue.Add(value)
			}
		}

		if len(readyWithdrawals) == 0 {
			continue
		}

		// Determine if pro-rata distribution is needed
		needsProRata := totalPendingValue.GT(availableToday)

		// Process each ready withdrawal
		for _, w := range readyWithdrawals {
			pendingShares := w.SharesRequested.Sub(w.SharesRedeemed)
			if pendingShares.LTE(math.LegacyZeroDec()) {
				continue
			}

			pendingValue := pendingShares.Mul(pool.NAV)
			var sharesToProcess math.LegacyDec

			if needsProRata {
				// Pro-rata allocation: user's share of available funds
				userRatio := pendingValue.Quo(totalPendingValue)
				userAllocation := availableToday.Mul(userRatio)
				sharesToProcess = userAllocation.Quo(pool.NAV)

				// Ensure we don't process more than requested
				if sharesToProcess.GT(pendingShares) {
					sharesToProcess = pendingShares
				}
			} else {
				sharesToProcess = pendingShares
			}

			// Mark shares as redeemed (partial or full)
			w.SharesRedeemed = w.SharesRedeemed.Add(sharesToProcess)
			amountToSend := sharesToProcess.Mul(pool.NAV)
			w.AmountReceived = w.AmountReceived.Add(amountToSend)

			// Check if withdrawal is fully complete
			if w.SharesRedeemed.GTE(w.SharesRequested) {
				w.Status = types.WithdrawalStatusCompleted
				w.CompletedAt = now
			} else {
				// Partial fill - still processing
				w.Status = types.WithdrawalStatusProcessing
			}

			// Save updated withdrawal
			k.SetWithdrawal(ctx, w)

			// Update pool totals
			pool.TotalShares = pool.TotalShares.Sub(sharesToProcess)
			pool.TotalDeposits = pool.TotalDeposits.Sub(amountToSend)

			// Record daily processed amount
			k.AddDailyProcessedAmount(ctx, pool.PoolID, amountToSend)

			processedCount++

			// Emit withdrawal processed event
			ctx.EventManager().EmitEvent(
				sdk.NewEvent(
					"riverpool_withdrawal_processed",
					sdk.NewAttribute("withdrawal_id", w.WithdrawalID),
					sdk.NewAttribute("pool_id", pool.PoolID),
					sdk.NewAttribute("withdrawer", w.Withdrawer),
					sdk.NewAttribute("shares_redeemed", sharesToProcess.String()),
					sdk.NewAttribute("amount_sent", amountToSend.String()),
					sdk.NewAttribute("is_complete", math.NewInt(boolToInt(w.Status == types.WithdrawalStatusCompleted)).String()),
					sdk.NewAttribute("pro_rata", math.NewInt(boolToInt(needsProRata)).String()),
				),
			)

			k.logger.Info("Withdrawal processed",
				"withdrawal_id", w.WithdrawalID,
				"pool_id", pool.PoolID,
				"shares_redeemed", sharesToProcess.String(),
				"amount_sent", amountToSend.String(),
				"status", w.Status,
				"pro_rata", needsProRata,
			)
		}

		// Save updated pool
		k.SetPool(ctx, pool)
	}

	return processedCount
}

// CheckDDGuardActions checks DDGuard levels and takes automatic actions
func (k *Keeper) CheckDDGuardActions(ctx sdk.Context) {
	pools := k.GetAllPools(ctx)

	for _, pool := range pools {
		if pool.Status == types.PoolStatusClosed {
			continue
		}

		state := k.GetDDGuardState(ctx, pool.PoolID)
		if state == nil {
			continue
		}

		// Take automatic actions based on DDGuard level
		switch state.Level {
		case types.DDGuardLevelHalt:
			// If pool is not already paused, pause it
			if pool.Status == types.PoolStatusActive {
				pool.Status = types.PoolStatusPaused
				k.SetPool(ctx, pool)

				ctx.EventManager().EmitEvent(
					sdk.NewEvent(
						"riverpool_pool_paused",
						sdk.NewAttribute("pool_id", pool.PoolID),
						sdk.NewAttribute("reason", "ddguard_halt"),
						sdk.NewAttribute("drawdown", state.DrawdownPercent.String()),
					),
				)

				k.logger.Warn("Pool paused due to DDGuard halt level",
					"pool_id", pool.PoolID,
					"drawdown", state.DrawdownPercent.String(),
				)
			}

		case types.DDGuardLevelReduce:
			// Log warning but don't auto-pause
			// Exposure reduction is handled at order placement time
			k.logger.Info("Pool in DDGuard reduce level",
				"pool_id", pool.PoolID,
				"drawdown", state.DrawdownPercent.String(),
				"max_exposure", state.MaxExposureLimit.String(),
			)

		case types.DDGuardLevelWarning:
			// Just log for monitoring
			k.logger.Debug("Pool in DDGuard warning level",
				"pool_id", pool.PoolID,
				"drawdown", state.DrawdownPercent.String(),
			)
		}

		// Check if pool can be unpaused (recovered from drawdown)
		if pool.Status == types.PoolStatusPaused && state.Level == types.DDGuardLevelNormal {
			// Auto-unpause only if drawdown has fully recovered
			if state.DrawdownPercent.LT(math.LegacyMustNewDecFromStr("0.05")) {
				pool.Status = types.PoolStatusActive
				k.SetPool(ctx, pool)

				ctx.EventManager().EmitEvent(
					sdk.NewEvent(
						"riverpool_pool_resumed",
						sdk.NewAttribute("pool_id", pool.PoolID),
						sdk.NewAttribute("reason", "ddguard_recovered"),
						sdk.NewAttribute("drawdown", state.DrawdownPercent.String()),
					),
				)

				k.logger.Info("Pool resumed after DDGuard recovery",
					"pool_id", pool.PoolID,
					"drawdown", state.DrawdownPercent.String(),
				)
			}
		}
	}
}

// GetDailyProcessedAmount gets the amount processed today for a pool
func (k *Keeper) GetDailyProcessedAmount(ctx sdk.Context, poolID string) math.LegacyDec {
	store := k.GetStore(ctx)
	key := k.getDailyProcessedKey(poolID)

	bz := store.Get(key)
	if bz == nil {
		return math.LegacyZeroDec()
	}

	var processed DailyProcessed
	if err := json.Unmarshal(bz, &processed); err != nil {
		k.logger.Error("Failed to unmarshal daily processed", "error", err)
		return math.LegacyZeroDec()
	}

	// Check if it's a new day
	today := time.Now().UTC().Truncate(24 * time.Hour).Unix()
	if processed.Date != today {
		return math.LegacyZeroDec()
	}

	return processed.Amount
}

// AddDailyProcessedAmount adds to the daily processed amount
func (k *Keeper) AddDailyProcessedAmount(ctx sdk.Context, poolID string, amount math.LegacyDec) {
	store := k.GetStore(ctx)
	key := k.getDailyProcessedKey(poolID)

	today := time.Now().UTC().Truncate(24 * time.Hour).Unix()
	current := k.GetDailyProcessedAmount(ctx, poolID)

	processed := DailyProcessed{
		PoolID: poolID,
		Date:   today,
		Amount: current.Add(amount),
	}

	bz, err := json.Marshal(&processed)
	if err != nil {
		k.logger.Error("Failed to marshal daily processed", "error", err)
		return
	}
	store.Set(key, bz)
}

// getDailyProcessedKey generates the store key for daily processed amounts
func (k *Keeper) getDailyProcessedKey(poolID string) []byte {
	return append(DailyProcessedKeyPrefix, []byte(poolID)...)
}

// DailyProcessedKeyPrefix is the prefix for daily processed amounts
var DailyProcessedKeyPrefix = []byte{0x0A}

// DailyProcessed tracks daily withdrawal processing
type DailyProcessed struct {
	PoolID string
	Date   int64
	Amount math.LegacyDec
}

// Helper function to convert bool to int
func boolToInt(b bool) int64 {
	if b {
		return 1
	}
	return 0
}
