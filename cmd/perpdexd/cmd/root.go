package cmd

import (
	"errors"
	"io"
	"os"
	"time"

	"cosmossdk.io/log"
	confixcmd "cosmossdk.io/tools/confix/cmd"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/config"
	"github.com/cosmos/cosmos-sdk/client/debug"
	"github.com/cosmos/cosmos-sdk/client/keys"
	"github.com/cosmos/cosmos-sdk/client/pruning"
	"github.com/cosmos/cosmos-sdk/client/snapshot"
	"github.com/cosmos/cosmos-sdk/server"
	serverconfig "github.com/cosmos/cosmos-sdk/server/config"
	servertypes "github.com/cosmos/cosmos-sdk/server/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/cosmos/cosmos-sdk/x/crisis"
	genutilcli "github.com/cosmos/cosmos-sdk/x/genutil/client/cli"
	authcli "github.com/cosmos/cosmos-sdk/x/auth/client/cli"
	"github.com/spf13/cobra"

	tmcfg "github.com/cometbft/cometbft/config"

	"github.com/openalpha/perp-dex/app"
	clearinghousecli "github.com/openalpha/perp-dex/x/clearinghouse/client/cli"
	orderbookcli "github.com/openalpha/perp-dex/x/orderbook/client/cli"
	perpetualcli "github.com/openalpha/perp-dex/x/perpetual/client/cli"
)

// NewRootCmd creates a new root command for perpdexd
func NewRootCmd() *cobra.Command {
	// Set config
	initSDKConfig()

	tempApp := app.NewApp(
		log.NewNopLogger(),
		dbm.NewMemDB(),
		nil,
		false,
		nil,
	)
	encodingConfig := app.MakeEncodingConfig()

	initClientCtx := client.Context{}.
		WithCodec(encodingConfig.Codec).
		WithInterfaceRegistry(encodingConfig.InterfaceRegistry).
		WithTxConfig(encodingConfig.TxConfig).
		WithLegacyAmino(encodingConfig.Amino).
		WithInput(os.Stdin).
		WithAccountRetriever(types.AccountRetriever{}).
		WithHomeDir(app.DefaultNodeHome).
		WithViper("PERPDEX")

	rootCmd := &cobra.Command{
		Use:   "perpdexd",
		Short: "PerpDEX - Perpetual Decentralized Exchange",
		Long: `PerpDEX is a perpetual contract exchange built on Cosmos SDK.
Inspired by Hyperliquid architecture for high-performance trading.`,
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			// Set the default command outputs
			cmd.SetOut(cmd.OutOrStdout())
			cmd.SetErr(cmd.ErrOrStderr())

			initClientCtx = initClientCtx.WithCmdContext(cmd.Context())
			initClientCtx, err := client.ReadPersistentCommandFlags(initClientCtx, cmd.Flags())
			if err != nil {
				return err
			}

			initClientCtx, err = config.ReadFromClientConfig(initClientCtx)
			if err != nil {
				return err
			}

			if err := client.SetCmdClientContextHandler(initClientCtx, cmd); err != nil {
				return err
			}

			customAppTemplate, customAppConfig := initAppConfig()
			customCMTConfig := initCometBFTConfig()

			return server.InterceptConfigsPreRunHandler(cmd, customAppTemplate, customAppConfig, customCMTConfig)
		},
	}

	initRootCmd(rootCmd, encodingConfig, tempApp.BasicModuleManager)

	return rootCmd
}

func initRootCmd(rootCmd *cobra.Command, encodingConfig app.EncodingConfig, basicManager module.BasicManager) {
	rootCmd.AddCommand(
		genutilcli.InitCmd(basicManager, app.DefaultNodeHome),
		debug.Cmd(),
		confixcmd.ConfigCommand(),
		pruning.Cmd(newApp, app.DefaultNodeHome),
		snapshot.Cmd(newApp),
	)

	server.AddCommands(rootCmd, app.DefaultNodeHome, newApp, appExport, addModuleInitFlags)

	// Add genesis commands
	genesisCmd := genutilcli.Commands(encodingConfig.TxConfig, basicManager, app.DefaultNodeHome)
	rootCmd.AddCommand(genesisCmd)

	// Add query commands
	queryCmd := &cobra.Command{
		Use:                        "query",
		Aliases:                    []string{"q"},
		Short:                      "Querying subcommands",
		DisableFlagParsing:         false,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}
	queryCmd.AddCommand(
		authcli.QueryTxsByEventsCmd(),
		authcli.QueryTxCmd(),
		perpetualcli.GetQueryCmd(),
		orderbookcli.GetQueryCmd(),
		clearinghousecli.GetQueryCmd(),
	)
	rootCmd.AddCommand(queryCmd)

	// Add transaction commands
	txCmd := &cobra.Command{
		Use:                        "tx",
		Short:                      "Transactions subcommands",
		DisableFlagParsing:         false,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}
	txCmd.AddCommand(
		authcli.GetSignCommand(),
		authcli.GetBroadcastCommand(),
		perpetualcli.GetTxCmd(),
		orderbookcli.GetTxCmd(),
	)
	rootCmd.AddCommand(txCmd)

	// Add keybase commands
	rootCmd.AddCommand(
		keys.Commands(),
		VersionCmd(),
	)
}

