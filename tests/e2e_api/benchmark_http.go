package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

type PlaceOrderRequest struct {
	MarketID string `json:"market_id"`
	Side     string `json:"side"`
	Type     string `json:"type"`
	Price    string `json:"price"`
	Quantity string `json:"quantity"`
}

type BenchmarkResult struct {
	TotalOrders    int64
	SuccessOrders  int64
	FailedOrders   int64
	TotalDuration  time.Duration
	Throughput     float64
	AvgLatency     time.Duration
	P50Latency     time.Duration
	P95Latency     time.Duration
	P99Latency     time.Duration
	MaxLatency     time.Duration
	MinLatency     time.Duration
}

func main() {
	// Command line flags
	baseURL := flag.String("url", "http://localhost:8080", "API base URL")
	numOrders := flag.Int("n", 1000, "Number of orders to place")
	concurrency := flag.Int("c", 8, "Number of concurrent workers")
	marketID := flag.String("market", "BTC-USDC", "Market ID")
	flag.Parse()

	fmt.Println("╔══════════════════════════════════════════════════════════════╗")
	fmt.Println("║           PerpDEX HTTP API Benchmark                         ║")
	fmt.Println("╚══════════════════════════════════════════════════════════════╝")
	fmt.Println()
	fmt.Printf("Target URL:    %s\n", *baseURL)
	fmt.Printf("Total Orders:  %d\n", *numOrders)
	fmt.Printf("Concurrency:   %d\n", *concurrency)
	fmt.Printf("Market:        %s\n", *marketID)
	fmt.Println()

	// Check health
	fmt.Print("Checking API health... ")
	resp, err := http.Get(*baseURL + "/health")
	if err != nil {
		fmt.Printf("FAILED: %v\n", err)
		return
	}
	resp.Body.Close()
	if resp.StatusCode != 200 {
		fmt.Printf("FAILED: status %d\n", resp.StatusCode)
		return
	}
	fmt.Println("OK")
	fmt.Println()

	// Run benchmark
	result := runBenchmark(*baseURL, *numOrders, *concurrency, *marketID)

	// Print results
	fmt.Println()
	fmt.Println("╔══════════════════════════════════════════════════════════════╗")
	fmt.Println("║                    Benchmark Results                         ║")
	fmt.Println("╚══════════════════════════════════════════════════════════════╝")
	fmt.Println()
	fmt.Printf("Total Orders:      %d\n", result.TotalOrders)
	fmt.Printf("Successful:        %d\n", result.SuccessOrders)
	fmt.Printf("Failed:            %d\n", result.FailedOrders)
	fmt.Printf("Success Rate:      %.2f%%\n", float64(result.SuccessOrders)/float64(result.TotalOrders)*100)
	fmt.Println()
	fmt.Printf("Total Duration:    %v\n", result.TotalDuration.Round(time.Millisecond))
	fmt.Printf("Throughput:        %.2f orders/sec\n", result.Throughput)
	fmt.Println()
	fmt.Println("Latency Distribution:")
	fmt.Printf("  Min:             %v\n", result.MinLatency.Round(time.Microsecond))
	fmt.Printf("  Avg:             %v\n", result.AvgLatency.Round(time.Microsecond))
	fmt.Printf("  P50:             %v\n", result.P50Latency.Round(time.Microsecond))
	fmt.Printf("  P95:             %v\n", result.P95Latency.Round(time.Microsecond))
	fmt.Printf("  P99:             %v\n", result.P99Latency.Round(time.Microsecond))
	fmt.Printf("  Max:             %v\n", result.MaxLatency.Round(time.Microsecond))
	fmt.Println()

	// Query final orderbook
	fmt.Println("Final Orderbook (port 8081):")
	resp, err = http.Get("http://localhost:8081/orderbook/" + *marketID)
	if err == nil {
		defer resp.Body.Close()
		var ob map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&ob)
		if bids, ok := ob["bids"].([]interface{}); ok {
			fmt.Printf("  Bids: %d levels\n", len(bids))
		}
		if asks, ok := ob["asks"].([]interface{}); ok {
			fmt.Printf("  Asks: %d levels\n", len(asks))
		}
	}
}

func runBenchmark(baseURL string, numOrders, concurrency int, marketID string) BenchmarkResult {
	var wg sync.WaitGroup
	var successCount, failedCount atomic.Int64
	latencies := make([]time.Duration, 0, numOrders)
	var latencyMu sync.Mutex

	ordersPerWorker := numOrders / concurrency
	client := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:        concurrency * 2,
			MaxIdleConnsPerHost: concurrency * 2,
			IdleConnTimeout:     90 * time.Second,
		},
	}

	fmt.Printf("Starting benchmark with %d workers...\n", concurrency)
	progressTicker := time.NewTicker(500 * time.Millisecond)
	done := make(chan bool)

	go func() {
		for {
			select {
			case <-progressTicker.C:
				total := successCount.Load() + failedCount.Load()
				fmt.Printf("\r  Progress: %d/%d orders (%.1f%%)", total, numOrders, float64(total)/float64(numOrders)*100)
			case <-done:
				progressTicker.Stop()
				return
			}
		}
	}()

	start := time.Now()

	for w := 0; w < concurrency; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			for i := 0; i < ordersPerWorker; i++ {
				// Alternate between buy and sell
				side := "buy"
				price := fmt.Sprintf("%d", 90000+i%1000)
				if i%2 == 1 {
					side = "sell"
					price = fmt.Sprintf("%d", 100000+i%1000)
				}

				reqBody := PlaceOrderRequest{
					MarketID: marketID,
					Side:     side,
					Type:     "limit",
					Price:    price,
					Quantity: "0.01",
				}
				body, _ := json.Marshal(reqBody)

				req, _ := http.NewRequest("POST", baseURL+"/v1/orders", bytes.NewBuffer(body))
				req.Header.Set("Content-Type", "application/json")
				req.Header.Set("X-Trader-Address", fmt.Sprintf("worker%d", workerID))

				orderStart := time.Now()
				resp, err := client.Do(req)
				latency := time.Since(orderStart)

				if err != nil {
					failedCount.Add(1)
					continue
				}
				io.Copy(io.Discard, resp.Body)
				resp.Body.Close()

				if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusCreated {
					successCount.Add(1)
					latencyMu.Lock()
					latencies = append(latencies, latency)
					latencyMu.Unlock()
				} else {
					failedCount.Add(1)
				}
			}
		}(w)
	}

	wg.Wait()
	totalDuration := time.Since(start)
	done <- true
	fmt.Println() // New line after progress

	// Calculate statistics
	result := BenchmarkResult{
		TotalOrders:   int64(numOrders),
		SuccessOrders: successCount.Load(),
		FailedOrders:  failedCount.Load(),
		TotalDuration: totalDuration,
		Throughput:    float64(numOrders) / totalDuration.Seconds(),
	}

	if len(latencies) > 0 {
		sort.Slice(latencies, func(i, j int) bool {
			return latencies[i] < latencies[j]
		})

		var totalLatency time.Duration
		for _, l := range latencies {
			totalLatency += l
		}

		result.AvgLatency = totalLatency / time.Duration(len(latencies))
		result.MinLatency = latencies[0]
		result.MaxLatency = latencies[len(latencies)-1]
		result.P50Latency = latencies[len(latencies)*50/100]
		result.P95Latency = latencies[len(latencies)*95/100]
		result.P99Latency = latencies[len(latencies)*99/100]
	}

	return result
}
