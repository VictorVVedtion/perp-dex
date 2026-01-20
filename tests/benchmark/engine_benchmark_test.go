package benchmark

import (
	"fmt"
	"math/rand"
	"sort"
	"testing"
	"time"

	"cosmossdk.io/log"
	"cosmossdk.io/math"
	"cosmossdk.io/store"
	"cosmossdk.io/store/metrics"
	storetypes "cosmossdk.io/store/types"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	obkeeper "github.com/openalpha/perp-dex/x/orderbook/keeper"
	obtypes "github.com/openalpha/perp-dex/x/orderbook/types"
)

// SimplePerpetualKeeper for benchmark tests
type SimplePerpetualKeeper struct {
	markets map[string]*obkeeper.Market
}

func NewSimplePerpetualKeeper() *SimplePerpetualKeeper {
	pk := &SimplePerpetualKeeper{
		markets: make(map[string]*obkeeper.Market),
	}
	takerFee, _ := math.LegacyNewDecFromStr("0.0006")
	makerFee, _ := math.LegacyNewDecFromStr("0.0001")
	initMargin, _ := math.LegacyNewDecFromStr("0.01")

	pk.markets["BTC-USDC"] = &obkeeper.Market{
		MarketID:      "BTC-USDC",
		TakerFeeRate:  takerFee,
		MakerFeeRate:  makerFee,
		InitialMargin: initMargin,
	}
	return pk
}

func (pk *SimplePerpetualKeeper) GetMarket(ctx sdk.Context, marketID string) *obkeeper.Market {
	return pk.markets[marketID]
}

func (pk *SimplePerpetualKeeper) GetMarkPrice(ctx sdk.Context, marketID string) (math.LegacyDec, bool) {
	return math.LegacyNewDec(50000), true
}

func (pk *SimplePerpetualKeeper) UpdatePosition(ctx sdk.Context, trader, marketID string, side obtypes.Side, qty, price, fee interface{}) error {
	return nil
}

func (pk *SimplePerpetualKeeper) CheckMarginRequirement(ctx sdk.Context, trader, marketID string, side obtypes.Side, qty, price interface{}) error {
	return nil
}

// setupBenchmarkKeeper creates a keeper for benchmarking
func setupBenchmarkKeeper(b *testing.B) (*obkeeper.Keeper, sdk.Context) {
	b.Helper()

	db := dbm.NewMemDB()
	storeKey := storetypes.NewKVStoreKey("orderbook")

	cms := store.NewCommitMultiStore(db, log.NewNopLogger(), metrics.NewNoOpMetrics())
	cms.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, db)
	if err := cms.LoadLatestVersion(); err != nil {
		b.Fatalf("failed to load store: %v", err)
	}

	interfaceRegistry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(interfaceRegistry)

	perpKeeper := NewSimplePerpetualKeeper()
	keeper := obkeeper.NewKeeper(cdc, storeKey, perpKeeper, log.NewNopLogger())

	header := tmproto.Header{Height: 1}
	sdkCtx := sdk.NewContext(cms, header, false, log.NewNopLogger())

	return keeper, sdkCtx
}

// createTestOrder creates a test order with random parameters
func createTestOrder(id int, side obtypes.Side, price, qty math.LegacyDec) *obtypes.Order {
	return obtypes.NewOrder(
		fmt.Sprintf("order-%d", id),
		fmt.Sprintf("trader-%d", id%100),
		"BTC-USDC",
		side,
		obtypes.OrderTypeLimit,
		price,
		qty,
	)
}

// ============ OrderBook V2 Benchmarks ============

// BenchmarkOrderBookV2_AddOrder tests order addition performance
func BenchmarkOrderBookV2_AddOrder(b *testing.B) {
	ob := obkeeper.NewOrderBookV2("BTC-USDC")
	basePrice := math.LegacyNewDec(50000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		side := obtypes.SideBuy
		if i%2 == 0 {
			side = obtypes.SideSell
		}

		// Create price variance
		priceOffset := math.LegacyNewDec(int64(i % 1000))
		price := basePrice.Add(priceOffset)
		qty := math.LegacyNewDec(1)

		order := createTestOrder(i, side, price, qty)
		ob.AddOrder(order)
	}
}

