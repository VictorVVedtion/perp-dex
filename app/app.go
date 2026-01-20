package app

import (
	"encoding/base64"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"time"

	"cosmossdk.io/core/appmodule"
	"cosmossdk.io/log"
	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	abci "github.com/cometbft/cometbft/abci/types"
	cmtcrypto "github.com/cometbft/cometbft/proto/tendermint/crypto"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/grpc/cmtservice"
	nodeservice "github.com/cosmos/cosmos-sdk/client/grpc/node"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/address"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	"github.com/cosmos/cosmos-sdk/server/api"
	"github.com/cosmos/cosmos-sdk/server/config"
	servertypes "github.com/cosmos/cosmos-sdk/server/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/cosmos/cosmos-sdk/x/auth"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	authtx "github.com/cosmos/cosmos-sdk/x/auth/tx"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/cosmos/cosmos-sdk/x/bank"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/cosmos/cosmos-sdk/x/consensus"
	consensusparamkeeper "github.com/cosmos/cosmos-sdk/x/consensus/keeper"
	consensusparamtypes "github.com/cosmos/cosmos-sdk/x/consensus/types"
	"github.com/cosmos/cosmos-sdk/x/genutil"
	genutiltypes "github.com/cosmos/cosmos-sdk/x/genutil/types"
	"github.com/cosmos/cosmos-sdk/x/staking"
	gogoprotograpc "github.com/cosmos/gogoproto/grpc"

	clearinghousekeeper "github.com/openalpha/perp-dex/x/clearinghouse/keeper"
	orderbookkeeper "github.com/openalpha/perp-dex/x/orderbook/keeper"
	orderbooktypes "github.com/openalpha/perp-dex/x/orderbook/types"
	perpetualkeeper "github.com/openalpha/perp-dex/x/perpetual/keeper"
	perpetualtypes "github.com/openalpha/perp-dex/x/perpetual/types"
)

const (
	Name = "perpdex"
)

var (
	// DefaultNodeHome default home directories for the application daemon
	DefaultNodeHome string

	// ModuleBasics defines the module BasicManager used for codec registration
	ModuleBasics = module.NewBasicManager(
		auth.AppModuleBasic{},
		bank.AppModuleBasic{},
		staking.AppModuleBasic{},
		genutil.NewAppModuleBasic(genutiltypes.DefaultMessageValidator),
		consensus.AppModuleBasic{},
	)
)

func init() {
	userHomeDir, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}
	DefaultNodeHome = filepath.Join(userHomeDir, ".perpdex")
}

// App extends an ABCI application
type App struct {
	*baseapp.BaseApp

	legacyAmino       *codec.LegacyAmino
	appCodec          codec.Codec
	interfaceRegistry codectypes.InterfaceRegistry
	txConfig          client.TxConfig

	// Keys
	keys    map[string]*storetypes.KVStoreKey
	tkeys   map[string]*storetypes.TransientStoreKey
	memKeys map[string]*storetypes.MemoryStoreKey

	// SDK Keepers
	ConsensusParamsKeeper consensusparamkeeper.Keeper
	AccountKeeper         authkeeper.AccountKeeper
	BankKeeper            bankkeeper.BaseKeeper

	// Custom module keepers
	OrderbookKeeper     *orderbookkeeper.Keeper
	PerpetualKeeper     *perpetualkeeper.Keeper
	ClearinghouseKeeper *clearinghousekeeper.Keeper

	// Module Manager
	BasicModuleManager module.BasicManager
}

