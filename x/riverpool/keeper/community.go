package keeper

import (
	"context"
	"encoding/json"
	"time"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/openalpha/perp-dex/x/riverpool/types"
)

// CommunityPoolConfig defines parameters for creating a community pool
type CommunityPoolConfig struct {
	Name                 string
	Description          string
	Owner                string
	MinDeposit           math.LegacyDec
	MaxDeposit           math.LegacyDec // 0 = no max
	LockPeriodDays       int64
	RedemptionDelayDays  int64
	DailyRedemptionLimit math.LegacyDec // e.g., 0.15 for 15%
	ManagementFee        math.LegacyDec // Annual % (e.g., 0.02 for 2%)
	PerformanceFee       math.LegacyDec // % of profits (e.g., 0.20 for 20%)
	OwnerMinStake        math.LegacyDec // Min % owner must stake (e.g., 0.05 for 5%)
	IsPrivate            bool           // Requires invite code
	MaxSeats             int64          // 0 = unlimited
	MaxLeverage          math.LegacyDec // Max leverage allowed
	AllowedMarkets       []string       // Markets owner can trade
	Tags                 []string       // Pool tags for discovery
}

// CreateCommunityPool creates a new community pool
func (k *Keeper) CreateCommunityPool(
	ctx sdk.Context,
	config CommunityPoolConfig,
) (*types.Pool, error) {
	// Validate config
	if err := k.validateCommunityPoolConfig(config); err != nil {
		return nil, err
	}

	// Generate pool ID
	poolID := k.generateCommunityPoolID(config.Owner)

	// Check if pool ID already exists
	if k.GetPool(ctx, poolID) != nil {
		return nil, types.ErrPoolAlreadyExists
	}

	now := time.Now().Unix()

	// Create the pool
	pool := &types.Pool{
		PoolID:              poolID,
		PoolType:            types.PoolTypeCommunity,
		Name:                config.Name,
		Description:         config.Description,
		Owner:               config.Owner,
		Status:              types.PoolStatusActive,
		TotalDeposits:       math.LegacyZeroDec(),
		TotalShares:         math.LegacyZeroDec(),
		NAV:                 math.LegacyOneDec(),
		HighWaterMark:       math.LegacyOneDec(),
		CurrentDrawdown:     math.LegacyZeroDec(),
		DDGuardLevel:        types.DDGuardLevelNormal,
		MinDeposit:          config.MinDeposit,
		MaxDeposit:          config.MaxDeposit,
		LockPeriodDays:      config.LockPeriodDays,
		RedemptionDelayDays: config.RedemptionDelayDays,
		DailyRedemptionLimit: config.DailyRedemptionLimit,
		SeatsAvailable:      config.MaxSeats,
		SeatsTotal:          config.MaxSeats,
		CreatedAt:           now,
		UpdatedAt:           now,
	}

	// Set community pool specific fields
	pool.ManagementFee = config.ManagementFee
	pool.PerformanceFee = config.PerformanceFee
	pool.OwnerMinStake = config.OwnerMinStake
	pool.OwnerCurrentStake = math.LegacyZeroDec()
	pool.IsPrivate = config.IsPrivate
	pool.TotalHolders = 0
	pool.MaxLeverage = config.MaxLeverage
	pool.AllowedMarkets = config.AllowedMarkets
	pool.Tags = config.Tags

	// Store the pool
	k.SetPool(ctx, pool)

	// Initialize DDGuard state
	ddState := &types.DDGuardState{
		PoolID:           poolID,
		Level:            types.DDGuardLevelNormal,
		PeakNAV:          math.LegacyOneDec(),
		CurrentNAV:       math.LegacyOneDec(),
		DrawdownPercent:  math.LegacyZeroDec(),
		MaxExposureLimit: math.LegacyOneDec(),
		TriggeredAt:      now,
		LastCheckedAt:    now,
	}
	k.SetDDGuardState(ctx, ddState)

	// Generate invite code if private
	if config.IsPrivate {
		k.GenerateInviteCode(ctx, poolID, 0, 0) // Unlimited uses, never expires
	}

	// Emit event
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"riverpool_community_pool_created",
			sdk.NewAttribute("pool_id", poolID),
			sdk.NewAttribute("owner", config.Owner),
			sdk.NewAttribute("name", config.Name),
			sdk.NewAttribute("is_private", math.NewInt(boolToInt(config.IsPrivate)).String()),
		),
	)

	k.logger.Info("Community pool created",
		"pool_id", poolID,
		"owner", config.Owner,
		"name", config.Name,
	)

	return pool, nil
}

