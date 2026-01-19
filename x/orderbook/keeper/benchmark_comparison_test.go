package keeper

import (
	"fmt"
	"math/rand"
	"testing"
	"time"

	"cosmossdk.io/math"
	"github.com/openalpha/perp-dex/x/orderbook/types"
)

// ============================================================================
// Benchmark Comparison Tests
// ============================================================================
// Compares performance of all order book implementations:
// 1. Skip List (OrderBookV2) - Original implementation
// 2. HashMap + Heap (OrderBookHashMap) - dYdX style
// 3. B+Tree (OrderBookBTree) - CEX style (Bybit, Binance)
// 4. ART (OrderBookART) - ExchangeCore style
// ============================================================================

// generateComparisonOrders generates n random orders for comparison benchmarks
func generateComparisonOrders(n int, marketID string) []*types.Order {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	orders := make([]*types.Order, n)

	basePrice := 50000.0 // Base price around $50,000

	for i := 0; i < n; i++ {
		var side types.Side
		var price math.LegacyDec

		if r.Float32() < 0.5 {
			side = types.SideBuy
			// Bids: slightly below base price, spread across 100 levels
			price = math.LegacyNewDecWithPrec(int64((basePrice-r.Float64()*100)*100), 2)
		} else {
			side = types.SideSell
			// Asks: slightly above base price, spread across 100 levels
			price = math.LegacyNewDecWithPrec(int64((basePrice+r.Float64()*100)*100), 2)
		}

		quantity := math.LegacyNewDecWithPrec(int64((0.1+r.Float64()*10)*1000), 3)

		orders[i] = types.NewOrder(
			fmt.Sprintf("comp-order-%d", i),
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
// Add Order Benchmarks
// ============================================================================

func BenchmarkAddOrder_SkipList(b *testing.B) {
	marketID := "BTC-USD"
	orders := generateComparisonOrders(b.N, marketID)

	b.ResetTimer()
	ob := NewOrderBookV2(marketID)
	for i := 0; i < b.N; i++ {
		ob.AddOrder(orders[i])
	}
}

func BenchmarkAddOrder_HashMap(b *testing.B) {
	marketID := "BTC-USD"
	orders := generateComparisonOrders(b.N, marketID)

	b.ResetTimer()
	ob := NewOrderBookHashMap(marketID)
	for i := 0; i < b.N; i++ {
		ob.AddOrder(orders[i])
	}
}

func BenchmarkAddOrder_BTree(b *testing.B) {
	marketID := "BTC-USD"
	orders := generateComparisonOrders(b.N, marketID)

	b.ResetTimer()
	ob := NewOrderBookBTree(marketID)
	for i := 0; i < b.N; i++ {
		ob.AddOrder(orders[i])
	}
}

func BenchmarkAddOrder_ART(b *testing.B) {
	marketID := "BTC-USD"
	orders := generateComparisonOrders(b.N, marketID)

	b.ResetTimer()
	ob := NewOrderBookART(marketID)
	for i := 0; i < b.N; i++ {
		ob.AddOrder(orders[i])
	}
}

// ============================================================================
// Remove Order Benchmarks
// ============================================================================

func BenchmarkRemoveOrder_SkipList(b *testing.B) {
	marketID := "BTC-USD"
	orders := generateComparisonOrders(b.N, marketID)

	ob := NewOrderBookV2(marketID)
	for _, order := range orders {
		ob.AddOrder(order)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ob.RemoveOrder(orders[i])
	}
}

func BenchmarkRemoveOrder_HashMap(b *testing.B) {
	marketID := "BTC-USD"
	orders := generateComparisonOrders(b.N, marketID)

	ob := NewOrderBookHashMap(marketID)
	for _, order := range orders {
		ob.AddOrder(order)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ob.RemoveOrder(orders[i])
	}
}

func BenchmarkRemoveOrder_BTree(b *testing.B) {
	marketID := "BTC-USD"
	orders := generateComparisonOrders(b.N, marketID)

	ob := NewOrderBookBTree(marketID)
	for _, order := range orders {
		ob.AddOrder(order)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ob.RemoveOrder(orders[i])
	}
}

func BenchmarkRemoveOrder_ART(b *testing.B) {
	marketID := "BTC-USD"
	orders := generateComparisonOrders(b.N, marketID)

	ob := NewOrderBookART(marketID)
	for _, order := range orders {
		ob.AddOrder(order)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ob.RemoveOrder(orders[i])
	}
}

// ============================================================================
// Get Best Levels Benchmarks
// ============================================================================

func BenchmarkGetBest_SkipList(b *testing.B) {
	marketID := "BTC-USD"
	orders := generateComparisonOrders(1000, marketID)

	ob := NewOrderBookV2(marketID)
	for _, order := range orders {
		ob.AddOrder(order)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = ob.GetBestLevels()
	}
}

func BenchmarkGetBest_HashMap(b *testing.B) {
	marketID := "BTC-USD"
	orders := generateComparisonOrders(1000, marketID)

	ob := NewOrderBookHashMap(marketID)
	for _, order := range orders {
		ob.AddOrder(order)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = ob.GetBestLevels()
	}
}

func BenchmarkGetBest_BTree(b *testing.B) {
	marketID := "BTC-USD"
	orders := generateComparisonOrders(1000, marketID)

	ob := NewOrderBookBTree(marketID)
	for _, order := range orders {
		ob.AddOrder(order)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = ob.GetBestLevels()
	}
}

func BenchmarkGetBest_ART(b *testing.B) {
	marketID := "BTC-USD"
	orders := generateComparisonOrders(1000, marketID)

	ob := NewOrderBookART(marketID)
	for _, order := range orders {
		ob.AddOrder(order)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = ob.GetBestLevels()
	}
}

// ============================================================================
// Get Top N Levels Benchmarks
// ============================================================================

func BenchmarkGetTop10_SkipList(b *testing.B) {
	marketID := "BTC-USD"
	orders := generateComparisonOrders(1000, marketID)

	ob := NewOrderBookV2(marketID)
	for _, order := range orders {
		ob.AddOrder(order)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ob.GetBidLevels(10)
		_ = ob.GetAskLevels(10)
	}
}

func BenchmarkGetTop10_HashMap(b *testing.B) {
	marketID := "BTC-USD"
	orders := generateComparisonOrders(1000, marketID)

	ob := NewOrderBookHashMap(marketID)
	for _, order := range orders {
		ob.AddOrder(order)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ob.GetBidLevels(10)
		_ = ob.GetAskLevels(10)
	}
}

func BenchmarkGetTop10_BTree(b *testing.B) {
	marketID := "BTC-USD"
	orders := generateComparisonOrders(1000, marketID)

	ob := NewOrderBookBTree(marketID)
	for _, order := range orders {
		ob.AddOrder(order)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ob.GetBidLevels(10)
		_ = ob.GetAskLevels(10)
	}
}

func BenchmarkGetTop10_ART(b *testing.B) {
	marketID := "BTC-USD"
	orders := generateComparisonOrders(1000, marketID)

	ob := NewOrderBookART(marketID)
	for _, order := range orders {
		ob.AddOrder(order)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ob.GetBidLevels(10)
		_ = ob.GetAskLevels(10)
	}
}

// ============================================================================
// Mixed Operations Benchmarks (Add + Remove + Query)
// ============================================================================

func BenchmarkMixedOps_SkipList(b *testing.B) {
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

		// Query best levels 10 times
		for j := 0; j < 10; j++ {
			_, _ = ob.GetBestLevels()
		}

		// Remove 50 orders
		for j := 0; j < 50; j++ {
			ob.RemoveOrder(orders[j*2])
		}
	}
}

func BenchmarkMixedOps_HashMap(b *testing.B) {
	marketID := "BTC-USD"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ob := NewOrderBookHashMap(marketID)
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

		// Query best levels 10 times
		for j := 0; j < 10; j++ {
			_, _ = ob.GetBestLevels()
		}

		// Remove 50 orders
		for j := 0; j < 50; j++ {
			ob.RemoveOrder(orders[j*2])
		}
	}
}

func BenchmarkMixedOps_BTree(b *testing.B) {
	marketID := "BTC-USD"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ob := NewOrderBookBTree(marketID)
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

		// Query best levels 10 times
		for j := 0; j < 10; j++ {
			_, _ = ob.GetBestLevels()
		}

		// Remove 50 orders
		for j := 0; j < 50; j++ {
			ob.RemoveOrder(orders[j*2])
		}
	}
}

