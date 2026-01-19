package keeper

import (
	"encoding/json"

	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/openalpha/perp-dex/x/orderbook/types"
)

// Store key prefixes for trade history indexing
var (
	TradeByTraderPrefix = []byte{0x10}
	TradeByMarketPrefix = []byte{0x11}
)

// ============ Trade History Queries ============

// GetTradeHistory returns trade history for a trader with pagination
func (k *Keeper) GetTradeHistory(ctx sdk.Context, trader string, limit, offset int) []*types.Trade {
	store := k.GetStore(ctx)

	// Use reverse iterator to get most recent first
	iterator := storetypes.KVStoreReversePrefixIterator(store, TradeKeyPrefix)
	defer iterator.Close()

	var trades []*types.Trade
	count := 0
	skipped := 0

	for ; iterator.Valid(); iterator.Next() {
		var trade types.Trade
		if err := json.Unmarshal(iterator.Value(), &trade); err != nil {
			continue
		}

		// Check if trader is involved
		if trade.Taker == trader || trade.Maker == trader {
			// Handle offset
			if skipped < offset {
				skipped++
				continue
			}

			// Check limit
			if count >= limit {
				break
			}

			trades = append(trades, &trade)
			count++
		}
	}

	return trades
}

// GetTradeHistoryByMarket returns trade history for a market
func (k *Keeper) GetTradeHistoryByMarket(ctx sdk.Context, marketID string, limit int) []*types.Trade {
	store := k.GetStore(ctx)

	iterator := storetypes.KVStoreReversePrefixIterator(store, TradeKeyPrefix)
	defer iterator.Close()

	var trades []*types.Trade
	count := 0

	for ; iterator.Valid() && count < limit; iterator.Next() {
		var trade types.Trade
		if err := json.Unmarshal(iterator.Value(), &trade); err != nil {
			continue
		}

		if trade.MarketID == marketID {
			trades = append(trades, &trade)
			count++
		}
	}

	return trades
}

// GetTradeHistoryFiltered returns filtered trade history
func (k *Keeper) GetTradeHistoryFiltered(
	ctx sdk.Context,
	trader string,
	marketID string,
	side *types.Side,
	fromTime, toTime int64,
	limit, offset int,
) []*types.Trade {
	store := k.GetStore(ctx)

	iterator := storetypes.KVStoreReversePrefixIterator(store, TradeKeyPrefix)
	defer iterator.Close()

	var trades []*types.Trade
	count := 0
	skipped := 0

	for ; iterator.Valid(); iterator.Next() {
		var trade types.Trade
		if err := json.Unmarshal(iterator.Value(), &trade); err != nil {
			continue
		}

		// Apply filters
		// Trader filter
		if trader != "" && trade.Taker != trader && trade.Maker != trader {
			continue
		}

		// Market filter
		if marketID != "" && trade.MarketID != marketID {
			continue
		}

		// Side filter
		if side != nil {
			isTaker := trade.Taker == trader
			if isTaker && trade.TakerSide != *side {
				continue
			}
			if !isTaker && trade.TakerSide == *side { // Maker is opposite side
				continue
			}
		}

		// Time filter
		if fromTime > 0 && trade.Timestamp.Unix() < fromTime {
			continue
		}
		if toTime > 0 && trade.Timestamp.Unix() > toTime {
			break // Since we're iterating in reverse, we can stop early
		}

		// Handle offset
		if skipped < offset {
			skipped++
			continue
		}

		// Check limit
		if count >= limit {
			break
		}

		trades = append(trades, &trade)
		count++
	}

	return trades
}

// GetTradeCount returns the total trade count for a trader
func (k *Keeper) GetTradeCount(ctx sdk.Context, trader string) int64 {
	store := k.GetStore(ctx)

	iterator := storetypes.KVStorePrefixIterator(store, TradeKeyPrefix)
	defer iterator.Close()

	var count int64
	for ; iterator.Valid(); iterator.Next() {
		var trade types.Trade
		if err := json.Unmarshal(iterator.Value(), &trade); err != nil {
			continue
		}

		if trade.Taker == trader || trade.Maker == trader {
			count++
		}
	}

	return count
}

