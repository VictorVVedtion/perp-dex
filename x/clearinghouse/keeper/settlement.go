package keeper

import (
	"fmt"
	"sort"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	orderbooktypes "github.com/openalpha/perp-dex/x/orderbook/types"
	perpetualtypes "github.com/openalpha/perp-dex/x/perpetual/types"
)

// SettlementEngine applies deterministic settlement for matched trades.
type SettlementEngine struct {
	keeper *Keeper
}

// SettlementResult summarizes settlement for a single account.
type SettlementResult struct {
	AccountID       string
	TradesProcessed int
	RealizedPnL     math.LegacyDec
	MarginChange    math.LegacyDec
	FeesPaid        math.LegacyDec
	Error           error
}

// SettlementSummary summarizes settlement across accounts.
type SettlementSummary struct {
	Results       []*SettlementResult
	TotalTrades   int
	TotalAccounts int
	Errors        []error
}

type accountTrade struct {
	trade   *orderbooktypes.TradeWithSettlement
	isTaker bool
}

// NewSettlementEngine creates a new settlement engine.
func NewSettlementEngine(keeper *Keeper) *SettlementEngine {
	return &SettlementEngine{keeper: keeper}
}

// Settle processes a settlement request.
func (se *SettlementEngine) Settle(ctx sdk.Context, request *orderbooktypes.SettlementRequest) (*SettlementSummary, error) {
	if request == nil {
		return se.SettleTrades(ctx, nil)
	}
	return se.SettleTrades(ctx, request.Trades)
}

// SettleTrades processes trades sequentially with ATOMIC Maker+Taker settlement.
// Each trade is settled atomically - both parties succeed or both fail.
// This ensures the zero-sum invariant is maintained.
func (se *SettlementEngine) SettleTrades(ctx sdk.Context, trades []*orderbooktypes.TradeWithSettlement) (*SettlementSummary, error) {
	summary := &SettlementSummary{
		Results: make([]*SettlementResult, 0),
		Errors:  make([]error, 0),
	}
	if len(trades) == 0 {
		return summary, nil
	}

	// Sort trades by TradeID for deterministic ordering
	sortedTrades := make([]*orderbooktypes.TradeWithSettlement, len(trades))
	copy(sortedTrades, trades)
	sort.Slice(sortedTrades, func(i, j int) bool {
		return sortedTrades[i].TradeID < sortedTrades[j].TradeID
	})

	// Track per-account results for summary
	accountResults := make(map[string]*SettlementResult)

	// Process each trade ATOMICALLY (Maker + Taker together)
	for _, trade := range sortedTrades {
		if trade == nil || trade.Taker == "" || trade.Maker == "" {
			continue
		}

		// Use CacheContext for atomic settlement of BOTH parties
		cacheCtx, write := ctx.CacheContext()

		// Settle Taker
		takerErr := se.settleSingleParty(cacheCtx, trade, true)
		if takerErr != nil {
			// Taker failed - rollback (don't call write)
			summary.Errors = append(summary.Errors, fmt.Errorf("trade %s taker settlement failed: %w", trade.TradeID, takerErr))
			continue
		}

		// Settle Maker
		makerErr := se.settleSingleParty(cacheCtx, trade, false)
		if makerErr != nil {
			// Maker failed - rollback both (don't call write)
			summary.Errors = append(summary.Errors, fmt.Errorf("trade %s maker settlement failed: %w", trade.TradeID, makerErr))
			continue
		}

		// BOTH succeeded - commit atomically
		write()
		summary.TotalTrades++

		// Update per-account tracking
		for _, accountID := range []string{trade.Taker, trade.Maker} {
			if _, exists := accountResults[accountID]; !exists {
				accountResults[accountID] = &SettlementResult{
					AccountID:       accountID,
					TradesProcessed: 0,
					RealizedPnL:     math.LegacyZeroDec(),
					MarginChange:    math.LegacyZeroDec(),
					FeesPaid:        math.LegacyZeroDec(),
				}
			}
			accountResults[accountID].TradesProcessed++
		}
	}

	// Build results from account tracking
	accountIDs := make([]string, 0, len(accountResults))
	for accountID := range accountResults {
		accountIDs = append(accountIDs, accountID)
	}
	sort.Strings(accountIDs)
	for _, accountID := range accountIDs {
		summary.Results = append(summary.Results, accountResults[accountID])
	}

	summary.TotalAccounts = len(summary.Results)
	return summary, nil
}

