package keeper

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"sort"
	"time"

	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/openalpha/perp-dex/x/perpetual/types"
)

// Store key prefixes for oracle
var (
	OracleSourceKeyPrefix   = []byte{0x40}
	OraclePriceKeyPrefix    = []byte{0x41}
	OracleConfigKeyPrefix   = []byte{0x42}
	OracleEMAKeyPrefix      = []byte{0x43}
	OracleHistoryKeyPrefix  = []byte{0x44}
)

// ============ Oracle Source Types ============

// OracleSource represents a price data source
type OracleSource struct {
	SourceID   string         // e.g., "binance", "okx", "coinbase"
	Weight     int            // Weight for weighted average (e.g., 3 for Binance, 2 for OKX)
	IsActive   bool           // Whether source is currently active
	LastUpdate time.Time      // Last successful update
	LastPrice  math.LegacyDec // Last reported price
	Reliability float64       // Historical reliability score (0-1)
}

// OracleSourcePrice represents a price submission from a source
type OracleSourcePrice struct {
	SourceID  string
	MarketID  string
	Price     math.LegacyDec
	Timestamp time.Time
}

// weightedPrice represents a price with its source weight for aggregation
type weightedPrice struct {
	price  math.LegacyDec
	weight int
}

// OracleConfig contains oracle configuration
type OracleConfig struct {
	// Price aggregation
	MinSources         int            // Minimum sources required for valid price
	MaxPriceAge        time.Duration  // Maximum age of price before stale
	MaxDeviation       math.LegacyDec // Maximum deviation from median (e.g., 0.02 = 2%)

	// EMA configuration
	EMAAlpha           math.LegacyDec // EMA smoothing factor (0 < alpha < 1)
	EMAPeriodBlocks    int64          // Number of blocks for EMA period

	// Price protection
	MaxPriceChange     math.LegacyDec // Max single-block price change (e.g., 0.05 = 5%)
	CircuitBreakerPct  math.LegacyDec // Circuit breaker threshold (e.g., 0.10 = 10%)

	// Source weights
	SourceWeights      map[string]int // Weight per source
}

// DefaultOracleConfig returns default oracle configuration
func DefaultOracleConfig() OracleConfig {
	return OracleConfig{
		MinSources:        2,
		MaxPriceAge:       time.Minute * 5,
		MaxDeviation:      math.LegacyNewDecWithPrec(2, 2),  // 2%
		EMAAlpha:          math.LegacyNewDecWithPrec(1, 1),  // 0.1
		EMAPeriodBlocks:   100,
		MaxPriceChange:    math.LegacyNewDecWithPrec(5, 2),  // 5%
		CircuitBreakerPct: math.LegacyNewDecWithPrec(10, 2), // 10%
		SourceWeights: map[string]int{
			"binance":  3,
			"okx":      2,
			"coinbase": 2,
			"kraken":   1,
			"bybit":    1,
		},
	}
}

// EMAPrice stores the Exponential Moving Average price
type EMAPrice struct {
	MarketID    string
	EMAValue    math.LegacyDec
	LastUpdated time.Time
	BlockHeight int64
}

// PriceHistory stores historical prices for analysis
type PriceHistory struct {
	MarketID  string
	Prices    []math.LegacyDec
	Timestamps []time.Time
	MaxLength int
}

// ============ Oracle Configuration Storage ============

// SetOracleConfig saves oracle configuration
func (k *Keeper) SetOracleConfig(ctx sdk.Context, config OracleConfig) {
	store := k.GetStore(ctx)
	bz, _ := json.Marshal(config)
	store.Set(OracleConfigKeyPrefix, bz)
}

// GetOracleConfig retrieves oracle configuration
func (k *Keeper) GetOracleConfig(ctx sdk.Context) OracleConfig {
	store := k.GetStore(ctx)
	bz := store.Get(OracleConfigKeyPrefix)
	if bz == nil {
		return DefaultOracleConfig()
	}
	var config OracleConfig
	if err := json.Unmarshal(bz, &config); err != nil {
		return DefaultOracleConfig()
	}
	return config
}

