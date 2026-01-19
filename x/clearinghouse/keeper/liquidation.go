package keeper

import (
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/openalpha/perp-dex/x/clearinghouse/types"
	perpetualtypes "github.com/openalpha/perp-dex/x/perpetual/types"
)

// LiquidationStats contains performance statistics for liquidation processing
type LiquidationStats struct {
	LiquidationsCount int
	TotalVolume       math.LegacyDec
	TotalPenalties    math.LegacyDec
}

// LiquidationEngine handles the liquidation process
type LiquidationEngine struct {
	keeper *Keeper
}

// NewLiquidationEngine creates a new liquidation engine
func NewLiquidationEngine(keeper *Keeper) *LiquidationEngine {
	return &LiquidationEngine{keeper: keeper}
}

// LiquidationResult contains the result of a liquidation
type LiquidationResult struct {
	LiquidationID     string
	LiquidatedSize    math.LegacyDec
	LiquidationPrice  math.LegacyDec
	PenaltyPaid       math.LegacyDec
	LiquidatorReward  math.LegacyDec // Liquidator reward (30% of penalty)
	InsuranceFundFee  math.LegacyDec // Insurance fund share (70% of penalty)
	Success           bool
	Error             error
}

// Liquidator reward rate (30% of penalty)
var LiquidatorRewardRate = math.LegacyNewDecWithPrec(3, 1) // 0.3 = 30%

// CheckAndLiquidate checks if a position should be liquidated and executes if needed
func (le *LiquidationEngine) CheckAndLiquidate(ctx sdk.Context, trader, marketID string) (*LiquidationResult, error) {
	// Get position
	position := le.keeper.perpetualKeeper.GetPosition(ctx, trader, marketID)
	if position == nil {
		return nil, types.ErrPositionNotFound
	}

	// Get current price
	priceInfo := le.keeper.perpetualKeeper.GetPrice(ctx, marketID)
	if priceInfo == nil {
		return nil, types.ErrPositionNotFound
	}

	markPrice := priceInfo.MarkPrice

	// Check if position is healthy
	if position.IsHealthy(markPrice) {
		return nil, types.ErrPositionHealthy
	}

	// Execute liquidation
	return le.ExecuteLiquidation(ctx, position, markPrice)
}

// ExecuteLiquidation executes the liquidation of an unhealthy position
// Updated with liquidator reward mechanism (30% of penalty to liquidator, 70% to insurance fund)
func (le *LiquidationEngine) ExecuteLiquidation(
	ctx sdk.Context,
	position *perpetualtypes.Position,
	markPrice math.LegacyDec,
) (*LiquidationResult, error) {
	return le.ExecuteLiquidationWithReward(ctx, position, markPrice, "")
}

