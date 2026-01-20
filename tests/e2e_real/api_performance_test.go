// Package e2e_real provides real end-to-end testing infrastructure
// API performance tests for REST endpoints and WebSocket connections
package e2e_real

import (
	"fmt"
	"sort"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// ============ API Latency Baseline Tests ============

// TestAPI_LatencyBaseline measures baseline latency for each API endpoint
func TestAPI_LatencyBaseline(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping API performance test in short mode")
	}

	config := DefaultConfig()
	client := NewHTTPClient(config)

	endpoints := []struct {
		name     string
		method   string
		path     string
		body     interface{}
		warmup   int // warmup requests before measurement
		count    int // measurement requests
	}{
		{"GET /v1/health", "GET", "/v1/health", nil, 10, 100},
		{"GET /v1/markets", "GET", "/v1/markets", nil, 10, 100},
		{"GET /v1/markets/BTC-USDC/orderbook", "GET", "/v1/markets/BTC-USDC/orderbook", nil, 10, 100},
		{"GET /v1/markets/BTC-USDC/trades", "GET", "/v1/markets/BTC-USDC/trades", nil, 10, 100},
		{"GET /v1/markets/BTC-USDC/ticker", "GET", "/v1/markets/BTC-USDC/ticker", nil, 10, 100},
		{"POST /v1/orders (limit)", "POST", "/v1/orders", &Order{
			MarketID:  "BTC-USDC",
			Trader:    "perf-test-trader",
			Side:      "buy",
			OrderType: "limit",
			Price:     "50000",
			Quantity:  "0.01",
			Leverage:  "10",
		}, 5, 50},
	}

	t.Logf("\n╔══════════════════════════════════════════════════════════════╗")
	t.Logf("║  API Endpoint Latency Baseline                               ║")
	t.Logf("╠══════════════════════════════════════════════════════════════╣")

	for _, ep := range endpoints {
		latencies := make([]time.Duration, 0, ep.count)

		// Warmup
		for i := 0; i < ep.warmup; i++ {
			if ep.method == "GET" {
				client.GET(ep.path)
			} else {
				client.POST(ep.path, ep.body)
			}
		}

		// Measurement
		for i := 0; i < ep.count; i++ {
			var result *RequestResult
			if ep.method == "GET" {
				result = client.GET(ep.path)
			} else {
				result = client.POST(ep.path, ep.body)
			}

			if result.Error == nil {
				latencies = append(latencies, result.Latency)
			}
		}

		if len(latencies) > 0 {
			stats := calculateLatencyStats(latencies)
			t.Logf("║  %-45s             ║", ep.name)
			t.Logf("║    Avg: %-10v P50: %-10v P99: %-10v         ║", stats.avg, stats.p50, stats.p99)
		} else {
			t.Logf("║  %-45s  [FAILED]       ║", ep.name)
		}
	}

	t.Logf("╚══════════════════════════════════════════════════════════════╝")
}

// TestAPI_ThroughputLimit tests maximum API throughput
func TestAPI_ThroughputLimit(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping throughput test in short mode")
	}

	config := DefaultConfig()

	concurrencyLevels := []int{1, 10, 50, 100}
	duration := 10 * time.Second

	t.Logf("\n╔══════════════════════════════════════════════════════════════╗")
	t.Logf("║  API Throughput Test (Duration: %v)                      ║", duration)
	t.Logf("╠══════════════════════════════════════════════════════════════╣")

	for _, concurrency := range concurrencyLevels {
		result := runThroughputTest(config, concurrency, duration)
		t.Logf("║  Concurrency: %-3d  RPS: %-8.2f  P99: %-12v       ║",
			concurrency, result.requestsPerSecond, result.p99Latency)
	}

	t.Logf("╚══════════════════════════════════════════════════════════════╝")
}

type throughputResult struct {
	concurrency       int
	totalRequests     int64
	successfulReqs    int64
	failedReqs        int64
	requestsPerSecond float64
	avgLatency        time.Duration
	p99Latency        time.Duration
}

func runThroughputTest(config *TestConfig, concurrency int, duration time.Duration) *throughputResult {
	result := &throughputResult{concurrency: concurrency}

	var wg sync.WaitGroup
	var totalReqs, successReqs, failedReqs int64
	latencies := make([]time.Duration, 0, 10000)
	var latencyMu sync.Mutex

	done := make(chan struct{})
	time.AfterFunc(duration, func() { close(done) })

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			client := NewHTTPClient(config)

			for {
				select {
				case <-done:
					return
				default:
					r := client.GET("/v1/markets/BTC-USDC/orderbook")
					atomic.AddInt64(&totalReqs, 1)

					if r.Error == nil && r.StatusCode == 200 {
						atomic.AddInt64(&successReqs, 1)
						latencyMu.Lock()
						latencies = append(latencies, r.Latency)
						latencyMu.Unlock()
					} else {
						atomic.AddInt64(&failedReqs, 1)
					}
				}
			}
		}()
	}

	wg.Wait()

	result.totalRequests = totalReqs
	result.successfulReqs = successReqs
	result.failedReqs = failedReqs
	result.requestsPerSecond = float64(totalReqs) / duration.Seconds()

	if len(latencies) > 0 {
		stats := calculateLatencyStats(latencies)
		result.avgLatency = stats.avg
		result.p99Latency = stats.p99
	}

	return result
}

