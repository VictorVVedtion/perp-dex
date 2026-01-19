package keeper

import (
	"encoding/json"
	"time"

	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/openalpha/perp-dex/x/perpetual/types"
)

// CreateMarket creates a new market with the given configuration
func (k *Keeper) CreateMarket(ctx sdk.Context, config types.MarketConfig) error {
	// Check if market already exists
	if k.GetMarket(ctx, config.MarketID) != nil {
		return types.ErrMarketExists
	}

	// Validate market configuration
	if err := k.validateMarketConfig(config); err != nil {
		return err
	}

	// Create market
	market := types.NewMarketWithConfig(config)
	k.SetMarket(ctx, market)

	// Initialize price
	k.SetPrice(ctx, types.NewPriceInfo(config.MarketID, math.LegacyZeroDec()))

	// Set next funding time
	nextFundingTime := nextFundingTimeUTC(ctx.BlockTime())
	k.SetNextFundingTime(ctx, config.MarketID, nextFundingTime)

	// Emit event
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"market_created",
			sdk.NewAttribute("market_id", config.MarketID),
			sdk.NewAttribute("base_asset", config.BaseAsset),
			sdk.NewAttribute("quote_asset", config.QuoteAsset),
			sdk.NewAttribute("max_leverage", config.MaxLeverage.String()),
		),
	)

	k.Logger().Info("market created",
		"market_id", config.MarketID,
		"base_asset", config.BaseAsset,
		"quote_asset", config.QuoteAsset,
	)

	return nil
}

// validateMarketConfig validates a market configuration
func (k *Keeper) validateMarketConfig(config types.MarketConfig) error {
	if config.MarketID == "" {
		return types.ErrInvalidMarketID
	}
	if config.BaseAsset == "" {
		return types.ErrInvalidBaseAsset
	}
	if config.QuoteAsset == "" {
		return types.ErrInvalidQuoteAsset
	}
	if config.MaxLeverage.IsNil() || config.MaxLeverage.LTE(math.LegacyZeroDec()) {
		return types.ErrInvalidLeverage
	}
	return nil
}

// UpdateMarket updates an existing market's parameters
func (k *Keeper) UpdateMarket(ctx sdk.Context, marketID string, updates map[string]interface{}) error {
	market := k.GetMarket(ctx, marketID)
	if market == nil {
		return types.ErrMarketNotFound
	}

	// Apply updates
	if maxLeverage, ok := updates["max_leverage"].(math.LegacyDec); ok {
		market.MaxLeverage = maxLeverage
	}
	if takerFeeRate, ok := updates["taker_fee_rate"].(math.LegacyDec); ok {
		market.TakerFeeRate = takerFeeRate
	}
	if makerFeeRate, ok := updates["maker_fee_rate"].(math.LegacyDec); ok {
		market.MakerFeeRate = makerFeeRate
	}
	if minOrderSize, ok := updates["min_order_size"].(math.LegacyDec); ok {
		market.MinOrderSize = minOrderSize
	}
	if maxOrderSize, ok := updates["max_order_size"].(math.LegacyDec); ok {
		market.MaxOrderSize = maxOrderSize
	}
	if maxPositionSize, ok := updates["max_position_size"].(math.LegacyDec); ok {
		market.MaxPositionSize = maxPositionSize
	}

	market.UpdatedAt = ctx.BlockTime()
	k.SetMarket(ctx, market)

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"market_updated",
			sdk.NewAttribute("market_id", marketID),
		),
	)

	return nil
}

// SetMarketStatus sets the status of a market
func (k *Keeper) SetMarketStatus(ctx sdk.Context, marketID string, status types.MarketStatus) error {
	market := k.GetMarket(ctx, marketID)
	if market == nil {
		return types.ErrMarketNotFound
	}

	market.Status = status
	market.IsActive = status.IsActive()
	market.UpdatedAt = ctx.BlockTime()
	k.SetMarket(ctx, market)

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"market_status_changed",
			sdk.NewAttribute("market_id", marketID),
			sdk.NewAttribute("status", status.String()),
		),
	)

	return nil
}

// ListActiveMarkets returns all active markets
func (k *Keeper) ListActiveMarkets(ctx sdk.Context) []*types.Market {
	markets := k.GetAllMarkets(ctx)
	var active []*types.Market
	for _, m := range markets {
		if m.Status.IsActive() {
			active = append(active, m)
		}
	}
	return active
}

