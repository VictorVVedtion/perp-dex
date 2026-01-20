package e2e_real

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"
)

// TestLiquidation_PositionAtRisk tests position at liquidation risk
func TestLiquidation_PositionAtRisk(t *testing.T) {
	config := DefaultConfig()
	client := NewHTTPClient(config)

	// Check server is running
	result := client.GET("/v1/markets")
	if result.Error != nil {
		t.Skipf("API server not running: %v", result.Error)
	}

	// Create a position with high leverage
	trader := fmt.Sprintf("liq_trader_%d", time.Now().UnixNano())

	// Place a long order with max leverage
	order := &Order{
		MarketID:  "BTC-USDC",
		Trader:    trader,
		Side:      "buy",
		OrderType: "limit",
		Price:     "50000",
		Quantity:  "1.0",
		Leverage:  "50", // Max leverage
	}

	result, err := PlaceOrder(client, order)
	if err != nil {
		t.Fatalf("Failed to place order: %v", err)
	}

	t.Logf("High-leverage order placed: status=%d, latency=%v", result.StatusCode, result.Latency)

	// Check position
	posResult, _ := GetPositions(client, trader)
	t.Logf("Position status: %v", posResult.Response)
}

// TestLiquidation_GetLiquidablePositions tests fetching liquidable positions
func TestLiquidation_GetLiquidablePositions(t *testing.T) {
	config := DefaultConfig()
	client := NewHTTPClient(config)

	result := client.GET("/v1/liquidations/positions")
	if result.Error != nil {
		t.Skipf("API server not running: %v", result.Error)
	}

	t.Logf("Liquidable positions: status=%d, latency=%v", result.StatusCode, result.Latency)

	if result.Response != nil {
		t.Logf("Response: %s", string(result.Response.Data))
	}
}

// TestLiquidation_ExecuteLiquidation tests executing a liquidation
func TestLiquidation_ExecuteLiquidation(t *testing.T) {
	config := DefaultConfig()
	client := NewHTTPClient(config)

	// Check server is running
	result := client.GET("/v1/markets")
	if result.Error != nil {
		t.Skipf("API server not running: %v", result.Error)
	}

	// Try to execute liquidation (may fail if no positions to liquidate)
	liquidateReq := map[string]interface{}{
		"marketId": "BTC-USDC",
		"trader":   "test_liquidation_target",
	}

	result = client.POST("/v1/liquidations/execute", liquidateReq)
	t.Logf("Liquidation execute: status=%d, latency=%v", result.StatusCode, result.Latency)
}

// TestLiquidation_GetInsuranceFund tests fetching insurance fund status
func TestLiquidation_GetInsuranceFund(t *testing.T) {
	config := DefaultConfig()
	client := NewHTTPClient(config)

	result := client.GET("/v1/insurance-fund")
	if result.Error != nil {
		t.Skipf("API server not running: %v", result.Error)
	}

	t.Logf("Insurance fund: status=%d, latency=%v", result.StatusCode, result.Latency)

	if result.Response != nil {
		t.Logf("Response: %s", string(result.Response.Data))
	}
}

// TestLiquidation_ADLQueue tests ADL (Auto-Deleveraging) queue
func TestLiquidation_ADLQueue(t *testing.T) {
	config := DefaultConfig()
	client := NewHTTPClient(config)

	result := client.GET("/v1/adl/queue")
	if result.Error != nil {
		t.Skipf("API server not running: %v", result.Error)
	}

	t.Logf("ADL queue: status=%d, latency=%v", result.StatusCode, result.Latency)
}

// TestLiquidation_MarginCall tests margin call mechanics
func TestLiquidation_MarginCall(t *testing.T) {
	config := DefaultConfig()
	client := NewHTTPClient(config)

	// Check server
	result := client.GET("/v1/markets")
	if result.Error != nil {
		t.Skipf("API server not running: %v", result.Error)
	}

	trader := fmt.Sprintf("margin_trader_%d", time.Now().UnixNano())

	// Create position
	order := &Order{
		MarketID:  "BTC-USDC",
		Trader:    trader,
		Side:      "buy",
		OrderType: "limit",
		Price:     "50000",
		Quantity:  "0.5",
		Leverage:  "20",
	}

	PlaceOrder(client, order)

	// Check margin ratio
	result = client.GET(fmt.Sprintf("/v1/accounts/%s/margin", trader))
	t.Logf("Margin status: status=%d", result.StatusCode)

	if result.Response != nil {
		t.Logf("Margin data: %s", string(result.Response.Data))
	}
}

