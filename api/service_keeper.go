package api

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"cosmossdk.io/log"
	"cosmossdk.io/math"
	"cosmossdk.io/store"
	"cosmossdk.io/store/metrics"
	storetypes "cosmossdk.io/store/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/openalpha/perp-dex/api/types"
	"github.com/openalpha/perp-dex/x/orderbook/keeper"
	orderbooktypes "github.com/openalpha/perp-dex/x/orderbook/types"
)

// KeeperService implements OrderService, PositionService, AccountService
// by connecting to a real Keeper instance
type KeeperService struct {
	keeper   *keeper.Keeper
	ctx      sdk.Context
	mu       sync.RWMutex
	orderSeq atomic.Uint64

	// In-memory account balances for testing
	accounts map[string]*types.Account
	accMu    sync.RWMutex
}

// mockPerpetualKeeper is a mock implementation for the perpetual module
type mockPerpetualKeeper struct{}

func (m *mockPerpetualKeeper) GetMarket(ctx sdk.Context, marketID string) *keeper.Market {
	return &keeper.Market{
		MarketID:      marketID,
		TakerFeeRate:  math.LegacyNewDecWithPrec(1, 4),  // 0.01%
		MakerFeeRate:  math.LegacyNewDecWithPrec(5, 5),  // 0.005%
		InitialMargin: math.LegacyNewDecWithPrec(10, 2), // 10%
	}
}

func (m *mockPerpetualKeeper) GetMarkPrice(ctx sdk.Context, marketID string) (math.LegacyDec, bool) {
	// Return a reasonable mark price based on market
	switch marketID {
	case "BTC-USDC", "BTC-USD":
		return math.LegacyNewDec(97500), true
	case "ETH-USDC", "ETH-USD":
		return math.LegacyNewDec(3500), true
	default:
		return math.LegacyNewDec(1000), true
	}
}

func (m *mockPerpetualKeeper) UpdatePosition(ctx sdk.Context, trader, marketID string, side orderbooktypes.Side, qty, price, fee interface{}) error {
	return nil
}

func (m *mockPerpetualKeeper) CheckMarginRequirement(ctx sdk.Context, trader, marketID string, side orderbooktypes.Side, qty, price interface{}) error {
	return nil
}

// NewKeeperService creates a new KeeperService with an in-memory keeper
func NewKeeperService() *KeeperService {
	// Create codec
	interfaceRegistry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(interfaceRegistry)

	// Create in-memory store
	storeKey := storetypes.NewKVStoreKey("orderbook")
	db := dbm.NewMemDB()
	stateStore := store.NewCommitMultiStore(db, log.NewNopLogger(), metrics.NewNoOpMetrics())
	stateStore.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, db)
	if err := stateStore.LoadLatestVersion(); err != nil {
		panic(fmt.Sprintf("failed to load store: %v", err))
	}

	// Create context
	ctx := sdk.NewContext(stateStore, cmtproto.Header{
		Time:   time.Now(),
		Height: 1,
	}, false, log.NewNopLogger())

	// Create keeper
	k := keeper.NewKeeper(
		cdc,
		storeKey,
		&mockPerpetualKeeper{},
		log.NewNopLogger(),
	)

	return &KeeperService{
		keeper:   k,
		ctx:      ctx,
		accounts: make(map[string]*types.Account),
	}
}

// ============================================================================
// OrderService Implementation
// ============================================================================

