package keeper

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"time"

	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/openalpha/perp-dex/x/clearinghouse/types"
)

// Store key prefixes for insurance
var (
	InsuranceFundKeyPrefix   = []byte{0x10}
	InsuranceConfigKeyPrefix = []byte{0x11}
	InsuranceEventKeyPrefix  = []byte{0x12}
	InsuranceEventCounterKey = []byte{0x13}
)

// GlobalFundID is the identifier for the global insurance fund
const GlobalFundID = "global"

// ============ Insurance Fund Storage ============

// SetInsuranceFund saves an insurance fund to the store
func (k *Keeper) SetInsuranceFund(ctx sdk.Context, fund *types.InsuranceFund) {
	store := k.GetStore(ctx)
	key := append(InsuranceFundKeyPrefix, []byte(fund.FundID)...)
	bz, _ := json.Marshal(fund)
	store.Set(key, bz)
}

// GetInsuranceFund retrieves an insurance fund from the store
func (k *Keeper) GetInsuranceFund(ctx sdk.Context, fundID string) *types.InsuranceFund {
	store := k.GetStore(ctx)
	key := append(InsuranceFundKeyPrefix, []byte(fundID)...)
	bz := store.Get(key)
	if bz == nil {
		return nil
	}
	var fund types.InsuranceFund
	if err := json.Unmarshal(bz, &fund); err != nil {
		return nil
	}
	return &fund
}

// GetGlobalInsuranceFund returns the global insurance fund
func (k *Keeper) GetGlobalInsuranceFund(ctx sdk.Context) *types.InsuranceFund {
	fund := k.GetInsuranceFund(ctx, GlobalFundID)
	if fund == nil {
		// Initialize if not exists
		fund = types.NewInsuranceFund(GlobalFundID, "")
		k.SetInsuranceFund(ctx, fund)
	}
	return fund
}

// GetMarketInsuranceFund returns the insurance fund for a specific market
func (k *Keeper) GetMarketInsuranceFund(ctx sdk.Context, marketID string) *types.InsuranceFund {
	fundID := "market-" + marketID
	fund := k.GetInsuranceFund(ctx, fundID)
	if fund == nil {
		// Use global fund if market-specific doesn't exist
		return k.GetGlobalInsuranceFund(ctx)
	}
	return fund
}

// ============ Insurance Fund Configuration ============

// SetInsuranceFundConfig saves insurance fund configuration
func (k *Keeper) SetInsuranceFundConfig(ctx sdk.Context, config types.InsuranceFundConfig) {
	store := k.GetStore(ctx)
	bz, _ := json.Marshal(config)
	store.Set(InsuranceConfigKeyPrefix, bz)
}

// GetInsuranceFundConfig retrieves insurance fund configuration
func (k *Keeper) GetInsuranceFundConfig(ctx sdk.Context) types.InsuranceFundConfig {
	store := k.GetStore(ctx)
	bz := store.Get(InsuranceConfigKeyPrefix)
	if bz == nil {
		return types.DefaultInsuranceFundConfig()
	}
	var config types.InsuranceFundConfig
	if err := json.Unmarshal(bz, &config); err != nil {
		return types.DefaultInsuranceFundConfig()
	}
	return config
}

// ============ Insurance Fund Operations ============

// DepositToInsuranceFund deposits funds to the insurance fund
func (k *Keeper) DepositToInsuranceFund(ctx sdk.Context, fundID string, amount math.LegacyDec, eventType types.InsuranceEventType, relatedID string) error {
	fund := k.GetInsuranceFund(ctx, fundID)
	if fund == nil {
		fund = types.NewInsuranceFund(fundID, "")
	}

	fund.Deposit(amount)
	k.SetInsuranceFund(ctx, fund)

	// Record event
	k.recordInsuranceEvent(ctx, fundID, eventType, amount, relatedID, "deposit")

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"insurance_deposit",
			sdk.NewAttribute("fund_id", fundID),
			sdk.NewAttribute("amount", amount.String()),
			sdk.NewAttribute("event_type", eventType.String()),
			sdk.NewAttribute("new_balance", fund.Balance.String()),
		),
	)

	return nil
}