// ============ WebSocket Connection Scaling Tests ============

// TestWebSocket_ConnectionScaling tests WebSocket with increasing connections
func TestWebSocket_ConnectionScaling(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping WebSocket scaling test in short mode")
	}

	config := DefaultConfig()
	connectionCounts := []int{1, 10, 50, 100}

	t.Logf("\n╔══════════════════════════════════════════════════════════════╗")
	t.Logf("║  WebSocket Connection Scaling Test                           ║")
	t.Logf("╠══════════════════════════════════════════════════════════════╣")

	for _, count := range connectionCounts {
		result := testWebSocketConnections(config, count)
		t.Logf("║  Connections: %-4d  Success: %-4d  Avg Connect: %-10v ║",
			count, result.successfulConns, result.avgConnectTime)
	}

	t.Logf("╚══════════════════════════════════════════════════════════════╝")
}

type wsConnectionResult struct {
	totalConns      int
	successfulConns int
	failedConns     int
	avgConnectTime  time.Duration
}

func testWebSocketConnections(config *TestConfig, count int) *wsConnectionResult {
	result := &wsConnectionResult{totalConns: count}

	var wg sync.WaitGroup
	connectTimes := make([]time.Duration, 0, count)
	var mu sync.Mutex
	var successCount int

	for i := 0; i < count; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			client := NewWSClient(config)
			start := time.Now()

			err := client.Connect("/ws")
			connectTime := time.Since(start)

			mu.Lock()
			if err == nil {
				successCount++
				connectTimes = append(connectTimes, connectTime)
				defer client.Close()
			}
			mu.Unlock()

			// Keep connection alive briefly
			time.Sleep(100 * time.Millisecond)
		}()
	}

	wg.Wait()

	result.successfulConns = successCount
	result.failedConns = count - successCount

	if len(connectTimes) > 0 {
		var total time.Duration
		for _, t := range connectTimes {
			total += t
		}
		result.avgConnectTime = total / time.Duration(len(connectTimes))
	}

	return result
}

// TestWebSocket_MessageLatency tests WebSocket message round-trip latency
func TestWebSocket_MessageLatency(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping WebSocket latency test in short mode")
	}

	config := DefaultConfig()
	client := NewWSClient(config)

	err := client.Connect("/ws")
	if err != nil {
		t.Skipf("WebSocket not available: %v", err)
		return
	}
	defer client.Close()

	// Subscribe to orderbook
	subscribeMsg := map[string]interface{}{
		"type":     "subscribe",
		"channel":  "orderbook",
		"marketId": "BTC-USDC",
	}

	latencies := make([]time.Duration, 0, 100)

	// Measure round-trip latency for 100 subscribe/unsubscribe cycles
	for i := 0; i < 100; i++ {
		start := time.Now()

		err := client.Send(subscribeMsg)
		if err != nil {
			continue
		}

		_, err = client.Receive(5 * time.Second)
		if err == nil {
			latencies = append(latencies, time.Since(start))
		}
	}

	if len(latencies) > 0 {
		stats := calculateLatencyStats(latencies)
		t.Logf("\n╔══════════════════════════════════════════════════════════════╗")
		t.Logf("║  WebSocket Message Latency                                   ║")
		t.Logf("╠══════════════════════════════════════════════════════════════╣")
		t.Logf("║  Messages:    %-48d ║", len(latencies))
		t.Logf("║  Avg Latency: %-48v ║", stats.avg)
		t.Logf("║  P50 Latency: %-48v ║", stats.p50)
		t.Logf("║  P99 Latency: %-48v ║", stats.p99)
		t.Logf("║  Min Latency: %-48v ║", stats.min)
		t.Logf("║  Max Latency: %-48v ║", stats.max)
		t.Logf("╚══════════════════════════════════════════════════════════════╝")
	}
}

// ============ Order Placement Performance Tests ============

