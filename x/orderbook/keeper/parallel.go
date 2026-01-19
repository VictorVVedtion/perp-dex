package keeper

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/openalpha/perp-dex/x/orderbook/types"
)

// ParallelConfig holds configuration for parallel matching
type ParallelConfig struct {
	// Enabled determines if parallel matching is active
	Enabled bool
	// Workers is the number of worker goroutines
	Workers int
	// BatchSize is the maximum orders per batch
	BatchSize int
	// Timeout is the maximum time for matching operations
	Timeout time.Duration
}

// DefaultParallelConfig returns the default parallel matching configuration
func DefaultParallelConfig() ParallelConfig {
	return ParallelConfig{
		Enabled:   true,
		Workers:   4,
		BatchSize: 100,
		Timeout:   5 * time.Second,
	}
}

// MarketOrders groups orders by market
type MarketOrders struct {
	MarketID string
	Orders   []*types.Order
}

// ParallelMatchResult contains the result of parallel matching for a market
type ParallelMatchResult struct {
	MarketID     string
	Trades       []*types.Trade
	UpdatedOrders []*types.Order
	Error        error
	ProcessedAt  time.Time
}

// AggregatedMatchResult contains combined results from all markets
type AggregatedMatchResult struct {
	Results      []*ParallelMatchResult
	TotalTrades  int
	TotalMatched int
	Duration     time.Duration
	Errors       []error
}

// ParallelMatcher handles parallel order matching across multiple markets
type ParallelMatcher struct {
	keeper    *Keeper
	config    ParallelConfig
	scheduler *MatchingScheduler
	mu        sync.RWMutex
}

// NewParallelMatcher creates a new parallel matcher
func NewParallelMatcher(keeper *Keeper, config ParallelConfig) *ParallelMatcher {
	pm := &ParallelMatcher{
		keeper: keeper,
		config: config,
	}
	pm.scheduler = NewMatchingScheduler(config.Workers, config.BatchSize, keeper)
	return pm
}

// GroupOrdersByMarket groups a slice of orders by their market ID
func (pm *ParallelMatcher) GroupOrdersByMarket(orders []*types.Order) map[string]*MarketOrders {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	grouped := make(map[string]*MarketOrders)
	for _, order := range orders {
		if order == nil || !order.IsActive() {
			continue
		}
		if _, exists := grouped[order.MarketID]; !exists {
			grouped[order.MarketID] = &MarketOrders{
				MarketID: order.MarketID,
				Orders:   make([]*types.Order, 0),
			}
		}
		grouped[order.MarketID].Orders = append(grouped[order.MarketID].Orders, order)
	}
	return grouped
}

// MatchParallel performs parallel matching for multiple markets
func (pm *ParallelMatcher) MatchParallel(ctx sdk.Context, orders []*types.Order) (*AggregatedMatchResult, error) {
	startTime := time.Now()

	if !pm.config.Enabled {
		// Fall back to sequential matching
		return pm.matchSequential(ctx, orders)
	}

	// Group orders by market
	grouped := pm.GroupOrdersByMarket(orders)
	if len(grouped) == 0 {
		return &AggregatedMatchResult{
			Results:  make([]*ParallelMatchResult, 0),
			Duration: time.Since(startTime),
		}, nil
	}

	// Create context with timeout
	matchCtx, cancel := context.WithTimeout(ctx.Context(), pm.config.Timeout)
	defer cancel()

	// Start scheduler if not running
	pm.scheduler.Start()

	// Submit matching tasks
	resultChan := make(chan *ParallelMatchResult, len(grouped))
	var wg sync.WaitGroup

	for _, marketOrders := range grouped {
		wg.Add(1)
		go func(mo *MarketOrders) {
			defer wg.Done()

			result := &ParallelMatchResult{
				MarketID:    mo.MarketID,
				Trades:      make([]*types.Trade, 0),
				ProcessedAt: time.Now(),
			}

			// Check for context cancellation
			select {
			case <-matchCtx.Done():
				result.Error = matchCtx.Err()
				resultChan <- result
				return
			default:
			}

			// Process orders for this market
			trades, updatedOrders, err := pm.matchMarket(ctx, mo)
			result.Trades = trades
			result.UpdatedOrders = updatedOrders
			result.Error = err

			resultChan <- result
		}(marketOrders)
	}

	// Wait for all goroutines to complete
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// Collect results
	aggregated := &AggregatedMatchResult{
		Results: make([]*ParallelMatchResult, 0, len(grouped)),
		Errors:  make([]error, 0),
	}

	for result := range resultChan {
		aggregated.Results = append(aggregated.Results, result)
		aggregated.TotalTrades += len(result.Trades)
		if result.Error != nil {
			aggregated.Errors = append(aggregated.Errors, result.Error)
		}
	}

	// Sort results by market ID for deterministic ordering
	sort.Slice(aggregated.Results, func(i, j int) bool {
		return aggregated.Results[i].MarketID < aggregated.Results[j].MarketID
	})

	aggregated.Duration = time.Since(startTime)
	aggregated.TotalMatched = len(orders)

	return aggregated, nil
}

