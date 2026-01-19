package api

import (
	"fmt"
	"math"
	"math/rand"
	"time"

	"github.com/openalpha/perp-dex/api/websocket"
)

// Mock data generation for development and testing

// getMockMarkets returns mock market list
func (s *Server) getMockMarkets() []map[string]interface{} {
	return []map[string]interface{}{
		{
			"market_id":                "BTC-USDC",
			"base_asset":               "BTC",
			"quote_asset":              "USDC",
			"status":                   "active",
			"max_leverage":             50,
			"initial_margin_rate":      "0.05",
			"maintenance_margin_rate":  "0.025",
			"taker_fee_rate":           "0.0005",
			"maker_fee_rate":           "0.0002",
			"min_order_size":           "0.001",
			"tick_size":                "0.1",
		},
		{
			"market_id":                "ETH-USDC",
			"base_asset":               "ETH",
			"quote_asset":              "USDC",
			"status":                   "active",
			"max_leverage":             50,
			"initial_margin_rate":      "0.05",
			"maintenance_margin_rate":  "0.025",
			"taker_fee_rate":           "0.0005",
			"maker_fee_rate":           "0.0002",
			"min_order_size":           "0.01",
			"tick_size":                "0.01",
		},
		{
			"market_id":                "SOL-USDC",
			"base_asset":               "SOL",
			"quote_asset":              "USDC",
			"status":                   "active",
			"max_leverage":             25,
			"initial_margin_rate":      "0.05",
			"maintenance_margin_rate":  "0.025",
			"taker_fee_rate":           "0.0005",
			"maker_fee_rate":           "0.0002",
			"min_order_size":           "0.1",
			"tick_size":                "0.001",
		},
	}
}

// getMockMarket returns a single mock market
func (s *Server) getMockMarket(marketID string) map[string]interface{} {
	markets := s.getMockMarkets()
	for _, m := range markets {
		if m["market_id"] == marketID {
			return m
		}
	}
	return nil
}

// Base prices for mock data
var basePrices = map[string]float64{
	"BTC-USDC": 97500.0,
	"ETH-USDC": 3350.0,
	"SOL-USDC": 185.0,
}

// getMockTicker returns mock ticker data with realistic fluctuations
func (s *Server) getMockTicker(marketID string) map[string]interface{} {
	basePrice, ok := basePrices[marketID]
	if !ok {
		basePrice = 100.0
	}

	// Add some randomness
	fluctuation := (rand.Float64() - 0.5) * 0.002 * basePrice
	price := basePrice + fluctuation

	change24h := (rand.Float64() - 0.5) * 5 // -2.5% to +2.5%
	volume24h := basePrice * (1000 + rand.Float64()*2000)

	return map[string]interface{}{
		"market_id":    marketID,
		"mark_price":   formatPrice(price),
		"index_price":  formatPrice(price * (1 + (rand.Float64()-0.5)*0.0002)),
		"last_price":   formatPrice(price),
		"high_24h":     formatPrice(price * 1.025),
		"low_24h":      formatPrice(price * 0.975),
		"volume_24h":   formatPrice(volume24h),
		"change_24h":   formatPercent(change24h),
		"funding_rate": formatPercent((rand.Float64() - 0.5) * 0.02),
		"next_funding": time.Now().Add(time.Hour).Unix(),
		"open_interest": formatPrice(volume24h * 3),
		"timestamp":    time.Now().UnixMilli(),
	}
}

// getMockOrderbook returns mock orderbook
func (s *Server) getMockOrderbook(marketID string, depth int) map[string]interface{} {
	basePrice, ok := basePrices[marketID]
	if !ok {
		basePrice = 100.0
	}

	bids := make([][]string, depth)
	asks := make([][]string, depth)

	spread := basePrice * 0.0001 // 0.01% spread

	for i := 0; i < depth; i++ {
		// Bids: decreasing prices
		bidPrice := basePrice - spread - float64(i)*basePrice*0.0001
		bidQty := 0.1 + rand.Float64()*2

		// Asks: increasing prices
		askPrice := basePrice + spread + float64(i)*basePrice*0.0001
		askQty := 0.1 + rand.Float64()*2

		bids[i] = []string{formatPrice(bidPrice), formatQty(bidQty)}
		asks[i] = []string{formatPrice(askPrice), formatQty(askQty)}
	}

	return map[string]interface{}{
		"market_id": marketID,
		"bids":      bids,
		"asks":      asks,
		"timestamp": time.Now().UnixMilli(),
	}
}

