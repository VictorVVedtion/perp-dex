package keeper

import (
	"encoding/json"
	"fmt"
	"time"

	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Store key prefixes for K-line data
var (
	KlineKeyPrefix = []byte{0x40}
)

// KlineInterval represents K-line time intervals
type KlineInterval string

const (
	Kline1m  KlineInterval = "1m"
	Kline5m  KlineInterval = "5m"
	Kline15m KlineInterval = "15m"
	Kline30m KlineInterval = "30m"
	Kline1h  KlineInterval = "1h"
	Kline4h  KlineInterval = "4h"
	Kline1d  KlineInterval = "1d"
)

// IntervalDuration returns the duration for each interval
func (i KlineInterval) Duration() time.Duration {
	switch i {
	case Kline1m:
		return time.Minute
	case Kline5m:
		return 5 * time.Minute
	case Kline15m:
		return 15 * time.Minute
	case Kline30m:
		return 30 * time.Minute
	case Kline1h:
		return time.Hour
	case Kline4h:
		return 4 * time.Hour
	case Kline1d:
		return 24 * time.Hour
	default:
		return time.Minute
	}
}

// Kline represents a single K-line (candlestick) data point
type Kline struct {
	MarketID  string         `json:"market_id"`
	Interval  KlineInterval  `json:"interval"`
	Timestamp int64          `json:"timestamp"` // Start of the candle (Unix seconds)
	Open      math.LegacyDec `json:"open"`
	High      math.LegacyDec `json:"high"`
	Low       math.LegacyDec `json:"low"`
	Close     math.LegacyDec `json:"close"`
	Volume    math.LegacyDec `json:"volume"`    // Volume in base asset
	Turnover  math.LegacyDec `json:"turnover"`  // Turnover in quote asset
	TradeCount int64         `json:"trade_count"`
}

// NewKline creates a new K-line with initial trade
func NewKline(marketID string, interval KlineInterval, timestamp int64, price, volume math.LegacyDec) *Kline {
	return &Kline{
		MarketID:   marketID,
		Interval:   interval,
		Timestamp:  timestamp,
		Open:       price,
		High:       price,
		Low:        price,
		Close:      price,
		Volume:     volume,
		Turnover:   price.Mul(volume),
		TradeCount: 1,
	}
}

// Update updates the K-line with a new trade
func (k *Kline) Update(price, volume math.LegacyDec) {
	if price.GT(k.High) {
		k.High = price
	}
	if price.LT(k.Low) {
		k.Low = price
	}
	k.Close = price
	k.Volume = k.Volume.Add(volume)
	k.Turnover = k.Turnover.Add(price.Mul(volume))
	k.TradeCount++
}

// ToLightweightCharts returns data in lightweight-charts format
func (k *Kline) ToLightweightCharts() map[string]interface{} {
	return map[string]interface{}{
		"time":  k.Timestamp,
		"open":  k.Open.MustFloat64(),
		"high":  k.High.MustFloat64(),
		"low":   k.Low.MustFloat64(),
		"close": k.Close.MustFloat64(),
	}
}

// ============ K-line Storage ============

// klineKey generates a storage key for a K-line
func klineKey(marketID string, interval KlineInterval, timestamp int64) []byte {
	return append(KlineKeyPrefix, []byte(fmt.Sprintf("%s:%s:%d", marketID, interval, timestamp))...)
}

// SetKline saves a K-line
func (k *Keeper) SetKline(ctx sdk.Context, kline *Kline) {
	store := k.GetStore(ctx)
	key := klineKey(kline.MarketID, kline.Interval, kline.Timestamp)
	bz, _ := json.Marshal(kline)
	store.Set(key, bz)
}

// GetKline retrieves a K-line
func (k *Keeper) GetKline(ctx sdk.Context, marketID string, interval KlineInterval, timestamp int64) *Kline {
	store := k.GetStore(ctx)
	key := klineKey(marketID, interval, timestamp)
	bz := store.Get(key)
	if bz == nil {
		return nil
	}

	var kline Kline
	if err := json.Unmarshal(bz, &kline); err != nil {
		return nil
	}
	return &kline
}

// GetKlines retrieves K-lines for a market within a time range
func (k *Keeper) GetKlines(ctx sdk.Context, marketID string, interval KlineInterval, from, to int64, limit int) []*Kline {
	store := k.GetStore(ctx)
	prefix := append(KlineKeyPrefix, []byte(fmt.Sprintf("%s:%s:", marketID, interval))...)

	iterator := storetypes.KVStoreReversePrefixIterator(store, prefix)
	defer iterator.Close()

	var klines []*Kline
	count := 0

	for ; iterator.Valid() && count < limit; iterator.Next() {
		var kline Kline
		if err := json.Unmarshal(iterator.Value(), &kline); err != nil {
			continue
		}

		// Check time range
		if kline.Timestamp >= from && kline.Timestamp <= to {
			klines = append(klines, &kline)
			count++
		}
	}

	// Reverse to get chronological order
	for i, j := 0, len(klines)-1; i < j; i, j = i+1, j-1 {
		klines[i], klines[j] = klines[j], klines[i]
	}

	return klines
}

// GetLatestKlines retrieves the most recent K-lines
func (k *Keeper) GetLatestKlines(ctx sdk.Context, marketID string, interval KlineInterval, limit int) []*Kline {
	now := ctx.BlockTime().Unix()
	from := now - (int64(limit) * int64(interval.Duration().Seconds()))
	return k.GetKlines(ctx, marketID, interval, from, now, limit)
}

// ============ K-line Updates ============

// getKlineTimestamp returns the start timestamp for a K-line given a trade time
func getKlineTimestamp(tradeTime time.Time, interval KlineInterval) int64 {
	duration := interval.Duration()
	return tradeTime.Truncate(duration).Unix()
}

// UpdateKline updates K-line data with a new trade
func (k *Keeper) UpdateKline(ctx sdk.Context, marketID string, price, volume math.LegacyDec) {
	tradeTime := ctx.BlockTime()

	// Update all intervals
	intervals := []KlineInterval{Kline1m, Kline5m, Kline15m, Kline30m, Kline1h, Kline4h, Kline1d}

	for _, interval := range intervals {
		timestamp := getKlineTimestamp(tradeTime, interval)
		kline := k.GetKline(ctx, marketID, interval, timestamp)

		if kline == nil {
			// Create new K-line
			kline = NewKline(marketID, interval, timestamp, price, volume)
		} else {
			// Update existing K-line
			kline.Update(price, volume)
		}

		k.SetKline(ctx, kline)
	}
}

// AggregateKlines aggregates lower interval K-lines to higher intervals
// This is called periodically to ensure data consistency
func (k *Keeper) AggregateKlines(ctx sdk.Context, marketID string) {
	// Aggregate 1m -> 5m
	k.aggregateInterval(ctx, marketID, Kline1m, Kline5m, 5)
	// Aggregate 5m -> 15m
	k.aggregateInterval(ctx, marketID, Kline5m, Kline15m, 3)
	// Aggregate 15m -> 30m
	k.aggregateInterval(ctx, marketID, Kline15m, Kline30m, 2)
	// Aggregate 30m -> 1h
	k.aggregateInterval(ctx, marketID, Kline30m, Kline1h, 2)
	// Aggregate 1h -> 4h
	k.aggregateInterval(ctx, marketID, Kline1h, Kline4h, 4)
	// Aggregate 4h -> 1d
	k.aggregateInterval(ctx, marketID, Kline4h, Kline1d, 6)
}

// aggregateInterval aggregates K-lines from source interval to target interval
func (k *Keeper) aggregateInterval(ctx sdk.Context, marketID string, source, target KlineInterval, count int) {
	now := ctx.BlockTime()
	targetTimestamp := getKlineTimestamp(now, target)

	// Get source K-lines for this target period
	sourceTimestamp := targetTimestamp
	var sourceKlines []*Kline

	for i := 0; i < count; i++ {
		kline := k.GetKline(ctx, marketID, source, sourceTimestamp)
		if kline != nil {
			sourceKlines = append(sourceKlines, kline)
		}
		sourceTimestamp += int64(source.Duration().Seconds())
	}

	if len(sourceKlines) == 0 {
		return
	}

	// Aggregate
	aggregated := &Kline{
		MarketID:   marketID,
		Interval:   target,
		Timestamp:  targetTimestamp,
		Open:       sourceKlines[0].Open,
		High:       sourceKlines[0].High,
		Low:        sourceKlines[0].Low,
		Close:      sourceKlines[len(sourceKlines)-1].Close,
		Volume:     math.LegacyZeroDec(),
		Turnover:   math.LegacyZeroDec(),
		TradeCount: 0,
	}

	for _, kline := range sourceKlines {
		if kline.High.GT(aggregated.High) {
			aggregated.High = kline.High
		}
		if kline.Low.LT(aggregated.Low) {
			aggregated.Low = kline.Low
		}
		aggregated.Volume = aggregated.Volume.Add(kline.Volume)
		aggregated.Turnover = aggregated.Turnover.Add(kline.Turnover)
		aggregated.TradeCount += kline.TradeCount
	}

	k.SetKline(ctx, aggregated)
}

// KlineEndBlocker updates K-lines at end of block
// Called after all trades in a block have been processed
func (k *Keeper) KlineEndBlocker(ctx sdk.Context) {
	markets := k.ListActiveMarkets(ctx)

	for _, market := range markets {
		k.AggregateKlines(ctx, market.MarketID)
	}
}
