package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"cosmossdk.io/math"
	"github.com/openalpha/perp-dex/offchain/matcher"
	"github.com/openalpha/perp-dex/x/orderbook/types"
)

// Config holds the application configuration
type Config struct {
	BatchSize     int           `json:"batch_size"`
	BatchInterval time.Duration `json:"batch_interval"`
	WebSocketURL  string        `json:"websocket_url"`
	ChainRPCURL   string        `json:"chain_rpc_url"`
	SubmitterType string        `json:"submitter_type"` // "mock" or "batch"
	Demo          bool          `json:"demo"`           // run demo mode
}

// DefaultConfig returns the default configuration
func DefaultConfig() *Config {
	return &Config{
		BatchSize:     100,
		BatchInterval: 500 * time.Millisecond,
		WebSocketURL:  "ws://localhost:26657/websocket",
		ChainRPCURL:   "http://localhost:26657",
		SubmitterType: "mock",
		Demo:          false,
	}
}

// LoadConfig loads configuration from a file
func LoadConfig(path string) (*Config, error) {
	config := DefaultConfig()

	if path == "" {
		return config, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return config, nil
		}
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	if err := json.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return config, nil
}

func main() {
	// Parse command line flags
	configPath := flag.String("config", "", "Path to config file")
	batchSize := flag.Int("batch-size", 0, "Maximum trades per batch")
	batchInterval := flag.Duration("batch-interval", 0, "Time interval for batch submission")
	rpcURL := flag.String("rpc", "", "Chain RPC URL")
	wsURL := flag.String("ws", "", "WebSocket URL")
	submitterType := flag.String("submitter", "", "Submitter type (mock or batch)")
	demo := flag.Bool("demo", false, "Run demo mode with sample orders")
	flag.Parse()

	// Load configuration
	config, err := LoadConfig(*configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Override with command line flags
	if *batchSize > 0 {
		config.BatchSize = *batchSize
	}
	if *batchInterval > 0 {
		config.BatchInterval = *batchInterval
	}
	if *rpcURL != "" {
		config.ChainRPCURL = *rpcURL
	}
	if *wsURL != "" {
		config.WebSocketURL = *wsURL
	}
	if *submitterType != "" {
		config.SubmitterType = *submitterType
	}
	if *demo {
		config.Demo = true
	}

	// Print configuration
	log.Println("=== PerpDEX Offchain Matcher ===")
	log.Printf("Batch Size: %d", config.BatchSize)
	log.Printf("Batch Interval: %v", config.BatchInterval)
	log.Printf("Chain RPC: %s", config.ChainRPCURL)
	log.Printf("WebSocket: %s", config.WebSocketURL)
	log.Printf("Submitter: %s", config.SubmitterType)
	log.Println("================================")

	// Create submitter
	factory := matcher.NewSubmitterFactory()
	submitter := factory.Create(config.SubmitterType, &matcher.BatchSubmitterConfig{
		RPCURL:        config.ChainRPCURL,
		BatchSize:     config.BatchSize,
		RetryAttempts: 3,
		RetryDelay:    time.Second,
	})

	// Create matcher
	matcherConfig := &matcher.Config{
		BatchSize:     config.BatchSize,
		BatchInterval: config.BatchInterval,
		WebSocketURL:  config.WebSocketURL,
		ChainRPCURL:   config.ChainRPCURL,
	}
	m := matcher.NewOffchainMatcher(matcherConfig, submitter)

	// Setup context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start the matcher
	if err := m.Start(ctx); err != nil {
		log.Fatalf("Failed to start matcher: %v", err)
	}

	// Run demo if requested
	if config.Demo {
		go runDemo(m)
	}

	// Wait for interrupt signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	// Periodic stats logging
	statsTicker := time.NewTicker(10 * time.Second)
	defer statsTicker.Stop()

	log.Println("Matcher is running. Press Ctrl+C to stop.")

	for {
		select {
		case sig := <-sigCh:
			log.Printf("Received signal: %v", sig)
			cancel()
			if err := m.Stop(); err != nil {
				log.Printf("Error stopping matcher: %v", err)
			}
			log.Println("Matcher stopped")
			return
		case <-statsTicker.C:
			stats := m.GetStats()
			log.Printf("Stats: Orders=%d, OrderBooks=%d, PendingTrades=%d, CacheSize=%d",
				stats.OrderCount, stats.OrderBookCount, stats.PendingTrades, stats.CacheSize)
		}
	}
}

// runDemo runs a demonstration with sample orders
func runDemo(m *matcher.OffchainMatcher) {
	log.Println("Starting demo mode...")
	time.Sleep(time.Second)

	marketID := "BTC-USDT-PERP"

	// Create some sell orders (asks)
	sellPrices := []string{"50100", "50200", "50300"}
	for i, price := range sellPrices {
		priceVal, _ := math.LegacyNewDecFromStr(price)
		qtyVal, _ := math.LegacyNewDecFromStr("1.5")
		order := types.NewOrder(
			fmt.Sprintf("sell-order-%d", i+1),
			fmt.Sprintf("trader-sell-%d", i+1),
			marketID,
			types.SideSell,
			types.OrderTypeLimit,
			priceVal,
			qtyVal,
		)
		log.Printf("Submitting sell order: %s @ %s", order.OrderID, price)
		m.SubmitOrder(order)
		time.Sleep(100 * time.Millisecond)
	}

	// Create some buy orders (bids)
	buyPrices := []string{"49900", "49800", "49700"}
	for i, price := range buyPrices {
		priceVal, _ := math.LegacyNewDecFromStr(price)
		qtyVal, _ := math.LegacyNewDecFromStr("2.0")
		order := types.NewOrder(
			fmt.Sprintf("buy-order-%d", i+1),
			fmt.Sprintf("trader-buy-%d", i+1),
			marketID,
			types.SideBuy,
			types.OrderTypeLimit,
			priceVal,
			qtyVal,
		)
		log.Printf("Submitting buy order: %s @ %s", order.OrderID, price)
		m.SubmitOrder(order)
		time.Sleep(100 * time.Millisecond)
	}

	time.Sleep(500 * time.Millisecond)
	printOrderBook(m, marketID)

	// Submit a market buy order that will match against sells
	log.Println("\n=== Submitting Market Buy Order ===")
	marketBuyQty, _ := math.LegacyNewDecFromStr("2.0")
	marketBuyOrder := types.NewOrder(
		"market-buy-1",
		"trader-market-1",
		marketID,
		types.SideBuy,
		types.OrderTypeMarket,
		math.LegacyZeroDec(), // Market orders don't have a price
		marketBuyQty,
	)
	m.SubmitOrder(marketBuyOrder)
	time.Sleep(500 * time.Millisecond)

	log.Println("\n=== Order Book After Market Buy ===")
	printOrderBook(m, marketID)

	// Submit a limit buy order that crosses the spread
	log.Println("\n=== Submitting Aggressive Limit Buy Order ===")
	aggressivePrice, _ := math.LegacyNewDecFromStr("50250")
	aggressiveQty, _ := math.LegacyNewDecFromStr("1.0")
	aggressiveOrder := types.NewOrder(
		"aggressive-buy-1",
		"trader-aggressive-1",
		marketID,
		types.SideBuy,
		types.OrderTypeLimit,
		aggressivePrice,
		aggressiveQty,
	)
	m.SubmitOrder(aggressiveOrder)
	time.Sleep(500 * time.Millisecond)

	log.Println("\n=== Final Order Book ===")
	printOrderBook(m, marketID)

	log.Println("\nDemo completed!")
}

// printOrderBook prints the current state of the order book
func printOrderBook(m *matcher.OffchainMatcher, marketID string) {
	ob := m.GetOrderBook(marketID)
	if ob == nil {
		log.Println("Order book not found")
		return
	}

	log.Printf("Order Book for %s:", marketID)
	log.Println("  Asks (Sells):")
	if len(ob.Asks) == 0 {
		log.Println("    (empty)")
	}
	for i := len(ob.Asks) - 1; i >= 0; i-- {
		level := ob.Asks[i]
		log.Printf("    %s @ %s (orders: %d)", level.Quantity.String(), level.Price.String(), len(level.OrderIDs))
	}
	log.Println("  -----------")
	log.Println("  Bids (Buys):")
	if len(ob.Bids) == 0 {
		log.Println("    (empty)")
	}
	for _, level := range ob.Bids {
		log.Printf("    %s @ %s (orders: %d)", level.Quantity.String(), level.Price.String(), len(level.OrderIDs))
	}
}
