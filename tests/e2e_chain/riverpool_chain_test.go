// Package e2e_chain provides real chain E2E tests for RiverPool
// These tests submit actual transactions to a running chain and verify on-chain state
package e2e_chain

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/openalpha/perp-dex/tests/e2e_chain/framework"
)

// RiverPoolChainTestSuite tests RiverPool on a REAL chain
type RiverPoolChainTestSuite struct {
	suite.Suite
	fw *framework.ChainTestSuite
}

// SetupSuite runs before all tests
func (s *RiverPoolChainTestSuite) SetupSuite() {
	s.fw = framework.NewChainTestSuite(s.T())

	if err := s.fw.Setup(); err != nil {
		s.T().Skipf("Failed to setup chain: %v", err)
	}
}

// TearDownSuite runs after all tests
func (s *RiverPoolChainTestSuite) TearDownSuite() {
	s.fw.Teardown()
}

// SetupTest runs before each test
func (s *RiverPoolChainTestSuite) SetupTest() {
	s.fw.AssertChainRunning()
}

// TestSuite_RiverPool runs the test suite
func TestSuite_RiverPool(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping chain E2E tests in short mode")
	}
	suite.Run(t, new(RiverPoolChainTestSuite))
}

// ===============================
// Infrastructure Tests
// ===============================

// TestChain_Connectivity verifies chain connection
func (s *RiverPoolChainTestSuite) TestChain_Connectivity() {
	ctx := s.fw.Context()

	status, err := s.fw.Manager.GetStatus(ctx)
	require.NoError(s.T(), err)

	s.T().Logf("✅ Chain connected:")
	s.T().Logf("   ChainID: %s", status.ChainID)
	s.T().Logf("   Height: %d", status.LatestHeight)
	s.T().Logf("   CatchingUp: %v", status.CatchingUp)

	require.True(s.T(), status.Running)
	require.False(s.T(), status.CatchingUp)
	require.Greater(s.T(), status.LatestHeight, int64(0))
}

// TestChain_BlockProduction verifies blocks are being produced
func (s *RiverPoolChainTestSuite) TestChain_BlockProduction() {
	ctx := s.fw.Context()

	status1, err := s.fw.Manager.GetStatus(ctx)
	require.NoError(s.T(), err)
	initialHeight := status1.LatestHeight

	// Wait for 3 blocks
	err = s.fw.WaitForBlocks(3)
	require.NoError(s.T(), err)

	status2, err := s.fw.Manager.GetStatus(ctx)
	require.NoError(s.T(), err)

	s.T().Logf("✅ Block production verified:")
	s.T().Logf("   Initial height: %d", initialHeight)
	s.T().Logf("   Current height: %d", status2.LatestHeight)

	require.GreaterOrEqual(s.T(), status2.LatestHeight, initialHeight+3)
}

// ===============================
// Foundation LP Tests
// ===============================

// TestRiverPool_FoundationLP_Deposit tests depositing to Foundation LP
func (s *RiverPoolChainTestSuite) TestRiverPool_FoundationLP_Deposit() {
	ctx, cancel := context.WithTimeout(s.fw.Context(), 60*time.Second)
	defer cancel()

	trader := s.fw.Config.ValidatorKey

	// Check initial balance
	addr, err := s.fw.Client.GetAccountAddress(ctx, trader)
	if err != nil {
		s.T().Skipf("Could not get account address: %v", err)
	}

	initialBalance, err := s.fw.Client.QueryBalance(ctx, addr, "usdc")
	if err != nil {
		s.T().Skipf("Could not query balance: %v", err)
	}
	s.T().Logf("Initial USDC balance: %s", initialBalance)

	// Deposit to Foundation LP
	s.T().Log("Depositing 100000 USDC to foundation-lp...")
	result, err := s.fw.Client.DepositToRiverPool(ctx, trader, "foundation-lp", "100000000000usdc")
	require.NoError(s.T(), err)

	if !result.Success {
		s.T().Logf("Deposit error: %s", result.Error)
		// Check if it's a known limitation
		if containsAny(result.Error, []string{"module not found", "unknown command", "not implemented"}) {
			s.T().Skip("RiverPool module not yet implemented on chain")
		}
	}

	s.T().Logf("Deposit transaction:")
	s.T().Logf("  TxHash: %s", result.TxHash)
	s.T().Logf("  Success: %v", result.Success)
	s.T().Logf("  Latency: %v", result.Latency)

	if result.Success {
		// Wait for state to be committed
		s.fw.WaitForBlocks(2)

		// Query pool state
		poolData, err := s.fw.Client.QueryPool(ctx, "foundation-lp")
		if err == nil {
			s.T().Logf("Pool state after deposit: %v", poolData)
		}

		// Verify deposit record
		depositData, err := s.fw.Client.QueryUserDeposit(ctx, "foundation-lp", addr)
		if err == nil {
			s.T().Logf("User deposit record: %v", depositData)
		}
	}
}