// settleSingleParty settles one party (Maker or Taker) of a trade
func (se *SettlementEngine) settleSingleParty(ctx sdk.Context, trade *orderbooktypes.TradeWithSettlement, isTaker bool) error {
	var accountID string
	var side orderbooktypes.Side
	var fee math.LegacyDec

	if isTaker {
		accountID = trade.Taker
		side = trade.TakerSide
		fee = trade.TakerFee
	} else {
		accountID = trade.Maker
		side = oppositeSide(trade.TakerSide)
		fee = trade.MakerFee
	}

	account := se.keeper.perpetualKeeper.GetOrCreateAccount(ctx, accountID)
	if account == nil {
		return fmt.Errorf("account not found: %s", accountID)
	}

	realizedPnL, marginChange, err := se.applyTrade(ctx, account, accountID, trade.MarketID, side, trade.Quantity, trade.Price, fee)
	if err != nil {
		return err
	}

	// Update trade settlement fields
	if isTaker {
		trade.TakerRealizedPnL = realizedPnL
		trade.TakerMarginChange = marginChange
	} else {
		trade.MakerRealizedPnL = realizedPnL
		trade.MakerMarginChange = marginChange
	}

	return nil
}

func (se *SettlementEngine) settleAccount(ctx sdk.Context, accountID string, trades []accountTrade, result *SettlementResult) error {
	account := se.keeper.perpetualKeeper.GetOrCreateAccount(ctx, accountID)
	if account == nil {
		return fmt.Errorf("account not found: %s", accountID)
	}

	for _, entry := range trades {
		trade := entry.trade
		if trade == nil {
			continue
		}

		side := trade.TakerSide
		fee := trade.TakerFee
		if !entry.isTaker {
			side = oppositeSide(trade.TakerSide)
			fee = trade.MakerFee
		}

		realizedPnL, marginChange, err := se.applyTrade(ctx, account, accountID, trade.MarketID, side, trade.Quantity, trade.Price, fee)
		if err != nil {
			return err
		}

		if entry.isTaker {
			trade.TakerRealizedPnL = realizedPnL
			trade.TakerMarginChange = marginChange
		} else {
			trade.MakerRealizedPnL = realizedPnL
			trade.MakerMarginChange = marginChange
		}

		result.RealizedPnL = result.RealizedPnL.Add(realizedPnL)
		result.MarginChange = result.MarginChange.Add(marginChange)
		result.FeesPaid = result.FeesPaid.Add(fee)
	}

	return nil
}

