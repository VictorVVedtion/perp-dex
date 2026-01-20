// Package performance provides performance metrics collection and analysis for PerpDEX testing
package performance

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"sort"
	"sync"
	"time"
)

// LatencyStats holds detailed latency statistics
type LatencyStats struct {
	Min   time.Duration `json:"min"`
	Max   time.Duration `json:"max"`
	Avg   time.Duration `json:"avg"`
	P50   time.Duration `json:"p50"`
	P90   time.Duration `json:"p90"`
	P95   time.Duration `json:"p95"`
	P99   time.Duration `json:"p99"`
	Count int           `json:"count"`
}

// PerformanceMetrics holds comprehensive performance metrics
type PerformanceMetrics struct {
	// Test identification
	TestName    string    `json:"test_name"`
	StartTime   time.Time `json:"start_time"`
	EndTime     time.Time `json:"end_time"`
	Duration    time.Duration `json:"duration"`

	// Latency statistics
	Latencies   *LatencyStats `json:"latencies"`

	// Throughput
	Throughput  float64 `json:"throughput_ops_per_sec"`

	// Success metrics
	TotalOps    int64   `json:"total_ops"`
	SuccessOps  int64   `json:"success_ops"`
	FailedOps   int64   `json:"failed_ops"`
	SuccessRate float64 `json:"success_rate_percent"`

	// Resource usage
	MemoryUsage    int64 `json:"memory_usage_bytes"`
	PeakMemory     int64 `json:"peak_memory_bytes"`
	AllocsPerOp    int64 `json:"allocs_per_op"`
	BytesPerOp     int64 `json:"bytes_per_op"`

	// Block-related metrics (for chain tests)
	BlocksProduced   int64         `json:"blocks_produced"`
	AvgBlockTime     time.Duration `json:"avg_block_time"`
	OrdersPerBlock   float64       `json:"orders_per_block"`
	TradesExecuted   int64         `json:"trades_executed"`

	// Gas metrics
	TotalGasUsed     int64 `json:"total_gas_used"`
	AvgGasPerTx      int64 `json:"avg_gas_per_tx"`
}

// MetricsCollector collects performance metrics during test execution
type MetricsCollector struct {
	mu          sync.Mutex
	testName    string
	startTime   time.Time
	latencies   []time.Duration
	successOps  int64
	failedOps   int64
	gasUsed     int64
	tradesCount int64
	memoryPeak  int64
}

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector(testName string) *MetricsCollector {
	return &MetricsCollector{
		testName:  testName,
		startTime: time.Now(),
		latencies: make([]time.Duration, 0, 10000),
	}
}

// RecordLatency records a single operation latency
func (mc *MetricsCollector) RecordLatency(d time.Duration) {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	mc.latencies = append(mc.latencies, d)
}

// RecordSuccess records a successful operation
func (mc *MetricsCollector) RecordSuccess() {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	mc.successOps++
}

// RecordFailure records a failed operation
func (mc *MetricsCollector) RecordFailure() {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	mc.failedOps++
}

// RecordGas records gas usage
func (mc *MetricsCollector) RecordGas(gas int64) {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	mc.gasUsed += gas
}

// RecordTrades records executed trades count
func (mc *MetricsCollector) RecordTrades(count int) {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	mc.tradesCount += int64(count)
}

// RecordOperation records a complete operation with latency and success status
func (mc *MetricsCollector) RecordOperation(latency time.Duration, success bool, gas int64) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	mc.latencies = append(mc.latencies, latency)
	if success {
		mc.successOps++
	} else {
		mc.failedOps++
	}
	mc.gasUsed += gas
}