// NewApp returns a new App instance
func NewApp(
	logger log.Logger,
	db dbm.DB,
	traceStore io.Writer,
	loadLatest bool,
	appOpts servertypes.AppOptions,
	baseAppOptions ...func(*baseapp.BaseApp),
) *App {
	// Create codec
	encodingConfig := MakeEncodingConfig()
	appCodec := encodingConfig.Codec
	legacyAmino := encodingConfig.Amino
	interfaceRegistry := encodingConfig.InterfaceRegistry

	// Create base app
	bApp := baseapp.NewBaseApp(Name, logger, db, encodingConfig.TxConfig.TxDecoder(), baseAppOptions...)
	bApp.SetCommitMultiStoreTracer(traceStore)
	bApp.SetInterfaceRegistry(interfaceRegistry)

	// Define store keys
	keys := storetypes.NewKVStoreKeys(
		authtypes.StoreKey,
		banktypes.StoreKey,
		"orderbook",
		"perpetual",
		"clearinghouse",
		consensusparamtypes.StoreKey,
	)
	tkeys := storetypes.NewTransientStoreKeys()
	memKeys := storetypes.NewMemoryStoreKeys()

	app := &App{
		BaseApp:            bApp,
		legacyAmino:        legacyAmino,
		appCodec:           appCodec,
		interfaceRegistry:  interfaceRegistry,
		txConfig:           encodingConfig.TxConfig,
		keys:               keys,
		tkeys:              tkeys,
		memKeys:            memKeys,
		BasicModuleManager: ModuleBasics,
	}

	// Initialize consensus params keeper
	app.ConsensusParamsKeeper = consensusparamkeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(keys[consensusparamtypes.StoreKey]),
		"", // authority - empty for MVP
		runtime.EventService{},
	)
	bApp.SetParamStore(app.ConsensusParamsKeeper.ParamsStore)

	// Module account permissions
	maccPerms := map[string][]string{
		authtypes.FeeCollectorName: nil,
		"perpetual":                {authtypes.Minter, authtypes.Burner},
	}

	// Create address codec
	addrCodec := address.NewBech32Codec(sdk.GetConfig().GetBech32AccountAddrPrefix())

	// Initialize account keeper
	app.AccountKeeper = authkeeper.NewAccountKeeper(
		appCodec,
		runtime.NewKVStoreService(keys[authtypes.StoreKey]),
		authtypes.ProtoBaseAccount,
		maccPerms,
		addrCodec,
		sdk.GetConfig().GetBech32AccountAddrPrefix(),
		"", // authority - empty for MVP
	)

	// Initialize bank keeper
	// Authority is set to the governance module address for bank operations
	bankAuthority := authtypes.NewModuleAddress("gov").String()
	app.BankKeeper = bankkeeper.NewBaseKeeper(
		appCodec,
		runtime.NewKVStoreService(keys[banktypes.StoreKey]),
		app.AccountKeeper,
		BlockedModuleAccountAddrs(maccPerms),
		bankAuthority,
		logger,
	)

	// Initialize custom keepers
	app.PerpetualKeeper = perpetualkeeper.NewKeeper(
		appCodec,
		keys["perpetual"],
		app.BankKeeper, // bank keeper for fund transfers
		"",             // authority
		logger,
	)

	orderbookPerpAdapter := newOrderbookPerpetualAdapter(app.PerpetualKeeper)
	app.OrderbookKeeper = orderbookkeeper.NewKeeper(
		appCodec,
		keys["orderbook"],
		orderbookPerpAdapter,
		logger,
	)

	app.ClearinghouseKeeper = clearinghousekeeper.NewKeeper(
		appCodec,
		keys["clearinghouse"],
		app.PerpetualKeeper,
		nil, // orderbook keeper interface
		logger,
	)

	// Register message types with the interface registry
	orderbooktypes.RegisterInterfaces(interfaceRegistry)
	perpetualtypes.RegisterInterfaces(interfaceRegistry)

	// Register MsgServer for custom modules with the message service router
	orderbooktypes.RegisterMsgServer(bApp.MsgServiceRouter(), orderbookkeeper.NewMsgServerImpl(app.OrderbookKeeper))

	// Register QueryServers for SDK modules
	authtypes.RegisterQueryServer(bApp.GRPCQueryRouter(), authkeeper.NewQueryServer(app.AccountKeeper))
	banktypes.RegisterQueryServer(bApp.GRPCQueryRouter(), bankkeeper.NewQuerier(&app.BankKeeper))

	// Mount stores
	app.MountKVStores(keys)
	app.MountTransientStores(tkeys)
	app.MountMemoryStores(memKeys)

	// Initialize and finalize
	app.SetInitChainer(app.InitChainer)
	app.SetBeginBlocker(app.BeginBlocker)
	app.SetEndBlocker(app.EndBlocker)

	if loadLatest {
		if err := app.LoadLatestVersion(); err != nil {
			panic(err)
		}
	}

	return app
}