// BenchmarkOrderBookV2_RemoveOrder tests order removal performance
func BenchmarkOrderBookV2_RemoveOrder(b *testing.B) {
	ob := obkeeper.NewOrderBookV2("BTC-USDC")
	basePrice := math.LegacyNewDec(50000)

	// Pre-populate order book
	orders := make([]*obtypes.Order, b.N)
	for i := 0; i < b.N; i++ {
		side := obtypes.SideBuy
		if i%2 == 0 {
			side = obtypes.SideSell
		}
		priceOffset := math.LegacyNewDec(int64(i % 1000))
		price := basePrice.Add(priceOffset)
		qty := math.LegacyNewDec(1)

		orders[i] = createTestOrder(i, side, price, qty)
		ob.AddOrder(orders[i])
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ob.RemoveOrder(orders[i])
	}
}

// BenchmarkOrderBookV2_GetBestBid tests best bid retrieval
func BenchmarkOrderBookV2_GetBestBid(b *testing.B) {
	ob := obkeeper.NewOrderBookV2("BTC-USDC")
	basePrice := math.LegacyNewDec(50000)

	// Populate with many bid orders
	for i := 0; i < 10000; i++ {
		priceOffset := math.LegacyNewDec(int64(i % 1000))
		price := basePrice.Sub(priceOffset)
		qty := math.LegacyNewDec(1)
		order := createTestOrder(i, obtypes.SideBuy, price, qty)
		ob.AddOrder(order)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ob.GetBestBid()
	}
}

// ============ Matching Engine V2 Benchmarks ============

// BenchmarkMatchingEngineV2_ProcessOrder tests single order processing
func BenchmarkMatchingEngineV2_ProcessOrder(b *testing.B) {
	keeper, ctx := setupBenchmarkKeeper(b)
	engine := obkeeper.NewMatchingEngineV2(keeper)

	// Pre-populate order book with maker orders
	basePrice := math.LegacyNewDec(50000)
	for i := 0; i < 1000; i++ {
		// Add sell orders above base price
		sellPrice := basePrice.Add(math.LegacyNewDec(int64(i + 1)))
		sellOrder := createTestOrder(i, obtypes.SideSell, sellPrice, math.LegacyNewDec(10))
		engine.ProcessOrderOptimized(ctx, sellOrder)

		// Add buy orders below base price
		buyPrice := basePrice.Sub(math.LegacyNewDec(int64(i + 1)))
		buyOrder := createTestOrder(i+1000, obtypes.SideBuy, buyPrice, math.LegacyNewDec(10))
		engine.ProcessOrderOptimized(ctx, buyOrder)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Alternate between buy and sell market orders
		side := obtypes.SideBuy
		if i%2 == 0 {
			side = obtypes.SideSell
		}

		order := obtypes.NewOrder(
			fmt.Sprintf("taker-%d", i),
			fmt.Sprintf("trader-taker-%d", i),
			"BTC-USDC",
			side,
			obtypes.OrderTypeMarket,
			basePrice,
			math.LegacyNewDecWithPrec(1, 1), // 0.1 quantity
		)
		engine.ProcessOrderOptimized(ctx, order)
	}
}

// BenchmarkMatchingEngineV2_Match10K tests matching 10,000 orders
func BenchmarkMatchingEngineV2_Match10K(b *testing.B) {
	for n := 0; n < b.N; n++ {
		b.StopTimer()

		keeper, ctx := setupBenchmarkKeeper(b)
		engine := obkeeper.NewMatchingEngineV2(keeper)
		basePrice := math.LegacyNewDec(50000)

		// Create 10,000 orders (5,000 buy + 5,000 sell)
		orders := make([]*obtypes.Order, 10000)
		for i := 0; i < 5000; i++ {
			// Buy orders below base price
			buyPrice := basePrice.Sub(math.LegacyNewDec(int64(rand.Intn(100))))
			orders[i] = createTestOrder(i, obtypes.SideBuy, buyPrice, math.LegacyNewDec(1))

			// Sell orders above base price
			sellPrice := basePrice.Add(math.LegacyNewDec(int64(rand.Intn(100))))
			orders[i+5000] = createTestOrder(i+5000, obtypes.SideSell, sellPrice, math.LegacyNewDec(1))
		}

		b.StartTimer()

		// Process all orders
		for _, order := range orders {
			engine.ProcessOrderOptimized(ctx, order)
		}

		// Flush at the end
		engine.Flush(ctx)
	}
}