// getMockTrades returns mock recent trades
func (s *Server) getMockTrades(marketID string, limit int) []map[string]interface{} {
	basePrice, ok := basePrices[marketID]
	if !ok {
		basePrice = 100.0
	}

	trades := make([]map[string]interface{}, limit)
	now := time.Now()

	for i := 0; i < limit; i++ {
		price := basePrice * (1 + (rand.Float64()-0.5)*0.001)
		side := "buy"
		if rand.Float64() > 0.5 {
			side = "sell"
		}

		trades[i] = map[string]interface{}{
			"trade_id":  fmt.Sprintf("T%d", 1000000+i),
			"market_id": marketID,
			"price":     formatPrice(price),
			"quantity":  formatQty(0.01 + rand.Float64()*0.5),
			"side":      side,
			"timestamp": now.Add(-time.Duration(i) * time.Second).UnixMilli(),
		}
	}

	return trades
}

// getMockKlines returns mock K-line data
func (s *Server) getMockKlines(marketID string, interval string, limit int) []map[string]interface{} {
	basePrice, ok := basePrices[marketID]
	if !ok {
		basePrice = 100.0
	}

	// Parse interval
	var duration time.Duration
	switch interval {
	case "1m":
		duration = time.Minute
	case "5m":
		duration = 5 * time.Minute
	case "15m":
		duration = 15 * time.Minute
	case "30m":
		duration = 30 * time.Minute
	case "1h":
		duration = time.Hour
	case "4h":
		duration = 4 * time.Hour
	case "1d":
		duration = 24 * time.Hour
	default:
		duration = time.Hour
	}

	klines := make([]map[string]interface{}, limit)
	now := time.Now().Truncate(duration)

	// Generate price path with random walk
	price := basePrice
	for i := limit - 1; i >= 0; i-- {
		timestamp := now.Add(-time.Duration(i) * duration)

		// Random walk
		change := (rand.Float64() - 0.5) * 0.01 * price
		open := price
		close := price + change

		high := math.Max(open, close) * (1 + rand.Float64()*0.005)
		low := math.Min(open, close) * (1 - rand.Float64()*0.005)
		volume := basePrice * (10 + rand.Float64()*50)

		klines[limit-1-i] = map[string]interface{}{
			"time":   timestamp.Unix(),
			"open":   open,
			"high":   high,
			"low":    low,
			"close":  close,
			"volume": volume,
		}

		price = close
	}

	return klines
}

// getMockFunding returns mock funding rate
func (s *Server) getMockFunding(marketID string) map[string]interface{} {
	rate := (rand.Float64() - 0.5) * 0.0002 // -0.01% to +0.01%

	return map[string]interface{}{
		"market_id":     marketID,
		"funding_rate":  formatPercent(rate * 100),
		"funding_time":  time.Now().Truncate(time.Hour).Add(time.Hour).Unix(),
		"interval":      "1h",
		"estimated_rate": formatPercent(rate * 100 * 0.9),
	}
}

// getMockAccount returns mock account data
func (s *Server) getMockAccount(address string) map[string]interface{} {
	return map[string]interface{}{
		"address":           address,
		"total_equity":      "12500.00",
		"available_balance": "8500.00",
		"margin_used":       "4000.00",
		"unrealized_pnl":    "250.00",
		"margin_ratio":      "0.32",
	}
}

// getMockPositions returns mock positions
func (s *Server) getMockPositions(address string) []map[string]interface{} {
	return []map[string]interface{}{
		{
			"market_id":        "BTC-USDC",
			"side":             "long",
			"size":             "0.1",
			"entry_price":      "97200.00",
			"mark_price":       "97500.00",
			"margin":           "1944.00",
			"leverage":         "5",
			"unrealized_pnl":   "30.00",
			"liquidation_price": "88560.00",
			"margin_mode":      "isolated",
		},
	}
}

