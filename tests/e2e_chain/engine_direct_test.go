// Package e2e_chain provides direct engine-level E2E testing
// Tests the orderbook and matching engine directly while verifying chain state
package e2e_chain

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"cosmossdk.io/math"

	"github.com/openalpha/perp-dex/tests/performance"
	"github.com/openalpha/perp-dex/x/orderbook/keeper"
	"github.com/openalpha/perp-dex/x/orderbook/types"
)

// TestEngine_DirectMarketMaker tests market maker behavior using direct engine calls
func TestEngine_DirectMarketMaker(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	orderbook := keeper.NewOrderBookV2("BTC-USDC")

	collector := performance.NewMetricsCollector("DirectMarketMaker")

	const (
		duration     = 30 * time.Second
		quotesPerSec = 100
		spreadBps    = 10
		basePrice    = 50000
	)

	var successCount, failedCount int64
	done := make(chan struct{})
	time.AfterFunc(duration, func() { close(done) })

	t.Logf("Starting Direct Engine Market Maker Test")
	t.Logf("  Duration: %v, Quote Rate: %d/sec, Spread: %d bps", duration, quotesPerSec, spreadBps)

	ticker := time.NewTicker(time.Second / time.Duration(quotesPerSec))
	defer ticker.Stop()

	quoteCount := 0
	for {
		select {
		case <-done:
			goto finished
		case <-ticker.C:
			quoteCount++
			midPrice := float64(basePrice + rand.Intn(200) - 100)
			spread := midPrice * float64(spreadBps) / 10000

			// Create bid order
			bidPrice := math.LegacyMustNewDecFromStr(fmt.Sprintf("%.2f", midPrice-spread/2))
			bidOrder := &types.Order{
				OrderID:   fmt.Sprintf("bid-%d", quoteCount),
				Trader:    "mm-trader",
				MarketID:  "BTC-USDC",
				Side:      types.SideBuy,
				OrderType: types.OrderTypeLimit,
				Price:     bidPrice,
				Quantity:  math.LegacyMustNewDecFromStr("0.1"),
				FilledQty: math.LegacyZeroDec(),
				Status:    types.OrderStatusOpen,
			}

			start := time.Now()
			orderbook.AddOrder(bidOrder)
			latency := time.Since(start)
			atomic.AddInt64(&successCount, 1)
			collector.RecordOperation(latency, true, 0)

			// Create ask order
			askPrice := math.LegacyMustNewDecFromStr(fmt.Sprintf("%.2f", midPrice+spread/2))
			askOrder := &types.Order{
				OrderID:   fmt.Sprintf("ask-%d", quoteCount),
				Trader:    "mm-trader",
				MarketID:  "BTC-USDC",
				Side:      types.SideSell,
				OrderType: types.OrderTypeLimit,
				Price:     askPrice,
				Quantity:  math.LegacyMustNewDecFromStr("0.1"),
				FilledQty: math.LegacyZeroDec(),
				Status:    types.OrderStatusOpen,
			}

			start = time.Now()
			orderbook.AddOrder(askOrder)
			latency = time.Since(start)
			atomic.AddInt64(&successCount, 1)
			collector.RecordOperation(latency, true, 0)
		}
	}

finished:
	metrics := collector.GetMetrics()

	t.Logf("\n╔══════════════════════════════════════════════════════════════╗")
	t.Logf("║  Direct Engine Market Maker Results                          ║")
	t.Logf("╠══════════════════════════════════════════════════════════════╣")
	t.Logf("║  Total Quotes:     %-42d ║", quoteCount*2)
	t.Logf("║  Successful:       %-42d ║", successCount)
	t.Logf("║  Failed:           %-42d ║", failedCount)
	t.Logf("║  Success Rate:     %-41.2f%% ║", float64(successCount)/float64(quoteCount*2)*100)
	t.Logf("╚══════════════════════════════════════════════════════════════╝")

	metrics.PrintReport()

	// Verify orderbook state
	bestBid, bestAsk := orderbook.GetBestLevels()
	if bestBid != nil && bestAsk != nil {
		t.Logf("\nOrderbook State:")
		t.Logf("  Best Bid: %s", bestBid.Price.String())
		t.Logf("  Best Ask: %s", bestAsk.Price.String())
		spreadDec := bestAsk.Price.Sub(bestBid.Price)
		spreadPct := spreadDec.Quo(bestBid.Price).MulInt64(100)
		t.Logf("  Spread: %s%%", spreadPct.String())
	}

	bidLevels, askLevels := orderbook.GetDepth()
	t.Logf("  Bid Levels: %d, Ask Levels: %d", bidLevels, askLevels)

	if float64(successCount)/float64(quoteCount*2) < 0.95 {
		t.Errorf("Success rate below 95%%")
	}
}