// BenchmarkMatchingEngineV2_BatchProcess tests batch order processing
func BenchmarkMatchingEngineV2_BatchProcess(b *testing.B) {
	keeper, ctx := setupBenchmarkKeeper(b)
	engine := obkeeper.NewMatchingEngineV2(keeper)
	basePrice := math.LegacyNewDec(50000)

	batchSize := 100

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		b.StopTimer()

		// Create batch
		orders := make([]*obtypes.Order, batchSize)
		for i := 0; i < batchSize; i++ {
			side := obtypes.SideBuy
			priceOffset := math.LegacyNewDec(int64(rand.Intn(100)))
			if i%2 == 0 {
				side = obtypes.SideSell
				priceOffset = priceOffset.Neg()
			}
			orders[i] = createTestOrder(n*batchSize+i, side, basePrice.Add(priceOffset), math.LegacyNewDec(1))
		}

		b.StartTimer()

		// Process batch
		engine.ProcessBatch(ctx, orders)
	}
}

// ============ Concurrent Benchmarks ============

// BenchmarkOrderBookV2_ConcurrentAdd tests concurrent order additions
func BenchmarkOrderBookV2_ConcurrentAdd(b *testing.B) {
	ob := obkeeper.NewOrderBookV2("BTC-USDC")
	basePrice := math.LegacyNewDec(50000)

	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			side := obtypes.SideBuy
			if i%2 == 0 {
				side = obtypes.SideSell
			}

			priceOffset := math.LegacyNewDec(int64(rand.Intn(1000)))
			price := basePrice.Add(priceOffset)
			qty := math.LegacyNewDec(1)

			order := createTestOrder(rand.Int(), side, price, qty)
			ob.AddOrder(order)
			i++
		}
	})
}

// ============ Memory Benchmarks ============

// BenchmarkOrderBookV2_MemoryUsage tests memory allocation
func BenchmarkOrderBookV2_MemoryUsage(b *testing.B) {
	b.ReportAllocs()

	ob := obkeeper.NewOrderBookV2("BTC-USDC")
	basePrice := math.LegacyNewDec(50000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		side := obtypes.SideBuy
		if i%2 == 0 {
			side = obtypes.SideSell
		}
		priceOffset := math.LegacyNewDec(int64(i % 1000))
		price := basePrice.Add(priceOffset)
		qty := math.LegacyNewDec(1)

		order := createTestOrder(i, side, price, qty)
		ob.AddOrder(order)
	}
}

// ============ Latency Benchmarks ============

// BenchmarkMatchingEngineV2_Latency measures detailed latency statistics
func BenchmarkMatchingEngineV2_Latency(b *testing.B) {
	keeper, ctx := setupBenchmarkKeeper(b)
	engine := obkeeper.NewMatchingEngineV2(keeper)
	basePrice := math.LegacyNewDec(50000)

	// Pre-populate
	for i := 0; i < 1000; i++ {
		sellPrice := basePrice.Add(math.LegacyNewDec(int64(i + 1)))
		sellOrder := createTestOrder(i, obtypes.SideSell, sellPrice, math.LegacyNewDec(10))
		engine.ProcessOrderOptimized(ctx, sellOrder)

		buyPrice := basePrice.Sub(math.LegacyNewDec(int64(i + 1)))
		buyOrder := createTestOrder(i+1000, obtypes.SideBuy, buyPrice, math.LegacyNewDec(10))
		engine.ProcessOrderOptimized(ctx, buyOrder)
	}

	latencies := make([]time.Duration, 0, b.N)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		side := obtypes.SideBuy
		if i%2 == 0 {
			side = obtypes.SideSell
		}

		order := obtypes.NewOrder(
			fmt.Sprintf("latency-test-%d", i),
			"latency-trader",
			"BTC-USDC",
			side,
			obtypes.OrderTypeMarket,
			basePrice,
			math.LegacyNewDecWithPrec(1, 2), // 0.01 quantity
		)

		start := time.Now()
		engine.ProcessOrderOptimized(ctx, order)
		latencies = append(latencies, time.Since(start))
	}

	// Report statistics
	if len(latencies) > 0 {
		var total time.Duration
		var min, max time.Duration = latencies[0], latencies[0]
		for _, l := range latencies {
			total += l
			if l < min {
				min = l
			}
			if l > max {
				max = l
			}
		}
		avg := total / time.Duration(len(latencies))

		b.ReportMetric(float64(avg.Nanoseconds()), "ns/op-avg")
		b.ReportMetric(float64(min.Nanoseconds()), "ns/op-min")
		b.ReportMetric(float64(max.Nanoseconds()), "ns/op-max")
	}
}

// ============ Scalability Benchmarks ============

