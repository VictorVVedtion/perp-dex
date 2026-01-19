package types

// Message types for perpetual module
const (
	TypeMsgDeposit  = "deposit"
	TypeMsgWithdraw = "withdraw"
)

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

// Proto interface implementations for MsgWithdraw
func (msg *MsgWithdraw) Reset()         { *msg = MsgWithdraw{} }
func (msg *MsgWithdraw) String() string { return msg.Trader }
func (msg *MsgWithdraw) ProtoMessage()  {}

// ValidateBasic for MsgDeposit
func (msg *MsgDeposit) ValidateBasic() error {
	if msg.Trader == "" {
		return ErrUnauthorized
	}
	return nil
}

// ValidateBasic for MsgWithdraw
func (msg *MsgWithdraw) ValidateBasic() error {
	if msg.Trader == "" {
		return ErrUnauthorized
	}
	return nil
}