// TestAPI_OrderPlacementPerformance tests order placement throughput
func TestAPI_OrderPlacementPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping order placement test in short mode")
	}

	config := DefaultConfig()
	client := NewHTTPClient(config)

	orderCount := 1000
	latencies := make([]time.Duration, 0, orderCount)
	successCount := 0

	t.Logf("Placing %d orders...", orderCount)
	start := time.Now()

	for i := 0; i < orderCount; i++ {
		order := &Order{
			MarketID:  "BTC-USDC",
			Trader:    fmt.Sprintf("perf-trader-%d", i),
			Side:      "buy",
			OrderType: "limit",
			Price:     fmt.Sprintf("%d", 49000+i%2000),
			Quantity:  "0.01",
			Leverage:  "10",
		}
		if i%2 == 0 {
			order.Side = "sell"
			order.Price = fmt.Sprintf("%d", 51000+i%2000)
		}

		result, _ := PlaceOrder(client, order)
		if result != nil {
			latencies = append(latencies, result.Latency)
			if result.Error == nil && (result.StatusCode == 200 || result.StatusCode == 201) {
				successCount++
			}
		}
	}

	totalDuration := time.Since(start)

	if len(latencies) > 0 {
		stats := calculateLatencyStats(latencies)
		ordersPerSecond := float64(orderCount) / totalDuration.Seconds()

		t.Logf("\n╔══════════════════════════════════════════════════════════════╗")
		t.Logf("║  Order Placement Performance                                 ║")
		t.Logf("╠══════════════════════════════════════════════════════════════╣")
		t.Logf("║  Total Orders:     %-42d ║", orderCount)
		t.Logf("║  Successful:       %-42d ║", successCount)
		t.Logf("║  Success Rate:     %-41.2f%% ║", float64(successCount)/float64(orderCount)*100)
		t.Logf("║  Total Duration:   %-42v ║", totalDuration.Round(time.Millisecond))
		t.Logf("║  Orders/Second:    %-42.2f ║", ordersPerSecond)
		t.Logf("╠══════════════════════════════════════════════════════════════╣")
		t.Logf("║  Latency Statistics                                          ║")
		t.Logf("╠══════════════════════════════════════════════════════════════╣")
		t.Logf("║  Avg:              %-43v ║", stats.avg)
		t.Logf("║  P50:              %-43v ║", stats.p50)
		t.Logf("║  P95:              %-43v ║", stats.p95)
		t.Logf("║  P99:              %-43v ║", stats.p99)
		t.Logf("║  Min:              %-43v ║", stats.min)
		t.Logf("║  Max:              %-43v ║", stats.max)
		t.Logf("╚══════════════════════════════════════════════════════════════╝")

		// Verify P99 target
		if stats.p99 <= 100*time.Millisecond {
			t.Logf("✅ PASS: P99 latency (%v) <= 100ms target", stats.p99)
		} else {
			t.Logf("⚠️ WARNING: P99 latency (%v) > 100ms target", stats.p99)
		}
	}
}

// TestAPI_ConcurrentOrderPlacement tests concurrent order placement
func TestAPI_ConcurrentOrderPlacement(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping concurrent order test in short mode")
	}

	config := DefaultConfig()

	concurrency := 50
	ordersPerWorker := 100
	totalOrders := concurrency * ordersPerWorker

	var wg sync.WaitGroup
	var successCount, failedCount int64
	allLatencies := make([]time.Duration, 0, totalOrders)
	var mu sync.Mutex

	t.Logf("Starting %d concurrent workers, %d orders each...", concurrency, ordersPerWorker)
	start := time.Now()

	for w := 0; w < concurrency; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			client := NewHTTPClient(config)

			for i := 0; i < ordersPerWorker; i++ {
				order := &Order{
					MarketID:  "BTC-USDC",
					Trader:    fmt.Sprintf("worker-%d-order-%d", workerID, i),
					Side:      "buy",
					OrderType: "limit",
					Price:     fmt.Sprintf("%d", 49000+(workerID*100+i)%2000),
					Quantity:  "0.01",
					Leverage:  "10",
				}
				if (workerID+i)%2 == 0 {
					order.Side = "sell"
					order.Price = fmt.Sprintf("%d", 51000+(workerID*100+i)%2000)
				}

				result, _ := PlaceOrder(client, order)
				if result != nil {
					mu.Lock()
					allLatencies = append(allLatencies, result.Latency)
					mu.Unlock()

					if result.Error == nil && (result.StatusCode == 200 || result.StatusCode == 201) {
						atomic.AddInt64(&successCount, 1)
					} else {
						atomic.AddInt64(&failedCount, 1)
					}
				} else {
					atomic.AddInt64(&failedCount, 1)
				}
			}
		}(w)
	}

	wg.Wait()
	totalDuration := time.Since(start)

	if len(allLatencies) > 0 {
		stats := calculateLatencyStats(allLatencies)
		ordersPerSecond := float64(totalOrders) / totalDuration.Seconds()

		t.Logf("\n╔══════════════════════════════════════════════════════════════╗")
		t.Logf("║  Concurrent Order Placement Performance                      ║")
		t.Logf("╠══════════════════════════════════════════════════════════════╣")
		t.Logf("║  Concurrency:      %-42d ║", concurrency)
		t.Logf("║  Total Orders:     %-42d ║", totalOrders)
		t.Logf("║  Successful:       %-42d ║", successCount)
		t.Logf("║  Failed:           %-42d ║", failedCount)
		t.Logf("║  Success Rate:     %-41.2f%% ║", float64(successCount)/float64(totalOrders)*100)
		t.Logf("║  Total Duration:   %-42v ║", totalDuration.Round(time.Millisecond))
		t.Logf("║  Orders/Second:    %-42.2f ║", ordersPerSecond)
		t.Logf("╠══════════════════════════════════════════════════════════════╣")
		t.Logf("║  Latency Statistics                                          ║")
		t.Logf("╠══════════════════════════════════════════════════════════════╣")
		t.Logf("║  Avg:              %-43v ║", stats.avg)
		t.Logf("║  P50:              %-43v ║", stats.p50)
		t.Logf("║  P95:              %-43v ║", stats.p95)
		t.Logf("║  P99:              %-43v ║", stats.p99)
		t.Logf("╚══════════════════════════════════════════════════════════════╝")

		// Verify targets
		if ordersPerSecond >= 500 {
			t.Logf("✅ PASS: Throughput (%.2f ops/sec) >= 500 target", ordersPerSecond)
		} else {
			t.Logf("⚠️ WARNING: Throughput (%.2f ops/sec) < 500 target", ordersPerSecond)
		}
	}
}