// Name returns the name of the App
func (app *App) Name() string { return app.BaseApp.Name() }

// BeginBlocker executes begin block logic
func (app *App) BeginBlocker(ctx sdk.Context) (sdk.BeginBlock, error) {
	return sdk.BeginBlock{}, nil
}

// EndBlocker executes end block logic with performance metrics
func (app *App) EndBlocker(ctx sdk.Context) (sdk.EndBlock, error) {
	logger := app.Logger()
	blockHeight := ctx.BlockHeight()
	totalStart := time.Now()

	// Track individual operation timings
	var oracleDuration, matchingDuration, liquidationDuration, fundingDuration, conditionalDuration time.Duration

	// ===========================================
	// Phase 1: Oracle Price Updates
	// ===========================================
	oracleStart := time.Now()
	oracle := perpetualkeeper.NewOracleSimulator(app.PerpetualKeeper)
	oracle.EndBlockPriceUpdate(ctx)
	oracleDuration = time.Since(oracleStart)

	// ===========================================
	// Phase 2: Order Matching (Optimized)
	// ===========================================
	matchingStart := time.Now()
	matchingResult, matchErr := app.OrderbookKeeper.ParallelEndBlockerV2(ctx)
	if matchErr != nil {
		logger.Error("parallel matching v2 failed", "error", matchErr)
	}
	matchingDuration = time.Since(matchingStart)

	settlementRequest := convertToSettlementRequest(matchingResult)
	if settlementRequest != nil && len(settlementRequest.Trades) > 0 {
		settlementEngine := clearinghousekeeper.NewSettlementEngine(app.ClearinghouseKeeper)
		if _, err := settlementEngine.Settle(ctx, settlementRequest); err != nil {
			logger.Error("settlement failed", "error", err)
		}
	}

	// ===========================================
	// Phase 3: Liquidation Processing
	// ===========================================
	liquidationStart := time.Now()
	liquidationEngine := clearinghousekeeper.NewLiquidationEngine(app.ClearinghouseKeeper)
	liquidationStats := liquidationEngine.EndBlockLiquidations(ctx)
	liquidationDuration = time.Since(liquidationStart)

	// ===========================================
	// Phase 4: Funding Settlement
	// ===========================================
	fundingStart := time.Now()
	app.PerpetualKeeper.FundingEndBlocker(ctx)
	fundingDuration = time.Since(fundingStart)

	// ===========================================
	// Phase 5: Conditional Orders
	// ===========================================
	conditionalStart := time.Now()
	app.OrderbookKeeper.ConditionalOrderEndBlocker(ctx)
	conditionalDuration = time.Since(conditionalStart)

	// ===========================================
	// Performance Logging
	// ===========================================
	totalDuration := time.Since(totalStart)

	// Log performance metrics
	logger.Info("EndBlocker performance",
		"block", blockHeight,
		"total_ms", totalDuration.Milliseconds(),
		"oracle_ms", oracleDuration.Milliseconds(),
		"matching_ms", matchingDuration.Milliseconds(),
		"liquidation_ms", liquidationDuration.Milliseconds(),
		"funding_ms", fundingDuration.Milliseconds(),
		"conditional_ms", conditionalDuration.Milliseconds(),
	)

	// Log matching statistics if any markets were processed
	if matchingResult != nil && len(matchingResult.Results) > 0 {
		totalVolume := math.LegacyZeroDec()
		marketsProcessed := 0
		for _, marketResult := range matchingResult.Results {
			if marketResult == nil {
				continue
			}
			marketsProcessed++
			for _, trade := range marketResult.Trades {
				if trade == nil {
					continue
				}
				totalVolume = totalVolume.Add(trade.Quantity.Mul(trade.Price))
			}
		}

		logger.Info("Matching engine stats",
			"block", blockHeight,
			"markets_processed", marketsProcessed,
			"trades_executed", matchingResult.TotalTrades,
			"volume", totalVolume.String(),
			"errors", len(matchingResult.Errors),
			"duration_ms", matchingResult.Duration.Milliseconds(),
		)
	}

	// Log liquidation statistics if any liquidations occurred
	if liquidationStats.LiquidationsCount > 0 {
		logger.Info("Liquidation stats",
			"block", blockHeight,
			"liquidations", liquidationStats.LiquidationsCount,
			"volume", liquidationStats.TotalVolume.String(),
		)
	}

	// Warn if EndBlocker takes too long (> 100ms)
	if totalDuration > 100*time.Millisecond {
		logger.Warn("EndBlocker exceeded latency threshold",
			"block", blockHeight,
			"duration_ms", totalDuration.Milliseconds(),
			"threshold_ms", 100,
		)
	}

	return sdk.EndBlock{}, nil
}