// ============ Oracle Source Management ============

// SetOracleSource saves an oracle source
func (k *Keeper) SetOracleSource(ctx sdk.Context, source *OracleSource) {
	store := k.GetStore(ctx)
	key := append(OracleSourceKeyPrefix, []byte(source.SourceID)...)
	bz, _ := json.Marshal(source)
	store.Set(key, bz)
}

// GetOracleSource retrieves an oracle source
func (k *Keeper) GetOracleSource(ctx sdk.Context, sourceID string) *OracleSource {
	store := k.GetStore(ctx)
	key := append(OracleSourceKeyPrefix, []byte(sourceID)...)
	bz := store.Get(key)
	if bz == nil {
		return nil
	}
	var source OracleSource
	if err := json.Unmarshal(bz, &source); err != nil {
		return nil
	}
	return &source
}

// GetAllOracleSources retrieves all oracle sources
func (k *Keeper) GetAllOracleSources(ctx sdk.Context) []*OracleSource {
	store := k.GetStore(ctx)
	iterator := storetypes.KVStorePrefixIterator(store, OracleSourceKeyPrefix)
	defer iterator.Close()

	var sources []*OracleSource
	for ; iterator.Valid(); iterator.Next() {
		var source OracleSource
		if err := json.Unmarshal(iterator.Value(), &source); err != nil {
			continue
		}
		sources = append(sources, &source)
	}
	return sources
}

// InitDefaultOracleSources initializes default oracle sources
func (k *Keeper) InitDefaultOracleSources(ctx sdk.Context) {
	config := k.GetOracleConfig(ctx)

	for sourceID, weight := range config.SourceWeights {
		source := &OracleSource{
			SourceID:    sourceID,
			Weight:      weight,
			IsActive:    true,
			LastUpdate:  time.Time{},
			LastPrice:   math.LegacyZeroDec(),
			Reliability: 1.0,
		}
		k.SetOracleSource(ctx, source)
	}
}

// ============ Price Submission and Aggregation ============

// SubmitSourcePrice submits a price from an oracle source
func (k *Keeper) SubmitSourcePrice(ctx sdk.Context, sourceID, marketID string, price math.LegacyDec) error {
	source := k.GetOracleSource(ctx, sourceID)
	if source == nil {
		return fmt.Errorf("oracle source not found: %s", sourceID)
	}

	if !source.IsActive {
		return fmt.Errorf("oracle source is inactive: %s", sourceID)
	}

	config := k.GetOracleConfig(ctx)

	// Validate price deviation from current price
	currentPrice := k.GetPrice(ctx, marketID)
	if currentPrice != nil && currentPrice.MarkPrice.IsPositive() {
		deviation := price.Sub(currentPrice.MarkPrice).Abs().Quo(currentPrice.MarkPrice)
		if deviation.GT(config.CircuitBreakerPct) {
			k.Logger().Warn("price submission rejected: exceeds circuit breaker",
				"source", sourceID,
				"market", marketID,
				"submitted_price", price.String(),
				"current_price", currentPrice.MarkPrice.String(),
				"deviation", deviation.String(),
			)
			// CRITICAL FIX: Use String() instead of MustFloat64() to avoid potential panic
			return fmt.Errorf("price deviation %s%% exceeds circuit breaker %s%%",
				deviation.MulInt64(100).String(),
				config.CircuitBreakerPct.MulInt64(100).String())
		}
	}

	// Update source
	source.LastPrice = price
	source.LastUpdate = ctx.BlockTime()
	k.SetOracleSource(ctx, source)

	// Store source price for this market
	k.storeSourcePrice(ctx, sourceID, marketID, price)

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"oracle_price_submitted",
			sdk.NewAttribute("source", sourceID),
			sdk.NewAttribute("market_id", marketID),
			sdk.NewAttribute("price", price.String()),
		),
	)

	return nil
}

