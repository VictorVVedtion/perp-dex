package main

import (
	"context"
	"encoding/json"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/openalpha/perp-dex/api"
)

func main() {
	// Command line flags
	host := flag.String("host", "0.0.0.0", "Server host")
	port := flag.Int("port", 8080, "Server port")
	mockMode := flag.Bool("mock", false, "Enable mock data mode")
	keeperMode := flag.Bool("keeper", false, "Enable real keeper mode (connects to order book engine)")
	benchMode := flag.Bool("bench", false, "Enable benchmark mode (no rate limiting)")
	flag.Parse()

	// In bench mode, disable rate limiting by setting high limit
	if *benchMode {
		log.Println("Benchmark mode: Rate limiting disabled")
	}

	// Create configuration
	config := &api.Config{
		Host:         *host,
		Port:         *port,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		MockMode:     *mockMode,
	}

	// Create server
	var server *api.Server
	var keeperService *api.KeeperService
	if *keeperMode {
		// Use real KeeperService connected to order book engine
		keeperService = api.NewKeeperService()
		server = api.NewServerWithServices(config, keeperService, keeperService, keeperService)
		log.Println("Using KeeperService (real order book engine)")
	} else {
		server = api.NewServer(config)
	}

	// Add real orderbook endpoint if in keeper mode
	if *keeperMode && keeperService != nil {
		go func() {
			time.Sleep(100 * time.Millisecond) // Wait for main server to start
			mux := http.NewServeMux()
			mux.HandleFunc("/orderbook/", func(w http.ResponseWriter, r *http.Request) {
				marketID := r.URL.Path[len("/orderbook/"):]
				if marketID == "" {
					marketID = "BTC-USDC"
				}
				bids, asks := keeperService.GetOrderBookDepth(marketID, 20)
				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("Access-Control-Allow-Origin", "*")
				json.NewEncoder(w).Encode(map[string]interface{}{
					"market_id": marketID,
					"bids":      bids,
					"asks":      asks,
					"timestamp": time.Now().UnixMilli(),
				})
			})
			log.Println("Real orderbook endpoint: http://localhost:8081/orderbook/BTC-USDC")
			http.ListenAndServe(":8081", mux)
		}()
	}

	// Start server in goroutine
	go func() {
		if err := server.Start(); err != nil {
			log.Printf("Server error: %v", err)
		}
	}()

	log.Printf("PerpDEX API Server started on %s:%d", *host, *port)
	log.Printf("Mock mode: %v", *mockMode)
	log.Printf("WebSocket endpoint: ws://%s:%d/ws", *host, *port)
	log.Printf("Health check: http://%s:%d/health", *host, *port)

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Stop(ctx); err != nil {
		log.Printf("Server shutdown error: %v", err)
	}

	log.Println("Server exited")
}
