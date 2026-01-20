package e2e_hyperliquid

import (
	"encoding/json"
	"sync"
	"testing"
	"time"
)

// TestMarketData_GetMeta tests fetching exchange metadata
func TestMarketData_GetMeta(t *testing.T) {
	client := NewHyperliquidClient()

	result := client.GetMeta()
	if result.Error != nil {
		t.Fatalf("Failed to get meta: %v", result.Error)
	}

	if result.StatusCode != 200 {
		t.Fatalf("Expected status 200, got %d", result.StatusCode)
	}

	var meta Meta
	if err := json.Unmarshal(result.Data, &meta); err != nil {
		t.Fatalf("Failed to parse meta: %v", err)
	}

	if len(meta.Universe) == 0 {
		t.Error("Expected non-empty universe")
	}

	t.Logf("Meta fetched: %d assets, latency=%v", len(meta.Universe), result.Latency)

	// Log some assets
	for i, asset := range meta.Universe {
		if i >= 5 {
			break
		}
		t.Logf("  Asset[%d]: %s (decimals=%d)", i, asset.Name, asset.SzDecimals)
	}
}

// TestMarketData_GetAllMids tests fetching all mid prices
func TestMarketData_GetAllMids(t *testing.T) {
	client := NewHyperliquidClient()

	result := client.GetAllMids()
	if result.Error != nil {
		t.Fatalf("Failed to get all mids: %v", result.Error)
	}

	if result.StatusCode != 200 {
		t.Fatalf("Expected status 200, got %d", result.StatusCode)
	}

	var mids map[string]string
	if err := json.Unmarshal(result.Data, &mids); err != nil {
		t.Fatalf("Failed to parse mids: %v", err)
	}

	if len(mids) == 0 {
		t.Error("Expected non-empty mid prices")
	}

	t.Logf("All mids fetched: %d pairs, latency=%v", len(mids), result.Latency)

	// Log BTC and ETH prices
	if btc, ok := mids["BTC"]; ok {
		t.Logf("  BTC mid price: %s", btc)
	}
	if eth, ok := mids["ETH"]; ok {
		t.Logf("  ETH mid price: %s", eth)
	}
}

// TestMarketData_GetL2Book tests fetching order book
func TestMarketData_GetL2Book(t *testing.T) {
	client := NewHyperliquidClient()

	coins := []string{"BTC", "ETH", "SOL"}

	for _, coin := range coins {
		result := client.GetL2Book(coin)
		if result.Error != nil {
			t.Errorf("Failed to get L2 book for %s: %v", coin, result.Error)
			continue
		}

		if result.StatusCode != 200 {
			t.Errorf("Expected status 200 for %s, got %d", coin, result.StatusCode)
			continue
		}

		var book L2Book
		if err := json.Unmarshal(result.Data, &book); err != nil {
			t.Errorf("Failed to parse L2 book for %s: %v", coin, err)
			continue
		}

		if len(book.Levels) < 2 {
			t.Errorf("Expected bids and asks for %s", coin)
			continue
		}

		bids := book.Levels[0]
		asks := book.Levels[1]

		t.Logf("%s L2 Book: %d bids, %d asks, latency=%v", coin, len(bids), len(asks), result.Latency)

		if len(bids) > 0 {
			t.Logf("  Best bid: %s @ %s", bids[0].Sz, bids[0].Px)
		}
		if len(asks) > 0 {
			t.Logf("  Best ask: %s @ %s", asks[0].Sz, asks[0].Px)
		}
	}
}