// storeSourcePrice stores a source price for aggregation
func (k *Keeper) storeSourcePrice(ctx sdk.Context, sourceID, marketID string, price math.LegacyDec) {
	store := k.GetStore(ctx)
	priceData := &OracleSourcePrice{
		SourceID:  sourceID,
		MarketID:  marketID,
		Price:     price,
		Timestamp: ctx.BlockTime(),
	}
	key := append(OraclePriceKeyPrefix, []byte(sourceID+":"+marketID)...)
	bz, _ := json.Marshal(priceData)
	store.Set(key, bz)
}

// getSourcePrice retrieves a stored source price
func (k *Keeper) getSourcePrice(ctx sdk.Context, sourceID, marketID string) *OracleSourcePrice {
	store := k.GetStore(ctx)
	key := append(OraclePriceKeyPrefix, []byte(sourceID+":"+marketID)...)
	bz := store.Get(key)
	if bz == nil {
		return nil
	}
	var priceData OracleSourcePrice
	if err := json.Unmarshal(bz, &priceData); err != nil {
		return nil
	}
	return &priceData
}

// AggregatePrice aggregates prices from multiple sources using weighted median
// CRITICAL FIX: Added time-based weight decay to prevent stale price manipulation
func (k *Keeper) AggregatePrice(ctx sdk.Context, marketID string) (math.LegacyDec, error) {
	config := k.GetOracleConfig(ctx)
	sources := k.GetAllOracleSources(ctx)

	var validPrices []weightedPrice
	now := ctx.BlockTime()

	for _, source := range sources {
		if !source.IsActive {
			continue
		}

		priceData := k.getSourcePrice(ctx, source.SourceID, marketID)
		if priceData == nil {
			continue
		}

		// Check if price is stale
		age := now.Sub(priceData.Timestamp)
		if age > config.MaxPriceAge {
			k.Logger().Debug("skipping stale price",
				"source", source.SourceID,
				"age", age.String(),
			)
			continue
		}

		// CRITICAL FIX: Apply time-based weight decay
		// Newer prices get higher weight, older prices get lower weight
		// Formula: adjustedWeight = sourceWeight * (1 - age/maxAge)^2
		// Using quadratic decay for stronger recency preference
		maxAgeSeconds := float64(config.MaxPriceAge.Seconds())
		ageSeconds := float64(age.Seconds())
		if maxAgeSeconds > 0 {
			// Time decay factor: 1.0 for fresh prices, approaches 0 for oldest valid prices
			// Using quadratic decay: (1 - age/maxAge)^2
			timeFactor := 1.0 - (ageSeconds / maxAgeSeconds)
			timeWeight := timeFactor * timeFactor // Quadratic decay

			// Ensure minimum weight of 10% to not completely ignore valid prices
			if timeWeight < 0.1 {
				timeWeight = 0.1
			}

			// Apply time weight to source weight
			adjustedWeight := int(float64(source.Weight) * timeWeight * 10) // Scale by 10 to preserve precision
			if adjustedWeight < 1 {
				adjustedWeight = 1
			}

			validPrices = append(validPrices, weightedPrice{
				price:  priceData.Price,
				weight: adjustedWeight,
			})
		} else {
			// Fallback: no time decay if maxAgeSeconds is 0
			validPrices = append(validPrices, weightedPrice{
				price:  priceData.Price,
				weight: source.Weight,
			})
		}
	}

	if len(validPrices) < config.MinSources {
		return math.LegacyZeroDec(), fmt.Errorf("insufficient price sources: %d < %d required",
			len(validPrices), config.MinSources)
	}

	// Calculate weighted median
	return k.calculateWeightedMedian(validPrices, config.MaxDeviation)
}

