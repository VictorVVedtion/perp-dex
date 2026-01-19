package keeper

import (
	"encoding/binary"
	"encoding/json"
	"fmt"

	"cosmossdk.io/log"
	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/openalpha/perp-dex/x/clearinghouse/types"
	perpetualtypes "github.com/openalpha/perp-dex/x/perpetual/types"
)

// Store key prefixes
var (
	LiquidationKeyPrefix  = []byte{0x01}
	LiquidationCounterKey = []byte{0x02}
)

// PerpetualKeeper defines the expected interface for the perpetual module
type PerpetualKeeper interface {
	GetAllPositions(ctx sdk.Context) []*perpetualtypes.Position
	GetPosition(ctx sdk.Context, trader, marketID string) *perpetualtypes.Position
	GetPositionsByTrader(ctx sdk.Context, trader string) []*perpetualtypes.Position
	GetPrice(ctx sdk.Context, marketID string) *perpetualtypes.PriceInfo
	GetAccount(ctx sdk.Context, trader string) *perpetualtypes.Account
	GetOrCreateAccount(ctx sdk.Context, trader string) *perpetualtypes.Account
	SetAccount(ctx sdk.Context, account *perpetualtypes.Account)
	SetPosition(ctx sdk.Context, position *perpetualtypes.Position)
	DeletePosition(ctx sdk.Context, trader, marketID string)
}

// OrderbookKeeper defines the expected interface for the orderbook module
type OrderbookKeeper interface {
	// PlaceMarketOrder places a market order for liquidation
	PlaceMarketOrder(ctx sdk.Context, trader, marketID string, side int, quantity math.LegacyDec) error
}

// Keeper manages the clearinghouse module state
type Keeper struct {
	cdc             codec.BinaryCodec
	storeKey        storetypes.StoreKey
	perpetualKeeper PerpetualKeeper
	orderbookKeeper OrderbookKeeper
	logger          log.Logger
}

// NewKeeper creates a new clearinghouse keeper
func NewKeeper(
	cdc codec.BinaryCodec,
	storeKey storetypes.StoreKey,
	perpetualKeeper PerpetualKeeper,
	orderbookKeeper OrderbookKeeper,
	logger log.Logger,
) *Keeper {
	return &Keeper{
		cdc:             cdc,
		storeKey:        storeKey,
		perpetualKeeper: perpetualKeeper,
		orderbookKeeper: orderbookKeeper,
		logger:          logger.With("module", "x/clearinghouse"),
	}
}

// Logger returns the module logger
func (k *Keeper) Logger() log.Logger {
	return k.logger
}

// GetStore returns the KVStore
func (k *Keeper) GetStore(ctx sdk.Context) storetypes.KVStore {
	return ctx.KVStore(k.storeKey)
}

// ============ Liquidation Store Operations ============

// SetLiquidation saves a liquidation to the store
func (k *Keeper) SetLiquidation(ctx sdk.Context, liquidation *types.Liquidation) {
	store := k.GetStore(ctx)
	key := append(LiquidationKeyPrefix, []byte(liquidation.LiquidationID)...)
	bz, _ := json.Marshal(liquidation)
	store.Set(key, bz)
}

// GetLiquidation retrieves a liquidation from the store
func (k *Keeper) GetLiquidation(ctx sdk.Context, liquidationID string) *types.Liquidation {
	store := k.GetStore(ctx)
	key := append(LiquidationKeyPrefix, []byte(liquidationID)...)
	bz := store.Get(key)
	if bz == nil {
		return nil
	}
	var liquidation types.Liquidation
	if err := json.Unmarshal(bz, &liquidation); err != nil {
		return nil
	}
	return &liquidation
}

