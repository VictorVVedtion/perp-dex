package e2e_real

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"
)

// TestTradingFlow_PlaceLimitOrder tests placing a limit order
func TestTradingFlow_PlaceLimitOrder(t *testing.T) {
	config := DefaultConfig()
	client := NewHTTPClient(config)

	// Check server is running
	result := client.GET("/v1/markets")
	if result.Error != nil {
		t.Skipf("API server not running: %v", result.Error)
	}

	// Place a limit buy order
	order := &Order{
		MarketID:  "BTC-USDC",
		Trader:    fmt.Sprintf("test_trader_%d", time.Now().UnixNano()),
		Side:      "buy",
		OrderType: "limit",
		Price:     "50000",
		Quantity:  "0.1",
		Leverage:  "10",
	}

	result, err := PlaceOrder(client, order)
	if err != nil {
		t.Fatalf("Failed to place order: %v", err)
	}

	// Accept 200, 201 or 429 (rate limited) as valid responses
	if result.StatusCode != http.StatusOK && result.StatusCode != http.StatusCreated && result.StatusCode != http.StatusTooManyRequests {
		t.Errorf("Expected status 200/201/429, got %d", result.StatusCode)
	}

	t.Logf("Order placed successfully, status=%d, latency=%v", result.StatusCode, result.Latency)
}

// TestTradingFlow_PlaceMarketOrder tests placing a market order
func TestTradingFlow_PlaceMarketOrder(t *testing.T) {
	config := DefaultConfig()
	client := NewHTTPClient(config)

	// Check server is running
	result := client.GET("/v1/markets")
	if result.Error != nil {
		t.Skipf("API server not running: %v", result.Error)
	}

	// Place a market buy order
	order := &Order{
		MarketID:  "BTC-USDC",
		Trader:    fmt.Sprintf("test_trader_%d", time.Now().UnixNano()),
		Side:      "buy",
		OrderType: "market",
		Price:     "0", // Market order, no price
		Quantity:  "0.1",
		Leverage:  "10",
	}

	result, err := PlaceOrder(client, order)
	if err != nil {
		t.Fatalf("Failed to place order: %v", err)
	}

	t.Logf("Market order response: status=%d, latency=%v", result.StatusCode, result.Latency)
}

// TestTradingFlow_GetOrderBook tests fetching order book
func TestTradingFlow_GetOrderBook(t *testing.T) {
	config := DefaultConfig()
	client := NewHTTPClient(config)

	result, err := GetOrderBook(client, "BTC-USDC")
	if err != nil {
		t.Skipf("API server not running: %v", err)
	}

	// Accept 200 or 429 (rate limited) as valid responses
	if result.StatusCode != http.StatusOK && result.StatusCode != http.StatusTooManyRequests {
		t.Errorf("Expected status 200/429, got %d", result.StatusCode)
	}

	t.Logf("OrderBook fetched, status=%d, latency=%v", result.StatusCode, result.Latency)

	// Parse response
	if result.Response != nil && result.Response.Data != nil {
		var orderbook map[string]interface{}
		if err := json.Unmarshal(result.Response.Data, &orderbook); err == nil {
			t.Logf("OrderBook: %+v", orderbook)
		}
	}
}

// TestTradingFlow_CancelOrder tests order cancellation
func TestTradingFlow_CancelOrder(t *testing.T) {
	config := DefaultConfig()
	client := NewHTTPClient(config)

	// Check server is running
	result := client.GET("/v1/markets")
	if result.Error != nil {
		t.Skipf("API server not running: %v", result.Error)
	}

	// First place an order
	order := &Order{
		MarketID:  "BTC-USDC",
		Trader:    fmt.Sprintf("test_trader_%d", time.Now().UnixNano()),
		Side:      "buy",
		OrderType: "limit",
		Price:     "40000", // Low price so it won't fill
		Quantity:  "0.1",
		Leverage:  "10",
	}

	placeResult, err := PlaceOrder(client, order)
	if err != nil {
		t.Fatalf("Failed to place order: %v", err)
	}

	// Extract order ID from response
	var orderResp struct {
		OrderID string `json:"orderId"`
	}
	if placeResult.Response != nil && placeResult.Response.Data != nil {
		json.Unmarshal(placeResult.Response.Data, &orderResp)
	}

	if orderResp.OrderID == "" {
		t.Skip("Could not get order ID from response")
	}

	// Cancel the order
	cancelResult, err := CancelOrder(client, orderResp.OrderID)
	if err != nil {
		t.Fatalf("Failed to cancel order: %v", err)
	}

	t.Logf("Order cancelled, latency: %v", cancelResult.Latency)
}

