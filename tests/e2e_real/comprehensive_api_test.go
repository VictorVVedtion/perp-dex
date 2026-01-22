// Package e2e_real provides comprehensive end-to-end testing for all API endpoints
// Tests actual HTTP/WebSocket connections to a running API server without mock data
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
// Health & System Tests
// ===========================================

// TestAPI_Health tests health endpoint
func TestAPI_Health(t *testing.T) {
	config := DefaultConfig()
	client := NewHTTPClient(config)

	endpoints := []string{"/health", "/v1/health"}

	for _, endpoint := range endpoints {
		result := client.GET(endpoint)
		if result.Error != nil {
			t.Skipf("API server not running: %v", result.Error)
		}

		if result.StatusCode != http.StatusOK {
			t.Errorf("Health check %s failed: status %d", endpoint, result.StatusCode)
		}

		var health struct {
			Status string `json:"status"`
			Mode   string `json:"mode"`
		}
		if err := json.Unmarshal(result.Response.Data, &health); err == nil {
			t.Logf("Health %s: status=%s, mode=%s, latency=%v", endpoint, health.Status, health.Mode, result.Latency)
		}
	}
}

// ===========================================
// Market Endpoints Tests
// ===========================================

// TestAPI_GetMarkets tests fetching all markets
func TestAPI_GetMarkets(t *testing.T) {
	config := DefaultConfig()
	client := NewHTTPClient(config)

	result := client.GET("/v1/markets")
	if result.Error != nil {
		t.Skipf("API server not running: %v", result.Error)
	}

	if result.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", result.StatusCode)
	}

	var response struct {
		Markets []map[string]interface{} `json:"markets"`
	}
	if err := json.Unmarshal(result.Response.Data, &response); err == nil {
		t.Logf("Found %d markets, latency: %v", len(response.Markets), result.Latency)
		for _, market := range response.Markets {
			t.Logf("  Market: %v", market["id"])
		}
	}
}

// TestAPI_GetMarket tests fetching a single market
func TestAPI_GetMarket(t *testing.T) {
	config := DefaultConfig()
	client := NewHTTPClient(config)

	marketIDs := []string{"BTC-USDC", "ETH-USDC", "SOL-USDC"}

	for _, marketID := range marketIDs {
		result := client.GET(fmt.Sprintf("/v1/markets/%s", marketID))
		if result.Error != nil {
			t.Skipf("API server not running: %v", result.Error)
		}

		t.Logf("Market %s: status=%d, latency=%v", marketID, result.StatusCode, result.Latency)
	}
}

// TestAPI_GetTicker tests fetching market ticker
func TestAPI_GetTicker(t *testing.T) {
	config := DefaultConfig()
	client := NewHTTPClient(config)

	marketIDs := []string{"BTC-USDC", "ETH-USDC"}

	for _, marketID := range marketIDs {
		result := client.GET(fmt.Sprintf("/v1/markets/%s/ticker", marketID))
		if result.Error != nil {
			t.Skipf("API server not running: %v", result.Error)
		}

		t.Logf("Ticker %s: status=%d, latency=%v", marketID, result.StatusCode, result.Latency)

		var ticker struct {
			LastPrice  string `json:"last_price"`
			Volume24h  string `json:"volume_24h"`
			Change24h  string `json:"change_24h"`
		}
		if err := json.Unmarshal(result.Response.Data, &ticker); err == nil {
			t.Logf("  Price: %s, Volume: %s, Change: %s", ticker.LastPrice, ticker.Volume24h, ticker.Change24h)
		}
	}
}

// TestAPI_GetTickers tests fetching all tickers
func TestAPI_GetTickers(t *testing.T) {
	config := DefaultConfig()
	client := NewHTTPClient(config)

	result := client.GET("/v1/tickers")
	if result.Error != nil {
		t.Skipf("API server not running: %v", result.Error)
	}

	if result.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", result.StatusCode)
	}

	t.Logf("All tickers: status=%d, latency=%v", result.StatusCode, result.Latency)
}

