package keeper

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"runtime"
	"sort"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"cosmossdk.io/math"
	"github.com/openalpha/perp-dex/x/orderbook/types"
)

// ============================================================================
// E2E Stress Test Suite
// ============================================================================
// Comprehensive stress tests for order book implementations:
// 1. High throughput order ingestion
// 2. Concurrent read/write operations
// 3. Memory efficiency under load
// 4. Latency distribution analysis
// ============================================================================

// StressTestConfig defines parameters for stress testing
type StressTestConfig struct {
	OrderCount      int           // Total orders to process
	PriceLevels     int           // Number of unique price levels
	Concurrency     int           // Number of concurrent goroutines
	ReadRatio       float64       // Ratio of read vs write operations (0.0-1.0)
	Duration        time.Duration // Test duration (0 = run to completion)
	WarmupOrders    int           // Orders to add before timing starts
	CollectLatency  bool          // Whether to collect per-operation latency
	LatencySampling int           // Sample every N operations (0 = all)
}

// DefaultStressConfig returns default stress test configuration
func DefaultStressConfig() StressTestConfig {
	return StressTestConfig{
		OrderCount:      10000,
		PriceLevels:     100,
		Concurrency:     runtime.NumCPU(),
		ReadRatio:       0.3,
		Duration:        0,
		WarmupOrders:    1000,
		CollectLatency:  true,
		LatencySampling: 10,
	}
}

// StressTestResult holds the results of a stress test
type StressTestResult struct {
	Implementation    string          `json:"implementation"`
	Config            StressTestConfig `json:"config"`
	TotalOperations   int64           `json:"total_operations"`
	WriteOperations   int64           `json:"write_operations"`
	ReadOperations    int64           `json:"read_operations"`
	TotalDuration     time.Duration   `json:"total_duration_ns"`
	ThroughputOps     float64         `json:"throughput_ops_per_sec"`
	AvgLatencyNs      float64         `json:"avg_latency_ns"`
	P50LatencyNs      float64         `json:"p50_latency_ns"`
	P95LatencyNs      float64         `json:"p95_latency_ns"`
	P99LatencyNs      float64         `json:"p99_latency_ns"`
	MaxLatencyNs      float64         `json:"max_latency_ns"`
	MemoryAllocBytes  uint64          `json:"memory_alloc_bytes"`
	MemoryTotalAlloc  uint64          `json:"memory_total_alloc_bytes"`
	GCPauses          uint32          `json:"gc_pauses"`
	Errors            int64           `json:"errors"`
}

// LatencyRecorder efficiently records operation latencies
type LatencyRecorder struct {
	latencies []int64
	mu        sync.Mutex
	sampling  int
	counter   int64
}

func NewLatencyRecorder(sampling int, capacity int) *LatencyRecorder {
	return &LatencyRecorder{
		latencies: make([]int64, 0, capacity),
		sampling:  sampling,
	}
}

func (r *LatencyRecorder) Record(latency time.Duration) {
	if r.sampling > 0 {
		count := atomic.AddInt64(&r.counter, 1)
		if count%int64(r.sampling) != 0 {
			return
		}
	}
	r.mu.Lock()
	r.latencies = append(r.latencies, int64(latency))
	r.mu.Unlock()
}

func (r *LatencyRecorder) GetPercentiles() (p50, p95, p99, max float64) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if len(r.latencies) == 0 {
		return 0, 0, 0, 0
	}

	sorted := make([]int64, len(r.latencies))
	copy(sorted, r.latencies)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })

	n := len(sorted)
	p50 = float64(sorted[n*50/100])
	p95 = float64(sorted[n*95/100])
	p99 = float64(sorted[n*99/100])
	max = float64(sorted[n-1])
	return
}

func (r *LatencyRecorder) GetAverage() float64 {
	r.mu.Lock()
	defer r.mu.Unlock()

	if len(r.latencies) == 0 {
		return 0
	}

	var sum int64
	for _, l := range r.latencies {
		sum += l
	}
	return float64(sum) / float64(len(r.latencies))
}

// generateStressOrders generates orders with controlled price distribution
func generateStressOrders(n int, priceLevels int, marketID string) []*types.Order {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	orders := make([]*types.Order, n)

	basePrice := 50000.0
	priceStep := 1.0 // $1 per level

	for i := 0; i < n; i++ {
		var side types.Side
		var price math.LegacyDec

		levelOffset := r.Intn(priceLevels)

		if r.Float32() < 0.5 {
			side = types.SideBuy
			price = math.LegacyNewDecWithPrec(int64((basePrice-float64(levelOffset)*priceStep)*100), 2)
		} else {
			side = types.SideSell
			price = math.LegacyNewDecWithPrec(int64((basePrice+float64(levelOffset)*priceStep)*100), 2)
		}

		quantity := math.LegacyNewDecWithPrec(int64((0.1+r.Float64()*10)*1000), 3)

		orders[i] = types.NewOrder(
			fmt.Sprintf("stress-order-%d", i),
			fmt.Sprintf("trader-%d", i%1000),
			marketID,
			side,
			types.OrderTypeLimit,
			price,
			quantity,
		)
	}

	return orders
}

