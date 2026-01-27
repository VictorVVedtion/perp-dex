package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	clog "cosmossdk.io/log"
	"github.com/openalpha/perp-dex/api/handlers"
	"github.com/openalpha/perp-dex/api/middleware"
	"github.com/openalpha/perp-dex/api/types"
	"github.com/openalpha/perp-dex/api/websocket"
)

// Server represents the API server
type Server struct {
	httpServer *http.Server
	wsServer   *websocket.Server
	config     *Config
	mockMode   bool

	// Services
	orderService     types.OrderService
	positionService  types.PositionService
	accountService   types.AccountService
	riverpoolService types.RiverpoolService

	// Handlers
	orderHandler     *handlers.OrderHandler
	positionHandler  *handlers.PositionHandler
	accountHandler   *handlers.AccountHandler
	riverpoolHandler *handlers.RiverpoolStandaloneHandler

	// Rate limiter
	rateLimiter *middleware.RateLimiter

	// Oracle for real-time prices (Hyperliquid)
	oracle *HyperliquidOracle
}

// Config contains server configuration
type Config struct {
	Host             string
	Port             int
	ReadTimeout      time.Duration
	WriteTimeout     time.Duration
	MockMode         bool
	DisableRateLimit bool // For testing purposes
}

// DefaultConfig returns default configuration
// NOTE: MockMode defaults to false (real mode) for production safety.
// Use --mock flag explicitly for development/testing with mock data.
func DefaultConfig() *Config {
	return &Config{
		Host:         "0.0.0.0",
		Port:         8080,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		MockMode:     false, // Default to REAL mode - use --mock for development
	}
}

// NewServer creates a new API server
func NewServer(config *Config) *Server {
	if config == nil {
		config = DefaultConfig()
	}

	wsConfig := websocket.DefaultServerConfig()
	wsConfig.Port = config.Port

	// Create mock service (default for now)
	mockService := NewMockService()

	// Create riverpool mock service
	riverpoolService := NewMockRiverpoolService()

	// Create rate limiter
	rateLimiter := middleware.NewRateLimiter(middleware.DefaultRateLimitConfig())

	// Create Hyperliquid Oracle for real-time prices
	oracle := NewHyperliquidOracle()

	s := &Server{
		config:           config,
		wsServer:         websocket.NewServer(wsConfig),
		mockMode:         config.MockMode,
		orderService:     mockService,
		positionService:  mockService,
		accountService:   mockService,
		riverpoolService: riverpoolService,
		rateLimiter:      rateLimiter,
		oracle:           oracle,
	}

	// Create handlers
	s.orderHandler = handlers.NewOrderHandler(s.orderService)
	s.positionHandler = handlers.NewPositionHandler(s.positionService)
	s.accountHandler = handlers.NewAccountHandler(s.accountService)
	s.riverpoolHandler = handlers.NewRiverpoolStandaloneHandler(s.riverpoolService)

	return s
}

// NewServerWithServices creates a new API server with custom services
func NewServerWithServices(config *Config, orderSvc types.OrderService, positionSvc types.PositionService, accountSvc types.AccountService) *Server {
	if config == nil {
		config = DefaultConfig()
	}

	wsConfig := websocket.DefaultServerConfig()
	wsConfig.Port = config.Port

	// Create rate limiter
	rateLimiter := middleware.NewRateLimiter(middleware.DefaultRateLimitConfig())

	// Create riverpool mock service
	riverpoolService := NewMockRiverpoolService()

	// Create Hyperliquid Oracle for real-time prices
	oracle := NewHyperliquidOracle()

	s := &Server{
		config:           config,
		wsServer:         websocket.NewServer(wsConfig),
		mockMode:         config.MockMode,
		orderService:     orderSvc,
		positionService:  positionSvc,
		accountService:   accountSvc,
		riverpoolService: riverpoolService,
		rateLimiter:      rateLimiter,
		oracle:           oracle,
	}

	// Create handlers
	s.orderHandler = handlers.NewOrderHandler(s.orderService)
	s.positionHandler = handlers.NewPositionHandler(s.positionService)
	s.accountHandler = handlers.NewAccountHandler(s.accountService)
	s.riverpoolHandler = handlers.NewRiverpoolStandaloneHandler(s.riverpoolService)

	return s
}