// WithdrawFromInsuranceFund withdraws funds from the insurance fund
func (k *Keeper) WithdrawFromInsuranceFund(ctx sdk.Context, fundID string, amount math.LegacyDec, relatedID, description string) error {
	fund := k.GetInsuranceFund(ctx, fundID)
	if fund == nil {
		return fmt.Errorf("insurance fund not found: %s", fundID)
	}

	if !fund.Withdraw(amount) {
		return fmt.Errorf("insufficient insurance fund balance")
	}
	k.SetInsuranceFund(ctx, fund)

	// Record event
	k.recordInsuranceEvent(ctx, fundID, types.InsuranceEventDeficitCover, amount.Neg(), relatedID, description)

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"insurance_withdrawal",
			sdk.NewAttribute("fund_id", fundID),
			sdk.NewAttribute("amount", amount.String()),
			sdk.NewAttribute("related_id", relatedID),
			sdk.NewAttribute("new_balance", fund.Balance.String()),
		),
	)

	return nil
}

// CoverDeficit attempts to cover a deficit using the insurance fund
func (k *Keeper) CoverDeficit(ctx sdk.Context, marketID string, deficit math.LegacyDec, liquidationID string) (covered math.LegacyDec, remaining math.LegacyDec, err error) {
	// Try market-specific fund first
	marketFundID := "market-" + marketID
	marketFund := k.GetInsuranceFund(ctx, marketFundID)

	covered = math.LegacyZeroDec()
	remaining = deficit

	// Cover from market fund if available
	if marketFund != nil && marketFund.Balance.IsPositive() {
		coverAmount := math.LegacyMinDec(marketFund.Balance, remaining)
		if coverAmount.IsPositive() {
			if err := k.WithdrawFromInsuranceFund(ctx, marketFundID, coverAmount, liquidationID, "deficit cover"); err == nil {
				covered = covered.Add(coverAmount)
				remaining = remaining.Sub(coverAmount)
			}
		}
	}

	// Cover remaining from global fund
	if remaining.IsPositive() {
		globalFund := k.GetGlobalInsuranceFund(ctx)
		if globalFund.Balance.IsPositive() {
			coverAmount := math.LegacyMinDec(globalFund.Balance, remaining)
			if coverAmount.IsPositive() {
				if err := k.WithdrawFromInsuranceFund(ctx, GlobalFundID, coverAmount, liquidationID, "deficit cover"); err == nil {
					covered = covered.Add(coverAmount)
					remaining = remaining.Sub(coverAmount)
				}
			}
		}
	}

	k.Logger().Info("deficit coverage attempted",
		"market_id", marketID,
		"deficit", deficit.String(),
		"covered", covered.String(),
		"remaining", remaining.String(),
	)

	return covered, remaining, nil
}

// ProcessLiquidationPenalty processes the liquidation penalty for insurance fund
func (k *Keeper) ProcessLiquidationPenalty(ctx sdk.Context, liquidation *types.Liquidation) error {
	config := k.GetInsuranceFundConfig(ctx)

	// Calculate penalty amount (portion of liquidation going to insurance)
	penaltyAmount := liquidation.Penalty.Mul(config.LiquidationPenaltyRate)

	if penaltyAmount.IsPositive() {
		// Deposit to global fund
		if err := k.DepositToInsuranceFund(ctx, GlobalFundID, penaltyAmount,
			types.InsuranceEventLiquidationPenalty, liquidation.LiquidationID); err != nil {
			return err
		}
	}

	return nil
}

// ProcessTradingFee processes trading fees for insurance fund
func (k *Keeper) ProcessTradingFee(ctx sdk.Context, marketID string, totalFee math.LegacyDec, tradeID string) error {
	config := k.GetInsuranceFundConfig(ctx)

	// Calculate insurance portion
	insurancePortion := totalFee.Mul(config.TradingFeeRate)

	if insurancePortion.IsPositive() {
		if err := k.DepositToInsuranceFund(ctx, GlobalFundID, insurancePortion,
			types.InsuranceEventTradingFee, tradeID); err != nil {
			return err
		}
	}

	return nil
}

