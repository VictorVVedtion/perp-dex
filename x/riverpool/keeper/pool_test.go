package keeper

import (
	"testing"

	"cosmossdk.io/math"
	"github.com/openalpha/perp-dex/x/riverpool/types"
)

// TestNewFoundationPool tests foundation pool creation with default values
func TestNewFoundationPool(t *testing.T) {
	pool := types.NewFoundationPool()

	// Check basic fields
	if pool.PoolID != "foundation-lp" {
		t.Errorf("expected pool ID foundation-lp, got %s", pool.PoolID)
	}
	if pool.PoolType != types.PoolTypeFoundation {
		t.Errorf("expected pool type foundation, got %s", pool.PoolType)
	}
	if pool.Name != "Foundation LP" {
		t.Errorf("expected name Foundation LP, got %s", pool.Name)
	}

	// Check NAV starts at 1.0
	if !pool.NAV.Equal(math.LegacyOneDec()) {
		t.Errorf("expected NAV 1.0, got %s", pool.NAV.String())
	}

	// Check high water mark starts at 1.0
	if !pool.HighWaterMark.Equal(math.LegacyOneDec()) {
		t.Errorf("expected high water mark 1.0, got %s", pool.HighWaterMark.String())
	}

	// Check DDGuard level is normal
	if pool.DDGuardLevel != types.DDGuardLevelNormal {
		t.Errorf("expected DDGuard level normal, got %s", pool.DDGuardLevel)
	}

	// Check status is active
	if pool.Status != types.PoolStatusActive {
		t.Errorf("expected active status, got %s", pool.Status)
	}

	// Check lock period (180 days)
	if pool.LockPeriodDays != 180 {
		t.Errorf("expected lock period 180 days, got %d", pool.LockPeriodDays)
	}

	// Check seat configuration
	if pool.SeatsTotal != 100 {
		t.Errorf("expected 100 total seats, got %d", pool.SeatsTotal)
	}
	if pool.SeatsAvailable != 100 {
		t.Errorf("expected 100 available seats, got %d", pool.SeatsAvailable)
	}

	// Check min/max deposit ($100K)
	expectedDeposit := math.LegacyMustNewDecFromStr("100000")
	if !pool.MinDeposit.Equal(expectedDeposit) {
		t.Errorf("expected min deposit 100000, got %s", pool.MinDeposit.String())
	}
	if !pool.MaxDeposit.Equal(expectedDeposit) {
		t.Errorf("expected max deposit 100000, got %s", pool.MaxDeposit.String())
	}
}

// TestNewMainPool tests main pool creation with default values
func TestNewMainPool(t *testing.T) {
	pool := types.NewMainPool()

	// Check basic fields
	if pool.PoolID != "main-lp" {
		t.Errorf("expected pool ID main-lp, got %s", pool.PoolID)
	}
	if pool.PoolType != types.PoolTypeMain {
		t.Errorf("expected pool type main, got %s", pool.PoolType)
	}

	// Check min deposit ($100)
	expectedMin := math.LegacyMustNewDecFromStr("100")
	if !pool.MinDeposit.Equal(expectedMin) {
		t.Errorf("expected min deposit 100, got %s", pool.MinDeposit.String())
	}

	// Check no max deposit
	if !pool.MaxDeposit.IsZero() {
		t.Errorf("expected no max deposit (0), got %s", pool.MaxDeposit.String())
	}

	// Check T+4 redemption delay
	if pool.RedemptionDelayDays != 4 {
		t.Errorf("expected redemption delay 4 days, got %d", pool.RedemptionDelayDays)
	}

	// Check 15% daily limit
	expectedLimit := math.LegacyMustNewDecFromStr("0.15")
	if !pool.DailyRedemptionLimit.Equal(expectedLimit) {
		t.Errorf("expected daily limit 0.15, got %s", pool.DailyRedemptionLimit.String())
	}

	// Check no lock period
	if pool.LockPeriodDays != 0 {
		t.Errorf("expected no lock period, got %d", pool.LockPeriodDays)
	}
}

