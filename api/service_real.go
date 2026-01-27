package api

import (
	"context"
	"fmt"
	"sync"
	"time"

	"cosmossdk.io/log"
	"cosmossdk.io/math"
	"cosmossdk.io/store"
	"cosmossdk.io/store/metrics"
	storetypes "cosmossdk.io/store/types"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/openalpha/perp-dex/api/types"
	obkeeper "github.com/openalpha/perp-dex/x/orderbook/keeper"
	obtypes "github.com/openalpha/perp-dex/x/orderbook/types"
	perpkeeper "github.com/openalpha/perp-dex/x/perpetual/keeper"
	perptypes "github.com/openalpha/perp-dex/x/perpetual/types"
)

// RealService implements all service interfaces with real orderbook engine
// This bridges the API layer to the actual Cosmos SDK Keepers
type RealService struct {
	obKeeper    *obkeeper.Keeper
	perpKeeper  *perpkeeper.Keeper
	matchEngine *obkeeper.MatchingEngineV2
	sdkCtx      sdk.Context
	mu          sync.RWMutex
	logger      log.Logger
}

// SimplePerpetualKeeper is a minimal implementation of PerpetualKeeper interface
// for standalone API server usage (without full chain integration)
type SimplePerpetualKeeper struct {
	markets map[string]*obkeeper.Market
	oracle  *HyperliquidOracle
	mu      sync.RWMutex
}

func NewSimplePerpetualKeeper() *SimplePerpetualKeeper {
	pk := &SimplePerpetualKeeper{
		markets: make(map[string]*obkeeper.Market),
		oracle:  NewHyperliquidOracle(), // Use Hyperliquid Oracle for real-time prices
	}
	// Initialize default markets
	pk.initDefaultMarkets()
	return pk
}

func (pk *SimplePerpetualKeeper) initDefaultMarkets() {
	defaultMarkets := []struct {
		id         string
		takerFee   string
		makerFee   string
		initMargin string
	}{
		{"BTC-USDC", "0.0006", "0.0001", "0.01"},
		{"ETH-USDC", "0.0006", "0.0001", "0.01"},
		{"SOL-USDC", "0.0006", "0.0001", "0.01"},
	}

	for _, m := range defaultMarkets {
		takerFee, _ := math.LegacyNewDecFromStr(m.takerFee)
		makerFee, _ := math.LegacyNewDecFromStr(m.makerFee)
		initMargin, _ := math.LegacyNewDecFromStr(m.initMargin)
		pk.markets[m.id] = &obkeeper.Market{
			MarketID:      m.id,
			TakerFeeRate:  takerFee,
			MakerFeeRate:  makerFee,
			InitialMargin: initMargin,
		}
	}
}

func (pk *SimplePerpetualKeeper) GetMarket(ctx sdk.Context, marketID string) *obkeeper.Market {
	pk.mu.RLock()
	defer pk.mu.RUnlock()
	return pk.markets[marketID]
}

func (pk *SimplePerpetualKeeper) GetMarkPrice(ctx sdk.Context, marketID string) (math.LegacyDec, bool) {
	// Use Hyperliquid Oracle for real-time prices
	if pk.oracle != nil {
		price, err := pk.oracle.GetPrice(marketID)
		if err == nil && !price.IsZero() {
			return price, true
		}
	}
	// No fallback - return zero if Oracle fails
	return math.LegacyZeroDec(), false
}

func (pk *SimplePerpetualKeeper) UpdatePosition(ctx sdk.Context, trader, marketID string, side obtypes.Side, qty, price, fee interface{}) error {
	// Position updates are handled separately
	return nil
}

func (pk *SimplePerpetualKeeper) CheckMarginRequirement(ctx sdk.Context, trader, marketID string, side obtypes.Side, qty, price interface{}) error {
	// Margin checks are handled separately
	return nil
}

