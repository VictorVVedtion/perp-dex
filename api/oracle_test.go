package api

import (
	"testing"
)

func TestHyperliquidOracle_GetPrice(t *testing.T) {
	oracle := NewHyperliquidOracle()

	tests := []struct {
		marketID    string
		expectValid bool
	}{
		{"BTC-USDC", true},
		{"ETH-USDC", true},
		{"SOL-USDC", true},
		{"INVALID", false},
	}

	for _, tt := range tests {
		t.Run(tt.marketID, func(t *testing.T) {
			price, err := oracle.GetPrice(tt.marketID)

			if tt.expectValid {
				if err != nil {
					t.Errorf("GetPrice(%s) error = %v, want nil", tt.marketID, err)
					return
				}
				if price.IsZero() {
					t.Errorf("GetPrice(%s) returned zero price", tt.marketID)
					return
				}
				// Validate BTC price is in reasonable range (50k - 150k)
				if tt.marketID == "BTC-USDC" {
					priceFloat := price.MustFloat64()
					if priceFloat < 50000 || priceFloat > 150000 {
						t.Errorf("GetPrice(BTC-USDC) = %v, outside expected range [50000, 150000]", priceFloat)
					}
					t.Logf("BTC-USDC real price: $%.2f", priceFloat)
				}
			} else {
				if err == nil {
					t.Errorf("GetPrice(%s) expected error, got nil", tt.marketID)
				}
			}
		})
	}
}

func TestHyperliquidOracle_GetTicker(t *testing.T) {
	oracle := NewHyperliquidOracle()

	ticker, err := oracle.GetTicker("BTC-USDC")
	if err != nil {
		t.Fatalf("GetTicker(BTC-USDC) error = %v", err)
	}

	if ticker.MarketID != "BTC-USDC" {
		t.Errorf("ticker.MarketID = %v, want BTC-USDC", ticker.MarketID)
	}
	if ticker.MarkPrice == "" || ticker.MarkPrice == "0" {
		t.Errorf("ticker.MarkPrice = %v, want non-zero", ticker.MarkPrice)
	}

	t.Logf("BTC-USDC ticker: markPrice=%s, volume24h=%s", ticker.MarkPrice, ticker.Volume24h)
}

func TestHyperliquidOracle_GetOrderbook(t *testing.T) {
	oracle := NewHyperliquidOracle()

	ob, err := oracle.GetOrderbook("BTC-USDC", 5)
	if err != nil {
		t.Fatalf("GetOrderbook(BTC-USDC, 5) error = %v", err)
	}

	if ob.MarketID != "BTC-USDC" {
		t.Errorf("ob.MarketID = %v, want BTC-USDC", ob.MarketID)
	}
	if len(ob.Bids) == 0 {
		t.Error("ob.Bids is empty")
	}
	if len(ob.Asks) == 0 {
		t.Error("ob.Asks is empty")
	}

	t.Logf("BTC-USDC orderbook: %d bids, %d asks", len(ob.Bids), len(ob.Asks))
	if len(ob.Bids) > 0 {
		t.Logf("  Best bid: %s @ %s", ob.Bids[0].Quantity, ob.Bids[0].Price)
	}
	if len(ob.Asks) > 0 {
		t.Logf("  Best ask: %s @ %s", ob.Asks[0].Quantity, ob.Asks[0].Price)
	}
}

func TestHyperliquidOracle_GetRecentTrades(t *testing.T) {
	oracle := NewHyperliquidOracle()

	trades, err := oracle.GetRecentTrades("BTC-USDC", 5)
	if err != nil {
		t.Fatalf("GetRecentTrades(BTC-USDC, 5) error = %v", err)
	}

	if len(trades) == 0 {
		t.Error("trades is empty")
	}

	t.Logf("BTC-USDC recent trades: %d trades", len(trades))
	if len(trades) > 0 {
		t.Logf("  Last trade: %s @ %s (%s)", trades[0].Quantity, trades[0].Price, trades[0].Side)
	}
}

func TestHyperliquidOracle_GetKlines(t *testing.T) {
	oracle := NewHyperliquidOracle()

	klines, err := oracle.GetKlines("BTC-USDC", "1h", 10)
	if err != nil {
		t.Fatalf("GetKlines(BTC-USDC, 1h, 10) error = %v", err)
	}

	if len(klines) == 0 {
		t.Error("klines is empty")
	}

	t.Logf("BTC-USDC klines (1h): %d candles", len(klines))
	if len(klines) > 0 {
		k := klines[len(klines)-1]
		t.Logf("  Latest candle: O=%.2f H=%.2f L=%.2f C=%.2f V=%.2f", k.Open, k.High, k.Low, k.Close, k.Volume)
	}
}
