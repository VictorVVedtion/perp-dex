// Package e2e_real provides real end-to-end testing for RiverPool API
// Tests actual HTTP connections to a running API server without mock data
package e2e_real

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"
)

// RiverpoolTestConfig extends TestConfig with riverpool-specific settings
type RiverpoolTestConfig struct {
	*TestConfig
	TestUser     string
	TestPoolID   string
	DepositAmount string
}

// DefaultRiverpoolConfig returns config for riverpool testing
func DefaultRiverpoolConfig() *RiverpoolTestConfig {
	return &RiverpoolTestConfig{
		TestConfig:    DefaultConfig(),
		TestUser:      fmt.Sprintf("test_user_%d", time.Now().UnixNano()),
		TestPoolID:    "foundation-lp",
		DepositAmount: "1000.00",
	}
}

// ===========================================
// Pool Query Tests
// ===========================================

// TestRiverpool_GetPools tests fetching all pools
func TestRiverpool_GetPools(t *testing.T) {
	config := DefaultConfig()
	client := NewHTTPClient(config)

	result := client.GET("/v1/riverpool/pools")
	if result.Error != nil {
		t.Skipf("API server not running: %v", result.Error)
	}

	// In standalone API mode, RiverPool routes may not be registered
	if result.StatusCode == http.StatusNotFound {
		t.Skipf("RiverPool routes not available in standalone mode (404)")
	}

	if result.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", result.StatusCode)
	}

	var response struct {
		Pools []map[string]interface{} `json:"pools"`
		Total int                      `json:"total"`
	}
	if err := json.Unmarshal(result.Response.Data, &response); err == nil {
		t.Logf("Found %d pools, latency: %v", len(response.Pools), result.Latency)
		for _, pool := range response.Pools {
			t.Logf("  Pool: %s (%s) - NAV: %v", pool["name"], pool["pool_id"], pool["nav"])
		}
	}
}

// TestRiverpool_GetPool tests fetching a single pool
func TestRiverpool_GetPool(t *testing.T) {
	config := DefaultConfig()
	client := NewHTTPClient(config)

	// First get all pools to find a valid pool ID
	listResult := client.GET("/v1/riverpool/pools")
	if listResult.Error != nil {
		t.Skipf("API server not running: %v", listResult.Error)
	}

	var listResponse struct {
		Pools []map[string]interface{} `json:"pools"`
	}
	if err := json.Unmarshal(listResult.Response.Data, &listResponse); err != nil || len(listResponse.Pools) == 0 {
		t.Skip("No pools available for testing")
	}

	poolID := listResponse.Pools[0]["pool_id"].(string)
	result := client.GET(fmt.Sprintf("/v1/riverpool/pools/%s", poolID))

	if result.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", result.StatusCode)
	}

	t.Logf("Pool details fetched, latency: %v", result.Latency)
}

// TestRiverpool_GetPoolsByType tests filtering pools by type
func TestRiverpool_GetPoolsByType(t *testing.T) {
	config := DefaultConfig()
	client := NewHTTPClient(config)

	poolTypes := []string{"foundation", "main", "community"}

	for _, poolType := range poolTypes {
		result := client.GET(fmt.Sprintf("/v1/riverpool/pools/type/%s", poolType))
		if result.Error != nil {
			t.Skipf("API server not running: %v", result.Error)
		}

		t.Logf("Pool type '%s': status=%d, latency=%v", poolType, result.StatusCode, result.Latency)
	}
}

// TestRiverpool_GetPoolStats tests fetching pool statistics
func TestRiverpool_GetPoolStats(t *testing.T) {
	config := DefaultConfig()
	client := NewHTTPClient(config)

	// Get a pool ID first
	poolID := getFirstPoolID(t, client)
	if poolID == "" {
		t.Skip("No pools available")
	}

	result := client.GET(fmt.Sprintf("/v1/riverpool/pools/%s/stats", poolID))
	if result.Error != nil {
		t.Skipf("Failed to get stats: %v", result.Error)
	}

	t.Logf("Pool stats: status=%d, latency=%v", result.StatusCode, result.Latency)
}