// calculateWeightedMedian calculates the weighted median price with outlier filtering
func (k *Keeper) calculateWeightedMedian(prices []weightedPrice, maxDeviation math.LegacyDec) (math.LegacyDec, error) {
	if len(prices) == 0 {
		return math.LegacyZeroDec(), fmt.Errorf("no prices to aggregate")
	}

	// Sort prices by value
	sort.Slice(prices, func(i, j int) bool {
		return prices[i].price.LT(prices[j].price)
	})

	// Calculate simple median first for outlier detection
	medianIdx := len(prices) / 2
	simpleMedian := prices[medianIdx].price

	// Filter outliers
	var filteredPrices []weightedPrice
	for _, wp := range prices {
		deviation := wp.price.Sub(simpleMedian).Abs().Quo(simpleMedian)
		if deviation.LTE(maxDeviation) {
			filteredPrices = append(filteredPrices, wp)
		}
	}

	if len(filteredPrices) == 0 {
		return math.LegacyZeroDec(), fmt.Errorf("all prices filtered as outliers")
	}

	// Calculate weighted average of filtered prices
	totalWeight := 0
	weightedSum := math.LegacyZeroDec()

	for _, wp := range filteredPrices {
		weightedSum = weightedSum.Add(wp.price.MulInt64(int64(wp.weight)))
		totalWeight += wp.weight
	}

	if totalWeight == 0 {
		return math.LegacyZeroDec(), fmt.Errorf("total weight is zero")
	}

	return weightedSum.QuoInt64(int64(totalWeight)), nil
}

// ============ EMA Price Calculation ============

// GetEMAPrice retrieves the EMA price for a market
func (k *Keeper) GetEMAPrice(ctx sdk.Context, marketID string) *EMAPrice {
	store := k.GetStore(ctx)
	key := append(OracleEMAKeyPrefix, []byte(marketID)...)
	bz := store.Get(key)
	if bz == nil {
		return nil
	}
	var ema EMAPrice
	if err := json.Unmarshal(bz, &ema); err != nil {
		return nil
	}
	return &ema
}

// SetEMAPrice saves the EMA price for a market
func (k *Keeper) SetEMAPrice(ctx sdk.Context, ema *EMAPrice) {
	store := k.GetStore(ctx)
	key := append(OracleEMAKeyPrefix, []byte(ema.MarketID)...)
	bz, _ := json.Marshal(ema)
	store.Set(key, bz)
}

// UpdateEMAPrice updates the EMA price based on new index price
// EMA = alpha * currentPrice + (1 - alpha) * previousEMA
func (k *Keeper) UpdateEMAPrice(ctx sdk.Context, marketID string, currentPrice math.LegacyDec) math.LegacyDec {
	config := k.GetOracleConfig(ctx)
	ema := k.GetEMAPrice(ctx, marketID)

	if ema == nil {
		// Initialize EMA with current price
		ema = &EMAPrice{
			MarketID:    marketID,
			EMAValue:    currentPrice,
			LastUpdated: ctx.BlockTime(),
			BlockHeight: ctx.BlockHeight(),
		}
		k.SetEMAPrice(ctx, ema)
		return currentPrice
	}

	// Calculate new EMA: alpha * current + (1-alpha) * previous
	alpha := config.EMAAlpha
	oneMinusAlpha := math.LegacyOneDec().Sub(alpha)

	newEMA := alpha.Mul(currentPrice).Add(oneMinusAlpha.Mul(ema.EMAValue))

	ema.EMAValue = newEMA
	ema.LastUpdated = ctx.BlockTime()
	ema.BlockHeight = ctx.BlockHeight()
	k.SetEMAPrice(ctx, ema)

	return newEMA
}

// CalculateMarkPrice calculates the mark price using EMA of index price
func (k *Keeper) CalculateMarkPrice(ctx sdk.Context, marketID string, indexPrice math.LegacyDec) math.LegacyDec {
	// Mark Price = EMA(Index Price)
	// This smooths out short-term volatility
	return k.UpdateEMAPrice(ctx, marketID, indexPrice)
}

