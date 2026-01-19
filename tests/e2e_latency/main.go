package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	hlAPIURL = "https://api.hyperliquid.xyz/info"
	hlWSURL  = "wss://api.hyperliquid.xyz/ws"
)

// LatencyStats holds latency statistics
type LatencyStats struct {
	Name      string    `json:"name"`
	Count     int       `json:"count"`
	Min       float64   `json:"min_ms"`
	Max       float64   `json:"max_ms"`
	Avg       float64   `json:"avg_ms"`
	P50       float64   `json:"p50_ms"`
	P90       float64   `json:"p90_ms"`
	P99       float64   `json:"p99_ms"`
	Latencies []float64 `json:"-"`
}

// TestReport holds the complete test report
type TestReport struct {
	Timestamp   string                   `json:"timestamp"`
	RESTResults map[string]*LatencyStats `json:"rest_api"`
	WSResults   map[string]*LatencyStats `json:"websocket"`
	Summary     map[string]interface{}   `json:"summary"`
}

func main() {
	iterations := flag.Int("n", 50, "Number of iterations for REST API tests")
	wsDuration := flag.Int("ws-duration", 30, "WebSocket test duration in seconds")
	outputFile := flag.String("o", "", "Output JSON file")
	restOnly := flag.Bool("rest-only", false, "Run REST API tests only")
	wsOnly := flag.Bool("ws-only", false, "Run WebSocket tests only")
	flag.Parse()

	fmt.Println("╔══════════════════════════════════════════════════════════════╗")
	fmt.Println("║        Hyperliquid E2E Latency Test Suite                    ║")
	fmt.Println("╚══════════════════════════════════════════════════════════════╝")
	fmt.Println()

	report := &TestReport{
		Timestamp:   time.Now().Format(time.RFC3339),
		RESTResults: make(map[string]*LatencyStats),
		WSResults:   make(map[string]*LatencyStats),
		Summary:     make(map[string]interface{}),
	}

	// Run REST API tests
	if !*wsOnly {
		fmt.Println("═══ REST API Latency Tests ═══")
		fmt.Printf("Iterations: %d\n\n", *iterations)
		runRESTTests(report, *iterations)
	}

	// Run WebSocket tests
	if !*restOnly {
		fmt.Println("\n═══ WebSocket Latency Tests ═══")
		fmt.Printf("Duration: %d seconds\n\n", *wsDuration)
		runWSTests(report, *wsDuration)
	}

	// Print summary
	printSummary(report)

	// Save to file if specified
	if *outputFile != "" {
		saveReport(report, *outputFile)
	}
}

func runRESTTests(report *TestReport, iterations int) {
	endpoints := []struct {
		name    string
		payload map[string]interface{}
	}{
		{
			name:    "metaAndAssetCtxs",
			payload: map[string]interface{}{"type": "metaAndAssetCtxs"},
		},
		{
			name:    "l2Book (BTC)",
			payload: map[string]interface{}{"type": "l2Book", "coin": "BTC"},
		},
		{
			name:    "l2Book (ETH)",
			payload: map[string]interface{}{"type": "l2Book", "coin": "ETH"},
		},
		{
			name:    "recentTrades (BTC)",
			payload: map[string]interface{}{"type": "recentTrades", "coin": "BTC"},
		},
	}

	// Use connection pooling for accurate latency measurement
	transport := &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 100,
		IdleConnTimeout:     90 * time.Second,
	}
	client := &http.Client{
		Timeout:   30 * time.Second,
		Transport: transport,
	}

	for _, ep := range endpoints {
		stats := &LatencyStats{
			Name:      ep.name,
			Latencies: make([]float64, 0, iterations),
		}

		fmt.Printf("Testing %s... ", ep.name)

		for i := 0; i < iterations; i++ {
			latency, err := measureRESTLatency(client, ep.payload)
			if err != nil {
				fmt.Printf("\n  Error on iteration %d: %v\n", i+1, err)
				continue
			}
			stats.Latencies = append(stats.Latencies, latency)
		}

		calculateStats(stats)
		report.RESTResults[ep.name] = stats

		fmt.Printf("done\n")
		fmt.Printf("  P50: %.2fms  P90: %.2fms  P99: %.2fms  Avg: %.2fms\n",
			stats.P50, stats.P90, stats.P99, stats.Avg)
	}
}

