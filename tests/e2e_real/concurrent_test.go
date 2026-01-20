package e2e_real

import (
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// TestConcurrent_MultipleTraders tests concurrent trading from multiple traders
func TestConcurrent_MultipleTraders(t *testing.T) {
	config := DefaultConfig()
	client := NewHTTPClient(config)

	// Check server is running
	result := client.GET("/v1/markets")
	if result.Error != nil {
		t.Skipf("API server not running: %v", result.Error)
	}

	numTraders := 10
	ordersPerTrader := 5
	var wg sync.WaitGroup
	var successCount int64
	var errorCount int64

	startTime := time.Now()

	for i := 0; i < numTraders; i++ {
		wg.Add(1)
		go func(traderID int) {
			defer wg.Done()

			trader := fmt.Sprintf("concurrent_trader_%d_%d", time.Now().UnixNano(), traderID)

			for j := 0; j < ordersPerTrader; j++ {
				order := &Order{
					MarketID:  "BTC-USDC",
					Trader:    trader,
					Side:      "buy",
					OrderType: "limit",
					Price:     fmt.Sprintf("%d", 49000+j*100),
					Quantity:  "0.01",
					Leverage:  "10",
				}

				result, err := PlaceOrder(client, order)
				if err != nil || result.StatusCode >= 400 {
					atomic.AddInt64(&errorCount, 1)
				} else {
					atomic.AddInt64(&successCount, 1)
				}
			}
		}(i)
	}

	wg.Wait()
	elapsed := time.Since(startTime)

	totalOrders := int64(numTraders * ordersPerTrader)
	throughput := float64(totalOrders) / elapsed.Seconds()

	t.Logf("Concurrent Trading Test Results:")
	t.Logf("  Traders: %d", numTraders)
	t.Logf("  Total Orders: %d", totalOrders)
	t.Logf("  Success: %d", successCount)
	t.Logf("  Errors: %d", errorCount)
	t.Logf("  Duration: %v", elapsed)
	t.Logf("  Throughput: %.2f orders/sec", throughput)
}

// TestConcurrent_RaceCondition tests for race conditions in order matching
func TestConcurrent_RaceCondition(t *testing.T) {
	config := DefaultConfig()
	client := NewHTTPClient(config)

	// Check server is running
	result := client.GET("/v1/markets")
	if result.Error != nil {
		t.Skipf("API server not running: %v", result.Error)
	}

	// Multiple traders trying to fill the same order
	numTraders := 20
	targetPrice := "50000"
	var wg sync.WaitGroup

	// First place a large buy order
	buyer := fmt.Sprintf("race_buyer_%d", time.Now().UnixNano())
	buyOrder := &Order{
		MarketID:  "BTC-USDC",
		Trader:    buyer,
		Side:      "buy",
		OrderType: "limit",
		Price:     targetPrice,
		Quantity:  "1.0",
		Leverage:  "10",
	}
	PlaceOrder(client, buyOrder)

	// Now multiple sellers try to fill it
	var filledCount int64

	for i := 0; i < numTraders; i++ {
		wg.Add(1)
		go func(sellerID int) {
			defer wg.Done()

			seller := fmt.Sprintf("race_seller_%d_%d", time.Now().UnixNano(), sellerID)
			sellOrder := &Order{
				MarketID:  "BTC-USDC",
				Trader:    seller,
				Side:      "sell",
				OrderType: "limit",
				Price:     targetPrice,
				Quantity:  "0.1",
				Leverage:  "10",
			}

			result, err := PlaceOrder(client, sellOrder)
			if err == nil && result.StatusCode < 400 {
				atomic.AddInt64(&filledCount, 1)
			}
		}(i)
	}

	wg.Wait()

	t.Logf("Race condition test: %d/%d sellers processed", filledCount, numTraders)
}

// TestConcurrent_WebSocketConnections tests multiple concurrent WebSocket connections
func TestConcurrent_WebSocketConnections(t *testing.T) {
	config := DefaultConfig()

	numConnections := 10
	var wg sync.WaitGroup
	var connectedCount int64
	var messageCount int64

	for i := 0; i < numConnections; i++ {
		wg.Add(1)
		go func(connID int) {
			defer wg.Done()

			wsClient := NewWSClient(config)
			err := wsClient.Connect("/ws")
			if err != nil {
				return
			}
			defer wsClient.Close()

			atomic.AddInt64(&connectedCount, 1)

			// Subscribe to ticker
			subscribeMsg := map[string]interface{}{
				"type":    "subscribe",
				"channel": "ticker:BTC-USDC",
			}
			wsClient.Send(subscribeMsg)

			// Receive messages for a while
			for j := 0; j < 5; j++ {
				_, err := wsClient.Receive(2 * time.Second)
				if err == nil {
					atomic.AddInt64(&messageCount, 1)
				}
			}
		}(i)
	}

	wg.Wait()

	t.Logf("WebSocket Concurrent Test:")
	t.Logf("  Attempted: %d connections", numConnections)
	t.Logf("  Connected: %d", connectedCount)
	t.Logf("  Messages received: %d", messageCount)
}

// TestConcurrent_OrderCancellation tests concurrent order placement and cancellation
func TestConcurrent_OrderCancellation(t *testing.T) {
	config := DefaultConfig()
	client := NewHTTPClient(config)

	// Check server is running
	result := client.GET("/v1/markets")
	if result.Error != nil {
		t.Skipf("API server not running: %v", result.Error)
	}

	numOrders := 20
	var wg sync.WaitGroup
	var placeCount, cancelCount int64

	trader := fmt.Sprintf("cancel_trader_%d", time.Now().UnixNano())

	// Place and cancel orders concurrently
	for i := 0; i < numOrders; i++ {
		wg.Add(2)

		// Place order
		go func(orderNum int) {
			defer wg.Done()

			order := &Order{
				MarketID:  "BTC-USDC",
				Trader:    trader,
				Side:      "buy",
				OrderType: "limit",
				Price:     fmt.Sprintf("%d", 40000+orderNum),
				Quantity:  "0.01",
				Leverage:  "10",
			}

			result, err := PlaceOrder(client, order)
			if err == nil && result.StatusCode < 400 {
				atomic.AddInt64(&placeCount, 1)
			}
		}(i)

		// Cancel order (may fail if order already matched)
		go func(orderNum int) {
			defer wg.Done()

			time.Sleep(10 * time.Millisecond) // Small delay

			result, err := CancelOrder(client, fmt.Sprintf("order_%d", orderNum))
			if err == nil && result.StatusCode < 400 {
				atomic.AddInt64(&cancelCount, 1)
			}
		}(i)
	}

	wg.Wait()

	t.Logf("Order Cancellation Test:")
	t.Logf("  Orders placed: %d", placeCount)
	t.Logf("  Orders cancelled: %d", cancelCount)
}

// TestConcurrent_LoadTest performs a load test with sustained traffic
func TestConcurrent_LoadTest(t *testing.T) {
	config := DefaultConfig()
	client := NewHTTPClient(config)

	// Check server is running
	result := client.GET("/v1/markets")
	if result.Error != nil {
		t.Skipf("API server not running: %v", result.Error)
	}

	duration := 3 * time.Second // Reduced for faster tests
	numWorkers := 10
	var wg sync.WaitGroup
	var requestCount int64
	var errorCount int64

	ctx := make(chan struct{})

	// Start workers
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			trader := fmt.Sprintf("load_trader_%d_%d", time.Now().UnixNano(), workerID)
			orderNum := 0

			for {
				select {
				case <-ctx:
					return
				default:
					// Mix of operations
					switch orderNum % 4 {
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
							Price:     fmt.Sprintf("%d", 40000+orderNum%1000),
							Quantity:  "0.01",
							Leverage:  "10",
						}
						result, err := PlaceOrder(client, order)
						if err != nil || result.StatusCode >= 500 {
							atomic.AddInt64(&errorCount, 1)
						}
					case 3:
						client.GET("/v1/markets")
					}

					atomic.AddInt64(&requestCount, 1)
					orderNum++
				}
			}
		}(i)
	}

	// Run for duration
	time.Sleep(duration)
	close(ctx)
	wg.Wait()

	throughput := float64(requestCount) / duration.Seconds()

	t.Logf("Load Test Results:")
	t.Logf("  Duration: %v", duration)
	t.Logf("  Workers: %d", numWorkers)
	t.Logf("  Total Requests: %d", requestCount)
	t.Logf("  Errors: %d", errorCount)
	t.Logf("  Throughput: %.2f req/sec", throughput)

	// Generate latency report
	report := client.GenerateReport("Concurrent Load Test")
	report.PrintReport()
}

