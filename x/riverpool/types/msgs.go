package types

import (
	"errors"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Message types
const (
	TypeMsgDeposit              = "deposit"
	TypeMsgRequestWithdrawal    = "request_withdrawal"
	TypeMsgClaimWithdrawal      = "claim_withdrawal"
	TypeMsgCancelWithdrawal     = "cancel_withdrawal"
	TypeMsgCreateCommunityPool  = "create_community_pool"
	TypeMsgUpdateDDGuard        = "update_dd_guard"
)

// MsgDeposit defines the Deposit message
type MsgDeposit struct {
	Depositor  string `json:"depositor"`
	PoolID     string `json:"pool_id"`
	Amount     string `json:"amount"`
	InviteCode string `json:"invite_code,omitempty"`
}

// Route implements sdk.Msg
func (msg MsgDeposit) Route() string { return ModuleName }

// Type implements sdk.Msg
func (msg MsgDeposit) Type() string { return TypeMsgDeposit }

// ValidateBasic implements sdk.Msg
func (msg MsgDeposit) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Depositor); err != nil {
		return err
	}
	if msg.PoolID == "" {
		return ErrPoolNotFound
	}
	return nil
}

// GetSigners implements sdk.Msg
func (msg MsgDeposit) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(msg.Depositor)
	return []sdk.AccAddress{addr}
}

// ProtoMessage implements proto.Message
func (*MsgDeposit) ProtoMessage() {}

// Reset implements proto.Message
func (msg *MsgDeposit) Reset() { *msg = MsgDeposit{} }

// String implements proto.Message
func (msg MsgDeposit) String() string {
	return fmt.Sprintf("MsgDeposit{Depositor: %s, PoolID: %s, Amount: %s}", msg.Depositor, msg.PoolID, msg.Amount)
}

// MsgDepositResponse defines the Deposit response
type MsgDepositResponse struct {
	DepositID      string `json:"deposit_id"`
	SharesReceived string `json:"shares_received"`
	NAVAtDeposit   string `json:"nav_at_deposit"`
	UnlockAt       int64  `json:"unlock_at"`
}

// MsgRequestWithdrawal defines the RequestWithdrawal message
type MsgRequestWithdrawal struct {
	Withdrawer string `json:"withdrawer"`
	PoolID     string `json:"pool_id"`
	Shares     string `json:"shares"`
}

// Route implements sdk.Msg
func (msg MsgRequestWithdrawal) Route() string { return ModuleName }

// Type implements sdk.Msg
func (msg MsgRequestWithdrawal) Type() string { return TypeMsgRequestWithdrawal }

// ValidateBasic implements sdk.Msg
func (msg MsgRequestWithdrawal) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Withdrawer); err != nil {
		return err
	}
	if msg.PoolID == "" {
		return ErrPoolNotFound
	}
	return nil
}

// GetSigners implements sdk.Msg
func (msg MsgRequestWithdrawal) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(msg.Withdrawer)
	return []sdk.AccAddress{addr}
}

// ProtoMessage implements proto.Message
func (*MsgRequestWithdrawal) ProtoMessage() {}

// Reset implements proto.Message
func (msg *MsgRequestWithdrawal) Reset() { *msg = MsgRequestWithdrawal{} }

// String implements proto.Message
func (msg MsgRequestWithdrawal) String() string {
	return fmt.Sprintf("MsgRequestWithdrawal{Withdrawer: %s, PoolID: %s, Shares: %s}", msg.Withdrawer, msg.PoolID, msg.Shares)
}

// MsgRequestWithdrawalResponse defines the RequestWithdrawal response
type MsgRequestWithdrawalResponse struct {
	WithdrawalID    string `json:"withdrawal_id"`
	SharesRequested string `json:"shares_requested"`
	EstimatedAmount string `json:"estimated_amount"`
	AvailableAt     int64  `json:"available_at"`
	QueuePosition   string `json:"queue_position"`
}

// MsgClaimWithdrawal defines the ClaimWithdrawal message
type MsgClaimWithdrawal struct {
	Withdrawer   string `json:"withdrawer"`
	WithdrawalID string `json:"withdrawal_id"`
}

// Route implements sdk.Msg
func (msg MsgClaimWithdrawal) Route() string { return ModuleName }

// Type implements sdk.Msg
func (msg MsgClaimWithdrawal) Type() string { return TypeMsgClaimWithdrawal }

// ValidateBasic implements sdk.Msg
func (msg MsgClaimWithdrawal) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Withdrawer); err != nil {
		return err
	}
	if msg.WithdrawalID == "" {
		return ErrWithdrawalNotFound
	}
	return nil
}

// GetSigners implements sdk.Msg
func (msg MsgClaimWithdrawal) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(msg.Withdrawer)
	return []sdk.AccAddress{addr}
}

// ProtoMessage implements proto.Message
func (*MsgClaimWithdrawal) ProtoMessage() {}

// Reset implements proto.Message
func (msg *MsgClaimWithdrawal) Reset() { *msg = MsgClaimWithdrawal{} }

// String implements proto.Message
func (msg MsgClaimWithdrawal) String() string {
	return fmt.Sprintf("MsgClaimWithdrawal{Withdrawer: %s, WithdrawalID: %s}", msg.Withdrawer, msg.WithdrawalID)
}

// MsgClaimWithdrawalResponse defines the ClaimWithdrawal response
type MsgClaimWithdrawalResponse struct {
	AmountReceived  string `json:"amount_received"`
	SharesRedeemed  string `json:"shares_redeemed"`
	RemainingShares string `json:"remaining_shares"`
}

