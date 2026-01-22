// Package framework provides real chain E2E testing infrastructure
package framework

import (
	"os"
	"path/filepath"
	"time"
)

// ChainTestConfig holds configuration for chain E2E tests
type ChainTestConfig struct {
	// Chain connection
	RPCURL   string
	APIURL   string
	GRPCAddr string
	ChainID  string

	// Binary and paths
	BinaryPath     string
	HomeDir        string
	KeyringBackend string

	// Timing
	BlockTime         time.Duration
	TxConfirmTimeout  time.Duration
	ChainStartTimeout time.Duration

	// Test accounts
	ValidatorKey string
	TestAccounts []string

	// Feature flags
	AutoStartChain bool
	CleanupOnExit  bool
	Verbose        bool
}

// DefaultChainTestConfig returns default configuration for local testing
func DefaultChainTestConfig() *ChainTestConfig {
	projectRoot := findProjectRoot()

	return &ChainTestConfig{
		// Chain connection
		RPCURL:   getEnvOrDefault("PERPDEX_RPC_URL", "http://localhost:26657"),
		APIURL:   getEnvOrDefault("PERPDEX_API_URL", "http://localhost:1317"),
		GRPCAddr: getEnvOrDefault("PERPDEX_GRPC_ADDR", "localhost:9090"),
		ChainID:  getEnvOrDefault("PERPDEX_CHAIN_ID", "perpdex-test-1"),

		// Binary and paths
		BinaryPath:     filepath.Join(projectRoot, "build", "perpdexd"),
		HomeDir:        filepath.Join(projectRoot, ".perpdex-test"),
		KeyringBackend: "test",

		// Timing
		BlockTime:         500 * time.Millisecond,
		TxConfirmTimeout:  30 * time.Second,
		ChainStartTimeout: 60 * time.Second,

		// Test accounts
		ValidatorKey: "validator",
		TestAccounts: []string{"trader1", "trader2", "trader3"},

		// Feature flags
		AutoStartChain: getEnvBool("PERPDEX_AUTO_START", true),
		CleanupOnExit:  getEnvBool("PERPDEX_CLEANUP", false),
		Verbose:        getEnvBool("PERPDEX_VERBOSE", false),
	}
}

// CIChainTestConfig returns configuration optimized for CI environment
func CIChainTestConfig() *ChainTestConfig {
	config := DefaultChainTestConfig()
	config.AutoStartChain = true
	config.CleanupOnExit = true
	config.Verbose = true
	config.TxConfirmTimeout = 60 * time.Second
	config.ChainStartTimeout = 120 * time.Second
	return config
}

// findProjectRoot finds the project root by looking for go.mod
func findProjectRoot() string {
	// Try from current directory upward
	dir, _ := os.Getwd()
	for i := 0; i < 10; i++ {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	// Fallback to absolute path
	return "/Users/vvedition/Desktop/dex mvp/perp-dex_副本"
}

// getEnvOrDefault returns environment variable or default value
func getEnvOrDefault(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

// getEnvBool returns environment variable as bool
func getEnvBool(key string, defaultVal bool) bool {
	val := os.Getenv(key)
	if val == "" {
		return defaultVal
	}
	return val == "true" || val == "1" || val == "yes"
}

// TestAccount represents a test account with its configuration
type TestAccount struct {
	Name    string
	Address string
	Balance string
}

// MarketConfig represents a test market configuration
type MarketConfig struct {
	MarketID    string
	BaseDenom   string
	QuoteDenom  string
	MinQuantity string
	MaxQuantity string
	PriceTick   string
}

// DefaultMarkets returns default market configurations for testing
func DefaultMarkets() []MarketConfig {
	return []MarketConfig{
		{
			MarketID:    "BTC-USDC",
			BaseDenom:   "ubtc",
			QuoteDenom:  "usdc",
			MinQuantity: "0.001",
			MaxQuantity: "100",
			PriceTick:   "1",
		},
		{
			MarketID:    "ETH-USDC",
			BaseDenom:   "ueth",
			QuoteDenom:  "usdc",
			MinQuantity: "0.01",
			MaxQuantity: "1000",
			PriceTick:   "0.1",
		},
	}
}
