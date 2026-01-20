package keeper

import (
	"fmt"
	"time"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/openalpha/perp-dex/x/orderbook/types"
)

// MatchingStats contains performance statistics for order matching
type MatchingStats struct {
	OrdersProcessed  int
	TradesExecuted   int
	TotalVolume      math.LegacyDec
	AvgLatencyMicros int64
	StartTime        time.Time
	EndTime          time.Time
}

// MatchingEngine handles order matching with Price-Time Priority
type MatchingEngine struct {
	keeper *Keeper
}

// NewMatchingEngine creates a new matching engine
func NewMatchingEngine(keeper *Keeper) *MatchingEngine {
	return &MatchingEngine{keeper: keeper}
}

// MatchResult contains the result of order matching
type MatchResult struct {
	Trades       []*types.Trade
	FilledQty    math.LegacyDec
	AvgPrice     math.LegacyDec
	RemainingQty math.LegacyDec
}

// Match attempts to match an incoming order against the order book
// Uses Price-Time Priority algorithm
func (me *MatchingEngine) Match(ctx sdk.Context, order *types.Order) (*MatchResult, error) {
	orderBook := me.keeper.GetOrderBook(ctx, order.MarketID)
	if orderBook == nil {
		orderBook = types.NewOrderBook(order.MarketID)
	}

	result := &MatchResult{
		Trades:       make([]*types.Trade, 0),
		FilledQty:    math.LegacyZeroDec(),
		AvgPrice:     math.LegacyZeroDec(),
		RemainingQty: order.RemainingQty(),
	}

	// Get the opposite side of the order book
	var oppositeLevels []*types.PriceLevel
	if order.Side == types.SideBuy {
		oppositeLevels = orderBook.Asks
	} else {
		oppositeLevels = orderBook.Bids
	}

	// Track total value for average price calculation
	totalValue := math.LegacyZeroDec()

	// Match against each price level
	for _, level := range oppositeLevels {
		if result.RemainingQty.IsZero() {
			break
		}

		// Check price compatibility
		if !me.isPriceCompatible(order, level.Price) {
			break
		}

		// Match against orders at this price level (FIFO)
		for _, makerOrderID := range level.OrderIDs {
			if result.RemainingQty.IsZero() {
				break
			}

			makerOrder := me.keeper.GetOrder(ctx, makerOrderID)
			if makerOrder == nil || !makerOrder.IsActive() {
				continue
			}

			// Calculate match quantity
			matchQty := math.LegacyMinDec(result.RemainingQty, makerOrder.RemainingQty())
			matchPrice := level.Price // Maker's price

			// Calculate fees
			market := me.keeper.perpetualKeeper.GetMarket(ctx, order.MarketID)
			takerFee := me.calculateFee(matchQty, matchPrice, market.TakerFeeRate)
			makerFee := me.calculateFee(matchQty, matchPrice, market.MakerFeeRate)

			// Create trade
			tradeID := me.keeper.generateTradeID(ctx)
			trade := types.NewTrade(tradeID, order.MarketID, order, makerOrder, matchPrice, matchQty, takerFee, makerFee)
			result.Trades = append(result.Trades, trade)

			// Update quantities
			if err := order.Fill(matchQty); err != nil {
				return nil, fmt.Errorf("failed to fill taker order: %w", err)
			}
			if err := makerOrder.Fill(matchQty); err != nil {
				return nil, fmt.Errorf("failed to fill maker order: %w", err)
			}

			// Update positions for both traders (CRITICAL: creates real positions)
			// Taker: order.Side determines position direction (buy=long, sell=short)
			if err := me.keeper.perpetualKeeper.UpdatePosition(ctx, order.Trader, order.MarketID, order.Side, matchQty, matchPrice, takerFee); err != nil {
				me.keeper.Logger().Error("failed to update taker position", "trader", order.Trader, "error", err)
			}
			// Maker: makerOrder.Side determines position direction
			if err := me.keeper.perpetualKeeper.UpdatePosition(ctx, makerOrder.Trader, makerOrder.MarketID, makerOrder.Side, matchQty, matchPrice, makerFee); err != nil {
				me.keeper.Logger().Error("failed to update maker position", "trader", makerOrder.Trader, "error", err)
			}

			// Update tracking
			result.FilledQty = result.FilledQty.Add(matchQty)
			result.RemainingQty = result.RemainingQty.Sub(matchQty)
			totalValue = totalValue.Add(matchQty.Mul(matchPrice))

			// Save updated maker order
			me.keeper.SetOrder(ctx, makerOrder)

			// Update order book
			level.Quantity = level.Quantity.Sub(matchQty)
			if makerOrder.IsFilled() {
				level.RemoveOrder(makerOrderID, math.LegacyZeroDec())
			}

			// Emit trade event
			me.keeper.emitTradeEvent(ctx, trade)
		}
	}

	// Calculate average price
	if result.FilledQty.IsPositive() {
		result.AvgPrice = totalValue.Quo(result.FilledQty)
	}

	// Clean up empty price levels and save order book
	me.cleanupOrderBook(orderBook)
	me.keeper.SetOrderBook(ctx, orderBook)

	return result, nil
}