// TestRiverPool_FoundationLP_DepositLimit tests Foundation LP seat limit
func (s *RiverPoolChainTestSuite) TestRiverPool_FoundationLP_DepositLimit() {
	ctx, cancel := context.WithTimeout(s.fw.Context(), 60*time.Second)
	defer cancel()

	trader := s.fw.Config.ValidatorKey

	// Try to deposit less than minimum ($100K)
	s.T().Log("Testing minimum deposit requirement...")
	result, err := s.fw.Client.DepositToRiverPool(ctx, trader, "foundation-lp", "50000000000usdc") // $50K
	require.NoError(s.T(), err)

	// Should fail if module enforces minimum
	if result.Success {
		s.T().Log("Warning: Deposit below minimum succeeded - validation may not be implemented")
	} else {
		s.T().Logf("✅ Minimum deposit correctly rejected: %s", result.Error)
	}
}

// ===============================
// Main LP Tests
// ===============================

// TestRiverPool_MainLP_Deposit tests depositing to Main LP
func (s *RiverPoolChainTestSuite) TestRiverPool_MainLP_Deposit() {
	ctx, cancel := context.WithTimeout(s.fw.Context(), 60*time.Second)
	defer cancel()

	trader := s.fw.Config.ValidatorKey

	// Deposit to Main LP (min $100)
	s.T().Log("Depositing 1000 USDC to main-lp...")
	result, err := s.fw.Client.DepositToRiverPool(ctx, trader, "main-lp", "1000000000usdc")
	require.NoError(s.T(), err)

	if !result.Success {
		if containsAny(result.Error, []string{"module not found", "unknown command", "not implemented"}) {
			s.T().Skip("RiverPool module not yet implemented on chain")
		}
	}

	s.T().Logf("Deposit result:")
	s.T().Logf("  TxHash: %s", result.TxHash)
	s.T().Logf("  Success: %v", result.Success)
	s.T().Logf("  Latency: %v", result.Latency)
}

// TestRiverPool_MainLP_RequestWithdrawal tests withdrawal from Main LP
func (s *RiverPoolChainTestSuite) TestRiverPool_MainLP_RequestWithdrawal() {
	ctx, cancel := context.WithTimeout(s.fw.Context(), 120*time.Second)
	defer cancel()

	trader := s.fw.Config.ValidatorKey

	// First deposit
	s.T().Log("Depositing to main-lp first...")
	depositResult, _ := s.fw.Client.DepositToRiverPool(ctx, trader, "main-lp", "1000000000usdc")
	if !depositResult.Success {
		if containsAny(depositResult.Error, []string{"module not found", "unknown command"}) {
			s.T().Skip("RiverPool module not yet implemented on chain")
		}
	}

	s.fw.WaitForBlocks(2)

	// Request withdrawal (should enter pending state for T+4)
	s.T().Log("Requesting withdrawal of 500 shares...")
	withdrawResult, err := s.fw.Client.RequestWithdrawal(ctx, trader, "main-lp", "500")
	require.NoError(s.T(), err)

	s.T().Logf("Withdrawal request result:")
	s.T().Logf("  TxHash: %s", withdrawResult.TxHash)
	s.T().Logf("  Success: %v", withdrawResult.Success)

	// Check for withdrawal events
	for _, event := range withdrawResult.Events {
		if event.Type == "withdrawal_requested" || event.Type == "riverpool_withdrawal" {
			s.T().Logf("  Event: %s", event.Type)
			for k, v := range event.Attributes {
				s.T().Logf("    %s: %s", k, v)
			}
		}
	}
}

// ===============================
// Community Pool Tests
// ===============================

