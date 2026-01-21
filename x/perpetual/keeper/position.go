package keeper

import (
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/openalpha/perp-dex/x/perpetual/types"
)

// PositionManager handles position operations
type PositionManager struct {
	keeper        *Keeper
	marginChecker *MarginChecker
}

// NewPositionManager creates a new position manager
func NewPositionManager(keeper *Keeper) *PositionManager {
	return &PositionManager{
		keeper:        keeper,
		marginChecker: NewMarginChecker(keeper),
	}
}

// OpenPosition opens a new position or adds to an existing one
func (pm *PositionManager) OpenPosition(
	ctx sdk.Context,
	trader string,
	marketID string,
	side types.PositionSide,
	size math.LegacyDec,
	entryPrice math.LegacyDec,
) (*types.Position, error) {
	// Validate market
	market := pm.keeper.GetMarket(ctx, marketID)
	if market == nil {
		return nil, types.ErrMarketNotFound
	}
	if !market.IsActive {
		return nil, types.ErrMarketNotActive
	}

	// Validate size
	if size.IsZero() || size.IsNegative() {
		return nil, types.ErrInvalidQuantity
	}

	// Check initial margin requirement
	if err := pm.marginChecker.CheckInitialMarginRequirement(ctx, trader, marketID, size, entryPrice); err != nil {
		return nil, err
	}

	// Calculate required margin
	requiredMargin := pm.marginChecker.CalculateInitialMargin(size, entryPrice)

	// Get existing position
	existingPosition := pm.keeper.GetPosition(ctx, trader, marketID)

	// Get account
	account := pm.keeper.GetOrCreateAccount(ctx, trader)

	var position *types.Position

	if existingPosition == nil {
		// Create new position
		position = types.NewPosition(trader, marketID, side, size, entryPrice, requiredMargin)
	} else if existingPosition.Side == side {
		// Add to existing position (same side)
		existingPosition.AddSize(size, entryPrice)
		existingPosition.Margin = existingPosition.Margin.Add(requiredMargin)
		position = existingPosition
	} else {
		// Opposite side - reduce or flip position
		if size.LTE(existingPosition.Size) {
			// Reduce position
			_, _, _ = pm.ReducePosition(ctx, trader, marketID, size)
			return pm.keeper.GetPosition(ctx, trader, marketID), nil
		} else {
			// Close existing and open opposite
			_, _ = pm.ClosePosition(ctx, trader, marketID, entryPrice)
			remainingSize := size.Sub(existingPosition.Size)
			remainingMargin := pm.marginChecker.CalculateInitialMargin(remainingSize, entryPrice)
			position = types.NewPosition(trader, marketID, side, remainingSize, entryPrice, remainingMargin)
		}
	}

	// Lock margin
	account.LockMargin(requiredMargin)
	pm.keeper.SetAccount(ctx, account)

	// Save position
	pm.keeper.SetPosition(ctx, position)

	// Emit event
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"open_position",
			sdk.NewAttribute("trader", trader),
			sdk.NewAttribute("market_id", marketID),
			sdk.NewAttribute("side", side.String()),
			sdk.NewAttribute("size", size.String()),
			sdk.NewAttribute("entry_price", entryPrice.String()),
			sdk.NewAttribute("margin", requiredMargin.String()),
		),
	)

	return position, nil
}

// ReducePosition reduces the size of an existing position
func (pm *PositionManager) ReducePosition(
	ctx sdk.Context,
	trader string,
	marketID string,
	reduceSize math.LegacyDec,
) (*types.Position, math.LegacyDec, error) {
	position := pm.keeper.GetPosition(ctx, trader, marketID)
	if position == nil {
		return nil, math.LegacyDec{}, types.ErrPositionNotFound
	}

	if reduceSize.GT(position.Size) {
		return nil, math.LegacyDec{}, types.ErrCannotReducePosition
	}

	// Get current price for PnL calculation
	priceInfo := pm.keeper.GetPrice(ctx, marketID)
	if priceInfo == nil {
		return nil, math.LegacyDec{}, types.ErrMarketNotFound
	}
	closePrice := priceInfo.MarkPrice

	// Calculate realized PnL for the reduced portion
	priceDiff := closePrice.Sub(position.EntryPrice)
	if position.Side == types.PositionSideShort {
		priceDiff = priceDiff.Neg()
	}
	realizedPnL := reduceSize.Mul(priceDiff)

	// Calculate released margin (proportional)
	releasedMargin := position.Margin.Mul(reduceSize).Quo(position.Size)

	// Update position
	position.ReduceSize(reduceSize)
	position.Margin = position.Margin.Sub(releasedMargin)

	// Update account
	account := pm.keeper.GetAccount(ctx, trader)
	account.UnlockMargin(releasedMargin)
	account.Balance = account.Balance.Add(realizedPnL)
	pm.keeper.SetAccount(ctx, account)

	// Save or delete position
	if position.Size.IsZero() {
		pm.keeper.DeletePosition(ctx, trader, marketID)
		position = nil
	} else {
		pm.keeper.SetPosition(ctx, position)
	}

	// Emit event
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"reduce_position",
			sdk.NewAttribute("trader", trader),
			sdk.NewAttribute("market_id", marketID),
			sdk.NewAttribute("reduce_size", reduceSize.String()),
			sdk.NewAttribute("close_price", closePrice.String()),
			sdk.NewAttribute("realized_pnl", realizedPnL.String()),
		),
	)

	return position, realizedPnL, nil
}

