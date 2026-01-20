// Package e2e_chain provides chain client for E2E testing
package e2e_chain

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// ChainConfig holds the configuration for connecting to the chain
type ChainConfig struct {
	RPCURL      string
	APIURL      string
	GRPCAddr    string
	ChainID     string
	HomeDir     string
	BinaryPath  string
	KeyringBackend string
}

// DefaultChainConfig returns the default configuration for local testing
func DefaultChainConfig() *ChainConfig {
	// Find project root by looking for go.mod
	binaryPath := findBinaryPath()
	homeDir := findHomeDir()

	return &ChainConfig{
		RPCURL:         "http://localhost:26657",
		APIURL:         "http://localhost:1317",
		GRPCAddr:       "localhost:9090",
		ChainID:        "perpdex-1", // Match the running chain
		HomeDir:        homeDir,
		BinaryPath:     binaryPath,
		KeyringBackend: "test",
	}
}

// findHomeDir finds the chain home directory (returns absolute path)
func findHomeDir() string {
	// Try absolute path first (most reliable)
	absPath := "/Users/vvedition/Desktop/dex mvp/perp-dex_副本/.perpdex-test"
	if _, err := os.Stat(absPath + "/keyring-test"); err == nil {
		return absPath
	}

	// Fall back to relative paths
	relativePaths := []string{
		".perpdex-test",
		"../.perpdex-test",
		"../../.perpdex-test",
	}

	for _, p := range relativePaths {
		keyringPath := p + "/keyring-test"
		if _, err := os.Stat(keyringPath); err == nil {
			// Convert to absolute path for reliability
			if abs, err := filepath.Abs(p); err == nil {
				return abs
			}
			return p
		}
	}

	return ".perpdex-test"
}

// findBinaryPath searches for the perpdexd binary
func findBinaryPath() string {
	// Try common locations
	paths := []string{
		"./build/perpdexd",
		"../build/perpdexd",
		"../../build/perpdexd",
		"/Users/vvedition/Desktop/dex mvp/perp-dex_副本/build/perpdexd",
	}

	for _, p := range paths {
		if _, err := exec.LookPath(p); err == nil {
			return p
		}
		// Also check if file exists
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}

	// Default fallback
	return "./build/perpdexd"
}

// ChainClient provides methods for interacting with the chain
type ChainClient struct {
	config     *ChainConfig
	httpClient *http.Client
}

