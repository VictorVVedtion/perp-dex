package api

import (
	"fmt"
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

// NOTE: basePrices removed - now using HyperliquidOracle for real-time prices

// getMockTicker returns ticker data from Hyperliquid real-time prices
// Falls back to placeholder values if Oracle is unavailable
func (s *Server) getMockTicker(marketID string) map[string]interface{} {
	// Try to get real data from Oracle
	if s.oracle != nil {
		ticker, err := s.oracle.GetTicker(marketID)
		if err == nil {
			return map[string]interface{}{
				"market_id":     ticker.MarketID,
				"mark_price":    ticker.MarkPrice,
				"index_price":   ticker.IndexPrice,
				"last_price":    ticker.LastPrice,
				"high_24h":      ticker.High24h,
				"low_24h":       ticker.Low24h,
				"volume_24h":    ticker.Volume24h,
				"change_24h":    ticker.Change24h,
				"funding_rate":  ticker.FundingRate,
				"next_funding":  ticker.NextFunding,
				"open_interest": "0", // Not available from basic Hyperliquid API
				"timestamp":     ticker.Timestamp,
			}
		}
		// Log error but continue with fallback
		fmt.Printf("Oracle GetTicker error for %s: %v\n", marketID, err)
	}

	// Fallback: return error indicator
	return map[string]interface{}{
		"market_id":     marketID,
		"mark_price":    "0",
		"index_price":   "0",
		"last_price":    "0",
		"high_24h":      "0",
		"low_24h":       "0",
		"volume_24h":    "0",
		"change_24h":    "0",
		"funding_rate":  "0",
		"next_funding":  time.Now().Add(time.Hour).Unix(),
		"open_interest": "0",
		"timestamp":     time.Now().UnixMilli(),
		"error":         "price_unavailable",
	}
}

// getMockOrderbook returns orderbook data from Hyperliquid real-time L2 book
// Falls back to empty orderbook if Oracle is unavailable
func (s *Server) getMockOrderbook(marketID string, depth int) map[string]interface{} {
	// Try to get real data from Oracle
	if s.oracle != nil {
		ob, err := s.oracle.GetOrderbook(marketID, depth)
		if err == nil {
			// Convert to [][]string format for API compatibility
			bids := make([][]string, len(ob.Bids))
			asks := make([][]string, len(ob.Asks))

			for i, b := range ob.Bids {
				bids[i] = []string{b.Price, b.Quantity}
			}
			for i, a := range ob.Asks {
				asks[i] = []string{a.Price, a.Quantity}
			}

			return map[string]interface{}{
				"market_id": ob.MarketID,
				"bids":      bids,
				"asks":      asks,
				"timestamp": ob.Timestamp,
			}
		}
		// Log error but continue with fallback
		fmt.Printf("Oracle GetOrderbook error for %s: %v\n", marketID, err)
	}

	// Fallback: return empty orderbook
	return map[string]interface{}{
		"market_id": marketID,
		"bids":      [][]string{},
		"asks":      [][]string{},
		"timestamp": time.Now().UnixMilli(),
		"error":     "orderbook_unavailable",
	}
}

// getMockTrades returns recent trades from Hyperliquid real-time data
// Falls back to empty trades if Oracle is unavailable
func (s *Server) getMockTrades(marketID string, limit int) []map[string]interface{} {
	// Try to get real data from Oracle
	if s.oracle != nil {
		trades, err := s.oracle.GetRecentTrades(marketID, limit)
		if err == nil {
			result := make([]map[string]interface{}, len(trades))
			for i, t := range trades {
				result[i] = map[string]interface{}{
					"trade_id":  t.TradeID,
					"market_id": t.MarketID,
					"price":     t.Price,
					"quantity":  t.Quantity,
					"side":      t.Side,
					"timestamp": t.Timestamp,
				}
			}
			return result
		}
		// Log error but continue with fallback
		fmt.Printf("Oracle GetRecentTrades error for %s: %v\n", marketID, err)
	}

	// Fallback: return empty trades
	return []map[string]interface{}{}
}

// getMockKlines returns K-line data from Hyperliquid real-time candlesticks
// Falls back to empty klines if Oracle is unavailable
func (s *Server) getMockKlines(marketID string, interval string, limit int) []map[string]interface{} {
	// Try to get real data from Oracle
	if s.oracle != nil {
		klines, err := s.oracle.GetKlines(marketID, interval, limit)
		if err == nil {
			result := make([]map[string]interface{}, len(klines))
			for i, k := range klines {
				result[i] = map[string]interface{}{
					"time":   k.Time,
					"open":   k.Open,
					"high":   k.High,
					"low":    k.Low,
					"close":  k.Close,
					"volume": k.Volume,
				}
			}
			return result
		}
		// Log error but continue with fallback
		fmt.Printf("Oracle GetKlines error for %s: %v\n", marketID, err)
	}

	// Fallback: return empty klines
	return []map[string]interface{}{}
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

// getMockAccount returns account data
// NOTE: Returns empty account for new users - no hardcoded demo data
func (s *Server) getMockAccount(address string) map[string]interface{} {
	// Return fresh account with zero balances
	// Real balances come from chain state or deposits
	return map[string]interface{}{
		"address":           address,
		"total_equity":      "0.00",
		"available_balance": "0.00",
		"margin_used":       "0.00",
		"unrealized_pnl":    "0.00",
		"margin_ratio":      "0.00",
	}
}

// getMockPositions returns positions for an address
// NOTE: Returns empty positions - no hardcoded demo data
// Real positions are created through trading
func (s *Server) getMockPositions(address string) []map[string]interface{} {
	// Return empty positions - user must open positions through trading
	return []map[string]interface{}{}
}

// getMockOrders returns open orders for an address
// NOTE: Returns empty orders - no hardcoded demo data
// Real orders are created through the API
func (s *Server) getMockOrders(address string) []map[string]interface{} {
	// Return empty orders - user must place orders through trading
	return []map[string]interface{}{}
}

// getMockAccountTrades returns trade history for an address
// NOTE: Returns empty trades - no hardcoded demo data
// Real trades are recorded when orders are matched
func (s *Server) getMockAccountTrades(address string) []map[string]interface{} {
	// Return empty trades - trades are created when orders match
	return []map[string]interface{}{}
}

// startRealDataBroadcaster starts broadcasting real-time Hyperliquid data via WebSocket
// This broadcasts actual market data to all connected WebSocket clients
func (s *Server) startRealDataBroadcaster() {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	marketIDs := []string{"BTC-USDC", "ETH-USDC", "SOL-USDC"}

	for range ticker.C {
		for _, marketID := range marketIDs {
			// Broadcast ticker (real-time from Hyperliquid)
			tickerData := s.getMockTicker(marketID)

			// Skip if Oracle returned error
			if _, hasError := tickerData["error"]; hasError {
				continue
			}

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

			// Broadcast depth every 2 seconds (less frequent to reduce API load)
			if time.Now().Second()%2 == 0 {
				orderbookData := s.getMockOrderbook(marketID, 20)

				// Skip if Oracle returned error
				if _, hasError := orderbookData["error"]; hasError {
					continue
				}

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

			// Broadcast trade every ~3 seconds (sample from recent trades)
			if time.Now().Second()%3 == 0 {
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

// startMockDataBroadcaster is an alias for backward compatibility
// Deprecated: Use startRealDataBroadcaster instead
func (s *Server) startMockDataBroadcaster() {
	s.startRealDataBroadcaster()
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
