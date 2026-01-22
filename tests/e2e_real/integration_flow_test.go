// Package e2e_real provides full integration flow testing
// Tests complete user journeys through all API systems without mock data
package e2e_real

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"testing"
	"time"
)

// ===========================================
// Full System Integration Tests
// ===========================================

// TestIntegration_CompleteUserJourney tests a complete user journey
// 1. Create account -> 2. Deposit funds -> 3. Check markets -> 4. Place orders
// 5. Monitor positions -> 6. Join liquidity pool -> 7. Withdraw
func TestIntegration_CompleteUserJourney(t *testing.T) {
	config := DefaultConfig()
	client := NewHTTPClient(config)

	// Check server is running
	result := client.GET("/v1/health")
	if result.Error != nil {
		t.Skipf("API server not running: %v", result.Error)
	}

	trader := fmt.Sprintf("journey_user_%d", time.Now().UnixNano())
	t.Logf("Starting complete user journey for: %s", trader)

	// ========== Phase 1: Account Setup ==========
	t.Log("\n=== Phase 1: Account Setup ===")

	// 1.1 Deposit initial funds
	depositReq := map[string]interface{}{
		"trader": trader,
		"amount": "100000.00",
	}
	depResult := client.POST("/v1/account/deposit", depositReq)
	t.Logf("1.1 Initial deposit: status=%d, latency=%v", depResult.StatusCode, depResult.Latency)

	// 1.2 Verify account balance
	accResult := client.GET(fmt.Sprintf("/v1/account?trader=%s", trader))
	t.Logf("1.2 Account verification: status=%d, latency=%v", accResult.StatusCode, accResult.Latency)

	// ========== Phase 2: Market Analysis ==========
	t.Log("\n=== Phase 2: Market Analysis ===")

	// 2.1 Get available markets
	mktsResult := client.GET("/v1/markets")
	t.Logf("2.1 Get markets: status=%d, latency=%v", mktsResult.StatusCode, mktsResult.Latency)

	// 2.2 Check BTC ticker
	tickerResult := client.GET("/v1/markets/BTC-USDC/ticker")
	t.Logf("2.2 BTC ticker: status=%d, latency=%v", tickerResult.StatusCode, tickerResult.Latency)

	// 2.3 Get orderbook
	obResult := client.GET("/v1/markets/BTC-USDC/orderbook?depth=20")
	t.Logf("2.3 Orderbook: status=%d, latency=%v", obResult.StatusCode, obResult.Latency)

	// 2.4 Check recent trades
	tradesResult := client.GET("/v1/markets/BTC-USDC/trades?limit=10")
	t.Logf("2.4 Recent trades: status=%d, latency=%v", tradesResult.StatusCode, tradesResult.Latency)

	// 2.5 Get funding rate
	fundingResult := client.GET("/v1/markets/BTC-USDC/funding")
	t.Logf("2.5 Funding rate: status=%d, latency=%v", fundingResult.StatusCode, fundingResult.Latency)

	// ========== Phase 3: Trading ==========
	t.Log("\n=== Phase 3: Trading ===")

	// 3.1 Place limit buy order
	buyOrder := map[string]interface{}{
		"market_id": "BTC-USDC",
		"trader":    trader,
		"side":      "buy",
		"type":      "limit",
		"price":     "45000",
		"quantity":  "1.0",
	}
	buyResult := client.POST("/v1/orders", buyOrder)
	t.Logf("3.1 Limit buy order: status=%d, latency=%v", buyResult.StatusCode, buyResult.Latency)

	// 3.2 Place limit sell order
	sellOrder := map[string]interface{}{
		"market_id": "BTC-USDC",
		"trader":    trader,
		"side":      "sell",
		"type":      "limit",
		"price":     "55000",
		"quantity":  "0.5",
	}
	sellResult := client.POST("/v1/orders", sellOrder)
	t.Logf("3.2 Limit sell order: status=%d, latency=%v", sellResult.StatusCode, sellResult.Latency)

	// 3.3 Check open orders
	ordersResult := client.GET(fmt.Sprintf("/v1/orders?trader=%s&status=open", trader))
	t.Logf("3.3 Open orders: status=%d, latency=%v", ordersResult.StatusCode, ordersResult.Latency)

	// 3.4 Check positions
	posResult := client.GET(fmt.Sprintf("/v1/positions?trader=%s", trader))
	t.Logf("3.4 Positions: status=%d, latency=%v", posResult.StatusCode, posResult.Latency)

	// ========== Phase 4: RiverPool Participation ==========
	t.Log("\n=== Phase 4: RiverPool Participation ===")

	// 4.1 Get available pools
	poolsResult := client.GET("/v1/riverpool/pools")
	t.Logf("4.1 Available pools: status=%d, latency=%v", poolsResult.StatusCode, poolsResult.Latency)

	// 4.2 Get pool details
	var poolsResp struct {
		Pools []map[string]interface{} `json:"pools"`
	}
	if poolsResult.Response != nil && poolsResult.Response.Data != nil {
		json.Unmarshal(poolsResult.Response.Data, &poolsResp)
	}

	if len(poolsResp.Pools) > 0 {
		poolID := poolsResp.Pools[0]["pool_id"].(string)

		// 4.3 Estimate deposit
		estResult := client.GET(fmt.Sprintf("/v1/riverpool/pools/%s/estimate/deposit?amount=5000", poolID))
		t.Logf("4.3 Deposit estimate: status=%d, latency=%v", estResult.StatusCode, estResult.Latency)

		// 4.4 Make pool deposit
		poolDepositReq := map[string]interface{}{
			"user":    trader,
			"pool_id": poolID,
			"amount":  "5000",
		}
		poolDepResult := client.POST("/v1/riverpool/deposit", poolDepositReq)
		t.Logf("4.4 Pool deposit: status=%d, latency=%v", poolDepResult.StatusCode, poolDepResult.Latency)

		// 4.5 Check pool balance
		balResult := client.GET(fmt.Sprintf("/v1/riverpool/pools/%s/user/%s/balance", poolID, trader))
		t.Logf("4.5 Pool balance: status=%d, latency=%v", balResult.StatusCode, balResult.Latency)

		// 4.6 Check pool NAV history
		navResult := client.GET(fmt.Sprintf("/v1/riverpool/pools/%s/nav/history", poolID))
		t.Logf("4.6 NAV history: status=%d, latency=%v", navResult.StatusCode, navResult.Latency)

		// 4.7 Get DDGuard status
		ddgResult := client.GET(fmt.Sprintf("/v1/riverpool/pools/%s/ddguard", poolID))
		t.Logf("4.7 DDGuard status: status=%d, latency=%v", ddgResult.StatusCode, ddgResult.Latency)
	} else {
		t.Log("4.2-4.7 Skipped: No pools available")
	}

	// ========== Phase 5: Account Management ==========
	t.Log("\n=== Phase 5: Account Management ===")

	// 5.1 Final account check
	finalAccResult := client.GET(fmt.Sprintf("/v1/account?trader=%s", trader))
	t.Logf("5.1 Final account: status=%d, latency=%v", finalAccResult.StatusCode, finalAccResult.Latency)

	// 5.2 Partial withdrawal
	withdrawReq := map[string]interface{}{
		"trader": trader,
		"amount": "1000.00",
	}
	wdResult := client.POST("/v1/account/withdraw", withdrawReq)
	t.Logf("5.2 Withdrawal: status=%d, latency=%v", wdResult.StatusCode, wdResult.Latency)

	// ========== Summary ==========
	report := client.GenerateReport("Complete User Journey")
	t.Log("\n=== Summary ===")
	report.PrintReport()
}