// GetAllLiquidations returns all liquidations
func (k *Keeper) GetAllLiquidations(ctx sdk.Context, limit int) []*types.Liquidation {
	store := k.GetStore(ctx)
	iterator := storetypes.KVStoreReversePrefixIterator(store, LiquidationKeyPrefix)
	defer iterator.Close()

	var liquidations []*types.Liquidation
	count := 0
	for ; iterator.Valid() && count < limit; iterator.Next() {
		var liquidation types.Liquidation
		if err := json.Unmarshal(iterator.Value(), &liquidation); err != nil {
			continue
		}
		liquidations = append(liquidations, &liquidation)
		count++
	}
	return liquidations
}

// generateLiquidationID generates a unique liquidation ID
func (k *Keeper) generateLiquidationID(ctx sdk.Context) string {
	store := k.GetStore(ctx)
	bz := store.Get(LiquidationCounterKey)
	var counter uint64
	if bz != nil {
		counter = binary.BigEndian.Uint64(bz)
	}
	counter++

	newBz := make([]byte, 8)
	binary.BigEndian.PutUint64(newBz, counter)
	store.Set(LiquidationCounterKey, newBz)

	return fmt.Sprintf("liq-%d", counter)
}

// ============ Position Health Checks ============

// GetPositionHealth returns the health status of a position
func (k *Keeper) GetPositionHealth(ctx sdk.Context, trader, marketID string) *types.PositionHealth {
	position := k.perpetualKeeper.GetPosition(ctx, trader, marketID)
	if position == nil {
		return nil
	}

	priceInfo := k.perpetualKeeper.GetPrice(ctx, marketID)
	if priceInfo == nil {
		return nil
	}

	markPrice := priceInfo.MarkPrice
	maintenanceMarginRate := math.LegacyNewDecWithPrec(5, 2) // 5%
	maintenanceMargin := position.Size.Mul(markPrice).Mul(maintenanceMarginRate)

	// Calculate margin ratio
	unrealizedPnL := position.CalculateUnrealizedPnL(markPrice)
	equity := position.Margin.Add(unrealizedPnL)
	marginRatio := equity.Quo(position.Size.Mul(markPrice))

	isHealthy := marginRatio.GTE(maintenanceMarginRate)
	atRiskThreshold := maintenanceMarginRate.Mul(math.LegacyNewDecWithPrec(15, 1)) // 150% of maintenance
	atRisk := marginRatio.LT(atRiskThreshold)

	return &types.PositionHealth{
		Trader:            trader,
		MarketID:          marketID,
		MarginRatio:       marginRatio,
		MaintenanceMargin: maintenanceMargin,
		AccountEquity:     equity,
		IsHealthy:         isHealthy,
		AtRisk:            atRisk,
	}
}

// GetUnhealthyPositions returns all positions below maintenance margin
func (k *Keeper) GetUnhealthyPositions(ctx sdk.Context) []*types.PositionHealth {
	// Safety check for nil perpetual keeper (MVP simplified setup)
	if k.perpetualKeeper == nil {
		return nil
	}
	positions := k.perpetualKeeper.GetAllPositions(ctx)
	var unhealthy []*types.PositionHealth

	for _, position := range positions {
		health := k.GetPositionHealth(ctx, position.Trader, position.MarketID)
		if health != nil && !health.IsHealthy {
			unhealthy = append(unhealthy, health)
		}
	}

	return unhealthy
}

// GetAtRiskPositions returns positions close to liquidation
func (k *Keeper) GetAtRiskPositions(ctx sdk.Context, threshold math.LegacyDec) []*types.PositionHealth {
	// Safety check for nil perpetual keeper (MVP simplified setup)
	if k.perpetualKeeper == nil {
		return nil
	}
	positions := k.perpetualKeeper.GetAllPositions(ctx)
	var atRisk []*types.PositionHealth

	for _, position := range positions {
		health := k.GetPositionHealth(ctx, position.Trader, position.MarketID)
		if health != nil && health.AtRisk {
			atRisk = append(atRisk, health)
		}
	}

	return atRisk
}