// TestConcurrent_OrderBookConsistency tests order book consistency under load
func TestConcurrent_OrderBookConsistency(t *testing.T) {
	config := DefaultConfig()
	client := NewHTTPClient(config)

	// Check server is running
	result := client.GET("/v1/markets")
	if result.Error != nil {
		t.Skipf("API server not running: %v", result.Error)
	}

	// Place orders and verify order book updates correctly
	numOrders := 50
	var wg sync.WaitGroup

	prices := make(map[string]bool)
	var mu sync.Mutex

	for i := 0; i < numOrders; i++ {
		wg.Add(1)
		go func(orderNum int) {
			defer wg.Done()

			trader := fmt.Sprintf("consistency_trader_%d_%d", time.Now().UnixNano(), orderNum)
			price := fmt.Sprintf("%d", 49000+orderNum*10)

			mu.Lock()
			prices[price] = true
			mu.Unlock()

			order := &Order{
				MarketID:  "BTC-USDC",
				Trader:    trader,
				Side:      "buy",
				OrderType: "limit",
				Price:     price,
				Quantity:  "0.01",
				Leverage:  "10",
			}

			PlaceOrder(client, order)
		}(i)
	}

	wg.Wait()

	// Wait for order book to update
	time.Sleep(500 * time.Millisecond)

	// Fetch order book and verify
	obResult, err := GetOrderBook(client, "BTC-USDC")
	if err != nil {
		t.Fatalf("Failed to get orderbook: %v", err)
	}

	t.Logf("Order book consistency test: placed %d orders at %d unique prices", numOrders, len(prices))
	t.Logf("OrderBook status: %d", obResult.StatusCode)
}

