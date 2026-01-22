package keeper

import (
	"testing"

	"cosmossdk.io/math"
	"github.com/openalpha/perp-dex/x/riverpool/types"
)

// TestCommunityPoolConfig tests config validation
func TestCommunityPoolConfig(t *testing.T) {
	testCases := []struct {
		name        string
		config      *types.CommunityPoolConfig
		expectError bool
	}{
		{
			name: "valid config",
			config: &types.CommunityPoolConfig{
				Name:           "Alpha Trader Pool",
				Description:    "High-performance trading pool",
				Owner:          "cosmos1owner...",
				MinDeposit:     math.LegacyMustNewDecFromStr("100"),
				MaxDeposit:     math.LegacyMustNewDecFromStr("10000"),
				ManagementFee:  math.LegacyMustNewDecFromStr("0.02"),  // 2%
				PerformanceFee: math.LegacyMustNewDecFromStr("0.20"),  // 20%
				OwnerMinStake:  math.LegacyMustNewDecFromStr("0.05"),  // 5%
				MaxLeverage:    math.LegacyMustNewDecFromStr("10"),
				AllowedMarkets: []string{"BTC-USDC", "ETH-USDC"},
				Tags:           []string{"BTC", "ETH", "Trend"},
			},
			expectError: false,
		},
		{
			name: "empty name",
			config: &types.CommunityPoolConfig{
				Name:           "",
				Owner:          "cosmos1owner...",
				OwnerMinStake:  math.LegacyMustNewDecFromStr("0.05"),
				ManagementFee:  math.LegacyMustNewDecFromStr("0.02"),
				PerformanceFee: math.LegacyMustNewDecFromStr("0.20"),
			},
			expectError: true,
		},
		{
			name: "empty owner",
			config: &types.CommunityPoolConfig{
				Name:           "Test Pool",
				Owner:          "",
				OwnerMinStake:  math.LegacyMustNewDecFromStr("0.05"),
				ManagementFee:  math.LegacyMustNewDecFromStr("0.02"),
				PerformanceFee: math.LegacyMustNewDecFromStr("0.20"),
			},
			expectError: true,
		},
		{
			name: "owner stake too low",
			config: &types.CommunityPoolConfig{
				Name:           "Test Pool",
				Owner:          "cosmos1owner...",
				OwnerMinStake:  math.LegacyMustNewDecFromStr("0.03"), // 3% < 5% minimum
				ManagementFee:  math.LegacyMustNewDecFromStr("0.02"),
				PerformanceFee: math.LegacyMustNewDecFromStr("0.20"),
			},
			expectError: true,
		},
		{
			name: "management fee too high",
			config: &types.CommunityPoolConfig{
				Name:           "Test Pool",
				Owner:          "cosmos1owner...",
				OwnerMinStake:  math.LegacyMustNewDecFromStr("0.05"),
				ManagementFee:  math.LegacyMustNewDecFromStr("0.10"), // 10% > 5% max
				PerformanceFee: math.LegacyMustNewDecFromStr("0.20"),
			},
			expectError: true,
		},
		{
			name: "performance fee too high",
			config: &types.CommunityPoolConfig{
				Name:           "Test Pool",
				Owner:          "cosmos1owner...",
				OwnerMinStake:  math.LegacyMustNewDecFromStr("0.05"),
				ManagementFee:  math.LegacyMustNewDecFromStr("0.02"),
				PerformanceFee: math.LegacyMustNewDecFromStr("0.60"), // 60% > 50% max
			},
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.config.Validate()
			if tc.expectError && err == nil {
				t.Error("expected error but got nil")
			}
			if !tc.expectError && err != nil {
				t.Errorf("expected no error but got: %v", err)
			}
		})
	}
}

