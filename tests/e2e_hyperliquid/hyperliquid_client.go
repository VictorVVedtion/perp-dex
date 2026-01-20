// Package e2e_hyperliquid provides real E2E tests against Hyperliquid mainnet API
// NO MOCK - Direct connection to https://api.hyperliquid.xyz
package e2e_hyperliquid

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	// Hyperliquid Mainnet API endpoints
	HyperliquidRESTURL = "https://api.hyperliquid.xyz/info"
	HyperliquidWSURL   = "wss://api.hyperliquid.xyz/ws"
)

// HyperliquidClient provides HTTP client for Hyperliquid API
type HyperliquidClient struct {
	httpClient *http.Client
	baseURL    string
	latencies  []time.Duration
	mu         sync.Mutex
}

// NewHyperliquidClient creates a new client for Hyperliquid mainnet
func NewHyperliquidClient() *HyperliquidClient {
	transport := &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 100,
		IdleConnTimeout:     90 * time.Second,
	}

	return &HyperliquidClient{
		httpClient: &http.Client{
			Transport: transport,
			Timeout:   30 * time.Second,
		},
		baseURL:   HyperliquidRESTURL,
		latencies: make([]time.Duration, 0),
	}
}

// APIResult contains request result and timing
type APIResult struct {
	Data       json.RawMessage
	StatusCode int
	Latency    time.Duration
	Error      error
}

// Post sends a POST request to Hyperliquid API
func (c *HyperliquidClient) Post(request interface{}) *APIResult {
	result := &APIResult{}

	jsonBody, err := json.Marshal(request)
	if err != nil {
		result.Error = fmt.Errorf("failed to marshal request: %w", err)
		return result
	}

	req, err := http.NewRequest("POST", c.baseURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		result.Error = fmt.Errorf("failed to create request: %w", err)
		return result
	}

	req.Header.Set("Content-Type", "application/json")

	start := time.Now()
	resp, err := c.httpClient.Do(req)
	result.Latency = time.Since(start)

	c.mu.Lock()
	c.latencies = append(c.latencies, result.Latency)
	c.mu.Unlock()

	if err != nil {
		result.Error = fmt.Errorf("request failed: %w", err)
		return result
	}
	defer resp.Body.Close()

	result.StatusCode = resp.StatusCode

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		result.Error = fmt.Errorf("failed to read response: %w", err)
		return result
	}

	result.Data = body
	return result
}

// GetMeta fetches exchange metadata (markets, assets)
func (c *HyperliquidClient) GetMeta() *APIResult {
	return c.Post(map[string]string{"type": "meta"})
}

// GetAllMids fetches all mid prices
func (c *HyperliquidClient) GetAllMids() *APIResult {
	return c.Post(map[string]string{"type": "allMids"})
}

// GetL2Book fetches order book for a coin
func (c *HyperliquidClient) GetL2Book(coin string) *APIResult {
	return c.Post(map[string]interface{}{
		"type": "l2Book",
		"coin": coin,
	})
}

// GetRecentTrades fetches recent trades for a coin
func (c *HyperliquidClient) GetRecentTrades(coin string) *APIResult {
	return c.Post(map[string]interface{}{
		"type": "recentTrades",
		"coin": coin,
	})
}

// GetCandleSnapshot fetches candle data
func (c *HyperliquidClient) GetCandleSnapshot(coin string, interval string, startTime, endTime int64) *APIResult {
	return c.Post(map[string]interface{}{
		"type":      "candleSnapshot",
		"req": map[string]interface{}{
			"coin":      coin,
			"interval":  interval,
			"startTime": startTime,
			"endTime":   endTime,
		},
	})
}

// GetFundingHistory fetches funding rate history
func (c *HyperliquidClient) GetFundingHistory(coin string, startTime int64) *APIResult {
	return c.Post(map[string]interface{}{
		"type":      "fundingHistory",
		"coin":      coin,
		"startTime": startTime,
	})
}