// runStressTest executes a stress test on a single order book implementation
func runStressTest(engine OrderBookEngine, config StressTestConfig, name string) StressTestResult {
	marketID := engine.GetMarketID()
	orders := generateStressOrders(config.OrderCount+config.WarmupOrders, config.PriceLevels, marketID)

	// Warmup phase
	for i := 0; i < config.WarmupOrders; i++ {
		engine.AddOrder(orders[i])
	}
	orders = orders[config.WarmupOrders:]

	// Force GC before measurement
	runtime.GC()
	var memBefore runtime.MemStats
	runtime.ReadMemStats(&memBefore)

	recorder := NewLatencyRecorder(config.LatencySampling, config.OrderCount/config.LatencySampling)

	var writeOps, readOps, errors int64
	var wg sync.WaitGroup

	startTime := time.Now()

	if config.Concurrency <= 1 {
		// Single-threaded execution
		r := rand.New(rand.NewSource(time.Now().UnixNano()))
		addedOrders := make([]*types.Order, 0, len(orders))

		for i, order := range orders {
			opStart := time.Now()

			if r.Float64() < config.ReadRatio && len(addedOrders) > 0 {
				// Read operation
				switch r.Intn(3) {
				case 0:
					_, _ = engine.GetBestLevels()
				case 1:
					_ = engine.GetBidLevels(10)
				case 2:
					_ = engine.GetSpread()
				}
				readOps++
			} else {
				// Write operation
				if i%2 == 0 || len(addedOrders) == 0 {
					engine.AddOrder(order)
					addedOrders = append(addedOrders, order)
				} else if len(addedOrders) > 0 {
					idx := r.Intn(len(addedOrders))
					engine.RemoveOrder(addedOrders[idx])
					addedOrders = append(addedOrders[:idx], addedOrders[idx+1:]...)
				}
				writeOps++
			}

			if config.CollectLatency {
				recorder.Record(time.Since(opStart))
			}
		}
	} else {
		// Concurrent execution
		ordersPerWorker := len(orders) / config.Concurrency
		writeOpsAtomic := &writeOps
		readOpsAtomic := &readOps
		_ = &errors // Reserved for future error tracking

		for w := 0; w < config.Concurrency; w++ {
			wg.Add(1)
			start := w * ordersPerWorker
			end := start + ordersPerWorker
			if w == config.Concurrency-1 {
				end = len(orders)
			}

			go func(workerOrders []*types.Order) {
				defer wg.Done()
				r := rand.New(rand.NewSource(time.Now().UnixNano()))
				localAdded := make([]*types.Order, 0, len(workerOrders))

				for i, order := range workerOrders {
					opStart := time.Now()

					if r.Float64() < config.ReadRatio && len(localAdded) > 0 {
						switch r.Intn(3) {
						case 0:
							_, _ = engine.GetBestLevels()
						case 1:
							_ = engine.GetBidLevels(10)
						case 2:
							_ = engine.GetSpread()
						}
						atomic.AddInt64(readOpsAtomic, 1)
					} else {
						if i%2 == 0 || len(localAdded) == 0 {
							engine.AddOrder(order)
							localAdded = append(localAdded, order)
						} else if len(localAdded) > 0 {
							idx := r.Intn(len(localAdded))
							engine.RemoveOrder(localAdded[idx])
							localAdded = append(localAdded[:idx], localAdded[idx+1:]...)
						}
						atomic.AddInt64(writeOpsAtomic, 1)
					}

					if config.CollectLatency {
						recorder.Record(time.Since(opStart))
					}
				}
			}(orders[start:end])
		}
		wg.Wait()
	}

	duration := time.Since(startTime)

	// Collect memory stats
	var memAfter runtime.MemStats
	runtime.ReadMemStats(&memAfter)

	totalOps := writeOps + readOps
	p50, p95, p99, maxLat := recorder.GetPercentiles()

	return StressTestResult{
		Implementation:    name,
		Config:            config,
		TotalOperations:   totalOps,
		WriteOperations:   writeOps,
		ReadOperations:    readOps,
		TotalDuration:     duration,
		ThroughputOps:     float64(totalOps) / duration.Seconds(),
		AvgLatencyNs:      recorder.GetAverage(),
		P50LatencyNs:      p50,
		P95LatencyNs:      p95,
		P99LatencyNs:      p99,
		MaxLatencyNs:      maxLat,
		MemoryAllocBytes:  memAfter.Alloc - memBefore.Alloc,
		MemoryTotalAlloc:  memAfter.TotalAlloc - memBefore.TotalAlloc,
		GCPauses:          memAfter.NumGC - memBefore.NumGC,
		Errors:            errors,
	}
}