// CalculateLatencyStats computes percentile statistics from latency samples
func CalculateLatencyStats(latencies []time.Duration) *LatencyStats {
	if len(latencies) == 0 {
		return &LatencyStats{}
	}

	// Sort for percentile calculation
	sorted := make([]time.Duration, len(latencies))
	copy(sorted, latencies)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i] < sorted[j]
	})

	// Calculate sum for average
	var sum time.Duration
	for _, l := range sorted {
		sum += l
	}

	return &LatencyStats{
		Min:   sorted[0],
		Max:   sorted[len(sorted)-1],
		Avg:   sum / time.Duration(len(sorted)),
		P50:   percentile(sorted, 50),
		P90:   percentile(sorted, 90),
		P95:   percentile(sorted, 95),
		P99:   percentile(sorted, 99),
		Count: len(sorted),
	}
}

// percentile calculates the p-th percentile of sorted durations
func percentile(sorted []time.Duration, p int) time.Duration {
	if len(sorted) == 0 {
		return 0
	}
	if p >= 100 {
		return sorted[len(sorted)-1]
	}
	if p <= 0 {
		return sorted[0]
	}

	index := float64(len(sorted)-1) * float64(p) / 100.0
	lower := int(math.Floor(index))
	upper := int(math.Ceil(index))

	if lower == upper {
		return sorted[lower]
	}

	// Linear interpolation
	weight := index - float64(lower)
	return time.Duration(float64(sorted[lower])*(1-weight) + float64(sorted[upper])*weight)
}

// GetMetrics computes final metrics from collected data
func (mc *MetricsCollector) GetMetrics() *PerformanceMetrics {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	endTime := time.Now()
	duration := endTime.Sub(mc.startTime)
	totalOps := mc.successOps + mc.failedOps

	metrics := &PerformanceMetrics{
		TestName:    mc.testName,
		StartTime:   mc.startTime,
		EndTime:     endTime,
		Duration:    duration,
		Latencies:   CalculateLatencyStats(mc.latencies),
		TotalOps:    totalOps,
		SuccessOps:  mc.successOps,
		FailedOps:   mc.failedOps,
		TotalGasUsed:   mc.gasUsed,
		TradesExecuted: mc.tradesCount,
	}

	// Calculate derived metrics
	if totalOps > 0 {
		metrics.SuccessRate = float64(mc.successOps) / float64(totalOps) * 100
		metrics.AvgGasPerTx = mc.gasUsed / totalOps
	}

	if duration.Seconds() > 0 {
		metrics.Throughput = float64(totalOps) / duration.Seconds()
	}

	return metrics
}

// PrintReport prints a formatted performance report
func (m *PerformanceMetrics) PrintReport() {
	fmt.Println("╔══════════════════════════════════════════════════════════════╗")
	fmt.Printf("║  Performance Report: %-40s ║\n", m.TestName)
	fmt.Println("╠══════════════════════════════════════════════════════════════╣")
	fmt.Printf("║  Duration:        %-43v ║\n", m.Duration.Round(time.Millisecond))
	fmt.Printf("║  Total Operations: %-42d ║\n", m.TotalOps)
	fmt.Printf("║  Successful:       %-42d ║\n", m.SuccessOps)
	fmt.Printf("║  Failed:           %-42d ║\n", m.FailedOps)
	fmt.Printf("║  Success Rate:     %-41.2f%% ║\n", m.SuccessRate)
	fmt.Printf("║  Throughput:       %-37.2f ops/sec ║\n", m.Throughput)
	fmt.Println("╠══════════════════════════════════════════════════════════════╣")
	fmt.Println("║  Latency Statistics                                          ║")
	fmt.Println("╠══════════════════════════════════════════════════════════════╣")
	if m.Latencies != nil {
		fmt.Printf("║  Min:              %-43v ║\n", m.Latencies.Min)
		fmt.Printf("║  Max:              %-43v ║\n", m.Latencies.Max)
		fmt.Printf("║  Avg:              %-43v ║\n", m.Latencies.Avg)
		fmt.Printf("║  P50:              %-43v ║\n", m.Latencies.P50)
		fmt.Printf("║  P90:              %-43v ║\n", m.Latencies.P90)
		fmt.Printf("║  P95:              %-43v ║\n", m.Latencies.P95)
		fmt.Printf("║  P99:              %-43v ║\n", m.Latencies.P99)
	}
	if m.TotalGasUsed > 0 {
		fmt.Println("╠══════════════════════════════════════════════════════════════╣")
		fmt.Println("║  Gas Metrics                                                 ║")
		fmt.Println("╠══════════════════════════════════════════════════════════════╣")
		fmt.Printf("║  Total Gas Used:   %-42d ║\n", m.TotalGasUsed)
		fmt.Printf("║  Avg Gas/Tx:       %-42d ║\n", m.AvgGasPerTx)
	}
	if m.TradesExecuted > 0 {
		fmt.Printf("║  Trades Executed:  %-42d ║\n", m.TradesExecuted)
	}
	if m.BlocksProduced > 0 {
		fmt.Println("╠══════════════════════════════════════════════════════════════╣")
		fmt.Println("║  Block Metrics                                               ║")
		fmt.Println("╠══════════════════════════════════════════════════════════════╣")
		fmt.Printf("║  Blocks Produced:  %-42d ║\n", m.BlocksProduced)
		fmt.Printf("║  Avg Block Time:   %-43v ║\n", m.AvgBlockTime)
		fmt.Printf("║  Orders/Block:     %-42.2f ║\n", m.OrdersPerBlock)
	}
	fmt.Println("╚══════════════════════════════════════════════════════════════╝")
}

