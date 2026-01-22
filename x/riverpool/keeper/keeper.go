package keeper

import (
	"context"
	"encoding/json"
	"strconv"

	"cosmossdk.io/log"
	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/openalpha/perp-dex/x/riverpool/types"
)

// Store key prefixes
var (
	PoolKeyPrefix           = []byte{0x01}
	DepositKeyPrefix        = []byte{0x02}
	WithdrawalKeyPrefix     = []byte{0x03}
	DDGuardStateKeyPrefix   = []byte{0x04}
	PoolStatsKeyPrefix      = []byte{0x05}
	NAVHistoryKeyPrefix     = []byte{0x06}
	UserDepositsKeyPrefix   = []byte{0x07}
	UserWithdrawalsKeyPrefix = []byte{0x08}
	RevenueRecordKeyPrefix  = []byte{0x09}
)

// PerpetualKeeper defines the expected interface for perpetual module
type PerpetualKeeper interface {
	GetPrice(ctx sdk.Context, marketID string) interface{}
}

// BankKeeper defines the expected interface for the bank module
type BankKeeper interface {
	SendCoinsFromAccountToModule(ctx context.Context, senderAddr sdk.AccAddress, recipientModule string, amt sdk.Coins) error
	SendCoinsFromModuleToAccount(ctx context.Context, senderModule string, recipientAddr sdk.AccAddress, amt sdk.Coins) error
}

// Keeper manages the riverpool module state
type Keeper struct {
	cdc             codec.BinaryCodec
	storeKey        storetypes.StoreKey
	perpetualKeeper PerpetualKeeper
	bankKeeper      BankKeeper
	logger          log.Logger
	authority       string
}

// NewKeeper creates a new riverpool keeper
func NewKeeper(
	cdc codec.BinaryCodec,
	storeKey storetypes.StoreKey,
	perpetualKeeper PerpetualKeeper,
	bankKeeper BankKeeper,
	authority string,
	logger log.Logger,
) *Keeper {
	k := &Keeper{
		cdc:             cdc,
		storeKey:        storeKey,
		perpetualKeeper: perpetualKeeper,
		bankKeeper:      bankKeeper,
		authority:       authority,
		logger:          logger.With("module", "x/riverpool"),
	}
	return k
}

// Logger returns the module logger
func (k *Keeper) Logger() log.Logger {
	return k.logger
}

// GetAuthority returns the governance authority address
func (k *Keeper) GetAuthority() string {
	return k.authority
}

// GetStore returns the KVStore
func (k *Keeper) GetStore(ctx sdk.Context) storetypes.KVStore {
	return ctx.KVStore(k.storeKey)
}

// ============ Pool Operations ============

// SetPool saves a pool to the store
func (k *Keeper) SetPool(ctx sdk.Context, pool *types.Pool) {
	store := k.GetStore(ctx)
	key := append(PoolKeyPrefix, []byte(pool.PoolID)...)
	bz, _ := json.Marshal(pool)
	store.Set(key, bz)
}

// GetPool retrieves a pool from the store
func (k *Keeper) GetPool(ctx sdk.Context, poolID string) *types.Pool {
	store := k.GetStore(ctx)
	key := append(PoolKeyPrefix, []byte(poolID)...)
	bz := store.Get(key)
	if bz == nil {
		return nil
	}
	var pool types.Pool
	if err := json.Unmarshal(bz, &pool); err != nil {
		return nil
	}
	return &pool
}

// GetAllPools returns all pools
func (k *Keeper) GetAllPools(ctx sdk.Context) []*types.Pool {
	store := k.GetStore(ctx)
	iterator := storetypes.KVStorePrefixIterator(store, PoolKeyPrefix)
	defer iterator.Close()

	var pools []*types.Pool
	for ; iterator.Valid(); iterator.Next() {
		var pool types.Pool
		if err := json.Unmarshal(iterator.Value(), &pool); err != nil {
			continue
		}
		pools = append(pools, &pool)
	}
	return pools
}

// GetPoolsByType returns pools filtered by type
func (k *Keeper) GetPoolsByType(ctx sdk.Context, poolType string) []*types.Pool {
	allPools := k.GetAllPools(ctx)
	var filtered []*types.Pool
	for _, pool := range allPools {
		if pool.PoolType == poolType {
			filtered = append(filtered, pool)
		}
	}
	return filtered
}

// InitDefaultPools initializes Foundation LP and Main LP
func (k *Keeper) InitDefaultPools(ctx sdk.Context) {
	// Create Foundation LP if not exists
	if k.GetPool(ctx, "foundation-lp") == nil {
		foundationPool := types.NewFoundationPool()
		k.SetPool(ctx, foundationPool)
		k.SetPoolStats(ctx, types.NewPoolStats("foundation-lp"))
		k.logger.Info("Initialized Foundation LP pool")
	}

	// Create Main LP if not exists
	if k.GetPool(ctx, "main-lp") == nil {
		mainPool := types.NewMainPool()
		k.SetPool(ctx, mainPool)
		k.SetPoolStats(ctx, types.NewPoolStats("main-lp"))
		k.logger.Info("Initialized Main LP pool")
	}
}