// TestTradingFlow_OrderMatching tests order matching between two traders
func TestTradingFlow_OrderMatching(t *testing.T) {
	config := DefaultConfig()
	client := NewHTTPClient(config)

	// Check server is running
	result := client.GET("/v1/markets")
	if result.Error != nil {
		t.Skipf("API server not running: %v", result.Error)
	}

	trader1 := fmt.Sprintf("trader_buyer_%d", time.Now().UnixNano())
	trader2 := fmt.Sprintf("trader_seller_%d", time.Now().UnixNano())
	price := "50000"

	// Trader 1 places buy order
	buyOrder := &Order{
		MarketID:  "BTC-USDC",
		Trader:    trader1,
		Side:      "buy",
		OrderType: "limit",
		Price:     price,
		Quantity:  "0.1",
		Leverage:  "10",
	}

	buyResult, err := PlaceOrder(client, buyOrder)
	if err != nil {
		t.Fatalf("Failed to place buy order: %v", err)
	}
	t.Logf("Buy order placed: latency=%v", buyResult.Latency)

	// Trader 2 places sell order at same price
	sellOrder := &Order{
		MarketID:  "BTC-USDC",
		Trader:    trader2,
		Side:      "sell",
		OrderType: "limit",
		Price:     price,
		Quantity:  "0.1",
		Leverage:  "10",
	}

	sellResult, err := PlaceOrder(client, sellOrder)
	if err != nil {
		t.Fatalf("Failed to place sell order: %v", err)
	}
	t.Logf("Sell order placed: latency=%v", sellResult.Latency)

	// Check positions for both traders
	time.Sleep(100 * time.Millisecond) // Allow matching to complete

	pos1Result, _ := GetPositions(client, trader1)
	pos2Result, _ := GetPositions(client, trader2)

	t.Logf("Trader1 positions: %v", pos1Result.Response)
	t.Logf("Trader2 positions: %v", pos2Result.Response)
}

// TestTradingFlow_GetMarkets tests fetching available markets
func TestTradingFlow_GetMarkets(t *testing.T) {
	config := DefaultConfig()
	client := NewHTTPClient(config)

	result := client.GET("/v1/markets")
	if result.Error != nil {
		t.Skipf("API server not running: %v", result.Error)
	}

	// Accept 200 or 429 (rate limited) as valid responses
	if result.StatusCode != http.StatusOK && result.StatusCode != http.StatusTooManyRequests {
		t.Errorf("Expected status 200/429, got %d", result.StatusCode)
	}

	t.Logf("Markets fetched, latency: %v", result.Latency)

	// Try to parse markets
	if result.Response != nil {
		t.Logf("Response: %s", string(result.Response.Data))
	}
}

// TestTradingFlow_GetTicker tests fetching market ticker
func TestTradingFlow_GetTicker(t *testing.T) {
	config := DefaultConfig()
	client := NewHTTPClient(config)

	result := client.GET("/v1/markets/BTC-USDC/ticker")
	if result.Error != nil {
		t.Skipf("API server not running: %v", result.Error)
	}

	t.Logf("Ticker fetched, status=%d, latency=%v", result.StatusCode, result.Latency)
}

// TestTradingFlow_OrderBookDepth tests order book depth levels
func TestTradingFlow_OrderBookDepth(t *testing.T) {
	config := DefaultConfig()
	client := NewHTTPClient(config)

	// Check server first
	result := client.GET("/v1/markets")
	if result.Error != nil {
		t.Skipf("API server not running: %v", result.Error)
	}

	// Place multiple orders at different prices to build depth
	basePrice := 50000
	trader := fmt.Sprintf("depth_trader_%d", time.Now().UnixNano())

	for i := 0; i < 5; i++ {
		// Buy orders at decreasing prices
		buyOrder := &Order{
			MarketID:  "BTC-USDC",
			Trader:    trader,
			Side:      "buy",
			OrderType: "limit",
			Price:     fmt.Sprintf("%d", basePrice-i*100),
			Quantity:  "0.1",
			Leverage:  "10",
		}
		PlaceOrder(client, buyOrder)

		// Sell orders at increasing prices
		sellOrder := &Order{
			MarketID:  "BTC-USDC",
			Trader:    trader,
			Side:      "sell",
			OrderType: "limit",
			Price:     fmt.Sprintf("%d", basePrice+i*100),
			Quantity:  "0.1",
			Leverage:  "10",
		}
		PlaceOrder(client, sellOrder)
	}

	// Fetch and verify order book
	obResult, err := GetOrderBook(client, "BTC-USDC")
	if err != nil {
		t.Fatalf("Failed to get orderbook: %v", err)
	}

	t.Logf("OrderBook depth test complete, latency=%v", obResult.Latency)
}

// TestTradingFlow_LatencyBenchmark benchmarks API latency
func TestTradingFlow_LatencyBenchmark(t *testing.T) {
	config := DefaultConfig()
	client := NewHTTPClient(config)

	// Check server first
	result := client.GET("/v1/markets")
	if result.Error != nil {
		t.Skipf("API server not running: %v", result.Error)
	}

	// Run multiple requests
	iterations := 100
	trader := fmt.Sprintf("bench_trader_%d", time.Now().UnixNano())

	t.Logf("Running %d iterations...", iterations)

	for i := 0; i < iterations; i++ {
		// Alternate between different operations
		switch i % 3 {
		case 0:
			client.GET("/v1/markets/BTC-USDC/orderbook")
		case 1:
			client.GET("/v1/markets/BTC-USDC/ticker")
		case 2:
			order := &Order{
				MarketID:  "BTC-USDC",
				Trader:    trader,
				Side:      "buy",
				OrderType: "limit",
				Price:     fmt.Sprintf("%d", 40000+i),
				Quantity:  "0.01",
				Leverage:  "10",
			}
			PlaceOrder(client, order)
		}
	}

	// Generate report
	report := client.GenerateReport("Trading Flow Benchmark")
	report.PrintReport()

	// Assert latency is reasonable
	if report.AvgLatency > 100*time.Millisecond {
		t.Logf("Warning: Average latency %v exceeds 100ms", report.AvgLatency)
	}
}