// TestCalculateSharesForDeposit tests share calculation
func TestCalculateSharesForDeposit(t *testing.T) {
	pool := types.NewMainPool()

	testCases := []struct {
		name           string
		depositAmount  math.LegacyDec
		nav            math.LegacyDec
		expectedShares math.LegacyDec
	}{
		{
			name:           "deposit at NAV 1.0",
			depositAmount:  math.LegacyMustNewDecFromStr("1000"),
			nav:            math.LegacyOneDec(),
			expectedShares: math.LegacyMustNewDecFromStr("1000"),
		},
		{
			name:           "deposit at NAV 1.1",
			depositAmount:  math.LegacyMustNewDecFromStr("1100"),
			nav:            math.LegacyMustNewDecFromStr("1.1"),
			expectedShares: math.LegacyMustNewDecFromStr("1000"),
		},
		{
			name:           "deposit at NAV 0.9",
			depositAmount:  math.LegacyMustNewDecFromStr("900"),
			nav:            math.LegacyMustNewDecFromStr("0.9"),
			expectedShares: math.LegacyMustNewDecFromStr("1000"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			pool.NAV = tc.nav
			shares := pool.CalculateSharesForDeposit(tc.depositAmount)

			if !shares.Equal(tc.expectedShares) {
				t.Errorf("expected %s shares, got %s", tc.expectedShares.String(), shares.String())
			}
		})
	}
}

// TestCalculateValueForShares tests value calculation from shares
func TestCalculateValueForShares(t *testing.T) {
	pool := types.NewMainPool()

	testCases := []struct {
		name          string
		shares        math.LegacyDec
		nav           math.LegacyDec
		expectedValue math.LegacyDec
	}{
		{
			name:          "shares at NAV 1.0",
			shares:        math.LegacyMustNewDecFromStr("1000"),
			nav:           math.LegacyOneDec(),
			expectedValue: math.LegacyMustNewDecFromStr("1000"),
		},
		{
			name:          "shares at NAV 1.2",
			shares:        math.LegacyMustNewDecFromStr("1000"),
			nav:           math.LegacyMustNewDecFromStr("1.2"),
			expectedValue: math.LegacyMustNewDecFromStr("1200"),
		},
		{
			name:          "shares at NAV 0.8",
			shares:        math.LegacyMustNewDecFromStr("1000"),
			nav:           math.LegacyMustNewDecFromStr("0.8"),
			expectedValue: math.LegacyMustNewDecFromStr("800"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			pool.NAV = tc.nav
			value := pool.CalculateValueForShares(tc.shares)

			if !value.Equal(tc.expectedValue) {
				t.Errorf("expected %s value, got %s", tc.expectedValue.String(), value.String())
			}
		})
	}
}

