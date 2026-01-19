package keeper

import (
	"fmt"
	"math/rand"
	"testing"
	"time"

	"cosmossdk.io/log"
	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/openalpha/perp-dex/x/orderbook/types"

	"cosmossdk.io/store"
	"cosmossdk.io/store/metrics"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
)

// mockBenchPerpetualKeeper is a mock implementation for benchmarks
type mockBenchPerpetualKeeper struct{}

func (m *mockBenchPerpetualKeeper) GetMarket(ctx sdk.Context, marketID string) *Market {
	return &Market{
		MarketID:      marketID,
		TakerFeeRate:  math.LegacyNewDecWithPrec(1, 4),  // 0.01%
		MakerFeeRate:  math.LegacyNewDecWithPrec(5, 5),  // 0.005%
		InitialMargin: math.LegacyNewDecWithPrec(10, 2), // 10%
	}
}

func (m *mockBenchPerpetualKeeper) GetMarkPrice(ctx sdk.Context, marketID string) (math.LegacyDec, bool) {
	return math.LegacyNewDec(50000), true
}

func (m *mockBenchPerpetualKeeper) UpdatePosition(ctx sdk.Context, trader, marketID string, side types.Side, qty, price, fee interface{}) error {
	return nil
}

func (m *mockBenchPerpetualKeeper) CheckMarginRequirement(ctx sdk.Context, trader, marketID string, side types.Side, qty, price interface{}) error {
	return nil
}

// setupBenchKeeper creates a test keeper with in-memory store for benchmarks
func setupBenchKeeper(tb testing.TB) (*Keeper, sdk.Context) {
	tb.Helper()

	storeKey := storetypes.NewKVStoreKey("orderbook")
	db := dbm.NewMemDB()
	stateStore := store.NewCommitMultiStore(db, log.NewNopLogger(), metrics.NewNoOpMetrics())
	stateStore.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, db)
	if err := stateStore.LoadLatestVersion(); err != nil {
		tb.Fatalf("failed to load store: %v", err)
	}

	ctx := sdk.NewContext(stateStore, cmtproto.Header{}, false, log.NewNopLogger())

	interfaceRegistry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(interfaceRegistry)

	keeper := NewKeeper(cdc, storeKey, &mockBenchPerpetualKeeper{}, log.NewNopLogger())

	return keeper, ctx
}