// NewRealService creates a new real service with in-memory store
// This is for standalone API server usage without full chain
func NewRealService(logger log.Logger) (*RealService, error) {
	// Create in-memory database and store
	db := dbm.NewMemDB()
	storeKey := storetypes.NewKVStoreKey("orderbook")

	// Create multi-store with proper metrics
	cms := store.NewCommitMultiStore(db, logger, metrics.NewNoOpMetrics())
	cms.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, db)
	if err := cms.LoadLatestVersion(); err != nil {
		return nil, fmt.Errorf("failed to load store: %w", err)
	}

	// Create codec
	interfaceRegistry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(interfaceRegistry)

	// Create perpetual keeper (simplified)
	perpKeeper := NewSimplePerpetualKeeper()

	// Create orderbook keeper
	obKeeper := obkeeper.NewKeeper(cdc, storeKey, perpKeeper, logger)

	// Create SDK context with empty header
	header := tmproto.Header{Height: 1}
	sdkCtx := sdk.NewContext(cms, header, false, logger)

	// Create matching engine V2
	matchEngine := obkeeper.NewMatchingEngineV2(obKeeper)

	return &RealService{
		obKeeper:    obKeeper,
		perpKeeper:  nil, // Use simplified keeper via obKeeper
		matchEngine: matchEngine,
		sdkCtx:      sdkCtx,
		logger:      logger,
	}, nil
}

// NewRealServiceWithKeepers creates a real service with provided Keepers
// This is for full chain integration
func NewRealServiceWithKeepers(
	obKeeper *obkeeper.Keeper,
	perpKeeper *perpkeeper.Keeper,
	sdkCtx sdk.Context,
	logger log.Logger,
) *RealService {
	return &RealService{
		obKeeper:    obKeeper,
		perpKeeper:  perpKeeper,
		matchEngine: obkeeper.NewMatchingEngineV2(obKeeper),
		sdkCtx:      sdkCtx,
		logger:      logger,
	}
}

// ============ OrderService Implementation ============

func (rs *RealService) PlaceOrder(ctx context.Context, req *types.PlaceOrderRequest) (*types.PlaceOrderResponse, error) {
	rs.mu.Lock()
	defer rs.mu.Unlock()

	// Validate request
	if req.MarketID == "" {
		return nil, fmt.Errorf("market_id is required")
	}
	if req.Side != "buy" && req.Side != "sell" {
		return nil, fmt.Errorf("invalid side: %s", req.Side)
	}
	if req.Type != "limit" && req.Type != "market" {
		return nil, fmt.Errorf("invalid type: %s", req.Type)
	}

	// Parse parameters
	price, err := math.LegacyNewDecFromStr(req.Price)
	if err != nil && req.Type == "limit" {
		return nil, fmt.Errorf("invalid price: %s", req.Price)
	}
	if req.Type == "market" && price.IsZero() {
		// For market orders, use a large price for buy, small for sell
		if req.Side == "buy" {
			price = math.LegacyNewDec(1000000000) // Very high price
		} else {
			price = math.LegacyNewDec(1) // Very low price
		}
	}

	qty, err := math.LegacyNewDecFromStr(req.Quantity)
	if err != nil {
		return nil, fmt.Errorf("invalid quantity: %s", req.Quantity)
	}

	// Convert side and type
	side := obtypes.SideBuy
	if req.Side == "sell" {
		side = obtypes.SideSell
	}
	orderType := obtypes.OrderTypeLimit
	if req.Type == "market" {
		orderType = obtypes.OrderTypeMarket
	}

	// Place order through real Keeper (using internal SDK context, not HTTP context)
	order, matchResult, err := rs.obKeeper.PlaceOrder(rs.sdkCtx, req.Trader, req.MarketID, side, orderType, price, qty)
	if err != nil {
		return nil, fmt.Errorf("failed to place order: %w", err)
	}

	// Flush cache to persist changes
	rs.matchEngine.Flush(rs.sdkCtx)

	// Convert to API response
	return rs.convertPlaceOrderResponse(order, matchResult), nil
}