// TestUpdateNAV tests NAV update and DDGuard level changes
func TestUpdateNAV(t *testing.T) {
	testCases := []struct {
		name                string
		initialNAV          math.LegacyDec
		totalValue          math.LegacyDec
		totalShares         math.LegacyDec
		expectedNAV         math.LegacyDec
		expectedDDLevel     string
		expectedHighWater   math.LegacyDec
		expectedDrawdown    math.LegacyDec
	}{
		{
			name:              "NAV increases above high water mark",
			initialNAV:        math.LegacyOneDec(),
			totalValue:        math.LegacyMustNewDecFromStr("1100"),
			totalShares:       math.LegacyMustNewDecFromStr("1000"),
			expectedNAV:       math.LegacyMustNewDecFromStr("1.1"),
			expectedDDLevel:   types.DDGuardLevelNormal,
			expectedHighWater: math.LegacyMustNewDecFromStr("1.1"),
			expectedDrawdown:  math.LegacyZeroDec(),
		},
		{
			name:              "NAV drops 5% - still normal",
			initialNAV:        math.LegacyOneDec(),
			totalValue:        math.LegacyMustNewDecFromStr("950"),
			totalShares:       math.LegacyMustNewDecFromStr("1000"),
			expectedNAV:       math.LegacyMustNewDecFromStr("0.95"),
			expectedDDLevel:   types.DDGuardLevelNormal,
			expectedHighWater: math.LegacyOneDec(),
			expectedDrawdown:  math.LegacyMustNewDecFromStr("0.05"),
		},
		{
			name:              "NAV drops 12% - warning level",
			initialNAV:        math.LegacyOneDec(),
			totalValue:        math.LegacyMustNewDecFromStr("880"),
			totalShares:       math.LegacyMustNewDecFromStr("1000"),
			expectedNAV:       math.LegacyMustNewDecFromStr("0.88"),
			expectedDDLevel:   types.DDGuardLevelWarning,
			expectedHighWater: math.LegacyOneDec(),
			expectedDrawdown:  math.LegacyMustNewDecFromStr("0.12"),
		},
		{
			name:              "NAV drops 20% - reduce level",
			initialNAV:        math.LegacyOneDec(),
			totalValue:        math.LegacyMustNewDecFromStr("800"),
			totalShares:       math.LegacyMustNewDecFromStr("1000"),
			expectedNAV:       math.LegacyMustNewDecFromStr("0.8"),
			expectedDDLevel:   types.DDGuardLevelReduce,
			expectedHighWater: math.LegacyOneDec(),
			expectedDrawdown:  math.LegacyMustNewDecFromStr("0.2"),
		},
		{
			name:              "NAV drops 35% - halt level",
			initialNAV:        math.LegacyOneDec(),
			totalValue:        math.LegacyMustNewDecFromStr("650"),
			totalShares:       math.LegacyMustNewDecFromStr("1000"),
			expectedNAV:       math.LegacyMustNewDecFromStr("0.65"),
			expectedDDLevel:   types.DDGuardLevelHalt,
			expectedHighWater: math.LegacyOneDec(),
			expectedDrawdown:  math.LegacyMustNewDecFromStr("0.35"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			pool := types.NewMainPool()
			pool.NAV = tc.initialNAV
			pool.HighWaterMark = tc.initialNAV
			pool.TotalShares = tc.totalShares

			pool.UpdateNAV(tc.totalValue)

			if !pool.NAV.Equal(tc.expectedNAV) {
				t.Errorf("expected NAV %s, got %s", tc.expectedNAV.String(), pool.NAV.String())
			}

			if pool.DDGuardLevel != tc.expectedDDLevel {
				t.Errorf("expected DDGuard level %s, got %s", tc.expectedDDLevel, pool.DDGuardLevel)
			}

			if !pool.HighWaterMark.Equal(tc.expectedHighWater) {
				t.Errorf("expected high water mark %s, got %s", tc.expectedHighWater.String(), pool.HighWaterMark.String())
			}

			// Allow small tolerance for drawdown comparison
			drawdownDiff := pool.CurrentDrawdown.Sub(tc.expectedDrawdown).Abs()
			tolerance := math.LegacyMustNewDecFromStr("0.0001")
			if drawdownDiff.GT(tolerance) {
				t.Errorf("expected drawdown %s, got %s", tc.expectedDrawdown.String(), pool.CurrentDrawdown.String())
			}
		})
	}
}

// TestGetSeatCount tests seat counting for foundation pool
func TestGetSeatCount(t *testing.T) {
	pool := types.NewFoundationPool()

	// Initially 0 seats filled
	if pool.GetSeatCount() != 0 {
		t.Errorf("expected 0 seats filled, got %d", pool.GetSeatCount())
	}

	// After 1 seat worth of deposits ($100K)
	pool.TotalDeposits = math.LegacyMustNewDecFromStr("100000")
	if pool.GetSeatCount() != 1 {
		t.Errorf("expected 1 seat filled, got %d", pool.GetSeatCount())
	}

	// After 5 seats worth of deposits ($500K)
	pool.TotalDeposits = math.LegacyMustNewDecFromStr("500000")
	if pool.GetSeatCount() != 5 {
		t.Errorf("expected 5 seats filled, got %d", pool.GetSeatCount())
	}

	// After all 100 seats ($10M)
	pool.TotalDeposits = math.LegacyMustNewDecFromStr("10000000")
	if pool.GetSeatCount() != 100 {
		t.Errorf("expected 100 seats filled, got %d", pool.GetSeatCount())
	}
}