func convertToSettlementRequest(result *orderbookkeeper.AggregatedMatchResultV2) *orderbooktypes.SettlementRequest {
	if result == nil || result.TotalTrades == 0 {
		return nil
	}

	trades := make([]*orderbooktypes.TradeWithSettlement, 0, result.TotalTrades)
	for _, marketResult := range result.Results {
		if marketResult == nil || marketResult.Error != nil {
			continue
		}
		trades = append(trades, marketResult.Trades...)
	}
	if len(trades) == 0 {
		return nil
	}

	return orderbooktypes.NewSettlementRequest(trades)
}

// StakingGenesisState represents the staking module's genesis state
type StakingGenesisState struct {
	Validators []struct {
		ConsensusPubkey struct {
			Type string `json:"@type"`
			Key  string `json:"key"`
		} `json:"consensus_pubkey"`
		Tokens string `json:"tokens"`
		Status string `json:"status"`
	} `json:"validators"`
}

// GenutilGenesisState represents the genutil module's genesis state
type GenutilGenesisState struct {
	GenTxs []json.RawMessage `json:"gen_txs"`
}

// GenTx represents a genesis transaction
type GenTx struct {
	Body struct {
		Messages []json.RawMessage `json:"messages"`
	} `json:"body"`
}

// MsgCreateValidator represents the create validator message
type MsgCreateValidator struct {
	Type   string `json:"@type"`
	Pubkey struct {
		Type string `json:"@type"`
		Key  string `json:"key"`
	} `json:"pubkey"`
	Value struct {
		Denom  string `json:"denom"`
		Amount string `json:"amount"`
	} `json:"value"`
}

// InitChainer initializes the chain
func (app *App) InitChainer(ctx sdk.Context, req *abci.RequestInitChain) (*abci.ResponseInitChain, error) {
	var genesisState map[string]json.RawMessage
	if err := json.Unmarshal(req.AppStateBytes, &genesisState); err != nil {
		return nil, err
	}

	// Initialize default market
	app.PerpetualKeeper.InitDefaultMarket(ctx)

	// If validators are provided in request, use them
	if len(req.Validators) > 0 {
		return &abci.ResponseInitChain{
			Validators: req.Validators,
		}, nil
	}

	// Try to get validators from staking genesis state first
	var validators []abci.ValidatorUpdate
	if stakingGenesis, ok := genesisState["staking"]; ok {
		var stakingState StakingGenesisState
		if err := json.Unmarshal(stakingGenesis, &stakingState); err == nil {
			for _, val := range stakingState.Validators {
				if val.Status == "BOND_STATUS_BONDED" {
					pubKeyBytes, err := base64.StdEncoding.DecodeString(val.ConsensusPubkey.Key)
					if err != nil {
						continue
					}
					validators = append(validators, abci.ValidatorUpdate{
						PubKey: cmtcrypto.PublicKey{
							Sum: &cmtcrypto.PublicKey_Ed25519{
								Ed25519: pubKeyBytes,
							},
						},
						Power: 100,
					})
				}
			}
		}
	}

	// If no validators from staking, try to extract from gentx
	if len(validators) == 0 {
		if genutilGenesis, ok := genesisState["genutil"]; ok {
			var genutilState GenutilGenesisState
			if err := json.Unmarshal(genutilGenesis, &genutilState); err == nil {
				for _, genTxRaw := range genutilState.GenTxs {
					var genTx GenTx
					if err := json.Unmarshal(genTxRaw, &genTx); err != nil {
						continue
					}
					for _, msgRaw := range genTx.Body.Messages {
						var msg MsgCreateValidator
						if err := json.Unmarshal(msgRaw, &msg); err != nil {
							continue
						}
						if msg.Type == "/cosmos.staking.v1beta1.MsgCreateValidator" {
							pubKeyBytes, err := base64.StdEncoding.DecodeString(msg.Pubkey.Key)
							if err != nil {
								continue
							}
							validators = append(validators, abci.ValidatorUpdate{
								PubKey: cmtcrypto.PublicKey{
									Sum: &cmtcrypto.PublicKey_Ed25519{
										Ed25519: pubKeyBytes,
									},
								},
								Power: 100,
							})
						}
					}
				}
			}
		}
	}

	return &abci.ResponseInitChain{
		Validators: validators,
	}, nil
}

