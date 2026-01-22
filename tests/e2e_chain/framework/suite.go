// Package framework provides real chain E2E testing infrastructure
package framework

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"sync"
	"testing"
	"time"
)

// ChainTestSuite provides a complete test suite infrastructure for chain E2E tests
type ChainTestSuite struct {
	T       *testing.T
	Config  *ChainTestConfig
	Manager *ChainManager
	Client  *ChainClient

	// Test state
	ctx    context.Context
	cancel context.CancelFunc
}

// ChainClient provides methods for interacting with the chain during tests
type ChainClient struct {
	config  *ChainTestConfig
	manager *ChainManager
	seqLock sync.Mutex
	seqMap  map[string]uint64
}

// NewChainTestSuite creates a new test suite
func NewChainTestSuite(t *testing.T) *ChainTestSuite {
	config := DefaultChainTestConfig()

	// Check for CI environment
	if isCI() {
		config = CIChainTestConfig()
	}

	ctx, cancel := context.WithCancel(context.Background())

	manager := NewChainManager(config)
	client := &ChainClient{
		config:  config,
		manager: manager,
		seqMap:  make(map[string]uint64),
	}

	return &ChainTestSuite{
		T:       t,
		Config:  config,
		Manager: manager,
		Client:  client,
		ctx:     ctx,
		cancel:  cancel,
	}
}

// Setup initializes the test suite
func (s *ChainTestSuite) Setup() error {
	s.T.Log("Setting up chain test suite...")

	// Ensure chain is running
	if err := s.Manager.EnsureRunning(s.ctx); err != nil {
		return fmt.Errorf("failed to ensure chain running: %w", err)
	}

	// Verify chain status
	status, err := s.Manager.GetStatus(s.ctx)
	if err != nil {
		return fmt.Errorf("failed to get chain status: %w", err)
	}

	s.T.Logf("Chain running:")
	s.T.Logf("  Chain ID: %s", status.ChainID)
	s.T.Logf("  Height: %d", status.LatestHeight)
	s.T.Logf("  Validator: %s", status.ValidatorAddress)

	return nil
}

// Teardown cleans up the test suite
func (s *ChainTestSuite) Teardown() {
	s.T.Log("Tearing down chain test suite...")
	s.cancel()

	if err := s.Manager.Cleanup(); err != nil {
		s.T.Logf("Cleanup error: %v", err)
	}
}

// Context returns the test context
func (s *ChainTestSuite) Context() context.Context {
	return s.ctx
}

// WaitForBlocks waits for the specified number of blocks
func (s *ChainTestSuite) WaitForBlocks(blocks int) error {
	return s.Manager.WaitForBlocks(s.ctx, blocks)
}

// AssertChainRunning asserts the chain is running
func (s *ChainTestSuite) AssertChainRunning() {
	s.T.Helper()
	if !s.Manager.IsRunning(s.ctx) {
		s.T.Fatal("Chain is not running")
	}
}

// SkipIfChainNotRunning skips the test if chain is not running
func (s *ChainTestSuite) SkipIfChainNotRunning() {
	s.T.Helper()
	if !s.Manager.IsRunning(s.ctx) {
		s.T.Skip("Chain is not running")
	}
}

// ===============================
// ChainClient methods
// ===============================

// TxResult represents a transaction result
type TxResult struct {
	Success bool
	TxHash  string
	Height  int64
	GasUsed int64
	Error   string
	Latency time.Duration
	Events  []TxEvent
}

// TxEvent represents a transaction event
type TxEvent struct {
	Type       string
	Attributes map[string]string
}

// PlaceOrder places an order on the chain
func (c *ChainClient) PlaceOrder(ctx context.Context, trader, market, side, orderType, price, quantity string) (*TxResult, error) {
	start := time.Now()

	args := []string{
		"tx", "orderbook", "place-order",
		market, side, orderType, price, quantity,
		"--from", trader,
		"--home", c.config.HomeDir,
		"--chain-id", c.config.ChainID,
		"--keyring-backend", c.config.KeyringBackend,
		"--gas", "200000",
		"--fees", "1000usdc",
		"--broadcast-mode", "sync",
		"-y",
		"--output", "json",
	}

	result := c.execTx(ctx, args)
	result.Latency = time.Since(start)
	return result, nil
}

// CancelOrder cancels an order
func (c *ChainClient) CancelOrder(ctx context.Context, trader, orderID string) (*TxResult, error) {
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

	result := c.execTx(ctx, args)
	result.Latency = time.Since(start)
	return result, nil
}

// DepositToRiverPool deposits to a RiverPool
func (c *ChainClient) DepositToRiverPool(ctx context.Context, depositor, poolID, amount string) (*TxResult, error) {
	start := time.Now()

	args := []string{
		"tx", "riverpool", "deposit",
		poolID, amount,
		"--from", depositor,
		"--home", c.config.HomeDir,
		"--chain-id", c.config.ChainID,
		"--keyring-backend", c.config.KeyringBackend,
		"--gas", "200000",
		"--fees", "1000usdc",
		"--broadcast-mode", "sync",
		"-y",
		"--output", "json",
	}

	result := c.execTx(ctx, args)
	result.Latency = time.Since(start)
	return result, nil
}

// RequestWithdrawal requests a withdrawal from RiverPool
func (c *ChainClient) RequestWithdrawal(ctx context.Context, withdrawer, poolID, shares string) (*TxResult, error) {
	start := time.Now()

	args := []string{
		"tx", "riverpool", "request-withdrawal",
		poolID, shares,
		"--from", withdrawer,
		"--home", c.config.HomeDir,
		"--chain-id", c.config.ChainID,
		"--keyring-backend", c.config.KeyringBackend,
		"--gas", "200000",
		"--fees", "1000usdc",
		"--broadcast-mode", "sync",
		"-y",
		"--output", "json",
	}

	result := c.execTx(ctx, args)
	result.Latency = time.Since(start)
	return result, nil
}

