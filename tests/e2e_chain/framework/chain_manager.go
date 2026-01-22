// Package framework provides real chain E2E testing infrastructure
package framework

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
	"sync"
	"syscall"
	"time"
)

// ChainManager handles the lifecycle of a test chain
type ChainManager struct {
	config      *ChainTestConfig
	httpClient  *http.Client
	chainCmd    *exec.Cmd
	mu          sync.Mutex
	isRunning   bool
	startedByUs bool
	logFile     *os.File
}

// NewChainManager creates a new chain manager
func NewChainManager(config *ChainTestConfig) *ChainManager {
	return &ChainManager{
		config: config,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// ChainStatus represents the current chain status
type ChainStatus struct {
	Running          bool
	ChainID          string
	LatestHeight     int64
	LatestBlockTime  time.Time
	CatchingUp       bool
	ValidatorAddress string
}

// GetStatus returns the current chain status
func (m *ChainManager) GetStatus(ctx context.Context) (*ChainStatus, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", m.config.RPCURL+"/status", nil)
	if err != nil {
		return nil, err
	}

	resp, err := m.httpClient.Do(req)
	if err != nil {
		return &ChainStatus{Running: false}, nil
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result struct {
		Result struct {
			NodeInfo struct {
				Network string `json:"network"`
			} `json:"node_info"`
			SyncInfo struct {
				LatestBlockHeight string `json:"latest_block_height"`
				LatestBlockTime   string `json:"latest_block_time"`
				CatchingUp        bool   `json:"catching_up"`
			} `json:"sync_info"`
			ValidatorInfo struct {
				Address string `json:"address"`
			} `json:"validator_info"`
		} `json:"result"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse status: %w", err)
	}

	var height int64
	fmt.Sscanf(result.Result.SyncInfo.LatestBlockHeight, "%d", &height)

	blockTime, _ := time.Parse(time.RFC3339Nano, result.Result.SyncInfo.LatestBlockTime)

	return &ChainStatus{
		Running:          true,
		ChainID:          result.Result.NodeInfo.Network,
		LatestHeight:     height,
		LatestBlockTime:  blockTime,
		CatchingUp:       result.Result.SyncInfo.CatchingUp,
		ValidatorAddress: result.Result.ValidatorInfo.Address,
	}, nil
}

// IsRunning checks if the chain is running
func (m *ChainManager) IsRunning(ctx context.Context) bool {
	status, err := m.GetStatus(ctx)
	return err == nil && status.Running
}

// EnsureRunning ensures the chain is running, starting it if necessary
func (m *ChainManager) EnsureRunning(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.IsRunning(ctx) {
		m.isRunning = true
		return nil
	}

	if !m.config.AutoStartChain {
		return fmt.Errorf("chain not running and AutoStartChain is disabled")
	}

	return m.startChainLocked(ctx)
}

// startChainLocked starts the chain (must be called with lock held)
func (m *ChainManager) startChainLocked(ctx context.Context) error {
	// Ensure the home directory and chain are initialized
	if err := m.ensureInitialized(ctx); err != nil {
		return fmt.Errorf("failed to initialize chain: %w", err)
	}

	// Create log file
	logPath := filepath.Join(m.config.HomeDir, "chain.log")
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("failed to create log file: %w", err)
	}
	m.logFile = logFile

	// Start the chain
	args := []string{
		"start",
		"--home", m.config.HomeDir,
		"--minimum-gas-prices", "0usdc",
		"--log_level", "info",
	}

	m.chainCmd = exec.CommandContext(ctx, m.config.BinaryPath, args...)
	m.chainCmd.Stdout = logFile
	m.chainCmd.Stderr = logFile

	// Set process group for cleanup
	m.chainCmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	if err := m.chainCmd.Start(); err != nil {
		logFile.Close()
		return fmt.Errorf("failed to start chain: %w", err)
	}

	m.startedByUs = true

	// Wait for chain to be ready
	if err := m.waitForReady(ctx); err != nil {
		m.stopChainLocked()
		return fmt.Errorf("chain failed to become ready: %w", err)
	}

	m.isRunning = true
	return nil
}

// ensureInitialized ensures the chain is initialized
func (m *ChainManager) ensureInitialized(ctx context.Context) error {
	// Check if already initialized
	genesisPath := filepath.Join(m.config.HomeDir, "config", "genesis.json")
	if _, err := os.Stat(genesisPath); err == nil {
		return nil // Already initialized
	}

	// Run init script
	scriptPath := filepath.Join(findProjectRoot(), "scripts", "init-test-chain.sh")
	if _, err := os.Stat(scriptPath); err != nil {
		// Create a basic init script if not exists
		return m.basicInit(ctx)
	}

	cmd := exec.CommandContext(ctx, "bash", scriptPath)
	cmd.Dir = findProjectRoot()
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("init script failed: %v, stderr: %s", err, stderr.String())
	}

	return nil
}

// basicInit performs basic chain initialization
func (m *ChainManager) basicInit(ctx context.Context) error {
	binary := m.config.BinaryPath
	home := m.config.HomeDir
	chainID := m.config.ChainID
	keyring := m.config.KeyringBackend

	// Remove old data
	os.RemoveAll(home)

	commands := [][]string{
		{"init", "test-node", "--chain-id", chainID, "--home", home},
		{"keys", "add", "validator", "--keyring-backend", keyring, "--home", home},
	}

	for _, trader := range m.config.TestAccounts {
		commands = append(commands, []string{
			"keys", "add", trader, "--keyring-backend", keyring, "--home", home,
		})
	}

	for _, args := range commands {
		cmd := exec.CommandContext(ctx, binary, args...)
		var stderr bytes.Buffer
		cmd.Stderr = &stderr
		if err := cmd.Run(); err != nil {
			// Some commands fail if key exists, which is OK
			if !strings.Contains(stderr.String(), "already exists") {
				return fmt.Errorf("init command %v failed: %v, stderr: %s", args, err, stderr.String())
			}
		}
	}

	// Add genesis accounts with initial balance
	allAccounts := append([]string{m.config.ValidatorKey}, m.config.TestAccounts...)
	for _, account := range allAccounts {
		// Get address
		addrCmd := exec.CommandContext(ctx, binary, "keys", "show", account,
			"--keyring-backend", keyring, "--home", home, "-a")
		addrOut, err := addrCmd.Output()
		if err != nil {
			continue
		}
		addr := strings.TrimSpace(string(addrOut))

		// Add genesis account
		addCmd := exec.CommandContext(ctx, binary, "add-genesis-account", addr,
			"10000000000usdc,10000000000ubtc,10000000000ueth",
			"--home", home, "--keyring-backend", keyring)
		addCmd.Run()
	}

	// Create gentx
	gentxCmd := exec.CommandContext(ctx, binary, "gentx", "validator", "1000000usdc",
		"--chain-id", chainID, "--home", home, "--keyring-backend", keyring)
	gentxCmd.Run()

	// Collect gentxs
	collectCmd := exec.CommandContext(ctx, binary, "collect-gentxs", "--home", home)
	collectCmd.Run()

	return nil
}

// waitForReady waits for the chain to be ready
func (m *ChainManager) waitForReady(ctx context.Context) error {
	timeout := time.After(m.config.ChainStartTimeout)
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timeout:
			return fmt.Errorf("timeout waiting for chain to start")
		case <-ticker.C:
			status, err := m.GetStatus(ctx)
			if err == nil && status.Running && status.LatestHeight > 0 && !status.CatchingUp {
				return nil
			}
		}
	}
}

// Stop stops the chain if it was started by us
func (m *ChainManager) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.stopChainLocked()
}

// stopChainLocked stops the chain (must be called with lock held)
func (m *ChainManager) stopChainLocked() error {
	if !m.startedByUs || m.chainCmd == nil {
		return nil
	}

	// Send SIGTERM to process group
	if m.chainCmd.Process != nil {
		syscall.Kill(-m.chainCmd.Process.Pid, syscall.SIGTERM)

		// Wait for graceful shutdown
		done := make(chan error, 1)
		go func() {
			done <- m.chainCmd.Wait()
		}()

		select {
		case <-done:
			// Process exited
		case <-time.After(10 * time.Second):
			// Force kill
			syscall.Kill(-m.chainCmd.Process.Pid, syscall.SIGKILL)
		}
	}

	if m.logFile != nil {
		m.logFile.Close()
		m.logFile = nil
	}

	m.chainCmd = nil
	m.isRunning = false
	m.startedByUs = false

	return nil
}

// Reset resets the chain state (stop, clean, restart)
func (m *ChainManager) Reset(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Stop if running
	if err := m.stopChainLocked(); err != nil {
		return err
	}

	// Clean data
	dataPath := filepath.Join(m.config.HomeDir, "data")
	if err := os.RemoveAll(dataPath); err != nil {
		return fmt.Errorf("failed to clean data: %w", err)
	}

	// Restart
	return m.startChainLocked(ctx)
}

// WaitForBlocks waits for the specified number of blocks to be produced
func (m *ChainManager) WaitForBlocks(ctx context.Context, blocks int) error {
	startStatus, err := m.GetStatus(ctx)
	if err != nil {
		return err
	}

	targetHeight := startStatus.LatestHeight + int64(blocks)
	timeout := time.Duration(blocks) * m.config.BlockTime * 3 // 3x safety margin

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for blocks")
		case <-ticker.C:
			status, err := m.GetStatus(ctx)
			if err != nil {
				continue
			}
			if status.LatestHeight >= targetHeight {
				return nil
			}
		}
	}
}

// Cleanup performs cleanup after tests
func (m *ChainManager) Cleanup() error {
	if err := m.Stop(); err != nil {
		return err
	}

	if m.config.CleanupOnExit {
		return os.RemoveAll(m.config.HomeDir)
	}

	return nil
}

// GetChainLogs returns the chain logs
func (m *ChainManager) GetChainLogs() (string, error) {
	logPath := filepath.Join(m.config.HomeDir, "chain.log")
	data, err := os.ReadFile(logPath)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// GetTailLogs returns the last n lines of chain logs
func (m *ChainManager) GetTailLogs(lines int) (string, error) {
	logs, err := m.GetChainLogs()
	if err != nil {
		return "", err
	}

	logLines := strings.Split(logs, "\n")
	start := len(logLines) - lines
	if start < 0 {
		start = 0
	}

	return strings.Join(logLines[start:], "\n"), nil
}