// matchMarket performs matching for a single market
func (pm *ParallelMatcher) matchMarket(ctx sdk.Context, mo *MarketOrders) ([]*types.Trade, []*types.Order, error) {
	engine := NewMatchingEngine(pm.keeper)
	allTrades := make([]*types.Trade, 0)
	updatedOrders := make([]*types.Order, 0)

	// Sort orders by time for FIFO processing
	sort.Slice(mo.Orders, func(i, j int) bool {
		return mo.Orders[i].CreatedAt.Before(mo.Orders[j].CreatedAt)
	})

	for _, order := range mo.Orders {
		if !order.IsActive() {
			continue
		}

		result, err := engine.ProcessOrder(ctx, order)
		if err != nil {
			return allTrades, updatedOrders, fmt.Errorf("failed to process order %s: %w", order.OrderID, err)
		}

		if result != nil && len(result.Trades) > 0 {
			allTrades = append(allTrades, result.Trades...)
		}
		updatedOrders = append(updatedOrders, order)
	}

	return allTrades, updatedOrders, nil
}

// matchSequential performs sequential matching when parallel is disabled
func (pm *ParallelMatcher) matchSequential(ctx sdk.Context, orders []*types.Order) (*AggregatedMatchResult, error) {
	startTime := time.Now()
	engine := NewMatchingEngine(pm.keeper)

	aggregated := &AggregatedMatchResult{
		Results: make([]*ParallelMatchResult, 0),
		Errors:  make([]error, 0),
	}

	// Group results by market
	marketResults := make(map[string]*ParallelMatchResult)

	for _, order := range orders {
		if order == nil || !order.IsActive() {
			continue
		}

		result, err := engine.ProcessOrder(ctx, order)
		if err != nil {
			aggregated.Errors = append(aggregated.Errors, err)
			continue
		}

		if _, exists := marketResults[order.MarketID]; !exists {
			marketResults[order.MarketID] = &ParallelMatchResult{
				MarketID:    order.MarketID,
				Trades:      make([]*types.Trade, 0),
				ProcessedAt: time.Now(),
			}
		}

		if result != nil && len(result.Trades) > 0 {
			marketResults[order.MarketID].Trades = append(
				marketResults[order.MarketID].Trades,
				result.Trades...,
			)
			aggregated.TotalTrades += len(result.Trades)
		}
		aggregated.TotalMatched++
	}

	for _, result := range marketResults {
		aggregated.Results = append(aggregated.Results, result)
	}

	// Sort for deterministic ordering
	sort.Slice(aggregated.Results, func(i, j int) bool {
		return aggregated.Results[i].MarketID < aggregated.Results[j].MarketID
	})

	aggregated.Duration = time.Since(startTime)
	return aggregated, nil
}

// MatchingScheduler manages a pool of workers for order matching
type MatchingScheduler struct {
	workers    int
	batchSize  int
	keeper     *Keeper
	orderQueue chan *types.Order
	resultChan chan *MatchResult
	stopChan   chan struct{}
	wg         sync.WaitGroup
	running    bool
	mu         sync.Mutex
}

// NewMatchingScheduler creates a new matching scheduler
func NewMatchingScheduler(workers, batchSize int, keeper *Keeper) *MatchingScheduler {
	if workers <= 0 {
		workers = 4
	}
	if batchSize <= 0 {
		batchSize = 100
	}

	return &MatchingScheduler{
		workers:    workers,
		batchSize:  batchSize,
		keeper:     keeper,
		orderQueue: make(chan *types.Order, batchSize*workers),
		resultChan: make(chan *MatchResult, batchSize*workers),
		stopChan:   make(chan struct{}),
	}
}

// Start starts the worker pool
func (ms *MatchingScheduler) Start() {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	if ms.running {
		return
	}

	ms.running = true
	ms.stopChan = make(chan struct{})

	for i := 0; i < ms.workers; i++ {
		ms.wg.Add(1)
		go ms.worker(i)
	}
}

// Stop gracefully shuts down the worker pool
func (ms *MatchingScheduler) Stop() {
	ms.mu.Lock()
	if !ms.running {
		ms.mu.Unlock()
		return
	}
	ms.running = false
	ms.mu.Unlock()

	close(ms.stopChan)
	ms.wg.Wait()
}

// IsRunning returns whether the scheduler is running
func (ms *MatchingScheduler) IsRunning() bool {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	return ms.running
}

// worker is a worker goroutine that processes orders
func (ms *MatchingScheduler) worker(id int) {
	defer ms.wg.Done()

	for {
		select {
		case <-ms.stopChan:
			return
		case order, ok := <-ms.orderQueue:
			if !ok {
				return
			}
			// Process order - note: in real implementation, we'd need a context
			_ = order
			_ = id
		}
	}
}

// SubmitOrder submits an order for processing
func (ms *MatchingScheduler) SubmitOrder(order *types.Order) error {
	ms.mu.Lock()
	if !ms.running {
		ms.mu.Unlock()
		return fmt.Errorf("scheduler is not running")
	}
	ms.mu.Unlock()

	select {
	case ms.orderQueue <- order:
		return nil
	default:
		return fmt.Errorf("order queue is full")
	}
}