// TestRiverpool_GetNAVHistory tests fetching NAV history
func TestRiverpool_GetNAVHistory(t *testing.T) {
	config := DefaultConfig()
	client := NewHTTPClient(config)

	poolID := getFirstPoolID(t, client)
	if poolID == "" {
		t.Skip("No pools available")
	}

	// Test with time range parameters
	now := time.Now().Unix()
	from := now - 7*24*3600 // 7 days ago

	result := client.GET(fmt.Sprintf("/v1/riverpool/pools/%s/nav/history?from=%d&to=%d", poolID, from, now))
	if result.Error != nil {
		t.Skipf("Failed to get NAV history: %v", result.Error)
	}

	t.Logf("NAV history: status=%d, latency=%v", result.StatusCode, result.Latency)
}

// TestRiverpool_GetDDGuardState tests DDGuard state
func TestRiverpool_GetDDGuardState(t *testing.T) {
	config := DefaultConfig()
	client := NewHTTPClient(config)

	poolID := getFirstPoolID(t, client)
	if poolID == "" {
		t.Skip("No pools available")
	}

	result := client.GET(fmt.Sprintf("/v1/riverpool/pools/%s/ddguard", poolID))
	if result.Error != nil {
		t.Skipf("Failed to get DDGuard state: %v", result.Error)
	}

	t.Logf("DDGuard state: status=%d, latency=%v", result.StatusCode, result.Latency)

	var response struct {
		Level       string `json:"level"`
		Drawdown    string `json:"drawdown"`
		HighWaterMark string `json:"high_water_mark"`
	}
	if err := json.Unmarshal(result.Response.Data, &response); err == nil {
		t.Logf("  Level: %s, Drawdown: %s", response.Level, response.Drawdown)
	}
}

// ===========================================
// User Query Tests
// ===========================================

// TestRiverpool_GetUserDeposits tests user deposits query
func TestRiverpool_GetUserDeposits(t *testing.T) {
	config := DefaultRiverpoolConfig()
	client := NewHTTPClient(config.TestConfig)

	result := client.GET(fmt.Sprintf("/v1/riverpool/user/%s/deposits", config.TestUser))
	if result.Error != nil {
		t.Skipf("API server not running: %v", result.Error)
	}

	t.Logf("User deposits: status=%d, latency=%v", result.StatusCode, result.Latency)
}

// TestRiverpool_GetUserWithdrawals tests user withdrawals query
func TestRiverpool_GetUserWithdrawals(t *testing.T) {
	config := DefaultRiverpoolConfig()
	client := NewHTTPClient(config.TestConfig)

	result := client.GET(fmt.Sprintf("/v1/riverpool/user/%s/withdrawals", config.TestUser))
	if result.Error != nil {
		t.Skipf("API server not running: %v", result.Error)
	}

	t.Logf("User withdrawals: status=%d, latency=%v", result.StatusCode, result.Latency)
}

// TestRiverpool_GetUserPoolBalance tests user balance in a specific pool
func TestRiverpool_GetUserPoolBalance(t *testing.T) {
	config := DefaultRiverpoolConfig()
	client := NewHTTPClient(config.TestConfig)

	poolID := getFirstPoolID(t, client)
	if poolID == "" {
		t.Skip("No pools available")
	}

	result := client.GET(fmt.Sprintf("/v1/riverpool/pools/%s/user/%s/balance", poolID, config.TestUser))
	if result.Error != nil {
		t.Skipf("API server not running: %v", result.Error)
	}

	t.Logf("User pool balance: status=%d, latency=%v", result.StatusCode, result.Latency)
}

// ===========================================
// Pool Deposits & Withdrawals Tests
// ===========================================