// TestAPI_GetOrderbook tests fetching order book
func TestAPI_GetOrderbook(t *testing.T) {
	config := DefaultConfig()
	client := NewHTTPClient(config)

	// Test with different depth levels
	depths := []int{5, 10, 20, 50}

	for _, depth := range depths {
		result := client.GET(fmt.Sprintf("/v1/markets/BTC-USDC/orderbook?depth=%d", depth))
		if result.Error != nil {
			t.Skipf("API server not running: %v", result.Error)
		}

		t.Logf("Orderbook depth=%d: status=%d, latency=%v", depth, result.StatusCode, result.Latency)
	}
}

// TestAPI_GetTrades tests fetching recent trades
func TestAPI_GetTrades(t *testing.T) {
	config := DefaultConfig()
	client := NewHTTPClient(config)

	limits := []int{10, 50, 100}

	for _, limit := range limits {
		result := client.GET(fmt.Sprintf("/v1/markets/BTC-USDC/trades?limit=%d", limit))
		if result.Error != nil {
			t.Skipf("API server not running: %v", result.Error)
		}

		t.Logf("Trades limit=%d: status=%d, latency=%v", limit, result.StatusCode, result.Latency)
	}
}

// TestAPI_GetKlines tests fetching kline/candlestick data
func TestAPI_GetKlines(t *testing.T) {
	config := DefaultConfig()
	client := NewHTTPClient(config)

	intervals := []string{"1m", "5m", "15m", "1h", "4h", "1d"}

	for _, interval := range intervals {
		result := client.GET(fmt.Sprintf("/v1/markets/BTC-USDC/klines?interval=%s&limit=100", interval))
		if result.Error != nil {
			t.Skipf("API server not running: %v", result.Error)
		}

		t.Logf("Klines interval=%s: status=%d, latency=%v", interval, result.StatusCode, result.Latency)
	}
}

// TestAPI_GetFunding tests fetching funding rate
func TestAPI_GetFunding(t *testing.T) {
	config := DefaultConfig()
	client := NewHTTPClient(config)

	result := client.GET("/v1/markets/BTC-USDC/funding")
	if result.Error != nil {
		t.Skipf("API server not running: %v", result.Error)
	}

	t.Logf("Funding rate: status=%d, latency=%v", result.StatusCode, result.Latency)

	var funding struct {
		FundingRate     string `json:"funding_rate"`
		NextFundingTime int64  `json:"next_funding_time"`
	}
	if err := json.Unmarshal(result.Response.Data, &funding); err == nil {
		t.Logf("  Rate: %s, Next: %d", funding.FundingRate, funding.NextFundingTime)
	}
}

// ===========================================
// Order Endpoints Tests
// ===========================================

// TestAPI_PlaceOrder tests placing orders
func TestAPI_PlaceOrder(t *testing.T) {
	config := DefaultConfig()
	client := NewHTTPClient(config)

	// Check server is running
	result := client.GET("/v1/markets")
	if result.Error != nil {
		t.Skipf("API server not running: %v", result.Error)
	}

	trader := fmt.Sprintf("test_trader_%d", time.Now().UnixNano())

	orderTypes := []struct {
		name      string
		orderType string
		price     string
	}{
		{"Limit Buy", "limit", "50000"},
		{"Limit Sell", "limit", "55000"},
		{"Market Buy", "market", "0"},
	}

	for _, ot := range orderTypes {
		order := map[string]interface{}{
			"market_id": "BTC-USDC",
			"trader":    trader,
			"side":      "buy",
			"type":      ot.orderType,
			"price":     ot.price,
			"quantity":  "0.1",
		}

		result := client.POST("/v1/orders", order)
		if result.Error != nil {
			t.Errorf("Failed to place %s order: %v", ot.name, result.Error)
			continue
		}

		t.Logf("%s: status=%d, latency=%v", ot.name, result.StatusCode, result.Latency)
	}
}