// ============ Deposit Operations ============

// depositKey generates the key for a deposit
func depositKey(depositID string) []byte {
	return append(DepositKeyPrefix, []byte(depositID)...)
}

// userDepositsKey generates the key for user's deposits index
func userDepositsKey(user, depositID string) []byte {
	return append(UserDepositsKeyPrefix, []byte(user+":"+depositID)...)
}

// SetDeposit saves a deposit to the store
func (k *Keeper) SetDeposit(ctx sdk.Context, deposit *types.Deposit) {
	store := k.GetStore(ctx)

	// Store deposit by ID
	key := depositKey(deposit.DepositID)
	bz, _ := json.Marshal(deposit)
	store.Set(key, bz)

	// Index by user
	userKey := userDepositsKey(deposit.Depositor, deposit.DepositID)
	store.Set(userKey, []byte(deposit.DepositID))
}

// GetDeposit retrieves a deposit from the store
func (k *Keeper) GetDeposit(ctx sdk.Context, depositID string) *types.Deposit {
	store := k.GetStore(ctx)
	key := depositKey(depositID)
	bz := store.Get(key)
	if bz == nil {
		return nil
	}
	var deposit types.Deposit
	if err := json.Unmarshal(bz, &deposit); err != nil {
		return nil
	}
	return &deposit
}

// GetUserDeposits returns all deposits for a user
func (k *Keeper) GetUserDeposits(ctx sdk.Context, user string) []*types.Deposit {
	store := k.GetStore(ctx)
	prefix := append(UserDepositsKeyPrefix, []byte(user+":")...)
	iterator := storetypes.KVStorePrefixIterator(store, prefix)
	defer iterator.Close()

	var deposits []*types.Deposit
	for ; iterator.Valid(); iterator.Next() {
		depositID := string(iterator.Value())
		deposit := k.GetDeposit(ctx, depositID)
		if deposit != nil {
			deposits = append(deposits, deposit)
		}
	}
	return deposits
}

// GetPoolDeposits returns all deposits in a pool
func (k *Keeper) GetPoolDeposits(ctx sdk.Context, poolID string) []*types.Deposit {
	store := k.GetStore(ctx)
	iterator := storetypes.KVStorePrefixIterator(store, DepositKeyPrefix)
	defer iterator.Close()

	var deposits []*types.Deposit
	for ; iterator.Valid(); iterator.Next() {
		var deposit types.Deposit
		if err := json.Unmarshal(iterator.Value(), &deposit); err != nil {
			continue
		}
		if deposit.PoolID == poolID {
			deposits = append(deposits, &deposit)
		}
	}
	return deposits
}

// ============ Withdrawal Operations ============

// withdrawalKey generates the key for a withdrawal
func withdrawalKey(withdrawalID string) []byte {
	return append(WithdrawalKeyPrefix, []byte(withdrawalID)...)
}

// userWithdrawalsKey generates the key for user's withdrawals index
func userWithdrawalsKey(user, withdrawalID string) []byte {
	return append(UserWithdrawalsKeyPrefix, []byte(user+":"+withdrawalID)...)
}

// SetWithdrawal saves a withdrawal to the store
func (k *Keeper) SetWithdrawal(ctx sdk.Context, withdrawal *types.Withdrawal) {
	store := k.GetStore(ctx)

	// Store withdrawal by ID
	key := withdrawalKey(withdrawal.WithdrawalID)
	bz, _ := json.Marshal(withdrawal)
	store.Set(key, bz)

	// Index by user
	userKey := userWithdrawalsKey(withdrawal.Withdrawer, withdrawal.WithdrawalID)
	store.Set(userKey, []byte(withdrawal.WithdrawalID))
}

// GetWithdrawal retrieves a withdrawal from the store
func (k *Keeper) GetWithdrawal(ctx sdk.Context, withdrawalID string) *types.Withdrawal {
	store := k.GetStore(ctx)
	key := withdrawalKey(withdrawalID)
	bz := store.Get(key)
	if bz == nil {
		return nil
	}
	var withdrawal types.Withdrawal
	if err := json.Unmarshal(bz, &withdrawal); err != nil {
		return nil
	}
	return &withdrawal
}

// GetUserWithdrawals returns all withdrawals for a user
func (k *Keeper) GetUserWithdrawals(ctx sdk.Context, user string) []*types.Withdrawal {
	store := k.GetStore(ctx)
	prefix := append(UserWithdrawalsKeyPrefix, []byte(user+":")...)
	iterator := storetypes.KVStorePrefixIterator(store, prefix)
	defer iterator.Close()

	var withdrawals []*types.Withdrawal
	for ; iterator.Valid(); iterator.Next() {
		withdrawalID := string(iterator.Value())
		withdrawal := k.GetWithdrawal(ctx, withdrawalID)
		if withdrawal != nil {
			withdrawals = append(withdrawals, withdrawal)
		}
	}
	return withdrawals
}