// TestLiquidation_PartialLiquidation tests partial liquidation
func TestLiquidation_PartialLiquidation(t *testing.T) {
	config := DefaultConfig()
	client := NewHTTPClient(config)

	// Check server
	result := client.GET("/v1/markets")
	if result.Error != nil {
		t.Skipf("API server not running: %v", result.Error)
	}

	// Create a large position that might be partially liquidated
	trader := fmt.Sprintf("partial_liq_trader_%d", time.Now().UnixNano())

	order := &Order{
		MarketID:  "BTC-USDC",
		Trader:    trader,
		Side:      "buy",
		OrderType: "limit",
		Price:     "50000",
		Quantity:  "2.0",
		Leverage:  "25",
	}

	PlaceOrder(client, order)

	// Check position
	posResult, _ := GetPositions(client, trader)
	t.Logf("Position before potential liquidation: %v", posResult.Response)

	// Attempt partial liquidation
	partialLiqReq := map[string]interface{}{
		"marketId":   "BTC-USDC",
		"trader":     trader,
		"percentage": 50, // Liquidate 50%
	}

	result = client.POST("/v1/liquidations/partial", partialLiqReq)
	t.Logf("Partial liquidation: status=%d", result.StatusCode)
}

// TestLiquidation_LiquidationPriceCalculation tests liquidation price API
func TestLiquidation_LiquidationPriceCalculation(t *testing.T) {
	config := DefaultConfig()
	client := NewHTTPClient(config)

	// Check server
	result := client.GET("/v1/markets")
	if result.Error != nil {
		t.Skipf("API server not running: %v", result.Error)
	}

	// Calculate liquidation price for a hypothetical position
	calcReq := map[string]interface{}{
		"side":       "buy",
		"entryPrice": "50000",
		"quantity":   "1.0",
		"leverage":   "20",
		"margin":     "2500",
	}

	result = client.POST("/v1/calculate/liquidation-price", calcReq)
	t.Logf("Liquidation price calculation: status=%d", result.StatusCode)

	if result.Response != nil && result.StatusCode == http.StatusOK {
		var calcResult map[string]interface{}
		if json.Unmarshal(result.Response.Data, &calcResult) == nil {
			t.Logf("Liquidation price: %v", calcResult)
		}
	}
}

// TestLiquidation_BatchLiquidation tests batch liquidation processing
func TestLiquidation_BatchLiquidation(t *testing.T) {
	config := DefaultConfig()
	client := NewHTTPClient(config)

	// Check server
	result := client.GET("/v1/markets")
	if result.Error != nil {
		t.Skipf("API server not running: %v", result.Error)
	}

	// Create multiple positions
	traders := make([]string, 5)
	for i := 0; i < 5; i++ {
		traders[i] = fmt.Sprintf("batch_liq_trader_%d_%d", time.Now().UnixNano(), i)

		order := &Order{
			MarketID:  "BTC-USDC",
			Trader:    traders[i],
			Side:      "buy",
			OrderType: "limit",
			Price:     "50000",
			Quantity:  "0.5",
			Leverage:  "30",
		}
		PlaceOrder(client, order)
	}

	// Try batch liquidation
	batchReq := map[string]interface{}{
		"marketId": "BTC-USDC",
		"traders":  traders,
	}

	result = client.POST("/v1/liquidations/batch", batchReq)
	t.Logf("Batch liquidation: status=%d, latency=%v", result.StatusCode, result.Latency)
}

// TestLiquidation_WebSocketNotifications tests liquidation notifications via WebSocket
func TestLiquidation_WebSocketNotifications(t *testing.T) {
	config := DefaultConfig()
	wsClient := NewWSClient(config)
	httpClient := NewHTTPClient(config)

	err := wsClient.Connect("/ws")
	if err != nil {
		t.Skipf("WebSocket server not running: %v", err)
	}
	defer wsClient.Close()

	// Subscribe to liquidation channel
	subscribeMsg := map[string]interface{}{
		"type":    "subscribe",
		"channel": "liquidations:BTC-USDC",
	}
	wsClient.Send(subscribeMsg)

	// Create a risky position
	trader := fmt.Sprintf("ws_liq_trader_%d", time.Now().UnixNano())

	order := &Order{
		MarketID:  "BTC-USDC",
		Trader:    trader,
		Side:      "buy",
		OrderType: "limit",
		Price:     "50000",
		Quantity:  "1.0",
		Leverage:  "50",
	}
	PlaceOrder(httpClient, order)

	// Wait for any liquidation notifications
	msg, err := wsClient.Receive(5 * time.Second)
	if err != nil {
		t.Log("No liquidation notifications received (expected if no liquidations triggered)")
		return
	}

	t.Logf("Liquidation notification: %s", string(msg))
}

// TestLiquidation_FundingImpact tests funding rate impact on liquidation
func TestLiquidation_FundingImpact(t *testing.T) {
	config := DefaultConfig()
	client := NewHTTPClient(config)

	// Get current funding rate
	result := client.GET("/v1/markets/BTC-USDC/funding")
	if result.Error != nil {
		t.Skipf("API server not running: %v", result.Error)
	}

	t.Logf("Funding rate: status=%d", result.StatusCode)

	if result.Response != nil {
		t.Logf("Funding data: %s", string(result.Response.Data))
	}
}