// CheckTargets verifies if metrics meet target performance requirements
func (m *PerformanceMetrics) CheckTargets(targets *PerformanceTargets) *TargetCheckResult {
	result := &TargetCheckResult{
		AllPassed: true,
		Checks:    make([]TargetCheck, 0),
	}

	// Check throughput
	if targets.MinThroughput > 0 {
		passed := m.Throughput >= targets.MinThroughput
		result.Checks = append(result.Checks, TargetCheck{
			Name:     "Throughput",
			Target:   fmt.Sprintf(">= %.0f ops/sec", targets.MinThroughput),
			Actual:   fmt.Sprintf("%.2f ops/sec", m.Throughput),
			Passed:   passed,
		})
		if !passed {
			result.AllPassed = false
		}
	}

	// Check success rate
	if targets.MinSuccessRate > 0 {
		passed := m.SuccessRate >= targets.MinSuccessRate
		result.Checks = append(result.Checks, TargetCheck{
			Name:     "Success Rate",
			Target:   fmt.Sprintf(">= %.1f%%", targets.MinSuccessRate),
			Actual:   fmt.Sprintf("%.2f%%", m.SuccessRate),
			Passed:   passed,
		})
		if !passed {
			result.AllPassed = false
		}
	}

	// Check P99 latency
	if targets.MaxP99Latency > 0 && m.Latencies != nil {
		passed := m.Latencies.P99 <= targets.MaxP99Latency
		result.Checks = append(result.Checks, TargetCheck{
			Name:     "P99 Latency",
			Target:   fmt.Sprintf("<= %v", targets.MaxP99Latency),
			Actual:   fmt.Sprintf("%v", m.Latencies.P99),
			Passed:   passed,
		})
		if !passed {
			result.AllPassed = false
		}
	}

	// Check avg latency
	if targets.MaxAvgLatency > 0 && m.Latencies != nil {
		passed := m.Latencies.Avg <= targets.MaxAvgLatency
		result.Checks = append(result.Checks, TargetCheck{
			Name:     "Avg Latency",
			Target:   fmt.Sprintf("<= %v", targets.MaxAvgLatency),
			Actual:   fmt.Sprintf("%v", m.Latencies.Avg),
			Passed:   passed,
		})
		if !passed {
			result.AllPassed = false
		}
	}

	// Check block time
	if targets.MaxBlockTime > 0 && m.AvgBlockTime > 0 {
		passed := m.AvgBlockTime <= targets.MaxBlockTime
		result.Checks = append(result.Checks, TargetCheck{
			Name:     "Block Time",
			Target:   fmt.Sprintf("<= %v", targets.MaxBlockTime),
			Actual:   fmt.Sprintf("%v", m.AvgBlockTime),
			Passed:   passed,
		})
		if !passed {
			result.AllPassed = false
		}
	}

	// Check orders per block
	if targets.MinOrdersPerBlock > 0 && m.OrdersPerBlock > 0 {
		passed := m.OrdersPerBlock >= targets.MinOrdersPerBlock
		result.Checks = append(result.Checks, TargetCheck{
			Name:     "Orders/Block",
			Target:   fmt.Sprintf(">= %.0f", targets.MinOrdersPerBlock),
			Actual:   fmt.Sprintf("%.2f", m.OrdersPerBlock),
			Passed:   passed,
		})
		if !passed {
			result.AllPassed = false
		}
	}

	return result
}