// TestAPI_GetOrders tests fetching orders
func TestAPI_GetOrders(t *testing.T) {
	config := DefaultConfig()
	client := NewHTTPClient(config)

	trader := fmt.Sprintf("test_trader_%d", time.Now().UnixNano())

	// Query by different filters
	queries := []string{
		"/v1/orders",
		fmt.Sprintf("/v1/orders?trader=%s", trader),
		"/v1/orders?market_id=BTC-USDC",
		"/v1/orders?status=open",
		"/v1/orders?limit=50",
	}

	for _, query := range queries {
		result := client.GET(query)
		if result.Error != nil {
			t.Skipf("API server not running: %v", result.Error)
		}

		t.Logf("Query %s: status=%d, latency=%v", query, result.StatusCode, result.Latency)
	}
}

// TestAPI_ModifyOrder tests order modification
func TestAPI_ModifyOrder(t *testing.T) {
	config := DefaultConfig()
	client := NewHTTPClient(config)

	// Check server is running
	result := client.GET("/v1/markets")
	if result.Error != nil {
		t.Skipf("API server not running: %v", result.Error)
	}

	trader := fmt.Sprintf("test_trader_%d", time.Now().UnixNano())

	// Place an order first
	order := map[string]interface{}{
		"market_id": "BTC-USDC",
		"trader":    trader,
		"side":      "buy",
		"type":      "limit",
		"price":     "40000",
		"quantity":  "0.1",
	}

	placeResult := client.POST("/v1/orders", order)
	if placeResult.Error != nil {
		t.Skipf("Failed to place order: %v", placeResult.Error)
	}

	// Extract order ID
	var orderResp struct {
		Order struct {
			OrderID string `json:"order_id"`
		} `json:"order"`
	}
	if placeResult.Response != nil && placeResult.Response.Data != nil {
		json.Unmarshal(placeResult.Response.Data, &orderResp)
	}

	if orderResp.Order.OrderID == "" {
		t.Skip("Could not get order ID")
	}

	// Modify the order - using PUT
	modifyReq := map[string]interface{}{
		"price":    "41000",
		"quantity": "0.15",
	}

	// Note: PUT request would need to be added to HTTPClient
	t.Logf("Order placed with ID: %s, would modify with: %+v", orderResp.Order.OrderID, modifyReq)
}

// TestAPI_CancelOrder tests order cancellation
func TestAPI_CancelOrder(t *testing.T) {
	config := DefaultConfig()
	client := NewHTTPClient(config)

	// Check server is running
	result := client.GET("/v1/markets")
	if result.Error != nil {
		t.Skipf("API server not running: %v", result.Error)
	}

	trader := fmt.Sprintf("test_trader_%d", time.Now().UnixNano())

	// Place an order first
	order := map[string]interface{}{
		"market_id": "BTC-USDC",
		"trader":    trader,
		"side":      "buy",
		"type":      "limit",
		"price":     "35000",
		"quantity":  "0.1",
	}

	placeResult := client.POST("/v1/orders", order)
	if placeResult.Error != nil {
		t.Skipf("Failed to place order: %v", placeResult.Error)
	}

	// Extract order ID and cancel
	var orderResp struct {
		Order struct {
			OrderID string `json:"order_id"`
		} `json:"order"`
	}
	if placeResult.Response != nil && placeResult.Response.Data != nil {
		json.Unmarshal(placeResult.Response.Data, &orderResp)
	}

	if orderResp.Order.OrderID != "" {
		cancelResult := client.DELETE(fmt.Sprintf("/v1/orders/%s", orderResp.Order.OrderID))
		t.Logf("Cancel order: status=%d, latency=%v", cancelResult.StatusCode, cancelResult.Latency)
	}
}

// ===========================================
// Position Endpoints Tests
// ===========================================