// ============ Price Update with Protection ============

// UpdatePriceWithProtection updates price with deviation protection
func (k *Keeper) UpdatePriceWithProtection(ctx sdk.Context, marketID string) (*types.PriceInfo, error) {
	config := k.GetOracleConfig(ctx)

	// Aggregate index price from sources
	indexPrice, err := k.AggregatePrice(ctx, marketID)
	if err != nil {
		return nil, fmt.Errorf("failed to aggregate price: %w", err)
	}

	// Get current price for comparison
	currentPrice := k.GetPrice(ctx, marketID)

	// Apply price change protection
	if currentPrice != nil && currentPrice.IndexPrice.IsPositive() {
		priceChange := indexPrice.Sub(currentPrice.IndexPrice).Abs().Quo(currentPrice.IndexPrice)

		if priceChange.GT(config.MaxPriceChange) {
			// Limit price change to max allowed
			direction := math.LegacyOneDec()
			if indexPrice.LT(currentPrice.IndexPrice) {
				direction = direction.Neg()
			}

			maxChange := currentPrice.IndexPrice.Mul(config.MaxPriceChange).Mul(direction)
			indexPrice = currentPrice.IndexPrice.Add(maxChange)

			k.Logger().Warn("price change limited",
				"market", marketID,
				"original_change", priceChange.String(),
				"max_allowed", config.MaxPriceChange.String(),
				"adjusted_price", indexPrice.String(),
			)

			ctx.EventManager().EmitEvent(
				sdk.NewEvent(
					"price_change_limited",
					sdk.NewAttribute("market_id", marketID),
					sdk.NewAttribute("original_price", indexPrice.String()),
					sdk.NewAttribute("limited_price", indexPrice.String()),
				),
			)
		}
	}

	// Calculate mark price using EMA
	markPrice := k.CalculateMarkPrice(ctx, marketID, indexPrice)

	// Create and save price info
	priceInfo := &types.PriceInfo{
		MarketID:   marketID,
		MarkPrice:  markPrice,
		IndexPrice: indexPrice,
		LastPrice:  indexPrice, // Updated from trades
		Timestamp:  ctx.BlockTime(),
	}
	k.SetPrice(ctx, priceInfo)

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"price_updated",
			sdk.NewAttribute("market_id", marketID),
			sdk.NewAttribute("mark_price", markPrice.String()),
			sdk.NewAttribute("index_price", indexPrice.String()),
		),
	)

	return priceInfo, nil
}

// ============ Legacy OracleSimulator (for testing) ============

// OracleSimulator simulates price updates for testing
type OracleSimulator struct {
	keeper *Keeper
	rng    *rand.Rand
}