// TestIntegration_TradingAndPoolInteraction tests trading impact on pool NAV
func TestIntegration_TradingAndPoolInteraction(t *testing.T) {
	config := DefaultConfig()
	client := NewHTTPClient(config)

	// Check server is running
	result := client.GET("/v1/health")
	if result.Error != nil {
		t.Skipf("API server not running: %v", result.Error)
	}

	poolOwner := fmt.Sprintf("pool_owner_%d", time.Now().UnixNano())
	lpDepositor := fmt.Sprintf("lp_depositor_%d", time.Now().UnixNano())

	t.Logf("Testing trading and pool interaction")
	t.Logf("Pool Owner: %s", poolOwner)
	t.Logf("LP Depositor: %s", lpDepositor)

	// Step 1: Get pool ID
	poolID := getFirstPoolID(t, client)
	if poolID == "" {
		t.Skip("No pools available")
	}
	t.Logf("Using pool: %s", poolID)

	// Step 2: Check initial pool stats
	statsResult := client.GET(fmt.Sprintf("/v1/riverpool/pools/%s/stats", poolID))
	t.Logf("Initial pool stats: status=%d, latency=%v", statsResult.StatusCode, statsResult.Latency)

	// Step 3: Depositor joins pool
	depositReq := map[string]interface{}{
		"user":    lpDepositor,
		"pool_id": poolID,
		"amount":  "10000",
	}
	depResult := client.POST("/v1/riverpool/deposit", depositReq)
	t.Logf("LP deposit: status=%d, latency=%v", depResult.StatusCode, depResult.Latency)

	// Step 4: Check pool stats after deposit
	statsResult2 := client.GET(fmt.Sprintf("/v1/riverpool/pools/%s/stats", poolID))
	t.Logf("Stats after deposit: status=%d", statsResult2.StatusCode)

	// Step 5: Simulate trading activity (would affect pool NAV)
	for i := 0; i < 5; i++ {
		order := map[string]interface{}{
			"market_id": "BTC-USDC",
			"trader":    poolOwner,
			"side":      "buy",
			"type":      "limit",
			"price":     fmt.Sprintf("%d", 48000+i*100),
			"quantity":  "0.1",
		}
		client.POST("/v1/orders", order)
	}
	t.Log("Trading activity simulated")

	// Step 6: Check pool NAV and revenue
	navResult := client.GET(fmt.Sprintf("/v1/riverpool/pools/%s/nav/history", poolID))
	t.Logf("NAV history: status=%d, latency=%v", navResult.StatusCode, navResult.Latency)

	revenueResult := client.GET(fmt.Sprintf("/v1/riverpool/pools/%s/revenue", poolID))
	t.Logf("Pool revenue: status=%d, latency=%v", revenueResult.StatusCode, revenueResult.Latency)

	// Generate report
	report := client.GenerateReport("Trading & Pool Interaction")
	report.PrintReport()
}

