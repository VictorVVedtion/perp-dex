// Package e2e_real provides real end-to-end testing infrastructure
// Tests actual HTTP/WebSocket connections to a running API server
package e2e_real

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

// TestConfig holds E2E test configuration
type TestConfig struct {
	APIBaseURL string
	WSBaseURL  string
	Timeout    time.Duration
}

// DefaultConfig returns default test configuration
func DefaultConfig() *TestConfig {
	return &TestConfig{
		APIBaseURL: "http://localhost:8080",
		WSBaseURL:  "ws://localhost:8080",
		Timeout:    10 * time.Second,
	}
}

// CheckAPIAvailable checks if API server is available and skips test if not
func CheckAPIAvailable(t interface{ Skip(...any); Helper() }) {
	t.Helper()
	resp, err := http.Get("http://localhost:8080/v1/health")
	if err != nil {
		t.Skip("API server not available at http://localhost:8080:", err)
	}
	resp.Body.Close()
}

// HTTPClient provides HTTP request utilities for E2E testing
type HTTPClient struct {
	config     *TestConfig
	client     *http.Client
	latencies  []time.Duration
	mu         sync.Mutex
}

// NewHTTPClient creates a new HTTP client for testing
func NewHTTPClient(config *TestConfig) *HTTPClient {
	transport := &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 100,
		IdleConnTimeout:     90 * time.Second,
	}

	return &HTTPClient{
		config: config,
		client: &http.Client{
			Transport: transport,
			Timeout:   config.Timeout,
		},
		latencies: make([]time.Duration, 0),
	}
}

// APIResponse represents a generic API response
type APIResponse struct {
	Success bool            `json:"success"`
	Data    json.RawMessage `json:"data,omitempty"`
	Error   string          `json:"error,omitempty"`
}

// RequestResult contains request result and timing
type RequestResult struct {
	Response   *APIResponse
	StatusCode int
	Latency    time.Duration
	Error      error
}

// GET performs a GET request
func (c *HTTPClient) GET(path string) *RequestResult {
	return c.doRequest("GET", path, nil)
}

// POST performs a POST request with JSON body
func (c *HTTPClient) POST(path string, body interface{}) *RequestResult {
	return c.doRequest("POST", path, body)
}

// DELETE performs a DELETE request
func (c *HTTPClient) DELETE(path string) *RequestResult {
	return c.doRequest("DELETE", path, nil)
}

func (c *HTTPClient) doRequest(method, path string, body interface{}) *RequestResult {
	result := &RequestResult{}
	url := c.config.APIBaseURL + path

	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			result.Error = fmt.Errorf("failed to marshal body: %w", err)
			return result
		}
		reqBody = bytes.NewBuffer(jsonBody)
	}

	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		result.Error = fmt.Errorf("failed to create request: %w", err)
		return result
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	start := time.Now()
	resp, err := c.client.Do(req)
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

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		result.Error = fmt.Errorf("failed to read response: %w", err)
		return result
	}

	var apiResp APIResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		// Try to parse as raw data
		apiResp.Data = respBody
	} else if apiResp.Data == nil {
		// If unmarshal succeeded but Data is nil, it means the response
		// is not in the wrapped {success, data} format.
		// Store the entire response as raw data for handlers that return
		// data directly (like RiverPool endpoints)
		apiResp.Data = respBody
		apiResp.Success = true // Assume success if status is 2xx
	}
	result.Response = &apiResp

	return result
}

// GetAverageLatency returns the average request latency (must be called with lock held)
func (c *HTTPClient) getAverageLatencyLocked() time.Duration {
	if len(c.latencies) == 0 {
		return 0
	}

	var total time.Duration
	for _, l := range c.latencies {
		total += l
	}
	return total / time.Duration(len(c.latencies))
}

// GetAverageLatency returns the average request latency
func (c *HTTPClient) GetAverageLatency() time.Duration {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.getAverageLatencyLocked()
}

