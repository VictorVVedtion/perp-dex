package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/openalpha/perp-dex/api/websocket"
)

// Server represents the API server
type Server struct {
	httpServer *http.Server
	wsServer   *websocket.Server
	config     *Config
	mockMode   bool
}

// Config contains server configuration
type Config struct {
	Host         string
	Port         int
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	MockMode     bool
}

// DefaultConfig returns default configuration
func DefaultConfig() *Config {
	return &Config{
		Host:         "0.0.0.0",
		Port:         8080,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		MockMode:     true, // Default to mock mode for development
	}
}

// NewServer creates a new API server
func NewServer(config *Config) *Server {
	if config == nil {
		config = DefaultConfig()
	}

	wsConfig := websocket.DefaultServerConfig()
	wsConfig.Port = config.Port

	return &Server{
		config:   config,
		wsServer: websocket.NewServer(wsConfig),
		mockMode: config.MockMode,
	}
}

// Start starts the API server
func (s *Server) Start() error {
	mux := http.NewServeMux()

	// Health check
	mux.HandleFunc("/health", s.handleHealth)

	// Market endpoints
	mux.HandleFunc("/v1/markets", s.handleMarkets)
	mux.HandleFunc("/v1/markets/", s.handleMarket)

	// Account endpoints
	mux.HandleFunc("/v1/accounts/", s.handleAccount)

	// Tickers
	mux.HandleFunc("/v1/tickers", s.handleTickers)

	// WebSocket
	mux.HandleFunc("/ws", s.wsServer.GetHub().ServeWS)

	// CORS middleware
	handler := corsMiddleware(mux)

	addr := fmt.Sprintf("%s:%d", s.config.Host, s.config.Port)
	s.httpServer = &http.Server{
		Addr:         addr,
		Handler:      handler,
		ReadTimeout:  s.config.ReadTimeout,
		WriteTimeout: s.config.WriteTimeout,
	}

	// Start WebSocket hub
	go s.wsServer.GetHub().Run()

	// Start mock data broadcaster if in mock mode
	if s.mockMode {
		go s.startMockDataBroadcaster()
	}

	log.Printf("API server starting on %s (mock mode: %v)", addr, s.mockMode)
	return s.httpServer.ListenAndServe()
}

// Stop gracefully shuts down the server
func (s *Server) Stop(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}

// handleHealth handles health check requests
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().Unix(),
		"mock_mode": s.mockMode,
	})
}

// handleMarkets handles /v1/markets
func (s *Server) handleMarkets(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	markets := s.getMockMarkets()
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"markets": markets,
	})
}

// handleMarket handles /v1/markets/{id}/* endpoints
func (s *Server) handleMarket(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	// Parse path: /v1/markets/{id} or /v1/markets/{id}/{endpoint}
	path := r.URL.Path[len("/v1/markets/"):]

	// Extract market ID and endpoint
	marketID := path
	endpoint := ""
	for i, c := range path {
		if c == '/' {
			marketID = path[:i]
			endpoint = path[i+1:]
			break
		}
	}

	switch endpoint {
	case "":
		// Single market
		market := s.getMockMarket(marketID)
		if market == nil {
			writeError(w, http.StatusNotFound, "Market not found")
			return
		}
		writeJSON(w, http.StatusOK, market)

	case "ticker":
		ticker := s.getMockTicker(marketID)
		writeJSON(w, http.StatusOK, ticker)

	case "orderbook":
		depth := 20
		if d := r.URL.Query().Get("depth"); d != "" {
			fmt.Sscanf(d, "%d", &depth)
		}
		orderbook := s.getMockOrderbook(marketID, depth)
		writeJSON(w, http.StatusOK, orderbook)

	case "trades":
		limit := 100
		if l := r.URL.Query().Get("limit"); l != "" {
			fmt.Sscanf(l, "%d", &limit)
		}
		trades := s.getMockTrades(marketID, limit)
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"trades": trades,
		})

	case "klines":
		interval := r.URL.Query().Get("interval")
		if interval == "" {
			interval = "1h"
		}
		limit := 100
		if l := r.URL.Query().Get("limit"); l != "" {
			fmt.Sscanf(l, "%d", &limit)
		}
		klines := s.getMockKlines(marketID, interval, limit)
		writeJSON(w, http.StatusOK, klines)

	case "funding":
		funding := s.getMockFunding(marketID)
		writeJSON(w, http.StatusOK, funding)

	default:
		writeError(w, http.StatusNotFound, "Endpoint not found")
	}
}

// handleAccount handles /v1/accounts/{addr}/* endpoints
func (s *Server) handleAccount(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	path := r.URL.Path[len("/v1/accounts/"):]

	// Extract address and endpoint
	address := path
	endpoint := ""
	for i, c := range path {
		if c == '/' {
			address = path[:i]
			endpoint = path[i+1:]
			break
		}
	}

	switch endpoint {
	case "":
		account := s.getMockAccount(address)
		writeJSON(w, http.StatusOK, account)

	case "positions":
		positions := s.getMockPositions(address)
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"positions": positions,
		})

	case "orders":
		orders := s.getMockOrders(address)
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"orders": orders,
		})

	case "trades":
		trades := s.getMockAccountTrades(address)
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"trades": trades,
		})

	default:
		writeError(w, http.StatusNotFound, "Endpoint not found")
	}
}

// handleTickers handles /v1/tickers
func (s *Server) handleTickers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	tickers := []map[string]interface{}{
		s.getMockTicker("BTC-USDC"),
		s.getMockTicker("ETH-USDC"),
		s.getMockTicker("SOL-USDC"),
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"tickers": tickers,
	})
}

// Helper functions

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]interface{}{
		"error": message,
	})
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-API-Key")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}
