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
// by connecting to a real Keeper instance with BTree-based orderbook
type KeeperService struct {
	keeper   *keeper.Keeper
	ctx      sdk.Context
	mu       sync.RWMutex
	orderSeq atomic.Uint64

	// BTree-based orderbooks for O(log n) matching
	orderBooks map[string]*keeper.OrderBookBTree
	orders     map[string]*orderbooktypes.Order // Order storage by ID

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
		keeper:     k,
		ctx:        ctx,
		orderBooks: make(map[string]*keeper.OrderBookBTree),
		orders:     make(map[string]*orderbooktypes.Order),
		accounts:   make(map[string]*types.Account),
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

	// Generate order ID
	orderID := fmt.Sprintf("order-%d", s.orderSeq.Add(1))

	// Create order
	order := orderbooktypes.NewOrder(orderID, req.Trader, req.MarketID, side, orderType, price, quantity)

	// Get or create BTree orderbook for this market
	ob, exists := s.orderBooks[req.MarketID]
	if !exists {
		ob = keeper.NewOrderBookBTree(req.MarketID)
		s.orderBooks[req.MarketID] = ob
	}

	// Match order using BTree
	matchResult := s.matchOrderBTree(ob, order)

	// If remaining quantity and limit order, add to book
	if order.RemainingQty().IsPositive() && orderType == orderbooktypes.OrderTypeLimit {
		ob.AddOrder(order)
	}

	// Store order
	s.orders[order.OrderID] = order

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
	if matchResult != nil && matchResult.FilledQty.IsPositive() {
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

	// Get order from local storage
	order, exists := s.orders[orderID]
	if !exists {
		return nil, fmt.Errorf("order not found: %s", orderID)
	}

	if order.Trader != trader {
		return nil, fmt.Errorf("unauthorized: order belongs to different trader")
	}

	if !order.IsActive() {
		return nil, fmt.Errorf("order is not active: %s", orderID)
	}

	// Remove from BTree orderbook
	ob, exists := s.orderBooks[order.MarketID]
	if exists {
		ob.RemoveOrder(order)
	}

	// Cancel the order
	order.Cancel()

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

	order, exists := s.orders[orderID]
	if !exists {
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

	apiOrders := make([]*types.Order, 0)
	for _, order := range s.orders {
		// Filter by trader
		if order.Trader != req.Trader {
			continue
		}
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
// BTree Matching Engine
// ============================================================================

// btreeMatchResult contains the result of BTree order matching
type btreeMatchResult struct {
	Trades       []*orderbooktypes.Trade
	FilledQty    math.LegacyDec
	AvgPrice     math.LegacyDec
	RemainingQty math.LegacyDec
}

var tradeSeq atomic.Uint64

// matchOrderBTree performs order matching using BTree orderbook
func (s *KeeperService) matchOrderBTree(ob *keeper.OrderBookBTree, order *orderbooktypes.Order) *btreeMatchResult {
	result := &btreeMatchResult{
		Trades:       make([]*orderbooktypes.Trade, 0),
		FilledQty:    math.LegacyZeroDec(),
		AvgPrice:     math.LegacyZeroDec(),
		RemainingQty: order.RemainingQty(),
	}

	totalValue := math.LegacyZeroDec()

	// Match against opposite side
	for result.RemainingQty.IsPositive() {
		var bestLevel *keeper.PriceLevelV2
		if order.Side == orderbooktypes.SideBuy {
			bestLevel = ob.GetBestAsk()
		} else {
			bestLevel = ob.GetBestBid()
		}

		if bestLevel == nil {
			break
		}

		// Check price compatibility for limit orders
		if order.OrderType == orderbooktypes.OrderTypeLimit {
			if order.Side == orderbooktypes.SideBuy && order.Price.LT(bestLevel.Price) {
				break
			}
			if order.Side == orderbooktypes.SideSell && order.Price.GT(bestLevel.Price) {
				break
			}
		}

		// Match against orders at this level (FIFO)
		for len(bestLevel.Orders) > 0 && result.RemainingQty.IsPositive() {
			makerOrder := bestLevel.Orders[0]
			if makerOrder == nil || !makerOrder.IsActive() {
				bestLevel.Orders = bestLevel.Orders[1:]
				continue
			}

			// Calculate match quantity
			matchQty := math.LegacyMinDec(result.RemainingQty, makerOrder.RemainingQty())
			matchPrice := bestLevel.Price

			// Create trade
			tradeID := fmt.Sprintf("trade-%d", tradeSeq.Add(1))
			trade := &orderbooktypes.Trade{
				TradeID:   tradeID,
				MarketID:  order.MarketID,
				Taker:     order.Trader,
				Maker:     makerOrder.Trader,
				TakerSide: order.Side,
				Price:     matchPrice,
				Quantity:  matchQty,
				TakerFee:  math.LegacyZeroDec(),
				MakerFee:  math.LegacyZeroDec(),
				Timestamp: time.Now(),
			}
			result.Trades = append(result.Trades, trade)

			// Update quantities
			order.Fill(matchQty)
			makerOrder.Fill(matchQty)

			// Update tracking
			result.FilledQty = result.FilledQty.Add(matchQty)
			result.RemainingQty = result.RemainingQty.Sub(matchQty)
			totalValue = totalValue.Add(matchQty.Mul(matchPrice))

			// Update level quantity
			bestLevel.Quantity = bestLevel.Quantity.Sub(matchQty)

			// Remove filled maker order from level
			if makerOrder.IsFilled() {
				bestLevel.Orders = bestLevel.Orders[1:]
				// Update stored order
				s.orders[makerOrder.OrderID] = makerOrder
			}
		}

		// Remove empty level
		if bestLevel.IsEmpty() {
			if order.Side == orderbooktypes.SideBuy {
				ob.RemoveOrderByID("", orderbooktypes.SideSell, bestLevel.Price)
			} else {
				ob.RemoveOrderByID("", orderbooktypes.SideBuy, bestLevel.Price)
			}
		}
	}

	// Calculate average price
	if result.FilledQty.IsPositive() {
		result.AvgPrice = totalValue.Quo(result.FilledQty)
	}

	return result
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

// GetOrderBookDepth returns the order book depth for a market using BTree
func (s *KeeperService) GetOrderBookDepth(marketID string, depth int) (bids, asks [][]string) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ob, exists := s.orderBooks[marketID]
	if !exists {
		return [][]string{}, [][]string{}
	}

	// Get bids using BTree iteration (highest to lowest)
	bidLevels := ob.GetBidLevels(depth)
	bids = make([][]string, 0, len(bidLevels))
	for _, level := range bidLevels {
		bids = append(bids, []string{level.Price.String(), level.Quantity.String()})
	}

	// Get asks using BTree iteration (lowest to highest)
	askLevels := ob.GetAskLevels(depth)
	asks = make([][]string, 0, len(askLevels))
	for _, level := range askLevels {
		asks = append(asks, []string{level.Price.String(), level.Quantity.String()})
	}

	return bids, asks
}
