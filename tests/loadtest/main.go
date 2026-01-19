package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

// Test configuration
type Config struct {
	BaseURL     string
	Concurrency int
	Duration    time.Duration
	RampUp      time.Duration
	Markets     []string
	TraderCount int
}

// Test results
type Results struct {
	TotalRequests     int64
	SuccessRequests   int64
	FailedRequests    int64
	TotalLatency      int64 // microseconds
	MinLatency        int64
	MaxLatency        int64
	Latencies         []int64
	StatusCodes       map[int]int64
	Errors            map[string]int64
	StartTime         time.Time
	EndTime           time.Time
	RequestsPerSecond float64
	mu                sync.Mutex
}

// Order request
type PlaceOrderRequest struct {
	MarketID string `json:"market_id"`
	Side     string `json:"side"`
	Type     string `json:"type"`
	Price    string `json:"price"`
	Quantity string `json:"quantity"`
	Trader   string `json:"trader"`
}

// Test runner
type LoadTester struct {
	config  *Config
	results *Results
	client  *http.Client
	wg      sync.WaitGroup
	stopCh  chan struct{}
}

func NewLoadTester(config *Config) *LoadTester {
	return &LoadTester{
		config: config,
		results: &Results{
			MinLatency:  int64(^uint64(0) >> 1), // Max int64
			StatusCodes: make(map[int]int64),
			Errors:      make(map[string]int64),
			Latencies:   make([]int64, 0),
		},
		client: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        1000,
				MaxIdleConnsPerHost: 100,
				IdleConnTimeout:     90 * time.Second,
			},
		},
		stopCh: make(chan struct{}),
	}
}

func (lt *LoadTester) Run() {
	fmt.Println("╔══════════════════════════════════════════════════════════════╗")
	fmt.Println("║           PerpDEX API Load Test - Order Placement            ║")
	fmt.Println("╚══════════════════════════════════════════════════════════════╝")
	fmt.Println()

	// Print configuration
	fmt.Printf("Configuration:\n")
	fmt.Printf("  Base URL:     %s\n", lt.config.BaseURL)
	fmt.Printf("  Concurrency:  %d workers\n", lt.config.Concurrency)
	fmt.Printf("  Duration:     %v\n", lt.config.Duration)
	fmt.Printf("  Ramp-up:      %v\n", lt.config.RampUp)
	fmt.Printf("  Markets:      %v\n", lt.config.Markets)
	fmt.Printf("  Traders:      %d\n", lt.config.TraderCount)
	fmt.Println()

	// Check server health first
	fmt.Print("Checking server health... ")
	if err := lt.checkHealth(); err != nil {
		fmt.Printf("FAILED: %v\n", err)
		fmt.Println("\nPlease ensure the API server is running:")
		fmt.Println("  cd cmd/api && go run main.go")
		return
	}
	fmt.Println("OK")
	fmt.Println()

	// Start test
	fmt.Println("Starting load test...")
	lt.results.StartTime = time.Now()

	// Ramp-up workers
	workersPerInterval := lt.config.Concurrency / 10
	if workersPerInterval < 1 {
		workersPerInterval = 1
	}
	rampUpInterval := lt.config.RampUp / 10

	currentWorkers := 0
	for currentWorkers < lt.config.Concurrency {
		toAdd := workersPerInterval
		if currentWorkers+toAdd > lt.config.Concurrency {
			toAdd = lt.config.Concurrency - currentWorkers
		}

		for i := 0; i < toAdd; i++ {
			lt.wg.Add(1)
			go lt.worker(currentWorkers + i)
		}
		currentWorkers += toAdd

		fmt.Printf("\r  Workers: %d/%d", currentWorkers, lt.config.Concurrency)

		if currentWorkers < lt.config.Concurrency {
			time.Sleep(rampUpInterval)
		}
	}
	fmt.Println()
	fmt.Println()

	// Progress reporting
	go lt.reportProgress()

	// Wait for test duration
	time.Sleep(lt.config.Duration)

	// Stop workers
	close(lt.stopCh)
	lt.wg.Wait()

	lt.results.EndTime = time.Now()

	// Calculate final metrics
	lt.calculateMetrics()

	// Print results
	lt.printResults()
}

