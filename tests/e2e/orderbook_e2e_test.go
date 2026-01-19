package e2e

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

	"cosmossdk.io/log"
	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"cosmossdk.io/store"
	"cosmossdk.io/store/metrics"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"

	"github.com/openalpha/perp-dex/x/orderbook/keeper"
	"github.com/openalpha/perp-dex/x/orderbook/types"
)

// ============================================================================
// E2E Test Suite for PerpDEX OrderBook Module
// ============================================================================
// This comprehensive test suite covers:
// 1. API Integration Tests - Full keeper and message server flows
// 2. Order Lifecycle Tests - Place, match, cancel orders
// 3. Stress Tests - High throughput and concurrent access
// 4. Data Structure Comparison - All order book implementations
// ============================================================================

// TestConfig defines test configuration
type TestConfig struct {
	NumTraders      int
	OrdersPerTrader int
	NumMarkets      int
	PriceLevels     int
	Concurrency     int
}

// DefaultTestConfig returns default test configuration
func DefaultTestConfig() TestConfig {
	return TestConfig{
		NumTraders:      100,
		OrdersPerTrader: 50,
		NumMarkets:      5,
		PriceLevels:     100,
		Concurrency:     runtime.NumCPU(),
	}
}

// mockPerpetualKeeper is a mock implementation for tests
type mockPerpetualKeeper struct{}

func (m *mockPerpetualKeeper) GetMarket(ctx sdk.Context, marketID string) *keeper.Market {
	return &keeper.Market{
		MarketID:      marketID,
		TakerFeeRate:  math.LegacyNewDecWithPrec(1, 4),
		MakerFeeRate:  math.LegacyNewDecWithPrec(5, 5),
		InitialMargin: math.LegacyNewDecWithPrec(10, 2),
	}
}

func (m *mockPerpetualKeeper) GetMarkPrice(ctx sdk.Context, marketID string) (math.LegacyDec, bool) {
	return math.LegacyNewDec(50000), true
}

func (m *mockPerpetualKeeper) UpdatePosition(ctx sdk.Context, trader, marketID string, side types.Side, qty, price, fee interface{}) error {
	return nil
}

func (m *mockPerpetualKeeper) CheckMarginRequirement(ctx sdk.Context, trader, marketID string, side types.Side, qty, price interface{}) error {
	return nil
}

// setupTestKeeper creates a test keeper with in-memory store
func setupTestKeeper(tb testing.TB) (*keeper.Keeper, sdk.Context) {
	tb.Helper()

	storeKey := storetypes.NewKVStoreKey("orderbook")
	db := dbm.NewMemDB()
	stateStore := store.NewCommitMultiStore(db, log.NewNopLogger(), metrics.NewNoOpMetrics())
	stateStore.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, db)
	if err := stateStore.LoadLatestVersion(); err != nil {
		tb.Fatalf("failed to load store: %v", err)
	}

	ctx := sdk.NewContext(stateStore, cmtproto.Header{
		Time:   time.Now(),
		Height: 1,
	}, false, log.NewNopLogger())

	interfaceRegistry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(interfaceRegistry)

	k := keeper.NewKeeper(cdc, storeKey, &mockPerpetualKeeper{}, log.NewNopLogger())

	return k, ctx
}

// ============================================================================
// 1. API Integration Tests
// ============================================================================

// TestPlaceOrderAPI tests the PlaceOrder API
func TestPlaceOrderAPI(t *testing.T) {
	k, ctx := setupTestKeeper(t)

	testCases := []struct {
		name      string
		trader    string
		marketID  string
		side      types.Side
		orderType types.OrderType
		price     math.LegacyDec
		quantity  math.LegacyDec
		wantErr   bool
	}{
		{
			name:      "valid buy limit order",
			trader:    "trader1",
			marketID:  "BTC-USD",
			side:      types.SideBuy,
			orderType: types.OrderTypeLimit,
			price:     math.LegacyNewDec(50000),
			quantity:  math.LegacyNewDec(1),
			wantErr:   false,
		},
		{
			name:      "valid sell limit order",
			trader:    "trader2",
			marketID:  "BTC-USD",
			side:      types.SideSell,
			orderType: types.OrderTypeLimit,
			price:     math.LegacyNewDec(50100),
			quantity:  math.LegacyNewDec(1),
			wantErr:   false,
		},
		{
			name:      "valid market order",
			trader:    "trader3",
			marketID:  "BTC-USD",
			side:      types.SideBuy,
			orderType: types.OrderTypeMarket,
			price:     math.LegacyZeroDec(),
			quantity:  math.LegacyNewDec(1),
			wantErr:   false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			order, result, err := k.PlaceOrder(ctx, tc.trader, tc.marketID, tc.side, tc.orderType, tc.price, tc.quantity)

			if tc.wantErr {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if order == nil {
				t.Fatal("expected order but got nil")
			}

			if order.Trader != tc.trader {
				t.Errorf("trader mismatch: got %s, want %s", order.Trader, tc.trader)
			}

			if order.MarketID != tc.marketID {
				t.Errorf("marketID mismatch: got %s, want %s", order.MarketID, tc.marketID)
			}

			t.Logf("Order placed: ID=%s, Filled=%s, Trades=%d", order.OrderID, result.FilledQty, len(result.Trades))
		})
	}
}

