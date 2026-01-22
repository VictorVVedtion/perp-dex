package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

// PlaceOrderRequest represents the request to place an order
type PlaceOrderRequest struct {
	MarketID string `json:"market_id"`
	Side     string `json:"side"`
	Type     string `json:"type"`
	Price    string `json:"price"`
	Quantity string `json:"quantity"`
	Trader   string `json:"trader"`
}

// PlaceOrderResponse represents the response
type PlaceOrderResponse struct {
	Order struct {
		OrderID   string `json:"order_id"`
		Status    string `json:"status"`
		FilledQty string `json:"filled_qty"`
	} `json:"order"`
	Match struct {
		FilledQty string `json:"filled_qty"`
		Trades    []struct {
			TradeID  string `json:"trade_id"`
			Price    string `json:"price"`
			Quantity string `json:"quantity"`
		} `json:"trades"`
	} `json:"match"`
}

// LatencyRecord records latency for each order
type LatencyRecord struct {
	Side       string
	Latency    time.Duration
	Matched    bool
	MatchCount int
	Timestamp  time.Time
}

// BenchmarkResults holds all test results
type BenchmarkResults struct {
	BuyOrders      int64
	SellOrders     int64
	BuySuccess     int64
	SellSuccess    int64
	BuyFailed      int64
	SellFailed     int64
	TotalMatched   int64
	TotalTrades    int64
	BuyLatencies   []time.Duration
	SellLatencies  []time.Duration
	MatchLatencies []time.Duration
	mu             sync.Mutex
}

func (r *BenchmarkResults) AddBuy(latency time.Duration, success bool, matched bool, trades int) {
	atomic.AddInt64(&r.BuyOrders, 1)
	if success {
		atomic.AddInt64(&r.BuySuccess, 1)
	} else {
		atomic.AddInt64(&r.BuyFailed, 1)
	}
	if matched {
		atomic.AddInt64(&r.TotalMatched, 1)
		atomic.AddInt64(&r.TotalTrades, int64(trades))
	}
	r.mu.Lock()
	r.BuyLatencies = append(r.BuyLatencies, latency)
	if matched {
		r.MatchLatencies = append(r.MatchLatencies, latency)
	}
	r.mu.Unlock()
}

func (r *BenchmarkResults) AddSell(latency time.Duration, success bool, matched bool, trades int) {
	atomic.AddInt64(&r.SellOrders, 1)
	if success {
		atomic.AddInt64(&r.SellSuccess, 1)
	} else {
		atomic.AddInt64(&r.SellFailed, 1)
	}
	if matched {
		atomic.AddInt64(&r.TotalMatched, 1)
		atomic.AddInt64(&r.TotalTrades, int64(trades))
	}
	r.mu.Lock()
	r.SellLatencies = append(r.SellLatencies, latency)
	if matched {
		r.MatchLatencies = append(r.MatchLatencies, latency)
	}
	r.mu.Unlock()
}

func percentile(latencies []time.Duration, p float64) time.Duration {
	if len(latencies) == 0 {
		return 0
	}
	sorted := make([]time.Duration, len(latencies))
	copy(sorted, latencies)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })
	idx := int(float64(len(sorted)) * p)
	if idx >= len(sorted) {
		idx = len(sorted) - 1
	}
	return sorted[idx]
}

func avg(latencies []time.Duration) time.Duration {
	if len(latencies) == 0 {
		return 0
	}
	var total time.Duration
	for _, l := range latencies {
		total += l
	}
	return total / time.Duration(len(latencies))
}

func min(latencies []time.Duration) time.Duration {
	if len(latencies) == 0 {
		return 0
	}
	m := latencies[0]
	for _, l := range latencies {
		if l < m {
			m = l
		}
	}
	return m
}

func max(latencies []time.Duration) time.Duration {
	if len(latencies) == 0 {
		return 0
	}
	m := latencies[0]
	for _, l := range latencies {
		if l > m {
			m = l
		}
	}
	return m
}