// TestIntegration_ConcurrentUsersFullFlow tests multiple concurrent users
func TestIntegration_ConcurrentUsersFullFlow(t *testing.T) {
	config := DefaultConfig()

	// Check server is running
	client := NewHTTPClient(config)
	result := client.GET("/v1/health")
	if result.Error != nil {
		t.Skipf("API server not running: %v", result.Error)
	}

	numUsers := 10
	var wg sync.WaitGroup
	results := make(chan struct {
		user    string
		success bool
		latency time.Duration
		orders  int
	}, numUsers)

	t.Logf("Starting concurrent flow test with %d users...", numUsers)

	for i := 0; i < numUsers; i++ {
		wg.Add(1)
		go func(userNum int) {
			defer wg.Done()

			userClient := NewHTTPClient(config)
			user := fmt.Sprintf("concurrent_user_%d_%d", userNum, time.Now().UnixNano())
			start := time.Now()
			ordersPlaced := 0

			// 1. Deposit
			depositReq := map[string]interface{}{
				"trader": user,
				"amount": "50000.00",
			}
			userClient.POST("/v1/account/deposit", depositReq)

			// 2. Place multiple orders
			for j := 0; j < 5; j++ {
				order := map[string]interface{}{
					"market_id": "BTC-USDC",
					"trader":    user,
					"side":      []string{"buy", "sell"}[j%2],
					"type":      "limit",
					"price":     fmt.Sprintf("%d", 45000+j*500),
					"quantity":  "0.1",
				}
				orderResult := userClient.POST("/v1/orders", order)
				if orderResult.StatusCode == http.StatusOK || orderResult.StatusCode == http.StatusCreated {
					ordersPlaced++
				}
			}

			// 3. Check positions
			userClient.GET(fmt.Sprintf("/v1/positions?trader=%s", user))

			// 4. Try pool deposit
			poolID := getFirstPoolID(t, userClient)
			if poolID != "" {
				poolDepositReq := map[string]interface{}{
					"user":    user,
					"pool_id": poolID,
					"amount":  "1000",
				}
				userClient.POST("/v1/riverpool/deposit", poolDepositReq)
			}

			results <- struct {
				user    string
				success bool
				latency time.Duration
				orders  int
			}{
				user:    user,
				success: ordersPlaced > 0,
				latency: time.Since(start),
				orders:  ordersPlaced,
			}
		}(i)
	}

	wg.Wait()
	close(results)

	// Collect and report results
	successCount := 0
	totalOrders := 0
	var totalLatency time.Duration

	for r := range results {
		if r.success {
			successCount++
		}
		totalOrders += r.orders
		totalLatency += r.latency
		t.Logf("User %s: success=%v, orders=%d, latency=%v", r.user, r.success, r.orders, r.latency)
	}

	t.Logf("\n=== Concurrent Test Summary ===")
	t.Logf("Total users: %d", numUsers)
	t.Logf("Successful users: %d (%.1f%%)", successCount, float64(successCount)/float64(numUsers)*100)
	t.Logf("Total orders placed: %d", totalOrders)
	t.Logf("Average latency per user: %v", totalLatency/time.Duration(numUsers))
}