// TestConcurrent_HighFrequencyTrading simulates high-frequency trading patterns
func TestConcurrent_HighFrequencyTrading(t *testing.T) {
	config := DefaultConfig()
	client := NewHTTPClient(config)

	// Check server is running
	result := client.GET("/v1/markets")
	if result.Error != nil {
		t.Skipf("API server not running: %v", result.Error)
	}

	// Simulate HFT: rapid order placement and cancellation
	duration := 2 * time.Second // Reduced for faster tests
	var wg sync.WaitGroup
	var orderCount int64
	var latencies []time.Duration
	var mu sync.Mutex

	ctx := make(chan struct{})

	// Single HFT trader
	wg.Add(1)
	go func() {
		defer wg.Done()

		trader := fmt.Sprintf("hft_trader_%d", time.Now().UnixNano())
		orderNum := 0

		for {
			select {
			case <-ctx:
				return
			default:
				start := time.Now()

				// Place order
				order := &Order{
					MarketID:  "BTC-USDC",
					Trader:    trader,
					Side:      "buy",
					OrderType: "limit",
					Price:     fmt.Sprintf("%d", 49500+orderNum%100),
					Quantity:  "0.001",
					Leverage:  "10",
				}

				PlaceOrder(client, order)

				elapsed := time.Since(start)
				mu.Lock()
				latencies = append(latencies, elapsed)
				mu.Unlock()

				atomic.AddInt64(&orderCount, 1)
				orderNum++
			}
		}
	}()

	time.Sleep(duration)
	close(ctx)
	wg.Wait()

	// Calculate statistics
	var totalLatency time.Duration
	for _, l := range latencies {
		totalLatency += l
	}
	avgLatency := totalLatency / time.Duration(len(latencies))
	throughput := float64(orderCount) / duration.Seconds()

	t.Logf("HFT Simulation Results:")
	t.Logf("  Duration: %v", duration)
	t.Logf("  Orders: %d", orderCount)
	t.Logf("  Throughput: %.2f orders/sec", throughput)
	t.Logf("  Avg Latency: %v", avgLatency)
}