// BenchmarkOrderBookV2_1K_Orders tests performance with 1K orders in the book
func BenchmarkOrderBookV2_1K_Orders(b *testing.B) {
	benchmarkOrderBookWithSize(b, 1000)
}

// BenchmarkOrderBookV2_10K_Orders tests performance with 10K orders in the book
func BenchmarkOrderBookV2_10K_Orders(b *testing.B) {
	benchmarkOrderBookWithSize(b, 10000)
}

// BenchmarkOrderBookV2_100K_Orders tests performance with 100K orders in the book
func BenchmarkOrderBookV2_100K_Orders(b *testing.B) {
	benchmarkOrderBookWithSize(b, 100000)
}

// benchmarkOrderBookWithSize populates an orderbook and benchmarks operations
func benchmarkOrderBookWithSize(b *testing.B, size int) {
	ob := obkeeper.NewOrderBookV2("BTC-USDC")
	basePrice := math.LegacyNewDec(50000)

	// Pre-populate order book
	for i := 0; i < size; i++ {
		side := obtypes.SideBuy
		priceOffset := math.LegacyNewDec(int64(i % 5000))
		if i%2 == 0 {
			side = obtypes.SideSell
			priceOffset = priceOffset.Neg()
		}
		price := basePrice.Add(priceOffset)
		qty := math.LegacyNewDec(1)
		order := createTestOrder(i, side, price, qty)
		ob.AddOrder(order)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		side := obtypes.SideBuy
		priceOffset := math.LegacyNewDec(int64(rand.Intn(1000)))
		if i%2 == 0 {
			side = obtypes.SideSell
		}
		price := basePrice.Add(priceOffset)
		qty := math.LegacyNewDec(1)

		order := createTestOrder(size+i, side, price, qty)
		ob.AddOrder(order)

		// Also test retrieval
		_ = ob.GetBestBid()
		_ = ob.GetBestAsk()
	}
}

// BenchmarkMatchingEngine_ConcurrentAccess tests concurrent matching performance
func BenchmarkMatchingEngine_ConcurrentAccess(b *testing.B) {
	keeper, ctx := setupBenchmarkKeeper(b)
	engine := obkeeper.NewMatchingEngineV2(keeper)
	basePrice := math.LegacyNewDec(50000)

	// Pre-populate with maker orders
	for i := 0; i < 2000; i++ {
		sellPrice := basePrice.Add(math.LegacyNewDec(int64(i + 1)))
		sellOrder := createTestOrder(i, obtypes.SideSell, sellPrice, math.LegacyNewDec(100))
		engine.ProcessOrderOptimized(ctx, sellOrder)

		buyPrice := basePrice.Sub(math.LegacyNewDec(int64(i + 1)))
		buyOrder := createTestOrder(i+2000, obtypes.SideBuy, buyPrice, math.LegacyNewDec(100))
		engine.ProcessOrderOptimized(ctx, buyOrder)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			side := obtypes.SideBuy
			if i%2 == 0 {
				side = obtypes.SideSell
			}

			order := obtypes.NewOrder(
				fmt.Sprintf("concurrent-%d-%d", rand.Int(), i),
				fmt.Sprintf("trader-concurrent-%d", rand.Intn(100)),
				"BTC-USDC",
				side,
				obtypes.OrderTypeMarket,
				basePrice,
				math.LegacyNewDecWithPrec(1, 2), // 0.01 quantity
			)
			engine.ProcessOrderOptimized(ctx, order)
			i++
		}
	})
}

// BenchmarkOrderBookV2_DeepBook tests performance with deep order book (many price levels)
func BenchmarkOrderBookV2_DeepBook(b *testing.B) {
	ob := obkeeper.NewOrderBookV2("BTC-USDC")
	basePrice := math.LegacyNewDec(50000)

	// Create deep book with 10,000 unique price levels
	for i := 0; i < 10000; i++ {
		buyPrice := basePrice.Sub(math.LegacyNewDecWithPrec(int64(i), 1)) // 0.1 price increments
		buyOrder := createTestOrder(i, obtypes.SideBuy, buyPrice, math.LegacyNewDec(1))
		ob.AddOrder(buyOrder)

		sellPrice := basePrice.Add(math.LegacyNewDecWithPrec(int64(i+1), 1))
		sellOrder := createTestOrder(i+10000, obtypes.SideSell, sellPrice, math.LegacyNewDec(1))
		ob.AddOrder(sellOrder)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// Test operations on deep book
		_ = ob.GetBestBid()
		_ = ob.GetBestAsk()

		// Add and remove orders
		order := createTestOrder(20000+i, obtypes.SideBuy, basePrice.Sub(math.LegacyNewDec(500)), math.LegacyNewDec(1))
		ob.AddOrder(order)
		ob.RemoveOrder(order)
	}
}