func placeOrder(client *http.Client, baseURL string, req *PlaceOrderRequest) (time.Duration, bool, bool, int) {
	body, _ := json.Marshal(req)
	start := time.Now()

	httpReq, err := http.NewRequest("POST", baseURL+"/v1/orders", bytes.NewReader(body))
	if err != nil {
		return time.Since(start), false, false, 0
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(httpReq)
	latency := time.Since(start)

	if err != nil {
		return latency, false, false, 0
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return latency, false, false, 0
	}

	var orderResp PlaceOrderResponse
	if err := json.NewDecoder(resp.Body).Decode(&orderResp); err != nil {
		return latency, true, false, 0
	}

	matched := len(orderResp.Match.Trades) > 0
	return latency, true, matched, len(orderResp.Match.Trades)
}

func main() {
	baseURL := flag.String("url", "http://localhost:8080", "API base URL")
	orderCount := flag.Int("n", 10000, "Number of orders per side (buy and sell)")
	concurrency := flag.Int("c", 100, "Concurrency level")
	market := flag.String("market", "BTC-USDC", "Market ID")
	price := flag.String("price", "50000", "Order price for matching")
	quantity := flag.String("qty", "0.01", "Order quantity")
	outputFile := flag.String("o", "", "Output JSON report file")
	flag.Parse()

	fmt.Println("╔══════════════════════════════════════════════════════════════════╗")
	fmt.Println("║      PerpDEX Matching Engine Benchmark - Buy/Sell Stress Test    ║")
	fmt.Println("╚══════════════════════════════════════════════════════════════════╝")
	fmt.Println()
	fmt.Printf("Configuration:\n")
	fmt.Printf("  API URL:      %s\n", *baseURL)
	fmt.Printf("  Market:       %s\n", *market)
	fmt.Printf("  Orders/Side:  %d (total: %d)\n", *orderCount, *orderCount*2)
	fmt.Printf("  Concurrency:  %d\n", *concurrency)
	fmt.Printf("  Price:        %s\n", *price)
	fmt.Printf("  Quantity:     %s\n", *quantity)
	fmt.Println()

	// Check health
	fmt.Print("Checking API health... ")
	client := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:        1000,
			MaxIdleConnsPerHost: 200,
			IdleConnTimeout:     90 * time.Second,
		},
	}

	resp, err := client.Get(*baseURL + "/health")
	if err != nil {
		fmt.Printf("FAILED: %v\n", err)
		os.Exit(1)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		fmt.Printf("FAILED: status %d\n", resp.StatusCode)
		os.Exit(1)
	}
	fmt.Println("OK")
	fmt.Println()

	results := &BenchmarkResults{
		BuyLatencies:   make([]time.Duration, 0, *orderCount),
		SellLatencies:  make([]time.Duration, 0, *orderCount),
		MatchLatencies: make([]time.Duration, 0, *orderCount*2),
	}

	// Semaphore for concurrency control
	sem := make(chan struct{}, *concurrency)
	var wg sync.WaitGroup

	// Progress tracking
	var processed int64
	total := int64(*orderCount * 2)
	done := make(chan struct{})

	go func() {
		ticker := time.NewTicker(500 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				p := atomic.LoadInt64(&processed)
				pct := float64(p) / float64(total) * 100
				fmt.Printf("\r  Progress: %d/%d (%.1f%%) | Matched: %d | Trades: %d    ",
					p, total, pct,
					atomic.LoadInt64(&results.TotalMatched),
					atomic.LoadInt64(&results.TotalTrades))
			}
		}
	}()

	fmt.Println("Starting benchmark...")
	startTime := time.Now()

	// Launch buy and sell orders concurrently
	for i := 0; i < *orderCount; i++ {
		// Buy order
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			req := &PlaceOrderRequest{
				MarketID: *market,
				Side:     "buy",
				Type:     "limit",
				Price:    *price,
				Quantity: *quantity,
				Trader:   fmt.Sprintf("buyer_%d", idx),
			}

			latency, success, matched, trades := placeOrder(client, *baseURL, req)
			results.AddBuy(latency, success, matched, trades)
			atomic.AddInt64(&processed, 1)
		}(i)

		// Sell order
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			req := &PlaceOrderRequest{
				MarketID: *market,
				Side:     "sell",
				Type:     "limit",
				Price:    *price,
				Quantity: *quantity,
				Trader:   fmt.Sprintf("seller_%d", idx),
			}

			latency, success, matched, trades := placeOrder(client, *baseURL, req)
			results.AddSell(latency, success, matched, trades)
			atomic.AddInt64(&processed, 1)
		}(i)
	}

	wg.Wait()
	close(done)
	elapsed := time.Since(startTime)

	fmt.Printf("\r                                                                              \r")
	fmt.Println()
	fmt.Println()

	// Calculate statistics
	allLatencies := append(results.BuyLatencies, results.SellLatencies...)
	totalOrders := results.BuyOrders + results.SellOrders
	totalSuccess := results.BuySuccess + results.SellSuccess
	totalFailed := results.BuyFailed + results.SellFailed
	successRate := float64(totalSuccess) / float64(totalOrders) * 100
	throughput := float64(totalOrders) / elapsed.Seconds()

	// Print results
	fmt.Println("╔══════════════════════════════════════════════════════════════════╗")
	fmt.Println("║                       BENCHMARK RESULTS                          ║")
	fmt.Println("╚══════════════════════════════════════════════════════════════════╝")
	fmt.Println()

	fmt.Printf("Test Duration:        %v\n", elapsed.Round(time.Millisecond))
	fmt.Printf("Throughput:           %.2f orders/sec\n", throughput)
	fmt.Println()

	fmt.Println("── Order Statistics ───────────────────────────────────────────────")
	fmt.Printf("  Total Orders:       %d\n", totalOrders)
	fmt.Printf("  Buy Orders:         %d (success: %d, failed: %d)\n", results.BuyOrders, results.BuySuccess, results.BuyFailed)
	fmt.Printf("  Sell Orders:        %d (success: %d, failed: %d)\n", results.SellOrders, results.SellSuccess, results.SellFailed)
	fmt.Printf("  Success Rate:       %.2f%%\n", successRate)
	fmt.Println()

	fmt.Println("── Matching Statistics ────────────────────────────────────────────")
	fmt.Printf("  Orders Matched:     %d\n", results.TotalMatched)
	fmt.Printf("  Total Trades:       %d\n", results.TotalTrades)
	matchRate := float64(results.TotalMatched) / float64(totalSuccess) * 100
	fmt.Printf("  Match Rate:         %.2f%%\n", matchRate)
	fmt.Println()

	fmt.Println("── Overall Latency (all orders) ───────────────────────────────────")
	fmt.Printf("  Min:                %v\n", min(allLatencies))
	fmt.Printf("  Max:                %v\n", max(allLatencies))
	fmt.Printf("  Average:            %v\n", avg(allLatencies))
	fmt.Printf("  P50 (Median):       %v\n", percentile(allLatencies, 0.50))
	fmt.Printf("  P90:                %v\n", percentile(allLatencies, 0.90))
	fmt.Printf("  P95:                %v\n", percentile(allLatencies, 0.95))
	fmt.Printf("  P99:                %v\n", percentile(allLatencies, 0.99))
	fmt.Println()

	fmt.Println("── Buy Order Latency ──────────────────────────────────────────────")
	fmt.Printf("  Min:                %v\n", min(results.BuyLatencies))
	fmt.Printf("  Max:                %v\n", max(results.BuyLatencies))
	fmt.Printf("  Average:            %v\n", avg(results.BuyLatencies))
	fmt.Printf("  P99:                %v\n", percentile(results.BuyLatencies, 0.99))
	fmt.Println()

	fmt.Println("── Sell Order Latency ─────────────────────────────────────────────")
	fmt.Printf("  Min:                %v\n", min(results.SellLatencies))
	fmt.Printf("  Max:                %v\n", max(results.SellLatencies))
	fmt.Printf("  Average:            %v\n", avg(results.SellLatencies))
	fmt.Printf("  P99:                %v\n", percentile(results.SellLatencies, 0.99))
	fmt.Println()

	if len(results.MatchLatencies) > 0 {
		fmt.Println("── Matched Order Latency (orders that triggered trades) ──────────")
		fmt.Printf("  Min:                %v\n", min(results.MatchLatencies))
		fmt.Printf("  Max:                %v\n", max(results.MatchLatencies))
		fmt.Printf("  Average:            %v\n", avg(results.MatchLatencies))
		fmt.Printf("  P99:                %v\n", percentile(results.MatchLatencies, 0.99))
		fmt.Println()
	}

	fmt.Println("── Assessment ─────────────────────────────────────────────────────")
	if successRate >= 99.9 {
		fmt.Println("  ✅ Success Rate:    Excellent (>99.9%)")
	} else if successRate >= 99 {
		fmt.Println("  ✅ Success Rate:    Good (>99%)")
	} else if successRate >= 95 {
		fmt.Println("  ⚠️  Success Rate:    Acceptable (>95%)")
	} else {
		fmt.Println("  ❌ Success Rate:    Poor (<95%)")
	}

	avgLat := avg(allLatencies)
	if avgLat < 1*time.Millisecond {
		fmt.Println("  ✅ Latency:         Excellent (<1ms avg)")
	} else if avgLat < 10*time.Millisecond {
		fmt.Println("  ✅ Latency:         Good (<10ms avg)")
	} else if avgLat < 100*time.Millisecond {
		fmt.Println("  ⚠️  Latency:         Acceptable (<100ms avg)")
	} else {
		fmt.Println("  ❌ Latency:         High (>100ms avg)")
	}

	if throughput > 10000 {
		fmt.Println("  ✅ Throughput:      Excellent (>10K/s)")
	} else if throughput > 1000 {
		fmt.Println("  ✅ Throughput:      Good (>1K/s)")
	} else if throughput > 100 {
		fmt.Println("  ⚠️  Throughput:      Acceptable (>100/s)")
	} else {
		fmt.Println("  ❌ Throughput:      Low (<100/s)")
	}

	fmt.Println()
	fmt.Println("══════════════════════════════════════════════════════════════════")

	// Save report if requested
	if *outputFile != "" {
		report := map[string]interface{}{
			"config": map[string]interface{}{
				"api_url":         *baseURL,
				"market":          *market,
				"orders_per_side": *orderCount,
				"concurrency":     *concurrency,
				"price":           *price,
				"quantity":        *quantity,
			},
			"summary": map[string]interface{}{
				"duration_ms":        elapsed.Milliseconds(),
				"throughput_per_sec": throughput,
				"total_orders":       totalOrders,
				"success_orders":     totalSuccess,
				"failed_orders":      totalFailed,
				"success_rate":       successRate,
				"total_matched":      results.TotalMatched,
				"total_trades":       results.TotalTrades,
				"match_rate":         matchRate,
			},
			"latency_all": map[string]interface{}{
				"min_us": min(allLatencies).Microseconds(),
				"max_us": max(allLatencies).Microseconds(),
				"avg_us": avg(allLatencies).Microseconds(),
				"p50_us": percentile(allLatencies, 0.50).Microseconds(),
				"p90_us": percentile(allLatencies, 0.90).Microseconds(),
				"p95_us": percentile(allLatencies, 0.95).Microseconds(),
				"p99_us": percentile(allLatencies, 0.99).Microseconds(),
			},
			"latency_buy": map[string]interface{}{
				"min_us": min(results.BuyLatencies).Microseconds(),
				"max_us": max(results.BuyLatencies).Microseconds(),
				"avg_us": avg(results.BuyLatencies).Microseconds(),
				"p99_us": percentile(results.BuyLatencies, 0.99).Microseconds(),
			},
			"latency_sell": map[string]interface{}{
				"min_us": min(results.SellLatencies).Microseconds(),
				"max_us": max(results.SellLatencies).Microseconds(),
				"avg_us": avg(results.SellLatencies).Microseconds(),
				"p99_us": percentile(results.SellLatencies, 0.99).Microseconds(),
			},
			"timestamp": time.Now().Format(time.RFC3339),
		}

		if len(results.MatchLatencies) > 0 {
			report["latency_matched"] = map[string]interface{}{
				"min_us": min(results.MatchLatencies).Microseconds(),
				"max_us": max(results.MatchLatencies).Microseconds(),
				"avg_us": avg(results.MatchLatencies).Microseconds(),
				"p99_us": percentile(results.MatchLatencies, 0.99).Microseconds(),
			}
		}

		file, err := os.Create(*outputFile)
		if err != nil {
			fmt.Printf("Failed to create report file: %v\n", err)
		} else {
			defer file.Close()
			encoder := json.NewEncoder(file)
			encoder.SetIndent("", "  ")
			encoder.Encode(report)
			fmt.Printf("\nReport saved to: %s\n", *outputFile)
		}
	}
}