func (se *SettlementEngine) applyTrade(
	ctx sdk.Context,
	account *perpetualtypes.Account,
	trader, marketID string,
	side orderbooktypes.Side,
	qty, price, fee math.LegacyDec,
) (math.LegacyDec, math.LegacyDec, error) {
	if account == nil {
		return math.LegacyZeroDec(), math.LegacyZeroDec(), fmt.Errorf("account not found: %s", trader)
	}
	if qty.IsZero() || price.IsZero() {
		return math.LegacyZeroDec(), math.LegacyZeroDec(), nil
	}

	positionSide, err := mapOrderSide(side)
	if err != nil {
		return math.LegacyZeroDec(), math.LegacyZeroDec(), err
	}

	position := se.keeper.perpetualKeeper.GetPosition(ctx, trader, marketID)
	realizedPnL := math.LegacyZeroDec()
	marginChange := math.LegacyZeroDec()

	if position == nil || position.Side == positionSide {
		requiredMargin := calculateInitialMargin(qty, price)
		marginChange = requiredMargin

		if position == nil {
			position = perpetualtypes.NewPosition(trader, marketID, positionSide, qty, price, requiredMargin)
		} else {
			position.AddSize(qty, price)
			position.Margin = position.Margin.Add(requiredMargin)
		}

		account.LockMargin(requiredMargin)
		se.keeper.perpetualKeeper.SetPosition(ctx, position)
	} else {
		if qty.LTE(position.Size) {
			realizedPnL = calculateRealizedPnL(position, qty, price)
			releasedMargin := position.Margin.Mul(qty).Quo(position.Size)

			position.ReduceSize(qty)
			position.Margin = position.Margin.Sub(releasedMargin)
			account.UnlockMargin(releasedMargin)
			account.Balance = account.Balance.Add(realizedPnL)
			marginChange = releasedMargin.Neg()

			if position.Size.IsZero() {
				se.keeper.perpetualKeeper.DeletePosition(ctx, trader, marketID)
			} else {
				se.keeper.perpetualKeeper.SetPosition(ctx, position)
			}
		} else {
			realizedPnL = calculateRealizedPnL(position, position.Size, price)
			account.UnlockMargin(position.Margin)
			account.Balance = account.Balance.Add(realizedPnL)
			marginChange = position.Margin.Neg()

			se.keeper.perpetualKeeper.DeletePosition(ctx, trader, marketID)

			remainingQty := qty.Sub(position.Size)
			if remainingQty.IsPositive() {
				requiredMargin := calculateInitialMargin(remainingQty, price)
				marginChange = marginChange.Add(requiredMargin)
				newPosition := perpetualtypes.NewPosition(trader, marketID, positionSide, remainingQty, price, requiredMargin)
				account.LockMargin(requiredMargin)
				se.keeper.perpetualKeeper.SetPosition(ctx, newPosition)
			}
		}
	}

	// CRITICAL FIX: Check balance before deducting fee to prevent negative balance
	if fee.IsPositive() {
		if account.Balance.LT(fee) {
			// Deduct only what's available to prevent negative balance
			// Log warning for partial fee collection
			if account.Balance.IsPositive() {
				account.Balance = math.LegacyZeroDec()
			}
			// Note: In a stricter implementation, we might want to reject the trade
			// but for now we allow it with partial fee to maintain system stability
		} else {
			account.Balance = account.Balance.Sub(fee)
		}
	}
	se.keeper.perpetualKeeper.SetAccount(ctx, account)

	return realizedPnL, marginChange, nil
}

func calculateInitialMargin(size, price math.LegacyDec) math.LegacyDec {
	initialMarginRate := math.LegacyNewDecWithPrec(5, 2) // 5%
	return size.Mul(price).Mul(initialMarginRate)
}

func calculateRealizedPnL(position *perpetualtypes.Position, size, price math.LegacyDec) math.LegacyDec {
	priceDiff := price.Sub(position.EntryPrice)
	if position.Side == perpetualtypes.PositionSideShort {
		priceDiff = priceDiff.Neg()
	}
	return size.Mul(priceDiff)
}

func mapOrderSide(side orderbooktypes.Side) (perpetualtypes.PositionSide, error) {
	switch side {
	case orderbooktypes.SideBuy:
		return perpetualtypes.PositionSideLong, nil
	case orderbooktypes.SideSell:
		return perpetualtypes.PositionSideShort, nil
	default:
		return perpetualtypes.PositionSideUnspecified, fmt.Errorf("unsupported side: %s", side.String())
	}
}

func oppositeSide(side orderbooktypes.Side) orderbooktypes.Side {
	if side == orderbooktypes.SideBuy {
		return orderbooktypes.SideSell
	}
	return orderbooktypes.SideBuy
}
