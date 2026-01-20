package e2e_hyperliquid

import (
	"encoding/json"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// TestIntegration_FullTradingFlow tests complete trading data flow
func TestIntegration_FullTradingFlow(t *testing.T) {
	t.Log("=== Full Trading Flow Integration Test ===")
	t.Log("Testing: REST API -> WebSocket -> Data Consistency")

	// Phase 1: REST API - Get market info
	t.Log("\n[Phase 1] REST API - Market Information")
	restClient := NewHyperliquidClient()

	// Get meta
	metaResult := restClient.GetMeta()
	if metaResult.Error != nil {
		t.Fatalf("Failed to get meta: %v", metaResult.Error)
	}
	var meta Meta
	json.Unmarshal(metaResult.Data, &meta)
	t.Logf("  Available markets: %d", len(meta.Universe))

	// Get all mid prices
	midsResult := restClient.GetAllMids()
	if midsResult.Error != nil {
		t.Fatalf("Failed to get mids: %v", midsResult.Error)
	}
	var mids map[string]string
	json.Unmarshal(midsResult.Data, &mids)
	t.Logf("  BTC price: %s", mids["BTC"])
	t.Logf("  ETH price: %s", mids["ETH"])

	// Get order book
	bookResult := restClient.GetL2Book("BTC")
	if bookResult.Error != nil {
		t.Fatalf("Failed to get order book: %v", bookResult.Error)
	}
	var book L2Book
	json.Unmarshal(bookResult.Data, &book)
	if len(book.Levels) >= 2 {
		t.Logf("  BTC order book: %d bids, %d asks", len(book.Levels[0]), len(book.Levels[1]))
	}

	// Get recent trades
	tradesResult := restClient.GetRecentTrades("BTC")
	if tradesResult.Error != nil {
		t.Fatalf("Failed to get trades: %v", tradesResult.Error)
	}
	var trades []Trade
	json.Unmarshal(tradesResult.Data, &trades)
	t.Logf("  Recent BTC trades: %d", len(trades))

	// Phase 2: WebSocket - Real-time updates
	t.Log("\n[Phase 2] WebSocket - Real-time Updates")
	wsClient := NewHyperliquidWSClient()

	err := wsClient.Connect()
	if err != nil {
		t.Fatalf("Failed to connect WebSocket: %v", err)
	}
	defer wsClient.Close()
	t.Log("  WebSocket connected")

	// Subscribe to channels
	wsClient.SubscribeAllMids()
	wsClient.SubscribeL2Book("BTC")
	t.Log("  Subscribed to allMids and BTC L2 book")

	// Receive messages for a while
	wsMessages := 0
	timeout := time.After(10 * time.Second)
	for wsMessages < 10 {
		select {
		case <-timeout:
			break
		default:
			_, err := wsClient.Receive(1 * time.Second)
			if err == nil {
				wsMessages++
			}
		}
		if wsMessages >= 10 {
			break
		}
	}
	t.Logf("  Received %d WebSocket messages", wsMessages)

	// Phase 3: Data Consistency Check
	t.Log("\n[Phase 3] Data Consistency Validation")

	// Fetch fresh REST data
	newMidsResult := restClient.GetAllMids()
	var newMids map[string]string
	json.Unmarshal(newMidsResult.Data, &newMids)
	t.Logf("  Updated BTC price: %s", newMids["BTC"])

	// Compare with initial
	if mids["BTC"] != "" && newMids["BTC"] != "" {
		t.Log("  Both REST calls returned valid BTC prices")
	}

	// Summary
	t.Log("\n=== Integration Test Summary ===")
	stats := restClient.GetLatencyStats()
	t.Logf("REST API calls: %d", stats.Count)
	t.Logf("Average latency: %v", stats.Avg)
	t.Logf("WebSocket messages: %d", wsMessages)
	t.Log("Full trading flow test PASSED")
}

// TestIntegration_MultiAssetMonitoring tests monitoring multiple assets
func TestIntegration_MultiAssetMonitoring(t *testing.T) {
	t.Log("=== Multi-Asset Monitoring Test ===")

	restClient := NewHyperliquidClient()
	wsClient := NewHyperliquidWSClient()

	// Get available assets
	metaResult := restClient.GetMeta()
	var meta Meta
	json.Unmarshal(metaResult.Data, &meta)

	// Select top assets
	assets := []string{"BTC", "ETH", "SOL", "ARB", "DOGE"}
	t.Logf("Monitoring assets: %v", assets)

	// Fetch REST data for all assets
	t.Log("\n[REST] Fetching order books...")
	for _, asset := range assets {
		result := restClient.GetL2Book(asset)
		if result.Error != nil {
			t.Logf("  %s: ERROR - %v", asset, result.Error)
			continue
		}

		var book L2Book
		json.Unmarshal(result.Data, &book)

		if len(book.Levels) >= 2 && len(book.Levels[0]) > 0 && len(book.Levels[1]) > 0 {
			bestBid := book.Levels[0][0].Px
			bestAsk := book.Levels[1][0].Px
			t.Logf("  %s: Bid=%s, Ask=%s (latency=%v)", asset, bestBid, bestAsk, result.Latency)
		}
	}

	// WebSocket monitoring
	t.Log("\n[WebSocket] Subscribing to real-time updates...")
	err := wsClient.Connect()
	if err != nil {
		t.Fatalf("WebSocket connection failed: %v", err)
	}
	defer wsClient.Close()

	for _, asset := range assets {
		wsClient.SubscribeL2Book(asset)
		time.Sleep(50 * time.Millisecond)
	}

	// Count messages per asset
	msgCount := make(map[string]int)
	duration := 15 * time.Second
	endTime := time.Now().Add(duration)

	for time.Now().Before(endTime) {
		msg, err := wsClient.Receive(1 * time.Second)
		if err != nil {
			continue
		}

		var data map[string]interface{}
		json.Unmarshal(msg, &data)

		if dataObj, ok := data["data"].(map[string]interface{}); ok {
			if coin, ok := dataObj["coin"].(string); ok {
				msgCount[coin]++
			}
		}
	}

	t.Log("\n[Summary] Messages received per asset:")
	totalMsg := 0
	for _, asset := range assets {
		count := msgCount[asset]
		totalMsg += count
		t.Logf("  %s: %d messages", asset, count)
	}
	t.Logf("  Total: %d messages in %v", totalMsg, duration)

	stats := restClient.GetLatencyStats()
	stats.PrintStats("Multi-Asset Monitoring")
}

// TestIntegration_HighLoadScenario tests system under high load
func TestIntegration_HighLoadScenario(t *testing.T) {
	t.Log("=== High Load Scenario Test ===")

	restClient := NewHyperliquidClient()

	// Phase 1: Burst REST requests
	t.Log("\n[Phase 1] Burst REST Requests")
	numRequests := 100
	var wg sync.WaitGroup
	var successCount int64
	var errorCount int64

	startTime := time.Now()

	for i := 0; i < numRequests; i++ {
		wg.Add(1)
		go func(reqNum int) {
			defer wg.Done()

			// Alternate between different endpoints
			var result *APIResult
			switch reqNum % 4 {
			case 0:
				result = restClient.GetAllMids()
			case 1:
				result = restClient.GetL2Book("BTC")
			case 2:
				result = restClient.GetL2Book("ETH")
			case 3:
				result = restClient.GetRecentTrades("BTC")
			}

			if result.Error != nil || result.StatusCode != 200 {
				atomic.AddInt64(&errorCount, 1)
			} else {
				atomic.AddInt64(&successCount, 1)
			}
		}(i)
	}

	wg.Wait()
	burstDuration := time.Since(startTime)

	t.Logf("  Requests: %d", numRequests)
	t.Logf("  Success: %d", successCount)
	t.Logf("  Errors: %d", errorCount)
	t.Logf("  Duration: %v", burstDuration)
	t.Logf("  Throughput: %.2f req/sec", float64(numRequests)/burstDuration.Seconds())

	// Phase 2: Sustained WebSocket load
	t.Log("\n[Phase 2] Sustained WebSocket Load")
	numConnections := 3
	var wsWg sync.WaitGroup
	var totalMessages int64

	for i := 0; i < numConnections; i++ {
		wsWg.Add(1)
		go func(connID int) {
			defer wsWg.Done()

			client := NewHyperliquidWSClient()
			if err := client.Connect(); err != nil {
				return
			}
			defer client.Close()

			client.SubscribeAllMids()

			for j := 0; j < 20; j++ {
				_, err := client.Receive(2 * time.Second)
				if err == nil {
					atomic.AddInt64(&totalMessages, 1)
				}
			}
		}(i)
	}

	wsWg.Wait()
	t.Logf("  WebSocket connections: %d", numConnections)
	t.Logf("  Total messages: %d", totalMessages)

	// Summary
	stats := restClient.GetLatencyStats()
	stats.PrintStats("High Load Scenario")

	successRate := float64(successCount) / float64(numRequests) * 100
	t.Logf("\nSuccess Rate: %.1f%%", successRate)

	if successRate < 90 {
		t.Errorf("Success rate %.1f%% is below 90%% threshold", successRate)
	}
}

// TestIntegration_DataFreshness tests data freshness
func TestIntegration_DataFreshness(t *testing.T) {
	t.Log("=== Data Freshness Test ===")

	restClient := NewHyperliquidClient()

	// Track price changes over time
	t.Log("Tracking BTC price changes over 30 seconds...")

	prices := make([]string, 0)
	timestamps := make([]time.Time, 0)

	duration := 30 * time.Second
	interval := 3 * time.Second
	endTime := time.Now().Add(duration)

	for time.Now().Before(endTime) {
		result := restClient.GetAllMids()
		if result.Error == nil {
			var mids map[string]string
			json.Unmarshal(result.Data, &mids)
			if btc, ok := mids["BTC"]; ok {
				prices = append(prices, btc)
				timestamps = append(timestamps, time.Now())
			}
		}
		time.Sleep(interval)
	}

	t.Logf("Collected %d price snapshots", len(prices))

	// Analyze price changes
	changes := 0
	for i := 1; i < len(prices); i++ {
		if prices[i] != prices[i-1] {
			changes++
			t.Logf("  [%v] %s -> %s", timestamps[i].Sub(timestamps[0]).Round(time.Second), prices[i-1], prices[i])
		}
	}

	t.Logf("Price changed %d times in %v", changes, duration)
	t.Log("Data freshness test completed")
}

// TestIntegration_ErrorRecovery tests error recovery
func TestIntegration_ErrorRecovery(t *testing.T) {
	t.Log("=== Error Recovery Test ===")

	// Test 1: Invalid coin handling
	t.Log("\n[Test 1] Invalid Coin Handling")
	restClient := NewHyperliquidClient()

	result := restClient.GetL2Book("INVALID_COIN_XYZ")
	t.Logf("  Invalid coin request: status=%d, error=%v", result.StatusCode, result.Error)

	// Test 2: WebSocket reconnection
	t.Log("\n[Test 2] WebSocket Reconnection")
	wsClient := NewHyperliquidWSClient()

	err := wsClient.Connect()
	if err != nil {
		t.Fatalf("Initial connection failed: %v", err)
	}
	wsClient.SubscribeAllMids()

	// Receive a message
	_, _ = wsClient.Receive(3 * time.Second)
	t.Log("  First connection working")

	// Close and reconnect
	wsClient.Close()
	time.Sleep(500 * time.Millisecond)

	wsClient2 := NewHyperliquidWSClient()
	err = wsClient2.Connect()
	if err != nil {
		t.Fatalf("Reconnection failed: %v", err)
	}
	defer wsClient2.Close()

	wsClient2.SubscribeAllMids()
	_, err = wsClient2.Receive(5 * time.Second)
	if err != nil {
		t.Errorf("Failed to receive after reconnect: %v", err)
	} else {
		t.Log("  Reconnection successful")
	}

	// Test 3: Continued operation after errors
	t.Log("\n[Test 3] Continued Operation After Errors")
	validResult := restClient.GetAllMids()
	if validResult.Error != nil {
		t.Errorf("Valid request failed after error: %v", validResult.Error)
	} else {
		t.Log("  System continues to operate normally")
	}

	t.Log("\nError recovery test completed")
}

// TestIntegration_EndToEnd tests complete end-to-end flow
func TestIntegration_EndToEnd(t *testing.T) {
	t.Log("============================================")
	t.Log("    HYPERLIQUID E2E INTEGRATION TEST")
	t.Log("============================================")
	t.Log("NO MOCK - Real Hyperliquid Mainnet API")
	t.Log("")

	startTime := time.Now()

	// Step 1: Verify connectivity
	t.Log("[1/6] Verifying API Connectivity...")
	restClient := NewHyperliquidClient()

	metaResult := restClient.GetMeta()
	if metaResult.Error != nil {
		t.Fatalf("API not reachable: %v", metaResult.Error)
	}
	t.Logf("      REST API: OK (latency=%v)", metaResult.Latency)

	wsClient := NewHyperliquidWSClient()
	err := wsClient.Connect()
	if err != nil {
		t.Fatalf("WebSocket not reachable: %v", err)
	}
	defer wsClient.Close()
	t.Log("      WebSocket: OK")

	// Step 2: Fetch market data
	t.Log("\n[2/6] Fetching Market Data...")
	var meta Meta
	json.Unmarshal(metaResult.Data, &meta)
	t.Logf("      Available markets: %d", len(meta.Universe))

	midsResult := restClient.GetAllMids()
	var mids map[string]string
	json.Unmarshal(midsResult.Data, &mids)
	t.Logf("      BTC: %s", mids["BTC"])
	t.Logf("      ETH: %s", mids["ETH"])

	// Step 3: Fetch order book depth
	t.Log("\n[3/6] Fetching Order Book Depth...")
	bookResult := restClient.GetL2Book("BTC")
	var book L2Book
	json.Unmarshal(bookResult.Data, &book)

	if len(book.Levels) >= 2 {
		bids := book.Levels[0]
		asks := book.Levels[1]
		t.Logf("      BTC Bids: %d levels", len(bids))
		t.Logf("      BTC Asks: %d levels", len(asks))
		if len(bids) > 0 && len(asks) > 0 {
			t.Logf("      Spread: %s - %s", bids[0].Px, asks[0].Px)
		}
	}

	// Step 4: Fetch recent trades
	t.Log("\n[4/6] Fetching Recent Trades...")
	tradesResult := restClient.GetRecentTrades("BTC")
	var trades []Trade
	json.Unmarshal(tradesResult.Data, &trades)
	t.Logf("      Recent trades: %d", len(trades))
	if len(trades) > 0 {
		t.Logf("      Latest: %s %s @ %s", trades[0].Side, trades[0].Sz, trades[0].Px)
	}

	// Step 5: Test WebSocket streaming
	t.Log("\n[5/6] Testing WebSocket Streaming...")
	wsClient.SubscribeAllMids()
	wsClient.SubscribeL2Book("BTC")

	msgCount := 0
	timeout := time.After(10 * time.Second)
	for msgCount < 10 {
		select {
		case <-timeout:
			break
		default:
			_, err := wsClient.Receive(1 * time.Second)
			if err == nil {
				msgCount++
			}
		}
		if msgCount >= 10 {
			break
		}
	}
	t.Logf("      WebSocket messages: %d", msgCount)

	// Step 6: Performance summary
	t.Log("\n[6/6] Performance Summary...")
	stats := restClient.GetLatencyStats()
	t.Logf("      REST requests: %d", stats.Count)
	t.Logf("      Avg latency: %v", stats.Avg)
	t.Logf("      P95 latency: %v", stats.P95)
	t.Logf("      P99 latency: %v", stats.P99)

	elapsed := time.Since(startTime)

	t.Log("\n============================================")
	t.Log("    E2E TEST COMPLETED SUCCESSFULLY")
	t.Logf("    Total Duration: %v", elapsed)
	t.Log("============================================")
}

// BenchmarkHyperliquid_REST benchmarks REST API performance
func BenchmarkHyperliquid_REST(b *testing.B) {
	client := NewHyperliquidClient()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		client.GetAllMids()
	}
}

// BenchmarkHyperliquid_L2Book benchmarks L2 book fetching
func BenchmarkHyperliquid_L2Book(b *testing.B) {
	client := NewHyperliquidClient()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		client.GetL2Book("BTC")
	}
}