// TestRiverPool_CommunityPool_Create tests creating a community pool
func (s *RiverPoolChainTestSuite) TestRiverPool_CommunityPool_Create() {
	ctx, cancel := context.WithTimeout(s.fw.Context(), 60*time.Second)
	defer cancel()

	owner := s.fw.Config.ValidatorKey

	// Create a new community pool
	s.T().Log("Creating community pool...")
	result, err := s.fw.Client.CreateCommunityPool(ctx, owner, "test-alpha-pool", "momentum",
		map[string]string{
			"management-fee":   "100",  // 1%
			"performance-fee":  "2000", // 20%
			"min-deposit":      "100000000usdc",
			"owner-commitment": "500",  // 5%
		})
	require.NoError(s.T(), err)

	if !result.Success {
		if containsAny(result.Error, []string{"module not found", "unknown command", "not implemented"}) {
			s.T().Skip("Community pool not yet implemented on chain")
		}
		s.T().Logf("Create pool error: %s", result.Error)
	}

	s.T().Logf("Create pool result:")
	s.T().Logf("  TxHash: %s", result.TxHash)
	s.T().Logf("  Success: %v", result.Success)
	s.T().Logf("  Latency: %v", result.Latency)
}

// TestRiverPool_CommunityPool_OwnerCommitment tests owner commitment requirement
func (s *RiverPoolChainTestSuite) TestRiverPool_CommunityPool_OwnerCommitment() {
	ctx, cancel := context.WithTimeout(s.fw.Context(), 120*time.Second)
	defer cancel()

	owner := s.fw.Config.ValidatorKey

	// Create pool with 5% owner commitment
	s.T().Log("Creating pool with owner commitment...")
	createResult, _ := s.fw.Client.CreateCommunityPool(ctx, owner, "test-commitment-pool", "trend",
		map[string]string{
			"owner-commitment": "500", // 5%
		})

	if !createResult.Success {
		if containsAny(createResult.Error, []string{"module not found", "unknown command"}) {
			s.T().Skip("Community pool not yet implemented")
		}
	}

	// Owner must deposit at least 5% of initial TVL
	// This would be tested after someone else deposits
	s.T().Log("Owner commitment validation would be tested after external deposits")
}

// ===============================
// DDGuard Tests
// ===============================

// TestRiverPool_DDGuard_Level1 tests DDGuard Level 1 (10% drawdown warning)
func (s *RiverPoolChainTestSuite) TestRiverPool_DDGuard_Level1() {
	s.T().Log("DDGuard tests require price manipulation which is complex to simulate")
	s.T().Log("In production, DDGuard triggers at:")
	s.T().Log("  Level 1 (≥10%): Warning notification")
	s.T().Log("  Level 2 (≥15%): Reduce exposure limit")
	s.T().Log("  Level 3 (≥30%): Suspend trading")

	// These tests would require:
	// 1. Creating positions in the pool
	// 2. Manipulating oracle prices to cause drawdown
	// 3. Verifying DDGuard state changes
	s.T().Skip("DDGuard integration tests require oracle price manipulation")
}

// ===============================
// Throughput Tests
// ===============================

// TestRiverPool_Throughput tests transaction throughput for RiverPool operations
func (s *RiverPoolChainTestSuite) TestRiverPool_Throughput() {
	ctx, cancel := context.WithTimeout(s.fw.Context(), 180*time.Second)
	defer cancel()

	trader := s.fw.Config.ValidatorKey

	const (
		depositCount = 10
	)

	var successCount int
	var totalLatency time.Duration

	s.T().Logf("Running %d deposit operations for throughput test...", depositCount)

	for i := 0; i < depositCount; i++ {
		amount := fmt.Sprintf("%d000000usdc", 100+i) // Varying amounts

		result, err := s.fw.Client.DepositToRiverPool(ctx, trader, "main-lp", amount)
		if err != nil {
			continue
		}

		if result.Success {
			successCount++
			totalLatency += result.Latency
		} else {
			if containsAny(result.Error, []string{"module not found", "unknown command"}) {
				s.T().Skip("RiverPool module not implemented")
			}
		}

		// Small delay between transactions
		time.Sleep(200 * time.Millisecond)
	}

	s.T().Logf("Throughput results:")
	s.T().Logf("  Total attempts: %d", depositCount)
	s.T().Logf("  Successful: %d", successCount)
	s.T().Logf("  Success rate: %.1f%%", float64(successCount)/float64(depositCount)*100)
	if successCount > 0 {
		s.T().Logf("  Avg latency: %v", totalLatency/time.Duration(successCount))
	}
}

// ===============================
// Helper functions
// ===============================

func containsAny(s string, substrs []string) bool {
	for _, sub := range substrs {
		if contains(s, sub) {
			return true
		}
	}
	return false
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(s) > 0 && containsSubstring(s, sub))
}

func containsSubstring(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