func (rs *RealService) CancelOrder(ctx context.Context, trader, orderID string) (*types.CancelOrderResponse, error) {
	rs.mu.Lock()
	defer rs.mu.Unlock()

	// Cancel through real Keeper (using internal SDK context)
	order, err := rs.obKeeper.CancelOrder(rs.sdkCtx, trader, orderID)
	if err != nil {
		return nil, err
	}

	// Flush cache
	rs.matchEngine.Flush(rs.sdkCtx)

	return &types.CancelOrderResponse{
		Order:     rs.convertOrder(order),
		Cancelled: true,
	}, nil
}

func (rs *RealService) ModifyOrder(ctx context.Context, trader, orderID string, req *types.ModifyOrderRequest) (*types.ModifyOrderResponse, error) {
	rs.mu.Lock()
	defer rs.mu.Unlock()

	// Get existing order
	oldOrder := rs.obKeeper.GetOrder(rs.sdkCtx, orderID)
	if oldOrder == nil {
		return nil, fmt.Errorf("order not found: %s", orderID)
	}
	if oldOrder.Trader != trader {
		return nil, fmt.Errorf("unauthorized: order belongs to different trader")
	}
	if !oldOrder.IsActive() {
		return nil, fmt.Errorf("order is not active")
	}

	// Cancel old order (using internal SDK context)
	_, err := rs.obKeeper.CancelOrder(rs.sdkCtx, trader, orderID)
	if err != nil {
		return nil, fmt.Errorf("failed to cancel old order: %w", err)
	}

	// Create new order with modified values
	price := oldOrder.Price
	qty := oldOrder.Quantity
	if req.Price != "" {
		price, _ = math.LegacyNewDecFromStr(req.Price)
	}
	if req.Quantity != "" {
		qty, _ = math.LegacyNewDecFromStr(req.Quantity)
	}

	// Place new order (using internal SDK context)
	newOrder, matchResult, err := rs.obKeeper.PlaceOrder(rs.sdkCtx, trader, oldOrder.MarketID, oldOrder.Side, oldOrder.OrderType, price, qty)
	if err != nil {
		return nil, fmt.Errorf("failed to place new order: %w", err)
	}

	// Flush cache
	rs.matchEngine.Flush(rs.sdkCtx)

	return &types.ModifyOrderResponse{
		OldOrderID: orderID,
		Order:      rs.convertOrder(newOrder),
		Match:      rs.convertMatchResult(matchResult),
	}, nil
}

func (rs *RealService) GetOrder(ctx context.Context, orderID string) (*types.Order, error) {
	rs.mu.RLock()
	defer rs.mu.RUnlock()

	order := rs.obKeeper.GetOrder(rs.sdkCtx, orderID)
	if order == nil {
		return nil, fmt.Errorf("order not found: %s", orderID)
	}
	return rs.convertOrder(order), nil
}

func (rs *RealService) ListOrders(ctx context.Context, req *types.ListOrdersRequest) (*types.ListOrdersResponse, error) {
	rs.mu.RLock()
	defer rs.mu.RUnlock()

	var orders []*obtypes.Order
	if req.Trader != "" {
		orders = rs.obKeeper.GetOrdersByTrader(rs.sdkCtx, req.Trader)
	} else {
		orders = rs.obKeeper.GetAllPendingOrders(rs.sdkCtx)
	}

	// Filter and convert
	var result []*types.Order
	for _, order := range orders {
		if req.MarketID != "" && order.MarketID != req.MarketID {
			continue
		}
		if req.Status != "" && order.Status.String() != req.Status {
			continue
		}
		result = append(result, rs.convertOrder(order))
	}

	// Apply limit
	limit := req.Limit
	if limit <= 0 || limit > 100 {
		limit = 100
	}
	if len(result) > limit {
		result = result[:limit]
	}

	var nextCursor string
	if len(result) > 0 {
		nextCursor = result[len(result)-1].OrderID
	}

	return &types.ListOrdersResponse{
		Orders:     result,
		NextCursor: nextCursor,
		Total:      len(result),
	}, nil
}

