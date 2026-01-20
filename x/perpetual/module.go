package perpetual

import (
	"encoding/json"

	"cosmossdk.io/core/appmodule"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"

	"github.com/openalpha/perp-dex/x/perpetual/keeper"
	"github.com/openalpha/perp-dex/x/perpetual/types"
)

const (
	ModuleName = "perpetual"
)

var (
	_ module.AppModuleBasic = AppModuleBasic{}
	_ appmodule.AppModule   = AppModule{}
)

// AppModuleBasic defines the basic application module for perpetual
type AppModuleBasic struct{}

// Name returns the module's name
func (AppModuleBasic) Name() string {
	return ModuleName
}

// RegisterLegacyAminoCodec registers the module's types on the given LegacyAmino codec
func (AppModuleBasic) RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {
	cdc.RegisterConcrete(&types.MsgDeposit{}, "perpetual/MsgDeposit", nil)
	cdc.RegisterConcrete(&types.MsgWithdraw{}, "perpetual/MsgWithdraw", nil)
}

// RegisterInterfaces registers the module's interface types
func (AppModuleBasic) RegisterInterfaces(registry cdctypes.InterfaceRegistry) {
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&types.MsgDeposit{},
		&types.MsgWithdraw{},
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
	// No-op for now
}

// AppModule implements an application module for the perpetual module
type AppModule struct {
	AppModuleBasic
	keeper *keeper.Keeper
}

// NewAppModule creates a new AppModule object
func NewAppModule(keeper *keeper.Keeper) AppModule {
	return AppModule{
		AppModuleBasic: AppModuleBasic{},
		keeper:         keeper,
	}
}

// Name returns the module's name
func (am AppModule) Name() string {
	return ModuleName
}

// RegisterServices registers module services
func (am AppModule) RegisterServices(cfg module.Configurator) {
	types.RegisterMsgServer(cfg.MsgServer(), keeper.NewMsgServerImpl(am.keeper))
}

// IsOnePerModuleType implements the depinject.OnePerModuleType interface
func (am AppModule) IsOnePerModuleType() {}

// IsAppModule implements the appmodule.AppModule interface
func (am AppModule) IsAppModule() {}
