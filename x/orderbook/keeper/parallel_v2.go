package keeper

import (
	"fmt"
	"sort"
	"sync"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/openalpha/perp-dex/x/orderbook/types"
)

// ParallelMatchResultV2 contains the result of parallel matching for a market.
type ParallelMatchResultV2 struct {
	MarketID    string
	Trades      []*types.TradeWithSettlement
	Error       error
	ProcessedAt time.Time
	Commit      func()
}

// AggregatedMatchResultV2 contains combined results from all markets.
type AggregatedMatchResultV2 struct {
	Results     []*ParallelMatchResultV2
	TotalTrades int
	Duration    time.Duration
	Errors      []error
}

// ParallelMatcherV2 handles parallel order matching across multiple markets
// using isolated CacheKVStores.
type ParallelMatcherV2 struct {
	keeper *Keeper
	config ParallelConfig
}

// NewParallelMatcherV2 creates a new parallel matcher.
func NewParallelMatcherV2(keeper *Keeper, config ParallelConfig) *ParallelMatcherV2 {
	return &ParallelMatcherV2{
		keeper: keeper,
		config: config,
	}
}

// MatchParallel performs parallel matching for multiple markets using isolated cache contexts.
func (pm *ParallelMatcherV2) MatchParallel(ctx sdk.Context, orders []*types.Order) (*AggregatedMatchResultV2, error) {
	startTime := time.Now()

	if !pm.config.Enabled {
		return pm.matchSequential(ctx, orders)
	}

	grouped := groupOrdersByMarket(orders)
	if len(grouped) == 0 {
		return &AggregatedMatchResultV2{
			Results:  make([]*ParallelMatchResultV2, 0),
			Duration: time.Since(startTime),
		}, nil
	}

	resultChan := make(chan *ParallelMatchResultV2, len(grouped))
	var wg sync.WaitGroup

	for _, marketOrders := range grouped {
		wg.Add(1)
		go func(mo *MarketOrders) {
			defer wg.Done()

			result := &ParallelMatchResultV2{
				MarketID:    mo.MarketID,
				Trades:      make([]*types.TradeWithSettlement, 0),
				ProcessedAt: time.Now(),
			}

			// Panic recovery to prevent node crash
			defer func() {
				if r := recover(); r != nil {
					result.Error = fmt.Errorf("panic in market %s matching: %v", mo.MarketID, r)
					resultChan <- result
				}
			}()

			cacheCtx, write := ctx.CacheContext()
			trades, err := pm.matchMarket(cacheCtx, mo)
			if err != nil {
				result.Error = err
				resultChan <- result
				return
			}

			result.Trades = trades
			result.Commit = write
			resultChan <- result
		}(marketOrders)
	}

	go func() {
		wg.Wait()
		close(resultChan)
	}()

	aggregated := &AggregatedMatchResultV2{
		Results: make([]*ParallelMatchResultV2, 0, len(grouped)),
		Errors:  make([]error, 0),
	}

	for result := range resultChan {
		aggregated.Results = append(aggregated.Results, result)
		aggregated.TotalTrades += len(result.Trades)
		if result.Error != nil {
			aggregated.Errors = append(aggregated.Errors, result.Error)
		}
	}

	sort.Slice(aggregated.Results, func(i, j int) bool {
		return aggregated.Results[i].MarketID < aggregated.Results[j].MarketID
	})

	aggregated.Duration = time.Since(startTime)
	return aggregated, nil
}

func (pm *ParallelMatcherV2) matchSequential(ctx sdk.Context, orders []*types.Order) (*AggregatedMatchResultV2, error) {
	startTime := time.Now()
	grouped := groupOrdersByMarket(orders)

	marketIDs := make([]string, 0, len(grouped))
	for marketID := range grouped {
		marketIDs = append(marketIDs, marketID)
	}
	sort.Strings(marketIDs)

	aggregated := &AggregatedMatchResultV2{
		Results: make([]*ParallelMatchResultV2, 0, len(grouped)),
		Errors:  make([]error, 0),
	}

	for _, marketID := range marketIDs {
		mo := grouped[marketID]
		result := &ParallelMatchResultV2{
			MarketID:    marketID,
			Trades:      make([]*types.TradeWithSettlement, 0),
			ProcessedAt: time.Now(),
		}

		cacheCtx, write := ctx.CacheContext()
		trades, err := pm.matchMarket(cacheCtx, mo)
		if err != nil {
			result.Error = err
			aggregated.Errors = append(aggregated.Errors, err)
		} else {
			result.Trades = trades
			result.Commit = write
			aggregated.TotalTrades += len(trades)
		}
		aggregated.Results = append(aggregated.Results, result)
	}

	aggregated.Duration = time.Since(startTime)
	return aggregated, nil
}

func (pm *ParallelMatcherV2) matchMarket(ctx sdk.Context, mo *MarketOrders) ([]*types.TradeWithSettlement, error) {
	engine := NewMatchingEngineV2(pm.keeper)
	allTrades := make([]*types.TradeWithSettlement, 0)

	sort.Slice(mo.Orders, func(i, j int) bool {
		return mo.Orders[i].CreatedAt.Before(mo.Orders[j].CreatedAt)
	})

	for _, order := range mo.Orders {
		if order == nil || !order.IsActive() {
			continue
		}

		result, err := engine.ProcessOrderOptimized(ctx, order)
		if err != nil {
			return allTrades, fmt.Errorf("failed to process order %s: %w", order.OrderID, err)
		}

		if result != nil && len(result.TradesWithSettlement) > 0 {
			allTrades = append(allTrades, result.TradesWithSettlement...)
		}
	}

	if err := engine.Flush(ctx); err != nil {
		return allTrades, fmt.Errorf("failed to flush market %s: %w", mo.MarketID, err)
	}

	return allTrades, nil
}

func groupOrdersByMarket(orders []*types.Order) map[string]*MarketOrders {
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
