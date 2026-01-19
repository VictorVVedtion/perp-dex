package keeper

import (
	"context"
	"encoding/json"

	"cosmossdk.io/log"
	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/openalpha/perp-dex/x/perpetual/types"
)

// Store key prefixes
var (
	MarketKeyPrefix   = []byte{0x01}
	PositionKeyPrefix = []byte{0x02}
	AccountKeyPrefix  = []byte{0x03}
	PriceKeyPrefix    = []byte{0x04}
)

// BankKeeper defines the expected interface for the bank module
type BankKeeper interface {
	SendCoinsFromAccountToModule(ctx context.Context, senderAddr sdk.AccAddress, recipientModule string, amt sdk.Coins) error
	SendCoinsFromModuleToAccount(ctx context.Context, senderModule string, recipientAddr sdk.AccAddress, amt sdk.Coins) error
}

// Keeper manages the perpetual module state
type Keeper struct {
	cdc        codec.BinaryCodec
	storeKey   storetypes.StoreKey
	bankKeeper BankKeeper
	logger     log.Logger
	authority  string // governance authority address
}

// NewKeeper creates a new perpetual keeper
func NewKeeper(
	cdc codec.BinaryCodec,
	storeKey storetypes.StoreKey,
	bankKeeper BankKeeper,
	authority string,
	logger log.Logger,
) *Keeper {
	return &Keeper{
		cdc:        cdc,
		storeKey:   storeKey,
		bankKeeper: bankKeeper,
		authority:  authority,
		logger:     logger.With("module", "x/perpetual"),
	}
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

// ============ Market Operations ============

// SetMarket saves a market to the store
func (k *Keeper) SetMarket(ctx sdk.Context, market *types.Market) {
	store := k.GetStore(ctx)
	key := append(MarketKeyPrefix, []byte(market.MarketID)...)
	bz, _ := json.Marshal(market)
	store.Set(key, bz)
}

// GetMarket retrieves a market from the store
func (k *Keeper) GetMarket(ctx sdk.Context, marketID string) *types.Market {
	store := k.GetStore(ctx)
	key := append(MarketKeyPrefix, []byte(marketID)...)
	bz := store.Get(key)
	if bz == nil {
		return nil
	}
	var market types.Market
	if err := json.Unmarshal(bz, &market); err != nil {
		return nil
	}
	return &market
}

// GetAllMarkets returns all markets
func (k *Keeper) GetAllMarkets(ctx sdk.Context) []*types.Market {
	store := k.GetStore(ctx)
	iterator := storetypes.KVStorePrefixIterator(store, MarketKeyPrefix)
	defer iterator.Close()

	var markets []*types.Market
	for ; iterator.Valid(); iterator.Next() {
		var market types.Market
		if err := json.Unmarshal(iterator.Value(), &market); err != nil {
			continue
		}
		markets = append(markets, &market)
	}
	return markets
}

// InitDefaultMarket initializes the BTC-USDC market for MVP
func (k *Keeper) InitDefaultMarket(ctx sdk.Context) {
	market := types.NewMarket("BTC-USDC", "BTC", "USDC")
	k.SetMarket(ctx, market)

	// Set initial price
	price := types.NewPriceInfo("BTC-USDC", math.LegacyNewDec(50000))
	k.SetPrice(ctx, price)

	k.SetNextFundingTime(ctx, market.MarketID, nextFundingTimeUTC(ctx.BlockTime()))
}

// ============ Position Operations ============

// positionKey generates the key for a position
func positionKey(trader, marketID string) []byte {
	return append(PositionKeyPrefix, []byte(trader+":"+marketID)...)
}

// SetPosition saves a position to the store
func (k *Keeper) SetPosition(ctx sdk.Context, position *types.Position) {
	store := k.GetStore(ctx)
	key := positionKey(position.Trader, position.MarketID)
	bz, _ := json.Marshal(position)
	store.Set(key, bz)
}

// GetPosition retrieves a position from the store
func (k *Keeper) GetPosition(ctx sdk.Context, trader, marketID string) *types.Position {
	store := k.GetStore(ctx)
	key := positionKey(trader, marketID)
	bz := store.Get(key)
	if bz == nil {
		return nil
	}
	var position types.Position
	if err := json.Unmarshal(bz, &position); err != nil {
		return nil
	}
	return &position
}

// DeletePosition removes a position from the store
func (k *Keeper) DeletePosition(ctx sdk.Context, trader, marketID string) {
	store := k.GetStore(ctx)
	key := positionKey(trader, marketID)
	store.Delete(key)
}

// GetPositionsByTrader returns all positions for a trader
func (k *Keeper) GetPositionsByTrader(ctx sdk.Context, trader string) []*types.Position {
	store := k.GetStore(ctx)
	iterator := storetypes.KVStorePrefixIterator(store, PositionKeyPrefix)
	defer iterator.Close()

	var positions []*types.Position
	for ; iterator.Valid(); iterator.Next() {
		var position types.Position
		if err := json.Unmarshal(iterator.Value(), &position); err != nil {
			continue
		}
		if position.Trader == trader {
			positions = append(positions, &position)
		}
	}
	return positions
}

// GetAllPositions returns all positions
func (k *Keeper) GetAllPositions(ctx sdk.Context) []*types.Position {
	store := k.GetStore(ctx)
	iterator := storetypes.KVStorePrefixIterator(store, PositionKeyPrefix)
	defer iterator.Close()

	var positions []*types.Position
	for ; iterator.Valid(); iterator.Next() {
		var position types.Position
		if err := json.Unmarshal(iterator.Value(), &position); err != nil {
			continue
		}
		positions = append(positions, &position)
	}
	return positions
}

// ============ Account Operations ============

// SetAccount saves an account to the store
func (k *Keeper) SetAccount(ctx sdk.Context, account *types.Account) {
	store := k.GetStore(ctx)
	key := append(AccountKeyPrefix, []byte(account.Trader)...)
	bz, _ := json.Marshal(account)
	store.Set(key, bz)
}

// GetAccount retrieves an account from the store
func (k *Keeper) GetAccount(ctx sdk.Context, trader string) *types.Account {
	store := k.GetStore(ctx)
	key := append(AccountKeyPrefix, []byte(trader)...)
	bz := store.Get(key)
	if bz == nil {
		return nil
	}
	var account types.Account
	if err := json.Unmarshal(bz, &account); err != nil {
		return nil
	}
	return &account
}

// GetOrCreateAccount gets an existing account or creates a new one
func (k *Keeper) GetOrCreateAccount(ctx sdk.Context, trader string) *types.Account {
	account := k.GetAccount(ctx, trader)
	if account == nil {
		account = types.NewAccount(trader)
		k.SetAccount(ctx, account)
	}
	return account
}

// ============ Price Operations ============

// SetPrice saves price info to the store
func (k *Keeper) SetPrice(ctx sdk.Context, price *types.PriceInfo) {
	store := k.GetStore(ctx)
	key := append(PriceKeyPrefix, []byte(price.MarketID)...)
	bz, _ := json.Marshal(price)
	store.Set(key, bz)
}

// GetPrice retrieves price info from the store
func (k *Keeper) GetPrice(ctx sdk.Context, marketID string) *types.PriceInfo {
	store := k.GetStore(ctx)
	key := append(PriceKeyPrefix, []byte(marketID)...)
	bz := store.Get(key)
	if bz == nil {
		return nil
	}
	var price types.PriceInfo
	if err := json.Unmarshal(bz, &price); err != nil {
		return nil
	}
	return &price
}

// ============ Account Management ============

// Deposit handles margin deposit
func (k *Keeper) Deposit(ctx context.Context, trader string, amount math.LegacyDec) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// Get or create account
	account := k.GetOrCreateAccount(sdkCtx, trader)

	// Deposit funds
	account.Deposit(amount)
	k.SetAccount(sdkCtx, account)

	// Emit event
	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent(
			"deposit",
			sdk.NewAttribute("trader", trader),
			sdk.NewAttribute("amount", amount.String()),
			sdk.NewAttribute("new_balance", account.Balance.String()),
		),
	)

	return nil
}

// Withdraw handles margin withdrawal
func (k *Keeper) Withdraw(ctx context.Context, trader string, amount math.LegacyDec) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// Get account
	account := k.GetAccount(sdkCtx, trader)
	if account == nil {
		return types.ErrAccountNotFound
	}

	// Withdraw funds
	if err := account.Withdraw(amount); err != nil {
		return err
	}
	k.SetAccount(sdkCtx, account)

	// Emit event
	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent(
			"withdraw",
			sdk.NewAttribute("trader", trader),
			sdk.NewAttribute("amount", amount.String()),
			sdk.NewAttribute("new_balance", account.Balance.String()),
		),
	)

	return nil
}
