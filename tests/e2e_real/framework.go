package e2e_real

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// ============================================================================
// E2E Test Framework - Real End-to-End Testing Infrastructure
// ============================================================================
// This framework provides:
// 1. API Server lifecycle management
// 2. HTTP client for REST API testing
// 3. WebSocket client for real-time testing
// 4. Test utilities and assertions
// ============================================================================

const (
	DefaultAPIURL     = "http://localhost:8080"
	DefaultWSURL      = "ws://localhost:8080/ws"
	DefaultTimeout    = 30 * time.Second
	DefaultRetryCount = 3
	DefaultRetryDelay = 500 * time.Millisecond
)

// TestConfig holds E2E test configuration
type TestConfig struct {
	APIURL        string
	WSURL         string
	Timeout       time.Duration
	RetryCount    int
	RetryDelay    time.Duration
	Verbose       bool
}

// DefaultTestConfig returns default test configuration
func DefaultTestConfig() *TestConfig {
	return &TestConfig{
		APIURL:     DefaultAPIURL,
		WSURL:      DefaultWSURL,
		Timeout:    DefaultTimeout,
		RetryCount: DefaultRetryCount,
		RetryDelay: DefaultRetryDelay,
		Verbose:    true,
	}
}

// E2ETestSuite provides the main test infrastructure
type E2ETestSuite struct {
	config     *TestConfig
	httpClient *http.Client
	t          *testing.T
}

// NewE2ETestSuite creates a new E2E test suite
func NewE2ETestSuite(t *testing.T, config *TestConfig) *E2ETestSuite {
	if config == nil {
		config = DefaultTestConfig()
	}

	return &E2ETestSuite{
		config: config,
		httpClient: &http.Client{
			Timeout: config.Timeout,
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 100,
				IdleConnTimeout:     90 * time.Second,
			},
		},
		t: t,
	}
}

// WaitForServer waits for the API server to be ready
func (s *E2ETestSuite) WaitForServer(timeout time.Duration) error {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		resp, err := s.httpClient.Get(s.config.APIURL + "/health")
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				s.log("Server is ready")
				return nil
			}
		}
		time.Sleep(500 * time.Millisecond)
	}

	return fmt.Errorf("server not ready within %v", timeout)
}

// ============================================================================
// HTTP Client Methods
// ============================================================================

// APIResponse represents a generic API response
type APIResponse struct {
	StatusCode int
	Body       []byte
	Headers    http.Header
	Latency    time.Duration
}

// DoRequest performs an HTTP request with retry logic
func (s *E2ETestSuite) DoRequest(method, path string, body interface{}, headers map[string]string) (*APIResponse, error) {
	var lastErr error

	for i := 0; i < s.config.RetryCount; i++ {
		resp, err := s.doSingleRequest(method, path, body, headers)
		if err == nil {
			return resp, nil
		}
		lastErr = err
		time.Sleep(s.config.RetryDelay)
	}

	return nil, fmt.Errorf("request failed after %d retries: %v", s.config.RetryCount, lastErr)
}