func (lt *LoadTester) checkHealth() error {
	resp, err := lt.client.Get(lt.config.BaseURL + "/health")
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unhealthy status: %d", resp.StatusCode)
	}
	return nil
}

func (lt *LoadTester) worker(id int) {
	defer lt.wg.Done()

	traders := make([]string, lt.config.TraderCount)
	for i := range traders {
		traders[i] = fmt.Sprintf("perpdex1test%d%04d", id, i)
	}

	for {
		select {
		case <-lt.stopCh:
			return
		default:
			// Place order
			lt.placeOrder(traders[rand.Intn(len(traders))])

			// Small delay to avoid overwhelming
			time.Sleep(time.Duration(rand.Intn(10)) * time.Millisecond)
		}
	}
}

func (lt *LoadTester) placeOrder(trader string) {
	market := lt.config.Markets[rand.Intn(len(lt.config.Markets))]

	// Generate random order
	side := "buy"
	if rand.Float32() > 0.5 {
		side = "sell"
	}

	orderType := "limit"
	if rand.Float32() > 0.8 {
		orderType = "market"
	}

	// Random price around base prices
	basePrices := map[string]float64{
		"BTC-USDC": 50000,
		"ETH-USDC": 3000,
		"SOL-USDC": 100,
	}
	basePrice := basePrices[market]
	if basePrice == 0 {
		basePrice = 1000
	}

	// Add some variance
	priceVar := basePrice * (0.98 + rand.Float64()*0.04) // ±2%
	price := fmt.Sprintf("%.2f", priceVar)

	// Random quantity
	quantity := fmt.Sprintf("%.4f", rand.Float64()*0.5+0.001)

	req := PlaceOrderRequest{
		MarketID: market,
		Side:     side,
		Type:     orderType,
		Price:    price,
		Quantity: quantity,
		Trader:   trader,
	}

	body, _ := json.Marshal(req)

	start := time.Now()

	httpReq, err := http.NewRequest("POST", lt.config.BaseURL+"/v1/orders", bytes.NewReader(body))
	if err != nil {
		lt.recordError("create_request_error")
		return
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-Trader-Address", trader)

	resp, err := lt.client.Do(httpReq)
	latency := time.Since(start).Microseconds()

	if err != nil {
		lt.recordError("network_error")
		lt.recordLatency(latency, false, 0)
		return
	}
	defer resp.Body.Close()

	// Drain response body
	io.Copy(io.Discard, resp.Body)

	success := resp.StatusCode == http.StatusCreated || resp.StatusCode == http.StatusOK
	lt.recordLatency(latency, success, resp.StatusCode)
}

func (lt *LoadTester) recordLatency(latency int64, success bool, statusCode int) {
	atomic.AddInt64(&lt.results.TotalRequests, 1)
	atomic.AddInt64(&lt.results.TotalLatency, latency)

	if success {
		atomic.AddInt64(&lt.results.SuccessRequests, 1)
	} else {
		atomic.AddInt64(&lt.results.FailedRequests, 1)
	}

	lt.results.mu.Lock()
	lt.results.Latencies = append(lt.results.Latencies, latency)

	if latency < lt.results.MinLatency {
		lt.results.MinLatency = latency
	}
	if latency > lt.results.MaxLatency {
		lt.results.MaxLatency = latency
	}

	lt.results.StatusCodes[statusCode]++
	lt.results.mu.Unlock()
}

func (lt *LoadTester) recordError(errType string) {
	lt.results.mu.Lock()
	lt.results.Errors[errType]++
	lt.results.mu.Unlock()
}

func (lt *LoadTester) reportProgress() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-lt.stopCh:
			return
		case <-ticker.C:
			total := atomic.LoadInt64(&lt.results.TotalRequests)
			success := atomic.LoadInt64(&lt.results.SuccessRequests)
			failed := atomic.LoadInt64(&lt.results.FailedRequests)
			elapsed := time.Since(lt.results.StartTime).Seconds()
			rps := float64(total) / elapsed

			fmt.Printf("\r  Progress: %d requests (%.0f/s), Success: %d, Failed: %d",
				total, rps, success, failed)
		}
	}
}