// TestRiverpool_GetPoolDeposits tests getting pool deposits
func TestRiverpool_GetPoolDeposits(t *testing.T) {
	config := DefaultConfig()
	client := NewHTTPClient(config)

	poolID := getFirstPoolID(t, client)
	if poolID == "" {
		t.Skip("No pools available")
	}

	result := client.GET(fmt.Sprintf("/v1/riverpool/pools/%s/deposits", poolID))
	if result.Error != nil {
		t.Skipf("Failed to get deposits: %v", result.Error)
	}

	t.Logf("Pool deposits: status=%d, latency=%v", result.StatusCode, result.Latency)
}

// TestRiverpool_GetPendingWithdrawals tests getting pending withdrawals
func TestRiverpool_GetPendingWithdrawals(t *testing.T) {
	config := DefaultConfig()
	client := NewHTTPClient(config)

	poolID := getFirstPoolID(t, client)
	if poolID == "" {
		t.Skip("No pools available")
	}

	result := client.GET(fmt.Sprintf("/v1/riverpool/pools/%s/withdrawals/pending", poolID))
	if result.Error != nil {
		t.Skipf("Failed to get pending withdrawals: %v", result.Error)
	}

	t.Logf("Pending withdrawals: status=%d, latency=%v", result.StatusCode, result.Latency)
}

// ===========================================
// Estimation Tests
// ===========================================

// TestRiverpool_EstimateDeposit tests deposit estimation
func TestRiverpool_EstimateDeposit(t *testing.T) {
	config := DefaultConfig()
	client := NewHTTPClient(config)

	poolID := getFirstPoolID(t, client)
	if poolID == "" {
		t.Skip("No pools available")
	}

	result := client.GET(fmt.Sprintf("/v1/riverpool/pools/%s/estimate/deposit?amount=1000", poolID))
	if result.Error != nil {
		t.Skipf("Failed to estimate deposit: %v", result.Error)
	}

	t.Logf("Deposit estimation: status=%d, latency=%v", result.StatusCode, result.Latency)

	var response struct {
		Shares string `json:"shares"`
		NAV    string `json:"nav"`
	}
	if err := json.Unmarshal(result.Response.Data, &response); err == nil {
		t.Logf("  Estimated shares: %s at NAV: %s", response.Shares, response.NAV)
	}
}

// TestRiverpool_EstimateWithdrawal tests withdrawal estimation
func TestRiverpool_EstimateWithdrawal(t *testing.T) {
	config := DefaultConfig()
	client := NewHTTPClient(config)

	poolID := getFirstPoolID(t, client)
	if poolID == "" {
		t.Skip("No pools available")
	}

	result := client.GET(fmt.Sprintf("/v1/riverpool/pools/%s/estimate/withdrawal?shares=100", poolID))
	if result.Error != nil {
		t.Skipf("Failed to estimate withdrawal: %v", result.Error)
	}

	t.Logf("Withdrawal estimation: status=%d, latency=%v", result.StatusCode, result.Latency)

	var response struct {
		Amount      string `json:"amount"`
		AvailableAt int64  `json:"available_at"`
		MayProrate  bool   `json:"may_be_prorated"`
	}
	if err := json.Unmarshal(result.Response.Data, &response); err == nil {
		t.Logf("  Estimated amount: %s, May prorate: %v", response.Amount, response.MayProrate)
	}
}

// ===========================================
// Transaction Tests
// ===========================================

// TestRiverpool_Deposit tests deposit transaction
func TestRiverpool_Deposit(t *testing.T) {
	config := DefaultRiverpoolConfig()
	client := NewHTTPClient(config.TestConfig)

	poolID := getFirstPoolID(t, client)
	if poolID == "" {
		t.Skip("No pools available")
	}

	depositReq := map[string]interface{}{
		"user":    config.TestUser,
		"pool_id": poolID,
		"amount":  config.DepositAmount,
	}

	result := client.POST("/v1/riverpool/deposit", depositReq)
	if result.Error != nil {
		t.Skipf("Failed to deposit: %v", result.Error)
	}

	// Accept various status codes (200, 201, 400 for validation, 429 for rate limit)
	validStatuses := []int{http.StatusOK, http.StatusCreated, http.StatusBadRequest, http.StatusTooManyRequests}
	statusValid := false
	for _, s := range validStatuses {
		if result.StatusCode == s {
			statusValid = true
			break
		}
	}

	if !statusValid {
		t.Errorf("Unexpected status %d", result.StatusCode)
	}

	t.Logf("Deposit: status=%d, latency=%v", result.StatusCode, result.Latency)
}