// generateBenchOrders generates n random orders for benchmarking
func generateBenchOrders(n int, marketID string) []*types.Order {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	orders := make([]*types.Order, n)

	basePrice := 50000.0 // Base price around $50,000

	for i := 0; i < n; i++ {
		var side types.Side
		var price math.LegacyDec

		if r.Float32() < 0.5 {
			side = types.SideBuy
			// Bids: slightly below base price
			price = math.LegacyNewDecWithPrec(int64((basePrice-r.Float64()*100)*100), 2)
		} else {
			side = types.SideSell
			// Asks: slightly above base price
			price = math.LegacyNewDecWithPrec(int64((basePrice+r.Float64()*100)*100), 2)
		}

		quantity := math.LegacyNewDecWithPrec(int64((0.1+r.Float64()*10)*1000), 3)

		orders[i] = types.NewOrder(
			fmt.Sprintf("bench-order-%d", i),
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

// BenchmarkOldMatching benchmarks the original matching engine
func BenchmarkOldMatching(b *testing.B) {
	keeper, ctx := setupBenchKeeper(b)
	marketID := "BTC-USD"
	orders := generateBenchOrders(1000, marketID)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Clear previous state
		kvStore := keeper.GetStore(ctx)
		iterator := storetypes.KVStorePrefixIterator(kvStore, OrderKeyPrefix)
		keysToDelete := make([][]byte, 0)
		for ; iterator.Valid(); iterator.Next() {
			keysToDelete = append(keysToDelete, iterator.Key())
		}
		iterator.Close()
		for _, key := range keysToDelete {
			kvStore.Delete(key)
		}

		iterator = storetypes.KVStorePrefixIterator(kvStore, OrderBookKeyPrefix)
		keysToDelete = make([][]byte, 0)
		for ; iterator.Valid(); iterator.Next() {
			keysToDelete = append(keysToDelete, iterator.Key())
		}
		iterator.Close()
		for _, key := range keysToDelete {
			kvStore.Delete(key)
		}

		// Process orders with old engine
		engine := NewMatchingEngine(keeper)
		for _, order := range orders {
			// Create a copy to avoid mutation issues
			orderCopy := *order
			orderCopy.FilledQty = math.LegacyZeroDec()
			orderCopy.Status = types.OrderStatusOpen
			_, _ = engine.ProcessOrder(ctx, &orderCopy)
		}
	}
}

// BenchmarkNewMatching benchmarks the optimized V2 matching engine
func BenchmarkNewMatching(b *testing.B) {
	keeper, ctx := setupBenchKeeper(b)
	marketID := "BTC-USD"
	orders := generateBenchOrders(1000, marketID)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Clear previous state
		kvStore := keeper.GetStore(ctx)
		iterator := storetypes.KVStorePrefixIterator(kvStore, OrderKeyPrefix)
		keysToDelete := make([][]byte, 0)
		for ; iterator.Valid(); iterator.Next() {
			keysToDelete = append(keysToDelete, iterator.Key())
		}
		iterator.Close()
		for _, key := range keysToDelete {
			kvStore.Delete(key)
		}

		iterator = storetypes.KVStorePrefixIterator(kvStore, OrderBookKeyPrefix)
		keysToDelete = make([][]byte, 0)
		for ; iterator.Valid(); iterator.Next() {
			keysToDelete = append(keysToDelete, iterator.Key())
		}
		iterator.Close()
		for _, key := range keysToDelete {
			kvStore.Delete(key)
		}

		// Process orders with new engine
		engine := NewMatchingEngineV2(keeper)
		for _, order := range orders {
			// Create a copy to avoid mutation issues
			orderCopy := *order
			orderCopy.FilledQty = math.LegacyZeroDec()
			orderCopy.Status = types.OrderStatusOpen
			_, _ = engine.ProcessOrderOptimized(ctx, &orderCopy)
		}
		// Flush all changes at once
		_ = engine.Flush(ctx)
	}
}

// BenchmarkOldAddOrder benchmarks adding orders with the original order book
func BenchmarkOldAddOrder(b *testing.B) {
	marketID := "BTC-USD"
	orders := generateBenchOrders(b.N, marketID)

	b.ResetTimer()
	ob := types.NewOrderBook(marketID)
	for i := 0; i < b.N; i++ {
		ob.AddOrder(orders[i])
	}
}

// BenchmarkNewAddOrder benchmarks adding orders with the V2 order book
func BenchmarkNewAddOrder(b *testing.B) {
	marketID := "BTC-USD"
	orders := generateBenchOrders(b.N, marketID)

	b.ResetTimer()
	ob := NewOrderBookV2(marketID)
	for i := 0; i < b.N; i++ {
		ob.AddOrder(orders[i])
	}
}

// BenchmarkOldRemoveOrder benchmarks removing orders with the original order book
func BenchmarkOldRemoveOrder(b *testing.B) {
	marketID := "BTC-USD"
	orders := generateBenchOrders(b.N, marketID)

	// Setup: add all orders first
	ob := types.NewOrderBook(marketID)
	for _, order := range orders {
		ob.AddOrder(order)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ob.RemoveOrder(orders[i])
	}
}

// BenchmarkNewRemoveOrder benchmarks removing orders with the V2 order book
func BenchmarkNewRemoveOrder(b *testing.B) {
	marketID := "BTC-USD"
	orders := generateBenchOrders(b.N, marketID)

	// Setup: add all orders first
	ob := NewOrderBookV2(marketID)
	for _, order := range orders {
		ob.AddOrder(order)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ob.RemoveOrder(orders[i])
	}
}

// BenchmarkOldGetBest benchmarks getting best levels with original order book
func BenchmarkOldGetBest(b *testing.B) {
	marketID := "BTC-USD"
	orders := generateBenchOrders(1000, marketID)

	ob := types.NewOrderBook(marketID)
	for _, order := range orders {
		ob.AddOrder(order)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ob.BestBid()
		_ = ob.BestAsk()
	}
}

// BenchmarkNewGetBest benchmarks getting best levels with V2 order book
func BenchmarkNewGetBest(b *testing.B) {
	marketID := "BTC-USD"
	orders := generateBenchOrders(1000, marketID)

	ob := NewOrderBookV2(marketID)
	for _, order := range orders {
		ob.AddOrder(order)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = ob.GetBestLevels()
	}
}

// BenchmarkMixedOperationsOld tests mixed add/remove operations
func BenchmarkMixedOperationsOld(b *testing.B) {
	marketID := "BTC-USD"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ob := types.NewOrderBook(marketID)
		orders := make([]*types.Order, 100)

		// Add 100 orders
		for j := 0; j < 100; j++ {
			side := types.SideBuy
			if j%2 == 0 {
				side = types.SideSell
			}
			price := math.LegacyNewDecWithPrec(int64(50000+j), 0)
			qty := math.LegacyNewDecWithPrec(1, 0)
			order := types.NewOrder(fmt.Sprintf("order-%d-%d", i, j), "trader", marketID, side, types.OrderTypeLimit, price, qty)
			orders[j] = order
			ob.AddOrder(order)
		}

		// Remove 50 orders
		for j := 0; j < 50; j++ {
			ob.RemoveOrder(orders[j*2])
		}

		// Check best levels
		_ = ob.BestBid()
		_ = ob.BestAsk()
	}
}