func (s *KeeperService) PlaceOrder(ctx context.Context, req *types.PlaceOrderRequest) (*types.PlaceOrderResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Parse side
	var side orderbooktypes.Side
	switch req.Side {
	case "buy":
		side = orderbooktypes.SideBuy
	case "sell":
		side = orderbooktypes.SideSell
	default:
		return nil, fmt.Errorf("invalid side: %s", req.Side)
	}

	// Parse order type
	var orderType orderbooktypes.OrderType
	switch req.Type {
	case "limit":
		orderType = orderbooktypes.OrderTypeLimit
	case "market":
		orderType = orderbooktypes.OrderTypeMarket
	default:
		return nil, fmt.Errorf("invalid order type: %s", req.Type)
	}

	// Parse price and quantity
	price, err := math.LegacyNewDecFromStr(req.Price)
	if err != nil {
		return nil, fmt.Errorf("invalid price: %v", err)
	}
	quantity, err := math.LegacyNewDecFromStr(req.Quantity)
	if err != nil {
		return nil, fmt.Errorf("invalid quantity: %v", err)
	}

	// Place order through keeper
	order, matchResult, err := s.keeper.PlaceOrder(s.ctx, req.Trader, req.MarketID, side, orderType, price, quantity)
	if err != nil {
		return nil, fmt.Errorf("failed to place order: %v", err)
	}

	// Convert to API response
	apiOrder := &types.Order{
		OrderID:   order.OrderID,
		Trader:    order.Trader,
		MarketID:  order.MarketID,
		Side:      req.Side,
		Type:      req.Type,
		Price:     order.Price.String(),
		Quantity:  order.Quantity.String(),
		FilledQty: order.FilledQty.String(),
		Status:    order.Status.String(),
		CreatedAt: order.CreatedAt.UnixMilli(),
		UpdatedAt: order.UpdatedAt.UnixMilli(),
	}

	// Convert match result
	var apiMatch *types.MatchResult
	if matchResult != nil {
		trades := make([]types.TradeInfo, 0, len(matchResult.Trades))
		for _, t := range matchResult.Trades {
			trades = append(trades, types.TradeInfo{
				TradeID:   t.TradeID,
				Price:     t.Price.String(),
				Quantity:  t.Quantity.String(),
				Timestamp: t.Timestamp.UnixMilli(),
			})
		}
		apiMatch = &types.MatchResult{
			FilledQty:    matchResult.FilledQty.String(),
			AvgPrice:     matchResult.AvgPrice.String(),
			RemainingQty: matchResult.RemainingQty.String(),
			Trades:       trades,
		}
	}

	return &types.PlaceOrderResponse{
		Order: apiOrder,
		Match: apiMatch,
	}, nil
}

func (s *KeeperService) CancelOrder(ctx context.Context, trader, orderID string) (*types.CancelOrderResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	order, err := s.keeper.CancelOrder(s.ctx, trader, orderID)
	if err != nil {
		return nil, err
	}

	return &types.CancelOrderResponse{
		Order: &types.Order{
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
		},
		Cancelled: true,
	}, nil
}