// validateCommunityPoolConfig validates community pool configuration
func (k *Keeper) validateCommunityPoolConfig(config CommunityPoolConfig) error {
	if config.Name == "" {
		return types.ErrInvalidPoolName
	}

	if config.Owner == "" {
		return types.ErrInvalidOwner
	}

	// Min deposit must be at least $10
	minAllowed := math.LegacyNewDec(10)
	if config.MinDeposit.LT(minAllowed) {
		return types.ErrInvalidMinDeposit
	}

	// Owner min stake must be at least 5%
	minOwnerStake := math.LegacyMustNewDecFromStr("0.05")
	if config.OwnerMinStake.LT(minOwnerStake) {
		return types.ErrInvalidOwnerStake
	}

	// Management fee cannot exceed 5%
	maxManagementFee := math.LegacyMustNewDecFromStr("0.05")
	if config.ManagementFee.GT(maxManagementFee) {
		return types.ErrInvalidManagementFee
	}

	// Performance fee cannot exceed 50%
	maxPerformanceFee := math.LegacyMustNewDecFromStr("0.50")
	if config.PerformanceFee.GT(maxPerformanceFee) {
		return types.ErrInvalidPerformanceFee
	}

	// Daily redemption limit must be between 5% and 100%
	minRedemptionLimit := math.LegacyMustNewDecFromStr("0.05")
	maxRedemptionLimit := math.LegacyOneDec()
	if config.DailyRedemptionLimit.LT(minRedemptionLimit) || config.DailyRedemptionLimit.GT(maxRedemptionLimit) {
		return types.ErrInvalidRedemptionLimit
	}

	return nil
}

// generateCommunityPoolID generates a unique pool ID
func (k *Keeper) generateCommunityPoolID(owner string) string {
	timestamp := time.Now().UnixNano()
	// Use first 8 chars of owner + timestamp
	ownerShort := owner
	if len(owner) > 8 {
		ownerShort = owner[:8]
	}
	return "community-" + ownerShort + "-" + math.NewInt(timestamp).String()
}

// DepositOwnerStake handles the initial owner stake deposit
func (k *Keeper) DepositOwnerStake(
	ctx sdk.Context,
	owner string,
	poolID string,
	amount math.LegacyDec,
) error {
	pool := k.GetPool(ctx, poolID)
	if pool == nil {
		return types.ErrPoolNotFound
	}

	if pool.Owner != owner {
		return types.ErrNotPoolOwner
	}

	// Owner stake must meet minimum requirement
	// This is checked against total deposits after owner deposit
	minOwnerStake := pool.OwnerMinStake

	// Perform the deposit
	_, err := k.Deposit(context.Background(), owner, poolID, amount, "")
	if err != nil {
		return err
	}

	// Mark owner stake
	k.SetOwnerStake(ctx, poolID, amount)

	// After owner deposits, check if pool can accept other deposits
	pool = k.GetPool(ctx, poolID)
	ownerShare := amount.Quo(pool.TotalDeposits)

	if ownerShare.LT(minOwnerStake) {
		k.logger.Warn("Owner stake below minimum",
			"pool_id", poolID,
			"owner_share", ownerShare.String(),
			"required", minOwnerStake.String(),
		)
	}

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"riverpool_owner_stake_deposited",
			sdk.NewAttribute("pool_id", poolID),
			sdk.NewAttribute("owner", owner),
			sdk.NewAttribute("amount", amount.String()),
		),
	)

	return nil
}

// SetOwnerStake stores the owner's stake amount
func (k *Keeper) SetOwnerStake(ctx sdk.Context, poolID string, amount math.LegacyDec) {
	store := k.GetStore(ctx)
	key := append(OwnerStakeKeyPrefix, []byte(poolID)...)
	bz, err := json.Marshal(&OwnerStakeRecord{PoolID: poolID, Amount: amount})
	if err != nil {
		k.logger.Error("Failed to marshal owner stake", "error", err)
		return
	}
	store.Set(key, bz)
}