func (s *E2ETestSuite) doSingleRequest(method, path string, body interface{}, headers map[string]string) (*APIResponse, error) {
	var reqBody io.Reader

	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal body: %v", err)
		}
		reqBody = bytes.NewReader(jsonData)
	}

	req, err := http.NewRequest(method, s.config.APIURL+path, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	start := time.Now()
	resp, err := s.httpClient.Do(req)
	latency := time.Since(start)

	if err != nil {
		return nil, fmt.Errorf("request failed: %v", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %v", err)
	}

	return &APIResponse{
		StatusCode: resp.StatusCode,
		Body:       respBody,
		Headers:    resp.Header,
		Latency:    latency,
	}, nil
}

// GET performs a GET request
func (s *E2ETestSuite) GET(path string, headers map[string]string) (*APIResponse, error) {
	return s.DoRequest(http.MethodGet, path, nil, headers)
}

// POST performs a POST request
func (s *E2ETestSuite) POST(path string, body interface{}, headers map[string]string) (*APIResponse, error) {
	return s.DoRequest(http.MethodPost, path, body, headers)
}

// PUT performs a PUT request
func (s *E2ETestSuite) PUT(path string, body interface{}, headers map[string]string) (*APIResponse, error) {
	return s.DoRequest(http.MethodPut, path, body, headers)
}

// DELETE performs a DELETE request
func (s *E2ETestSuite) DELETE(path string, headers map[string]string) (*APIResponse, error) {
	return s.DoRequest(http.MethodDelete, path, nil, headers)
}

// ============================================================================
// WebSocket Client
// ============================================================================

// WSClient provides WebSocket testing capabilities
type WSClient struct {
	conn       *websocket.Conn
	url        string
	messages   chan []byte
	errors     chan error
	done       chan struct{}
	mu         sync.Mutex
	isClosing  bool
}

// NewWSClient creates a new WebSocket client
func (s *E2ETestSuite) NewWSClient() (*WSClient, error) {
	conn, _, err := websocket.DefaultDialer.Dial(s.config.WSURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to WebSocket: %v", err)
	}

	client := &WSClient{
		conn:     conn,
		url:      s.config.WSURL,
		messages: make(chan []byte, 100),
		errors:   make(chan error, 10),
		done:     make(chan struct{}),
	}

	go client.readLoop()

	return client, nil
}

func (ws *WSClient) readLoop() {
	defer close(ws.done)

	for {
		_, message, err := ws.conn.ReadMessage()
		if err != nil {
			ws.mu.Lock()
			closing := ws.isClosing
			ws.mu.Unlock()

			if !closing {
				ws.errors <- err
			}
			return
		}

		select {
		case ws.messages <- message:
		default:
			// Drop message if channel is full
		}
	}
}

// Subscribe sends a subscription message
func (ws *WSClient) Subscribe(channel string, params map[string]interface{}) error {
	msg := map[string]interface{}{
		"method": "subscribe",
		"subscription": map[string]interface{}{
			"type": channel,
		},
	}

	for k, v := range params {
		msg["subscription"].(map[string]interface{})[k] = v
	}

	return ws.conn.WriteJSON(msg)
}

// Unsubscribe sends an unsubscription message
func (ws *WSClient) Unsubscribe(channel string) error {
	msg := map[string]interface{}{
		"method": "unsubscribe",
		"subscription": map[string]interface{}{
			"type": channel,
		},
	}

	return ws.conn.WriteJSON(msg)
}

// WaitForMessage waits for a message matching the predicate
func (ws *WSClient) WaitForMessage(timeout time.Duration, predicate func([]byte) bool) ([]byte, error) {
	deadline := time.After(timeout)

	for {
		select {
		case msg := <-ws.messages:
			if predicate == nil || predicate(msg) {
				return msg, nil
			}
		case err := <-ws.errors:
			return nil, err
		case <-deadline:
			return nil, fmt.Errorf("timeout waiting for message")
		}
	}
}

// CollectMessages collects messages for a duration
func (ws *WSClient) CollectMessages(duration time.Duration) [][]byte {
	var messages [][]byte
	deadline := time.After(duration)

	for {
		select {
		case msg := <-ws.messages:
			messages = append(messages, msg)
		case <-deadline:
			return messages
		}
	}
}

// Close closes the WebSocket connection
func (ws *WSClient) Close() error {
	ws.mu.Lock()
	ws.isClosing = true
	ws.mu.Unlock()

	return ws.conn.Close()
}

// ============================================================================
// API Data Types
// ============================================================================

// PlaceOrderRequest represents an order placement request
type PlaceOrderRequest struct {
	MarketID  string `json:"market_id"`
	Side      string `json:"side"`
	Type      string `json:"type"`
	Price     string `json:"price"`
	Quantity  string `json:"quantity"`
	Trader    string `json:"trader"`
	TimeInForce string `json:"time_in_force,omitempty"`
}

// OrderResponse represents an order response
type OrderResponse struct {
	OrderID   string    `json:"order_id"`
	MarketID  string    `json:"market_id"`
	Trader    string    `json:"trader"`
	Side      string    `json:"side"`
	Type      string    `json:"type"`
	Price     string    `json:"price"`
	Quantity  string    `json:"quantity"`
	FilledQty string    `json:"filled_qty"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

// PositionResponse represents a position response
type PositionResponse struct {
	PositionID     string `json:"position_id"`
	MarketID       string `json:"market_id"`
	Trader         string `json:"trader"`
	Side           string `json:"side"`
	Size           string `json:"size"`
	EntryPrice     string `json:"entry_price"`
	MarkPrice      string `json:"mark_price"`
	UnrealizedPnL  string `json:"unrealized_pnl"`
	RealizedPnL    string `json:"realized_pnl"`
	Margin         string `json:"margin"`
	Leverage       string `json:"leverage"`
	LiquidationPrice string `json:"liquidation_price"`
}

// AccountResponse represents an account response
type AccountResponse struct {
	Address        string `json:"address"`
	Balance        string `json:"balance"`
	AvailableBalance string `json:"available_balance"`
	MarginUsed     string `json:"margin_used"`
	UnrealizedPnL  string `json:"unrealized_pnl"`
}

// DepositRequest represents a deposit request
type DepositRequest struct {
	Amount string `json:"amount"`
}

// WithdrawRequest represents a withdrawal request
type WithdrawRequest struct {
	Amount string `json:"amount"`
}

// ============================================================================
// Test Utilities
// ============================================================================

// TestUser represents a test user
type TestUser struct {
	Address string
	suite   *E2ETestSuite
}

// NewTestUser creates a new test user
func (s *E2ETestSuite) NewTestUser(address string) *TestUser {
	return &TestUser{
		Address: address,
		suite:   s,
	}
}

// Headers returns the standard headers for this user
func (u *TestUser) Headers() map[string]string {
	return map[string]string{
		"X-Trader-Address": u.Address,
	}
}

// PlaceOrder places an order for this user
func (u *TestUser) PlaceOrder(req *PlaceOrderRequest) (*OrderResponse, error) {
	req.Trader = u.Address

	resp, err := u.suite.POST("/v1/orders", req, u.Headers())
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, string(resp.Body))
	}

	var order OrderResponse
	if err := json.Unmarshal(resp.Body, &order); err != nil {
		return nil, fmt.Errorf("failed to parse order response: %v", err)
	}

	return &order, nil
}

// CancelOrder cancels an order
func (u *TestUser) CancelOrder(orderID string) error {
	resp, err := u.suite.DELETE("/v1/orders/"+orderID, u.Headers())
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, string(resp.Body))
	}

	return nil
}

// GetPositions gets all positions for this user
func (u *TestUser) GetPositions() ([]PositionResponse, error) {
	resp, err := u.suite.GET("/v1/positions", u.Headers())
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var result struct {
		Positions []PositionResponse `json:"positions"`
	}
	if err := json.Unmarshal(resp.Body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse positions: %v", err)
	}

	return result.Positions, nil
}

// GetAccount gets account information
func (u *TestUser) GetAccount() (*AccountResponse, error) {
	resp, err := u.suite.GET("/v1/account", u.Headers())
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var account AccountResponse
	if err := json.Unmarshal(resp.Body, &account); err != nil {
		return nil, fmt.Errorf("failed to parse account: %v", err)
	}

	return &account, nil
}

// Deposit deposits funds
func (u *TestUser) Deposit(amount string) error {
	req := DepositRequest{Amount: amount}
	resp, err := u.suite.POST("/v1/account/deposit", req, u.Headers())
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, string(resp.Body))
	}

	return nil
}

// Withdraw withdraws funds
func (u *TestUser) Withdraw(amount string) error {
	req := WithdrawRequest{Amount: amount}
	resp, err := u.suite.POST("/v1/account/withdraw", req, u.Headers())
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, string(resp.Body))
	}

	return nil
}

// ============================================================================
// Assertion Helpers
// ============================================================================

// AssertStatusCode asserts the response status code
func (s *E2ETestSuite) AssertStatusCode(resp *APIResponse, expected int) {
	if resp.StatusCode != expected {
		s.t.Errorf("Expected status code %d, got %d. Body: %s", expected, resp.StatusCode, string(resp.Body))
	}
}

// AssertNoError fails the test if there's an error
func (s *E2ETestSuite) AssertNoError(err error, msg string) {
	if err != nil {
		s.t.Fatalf("%s: %v", msg, err)
	}
}

// AssertEqual asserts two values are equal
func (s *E2ETestSuite) AssertEqual(expected, actual interface{}, msg string) {
	if expected != actual {
		s.t.Errorf("%s: expected %v, got %v", msg, expected, actual)
	}
}

// AssertNotEmpty asserts a string is not empty
func (s *E2ETestSuite) AssertNotEmpty(value, name string) {
	if value == "" {
		s.t.Errorf("%s should not be empty", name)
	}
}

// ============================================================================
// Logging
// ============================================================================

func (s *E2ETestSuite) log(format string, args ...interface{}) {
	if s.config.Verbose {
		s.t.Logf("[E2E] "+format, args...)
	}
}

// ============================================================================
// Test Results
// ============================================================================

// TestResult holds the result of a single test
type TestResult struct {
	Name     string        `json:"name"`
	Passed   bool          `json:"passed"`
	Duration time.Duration `json:"duration"`
	Error    string        `json:"error,omitempty"`
}

// TestReport holds the complete test report
type TestReport struct {
	StartTime    time.Time     `json:"start_time"`
	EndTime      time.Time     `json:"end_time"`
	TotalTests   int           `json:"total_tests"`
	PassedTests  int           `json:"passed_tests"`
	FailedTests  int           `json:"failed_tests"`
	Results      []TestResult  `json:"results"`
	Latencies    LatencyStats  `json:"latencies"`
}

// LatencyStats holds latency statistics
type LatencyStats struct {
	OrderPlacement LatencyMetrics `json:"order_placement"`
	OrderCancel    LatencyMetrics `json:"order_cancel"`
	GetPositions   LatencyMetrics `json:"get_positions"`
	GetAccount     LatencyMetrics `json:"get_account"`
	WebSocket      LatencyMetrics `json:"websocket"`
}

// LatencyMetrics holds latency metrics
type LatencyMetrics struct {
	Min     time.Duration `json:"min"`
	Max     time.Duration `json:"max"`
	Avg     time.Duration `json:"avg"`
	P50     time.Duration `json:"p50"`
	P95     time.Duration `json:"p95"`
	P99     time.Duration `json:"p99"`
	Count   int           `json:"count"`
}

// ============================================================================
// Server Management (for tests that start their own server)
// ============================================================================

// ServerManager manages API server lifecycle
type ServerManager struct {
	cmd    interface{} // *exec.Cmd when implemented
	ctx    context.Context
	cancel context.CancelFunc
}

// StartServer starts the API server
// Note: In a real implementation, this would start the actual server process
func StartServer(port int) (*ServerManager, error) {
	ctx, cancel := context.WithCancel(context.Background())

	// In a real implementation, this would exec the server binary:
	// cmd := exec.CommandContext(ctx, "./build/perpdexd", "start", "--api.enable", fmt.Sprintf("--api.port=%d", port))
	// cmd.Start()

	return &ServerManager{
		ctx:    ctx,
		cancel: cancel,
	}, nil
}

// StopServer stops the API server
func (sm *ServerManager) StopServer() error {
	sm.cancel()
	return nil
}