// TestNewCommunityPool tests community pool creation
func TestNewCommunityPool(t *testing.T) {
	config := &types.CommunityPoolConfig{
		Name:                 "Alpha Trader Pool",
		Description:          "High-performance trading pool",
		Owner:                "cosmos1owner...",
		MinDeposit:           math.LegacyMustNewDecFromStr("100"),
		MaxDeposit:           math.LegacyMustNewDecFromStr("10000"),
		ManagementFee:        math.LegacyMustNewDecFromStr("0.02"),
		PerformanceFee:       math.LegacyMustNewDecFromStr("0.20"),
		OwnerMinStake:        math.LegacyMustNewDecFromStr("0.05"),
		LockPeriodDays:       7,
		RedemptionDelayDays:  3,
		DailyRedemptionLimit: math.LegacyMustNewDecFromStr("0.10"),
		IsPrivate:            false,
		MaxLeverage:          math.LegacyMustNewDecFromStr("10"),
		AllowedMarkets:       []string{"BTC-USDC", "ETH-USDC"},
		Tags:                 []string{"BTC", "ETH", "Trend"},
	}

	pool, err := types.NewCommunityPool(config)
	if err != nil {
		t.Fatalf("unexpected error creating pool: %v", err)
	}

	// Check basic fields
	if pool.PoolType != types.PoolTypeCommunity {
		t.Errorf("expected pool type community, got %s", pool.PoolType)
	}
	if pool.Name != "Alpha Trader Pool" {
		t.Errorf("expected name Alpha Trader Pool, got %s", pool.Name)
	}
	if pool.Owner != "cosmos1owner..." {
		t.Errorf("expected owner cosmos1owner..., got %s", pool.Owner)
	}

	// Check fees
	if !pool.ManagementFee.Equal(math.LegacyMustNewDecFromStr("0.02")) {
		t.Errorf("expected management fee 0.02, got %s", pool.ManagementFee.String())
	}
	if !pool.PerformanceFee.Equal(math.LegacyMustNewDecFromStr("0.20")) {
		t.Errorf("expected performance fee 0.20, got %s", pool.PerformanceFee.String())
	}

	// Check owner stake
	if !pool.OwnerMinStake.Equal(math.LegacyMustNewDecFromStr("0.05")) {
		t.Errorf("expected owner min stake 0.05, got %s", pool.OwnerMinStake.String())
	}
	if !pool.OwnerCurrentStake.IsZero() {
		t.Errorf("expected owner current stake 0, got %s", pool.OwnerCurrentStake.String())
	}

	// Check leverage and markets
	if !pool.MaxLeverage.Equal(math.LegacyMustNewDecFromStr("10")) {
		t.Errorf("expected max leverage 10, got %s", pool.MaxLeverage.String())
	}
	if len(pool.AllowedMarkets) != 2 {
		t.Errorf("expected 2 allowed markets, got %d", len(pool.AllowedMarkets))
	}

	// Check tags
	if len(pool.Tags) != 3 {
		t.Errorf("expected 3 tags, got %d", len(pool.Tags))
	}

	// Check status
	if pool.Status != types.PoolStatusActive {
		t.Errorf("expected active status, got %s", pool.Status)
	}

	// Check holders count
	if pool.TotalHolders != 0 {
		t.Errorf("expected 0 holders, got %d", pool.TotalHolders)
	}

	// Check no invite code for public pool
	if pool.InviteCode != "" {
		t.Errorf("expected no invite code for public pool, got %s", pool.InviteCode)
	}
}

// TestNewCommunityPoolPrivate tests private community pool creation
func TestNewCommunityPoolPrivate(t *testing.T) {
	config := &types.CommunityPoolConfig{
		Name:           "Private Alpha Pool",
		Description:    "Exclusive trading pool",
		Owner:          "cosmos1owner...",
		OwnerMinStake:  math.LegacyMustNewDecFromStr("0.10"), // 10%
		ManagementFee:  math.LegacyMustNewDecFromStr("0.03"),
		PerformanceFee: math.LegacyMustNewDecFromStr("0.25"),
		IsPrivate:      true,
	}

	pool, err := types.NewCommunityPool(config)
	if err != nil {
		t.Fatalf("unexpected error creating pool: %v", err)
	}

	// Check private flag
	if !pool.IsPrivate {
		t.Error("expected pool to be private")
	}

	// Check invite code is generated
	if pool.InviteCode == "" {
		t.Error("expected invite code to be generated for private pool")
	}
}

