package api

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"sync/atomic"

	"github.com/openalpha/perp-dex/api/types"
)

// MockService implements all service interfaces with mock data
type MockService struct {
	orders    map[string]*types.Order
	positions map[string]*types.Position // key: trader:marketID
	accounts  map[string]*types.Account
	mu        sync.RWMutex
	orderSeq  int64
}

// NewMockService creates a new mock service
func NewMockService() *MockService {
	ms := &MockService{
		orders:    make(map[string]*types.Order),
		positions: make(map[string]*types.Position),
		accounts:  make(map[string]*types.Account),
	}
	ms.initMockData()
	return ms
}

// initMockData initializes the service
// NOTE: No hardcoded demo data - all data comes from real user actions
func (ms *MockService) initMockData() {
	// Empty initialization - no demo/mock data
	// Users start with empty accounts and must deposit/trade to see data
}

// ============ OrderService Implementation ============

func (ms *MockService) PlaceOrder(ctx context.Context, req *types.PlaceOrderRequest) (*types.PlaceOrderResponse, error) {
	ms.mu.Lock()
	defer ms.mu.Unlock()

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

	// Generate order ID
	seq := atomic.AddInt64(&ms.orderSeq, 1)
	orderID := fmt.Sprintf("order-%d", seq)

	now := types.NowMillis()
	order := &types.Order{
		OrderID:   orderID,
		Trader:    req.Trader,
		MarketID:  req.MarketID,
		Side:      req.Side,
		Type:      req.Type,
		Price:     req.Price,
		Quantity:  req.Quantity,
		FilledQty: "0.00",
		Status:    "open",
		CreatedAt: now,
		UpdatedAt: now,
	}

	ms.orders[orderID] = order

	// Simulate partial fill for market orders
	match := &types.MatchResult{
		FilledQty:    "0.00",
		AvgPrice:     "0.00",
		RemainingQty: req.Quantity,
		Trades:       []types.TradeInfo{},
	}

	if req.Type == "market" {
		// Simulate immediate fill
		match.FilledQty = req.Quantity
		match.AvgPrice = req.Price
		match.RemainingQty = "0.00"
		order.FilledQty = req.Quantity
		order.Status = "filled"
		order.UpdatedAt = types.NowMillis()

		// Add mock trade
		match.Trades = append(match.Trades, types.TradeInfo{
			TradeID:   fmt.Sprintf("trade-%d", rand.Intn(100000)),
			Price:     req.Price,
			Quantity:  req.Quantity,
			Timestamp: now,
		})
	}

	return &types.PlaceOrderResponse{
		Order: order,
		Match: match,
	}, nil
}

func (ms *MockService) CancelOrder(ctx context.Context, trader, orderID string) (*types.CancelOrderResponse, error) {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	order, ok := ms.orders[orderID]
	if !ok {
		return nil, fmt.Errorf("order not found: %s", orderID)
	}

	if order.Trader != trader {
		return nil, fmt.Errorf("unauthorized: order belongs to different trader")
	}

	if order.Status != "open" {
		return nil, fmt.Errorf("order cannot be cancelled: status is %s", order.Status)
	}

	order.Status = "cancelled"
	order.UpdatedAt = types.NowMillis()

	return &types.CancelOrderResponse{
		Order:     order,
		Cancelled: true,
	}, nil
}

func (ms *MockService) ModifyOrder(ctx context.Context, trader, orderID string, req *types.ModifyOrderRequest) (*types.ModifyOrderResponse, error) {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	oldOrder, ok := ms.orders[orderID]
	if !ok {
		return nil, fmt.Errorf("order not found: %s", orderID)
	}

	if oldOrder.Trader != trader {
		return nil, fmt.Errorf("unauthorized: order belongs to different trader")
	}

	if oldOrder.Status != "open" {
		return nil, fmt.Errorf("order cannot be modified: status is %s", oldOrder.Status)
	}

	// Cancel old order
	oldOrder.Status = "cancelled"
	oldOrder.UpdatedAt = types.NowMillis()

	// Create new order with modified values
	seq := atomic.AddInt64(&ms.orderSeq, 1)
	newOrderID := fmt.Sprintf("order-%d", seq)

	now := types.NowMillis()
	price := oldOrder.Price
	quantity := oldOrder.Quantity
	if req.Price != "" {
		price = req.Price
	}
	if req.Quantity != "" {
		quantity = req.Quantity
	}

	newOrder := &types.Order{
		OrderID:   newOrderID,
		Trader:    oldOrder.Trader,
		MarketID:  oldOrder.MarketID,
		Side:      oldOrder.Side,
		Type:      oldOrder.Type,
		Price:     price,
		Quantity:  quantity,
		FilledQty: "0.00",
		Status:    "open",
		CreatedAt: now,
		UpdatedAt: now,
	}

	ms.orders[newOrderID] = newOrder

	return &types.ModifyOrderResponse{
		OldOrderID: orderID,
		Order:      newOrder,
		Match: &types.MatchResult{
			FilledQty:    "0.00",
			AvgPrice:     "0.00",
			RemainingQty: quantity,
			Trades:       []types.TradeInfo{},
		},
	}, nil
}