// NewChainClient creates a new chain client
func NewChainClient(config *ChainConfig) *ChainClient {
	return &ChainClient{
		config: config,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// StatusInfo contains chain status information
type StatusInfo struct {
	NodeInfo struct {
		Network string `json:"network"`
		Moniker string `json:"moniker"`
	} `json:"node_info"`
	SyncInfo struct {
		LatestBlockHeight string `json:"latest_block_height"`
		LatestBlockTime   string `json:"latest_block_time"`
		CatchingUp        bool   `json:"catching_up"`
	} `json:"sync_info"`
}

// GetStatus returns the current chain status
func (c *ChainClient) GetStatus(ctx context.Context) (*StatusInfo, error) {
	resp, err := c.httpClient.Get(c.config.RPCURL + "/status")
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RPC: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var result struct {
		Result StatusInfo `json:"result"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &result.Result, nil
}

// OrderResult contains the result of an order operation
type OrderResult struct {
	Success   bool
	TxHash    string
	OrderID   string
	GasUsed   int64
	Latency   time.Duration
	Error     string
}

// sequenceTracker tracks account sequences for offline signing
var sequenceTracker = make(map[string]uint64)

// PlaceOrder places an order on the chain via CLI using offline signing
func (c *ChainClient) PlaceOrder(ctx context.Context, trader, market, side, orderType, price, quantity string) (*OrderResult, error) {
	start := time.Now()
	result := &OrderResult{}

	// Get or initialize sequence for this trader
	seq := sequenceTracker[trader]

	// Step 1: Generate unsigned transaction
	genArgs := []string{
		"tx", "orderbook", "place-order",
		market, side, orderType, price, quantity,
		"--from", trader,
		"--home", c.config.HomeDir,
		"--chain-id", c.config.ChainID,
		"--keyring-backend", c.config.KeyringBackend,
		"--gas", "200000",
		"--fees", "1000usdc",
		"--generate-only",
	}

	genCmd := exec.CommandContext(ctx, c.config.BinaryPath, genArgs...)
	var genStdout, genStderr bytes.Buffer
	genCmd.Stdout = &genStdout
	genCmd.Stderr = &genStderr
	err := genCmd.Run()
	if err != nil {
		result.Error = fmt.Sprintf("generate tx failed: %v - stderr: %s", err, genStderr.String())
		result.Latency = time.Since(start)
		return result, nil
	}
	unsignedTx := genStdout.Bytes()

	// Write unsigned tx to temp file
	tmpFile := fmt.Sprintf("/tmp/unsigned_tx_%s_%d.json", trader, time.Now().UnixNano())
	if err := os.WriteFile(tmpFile, unsignedTx, 0644); err != nil {
		result.Error = fmt.Sprintf("write unsigned tx failed: %v", err)
		result.Latency = time.Since(start)
		return result, nil
	}
	defer os.Remove(tmpFile)

	// Step 2: Sign transaction offline
	signedFile := tmpFile + ".signed"
	signArgs := []string{
		"tx", "sign", tmpFile,
		"--from", trader,
		"--home", c.config.HomeDir,
		"--chain-id", c.config.ChainID,
		"--keyring-backend", c.config.KeyringBackend,
		"--account-number", "0",
		"--sequence", fmt.Sprintf("%d", seq),
		"--offline",
		"--output-document", signedFile,
	}

	signCmd := exec.CommandContext(ctx, c.config.BinaryPath, signArgs...)
	var signStderr bytes.Buffer
	signCmd.Stderr = &signStderr
	if err := signCmd.Run(); err != nil {
		result.Error = fmt.Sprintf("sign tx failed: %v - %s", err, signStderr.String())
		result.Latency = time.Since(start)
		return result, nil
	}
	defer os.Remove(signedFile)

	// Step 3: Broadcast signed transaction
	broadcastArgs := []string{
		"tx", "broadcast", signedFile,
		"--home", c.config.HomeDir,
		"--broadcast-mode", "sync",
		"--output", "json",
	}

	broadcastCmd := exec.CommandContext(ctx, c.config.BinaryPath, broadcastArgs...)
	var stdout, stderr bytes.Buffer
	broadcastCmd.Stdout = &stdout
	broadcastCmd.Stderr = &stderr

	err = broadcastCmd.Run()
	result.Latency = time.Since(start)

	if err != nil {
		errOutput := stderr.String()
		if strings.Contains(errOutput, "sequence mismatch") || strings.Contains(errOutput, "account sequence mismatch") {
			// Reset sequence and retry
			sequenceTracker[trader] = 0
			result.Error = "sequence mismatch - will retry"
			return result, nil
		}
		result.Error = errOutput
		return result, nil
	}

	// Parse the output
	output := stdout.String()
	var txResp struct {
		TxHash string `json:"txhash"`
		Code   int    `json:"code"`
		RawLog string `json:"raw_log"`
	}

	if err := json.Unmarshal([]byte(output), &txResp); err != nil {
		result.Error = fmt.Sprintf("parse response failed: %v", err)
		return result, nil
	}

	result.TxHash = txResp.TxHash
	result.Success = txResp.Code == 0
	result.Error = txResp.RawLog

	// Increment sequence for next transaction
	if txResp.Code == 0 {
		sequenceTracker[trader] = seq + 1
	}

	return result, nil
}

// CancelOrder cancels an existing order
func (c *ChainClient) CancelOrder(ctx context.Context, trader, orderID string) (*OrderResult, error) {
	start := time.Now()

	args := []string{
		"tx", "orderbook", "cancel-order",
		orderID,
		"--from", trader,
		"--home", c.config.HomeDir,
		"--chain-id", c.config.ChainID,
		"--keyring-backend", c.config.KeyringBackend,
		"--gas", "auto",
		"--gas-adjustment", "1.5",
		"--fees", "500usdc",
		"--broadcast-mode", "sync",
		"-y",
		"--output", "json",
	}

	result := &OrderResult{
		Latency: time.Since(start),
	}

	cmd := exec.CommandContext(ctx, c.config.BinaryPath, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	result.Latency = time.Since(start)

	if err != nil {
		result.Error = stderr.String()
		return result, nil
	}

	output := stdout.String()
	var txResp struct {
		TxHash string `json:"txhash"`
		Code   int    `json:"code"`
	}

	if err := json.Unmarshal([]byte(output), &txResp); err == nil {
		result.TxHash = txResp.TxHash
		result.Success = txResp.Code == 0
	}

	return result, nil
}

// QueryOrderBook queries the order book for a market
func (c *ChainClient) QueryOrderBook(ctx context.Context, market string) ([]byte, error) {
	args := []string{
		"query", "orderbook", "book",
		market,
		"--home", c.config.HomeDir,
		"--node", "tcp://localhost:26657",
		"--output", "json",
	}

	cmd := exec.CommandContext(ctx, c.config.BinaryPath, args...)
	return cmd.Output()
}

// QueryPositions queries all positions for a trader
func (c *ChainClient) QueryPositions(ctx context.Context, trader string) ([]byte, error) {
	args := []string{
		"query", "perpetual", "positions",
		trader,
		"--home", c.config.HomeDir,
		"--node", "tcp://localhost:26657",
		"--output", "json",
	}

	cmd := exec.CommandContext(ctx, c.config.BinaryPath, args...)
	return cmd.Output()
}

// QueryMarkets queries all available markets
func (c *ChainClient) QueryMarkets(ctx context.Context) ([]byte, error) {
	args := []string{
		"query", "perpetual", "markets",
		"--home", c.config.HomeDir,
		"--node", "tcp://localhost:26657",
		"--output", "json",
	}

	cmd := exec.CommandContext(ctx, c.config.BinaryPath, args...)
	return cmd.Output()
}

// GetBlockHeight returns the current block height
func (c *ChainClient) GetBlockHeight(ctx context.Context) (int64, error) {
	status, err := c.GetStatus(ctx)
	if err != nil {
		return 0, err
	}

	var height int64
	fmt.Sscanf(status.SyncInfo.LatestBlockHeight, "%d", &height)
	return height, nil
}

// WaitForBlocks waits for a specified number of blocks to be produced
func (c *ChainClient) WaitForBlocks(ctx context.Context, blocks int) error {
	startHeight, err := c.GetBlockHeight(ctx)
	if err != nil {
		return err
	}

	targetHeight := startHeight + int64(blocks)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			currentHeight, err := c.GetBlockHeight(ctx)
			if err != nil {
				continue
			}
			if currentHeight >= targetHeight {
				return nil
			}
		}
	}
}

// SendTokens sends tokens from one account to another
func (c *ChainClient) SendTokens(ctx context.Context, from, to, amount string) (*OrderResult, error) {
	start := time.Now()

	args := []string{
		"tx", "bank", "send",
		from, to, amount,
		"--home", c.config.HomeDir,
		"--chain-id", c.config.ChainID,
		"--keyring-backend", c.config.KeyringBackend,
		"--gas", "auto",
		"--gas-adjustment", "1.5",
		"--fees", "500usdc",
		"--broadcast-mode", "sync",
		"-y",
		"--output", "json",
	}

	result := &OrderResult{
		Latency: time.Since(start),
	}

	cmd := exec.CommandContext(ctx, c.config.BinaryPath, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	result.Latency = time.Since(start)

	if err != nil {
		result.Error = stderr.String()
		return result, nil
	}

	output := stdout.String()
	var txResp struct {
		TxHash string `json:"txhash"`
		Code   int    `json:"code"`
	}

	if err := json.Unmarshal([]byte(output), &txResp); err == nil {
		result.TxHash = txResp.TxHash
		result.Success = txResp.Code == 0
	}

	return result, nil
}

// DepositMargin deposits margin for a trader
func (c *ChainClient) DepositMargin(ctx context.Context, trader, amount string) (*OrderResult, error) {
	start := time.Now()

	args := []string{
		"tx", "clearinghouse", "deposit",
		amount,
		"--from", trader,
		"--home", c.config.HomeDir,
		"--chain-id", c.config.ChainID,
		"--keyring-backend", c.config.KeyringBackend,
		"--gas", "auto",
		"--gas-adjustment", "1.5",
		"--fees", "500usdc",
		"--broadcast-mode", "sync",
		"-y",
		"--output", "json",
	}

	result := &OrderResult{
		Latency: time.Since(start),
	}

	cmd := exec.CommandContext(ctx, c.config.BinaryPath, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	result.Latency = time.Since(start)

	if err != nil {
		result.Error = stderr.String()
		return result, nil
	}

	output := stdout.String()
	var txResp struct {
		TxHash string `json:"txhash"`
		Code   int    `json:"code"`
	}

	if err := json.Unmarshal([]byte(output), &txResp); err == nil {
		result.TxHash = txResp.TxHash
		result.Success = txResp.Code == 0
	}

	return result, nil
}