func (lt *LoadTester) calculateMetrics() {
	elapsed := lt.results.EndTime.Sub(lt.results.StartTime).Seconds()
	lt.results.RequestsPerSecond = float64(lt.results.TotalRequests) / elapsed

	// Sort latencies for percentile calculation
	sort.Slice(lt.results.Latencies, func(i, j int) bool {
		return lt.results.Latencies[i] < lt.results.Latencies[j]
	})
}

func (lt *LoadTester) getPercentile(p float64) float64 {
	if len(lt.results.Latencies) == 0 {
		return 0
	}
	index := int(float64(len(lt.results.Latencies)) * p)
	if index >= len(lt.results.Latencies) {
		index = len(lt.results.Latencies) - 1
	}
	return float64(lt.results.Latencies[index]) / 1000 // Convert to ms
}

func (lt *LoadTester) printResults() {
	fmt.Println()
	fmt.Println()
	fmt.Println("╔══════════════════════════════════════════════════════════════╗")
	fmt.Println("║                      LOAD TEST RESULTS                       ║")
	fmt.Println("╚══════════════════════════════════════════════════════════════╝")
	fmt.Println()

	elapsed := lt.results.EndTime.Sub(lt.results.StartTime)
	avgLatency := float64(0)
	if lt.results.TotalRequests > 0 {
		avgLatency = float64(lt.results.TotalLatency) / float64(lt.results.TotalRequests) / 1000
	}

	successRate := float64(0)
	if lt.results.TotalRequests > 0 {
		successRate = float64(lt.results.SuccessRequests) / float64(lt.results.TotalRequests) * 100
	}

	fmt.Printf("Test Duration:        %v\n", elapsed.Round(time.Millisecond))
	fmt.Printf("Concurrency:          %d workers\n", lt.config.Concurrency)
	fmt.Println()

	fmt.Println("── Request Statistics ─────────────────────────────────────────")
	fmt.Printf("  Total Requests:     %d\n", lt.results.TotalRequests)
	fmt.Printf("  Successful:         %d (%.2f%%)\n", lt.results.SuccessRequests, successRate)
	fmt.Printf("  Failed:             %d (%.2f%%)\n", lt.results.FailedRequests, 100-successRate)
	fmt.Printf("  Requests/Second:    %.2f\n", lt.results.RequestsPerSecond)
	fmt.Println()

	fmt.Println("── Latency Statistics (ms) ────────────────────────────────────")
	fmt.Printf("  Min:                %.2f ms\n", float64(lt.results.MinLatency)/1000)
	fmt.Printf("  Max:                %.2f ms\n", float64(lt.results.MaxLatency)/1000)
	fmt.Printf("  Average:            %.2f ms\n", avgLatency)
	fmt.Printf("  P50 (Median):       %.2f ms\n", lt.getPercentile(0.50))
	fmt.Printf("  P90:                %.2f ms\n", lt.getPercentile(0.90))
	fmt.Printf("  P95:                %.2f ms\n", lt.getPercentile(0.95))
	fmt.Printf("  P99:                %.2f ms\n", lt.getPercentile(0.99))
	fmt.Println()

	fmt.Println("── Status Code Distribution ───────────────────────────────────")
	for code, count := range lt.results.StatusCodes {
		percentage := float64(count) / float64(lt.results.TotalRequests) * 100
		fmt.Printf("  HTTP %d:             %d (%.2f%%)\n", code, count, percentage)
	}
	fmt.Println()

	if len(lt.results.Errors) > 0 {
		fmt.Println("── Error Distribution ─────────────────────────────────────────")
		for errType, count := range lt.results.Errors {
			fmt.Printf("  %s: %d\n", errType, count)
		}
		fmt.Println()
	}

	// Generate summary assessment
	fmt.Println("── Assessment ─────────────────────────────────────────────────")
	if successRate >= 99.9 {
		fmt.Println("  ✅ Excellent: >99.9% success rate")
	} else if successRate >= 99 {
		fmt.Println("  ✅ Good: >99% success rate")
	} else if successRate >= 95 {
		fmt.Println("  ⚠️  Acceptable: >95% success rate")
	} else {
		fmt.Println("  ❌ Poor: <95% success rate")
	}

	if avgLatency < 10 {
		fmt.Println("  ✅ Excellent latency: <10ms average")
	} else if avgLatency < 50 {
		fmt.Println("  ✅ Good latency: <50ms average")
	} else if avgLatency < 200 {
		fmt.Println("  ⚠️  Acceptable latency: <200ms average")
	} else {
		fmt.Println("  ❌ High latency: >200ms average")
	}

	if lt.results.RequestsPerSecond > 1000 {
		fmt.Println("  ✅ High throughput: >1000 req/s")
	} else if lt.results.RequestsPerSecond > 500 {
		fmt.Println("  ✅ Good throughput: >500 req/s")
	} else if lt.results.RequestsPerSecond > 100 {
		fmt.Println("  ⚠️  Moderate throughput: >100 req/s")
	} else {
		fmt.Println("  ❌ Low throughput: <100 req/s")
	}

	fmt.Println()
	fmt.Println("══════════════════════════════════════════════════════════════")
}