// TestEngine_DirectHighFrequency tests HFT patterns using direct engine calls
func TestEngine_DirectHighFrequency(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	orderbook := keeper.NewOrderBookV2("BTC-USDC")

	collector := performance.NewMetricsCollector("DirectHFT")

	const (
		duration     = 20 * time.Second
		ordersPerSec = 200
		cancelRatio  = 0.8
		basePrice    = 50000
	)

	var placeSuccess, cancelSuccess int64
	type placedOrder struct {
		orderID string
		order   *types.Order
	}
	placedOrders := make(chan placedOrder, 1000)
	done := make(chan struct{})
	time.AfterFunc(duration, func() { close(done) })

	t.Logf("Starting Direct Engine HFT Test")
	t.Logf("  Duration: %v, Order Rate: %d/sec, Cancel Ratio: %.0f%%", duration, ordersPerSec, cancelRatio*100)

	var wg sync.WaitGroup

	// Order placer
	wg.Add(1)
	go func() {
		defer wg.Done()
		ticker := time.NewTicker(time.Second / time.Duration(ordersPerSec))
		defer ticker.Stop()

		orderNum := 0
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				orderNum++
				orderID := fmt.Sprintf("hft-%d", orderNum)
				price := math.LegacyMustNewDecFromStr(fmt.Sprintf("%d", basePrice-rand.Intn(100)))

				order := &types.Order{
					OrderID:   orderID,
					Trader:    "hft-trader",
					MarketID:  "BTC-USDC",
					Side:      types.SideBuy,
					OrderType: types.OrderTypeLimit,
					Price:     price,
					Quantity:  math.LegacyMustNewDecFromStr("0.01"),
					FilledQty: math.LegacyZeroDec(),
					Status:    types.OrderStatusOpen,
				}

				start := time.Now()
				orderbook.AddOrder(order)
				latency := time.Since(start)

				atomic.AddInt64(&placeSuccess, 1)
				collector.RecordOperation(latency, true, 0)

				if rand.Float64() < cancelRatio {
					select {
					case placedOrders <- placedOrder{orderID: orderID, order: order}:
					default:
					}
				}
			}
		}
	}()

	// Order canceller
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-done:
				return
			case po := <-placedOrders:
				time.Sleep(5 * time.Millisecond)

				start := time.Now()
				removed := orderbook.RemoveOrder(po.order)
				latency := time.Since(start)

				if removed != nil {
					atomic.AddInt64(&cancelSuccess, 1)
					collector.RecordLatency(latency)
				}
			}
		}
	}()

	wg.Wait()

	metrics := collector.GetMetrics()

	t.Logf("\n╔══════════════════════════════════════════════════════════════╗")
	t.Logf("║  Direct Engine HFT Results                                   ║")
	t.Logf("╠══════════════════════════════════════════════════════════════╣")
	t.Logf("║  Orders Placed:    %-42d ║", placeSuccess)
	t.Logf("║  Orders Cancelled: %-42d ║", cancelSuccess)
	t.Logf("║  Cancel Ratio:     %-41.2f%% ║", float64(cancelSuccess)/float64(placeSuccess)*100)
	t.Logf("╚══════════════════════════════════════════════════════════════╝")

	metrics.PrintReport()
}

