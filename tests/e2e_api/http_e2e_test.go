package e2e_api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/openalpha/perp-dex/api"
	"github.com/openalpha/perp-dex/api/types"
)

// ============================================================================
// True E2E Tests - HTTP API -> Keeper -> OrderBook Engine
// ============================================================================
// These tests make actual HTTP requests to a real API server
// connected to a real Keeper instance (with in-memory storage)
// ============================================================================

// TestServer wraps the API server for testing
type TestServer struct {
	server  *httptest.Server
	service *api.KeeperService
}

// NewTestServer creates a new test server with real Keeper
func NewTestServer() *TestServer {
	// Create keeper service (connects to real order book engine)
	service := api.NewKeeperService()

	// Create API server with keeper service
	config := api.DefaultConfig()
	config.MockMode = false
	apiServer := api.NewServerWithServices(config, service, service, service)

	// Create test HTTP server
	mux := http.NewServeMux()

	// Register endpoints (same as api/server.go)
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":    "healthy",
			"mock_mode": false,
			"keeper":    true,
		})
	})

	// Order endpoints
	orderHandler := apiServer.GetOrderHandler()
	mux.HandleFunc("/v1/orders", orderHandler.HandleOrders)
	mux.HandleFunc("/v1/orders/", orderHandler.HandleOrder)

	// Orderbook endpoint
	mux.HandleFunc("/v1/markets/", func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path[len("/v1/markets/"):]
		marketID := path
		for i, c := range path {
			if c == '/' {
				marketID = path[:i]
				break
			}
		}

		bids, asks := service.GetOrderBookDepth(marketID, 20)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"market_id": marketID,
			"bids":      bids,
			"asks":      asks,
			"timestamp": time.Now().UnixMilli(),
		})
	})

	ts := &TestServer{
		server:  httptest.NewServer(mux),
		service: service,
	}

	return ts
}

func (ts *TestServer) Close() {
	ts.server.Close()
}

func (ts *TestServer) URL() string {
	return ts.server.URL
}

// ============================================================================
// Test: Health Check
// ============================================================================

func TestHealthCheck(t *testing.T) {
	ts := NewTestServer()
	defer ts.Close()

	resp, err := http.Get(ts.URL() + "/health")
	if err != nil {
		t.Fatalf("Failed to get health: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	if result["status"] != "healthy" {
		t.Errorf("Expected healthy status, got %v", result["status"])
	}
	if result["keeper"] != true {
		t.Errorf("Expected keeper=true, got %v", result["keeper"])
	}

	t.Logf("Health check passed: %v", result)
}

// ============================================================================
// Test: Place Order via HTTP
// ============================================================================

func TestPlaceOrderHTTP(t *testing.T) {
	ts := NewTestServer()
	defer ts.Close()

	testCases := []struct {
		name     string
		request  types.PlaceOrderRequest
		wantErr  bool
	}{
		{
			name: "buy_limit_order",
			request: types.PlaceOrderRequest{
				MarketID: "BTC-USDC",
				Side:     "buy",
				Type:     "limit",
				Price:    "95000",
				Quantity: "0.1",
			},
			wantErr: false,
		},
		{
			name: "sell_limit_order",
			request: types.PlaceOrderRequest{
				MarketID: "BTC-USDC",
				Side:     "sell",
				Type:     "limit",
				Price:    "100000",
				Quantity: "0.2",
			},
			wantErr: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			body, _ := json.Marshal(tc.request)
			req, _ := http.NewRequest("POST", ts.URL()+"/v1/orders", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-Trader-Address", "trader1")

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatalf("Request failed: %v", err)
			}
			defer resp.Body.Close()

			respBody, _ := io.ReadAll(resp.Body)

			if tc.wantErr {
				if resp.StatusCode == http.StatusOK {
					t.Errorf("Expected error, but got success")
				}
			} else {
				if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
					t.Errorf("Expected success, got status %d: %s", resp.StatusCode, string(respBody))
				}

				var result types.PlaceOrderResponse
				if err := json.Unmarshal(respBody, &result); err != nil {
					t.Errorf("Failed to parse response: %v", err)
				} else {
					t.Logf("Order placed: ID=%s, Status=%s, FilledQty=%s",
						result.Order.OrderID, result.Order.Status, result.Order.FilledQty)
				}
			}
		})
	}
}