// GetOwnerStake gets the owner's stake amount
func (k *Keeper) GetOwnerStake(ctx sdk.Context, poolID string) math.LegacyDec {
	store := k.GetStore(ctx)
	key := append(OwnerStakeKeyPrefix, []byte(poolID)...)
	bz := store.Get(key)
	if bz == nil {
		return math.LegacyZeroDec()
	}
	var record OwnerStakeRecord
	if err := json.Unmarshal(bz, &record); err != nil {
		k.logger.Error("Failed to unmarshal owner stake", "error", err)
		return math.LegacyZeroDec()
	}
	return record.Amount
}

// OwnerStakeKeyPrefix is the prefix for owner stake records
var OwnerStakeKeyPrefix = []byte{0x0C}

// OwnerStakeRecord stores owner stake information
type OwnerStakeRecord struct {
	PoolID string
	Amount math.LegacyDec
}

// ValidateOwnerStake checks if owner has sufficient stake
func (k *Keeper) ValidateOwnerStake(ctx sdk.Context, poolID string) bool {
	pool := k.GetPool(ctx, poolID)
	if pool == nil {
		return false
	}

	if pool.TotalDeposits.IsZero() {
		return true // No deposits yet
	}

	ownerStake := k.GetOwnerStake(ctx, poolID)
	ownerShare := ownerStake.Quo(pool.TotalDeposits)

	return ownerShare.GTE(pool.OwnerMinStake)
}

// InviteCode represents an invite code for a private pool
type InviteCode struct {
	Code      string
	PoolID    string
	UsedCount int64
	MaxUses   int64 // 0 = unlimited
	ExpiresAt int64 // 0 = never
	CreatedAt int64
	IsActive  bool
}

// InviteCodeKeyPrefix is the prefix for invite codes
var InviteCodeKeyPrefix = []byte{0x0D}

// PoolInviteCodesKeyPrefix is the prefix for pool -> invite codes mapping
var PoolInviteCodesKeyPrefix = []byte{0x0E}

// GenerateInviteCode generates a new invite code for a pool
func (k *Keeper) GenerateInviteCode(ctx sdk.Context, poolID string, maxUses int, expiresInDays int) *InviteCode {
	now := time.Now().Unix()
	code := k.generateRandomCode()

	var expiresAt int64
	if expiresInDays > 0 {
		expiresAt = now + int64(expiresInDays)*24*60*60
	}

	inviteCode := &InviteCode{
		Code:      code,
		PoolID:    poolID,
		UsedCount: 0,
		MaxUses:   int64(maxUses),
		ExpiresAt: expiresAt,
		CreatedAt: now,
		IsActive:  true,
	}

	// Store the invite code
	k.SetInviteCode(ctx, inviteCode)

	// Add to pool's invite codes list
	k.addInviteCodeToPool(ctx, poolID, code)

	return inviteCode
}

// addInviteCodeToPool adds an invite code to a pool's list
func (k *Keeper) addInviteCodeToPool(ctx sdk.Context, poolID, code string) {
	store := k.GetStore(ctx)
	key := append(PoolInviteCodesKeyPrefix, []byte(poolID)...)

	var codes []string
	bz := store.Get(key)
	if bz != nil {
		if err := json.Unmarshal(bz, &codes); err != nil {
			k.logger.Error("Failed to unmarshal pool invite codes", "error", err)
		}
	}

	codes = append(codes, code)
	bzOut, err := json.Marshal(&codes)
	if err != nil {
		k.logger.Error("Failed to marshal pool invite codes", "error", err)
		return
	}
	store.Set(key, bzOut)
}

// GetPoolInviteCodes returns all invite codes for a pool
func (k *Keeper) GetPoolInviteCodes(ctx sdk.Context, poolID string) []*InviteCode {
	store := k.GetStore(ctx)
	key := append(PoolInviteCodesKeyPrefix, []byte(poolID)...)

	var codelist []string
	bz := store.Get(key)
	if bz != nil {
		if err := json.Unmarshal(bz, &codelist); err != nil {
			k.logger.Error("Failed to unmarshal pool invite codes", "error", err)
		}
	}

	codes := make([]*InviteCode, 0, len(codelist))
	for _, code := range codelist {
		inviteCode := k.GetInviteCode(ctx, code)
		if inviteCode != nil {
			codes = append(codes, inviteCode)
		}
	}

	return codes
}

// SetInviteCode stores an invite code
func (k *Keeper) SetInviteCode(ctx sdk.Context, code *InviteCode) {
	store := k.GetStore(ctx)
	key := append(InviteCodeKeyPrefix, []byte(code.Code)...)
	bz, err := json.Marshal(code)
	if err != nil {
		k.logger.Error("Failed to marshal invite code", "error", err)
		return
	}
	store.Set(key, bz)
}

