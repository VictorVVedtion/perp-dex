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
	storageWarning := ""
	if *realMode {
		engineMode = "REAL (MatchingEngineV2)"
		storageWarning = "\n⚠️  WARNING: Using in-memory storage. Data will be lost on restart.\n   For production, ensure connection to a running Cosmos chain."
	}

	log.Printf("╔══════════════════════════════════════════════════════════════╗")
	log.Printf("║  PerpDEX API Server                                          ║")
	log.Printf("╠══════════════════════════════════════════════════════════════╣")
	log.Printf("║  Address:   %s:%d", *host, *port)
	log.Printf("║  Mode:      %s", engineMode)
	log.Printf("║  WebSocket: ws://%s:%d/ws", *host, *port)
	log.Printf("║  Health:    http://%s:%d/health", *host, *port)
	log.Printf("╚══════════════════════════════════════════════════════════════╝")
	if storageWarning != "" {
		log.Print(storageWarning)
	}

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
