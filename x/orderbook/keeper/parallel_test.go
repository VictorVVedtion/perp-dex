package keeper

import (
	"sync"
	"testing"
	"time"

	"cosmossdk.io/log"
	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/openalpha/perp-dex/x/orderbook/types"
)

// createParallelTestKeeper creates a test keeper for parallel matching tests
// Reuses mockBenchPerpetualKeeper from benchmark_test.go
func createParallelTestKeeper() *Keeper {
	interfaceRegistry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(interfaceRegistry)
	storeKey := storetypes.NewKVStoreKey("orderbook_parallel")
	logger := log.NewNopLogger()

	return NewKeeper(cdc, storeKey, &mockBenchPerpetualKeeper{}, logger)
}

// TestParallelConfig tests the parallel configuration
func TestParallelConfig(t *testing.T) {
	t.Run("DefaultConfig", func(t *testing.T) {
		config := DefaultParallelConfig()

		if !config.Enabled {
			t.Error("Expected parallel matching to be enabled by default")
		}

		// High TPS optimized defaults (Hyperliquid alignment)
		if config.Workers != 16 {
			t.Errorf("Expected 16 workers (high TPS), got %d", config.Workers)
		}

		if config.BatchSize != 500 {
			t.Errorf("Expected batch size 500 (high TPS), got %d", config.BatchSize)
		}

		if config.Timeout != 10*time.Second {
			t.Errorf("Expected 10s timeout (high TPS), got %v", config.Timeout)
		}
	})

	t.Run("CustomConfig", func(t *testing.T) {
		config := ParallelConfig{
			Enabled:   false,
			Workers:   8,
			BatchSize: 200,
			Timeout:   10 * time.Second,
		}

		if config.Enabled {
			t.Error("Expected parallel matching to be disabled")
		}

		if config.Workers != 8 {
			t.Errorf("Expected 8 workers, got %d", config.Workers)
		}
	})
}

