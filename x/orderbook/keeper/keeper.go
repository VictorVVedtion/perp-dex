package keeper

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"

	"cosmossdk.io/log"
	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/openalpha/perp-dex/x/orderbook/types"
)

// Store key prefixes
var (
	OrderKeyPrefix     = []byte{0x01}
	OrderBookKeyPrefix = []byte{0x02}
	TradeKeyPrefix     = []byte{0x03}
	TradeCounterKey    = []byte{0x04}
	OrderCounterKey    = []byte{0x05}
)

// PerpetualKeeper defines the expected interface for the perpetual module
type PerpetualKeeper interface {
	GetMarket(ctx sdk.Context, marketID string) *Market
	GetMarkPrice(ctx sdk.Context, marketID string) (math.LegacyDec, bool)
	UpdatePosition(ctx sdk.Context, trader, marketID string, side types.Side, qty, price, fee interface{}) error
	CheckMarginRequirement(ctx sdk.Context, trader, marketID string, side types.Side, qty, price interface{}) error
}

// Market is a simplified market structure (will be replaced by perpetual types)
type Market struct {
	MarketID      string
	TakerFeeRate  math.LegacyDec
	MakerFeeRate  math.LegacyDec
	InitialMargin math.LegacyDec
}

// Keeper manages the orderbook state
type Keeper struct {
	cdc               codec.BinaryCodec
	storeKey          storetypes.StoreKey
	perpetualKeeper   PerpetualKeeper
	logger            log.Logger
	parallelConfig    ParallelConfig
	parallelMatcher   *ParallelMatcher
	parallelMatcherV2 *ParallelMatcherV2
}

// NewKeeper creates a new orderbook keeper
func NewKeeper(
	cdc codec.BinaryCodec,
	storeKey storetypes.StoreKey,
	perpetualKeeper PerpetualKeeper,
	logger log.Logger,
) *Keeper {
	k := &Keeper{
		cdc:             cdc,
		storeKey:        storeKey,
		perpetualKeeper: perpetualKeeper,
		logger:          logger.With("module", "x/orderbook"),
		parallelConfig:  DefaultParallelConfig(),
	}
	k.parallelMatcher = NewParallelMatcher(k, k.parallelConfig)
	k.parallelMatcherV2 = NewParallelMatcherV2(k, k.parallelConfig)
	return k
}

// NewKeeperWithConfig creates a new orderbook keeper with custom parallel config
func NewKeeperWithConfig(
	cdc codec.BinaryCodec,
	storeKey storetypes.StoreKey,
	perpetualKeeper PerpetualKeeper,
	logger log.Logger,
	parallelConfig ParallelConfig,
) *Keeper {
	k := &Keeper{
		cdc:             cdc,
		storeKey:        storeKey,
		perpetualKeeper: perpetualKeeper,
		logger:          logger.With("module", "x/orderbook"),
		parallelConfig:  parallelConfig,
	}
	k.parallelMatcher = NewParallelMatcher(k, parallelConfig)
	k.parallelMatcherV2 = NewParallelMatcherV2(k, parallelConfig)
	return k
}

// Logger returns the module logger
func (k *Keeper) Logger() log.Logger {
	return k.logger
}

// GetStore returns the KVStore for this module
func (k *Keeper) GetStore(ctx sdk.Context) storetypes.KVStore {
	return ctx.KVStore(k.storeKey)
}

// SetOrder saves an order to the store
func (k *Keeper) SetOrder(ctx sdk.Context, order *types.Order) {
	store := k.GetStore(ctx)
	key := append(OrderKeyPrefix, []byte(order.OrderID)...)
	bz, _ := json.Marshal(order)
	store.Set(key, bz)
}

// GetOrder retrieves an order from the store
func (k *Keeper) GetOrder(ctx sdk.Context, orderID string) *types.Order {
	store := k.GetStore(ctx)
	key := append(OrderKeyPrefix, []byte(orderID)...)
	bz := store.Get(key)
	if bz == nil {
		return nil
	}
	var order types.Order
	if err := json.Unmarshal(bz, &order); err != nil {
		return nil
	}
	return &order
}

