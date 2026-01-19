package keeper

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"time"

	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/openalpha/perp-dex/x/perpetual/types"
)

// Store key prefixes for funding
var (
	FundingRateKeyPrefix     = []byte{0x05}
	FundingPaymentKeyPrefix  = []byte{0x06}
	NextFundingTimeKeyPrefix = []byte{0x07}
	FundingConfigKeyPrefix   = []byte{0x08}
	FundingPaymentCounterKey = []byte{0x09}
)

const fundingIntervalHours = 8

func nextFundingTimeUTC(now time.Time) time.Time {
	utc := now.UTC()
	dayStart := time.Date(utc.Year(), utc.Month(), utc.Day(), 0, 0, 0, 0, time.UTC)
	period := time.Duration(fundingIntervalHours) * time.Hour
	elapsed := utc.Sub(dayStart)
	return dayStart.Add((elapsed/period + 1) * period)
}

// ============ Funding Rate Storage ============

// SetFundingRate saves a funding rate record
func (k *Keeper) SetFundingRate(ctx sdk.Context, rate *types.FundingRate) {
	store := k.GetStore(ctx)
	// Key: prefix + marketID + timestamp
	key := append(FundingRateKeyPrefix, []byte(rate.MarketID+":"+fmt.Sprintf("%d", rate.Timestamp.Unix()))...)
	bz, _ := json.Marshal(rate)
	store.Set(key, bz)
}

// GetFundingRateHistory returns funding rate history for a market
func (k *Keeper) GetFundingRateHistory(ctx sdk.Context, marketID string, limit int) []*types.FundingRate {
	store := k.GetStore(ctx)
	prefix := append(FundingRateKeyPrefix, []byte(marketID+":")...)
	iterator := storetypes.KVStoreReversePrefixIterator(store, prefix)
	defer iterator.Close()

	var rates []*types.FundingRate
	count := 0
	for ; iterator.Valid() && count < limit; iterator.Next() {
		var rate types.FundingRate
		if err := json.Unmarshal(iterator.Value(), &rate); err != nil {
			continue
		}
		rates = append(rates, &rate)
		count++
	}
	return rates
}

// ============ Funding Payment Storage ============

// SaveFundingPayment saves a funding payment record
func (k *Keeper) SaveFundingPayment(ctx sdk.Context, payment *types.FundingPayment) {
	store := k.GetStore(ctx)
	key := append(FundingPaymentKeyPrefix, []byte(payment.PaymentID)...)
	bz, _ := json.Marshal(payment)
	store.Set(key, bz)
}

// GetFundingPaymentsByTrader returns funding payments for a trader
func (k *Keeper) GetFundingPaymentsByTrader(ctx sdk.Context, trader string, limit int) []*types.FundingPayment {
	store := k.GetStore(ctx)
	iterator := storetypes.KVStoreReversePrefixIterator(store, FundingPaymentKeyPrefix)
	defer iterator.Close()

	var payments []*types.FundingPayment
	count := 0
	for ; iterator.Valid() && count < limit; iterator.Next() {
		var payment types.FundingPayment
		if err := json.Unmarshal(iterator.Value(), &payment); err != nil {
			continue
		}
		if payment.Trader == trader {
			payments = append(payments, &payment)
			count++
		}
	}
	return payments
}

// generatePaymentID generates a unique payment ID
func (k *Keeper) generatePaymentID(ctx sdk.Context) string {
	store := k.GetStore(ctx)
	bz := store.Get(FundingPaymentCounterKey)
	var counter uint64
	if bz != nil {
		counter = binary.BigEndian.Uint64(bz)
	}
	counter++

	newBz := make([]byte, 8)
	binary.BigEndian.PutUint64(newBz, counter)
	store.Set(FundingPaymentCounterKey, newBz)

	return fmt.Sprintf("funding-%d", counter)
}

// ============ Funding Time Storage ============

// SetNextFundingTime sets the next funding time for a market
func (k *Keeper) SetNextFundingTime(ctx sdk.Context, marketID string, nextTime time.Time) {
	store := k.GetStore(ctx)
	key := append(NextFundingTimeKeyPrefix, []byte(marketID)...)
	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, uint64(nextTime.Unix()))
	store.Set(key, bz)
}

// GetNextFundingTime gets the next funding time for a market
func (k *Keeper) GetNextFundingTime(ctx sdk.Context, marketID string) time.Time {
	store := k.GetStore(ctx)
	key := append(NextFundingTimeKeyPrefix, []byte(marketID)...)
	bz := store.Get(key)
	if bz == nil {
		return time.Time{}
	}
	timestamp := binary.BigEndian.Uint64(bz)
	return time.Unix(int64(timestamp), 0)
}

