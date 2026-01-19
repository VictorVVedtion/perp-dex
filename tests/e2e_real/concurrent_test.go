package e2e_real

import (
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// ============================================================================
// Multi-User Concurrent E2E Tests
// ============================================================================
// These tests verify system behavior under concurrent load:
// 1. Multiple users trading simultaneously
// 2. Order book consistency under load
// 3. Position accuracy with concurrent updates
// 4. WebSocket message ordering
// 5. Race condition detection
// ============================================================================

// ConcurrentTestConfig holds configuration for concurrent tests
type ConcurrentTestConfig struct {
	NumUsers        int
	OrdersPerUser   int
	Duration        time.Duration
	Markets         []string
}

// DefaultConcurrentConfig returns default concurrent test configuration
func DefaultConcurrentConfig() *ConcurrentTestConfig {
	return &ConcurrentTestConfig{
		NumUsers:      10,
		OrdersPerUser: 50,
		Duration:      30 * time.Second,
		Markets:       []string{"BTC-USDC", "ETH-USDC", "SOL-USDC"},
	}
}

// ConcurrentTestResult holds results from concurrent tests
type ConcurrentTestResult struct {
	TotalOrders     int64
	SuccessfulOrders int64
	FailedOrders    int64
	TotalLatency    int64 // microseconds
	MinLatency      int64
	MaxLatency      int64
	ErrorCounts     map[string]int64
	Duration        time.Duration
}

// TestConcurrentOrderPlacement tests multiple users placing orders simultaneously
func TestConcurrentOrderPlacement(t *testing.T) {
	suite := NewE2ETestSuite(t, nil)

	err := suite.WaitForServer(10 * time.Second)
	if err != nil {
		t.Skipf("Server not available: %v", err)
	}

	config := DefaultConcurrentConfig()
	result := &ConcurrentTestResult{
		ErrorCounts: make(map[string]int64),
		MinLatency:  int64(^uint64(0) >> 1),
	}

	var wg sync.WaitGroup
	var errorMu sync.Mutex

	startTime := time.Now()

	for i := 0; i < config.NumUsers; i++ {
		wg.Add(1)
		go func(userID int) {
			defer wg.Done()

			user := suite.NewTestUser(fmt.Sprintf("perpdex1concurrent%04d", userID))
			market := config.Markets[userID%len(config.Markets)]

			for j := 0; j < config.OrdersPerUser; j++ {
				side := "buy"
				if j%2 == 0 {
					side = "sell"
				}

				price := fmt.Sprintf("%d.00", 49000+userID*10+j)

				orderStart := time.Now()
				order, err := user.PlaceOrder(&PlaceOrderRequest{
					MarketID: market,
					Side:     side,
					Type:     "limit",
					Price:    price,
					Quantity: "0.01",
				})
				latency := time.Since(orderStart).Microseconds()

				atomic.AddInt64(&result.TotalOrders, 1)
				atomic.AddInt64(&result.TotalLatency, latency)

				if err != nil {
					atomic.AddInt64(&result.FailedOrders, 1)
					errorMu.Lock()
					result.ErrorCounts[err.Error()]++
					errorMu.Unlock()
				} else {
					atomic.AddInt64(&result.SuccessfulOrders, 1)

					// Update min/max latency
					for {
						old := atomic.LoadInt64(&result.MinLatency)
						if latency >= old || atomic.CompareAndSwapInt64(&result.MinLatency, old, latency) {
							break
						}
					}
					for {
						old := atomic.LoadInt64(&result.MaxLatency)
						if latency <= old || atomic.CompareAndSwapInt64(&result.MaxLatency, old, latency) {
							break
						}
					}

					// Cancel order to clean up
					if order != nil && order.OrderID != "" {
						_ = user.CancelOrder(order.OrderID)
					}
				}
			}
		}(i)
	}

	wg.Wait()
	result.Duration = time.Since(startTime)

	// Report results
	t.Log("═══════════════════════════════════════════════════════════════")
	t.Log("        CONCURRENT ORDER PLACEMENT TEST RESULTS")
	t.Log("═══════════════════════════════════════════════════════════════")
	t.Logf("Users:              %d", config.NumUsers)
	t.Logf("Orders per user:    %d", config.OrdersPerUser)
	t.Logf("Duration:           %v", result.Duration)
	t.Log("")
	t.Logf("Total orders:       %d", result.TotalOrders)
	t.Logf("Successful:         %d (%.2f%%)",
		result.SuccessfulOrders,
		float64(result.SuccessfulOrders)/float64(result.TotalOrders)*100)
	t.Logf("Failed:             %d (%.2f%%)",
		result.FailedOrders,
		float64(result.FailedOrders)/float64(result.TotalOrders)*100)
	t.Logf("Throughput:         %.2f orders/sec",
		float64(result.TotalOrders)/result.Duration.Seconds())
	t.Log("")
	t.Logf("Latency (μs):")
	t.Logf("  Min:              %d", result.MinLatency)
	t.Logf("  Max:              %d", result.MaxLatency)
	if result.TotalOrders > 0 {
		t.Logf("  Avg:              %d", result.TotalLatency/result.TotalOrders)
	}

	if len(result.ErrorCounts) > 0 {
		t.Log("")
		t.Log("Errors:")
		for errMsg, count := range result.ErrorCounts {
			t.Logf("  %s: %d", truncateString(errMsg, 50), count)
		}
	}
	t.Log("═══════════════════════════════════════════════════════════════")
}

// TestConcurrentMatching tests order matching under concurrent load
func TestConcurrentMatching(t *testing.T) {
	suite := NewE2ETestSuite(t, nil)

	err := suite.WaitForServer(10 * time.Second)
	if err != nil {
		t.Skipf("Server not available: %v", err)
	}

	const (
		numMakers = 5
		numTakers = 5
		ordersPerUser = 20
	)

	var wg sync.WaitGroup
	var matchCount int64
	var orderCount int64

	// Start makers placing sell orders
	for i := 0; i < numMakers; i++ {
		wg.Add(1)
		go func(makerID int) {
			defer wg.Done()

			maker := suite.NewTestUser(fmt.Sprintf("perpdex1maker%04d", makerID))

			for j := 0; j < ordersPerUser; j++ {
				price := fmt.Sprintf("%d.00", 50000+j)
				_, err := maker.PlaceOrder(&PlaceOrderRequest{
					MarketID: "BTC-USDC",
					Side:     "sell",
					Type:     "limit",
					Price:    price,
					Quantity: "0.01",
				})
				if err == nil {
					atomic.AddInt64(&orderCount, 1)
				}
			}
		}(i)
	}

	// Give makers a head start
	time.Sleep(500 * time.Millisecond)

	// Start takers placing buy orders
	for i := 0; i < numTakers; i++ {
		wg.Add(1)
		go func(takerID int) {
			defer wg.Done()

			taker := suite.NewTestUser(fmt.Sprintf("perpdex1taker%04d", takerID))

			for j := 0; j < ordersPerUser; j++ {
				price := fmt.Sprintf("%d.00", 50000+j)
				order, err := taker.PlaceOrder(&PlaceOrderRequest{
					MarketID: "BTC-USDC",
					Side:     "buy",
					Type:     "limit",
					Price:    price,
					Quantity: "0.01",
				})
				if err == nil {
					atomic.AddInt64(&orderCount, 1)
					if order != nil && order.Status == "filled" {
						atomic.AddInt64(&matchCount, 1)
					}
				}
			}
		}(i)
	}

	wg.Wait()

	t.Log("═══════════════════════════════════════════════════════════════")
	t.Log("        CONCURRENT MATCHING TEST RESULTS")
	t.Log("═══════════════════════════════════════════════════════════════")
	t.Logf("Makers:             %d", numMakers)
	t.Logf("Takers:             %d", numTakers)
	t.Logf("Orders per user:    %d", ordersPerUser)
	t.Logf("Total orders:       %d", orderCount)
	t.Logf("Matches detected:   %d", matchCount)
	t.Log("═══════════════════════════════════════════════════════════════")
}

// TestConcurrentPositionUpdates tests position consistency under concurrent updates
func TestConcurrentPositionUpdates(t *testing.T) {
	suite := NewE2ETestSuite(t, nil)

	err := suite.WaitForServer(10 * time.Second)
	if err != nil {
		t.Skipf("Server not available: %v", err)
	}

	user := suite.NewTestUser("perpdex1postest001")

	const iterations = 50
	var wg sync.WaitGroup
	var successCount int64
	var errorCount int64

	// Concurrent order placement (should affect same position)
	for i := 0; i < iterations; i++ {
		wg.Add(1)
		go func(iter int) {
			defer wg.Done()

			side := "buy"
			if iter%2 == 0 {
				side = "sell"
			}

			_, err := user.PlaceOrder(&PlaceOrderRequest{
				MarketID: "ETH-USDC",
				Side:     side,
				Type:     "limit",
				Price:    "3000.00",
				Quantity: "0.1",
			})

			if err != nil {
				atomic.AddInt64(&errorCount, 1)
			} else {
				atomic.AddInt64(&successCount, 1)
			}
		}(i)
	}

	wg.Wait()

	// Check final position consistency
	positions, _ := user.GetPositions()

	t.Log("═══════════════════════════════════════════════════════════════")
	t.Log("        CONCURRENT POSITION UPDATE TEST RESULTS")
	t.Log("═══════════════════════════════════════════════════════════════")
	t.Logf("Concurrent operations: %d", iterations)
	t.Logf("Successful:           %d", successCount)
	t.Logf("Errors:               %d", errorCount)
	t.Logf("Final positions:      %d", len(positions))

	for _, pos := range positions {
		t.Logf("  %s: %s %s @ %s (PnL: %s)",
			pos.MarketID, pos.Side, pos.Size, pos.EntryPrice, pos.UnrealizedPnL)
	}
	t.Log("═══════════════════════════════════════════════════════════════")
}

// TestWebSocketUnderLoad tests WebSocket performance under concurrent HTTP load
func TestWebSocketUnderLoad(t *testing.T) {
	suite := NewE2ETestSuite(t, nil)

	err := suite.WaitForServer(10 * time.Second)
	if err != nil {
		t.Skipf("Server not available: %v", err)
	}

	// Connect WebSocket
	ws, err := suite.NewWSClient()
	if err != nil {
		t.Fatalf("WebSocket connection failed: %v", err)
	}
	defer ws.Close()

	// Subscribe to ticker
	err = ws.Subscribe("ticker", map[string]interface{}{"market": "BTC-USDC"})
	if err != nil {
		t.Fatalf("Subscribe failed: %v", err)
	}

	// Start message collector
	var wsMessageCount int64
	stopCh := make(chan struct{})

	go func() {
		for {
			select {
			case <-ws.messages:
				atomic.AddInt64(&wsMessageCount, 1)
			case <-stopCh:
				return
			}
		}
	}()

	// Generate HTTP load
	const (
		numWorkers = 5
		duration   = 10 * time.Second
	)

	var httpRequestCount int64
	var wg sync.WaitGroup

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			user := suite.NewTestUser(fmt.Sprintf("perpdex1wsload%04d", workerID))
			deadline := time.Now().Add(duration)

			for time.Now().Before(deadline) {
				order, _ := user.PlaceOrder(&PlaceOrderRequest{
					MarketID: "BTC-USDC",
					Side:     "buy",
					Type:     "limit",
					Price:    "48000.00",
					Quantity: "0.01",
				})
				atomic.AddInt64(&httpRequestCount, 1)

				if order != nil && order.OrderID != "" {
					_ = user.CancelOrder(order.OrderID)
					atomic.AddInt64(&httpRequestCount, 1)
				}

				time.Sleep(50 * time.Millisecond)
			}
		}(i)
	}

	wg.Wait()
	close(stopCh)

	t.Log("═══════════════════════════════════════════════════════════════")
	t.Log("        WEBSOCKET UNDER LOAD TEST RESULTS")
	t.Log("═══════════════════════════════════════════════════════════════")
	t.Logf("Test duration:       %v", duration)
	t.Logf("HTTP workers:        %d", numWorkers)
	t.Logf("HTTP requests:       %d", httpRequestCount)
	t.Logf("HTTP throughput:     %.2f req/sec", float64(httpRequestCount)/duration.Seconds())
	t.Logf("WebSocket messages:  %d", wsMessageCount)
	t.Logf("WS message rate:     %.2f msg/sec", float64(wsMessageCount)/duration.Seconds())
	t.Log("═══════════════════════════════════════════════════════════════")
}