// DeleteOrder removes an order from the store
func (k *Keeper) DeleteOrder(ctx sdk.Context, orderID string) {
	store := k.GetStore(ctx)
	key := append(OrderKeyPrefix, []byte(orderID)...)
	store.Delete(key)
}

// GetOrdersByTrader returns all orders for a trader
func (k *Keeper) GetOrdersByTrader(ctx sdk.Context, trader string) []*types.Order {
	store := k.GetStore(ctx)
	iterator := storetypes.KVStorePrefixIterator(store, OrderKeyPrefix)
	defer iterator.Close()

	var orders []*types.Order
	for ; iterator.Valid(); iterator.Next() {
		var order types.Order
		if err := json.Unmarshal(iterator.Value(), &order); err != nil {
			continue
		}
		if order.Trader == trader {
			orders = append(orders, &order)
		}
	}
	return orders
}

// SetOrderBook saves an order book to the store
func (k *Keeper) SetOrderBook(ctx sdk.Context, ob *types.OrderBook) {
	store := k.GetStore(ctx)
	key := append(OrderBookKeyPrefix, []byte(ob.MarketID)...)
	bz, _ := json.Marshal(ob)
	store.Set(key, bz)
}

// GetOrderBook retrieves an order book from the store
func (k *Keeper) GetOrderBook(ctx sdk.Context, marketID string) *types.OrderBook {
	store := k.GetStore(ctx)
	key := append(OrderBookKeyPrefix, []byte(marketID)...)
	bz := store.Get(key)
	if bz == nil {
		return nil
	}
	var ob types.OrderBook
	if err := json.Unmarshal(bz, &ob); err != nil {
		return nil
	}
	return &ob
}

// SetTrade saves a trade to the store
func (k *Keeper) SetTrade(ctx sdk.Context, trade *types.Trade) {
	store := k.GetStore(ctx)
	key := append(TradeKeyPrefix, []byte(trade.TradeID)...)
	bz, _ := json.Marshal(trade)
	store.Set(key, bz)
}

// GetRecentTrades returns recent trades for a market
func (k *Keeper) GetRecentTrades(ctx sdk.Context, marketID string, limit int) []*types.Trade {
	store := k.GetStore(ctx)
	iterator := storetypes.KVStoreReversePrefixIterator(store, TradeKeyPrefix)
	defer iterator.Close()

	var trades []*types.Trade
	count := 0
	for ; iterator.Valid() && count < limit; iterator.Next() {
		var trade types.Trade
		if err := json.Unmarshal(iterator.Value(), &trade); err != nil {
			continue
		}
		if trade.MarketID == marketID {
			trades = append(trades, &trade)
			count++
		}
	}
	return trades
}

// generateOrderID generates a unique order ID
func (k *Keeper) generateOrderID(ctx sdk.Context) string {
	store := k.GetStore(ctx)
	bz := store.Get(OrderCounterKey)
	var counter uint64
	if bz != nil {
		counter = binary.BigEndian.Uint64(bz)
	}
	counter++

	newBz := make([]byte, 8)
	binary.BigEndian.PutUint64(newBz, counter)
	store.Set(OrderCounterKey, newBz)

	return fmt.Sprintf("order-%d", counter)
}

// generateTradeID generates a unique trade ID
func (k *Keeper) generateTradeID(ctx sdk.Context) string {
	store := k.GetStore(ctx)
	bz := store.Get(TradeCounterKey)
	var counter uint64
	if bz != nil {
		counter = binary.BigEndian.Uint64(bz)
	}
	counter++

	newBz := make([]byte, 8)
	binary.BigEndian.PutUint64(newBz, counter)
	store.Set(TradeCounterKey, newBz)

	return fmt.Sprintf("trade-%d", counter)
}