// ExecuteLiquidationWithReward executes liquidation with optional liquidator reward
// If liquidator is empty, the reward goes to the insurance fund
func (le *LiquidationEngine) ExecuteLiquidationWithReward(
	ctx sdk.Context,
	position *perpetualtypes.Position,
	markPrice math.LegacyDec,
	liquidator string,
) (*LiquidationResult, error) {
	// Calculate liquidation penalty (1% of notional value)
	penaltyRate := math.LegacyNewDecWithPrec(1, 2) // 1%
	notionalValue := position.Size.Mul(markPrice)
	penalty := notionalValue.Mul(penaltyRate)

	// Calculate liquidator reward and insurance fund share
	// Liquidator gets 30% of penalty, insurance fund gets 70%
	liquidatorReward := penalty.Mul(LiquidatorRewardRate)        // 30%
	insuranceFundShare := penalty.Sub(liquidatorReward)          // 70%

	// Calculate margin deficit (using 2.5% maintenance margin rate)
	maintenanceMarginRate := math.LegacyNewDecWithPrec(25, 3) // 2.5% (updated from 5%)
	maintenanceMargin := position.Size.Mul(markPrice).Mul(maintenanceMarginRate)
	unrealizedPnL := position.CalculateUnrealizedPnL(markPrice)
	equity := position.Margin.Add(unrealizedPnL)
	marginDeficit := maintenanceMargin.Sub(equity)
	if marginDeficit.IsNegative() {
		marginDeficit = math.LegacyZeroDec()
	}

	// Generate liquidation ID
	liquidationID := le.keeper.generateLiquidationID(ctx)

	// Create liquidation record
	liquidation := types.NewLiquidation(
		liquidationID,
		position.Trader,
		position.MarketID,
		position.Size,
		position.EntryPrice,
		markPrice,
		position.LiquidationPrice,
		marginDeficit,
		penalty,
	)

	// Close the position at mark price
	// In production, this would create a market order to close the position
	// For MVP, we directly close at mark price

	// Calculate realized PnL
	priceDiff := markPrice.Sub(position.EntryPrice)
	if position.Side == perpetualtypes.PositionSideShort {
		priceDiff = priceDiff.Neg()
	}
	realizedPnL := position.Size.Mul(priceDiff)

	// Update trader's account
	account := le.keeper.perpetualKeeper.GetAccount(ctx, position.Trader)
	if account != nil {
		// Return margin minus losses and penalty
		returnAmount := position.Margin.Add(realizedPnL).Sub(penalty)
		if returnAmount.IsNegative() {
			returnAmount = math.LegacyZeroDec()
		}

		account.Balance = account.Balance.Add(returnAmount)
		account.LockedMargin = account.LockedMargin.Sub(position.Margin)
		if account.LockedMargin.IsNegative() {
			account.LockedMargin = math.LegacyZeroDec()
		}
		le.keeper.perpetualKeeper.SetAccount(ctx, account)
	}

	// Distribute liquidator reward
	if liquidator != "" && liquidatorReward.IsPositive() {
		liquidatorAccount := le.keeper.perpetualKeeper.GetOrCreateAccount(ctx, liquidator)
		liquidatorAccount.Balance = liquidatorAccount.Balance.Add(liquidatorReward)
		le.keeper.perpetualKeeper.SetAccount(ctx, liquidatorAccount)

		le.keeper.Logger().Info("Liquidator reward distributed",
			"liquidator", liquidator,
			"reward", liquidatorReward.String(),
		)
	} else {
		// If no liquidator specified, entire penalty goes to insurance fund
		insuranceFundShare = penalty
		liquidatorReward = math.LegacyZeroDec()
	}

	// Transfer to insurance fund
	// TODO: Implement insurance fund keeper integration
	// le.keeper.insuranceKeeper.AddToFund(ctx, position.MarketID, insuranceFundShare)

	// Delete the position
	le.keeper.perpetualKeeper.DeletePosition(ctx, position.Trader, position.MarketID)

	// Mark liquidation as executed
	liquidation.Status = types.LiquidationStatusExecuted
	le.keeper.SetLiquidation(ctx, liquidation)

	// Emit liquidation event
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"liquidation",
			sdk.NewAttribute("liquidation_id", liquidationID),
			sdk.NewAttribute("trader", position.Trader),
			sdk.NewAttribute("market_id", position.MarketID),
			sdk.NewAttribute("position_size", position.Size.String()),
			sdk.NewAttribute("entry_price", position.EntryPrice.String()),
			sdk.NewAttribute("mark_price", markPrice.String()),
			sdk.NewAttribute("realized_pnl", realizedPnL.String()),
			sdk.NewAttribute("penalty", penalty.String()),
			sdk.NewAttribute("liquidator", liquidator),
			sdk.NewAttribute("liquidator_reward", liquidatorReward.String()),
			sdk.NewAttribute("insurance_fund_share", insuranceFundShare.String()),
		),
	)

	le.keeper.Logger().Info("Position liquidated",
		"trader", position.Trader,
		"market", position.MarketID,
		"size", position.Size.String(),
		"mark_price", markPrice.String(),
		"liquidator_reward", liquidatorReward.String(),
		"insurance_fund_share", insuranceFundShare.String(),
	)

	return &LiquidationResult{
		LiquidationID:    liquidationID,
		LiquidatedSize:   position.Size,
		LiquidationPrice: markPrice,
		PenaltyPaid:      penalty,
		LiquidatorReward: liquidatorReward,
		InsuranceFundFee: insuranceFundShare,
		Success:          true,
	}, nil
}