func addModuleInitFlags(startCmd *cobra.Command) {
	crisis.AddModuleInitFlags(startCmd)
}

// newApp creates a new Cosmos SDK app
func newApp(
	logger log.Logger,
	db dbm.DB,
	traceStore io.Writer,
	appOpts servertypes.AppOptions,
) servertypes.Application {
	baseappOptions := server.DefaultBaseappOptions(appOpts)

	return app.NewApp(
		logger,
		db,
		traceStore,
		true,
		appOpts,
		baseappOptions...,
	)
}

// appExport creates a new app (optionally at a given height) and exports state
func appExport(
	logger log.Logger,
	db dbm.DB,
	traceStore io.Writer,
	height int64,
	forZeroHeight bool,
	jailAllowedAddrs []string,
	appOpts servertypes.AppOptions,
	modulesToExport []string,
) (servertypes.ExportedApp, error) {
	// Create app without loading latest version
	perpdexApp := app.NewApp(
		logger,
		db,
		traceStore,
		false,
		appOpts,
	)

	if height != -1 {
		if err := perpdexApp.LoadHeight(height); err != nil {
			return servertypes.ExportedApp{}, err
		}
	}

	return servertypes.ExportedApp{}, errors.New("export not implemented")
}

// initSDKConfig initializes the SDK config
func initSDKConfig() {
	// Set prefixes (optional, using defaults)
}

// initAppConfig returns custom app config template and config
func initAppConfig() (string, interface{}) {
	type CustomAppConfig struct {
		serverconfig.Config
	}

	customAppConfig := CustomAppConfig{
		Config: *serverconfig.DefaultConfig(),
	}

	customAppTemplate := serverconfig.DefaultConfigTemplate

	return customAppTemplate, customAppConfig
}

// initCometBFTConfig returns custom CometBFT config optimized for high-performance trading
func initCometBFTConfig() *tmcfg.Config {
	cfg := tmcfg.DefaultConfig()

	// ===========================================
	// Consensus Configuration - Optimized for Fast Block Times
	// ===========================================
	// Reduce timeout for proposing a block
	cfg.Consensus.TimeoutPropose = 500 * time.Millisecond
	cfg.Consensus.TimeoutProposeDelta = 100 * time.Millisecond

	// Reduce timeout for prevote step
	cfg.Consensus.TimeoutPrevote = 500 * time.Millisecond
	cfg.Consensus.TimeoutPrevoteDelta = 100 * time.Millisecond

	// Reduce timeout for precommit step
	cfg.Consensus.TimeoutPrecommit = 500 * time.Millisecond
	cfg.Consensus.TimeoutPrecommitDelta = 100 * time.Millisecond

	// Reduce commit timeout - this is the key parameter for block time
	cfg.Consensus.TimeoutCommit = 500 * time.Millisecond

	// Skip timeout commit when block is empty (faster empty blocks)
	cfg.Consensus.SkipTimeoutCommit = false

	// ===========================================
	// Mempool Configuration - Optimized for High Throughput
	// ===========================================
	// Increase mempool size for handling more pending transactions
	cfg.Mempool.Size = 10000

	// Increase max transaction bytes (10 MB)
	cfg.Mempool.MaxTxBytes = 10485760

	// Increase max transactions per block
	cfg.Mempool.MaxTxsBytes = 104857600 // 100 MB

	// Enable recheck for faster tx processing
	cfg.Mempool.Recheck = true

	// Keep invalid transactions in cache for faster rejection
	cfg.Mempool.KeepInvalidTxsInCache = false

	// ===========================================
	// P2P Configuration - Optimized for Low Latency
	// ===========================================
	// Reduce flush throttle timeout for faster message delivery
	cfg.P2P.FlushThrottleTimeout = 10 * time.Millisecond

	// Increase send/receive rates for better throughput
	cfg.P2P.SendRate = 20480000    // 20 MB/s
	cfg.P2P.RecvRate = 20480000    // 20 MB/s

	// Increase max packet payload size
	cfg.P2P.MaxPacketMsgPayloadSize = 10240 // 10 KB

	return cfg
}

// VersionCmd returns a command to print the version
func VersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the application version",
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Println("PerpDEX v0.1.0")
		},
	}
}