// BenchmarkMixedOperationsNew tests mixed add/remove operations with V2
func BenchmarkMixedOperationsNew(b *testing.B) {
	marketID := "BTC-USD"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ob := NewOrderBookV2(marketID)
		orders := make([]*types.Order, 100)

		// Add 100 orders
		for j := 0; j < 100; j++ {
			side := types.SideBuy
			if j%2 == 0 {
				side = types.SideSell
			}
			price := math.LegacyNewDecWithPrec(int64(50000+j), 0)
			qty := math.LegacyNewDecWithPrec(1, 0)
			order := types.NewOrder(fmt.Sprintf("order-%d-%d", i, j), "trader", marketID, side, types.OrderTypeLimit, price, qty)
			orders[j] = order
			ob.AddOrder(order)
		}

		// Remove 50 orders
		for j := 0; j < 50; j++ {
			ob.RemoveOrder(orders[j*2])
		}

		// Check best levels
		_, _ = ob.GetBestLevels()
	}
}

// TestOrderBookV2Correctness tests that V2 produces same results as V1
func TestOrderBookV2Correctness(t *testing.T) {
	marketID := "BTC-USD"
	orders := generateBenchOrders(100, marketID)

	obV1 := types.NewOrderBook(marketID)
	obV2 := NewOrderBookV2(marketID)

	// Add orders to both
	for _, order := range orders {
		obV1.AddOrder(order)
		obV2.AddOrder(order)
	}

	// Compare best levels
	bestBidV1 := obV1.BestBid()
	bestAskV1 := obV1.BestAsk()
	bestBidV2 := obV2.GetBestBid()
	bestAskV2 := obV2.GetBestAsk()

	if bestBidV1 != nil && bestBidV2 != nil {
		if !bestBidV1.Price.Equal(bestBidV2.Price) {
			t.Errorf("Best bid price mismatch: V1=%s, V2=%s", bestBidV1.Price, bestBidV2.Price)
		}
	} else if (bestBidV1 == nil) != (bestBidV2 == nil) {
		t.Error("Best bid nil mismatch")
	}

	if bestAskV1 != nil && bestAskV2 != nil {
		if !bestAskV1.Price.Equal(bestAskV2.Price) {
			t.Errorf("Best ask price mismatch: V1=%s, V2=%s", bestAskV1.Price, bestAskV2.Price)
		}
	} else if (bestAskV1 == nil) != (bestAskV2 == nil) {
		t.Error("Best ask nil mismatch")
	}

	// Test spread
	spreadV1 := obV1.Spread()
	spreadV2 := obV2.GetSpread()
	if !spreadV1.Equal(spreadV2) {
		t.Errorf("Spread mismatch: V1=%s, V2=%s", spreadV1, spreadV2)
	}

	t.Logf("V1 Best Bid: %v, Best Ask: %v, Spread: %s", bestBidV1, bestAskV1, spreadV1)
	t.Logf("V2 Best Bid: %v, Best Ask: %v, Spread: %s", bestBidV2, bestAskV2, spreadV2)
}

