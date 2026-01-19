package websocket

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// Hub maintains the set of active clients and broadcasts messages
type Hub struct {
	// Registered clients by channel
	clients    map[*Client]bool
	channels   map[string]map[*Client]bool // channel -> clients

	// Subscription management
	subscriptions map[string]map[*Client]bool // topic -> clients

	// Inbound messages from clients
	broadcast  chan []byte

	// Register/unregister requests
	register   chan *Client
	unregister chan *Client

	// Channel subscription requests
	subscribe   chan *SubscriptionRequest
	unsubscribe chan *SubscriptionRequest

	// Message buffers for different channels
	tickerBuffer  map[string]*TickerMessage
	depthBuffer   map[string]*DepthMessage

	// Mutex for thread-safe operations
	mu sync.RWMutex

	// Configuration
	config *HubConfig
}

// HubConfig contains hub configuration
type HubConfig struct {
	// Update intervals
	TickerInterval time.Duration // Default: 100ms
	DepthInterval  time.Duration // Default: 100ms
	TradesBuffer   int           // Number of trades to buffer

	// Connection limits
	MaxClientsPerIP    int
	MaxSubscriptions   int

	// Rate limiting
	MessageRateLimit   int // Messages per second per client
}

// DefaultHubConfig returns default hub configuration
func DefaultHubConfig() *HubConfig {
	return &HubConfig{
		TickerInterval:     100 * time.Millisecond,
		DepthInterval:      100 * time.Millisecond,
		TradesBuffer:       100,
		MaxClientsPerIP:    10,
		MaxSubscriptions:   50,
		MessageRateLimit:   100,
	}
}

// SubscriptionRequest represents a subscription request
type SubscriptionRequest struct {
	Client  *Client
	Channel string
	Action  string // "subscribe" or "unsubscribe"
}

// NewHub creates a new Hub
func NewHub(config *HubConfig) *Hub {
	if config == nil {
		config = DefaultHubConfig()
	}

	return &Hub{
		clients:       make(map[*Client]bool),
		channels:      make(map[string]map[*Client]bool),
		subscriptions: make(map[string]map[*Client]bool),
		broadcast:     make(chan []byte, 256),
		register:      make(chan *Client),
		unregister:    make(chan *Client),
		subscribe:     make(chan *SubscriptionRequest, 256),
		unsubscribe:   make(chan *SubscriptionRequest, 256),
		tickerBuffer:  make(map[string]*TickerMessage),
		depthBuffer:   make(map[string]*DepthMessage),
		config:        config,
	}
}

// Run starts the hub's main loop
func (h *Hub) Run() {
	// Start ticker broadcast
	tickerTicker := time.NewTicker(h.config.TickerInterval)
	depthTicker := time.NewTicker(h.config.DepthInterval)

	defer tickerTicker.Stop()
	defer depthTicker.Stop()

	for {
		select {
		case client := <-h.register:
			h.registerClient(client)

		case client := <-h.unregister:
			h.unregisterClient(client)

		case req := <-h.subscribe:
			h.handleSubscription(req)

		case req := <-h.unsubscribe:
			h.handleUnsubscription(req)

		case message := <-h.broadcast:
			h.broadcastMessage(message)

		case <-tickerTicker.C:
			h.broadcastTickers()

		case <-depthTicker.C:
			h.broadcastDepths()
		}
	}
}

// registerClient adds a new client
func (h *Hub) registerClient(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.clients[client] = true
}

// unregisterClient removes a client
func (h *Hub) unregisterClient(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if _, ok := h.clients[client]; ok {
		delete(h.clients, client)

		// Remove from all channels
		for channel, clients := range h.channels {
			delete(clients, client)
			if len(clients) == 0 {
				delete(h.channels, channel)
			}
		}

		// Remove from all subscriptions
		for topic := range h.subscriptions {
			delete(h.subscriptions[topic], client)
		}

		close(client.send)
	}
}