// GetPendingWithdrawals returns all pending withdrawals for a pool
func (k *Keeper) GetPendingWithdrawals(ctx sdk.Context, poolID string) []*types.Withdrawal {
	store := k.GetStore(ctx)
	iterator := storetypes.KVStorePrefixIterator(store, WithdrawalKeyPrefix)
	defer iterator.Close()

	var withdrawals []*types.Withdrawal
	for ; iterator.Valid(); iterator.Next() {
		var withdrawal types.Withdrawal
		if err := json.Unmarshal(iterator.Value(), &withdrawal); err != nil {
			continue
		}
		if withdrawal.PoolID == poolID && withdrawal.Status == types.WithdrawalStatusPending {
			withdrawals = append(withdrawals, &withdrawal)
		}
	}
	return withdrawals
}

// ============ DDGuard State Operations ============

// SetDDGuardState saves DDGuard state to the store
func (k *Keeper) SetDDGuardState(ctx sdk.Context, state *types.DDGuardState) {
	store := k.GetStore(ctx)
	key := append(DDGuardStateKeyPrefix, []byte(state.PoolID)...)
	bz, _ := json.Marshal(state)
	store.Set(key, bz)
}

// GetDDGuardState retrieves DDGuard state from the store
func (k *Keeper) GetDDGuardState(ctx sdk.Context, poolID string) *types.DDGuardState {
	store := k.GetStore(ctx)
	key := append(DDGuardStateKeyPrefix, []byte(poolID)...)
	bz := store.Get(key)
	if bz == nil {
		return nil
	}
	var state types.DDGuardState
	if err := json.Unmarshal(bz, &state); err != nil {
		return nil
	}
	return &state
}

// ============ Pool Stats Operations ============

// SetPoolStats saves pool stats to the store
func (k *Keeper) SetPoolStats(ctx sdk.Context, stats *types.PoolStats) {
	store := k.GetStore(ctx)
	key := append(PoolStatsKeyPrefix, []byte(stats.PoolID)...)
	bz, _ := json.Marshal(stats)
	store.Set(key, bz)
}

// GetPoolStats retrieves pool stats from the store
func (k *Keeper) GetPoolStats(ctx sdk.Context, poolID string) *types.PoolStats {
	store := k.GetStore(ctx)
	key := append(PoolStatsKeyPrefix, []byte(poolID)...)
	bz := store.Get(key)
	if bz == nil {
		return types.NewPoolStats(poolID)
	}
	var stats types.PoolStats
	if err := json.Unmarshal(bz, &stats); err != nil {
		return types.NewPoolStats(poolID)
	}
	return &stats
}

// ============ NAV History Operations ============

// navHistoryKey generates the key for NAV history
func navHistoryKey(poolID string, timestamp int64) []byte {
	// Format timestamp as fixed-width string for proper ordering
	return append(NAVHistoryKeyPrefix, []byte(poolID+":"+strconv.FormatInt(timestamp, 10))...)
}

// AddNAVHistory adds a NAV history record
func (k *Keeper) AddNAVHistory(ctx sdk.Context, history *types.NAVHistory) {
	store := k.GetStore(ctx)
	key := navHistoryKey(history.PoolID, history.Timestamp)
	bz, _ := json.Marshal(history)
	store.Set(key, bz)
}

// GetNAVHistory retrieves NAV history for a pool
func (k *Keeper) GetNAVHistory(ctx sdk.Context, poolID string, fromTime, toTime int64) []*types.NAVHistory {
	store := k.GetStore(ctx)
	prefix := append(NAVHistoryKeyPrefix, []byte(poolID+":")...)
	iterator := storetypes.KVStorePrefixIterator(store, prefix)
	defer iterator.Close()

	var history []*types.NAVHistory
	for ; iterator.Valid(); iterator.Next() {
		var h types.NAVHistory
		if err := json.Unmarshal(iterator.Value(), &h); err != nil {
			continue
		}
		if (fromTime == 0 || h.Timestamp >= fromTime) && (toTime == 0 || h.Timestamp <= toTime) {
			history = append(history, &h)
		}
	}
	return history
}

// ============ User Balance Operations ============

// GetUserPoolBalance calculates user's balance in a pool
func (k *Keeper) GetUserPoolBalance(ctx sdk.Context, poolID, user string) (shares, value, costBasis math.LegacyDec) {
	deposits := k.GetUserDeposits(ctx, user)
	pool := k.GetPool(ctx, poolID)
	if pool == nil {
		return math.LegacyZeroDec(), math.LegacyZeroDec(), math.LegacyZeroDec()
	}

	shares = math.LegacyZeroDec()
	costBasis = math.LegacyZeroDec()

	for _, deposit := range deposits {
		if deposit.PoolID == poolID {
			shares = shares.Add(deposit.Shares)
			costBasis = costBasis.Add(deposit.Amount)
		}
	}

	value = shares.Mul(pool.NAV)
	return shares, value, costBasis
}
