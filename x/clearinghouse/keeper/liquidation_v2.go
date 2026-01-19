package keeper

import (
	"fmt"
	"time"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/openalpha/perp-dex/x/clearinghouse/types"
	perpetualtypes "github.com/openalpha/perp-dex/x/perpetual/types"
)

// LiquidationEngineV2 implements the three-tier liquidation mechanism
// aligned with Hyperliquid's liquidation system
type LiquidationEngineV2 struct {
	keeper *Keeper
	config types.LiquidationConfig

	// In-memory cache of liquidation states (for positions being liquidated)
	liquidationStates map[string]*types.LiquidationState
}

// NewLiquidationEngineV2 creates a new V2 liquidation engine
func NewLiquidationEngineV2(keeper *Keeper) *LiquidationEngineV2 {
	return &LiquidationEngineV2{
		keeper:            keeper,
		config:            types.DefaultLiquidationConfig(),
		liquidationStates: make(map[string]*types.LiquidationState),
	}
}

// NewLiquidationEngineV2WithConfig creates a new V2 liquidation engine with custom config
func NewLiquidationEngineV2WithConfig(keeper *Keeper, config types.LiquidationConfig) *LiquidationEngineV2 {
	return &LiquidationEngineV2{
		keeper:            keeper,
		config:            config,
		liquidationStates: make(map[string]*types.LiquidationState),
	}
}

// GetConfig returns the current liquidation configuration
func (le *LiquidationEngineV2) GetConfig() types.LiquidationConfig {
	return le.config
}

// UpdateConfig updates the liquidation configuration
func (le *LiquidationEngineV2) UpdateConfig(config types.LiquidationConfig) {
	le.config = config
}

// LiquidationResultV2 contains the detailed result of a liquidation
type LiquidationResultV2 struct {
	LiquidationID     string
	Trader            string
	MarketID          string
	Tier              types.LiquidationTier
	LiquidatedSize    math.LegacyDec
	RemainingSize     math.LegacyDec
	LiquidationPrice  math.LegacyDec
	PenaltyPaid       math.LegacyDec
	LiquidatorReward  math.LegacyDec
	InsuranceFundFee  math.LegacyDec
	IsPartial         bool
	CooldownEndTime   time.Time
	Success           bool
	Error             error
	HealthAfter       *types.PositionHealthV2
}

// AssessPositionHealth assesses the health of a position and determines liquidation needs
func (le *LiquidationEngineV2) AssessPositionHealth(
	ctx sdk.Context,
	trader, marketID string,
) (*types.PositionHealthV2, error) {
	// Get position
	position := le.keeper.perpetualKeeper.GetPosition(ctx, trader, marketID)
	if position == nil {
		return nil, types.ErrPositionNotFound
	}

	// Get current price
	priceInfo := le.keeper.perpetualKeeper.GetPrice(ctx, marketID)
	if priceInfo == nil {
		return nil, fmt.Errorf("price not found for market %s", marketID)
	}

	// Use configured maintenance margin rate
	// In production, this would be fetched from market config
	maintenanceMarginRate := le.config.MinMaintenanceMarginRate

	health := types.NewPositionHealthV2(
		trader,
		marketID,
		position.Size,
		position.EntryPrice,
		priceInfo.MarkPrice,
		position.Margin,
		maintenanceMarginRate,
		le.config.LargePositionThreshold,
	)

	return health, nil
}

// ProcessLiquidation processes a liquidation using the three-tier mechanism
func (le *LiquidationEngineV2) ProcessLiquidation(
	ctx sdk.Context,
	trader, marketID string,
	liquidator string,
) (*LiquidationResultV2, error) {
	// Assess position health
	health, err := le.AssessPositionHealth(ctx, trader, marketID)
	if err != nil {
		return nil, err
	}

	// Check if liquidation is needed
	if !health.NeedsLiquidation() {
		return nil, types.ErrPositionHealthy
	}

	// Get or create liquidation state
	stateKey := fmt.Sprintf("%s:%s", trader, marketID)
	state, exists := le.liquidationStates[stateKey]
	if !exists {
		state = types.NewLiquidationState(stateKey, trader, marketID, health.PositionSize)
		le.liquidationStates[stateKey] = state
	}

	// Check cooldown
	currentTime := ctx.BlockTime()
	if !state.CanLiquidate(currentTime) {
		return &LiquidationResultV2{
			Trader:          trader,
			MarketID:        marketID,
			Success:         false,
			Error:           fmt.Errorf("position in cooldown until %v", state.CooldownEndTime),
			CooldownEndTime: state.CooldownEndTime,
		}, nil
	}

	// Determine liquidation tier and execute
	tier := health.RecommendedTier

	// For backstop tier, check if Vault has capacity
	if tier == types.TierBackstopLiquidation {
		return le.executeBackstopLiquidation(ctx, health, state, liquidator)
	}

	// For large positions, use partial liquidation
	if health.IsLargePosition && tier != types.TierBackstopLiquidation {
		return le.executePartialLiquidation(ctx, health, state, liquidator)
	}

	// Standard market order liquidation
	return le.executeMarketOrderLiquidation(ctx, health, state, liquidator)
}