// handleSubscription handles a subscription request
func (h *Hub) handleSubscription(req *SubscriptionRequest) {
	h.mu.Lock()
	defer h.mu.Unlock()

	channel := req.Channel
	client := req.Client

	if _, ok := h.channels[channel]; !ok {
		h.channels[channel] = make(map[*Client]bool)
	}
	h.channels[channel][client] = true

	// Send subscription confirmation
	confirmation := &WSMessage{
		Type:    "subscribed",
		Channel: channel,
		Data:    nil,
	}
	data, _ := json.Marshal(confirmation)
	client.send <- data
}

// handleUnsubscription handles an unsubscription request
func (h *Hub) handleUnsubscription(req *SubscriptionRequest) {
	h.mu.Lock()
	defer h.mu.Unlock()

	channel := req.Channel
	client := req.Client

	if clients, ok := h.channels[channel]; ok {
		delete(clients, client)
		if len(clients) == 0 {
			delete(h.channels, channel)
		}
	}

	// Send unsubscription confirmation
	confirmation := &WSMessage{
		Type:    "unsubscribed",
		Channel: channel,
		Data:    nil,
	}
	data, _ := json.Marshal(confirmation)
	client.send <- data
}

// broadcastMessage sends a message to all clients in a channel
func (h *Hub) broadcastMessage(message []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for client := range h.clients {
		select {
		case client.send <- message:
		default:
			// Client buffer is full, skip
		}
	}
}

// BroadcastToChannel sends a message to all clients subscribed to a channel
func (h *Hub) BroadcastToChannel(channel string, message interface{}) {
	h.mu.RLock()
	clients, ok := h.channels[channel]
	if !ok {
		h.mu.RUnlock()
		return
	}

	// Make a copy of clients to avoid holding lock during send
	clientList := make([]*Client, 0, len(clients))
	for client := range clients {
		clientList = append(clientList, client)
	}
	h.mu.RUnlock()

	data, err := json.Marshal(message)
	if err != nil {
		return
	}

	for _, client := range clientList {
		select {
		case client.send <- data:
		default:
			// Client buffer is full, skip
		}
	}
}

// ============ Channel-specific broadcasts ============

// UpdateTicker updates the ticker buffer for a market
func (h *Hub) UpdateTicker(marketID string, ticker *TickerMessage) {
	h.mu.Lock()
	h.tickerBuffer[marketID] = ticker
	h.mu.Unlock()
}

// UpdateDepth updates the depth buffer for a market
func (h *Hub) UpdateDepth(marketID string, depth *DepthMessage) {
	h.mu.Lock()
	h.depthBuffer[marketID] = depth
	h.mu.Unlock()
}

// broadcastTickers broadcasts all ticker updates
func (h *Hub) broadcastTickers() {
	h.mu.RLock()
	tickers := make(map[string]*TickerMessage)
	for k, v := range h.tickerBuffer {
		tickers[k] = v
	}
	h.mu.RUnlock()

	for marketID, ticker := range tickers {
		channel := "ticker:" + marketID
		msg := &WSMessage{
			Type:    "ticker",
			Channel: channel,
			Data:    ticker,
		}
		h.BroadcastToChannel(channel, msg)
	}
}

// broadcastDepths broadcasts all depth updates
func (h *Hub) broadcastDepths() {
	h.mu.RLock()
	depths := make(map[string]*DepthMessage)
	for k, v := range h.depthBuffer {
		depths[k] = v
	}
	h.mu.RUnlock()

	for marketID, depth := range depths {
		channel := "depth:" + marketID
		msg := &WSMessage{
			Type:    "depth",
			Channel: channel,
			Data:    depth,
		}
		h.BroadcastToChannel(channel, msg)
	}
}

// BroadcastTrade broadcasts a trade to subscribers
func (h *Hub) BroadcastTrade(marketID string, trade *TradeMessage) {
	channel := "trades:" + marketID
	msg := &WSMessage{
		Type:    "trade",
		Channel: channel,
		Data:    trade,
	}
	h.BroadcastToChannel(channel, msg)
}

// BroadcastPosition broadcasts a position update to a specific user
func (h *Hub) BroadcastPosition(userID string, position *PositionMessage) {
	channel := "positions:" + userID
	msg := &WSMessage{
		Type:    "position",
		Channel: channel,
		Data:    position,
	}
	h.BroadcastToChannel(channel, msg)
}