// ============ PositionService Implementation ============

func (rs *RealService) GetPositions(ctx context.Context, trader string) ([]*types.Position, error) {
	rs.mu.RLock()
	defer rs.mu.RUnlock()

	if rs.perpKeeper == nil {
		// Return empty for standalone mode
		return []*types.Position{}, nil
	}

	positions := rs.perpKeeper.GetPositionsByTrader(rs.sdkCtx, trader)
	var result []*types.Position
	for _, pos := range positions {
		result = append(result, rs.convertPosition(pos))
	}
	return result, nil
}

func (rs *RealService) GetPosition(ctx context.Context, trader, marketID string) (*types.Position, error) {
	rs.mu.RLock()
	defer rs.mu.RUnlock()

	if rs.perpKeeper == nil {
		return nil, fmt.Errorf("position not found")
	}

	pos := rs.perpKeeper.GetPosition(rs.sdkCtx, trader, marketID)
	if pos == nil {
		return nil, fmt.Errorf("position not found")
	}
	return rs.convertPosition(pos), nil
}

func (rs *RealService) ClosePosition(ctx context.Context, req *types.ClosePositionRequest) (*types.ClosePositionResponse, error) {
	rs.mu.Lock()
	defer rs.mu.Unlock()

	// For standalone mode, return error
	if rs.perpKeeper == nil {
		return nil, fmt.Errorf("position management not available in standalone mode")
	}

	// TODO: Implement position closing via perpetual keeper
	return nil, fmt.Errorf("not implemented")
}

// ============ AccountService Implementation ============

func (rs *RealService) GetAccount(ctx context.Context, trader string) (*types.Account, error) {
	rs.mu.RLock()
	defer rs.mu.RUnlock()

	if rs.perpKeeper == nil {
		// Return default account for standalone mode
		return &types.Account{
			Trader:           trader,
			Balance:          "10000.00",
			LockedMargin:     "0.00",
			AvailableBalance: "10000.00",
			MarginMode:       "isolated",
			UpdatedAt:        types.NowMillis(),
		}, nil
	}

	account := rs.perpKeeper.GetAccount(rs.sdkCtx, trader)
	if account == nil {
		return &types.Account{
			Trader:           trader,
			Balance:          "0.00",
			LockedMargin:     "0.00",
			AvailableBalance: "0.00",
			MarginMode:       "isolated",
			UpdatedAt:        types.NowMillis(),
		}, nil
	}
	return rs.convertAccount(account), nil
}

func (rs *RealService) Deposit(ctx context.Context, req *types.DepositRequest) (*types.AccountResponse, error) {
	rs.mu.Lock()
	defer rs.mu.Unlock()

	if rs.perpKeeper == nil {
		return nil, fmt.Errorf("deposit not available in standalone mode")
	}

	amount, _ := math.LegacyNewDecFromStr(req.Amount)
	err := rs.perpKeeper.Deposit(ctx, req.Trader, amount)
	if err != nil {
		return nil, err
	}

	account := rs.perpKeeper.GetAccount(rs.sdkCtx, req.Trader)
	return &types.AccountResponse{Account: rs.convertAccount(account)}, nil
}

func (rs *RealService) Withdraw(ctx context.Context, req *types.WithdrawRequest) (*types.AccountResponse, error) {
	rs.mu.Lock()
	defer rs.mu.Unlock()

	if rs.perpKeeper == nil {
		return nil, fmt.Errorf("withdraw not available in standalone mode")
	}

	amount, _ := math.LegacyNewDecFromStr(req.Amount)
	err := rs.perpKeeper.Withdraw(ctx, req.Trader, amount)
	if err != nil {
		return nil, err
	}

	account := rs.perpKeeper.GetAccount(rs.sdkCtx, req.Trader)
	return &types.AccountResponse{Account: rs.convertAccount(account)}, nil
}