// executeMarketOrderLiquidation executes Tier 1 market order liquidation
func (le *LiquidationEngineV2) executeMarketOrderLiquidation(
	ctx sdk.Context,
	health *types.PositionHealthV2,
	state *types.LiquidationState,
	liquidator string,
) (*LiquidationResultV2, error) {
	position := le.keeper.perpetualKeeper.GetPosition(ctx, health.Trader, health.MarketID)
	if position == nil {
		return nil, types.ErrPositionNotFound
	}

	// Calculate liquidation amounts
	liquidatedSize := state.RemainingSize
	notionalValue := liquidatedSize.Mul(health.MarkPrice)
	penalty := notionalValue.Mul(le.config.LiquidationPenaltyRate)
	liquidatorReward := penalty.Mul(le.config.LiquidatorRewardRate)
	insuranceFundFee := penalty.Sub(liquidatorReward)

	// Generate liquidation ID
	liquidationID := le.keeper.generateLiquidationID(ctx)

	// Calculate realized PnL
	priceDiff := health.MarkPrice.Sub(position.EntryPrice)
	if position.Side == perpetualtypes.PositionSideShort {
		priceDiff = priceDiff.Neg()
	}
	realizedPnL := liquidatedSize.Mul(priceDiff)

	// Update trader's account
	account := le.keeper.perpetualKeeper.GetAccount(ctx, position.Trader)
	if account != nil {
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
	} else {
		insuranceFundFee = penalty
		liquidatorReward = math.LegacyZeroDec()
	}

	// Create liquidation record
	liquidation := types.NewLiquidation(
		liquidationID,
		position.Trader,
		position.MarketID,
		liquidatedSize,
		position.EntryPrice,
		health.MarkPrice,
		position.LiquidationPrice,
		math.LegacyZeroDec(),
		penalty,
	)
	liquidation.Status = types.LiquidationStatusExecuted
	le.keeper.SetLiquidation(ctx, liquidation)

	// Delete the position
	le.keeper.perpetualKeeper.DeletePosition(ctx, position.Trader, position.MarketID)

	// Update state
	state.UpdateAfterLiquidation(liquidatedSize, penalty, types.TierMarketOrder)

	// Clean up state if fully liquidated
	if state.IsFullyLiquidated() {
		delete(le.liquidationStates, fmt.Sprintf("%s:%s", health.Trader, health.MarketID))
	}

	// Emit event
	le.emitLiquidationEvent(ctx, liquidationID, position.Trader, position.MarketID,
		liquidatedSize, health.MarkPrice, penalty, liquidatorReward, insuranceFundFee,
		types.TierMarketOrder, false, liquidator)

	le.keeper.Logger().Info("Tier 1 market order liquidation executed",
		"liquidation_id", liquidationID,
		"trader", position.Trader,
		"market", position.MarketID,
		"size", liquidatedSize.String(),
		"price", health.MarkPrice.String(),
	)

	return &LiquidationResultV2{
		LiquidationID:    liquidationID,
		Trader:           health.Trader,
		MarketID:         health.MarketID,
		Tier:             types.TierMarketOrder,
		LiquidatedSize:   liquidatedSize,
		RemainingSize:    math.LegacyZeroDec(),
		LiquidationPrice: health.MarkPrice,
		PenaltyPaid:      penalty,
		LiquidatorReward: liquidatorReward,
		InsuranceFundFee: insuranceFundFee,
		IsPartial:        false,
		Success:          true,
	}, nil
}