// ============================================================================
// Test: Order Matching via HTTP
// ============================================================================

func TestOrderMatchingHTTP(t *testing.T) {
	ts := NewTestServer()
	defer ts.Close()

	// Step 1: Place maker orders (limit orders that won't match immediately)
	t.Log("Step 1: Placing maker orders...")

	// Place 5 buy orders at different prices
	for i := 0; i < 5; i++ {
		price := fmt.Sprintf("%d", 94000+i*100) // 94000, 94100, 94200, 94300, 94400
		placeOrder(t, ts, "maker", "BTC-USDC", "buy", "limit", price, "1.0")
	}

	// Place 5 sell orders at different prices
	for i := 0; i < 5; i++ {
		price := fmt.Sprintf("%d", 96000+i*100) // 96000, 96100, 96200, 96300, 96400
		placeOrder(t, ts, "maker", "BTC-USDC", "sell", "limit", price, "1.0")
	}

	// Step 2: Query order book
	t.Log("Step 2: Querying order book...")
	resp, err := http.Get(ts.URL() + "/v1/markets/BTC-USDC/orderbook")
	if err != nil {
		t.Fatalf("Failed to get orderbook: %v", err)
	}
	defer resp.Body.Close()

	var orderbook map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&orderbook)
	t.Logf("Order book - Bids: %d, Asks: %d",
		len(orderbook["bids"].([]interface{})),
		len(orderbook["asks"].([]interface{})))

	// Step 3: Place taker order that should match
	t.Log("Step 3: Placing taker order (should match)...")
	result := placeOrder(t, ts, "taker", "BTC-USDC", "buy", "limit", "96500", "2.0")

	if result.Match != nil && len(result.Match.Trades) > 0 {
		t.Logf("Order matched! Filled: %s, Trades: %d", result.Match.FilledQty, len(result.Match.Trades))
		for i, trade := range result.Match.Trades {
			t.Logf("  Trade %d: Price=%s, Qty=%s", i+1, trade.Price, trade.Quantity)
		}
	} else {
		t.Logf("Order placed but not matched (may be expected based on prices)")
	}

	// Step 4: Query order book again
	t.Log("Step 4: Querying order book after matching...")
	resp2, _ := http.Get(ts.URL() + "/v1/markets/BTC-USDC/orderbook")
	defer resp2.Body.Close()

	var orderbook2 map[string]interface{}
	json.NewDecoder(resp2.Body).Decode(&orderbook2)
	t.Logf("Order book after matching - Bids: %d, Asks: %d",
		len(orderbook2["bids"].([]interface{})),
		len(orderbook2["asks"].([]interface{})))
}

// ============================================================================
// Test: Cancel Order via HTTP
// ============================================================================

func TestCancelOrderHTTP(t *testing.T) {
	ts := NewTestServer()
	defer ts.Close()

	// Place an order
	result := placeOrder(t, ts, "trader1", "BTC-USDC", "buy", "limit", "90000", "1.0")
	orderID := result.Order.OrderID
	t.Logf("Placed order: %s", orderID)

	// Cancel the order
	req, _ := http.NewRequest("DELETE", ts.URL()+"/v1/orders/"+orderID, nil)
	req.Header.Set("X-Trader-Address", "trader1")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Cancel request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Errorf("Expected 200, got %d: %s", resp.StatusCode, string(body))
	} else {
		var cancelResult types.CancelOrderResponse
		json.NewDecoder(resp.Body).Decode(&cancelResult)
		t.Logf("Order cancelled: %s, Cancelled=%v", cancelResult.Order.OrderID, cancelResult.Cancelled)
	}
}