// ============ Funding Config Storage ============

// SetFundingConfig sets the funding configuration for a market
func (k *Keeper) SetFundingConfig(ctx sdk.Context, marketID string, config types.FundingConfig) {
	store := k.GetStore(ctx)
	key := append(FundingConfigKeyPrefix, []byte(marketID)...)
	bz, _ := json.Marshal(config)
	store.Set(key, bz)
}

// GetFundingConfig gets the funding configuration for a market
func (k *Keeper) GetFundingConfig(ctx sdk.Context, marketID string) types.FundingConfig {
	store := k.GetStore(ctx)
	key := append(FundingConfigKeyPrefix, []byte(marketID)...)
	bz := store.Get(key)
	if bz == nil {
		return types.DefaultFundingConfig()
	}
	var config types.FundingConfig
	if err := json.Unmarshal(bz, &config); err != nil {
		return types.DefaultFundingConfig()
	}
	return config
}

// ============ Funding Rate Calculation ============

// CalculateFundingRate calculates the current funding rate for a market
// Formula: R = dampingFactor × (markPrice - indexPrice) / indexPrice
// Clamped to [minRate, maxRate]
func (k *Keeper) CalculateFundingRate(ctx sdk.Context, marketID string) math.LegacyDec {
	priceInfo := k.GetPrice(ctx, marketID)
	if priceInfo == nil || priceInfo.IndexPrice.IsZero() {
		return math.LegacyZeroDec()
	}

	config := k.GetFundingConfig(ctx, marketID)

	// R = dampingFactor × (mark - index) / index
	priceDiff := priceInfo.MarkPrice.Sub(priceInfo.IndexPrice)
	rate := config.DampingFactor.Mul(priceDiff).Quo(priceInfo.IndexPrice)

	// Clamp to [minRate, maxRate]
	if rate.GT(config.MaxRate) {
		rate = config.MaxRate
	} else if rate.LT(config.MinRate) {
		rate = config.MinRate
	}

	return rate
}

// OI imbalance multiplier for funding rate adjustment
var ImbalanceMultiplier = math.LegacyNewDecWithPrec(5, 2) // 0.05 = 5%

// CalculateFundingRateV2 calculates the funding rate with OI imbalance adjustment
// This version adds an adjustment based on the open interest imbalance between longs and shorts
// Formula: adjustedRate = baseRate + (imbalanceRate × ImbalanceMultiplier)
// Where imbalanceRate = (totalLongOI - totalShortOI) / totalOI
func (k *Keeper) CalculateFundingRateV2(ctx sdk.Context, marketID string) math.LegacyDec {
	// Get base funding rate
	baseRate := k.CalculateFundingRate(ctx, marketID)

	// Get funding info for OI data
	info := k.GetFundingInfo(ctx, marketID)
	if info == nil {
		return baseRate
	}

	// Calculate total OI
	totalOI := info.TotalLongSize.Add(info.TotalShortSize)
	if !totalOI.IsPositive() {
		return baseRate
	}

	// Calculate OI imbalance adjustment
	// Positive imbalance = more longs than shorts → increase funding rate (longs pay more)
	// Negative imbalance = more shorts than longs → decrease funding rate (shorts pay more)
	oiImbalance := info.TotalLongSize.Sub(info.TotalShortSize)
	imbalanceRate := oiImbalance.Quo(totalOI)
	adjustment := imbalanceRate.Mul(ImbalanceMultiplier)

	// Apply adjustment to base rate
	adjustedRate := baseRate.Add(adjustment)

	// Clamp to [minRate, maxRate]
	config := k.GetFundingConfig(ctx, marketID)
	if adjustedRate.GT(config.MaxRate) {
		adjustedRate = config.MaxRate
	} else if adjustedRate.LT(config.MinRate) {
		adjustedRate = config.MinRate
	}

	return adjustedRate
}

// ============ Funding Settlement ============

