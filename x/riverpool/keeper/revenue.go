package keeper

import (
	"encoding/json"
	"time"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/openalpha/perp-dex/x/riverpool/types"
)

// RevenueSource represents different sources of pool revenue
type RevenueSource string

const (
	RevenueSourceSpread      RevenueSource = "spread"      // Bid-ask spread earnings
	RevenueSourceFunding     RevenueSource = "funding"     // Funding rate payments
	RevenueSourceLiquidation RevenueSource = "liquidation" // Liquidation profits
	RevenueSourceTrading     RevenueSource = "trading"     // Trading PnL
	RevenueSourceFees        RevenueSource = "fees"        // Fee rebates
)

// RevenueRecord tracks individual revenue events
type RevenueRecord struct {
	RecordID    string
	PoolID      string
	Source      RevenueSource
	Amount      math.LegacyDec
	NAVImpact   math.LegacyDec // Impact on NAV per share
	Timestamp   int64
	BlockHeight int64
	MarketID    string // Optional: relevant market
	PositionID  string // Optional: relevant position
	Details     string // Additional context
}

// PoolRevenueStats aggregates revenue statistics for a pool
type PoolRevenueStats struct {
	PoolID            string
	TotalRevenue      math.LegacyDec
	SpreadRevenue     math.LegacyDec
	FundingRevenue    math.LegacyDec
	LiquidationProfit math.LegacyDec
	TradingPnL        math.LegacyDec
	FeeRebates        math.LegacyDec
	LastUpdated       int64
}

// RecordRevenue records a new revenue event for a pool
func (k *Keeper) RecordRevenue(
	ctx sdk.Context,
	poolID string,
	source RevenueSource,
	amount math.LegacyDec,
	marketID string,
	positionID string,
	details string,
) error {
	pool := k.GetPool(ctx, poolID)
	if pool == nil {
		return types.ErrPoolNotFound
	}

	// Calculate NAV impact
	navImpact := math.LegacyZeroDec()
	if pool.TotalShares.GT(math.LegacyZeroDec()) {
		navImpact = amount.Quo(pool.TotalShares)
	}

	// Create revenue record
	record := &RevenueRecord{
		RecordID:    k.generateRevenueRecordID(poolID, ctx.BlockHeight()),
		PoolID:      poolID,
		Source:      source,
		Amount:      amount,
		NAVImpact:   navImpact,
		Timestamp:   time.Now().Unix(),
		BlockHeight: ctx.BlockHeight(),
		MarketID:    marketID,
		PositionID:  positionID,
		Details:     details,
	}

	// Store revenue record
	k.SetRevenueRecord(ctx, record)

	// Update pool revenue stats
	k.updatePoolRevenueStats(ctx, record)

	// Update pool NAV with new revenue
	if amount.GT(math.LegacyZeroDec()) {
		pool.TotalDeposits = pool.TotalDeposits.Add(amount)
		pool.UpdateNAV(pool.TotalDeposits)
		k.SetPool(ctx, pool)
	}

	// Emit event
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"riverpool_revenue_recorded",
			sdk.NewAttribute("pool_id", poolID),
			sdk.NewAttribute("source", string(source)),
			sdk.NewAttribute("amount", amount.String()),
			sdk.NewAttribute("nav_impact", navImpact.String()),
			sdk.NewAttribute("market_id", marketID),
		),
	)

	k.logger.Info("Revenue recorded",
		"pool_id", poolID,
		"source", source,
		"amount", amount.String(),
		"nav_impact", navImpact.String(),
	)

	return nil
}

// RecordLoss records a loss event for a pool
func (k *Keeper) RecordLoss(
	ctx sdk.Context,
	poolID string,
	source RevenueSource,
	amount math.LegacyDec,
	marketID string,
	positionID string,
	details string,
) error {
	// Record as negative revenue
	return k.RecordRevenue(ctx, poolID, source, amount.Neg(), marketID, positionID, details)
}

// SetRevenueRecord stores a revenue record
func (k *Keeper) SetRevenueRecord(ctx sdk.Context, record *RevenueRecord) {
	store := k.GetStore(ctx)
	key := k.getRevenueRecordKey(record.PoolID, record.RecordID)
	bz, err := json.Marshal(record)
	if err != nil {
		k.logger.Error("Failed to marshal revenue record", "error", err)
		return
	}
	store.Set(key, bz)
}

// GetRevenueRecord retrieves a revenue record
func (k *Keeper) GetRevenueRecord(ctx sdk.Context, poolID, recordID string) *RevenueRecord {
	store := k.GetStore(ctx)
	key := k.getRevenueRecordKey(poolID, recordID)
	bz := store.Get(key)
	if bz == nil {
		return nil
	}

	var record RevenueRecord
	if err := json.Unmarshal(bz, &record); err != nil {
		k.logger.Error("Failed to unmarshal revenue record", "error", err)
		return nil
	}
	return &record
}

// GetPoolRevenueRecords retrieves all revenue records for a pool
func (k *Keeper) GetPoolRevenueRecords(ctx sdk.Context, poolID string, from, to int64) []*RevenueRecord {
	store := k.GetStore(ctx)
	prefix := append(RevenueRecordKeyPrefix, []byte(poolID)...)
	iterator := store.Iterator(prefix, nil)
	defer iterator.Close()

	var records []*RevenueRecord
	for ; iterator.Valid(); iterator.Next() {
		var record RevenueRecord
		if err := json.Unmarshal(iterator.Value(), &record); err != nil {
			k.logger.Error("Failed to unmarshal revenue record", "error", err)
			continue
		}

		// Apply time filter if specified
		if from > 0 && record.Timestamp < from {
			continue
		}
		if to > 0 && record.Timestamp > to {
			continue
		}

		records = append(records, &record)
	}

	return records
}