func (lt *LoadTester) SaveReport(filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	elapsed := lt.results.EndTime.Sub(lt.results.StartTime)
	avgLatency := float64(0)
	if lt.results.TotalRequests > 0 {
		avgLatency = float64(lt.results.TotalLatency) / float64(lt.results.TotalRequests) / 1000
	}
	successRate := float64(0)
	if lt.results.TotalRequests > 0 {
		successRate = float64(lt.results.SuccessRequests) / float64(lt.results.TotalRequests) * 100
	}

	report := map[string]interface{}{
		"test_config": map[string]interface{}{
			"base_url":     lt.config.BaseURL,
			"concurrency":  lt.config.Concurrency,
			"duration":     lt.config.Duration.String(),
			"markets":      lt.config.Markets,
			"trader_count": lt.config.TraderCount,
		},
		"summary": map[string]interface{}{
			"test_duration":      elapsed.String(),
			"total_requests":     lt.results.TotalRequests,
			"success_requests":   lt.results.SuccessRequests,
			"failed_requests":    lt.results.FailedRequests,
			"success_rate":       fmt.Sprintf("%.2f%%", successRate),
			"requests_per_second": lt.results.RequestsPerSecond,
		},
		"latency": map[string]interface{}{
			"min_ms": float64(lt.results.MinLatency) / 1000,
			"max_ms": float64(lt.results.MaxLatency) / 1000,
			"avg_ms": avgLatency,
			"p50_ms": lt.getPercentile(0.50),
			"p90_ms": lt.getPercentile(0.90),
			"p95_ms": lt.getPercentile(0.95),
			"p99_ms": lt.getPercentile(0.99),
		},
		"status_codes": lt.results.StatusCodes,
		"errors":       lt.results.Errors,
		"timestamp":    time.Now().Format(time.RFC3339),
	}

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(report)
}

func main() {
	// Parse command line flags
	baseURL := flag.String("url", "http://localhost:8080", "API base URL")
	concurrency := flag.Int("c", 50, "Number of concurrent workers")
	duration := flag.Duration("d", 60*time.Second, "Test duration")
	rampUp := flag.Duration("ramp", 5*time.Second, "Ramp-up time")
	outputFile := flag.String("o", "", "Output JSON report file")
	realistic := flag.Bool("realistic", false, "Run realistic test suite")
	flag.Parse()

	if *realistic {
		runRealisticTests(*baseURL, *outputFile)
		return
	}

	config := &Config{
		BaseURL:     *baseURL,
		Concurrency: *concurrency,
		Duration:    *duration,
		RampUp:      *rampUp,
		Markets:     []string{"BTC-USDC", "ETH-USDC", "SOL-USDC"},
		TraderCount: 100,
	}

	tester := NewLoadTester(config)
	tester.Run()

	if *outputFile != "" {
		if err := tester.SaveReport(*outputFile); err != nil {
			fmt.Printf("Failed to save report: %v\n", err)
		} else {
			fmt.Printf("\nReport saved to: %s\n", *outputFile)
		}
	}
}