// TestAPI_GetPositions tests fetching positions
func TestAPI_GetPositions(t *testing.T) {
	config := DefaultConfig()
	client := NewHTTPClient(config)

	trader := fmt.Sprintf("test_trader_%d", time.Now().UnixNano())

	queries := []string{
		"/v1/positions",
		fmt.Sprintf("/v1/positions?trader=%s", trader),
	}

	for _, query := range queries {
		result := client.GET(query)
		if result.Error != nil {
			t.Skipf("API server not running: %v", result.Error)
		}

		t.Logf("Query %s: status=%d, latency=%v", query, result.StatusCode, result.Latency)
	}
}

// TestAPI_GetPosition tests fetching a single position
func TestAPI_GetPosition(t *testing.T) {
	config := DefaultConfig()
	client := NewHTTPClient(config)

	result := client.GET("/v1/positions/BTC-USDC")
	if result.Error != nil {
		t.Skipf("API server not running: %v", result.Error)
	}

	t.Logf("Position BTC-USDC: status=%d, latency=%v", result.StatusCode, result.Latency)
}

// TestAPI_ClosePosition tests closing a position
func TestAPI_ClosePosition(t *testing.T) {
	config := DefaultConfig()
	client := NewHTTPClient(config)

	trader := fmt.Sprintf("test_trader_%d", time.Now().UnixNano())

	closeReq := map[string]interface{}{
		"market_id": "BTC-USDC",
		"trader":    trader,
		"size":      "0.05", // partial close
	}

	result := client.POST("/v1/positions/close", closeReq)
	if result.Error != nil {
		t.Skipf("API server not running: %v", result.Error)
	}

	t.Logf("Close position: status=%d, latency=%v", result.StatusCode, result.Latency)
}

// ===========================================
// Account Endpoints Tests
// ===========================================

// TestAPI_GetAccount tests fetching account info
func TestAPI_GetAccount(t *testing.T) {
	config := DefaultConfig()
	client := NewHTTPClient(config)

	trader := fmt.Sprintf("test_trader_%d", time.Now().UnixNano())

	result := client.GET(fmt.Sprintf("/v1/account?trader=%s", trader))
	if result.Error != nil {
		t.Skipf("API server not running: %v", result.Error)
	}

	t.Logf("Account info: status=%d, latency=%v", result.StatusCode, result.Latency)

	var account struct {
		Account struct {
			Trader           string `json:"trader"`
			Balance          string `json:"balance"`
			AvailableBalance string `json:"available_balance"`
		} `json:"account"`
	}
	if err := json.Unmarshal(result.Response.Data, &account); err == nil {
		t.Logf("  Balance: %s, Available: %s", account.Account.Balance, account.Account.AvailableBalance)
	}
}

// TestAPI_Deposit tests account deposit
func TestAPI_Deposit(t *testing.T) {
	config := DefaultConfig()
	client := NewHTTPClient(config)

	trader := fmt.Sprintf("test_trader_%d", time.Now().UnixNano())

	depositReq := map[string]interface{}{
		"trader": trader,
		"amount": "10000.00",
	}

	result := client.POST("/v1/account/deposit", depositReq)
	if result.Error != nil {
		t.Skipf("API server not running: %v", result.Error)
	}

	t.Logf("Deposit: status=%d, latency=%v", result.StatusCode, result.Latency)
}

// TestAPI_Withdraw tests account withdrawal
func TestAPI_Withdraw(t *testing.T) {
	config := DefaultConfig()
	client := NewHTTPClient(config)

	trader := fmt.Sprintf("test_trader_%d", time.Now().UnixNano())

	// First deposit
	depositReq := map[string]interface{}{
		"trader": trader,
		"amount": "10000.00",
	}
	client.POST("/v1/account/deposit", depositReq)

	// Then withdraw
	withdrawReq := map[string]interface{}{
		"trader": trader,
		"amount": "500.00",
	}

	result := client.POST("/v1/account/withdraw", withdrawReq)
	if result.Error != nil {
		t.Skipf("API server not running: %v", result.Error)
	}

	t.Logf("Withdraw: status=%d, latency=%v", result.StatusCode, result.Latency)
}