func BenchmarkMixedOps_ART(b *testing.B) {
	marketID := "BTC-USD"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ob := NewOrderBookART(marketID)
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

		// Query best levels 10 times
		for j := 0; j < 10; j++ {
			_, _ = ob.GetBestLevels()
		}

		// Remove 50 orders
		for j := 0; j < 50; j++ {
			ob.RemoveOrder(orders[j*2])
		}
	}
}

// ============================================================================
// Correctness Tests
// ============================================================================

// TestAllImplementationsCorrectness verifies all implementations produce same results
func TestAllImplementationsCorrectness(t *testing.T) {
	marketID := "BTC-USD"
	orders := generateComparisonOrders(100, marketID)

	// Create all order books
	obSkipList := NewOrderBookV2(marketID)
	obHashMap := NewOrderBookHashMap(marketID)
	obBTree := NewOrderBookBTree(marketID)
	obART := NewOrderBookART(marketID)

	// Add orders to all
	for _, order := range orders {
		obSkipList.AddOrder(order)
		obHashMap.AddOrder(order)
		obBTree.AddOrder(order)
		obART.AddOrder(order)
	}

	// Compare depths
	slBids, slAsks := obSkipList.GetDepth()
	hmBids, hmAsks := obHashMap.GetDepth()
	btBids, btAsks := obBTree.GetDepth()
	artBids, artAsks := obART.GetDepth()

	if slBids != hmBids || slBids != btBids || slBids != artBids {
		t.Errorf("Bid depth mismatch: SkipList=%d, HashMap=%d, BTree=%d, ART=%d",
			slBids, hmBids, btBids, artBids)
	}
	if slAsks != hmAsks || slAsks != btAsks || slAsks != artAsks {
		t.Errorf("Ask depth mismatch: SkipList=%d, HashMap=%d, BTree=%d, ART=%d",
			slAsks, hmAsks, btAsks, artAsks)
	}

	// Compare best levels
	slBestBid := obSkipList.GetBestBid()
	hmBestBid := obHashMap.GetBestBid()
	btBestBid := obBTree.GetBestBid()
	artBestBid := obART.GetBestBid()

	if slBestBid != nil && hmBestBid != nil && btBestBid != nil && artBestBid != nil {
		if !slBestBid.Price.Equal(hmBestBid.Price) {
			t.Errorf("Best bid mismatch: SkipList=%s, HashMap=%s", slBestBid.Price, hmBestBid.Price)
		}
		if !slBestBid.Price.Equal(btBestBid.Price) {
			t.Errorf("Best bid mismatch: SkipList=%s, BTree=%s", slBestBid.Price, btBestBid.Price)
		}
		if !slBestBid.Price.Equal(artBestBid.Price) {
			t.Errorf("Best bid mismatch: SkipList=%s, ART=%s", slBestBid.Price, artBestBid.Price)
		}
	}

	slBestAsk := obSkipList.GetBestAsk()
	hmBestAsk := obHashMap.GetBestAsk()
	btBestAsk := obBTree.GetBestAsk()
	artBestAsk := obART.GetBestAsk()

	if slBestAsk != nil && hmBestAsk != nil && btBestAsk != nil && artBestAsk != nil {
		if !slBestAsk.Price.Equal(hmBestAsk.Price) {
			t.Errorf("Best ask mismatch: SkipList=%s, HashMap=%s", slBestAsk.Price, hmBestAsk.Price)
		}
		if !slBestAsk.Price.Equal(btBestAsk.Price) {
			t.Errorf("Best ask mismatch: SkipList=%s, BTree=%s", slBestAsk.Price, btBestAsk.Price)
		}
		if !slBestAsk.Price.Equal(artBestAsk.Price) {
			t.Errorf("Best ask mismatch: SkipList=%s, ART=%s", slBestAsk.Price, artBestAsk.Price)
		}
	}

	// Compare spreads
	slSpread := obSkipList.GetSpread()
	hmSpread := obHashMap.GetSpread()
	btSpread := obBTree.GetSpread()
	artSpread := obART.GetSpread()

	if !slSpread.Equal(hmSpread) {
		t.Errorf("Spread mismatch: SkipList=%s, HashMap=%s", slSpread, hmSpread)
	}
	if !slSpread.Equal(btSpread) {
		t.Errorf("Spread mismatch: SkipList=%s, BTree=%s", slSpread, btSpread)
	}
	if !slSpread.Equal(artSpread) {
		t.Errorf("Spread mismatch: SkipList=%s, ART=%s", slSpread, artSpread)
	}

	t.Logf("All implementations passed correctness test")
	t.Logf("Depth: Bids=%d, Asks=%d", slBids, slAsks)
	t.Logf("Best Bid: %v", slBestBid)
	t.Logf("Best Ask: %v", slBestAsk)
	t.Logf("Spread: %s", slSpread)
}