// BenchmarkMatchingEngine_HighFrequency tests high-frequency trading pattern
func BenchmarkMatchingEngine_HighFrequency(b *testing.B) {
	keeper, ctx := setupBenchmarkKeeper(b)
	engine := obkeeper.NewMatchingEngineV2(keeper)
	basePrice := math.LegacyNewDec(50000)

	// Pre-populate thin book (simulating HFT environment)
	for i := 0; i < 100; i++ {
		sellPrice := basePrice.Add(math.LegacyNewDecWithPrec(int64(i+1), 1))
		sellOrder := createTestOrder(i, obtypes.SideSell, sellPrice, math.LegacyNewDec(10))
		engine.ProcessOrderOptimized(ctx, sellOrder)

		buyPrice := basePrice.Sub(math.LegacyNewDecWithPrec(int64(i+1), 1))
		buyOrder := createTestOrder(i+100, obtypes.SideBuy, buyPrice, math.LegacyNewDec(10))
		engine.ProcessOrderOptimized(ctx, buyOrder)
	}

	b.ResetTimer()

	// Simulate HFT: rapid place and cancel
	for i := 0; i < b.N; i++ {
		// Place order
		order := obtypes.NewOrder(
			fmt.Sprintf("hft-%d", i),
			"hft-trader",
			"BTC-USDC",
			obtypes.SideBuy,
			obtypes.OrderTypeLimit,
			basePrice.Sub(math.LegacyNewDecWithPrec(int64(rand.Intn(10)+1), 1)),
			math.LegacyNewDec(1),
		)
		engine.ProcessOrderOptimized(ctx, order)

		// Cancel immediately (simulating HFT behavior)
		engine.CancelOrderOptimized(ctx, order.OrderID)
	}
}

// BenchmarkMatchingEngine_BatchVsSingle compares batch vs single order processing
func BenchmarkMatchingEngine_BatchVsSingle(b *testing.B) {
	b.Run("SingleOrder", func(b *testing.B) {
		keeper, ctx := setupBenchmarkKeeper(b)
		engine := obkeeper.NewMatchingEngineV2(keeper)
		basePrice := math.LegacyNewDec(50000)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			side := obtypes.SideBuy
			if i%2 == 0 {
				side = obtypes.SideSell
			}
			order := createTestOrder(i, side, basePrice, math.LegacyNewDec(1))
			engine.ProcessOrderOptimized(ctx, order)
		}
	})

	b.Run("Batch100", func(b *testing.B) {
		keeper, ctx := setupBenchmarkKeeper(b)
		engine := obkeeper.NewMatchingEngineV2(keeper)
		basePrice := math.LegacyNewDec(50000)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			orders := make([]*obtypes.Order, 100)
			for j := 0; j < 100; j++ {
				side := obtypes.SideBuy
				if j%2 == 0 {
					side = obtypes.SideSell
				}
				orders[j] = createTestOrder(i*100+j, side, basePrice.Add(math.LegacyNewDec(int64(j-50))), math.LegacyNewDec(1))
			}
			engine.ProcessBatch(ctx, orders)
		}
	})
}

// ============ Comprehensive Latency Analysis ============

