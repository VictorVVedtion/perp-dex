package types

// Message types for orderbook module
const (
	TypeMsgPlaceOrder  = "place_order"
	TypeMsgCancelOrder = "cancel_order"
)

// OrderSide constants for CLI (use different names to avoid redeclaration)
const (
	CLIOrderSideBuy  = 1
	CLIOrderSideSell = 2
)

// OrderType constants for CLI (use different names to avoid redeclaration)
const (
	CLIOrderTypeLimit  = 1
	CLIOrderTypeMarket = 2
)

// MsgPlaceOrder represents a place order message
type MsgPlaceOrder struct {
	Trader    string `json:"trader"`
	MarketID  string `json:"market_id"`
	Side      int32  `json:"side"`       // 1=buy, 2=sell
	OrderType int32  `json:"order_type"` // 1=limit, 2=market
	Price     string `json:"price"`
	Quantity  string `json:"quantity"`
}

// MsgCancelOrder represents a cancel order message
type MsgCancelOrder struct {
	Trader  string `json:"trader"`
	OrderID string `json:"order_id"`
}

// Proto interface implementations for MsgPlaceOrder
func (msg *MsgPlaceOrder) Reset()         { *msg = MsgPlaceOrder{} }
func (msg *MsgPlaceOrder) String() string { return msg.Trader }
func (msg *MsgPlaceOrder) ProtoMessage()  {}

// Proto interface implementations for MsgCancelOrder
func (msg *MsgCancelOrder) Reset()         { *msg = MsgCancelOrder{} }
func (msg *MsgCancelOrder) String() string { return msg.OrderID }
func (msg *MsgCancelOrder) ProtoMessage()  {}

// ValidateBasic for MsgPlaceOrder
func (msg *MsgPlaceOrder) ValidateBasic() error {
	if msg.Trader == "" {
		return ErrInvalidTrader
	}
	if msg.MarketID == "" {
		return ErrInvalidMarketID
	}
	return nil
}

// ValidateBasic for MsgCancelOrder
func (msg *MsgCancelOrder) ValidateBasic() error {
	if msg.Trader == "" {
		return ErrInvalidTrader
	}
	if msg.OrderID == "" {
		return ErrOrderNotFound
	}
	return nil
}