// executePartialLiquidation executes Tier 2 partial liquidation for large positions
func (le *LiquidationEngineV2) executePartialLiquidation(
	ctx sdk.Context,
	health *types.PositionHealthV2,
	state *types.LiquidationState,
	liquidator string,
) (*LiquidationResultV2, error) {
	position := le.keeper.perpetualKeeper.GetPosition(ctx, health.Trader, health.MarketID)
	if position == nil {
		return nil, types.ErrPositionNotFound
	}

	// Calculate partial liquidation size (20% of remaining)
	liquidatedSize := state.RemainingSize.Mul(le.config.PartialLiquidationRate)
	remainingAfter := state.RemainingSize.Sub(liquidatedSize)

	notionalValue := liquidatedSize.Mul(health.MarkPrice)
	penalty := notionalValue.Mul(le.config.LiquidationPenaltyRate)
	liquidatorReward := penalty.Mul(le.config.LiquidatorRewardRate)
	insuranceFundFee := penalty.Sub(liquidatorReward)

	// Generate liquidation ID
	liquidationID := le.keeper.generateLiquidationID(ctx)

	// Calculate realized PnL for the partial liquidation
	priceDiff := health.MarkPrice.Sub(position.EntryPrice)
	if position.Side == perpetualtypes.PositionSideShort {
		priceDiff = priceDiff.Neg()
	}
	realizedPnL := liquidatedSize.Mul(priceDiff)

	// Calculate margin to release (proportional)
	marginToRelease := position.Margin.Mul(liquidatedSize).Quo(position.Size)

	// Update trader's account
	account := le.keeper.perpetualKeeper.GetAccount(ctx, position.Trader)
	if account != nil {
		returnAmount := marginToRelease.Add(realizedPnL).Sub(penalty)
		if returnAmount.IsNegative() {
			returnAmount = math.LegacyZeroDec()
		}

		account.Balance = account.Balance.Add(returnAmount)
		account.LockedMargin = account.LockedMargin.Sub(marginToRelease)
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
	} else {
		insuranceFundFee = penalty
		liquidatorReward = math.LegacyZeroDec()
	}

	// Update position (reduce size and margin)
	position.Size = position.Size.Sub(liquidatedSize)
	position.Margin = position.Margin.Sub(marginToRelease)
	le.keeper.perpetualKeeper.SetPosition(ctx, position)

	// Create liquidation record
	liquidation := types.NewLiquidation(
		liquidationID,
		position.Trader,
		position.MarketID,
		liquidatedSize,
		position.EntryPrice,
		health.MarkPrice,
		position.LiquidationPrice,
		math.LegacyZeroDec(),
		penalty,
	)
	liquidation.Status = types.LiquidationStatusExecuted
	le.keeper.SetLiquidation(ctx, liquidation)

	// Update state and start cooldown
	state.UpdateAfterLiquidation(liquidatedSize, penalty, types.TierPartialLiquidation)
	state.StartCooldown(le.config.CooldownPeriod)

	// Emit event
	le.emitLiquidationEvent(ctx, liquidationID, position.Trader, position.MarketID,
		liquidatedSize, health.MarkPrice, penalty, liquidatorReward, insuranceFundFee,
		types.TierPartialLiquidation, true, liquidator)

	le.keeper.Logger().Info("Tier 2 partial liquidation executed",
		"liquidation_id", liquidationID,
		"trader", position.Trader,
		"market", position.MarketID,
		"liquidated_size", liquidatedSize.String(),
		"remaining_size", remainingAfter.String(),
		"cooldown_ends", state.CooldownEndTime.String(),
	)

	return &LiquidationResultV2{
		LiquidationID:    liquidationID,
		Trader:           health.Trader,
		MarketID:         health.MarketID,
		Tier:             types.TierPartialLiquidation,
		LiquidatedSize:   liquidatedSize,
		RemainingSize:    remainingAfter,
		LiquidationPrice: health.MarkPrice,
		PenaltyPaid:      penalty,
		LiquidatorReward: liquidatorReward,
		InsuranceFundFee: insuranceFundFee,
		IsPartial:        true,
		CooldownEndTime:  state.CooldownEndTime,
		Success:          true,
	}, nil
}