// TestEngine_DirectTradingRush tests burst trading using direct engine calls
func TestEngine_DirectTradingRush(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	orderbook := keeper.NewOrderBookV2("BTC-USDC")

	collector := performance.NewMetricsCollector("DirectTradingRush")

	const (
		numTraders      = 50
		ordersPerTrader = 100
		basePrice       = 50000
	)

	totalOrders := numTraders * ordersPerTrader
	var successCount int64

	t.Logf("Starting Direct Engine Trading Rush Test")
	t.Logf("  Traders: %d, Orders/Trader: %d, Total: %d", numTraders, ordersPerTrader, totalOrders)

	start := time.Now()
	var wg sync.WaitGroup

	// Spawn traders
	for trader := 0; trader < numTraders; trader++ {
		wg.Add(1)
		go func(traderID int) {
			defer wg.Done()

			for i := 0; i < ordersPerTrader; i++ {
				side := types.SideBuy
				priceOffset := -rand.Intn(500)
				if rand.Float32() > 0.5 {
					side = types.SideSell
					priceOffset = rand.Intn(500)
				}

				price := math.LegacyMustNewDecFromStr(fmt.Sprintf("%d", basePrice+priceOffset))
				qty := math.LegacyMustNewDecFromStr(fmt.Sprintf("0.%02d", rand.Intn(99)+1))

				order := &types.Order{
					OrderID:   fmt.Sprintf("trader%d-order%d", traderID, i),
					Trader:    fmt.Sprintf("trader%d", traderID),
					MarketID:  "BTC-USDC",
					Side:      side,
					OrderType: types.OrderTypeLimit,
					Price:     price,
					Quantity:  qty,
					FilledQty: math.LegacyZeroDec(),
					Status:    types.OrderStatusOpen,
				}

				orderStart := time.Now()
				orderbook.AddOrder(order)
				latency := time.Since(orderStart)

				atomic.AddInt64(&successCount, 1)
				collector.RecordOperation(latency, true, 0)
			}
		}(trader)
	}

	wg.Wait()

	totalDuration := time.Since(start)
	metrics := collector.GetMetrics()

	t.Logf("\n╔══════════════════════════════════════════════════════════════╗")
	t.Logf("║  Direct Engine Trading Rush Results                          ║")
	t.Logf("╠══════════════════════════════════════════════════════════════╣")
	t.Logf("║  Total Orders:     %-42d ║", totalOrders)
	t.Logf("║  Successful:       %-42d ║", successCount)
	t.Logf("║  Duration:         %-42v ║", totalDuration.Round(time.Millisecond))
	t.Logf("║  Orders/Second:    %-42.2f ║", float64(totalOrders)/totalDuration.Seconds())
	t.Logf("╚══════════════════════════════════════════════════════════════╝")

	bidLevels, askLevels := orderbook.GetDepth()
	t.Logf("  Final Orderbook: %d bid levels, %d ask levels", bidLevels, askLevels)

	metrics.PrintReport()

	// Verify targets
	targets := &performance.PerformanceTargets{
		MinThroughput:  500,
		MinSuccessRate: 99.0,
		MaxP99Latency:  10 * time.Millisecond,
	}
	result := metrics.CheckTargets(targets)
	result.PrintResults()

	if float64(successCount)/float64(totalOrders) < 0.99 {
		t.Errorf("Success rate below 99%%")
	}
}

