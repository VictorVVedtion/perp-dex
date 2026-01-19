package websocket

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	// Time allowed to write a message to the peer
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer
	pongWait = 60 * time.Second

	// Send pings to peer with this period (must be less than pongWait)
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer
	maxMessageSize = 4096

	// Size of the send buffer
	sendBufferSize = 256
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// TODO: Implement proper origin checking for production
		return true
	},
}

// Client represents a WebSocket client connection
type Client struct {
	hub  *Hub
	conn *websocket.Conn
	send chan []byte

	// Client identification
	id       string
	userID   string // Empty for anonymous clients
	ip       string

	// Subscriptions
	subscriptions map[string]bool
	subMu         sync.RWMutex

	// Rate limiting
	messageCount int
	lastReset    time.Time
	rateMu       sync.Mutex

	// Connection stats
	connectedAt   time.Time
	lastMessageAt time.Time
}

// ClientMessage represents a message from a client
type ClientMessage struct {
	Action  string          `json:"action"`  // "subscribe", "unsubscribe", "ping"
	Channel string          `json:"channel"` // Channel to subscribe/unsubscribe
	Data    json.RawMessage `json:"data,omitempty"`
}

// NewClient creates a new Client
func NewClient(hub *Hub, conn *websocket.Conn, id, userID, ip string) *Client {
	return &Client{
		hub:           hub,
		conn:          conn,
		send:          make(chan []byte, sendBufferSize),
		id:            id,
		userID:        userID,
		ip:            ip,
		subscriptions: make(map[string]bool),
		connectedAt:   time.Now(),
		lastReset:     time.Now(),
	}
}

// readPump pumps messages from the WebSocket connection to the hub
func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	_ = c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		_ = c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("websocket error: %v", err)
			}
			break
		}

		c.lastMessageAt = time.Now()

		// Rate limiting check
		if !c.checkRateLimit() {
			c.sendError("rate_limit_exceeded", "Too many messages, please slow down")
			continue
		}

		// Parse and handle message
		var msg ClientMessage
		if err := json.Unmarshal(message, &msg); err != nil {
			c.sendError("invalid_message", "Failed to parse message")
			continue
		}

		c.handleMessage(&msg)
	}
}

// writePump pumps messages from the hub to the WebSocket connection
func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// The hub closed the channel
				_ = c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			_, _ = w.Write(message)

			// Add queued messages to the current WebSocket message
			n := len(c.send)
			for i := 0; i < n; i++ {
				_, _ = w.Write([]byte{'\n'})
				_, _ = w.Write(<-c.send)
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// handleMessage handles incoming messages from the client
func (c *Client) handleMessage(msg *ClientMessage) {
	switch msg.Action {
	case "subscribe":
		c.handleSubscribe(msg.Channel)
	case "unsubscribe":
		c.handleUnsubscribe(msg.Channel)
	case "ping":
		c.handlePing()
	case "auth":
		c.handleAuth(msg.Data)
	default:
		c.sendError("unknown_action", "Unknown action: "+msg.Action)
	}
}

// handleSubscribe handles a subscription request
func (c *Client) handleSubscribe(channel string) {
	if channel == "" {
		c.sendError("invalid_channel", "Channel cannot be empty")
		return
	}

	// Check subscription limit
	c.subMu.Lock()
	if len(c.subscriptions) >= c.hub.config.MaxSubscriptions {
		c.subMu.Unlock()
		c.sendError("subscription_limit", "Maximum subscription limit reached")
		return
	}
	c.subscriptions[channel] = true
	c.subMu.Unlock()

	// Validate channel access
	if !c.canAccessChannel(channel) {
		c.sendError("unauthorized", "Not authorized to access channel: "+channel)
		return
	}

	c.hub.subscribe <- &SubscriptionRequest{
		Client:  c,
		Channel: channel,
		Action:  "subscribe",
	}
}

// handleUnsubscribe handles an unsubscription request
func (c *Client) handleUnsubscribe(channel string) {
	c.subMu.Lock()
	delete(c.subscriptions, channel)
	c.subMu.Unlock()

	c.hub.unsubscribe <- &SubscriptionRequest{
		Client:  c,
		Channel: channel,
		Action:  "unsubscribe",
	}
}

// handlePing handles a ping request
func (c *Client) handlePing() {
	response := &WSMessage{
		Type: "pong",
		Data: map[string]interface{}{
			"timestamp": time.Now().UnixMilli(),
		},
	}
	data, _ := json.Marshal(response)
	c.send <- data
}

// handleAuth handles an authentication request
func (c *Client) handleAuth(data json.RawMessage) {
	// Parse auth data
	var authData struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(data, &authData); err != nil {
		c.sendError("invalid_auth", "Invalid auth data")
		return
	}

	// TODO: Validate token and extract user ID
	// For now, just acknowledge
	if authData.Token != "" {
		c.userID = "authenticated_user" // Replace with actual user ID from token
	}

	response := &WSMessage{
		Type: "authenticated",
		Data: map[string]interface{}{
			"user_id": c.userID,
		},
	}
	data, _ = json.Marshal(response)
	c.send <- data
}

// canAccessChannel checks if the client can access a channel
func (c *Client) canAccessChannel(channel string) bool {
	// Public channels
	publicPrefixes := []string{"ticker:", "depth:", "trades:"}
	for _, prefix := range publicPrefixes {
		if len(channel) >= len(prefix) && channel[:len(prefix)] == prefix {
			return true
		}
	}

	// Private channels require authentication
	privatePrefixes := []string{"positions:", "orders:"}
	for _, prefix := range privatePrefixes {
		if len(channel) >= len(prefix) && channel[:len(prefix)] == prefix {
			// Check if user is authenticated
			if c.userID == "" {
				return false
			}
			// Check if channel belongs to user
			expectedChannel := prefix + c.userID
			return channel == expectedChannel
		}
	}

	return false
}

// checkRateLimit checks if the client is within rate limits
func (c *Client) checkRateLimit() bool {
	c.rateMu.Lock()
	defer c.rateMu.Unlock()

	now := time.Now()
	if now.Sub(c.lastReset) >= time.Second {
		c.messageCount = 0
		c.lastReset = now
	}

	c.messageCount++
	return c.messageCount <= c.hub.config.MessageRateLimit
}

// sendError sends an error message to the client
func (c *Client) sendError(code, message string) {
	response := &WSMessage{
		Type: "error",
		Data: map[string]string{
			"code":    code,
			"message": message,
		},
	}
	data, _ := json.Marshal(response)
	c.send <- data
}

// Send sends a message to the client
func (c *Client) Send(message []byte) {
	select {
	case c.send <- message:
	default:
		// Buffer is full, message dropped
	}
}

// GetID returns the client ID
func (c *Client) GetID() string {
	return c.id
}

// GetUserID returns the user ID
func (c *Client) GetUserID() string {
	return c.userID
}

// GetIP returns the client IP
func (c *Client) GetIP() string {
	return c.ip
}

// IsAuthenticated returns whether the client is authenticated
func (c *Client) IsAuthenticated() bool {
	return c.userID != ""
}

// GetSubscriptions returns the client's subscriptions
func (c *Client) GetSubscriptions() []string {
	c.subMu.RLock()
	defer c.subMu.RUnlock()

	subs := make([]string, 0, len(c.subscriptions))
	for sub := range c.subscriptions {
		subs = append(subs, sub)
	}
	return subs
}

// GetConnectionDuration returns how long the client has been connected
func (c *Client) GetConnectionDuration() time.Duration {
	return time.Since(c.connectedAt)
}