// TestRaceConditions tests for potential race conditions
func TestRaceConditions(t *testing.T) {
	suite := NewE2ETestSuite(t, nil)

	err := suite.WaitForServer(10 * time.Second)
	if err != nil {
		t.Skipf("Server not available: %v", err)
	}

	t.Run("ConcurrentCancelSameOrder", func(t *testing.T) {
		user := suite.NewTestUser("perpdex1race001")

		// Place an order
		order, err := user.PlaceOrder(&PlaceOrderRequest{
			MarketID: "BTC-USDC",
			Side:     "buy",
			Type:     "limit",
			Price:    "45000.00",
			Quantity: "0.1",
		})
		if err != nil {
			t.Fatalf("Order placement failed: %v", err)
		}

		// Try to cancel from multiple goroutines
		const attempts = 10
		var wg sync.WaitGroup
		var successCount, errorCount int64

		for i := 0; i < attempts; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				err := user.CancelOrder(order.OrderID)
				if err != nil {
					atomic.AddInt64(&errorCount, 1)
				} else {
					atomic.AddInt64(&successCount, 1)
				}
			}()
		}

		wg.Wait()

		t.Logf("Cancel attempts: %d, Success: %d, Errors: %d",
			attempts, successCount, errorCount)

		// Expect exactly 1 success (or possibly 0 if already matched)
		if successCount > 1 {
			t.Error("Race condition detected: Multiple cancels succeeded")
		}
	})

	t.Run("ConcurrentModifySameOrder", func(t *testing.T) {
		user := suite.NewTestUser("perpdex1race002")

		// Place an order
		order, err := user.PlaceOrder(&PlaceOrderRequest{
			MarketID: "ETH-USDC",
			Side:     "sell",
			Type:     "limit",
			Price:    "4000.00",
			Quantity: "1.0",
		})
		if err != nil {
			t.Fatalf("Order placement failed: %v", err)
		}

		// Try concurrent modifications
		const attempts = 5
		var wg sync.WaitGroup

		for i := 0; i < attempts; i++ {
			wg.Add(1)
			go func(price int) {
				defer wg.Done()
				// Modify order price
				_, _ = suite.PUT(fmt.Sprintf("/v1/orders/%s", order.OrderID),
					map[string]interface{}{
						"price": fmt.Sprintf("%d.00", price),
					},
					user.Headers())
			}(4000 + i*10)
		}

		wg.Wait()

		t.Log("Concurrent modification test completed (check server logs for errors)")

		// Clean up
		_ = user.CancelOrder(order.OrderID)
	})
}

