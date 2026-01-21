package keeper

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"sort"

	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/openalpha/perp-dex/x/clearinghouse/types"
	perpetualtypes "github.com/openalpha/perp-dex/x/perpetual/types"
)

// Store key prefixes for ADL
var (
	ADLEventKeyPrefix   = []byte{0x20}
	ADLConfigKeyPrefix  = []byte{0x21}
	ADLEventCounterKey  = []byte{0x22}
	ADLQueueKeyPrefix   = []byte{0x23}
)

// ============ ADL Configuration ============

// SetADLConfig saves ADL configuration
func (k *Keeper) SetADLConfig(ctx sdk.Context, config types.ADLConfig) {
	store := k.GetStore(ctx)
	bz, _ := json.Marshal(config)
	store.Set(ADLConfigKeyPrefix, bz)
}

// GetADLConfig retrieves ADL configuration
func (k *Keeper) GetADLConfig(ctx sdk.Context) types.ADLConfig {
	store := k.GetStore(ctx)
	bz := store.Get(ADLConfigKeyPrefix)
	if bz == nil {
		return types.DefaultADLConfig()
	}
	var config types.ADLConfig
	if err := json.Unmarshal(bz, &config); err != nil {
		return types.DefaultADLConfig()
	}
	return config
}

// ============ ADL Queue Management ============

// BuildADLQueue builds the ADL queue for a market and side
func (k *Keeper) BuildADLQueue(ctx sdk.Context, marketID, side string) *types.ADLQueue {
	queue := types.NewADLQueue(marketID, side)

	// Get all positions for the market
	positions := k.perpetualKeeper.GetAllPositions(ctx)
	priceInfo := k.perpetualKeeper.GetPrice(ctx, marketID)

	if priceInfo == nil {
		return queue
	}

	markPrice := priceInfo.MarkPrice
	targetSide := perpetualtypes.PositionSideLong
	if side == "short" {
		targetSide = perpetualtypes.PositionSideShort
	}

	// Filter and calculate PnL for each position
	// CRITICAL: Only include profitable positions for ADL (positive PnL)
	// ADL should only deleverage profitable positions to cover system deficits
	for _, pos := range positions {
		if pos.MarketID != marketID || pos.Side != targetSide {
			continue
		}

		// Calculate unrealized PnL
		pnl := pos.CalculateUnrealizedPnL(markPrice)

		// CRITICAL FIX: Skip positions with zero or negative PnL
		// ADL is designed to take profit from winning positions to cover losses
		// Deleveraging losing positions doesn't help cover the deficit
		if pnl.IsNegative() || pnl.IsZero() {
			continue
		}

		pnlPercent := math.LegacyZeroDec()
		if pos.Margin.IsPositive() {
			pnlPercent = pnl.Quo(pos.Margin)
		}

		adlPos := &types.ADLPosition{
			Trader:        pos.Trader,
			MarketID:      pos.MarketID,
			Side:          side,
			Size:          pos.Size,
			EntryPrice:    pos.EntryPrice,
			UnrealizedPnL: pnl,
			PnLPercent:    pnlPercent,
		}

		queue.Positions = append(queue.Positions, adlPos)
		queue.TotalSize = queue.TotalSize.Add(pos.Size)
	}

	// Sort by PnL percentage (highest profit first - they get deleveraged first)
	sort.Slice(queue.Positions, func(i, j int) bool {
		return queue.Positions[i].PnLPercent.GT(queue.Positions[j].PnLPercent)
	})

	// Assign rankings
	for i, pos := range queue.Positions {
		pos.ADLRanking = i + 1
	}

	return queue
}

// ============ ADL Execution ============