// TestE2EStressAllImplementations runs stress tests on all implementations
func TestE2EStressAllImplementations(t *testing.T) {
	config := DefaultStressConfig()
	config.OrderCount = 50000
	config.WarmupOrders = 5000
	config.Concurrency = 1 // Single-threaded for accurate comparison

	marketID := "BTC-USD"

	implementations := []struct {
		name   string
		create func() OrderBookEngine
	}{
		{"SkipList", func() OrderBookEngine { return NewOrderBookV2(marketID) }},
		{"HashMap", func() OrderBookEngine { return NewOrderBookHashMap(marketID) }},
		{"BTree", func() OrderBookEngine { return NewOrderBookBTree(marketID) }},
		{"ART", func() OrderBookEngine { return NewOrderBookART(marketID) }},
	}

	results := make([]StressTestResult, 0, len(implementations))

	for _, impl := range implementations {
		t.Run(impl.name, func(t *testing.T) {
			engine := impl.create()
			result := runStressTest(engine, config, impl.name)
			results = append(results, result)

			t.Logf("Implementation: %s", impl.name)
			t.Logf("  Throughput: %.2f ops/sec", result.ThroughputOps)
			t.Logf("  Avg Latency: %.2f ns", result.AvgLatencyNs)
			t.Logf("  P50 Latency: %.2f ns", result.P50LatencyNs)
			t.Logf("  P95 Latency: %.2f ns", result.P95LatencyNs)
			t.Logf("  P99 Latency: %.2f ns", result.P99LatencyNs)
			t.Logf("  Max Latency: %.2f ns", result.MaxLatencyNs)
			t.Logf("  Memory Alloc: %.2f MB", float64(result.MemoryAllocBytes)/1024/1024)
			t.Logf("  GC Pauses: %d", result.GCPauses)
		})
	}

	// Generate JSON report
	reportJSON, _ := json.MarshalIndent(results, "", "  ")
	t.Logf("Full Report:\n%s", string(reportJSON))
}

// TestE2EConcurrentStress tests concurrent access patterns
func TestE2EConcurrentStress(t *testing.T) {
	config := DefaultStressConfig()
	config.OrderCount = 20000
	config.WarmupOrders = 2000
	config.Concurrency = runtime.NumCPU()

	marketID := "BTC-USD"

	implementations := []struct {
		name   string
		create func() OrderBookEngine
	}{
		{"SkipList", func() OrderBookEngine { return NewOrderBookV2(marketID) }},
		{"HashMap", func() OrderBookEngine { return NewOrderBookHashMap(marketID) }},
		{"BTree", func() OrderBookEngine { return NewOrderBookBTree(marketID) }},
		{"ART", func() OrderBookEngine { return NewOrderBookART(marketID) }},
	}

	t.Logf("Running concurrent stress test with %d goroutines", config.Concurrency)

	for _, impl := range implementations {
		t.Run(impl.name+"_Concurrent", func(t *testing.T) {
			engine := impl.create()
			result := runStressTest(engine, config, impl.name)

			t.Logf("Implementation: %s (Concurrent)", impl.name)
			t.Logf("  Throughput: %.2f ops/sec", result.ThroughputOps)
			t.Logf("  Write Ops: %d, Read Ops: %d", result.WriteOperations, result.ReadOperations)
			t.Logf("  P99 Latency: %.2f ns", result.P99LatencyNs)
		})
	}
}

// TestE2EHighReadRatio tests performance under high read load
func TestE2EHighReadRatio(t *testing.T) {
	config := DefaultStressConfig()
	config.OrderCount = 30000
	config.WarmupOrders = 5000
	config.ReadRatio = 0.8 // 80% reads
	config.Concurrency = 1

	marketID := "BTC-USD"

	implementations := []struct {
		name   string
		create func() OrderBookEngine
	}{
		{"SkipList", func() OrderBookEngine { return NewOrderBookV2(marketID) }},
		{"HashMap", func() OrderBookEngine { return NewOrderBookHashMap(marketID) }},
		{"BTree", func() OrderBookEngine { return NewOrderBookBTree(marketID) }},
		{"ART", func() OrderBookEngine { return NewOrderBookART(marketID) }},
	}

	t.Logf("Running high read ratio stress test (80%% reads)")

	for _, impl := range implementations {
		t.Run(impl.name+"_HighRead", func(t *testing.T) {
			engine := impl.create()
			result := runStressTest(engine, config, impl.name)

			t.Logf("Implementation: %s (High Read)", impl.name)
			t.Logf("  Throughput: %.2f ops/sec", result.ThroughputOps)
			t.Logf("  Read Ops: %d (%.1f%%)", result.ReadOperations,
				float64(result.ReadOperations)/float64(result.TotalOperations)*100)
			t.Logf("  P95 Latency: %.2f ns", result.P95LatencyNs)
		})
	}
}