// PerformanceTargets defines target performance metrics
type PerformanceTargets struct {
	MinThroughput     float64       `json:"min_throughput_ops_per_sec"`
	MinSuccessRate    float64       `json:"min_success_rate_percent"`
	MaxP99Latency     time.Duration `json:"max_p99_latency"`
	MaxAvgLatency     time.Duration `json:"max_avg_latency"`
	MaxBlockTime      time.Duration `json:"max_block_time"`
	MinOrdersPerBlock float64       `json:"min_orders_per_block"`
}

// DefaultTargets returns the default performance targets for PerpDEX
func DefaultTargets() *PerformanceTargets {
	return &PerformanceTargets{
		MinThroughput:     500,                    // 500+ orders/sec
		MinSuccessRate:    99.9,                   // 99.9% success rate
		MaxP99Latency:     100 * time.Millisecond, // P99 < 100ms
		MaxAvgLatency:     50 * time.Millisecond,  // Avg < 50ms
		MaxBlockTime:      500 * time.Millisecond, // ~500ms block time
		MinOrdersPerBlock: 1000,                   // 1000+ orders/block
	}
}

// TargetCheck represents a single target check result
type TargetCheck struct {
	Name   string `json:"name"`
	Target string `json:"target"`
	Actual string `json:"actual"`
	Passed bool   `json:"passed"`
}

// TargetCheckResult holds all target check results
type TargetCheckResult struct {
	AllPassed bool          `json:"all_passed"`
	Checks    []TargetCheck `json:"checks"`
}

// PrintResults prints formatted target check results
func (r *TargetCheckResult) PrintResults() {
	fmt.Println("\n╔══════════════════════════════════════════════════════════════╗")
	fmt.Println("║  Performance Target Verification                             ║")
	fmt.Println("╠══════════════════════════════════════════════════════════════╣")

	for _, check := range r.Checks {
		status := "✅ PASS"
		if !check.Passed {
			status = "❌ FAIL"
		}
		fmt.Printf("║  %-14s %s                                      ║\n", check.Name+":", status)
		fmt.Printf("║    Target: %-50s ║\n", check.Target)
		fmt.Printf("║    Actual: %-50s ║\n", check.Actual)
	}

	fmt.Println("╠══════════════════════════════════════════════════════════════╣")
	if r.AllPassed {
		fmt.Println("║  Overall Result: ✅ ALL TARGETS MET                          ║")
	} else {
		fmt.Println("║  Overall Result: ❌ SOME TARGETS FAILED                       ║")
	}
	fmt.Println("╚══════════════════════════════════════════════════════════════╝")
}

// SaveToFile saves metrics to a JSON file
func (m *PerformanceMetrics) SaveToFile(filepath string) error {
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal metrics: %w", err)
	}

	return os.WriteFile(filepath, data, 0644)
}