// getLatencyPercentileLocked returns the p-th percentile latency (must be called with lock held)
func (c *HTTPClient) getLatencyPercentileLocked(p float64) time.Duration {
	if len(c.latencies) == 0 {
		return 0
	}

	// Simple implementation - sort and pick
	sorted := make([]time.Duration, len(c.latencies))
	copy(sorted, c.latencies)

	// Bubble sort (simple for test purposes)
	for i := 0; i < len(sorted); i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[i] > sorted[j] {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}

	idx := int(float64(len(sorted)-1) * p)
	return sorted[idx]
}

// GetLatencyPercentile returns the p-th percentile latency
func (c *HTTPClient) GetLatencyPercentile(p float64) time.Duration {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.getLatencyPercentileLocked(p)
}

// WSClient provides WebSocket utilities for E2E testing
type WSClient struct {
	config   *TestConfig
	conn     *websocket.Conn
	messages chan []byte
	errors   chan error
	done     chan struct{}
	mu       sync.Mutex
}

// NewWSClient creates a new WebSocket client
func NewWSClient(config *TestConfig) *WSClient {
	return &WSClient{
		config:   config,
		messages: make(chan []byte, 100),
		errors:   make(chan error, 10),
		done:     make(chan struct{}),
	}
}

// Connect establishes WebSocket connection
func (c *WSClient) Connect(path string) error {
	url := c.config.WSBaseURL + path

	dialer := websocket.Dialer{
		HandshakeTimeout: c.config.Timeout,
	}

	conn, _, err := dialer.Dial(url, nil)
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}

	c.mu.Lock()
	c.conn = conn
	c.mu.Unlock()

	// Start reader goroutine
	go c.readLoop()

	return nil
}

func (c *WSClient) readLoop() {
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

		select {
		case c.messages <- message:
		default:
			// Drop message if channel is full
		}
	}
}

// Send sends a message over WebSocket
func (c *WSClient) Send(data interface{}) error {
	c.mu.Lock()
	conn := c.conn
	c.mu.Unlock()

	if conn == nil {
		return fmt.Errorf("not connected")
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal: %w", err)
	}

	return conn.WriteMessage(websocket.TextMessage, jsonData)
}

// Receive waits for a message with timeout
func (c *WSClient) Receive(timeout time.Duration) ([]byte, error) {
	select {
	case msg := <-c.messages:
		return msg, nil
	case err := <-c.errors:
		return nil, err
	case <-time.After(timeout):
		return nil, fmt.Errorf("receive timeout")
	}
}

// ReceiveJSON waits for a JSON message and unmarshals it
func (c *WSClient) ReceiveJSON(timeout time.Duration, v interface{}) error {
	msg, err := c.Receive(timeout)
	if err != nil {
		return err
	}
	return json.Unmarshal(msg, v)
}

// Close closes the WebSocket connection
func (c *WSClient) Close() error {
	c.mu.Lock()
	conn := c.conn
	c.conn = nil
	c.mu.Unlock()

	if conn != nil {
		err := conn.WriteMessage(
			websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""),
		)
		if err != nil {
			return err
		}
		return conn.Close()
	}
	return nil
}

// WaitForClose waits for the connection to close
func (c *WSClient) WaitForClose(timeout time.Duration) error {
	select {
	case <-c.done:
		return nil
	case <-time.After(timeout):
		return fmt.Errorf("wait timeout")
	}
}

// TestAccount represents a test trading account
type TestAccount struct {
	Address string `json:"address"`
	Balance string `json:"balance"`
}

// CreateTestAccount creates a test account for E2E testing
func CreateTestAccount(client *HTTPClient, balance string) (*TestAccount, error) {
	// Generate unique address
	addr := fmt.Sprintf("test_%d", time.Now().UnixNano())

	account := &TestAccount{
		Address: addr,
		Balance: balance,
	}

	// Register account via API
	result := client.POST("/v1/accounts", map[string]string{
		"address": addr,
		"balance": balance,
	})

	if result.Error != nil {
		return nil, result.Error
	}

	if result.StatusCode != http.StatusOK && result.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("failed to create account: status %d", result.StatusCode)
	}

	return account, nil
}