func measureRESTLatency(client *http.Client, payload map[string]interface{}) (float64, error) {
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return 0, err
	}

	start := time.Now()

	resp, err := client.Post(hlAPIURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	// Read full response to measure complete latency
	_, err = io.ReadAll(resp.Body)
	if err != nil {
		return 0, err
	}

	latency := float64(time.Since(start).Microseconds()) / 1000.0

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	return latency, nil
}

func runWSTests(report *TestReport, durationSec int) {
	// Test 1: Connection establishment time
	fmt.Print("Testing connection establishment... ")
	connStats := testWSConnection(10)
	report.WSResults["connection"] = connStats
	fmt.Printf("done\n")
	fmt.Printf("  P50: %.2fms  P90: %.2fms  P99: %.2fms\n", connStats.P50, connStats.P90, connStats.P99)

	// Test 2: Message latency
	fmt.Printf("Testing message latency (%ds)... ", durationSec)
	msgStats := testWSMessageLatency(durationSec)
	report.WSResults["allMids_messages"] = msgStats
	fmt.Printf("done\n")
	fmt.Printf("  Messages: %d  Avg interval: %.2fms\n", msgStats.Count, msgStats.Avg)
}

func testWSConnection(iterations int) *LatencyStats {
	stats := &LatencyStats{
		Name:      "connection",
		Latencies: make([]float64, 0, iterations),
	}

	for i := 0; i < iterations; i++ {
		start := time.Now()

		conn, _, err := websocket.DefaultDialer.Dial(hlWSURL, nil)
		if err != nil {
			continue
		}

		latency := float64(time.Since(start).Microseconds()) / 1000.0
		stats.Latencies = append(stats.Latencies, latency)

		conn.Close()
		time.Sleep(100 * time.Millisecond) // Small delay between connections
	}

	calculateStats(stats)
	return stats
}

func testWSMessageLatency(durationSec int) *LatencyStats {
	stats := &LatencyStats{
		Name:      "allMids_messages",
		Latencies: make([]float64, 0),
	}

	conn, _, err := websocket.DefaultDialer.Dial(hlWSURL, nil)
	if err != nil {
		fmt.Printf("Connection failed: %v\n", err)
		return stats
	}
	defer conn.Close()

	// Subscribe to allMids
	subscribeMsg := map[string]interface{}{
		"method": "subscribe",
		"subscription": map[string]interface{}{
			"type": "allMids",
		},
	}

	if err := conn.WriteJSON(subscribeMsg); err != nil {
		fmt.Printf("Subscribe failed: %v\n", err)
		return stats
	}

	// Measure message intervals
	var mu sync.Mutex
	done := make(chan struct{})
	lastMsgTime := time.Now()
	firstMsg := true

	go func() {
		for {
			select {
			case <-done:
				return
			default:
				conn.SetReadDeadline(time.Now().Add(5 * time.Second))
				_, _, err := conn.ReadMessage()
				if err != nil {
					continue
				}

				now := time.Now()
				if !firstMsg {
					interval := float64(now.Sub(lastMsgTime).Microseconds()) / 1000.0
					mu.Lock()
					stats.Latencies = append(stats.Latencies, interval)
					mu.Unlock()
				}
				firstMsg = false
				lastMsgTime = now
			}
		}
	}()

	time.Sleep(time.Duration(durationSec) * time.Second)
	close(done)

	calculateStats(stats)
	return stats
}

func calculateStats(stats *LatencyStats) {
	if len(stats.Latencies) == 0 {
		return
	}

	stats.Count = len(stats.Latencies)

	// Sort for percentile calculation
	sorted := make([]float64, len(stats.Latencies))
	copy(sorted, stats.Latencies)
	sort.Float64s(sorted)

	// Calculate min, max, avg
	stats.Min = sorted[0]
	stats.Max = sorted[len(sorted)-1]

	var sum float64
	for _, v := range sorted {
		sum += v
	}
	stats.Avg = sum / float64(len(sorted))

	// Calculate percentiles
	stats.P50 = percentile(sorted, 50)
	stats.P90 = percentile(sorted, 90)
	stats.P99 = percentile(sorted, 99)
}