// emitTradeEvent emits a trade event
func (k *Keeper) emitTradeEvent(ctx sdk.Context, trade *types.Trade) {
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"trade",
			sdk.NewAttribute("trade_id", trade.TradeID),
			sdk.NewAttribute("market_id", trade.MarketID),
			sdk.NewAttribute("taker", trade.Taker),
			sdk.NewAttribute("maker", trade.Maker),
			sdk.NewAttribute("price", trade.Price.String()),
			sdk.NewAttribute("quantity", trade.Quantity.String()),
			sdk.NewAttribute("taker_side", trade.TakerSide.String()),
		),
	)
}

// PlaceOrder handles placing a new order
func (k *Keeper) PlaceOrder(ctx context.Context, trader, marketID string, side types.Side, orderType types.OrderType, price, quantity math.LegacyDec) (*types.Order, *MatchResult, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// Generate order ID
	orderID := k.generateOrderID(sdkCtx)

	// Create order
	order := types.NewOrder(orderID, trader, marketID, side, orderType, price, quantity)

	// Check margin requirement via perpetualKeeper (REAL margin validation)
	if err := k.perpetualKeeper.CheckMarginRequirement(sdkCtx, trader, marketID, side, quantity, price); err != nil {
		return nil, nil, fmt.Errorf("insufficient margin: %w", err)
	}

	// Process order through matching engine
	engine := NewMatchingEngine(k)
	result, err := engine.ProcessOrder(sdkCtx, order)
	if err != nil {
		return nil, nil, err
	}

	return order, result, nil
}

// CancelOrder handles order cancellation
func (k *Keeper) CancelOrder(ctx context.Context, trader, orderID string) (*types.Order, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	order := k.GetOrder(sdkCtx, orderID)
	if order == nil {
		return nil, fmt.Errorf("order not found: %s", orderID)
	}

	if order.Trader != trader {
		return nil, fmt.Errorf("unauthorized: order belongs to different trader")
	}

	engine := NewMatchingEngine(k)
	return engine.CancelOrder(sdkCtx, orderID)
}

// GetParallelConfig returns the current parallel matching configuration
func (k *Keeper) GetParallelConfig() ParallelConfig {
	return k.parallelConfig
}

// SetParallelConfig updates the parallel matching configuration
func (k *Keeper) SetParallelConfig(config ParallelConfig) {
	k.parallelConfig = config
	k.parallelMatcher = NewParallelMatcher(k, config)
	k.parallelMatcherV2 = NewParallelMatcherV2(k, config)
}

// IsParallelEnabled returns whether parallel matching is enabled
func (k *Keeper) IsParallelEnabled() bool {
	return k.parallelConfig.Enabled
}

// GetAllPendingOrders retrieves all pending orders from the store
func (k *Keeper) GetAllPendingOrders(ctx sdk.Context) []*types.Order {
	store := k.GetStore(ctx)
	iterator := storetypes.KVStorePrefixIterator(store, OrderKeyPrefix)
	defer iterator.Close()

	var orders []*types.Order
	for ; iterator.Valid(); iterator.Next() {
		var order types.Order
		if err := json.Unmarshal(iterator.Value(), &order); err != nil {
			continue
		}
		if order.IsActive() {
			orders = append(orders, &order)
		}
	}
	return orders
}

// GetPendingOrdersByMarket retrieves all pending orders for a specific market
func (k *Keeper) GetPendingOrdersByMarket(ctx sdk.Context, marketID string) []*types.Order {
	store := k.GetStore(ctx)
	iterator := storetypes.KVStorePrefixIterator(store, OrderKeyPrefix)
	defer iterator.Close()

	var orders []*types.Order
	for ; iterator.Valid(); iterator.Next() {
		var order types.Order
		if err := json.Unmarshal(iterator.Value(), &order); err != nil {
			continue
		}
		if order.MarketID == marketID && order.IsActive() {
			orders = append(orders, &order)
		}
	}
	return orders
}

// EndBlocker is called at the end of each block to process pending orders
func (k *Keeper) EndBlocker(ctx sdk.Context) error {
	if k.parallelConfig.Enabled {
		return k.ParallelEndBlocker(ctx)
	}
	return k.SequentialEndBlocker(ctx)
}