// TestRiverpool_RequestWithdrawal tests withdrawal request
func TestRiverpool_RequestWithdrawal(t *testing.T) {
	config := DefaultRiverpoolConfig()
	client := NewHTTPClient(config.TestConfig)

	poolID := getFirstPoolID(t, client)
	if poolID == "" {
		t.Skip("No pools available")
	}

	withdrawReq := map[string]interface{}{
		"user":    config.TestUser,
		"pool_id": poolID,
		"shares":  "100",
	}

	result := client.POST("/v1/riverpool/withdrawal/request", withdrawReq)
	if result.Error != nil {
		t.Skipf("Failed to request withdrawal: %v", result.Error)
	}

	t.Logf("Withdrawal request: status=%d, latency=%v", result.StatusCode, result.Latency)
}

// TestRiverpool_ClaimWithdrawal tests withdrawal claim
func TestRiverpool_ClaimWithdrawal(t *testing.T) {
	config := DefaultRiverpoolConfig()
	client := NewHTTPClient(config.TestConfig)

	claimReq := map[string]interface{}{
		"user":          config.TestUser,
		"withdrawal_id": "test-withdrawal-123",
	}

	result := client.POST("/v1/riverpool/withdrawal/claim", claimReq)
	if result.Error != nil {
		t.Skipf("Failed to claim withdrawal: %v", result.Error)
	}

	t.Logf("Withdrawal claim: status=%d, latency=%v", result.StatusCode, result.Latency)
}

// TestRiverpool_CancelWithdrawal tests withdrawal cancellation
func TestRiverpool_CancelWithdrawal(t *testing.T) {
	config := DefaultRiverpoolConfig()
	client := NewHTTPClient(config.TestConfig)

	cancelReq := map[string]interface{}{
		"user":          config.TestUser,
		"withdrawal_id": "test-withdrawal-123",
	}

	result := client.POST("/v1/riverpool/withdrawal/cancel", cancelReq)
	if result.Error != nil {
		t.Skipf("Failed to cancel withdrawal: %v", result.Error)
	}

	t.Logf("Withdrawal cancel: status=%d, latency=%v", result.StatusCode, result.Latency)
}

// ===========================================
// Revenue Tests
// ===========================================

// TestRiverpool_GetPoolRevenue tests pool revenue
func TestRiverpool_GetPoolRevenue(t *testing.T) {
	config := DefaultConfig()
	client := NewHTTPClient(config)

	poolID := getFirstPoolID(t, client)
	if poolID == "" {
		t.Skip("No pools available")
	}

	result := client.GET(fmt.Sprintf("/v1/riverpool/pools/%s/revenue", poolID))
	if result.Error != nil {
		t.Skipf("Failed to get revenue: %v", result.Error)
	}

	t.Logf("Pool revenue: status=%d, latency=%v", result.StatusCode, result.Latency)
}

// TestRiverpool_GetRevenueRecords tests revenue records
func TestRiverpool_GetRevenueRecords(t *testing.T) {
	config := DefaultConfig()
	client := NewHTTPClient(config)

	poolID := getFirstPoolID(t, client)
	if poolID == "" {
		t.Skip("No pools available")
	}

	result := client.GET(fmt.Sprintf("/v1/riverpool/pools/%s/revenue/records?limit=10", poolID))
	if result.Error != nil {
		t.Skipf("Failed to get revenue records: %v", result.Error)
	}

	t.Logf("Revenue records: status=%d, latency=%v", result.StatusCode, result.Latency)
}