// SettleFunding settles funding for a market
func (k *Keeper) SettleFunding(ctx sdk.Context, marketID string) error {
	logger := k.Logger()

	market := k.GetMarket(ctx, marketID)
	if market == nil {
		return types.ErrMarketNotFound
	}

	priceInfo := k.GetPrice(ctx, marketID)
	if priceInfo == nil {
		return types.ErrMarketNotFound
	}

	// Calculate funding rate with OI imbalance adjustment
	rate := k.CalculateFundingRateV2(ctx, marketID)

	// Save funding rate record
	fundingRate := types.NewFundingRate(marketID, rate, priceInfo.MarkPrice, priceInfo.IndexPrice)
	k.SetFundingRate(ctx, fundingRate)

	// Get all positions for this market
	positions := k.GetPositionsByMarket(ctx, marketID)

	// Track totals for logging
	var totalLongPayment, totalShortPayment math.LegacyDec
	totalLongPayment = math.LegacyZeroDec()
	totalShortPayment = math.LegacyZeroDec()
	affectedPositions := 0

	// Calculate and apply funding payments
	for _, pos := range positions {
		// Funding payment = notional × rate
		notional := pos.Size.Mul(priceInfo.MarkPrice)
		payment := notional.Mul(rate)

		// Long pays, Short receives (when rate is positive)
		// Long receives, Short pays (when rate is negative)
		if pos.Side == types.PositionSideLong {
			payment = payment.Neg() // Long pays
			totalLongPayment = totalLongPayment.Add(payment)
		} else {
			totalShortPayment = totalShortPayment.Add(payment)
		}

		// Update account balance
		account := k.GetOrCreateAccount(ctx, pos.Trader)
		account.Balance = account.Balance.Add(payment)
		account.UpdatedAt = ctx.BlockTime()
		k.SetAccount(ctx, account)

		// Record payment
		k.SaveFundingPayment(ctx, &types.FundingPayment{
			PaymentID: k.generatePaymentID(ctx),
			Trader:    pos.Trader,
			MarketID:  marketID,
			Amount:    payment,
			Rate:      rate,
			Timestamp: ctx.BlockTime(),
		})

		affectedPositions++
	}

	// Update next funding time
	nextTime := nextFundingTimeUTC(ctx.BlockTime())
	k.SetNextFundingTime(ctx, marketID, nextTime)

	// Emit event
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"funding_settled",
			sdk.NewAttribute("market_id", marketID),
			sdk.NewAttribute("rate", rate.String()),
			sdk.NewAttribute("mark_price", priceInfo.MarkPrice.String()),
			sdk.NewAttribute("index_price", priceInfo.IndexPrice.String()),
			sdk.NewAttribute("positions_affected", fmt.Sprintf("%d", affectedPositions)),
		),
	)

	logger.Info("funding settled",
		"market_id", marketID,
		"rate", rate.String(),
		"positions_affected", affectedPositions,
		"total_long_payment", totalLongPayment.String(),
		"total_short_payment", totalShortPayment.String(),
		"next_funding", nextTime.String(),
	)

	return nil
}

// FundingEndBlocker checks and settles funding for all markets
func (k *Keeper) FundingEndBlocker(ctx sdk.Context) {
	markets := k.ListActiveMarkets(ctx)
	currentTime := ctx.BlockTime()

	for _, market := range markets {
		nextFundingTime := k.GetNextFundingTime(ctx, market.MarketID)
		if nextFundingTime.IsZero() {
			nextFundingTime = nextFundingTimeUTC(currentTime)
			k.SetNextFundingTime(ctx, market.MarketID, nextFundingTime)
		}

		// Check if funding is due
		if currentTime.After(nextFundingTime) || currentTime.Equal(nextFundingTime) {
			// Set market status to settling
			market.Status = types.MarketStatusSettling
			k.SetMarket(ctx, market)

			// Settle funding
			if err := k.SettleFunding(ctx, market.MarketID); err != nil {
				k.Logger().Error("failed to settle funding",
					"market_id", market.MarketID,
					"error", err,
				)
			}

			// Restore market status to active
			market.Status = types.MarketStatusActive
			k.SetMarket(ctx, market)
		}
	}
}

// ============ Funding Info Query ============

// GetFundingInfo returns current funding information for a market
func (k *Keeper) GetFundingInfo(ctx sdk.Context, marketID string) *types.FundingInfo {
	market := k.GetMarket(ctx, marketID)
	if market == nil {
		return nil
	}

	priceInfo := k.GetPrice(ctx, marketID)
	positions := k.GetPositionsByMarket(ctx, marketID)

	info := &types.FundingInfo{
		MarketID:       marketID,
		CurrentRate:    k.CalculateFundingRate(ctx, marketID),
		NextSettlement: k.GetNextFundingTime(ctx, marketID),
		TotalLongSize:  math.LegacyZeroDec(),
		TotalShortSize: math.LegacyZeroDec(),
	}

	for _, pos := range positions {
		if pos.Side == types.PositionSideLong {
			info.TotalLongSize = info.TotalLongSize.Add(pos.Size)
		} else {
			info.TotalShortSize = info.TotalShortSize.Add(pos.Size)
		}
	}

	// Calculate predicted payment for 1 unit position
	if priceInfo != nil {
		info.PredictedPayment = priceInfo.MarkPrice.Mul(info.CurrentRate)
	}

	// Get last settlement time
	history := k.GetFundingRateHistory(ctx, marketID, 1)
	if len(history) > 0 {
		info.LastSettlement = history[0].Timestamp
	}

	return info
}
