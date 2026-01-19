package websocket

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Server represents the WebSocket server
type Server struct {
	hub        *Hub
	httpServer *http.Server
	config     *ServerConfig

	// Connection management
	connections    map[string]*Client
	connectionsMu  sync.RWMutex
	connectionsPerIP map[string]int
	ipMu           sync.RWMutex

	// Metrics
	totalConnections   int64
	totalMessages      int64
	activeConnections  int64
	metricsMu          sync.RWMutex

	// Shutdown
	shutdownCh chan struct{}
}

// ServerConfig contains server configuration
type ServerConfig struct {
	// Server settings
	Host            string
	Port            int
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration

	// Security
	AllowedOrigins  []string
	MaxConnPerIP    int

	// TLS (optional)
	TLSCertFile     string
	TLSKeyFile      string

	// Hub configuration
	HubConfig       *HubConfig
}

// DefaultServerConfig returns default server configuration
func DefaultServerConfig() *ServerConfig {
	return &ServerConfig{
		Host:           "0.0.0.0",
		Port:           8080,
		ReadTimeout:    30 * time.Second,
		WriteTimeout:   30 * time.Second,
		AllowedOrigins: []string{"*"},
		MaxConnPerIP:   10,
		HubConfig:      DefaultHubConfig(),
	}
}

// NewServer creates a new WebSocket server
func NewServer(config *ServerConfig) *Server {
	if config == nil {
		config = DefaultServerConfig()
	}

	hub := NewHub(config.HubConfig)

	return &Server{
		hub:              hub,
		config:           config,
		connections:      make(map[string]*Client),
		connectionsPerIP: make(map[string]int),
		shutdownCh:       make(chan struct{}),
	}
}

// Start starts the WebSocket server
func (s *Server) Start() error {
	// Start the hub
	go s.hub.Run()

	// Setup HTTP handlers
	mux := http.NewServeMux()
	mux.HandleFunc("/ws", s.handleWebSocket)
	mux.HandleFunc("/health", s.handleHealth)
	mux.HandleFunc("/stats", s.handleStats)

	addr := fmt.Sprintf("%s:%d", s.config.Host, s.config.Port)
	s.httpServer = &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  s.config.ReadTimeout,
		WriteTimeout: s.config.WriteTimeout,
	}

	log.Printf("WebSocket server starting on %s", addr)

	if s.config.TLSCertFile != "" && s.config.TLSKeyFile != "" {
		return s.httpServer.ListenAndServeTLS(s.config.TLSCertFile, s.config.TLSKeyFile)
	}

	return s.httpServer.ListenAndServe()
}

// Stop gracefully shuts down the server
func (s *Server) Stop(ctx context.Context) error {
	close(s.shutdownCh)
	return s.httpServer.Shutdown(ctx)
}

// handleWebSocket handles WebSocket upgrade requests
func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	// Get client IP
	ip := getClientIP(r)

	// Check IP connection limit
	if !s.checkIPLimit(ip) {
		http.Error(w, "Too many connections from this IP", http.StatusTooManyRequests)
		return
	}

	// Upgrade to WebSocket
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}

	// Create client
	clientID := uuid.New().String()
	userID := r.URL.Query().Get("user_id") // Optional: from query param or auth header

	client := NewClient(s.hub, conn, clientID, userID, ip)

	// Register client
	s.registerConnection(client)

	// Start client pumps
	go client.writePump()
	go client.readPump()

	// Update metrics
	s.metricsMu.Lock()
	s.totalConnections++
	s.activeConnections++
	s.metricsMu.Unlock()
}

// handleHealth handles health check requests
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"healthy"}`))
}

// handleStats handles stats requests
func (s *Server) handleStats(w http.ResponseWriter, r *http.Request) {
	s.metricsMu.RLock()
	stats := map[string]interface{}{
		"total_connections":  s.totalConnections,
		"active_connections": s.activeConnections,
		"total_messages":     s.totalMessages,
		"channels":           s.hub.GetChannelCount(),
	}
	s.metricsMu.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"total_connections":%d,"active_connections":%d,"total_messages":%d,"channels":%d}`,
		stats["total_connections"], stats["active_connections"], stats["total_messages"], stats["channels"])
}

// registerConnection registers a new connection
func (s *Server) registerConnection(client *Client) {
	s.connectionsMu.Lock()
	s.connections[client.GetID()] = client
	s.connectionsMu.Unlock()

	s.ipMu.Lock()
	s.connectionsPerIP[client.GetIP()]++
	s.ipMu.Unlock()

	s.hub.register <- client
}

// unregisterConnection unregisters a connection
// nolint:unused // Reserved for future use
func (s *Server) unregisterConnection(client *Client) {
	s.connectionsMu.Lock()
	delete(s.connections, client.GetID())
	s.connectionsMu.Unlock()

	s.ipMu.Lock()
	s.connectionsPerIP[client.GetIP()]--
	if s.connectionsPerIP[client.GetIP()] <= 0 {
		delete(s.connectionsPerIP, client.GetIP())
	}
	s.ipMu.Unlock()

	s.metricsMu.Lock()
	s.activeConnections--
	s.metricsMu.Unlock()
}

// checkIPLimit checks if an IP has reached the connection limit
func (s *Server) checkIPLimit(ip string) bool {
	s.ipMu.RLock()
	defer s.ipMu.RUnlock()

	count := s.connectionsPerIP[ip]
	return count < s.config.MaxConnPerIP
}

// GetHub returns the hub
func (s *Server) GetHub() *Hub {
	return s.hub
}

// GetConnection returns a client by ID
func (s *Server) GetConnection(clientID string) *Client {
	s.connectionsMu.RLock()
	defer s.connectionsMu.RUnlock()
	return s.connections[clientID]
}

// GetActiveConnections returns the number of active connections
func (s *Server) GetActiveConnections() int64 {
	s.metricsMu.RLock()
	defer s.metricsMu.RUnlock()
	return s.activeConnections
}

// BroadcastTicker broadcasts a ticker update
func (s *Server) BroadcastTicker(ticker *TickerMessage) {
	s.hub.UpdateTicker(ticker.MarketID, ticker)
}

// BroadcastDepth broadcasts a depth update
func (s *Server) BroadcastDepth(depth *DepthMessage) {
	s.hub.UpdateDepth(depth.MarketID, depth)
}

// BroadcastTrade broadcasts a trade
func (s *Server) BroadcastTrade(trade *TradeMessage) {
	s.hub.BroadcastTrade(trade.MarketID, trade)
}

// BroadcastPosition broadcasts a position update to a user
func (s *Server) BroadcastPosition(userID string, position *PositionMessage) {
	s.hub.BroadcastPosition(userID, position)
}

// BroadcastOrder broadcasts an order update to a user
func (s *Server) BroadcastOrder(userID string, order *OrderMessage) {
	s.hub.BroadcastOrder(userID, order)
}

// getClientIP extracts the client IP from the request
func getClientIP(r *http.Request) string {
	// Check for forwarded headers
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// Take the first IP in the list
		for i := 0; i < len(xff); i++ {
			if xff[i] == ',' {
				return xff[:i]
			}
		}
		return xff
	}

	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}

	// Fall back to remote address
	ip := r.RemoteAddr
	// Remove port if present
	for i := len(ip) - 1; i >= 0; i-- {
		if ip[i] == ':' {
			return ip[:i]
		}
	}
	return ip
}