// ExecuteADL executes Auto-Deleveraging to cover a deficit
func (k *Keeper) ExecuteADL(ctx sdk.Context, marketID string, deficit math.LegacyDec, reason types.ADLTriggerReason) (*types.ADLResult, error) {
	logger := k.Logger()
	config := k.GetADLConfig(ctx)

	if !config.Enabled {
		return nil, fmt.Errorf("ADL is disabled")
	}

	result := &types.ADLResult{
		Success:          false,
		DeficitCovered:   math.LegacyZeroDec(),
		RemainingDeficit: deficit,
		Errors:           make([]string, 0),
	}

	// Determine which side to deleverage (opposite of losing side)
	// If longs are being liquidated with deficit, deleverage profitable shorts
	// If shorts are being liquidated with deficit, deleverage profitable longs
	sides := []string{"long", "short"}

	for _, side := range sides {
		if result.RemainingDeficit.IsZero() {
			break
		}

		queue := k.BuildADLQueue(ctx, marketID, side)
		if len(queue.Positions) == 0 {
			continue
		}

		// Deleverage positions starting from most profitable
		for _, adlPos := range queue.Positions {
			if result.RemainingDeficit.IsZero() {
				break
			}

			// Calculate how much to deleverage
			maxDeleverage := adlPos.Size.Mul(config.MaxDeleverageRatio)
			if maxDeleverage.LT(config.MinPositionForADL) {
				continue
			}

			// Deleverage enough to cover remaining deficit
			priceInfo := k.perpetualKeeper.GetPrice(ctx, marketID)
			if priceInfo == nil {
				continue
			}

			// Calculate notional value
			notional := adlPos.Size.Mul(priceInfo.MarkPrice)
			deficitRatio := result.RemainingDeficit.Quo(notional)
			deleverageQty := adlPos.Size.Mul(deficitRatio)

			// Cap at max deleverage ratio
			if deleverageQty.GT(maxDeleverage) {
				deleverageQty = maxDeleverage
			}

			// Execute deleverage
			coveredAmount, err := k.deleveragePosition(ctx, adlPos, deleverageQty, priceInfo.MarkPrice)
			if err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("failed to deleverage %s: %v", adlPos.Trader, err))
				continue
			}

			result.PositionsAffected++
			result.TotalDeleveraged = result.TotalDeleveraged.Add(deleverageQty)
			result.DeficitCovered = result.DeficitCovered.Add(coveredAmount)
			result.RemainingDeficit = result.RemainingDeficit.Sub(coveredAmount)

			logger.Info("position deleveraged",
				"trader", adlPos.Trader,
				"market_id", marketID,
				"side", side,
				"quantity", deleverageQty.String(),
				"covered", coveredAmount.String(),
			)
		}
	}

	// Record ADL event
	result.EventID = k.recordADLEvent(ctx, marketID, reason, deficit, result)
	result.Success = result.DeficitCovered.IsPositive()

	// Emit event
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"adl_executed",
			sdk.NewAttribute("market_id", marketID),
			sdk.NewAttribute("reason", reason.String()),
			sdk.NewAttribute("deficit", deficit.String()),
			sdk.NewAttribute("covered", result.DeficitCovered.String()),
			sdk.NewAttribute("positions_affected", fmt.Sprintf("%d", result.PositionsAffected)),
		),
	)

	return result, nil
}