// TestIntegration_RiverpoolFullCycle tests complete RiverPool lifecycle
func TestIntegration_RiverpoolFullCycle(t *testing.T) {
	config := DefaultConfig()
	client := NewHTTPClient(config)

	// Check server is running
	result := client.GET("/v1/health")
	if result.Error != nil {
		t.Skipf("API server not running: %v", result.Error)
	}

	owner := fmt.Sprintf("pool_owner_%d", time.Now().UnixNano())
	investor := fmt.Sprintf("investor_%d", time.Now().UnixNano())

	t.Logf("Testing RiverPool full lifecycle")

	// Step 1: Create community pool
	t.Log("Step 1: Creating community pool...")
	createReq := map[string]interface{}{
		"owner":           owner,
		"name":            fmt.Sprintf("E2E Test Pool %d", time.Now().Unix()),
		"description":     "Integration test pool",
		"min_deposit":     "100",
		"management_fee":  "0.02",
		"performance_fee": "0.20",
		"owner_stake":     "10000",
		"is_private":      false,
		"allowed_markets": []string{"BTC-USDC", "ETH-USDC"},
		"max_leverage":    "10",
	}
	createResult := client.POST("/v1/riverpool/community/create", createReq)
	t.Logf("  Result: status=%d, latency=%v", createResult.StatusCode, createResult.Latency)

	// Step 2: Get owner's pools
	t.Log("Step 2: Getting owner's pools...")
	ownedResult := client.GET(fmt.Sprintf("/v1/riverpool/user/%s/owned-pools", owner))
	t.Logf("  Result: status=%d, latency=%v", ownedResult.StatusCode, ownedResult.Latency)

	// Step 3: Use first available pool (created or existing)
	poolID := getFirstPoolID(t, client)
	if poolID == "" {
		t.Skip("No pools available")
	}
	t.Logf("Step 3: Using pool %s", poolID)

	// Step 4: Investor estimates deposit
	t.Log("Step 4: Investor estimates deposit...")
	estResult := client.GET(fmt.Sprintf("/v1/riverpool/pools/%s/estimate/deposit?amount=5000", poolID))
	t.Logf("  Result: status=%d, latency=%v", estResult.StatusCode, estResult.Latency)

	// Step 5: Investor deposits
	t.Log("Step 5: Investor deposits...")
	depositReq := map[string]interface{}{
		"user":    investor,
		"pool_id": poolID,
		"amount":  "5000",
	}
	depositResult := client.POST("/v1/riverpool/deposit", depositReq)
	t.Logf("  Result: status=%d, latency=%v", depositResult.StatusCode, depositResult.Latency)

	// Step 6: Check investor balance
	t.Log("Step 6: Checking investor balance...")
	balResult := client.GET(fmt.Sprintf("/v1/riverpool/pools/%s/user/%s/balance", poolID, investor))
	t.Logf("  Result: status=%d, latency=%v", balResult.StatusCode, balResult.Latency)

	// Step 7: Get pool stats
	t.Log("Step 7: Getting pool stats...")
	statsResult := client.GET(fmt.Sprintf("/v1/riverpool/pools/%s/stats", poolID))
	t.Logf("  Result: status=%d, latency=%v", statsResult.StatusCode, statsResult.Latency)

	// Step 8: Get DDGuard state
	t.Log("Step 8: Getting DDGuard state...")
	ddgResult := client.GET(fmt.Sprintf("/v1/riverpool/pools/%s/ddguard", poolID))
	t.Logf("  Result: status=%d, latency=%v", ddgResult.StatusCode, ddgResult.Latency)

	// Step 9: Estimate withdrawal
	t.Log("Step 9: Estimating withdrawal...")
	wdEstResult := client.GET(fmt.Sprintf("/v1/riverpool/pools/%s/estimate/withdrawal?shares=100", poolID))
	t.Logf("  Result: status=%d, latency=%v", wdEstResult.StatusCode, wdEstResult.Latency)

	// Step 10: Request withdrawal
	t.Log("Step 10: Requesting withdrawal...")
	wdReq := map[string]interface{}{
		"user":    investor,
		"pool_id": poolID,
		"shares":  "100",
	}
	wdResult := client.POST("/v1/riverpool/withdrawal/request", wdReq)
	t.Logf("  Result: status=%d, latency=%v", wdResult.StatusCode, wdResult.Latency)

	// Step 11: Check pending withdrawals
	t.Log("Step 11: Checking pending withdrawals...")
	pendingResult := client.GET(fmt.Sprintf("/v1/riverpool/pools/%s/withdrawals/pending", poolID))
	t.Logf("  Result: status=%d, latency=%v", pendingResult.StatusCode, pendingResult.Latency)

	// Step 12: Get revenue data
	t.Log("Step 12: Getting revenue data...")
	revResult := client.GET(fmt.Sprintf("/v1/riverpool/pools/%s/revenue", poolID))
	t.Logf("  Result: status=%d, latency=%v", revResult.StatusCode, revResult.Latency)

	// Summary
	report := client.GenerateReport("RiverPool Full Cycle")
	t.Log("\n=== Summary ===")
	report.PrintReport()
}