// updatePoolRevenueStats updates the aggregate revenue stats for a pool
func (k *Keeper) updatePoolRevenueStats(ctx sdk.Context, record *RevenueRecord) {
	stats := k.GetPoolRevenueStats(ctx, record.PoolID)
	if stats == nil {
		stats = &PoolRevenueStats{
			PoolID:            record.PoolID,
			TotalRevenue:      math.LegacyZeroDec(),
			SpreadRevenue:     math.LegacyZeroDec(),
			FundingRevenue:    math.LegacyZeroDec(),
			LiquidationProfit: math.LegacyZeroDec(),
			TradingPnL:        math.LegacyZeroDec(),
			FeeRebates:        math.LegacyZeroDec(),
		}
	}

	// Update total
	stats.TotalRevenue = stats.TotalRevenue.Add(record.Amount)
	stats.LastUpdated = record.Timestamp

	// Update source-specific stat
	switch record.Source {
	case RevenueSourceSpread:
		stats.SpreadRevenue = stats.SpreadRevenue.Add(record.Amount)
	case RevenueSourceFunding:
		stats.FundingRevenue = stats.FundingRevenue.Add(record.Amount)
	case RevenueSourceLiquidation:
		stats.LiquidationProfit = stats.LiquidationProfit.Add(record.Amount)
	case RevenueSourceTrading:
		stats.TradingPnL = stats.TradingPnL.Add(record.Amount)
	case RevenueSourceFees:
		stats.FeeRebates = stats.FeeRebates.Add(record.Amount)
	}

	k.SetPoolRevenueStats(ctx, stats)
}

// GetPoolRevenueStats retrieves aggregate revenue stats for a pool
func (k *Keeper) GetPoolRevenueStats(ctx sdk.Context, poolID string) *PoolRevenueStats {
	store := k.GetStore(ctx)
	key := k.getRevenueStatsKey(poolID)
	bz := store.Get(key)
	if bz == nil {
		return nil
	}

	var stats PoolRevenueStats
	if err := json.Unmarshal(bz, &stats); err != nil {
		k.logger.Error("Failed to unmarshal pool revenue stats", "error", err)
		return nil
	}
	return &stats
}

// SetPoolRevenueStats stores aggregate revenue stats for a pool
func (k *Keeper) SetPoolRevenueStats(ctx sdk.Context, stats *PoolRevenueStats) {
	store := k.GetStore(ctx)
	key := k.getRevenueStatsKey(stats.PoolID)
	bz, err := json.Marshal(stats)
	if err != nil {
		k.logger.Error("Failed to marshal pool revenue stats", "error", err)
		return
	}
	store.Set(key, bz)
}

// GetPoolRevenueByPeriod calculates revenue for a specific time period
func (k *Keeper) GetPoolRevenueByPeriod(ctx sdk.Context, poolID string, periodDays int) math.LegacyDec {
	now := time.Now().Unix()
	periodStart := now - int64(periodDays*24*60*60)

	records := k.GetPoolRevenueRecords(ctx, poolID, periodStart, now)

	total := math.LegacyZeroDec()
	for _, record := range records {
		total = total.Add(record.Amount)
	}

	return total
}

// CalculatePoolReturn calculates the return percentage for a period
func (k *Keeper) CalculatePoolReturn(ctx sdk.Context, poolID string, periodDays int) math.LegacyDec {
	pool := k.GetPool(ctx, poolID)
	if pool == nil || pool.TotalDeposits.IsZero() {
		return math.LegacyZeroDec()
	}

	revenue := k.GetPoolRevenueByPeriod(ctx, poolID, periodDays)

	// Calculate return as percentage
	// Return = (Revenue / Starting TVL) * 100
	// For simplicity, using current TVL as approximation
	return revenue.Quo(pool.TotalDeposits).Mul(math.LegacyNewDec(100))
}

// GetPoolRevenueBreakdown returns revenue breakdown by source for a period
func (k *Keeper) GetPoolRevenueBreakdown(ctx sdk.Context, poolID string, periodDays int) map[RevenueSource]math.LegacyDec {
	now := time.Now().Unix()
	periodStart := now - int64(periodDays*24*60*60)

	records := k.GetPoolRevenueRecords(ctx, poolID, periodStart, now)

	breakdown := make(map[RevenueSource]math.LegacyDec)
	breakdown[RevenueSourceSpread] = math.LegacyZeroDec()
	breakdown[RevenueSourceFunding] = math.LegacyZeroDec()
	breakdown[RevenueSourceLiquidation] = math.LegacyZeroDec()
	breakdown[RevenueSourceTrading] = math.LegacyZeroDec()
	breakdown[RevenueSourceFees] = math.LegacyZeroDec()

	for _, record := range records {
		current := breakdown[record.Source]
		breakdown[record.Source] = current.Add(record.Amount)
	}

	return breakdown
}

// getRevenueRecordKey generates the store key for a revenue record
func (k *Keeper) getRevenueRecordKey(poolID, recordID string) []byte {
	return append(append(RevenueRecordKeyPrefix, []byte(poolID)...), []byte(recordID)...)
}

// getRevenueStatsKey generates the store key for revenue stats
func (k *Keeper) getRevenueStatsKey(poolID string) []byte {
	return append(RevenueStatsKeyPrefix, []byte(poolID)...)
}

// generateRevenueRecordID generates a unique ID for a revenue record
func (k *Keeper) generateRevenueRecordID(poolID string, blockHeight int64) string {
	return poolID + "-" + math.NewInt(blockHeight).String() + "-" + math.NewInt(time.Now().UnixNano()).String()
}

// RevenueStatsKeyPrefix is the prefix for revenue stats
var RevenueStatsKeyPrefix = []byte{0x0B}