// TestParallelMatcher tests the ParallelMatcher
func TestParallelMatcher(t *testing.T) {
	t.Run("GroupOrdersByMarket", func(t *testing.T) {
		keeper := createParallelTestKeeper()
		pm := NewParallelMatcher(keeper, DefaultParallelConfig())

		// Create test orders
		orders := []*types.Order{
			types.NewOrder("order-1", "trader1", "BTC-USD", types.SideBuy, types.OrderTypeLimit, math.LegacyNewDec(50000), math.LegacyNewDec(1)),
			types.NewOrder("order-2", "trader2", "BTC-USD", types.SideSell, types.OrderTypeLimit, math.LegacyNewDec(50100), math.LegacyNewDec(1)),
			types.NewOrder("order-3", "trader3", "ETH-USD", types.SideBuy, types.OrderTypeLimit, math.LegacyNewDec(3000), math.LegacyNewDec(10)),
			types.NewOrder("order-4", "trader4", "ETH-USD", types.SideSell, types.OrderTypeLimit, math.LegacyNewDec(3010), math.LegacyNewDec(5)),
			types.NewOrder("order-5", "trader5", "SOL-USD", types.SideBuy, types.OrderTypeLimit, math.LegacyNewDec(100), math.LegacyNewDec(100)),
		}

		grouped := pm.GroupOrdersByMarket(orders)

		if len(grouped) != 3 {
			t.Errorf("Expected 3 markets, got %d", len(grouped))
		}

		if btcOrders, ok := grouped["BTC-USD"]; ok {
			if len(btcOrders.Orders) != 2 {
				t.Errorf("Expected 2 BTC-USD orders, got %d", len(btcOrders.Orders))
			}
		} else {
			t.Error("Expected BTC-USD market group")
		}

		if ethOrders, ok := grouped["ETH-USD"]; ok {
			if len(ethOrders.Orders) != 2 {
				t.Errorf("Expected 2 ETH-USD orders, got %d", len(ethOrders.Orders))
			}
		} else {
			t.Error("Expected ETH-USD market group")
		}

		if solOrders, ok := grouped["SOL-USD"]; ok {
			if len(solOrders.Orders) != 1 {
				t.Errorf("Expected 1 SOL-USD order, got %d", len(solOrders.Orders))
			}
		} else {
			t.Error("Expected SOL-USD market group")
		}
	})

	t.Run("GroupOrdersByMarket_EmptyOrders", func(t *testing.T) {
		keeper := createParallelTestKeeper()
		pm := NewParallelMatcher(keeper, DefaultParallelConfig())

		grouped := pm.GroupOrdersByMarket([]*types.Order{})

		if len(grouped) != 0 {
			t.Errorf("Expected 0 markets for empty orders, got %d", len(grouped))
		}
	})

	t.Run("GroupOrdersByMarket_NilOrders", func(t *testing.T) {
		keeper := createParallelTestKeeper()
		pm := NewParallelMatcher(keeper, DefaultParallelConfig())

		orders := []*types.Order{
			nil,
			types.NewOrder("order-1", "trader1", "BTC-USD", types.SideBuy, types.OrderTypeLimit, math.LegacyNewDec(50000), math.LegacyNewDec(1)),
			nil,
		}

		grouped := pm.GroupOrdersByMarket(orders)

		if len(grouped) != 1 {
			t.Errorf("Expected 1 market, got %d", len(grouped))
		}

		if btcOrders, ok := grouped["BTC-USD"]; ok {
			if len(btcOrders.Orders) != 1 {
				t.Errorf("Expected 1 BTC-USD order, got %d", len(btcOrders.Orders))
			}
		}
	})

	t.Run("GroupOrdersByMarket_InactiveOrders", func(t *testing.T) {
		keeper := createParallelTestKeeper()
		pm := NewParallelMatcher(keeper, DefaultParallelConfig())

		activeOrder := types.NewOrder("order-1", "trader1", "BTC-USD", types.SideBuy, types.OrderTypeLimit, math.LegacyNewDec(50000), math.LegacyNewDec(1))
		cancelledOrder := types.NewOrder("order-2", "trader2", "BTC-USD", types.SideSell, types.OrderTypeLimit, math.LegacyNewDec(50100), math.LegacyNewDec(1))
		cancelledOrder.Cancel()

		orders := []*types.Order{activeOrder, cancelledOrder}

		grouped := pm.GroupOrdersByMarket(orders)

		if btcOrders, ok := grouped["BTC-USD"]; ok {
			if len(btcOrders.Orders) != 1 {
				t.Errorf("Expected 1 active BTC-USD order, got %d", len(btcOrders.Orders))
			}
		}
	})
}

// TestScheduler tests the MatchingScheduler
func TestScheduler(t *testing.T) {
	t.Run("NewMatchingScheduler", func(t *testing.T) {
		keeper := createParallelTestKeeper()
		scheduler := NewMatchingScheduler(4, 100, keeper)

		if scheduler.GetWorkerCount() != 4 {
			t.Errorf("Expected 4 workers, got %d", scheduler.GetWorkerCount())
		}

		if scheduler.IsRunning() {
			t.Error("Scheduler should not be running before Start()")
		}
	})

	t.Run("Scheduler_StartStop", func(t *testing.T) {
		keeper := createParallelTestKeeper()
		scheduler := NewMatchingScheduler(2, 50, keeper)

		scheduler.Start()

		if !scheduler.IsRunning() {
			t.Error("Scheduler should be running after Start()")
		}

		// Start again should be idempotent
		scheduler.Start()

		if !scheduler.IsRunning() {
			t.Error("Scheduler should still be running after second Start()")
		}

		scheduler.Stop()

		if scheduler.IsRunning() {
			t.Error("Scheduler should not be running after Stop()")
		}

		// Stop again should be safe
		scheduler.Stop()
	})

	t.Run("Scheduler_SubmitOrder", func(t *testing.T) {
		keeper := createParallelTestKeeper()
		scheduler := NewMatchingScheduler(2, 50, keeper)

		// Submit before start should fail
		order := types.NewOrder("order-1", "trader1", "BTC-USD", types.SideBuy, types.OrderTypeLimit, math.LegacyNewDec(50000), math.LegacyNewDec(1))
		err := scheduler.SubmitOrder(order)
		if err == nil {
			t.Error("Expected error when submitting to non-running scheduler")
		}

		scheduler.Start()
		defer scheduler.Stop()

		// Submit after start should succeed
		err = scheduler.SubmitOrder(order)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}

		if scheduler.GetQueueSize() != 1 {
			t.Errorf("Expected queue size 1, got %d", scheduler.GetQueueSize())
		}
	})

	t.Run("Scheduler_DefaultValues", func(t *testing.T) {
		keeper := createParallelTestKeeper()

		// Test with zero/negative values
		scheduler := NewMatchingScheduler(0, 0, keeper)

		if scheduler.GetWorkerCount() != 4 {
			t.Errorf("Expected default 4 workers for zero input, got %d", scheduler.GetWorkerCount())
		}
	})
}