// EndBlockLiquidations checks all positions and liquidates unhealthy ones
// Called at the end of each block
// Returns statistics about liquidations performed
func (le *LiquidationEngine) EndBlockLiquidations(ctx sdk.Context) LiquidationStats {
	stats := LiquidationStats{
		TotalVolume:    math.LegacyZeroDec(),
		TotalPenalties: math.LegacyZeroDec(),
	}

	// Get all unhealthy positions
	unhealthyPositions := le.keeper.GetUnhealthyPositions(ctx)

	for _, health := range unhealthyPositions {
		position := le.keeper.perpetualKeeper.GetPosition(ctx, health.Trader, health.MarketID)
		if position == nil {
			continue
		}

		priceInfo := le.keeper.perpetualKeeper.GetPrice(ctx, health.MarketID)
		if priceInfo == nil {
			continue
		}

		// Execute liquidation
		result, err := le.ExecuteLiquidation(ctx, position, priceInfo.MarkPrice)
		if err != nil {
			le.keeper.Logger().Error("Failed to liquidate position",
				"trader", health.Trader,
				"market", health.MarketID,
				"error", err,
			)
			continue
		}

		// Update statistics
		stats.LiquidationsCount++
		liquidationVolume := result.LiquidatedSize.Mul(result.LiquidationPrice)
		stats.TotalVolume = stats.TotalVolume.Add(liquidationVolume)
		stats.TotalPenalties = stats.TotalPenalties.Add(result.PenaltyPaid)

		le.keeper.Logger().Info("Auto-liquidation executed",
			"liquidation_id", result.LiquidationID,
			"trader", health.Trader,
			"market", health.MarketID,
		)
	}

	return stats
}

// TriggerLiquidation allows anyone to trigger liquidation of an unhealthy position
// Liquidator receives 30% of the liquidation penalty as incentive
func (le *LiquidationEngine) TriggerLiquidation(
	ctx sdk.Context,
	liquidator string,
	trader string,
	marketID string,
) (*LiquidationResult, error) {
	// Get position
	position := le.keeper.perpetualKeeper.GetPosition(ctx, trader, marketID)
	if position == nil {
		return nil, types.ErrPositionNotFound
	}

	// Get current price
	priceInfo := le.keeper.perpetualKeeper.GetPrice(ctx, marketID)
	if priceInfo == nil {
		return nil, types.ErrPositionNotFound
	}

	markPrice := priceInfo.MarkPrice

	// Check if position is healthy
	if position.IsHealthy(markPrice) {
		return nil, types.ErrPositionHealthy
	}

	// Execute liquidation with reward to the liquidator
	result, err := le.ExecuteLiquidationWithReward(ctx, position, markPrice, liquidator)
	if err != nil {
		return nil, err
	}

	le.keeper.Logger().Info("Liquidation triggered by external party",
		"liquidator", liquidator,
		"trader", trader,
		"market", marketID,
		"reward", result.LiquidatorReward.String(),
	)

	return result, nil
}

// CheckAndLiquidateAll checks all positions for a trader and liquidates unhealthy ones (cascade liquidation)
// This is used for cross-margin mode where one liquidation may trigger others
func (le *LiquidationEngine) CheckAndLiquidateAll(ctx sdk.Context, trader string) ([]*LiquidationResult, error) {
	positions := le.keeper.perpetualKeeper.GetPositionsByTrader(ctx, trader)
	var results []*LiquidationResult

	for _, position := range positions {
		priceInfo := le.keeper.perpetualKeeper.GetPrice(ctx, position.MarketID)
		if priceInfo == nil {
			continue
		}

		markPrice := priceInfo.MarkPrice

		// Check if position is healthy
		if !position.IsHealthy(markPrice) {
			result, err := le.ExecuteLiquidation(ctx, position, markPrice)
			if err != nil {
				le.keeper.Logger().Error("Failed to liquidate position in cascade",
					"trader", trader,
					"market", position.MarketID,
					"error", err,
				)
				continue
			}
			results = append(results, result)
		}
	}

	if len(results) > 0 {
		le.keeper.Logger().Info("Cascade liquidation completed",
			"trader", trader,
			"positions_liquidated", len(results),
		)
	}

	return results, nil
}
