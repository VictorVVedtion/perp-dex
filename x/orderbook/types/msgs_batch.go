package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// MsgPlaceOrderBatch represents a batch of orders to place in a single transaction
// This significantly reduces per-order overhead and improves throughput
type MsgPlaceOrderBatch struct {
	Trader string            `json:"trader"`
	Orders []*BatchOrderItem `json:"orders"`
}

// BatchOrderItem represents a single order in a batch
type BatchOrderItem struct {
	MarketId  string `json:"market_id"`
	Side      string `json:"side"`       // "buy" or "sell"
	OrderType string `json:"order_type"` // "limit" or "market"
	Price     string `json:"price"`
	Quantity  string `json:"quantity"`
	Leverage  string `json:"leverage,omitempty"`
}

// MsgPlaceOrderBatchResponse contains results for each order in the batch
type MsgPlaceOrderBatchResponse struct {
	Results []*OrderResult `json:"results"`
}

// OrderResult represents the result of a single order operation
type OrderResult struct {
	OrderId   string `json:"order_id"`
	Success   bool   `json:"success"`
	Error     string `json:"error,omitempty"`
	FilledQty string `json:"filled_qty,omitempty"`
	AvgPrice  string `json:"avg_price,omitempty"`
}

// ValidateBasic validates the batch message
func (msg *MsgPlaceOrderBatch) ValidateBasic() error {
	if msg.Trader == "" {
		return ErrInvalidTrader
	}
	if len(msg.Orders) == 0 {
		return ErrInvalidOrder
	}
	if len(msg.Orders) > 100 {
		return ErrBatchTooLarge
	}
	for _, order := range msg.Orders {
		if order.MarketId == "" {
			return ErrInvalidMarketID
		}
		if order.Side != "buy" && order.Side != "sell" {
			return ErrInvalidSide
		}
	}
	return nil
}

// GetSigners returns the signer addresses for MsgPlaceOrderBatch
func (msg *MsgPlaceOrderBatch) GetSigners() []sdk.AccAddress {
	trader, _ := sdk.AccAddressFromBech32(msg.Trader)
	return []sdk.AccAddress{trader}
}

// MsgCancelOrderBatch represents a batch of orders to cancel
type MsgCancelOrderBatch struct {
	Trader   string   `json:"trader"`
	OrderIds []string `json:"order_ids"`
}

// MsgCancelOrderBatchResponse contains results for each cancellation
type MsgCancelOrderBatchResponse struct {
	Results []*CancelResult `json:"results"`
}

// CancelResult represents the result of a single cancel operation
type CancelResult struct {
	OrderId   string `json:"order_id"`
	Success   bool   `json:"success"`
	Error     string `json:"error,omitempty"`
	Cancelled bool   `json:"cancelled"`
}

// ValidateBasic validates the cancel batch message
func (msg *MsgCancelOrderBatch) ValidateBasic() error {
	if msg.Trader == "" {
		return ErrInvalidTrader
	}
	if len(msg.OrderIds) == 0 {
		return ErrOrderNotFound
	}
	if len(msg.OrderIds) > 100 {
		return ErrBatchTooLarge
	}
	return nil
}

// GetSigners returns the signer addresses for MsgCancelOrderBatch
func (msg *MsgCancelOrderBatch) GetSigners() []sdk.AccAddress {
	trader, _ := sdk.AccAddressFromBech32(msg.Trader)
	return []sdk.AccAddress{trader}
}

// Message type constants
const (
	TypeMsgPlaceOrderBatch  = "place_order_batch"
	TypeMsgCancelOrderBatch = "cancel_order_batch"
)