// ============ Insurance Event Recording ============

func (k *Keeper) recordInsuranceEvent(ctx sdk.Context, fundID string, eventType types.InsuranceEventType, amount math.LegacyDec, relatedID, description string) {
	eventID := k.generateInsuranceEventID(ctx)

	event := &types.InsuranceEvent{
		EventID:     eventID,
		FundID:      fundID,
		EventType:   eventType,
		Amount:      amount,
		RelatedID:   relatedID,
		Description: description,
		Timestamp:   ctx.BlockTime(),
	}

	store := k.GetStore(ctx)
	key := append(InsuranceEventKeyPrefix, []byte(eventID)...)
	bz, _ := json.Marshal(event)
	store.Set(key, bz)
}

func (k *Keeper) generateInsuranceEventID(ctx sdk.Context) string {
	store := k.GetStore(ctx)
	bz := store.Get(InsuranceEventCounterKey)
	var counter uint64
	if bz != nil {
		counter = binary.BigEndian.Uint64(bz)
	}
	counter++

	newBz := make([]byte, 8)
	binary.BigEndian.PutUint64(newBz, counter)
	store.Set(InsuranceEventCounterKey, newBz)

	return fmt.Sprintf("ins-event-%d", counter)
}

// GetInsuranceEvents returns recent insurance events
func (k *Keeper) GetInsuranceEvents(ctx sdk.Context, fundID string, limit int) []*types.InsuranceEvent {
	store := k.GetStore(ctx)
	iterator := storetypes.KVStoreReversePrefixIterator(store, InsuranceEventKeyPrefix)
	defer iterator.Close()

	var events []*types.InsuranceEvent
	count := 0
	for ; iterator.Valid() && count < limit; iterator.Next() {
		var event types.InsuranceEvent
		if err := json.Unmarshal(iterator.Value(), &event); err != nil {
			continue
		}
		if fundID == "" || event.FundID == fundID {
			events = append(events, &event)
			count++
		}
	}
	return events
}

// ============ Insurance Fund Status ============

// InsuranceFundStatus represents the current status of insurance funds
type InsuranceFundStatus struct {
	GlobalBalance    math.LegacyDec
	MarketBalances   map[string]math.LegacyDec
	TotalBalance     math.LegacyDec
	ADLThreshold     math.LegacyDec
	IsADLTriggered   bool
	LastUpdated      time.Time
}

// GetInsuranceFundStatus returns the current status of all insurance funds
func (k *Keeper) GetInsuranceFundStatus(ctx sdk.Context) *InsuranceFundStatus {
	globalFund := k.GetGlobalInsuranceFund(ctx)
	config := k.GetInsuranceFundConfig(ctx)

	status := &InsuranceFundStatus{
		GlobalBalance:  globalFund.Balance,
		MarketBalances: make(map[string]math.LegacyDec),
		TotalBalance:   globalFund.Balance,
		ADLThreshold:   config.MinFundBalance,
		LastUpdated:    ctx.BlockTime(),
	}

	// Check if ADL should be triggered
	status.IsADLTriggered = status.TotalBalance.LT(config.MinFundBalance)

	return status
}

// ShouldTriggerADL checks if ADL should be triggered
func (k *Keeper) ShouldTriggerADL(ctx sdk.Context, deficit math.LegacyDec) bool {
	globalFund := k.GetGlobalInsuranceFund(ctx)
	config := k.GetInsuranceFundConfig(ctx)

	// Trigger ADL if:
	// 1. Fund balance is below minimum threshold
	// 2. Fund cannot cover the deficit
	if globalFund.Balance.LT(config.MinFundBalance) {
		return true
	}

	if deficit.IsPositive() && globalFund.Balance.LT(deficit) {
		return true
	}

	return false
}
