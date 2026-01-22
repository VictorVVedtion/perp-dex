package riverpool

import (
	"encoding/json"

	"cosmossdk.io/core/appmodule"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"

	"github.com/openalpha/perp-dex/x/riverpool/keeper"
	"github.com/openalpha/perp-dex/x/riverpool/types"
)

const (
	ModuleName = types.ModuleName
)

var (
	_ module.AppModuleBasic = AppModuleBasic{}
	_ appmodule.AppModule   = AppModule{}
)

// AppModuleBasic defines the basic application module for riverpool
type AppModuleBasic struct{}

// Name returns the module's name
func (AppModuleBasic) Name() string {
	return ModuleName
}

// RegisterLegacyAminoCodec registers the module's types on the given LegacyAmino codec
func (AppModuleBasic) RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {
	cdc.RegisterConcrete(&types.MsgDeposit{}, "riverpool/MsgDeposit", nil)
	cdc.RegisterConcrete(&types.MsgRequestWithdrawal{}, "riverpool/MsgRequestWithdrawal", nil)
	cdc.RegisterConcrete(&types.MsgClaimWithdrawal{}, "riverpool/MsgClaimWithdrawal", nil)
	cdc.RegisterConcrete(&types.MsgCancelWithdrawal{}, "riverpool/MsgCancelWithdrawal", nil)
	cdc.RegisterConcrete(&types.MsgCreateCommunityPool{}, "riverpool/MsgCreateCommunityPool", nil)
	cdc.RegisterConcrete(&types.MsgUpdateDDGuard{}, "riverpool/MsgUpdateDDGuard", nil)
}

// RegisterInterfaces registers the module's interface types
func (AppModuleBasic) RegisterInterfaces(registry cdctypes.InterfaceRegistry) {
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&types.MsgDeposit{},
		&types.MsgRequestWithdrawal{},
		&types.MsgClaimWithdrawal{},
		&types.MsgCancelWithdrawal{},
		&types.MsgCreateCommunityPool{},
		&types.MsgUpdateDDGuard{},
	)
}

// DefaultGenesis returns default genesis state as raw bytes
func (AppModuleBasic) DefaultGenesis(cdc codec.JSONCodec) json.RawMessage {
	return nil
}

// ValidateGenesis performs genesis state validation
func (AppModuleBasic) ValidateGenesis(cdc codec.JSONCodec, config client.TxEncodingConfig, bz json.RawMessage) error {
	return nil
}

// RegisterGRPCGatewayRoutes registers the gRPC Gateway routes for the module
func (AppModuleBasic) RegisterGRPCGatewayRoutes(clientCtx client.Context, mux *runtime.ServeMux) {
	// TODO: Register gRPC gateway routes when proto generation is set up
}

// AppModule implements an application module for the riverpool module
type AppModule struct {
	AppModuleBasic
	keeper *keeper.Keeper
}

// NewAppModule creates a new AppModule object
func NewAppModule(k *keeper.Keeper) AppModule {
	return AppModule{
		AppModuleBasic: AppModuleBasic{},
		keeper:         k,
	}
}

// Name returns the module's name
func (am AppModule) Name() string {
	return ModuleName
}

// RegisterServices registers module services
func (am AppModule) RegisterServices(cfg module.Configurator) {
	// Register MsgServer
	// Note: In a full implementation, you would register the proto-generated server
	// For now, we'll use the custom MsgServer
	_ = keeper.NewMsgServerImpl(am.keeper)
}

// IsOnePerModuleType implements the depinject.OnePerModuleType interface
func (am AppModule) IsOnePerModuleType() {}

// IsAppModule implements the appmodule.AppModule interface
func (am AppModule) IsAppModule() {}

// EndBlocker is called at the end of each block
// It handles:
// 1. NAV updates for all active pools
// 2. DDGuard level checks
// 3. Pending withdrawal processing
func (am AppModule) EndBlocker(ctx sdk.Context) error {
	return am.keeper.EndBlocker(ctx)
}
