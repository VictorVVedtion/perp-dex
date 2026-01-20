package keeper

import (
	"context"
	"fmt"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/openalpha/perp-dex/x/perpetual/types"
)

var _ types.MsgServer = (*msgServer)(nil)

type msgServer struct {
	Keeper *Keeper
}

// NewMsgServerImpl returns an implementation of the MsgServer interface
func NewMsgServerImpl(keeper *Keeper) types.MsgServer {
	return &msgServer{Keeper: keeper}
}

// Deposit handles the MsgDeposit message
func (m *msgServer) Deposit(ctx context.Context, msg *types.MsgDeposit) (*types.MsgDepositResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// Validate message
	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}

	// Parse amount
	amount, err := math.LegacyNewDecFromStr(msg.Amount)
	if err != nil {
		return nil, fmt.Errorf("invalid amount: %w", err)
	}

	if amount.IsNegative() || amount.IsZero() {
		return nil, fmt.Errorf("amount must be positive")
	}

	// Perform deposit through keeper
	if err := m.Keeper.Deposit(sdkCtx, msg.Trader, amount); err != nil {
		return nil, err
	}

	// Get updated balance
	account := m.Keeper.GetAccount(sdkCtx, msg.Trader)
	newBalance := math.LegacyZeroDec()
	if account != nil {
		newBalance = account.Balance
	}

	return &types.MsgDepositResponse{
		NewBalance: newBalance.String(),
	}, nil
}

// Withdraw handles the MsgWithdraw message
func (m *msgServer) Withdraw(ctx context.Context, msg *types.MsgWithdraw) (*types.MsgWithdrawResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// Validate message
	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}

	// Parse amount
	amount, err := math.LegacyNewDecFromStr(msg.Amount)
	if err != nil {
		return nil, fmt.Errorf("invalid amount: %w", err)
	}

	if amount.IsNegative() || amount.IsZero() {
		return nil, fmt.Errorf("amount must be positive")
	}

	// Perform withdrawal through keeper
	if err := m.Keeper.Withdraw(sdkCtx, msg.Trader, amount); err != nil {
		return nil, err
	}

	// Get updated balance
	account := m.Keeper.GetAccount(sdkCtx, msg.Trader)
	newBalance := math.LegacyZeroDec()
	if account != nil {
		newBalance = account.Balance
	}

	return &types.MsgWithdrawResponse{
		NewBalance: newBalance.String(),
	}, nil
}