// deleveragePosition deleverages a single position
func (k *Keeper) deleveragePosition(ctx sdk.Context, adlPos *types.ADLPosition, quantity, markPrice math.LegacyDec) (math.LegacyDec, error) {
	// Get the position
	position := k.perpetualKeeper.GetPosition(ctx, adlPos.Trader, adlPos.MarketID)
	if position == nil {
		return math.LegacyZeroDec(), fmt.Errorf("position not found")
	}

	// Calculate PnL for deleveraged portion
	pnlPerUnit := markPrice.Sub(position.EntryPrice)
	if position.Side == perpetualtypes.PositionSideShort {
		pnlPerUnit = pnlPerUnit.Neg()
	}
	totalPnL := pnlPerUnit.Mul(quantity)

	// Reduce position size
	position.ReduceSize(quantity)

	// Calculate margin to release
	marginRatio := quantity.Quo(adlPos.Size)
	marginRelease := position.Margin.Mul(marginRatio)

	// Update account
	account := k.perpetualKeeper.GetAccount(ctx, adlPos.Trader)
	if account != nil {
		// Release margin and apply PnL
		account.Balance = account.Balance.Add(marginRelease).Add(totalPnL)
		account.UnlockMargin(marginRelease)
		k.perpetualKeeper.SetAccount(ctx, account)
	}

	// Save or delete position
	if position.Size.IsZero() {
		k.perpetualKeeper.DeletePosition(ctx, adlPos.Trader, adlPos.MarketID)
	} else {
		k.perpetualKeeper.SetPosition(ctx, position)
	}

	// Return the deficit coverage (realized profit from forced closure)
	// CRITICAL: Only positive PnL can cover deficit
	// With the BuildADLQueue fix, we should only get profitable positions
	// This is a defensive check in case of edge cases
	if totalPnL.IsPositive() {
		return totalPnL, nil
	}

	// If we reach here with non-positive PnL, something went wrong
	// Log warning but don't return error to avoid breaking the ADL loop
	// The position was already reduced, so we return 0 coverage
	return math.LegacyZeroDec(), nil
}

// ============ ADL Event Recording ============

func (k *Keeper) recordADLEvent(ctx sdk.Context, marketID string, reason types.ADLTriggerReason, deficit math.LegacyDec, result *types.ADLResult) string {
	eventID := k.generateADLEventID(ctx)

	fund := k.GetGlobalInsuranceFund(ctx)

	event := &types.ADLEvent{
		EventID:              eventID,
		MarketID:             marketID,
		TriggerReason:        reason,
		InsuranceFundBalance: fund.Balance,
		TotalDeficit:         deficit,
		PositionsAffected:    result.PositionsAffected,
		TotalDeleveraged:     result.TotalDeleveraged,
		Timestamp:            ctx.BlockTime(),
	}

	store := k.GetStore(ctx)
	key := append(ADLEventKeyPrefix, []byte(eventID)...)
	bz, _ := json.Marshal(event)
	store.Set(key, bz)

	return eventID
}

func (k *Keeper) generateADLEventID(ctx sdk.Context) string {
	store := k.GetStore(ctx)
	bz := store.Get(ADLEventCounterKey)
	var counter uint64
	if bz != nil {
		counter = binary.BigEndian.Uint64(bz)
	}
	counter++

	newBz := make([]byte, 8)
	binary.BigEndian.PutUint64(newBz, counter)
	store.Set(ADLEventCounterKey, newBz)

	return fmt.Sprintf("adl-%d", counter)
}

// GetADLEvents returns recent ADL events
func (k *Keeper) GetADLEvents(ctx sdk.Context, marketID string, limit int) []*types.ADLEvent {
	store := k.GetStore(ctx)
	iterator := storetypes.KVStoreReversePrefixIterator(store, ADLEventKeyPrefix)
	defer iterator.Close()

	var events []*types.ADLEvent
	count := 0
	for ; iterator.Valid() && count < limit; iterator.Next() {
		var event types.ADLEvent
		if err := json.Unmarshal(iterator.Value(), &event); err != nil {
			continue
		}
		if marketID == "" || event.MarketID == marketID {
			events = append(events, &event)
			count++
		}
	}
	return events
}

// ============ ADL Rankings Query ============

// GetADLRankings returns the current ADL rankings for a market
func (k *Keeper) GetADLRankings(ctx sdk.Context, marketID string, limit int) map[string][]*types.ADLPosition {
	rankings := make(map[string][]*types.ADLPosition)

	for _, side := range []string{"long", "short"} {
		queue := k.BuildADLQueue(ctx, marketID, side)
		if limit > 0 && len(queue.Positions) > limit {
			rankings[side] = queue.Positions[:limit]
		} else {
			rankings[side] = queue.Positions
		}
	}

	return rankings
}