// ParallelEndBlocker processes pending orders using parallel matching
func (k *Keeper) ParallelEndBlocker(ctx sdk.Context) error {
	logger := k.Logger()

	// Get all pending orders
	pendingOrders := k.GetAllPendingOrders(ctx)
	if len(pendingOrders) == 0 {
		return nil
	}

	logger.Info("starting parallel end block matching",
		"pending_orders", len(pendingOrders),
		"workers", k.parallelConfig.Workers,
	)

	// Perform parallel matching
	result, err := k.parallelMatcher.MatchParallel(ctx, pendingOrders)
	if err != nil {
		logger.Error("parallel matching failed", "error", err)
		return fmt.Errorf("parallel matching failed: %w", err)
	}

	// Log results
	logger.Info("parallel matching completed",
		"total_trades", result.TotalTrades,
		"total_matched", result.TotalMatched,
		"duration", result.Duration.String(),
		"errors", len(result.Errors),
	)

	// Save trades
	for _, marketResult := range result.Results {
		for _, trade := range marketResult.Trades {
			k.SetTrade(ctx, trade)
		}
	}

	// Handle any errors
	if len(result.Errors) > 0 {
		for _, e := range result.Errors {
			logger.Error("matching error", "error", e)
		}
	}

	return nil
}

// ParallelEndBlockerV2 processes pending orders using the V2 parallel matcher.
func (k *Keeper) ParallelEndBlockerV2(ctx sdk.Context) (*AggregatedMatchResultV2, error) {
	logger := k.Logger()

	pendingOrders := k.GetAllPendingOrders(ctx)
	if len(pendingOrders) == 0 {
		return &AggregatedMatchResultV2{
			Results:  make([]*ParallelMatchResultV2, 0),
			Errors:   make([]error, 0),
			Duration: 0,
		}, nil
	}

	logger.Info("starting parallel end block matching v2",
		"pending_orders", len(pendingOrders),
		"workers", k.parallelConfig.Workers,
	)

	result, err := k.parallelMatcherV2.MatchParallel(ctx, pendingOrders)
	if err != nil {
		logger.Error("parallel matching v2 failed", "error", err)
		if result == nil {
			result = &AggregatedMatchResultV2{
				Results: make([]*ParallelMatchResultV2, 0),
				Errors:  []error{err},
			}
		} else {
			result.Errors = append(result.Errors, err)
		}
		return result, fmt.Errorf("parallel matching v2 failed: %w", err)
	}

	for _, marketResult := range result.Results {
		if marketResult == nil {
			continue
		}
		if marketResult.Error != nil {
			logger.Error("parallel matching v2 market failed",
				"market_id", marketResult.MarketID,
				"error", marketResult.Error,
			)
			continue
		}
		if marketResult.Commit != nil {
			marketResult.Commit()
		}
	}

	if len(result.Errors) > 0 {
		for _, e := range result.Errors {
			logger.Error("parallel matching v2 error", "error", e)
		}
	}

	logger.Info("parallel matching v2 completed",
		"total_trades", result.TotalTrades,
		"duration", result.Duration.String(),
		"errors", len(result.Errors),
	)

	return result, nil
}

// SequentialEndBlocker processes pending orders sequentially (fallback)
func (k *Keeper) SequentialEndBlocker(ctx sdk.Context) error {
	logger := k.Logger()
	engine := NewMatchingEngine(k)

	// Get all pending orders
	pendingOrders := k.GetAllPendingOrders(ctx)
	if len(pendingOrders) == 0 {
		return nil
	}

	logger.Info("starting sequential end block matching",
		"pending_orders", len(pendingOrders),
	)

	totalTrades := 0
	for _, order := range pendingOrders {
		if !order.IsActive() {
			continue
		}

		result, err := engine.ProcessOrder(ctx, order)
		if err != nil {
			logger.Error("failed to process order", "order_id", order.OrderID, "error", err)
			continue
		}

		if result != nil && len(result.Trades) > 0 {
			totalTrades += len(result.Trades)
			for _, trade := range result.Trades {
				k.SetTrade(ctx, trade)
			}
		}
	}

	logger.Info("sequential matching completed",
		"total_trades", totalTrades,
		"orders_processed", len(pendingOrders),
	)

	return nil
}