// executeBackstopLiquidation executes Tier 3 backstop liquidation via Vault
func (le *LiquidationEngineV2) executeBackstopLiquidation(
	ctx sdk.Context,
	health *types.PositionHealthV2,
	state *types.LiquidationState,
	liquidator string,
) (*LiquidationResultV2, error) {
	position := le.keeper.perpetualKeeper.GetPosition(ctx, health.Trader, health.MarketID)
	if position == nil {
		return nil, types.ErrPositionNotFound
	}

	// For backstop liquidation, the Liquidator Vault takes over the position
	// The position is transferred to the vault at a discounted price

	liquidatedSize := state.RemainingSize
	notionalValue := liquidatedSize.Mul(health.MarkPrice)
	penalty := notionalValue.Mul(le.config.LiquidationPenaltyRate)

	// In backstop liquidation, all penalty goes to insurance fund (no liquidator reward)
	// as the vault is taking on the risk
	insuranceFundFee := penalty
	liquidatorReward := math.LegacyZeroDec()

	// Generate liquidation ID
	liquidationID := le.keeper.generateLiquidationID(ctx)

	// Calculate realized PnL
	priceDiff := health.MarkPrice.Sub(position.EntryPrice)
	if position.Side == perpetualtypes.PositionSideShort {
		priceDiff = priceDiff.Neg()
	}
	realizedPnL := liquidatedSize.Mul(priceDiff)

	// Update trader's account
	account := le.keeper.perpetualKeeper.GetAccount(ctx, position.Trader)
	if account != nil {
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

	// TODO: Transfer position to Liquidator Vault
	// This would involve:
	// 1. Creating a position in the Vault's account
	// 2. The Vault would then manage the position (close it or hold it)
	// For MVP, we simply close the position

	// Create liquidation record
	liquidation := types.NewLiquidation(
		liquidationID,
		position.Trader,
		position.MarketID,
		liquidatedSize,
		position.EntryPrice,
		health.MarkPrice,
		position.LiquidationPrice,
		math.LegacyZeroDec(),
		penalty,
	)
	liquidation.Status = types.LiquidationStatusExecuted
	le.keeper.SetLiquidation(ctx, liquidation)

	// Delete the position
	le.keeper.perpetualKeeper.DeletePosition(ctx, position.Trader, position.MarketID)

	// Update state
	state.UpdateAfterLiquidation(liquidatedSize, penalty, types.TierBackstopLiquidation)
	state.IsBackstopTriggered = true

	// Clean up state
	delete(le.liquidationStates, fmt.Sprintf("%s:%s", health.Trader, health.MarketID))

	// Emit event
	le.emitLiquidationEvent(ctx, liquidationID, position.Trader, position.MarketID,
		liquidatedSize, health.MarkPrice, penalty, liquidatorReward, insuranceFundFee,
		types.TierBackstopLiquidation, false, liquidator)

	le.keeper.Logger().Info("Tier 3 backstop liquidation executed",
		"liquidation_id", liquidationID,
		"trader", position.Trader,
		"market", position.MarketID,
		"size", liquidatedSize.String(),
		"price", health.MarkPrice.String(),
	)

	return &LiquidationResultV2{
		LiquidationID:    liquidationID,
		Trader:           health.Trader,
		MarketID:         health.MarketID,
		Tier:             types.TierBackstopLiquidation,
		LiquidatedSize:   liquidatedSize,
		RemainingSize:    math.LegacyZeroDec(),
		LiquidationPrice: health.MarkPrice,
		PenaltyPaid:      penalty,
		LiquidatorReward: liquidatorReward,
		InsuranceFundFee: insuranceFundFee,
		IsPartial:        false,
		Success:          true,
	}, nil
}

// emitLiquidationEvent emits a liquidation event
func (le *LiquidationEngineV2) emitLiquidationEvent(
	ctx sdk.Context,
	liquidationID, trader, marketID string,
	size, price, penalty, liquidatorReward, insuranceFundFee math.LegacyDec,
	tier types.LiquidationTier,
	isPartial bool,
	liquidator string,
) {
	partialStr := "false"
	if isPartial {
		partialStr = "true"
	}

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"liquidation_v2",
			sdk.NewAttribute("liquidation_id", liquidationID),
			sdk.NewAttribute("trader", trader),
			sdk.NewAttribute("market_id", marketID),
			sdk.NewAttribute("tier", tier.String()),
			sdk.NewAttribute("size", size.String()),
			sdk.NewAttribute("price", price.String()),
			sdk.NewAttribute("penalty", penalty.String()),
			sdk.NewAttribute("liquidator", liquidator),
			sdk.NewAttribute("liquidator_reward", liquidatorReward.String()),
			sdk.NewAttribute("insurance_fund_fee", insuranceFundFee.String()),
			sdk.NewAttribute("is_partial", partialStr),
		),
	)
}