// TestHasAvailableSeats tests seat availability checking
func TestHasAvailableSeats(t *testing.T) {
	// Foundation pool
	foundationPool := types.NewFoundationPool()

	if !foundationPool.HasAvailableSeats() {
		t.Error("expected foundation pool to have available seats initially")
	}

	// Fill all seats
	foundationPool.TotalDeposits = math.LegacyMustNewDecFromStr("10000000") // $10M = 100 seats
	if foundationPool.HasAvailableSeats() {
		t.Error("expected foundation pool to have no available seats when full")
	}

	// Main pool should always return true (no seat limit)
	mainPool := types.NewMainPool()
	if !mainPool.HasAvailableSeats() {
		t.Error("expected main pool to always have available seats")
	}
}

// TestNewDeposit tests deposit creation
func TestNewDeposit(t *testing.T) {
	deposit := types.NewDeposit(
		"main-lp",
		"cosmos1abc...",
		math.LegacyMustNewDecFromStr("1000"),
		math.LegacyMustNewDecFromStr("1000"),
		math.LegacyOneDec(),
		0, // No lock
	)

	if deposit.PoolID != "main-lp" {
		t.Errorf("expected pool ID main-lp, got %s", deposit.PoolID)
	}
	if deposit.Depositor != "cosmos1abc..." {
		t.Errorf("expected depositor cosmos1abc..., got %s", deposit.Depositor)
	}
	if !deposit.Amount.Equal(math.LegacyMustNewDecFromStr("1000")) {
		t.Errorf("expected amount 1000, got %s", deposit.Amount.String())
	}
	if !deposit.Shares.Equal(math.LegacyMustNewDecFromStr("1000")) {
		t.Errorf("expected shares 1000, got %s", deposit.Shares.String())
	}
	if deposit.UnlockAt != 0 {
		t.Errorf("expected no unlock time, got %d", deposit.UnlockAt)
	}
	if deposit.IsLocked() {
		t.Error("expected deposit not to be locked")
	}
}

// TestDepositWithLock tests deposit creation with lock period
func TestDepositWithLock(t *testing.T) {
	deposit := types.NewDeposit(
		"foundation-lp",
		"cosmos1abc...",
		math.LegacyMustNewDecFromStr("100000"),
		math.LegacyMustNewDecFromStr("100000"),
		math.LegacyOneDec(),
		180, // 180 days lock
	)

	if deposit.UnlockAt == 0 {
		t.Error("expected unlock time to be set")
	}

	// Deposit should be locked
	if !deposit.IsLocked() {
		t.Error("expected deposit to be locked")
	}
}

// TestNewWithdrawal tests withdrawal request creation
func TestNewWithdrawal(t *testing.T) {
	withdrawal := types.NewWithdrawal(
		"main-lp",
		"cosmos1abc...",
		math.LegacyMustNewDecFromStr("500"),
		math.LegacyOneDec(),
		4, // T+4 delay
	)

	if withdrawal.PoolID != "main-lp" {
		t.Errorf("expected pool ID main-lp, got %s", withdrawal.PoolID)
	}
	if withdrawal.Withdrawer != "cosmos1abc..." {
		t.Errorf("expected withdrawer cosmos1abc..., got %s", withdrawal.Withdrawer)
	}
	if !withdrawal.SharesRequested.Equal(math.LegacyMustNewDecFromStr("500")) {
		t.Errorf("expected shares requested 500, got %s", withdrawal.SharesRequested.String())
	}
	if withdrawal.Status != types.WithdrawalStatusPending {
		t.Errorf("expected pending status, got %s", withdrawal.Status)
	}

	// Should not be ready immediately (T+4)
	if withdrawal.IsReady() {
		t.Error("expected withdrawal not to be ready immediately")
	}
}