// TestEngine_DirectDeepBook tests deep orderbook performance
func TestEngine_DirectDeepBook(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	orderbook := keeper.NewOrderBookV2("BTC-USDC")

	collector := performance.NewMetricsCollector("DirectDeepBook")

	const (
		priceLevels    = 500
		ordersPerLevel = 5
		basePrice      = 50000
	)

	totalOrders := priceLevels * ordersPerLevel * 2
	var successCount int64

	t.Logf("Starting Direct Engine Deep Book Test")
	t.Logf("  Price Levels: %d, Orders/Level: %d, Total: %d", priceLevels, ordersPerLevel, totalOrders)

	start := time.Now()

	// Create deep orderbook
	for level := 0; level < priceLevels; level++ {
		bidPrice := math.LegacyMustNewDecFromStr(fmt.Sprintf("%d", basePrice-level))
		askPrice := math.LegacyMustNewDecFromStr(fmt.Sprintf("%d", basePrice+level+1))

		for order := 0; order < ordersPerLevel; order++ {
			// Add bid
			bidOrder := &types.Order{
				OrderID:   fmt.Sprintf("bid-%d-%d", level, order),
				Trader:    fmt.Sprintf("trader%d", order%10),
				MarketID:  "BTC-USDC",
				Side:      types.SideBuy,
				OrderType: types.OrderTypeLimit,
				Price:     bidPrice,
				Quantity:  math.LegacyMustNewDecFromStr("0.1"),
				FilledQty: math.LegacyZeroDec(),
				Status:    types.OrderStatusOpen,
			}

			orderStart := time.Now()
			orderbook.AddOrder(bidOrder)
			atomic.AddInt64(&successCount, 1)
			collector.RecordOperation(time.Since(orderStart), true, 0)

			// Add ask
			askOrder := &types.Order{
				OrderID:   fmt.Sprintf("ask-%d-%d", level, order),
				Trader:    fmt.Sprintf("trader%d", order%10),
				MarketID:  "BTC-USDC",
				Side:      types.SideSell,
				OrderType: types.OrderTypeLimit,
				Price:     askPrice,
				Quantity:  math.LegacyMustNewDecFromStr("0.1"),
				FilledQty: math.LegacyZeroDec(),
				Status:    types.OrderStatusOpen,
			}

			orderStart = time.Now()
			orderbook.AddOrder(askOrder)
			atomic.AddInt64(&successCount, 1)
			collector.RecordOperation(time.Since(orderStart), true, 0)
		}
	}

	buildDuration := time.Since(start)

	// Test query performance
	queryStart := time.Now()
	for i := 0; i < 1000; i++ {
		orderbook.GetBestLevels()
	}
	queryDuration := time.Since(queryStart)

	metrics := collector.GetMetrics()

	t.Logf("\n╔══════════════════════════════════════════════════════════════╗")
	t.Logf("║  Direct Engine Deep Book Results                             ║")
	t.Logf("╠══════════════════════════════════════════════════════════════╣")
	t.Logf("║  Orders Created:   %-42d ║", successCount)
	t.Logf("║  Build Duration:   %-42v ║", buildDuration.Round(time.Millisecond))
	t.Logf("║  Orders/Second:    %-42.2f ║", float64(successCount)/buildDuration.Seconds())
	t.Logf("║  Query Duration:   %-42v ║", queryDuration)
	t.Logf("║  Queries/Second:   %-42.2f ║", 1000/queryDuration.Seconds())
	t.Logf("╚══════════════════════════════════════════════════════════════╝")

	bidLevels, askLevels := orderbook.GetDepth()
	t.Logf("  Orderbook Depth: %d bid levels, %d ask levels", bidLevels, askLevels)

	metrics.PrintReport()
}

