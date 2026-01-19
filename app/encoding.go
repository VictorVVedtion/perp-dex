package app

import (
	"cosmossdk.io/x/tx/signing"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/address"
	"github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/gogoproto/proto"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/std"
	"github.com/cosmos/cosmos-sdk/x/auth/tx"
)

// EncodingConfig specifies the concrete encoding types to use
type EncodingConfig struct {
	InterfaceRegistry types.InterfaceRegistry
	Codec             codec.Codec
	TxConfig          client.TxConfig
	Amino             *codec.LegacyAmino
}

// MakeEncodingConfig creates an EncodingConfig for the app
func MakeEncodingConfig() EncodingConfig {
	amino := codec.NewLegacyAmino()

	// Create address codecs with the proper Bech32 prefixes
	sdkConfig := sdk.GetConfig()
	accountAddrPrefix := sdkConfig.GetBech32AccountAddrPrefix()
	validatorAddrPrefix := sdkConfig.GetBech32ValidatorAddrPrefix()

	// Create address codecs
	addrCodec := address.NewBech32Codec(accountAddrPrefix)
	valAddrCodec := address.NewBech32Codec(validatorAddrPrefix)

	// Create signing options with address codecs
	signingOptions := signing.Options{
		AddressCodec:          addrCodec,
		ValidatorAddressCodec: valAddrCodec,
	}

	// Create interface registry with signing options and proto resolver
	interfaceRegistry, err := types.NewInterfaceRegistryWithOptions(types.InterfaceRegistryOptions{
		ProtoFiles:     proto.HybridResolver,
		SigningOptions: signingOptions,
	})
	if err != nil {
		panic(err)
	}

	// Create codec
	cdc := codec.NewProtoCodec(interfaceRegistry)

	// Create TxConfig with signing options
	txCfg, err := tx.NewTxConfigWithOptions(cdc, tx.ConfigOptions{
		EnabledSignModes: tx.DefaultSignModes,
		SigningOptions:   &signingOptions,
	})
	if err != nil {
		panic(err)
	}

	// Register standard types
	std.RegisterLegacyAminoCodec(amino)
	std.RegisterInterfaces(interfaceRegistry)

	// Register module interfaces
	ModuleBasics.RegisterLegacyAminoCodec(amino)
	ModuleBasics.RegisterInterfaces(interfaceRegistry)

	return EncodingConfig{
		InterfaceRegistry: interfaceRegistry,
		Codec:             cdc,
		TxConfig:          txCfg,
		Amino:             amino,
	}
}