// getMockOrders returns mock open orders
func (s *Server) getMockOrders(address string) []map[string]interface{} {
	return []map[string]interface{}{
		{
			"order_id":   "O123456",
			"market_id":  "BTC-USDC",
			"side":       "buy",
			"type":       "limit",
			"price":      "96000.00",
			"size":       "0.05",
			"filled":     "0.00",
			"status":     "open",
			"created_at": time.Now().Add(-time.Hour).UnixMilli(),
		},
	}
}

// getMockAccountTrades returns mock trade history
func (s *Server) getMockAccountTrades(address string) []map[string]interface{} {
	return []map[string]interface{}{
		{
			"trade_id":   "T789",
			"order_id":   "O789",
			"market_id":  "BTC-USDC",
			"side":       "buy",
			"price":      "97200.00",
			"quantity":   "0.1",
			"fee":        "4.86",
			"realized_pnl": "0.00",
			"timestamp":  time.Now().Add(-2 * time.Hour).UnixMilli(),
		},
	}
}

// startMockDataBroadcaster starts broadcasting mock data via WebSocket
func (s *Server) startMockDataBroadcaster() {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	marketIDs := []string{"BTC-USDC", "ETH-USDC", "SOL-USDC"}

	for range ticker.C {
		for _, marketID := range marketIDs {
			// Broadcast ticker
			tickerData := s.getMockTicker(marketID)
			s.wsServer.BroadcastTicker(&websocket.TickerMessage{
				MarketID:    marketID,
				MarkPrice:   tickerData["mark_price"].(string),
				IndexPrice:  tickerData["index_price"].(string),
				LastPrice:   tickerData["last_price"].(string),
				High24h:     tickerData["high_24h"].(string),
				Low24h:      tickerData["low_24h"].(string),
				Volume24h:   tickerData["volume_24h"].(string),
				Change24h:   tickerData["change_24h"].(string),
				FundingRate: tickerData["funding_rate"].(string),
				NextFunding: tickerData["next_funding"].(int64),
				Timestamp:   tickerData["timestamp"].(int64),
			})

			// Broadcast depth every 2 seconds (less frequent)
			if time.Now().Second()%2 == 0 {
				orderbookData := s.getMockOrderbook(marketID, 20)
				bids := orderbookData["bids"].([][]string)
				asks := orderbookData["asks"].([][]string)

				depthBids := make([]websocket.PriceLevel, len(bids))
				depthAsks := make([]websocket.PriceLevel, len(asks))

				for i, b := range bids {
					depthBids[i] = websocket.PriceLevel{Price: b[0], Quantity: b[1]}
				}
				for i, a := range asks {
					depthAsks[i] = websocket.PriceLevel{Price: a[0], Quantity: a[1]}
				}

				s.wsServer.BroadcastDepth(&websocket.DepthMessage{
					MarketID:  marketID,
					Bids:      depthBids,
					Asks:      depthAsks,
					Timestamp: orderbookData["timestamp"].(int64),
				})
			}

			// Broadcast random trade occasionally
			if rand.Float64() > 0.7 {
				trades := s.getMockTrades(marketID, 1)
				if len(trades) > 0 {
					t := trades[0]
					s.wsServer.BroadcastTrade(&websocket.TradeMessage{
						TradeID:   t["trade_id"].(string),
						MarketID:  marketID,
						Price:     t["price"].(string),
						Quantity:  t["quantity"].(string),
						Side:      t["side"].(string),
						Timestamp: t["timestamp"].(int64),
					})
				}
			}
		}
	}
}

// Helper formatting functions
func formatPrice(price float64) string {
	return fmt.Sprintf("%.2f", price)
}

func formatQty(qty float64) string {
	return fmt.Sprintf("%.4f", qty)
}

func formatPercent(pct float64) string {
	return fmt.Sprintf("%.4f", pct)
}

// fmt is imported at package level