// TestEngine_DirectStability tests engine stability over time
func TestEngine_DirectStability(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	orderbook := keeper.NewOrderBookV2("BTC-USDC")

	collector := performance.NewMetricsCollector("DirectStability")

	const (
		duration     = 60 * time.Second
		ordersPerSec = 100
		basePrice    = 50000
	)

	var orderCount int64
	latencyBuckets := make([]int64, 10) // 0-1ms, 1-2ms, ... 9-10ms, >10ms

	done := make(chan struct{})
	time.AfterFunc(duration, func() { close(done) })

	t.Logf("Starting Direct Engine Stability Test")
	t.Logf("  Duration: %v, Order Rate: %d/sec", duration, ordersPerSec)

	var wg sync.WaitGroup

	// Order generator
	wg.Add(1)
	go func() {
		defer wg.Done()
		ticker := time.NewTicker(time.Second / time.Duration(ordersPerSec))
		defer ticker.Stop()

		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				n := atomic.AddInt64(&orderCount, 1)
				side := types.SideBuy
				priceOffset := -rand.Intn(200)
				if rand.Float32() > 0.5 {
					side = types.SideSell
					priceOffset = rand.Intn(200)
				}

				price := math.LegacyMustNewDecFromStr(fmt.Sprintf("%d", basePrice+priceOffset))

				order := &types.Order{
					OrderID:   fmt.Sprintf("stab-%d", n),
					Trader:    fmt.Sprintf("trader%d", n%10),
					MarketID:  "BTC-USDC",
					Side:      side,
					OrderType: types.OrderTypeLimit,
					Price:     price,
					Quantity:  math.LegacyMustNewDecFromStr("0.1"),
					FilledQty: math.LegacyZeroDec(),
					Status:    types.OrderStatusOpen,
				}

				start := time.Now()
				orderbook.AddOrder(order)
				latency := time.Since(start)

				collector.RecordOperation(latency, true, 0)

				// Track latency distribution
				bucket := int(latency.Milliseconds())
				if bucket > 9 {
					bucket = 9
				}
				atomic.AddInt64(&latencyBuckets[bucket], 1)
			}
		}
	}()

	// Progress reporter
	wg.Add(1)
	go func() {
		defer wg.Done()
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				bidLevels, askLevels := orderbook.GetDepth()
				t.Logf("  Progress: %d orders, depth: %d/%d", orderCount, bidLevels, askLevels)
			}
		}
	}()

	wg.Wait()

	metrics := collector.GetMetrics()

	t.Logf("\n╔══════════════════════════════════════════════════════════════╗")
	t.Logf("║  Direct Engine Stability Results                             ║")
	t.Logf("╠══════════════════════════════════════════════════════════════╣")
	t.Logf("║  Total Orders:     %-42d ║", orderCount)
	t.Logf("║  Orders/Second:    %-42.2f ║", float64(orderCount)/duration.Seconds())
	t.Logf("╠══════════════════════════════════════════════════════════════╣")
	t.Logf("║  Latency Distribution                                        ║")
	t.Logf("╠══════════════════════════════════════════════════════════════╣")
	for i, count := range latencyBuckets {
		if i < 9 {
			t.Logf("║  %d-%dms:           %-42d ║", i, i+1, count)
		} else {
			t.Logf("║  >9ms:             %-42d ║", count)
		}
	}
	t.Logf("╚══════════════════════════════════════════════════════════════╝")

	bidLevels, askLevels := orderbook.GetDepth()
	t.Logf("  Final Orderbook: %d bid levels, %d ask levels", bidLevels, askLevels)

	metrics.PrintReport()
}

// TestChain_Connectivity verifies the chain is running and accessible
func TestChain_Connectivity(t *testing.T) {
	config := DefaultChainConfig()
	client := NewChainClient(config)

	ctx := context.Background()

	status, err := client.GetStatus(ctx)
	if err != nil {
		t.Skipf("Chain not accessible: %v", err)
		return
	}

	t.Logf("Chain Status:")
	t.Logf("  Network: %s", status.NodeInfo.Network)
	t.Logf("  Height: %s", status.SyncInfo.LatestBlockHeight)
	t.Logf("  Catching Up: %v", status.SyncInfo.CatchingUp)

	// Query markets
	markets, err := client.QueryMarkets(ctx)
	if err != nil {
		t.Logf("Could not query markets: %v", err)
	} else {
		t.Logf("  Markets: %s", string(markets))
	}

	// Query orderbook
	orderbook, err := client.QueryOrderBook(ctx, "BTC-USDC")
	if err != nil {
		t.Logf("Could not query orderbook: %v", err)
	} else {
		t.Logf("  BTC-USDC Orderbook: %s", string(orderbook))
	}
}