// ============ Conversion Helpers ============

func (rs *RealService) convertOrder(order *obtypes.Order) *types.Order {
	if order == nil {
		return nil
	}
	return &types.Order{
		OrderID:   order.OrderID,
		Trader:    order.Trader,
		MarketID:  order.MarketID,
		Side:      order.Side.String(),
		Type:      order.OrderType.String(),
		Price:     order.Price.String(),
		Quantity:  order.Quantity.String(),
		FilledQty: order.FilledQty.String(),
		Status:    order.Status.String(),
		CreatedAt: order.CreatedAt.UnixMilli(),
		UpdatedAt: order.UpdatedAt.UnixMilli(),
	}
}

func (rs *RealService) convertMatchResult(result *obkeeper.MatchResult) *types.MatchResult {
	if result == nil {
		return &types.MatchResult{
			FilledQty:    "0.00",
			AvgPrice:     "0.00",
			RemainingQty: "0.00",
			Trades:       []types.TradeInfo{},
		}
	}

	trades := make([]types.TradeInfo, 0, len(result.Trades))
	for _, t := range result.Trades {
		trades = append(trades, types.TradeInfo{
			TradeID:   t.TradeID,
			Price:     t.Price.String(),
			Quantity:  t.Quantity.String(),
			Timestamp: t.Timestamp.UnixMilli(),
		})
	}

	return &types.MatchResult{
		FilledQty:    result.FilledQty.String(),
		AvgPrice:     result.AvgPrice.String(),
		RemainingQty: result.RemainingQty.String(),
		Trades:       trades,
	}
}

func (rs *RealService) convertPlaceOrderResponse(order *obtypes.Order, result *obkeeper.MatchResult) *types.PlaceOrderResponse {
	return &types.PlaceOrderResponse{
		Order: rs.convertOrder(order),
		Match: rs.convertMatchResult(result),
	}
}

func (rs *RealService) convertPosition(pos *perptypes.Position) *types.Position {
	if pos == nil {
		return nil
	}
	// Calculate mark price and unrealized PnL (using entry price as placeholder for standalone mode)
	markPrice := pos.EntryPrice // In full mode, this would come from oracle
	unrealizedPnL := pos.CalculateUnrealizedPnL(markPrice)

	return &types.Position{
		MarketID:         pos.MarketID,
		Trader:           pos.Trader,
		Side:             pos.Side.String(), // Convert PositionSide to string
		Size:             pos.Size.String(),
		EntryPrice:       pos.EntryPrice.String(),
		MarkPrice:        markPrice.String(),
		Margin:           pos.Margin.String(),
		Leverage:         pos.Leverage.String(),
		UnrealizedPnl:    unrealizedPnL.String(),
		LiquidationPrice: pos.LiquidationPrice.String(),
		MarginMode:       "isolated", // Default for standalone mode
	}
}

func (rs *RealService) convertAccount(account *perptypes.Account) *types.Account {
	if account == nil {
		return nil
	}
	return &types.Account{
		Trader:           account.Trader,
		Balance:          account.Balance.String(),
		LockedMargin:     account.LockedMargin.String(),
		AvailableBalance: account.AvailableBalance().String(),
		MarginMode:       account.MarginMode.String(), // Convert MarginMode to string
		UpdatedAt:        time.Now().UnixMilli(),
	}
}

// ============ Performance Metrics ============

// GetEngineStats returns performance statistics from the matching engine
func (rs *RealService) GetEngineStats() map[string]interface{} {
	rs.mu.RLock()
	defer rs.mu.RUnlock()

	cache := rs.matchEngine.GetCache()
	if cache == nil {
		return map[string]interface{}{
			"status": "no cache",
		}
	}

	return map[string]interface{}{
		"status":      "active",
		"engine_type": "MatchingEngineV2",
		"features": []string{
			"skip_list_orderbook",
			"memory_caching",
			"batch_processing",
			"parallel_matching",
		},
	}
}