// TestRiverpool_GetRevenueBreakdown tests revenue breakdown
func TestRiverpool_GetRevenueBreakdown(t *testing.T) {
	config := DefaultConfig()
	client := NewHTTPClient(config)

	poolID := getFirstPoolID(t, client)
	if poolID == "" {
		t.Skip("No pools available")
	}

	result := client.GET(fmt.Sprintf("/v1/riverpool/pools/%s/revenue/breakdown", poolID))
	if result.Error != nil {
		t.Skipf("Failed to get revenue breakdown: %v", result.Error)
	}

	t.Logf("Revenue breakdown: status=%d, latency=%v", result.StatusCode, result.Latency)
}

// ===========================================
// Community Pool Tests
// ===========================================

// TestRiverpool_CreateCommunityPool tests community pool creation
func TestRiverpool_CreateCommunityPool(t *testing.T) {
	config := DefaultRiverpoolConfig()
	client := NewHTTPClient(config.TestConfig)

	createReq := map[string]interface{}{
		"owner":           config.TestUser,
		"name":            fmt.Sprintf("Test Pool %d", time.Now().Unix()),
		"description":     "E2E test community pool",
		"min_deposit":     "100",
		"management_fee":  "0.02",
		"performance_fee": "0.20",
		"owner_stake":     "5000",
		"is_private":      false,
		"allowed_markets": []string{"BTC-USDC", "ETH-USDC"},
		"max_leverage":    "10",
	}

	result := client.POST("/v1/riverpool/community/create", createReq)
	if result.Error != nil {
		t.Skipf("Failed to create community pool: %v", result.Error)
	}

	t.Logf("Create community pool: status=%d, latency=%v", result.StatusCode, result.Latency)
}

// TestRiverpool_GetPoolHolders tests getting pool holders
func TestRiverpool_GetPoolHolders(t *testing.T) {
	config := DefaultConfig()
	client := NewHTTPClient(config)

	poolID := getCommunityPoolID(t, client)
	if poolID == "" {
		t.Skip("No community pools available")
	}

	result := client.GET(fmt.Sprintf("/v1/riverpool/community/%s/holders", poolID))
	if result.Error != nil {
		t.Skipf("Failed to get holders: %v", result.Error)
	}

	t.Logf("Pool holders: status=%d, latency=%v", result.StatusCode, result.Latency)
}

// TestRiverpool_GetPoolPositions tests getting pool positions
func TestRiverpool_GetPoolPositions(t *testing.T) {
	config := DefaultConfig()
	client := NewHTTPClient(config)

	poolID := getCommunityPoolID(t, client)
	if poolID == "" {
		t.Skip("No community pools available")
	}

	result := client.GET(fmt.Sprintf("/v1/riverpool/community/%s/positions", poolID))
	if result.Error != nil {
		t.Skipf("Failed to get positions: %v", result.Error)
	}

	t.Logf("Pool positions: status=%d, latency=%v", result.StatusCode, result.Latency)
}

// TestRiverpool_GetPoolTrades tests getting pool trades
func TestRiverpool_GetPoolTrades(t *testing.T) {
	config := DefaultConfig()
	client := NewHTTPClient(config)

	poolID := getCommunityPoolID(t, client)
	if poolID == "" {
		t.Skip("No community pools available")
	}

	result := client.GET(fmt.Sprintf("/v1/riverpool/community/%s/trades?limit=50", poolID))
	if result.Error != nil {
		t.Skipf("Failed to get trades: %v", result.Error)
	}

	t.Logf("Pool trades: status=%d, latency=%v", result.StatusCode, result.Latency)
}

// TestRiverpool_GetInviteCodes tests getting invite codes
func TestRiverpool_GetInviteCodes(t *testing.T) {
	config := DefaultRiverpoolConfig()
	client := NewHTTPClient(config.TestConfig)

	poolID := getCommunityPoolID(t, client)
	if poolID == "" {
		t.Skip("No community pools available")
	}

	result := client.GET(fmt.Sprintf("/v1/riverpool/community/%s/invites", poolID))
	if result.Error != nil {
		t.Skipf("Failed to get invite codes: %v", result.Error)
	}

	t.Logf("Invite codes: status=%d, latency=%v", result.StatusCode, result.Latency)
}