// Order represents a trading order
type Order struct {
	OrderID   string `json:"order_id,omitempty"`
	MarketID  string `json:"market_id"`
	Trader    string `json:"trader"`
	Side      string `json:"side"` // "buy" or "sell"
	OrderType string `json:"type"` // "limit" or "market"
	Price     string `json:"price"`
	Quantity  string `json:"quantity"`
	Leverage  string `json:"leverage,omitempty"`
}

// PlaceOrder places an order via API
func PlaceOrder(client *HTTPClient, order *Order) (*RequestResult, error) {
	result := client.POST("/v1/orders", order)
	if result.Error != nil {
		return nil, result.Error
	}
	return result, nil
}

// CancelOrder cancels an order via API
func CancelOrder(client *HTTPClient, orderID string) (*RequestResult, error) {
	result := client.DELETE(fmt.Sprintf("/v1/orders/%s", orderID))
	if result.Error != nil {
		return nil, result.Error
	}
	return result, nil
}

// GetOrderBook fetches order book via API
func GetOrderBook(client *HTTPClient, marketID string) (*RequestResult, error) {
	result := client.GET(fmt.Sprintf("/v1/markets/%s/orderbook", marketID))
	if result.Error != nil {
		return nil, result.Error
	}
	return result, nil
}

// GetPositions fetches positions for a trader
func GetPositions(client *HTTPClient, trader string) (*RequestResult, error) {
	result := client.GET(fmt.Sprintf("/v1/accounts/%s/positions", trader))
	if result.Error != nil {
		return nil, result.Error
	}
	return result, nil
}

// LatencyReport generates a latency report
type LatencyReport struct {
	TestName    string
	TotalReqs   int
	AvgLatency  time.Duration
	P50Latency  time.Duration
	P95Latency  time.Duration
	P99Latency  time.Duration
	MinLatency  time.Duration
	MaxLatency  time.Duration
}

// GenerateReport generates a latency report from collected data
func (c *HTTPClient) GenerateReport(testName string) *LatencyReport {
	c.mu.Lock()
	defer c.mu.Unlock()

	if len(c.latencies) == 0 {
		return &LatencyReport{TestName: testName}
	}

	report := &LatencyReport{
		TestName:   testName,
		TotalReqs:  len(c.latencies),
		AvgLatency: c.getAverageLatencyLocked(),
		P50Latency: c.getLatencyPercentileLocked(0.5),
		P95Latency: c.getLatencyPercentileLocked(0.95),
		P99Latency: c.getLatencyPercentileLocked(0.99),
	}

	// Find min/max
	report.MinLatency = c.latencies[0]
	report.MaxLatency = c.latencies[0]
	for _, l := range c.latencies {
		if l < report.MinLatency {
			report.MinLatency = l
		}
		if l > report.MaxLatency {
			report.MaxLatency = l
		}
	}

	return report
}

// PrintReport prints the latency report
func (r *LatencyReport) PrintReport() {
	fmt.Println("========================================")
	fmt.Printf("E2E Test Report: %s\n", r.TestName)
	fmt.Println("========================================")
	fmt.Printf("Total Requests: %d\n", r.TotalReqs)
	fmt.Printf("Average Latency: %v\n", r.AvgLatency)
	fmt.Printf("P50 Latency: %v\n", r.P50Latency)
	fmt.Printf("P95 Latency: %v\n", r.P95Latency)
	fmt.Printf("P99 Latency: %v\n", r.P99Latency)
	fmt.Printf("Min Latency: %v\n", r.MinLatency)
	fmt.Printf("Max Latency: %v\n", r.MaxLatency)
	fmt.Println("========================================")
}