// ClosePosition closes an entire position
func (pm *PositionManager) ClosePosition(
	ctx sdk.Context,
	trader string,
	marketID string,
	closePrice math.LegacyDec,
) (math.LegacyDec, error) {
	position := pm.keeper.GetPosition(ctx, trader, marketID)
	if position == nil {
		return math.LegacyDec{}, types.ErrPositionNotFound
	}

	// Calculate realized PnL
	priceDiff := closePrice.Sub(position.EntryPrice)
	if position.Side == types.PositionSideShort {
		priceDiff = priceDiff.Neg()
	}
	realizedPnL := position.Size.Mul(priceDiff)

	// Update account
	account := pm.keeper.GetAccount(ctx, trader)
	account.UnlockMargin(position.Margin)
	account.Balance = account.Balance.Add(realizedPnL)
	pm.keeper.SetAccount(ctx, account)

	// Delete position
	pm.keeper.DeletePosition(ctx, trader, marketID)

	// Emit event
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"close_position",
			sdk.NewAttribute("trader", trader),
			sdk.NewAttribute("market_id", marketID),
			sdk.NewAttribute("size", position.Size.String()),
			sdk.NewAttribute("entry_price", position.EntryPrice.String()),
			sdk.NewAttribute("close_price", closePrice.String()),
			sdk.NewAttribute("realized_pnl", realizedPnL.String()),
		),
	)

	return realizedPnL, nil
}

// UpdatePositionFromTrade updates position based on a trade execution
// Called by the orderbook module after a trade is matched
func (pm *PositionManager) UpdatePositionFromTrade(
	ctx sdk.Context,
	trader string,
	marketID string,
	isBuy bool,
	size math.LegacyDec,
	price math.LegacyDec,
	fee math.LegacyDec,
) error {
	// Determine position side based on trade direction
	var side types.PositionSide
	if isBuy {
		side = types.PositionSideLong
	} else {
		side = types.PositionSideShort
	}

	// Check for existing position
	existingPosition := pm.keeper.GetPosition(ctx, trader, marketID)

	if existingPosition == nil || existingPosition.Side == side {
		// Open or add to position
		_, err := pm.OpenPosition(ctx, trader, marketID, side, size, price)
		if err != nil {
			return err
		}
	} else {
		// Opposite trade - reduce or close position
		if size.LTE(existingPosition.Size) {
			_, _, err := pm.ReducePosition(ctx, trader, marketID, size)
			if err != nil {
				return err
			}
		} else {
			// Close and open opposite
			_, err := pm.ClosePosition(ctx, trader, marketID, price)
			if err != nil {
				return err
			}
			remainingSize := size.Sub(existingPosition.Size)
			_, err = pm.OpenPosition(ctx, trader, marketID, side, remainingSize, price)
			if err != nil {
				return err
			}
		}
	}

	// Deduct fee from account with balance check
	account := pm.keeper.GetAccount(ctx, trader)
	if account != nil && fee.IsPositive() {
		// CRITICAL: Check if account has sufficient balance for fee
		if account.Balance.LT(fee) {
			// If insufficient balance, deduct what's available and log warning
			// This prevents negative balance while allowing trade to complete
			availableFee := account.Balance
			if availableFee.IsPositive() {
				account.Balance = math.LegacyZeroDec()
				pm.keeper.SetAccount(ctx, account)
				// Emit warning event for partial fee collection
				ctx.EventManager().EmitEvent(
					sdk.NewEvent(
						"partial_fee_collected",
						sdk.NewAttribute("trader", trader),
						sdk.NewAttribute("expected_fee", fee.String()),
						sdk.NewAttribute("collected_fee", availableFee.String()),
					),
				)
			}
		} else {
			account.Balance = account.Balance.Sub(fee)
			pm.keeper.SetAccount(ctx, account)
		}
	}

	return nil
}