// TestE2EMemoryPressure tests behavior under memory pressure
func TestE2EMemoryPressure(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping memory pressure test in short mode")
	}

	config := DefaultStressConfig()
	config.OrderCount = 100000
	config.WarmupOrders = 10000
	config.PriceLevels = 1000 // More price levels = more memory
	config.Concurrency = 1
	config.CollectLatency = false // Reduce overhead

	marketID := "BTC-USD"

	implementations := []struct {
		name   string
		create func() OrderBookEngine
	}{
		{"SkipList", func() OrderBookEngine { return NewOrderBookV2(marketID) }},
		{"HashMap", func() OrderBookEngine { return NewOrderBookHashMap(marketID) }},
		{"BTree", func() OrderBookEngine { return NewOrderBookBTree(marketID) }},
		{"ART", func() OrderBookEngine { return NewOrderBookART(marketID) }},
	}

	t.Logf("Running memory pressure test (100K orders, 1000 price levels)")

	for _, impl := range implementations {
		t.Run(impl.name+"_MemPressure", func(t *testing.T) {
			engine := impl.create()

			runtime.GC()
			var memBefore runtime.MemStats
			runtime.ReadMemStats(&memBefore)

			result := runStressTest(engine, config, impl.name)

			runtime.GC()
			var memAfter runtime.MemStats
			runtime.ReadMemStats(&memAfter)

			t.Logf("Implementation: %s (Memory Pressure)", impl.name)
			t.Logf("  Throughput: %.2f ops/sec", result.ThroughputOps)
			t.Logf("  Memory Growth: %.2f MB", float64(memAfter.Alloc-memBefore.Alloc)/1024/1024)
			t.Logf("  Total Alloc: %.2f MB", float64(result.MemoryTotalAlloc)/1024/1024)
			t.Logf("  GC Pauses: %d", result.GCPauses)
		})
	}
}

// BenchmarkE2EThroughput benchmarks raw throughput
func BenchmarkE2EThroughput(b *testing.B) {
	marketID := "BTC-USD"
	orders := generateStressOrders(b.N+1000, 100, marketID)

	implementations := []struct {
		name   string
		create func() OrderBookEngine
	}{
		{"SkipList", func() OrderBookEngine { return NewOrderBookV2(marketID) }},
		{"HashMap", func() OrderBookEngine { return NewOrderBookHashMap(marketID) }},
		{"BTree", func() OrderBookEngine { return NewOrderBookBTree(marketID) }},
		{"ART", func() OrderBookEngine { return NewOrderBookART(marketID) }},
	}

	for _, impl := range implementations {
		b.Run(impl.name, func(b *testing.B) {
			engine := impl.create()

			// Warmup
			for i := 0; i < 1000; i++ {
				engine.AddOrder(orders[i])
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				engine.AddOrder(orders[1000+i%len(orders[1000:])])
			}
		})
	}
}

// GenerateStressReport generates a comprehensive stress test report
func GenerateStressReport(outputPath string) error {
	config := DefaultStressConfig()
	config.OrderCount = 50000
	config.WarmupOrders = 5000

	marketID := "BTC-USD"

	implementations := []struct {
		name   string
		create func() OrderBookEngine
	}{
		{"SkipList", func() OrderBookEngine { return NewOrderBookV2(marketID) }},
		{"HashMap", func() OrderBookEngine { return NewOrderBookHashMap(marketID) }},
		{"BTree", func() OrderBookEngine { return NewOrderBookBTree(marketID) }},
		{"ART", func() OrderBookEngine { return NewOrderBookART(marketID) }},
	}

	report := struct {
		GeneratedAt time.Time           `json:"generated_at"`
		CPUs        int                 `json:"cpus"`
		GOOS        string              `json:"goos"`
		GOARCH      string              `json:"goarch"`
		Results     []StressTestResult  `json:"results"`
	}{
		GeneratedAt: time.Now(),
		CPUs:        runtime.NumCPU(),
		GOOS:        runtime.GOOS,
		GOARCH:      runtime.GOARCH,
		Results:     make([]StressTestResult, 0, len(implementations)),
	}

	for _, impl := range implementations {
		engine := impl.create()
		result := runStressTest(engine, config, impl.name)
		report.Results = append(report.Results, result)
	}

	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(outputPath, data, 0644)
}