// ProcessBatch processes a batch of orders
func (ms *MatchingScheduler) ProcessBatch(ctx sdk.Context, orders []*types.Order) ([]*MatchResult, error) {
	if len(orders) == 0 {
		return nil, nil
	}

	ms.mu.Lock()
	if !ms.running {
		ms.mu.Unlock()
		return nil, fmt.Errorf("scheduler is not running")
	}
	ms.mu.Unlock()

	engine := NewMatchingEngine(ms.keeper)
	results := make([]*MatchResult, 0, len(orders))

	// Process in batches
	for i := 0; i < len(orders); i += ms.batchSize {
		end := i + ms.batchSize
		if end > len(orders) {
			end = len(orders)
		}

		batch := orders[i:end]
		for _, order := range batch {
			if order == nil || !order.IsActive() {
				continue
			}

			result, err := engine.ProcessOrder(ctx, order)
			if err != nil {
				return results, fmt.Errorf("failed to process order %s: %w", order.OrderID, err)
			}
			results = append(results, result)
		}
	}

	return results, nil
}

// GetQueueSize returns the current size of the order queue
func (ms *MatchingScheduler) GetQueueSize() int {
	return len(ms.orderQueue)
}

// GetWorkerCount returns the number of workers
func (ms *MatchingScheduler) GetWorkerCount() int {
	return ms.workers
}

// BatchMatchResult contains results from batch matching
type BatchMatchResult struct {
	Results      []*MatchResult
	TotalTrades  int
	ProcessedQty math.LegacyDec
	Errors       []error
}

// WorkerPool manages concurrent order processing
type WorkerPool struct {
	workers    int
	keeper     *Keeper
	taskChan   chan func()
	stopChan   chan struct{}
	wg         sync.WaitGroup
	running    bool
	mu         sync.Mutex
}

// NewWorkerPool creates a new worker pool
func NewWorkerPool(workers int, keeper *Keeper) *WorkerPool {
	if workers <= 0 {
		workers = 4
	}
	return &WorkerPool{
		workers:  workers,
		keeper:   keeper,
		taskChan: make(chan func(), workers*10),
		stopChan: make(chan struct{}),
	}
}

// Start starts all workers
func (wp *WorkerPool) Start() {
	wp.mu.Lock()
	defer wp.mu.Unlock()

	if wp.running {
		return
	}

	wp.running = true
	wp.stopChan = make(chan struct{})

	for i := 0; i < wp.workers; i++ {
		wp.wg.Add(1)
		go wp.runWorker()
	}
}

// Stop stops all workers gracefully
func (wp *WorkerPool) Stop() {
	wp.mu.Lock()
	if !wp.running {
		wp.mu.Unlock()
		return
	}
	wp.running = false
	wp.mu.Unlock()

	close(wp.stopChan)
	wp.wg.Wait()
}

// runWorker runs a single worker
func (wp *WorkerPool) runWorker() {
	defer wp.wg.Done()

	for {
		select {
		case <-wp.stopChan:
			return
		case task, ok := <-wp.taskChan:
			if !ok {
				return
			}
			task()
		}
	}
}

// Submit submits a task to the worker pool
func (wp *WorkerPool) Submit(task func()) error {
	wp.mu.Lock()
	if !wp.running {
		wp.mu.Unlock()
		return fmt.Errorf("worker pool is not running")
	}
	wp.mu.Unlock()

	select {
	case wp.taskChan <- task:
		return nil
	default:
		return fmt.Errorf("task queue is full")
	}
}

// ProcessBatch processes a batch of orders using the worker pool
func (wp *WorkerPool) ProcessBatch(ctx sdk.Context, orders []*types.Order) (*BatchMatchResult, error) {
	if len(orders) == 0 {
		return &BatchMatchResult{
			Results:      make([]*MatchResult, 0),
			ProcessedQty: math.LegacyZeroDec(),
			Errors:       make([]error, 0),
		}, nil
	}

	var resultMu sync.Mutex
	result := &BatchMatchResult{
		Results:      make([]*MatchResult, 0, len(orders)),
		ProcessedQty: math.LegacyZeroDec(),
		Errors:       make([]error, 0),
	}

	var wg sync.WaitGroup
	engine := NewMatchingEngine(wp.keeper)

	for _, order := range orders {
		if order == nil || !order.IsActive() {
			continue
		}

		wg.Add(1)
		orderCopy := order
		err := wp.Submit(func() {
			defer wg.Done()

			matchResult, err := engine.ProcessOrder(ctx, orderCopy)

			resultMu.Lock()
			defer resultMu.Unlock()

			if err != nil {
				result.Errors = append(result.Errors, err)
				return
			}

			if matchResult != nil {
				result.Results = append(result.Results, matchResult)
				result.TotalTrades += len(matchResult.Trades)
				result.ProcessedQty = result.ProcessedQty.Add(matchResult.FilledQty)
			}
		})

		if err != nil {
			wg.Done()
			result.Errors = append(result.Errors, err)
		}
	}

	wg.Wait()
	return result, nil
}