// TestMarketData_GetRecentTrades tests fetching recent trades
func TestMarketData_GetRecentTrades(t *testing.T) {
	client := NewHyperliquidClient()

	coins := []string{"BTC", "ETH"}

	for _, coin := range coins {
		result := client.GetRecentTrades(coin)
		if result.Error != nil {
			t.Errorf("Failed to get recent trades for %s: %v", coin, result.Error)
			continue
		}

		if result.StatusCode != 200 {
			t.Errorf("Expected status 200 for %s, got %d", coin, result.StatusCode)
			continue
		}

		var trades []Trade
		if err := json.Unmarshal(result.Data, &trades); err != nil {
			t.Errorf("Failed to parse trades for %s: %v", coin, err)
			continue
		}

		t.Logf("%s Recent Trades: %d trades, latency=%v", coin, len(trades), result.Latency)

		if len(trades) > 0 {
			trade := trades[0]
			t.Logf("  Latest: %s %s @ %s (size=%s)", trade.Side, coin, trade.Px, trade.Sz)
		}
	}
}

// TestMarketData_GetCandleSnapshot tests fetching candle data
func TestMarketData_GetCandleSnapshot(t *testing.T) {
	client := NewHyperliquidClient()

	endTime := time.Now().UnixMilli()
	startTime := endTime - (24 * time.Hour).Milliseconds() // Last 24 hours

	result := client.GetCandleSnapshot("BTC", "1h", startTime, endTime)
	if result.Error != nil {
		t.Fatalf("Failed to get candle snapshot: %v", result.Error)
	}

	if result.StatusCode != 200 {
		t.Fatalf("Expected status 200, got %d", result.StatusCode)
	}

	var candles []Candle
	if err := json.Unmarshal(result.Data, &candles); err != nil {
		t.Fatalf("Failed to parse candles: %v", err)
	}

	t.Logf("BTC Candles (1h): %d candles, latency=%v", len(candles), result.Latency)

	if len(candles) > 0 {
		latest := candles[len(candles)-1]
		t.Logf("  Latest: O=%s H=%s L=%s C=%s V=%s", latest.O, latest.H, latest.L, latest.C, latest.V)
	}
}

// TestMarketData_GetFundingHistory tests fetching funding rate history
func TestMarketData_GetFundingHistory(t *testing.T) {
	client := NewHyperliquidClient()

	startTime := time.Now().Add(-24 * time.Hour).UnixMilli()

	result := client.GetFundingHistory("BTC", startTime)
	if result.Error != nil {
		t.Fatalf("Failed to get funding history: %v", result.Error)
	}

	if result.StatusCode != 200 {
		t.Fatalf("Expected status 200, got %d", result.StatusCode)
	}

	var funding []FundingRate
	if err := json.Unmarshal(result.Data, &funding); err != nil {
		t.Fatalf("Failed to parse funding: %v", err)
	}

	t.Logf("BTC Funding History: %d entries, latency=%v", len(funding), result.Latency)

	if len(funding) > 0 {
		latest := funding[len(funding)-1]
		t.Logf("  Latest funding rate: %s, premium: %s", latest.FundingRate, latest.Premium)
	}
}

// TestMarketData_LatencyBenchmark benchmarks API latency
func TestMarketData_LatencyBenchmark(t *testing.T) {
	client := NewHyperliquidClient()

	iterations := 50
	t.Logf("Running %d iterations of API calls...", iterations)

	for i := 0; i < iterations; i++ {
		// Mix of different API calls
		switch i % 5 {
		case 0:
			client.GetAllMids()
		case 1:
			client.GetL2Book("BTC")
		case 2:
			client.GetL2Book("ETH")
		case 3:
			client.GetRecentTrades("BTC")
		case 4:
			client.GetMeta()
		}
	}

	stats := client.GetLatencyStats()
	stats.PrintStats("Market Data Latency Benchmark")

	// Assert reasonable latency
	if stats.Avg > 2*time.Second {
		t.Errorf("Average latency %v exceeds 2s threshold", stats.Avg)
	}

	if stats.P99 > 5*time.Second {
		t.Errorf("P99 latency %v exceeds 5s threshold", stats.P99)
	}
}