// LoadHeight loads a particular height
func (app *App) LoadHeight(height int64) error {
	return app.LoadVersion(height)
}

// LegacyAmino returns the legacy amino codec
func (app *App) LegacyAmino() *codec.LegacyAmino {
	return app.legacyAmino
}

// AppCodec returns the app codec
func (app *App) AppCodec() codec.Codec {
	return app.appCodec
}

// InterfaceRegistry returns the InterfaceRegistry
func (app *App) InterfaceRegistry() codectypes.InterfaceRegistry {
	return app.interfaceRegistry
}

// RegisterAPIRoutes registers all application module routes
func (app *App) RegisterAPIRoutes(apiSvr *api.Server, apiConfig config.APIConfig) {
	clientCtx := apiSvr.ClientCtx
	// Register new routes
	ModuleBasics.RegisterGRPCGatewayRoutes(clientCtx, apiSvr.GRPCGatewayRouter)
}

// GetKey returns a store key
func (app *App) GetKey(storeKey string) *storetypes.KVStoreKey {
	return app.keys[storeKey]
}

// GetTKey returns a transient store key
func (app *App) GetTKey(storeKey string) *storetypes.TransientStoreKey {
	return app.tkeys[storeKey]
}

// GetMemKey returns a memory store key
func (app *App) GetMemKey(storeKey string) *storetypes.MemoryStoreKey {
	return app.memKeys[storeKey]
}

// TxConfig returns the transaction config
func (app *App) TxConfig() client.TxConfig {
	return app.txConfig
}

// AutoCliOpts returns the autocli options for the app
func (app *App) AutoCliOpts() map[string]appmodule.AppModule {
	return map[string]appmodule.AppModule{}
}

// RegisterTxService implements the Application.RegisterTxService method
func (app *App) RegisterTxService(clientCtx client.Context) {
	authtx.RegisterTxService(app.BaseApp.GRPCQueryRouter(), clientCtx, app.BaseApp.Simulate, app.interfaceRegistry)
}

// RegisterTendermintService implements the Application.RegisterTendermintService method
func (app *App) RegisterTendermintService(clientCtx client.Context) {
	cmtservice.RegisterTendermintService(
		clientCtx,
		app.BaseApp.GRPCQueryRouter(),
		app.interfaceRegistry,
		app.Query,
	)
}

// RegisterNodeService implements the Application.RegisterNodeService method
func (app *App) RegisterNodeService(clientCtx client.Context, cfg config.Config) {
	nodeservice.RegisterNodeService(clientCtx, app.BaseApp.GRPCQueryRouter(), cfg)
}

// RegisterGRPCServer registers the app's gRPC services
func (app *App) RegisterGRPCServer(server gogoprotograpc.Server) {
	// Custom gRPC services are now registered via MsgServiceRouter in NewApp
}

// SimulationManager returns the app's simulation manager
func (app *App) SimulationManager() *module.SimulationManager {
	return nil
}

// BlockedModuleAccountAddrs returns module account addresses that should not
// receive coins (these accounts are typically module accounts like fee collector)
func BlockedModuleAccountAddrs(maccPerms map[string][]string) map[string]bool {
	blockedAddrs := make(map[string]bool)
	for acc := range maccPerms {
		blockedAddrs[authtypes.NewModuleAddress(acc).String()] = true
	}
	// Remove perpetual from blocked addrs as it needs to receive/send funds
	delete(blockedAddrs, authtypes.NewModuleAddress("perpetual").String())
	return blockedAddrs
}