// TestMatchingEngineV2Correctness tests that V2 engine produces correct results
func TestMatchingEngineV2Correctness(t *testing.T) {
	keeper, ctx := setupBenchKeeper(t)
	marketID := "BTC-USD"

	// Add some maker orders first
	makerOrders := []*types.Order{
		types.NewOrder("maker-1", "maker", marketID, types.SideSell, types.OrderTypeLimit,
			math.LegacyNewDecWithPrec(50100, 0), math.LegacyNewDecWithPrec(10, 0)),
		types.NewOrder("maker-2", "maker", marketID, types.SideSell, types.OrderTypeLimit,
			math.LegacyNewDecWithPrec(50200, 0), math.LegacyNewDecWithPrec(10, 0)),
		types.NewOrder("maker-3", "maker", marketID, types.SideBuy, types.OrderTypeLimit,
			math.LegacyNewDecWithPrec(49900, 0), math.LegacyNewDecWithPrec(10, 0)),
		types.NewOrder("maker-4", "maker", marketID, types.SideBuy, types.OrderTypeLimit,
			math.LegacyNewDecWithPrec(49800, 0), math.LegacyNewDecWithPrec(10, 0)),
	}

	engine := NewMatchingEngineV2(keeper)

	// Add maker orders
	for _, order := range makerOrders {
		_, err := engine.ProcessOrderOptimized(ctx, order)
		if err != nil {
			t.Fatalf("Failed to add maker order: %v", err)
		}
	}

	// Flush to persist
	if err := engine.Flush(ctx); err != nil {
		t.Fatalf("Failed to flush: %v", err)
	}

	// Check order book state
	ob := engine.GetOrderBookV2(ctx, marketID)
	bestBid, bestAsk := ob.GetBestLevels()

	if bestBid == nil || !bestBid.Price.Equal(math.LegacyNewDecWithPrec(49900, 0)) {
		t.Errorf("Expected best bid 49900, got %v", bestBid)
	}
	if bestAsk == nil || !bestAsk.Price.Equal(math.LegacyNewDecWithPrec(50100, 0)) {
		t.Errorf("Expected best ask 50100, got %v", bestAsk)
	}

	// Place a market buy order that should match
	takerOrder := types.NewOrder("taker-1", "taker", marketID, types.SideBuy, types.OrderTypeMarket,
		math.LegacyZeroDec(), math.LegacyNewDecWithPrec(5, 0))

	result, err := engine.ProcessOrderOptimized(ctx, takerOrder)
	if err != nil {
		t.Fatalf("Failed to process taker order: %v", err)
	}

	if len(result.Trades) != 1 {
		t.Errorf("Expected 1 trade, got %d", len(result.Trades))
	}

	if !result.FilledQty.Equal(math.LegacyNewDecWithPrec(5, 0)) {
		t.Errorf("Expected filled qty 5, got %s", result.FilledQty)
	}

	t.Logf("Trades: %d, Filled: %s, Avg Price: %s", len(result.Trades), result.FilledQty, result.AvgPrice)
}

// BenchmarkCacheFlush benchmarks the cache flush operation
func BenchmarkCacheFlush(b *testing.B) {
	keeper, ctx := setupBenchKeeper(b)
	marketID := "BTC-USD"
	orders := generateBenchOrders(100, marketID)

	for i := 0; i < b.N; i++ {
		b.StopTimer()
		engine := NewMatchingEngineV2(keeper)
		for _, order := range orders {
			orderCopy := *order
			orderCopy.FilledQty = math.LegacyZeroDec()
			orderCopy.Status = types.OrderStatusOpen
			_, _ = engine.ProcessOrderOptimized(ctx, &orderCopy)
		}
		b.StartTimer()

		_ = engine.Flush(ctx)
	}
}