// GetInviteCode retrieves an invite code
func (k *Keeper) GetInviteCode(ctx sdk.Context, code string) *InviteCode {
	store := k.GetStore(ctx)
	key := append(InviteCodeKeyPrefix, []byte(code)...)
	bz := store.Get(key)
	if bz == nil {
		return nil
	}
	var inviteCode InviteCode
	if err := json.Unmarshal(bz, &inviteCode); err != nil {
		k.logger.Error("Failed to unmarshal invite code", "error", err)
		return nil
	}
	return &inviteCode
}

// ValidateInviteCode validates an invite code for a pool
func (k *Keeper) ValidateInviteCode(ctx sdk.Context, poolID, code string) bool {
	inviteCode := k.GetInviteCode(ctx, code)
	if inviteCode == nil {
		return false
	}

	// Check if code belongs to this pool
	if inviteCode.PoolID != poolID {
		return false
	}

	// Check if code is active
	if !inviteCode.IsActive {
		return false
	}

	// Check expiration
	if inviteCode.ExpiresAt > 0 && time.Now().Unix() > inviteCode.ExpiresAt {
		return false
	}

	// Check max uses
	if inviteCode.MaxUses > 0 && inviteCode.UsedCount >= inviteCode.MaxUses {
		return false
	}

	return true
}

// UseInviteCode marks an invite code as used
func (k *Keeper) UseInviteCode(ctx sdk.Context, code string) {
	inviteCode := k.GetInviteCode(ctx, code)
	if inviteCode == nil {
		return
	}

	inviteCode.UsedCount++
	k.SetInviteCode(ctx, inviteCode)
}

// generateRandomCode generates a random invite code
func (k *Keeper) generateRandomCode() string {
	// Use timestamp + random for simplicity
	// In production, use crypto/rand
	timestamp := time.Now().UnixNano()
	return math.NewInt(timestamp).String()[:8]
}

// CollectManagementFee collects management fee from a pool
func (k *Keeper) CollectManagementFee(ctx sdk.Context, poolID string) math.LegacyDec {
	pool := k.GetPool(ctx, poolID)
	if pool == nil || pool.PoolType != types.PoolTypeCommunity {
		return math.LegacyZeroDec()
	}

	// Calculate daily fee (annual fee / 365)
	dailyFeeRate := pool.ManagementFee.Quo(math.LegacyNewDec(365))
	feeAmount := pool.TotalDeposits.Mul(dailyFeeRate)

	if feeAmount.LTE(math.LegacyZeroDec()) {
		return math.LegacyZeroDec()
	}

	// Deduct fee from pool
	pool.TotalDeposits = pool.TotalDeposits.Sub(feeAmount)
	pool.UpdateNAV(pool.TotalDeposits)
	k.SetPool(ctx, pool)

	// Record fee collection
	k.RecordRevenue(ctx, poolID, RevenueSourceFees, feeAmount.Neg(), "", "", "Management fee")

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"riverpool_management_fee_collected",
			sdk.NewAttribute("pool_id", poolID),
			sdk.NewAttribute("amount", feeAmount.String()),
		),
	)

	return feeAmount
}

// CollectPerformanceFee collects performance fee when NAV exceeds high water mark
func (k *Keeper) CollectPerformanceFee(ctx sdk.Context, poolID string, profit math.LegacyDec) math.LegacyDec {
	pool := k.GetPool(ctx, poolID)
	if pool == nil || pool.PoolType != types.PoolTypeCommunity {
		return math.LegacyZeroDec()
	}

	if profit.LTE(math.LegacyZeroDec()) {
		return math.LegacyZeroDec()
	}

	// Calculate performance fee
	feeAmount := profit.Mul(pool.PerformanceFee)

	// Deduct fee from pool
	pool.TotalDeposits = pool.TotalDeposits.Sub(feeAmount)
	pool.UpdateNAV(pool.TotalDeposits)
	k.SetPool(ctx, pool)

	// Record fee collection
	k.RecordRevenue(ctx, poolID, RevenueSourceFees, feeAmount.Neg(), "", "", "Performance fee")

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"riverpool_performance_fee_collected",
			sdk.NewAttribute("pool_id", poolID),
			sdk.NewAttribute("profit", profit.String()),
			sdk.NewAttribute("fee", feeAmount.String()),
		),
	)

	return feeAmount
}

