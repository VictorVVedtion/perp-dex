package types

import (
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// RegisterInterfaces registers the module's interface types
func RegisterInterfaces(registry cdctypes.InterfaceRegistry) {
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgPlaceOrder{},
		&MsgCancelOrder{},
	)
}

// Message types for orderbook module
const (
	TypeMsgPlaceOrder  = "place_order"
	TypeMsgCancelOrder = "cancel_order"
)

// ValidateBasic for MsgPlaceOrder
func (msg *MsgPlaceOrder) ValidateBasic() error {
	if msg.Trader == "" {
		return ErrInvalidTrader
	}
	if msg.MarketId == "" {
		return ErrInvalidMarketID
	}
	return nil
}

// GetSigners returns the signer addresses for MsgPlaceOrder
func (msg *MsgPlaceOrder) GetSigners() []sdk.AccAddress {
	trader, _ := sdk.AccAddressFromBech32(msg.Trader)
	return []sdk.AccAddress{trader}
}

// ValidateBasic for MsgCancelOrder
func (msg *MsgCancelOrder) ValidateBasic() error {
	if msg.Trader == "" {
		return ErrInvalidTrader
	}
	if msg.OrderId == "" {
		return ErrOrderNotFound
	}
	return nil
}

// GetSigners returns the signer addresses for MsgCancelOrder
func (msg *MsgCancelOrder) GetSigners() []sdk.AccAddress {
	trader, _ := sdk.AccAddressFromBech32(msg.Trader)
	return []sdk.AccAddress{trader}
}