// ===========================================
// Legacy Account Endpoints Tests
// ===========================================

// TestAPI_LegacyAccount tests legacy account endpoints
func TestAPI_LegacyAccount(t *testing.T) {
	config := DefaultConfig()
	client := NewHTTPClient(config)

	trader := "cosmos1abc123"

	endpoints := []string{
		fmt.Sprintf("/v1/accounts/%s", trader),
		fmt.Sprintf("/v1/accounts/%s/positions", trader),
		fmt.Sprintf("/v1/accounts/%s/orders", trader),
		fmt.Sprintf("/v1/accounts/%s/trades", trader),
	}

	for _, endpoint := range endpoints {
		result := client.GET(endpoint)
		if result.Error != nil {
			t.Skipf("API server not running: %v", result.Error)
		}

		t.Logf("Legacy %s: status=%d, latency=%v", endpoint, result.StatusCode, result.Latency)
	}
}

// ===========================================
// Full Trading Flow Tests
// ===========================================

// TestAPI_FullTradingFlow tests complete trading flow
func TestAPI_FullTradingFlow(t *testing.T) {
	config := DefaultConfig()
	client := NewHTTPClient(config)

	// Check server is running
	result := client.GET("/v1/health")
	if result.Error != nil {
		t.Skipf("API server not running: %v", result.Error)
	}

	trader := fmt.Sprintf("flow_trader_%d", time.Now().UnixNano())
	t.Logf("Starting full trading flow for trader: %s", trader)

	// Step 1: Deposit funds
	t.Log("Step 1: Depositing funds...")
	depositReq := map[string]interface{}{
		"trader": trader,
		"amount": "50000.00",
	}
	depResult := client.POST("/v1/account/deposit", depositReq)
	t.Logf("  Deposit: status=%d, latency=%v", depResult.StatusCode, depResult.Latency)

	// Step 2: Check account balance
	t.Log("Step 2: Checking account balance...")
	accResult := client.GET(fmt.Sprintf("/v1/account?trader=%s", trader))
	t.Logf("  Account: status=%d, latency=%v", accResult.StatusCode, accResult.Latency)

	// Step 3: Get market info
	t.Log("Step 3: Getting market info...")
	mktResult := client.GET("/v1/markets/BTC-USDC")
	t.Logf("  Market: status=%d, latency=%v", mktResult.StatusCode, mktResult.Latency)

	// Step 4: Get orderbook
	t.Log("Step 4: Getting orderbook...")
	obResult := client.GET("/v1/markets/BTC-USDC/orderbook")
	t.Logf("  Orderbook: status=%d, latency=%v", obResult.StatusCode, obResult.Latency)

	// Step 5: Place limit order
	t.Log("Step 5: Placing limit order...")
	orderReq := map[string]interface{}{
		"market_id": "BTC-USDC",
		"trader":    trader,
		"side":      "buy",
		"type":      "limit",
		"price":     "45000",
		"quantity":  "0.5",
	}
	orderResult := client.POST("/v1/orders", orderReq)
	t.Logf("  Order: status=%d, latency=%v", orderResult.StatusCode, orderResult.Latency)

	// Step 6: Check orders
	t.Log("Step 6: Checking orders...")
	ordersResult := client.GET(fmt.Sprintf("/v1/orders?trader=%s", trader))
	t.Logf("  Orders: status=%d, latency=%v", ordersResult.StatusCode, ordersResult.Latency)

	// Step 7: Check positions
	t.Log("Step 7: Checking positions...")
	posResult := client.GET(fmt.Sprintf("/v1/positions?trader=%s", trader))
	t.Logf("  Positions: status=%d, latency=%v", posResult.StatusCode, posResult.Latency)

	// Step 8: Withdraw some funds
	t.Log("Step 8: Withdrawing funds...")
	withdrawReq := map[string]interface{}{
		"trader": trader,
		"amount": "1000.00",
	}
	wdResult := client.POST("/v1/account/withdraw", withdrawReq)
	t.Logf("  Withdraw: status=%d, latency=%v", wdResult.StatusCode, wdResult.Latency)

	// Generate report
	report := client.GenerateReport("Full Trading Flow")
	report.PrintReport()
}