// BenchmarkLatencyDistribution measures latency distribution across different scenarios
func BenchmarkLatencyDistribution(b *testing.B) {
	scenarios := []struct {
		name      string
		bookSize  int
		orderQty  int64
		orderType obtypes.OrderType
	}{
		{"EmptyBook_Market", 0, 1, obtypes.OrderTypeMarket},
		{"SmallBook_Market", 100, 1, obtypes.OrderTypeMarket},
		{"MediumBook_Market", 1000, 1, obtypes.OrderTypeMarket},
		{"LargeBook_Market", 10000, 1, obtypes.OrderTypeMarket},
		{"SmallBook_Limit", 100, 1, obtypes.OrderTypeLimit},
		{"LargeBook_Limit", 10000, 1, obtypes.OrderTypeLimit},
	}

	for _, sc := range scenarios {
		b.Run(sc.name, func(b *testing.B) {
			keeper, ctx := setupBenchmarkKeeper(b)
			engine := obkeeper.NewMatchingEngineV2(keeper)
			basePrice := math.LegacyNewDec(50000)

			// Populate book
			for i := 0; i < sc.bookSize; i++ {
				sellPrice := basePrice.Add(math.LegacyNewDec(int64(i%1000 + 1)))
				sellOrder := createTestOrder(i, obtypes.SideSell, sellPrice, math.LegacyNewDec(10))
				engine.ProcessOrderOptimized(ctx, sellOrder)

				buyPrice := basePrice.Sub(math.LegacyNewDec(int64(i%1000 + 1)))
				buyOrder := createTestOrder(i+sc.bookSize, obtypes.SideBuy, buyPrice, math.LegacyNewDec(10))
				engine.ProcessOrderOptimized(ctx, buyOrder)
			}

			latencies := make([]time.Duration, 0, b.N)

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				side := obtypes.SideBuy
				if i%2 == 0 {
					side = obtypes.SideSell
				}

				order := obtypes.NewOrder(
					fmt.Sprintf("latency-%s-%d", sc.name, i),
					"latency-trader",
					"BTC-USDC",
					side,
					sc.orderType,
					basePrice,
					math.LegacyNewDec(sc.orderQty),
				)

				start := time.Now()
				engine.ProcessOrderOptimized(ctx, order)
				latencies = append(latencies, time.Since(start))
			}

			// Report percentiles
			if len(latencies) > 0 {
				sort.Slice(latencies, func(i, j int) bool {
					return latencies[i] < latencies[j]
				})

				var total time.Duration
				for _, l := range latencies {
					total += l
				}

				b.ReportMetric(float64(total.Nanoseconds())/float64(len(latencies)), "ns/op-avg")
				b.ReportMetric(float64(latencies[len(latencies)*50/100].Nanoseconds()), "ns/op-p50")
				b.ReportMetric(float64(latencies[len(latencies)*99/100].Nanoseconds()), "ns/op-p99")
			}
		})
	}
}

// ============ Stress Test ============

// TestStress10K runs a stress test with 10,000 orders and reports metrics
func TestStress10K(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping stress test in short mode")
	}

	db := dbm.NewMemDB()
	storeKey := storetypes.NewKVStoreKey("orderbook")

	cms := store.NewCommitMultiStore(db, log.NewNopLogger(), metrics.NewNoOpMetrics())
	cms.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, db)
	if err := cms.LoadLatestVersion(); err != nil {
		t.Fatalf("failed to load store: %v", err)
	}

	interfaceRegistry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(interfaceRegistry)

	perpKeeper := NewSimplePerpetualKeeper()
	keeper := obkeeper.NewKeeper(cdc, storeKey, perpKeeper, log.NewNopLogger())
	header := tmproto.Header{Height: 1}
	ctx := sdk.NewContext(cms, header, false, log.NewNopLogger())

	engine := obkeeper.NewMatchingEngineV2(keeper)
	basePrice := math.LegacyNewDec(50000)

	const orderCount = 10000

	// Create orders
	orders := make([]*obtypes.Order, orderCount)
	for i := 0; i < orderCount; i++ {
		side := obtypes.SideBuy
		priceOffset := math.LegacyNewDec(int64(rand.Intn(200) - 100))
		if i%2 == 0 {
			side = obtypes.SideSell
		}
		orders[i] = createTestOrder(i, side, basePrice.Add(priceOffset), math.LegacyNewDecWithPrec(1, 1))
	}

	// Measure processing time
	start := time.Now()
	tradeCount := 0

	for _, order := range orders {
		result, err := engine.ProcessOrderOptimized(ctx, order)
		if err != nil {
			t.Errorf("failed to process order: %v", err)
		}
		if result != nil {
			tradeCount += len(result.Trades)
		}
	}

	// Flush
	engine.Flush(ctx)
	elapsed := time.Since(start)

	// Report results
	t.Logf("=== 10K Order Stress Test Results ===")
	t.Logf("Orders processed: %d", orderCount)
	t.Logf("Trades executed:  %d", tradeCount)
	t.Logf("Total time:       %v", elapsed)
	t.Logf("Orders/second:    %.2f", float64(orderCount)/elapsed.Seconds())
	t.Logf("Avg latency:      %v", elapsed/time.Duration(orderCount))

	// Verify performance target
	if elapsed > 100*time.Millisecond {
		t.Logf("WARNING: Performance below target (>100ms for 10K orders)")
	} else {
		t.Logf("PASS: Performance within target (<100ms for 10K orders)")
	}
}