// NewServerWithRealService creates an API server with real orderbook engine
// This uses the actual MatchingEngineV2 for order processing
func NewServerWithRealService(config *Config) (*Server, error) {
	if config == nil {
		config = DefaultConfig()
	}
	config.MockMode = false

	// Create real service with in-memory store
	logger := clog.NewNopLogger()
	realService, err := NewRealService(logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create real service: %w", err)
	}

	wsConfig := websocket.DefaultServerConfig()
	wsConfig.Port = config.Port

	// Create rate limiter
	rateLimiter := middleware.NewRateLimiter(middleware.DefaultRateLimitConfig())

	// Create riverpool mock service
	riverpoolService := NewMockRiverpoolService()

	// Create Hyperliquid Oracle for real-time prices
	oracle := NewHyperliquidOracle()

	s := &Server{
		config:           config,
		wsServer:         websocket.NewServer(wsConfig),
		mockMode:         false,
		orderService:     realService,
		positionService:  realService,
		accountService:   realService,
		riverpoolService: riverpoolService,
		rateLimiter:      rateLimiter,
		oracle:           oracle,
	}

	// Create handlers
	s.orderHandler = handlers.NewOrderHandler(s.orderService)
	s.positionHandler = handlers.NewPositionHandler(s.positionService)
	s.accountHandler = handlers.NewAccountHandler(s.accountService)
	s.riverpoolHandler = handlers.NewRiverpoolStandaloneHandler(s.riverpoolService)

	return s, nil
}

// Start starts the API server
func (s *Server) Start() error {
	mux := http.NewServeMux()

	// Health check (support both /health and /v1/health for compatibility)
	mux.HandleFunc("/health", s.handleHealth)
	mux.HandleFunc("/v1/health", s.handleHealth)

	// Market endpoints (read-only)
	mux.HandleFunc("/v1/markets", s.handleMarkets)
	mux.HandleFunc("/v1/markets/", s.handleMarket)

	// Account endpoints (legacy read-only)
	mux.HandleFunc("/v1/accounts/", s.handleAccountLegacy)

	// Tickers
	mux.HandleFunc("/v1/tickers", s.handleTickers)

	// === NEW ENDPOINTS ===

	// Order endpoints (POST, GET, PUT, DELETE)
	mux.HandleFunc("/v1/orders", s.orderHandler.HandleOrders)
	mux.HandleFunc("/v1/orders/", s.orderHandler.HandleOrder)

	// Position endpoints (GET, POST close)
	mux.HandleFunc("/v1/positions", s.positionHandler.HandlePositions)
	mux.HandleFunc("/v1/positions/close", s.positionHandler.HandleClosePosition)
	mux.HandleFunc("/v1/positions/", s.positionHandler.HandlePosition)

	// Account endpoints (GET, POST deposit/withdraw)
	mux.HandleFunc("/v1/account", s.accountHandler.HandleAccount)
	mux.HandleFunc("/v1/account/deposit", s.accountHandler.HandleDeposit)
	mux.HandleFunc("/v1/account/withdraw", s.accountHandler.HandleWithdraw)

	// WebSocket
	mux.HandleFunc("/ws", s.wsServer.GetHub().ServeWS)

	// === RIVERPOOL ENDPOINTS ===
	// Pool listing and details
	mux.HandleFunc("/v1/riverpool/pools", s.riverpoolHandler.GetPools)
	mux.HandleFunc("/v1/riverpool/pools/", s.handleRiverpoolPoolRoutes)

	// Deposit and withdrawal operations
	mux.HandleFunc("/v1/riverpool/deposit", s.riverpoolHandler.Deposit)
	mux.HandleFunc("/v1/riverpool/withdrawal/request", s.riverpoolHandler.RequestWithdrawal)
	mux.HandleFunc("/v1/riverpool/withdrawal/claim", s.riverpoolHandler.ClaimWithdrawal)
	mux.HandleFunc("/v1/riverpool/withdrawals/pending", s.riverpoolHandler.GetPendingWithdrawals)

	// User-specific endpoints
	mux.HandleFunc("/v1/riverpool/user/", s.handleRiverpoolUserRoutes)

	// Community pool management
	mux.HandleFunc("/v1/riverpool/community/create", s.riverpoolHandler.CreateCommunityPool)
	mux.HandleFunc("/v1/riverpool/community/", s.handleRiverpoolCommunityRoutes)

	// Apply middleware chain: CORS -> RateLimit -> Handler
	var handler http.Handler
	if s.config.DisableRateLimit {
		handler = corsMiddleware(mux)
	} else {
		handler = corsMiddleware(
			middleware.RateLimitMiddleware(s.rateLimiter)(mux),
		)
	}

	addr := fmt.Sprintf("%s:%d", s.config.Host, s.config.Port)
	s.httpServer = &http.Server{
		Addr:         addr,
		Handler:      handler,
		ReadTimeout:  s.config.ReadTimeout,
		WriteTimeout: s.config.WriteTimeout,
	}

	// Start WebSocket hub
	go s.wsServer.GetHub().Run()

	// Start real-time data broadcaster (uses Hyperliquid Oracle)
	// Now broadcasts real data in all modes
	go s.startRealDataBroadcaster()

	log.Printf("API server starting on %s (mock mode: %v)", addr, s.mockMode)
	log.Printf("Using Hyperliquid Oracle for real-time prices")
	log.Printf("New endpoints enabled: /v1/orders, /v1/positions, /v1/account")
	if s.config.DisableRateLimit {
		log.Printf("Rate limiting DISABLED (for testing)")
	} else {
		log.Printf("Rate limiting enabled: %d req/s per IP", 100)
	}
	return s.httpServer.ListenAndServe()
}

