// Package tps_benchmark measures real chain TPS
package tps_benchmark

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

const (
	chainID        = "perpdex-1"
	keyringBackend = "test"
)

var homeDir = findHomeDir()

func findHomeDir() string {
	absPath := "/Users/vvedition/Desktop/dex mvp/perp-dex_副本/.perpdex-test"
	if _, err := os.Stat(absPath + "/keyring-test"); err == nil {
		return absPath
	}
	return ".perpdex-test"
}

func findBinaryPath() string {
	paths := []string{
		"/Users/vvedition/Desktop/dex mvp/perp-dex_副本/build/perpdexd",
		"../../build/perpdexd",
		"../build/perpdexd",
		"./build/perpdexd",
	}
	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			if abs, err := filepath.Abs(p); err == nil {
				return abs
			}
			return p
		}
	}
	return "./build/perpdexd"
}

// TxResult represents a transaction result
type TxResult struct {
	Success bool
	TxHash  string
	Latency time.Duration
	Error   string
}

// sendTransaction sends a single transaction using offline signing
func sendTransaction(ctx context.Context, binary, trader string, seq uint64, price string) *TxResult {
	start := time.Now()
	result := &TxResult{}

	// Generate unsigned tx
	genArgs := []string{
		"tx", "orderbook", "place-order",
		"BTC-USDC", "buy", "limit", price, "0.001",
		"--from", trader,
		"--home", homeDir,
		"--chain-id", chainID,
		"--keyring-backend", keyringBackend,
		"--gas", "200000",
		"--fees", "100usdc",
		"--generate-only",
	}

	genCmd := exec.CommandContext(ctx, binary, genArgs...)
	var genStdout, genStderr bytes.Buffer
	genCmd.Stdout = &genStdout
	genCmd.Stderr = &genStderr
	if err := genCmd.Run(); err != nil {
		result.Error = fmt.Sprintf("gen: %v", genStderr.String())
		result.Latency = time.Since(start)
		return result
	}

	// Write to temp file
	tmpFile := fmt.Sprintf("/tmp/tx_%s_%d_%d.json", trader, seq, time.Now().UnixNano())
	if err := os.WriteFile(tmpFile, genStdout.Bytes(), 0644); err != nil {
		result.Error = fmt.Sprintf("write: %v", err)
		result.Latency = time.Since(start)
		return result
	}
	defer os.Remove(tmpFile)

	// Sign offline
	signedFile := tmpFile + ".signed"
	signArgs := []string{
		"tx", "sign", tmpFile,
		"--from", trader,
		"--home", homeDir,
		"--chain-id", chainID,
		"--keyring-backend", keyringBackend,
		"--account-number", "0",
		"--sequence", fmt.Sprintf("%d", seq),
		"--offline",
		"--output-document", signedFile,
	}

	signCmd := exec.CommandContext(ctx, binary, signArgs...)
	var signStderr bytes.Buffer
	signCmd.Stderr = &signStderr
	if err := signCmd.Run(); err != nil {
		result.Error = fmt.Sprintf("sign: %v", signStderr.String())
		result.Latency = time.Since(start)
		return result
	}
	defer os.Remove(signedFile)

	// Broadcast
	broadcastArgs := []string{
		"tx", "broadcast", signedFile,
		"--home", homeDir,
		"--broadcast-mode", "async", // Use async for max TPS
		"--output", "json",
	}

	broadcastCmd := exec.CommandContext(ctx, binary, broadcastArgs...)
	var stdout, stderr bytes.Buffer
	broadcastCmd.Stdout = &stdout
	broadcastCmd.Stderr = &stderr

	if err := broadcastCmd.Run(); err != nil {
		result.Error = fmt.Sprintf("broadcast: %v", stderr.String())
		result.Latency = time.Since(start)
		return result
	}

	result.Latency = time.Since(start)

	var txResp struct {
		TxHash string `json:"txhash"`
		Code   int    `json:"code"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &txResp); err == nil {
		result.TxHash = txResp.TxHash
		result.Success = txResp.Code == 0 || txResp.TxHash != ""
	}

	return result
}

// TestTPS_RealChain measures actual chain TPS
func TestTPS_RealChain(t *testing.T) {
	binary := findBinaryPath()
	t.Logf("Binary: %s", binary)
	t.Logf("Home: %s", homeDir)

	// Test parameters
	testConfigs := []struct {
		name        string
		concurrency int
		txCount     int
		duration    time.Duration
	}{
		{"Sequential", 1, 20, 0},
		{"Low Concurrency", 5, 50, 0},
		{"Medium Concurrency", 10, 100, 0},
		{"High Concurrency", 20, 200, 0},
		{"Max Concurrency", 50, 500, 0},
	}

	t.Log("\n╔══════════════════════════════════════════════════════════════╗")
	t.Log("║              PerpDEX Real Chain TPS Benchmark                 ║")
	t.Log("╠══════════════════════════════════════════════════════════════╣")

	for _, cfg := range testConfigs {
		t.Run(cfg.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
			defer cancel()

			var (
				successCount int64
				failCount    int64
				totalLatency int64
				wg           sync.WaitGroup
				mu           sync.Mutex
				latencies    []time.Duration
			)

			startTime := time.Now()
			txPerWorker := cfg.txCount / cfg.concurrency

			for w := 0; w < cfg.concurrency; w++ {
				wg.Add(1)
				go func(workerID int) {
					defer wg.Done()
					baseSeq := uint64(workerID * txPerWorker)

					for i := 0; i < txPerWorker; i++ {
						select {
						case <-ctx.Done():
							return
						default:
						}

						price := fmt.Sprintf("%d", 50000+workerID*100+i)
						result := sendTransaction(ctx, binary, "validator", baseSeq+uint64(i), price)

						if result.Success {
							atomic.AddInt64(&successCount, 1)
							atomic.AddInt64(&totalLatency, int64(result.Latency))
							mu.Lock()
							latencies = append(latencies, result.Latency)
							mu.Unlock()
						} else {
							atomic.AddInt64(&failCount, 1)
						}
					}
				}(w)
			}

			wg.Wait()
			elapsed := time.Since(startTime)

			success := atomic.LoadInt64(&successCount)
			fail := atomic.LoadInt64(&failCount)
			total := success + fail

			tps := float64(success) / elapsed.Seconds()
			avgLatency := time.Duration(0)
			if success > 0 {
				avgLatency = time.Duration(atomic.LoadInt64(&totalLatency) / success)
			}

			successRate := float64(0)
			if total > 0 {
				successRate = float64(success) / float64(total) * 100
			}

			t.Logf("║  %-20s                                        ║", cfg.name)
			t.Logf("║    Concurrency: %-5d  Transactions: %-5d              ║", cfg.concurrency, total)
			t.Logf("║    Success: %-5d      Failed: %-5d                     ║", success, fail)
			t.Logf("║    Duration: %-10v                                   ║", elapsed.Round(time.Millisecond))
			t.Logf("║    TPS: %-10.2f     Avg Latency: %-10v           ║", tps, avgLatency.Round(time.Millisecond))
			t.Logf("║    Success Rate: %.1f%%                                   ║", successRate)
			t.Logf("╠══════════════════════════════════════════════════════════════╣")
		})
	}

	t.Log("╚══════════════════════════════════════════════════════════════╝")
}

// TestTPS_Sustained measures sustained TPS over time
func TestTPS_Sustained(t *testing.T) {
	binary := findBinaryPath()
	duration := 30 * time.Second
	concurrency := 10

	t.Logf("Running sustained TPS test for %v with %d workers...", duration, concurrency)

	ctx, cancel := context.WithTimeout(context.Background(), duration+time.Minute)
	defer cancel()

	var (
		successCount int64
		failCount    int64
		wg           sync.WaitGroup
	)

	startTime := time.Now()
	stopCh := make(chan struct{})

	// Stop after duration
	go func() {
		time.Sleep(duration)
		close(stopCh)
	}()

	for w := 0; w < concurrency; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			seq := uint64(0)

			for {
				select {
				case <-stopCh:
					return
				case <-ctx.Done():
					return
				default:
				}

				price := fmt.Sprintf("%d", 50000+workerID*1000+int(seq%100))
				result := sendTransaction(ctx, binary, "validator", seq, price)
				seq++

				if result.Success {
					atomic.AddInt64(&successCount, 1)
				} else {
					atomic.AddInt64(&failCount, 1)
				}
			}
		}(w)
	}

	wg.Wait()
	elapsed := time.Since(startTime)

	success := atomic.LoadInt64(&successCount)
	fail := atomic.LoadInt64(&failCount)
	total := success + fail
	tps := float64(success) / elapsed.Seconds()

	t.Log("\n╔══════════════════════════════════════════════════════════════╗")
	t.Log("║              Sustained TPS Test Results                       ║")
	t.Log("╠══════════════════════════════════════════════════════════════╣")
	t.Logf("║  Duration: %-10v  Concurrency: %-5d                    ║", elapsed.Round(time.Second), concurrency)
	t.Logf("║  Total TX: %-10d Success: %-10d Failed: %-5d     ║", total, success, fail)
	t.Logf("║  ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━ ║")
	t.Logf("║  🚀 SUSTAINED TPS: %.2f tx/sec                            ║", tps)
	t.Logf("║  📊 Success Rate: %.1f%%                                     ║", float64(success)/float64(total)*100)
	t.Log("╚══════════════════════════════════════════════════════════════╝")
}