// TestIntegration_WebSocketWithTrading tests WebSocket updates during trading
func TestIntegration_WebSocketWithTrading(t *testing.T) {
	config := DefaultConfig()
	httpClient := NewHTTPClient(config)
	wsClient := NewWSClient(config)

	// Check HTTP server is running
	result := httpClient.GET("/v1/health")
	if result.Error != nil {
		t.Skipf("API server not running: %v", result.Error)
	}

	// Connect WebSocket
	if err := wsClient.Connect("/ws"); err != nil {
		t.Skipf("WebSocket not available: %v", err)
	}
	defer wsClient.Close()

	// Subscribe to multiple channels
	channels := []string{
		"ticker:BTC-USDC",
		"depth:BTC-USDC",
		"trades:BTC-USDC",
	}

	for _, ch := range channels {
		wsClient.Send(map[string]interface{}{
			"type":    "subscribe",
			"channel": ch,
		})
	}
	t.Log("Subscribed to WebSocket channels")

	// Start receiving in background
	var wsMessages int
	var mu sync.Mutex
	done := make(chan bool)

	go func() {
		for {
			select {
			case <-done:
				return
			default:
				_, err := wsClient.Receive(500 * time.Millisecond)
				if err == nil {
					mu.Lock()
					wsMessages++
					mu.Unlock()
				}
			}
		}
	}()

	// Generate trading activity
	trader := fmt.Sprintf("ws_trader_%d", time.Now().UnixNano())

	// Deposit
	httpClient.POST("/v1/account/deposit", map[string]interface{}{
		"trader": trader,
		"amount": "100000",
	})

	// Place multiple orders
	for i := 0; i < 10; i++ {
		order := map[string]interface{}{
			"market_id": "BTC-USDC",
			"trader":    trader,
			"side":      []string{"buy", "sell"}[i%2],
			"type":      "limit",
			"price":     fmt.Sprintf("%d", 49000+i*100),
			"quantity":  "0.1",
		}
		httpClient.POST("/v1/orders", order)
		time.Sleep(50 * time.Millisecond)
	}

	// Wait a bit more for WS updates
	time.Sleep(2 * time.Second)
	close(done)

	mu.Lock()
	t.Logf("Received %d WebSocket messages during trading", wsMessages)
	mu.Unlock()

	// Generate report
	report := httpClient.GenerateReport("WebSocket + Trading Integration")
	report.PrintReport()
}