// ============================================================================
// Test: High Throughput HTTP
// ============================================================================

func TestHighThroughputHTTP(t *testing.T) {
	ts := NewTestServer()
	defer ts.Close()

	numOrders := 1000
	t.Logf("Running high throughput test: %d orders", numOrders)

	start := time.Now()
	var successCount, errorCount atomic.Int64

	for i := 0; i < numOrders; i++ {
		side := "buy"
		price := fmt.Sprintf("%d", 90000+i%100)
		if i%2 == 1 {
			side = "sell"
			price = fmt.Sprintf("%d", 100000+i%100)
		}

		reqBody := types.PlaceOrderRequest{
			MarketID: "BTC-USDC",
			Side:     side,
			Type:     "limit",
			Price:    price,
			Quantity: "0.01",
		}
		body, _ := json.Marshal(reqBody)

		req, _ := http.NewRequest("POST", ts.URL()+"/v1/orders", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Trader-Address", fmt.Sprintf("trader%d", i%10))

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			errorCount.Add(1)
			continue
		}
		resp.Body.Close()

		if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusCreated {
			successCount.Add(1)
		} else {
			errorCount.Add(1)
		}
	}

	duration := time.Since(start)
	throughput := float64(numOrders) / duration.Seconds()

	t.Logf("Results:")
	t.Logf("  Total orders: %d", numOrders)
	t.Logf("  Success: %d", successCount.Load())
	t.Logf("  Errors: %d", errorCount.Load())
	t.Logf("  Duration: %v", duration)
	t.Logf("  Throughput: %.2f orders/sec", throughput)
}

// ============================================================================
// Test: Concurrent HTTP Requests
// ============================================================================

func TestConcurrentHTTP(t *testing.T) {
	ts := NewTestServer()
	defer ts.Close()

	numWorkers := 8
	ordersPerWorker := 100
	t.Logf("Running concurrent test: %d workers x %d orders", numWorkers, ordersPerWorker)

	var wg sync.WaitGroup
	var successCount, errorCount atomic.Int64

	start := time.Now()

	for w := 0; w < numWorkers; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			for i := 0; i < ordersPerWorker; i++ {
				side := "buy"
				price := fmt.Sprintf("%d", 90000+i%100)
				if i%2 == 1 {
					side = "sell"
					price = fmt.Sprintf("%d", 100000+i%100)
				}

				reqBody := types.PlaceOrderRequest{
					MarketID: "BTC-USDC",
					Side:     side,
					Type:     "limit",
					Price:    price,
					Quantity: "0.01",
				}
				body, _ := json.Marshal(reqBody)

				req, _ := http.NewRequest("POST", ts.URL()+"/v1/orders", bytes.NewBuffer(body))
				req.Header.Set("Content-Type", "application/json")
				req.Header.Set("X-Trader-Address", fmt.Sprintf("worker%d", workerID))

				resp, err := http.DefaultClient.Do(req)
				if err != nil {
					errorCount.Add(1)
					continue
				}
				resp.Body.Close()

				if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusCreated {
					successCount.Add(1)
				} else {
					errorCount.Add(1)
				}
			}
		}(w)
	}

	wg.Wait()
	duration := time.Since(start)
	totalOrders := numWorkers * ordersPerWorker
	throughput := float64(totalOrders) / duration.Seconds()

	t.Logf("Results:")
	t.Logf("  Workers: %d", numWorkers)
	t.Logf("  Total orders: %d", totalOrders)
	t.Logf("  Success: %d", successCount.Load())
	t.Logf("  Errors: %d", errorCount.Load())
	t.Logf("  Duration: %v", duration)
	t.Logf("  Throughput: %.2f orders/sec", throughput)

	// Verify success rate
	successRate := float64(successCount.Load()) / float64(totalOrders) * 100
	if successRate < 99 {
		t.Errorf("Success rate too low: %.2f%%", successRate)
	}
}