// TestInviteCode tests invite code functionality
func TestInviteCode(t *testing.T) {
	// Test valid invite code
	inviteCode := types.NewInviteCode("pool-123", "cosmos1owner...", 10, 86400) // 10 uses, 1 day expiry

	if inviteCode.PoolID != "pool-123" {
		t.Errorf("expected pool ID pool-123, got %s", inviteCode.PoolID)
	}
	if inviteCode.MaxUses != 10 {
		t.Errorf("expected max uses 10, got %d", inviteCode.MaxUses)
	}
	if !inviteCode.IsActive {
		t.Error("expected invite code to be active")
	}
	if !inviteCode.IsValid() {
		t.Error("expected invite code to be valid")
	}

	// Test unlimited uses
	unlimitedCode := types.NewInviteCode("pool-123", "cosmos1owner...", 0, 0)
	if unlimitedCode.MaxUses != 0 {
		t.Errorf("expected unlimited uses (0), got %d", unlimitedCode.MaxUses)
	}
	if unlimitedCode.ExpiresAt != 0 {
		t.Errorf("expected never expires (0), got %d", unlimitedCode.ExpiresAt)
	}
	if !unlimitedCode.IsValid() {
		t.Error("expected unlimited invite code to be valid")
	}
}

// TestInviteCodeMaxUses tests invite code max uses check
func TestInviteCodeMaxUses(t *testing.T) {
	inviteCode := types.NewInviteCode("pool-123", "cosmos1owner...", 2, 0) // 2 uses max

	// Initially valid
	if !inviteCode.IsValid() {
		t.Error("expected invite code to be valid initially")
	}

	// After 1 use
	inviteCode.UsedCount = 1
	if !inviteCode.IsValid() {
		t.Error("expected invite code to be valid after 1 use")
	}

	// After 2 uses (max reached)
	inviteCode.UsedCount = 2
	if inviteCode.IsValid() {
		t.Error("expected invite code to be invalid after max uses reached")
	}
}

// TestInviteCodeDeactivation tests invite code deactivation
func TestInviteCodeDeactivation(t *testing.T) {
	inviteCode := types.NewInviteCode("pool-123", "cosmos1owner...", 0, 0)

	// Initially valid
	if !inviteCode.IsValid() {
		t.Error("expected invite code to be valid initially")
	}

	// After deactivation
	inviteCode.IsActive = false
	if inviteCode.IsValid() {
		t.Error("expected invite code to be invalid after deactivation")
	}
}

// TestPoolHolder tests pool holder type
func TestPoolHolder(t *testing.T) {
	holder := types.PoolHolder{
		PoolID:      "cpool-123",
		Address:     "cosmos1user...",
		Shares:      math.LegacyMustNewDecFromStr("1000"),
		Value:       math.LegacyMustNewDecFromStr("1100"), // NAV = 1.1
		DepositedAt: 1704067200,
		IsOwner:     false,
	}

	if holder.PoolID != "cpool-123" {
		t.Errorf("expected pool ID cpool-123, got %s", holder.PoolID)
	}
	if !holder.Shares.Equal(math.LegacyMustNewDecFromStr("1000")) {
		t.Errorf("expected shares 1000, got %s", holder.Shares.String())
	}
	if holder.IsOwner {
		t.Error("expected holder not to be owner")
	}
}

// TestPoolPosition tests pool position type
func TestPoolPosition(t *testing.T) {
	position := types.NewPoolPosition(
		"cpool-123",
		"BTC-USDC",
		"long",
		math.LegacyMustNewDecFromStr("0.1"),     // 0.1 BTC
		math.LegacyMustNewDecFromStr("50000"),   // $50,000
		math.LegacyMustNewDecFromStr("10"),      // 10x leverage
		math.LegacyMustNewDecFromStr("500"),     // $500 margin
	)

	if position.PoolID != "cpool-123" {
		t.Errorf("expected pool ID cpool-123, got %s", position.PoolID)
	}
	if position.MarketID != "BTC-USDC" {
		t.Errorf("expected market ID BTC-USDC, got %s", position.MarketID)
	}
	if position.Side != "long" {
		t.Errorf("expected side long, got %s", position.Side)
	}
	if !position.Size.Equal(math.LegacyMustNewDecFromStr("0.1")) {
		t.Errorf("expected size 0.1, got %s", position.Size.String())
	}
	if !position.EntryPrice.Equal(math.LegacyMustNewDecFromStr("50000")) {
		t.Errorf("expected entry price 50000, got %s", position.EntryPrice.String())
	}
	if !position.Leverage.Equal(math.LegacyMustNewDecFromStr("10")) {
		t.Errorf("expected leverage 10, got %s", position.Leverage.String())
	}
	if !position.UnrealizedPnL.IsZero() {
		t.Errorf("expected unrealized PnL 0, got %s", position.UnrealizedPnL.String())
	}
}