// TestRiverpool_GenerateInviteCode tests generating invite code
func TestRiverpool_GenerateInviteCode(t *testing.T) {
	config := DefaultRiverpoolConfig()
	client := NewHTTPClient(config.TestConfig)

	poolID := getCommunityPoolID(t, client)
	if poolID == "" {
		t.Skip("No community pools available")
	}

	genReq := map[string]interface{}{
		"owner": config.TestUser,
		"count": 5,
	}

	result := client.POST(fmt.Sprintf("/v1/riverpool/community/%s/invites", poolID), genReq)
	if result.Error != nil {
		t.Skipf("Failed to generate invite code: %v", result.Error)
	}

	t.Logf("Generate invite code: status=%d, latency=%v", result.StatusCode, result.Latency)
}

// TestRiverpool_GetUserOwnedPools tests getting user's owned pools
func TestRiverpool_GetUserOwnedPools(t *testing.T) {
	config := DefaultRiverpoolConfig()
	client := NewHTTPClient(config.TestConfig)

	result := client.GET(fmt.Sprintf("/v1/riverpool/user/%s/owned-pools", config.TestUser))
	if result.Error != nil {
		t.Skipf("API server not running: %v", result.Error)
	}

	t.Logf("User owned pools: status=%d, latency=%v", result.StatusCode, result.Latency)
}

// ===========================================
// Full Flow Integration Tests
// ===========================================

// TestRiverpool_FullDepositWithdrawFlow tests complete deposit-withdraw flow
func TestRiverpool_FullDepositWithdrawFlow(t *testing.T) {
	config := DefaultRiverpoolConfig()
	config.TestUser = fmt.Sprintf("flow_test_%d", time.Now().UnixNano())
	client := NewHTTPClient(config.TestConfig)

	// 1. Get available pool
	poolID := getFirstPoolID(t, client)
	if poolID == "" {
		t.Skip("No pools available")
	}
	t.Logf("Step 1: Using pool %s", poolID)

	// 2. Estimate deposit
	estResult := client.GET(fmt.Sprintf("/v1/riverpool/pools/%s/estimate/deposit?amount=1000", poolID))
	if estResult.Error != nil {
		t.Skipf("Failed to estimate: %v", estResult.Error)
	}
	t.Logf("Step 2: Deposit estimation completed, latency=%v", estResult.Latency)

	// 3. Make deposit
	depositReq := map[string]interface{}{
		"user":    config.TestUser,
		"pool_id": poolID,
		"amount":  "1000",
	}
	depResult := client.POST("/v1/riverpool/deposit", depositReq)
	t.Logf("Step 3: Deposit result: status=%d, latency=%v", depResult.StatusCode, depResult.Latency)

	// 4. Check balance
	balResult := client.GET(fmt.Sprintf("/v1/riverpool/pools/%s/user/%s/balance", poolID, config.TestUser))
	t.Logf("Step 4: Balance check: status=%d, latency=%v", balResult.StatusCode, balResult.Latency)

	// 5. Request withdrawal
	withdrawReq := map[string]interface{}{
		"user":    config.TestUser,
		"pool_id": poolID,
		"shares":  "50",
	}
	wdResult := client.POST("/v1/riverpool/withdrawal/request", withdrawReq)
	t.Logf("Step 5: Withdrawal request: status=%d, latency=%v", wdResult.StatusCode, wdResult.Latency)

	// 6. Check pending withdrawals
	pendResult := client.GET(fmt.Sprintf("/v1/riverpool/pools/%s/withdrawals/pending", poolID))
	t.Logf("Step 6: Pending withdrawals: status=%d, latency=%v", pendResult.StatusCode, pendResult.Latency)

	// Generate report
	report := client.GenerateReport("RiverPool Full Flow")
	report.PrintReport()
}