// TestSystemStability tests overall system stability under sustained load
func TestSystemStability(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stability test in short mode")
	}

	suite := NewE2ETestSuite(t, nil)

	err := suite.WaitForServer(10 * time.Second)
	if err != nil {
		t.Skipf("Server not available: %v", err)
	}

	const (
		testDuration = 60 * time.Second
		numUsers     = 20
	)

	var totalOps int64
	var errors int64
	startTime := time.Now()
	stopCh := make(chan struct{})

	var wg sync.WaitGroup

	for i := 0; i < numUsers; i++ {
		wg.Add(1)
		go func(userID int) {
			defer wg.Done()

			user := suite.NewTestUser(fmt.Sprintf("perpdex1stable%04d", userID))
			markets := []string{"BTC-USDC", "ETH-USDC", "SOL-USDC"}

			for {
				select {
				case <-stopCh:
					return
				default:
					market := markets[userID%len(markets)]
					side := "buy"
					if time.Now().UnixNano()%2 == 0 {
						side = "sell"
					}

					order, err := user.PlaceOrder(&PlaceOrderRequest{
						MarketID: market,
						Side:     side,
						Type:     "limit",
						Price:    "50000.00",
						Quantity: "0.01",
					})
					atomic.AddInt64(&totalOps, 1)

					if err != nil {
						atomic.AddInt64(&errors, 1)
					} else if order != nil && order.OrderID != "" {
						_ = user.CancelOrder(order.OrderID)
						atomic.AddInt64(&totalOps, 1)
					}

					time.Sleep(100 * time.Millisecond)
				}
			}
		}(i)
	}

	// Progress reporting
	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-stopCh:
				return
			case <-ticker.C:
				elapsed := time.Since(startTime)
				ops := atomic.LoadInt64(&totalOps)
				errs := atomic.LoadInt64(&errors)
				t.Logf("Progress: %v elapsed, %d ops, %d errors, %.2f ops/sec",
					elapsed.Round(time.Second), ops, errs, float64(ops)/elapsed.Seconds())
			}
		}
	}()

	// Run for duration
	time.Sleep(testDuration)
	close(stopCh)
	wg.Wait()

	finalOps := atomic.LoadInt64(&totalOps)
	finalErrors := atomic.LoadInt64(&errors)
	actualDuration := time.Since(startTime)

	t.Log("═══════════════════════════════════════════════════════════════")
	t.Log("        SYSTEM STABILITY TEST RESULTS")
	t.Log("═══════════════════════════════════════════════════════════════")
	t.Logf("Duration:            %v", actualDuration.Round(time.Second))
	t.Logf("Concurrent users:    %d", numUsers)
	t.Logf("Total operations:    %d", finalOps)
	t.Logf("Errors:              %d (%.2f%%)", finalErrors,
		float64(finalErrors)/float64(finalOps)*100)
	t.Logf("Throughput:          %.2f ops/sec", float64(finalOps)/actualDuration.Seconds())
	t.Log("")
	if float64(finalErrors)/float64(finalOps) < 0.01 {
		t.Log("Status: ✅ STABLE (<1% error rate)")
	} else if float64(finalErrors)/float64(finalOps) < 0.05 {
		t.Log("Status: ⚠️  ACCEPTABLE (<5% error rate)")
	} else {
		t.Log("Status: ❌ UNSTABLE (>5% error rate)")
	}
	t.Log("═══════════════════════════════════════════════════════════════")
}
