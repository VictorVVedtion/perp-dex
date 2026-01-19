package types

import (
	"context"
	"time"
)

// Order represents an order in the API response
type Order struct {
	OrderID   string `json:"order_id"`
	Trader    string `json:"trader"`
	MarketID  string `json:"market_id"`
	Side      string `json:"side"`
	Type      string `json:"type"`
	Price     string `json:"price"`
	Quantity  string `json:"quantity"`
	FilledQty string `json:"filled_qty"`
	Status    string `json:"status"`
	CreatedAt int64  `json:"created_at"`
	UpdatedAt int64  `json:"updated_at"`
}

// MatchResult represents matching result in API response
type MatchResult struct {
	FilledQty    string      `json:"filled_qty"`
	AvgPrice     string      `json:"avg_price"`
	RemainingQty string      `json:"remaining_qty"`
	Trades       []TradeInfo `json:"trades"`
}

// TradeInfo represents a trade in match result
type TradeInfo struct {
	TradeID   string `json:"trade_id"`
	Price     string `json:"price"`
	Quantity  string `json:"quantity"`
	Timestamp int64  `json:"timestamp"`
}

// Position represents a position in the API response
type Position struct {
	MarketID         string `json:"market_id"`
	Trader           string `json:"trader"`
	Side             string `json:"side"`
	Size             string `json:"size"`
	EntryPrice       string `json:"entry_price"`
	MarkPrice        string `json:"mark_price"`
	Margin           string `json:"margin"`
	Leverage         string `json:"leverage"`
	UnrealizedPnl    string `json:"unrealized_pnl"`
	LiquidationPrice string `json:"liquidation_price"`
	MarginMode       string `json:"margin_mode"`
}

// Account represents an account in the API response
type Account struct {
	Trader           string `json:"trader"`
	Balance          string `json:"balance"`
	LockedMargin     string `json:"locked_margin"`
	AvailableBalance string `json:"available_balance"`
	MarginMode       string `json:"margin_mode"`
	UpdatedAt        int64  `json:"updated_at"`
}

// PlaceOrderRequest represents the request to place an order
type PlaceOrderRequest struct {
	MarketID string `json:"market_id"`
	Side     string `json:"side"`
	Type     string `json:"type"`
	Price    string `json:"price"`
	Quantity string `json:"quantity"`
	Trader   string `json:"trader"`
}

// PlaceOrderResponse represents the response after placing an order
type PlaceOrderResponse struct {
	Order *Order       `json:"order"`
	Match *MatchResult `json:"match,omitempty"`
}

// CancelOrderResponse represents the response after cancelling an order
type CancelOrderResponse struct {
	Order     *Order `json:"order"`
	Cancelled bool   `json:"cancelled"`
}

// ModifyOrderRequest represents the request to modify an order
type ModifyOrderRequest struct {
	Price    string `json:"price,omitempty"`
	Quantity string `json:"quantity,omitempty"`
}

// ModifyOrderResponse represents the response after modifying an order
type ModifyOrderResponse struct {
	OldOrderID string       `json:"old_order_id"`
	Order      *Order       `json:"order"`
	Match      *MatchResult `json:"match,omitempty"`
}

// ListOrdersRequest represents the request to list orders
type ListOrdersRequest struct {
	Trader   string `json:"trader"`
	MarketID string `json:"market_id,omitempty"`
	Status   string `json:"status,omitempty"`
	Limit    int    `json:"limit,omitempty"`
	Cursor   string `json:"cursor,omitempty"`
}

// ListOrdersResponse represents the response for listing orders
type ListOrdersResponse struct {
	Orders     []*Order `json:"orders"`
	NextCursor string   `json:"next_cursor,omitempty"`
	Total      int      `json:"total"`
}

// ClosePositionRequest represents the request to close a position
type ClosePositionRequest struct {
	Trader   string `json:"trader"`
	MarketID string `json:"market_id"`
	Size     string `json:"size,omitempty"`
	Price    string `json:"price,omitempty"`
}

// ClosePositionResponse represents the response after closing a position
type ClosePositionResponse struct {
	MarketID    string   `json:"market_id"`
	ClosedSize  string   `json:"closed_size"`
	ClosePrice  string   `json:"close_price"`
	RealizedPnl string   `json:"realized_pnl"`
	Account     *Account `json:"account"`
}

// DepositRequest represents the request to deposit funds
type DepositRequest struct {
	Trader string `json:"trader"`
	Amount string `json:"amount"`
}

// WithdrawRequest represents the request to withdraw funds
type WithdrawRequest struct {
	Trader string `json:"trader"`
	Amount string `json:"amount"`
}

// AccountResponse represents the response for account operations
type AccountResponse struct {
	Account *Account `json:"account"`
}

// OrderService defines the interface for order operations
type OrderService interface {
	PlaceOrder(ctx context.Context, req *PlaceOrderRequest) (*PlaceOrderResponse, error)
	CancelOrder(ctx context.Context, trader, orderID string) (*CancelOrderResponse, error)
	ModifyOrder(ctx context.Context, trader, orderID string, req *ModifyOrderRequest) (*ModifyOrderResponse, error)
	GetOrder(ctx context.Context, orderID string) (*Order, error)
	ListOrders(ctx context.Context, req *ListOrdersRequest) (*ListOrdersResponse, error)
}

// PositionService defines the interface for position operations
type PositionService interface {
	GetPositions(ctx context.Context, trader string) ([]*Position, error)
	GetPosition(ctx context.Context, trader, marketID string) (*Position, error)
	ClosePosition(ctx context.Context, req *ClosePositionRequest) (*ClosePositionResponse, error)
}

// AccountService defines the interface for account operations
type AccountService interface {
	GetAccount(ctx context.Context, trader string) (*Account, error)
	Deposit(ctx context.Context, req *DepositRequest) (*AccountResponse, error)
	Withdraw(ctx context.Context, req *WithdrawRequest) (*AccountResponse, error)
}

// Helper function to get current timestamp in milliseconds
func NowMillis() int64 {
	return time.Now().UnixMilli()
}