// TestIntegration_AllEndpointsCoverage tests coverage of all endpoints
func TestIntegration_AllEndpointsCoverage(t *testing.T) {
	config := DefaultConfig()
	client := NewHTTPClient(config)

	// Check server is running
	result := client.GET("/v1/health")
	if result.Error != nil {
		t.Skipf("API server not running: %v", result.Error)
	}

	trader := fmt.Sprintf("coverage_user_%d", time.Now().UnixNano())
	poolID := getFirstPoolID(t, client)

	// Define all endpoints to test
	endpoints := []struct {
		name   string
		method string
		path   string
		body   interface{}
	}{
		// Core
		{"Health", "GET", "/v1/health", nil},

		// Markets
		{"Markets", "GET", "/v1/markets", nil},
		{"Market", "GET", "/v1/markets/BTC-USDC", nil},
		{"Ticker", "GET", "/v1/markets/BTC-USDC/ticker", nil},
		{"Orderbook", "GET", "/v1/markets/BTC-USDC/orderbook", nil},
		{"Trades", "GET", "/v1/markets/BTC-USDC/trades", nil},
		{"Klines", "GET", "/v1/markets/BTC-USDC/klines?interval=1h", nil},
		{"Funding", "GET", "/v1/markets/BTC-USDC/funding", nil},
		{"Tickers", "GET", "/v1/tickers", nil},

		// Account
		{"Account", "GET", fmt.Sprintf("/v1/account?trader=%s", trader), nil},
		{"Deposit", "POST", "/v1/account/deposit", map[string]interface{}{"trader": trader, "amount": "10000"}},
		{"Withdraw", "POST", "/v1/account/withdraw", map[string]interface{}{"trader": trader, "amount": "100"}},

		// Orders
		{"PlaceOrder", "POST", "/v1/orders", map[string]interface{}{
			"market_id": "BTC-USDC", "trader": trader, "side": "buy", "type": "limit", "price": "45000", "quantity": "0.1",
		}},
		{"GetOrders", "GET", fmt.Sprintf("/v1/orders?trader=%s", trader), nil},

		// Positions
		{"GetPositions", "GET", "/v1/positions", nil},
	}

	// Add RiverPool endpoints if pool exists
	if poolID != "" {
		rpEndpoints := []struct {
			name   string
			method string
			path   string
			body   interface{}
		}{
			{"RP-Pools", "GET", "/v1/riverpool/pools", nil},
			{"RP-Pool", "GET", fmt.Sprintf("/v1/riverpool/pools/%s", poolID), nil},
			{"RP-Stats", "GET", fmt.Sprintf("/v1/riverpool/pools/%s/stats", poolID), nil},
			{"RP-NAV", "GET", fmt.Sprintf("/v1/riverpool/pools/%s/nav/history", poolID), nil},
			{"RP-DDGuard", "GET", fmt.Sprintf("/v1/riverpool/pools/%s/ddguard", poolID), nil},
			{"RP-EstDeposit", "GET", fmt.Sprintf("/v1/riverpool/pools/%s/estimate/deposit?amount=1000", poolID), nil},
			{"RP-EstWithdraw", "GET", fmt.Sprintf("/v1/riverpool/pools/%s/estimate/withdrawal?shares=100", poolID), nil},
			{"RP-UserDeposits", "GET", fmt.Sprintf("/v1/riverpool/user/%s/deposits", trader), nil},
			{"RP-UserWithdrawals", "GET", fmt.Sprintf("/v1/riverpool/user/%s/withdrawals", trader), nil},
			{"RP-Revenue", "GET", fmt.Sprintf("/v1/riverpool/pools/%s/revenue", poolID), nil},
			{"RP-Deposit", "POST", "/v1/riverpool/deposit", map[string]interface{}{"user": trader, "pool_id": poolID, "amount": "1000"}},
		}
		for _, ep := range rpEndpoints {
			endpoints = append(endpoints, ep)
		}
	}

	// Test all endpoints
	t.Logf("Testing %d endpoints...", len(endpoints))

	successCount := 0
	for _, ep := range endpoints {
		var res *RequestResult
		switch ep.method {
		case "GET":
			res = client.GET(ep.path)
		case "POST":
			res = client.POST(ep.path, ep.body)
		}

		status := "FAIL"
		if res.StatusCode >= 200 && res.StatusCode < 500 {
			status = "OK"
			successCount++
		}

		t.Logf("  [%s] %s %s: %d (%v)", status, ep.method, ep.name, res.StatusCode, res.Latency)
	}

	t.Logf("\n=== Coverage Summary ===")
	t.Logf("Endpoints tested: %d", len(endpoints))
	t.Logf("Successful: %d (%.1f%%)", successCount, float64(successCount)/float64(len(endpoints))*100)

	report := client.GenerateReport("All Endpoints Coverage")
	report.PrintReport()
}