// UpdatePoolSettings updates pool settings (owner only)
func (k *Keeper) UpdatePoolSettings(
	ctx sdk.Context,
	owner string,
	poolID string,
	name string,
	description string,
) error {
	pool := k.GetPool(ctx, poolID)
	if pool == nil {
		return types.ErrPoolNotFound
	}

	if pool.Owner != owner {
		return types.ErrNotPoolOwner
	}

	if name != "" {
		pool.Name = name
	}
	if description != "" {
		pool.Description = description
	}
	pool.UpdatedAt = time.Now().Unix()

	k.SetPool(ctx, pool)

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"riverpool_pool_settings_updated",
			sdk.NewAttribute("pool_id", poolID),
			sdk.NewAttribute("owner", owner),
		),
	)

	return nil
}

// PausePool pauses a community pool (owner only)
func (k *Keeper) PausePool(ctx sdk.Context, owner, poolID string) error {
	pool := k.GetPool(ctx, poolID)
	if pool == nil {
		return types.ErrPoolNotFound
	}

	if pool.Owner != owner {
		return types.ErrNotPoolOwner
	}

	if pool.Status != types.PoolStatusActive {
		return types.ErrPoolNotActive
	}

	pool.Status = types.PoolStatusPaused
	pool.UpdatedAt = time.Now().Unix()
	k.SetPool(ctx, pool)

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"riverpool_pool_paused",
			sdk.NewAttribute("pool_id", poolID),
			sdk.NewAttribute("owner", owner),
			sdk.NewAttribute("reason", "owner_request"),
		),
	)

	return nil
}

// ResumePool resumes a paused community pool (owner only)
func (k *Keeper) ResumePool(ctx sdk.Context, owner, poolID string) error {
	pool := k.GetPool(ctx, poolID)
	if pool == nil {
		return types.ErrPoolNotFound
	}

	if pool.Owner != owner {
		return types.ErrNotPoolOwner
	}

	if pool.Status != types.PoolStatusPaused {
		return types.ErrPoolNotPaused
	}

	// Check DDGuard level before resuming
	ddState := k.GetDDGuardState(ctx, poolID)
	if ddState != nil && ddState.Level == types.DDGuardLevelHalt {
		return types.ErrDDGuardHalt
	}

	pool.Status = types.PoolStatusActive
	pool.UpdatedAt = time.Now().Unix()
	k.SetPool(ctx, pool)

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"riverpool_pool_resumed",
			sdk.NewAttribute("pool_id", poolID),
			sdk.NewAttribute("owner", owner),
		),
	)

	return nil
}

// ClosePool closes a community pool permanently (owner only, requires no deposits)
func (k *Keeper) ClosePool(ctx sdk.Context, owner, poolID string) error {
	pool := k.GetPool(ctx, poolID)
	if pool == nil {
		return types.ErrPoolNotFound
	}

	if pool.Owner != owner {
		return types.ErrNotPoolOwner
	}

	// Can only close if all deposits have been withdrawn
	if pool.TotalDeposits.GT(math.LegacyZeroDec()) {
		return types.ErrPoolHasDeposits
	}

	pool.Status = types.PoolStatusClosed
	pool.UpdatedAt = time.Now().Unix()
	k.SetPool(ctx, pool)

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"riverpool_pool_closed",
			sdk.NewAttribute("pool_id", poolID),
			sdk.NewAttribute("owner", owner),
		),
	)

	return nil
}

// GetCommunityPools returns all community pools with optional filters
func (k *Keeper) GetCommunityPools(ctx sdk.Context, onlyActive bool) []*types.Pool {
	allPools := k.GetAllPools(ctx)
	var communityPools []*types.Pool

	for _, pool := range allPools {
		if pool.PoolType != types.PoolTypeCommunity {
			continue
		}
		if onlyActive && pool.Status != types.PoolStatusActive {
			continue
		}
		communityPools = append(communityPools, pool)
	}

	return communityPools
}

// GetPoolsByOwner returns all pools owned by an address
func (k *Keeper) GetPoolsByOwner(ctx sdk.Context, owner string) []*types.Pool {
	allPools := k.GetAllPools(ctx)
	var ownerPools []*types.Pool

	for _, pool := range allPools {
		if pool.Owner == owner {
			ownerPools = append(ownerPools, pool)
		}
	}

	return ownerPools
}