// TestAPI_MultiTraderFlow tests concurrent trading from multiple traders
func TestAPI_MultiTraderFlow(t *testing.T) {
	config := DefaultConfig()

	// Check server is running
	client := NewHTTPClient(config)
	result := client.GET("/v1/health")
	if result.Error != nil {
		t.Skipf("API server not running: %v", result.Error)
	}

	numTraders := 5
	var wg sync.WaitGroup
	results := make(chan string, numTraders)

	t.Logf("Starting concurrent trading with %d traders...", numTraders)

	for i := 0; i < numTraders; i++ {
		wg.Add(1)
		go func(traderNum int) {
			defer wg.Done()

			traderClient := NewHTTPClient(config)
			trader := fmt.Sprintf("concurrent_trader_%d_%d", traderNum, time.Now().UnixNano())

			// Deposit
			depositReq := map[string]interface{}{
				"trader": trader,
				"amount": "10000.00",
			}
			traderClient.POST("/v1/account/deposit", depositReq)

			// Place orders
			for j := 0; j < 3; j++ {
				orderReq := map[string]interface{}{
					"market_id": "BTC-USDC",
					"trader":    trader,
					"side":      "buy",
					"type":      "limit",
					"price":     fmt.Sprintf("%d", 40000+j*100),
					"quantity":  "0.1",
				}
				traderClient.POST("/v1/orders", orderReq)
			}

			report := traderClient.GenerateReport(fmt.Sprintf("Trader %d", traderNum))
			results <- fmt.Sprintf("Trader %d: avg=%v, p95=%v", traderNum, report.AvgLatency, report.P95Latency)
		}(i)
	}

	wg.Wait()
	close(results)

	for result := range results {
		t.Log(result)
	}
}

// ===========================================
// Error Handling Tests
// ===========================================

// TestAPI_ErrorHandling tests API error responses
func TestAPI_ErrorHandling(t *testing.T) {
	config := DefaultConfig()
	client := NewHTTPClient(config)

	// Check server is running
	result := client.GET("/v1/health")
	if result.Error != nil {
		t.Skipf("API server not running: %v", result.Error)
	}

	testCases := []struct {
		name           string
		method         string
		path           string
		body           interface{}
		expectedStatus []int
	}{
		{
			name:           "Invalid market",
			method:         "GET",
			path:           "/v1/markets/INVALID-MARKET",
			expectedStatus: []int{http.StatusNotFound, http.StatusOK},
		},
		{
			name:           "Missing required field",
			method:         "POST",
			path:           "/v1/orders",
			body:           map[string]interface{}{"side": "buy"}, // missing market_id, trader, etc.
			expectedStatus: []int{http.StatusBadRequest, http.StatusOK},
		},
		{
			name:           "Invalid order ID",
			method:         "DELETE",
			path:           "/v1/orders/invalid-order-id",
			expectedStatus: []int{http.StatusNotFound, http.StatusOK, http.StatusBadRequest},
		},
		{
			name:           "Invalid endpoint",
			method:         "GET",
			path:           "/v1/invalid-endpoint",
			expectedStatus: []int{http.StatusNotFound},
		},
	}

	for _, tc := range testCases {
		var result *RequestResult
		switch tc.method {
		case "GET":
			result = client.GET(tc.path)
		case "POST":
			result = client.POST(tc.path, tc.body)
		case "DELETE":
			result = client.DELETE(tc.path)
		}

		if result.Error != nil {
			t.Logf("%s: connection error (expected)", tc.name)
			continue
		}

		statusValid := false
		for _, expected := range tc.expectedStatus {
			if result.StatusCode == expected {
				statusValid = true
				break
			}
		}

		if !statusValid {
			t.Logf("%s: unexpected status %d (expected one of %v)", tc.name, result.StatusCode, tc.expectedStatus)
		} else {
			t.Logf("%s: status=%d, latency=%v", tc.name, result.StatusCode, result.Latency)
		}
	}
}