func percentile(sorted []float64, p int) float64 {
	if len(sorted) == 0 {
		return 0
	}
	idx := int(float64(p) / 100.0 * float64(len(sorted)-1))
	if idx >= len(sorted) {
		idx = len(sorted) - 1
	}
	return sorted[idx]
}

func printSummary(report *TestReport) {
	fmt.Println("\n╔══════════════════════════════════════════════════════════════╗")
	fmt.Println("║                    测试结果汇总                              ║")
	fmt.Println("╚══════════════════════════════════════════════════════════════╝")

	fmt.Println("\n【REST API 延迟】")
	fmt.Println("┌────────────────────────┬─────────┬─────────┬─────────┬─────────┐")
	fmt.Println("│ Endpoint               │  P50    │  P90    │  P99    │  Avg    │")
	fmt.Println("├────────────────────────┼─────────┼─────────┼─────────┼─────────┤")

	for name, stats := range report.RESTResults {
		fmt.Printf("│ %-22s │ %6.1fms│ %6.1fms│ %6.1fms│ %6.1fms│\n",
			truncate(name, 22), stats.P50, stats.P90, stats.P99, stats.Avg)
	}
	fmt.Println("└────────────────────────┴─────────┴─────────┴─────────┴─────────┘")

	if len(report.WSResults) > 0 {
		fmt.Println("\n【WebSocket 延迟】")
		fmt.Println("┌────────────────────────┬─────────┬─────────┬─────────┬─────────┐")
		fmt.Println("│ Metric                 │  P50    │  P90    │  P99    │  Count  │")
		fmt.Println("├────────────────────────┼─────────┼─────────┼─────────┼─────────┤")

		for name, stats := range report.WSResults {
			fmt.Printf("│ %-22s │ %6.1fms│ %6.1fms│ %6.1fms│ %7d │\n",
				truncate(name, 22), stats.P50, stats.P90, stats.P99, stats.Count)
		}
		fmt.Println("└────────────────────────┴─────────┴─────────┴─────────┴─────────┘")
	}

	// Comparison with mock mode
	fmt.Println("\n【与 Mock 模式对比】")
	fmt.Println("┌────────────────────────┬──────────────┬──────────────┐")
	fmt.Println("│ 指标                   │  Mock 模式   │  真实 API    │")
	fmt.Println("├────────────────────────┼──────────────┼──────────────┤")
	fmt.Println("│ 本地后端 API P50       │    0.07 ms   │    0.07 ms   │")
	fmt.Println("│ 本地后端 API P99       │    0.20 ms   │    0.20 ms   │")

	if stats, ok := report.RESTResults["metaAndAssetCtxs"]; ok {
		fmt.Printf("│ 价格数据获取 P50       │    ~0 ms     │  %6.1f ms   │\n", stats.P50)
	}
	if stats, ok := report.RESTResults["l2Book (BTC)"]; ok {
		fmt.Printf("│ 订单簿获取 P50         │    ~0 ms     │  %6.1f ms   │\n", stats.P50)
	}

	fmt.Println("└────────────────────────┴──────────────┴──────────────┘")

	// Calculate total latency
	var totalP50 float64
	if stats, ok := report.RESTResults["metaAndAssetCtxs"]; ok {
		totalP50 += stats.P50
	}
	totalP50 += 0.07 // Local API latency

	fmt.Printf("\n📊 完整链路预估延迟 (P50): %.1f ms\n", totalP50)
	fmt.Println("   └── Hyperliquid API + 本地后端处理")
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

func saveReport(report *TestReport, filename string) {
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		fmt.Printf("Error marshaling report: %v\n", err)
		return
	}

	if err := os.WriteFile(filename, data, 0644); err != nil {
		fmt.Printf("Error saving report: %v\n", err)
		return
	}

	fmt.Printf("\n✅ Report saved to: %s\n", filename)
}