// ============ Helper Functions ============

type latencyStats struct {
	min time.Duration
	max time.Duration
	avg time.Duration
	p50 time.Duration
	p90 time.Duration
	p95 time.Duration
	p99 time.Duration
}

func calculateLatencyStats(latencies []time.Duration) *latencyStats {
	if len(latencies) == 0 {
		return &latencyStats{}
	}

	sorted := make([]time.Duration, len(latencies))
	copy(sorted, latencies)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i] < sorted[j]
	})

	var sum time.Duration
	for _, l := range sorted {
		sum += l
	}

	return &latencyStats{
		min: sorted[0],
		max: sorted[len(sorted)-1],
		avg: sum / time.Duration(len(sorted)),
		p50: sorted[len(sorted)*50/100],
		p90: sorted[len(sorted)*90/100],
		p95: sorted[len(sorted)*95/100],
		p99: sorted[len(sorted)*99/100],
	}
}

// ============ Memory Pressure Test ============

// TestAPI_MemoryPressure tests API under memory pressure
func TestAPI_MemoryPressure(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping memory pressure test in short mode")
	}

	config := DefaultConfig()
	client := NewHTTPClient(config)

	// Place many orders to create memory pressure
	orderCount := 10000
	successCount := 0

	t.Logf("Creating %d orders for memory pressure test...", orderCount)
	start := time.Now()

	for i := 0; i < orderCount; i++ {
		order := &Order{
			MarketID:  "BTC-USDC",
			Trader:    fmt.Sprintf("memory-test-%d", i),
			Side:      "buy",
			OrderType: "limit",
			Price:     fmt.Sprintf("%d", 40000+i%10000),
			Quantity:  "0.01",
			Leverage:  "10",
		}
		if i%2 == 0 {
			order.Side = "sell"
			order.Price = fmt.Sprintf("%d", 60000+i%10000)
		}

		result, _ := PlaceOrder(client, order)
		if result != nil && result.Error == nil {
			successCount++
		}

		// Log progress every 1000 orders
		if (i+1)%1000 == 0 {
			t.Logf("  Progress: %d/%d orders placed", i+1, orderCount)
		}
	}

	totalDuration := time.Since(start)

	t.Logf("\n╔══════════════════════════════════════════════════════════════╗")
	t.Logf("║  Memory Pressure Test Results                                ║")
	t.Logf("╠══════════════════════════════════════════════════════════════╣")
	t.Logf("║  Total Orders:     %-42d ║", orderCount)
	t.Logf("║  Successful:       %-42d ║", successCount)
	t.Logf("║  Duration:         %-42v ║", totalDuration.Round(time.Millisecond))
	t.Logf("║  Orders/Second:    %-42.2f ║", float64(orderCount)/totalDuration.Seconds())
	t.Logf("╚══════════════════════════════════════════════════════════════╝")

	// Verify API still responds after pressure
	t.Log("Verifying API responsiveness after pressure...")
	for i := 0; i < 10; i++ {
		result := client.GET("/v1/health")
		if result.Error != nil || result.StatusCode != 200 {
			t.Errorf("API unresponsive after memory pressure: %v", result.Error)
			return
		}
	}
	t.Log("✅ API remains responsive after memory pressure test")
}
