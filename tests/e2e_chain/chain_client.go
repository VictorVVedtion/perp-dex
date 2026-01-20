// Package e2e_chain provides chain client for E2E testing
package e2e_chain

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"
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
	return &ChainConfig{
		RPCURL:         "http://localhost:26657",
		APIURL:         "http://localhost:1317",
		GRPCAddr:       "localhost:9090",
		ChainID:        "perpdex-test-1",
		HomeDir:        ".perpdex-test",
		BinaryPath:     "./build/perpdexd",
		KeyringBackend: "test",
	}
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

// PlaceOrder places an order on the chain via CLI
func (c *ChainClient) PlaceOrder(ctx context.Context, trader, market, side, orderType, price, quantity string) (*OrderResult, error) {
	start := time.Now()

	// Build the transaction command
	args := []string{
		"tx", "orderbook", "place-order",
		market,
		side,
		orderType,
		price,
		quantity,
		"--from", trader,
		"--home", c.config.HomeDir,
		"--chain-id", c.config.ChainID,
		"--keyring-backend", c.config.KeyringBackend,
		"--gas", "auto",
		"--gas-adjustment", "1.5",
		"--fees", "1000usdc",
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
		// Check if it's a known error that we should handle
		errOutput := stderr.String()
		if strings.Contains(errOutput, "sequence mismatch") {
			// Retry with updated sequence
			result.Error = "sequence mismatch"
			return result, nil
		}
		result.Error = errOutput
		return result, fmt.Errorf("command failed: %w - %s", err, errOutput)
	}

	// Parse the output to get tx hash and result
	output := stdout.String()
	var txResp struct {
		TxHash string `json:"txhash"`
		Code   int    `json:"code"`
		RawLog string `json:"raw_log"`
		GasUsed string `json:"gas_used"`
	}

	if err := json.Unmarshal([]byte(output), &txResp); err != nil {
		// Try to extract txhash from non-JSON output
		if strings.Contains(output, "txhash:") {
			parts := strings.Split(output, "txhash:")
			if len(parts) > 1 {
				result.TxHash = strings.TrimSpace(strings.Split(parts[1], "\n")[0])
				result.Success = true
			}
		}
		return result, nil
	}

	result.TxHash = txResp.TxHash
	result.Success = txResp.Code == 0
	result.Error = txResp.RawLog

	// Parse gas used
	if txResp.GasUsed != "" {
		fmt.Sscanf(txResp.GasUsed, "%d", &result.GasUsed)
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