// TestPoolTrade tests pool trade type
func TestPoolTrade(t *testing.T) {
	trade := types.NewPoolTrade(
		"cpool-123",
		"BTC-USDC",
		"buy",
		math.LegacyMustNewDecFromStr("0.1"),     // 0.1 BTC
		math.LegacyMustNewDecFromStr("50000"),   // $50,000
		math.LegacyMustNewDecFromStr("2.5"),     // $2.5 fee
	)

	if trade.PoolID != "cpool-123" {
		t.Errorf("expected pool ID cpool-123, got %s", trade.PoolID)
	}
	if trade.MarketID != "BTC-USDC" {
		t.Errorf("expected market ID BTC-USDC, got %s", trade.MarketID)
	}
	if trade.Side != "buy" {
		t.Errorf("expected side buy, got %s", trade.Side)
	}
	if !trade.Fee.Equal(math.LegacyMustNewDecFromStr("2.5")) {
		t.Errorf("expected fee 2.5, got %s", trade.Fee.String())
	}
	if !trade.PnL.IsZero() {
		t.Errorf("expected PnL 0 for new trade, got %s", trade.PnL.String())
	}
	if trade.ExecutedAt == 0 {
		t.Error("expected executed timestamp to be set")
	}
}

// TestDDGuardState tests DDGuard state type
func TestDDGuardState(t *testing.T) {
	state := &types.DDGuardState{
		PoolID:           "main-lp",
		Level:            types.DDGuardLevelNormal,
		PeakNAV:          math.LegacyMustNewDecFromStr("1.2"),
		CurrentNAV:       math.LegacyMustNewDecFromStr("1.1"),
		DrawdownPercent:  math.LegacyMustNewDecFromStr("0.0833"), // ~8.33%
		MaxExposureLimit: math.LegacyOneDec(),
		TriggeredAt:      1704067200,
		LastCheckedAt:    1704067200,
	}

	if state.PoolID != "main-lp" {
		t.Errorf("expected pool ID main-lp, got %s", state.PoolID)
	}
	if state.Level != types.DDGuardLevelNormal {
		t.Errorf("expected level normal, got %s", state.Level)
	}
	if !state.PeakNAV.Equal(math.LegacyMustNewDecFromStr("1.2")) {
		t.Errorf("expected peak NAV 1.2, got %s", state.PeakNAV.String())
	}
}

// TestPoolStats tests pool stats type
func TestPoolStats(t *testing.T) {
	stats := types.NewPoolStats("main-lp")

	if stats.PoolID != "main-lp" {
		t.Errorf("expected pool ID main-lp, got %s", stats.PoolID)
	}
	if !stats.TotalValueLocked.IsZero() {
		t.Errorf("expected TVL 0, got %s", stats.TotalValueLocked.String())
	}
	if stats.TotalDepositors != 0 {
		t.Errorf("expected 0 depositors, got %d", stats.TotalDepositors)
	}
	if !stats.Return1d.IsZero() {
		t.Errorf("expected return 1d 0, got %s", stats.Return1d.String())
	}
}

// TestRevenueRecord tests revenue record type
func TestRevenueRecord(t *testing.T) {
	record := &types.RevenueRecord{
		RecordID:    "rev-001",
		PoolID:      "main-lp",
		Source:      "spread",
		Amount:      math.LegacyMustNewDecFromStr("100"),
		Description: "Bid-ask spread earnings",
		Timestamp:   1704067200,
	}

	if record.PoolID != "main-lp" {
		t.Errorf("expected pool ID main-lp, got %s", record.PoolID)
	}
	if record.Source != "spread" {
		t.Errorf("expected source spread, got %s", record.Source)
	}
	if !record.Amount.Equal(math.LegacyMustNewDecFromStr("100")) {
		t.Errorf("expected amount 100, got %s", record.Amount.String())
	}
}

// TestNAVHistory tests NAV history type
func TestNAVHistory(t *testing.T) {
	history := &types.NAVHistory{
		PoolID:     "main-lp",
		NAV:        math.LegacyMustNewDecFromStr("1.05"),
		TotalValue: math.LegacyMustNewDecFromStr("1050000"),
		Timestamp:  1704067200,
	}

	if history.PoolID != "main-lp" {
		t.Errorf("expected pool ID main-lp, got %s", history.PoolID)
	}
	if !history.NAV.Equal(math.LegacyMustNewDecFromStr("1.05")) {
		t.Errorf("expected NAV 1.05, got %s", history.NAV.String())
	}
}