func (s *KeeperService) ModifyOrder(ctx context.Context, trader, orderID string, req *types.ModifyOrderRequest) (*types.ModifyOrderResponse, error) {
	// Cancel old order and place new one
	oldOrder, err := s.CancelOrder(ctx, trader, orderID)
	if err != nil {
		return nil, fmt.Errorf("failed to cancel order for modification: %v", err)
	}

	// Determine new price and quantity
	newPrice := oldOrder.Order.Price
	newQuantity := oldOrder.Order.Quantity
	if req.Price != "" {
		newPrice = req.Price
	}
	if req.Quantity != "" {
		newQuantity = req.Quantity
	}

	// Place new order
	newOrderResp, err := s.PlaceOrder(ctx, &types.PlaceOrderRequest{
		MarketID: oldOrder.Order.MarketID,
		Side:     oldOrder.Order.Side,
		Type:     oldOrder.Order.Type,
		Price:    newPrice,
		Quantity: newQuantity,
		Trader:   trader,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to place new order: %v", err)
	}

	return &types.ModifyOrderResponse{
		OldOrderID: orderID,
		Order:      newOrderResp.Order,
		Match:      newOrderResp.Match,
	}, nil
}

func (s *KeeperService) GetOrder(ctx context.Context, orderID string) (*types.Order, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	order := s.keeper.GetOrder(s.ctx, orderID)
	if order == nil {
		return nil, fmt.Errorf("order not found: %s", orderID)
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
	}, nil
}

func (s *KeeperService) ListOrders(ctx context.Context, req *types.ListOrdersRequest) (*types.ListOrdersResponse, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	orders := s.keeper.GetOrdersByTrader(s.ctx, req.Trader)

	apiOrders := make([]*types.Order, 0, len(orders))
	for _, order := range orders {
		// Filter by market if specified
		if req.MarketID != "" && order.MarketID != req.MarketID {
			continue
		}
		// Filter by status if specified
		if req.Status != "" && order.Status.String() != req.Status {
			continue
		}

		apiOrders = append(apiOrders, &types.Order{
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
		})
	}

	return &types.ListOrdersResponse{
		Orders: apiOrders,
		Total:  len(apiOrders),
	}, nil
}

// ============================================================================
// PositionService Implementation (simplified)
// ============================================================================

func (s *KeeperService) GetPositions(ctx context.Context, trader string) ([]*types.Position, error) {
	// For now, return empty - positions would come from clearinghouse
	return []*types.Position{}, nil
}

func (s *KeeperService) GetPosition(ctx context.Context, trader, marketID string) (*types.Position, error) {
	return nil, fmt.Errorf("position not found")
}

func (s *KeeperService) ClosePosition(ctx context.Context, req *types.ClosePositionRequest) (*types.ClosePositionResponse, error) {
	return nil, fmt.Errorf("not implemented")
}

// ============================================================================
// AccountService Implementation (simplified)
// ============================================================================

func (s *KeeperService) GetAccount(ctx context.Context, trader string) (*types.Account, error) {
	s.accMu.RLock()
	acc, exists := s.accounts[trader]
	s.accMu.RUnlock()

	if !exists {
		// Create default account
		acc = &types.Account{
			Trader:           trader,
			Balance:          "100000.00",
			LockedMargin:     "0.00",
			AvailableBalance: "100000.00",
			MarginMode:       "isolated",
			UpdatedAt:        types.NowMillis(),
		}
		s.accMu.Lock()
		s.accounts[trader] = acc
		s.accMu.Unlock()
	}

	return acc, nil
}

func (s *KeeperService) Deposit(ctx context.Context, req *types.DepositRequest) (*types.AccountResponse, error) {
	acc, _ := s.GetAccount(ctx, req.Trader)
	// Simplified: just update balance
	acc.UpdatedAt = types.NowMillis()
	return &types.AccountResponse{Account: acc}, nil
}

func (s *KeeperService) Withdraw(ctx context.Context, req *types.WithdrawRequest) (*types.AccountResponse, error) {
	acc, _ := s.GetAccount(ctx, req.Trader)
	acc.UpdatedAt = types.NowMillis()
	return &types.AccountResponse{Account: acc}, nil
}

// ============================================================================
// Helper Methods
// ============================================================================

// GetKeeper returns the underlying keeper for direct access in tests
func (s *KeeperService) GetKeeper() *keeper.Keeper {
	return s.keeper
}

// GetContext returns the SDK context
func (s *KeeperService) GetContext() sdk.Context {
	return s.ctx
}

// GetOrderBookDepth returns the order book depth for a market
func (s *KeeperService) GetOrderBookDepth(marketID string, depth int) (bids, asks [][]string) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ob := s.keeper.GetOrderBook(s.ctx, marketID)
	if ob == nil {
		return [][]string{}, [][]string{}
	}

	// Get bids (limited by depth)
	bidCount := len(ob.Bids)
	if bidCount > depth {
		bidCount = depth
	}
	bids = make([][]string, 0, bidCount)
	for i := 0; i < bidCount; i++ {
		pl := ob.Bids[i]
		bids = append(bids, []string{pl.Price.String(), pl.Quantity.String()})
	}

	// Get asks (limited by depth)
	askCount := len(ob.Asks)
	if askCount > depth {
		askCount = depth
	}
	asks = make([][]string, 0, askCount)
	for i := 0; i < askCount; i++ {
		pl := ob.Asks[i]
		asks = append(asks, []string{pl.Price.String(), pl.Quantity.String()})
	}

	return bids, asks
}