// isPriceCompatible checks if the order can match at the given price
func (me *MatchingEngine) isPriceCompatible(order *types.Order, levelPrice math.LegacyDec) bool {
	// Market orders match at any price
	if order.OrderType == types.OrderTypeMarket {
		return true
	}

	// Limit orders: buy must be >= ask, sell must be <= bid
	if order.Side == types.SideBuy {
		return order.Price.GTE(levelPrice)
	}
	return order.Price.LTE(levelPrice)
}

// calculateFee calculates the trading fee
func (me *MatchingEngine) calculateFee(qty, price, feeRate math.LegacyDec) math.LegacyDec {
	if feeRate.IsZero() {
		return math.LegacyZeroDec()
	}
	return qty.Mul(price).Mul(feeRate)
}

// cleanupOrderBook removes empty price levels
func (me *MatchingEngine) cleanupOrderBook(ob *types.OrderBook) {
	// Clean bids
	cleanBids := make([]*types.PriceLevel, 0)
	for _, level := range ob.Bids {
		if !level.IsEmpty() {
			cleanBids = append(cleanBids, level)
		}
	}
	ob.Bids = cleanBids

	// Clean asks
	cleanAsks := make([]*types.PriceLevel, 0)
	for _, level := range ob.Asks {
		if !level.IsEmpty() {
			cleanAsks = append(cleanAsks, level)
		}
	}
	ob.Asks = cleanAsks
}

// ProcessOrder is the main entry point for order processing
// It matches the order and adds any remaining quantity to the book
func (me *MatchingEngine) ProcessOrder(ctx sdk.Context, order *types.Order) (*MatchResult, error) {
	// First, try to match the order
	result, err := me.Match(ctx, order)
	if err != nil {
		return nil, err
	}

	// If there's remaining quantity and it's a limit order, add to book
	if result.RemainingQty.IsPositive() && order.OrderType == types.OrderTypeLimit {
		orderBook := me.keeper.GetOrderBook(ctx, order.MarketID)
		if orderBook == nil {
			orderBook = types.NewOrderBook(order.MarketID)
		}
		orderBook.AddOrder(order)
		me.keeper.SetOrderBook(ctx, orderBook)
		me.keeper.SetOrder(ctx, order)
	} else if order.IsActive() && order.OrderType == types.OrderTypeMarket {
		// Market order with unfilled quantity - cancel the rest
		order.Cancel()
	}

	// Save the taker order
	me.keeper.SetOrder(ctx, order)

	return result, nil
}

// CancelOrder cancels an order and removes it from the order book
func (me *MatchingEngine) CancelOrder(ctx sdk.Context, orderID string) (*types.Order, error) {
	order := me.keeper.GetOrder(ctx, orderID)
	if order == nil {
		return nil, fmt.Errorf("order not found: %s", orderID)
	}

	if !order.IsActive() {
		return nil, fmt.Errorf("order is not active: %s", orderID)
	}

	// Remove from order book
	orderBook := me.keeper.GetOrderBook(ctx, order.MarketID)
	if orderBook != nil {
		orderBook.RemoveOrder(order)
		me.keeper.SetOrderBook(ctx, orderBook)
	}

	// Cancel the order
	order.Cancel()
	me.keeper.SetOrder(ctx, order)

	return order, nil
}

// ProcessPendingOrders processes all pending orders and returns performance statistics
// This is the optimized entry point for EndBlocker order matching
func (me *MatchingEngine) ProcessPendingOrders(ctx sdk.Context) MatchingStats {
	stats := MatchingStats{
		StartTime:   time.Now(),
		TotalVolume: math.LegacyZeroDec(),
	}

	// Get all pending orders
	pendingOrders := me.keeper.GetAllPendingOrders(ctx)
	if len(pendingOrders) == 0 {
		stats.EndTime = time.Now()
		return stats
	}

	// Track individual order processing times for latency calculation
	var totalLatency int64
	processedCount := 0

	// Process each order
	for _, order := range pendingOrders {
		if !order.IsActive() {
			continue
		}

		orderStart := time.Now()

		// Process the order through the matching engine
		result, err := me.ProcessOrder(ctx, order)
		if err != nil {
			me.keeper.Logger().Error("failed to process order in EndBlocker",
				"order_id", order.OrderID,
				"error", err,
			)
			continue
		}

		orderLatency := time.Since(orderStart).Microseconds()
		totalLatency += orderLatency
		processedCount++

		// Update statistics
		stats.OrdersProcessed++
		if result != nil {
			stats.TradesExecuted += len(result.Trades)

			// Calculate volume from trades
			for _, trade := range result.Trades {
				tradeValue := trade.Quantity.Mul(trade.Price)
				stats.TotalVolume = stats.TotalVolume.Add(tradeValue)

				// Save trade to store
				me.keeper.SetTrade(ctx, trade)
			}
		}
	}

	// Calculate average latency
	if processedCount > 0 {
		stats.AvgLatencyMicros = totalLatency / int64(processedCount)
	}

	stats.EndTime = time.Now()
	return stats
}