// CreateCommunityPool creates a community pool
func (c *ChainClient) CreateCommunityPool(ctx context.Context, owner, name, strategy string, params map[string]string) (*TxResult, error) {
	start := time.Now()

	args := []string{
		"tx", "riverpool", "create-community-pool",
		name, strategy,
		"--from", owner,
		"--home", c.config.HomeDir,
		"--chain-id", c.config.ChainID,
		"--keyring-backend", c.config.KeyringBackend,
		"--gas", "300000",
		"--fees", "2000usdc",
		"--broadcast-mode", "sync",
		"-y",
		"--output", "json",
	}

	// Add optional params
	for key, val := range params {
		args = append(args, "--"+key, val)
	}

	result := c.execTx(ctx, args)
	result.Latency = time.Since(start)
	return result, nil
}

// SendTokens sends tokens between accounts
func (c *ChainClient) SendTokens(ctx context.Context, from, to, amount string) (*TxResult, error) {
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

	result := c.execTx(ctx, args)
	result.Latency = time.Since(start)
	return result, nil
}

// execTx executes a transaction command
func (c *ChainClient) execTx(ctx context.Context, args []string) *TxResult {
	result := &TxResult{}

	cmd := exec.CommandContext(ctx, c.config.BinaryPath, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		result.Error = fmt.Sprintf("%v: %s", err, stderr.String())
		return result
	}

	// Parse JSON response
	var txResp struct {
		TxHash string `json:"txhash"`
		Code   int    `json:"code"`
		RawLog string `json:"raw_log"`
		Height string `json:"height"`
		Events []struct {
			Type       string `json:"type"`
			Attributes []struct {
				Key   string `json:"key"`
				Value string `json:"value"`
			} `json:"attributes"`
		} `json:"events"`
	}

	if err := json.Unmarshal(stdout.Bytes(), &txResp); err != nil {
		result.Error = fmt.Sprintf("failed to parse response: %v", err)
		return result
	}

	result.TxHash = txResp.TxHash
	result.Success = txResp.Code == 0
	result.Error = txResp.RawLog

	fmt.Sscanf(txResp.Height, "%d", &result.Height)

	// Parse events
	for _, ev := range txResp.Events {
		event := TxEvent{
			Type:       ev.Type,
			Attributes: make(map[string]string),
		}
		for _, attr := range ev.Attributes {
			event.Attributes[attr.Key] = attr.Value
		}
		result.Events = append(result.Events, event)
	}

	return result
}

// QueryOrderBook queries the order book
func (c *ChainClient) QueryOrderBook(ctx context.Context, market string) (map[string]interface{}, error) {
	args := []string{
		"query", "orderbook", "book",
		market,
		"--home", c.config.HomeDir,
		"--node", "tcp://localhost:26657",
		"--output", "json",
	}

	cmd := exec.CommandContext(ctx, c.config.BinaryPath, args...)
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(output, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// QueryPool queries a RiverPool
func (c *ChainClient) QueryPool(ctx context.Context, poolID string) (map[string]interface{}, error) {
	args := []string{
		"query", "riverpool", "pool",
		poolID,
		"--home", c.config.HomeDir,
		"--node", "tcp://localhost:26657",
		"--output", "json",
	}

	cmd := exec.CommandContext(ctx, c.config.BinaryPath, args...)
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(output, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// QueryUserDeposit queries a user's deposit in a pool
func (c *ChainClient) QueryUserDeposit(ctx context.Context, poolID, user string) (map[string]interface{}, error) {
	args := []string{
		"query", "riverpool", "deposit",
		poolID, user,
		"--home", c.config.HomeDir,
		"--node", "tcp://localhost:26657",
		"--output", "json",
	}

	cmd := exec.CommandContext(ctx, c.config.BinaryPath, args...)
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(output, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// QueryBalance queries an account balance
func (c *ChainClient) QueryBalance(ctx context.Context, address, denom string) (string, error) {
	args := []string{
		"query", "bank", "balances",
		address,
		"--denom", denom,
		"--home", c.config.HomeDir,
		"--node", "tcp://localhost:26657",
		"--output", "json",
	}

	cmd := exec.CommandContext(ctx, c.config.BinaryPath, args...)
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	var result struct {
		Balances []struct {
			Denom  string `json:"denom"`
			Amount string `json:"amount"`
		} `json:"balances"`
	}

	if err := json.Unmarshal(output, &result); err != nil {
		return "", err
	}

	for _, bal := range result.Balances {
		if bal.Denom == denom {
			return bal.Amount, nil
		}
	}

	return "0", nil
}

// GetAccountAddress gets the address for a key name
func (c *ChainClient) GetAccountAddress(ctx context.Context, keyName string) (string, error) {
	args := []string{
		"keys", "show", keyName,
		"--home", c.config.HomeDir,
		"--keyring-backend", c.config.KeyringBackend,
		"-a",
	}

	cmd := exec.CommandContext(ctx, c.config.BinaryPath, args...)
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(output)), nil
}

// isCI checks if running in CI environment
func isCI() bool {
	ciEnvVars := []string{"CI", "GITHUB_ACTIONS", "GITLAB_CI", "CIRCLECI", "JENKINS_URL"}
	for _, env := range ciEnvVars {
		if val := getEnvOrDefault(env, ""); val != "" {
			return true
		}
	}
	return false
}