// NewOracleSimulator creates a new oracle simulator
func NewOracleSimulator(keeper *Keeper) *OracleSimulator {
	return &OracleSimulator{
		keeper: keeper,
		rng:    rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// SimulatePriceUpdate simulates a random price movement
func (os *OracleSimulator) SimulatePriceUpdate(ctx sdk.Context, marketID string) *types.PriceInfo {
	priceInfo := os.keeper.GetPrice(ctx, marketID)
	if priceInfo == nil {
		// Get default price from market config
		market := os.keeper.GetMarket(ctx, marketID)
		defaultPrice := math.LegacyNewDec(50000) // BTC default
		if market != nil {
			switch marketID {
			case "ETH-USDC":
				defaultPrice = math.LegacyNewDec(3000)
			case "SOL-USDC":
				defaultPrice = math.LegacyNewDec(100)
			case "ARB-USDC":
				defaultPrice = math.LegacyNewDec(1)
			}
		}
		priceInfo = types.NewPriceInfo(marketID, defaultPrice)
		os.keeper.SetPrice(ctx, priceInfo)
		return priceInfo
	}

	// Generate random price change between -2% and +2%
	changePercent := os.randomPriceChange()
	multiplier := math.LegacyOneDec().Add(changePercent)
	newPrice := priceInfo.MarkPrice.Mul(multiplier)

	// Ensure price doesn't go below a minimum
	minPrice := math.LegacyNewDec(1)
	if newPrice.LT(minPrice) {
		newPrice = minPrice
	}

	// Update EMA for mark price
	markPrice := os.keeper.UpdateEMAPrice(ctx, marketID, newPrice)

	priceInfo.IndexPrice = newPrice
	priceInfo.MarkPrice = markPrice
	priceInfo.LastPrice = newPrice
	priceInfo.Timestamp = ctx.BlockTime()

	os.keeper.SetPrice(ctx, priceInfo)

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"price_update",
			sdk.NewAttribute("market_id", marketID),
			sdk.NewAttribute("mark_price", markPrice.String()),
			sdk.NewAttribute("index_price", newPrice.String()),
			sdk.NewAttribute("timestamp", priceInfo.Timestamp.String()),
		),
	)

	return priceInfo
}

// randomPriceChange generates a random price change between -2% and +2%
func (os *OracleSimulator) randomPriceChange() math.LegacyDec {
	basisPoints := os.rng.Intn(401) - 200
	return math.LegacyNewDecWithPrec(int64(basisPoints), 4)
}

// SetPrice manually sets the price
func (os *OracleSimulator) SetPrice(ctx sdk.Context, marketID string, price math.LegacyDec) {
	markPrice := os.keeper.UpdateEMAPrice(ctx, marketID, price)
	priceInfo := &types.PriceInfo{
		MarketID:   marketID,
		MarkPrice:  markPrice,
		IndexPrice: price,
		LastPrice:  price,
		Timestamp:  ctx.BlockTime(),
	}
	os.keeper.SetPrice(ctx, priceInfo)
}

// UpdatePriceFromTrade updates the last traded price
func (os *OracleSimulator) UpdatePriceFromTrade(ctx sdk.Context, marketID string, tradePrice math.LegacyDec) {
	priceInfo := os.keeper.GetPrice(ctx, marketID)
	if priceInfo == nil {
		priceInfo = types.NewPriceInfo(marketID, tradePrice)
	} else {
		priceInfo.LastPrice = tradePrice
		priceInfo.Timestamp = ctx.BlockTime()
	}
	os.keeper.SetPrice(ctx, priceInfo)
}

// EndBlockPriceUpdate is called at the end of each block to update prices
func (os *OracleSimulator) EndBlockPriceUpdate(ctx sdk.Context) {
	markets := os.keeper.GetAllMarkets(ctx)
	for _, market := range markets {
		if market.IsActive {
			os.SimulatePriceUpdate(ctx, market.MarketID)
		}
	}
}

// SimulateMultiSourcePrices simulates prices from multiple sources (for testing)
func (os *OracleSimulator) SimulateMultiSourcePrices(ctx sdk.Context, marketID string) error {
	basePrice := os.keeper.GetPrice(ctx, marketID)
	if basePrice == nil {
		return fmt.Errorf("no base price for market: %s", marketID)
	}

	sources := os.keeper.GetAllOracleSources(ctx)
	for _, source := range sources {
		if !source.IsActive {
			continue
		}

		// Add small random variation (-0.5% to +0.5%) to simulate source differences
		variation := math.LegacyNewDecWithPrec(int64(os.rng.Intn(101)-50), 4)
		sourcePrice := basePrice.IndexPrice.Mul(math.LegacyOneDec().Add(variation))

		if err := os.keeper.SubmitSourcePrice(ctx, source.SourceID, marketID, sourcePrice); err != nil {
			os.keeper.Logger().Error("failed to submit simulated price",
				"source", source.SourceID,
				"error", err,
			)
		}
	}

	return nil
}