// GetUserState fetches user account state
func (c *HyperliquidClient) GetUserState(address string) *APIResult {
	return c.Post(map[string]interface{}{
		"type": "clearinghouseState",
		"user": address,
	})
}

// GetOpenOrders fetches user's open orders
func (c *HyperliquidClient) GetOpenOrders(address string) *APIResult {
	return c.Post(map[string]interface{}{
		"type": "openOrders",
		"user": address,
	})
}

// GetUserFills fetches user's trade fills
func (c *HyperliquidClient) GetUserFills(address string) *APIResult {
	return c.Post(map[string]interface{}{
		"type": "userFills",
		"user": address,
	})
}

// GetLatencyStats returns latency statistics
func (c *HyperliquidClient) GetLatencyStats() *LatencyStats {
	c.mu.Lock()
	defer c.mu.Unlock()

	if len(c.latencies) == 0 {
		return &LatencyStats{}
	}

	stats := &LatencyStats{
		Count: len(c.latencies),
	}

	var total time.Duration
	stats.Min = c.latencies[0]
	stats.Max = c.latencies[0]

	for _, l := range c.latencies {
		total += l
		if l < stats.Min {
			stats.Min = l
		}
		if l > stats.Max {
			stats.Max = l
		}
	}

	stats.Avg = total / time.Duration(len(c.latencies))

	// Calculate percentiles
	sorted := make([]time.Duration, len(c.latencies))
	copy(sorted, c.latencies)
	for i := 0; i < len(sorted); i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[i] > sorted[j] {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}

	stats.P50 = sorted[int(float64(len(sorted)-1)*0.5)]
	stats.P95 = sorted[int(float64(len(sorted)-1)*0.95)]
	stats.P99 = sorted[int(float64(len(sorted)-1)*0.99)]

	return stats
}

// LatencyStats contains latency statistics
type LatencyStats struct {
	Count int
	Avg   time.Duration
	Min   time.Duration
	Max   time.Duration
	P50   time.Duration
	P95   time.Duration
	P99   time.Duration
}

// PrintStats prints latency statistics
func (s *LatencyStats) PrintStats(name string) {
	fmt.Println("========================================")
	fmt.Printf("Hyperliquid API Test: %s\n", name)
	fmt.Println("========================================")
	fmt.Printf("Total Requests: %d\n", s.Count)
	fmt.Printf("Average Latency: %v\n", s.Avg)
	fmt.Printf("P50 Latency: %v\n", s.P50)
	fmt.Printf("P95 Latency: %v\n", s.P95)
	fmt.Printf("P99 Latency: %v\n", s.P99)
	fmt.Printf("Min Latency: %v\n", s.Min)
	fmt.Printf("Max Latency: %v\n", s.Max)
	fmt.Println("========================================")
}

// HyperliquidWSClient provides WebSocket client for Hyperliquid
type HyperliquidWSClient struct {
	conn       *websocket.Conn
	messages   chan []byte
	errors     chan error
	done       chan struct{}
	mu         sync.Mutex
	latencies  []time.Duration
	msgCount   int64
}

// NewHyperliquidWSClient creates a new WebSocket client
func NewHyperliquidWSClient() *HyperliquidWSClient {
	return &HyperliquidWSClient{
		messages: make(chan []byte, 1000),
		errors:   make(chan error, 10),
		done:     make(chan struct{}),
	}
}

// Connect establishes WebSocket connection to Hyperliquid
func (c *HyperliquidWSClient) Connect() error {
	dialer := websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
	}

	conn, _, err := dialer.Dial(HyperliquidWSURL, nil)
	if err != nil {
		return fmt.Errorf("failed to connect to Hyperliquid WS: %w", err)
	}

	c.mu.Lock()
	c.conn = conn
	c.mu.Unlock()

	go c.readLoop()

	return nil
}