// ============ Order Queries ============

// GetOpenOrders returns all open (active) orders for a trader
func (k *Keeper) GetOpenOrders(ctx sdk.Context, trader string) []*types.Order {
	orders := k.GetOrdersByTrader(ctx, trader)

	var openOrders []*types.Order
	for _, order := range orders {
		if order.IsActive() {
			openOrders = append(openOrders, order)
		}
	}

	return openOrders
}

// GetOpenOrdersByMarket returns open orders for a trader in a specific market
func (k *Keeper) GetOpenOrdersByMarket(ctx sdk.Context, trader, marketID string) []*types.Order {
	orders := k.GetOrdersByTrader(ctx, trader)

	var filtered []*types.Order
	for _, order := range orders {
		if order.IsActive() && order.MarketID == marketID {
			filtered = append(filtered, order)
		}
	}

	return filtered
}

// GetOrderHistory returns historical orders (filled/cancelled) for a trader
func (k *Keeper) GetOrderHistory(ctx sdk.Context, trader string, limit, offset int) []*types.Order {
	store := k.GetStore(ctx)

	iterator := storetypes.KVStoreReversePrefixIterator(store, OrderKeyPrefix)
	defer iterator.Close()

	var orders []*types.Order
	count := 0
	skipped := 0

	for ; iterator.Valid(); iterator.Next() {
		var order types.Order
		if err := json.Unmarshal(iterator.Value(), &order); err != nil {
			continue
		}

		// Check if order belongs to trader and is not active
		if order.Trader == trader && !order.IsActive() {
			if skipped < offset {
				skipped++
				continue
			}

			if count >= limit {
				break
			}

			orders = append(orders, &order)
			count++
		}
	}

	return orders
}

// ============ Market Stats Queries ============

// MarketStats holds aggregated market statistics
type MarketStats struct {
	MarketID     string `json:"market_id"`
	TradeCount24h int64  `json:"trade_count_24h"`
	Volume24h    string `json:"volume_24h"`
	High24h      string `json:"high_24h"`
	Low24h       string `json:"low_24h"`
	LastPrice    string `json:"last_price"`
	PriceChange  string `json:"price_change"`
}

// GetMarketStats returns statistics for a market
func (k *Keeper) GetMarketStats(ctx sdk.Context, marketID string) *MarketStats {
	now := ctx.BlockTime().Unix()
	from := now - 86400 // 24 hours ago

	trades := k.GetTradeHistoryByMarket(ctx, marketID, 1000)

	stats := &MarketStats{
		MarketID: marketID,
	}

	if len(trades) == 0 {
		return stats
	}

	// Calculate stats from trades
	var count int64
	var firstPrice, lastPrice string
	high := trades[0].Price
	low := trades[0].Price
	volume := trades[0].Quantity

	lastPrice = trades[0].Price.String()

	for i, trade := range trades {
		if trade.Timestamp.Unix() < from {
			continue
		}

		count++
		volume = volume.Add(trade.Quantity.Mul(trade.Price))

		if trade.Price.GT(high) {
			high = trade.Price
		}
		if trade.Price.LT(low) {
			low = trade.Price
		}

		// Last trade in range is first price (oldest)
		if i == len(trades)-1 || trades[i+1].Timestamp.Unix() < from {
			firstPrice = trade.Price.String()
		}
	}

	stats.TradeCount24h = count
	stats.Volume24h = volume.String()
	stats.High24h = high.String()
	stats.Low24h = low.String()
	stats.LastPrice = lastPrice

	// Calculate price change
	if firstPrice != "" && lastPrice != "" {
		// Price change calculation would go here
		stats.PriceChange = "0" // Placeholder
	}

	return stats
}

// Note: GetConditionalOrdersByTrader is defined in conditional.go