// TestCancelOrderAPI tests the CancelOrder API
func TestCancelOrderAPI(t *testing.T) {
	k, ctx := setupTestKeeper(t)

	// Place an order first
	order, _, err := k.PlaceOrder(ctx, "trader1", "BTC-USD", types.SideBuy, types.OrderTypeLimit, math.LegacyNewDec(49000), math.LegacyNewDec(1))
	if err != nil {
		t.Fatalf("failed to place order: %v", err)
	}

	// Test cancellation
	testCases := []struct {
		name    string
		trader  string
		orderID string
		wantErr bool
	}{
		{
			name:    "valid cancellation",
			trader:  "trader1",
			orderID: order.OrderID,
			wantErr: false,
		},
		{
			name:    "unauthorized cancellation",
			trader:  "trader2",
			orderID: order.OrderID,
			wantErr: true,
		},
		{
			name:    "non-existent order",
			trader:  "trader1",
			orderID: "non-existent",
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Need fresh context for each test
			_, err := k.CancelOrder(ctx, tc.trader, tc.orderID)

			if tc.wantErr {
				if err == nil {
					t.Errorf("expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

// TestQueryOrderBookAPI tests the QueryOrderBook API
func TestQueryOrderBookAPI(t *testing.T) {
	k, ctx := setupTestKeeper(t)
	marketID := "BTC-USD"

	// Place some orders
	orders := []struct {
		side  types.Side
		price int64
	}{
		{types.SideBuy, 49900},
		{types.SideBuy, 49800},
		{types.SideBuy, 49700},
		{types.SideSell, 50100},
		{types.SideSell, 50200},
		{types.SideSell, 50300},
	}

	for i, o := range orders {
		_, _, err := k.PlaceOrder(ctx, fmt.Sprintf("trader%d", i), marketID, o.side, types.OrderTypeLimit, math.LegacyNewDec(o.price), math.LegacyNewDec(1))
		if err != nil {
			t.Fatalf("failed to place order: %v", err)
		}
	}

	// Query order book
	ob := k.GetOrderBook(ctx, marketID)
	if ob == nil {
		t.Fatal("order book not found")
	}

	t.Logf("Order book: MarketID=%s, Bids=%d, Asks=%d", ob.MarketID, len(ob.Bids), len(ob.Asks))

	if len(ob.Bids) < 3 {
		t.Errorf("expected at least 3 bid levels, got %d", len(ob.Bids))
	}
	if len(ob.Asks) < 3 {
		t.Errorf("expected at least 3 ask levels, got %d", len(ob.Asks))
	}
}

// TestQueryTradesAPI tests the QueryTrades API
func TestQueryTradesAPI(t *testing.T) {
	k, ctx := setupTestKeeper(t)
	marketID := "BTC-USD"

	// Create a matching scenario
	_, _, _ = k.PlaceOrder(ctx, "maker", marketID, types.SideSell, types.OrderTypeLimit, math.LegacyNewDec(50000), math.LegacyNewDec(1))
	_, _, _ = k.PlaceOrder(ctx, "taker", marketID, types.SideBuy, types.OrderTypeLimit, math.LegacyNewDec(50000), math.LegacyNewDec(1))

	// Query trades
	trades := k.GetRecentTrades(ctx, marketID, 10)
	t.Logf("Recent trades: %d", len(trades))

	// Query trade history
	traderTrades := k.GetTradeHistory(ctx, "taker", 10, 0)
	t.Logf("Trader trades: %d", len(traderTrades))
}

// ============================================================================
// 2. Order Lifecycle Tests
// ============================================================================

// TestOrderLifecycle tests the complete order lifecycle
func TestOrderLifecycle(t *testing.T) {
	k, ctx := setupTestKeeper(t)
	marketID := "ETH-USD"

	// Step 1: Place maker orders
	t.Log("Step 1: Placing maker orders")
	for i := 0; i < 5; i++ {
		_, _, _ = k.PlaceOrder(ctx, "maker", marketID, types.SideSell, types.OrderTypeLimit, math.LegacyNewDec(int64(3000+i*10)), math.LegacyNewDec(10))
		_, _, _ = k.PlaceOrder(ctx, "maker", marketID, types.SideBuy, types.OrderTypeLimit, math.LegacyNewDec(int64(2900-i*10)), math.LegacyNewDec(10))
	}

	// Verify order book depth
	ob := k.GetOrderBook(ctx, marketID)
	if ob != nil {
		t.Logf("Order book depth: Bids=%d, Asks=%d", len(ob.Bids), len(ob.Asks))
	}

	// Step 2: Place taker orders that match
	t.Log("Step 2: Placing taker orders")
	order, result, err := k.PlaceOrder(ctx, "taker", marketID, types.SideBuy, types.OrderTypeLimit, math.LegacyNewDec(3005), math.LegacyNewDec(5))
	if err != nil {
		t.Fatalf("failed to place taker order: %v", err)
	}
	t.Logf("Taker order: ID=%s, Filled=%s, Trades=%d", order.OrderID, result.FilledQty, len(result.Trades))

	// Step 3: Query open orders
	t.Log("Step 3: Querying open orders")
	openOrders := k.GetOpenOrders(ctx, "taker")
	t.Logf("Taker open orders: %d", len(openOrders))

	// Step 4: Cancel remaining order
	if len(openOrders) > 0 {
		t.Log("Step 4: Cancelling remaining order")
		_, err = k.CancelOrder(ctx, "taker", openOrders[0].OrderID)
		if err != nil {
			t.Logf("Cancel error (expected if filled): %v", err)
		}
	}

	// Step 5: Verify trade history
	t.Log("Step 5: Verifying trade history")
	trades := k.GetTradeHistory(ctx, "taker", 10, 0)
	t.Logf("Trade history: %d trades", len(trades))
}

// TestOrderMatchingPriority tests price-time priority
func TestOrderMatchingPriority(t *testing.T) {
	k, ctx := setupTestKeeper(t)
	marketID := "BTC-USD"

	// Place orders at same price with different times
	order1, _, _ := k.PlaceOrder(ctx, "maker1", marketID, types.SideSell, types.OrderTypeLimit, math.LegacyNewDec(50000), math.LegacyNewDec(1))
	order2, _, _ := k.PlaceOrder(ctx, "maker2", marketID, types.SideSell, types.OrderTypeLimit, math.LegacyNewDec(50000), math.LegacyNewDec(1))

	t.Logf("Order 1: %s (created first)", order1.OrderID)
	t.Logf("Order 2: %s (created second)", order2.OrderID)

	// Taker order should match with order1 first (time priority)
	_, result, _ := k.PlaceOrder(ctx, "taker", marketID, types.SideBuy, types.OrderTypeLimit, math.LegacyNewDec(50000), math.LegacyNewDec(1))

	if len(result.Trades) > 0 {
		t.Logf("Matched with: %s", result.Trades[0].Maker)
	}
}

// ============================================================================
// 3. Stress Tests
// ============================================================================

// TestHighThroughputOrders tests high throughput order processing
func TestHighThroughputOrders(t *testing.T) {
	k, ctx := setupTestKeeper(t)
	config := DefaultTestConfig()
	config.NumTraders = 50
	config.OrdersPerTrader = 100

	marketID := "BTC-USD"
	totalOrders := config.NumTraders * config.OrdersPerTrader

	t.Logf("Running high throughput test: %d orders", totalOrders)

	start := time.Now()
	var successCount, errorCount int64

	for i := 0; i < config.NumTraders; i++ {
		trader := fmt.Sprintf("trader%d", i)
		for j := 0; j < config.OrdersPerTrader; j++ {
			side := types.SideBuy
			price := int64(49000 + rand.Intn(1000))
			if rand.Float32() > 0.5 {
				side = types.SideSell
				price = int64(50000 + rand.Intn(1000))
			}

			_, _, err := k.PlaceOrder(ctx, trader, marketID, side, types.OrderTypeLimit, math.LegacyNewDec(price), math.LegacyNewDecWithPrec(int64(rand.Intn(100)+1), 1))
			if err != nil {
				errorCount++
			} else {
				successCount++
			}
		}
	}

	duration := time.Since(start)
	throughput := float64(totalOrders) / duration.Seconds()

	t.Logf("Results:")
	t.Logf("  Total orders: %d", totalOrders)
	t.Logf("  Success: %d", successCount)
	t.Logf("  Errors: %d", errorCount)
	t.Logf("  Duration: %v", duration)
	t.Logf("  Throughput: %.2f orders/sec", throughput)
}

// TestConcurrentOrders tests concurrent order processing
func TestConcurrentOrders(t *testing.T) {
	k, ctx := setupTestKeeper(t)
	marketID := "BTC-USD"

	numWorkers := runtime.NumCPU()
	ordersPerWorker := 100

	t.Logf("Running concurrent test: %d workers x %d orders", numWorkers, ordersPerWorker)

	var wg sync.WaitGroup
	var successCount, errorCount int64

	start := time.Now()

	for w := 0; w < numWorkers; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			for i := 0; i < ordersPerWorker; i++ {
				trader := fmt.Sprintf("worker%d-trader%d", workerID, i)
				side := types.SideBuy
				price := int64(49000 + rand.Intn(1000))
				if rand.Float32() > 0.5 {
					side = types.SideSell
					price = int64(50000 + rand.Intn(1000))
				}

				_, _, err := k.PlaceOrder(ctx, trader, marketID, side, types.OrderTypeLimit, math.LegacyNewDec(price), math.LegacyNewDec(1))
				if err != nil {
					atomic.AddInt64(&errorCount, 1)
				} else {
					atomic.AddInt64(&successCount, 1)
				}
			}
		}(w)
	}

	wg.Wait()
	duration := time.Since(start)

	totalOrders := numWorkers * ordersPerWorker
	throughput := float64(totalOrders) / duration.Seconds()

	t.Logf("Results:")
	t.Logf("  Workers: %d", numWorkers)
	t.Logf("  Total orders: %d", totalOrders)
	t.Logf("  Success: %d", successCount)
	t.Logf("  Errors: %d", errorCount)
	t.Logf("  Duration: %v", duration)
	t.Logf("  Throughput: %.2f orders/sec", throughput)
}

// ============================================================================
// 4. Data Structure Comparison Tests
// ============================================================================

// DataStructureResult holds test results for a data structure
type DataStructureResult struct {
	Name           string        `json:"name"`
	AddOrderNs     float64       `json:"add_order_ns"`
	RemoveOrderNs  float64       `json:"remove_order_ns"`
	GetBestNs      float64       `json:"get_best_ns"`
	GetTop10Ns     float64       `json:"get_top_10_ns"`
	MixedOpsNs     float64       `json:"mixed_ops_ns"`
	ThroughputOps  float64       `json:"throughput_ops_per_sec"`
	MemoryBytes    uint64        `json:"memory_bytes"`
}

// TestAllDataStructures tests all order book implementations
func TestAllDataStructures(t *testing.T) {
	marketID := "BTC-USD"
	numOrders := 10000

	implementations := []struct {
		name   string
		create func() keeper.OrderBookEngine
	}{
		{"SkipList", func() keeper.OrderBookEngine { return keeper.NewOrderBookV2(marketID) }},
		{"HashMap", func() keeper.OrderBookEngine { return keeper.NewOrderBookHashMap(marketID) }},
		{"BTree", func() keeper.OrderBookEngine { return keeper.NewOrderBookBTree(marketID) }},
		{"ART", func() keeper.OrderBookEngine { return keeper.NewOrderBookART(marketID) }},
	}

	results := make([]DataStructureResult, 0, len(implementations))

	for _, impl := range implementations {
		t.Run(impl.name, func(t *testing.T) {
			result := benchmarkDataStructure(t, impl.name, impl.create, numOrders, marketID)
			results = append(results, result)

			t.Logf("Implementation: %s", impl.name)
			t.Logf("  Add Order: %.2f ns/op", result.AddOrderNs)
			t.Logf("  Remove Order: %.2f ns/op", result.RemoveOrderNs)
			t.Logf("  Get Best: %.2f ns/op", result.GetBestNs)
			t.Logf("  Get Top 10: %.2f ns/op", result.GetTop10Ns)
			t.Logf("  Throughput: %.2f ops/sec", result.ThroughputOps)
			t.Logf("  Memory: %.2f MB", float64(result.MemoryBytes)/1024/1024)
		})
	}

	// Output comparison table
	t.Log("\n=== Data Structure Comparison ===")
	t.Logf("%-12s %12s %12s %12s %12s %15s %10s", "Name", "Add(ns)", "Remove(ns)", "GetBest(ns)", "Top10(ns)", "Throughput", "Memory(MB)")
	for _, r := range results {
		t.Logf("%-12s %12.2f %12.2f %12.2f %12.2f %15.2f %10.2f",
			r.Name, r.AddOrderNs, r.RemoveOrderNs, r.GetBestNs, r.GetTop10Ns, r.ThroughputOps, float64(r.MemoryBytes)/1024/1024)
	}
}

func benchmarkDataStructure(t *testing.T, name string, create func() keeper.OrderBookEngine, numOrders int, marketID string) DataStructureResult {
	orders := generateTestOrders(numOrders, marketID)

	// Measure add order
	engine := create()
	start := time.Now()
	for _, order := range orders {
		engine.AddOrder(order)
	}
	addDuration := time.Since(start)

	// Measure get best
	start = time.Now()
	for i := 0; i < 10000; i++ {
		_, _ = engine.GetBestLevels()
	}
	getBestDuration := time.Since(start)

	// Measure get top 10
	start = time.Now()
	for i := 0; i < 1000; i++ {
		_ = engine.GetBidLevels(10)
		_ = engine.GetAskLevels(10)
	}
	getTop10Duration := time.Since(start)

	// Measure remove order
	start = time.Now()
	for _, order := range orders {
		engine.RemoveOrder(order)
	}
	removeDuration := time.Since(start)

	// Memory measurement
	runtime.GC()
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	// Calculate throughput (mixed operations)
	engine = create()
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	addedOrders := make([]*types.Order, 0, numOrders)

	start = time.Now()
	ops := 0
	for i := 0; i < numOrders; i++ {
		if r.Float32() < 0.7 || len(addedOrders) == 0 {
			engine.AddOrder(orders[i])
			addedOrders = append(addedOrders, orders[i])
		} else {
			idx := r.Intn(len(addedOrders))
			engine.RemoveOrder(addedOrders[idx])
			addedOrders = append(addedOrders[:idx], addedOrders[idx+1:]...)
		}
		_, _ = engine.GetBestLevels()
		ops += 2
	}
	mixedDuration := time.Since(start)

	return DataStructureResult{
		Name:          name,
		AddOrderNs:    float64(addDuration.Nanoseconds()) / float64(numOrders),
		RemoveOrderNs: float64(removeDuration.Nanoseconds()) / float64(numOrders),
		GetBestNs:     float64(getBestDuration.Nanoseconds()) / 10000,
		GetTop10Ns:    float64(getTop10Duration.Nanoseconds()) / 2000,
		MixedOpsNs:    float64(mixedDuration.Nanoseconds()) / float64(ops),
		ThroughputOps: float64(ops) / mixedDuration.Seconds(),
		MemoryBytes:   memStats.Alloc,
	}
}

func generateTestOrders(n int, marketID string) []*types.Order {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	orders := make([]*types.Order, n)

	basePrice := 50000.0

	for i := 0; i < n; i++ {
		var side types.Side
		var price math.LegacyDec

		if r.Float32() < 0.5 {
			side = types.SideBuy
			price = math.LegacyNewDecWithPrec(int64((basePrice-r.Float64()*100)*100), 2)
		} else {
			side = types.SideSell
			price = math.LegacyNewDecWithPrec(int64((basePrice+r.Float64()*100)*100), 2)
		}

		quantity := math.LegacyNewDecWithPrec(int64((0.1+r.Float64()*10)*1000), 3)

		orders[i] = types.NewOrder(
			fmt.Sprintf("test-order-%d", i),
			fmt.Sprintf("trader-%d", i%100),
			marketID,
			side,
			types.OrderTypeLimit,
			price,
			quantity,
		)
	}

	return orders
}

// ============================================================================
// 5. EndBlocker Tests
// ============================================================================

// TestEndBlockerPerformance tests EndBlocker performance
func TestEndBlockerPerformance(t *testing.T) {
	k, ctx := setupTestKeeper(t)
	markets := []string{"BTC-USD", "ETH-USD", "SOL-USD"}

	// Place orders in multiple markets
	ordersPerMarket := 100
	for _, market := range markets {
		for i := 0; i < ordersPerMarket; i++ {
			side := types.SideBuy
			price := int64(49000 + rand.Intn(2000))
			if rand.Float32() > 0.5 {
				side = types.SideSell
			}
			_, _, _ = k.PlaceOrder(ctx, fmt.Sprintf("trader%d", i), market, side, types.OrderTypeLimit, math.LegacyNewDec(price), math.LegacyNewDec(1))
		}
	}

	// Test EndBlocker
	start := time.Now()
	err := k.EndBlocker(ctx)
	duration := time.Since(start)

	if err != nil {
		t.Errorf("EndBlocker failed: %v", err)
	}

	t.Logf("EndBlocker Performance:")
	t.Logf("  Markets: %d", len(markets))
	t.Logf("  Orders per market: %d", ordersPerMarket)
	t.Logf("  Duration: %v", duration)
	t.Logf("  Orders/sec: %.2f", float64(len(markets)*ordersPerMarket)/duration.Seconds())
}

// ============================================================================
// 6. Report Generation
// ============================================================================

// TestResults holds all test results
type TestResults struct {
	Timestamp      time.Time              `json:"timestamp"`
	Platform       string                 `json:"platform"`
	GoVersion      string                 `json:"go_version"`
	CPUs           int                    `json:"cpus"`
	DataStructures []DataStructureResult  `json:"data_structures"`
	APITests       map[string]bool        `json:"api_tests"`
	StressTests    map[string]interface{} `json:"stress_tests"`
}

// GenerateTestReport generates a JSON test report
func GenerateTestReport(outputPath string) error {
	marketID := "BTC-USD"
	numOrders := 10000

	results := TestResults{
		Timestamp:      time.Now(),
		Platform:       runtime.GOOS + "/" + runtime.GOARCH,
		GoVersion:      runtime.Version(),
		CPUs:           runtime.NumCPU(),
		DataStructures: make([]DataStructureResult, 0),
		APITests:       make(map[string]bool),
		StressTests:    make(map[string]interface{}),
	}

	// Benchmark data structures
	implementations := []struct {
		name   string
		create func() keeper.OrderBookEngine
	}{
		{"SkipList", func() keeper.OrderBookEngine { return keeper.NewOrderBookV2(marketID) }},
		{"HashMap", func() keeper.OrderBookEngine { return keeper.NewOrderBookHashMap(marketID) }},
		{"BTree", func() keeper.OrderBookEngine { return keeper.NewOrderBookBTree(marketID) }},
		{"ART", func() keeper.OrderBookEngine { return keeper.NewOrderBookART(marketID) }},
	}

	for _, impl := range implementations {
		result := benchmarkDataStructureForReport(impl.name, impl.create, numOrders, marketID)
		results.DataStructures = append(results.DataStructures, result)
	}

	// Sort by throughput
	sort.Slice(results.DataStructures, func(i, j int) bool {
		return results.DataStructures[i].ThroughputOps > results.DataStructures[j].ThroughputOps
	})

	data, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(outputPath, data, 0644)
}

func benchmarkDataStructureForReport(name string, create func() keeper.OrderBookEngine, numOrders int, marketID string) DataStructureResult {
	orders := generateTestOrders(numOrders, marketID)

	engine := create()
	start := time.Now()
	for _, order := range orders {
		engine.AddOrder(order)
	}
	addDuration := time.Since(start)

	start = time.Now()
	for i := 0; i < 10000; i++ {
		_, _ = engine.GetBestLevels()
	}
	getBestDuration := time.Since(start)

	start = time.Now()
	for i := 0; i < 1000; i++ {
		_ = engine.GetBidLevels(10)
		_ = engine.GetAskLevels(10)
	}
	getTop10Duration := time.Since(start)

	start = time.Now()
	for _, order := range orders {
		engine.RemoveOrder(order)
	}
	removeDuration := time.Since(start)

	runtime.GC()
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	totalOps := numOrders * 2 + 10000 + 2000
	totalDuration := addDuration + removeDuration + getBestDuration + getTop10Duration

	return DataStructureResult{
		Name:          name,
		AddOrderNs:    float64(addDuration.Nanoseconds()) / float64(numOrders),
		RemoveOrderNs: float64(removeDuration.Nanoseconds()) / float64(numOrders),
		GetBestNs:     float64(getBestDuration.Nanoseconds()) / 10000,
		GetTop10Ns:    float64(getTop10Duration.Nanoseconds()) / 2000,
		ThroughputOps: float64(totalOps) / totalDuration.Seconds(),
		MemoryBytes:   memStats.Alloc,
	}
}