// TestRiverpool_CommunityPoolFullFlow tests complete community pool flow
func TestRiverpool_CommunityPoolFullFlow(t *testing.T) {
	config := DefaultRiverpoolConfig()
	config.TestUser = fmt.Sprintf("community_owner_%d", time.Now().UnixNano())
	client := NewHTTPClient(config.TestConfig)

	// 1. Create community pool
	createReq := map[string]interface{}{
		"owner":           config.TestUser,
		"name":            fmt.Sprintf("E2E Test Pool %d", time.Now().Unix()),
		"description":     "Community pool for E2E testing",
		"min_deposit":     "100",
		"management_fee":  "0.02",
		"performance_fee": "0.20",
		"owner_stake":     "5000",
		"is_private":      true,
		"allowed_markets": []string{"BTC-USDC"},
		"max_leverage":    "10",
	}
	createResult := client.POST("/v1/riverpool/community/create", createReq)
	t.Logf("Step 1: Create pool: status=%d, latency=%v", createResult.StatusCode, createResult.Latency)

	// 2. Get owned pools
	ownedResult := client.GET(fmt.Sprintf("/v1/riverpool/user/%s/owned-pools", config.TestUser))
	t.Logf("Step 2: Get owned pools: status=%d, latency=%v", ownedResult.StatusCode, ownedResult.Latency)

	// Generate report
	report := client.GenerateReport("Community Pool Full Flow")
	report.PrintReport()
}

// ===========================================
// Latency Benchmark Tests
// ===========================================

// TestRiverpool_LatencyBenchmark benchmarks RiverPool API latency
func TestRiverpool_LatencyBenchmark(t *testing.T) {
	config := DefaultConfig()
	client := NewHTTPClient(config)

	// Check server first
	result := client.GET("/v1/riverpool/pools")
	if result.Error != nil {
		t.Skipf("API server not running: %v", result.Error)
	}

	poolID := getFirstPoolID(t, client)
	if poolID == "" {
		t.Skip("No pools available")
	}

	iterations := 50
	t.Logf("Running %d iterations across RiverPool endpoints...", iterations)

	for i := 0; i < iterations; i++ {
		switch i % 5 {
		case 0:
			client.GET("/v1/riverpool/pools")
		case 1:
			client.GET(fmt.Sprintf("/v1/riverpool/pools/%s", poolID))
		case 2:
			client.GET(fmt.Sprintf("/v1/riverpool/pools/%s/stats", poolID))
		case 3:
			client.GET(fmt.Sprintf("/v1/riverpool/pools/%s/nav/history", poolID))
		case 4:
			client.GET(fmt.Sprintf("/v1/riverpool/pools/%s/estimate/deposit?amount=1000", poolID))
		}
	}

	report := client.GenerateReport("RiverPool Latency Benchmark")
	report.PrintReport()

	if report.AvgLatency > 100*time.Millisecond {
		t.Logf("Warning: Average latency %v exceeds 100ms target", report.AvgLatency)
	}
}

// ===========================================
// Helper Functions
// ===========================================

// getFirstPoolID returns the first available pool ID
func getFirstPoolID(t *testing.T, client *HTTPClient) string {
	result := client.GET("/v1/riverpool/pools")
	if result.Error != nil {
		return ""
	}

	var response struct {
		Pools []map[string]interface{} `json:"pools"`
	}
	if err := json.Unmarshal(result.Response.Data, &response); err != nil || len(response.Pools) == 0 {
		return ""
	}

	if id, ok := response.Pools[0]["pool_id"].(string); ok {
		return id
	}
	return ""
}

// getCommunityPoolID returns a community pool ID if available
func getCommunityPoolID(t *testing.T, client *HTTPClient) string {
	result := client.GET("/v1/riverpool/pools/type/community")
	if result.Error != nil {
		return ""
	}

	var response struct {
		Pools []map[string]interface{} `json:"pools"`
	}
	if err := json.Unmarshal(result.Response.Data, &response); err != nil || len(response.Pools) == 0 {
		// Fallback to first pool
		return getFirstPoolID(t, client)
	}

	if id, ok := response.Pools[0]["pool_id"].(string); ok {
		return id
	}
	return getFirstPoolID(t, client)
}