// TestSkiplistOrderBookOperations tests basic skiplist operations
func TestSkiplistOrderBookOperations(t *testing.T) {
	marketID := "BTC-USD"
	ob := NewOrderBookV2(marketID)

	// Test adding orders
	order1 := types.NewOrder("order-1", "trader1", marketID, types.SideBuy, types.OrderTypeLimit,
		math.LegacyNewDec(50000), math.LegacyNewDec(1))
	order2 := types.NewOrder("order-2", "trader2", marketID, types.SideBuy, types.OrderTypeLimit,
		math.LegacyNewDec(49900), math.LegacyNewDec(2))
	order3 := types.NewOrder("order-3", "trader3", marketID, types.SideSell, types.OrderTypeLimit,
		math.LegacyNewDec(50100), math.LegacyNewDec(1))
	order4 := types.NewOrder("order-4", "trader4", marketID, types.SideSell, types.OrderTypeLimit,
		math.LegacyNewDec(50200), math.LegacyNewDec(2))

	ob.AddOrder(order1)
	ob.AddOrder(order2)
	ob.AddOrder(order3)
	ob.AddOrder(order4)

	// Test best levels
	bestBid := ob.GetBestBid()
	if bestBid == nil || !bestBid.Price.Equal(math.LegacyNewDec(50000)) {
		t.Errorf("Expected best bid 50000, got %v", bestBid)
	}

	bestAsk := ob.GetBestAsk()
	if bestAsk == nil || !bestAsk.Price.Equal(math.LegacyNewDec(50100)) {
		t.Errorf("Expected best ask 50100, got %v", bestAsk)
	}

	// Test depth
	bidLevels, askLevels := ob.GetDepth()
	if bidLevels != 2 {
		t.Errorf("Expected 2 bid levels, got %d", bidLevels)
	}
	if askLevels != 2 {
		t.Errorf("Expected 2 ask levels, got %d", askLevels)
	}

	// Test spread
	spread := ob.GetSpread()
	expectedSpread := math.LegacyNewDec(100) // 50100 - 50000
	if !spread.Equal(expectedSpread) {
		t.Errorf("Expected spread %s, got %s", expectedSpread, spread)
	}

	// Test mid price
	midPrice := ob.GetMidPrice()
	expectedMid := math.LegacyNewDec(50050) // (50000 + 50100) / 2
	if !midPrice.Equal(expectedMid) {
		t.Errorf("Expected mid price %s, got %s", expectedMid, midPrice)
	}

	// Test remove order
	ob.RemoveOrder(order1)
	bestBid = ob.GetBestBid()
	if bestBid == nil || !bestBid.Price.Equal(math.LegacyNewDec(49900)) {
		t.Errorf("After removal, expected best bid 49900, got %v", bestBid)
	}

	// Test conversion to V1
	obV1 := ob.ToOrderBook()
	if obV1.MarketID != marketID {
		t.Errorf("Expected market ID %s, got %s", marketID, obV1.MarketID)
	}
	if len(obV1.Bids) != 1 {
		t.Errorf("Expected 1 bid level, got %d", len(obV1.Bids))
	}
	if len(obV1.Asks) != 2 {
		t.Errorf("Expected 2 ask levels, got %d", len(obV1.Asks))
	}
}

// TestIterateLevels tests the iteration functionality
func TestIterateLevels(t *testing.T) {
	marketID := "BTC-USD"
	ob := NewOrderBookV2(marketID)

	// Add orders at different price levels
	prices := []int64{50000, 49900, 49800, 49700, 49600}
	for i, p := range prices {
		order := types.NewOrder(fmt.Sprintf("order-%d", i), "trader", marketID,
			types.SideBuy, types.OrderTypeLimit, math.LegacyNewDec(p), math.LegacyNewDec(1))
		ob.AddOrder(order)
	}

	// Test iterate bids (should be in descending order)
	var visitedPrices []math.LegacyDec
	ob.IterateBids(func(level *PriceLevelV2) bool {
		visitedPrices = append(visitedPrices, level.Price)
		return true
	})

	expectedOrder := []int64{50000, 49900, 49800, 49700, 49600}
	for i, p := range expectedOrder {
		expected := math.LegacyNewDec(p)
		if !visitedPrices[i].Equal(expected) {
			t.Errorf("Expected price %d at position %d, got %s", p, i, visitedPrices[i])
		}
	}

	// Test GetBidLevels
	topBids := ob.GetBidLevels(3)
	if len(topBids) != 3 {
		t.Errorf("Expected 3 bid levels, got %d", len(topBids))
	}
	if !topBids[0].Price.Equal(math.LegacyNewDec(50000)) {
		t.Errorf("Expected top bid 50000, got %s", topBids[0].Price)
	}
}
