package keeper

import (
	"context"
	"strconv"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/openalpha/perp-dex/x/riverpool/types"
)

// MsgServer defines the riverpool MsgServer
type MsgServer struct {
	keeper *Keeper
}

// NewMsgServerImpl creates a new MsgServer instance
func NewMsgServerImpl(keeper *Keeper) *MsgServer {
	return &MsgServer{keeper: keeper}
}

// Deposit handles MsgDeposit
func (m *MsgServer) Deposit(ctx context.Context, msg *types.MsgDeposit) (*types.MsgDepositResponse, error) {
	amount, err := math.LegacyNewDecFromStr(msg.Amount)
	if err != nil {
		return nil, err
	}

	deposit, err := m.keeper.Deposit(ctx, msg.Depositor, msg.PoolID, amount, msg.InviteCode)
	if err != nil {
		return nil, err
	}

	return &types.MsgDepositResponse{
		DepositID:      deposit.DepositID,
		SharesReceived: deposit.Shares.String(),
		NAVAtDeposit:   deposit.NAVAtDeposit.String(),
		UnlockAt:       deposit.UnlockAt,
	}, nil
}

// RequestWithdrawal handles MsgRequestWithdrawal
func (m *MsgServer) RequestWithdrawal(ctx context.Context, msg *types.MsgRequestWithdrawal) (*types.MsgRequestWithdrawalResponse, error) {
	shares, err := math.LegacyNewDecFromStr(msg.Shares)
	if err != nil {
		return nil, err
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	withdrawal, err := m.keeper.RequestWithdrawal(ctx, msg.Withdrawer, msg.PoolID, shares)
	if err != nil {
		return nil, err
	}

	// Get pool for estimated amount
	pool := m.keeper.GetPool(sdkCtx, msg.PoolID)
	estimatedAmount := math.LegacyZeroDec()
	if pool != nil {
		estimatedAmount = pool.CalculateValueForShares(shares)
	}

	// Get queue position
	queuePosition := m.keeper.GetQueuePosition(sdkCtx, msg.PoolID, withdrawal.WithdrawalID)

	return &types.MsgRequestWithdrawalResponse{
		WithdrawalID:    withdrawal.WithdrawalID,
		SharesRequested: withdrawal.SharesRequested.String(),
		EstimatedAmount: estimatedAmount.String(),
		AvailableAt:     withdrawal.AvailableAt,
		QueuePosition:   strconv.Itoa(queuePosition),
	}, nil
}

// ClaimWithdrawal handles MsgClaimWithdrawal
func (m *MsgServer) ClaimWithdrawal(ctx context.Context, msg *types.MsgClaimWithdrawal) (*types.MsgClaimWithdrawalResponse, error) {
	withdrawal, amountReceived, err := m.keeper.ClaimWithdrawal(ctx, msg.Withdrawer, msg.WithdrawalID)
	if err != nil {
		return nil, err
	}

	remainingShares := withdrawal.SharesRequested.Sub(withdrawal.SharesRedeemed)

	return &types.MsgClaimWithdrawalResponse{
		AmountReceived:  amountReceived.String(),
		SharesRedeemed:  withdrawal.SharesRedeemed.String(),
		RemainingShares: remainingShares.String(),
	}, nil
}

// CancelWithdrawal handles MsgCancelWithdrawal
func (m *MsgServer) CancelWithdrawal(ctx context.Context, msg *types.MsgCancelWithdrawal) (*types.MsgCancelWithdrawalResponse, error) {
	withdrawal, err := m.keeper.CancelWithdrawal(ctx, msg.Withdrawer, msg.WithdrawalID)
	if err != nil {
		return nil, err
	}

	sharesReturned := withdrawal.SharesRequested.Sub(withdrawal.SharesRedeemed)

	return &types.MsgCancelWithdrawalResponse{
		SharesReturned: sharesReturned.String(),
	}, nil
}

// CreateCommunityPool handles MsgCreateCommunityPool (Phase 3 - stub for now)
func (m *MsgServer) CreateCommunityPool(ctx context.Context, msg *types.MsgCreateCommunityPool) (*types.MsgCreateCommunityPoolResponse, error) {
	// Phase 3 implementation
	return &types.MsgCreateCommunityPoolResponse{
		PoolID:     "",
		InviteCode: "",
	}, types.ErrPoolNotFound // Not implemented yet
}

// UpdateDDGuard handles MsgUpdateDDGuard (admin only)
func (m *MsgServer) UpdateDDGuard(ctx context.Context, msg *types.MsgUpdateDDGuard) (*types.MsgUpdateDDGuardResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// Verify authority
	if msg.Authority != m.keeper.GetAuthority() {
		return nil, types.ErrUnauthorized
	}

	// Update NAV which triggers DDGuard check
	m.keeper.UpdatePoolNAV(sdkCtx, msg.PoolID)

	// Get updated pool
	pool := m.keeper.GetPool(sdkCtx, msg.PoolID)
	if pool == nil {
		return nil, types.ErrPoolNotFound
	}

	return &types.MsgUpdateDDGuardResponse{
		NewLevel:        pool.DDGuardLevel,
		CurrentDrawdown: pool.CurrentDrawdown.String(),
	}, nil
}