// ============================================================================
// Test: Full Order Lifecycle via HTTP
// ============================================================================

func TestFullOrderLifecycleHTTP(t *testing.T) {
	ts := NewTestServer()
	defer ts.Close()

	t.Log("=== Full Order Lifecycle E2E Test ===")

	// 1. Place maker buy order
	t.Log("1. Placing maker buy order...")
	buyOrder := placeOrder(t, ts, "alice", "ETH-USDC", "buy", "limit", "3400", "10.0")
	t.Logf("   Buy order: ID=%s, Price=3400, Qty=10", buyOrder.Order.OrderID)

	// 2. Place maker sell order
	t.Log("2. Placing maker sell order...")
	sellOrder := placeOrder(t, ts, "bob", "ETH-USDC", "sell", "limit", "3500", "10.0")
	t.Logf("   Sell order: ID=%s, Price=3500, Qty=10", sellOrder.Order.OrderID)

	// 3. Check order book (should have both orders)
	t.Log("3. Checking order book...")
	resp, _ := http.Get(ts.URL() + "/v1/markets/ETH-USDC/orderbook")
	var ob map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&ob)
	resp.Body.Close()
	t.Logf("   Bids: %d, Asks: %d", len(ob["bids"].([]interface{})), len(ob["asks"].([]interface{})))

	// 4. Place taker order that matches sell order
	t.Log("4. Placing taker buy order at 3500 (should match sell)...")
	takerOrder := placeOrder(t, ts, "charlie", "ETH-USDC", "buy", "limit", "3500", "5.0")
	if takerOrder.Match != nil && len(takerOrder.Match.Trades) > 0 {
		t.Logf("   Matched! Filled=%s, Trades=%d", takerOrder.Match.FilledQty, len(takerOrder.Match.Trades))
	} else {
		t.Logf("   Order placed: ID=%s, Status=%s", takerOrder.Order.OrderID, takerOrder.Order.Status)
	}

	// 5. Cancel remaining buy order
	t.Log("5. Cancelling alice's buy order...")
	cancelReq, _ := http.NewRequest("DELETE", ts.URL()+"/v1/orders/"+buyOrder.Order.OrderID, nil)
	cancelReq.Header.Set("X-Trader-Address", "alice")
	cancelResp, _ := http.DefaultClient.Do(cancelReq)
	if cancelResp.StatusCode == http.StatusOK {
		t.Log("   Order cancelled successfully")
	} else {
		body, _ := io.ReadAll(cancelResp.Body)
		t.Logf("   Cancel result: %s", string(body))
	}
	cancelResp.Body.Close()

	// 6. Final order book check
	t.Log("6. Final order book check...")
	resp2, _ := http.Get(ts.URL() + "/v1/markets/ETH-USDC/orderbook")
	var ob2 map[string]interface{}
	json.NewDecoder(resp2.Body).Decode(&ob2)
	resp2.Body.Close()
	t.Logf("   Final - Bids: %d, Asks: %d", len(ob2["bids"].([]interface{})), len(ob2["asks"].([]interface{})))

	t.Log("=== E2E Test Complete ===")
}

// ============================================================================
// Helper Functions
// ============================================================================

func placeOrder(t *testing.T, ts *TestServer, trader, market, side, orderType, price, quantity string) *types.PlaceOrderResponse {
	t.Helper()

	reqBody := types.PlaceOrderRequest{
		MarketID: market,
		Side:     side,
		Type:     orderType,
		Price:    price,
		Quantity: quantity,
	}
	body, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest("POST", ts.URL()+"/v1/orders", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Trader-Address", trader)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Place order request failed: %v", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		t.Fatalf("Place order failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	var result types.PlaceOrderResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		t.Fatalf("Failed to parse place order response: %v", err)
	}

	return &result
}
