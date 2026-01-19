package matcher

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/openalpha/perp-dex/x/orderbook/types"
)

// TxSubmitter defines the interface for submitting transactions to the chain
type TxSubmitter interface {
	// SubmitTrades submits a batch of trades to the chain
	SubmitTrades(ctx context.Context, trades []*types.Trade) error

	// SubmitOrderUpdate submits an order status update to the chain
	SubmitOrderUpdate(ctx context.Context, order *types.Order) error

	// GetStatus returns the submitter status
	GetStatus() SubmitterStatus
}

// SubmitterStatus represents the status of a submitter
type SubmitterStatus struct {
	Connected        bool
	PendingTxCount   int
	LastSubmitTime   time.Time
	LastError        string
	TotalSubmissions int64
	FailedSubmissions int64
}

// MockSubmitter is a mock implementation for testing
type MockSubmitter struct {
	mu              sync.Mutex
	trades          []*types.Trade
	orders          []*types.Order
	status          SubmitterStatus
	simulateFailure bool
}

// NewMockSubmitter creates a new mock submitter
func NewMockSubmitter() *MockSubmitter {
	return &MockSubmitter{
		trades: make([]*types.Trade, 0),
		orders: make([]*types.Order, 0),
		status: SubmitterStatus{
			Connected: true,
		},
	}
}

// SubmitTrades submits trades (mock implementation)
func (s *MockSubmitter) SubmitTrades(ctx context.Context, trades []*types.Trade) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.simulateFailure {
		s.status.FailedSubmissions++
		s.status.LastError = "simulated failure"
		return fmt.Errorf("simulated failure")
	}

	s.trades = append(s.trades, trades...)
	s.status.TotalSubmissions++
	s.status.LastSubmitTime = time.Now()

	log.Printf("[MockSubmitter] Submitted %d trades", len(trades))
	for _, trade := range trades {
		log.Printf("  Trade: %s, Price: %s, Qty: %s", trade.TradeID, trade.Price.String(), trade.Quantity.String())
	}

	return nil
}

// SubmitOrderUpdate submits an order update (mock implementation)
func (s *MockSubmitter) SubmitOrderUpdate(ctx context.Context, order *types.Order) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.simulateFailure {
		s.status.FailedSubmissions++
		s.status.LastError = "simulated failure"
		return fmt.Errorf("simulated failure")
	}

	s.orders = append(s.orders, order)
	s.status.TotalSubmissions++
	s.status.LastSubmitTime = time.Now()

	log.Printf("[MockSubmitter] Submitted order update: %s, Status: %s", order.OrderID, order.Status.String())

	return nil
}

// GetStatus returns the mock submitter status
func (s *MockSubmitter) GetStatus() SubmitterStatus {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.status
}

// GetSubmittedTrades returns all submitted trades (for testing)
func (s *MockSubmitter) GetSubmittedTrades() []*types.Trade {
	s.mu.Lock()
	defer s.mu.Unlock()
	result := make([]*types.Trade, len(s.trades))
	copy(result, s.trades)
	return result
}

// GetSubmittedOrders returns all submitted orders (for testing)
func (s *MockSubmitter) GetSubmittedOrders() []*types.Order {
	s.mu.Lock()
	defer s.mu.Unlock()
	result := make([]*types.Order, len(s.orders))
	copy(result, s.orders)
	return result
}

// SetSimulateFailure enables or disables failure simulation
func (s *MockSubmitter) SetSimulateFailure(fail bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.simulateFailure = fail
}

// Clear clears all submitted data (for testing)
func (s *MockSubmitter) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.trades = make([]*types.Trade, 0)
	s.orders = make([]*types.Order, 0)
}

// BatchSubmitter submits trades in batches to the chain
type BatchSubmitter struct {
	rpcURL        string
	batchSize     int
	retryAttempts int
	retryDelay    time.Duration

	mu     sync.Mutex
	status SubmitterStatus
}

// BatchSubmitterConfig holds configuration for BatchSubmitter
type BatchSubmitterConfig struct {
	RPCURL        string
	BatchSize     int
	RetryAttempts int
	RetryDelay    time.Duration
}

// DefaultBatchSubmitterConfig returns default configuration
func DefaultBatchSubmitterConfig() *BatchSubmitterConfig {
	return &BatchSubmitterConfig{
		RPCURL:        "http://localhost:26657",
		BatchSize:     100,
		RetryAttempts: 3,
		RetryDelay:    time.Second,
	}
}

// NewBatchSubmitter creates a new batch submitter
func NewBatchSubmitter(config *BatchSubmitterConfig) *BatchSubmitter {
	if config == nil {
		config = DefaultBatchSubmitterConfig()
	}

	return &BatchSubmitter{
		rpcURL:        config.RPCURL,
		batchSize:     config.BatchSize,
		retryAttempts: config.RetryAttempts,
		retryDelay:    config.RetryDelay,
		status: SubmitterStatus{
			Connected: true,
		},
	}
}