// BroadcastOrder broadcasts an order update to a specific user
func (h *Hub) BroadcastOrder(userID string, order *OrderMessage) {
	channel := "orders:" + userID
	msg := &WSMessage{
		Type:    "order",
		Channel: channel,
		Data:    order,
	}
	h.BroadcastToChannel(channel, msg)
}

// ============ Message Types ============

// WSMessage represents a WebSocket message
type WSMessage struct {
	Type    string      `json:"type"`
	Channel string      `json:"channel"`
	Data    interface{} `json:"data,omitempty"`
}

// TickerMessage represents a ticker update
type TickerMessage struct {
	MarketID    string `json:"market_id"`
	MarkPrice   string `json:"mark_price"`
	IndexPrice  string `json:"index_price"`
	LastPrice   string `json:"last_price"`
	High24h     string `json:"high_24h"`
	Low24h      string `json:"low_24h"`
	Volume24h   string `json:"volume_24h"`
	Change24h   string `json:"change_24h"`
	FundingRate string `json:"funding_rate"`
	NextFunding int64  `json:"next_funding"`
	Timestamp   int64  `json:"timestamp"`
}

// DepthMessage represents orderbook depth
type DepthMessage struct {
	MarketID  string       `json:"market_id"`
	Bids      []PriceLevel `json:"bids"`
	Asks      []PriceLevel `json:"asks"`
	Timestamp int64        `json:"timestamp"`
}

// PriceLevel represents a price level in the orderbook
type PriceLevel struct {
	Price    string `json:"price"`
	Quantity string `json:"quantity"`
}

// TradeMessage represents a trade
type TradeMessage struct {
	TradeID   string `json:"trade_id"`
	MarketID  string `json:"market_id"`
	Price     string `json:"price"`
	Quantity  string `json:"quantity"`
	Side      string `json:"side"` // "buy" or "sell"
	Timestamp int64  `json:"timestamp"`
}

// PositionMessage represents a position update
type PositionMessage struct {
	Trader           string `json:"trader"`
	MarketID         string `json:"market_id"`
	Side             string `json:"side"`
	Size             string `json:"size"`
	EntryPrice       string `json:"entry_price"`
	MarkPrice        string `json:"mark_price"`
	UnrealizedPnL    string `json:"unrealized_pnl"`
	Margin           string `json:"margin"`
	Leverage         string `json:"leverage"`
	LiquidationPrice string `json:"liquidation_price"`
	Timestamp        int64  `json:"timestamp"`
}

// OrderMessage represents an order update
type OrderMessage struct {
	OrderID    string `json:"order_id"`
	MarketID   string `json:"market_id"`
	Trader     string `json:"trader"`
	Side       string `json:"side"`
	Type       string `json:"type"`
	Price      string `json:"price"`
	Size       string `json:"size"`
	FilledSize string `json:"filled_size"`
	Status     string `json:"status"`
	Timestamp  int64  `json:"timestamp"`
}

// GetClientCount returns the number of connected clients
func (h *Hub) GetClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

// GetChannelCount returns the number of active channels
func (h *Hub) GetChannelCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.channels)
}

// GetChannelClientCount returns the number of clients in a channel
func (h *Hub) GetChannelClientCount(channel string) int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	if clients, ok := h.channels[channel]; ok {
		return len(clients)
	}
	return 0
}

// ServeWS handles WebSocket upgrade requests
func (h *Hub) ServeWS(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	clientID := r.URL.Query().Get("client_id")
	if clientID == "" {
		clientID = generateID()
	}

	userID := r.URL.Query().Get("user_id")
	ip := getClientIPFromRequest(r)

	client := NewClient(h, conn, clientID, userID, ip)

	h.register <- client

	go client.writePump()
	go client.readPump()
}

// Helper function to get client IP
func getClientIPFromRequest(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
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
	ip := r.RemoteAddr
	for i := len(ip) - 1; i >= 0; i-- {
		if ip[i] == ':' {
			return ip[:i]
		}
	}
	return ip
}

// Generate a simple ID
func generateID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}