func (c *HyperliquidWSClient) readLoop() {
	defer close(c.done)

	for {
		c.mu.Lock()
		conn := c.conn
		c.mu.Unlock()

		if conn == nil {
			return
		}

		_, message, err := conn.ReadMessage()
		if err != nil {
			if !websocket.IsCloseError(err, websocket.CloseNormalClosure) {
				select {
				case c.errors <- err:
				default:
				}
			}
			return
		}

		c.mu.Lock()
		c.msgCount++
		c.mu.Unlock()

		select {
		case c.messages <- message:
		default:
			// Drop if channel full
		}
	}
}

// Subscribe subscribes to a channel
func (c *HyperliquidWSClient) Subscribe(subscription map[string]interface{}) error {
	c.mu.Lock()
	conn := c.conn
	c.mu.Unlock()

	if conn == nil {
		return fmt.Errorf("not connected")
	}

	msg := map[string]interface{}{
		"method":       "subscribe",
		"subscription": subscription,
	}

	return conn.WriteJSON(msg)
}

// SubscribeAllMids subscribes to all mid prices
func (c *HyperliquidWSClient) SubscribeAllMids() error {
	return c.Subscribe(map[string]interface{}{
		"type": "allMids",
	})
}

// SubscribeL2Book subscribes to order book updates
func (c *HyperliquidWSClient) SubscribeL2Book(coin string) error {
	return c.Subscribe(map[string]interface{}{
		"type": "l2Book",
		"coin": coin,
	})
}

// SubscribeTrades subscribes to trade updates
func (c *HyperliquidWSClient) SubscribeTrades(coin string) error {
	return c.Subscribe(map[string]interface{}{
		"type": "trades",
		"coin": coin,
	})
}

// SubscribeCandle subscribes to candle updates
func (c *HyperliquidWSClient) SubscribeCandle(coin string, interval string) error {
	return c.Subscribe(map[string]interface{}{
		"type":     "candle",
		"coin":     coin,
		"interval": interval,
	})
}

// Receive waits for a message with timeout
func (c *HyperliquidWSClient) Receive(timeout time.Duration) ([]byte, error) {
	select {
	case msg := <-c.messages:
		return msg, nil
	case err := <-c.errors:
		return nil, err
	case <-time.After(timeout):
		return nil, fmt.Errorf("receive timeout")
	}
}

// GetMessageCount returns total messages received
func (c *HyperliquidWSClient) GetMessageCount() int64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.msgCount
}

// Close closes the WebSocket connection
func (c *HyperliquidWSClient) Close() error {
	c.mu.Lock()
	conn := c.conn
	c.conn = nil
	c.mu.Unlock()

	if conn != nil {
		conn.WriteMessage(
			websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""),
		)
		return conn.Close()
	}
	return nil
}

// Data structures for Hyperliquid responses

// Meta represents exchange metadata
type Meta struct {
	Universe []AssetInfo `json:"universe"`
}

// AssetInfo represents a trading pair
type AssetInfo struct {
	Name       string `json:"name"`
	SzDecimals int    `json:"szDecimals"`
}

// L2Book represents order book data
type L2Book struct {
	Coin   string      `json:"coin"`
	Levels [][]L2Level `json:"levels"`
	Time   int64       `json:"time"`
}

// L2Level represents a price level
type L2Level struct {
	Px string `json:"px"`
	Sz string `json:"sz"`
	N  int    `json:"n"`
}

// Trade represents a trade
type Trade struct {
	Coin  string `json:"coin"`
	Side  string `json:"side"`
	Px    string `json:"px"`
	Sz    string `json:"sz"`
	Time  int64  `json:"time"`
	Hash  string `json:"hash"`
}

// Candle represents OHLCV data
type Candle struct {
	T int64  `json:"t"` // timestamp
	O string `json:"o"` // open
	H string `json:"h"` // high
	L string `json:"l"` // low
	C string `json:"c"` // close
	V string `json:"v"` // volume
}

// FundingRate represents funding rate data
type FundingRate struct {
	Coin        string `json:"coin"`
	FundingRate string `json:"fundingRate"`
	Premium     string `json:"premium"`
	Time        int64  `json:"time"`
}
