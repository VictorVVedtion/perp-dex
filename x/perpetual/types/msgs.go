package types

import (
	"context"

	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// RegisterInterfaces registers the module's interface types
func RegisterInterfaces(registry cdctypes.InterfaceRegistry) {
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgDeposit{},
		&MsgWithdraw{},
	)
}

// Message types for perpetual module
const (
	TypeMsgDeposit  = "deposit"
	TypeMsgWithdraw = "withdraw"
)

// MsgServer defines the perpetual module's gRPC message service
type MsgServer interface {
	Deposit(context.Context, *MsgDeposit) (*MsgDepositResponse, error)
	Withdraw(context.Context, *MsgWithdraw) (*MsgWithdrawResponse, error)
}

// RegisterMsgServer registers the MsgServer to the configurator's MsgServer
func RegisterMsgServer(s interface{}, srv MsgServer) {
	// This is a placeholder - in production, this would use gRPC registration
	// For now, the messages are handled through the module's handler
}

// MsgDeposit represents a margin deposit message
type MsgDeposit struct {
	Trader string `json:"trader"`
	Amount string `json:"amount"`
}

// MsgWithdraw represents a margin withdraw message
type MsgWithdraw struct {
	Trader string `json:"trader"`
	Amount string `json:"amount"`
}

// Proto interface implementations for MsgDeposit
func (msg *MsgDeposit) Reset()         { *msg = MsgDeposit{} }
func (msg *MsgDeposit) String() string { return msg.Trader }
func (msg *MsgDeposit) ProtoMessage()  {}

// XXX_MessageName returns the message type URL for MsgDeposit
func (msg *MsgDeposit) XXX_MessageName() string {
	return "perpdex.perpetual.v1.MsgDeposit"
}

// Proto interface implementations for MsgWithdraw
func (msg *MsgWithdraw) Reset()         { *msg = MsgWithdraw{} }
func (msg *MsgWithdraw) String() string { return msg.Trader }
func (msg *MsgWithdraw) ProtoMessage()  {}

// XXX_MessageName returns the message type URL for MsgWithdraw
func (msg *MsgWithdraw) XXX_MessageName() string {
	return "perpdex.perpetual.v1.MsgWithdraw"
}

// ValidateBasic for MsgDeposit
func (msg *MsgDeposit) ValidateBasic() error {
	if msg.Trader == "" {
		return ErrUnauthorized
	}
	return nil
}

// GetSigners returns the signer addresses for MsgDeposit
func (msg *MsgDeposit) GetSigners() []sdk.AccAddress {
	trader, _ := sdk.AccAddressFromBech32(msg.Trader)
	return []sdk.AccAddress{trader}
}

// ValidateBasic for MsgWithdraw
func (msg *MsgWithdraw) ValidateBasic() error {
	if msg.Trader == "" {
		return ErrUnauthorized
	}
	return nil
}

// GetSigners returns the signer addresses for MsgWithdraw
func (msg *MsgWithdraw) GetSigners() []sdk.AccAddress {
	trader, _ := sdk.AccAddressFromBech32(msg.Trader)
	return []sdk.AccAddress{trader}
}

// MsgDepositResponse is the response for MsgDeposit
type MsgDepositResponse struct {
	NewBalance string `json:"new_balance"`
}

// Proto interface implementations for MsgDepositResponse
func (msg *MsgDepositResponse) Reset()         { *msg = MsgDepositResponse{} }
func (msg *MsgDepositResponse) String() string { return msg.NewBalance }
func (msg *MsgDepositResponse) ProtoMessage()  {}

// MsgWithdrawResponse is the response for MsgWithdraw
type MsgWithdrawResponse struct {
	NewBalance string `json:"new_balance"`
}

// Proto interface implementations for MsgWithdrawResponse
func (msg *MsgWithdrawResponse) Reset()         { *msg = MsgWithdrawResponse{} }
func (msg *MsgWithdrawResponse) String() string { return msg.NewBalance }
func (msg *MsgWithdrawResponse) ProtoMessage()  {}