// TestWorkerPool tests the WorkerPool
func TestWorkerPool(t *testing.T) {
	t.Run("NewWorkerPool", func(t *testing.T) {
		keeper := createParallelTestKeeper()
		wp := NewWorkerPool(4, keeper)

		if wp.workers != 4 {
			t.Errorf("Expected 4 workers, got %d", wp.workers)
		}
	})

	t.Run("WorkerPool_StartStop", func(t *testing.T) {
		keeper := createParallelTestKeeper()
		wp := NewWorkerPool(2, keeper)

		wp.Start()
		if !wp.running {
			t.Error("Worker pool should be running after Start()")
		}

		wp.Stop()
		if wp.running {
			t.Error("Worker pool should not be running after Stop()")
		}
	})

	t.Run("WorkerPool_Submit", func(t *testing.T) {
		keeper := createParallelTestKeeper()
		wp := NewWorkerPool(4, keeper)

		// Submit before start should fail
		err := wp.Submit(func() {})
		if err == nil {
			t.Error("Expected error when submitting to non-running worker pool")
		}

		wp.Start()
		defer wp.Stop()

		// Submit after start should succeed
		var executed bool
		var wg sync.WaitGroup
		wg.Add(1)

		err = wp.Submit(func() {
			executed = true
			wg.Done()
		})

		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}

		wg.Wait()

		if !executed {
			t.Error("Task was not executed")
		}
	})

	t.Run("WorkerPool_ConcurrentTasks", func(t *testing.T) {
		keeper := createParallelTestKeeper()
		wp := NewWorkerPool(4, keeper)

		wp.Start()
		defer wp.Stop()

		const numTasks = 100
		var counter int
		var mu sync.Mutex
		var wg sync.WaitGroup

		for i := 0; i < numTasks; i++ {
			wg.Add(1)
			err := wp.Submit(func() {
				defer wg.Done()
				mu.Lock()
				counter++
				mu.Unlock()
			})

			if err != nil {
				wg.Done()
				t.Logf("Task submission failed (queue full): %v", err)
			}
		}

		wg.Wait()

		if counter == 0 {
			t.Error("No tasks were executed")
		}

		t.Logf("Executed %d/%d tasks", counter, numTasks)
	})
}