// SubmitTrades submits trades in batches
func (s *BatchSubmitter) SubmitTrades(ctx context.Context, trades []*types.Trade) error {
	if len(trades) == 0 {
		return nil
	}

	s.mu.Lock()
	s.status.PendingTxCount = len(trades)
	s.mu.Unlock()

	// Split into batches
	for i := 0; i < len(trades); i += s.batchSize {
		end := i + s.batchSize
		if end > len(trades) {
			end = len(trades)
		}
		batch := trades[i:end]

		if err := s.submitBatchWithRetry(ctx, batch); err != nil {
			s.mu.Lock()
			s.status.FailedSubmissions++
			s.status.LastError = err.Error()
			s.mu.Unlock()
			return fmt.Errorf("failed to submit batch: %w", err)
		}
	}

	s.mu.Lock()
	s.status.TotalSubmissions++
	s.status.LastSubmitTime = time.Now()
	s.status.PendingTxCount = 0
	s.mu.Unlock()

	return nil
}

// submitBatchWithRetry submits a batch with retry logic
func (s *BatchSubmitter) submitBatchWithRetry(ctx context.Context, batch []*types.Trade) error {
	var lastErr error
	for attempt := 0; attempt < s.retryAttempts; attempt++ {
		if err := s.submitBatch(ctx, batch); err != nil {
			lastErr = err
			log.Printf("Batch submission attempt %d failed: %v", attempt+1, err)

			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(s.retryDelay):
				continue
			}
		}
		return nil
	}
	return fmt.Errorf("all retry attempts failed: %w", lastErr)
}

// submitBatch submits a single batch
func (s *BatchSubmitter) submitBatch(ctx context.Context, batch []*types.Trade) error {
	// Prepare the transaction message
	msg := struct {
		Jsonrpc string        `json:"jsonrpc"`
		ID      int           `json:"id"`
		Method  string        `json:"method"`
		Params  []interface{} `json:"params"`
	}{
		Jsonrpc: "2.0",
		ID:      1,
		Method:  "broadcast_tx_async",
		Params:  []interface{}{s.encodeTrades(batch)},
	}

	// Log the submission (in production, this would be an actual RPC call)
	msgBytes, _ := json.Marshal(msg)
	log.Printf("[BatchSubmitter] Submitting batch of %d trades to %s", len(batch), s.rpcURL)
	log.Printf("[BatchSubmitter] Message: %s", string(msgBytes))

	// In a real implementation, we would:
	// 1. Create a MsgSettleTrades transaction
	// 2. Sign the transaction
	// 3. Broadcast via RPC

	return nil
}

// encodeTrades encodes trades for submission
func (s *BatchSubmitter) encodeTrades(trades []*types.Trade) string {
	// In production, this would properly encode the trades
	// into a Cosmos SDK transaction
	data := make([]map[string]string, len(trades))
	for i, trade := range trades {
		data[i] = map[string]string{
			"trade_id":  trade.TradeID,
			"market_id": trade.MarketID,
			"taker":     trade.Taker,
			"maker":     trade.Maker,
			"price":     trade.Price.String(),
			"quantity":  trade.Quantity.String(),
		}
	}
	encoded, _ := json.Marshal(data)
	return string(encoded)
}

// SubmitOrderUpdate submits an order update
func (s *BatchSubmitter) SubmitOrderUpdate(ctx context.Context, order *types.Order) error {
	log.Printf("[BatchSubmitter] Submitting order update: %s -> %s", order.OrderID, order.Status.String())

	// In production, this would create and broadcast a transaction
	s.mu.Lock()
	s.status.TotalSubmissions++
	s.status.LastSubmitTime = time.Now()
	s.mu.Unlock()

	return nil
}

// GetStatus returns the submitter status
func (s *BatchSubmitter) GetStatus() SubmitterStatus {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.status
}

// SetRPCURL updates the RPC URL
func (s *BatchSubmitter) SetRPCURL(url string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.rpcURL = url
}

// SubmitterFactory creates submitters based on configuration
type SubmitterFactory struct{}

// NewSubmitterFactory creates a new submitter factory
func NewSubmitterFactory() *SubmitterFactory {
	return &SubmitterFactory{}
}

// Create creates a new submitter based on the type
func (f *SubmitterFactory) Create(submitterType string, config *BatchSubmitterConfig) TxSubmitter {
	switch submitterType {
	case "mock":
		return NewMockSubmitter()
	case "batch":
		return NewBatchSubmitter(config)
	default:
		return NewMockSubmitter()
	}
}