// Stop gracefully shuts down the server
func (s *Server) Stop(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}

// handleHealth handles health check requests
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	mode := "real"
	modeDescription := "Using in-memory orderbook engine (standalone mode)"
	if s.mockMode {
		mode = "mock"
		modeDescription = "Using mock data for development/testing"
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":           "healthy",
		"timestamp":        time.Now().Unix(),
		"mode":             mode,
		"mode_description": modeDescription,
		"mock_mode":        s.mockMode, // Deprecated: use "mode" instead
		"warning":          "This API uses in-memory storage. For production, connect to a running Cosmos chain.",
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

// handleAccountLegacy handles /v1/accounts/{addr}/* endpoints (legacy read-only)
func (s *Server) handleAccountLegacy(w http.ResponseWriter, r *http.Request) {
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
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-API-Key, X-Trader-Address")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// === RIVERPOOL ROUTE HANDLERS ===

// handleRiverpoolPoolRoutes handles /v1/riverpool/pools/{poolId}/* endpoints
func (s *Server) handleRiverpoolPoolRoutes(w http.ResponseWriter, r *http.Request) {
	// Parse path: /v1/riverpool/pools/{poolId} or /v1/riverpool/pools/{poolId}/{endpoint}
	path := r.URL.Path[len("/v1/riverpool/pools/"):]

	// Extract pool ID and endpoint
	poolID := path
	endpoint := ""
	for i, c := range path {
		if c == '/' {
			poolID = path[:i]
			endpoint = path[i+1:]
			break
		}
	}

	if poolID == "" {
		writeError(w, http.StatusBadRequest, "Pool ID required")
		return
	}

	// Set pool ID in request for handler
	r.Header.Set("X-Pool-ID", poolID)

	switch endpoint {
	case "":
		s.riverpoolHandler.GetPool(w, r)
	case "stats":
		s.riverpoolHandler.GetPoolStats(w, r)
	case "nav":
		s.riverpoolHandler.GetNAVHistory(w, r)
	case "ddguard":
		s.riverpoolHandler.GetDDGuardState(w, r)
	case "deposits":
		s.riverpoolHandler.GetPoolDeposits(w, r)
	case "withdrawals":
		s.riverpoolHandler.GetPoolWithdrawals(w, r)
	case "holders":
		s.riverpoolHandler.GetPoolHolders(w, r)
	case "positions":
		s.riverpoolHandler.GetPoolPositions(w, r)
	case "trades":
		s.riverpoolHandler.GetPoolTrades(w, r)
	case "revenue":
		s.riverpoolHandler.GetPoolRevenue(w, r)
	default:
		writeError(w, http.StatusNotFound, "Endpoint not found")
	}
}

// handleRiverpoolUserRoutes handles /v1/riverpool/user/{address}/* endpoints
func (s *Server) handleRiverpoolUserRoutes(w http.ResponseWriter, r *http.Request) {
	// Parse path: /v1/riverpool/user/{address} or /v1/riverpool/user/{address}/{endpoint}
	path := r.URL.Path[len("/v1/riverpool/user/"):]

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

	if address == "" {
		writeError(w, http.StatusBadRequest, "User address required")
		return
	}

	// Set address in request for handler
	r.Header.Set("X-User-Address", address)

	switch endpoint {
	case "", "deposits":
		s.riverpoolHandler.GetUserDeposits(w, r)
	case "withdrawals":
		s.riverpoolHandler.GetUserWithdrawals(w, r)
	case "pools":
		s.riverpoolHandler.GetUserPools(w, r)
	default:
		writeError(w, http.StatusNotFound, "Endpoint not found")
	}
}

// handleRiverpoolCommunityRoutes handles /v1/riverpool/community/{poolId}/* endpoints
func (s *Server) handleRiverpoolCommunityRoutes(w http.ResponseWriter, r *http.Request) {
	// Parse path: /v1/riverpool/community/{poolId}/{action}
	path := r.URL.Path[len("/v1/riverpool/community/"):]

	// Extract pool ID and action
	poolID := path
	action := ""
	for i, c := range path {
		if c == '/' {
			poolID = path[:i]
			action = path[i+1:]
			break
		}
	}

	if poolID == "" {
		writeError(w, http.StatusBadRequest, "Pool ID required")
		return
	}

	// Set pool ID in request for handler
	r.Header.Set("X-Pool-ID", poolID)

	switch action {
	case "update":
		s.riverpoolHandler.UpdateCommunityPool(w, r)
	case "invite":
		s.riverpoolHandler.GenerateInviteCode(w, r)
	case "order":
		s.riverpoolHandler.PlacePoolOrder(w, r)
	case "close":
		s.riverpoolHandler.ClosePoolPosition(w, r)
	case "pause":
		s.riverpoolHandler.PausePool(w, r)
	case "resume":
		s.riverpoolHandler.ResumePool(w, r)
	case "close-pool":
		s.riverpoolHandler.ClosePool(w, r)
	default:
		writeError(w, http.StatusNotFound, "Action not found")
	}
}
