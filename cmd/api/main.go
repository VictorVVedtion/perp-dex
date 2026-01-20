package main

import (
	"context"
	"flag"
	"log"
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
	mockMode := flag.Bool("mock", false, "Enable mock data mode (default: false for real mode)")
	realMode := flag.Bool("real", false, "Enable real orderbook engine mode (uses MatchingEngineV2)")
	flag.Parse()

	// Create configuration
	config := &api.Config{
		Host:         *host,
		Port:         *port,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		MockMode:     *mockMode && !*realMode,
	}

	var server *api.Server
	var err error

	// Create server based on mode
	if *realMode {
		log.Println("Initializing with REAL orderbook engine (MatchingEngineV2)...")
		server, err = api.NewServerWithRealService(config)
		if err != nil {
			log.Fatalf("Failed to create real service: %v", err)
		}
		log.Println("Real orderbook engine initialized successfully")
	} else {
		server = api.NewServer(config)
	}

	// Start server in goroutine
	go func() {
		if err := server.Start(); err != nil {
			log.Printf("Server error: %v", err)
		}
	}()

	engineMode := "mock"
	if *realMode {
		engineMode = "REAL (MatchingEngineV2)"
	}

	log.Printf("PerpDEX API Server started on %s:%d", *host, *port)
	log.Printf("Engine mode: %s", engineMode)
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