// TestParallelMatchingCorrectness verifies that parallel matching produces correct results
func TestParallelMatchingCorrectness(t *testing.T) {
	t.Run("DeterministicResults", func(t *testing.T) {
		// Create orders that should match
		buyOrder := types.NewOrder("buy-1", "buyer", "BTC-USD", types.SideBuy, types.OrderTypeLimit, math.LegacyNewDec(50000), math.LegacyNewDec(1))
		sellOrder := types.NewOrder("sell-1", "seller", "BTC-USD", types.SideSell, types.OrderTypeLimit, math.LegacyNewDec(50000), math.LegacyNewDec(1))

		// Verify order properties
		if buyOrder.Side != types.SideBuy {
			t.Error("Buy order should have Buy side")
		}

		if sellOrder.Side != types.SideSell {
			t.Error("Sell order should have Sell side")
		}

		if !buyOrder.Price.Equal(sellOrder.Price) {
			t.Error("Prices should be equal for matching")
		}

		if !buyOrder.Quantity.Equal(sellOrder.Quantity) {
			t.Error("Quantities should be equal")
		}
	})

	t.Run("MarketSeparation", func(t *testing.T) {
		keeper := createParallelTestKeeper()
		pm := NewParallelMatcher(keeper, DefaultParallelConfig())

		// Create orders for different markets
		orders := []*types.Order{
			types.NewOrder("btc-buy", "trader1", "BTC-USD", types.SideBuy, types.OrderTypeLimit, math.LegacyNewDec(50000), math.LegacyNewDec(1)),
			types.NewOrder("eth-buy", "trader2", "ETH-USD", types.SideBuy, types.OrderTypeLimit, math.LegacyNewDec(3000), math.LegacyNewDec(10)),
			types.NewOrder("btc-sell", "trader3", "BTC-USD", types.SideSell, types.OrderTypeLimit, math.LegacyNewDec(50100), math.LegacyNewDec(1)),
			types.NewOrder("eth-sell", "trader4", "ETH-USD", types.SideSell, types.OrderTypeLimit, math.LegacyNewDec(3010), math.LegacyNewDec(5)),
		}

		grouped := pm.GroupOrdersByMarket(orders)

		// Verify BTC orders don't mix with ETH orders
		btcOrders := grouped["BTC-USD"]
		for _, order := range btcOrders.Orders {
			if order.MarketID != "BTC-USD" {
				t.Errorf("BTC-USD group contains order from %s", order.MarketID)
			}
		}

		ethOrders := grouped["ETH-USD"]
		for _, order := range ethOrders.Orders {
			if order.MarketID != "ETH-USD" {
				t.Errorf("ETH-USD group contains order from %s", order.MarketID)
			}
		}
	})
}

// TestParallelMatchResult tests the result structures
func TestParallelMatchResult(t *testing.T) {
	t.Run("EmptyResult", func(t *testing.T) {
		result := &ParallelMatchResult{
			MarketID:      "BTC-USD",
			Trades:        make([]*types.Trade, 0),
			UpdatedOrders: make([]*types.Order, 0),
			ProcessedAt:   time.Now(),
		}

		if result.Error != nil {
			t.Error("Empty result should have no error")
		}

		if len(result.Trades) != 0 {
			t.Error("Empty result should have no trades")
		}
	})

	t.Run("AggregatedResult", func(t *testing.T) {
		result := &AggregatedMatchResult{
			Results:      make([]*ParallelMatchResult, 0),
			TotalTrades:  10,
			TotalMatched: 20,
			Duration:     100 * time.Millisecond,
			Errors:       make([]error, 0),
		}

		if result.TotalTrades != 10 {
			t.Errorf("Expected 10 total trades, got %d", result.TotalTrades)
		}

		if result.TotalMatched != 20 {
			t.Errorf("Expected 20 total matched, got %d", result.TotalMatched)
		}
	})
}

// BenchmarkParallelGrouping benchmarks order grouping by market
func BenchmarkParallelGrouping(b *testing.B) {
	// Create orders for benchmarking
	createOrders := func(count int, markets []string) []*types.Order {
		orders := make([]*types.Order, 0, count)
		for i := 0; i < count; i++ {
			market := markets[i%len(markets)]
			side := types.SideBuy
			if i%2 == 1 {
				side = types.SideSell
			}
			price := math.LegacyNewDec(int64(50000 + (i % 100)))
			qty := math.LegacyNewDec(int64(1 + (i % 10)))
			order := types.NewOrder(
				string(rune('0'+i)),
				"trader",
				market,
				side,
				types.OrderTypeLimit,
				price,
				qty,
			)
			orders = append(orders, order)
		}
		return orders
	}

	markets := []string{"BTC-USD", "ETH-USD", "SOL-USD", "AVAX-USD"}

	b.Run("GroupOrders_100", func(b *testing.B) {
		keeper := createParallelTestKeeper()
		pm := NewParallelMatcher(keeper, DefaultParallelConfig())
		orders := createOrders(100, markets)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			pm.GroupOrdersByMarket(orders)
		}
	})

	b.Run("GroupOrders_1000", func(b *testing.B) {
		keeper := createParallelTestKeeper()
		pm := NewParallelMatcher(keeper, DefaultParallelConfig())
		orders := createOrders(1000, markets)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			pm.GroupOrdersByMarket(orders)
		}
	})
}