// MsgCancelWithdrawal defines the CancelWithdrawal message
type MsgCancelWithdrawal struct {
	Withdrawer   string `json:"withdrawer"`
	WithdrawalID string `json:"withdrawal_id"`
}

// Route implements sdk.Msg
func (msg MsgCancelWithdrawal) Route() string { return ModuleName }

// Type implements sdk.Msg
func (msg MsgCancelWithdrawal) Type() string { return TypeMsgCancelWithdrawal }

// ValidateBasic implements sdk.Msg
func (msg MsgCancelWithdrawal) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Withdrawer); err != nil {
		return err
	}
	if msg.WithdrawalID == "" {
		return ErrWithdrawalNotFound
	}
	return nil
}

// GetSigners implements sdk.Msg
func (msg MsgCancelWithdrawal) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(msg.Withdrawer)
	return []sdk.AccAddress{addr}
}

// ProtoMessage implements proto.Message
func (*MsgCancelWithdrawal) ProtoMessage() {}

// Reset implements proto.Message
func (msg *MsgCancelWithdrawal) Reset() { *msg = MsgCancelWithdrawal{} }

// String implements proto.Message
func (msg MsgCancelWithdrawal) String() string {
	return fmt.Sprintf("MsgCancelWithdrawal{Withdrawer: %s, WithdrawalID: %s}", msg.Withdrawer, msg.WithdrawalID)
}

// MsgCancelWithdrawalResponse defines the CancelWithdrawal response
type MsgCancelWithdrawalResponse struct {
	SharesReturned string `json:"shares_returned"`
}

// MsgCreateCommunityPool defines the CreateCommunityPool message (Phase 3)
type MsgCreateCommunityPool struct {
	Owner              string `json:"owner"`
	Name               string `json:"name"`
	Description        string `json:"description"`
	MinDeposit         string `json:"min_deposit"`
	MaxDeposit         string `json:"max_deposit"`
	OwnerStakeAmount   string `json:"owner_stake_amount"`
	ManagementFeeRate  string `json:"management_fee_rate"`
	PerformanceFeeRate string `json:"performance_fee_rate"`
	IsPrivate          bool   `json:"is_private"`
}

// Route implements sdk.Msg
func (msg MsgCreateCommunityPool) Route() string { return ModuleName }

// Type implements sdk.Msg
func (msg MsgCreateCommunityPool) Type() string { return TypeMsgCreateCommunityPool }

// ValidateBasic implements sdk.Msg
func (msg MsgCreateCommunityPool) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Owner); err != nil {
		return err
	}
	if msg.Name == "" {
		return errors.New("pool name cannot be empty")
	}
	return nil
}

// GetSigners implements sdk.Msg
func (msg MsgCreateCommunityPool) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(msg.Owner)
	return []sdk.AccAddress{addr}
}

// ProtoMessage implements proto.Message
func (*MsgCreateCommunityPool) ProtoMessage() {}

// Reset implements proto.Message
func (msg *MsgCreateCommunityPool) Reset() { *msg = MsgCreateCommunityPool{} }

// String implements proto.Message
func (msg MsgCreateCommunityPool) String() string {
	return fmt.Sprintf("MsgCreateCommunityPool{Owner: %s, Name: %s}", msg.Owner, msg.Name)
}

// MsgCreateCommunityPoolResponse defines the CreateCommunityPool response
type MsgCreateCommunityPoolResponse struct {
	PoolID     string `json:"pool_id"`
	InviteCode string `json:"invite_code"`
}

// MsgUpdateDDGuard defines the UpdateDDGuard message
type MsgUpdateDDGuard struct {
	Authority string `json:"authority"`
	PoolID    string `json:"pool_id"`
}

// Route implements sdk.Msg
func (msg MsgUpdateDDGuard) Route() string { return ModuleName }

// Type implements sdk.Msg
func (msg MsgUpdateDDGuard) Type() string { return TypeMsgUpdateDDGuard }

// ValidateBasic implements sdk.Msg
func (msg MsgUpdateDDGuard) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Authority); err != nil {
		return err
	}
	if msg.PoolID == "" {
		return ErrPoolNotFound
	}
	return nil
}

// GetSigners implements sdk.Msg
func (msg MsgUpdateDDGuard) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(msg.Authority)
	return []sdk.AccAddress{addr}
}

// ProtoMessage implements proto.Message
func (*MsgUpdateDDGuard) ProtoMessage() {}

// Reset implements proto.Message
func (msg *MsgUpdateDDGuard) Reset() { *msg = MsgUpdateDDGuard{} }

// String implements proto.Message
func (msg MsgUpdateDDGuard) String() string {
	return fmt.Sprintf("MsgUpdateDDGuard{Authority: %s, PoolID: %s}", msg.Authority, msg.PoolID)
}

// MsgUpdateDDGuardResponse defines the UpdateDDGuard response
type MsgUpdateDDGuardResponse struct {
	NewLevel        string `json:"new_level"`
	CurrentDrawdown string `json:"current_drawdown"`
}

// Ensure all messages implement sdk.Msg interface
var (
	_ sdk.Msg = &MsgDeposit{}
	_ sdk.Msg = &MsgRequestWithdrawal{}
	_ sdk.Msg = &MsgClaimWithdrawal{}
	_ sdk.Msg = &MsgCancelWithdrawal{}
	_ sdk.Msg = &MsgCreateCommunityPool{}
	_ sdk.Msg = &MsgUpdateDDGuard{}
)