// LoadFromFile loads metrics from a JSON file
func LoadFromFile(filepath string) (*PerformanceMetrics, error) {
	data, err := os.ReadFile(filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	var metrics PerformanceMetrics
	if err := json.Unmarshal(data, &metrics); err != nil {
		return nil, fmt.Errorf("failed to unmarshal metrics: %w", err)
	}

	return &metrics, nil
}

// CompareWithBaseline compares current metrics with baseline
func (m *PerformanceMetrics) CompareWithBaseline(baseline *PerformanceMetrics) *RegressionReport {
	report := &RegressionReport{
		Current:  m,
		Baseline: baseline,
		Changes:  make([]MetricChange, 0),
	}

	// Compare throughput
	if baseline.Throughput > 0 {
		change := (m.Throughput - baseline.Throughput) / baseline.Throughput * 100
		report.Changes = append(report.Changes, MetricChange{
			Name:           "Throughput",
			BaselineValue:  fmt.Sprintf("%.2f ops/sec", baseline.Throughput),
			CurrentValue:   fmt.Sprintf("%.2f ops/sec", m.Throughput),
			PercentChange:  change,
			IsRegression:   change < -5, // 5% degradation threshold
		})
	}

	// Compare P99 latency
	if baseline.Latencies != nil && m.Latencies != nil && baseline.Latencies.P99 > 0 {
		change := float64(m.Latencies.P99-baseline.Latencies.P99) / float64(baseline.Latencies.P99) * 100
		report.Changes = append(report.Changes, MetricChange{
			Name:           "P99 Latency",
			BaselineValue:  baseline.Latencies.P99.String(),
			CurrentValue:   m.Latencies.P99.String(),
			PercentChange:  change,
			IsRegression:   change > 10, // 10% increase threshold
		})
	}

	// Compare success rate
	if baseline.SuccessRate > 0 {
		change := m.SuccessRate - baseline.SuccessRate
		report.Changes = append(report.Changes, MetricChange{
			Name:           "Success Rate",
			BaselineValue:  fmt.Sprintf("%.2f%%", baseline.SuccessRate),
			CurrentValue:   fmt.Sprintf("%.2f%%", m.SuccessRate),
			PercentChange:  change,
			IsRegression:   change < -0.1, // 0.1% drop threshold
		})
	}

	// Check for any regressions
	report.HasRegression = false
	for _, c := range report.Changes {
		if c.IsRegression {
			report.HasRegression = true
			break
		}
	}

	return report
}

// MetricChange represents a change in a metric
type MetricChange struct {
	Name          string  `json:"name"`
	BaselineValue string  `json:"baseline_value"`
	CurrentValue  string  `json:"current_value"`
	PercentChange float64 `json:"percent_change"`
	IsRegression  bool    `json:"is_regression"`
}

// RegressionReport compares current metrics with baseline
type RegressionReport struct {
	Current       *PerformanceMetrics `json:"current"`
	Baseline      *PerformanceMetrics `json:"baseline"`
	Changes       []MetricChange      `json:"changes"`
	HasRegression bool                `json:"has_regression"`
}

// PrintReport prints the regression comparison report
func (r *RegressionReport) PrintReport() {
	fmt.Println("\n╔══════════════════════════════════════════════════════════════╗")
	fmt.Println("║  Performance Regression Analysis                             ║")
	fmt.Println("╠══════════════════════════════════════════════════════════════╣")

	for _, change := range r.Changes {
		status := "→"
		if change.IsRegression {
			status = "⚠️"
		} else if change.PercentChange > 5 {
			status = "✨"
		}

		fmt.Printf("║  %s %-12s                                            ║\n", status, change.Name)
		fmt.Printf("║    Baseline: %-48s ║\n", change.BaselineValue)
		fmt.Printf("║    Current:  %-48s ║\n", change.CurrentValue)
		fmt.Printf("║    Change:   %+.2f%%                                          ║\n", change.PercentChange)
	}

	fmt.Println("╠══════════════════════════════════════════════════════════════╣")
	if r.HasRegression {
		fmt.Println("║  ⚠️  WARNING: Performance regression detected!               ║")
	} else {
		fmt.Println("║  ✅ No performance regression detected                        ║")
	}
	fmt.Println("╚══════════════════════════════════════════════════════════════╝")
}