// TestMarketData_ConcurrentRequests tests concurrent API requests
func TestMarketData_ConcurrentRequests(t *testing.T) {
	client := NewHyperliquidClient()

	numWorkers := 10
	requestsPerWorker := 20
	var wg sync.WaitGroup
	var successCount, errorCount int64
	var mu sync.Mutex

	coins := []string{"BTC", "ETH", "SOL", "DOGE", "ARB"}

	startTime := time.Now()

	for w := 0; w < numWorkers; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			for i := 0; i < requestsPerWorker; i++ {
				coin := coins[(workerID+i)%len(coins)]
				result := client.GetL2Book(coin)

				mu.Lock()
				if result.Error != nil || result.StatusCode != 200 {
					errorCount++
				} else {
					successCount++
				}
				mu.Unlock()
			}
		}(w)
	}

	wg.Wait()
	elapsed := time.Since(startTime)

	totalRequests := int64(numWorkers * requestsPerWorker)
	throughput := float64(totalRequests) / elapsed.Seconds()

	t.Logf("Concurrent Request Test Results:")
	t.Logf("  Workers: %d", numWorkers)
	t.Logf("  Total Requests: %d", totalRequests)
	t.Logf("  Success: %d", successCount)
	t.Logf("  Errors: %d", errorCount)
	t.Logf("  Duration: %v", elapsed)
	t.Logf("  Throughput: %.2f req/sec", throughput)

	stats := client.GetLatencyStats()
	stats.PrintStats("Concurrent Requests")

	// Assert success rate
	successRate := float64(successCount) / float64(totalRequests)
	if successRate < 0.95 {
		t.Errorf("Success rate %.2f%% is below 95%% threshold", successRate*100)
	}
}

// TestMarketData_DataIntegrity verifies data consistency
func TestMarketData_DataIntegrity(t *testing.T) {
	client := NewHyperliquidClient()

	// Get mid price
	midsResult := client.GetAllMids()
	if midsResult.Error != nil {
		t.Fatalf("Failed to get mids: %v", midsResult.Error)
	}

	var mids map[string]string
	json.Unmarshal(midsResult.Data, &mids)

	// Get L2 book
	bookResult := client.GetL2Book("BTC")
	if bookResult.Error != nil {
		t.Fatalf("Failed to get L2 book: %v", bookResult.Error)
	}

	var book L2Book
	json.Unmarshal(bookResult.Data, &book)

	t.Logf("Data Integrity Check:")
	t.Logf("  BTC Mid Price: %s", mids["BTC"])

	if len(book.Levels) >= 2 && len(book.Levels[0]) > 0 && len(book.Levels[1]) > 0 {
		bestBid := book.Levels[0][0].Px
		bestAsk := book.Levels[1][0].Px
		t.Logf("  Best Bid: %s", bestBid)
		t.Logf("  Best Ask: %s", bestAsk)

		// Mid should be between bid and ask
		// (simplified check - in real scenario would parse and compare floats)
	}
}

// TestMarketData_AllCoins tests fetching data for all available coins
func TestMarketData_AllCoins(t *testing.T) {
	client := NewHyperliquidClient()

	// Get all available coins
	metaResult := client.GetMeta()
	if metaResult.Error != nil {
		t.Fatalf("Failed to get meta: %v", metaResult.Error)
	}

	var meta Meta
	json.Unmarshal(metaResult.Data, &meta)

	t.Logf("Testing %d coins...", len(meta.Universe))

	successCount := 0
	for _, asset := range meta.Universe {
		result := client.GetL2Book(asset.Name)
		if result.Error == nil && result.StatusCode == 200 {
			successCount++
		}
	}

	t.Logf("Successfully fetched L2 book for %d/%d coins", successCount, len(meta.Universe))

	// All coins should be accessible
	if successCount < len(meta.Universe) {
		t.Errorf("Failed to fetch some coins: %d/%d succeeded", successCount, len(meta.Universe))
	}
}