// GetPositionsByMarket returns all positions for a specific market
func (k *Keeper) GetPositionsByMarket(ctx sdk.Context, marketID string) []*types.Position {
	store := k.GetStore(ctx)
	iterator := storetypes.KVStorePrefixIterator(store, PositionKeyPrefix)
	defer iterator.Close()

	var positions []*types.Position
	for ; iterator.Valid(); iterator.Next() {
		var position types.Position
		if err := json.Unmarshal(iterator.Value(), &position); err != nil {
			continue
		}
		if position.MarketID == marketID {
			positions = append(positions, &position)
		}
	}
	return positions
}

// InitDefaultMarkets initializes all default markets
func (k *Keeper) InitDefaultMarkets(ctx sdk.Context) {
	configs := types.DefaultMarketConfigs()

	// Set initial prices for each market
	initialPrices := map[string]math.LegacyDec{
		"BTC-USDC": math.LegacyNewDec(50000),
		"ETH-USDC": math.LegacyNewDec(3000),
		"SOL-USDC": math.LegacyNewDec(100),
		"ARB-USDC": math.LegacyNewDecWithPrec(1, 0), // 1.0
	}

	for marketID, config := range configs {
		if k.GetMarket(ctx, marketID) != nil {
			continue // Skip if already exists
		}

		if err := k.CreateMarket(ctx, config); err != nil {
			k.Logger().Error("failed to create market", "market_id", marketID, "error", err)
			continue
		}

		// Set initial price
		if price, ok := initialPrices[marketID]; ok {
			k.SetPrice(ctx, types.NewPriceInfo(marketID, price))
		}
	}

	k.Logger().Info("default markets initialized", "count", len(configs))
}

// ValidateOrderSize validates order size against market limits
func (k *Keeper) ValidateOrderSize(ctx sdk.Context, marketID string, size math.LegacyDec) error {
	market := k.GetMarket(ctx, marketID)
	if market == nil {
		return types.ErrMarketNotFound
	}

	if size.LT(market.MinOrderSize) {
		return types.ErrOrderSizeTooSmall
	}
	if size.GT(market.MaxOrderSize) {
		return types.ErrOrderSizeTooLarge
	}

	return nil
}

// ValidatePositionSize validates position size against market limits
func (k *Keeper) ValidatePositionSize(ctx sdk.Context, trader, marketID string, additionalSize math.LegacyDec) error {
	market := k.GetMarket(ctx, marketID)
	if market == nil {
		return types.ErrMarketNotFound
	}

	position := k.GetPosition(ctx, trader, marketID)
	currentSize := math.LegacyZeroDec()
	if position != nil {
		currentSize = position.Size
	}

	newSize := currentSize.Add(additionalSize)
	if newSize.GT(market.MaxPositionSize) {
		return types.ErrPositionSizeTooLarge
	}

	return nil
}

// GetMarketStats returns statistics for a market
type MarketStats struct {
	MarketID        string
	TotalLongSize   math.LegacyDec
	TotalShortSize  math.LegacyDec
	OpenInterest    math.LegacyDec
	PositionCount   int
	LastPrice       math.LegacyDec
	MarkPrice       math.LegacyDec
	IndexPrice      math.LegacyDec
	FundingRate     math.LegacyDec
	NextFundingTime time.Time
}

func (k *Keeper) GetMarketStats(ctx sdk.Context, marketID string) *MarketStats {
	market := k.GetMarket(ctx, marketID)
	if market == nil {
		return nil
	}

	positions := k.GetPositionsByMarket(ctx, marketID)
	priceInfo := k.GetPrice(ctx, marketID)

	stats := &MarketStats{
		MarketID:       marketID,
		TotalLongSize:  math.LegacyZeroDec(),
		TotalShortSize: math.LegacyZeroDec(),
		PositionCount:  len(positions),
	}

	for _, pos := range positions {
		if pos.Side == types.PositionSideLong {
			stats.TotalLongSize = stats.TotalLongSize.Add(pos.Size)
		} else {
			stats.TotalShortSize = stats.TotalShortSize.Add(pos.Size)
		}
	}

	stats.OpenInterest = stats.TotalLongSize.Add(stats.TotalShortSize)

	if priceInfo != nil {
		stats.LastPrice = priceInfo.LastPrice
		stats.MarkPrice = priceInfo.MarkPrice
		stats.IndexPrice = priceInfo.IndexPrice
	}

	stats.FundingRate = k.CalculateFundingRate(ctx, marketID)
	stats.NextFundingTime = k.GetNextFundingTime(ctx, marketID)

	return stats
}