// TestOrderBookInterface verifies all implementations satisfy the interface
func TestOrderBookInterface(t *testing.T) {
	marketID := "BTC-USD"

	engines := []struct {
		name   string
		engine OrderBookEngine
	}{
		{"SkipList", NewOrderBookV2(marketID)},
		{"HashMap", NewOrderBookHashMap(marketID)},
		{"BTree", NewOrderBookBTree(marketID)},
		{"ART", NewOrderBookART(marketID)},
	}

	for _, e := range engines {
		t.Run(e.name, func(t *testing.T) {
			engine := e.engine

			// Test basic operations
			order := types.NewOrder("test-1", "trader", marketID, types.SideBuy, types.OrderTypeLimit,
				math.LegacyNewDec(50000), math.LegacyNewDec(1))

			engine.AddOrder(order)

			if engine.GetMarketID() != marketID {
				t.Errorf("GetMarketID() = %s, want %s", engine.GetMarketID(), marketID)
			}

			bidLevels, _ := engine.GetDepth()
			if bidLevels != 1 {
				t.Errorf("GetDepth() bid levels = %d, want 1", bidLevels)
			}

			bestBid := engine.GetBestBid()
			if bestBid == nil || !bestBid.Price.Equal(math.LegacyNewDec(50000)) {
				t.Errorf("GetBestBid() = %v, want price 50000", bestBid)
			}

			engine.RemoveOrder(order)
			bidLevels, _ = engine.GetDepth()
			if bidLevels != 0 {
				t.Errorf("After remove, GetDepth() bid levels = %d, want 0", bidLevels)
			}
		})
	}
}