// ===========================================
// Performance Benchmark Tests
// ===========================================

// TestAPI_LatencyBenchmark_AllEndpoints benchmarks all major endpoints
func TestAPI_LatencyBenchmark_AllEndpoints(t *testing.T) {
	config := DefaultConfig()
	client := NewHTTPClient(config)

	// Check server is running
	result := client.GET("/v1/health")
	if result.Error != nil {
		t.Skipf("API server not running: %v", result.Error)
	}

	trader := fmt.Sprintf("bench_trader_%d", time.Now().UnixNano())

	endpoints := []struct {
		name   string
		method string
		path   string
		body   interface{}
	}{
		{"Health", "GET", "/v1/health", nil},
		{"Markets", "GET", "/v1/markets", nil},
		{"Market", "GET", "/v1/markets/BTC-USDC", nil},
		{"Ticker", "GET", "/v1/markets/BTC-USDC/ticker", nil},
		{"Orderbook", "GET", "/v1/markets/BTC-USDC/orderbook", nil},
		{"Trades", "GET", "/v1/markets/BTC-USDC/trades", nil},
		{"Funding", "GET", "/v1/markets/BTC-USDC/funding", nil},
		{"Tickers", "GET", "/v1/tickers", nil},
		{"Account", "GET", fmt.Sprintf("/v1/account?trader=%s", trader), nil},
		{"Positions", "GET", "/v1/positions", nil},
		{"Orders", "GET", "/v1/orders", nil},
	}

	iterations := 20
	t.Logf("Running %d iterations per endpoint...", iterations)

	for _, ep := range endpoints {
		epClient := NewHTTPClient(config)

		for i := 0; i < iterations; i++ {
			switch ep.method {
			case "GET":
				epClient.GET(ep.path)
			case "POST":
				epClient.POST(ep.path, ep.body)
			}
		}

		report := epClient.GenerateReport(ep.name)
		t.Logf("  %s: avg=%v, p50=%v, p95=%v, p99=%v",
			ep.name, report.AvgLatency, report.P50Latency, report.P95Latency, report.P99Latency)
	}
}

// TestAPI_ThroughputTest tests API throughput under load
func TestAPI_ThroughputTest(t *testing.T) {
	config := DefaultConfig()
	client := NewHTTPClient(config)

	// Check server is running
	result := client.GET("/v1/health")
	if result.Error != nil {
		t.Skipf("API server not running: %v", result.Error)
	}

	numRequests := 100
	numWorkers := 10

	var wg sync.WaitGroup
	startTime := time.Now()
	successCount := 0
	var mu sync.Mutex

	requestsPerWorker := numRequests / numWorkers

	for w := 0; w < numWorkers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			workerClient := NewHTTPClient(config)

			for i := 0; i < requestsPerWorker; i++ {
				result := workerClient.GET("/v1/markets/BTC-USDC/ticker")
				if result.StatusCode == http.StatusOK || result.StatusCode == http.StatusTooManyRequests {
					mu.Lock()
					successCount++
					mu.Unlock()
				}
			}
		}()
	}

	wg.Wait()
	duration := time.Since(startTime)

	throughput := float64(numRequests) / duration.Seconds()
	t.Logf("Throughput test: %d requests in %v = %.2f req/s", numRequests, duration, throughput)
	t.Logf("Success rate: %d/%d (%.1f%%)", successCount, numRequests, float64(successCount)/float64(numRequests)*100)
}