func (ms *MockService) GetOrder(ctx context.Context, orderID string) (*types.Order, error) {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	order, ok := ms.orders[orderID]
	if !ok {
		return nil, fmt.Errorf("order not found: %s", orderID)
	}
	return order, nil
}

func (ms *MockService) ListOrders(ctx context.Context, req *types.ListOrdersRequest) (*types.ListOrdersResponse, error) {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	var orders []*types.Order
	for _, order := range ms.orders {
		// Filter by trader
		if req.Trader != "" && order.Trader != req.Trader {
			continue
		}
		// Filter by market
		if req.MarketID != "" && order.MarketID != req.MarketID {
			continue
		}
		// Filter by status
		if req.Status != "" && order.Status != req.Status {
			continue
		}
		orders = append(orders, order)
	}

	// Apply limit
	limit := req.Limit
	if limit <= 0 || limit > 100 {
		limit = 100
	}
	if len(orders) > limit {
		orders = orders[:limit]
	}

	var nextCursor string
	if len(orders) > 0 {
		nextCursor = orders[len(orders)-1].OrderID
	}

	return &types.ListOrdersResponse{
		Orders:     orders,
		NextCursor: nextCursor,
		Total:      len(orders),
	}, nil
}

// ============ PositionService Implementation ============

func (ms *MockService) GetPositions(ctx context.Context, trader string) ([]*types.Position, error) {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	var positions []*types.Position
	for key, pos := range ms.positions {
		if trader == "" || pos.Trader == trader {
			_ = key
			positions = append(positions, pos)
		}
	}
	return positions, nil
}

func (ms *MockService) GetPosition(ctx context.Context, trader, marketID string) (*types.Position, error) {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	key := trader + ":" + marketID
	pos, ok := ms.positions[key]
	if !ok {
		return nil, fmt.Errorf("position not found")
	}
	return pos, nil
}

func (ms *MockService) ClosePosition(ctx context.Context, req *types.ClosePositionRequest) (*types.ClosePositionResponse, error) {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	key := req.Trader + ":" + req.MarketID
	pos, ok := ms.positions[key]
	if !ok {
		return nil, fmt.Errorf("position not found")
	}

	closeSize := req.Size
	if closeSize == "" {
		closeSize = pos.Size
	}

	closePrice := req.Price
	if closePrice == "" {
		closePrice = pos.MarkPrice
	}

	// Calculate realized PnL (simplified)
	realizedPnl := "0.00"
	if pos.Side == "long" {
		realizedPnl = fmt.Sprintf("%.2f", 30.0) // Mock PnL
	} else {
		realizedPnl = fmt.Sprintf("%.2f", -30.0)
	}

	// Remove position if fully closed
	if closeSize == pos.Size {
		delete(ms.positions, key)
	}

	// Update account
	account := ms.accounts[req.Trader]
	if account == nil {
		account = &types.Account{
			Trader:           req.Trader,
			Balance:          "10000.00",
			LockedMargin:     "0.00",
			AvailableBalance: "10000.00",
			MarginMode:       "isolated",
			UpdatedAt:        types.NowMillis(),
		}
		ms.accounts[req.Trader] = account
	}
	account.UpdatedAt = types.NowMillis()

	return &types.ClosePositionResponse{
		MarketID:    req.MarketID,
		ClosedSize:  closeSize,
		ClosePrice:  closePrice,
		RealizedPnl: realizedPnl,
		Account:     account,
	}, nil
}

// ============ AccountService Implementation ============

func (ms *MockService) GetAccount(ctx context.Context, trader string) (*types.Account, error) {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	account, ok := ms.accounts[trader]
	if !ok {
		// Return default account for new traders
		return &types.Account{
			Trader:           trader,
			Balance:          "0.00",
			LockedMargin:     "0.00",
			AvailableBalance: "0.00",
			MarginMode:       "isolated",
			UpdatedAt:        types.NowMillis(),
		}, nil
	}
	return account, nil
}

func (ms *MockService) Deposit(ctx context.Context, req *types.DepositRequest) (*types.AccountResponse, error) {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	if req.Amount == "" {
		return nil, fmt.Errorf("amount is required")
	}

	account, ok := ms.accounts[req.Trader]
	if !ok {
		account = &types.Account{
			Trader:           req.Trader,
			Balance:          "0.00",
			LockedMargin:     "0.00",
			AvailableBalance: "0.00",
			MarginMode:       "isolated",
		}
		ms.accounts[req.Trader] = account
	}

	// In real implementation, parse and add amounts
	// For mock, just set the amount
	account.Balance = req.Amount
	account.AvailableBalance = req.Amount
	account.UpdatedAt = types.NowMillis()

	return &types.AccountResponse{Account: account}, nil
}

func (ms *MockService) Withdraw(ctx context.Context, req *types.WithdrawRequest) (*types.AccountResponse, error) {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	if req.Amount == "" {
		return nil, fmt.Errorf("amount is required")
	}

	account, ok := ms.accounts[req.Trader]
	if !ok {
		return nil, fmt.Errorf("account not found")
	}

	// In real implementation, check balance and subtract
	// For mock, just return success
	account.UpdatedAt = types.NowMillis()

	return &types.AccountResponse{Account: account}, nil
}