// EndBlockLiquidationsV2 processes all liquidations at end of block
// Returns statistics about liquidations performed
func (le *LiquidationEngineV2) EndBlockLiquidationsV2(ctx sdk.Context) LiquidationStats {
	stats := LiquidationStats{
		TotalVolume:    math.LegacyZeroDec(),
		TotalPenalties: math.LegacyZeroDec(),
	}

	// Get all unhealthy positions
	unhealthyPositions := le.keeper.GetUnhealthyPositions(ctx)

	// Limit liquidations per block
	liquidationsThisBlock := 0

	for _, health := range unhealthyPositions {
		if liquidationsThisBlock >= le.config.MaxLiquidationsPerBlock {
			le.keeper.Logger().Info("Max liquidations per block reached",
				"max", le.config.MaxLiquidationsPerBlock,
			)
			break
		}

		// Process liquidation
		result, err := le.ProcessLiquidation(ctx, health.Trader, health.MarketID, "")
		if err != nil {
			le.keeper.Logger().Error("Failed to process liquidation",
				"trader", health.Trader,
				"market", health.MarketID,
				"error", err,
			)
			continue
		}

		if result.Success {
			liquidationsThisBlock++
			stats.LiquidationsCount++
			volume := result.LiquidatedSize.Mul(result.LiquidationPrice)
			stats.TotalVolume = stats.TotalVolume.Add(volume)
			stats.TotalPenalties = stats.TotalPenalties.Add(result.PenaltyPaid)
		}
	}

	// Process positions in cooldown that have expired
	currentTime := ctx.BlockTime()
	for key, state := range le.liquidationStates {
		if state.IsInCooldown && currentTime.After(state.CooldownEndTime) {
			state.EndCooldown()

			// Re-process the position
			result, err := le.ProcessLiquidation(ctx, state.Trader, state.MarketID, "")
			if err != nil {
				le.keeper.Logger().Error("Failed to continue liquidation after cooldown",
					"trader", state.Trader,
					"market", state.MarketID,
					"error", err,
				)
				continue
			}

			if result.Success {
				liquidationsThisBlock++
				stats.LiquidationsCount++
				volume := result.LiquidatedSize.Mul(result.LiquidationPrice)
				stats.TotalVolume = stats.TotalVolume.Add(volume)
				stats.TotalPenalties = stats.TotalPenalties.Add(result.PenaltyPaid)
			}
		}

		// Clean up fully liquidated states
		if state.IsFullyLiquidated() {
			delete(le.liquidationStates, key)
		}
	}

	return stats
}

// GetLiquidationState returns the current liquidation state for a position
func (le *LiquidationEngineV2) GetLiquidationState(trader, marketID string) *types.LiquidationState {
	key := fmt.Sprintf("%s:%s", trader, marketID)
	return le.liquidationStates[key]
}

// TriggerLiquidationV2 allows anyone to trigger liquidation of an unhealthy position
// Liquidator receives a reward for successful liquidations (except backstop)
func (le *LiquidationEngineV2) TriggerLiquidationV2(
	ctx sdk.Context,
	liquidator string,
	trader string,
	marketID string,
) (*LiquidationResultV2, error) {
	return le.ProcessLiquidation(ctx, trader, marketID, liquidator)
}

// GetPendingLiquidations returns all positions that are pending further liquidation
func (le *LiquidationEngineV2) GetPendingLiquidations() []*types.LiquidationState {
	var pending []*types.LiquidationState
	for _, state := range le.liquidationStates {
		if !state.IsFullyLiquidated() {
			pending = append(pending, state)
		}
	}
	return pending
}

// GetPositionsInCooldown returns all positions currently in cooldown
func (le *LiquidationEngineV2) GetPositionsInCooldown() []*types.LiquidationState {
	var inCooldown []*types.LiquidationState
	for _, state := range le.liquidationStates {
		if state.IsInCooldown {
			inCooldown = append(inCooldown, state)
		}
	}
	return inCooldown
}
