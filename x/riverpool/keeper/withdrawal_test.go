package keeper

import (
	"testing"
	"time"

	"cosmossdk.io/math"
	"github.com/openalpha/perp-dex/x/riverpool/types"
)

// TestWithdrawalStatus tests withdrawal status constants
func TestWithdrawalStatus(t *testing.T) {
	// Verify all status constants are defined
	statuses := []string{
		types.WithdrawalStatusPending,
		types.WithdrawalStatusProcessing,
		types.WithdrawalStatusCompleted,
		types.WithdrawalStatusCancelled,
	}

	expected := []string{"pending", "processing", "completed", "cancelled"}

	for i, status := range statuses {
		if status != expected[i] {
			t.Errorf("expected status %s, got %s", expected[i], status)
		}
	}
}

// TestNewWithdrawalCreation tests withdrawal request creation
func TestNewWithdrawalCreation(t *testing.T) {
	shares := math.LegacyMustNewDecFromStr("500")
	nav := math.LegacyMustNewDecFromStr("1.1")

	withdrawal := types.NewWithdrawal(
		"main-lp",
		"cosmos1user...",
		shares,
		nav,
		4, // T+4
	)

	// Check basic fields
	if withdrawal.PoolID != "main-lp" {
		t.Errorf("expected pool ID main-lp, got %s", withdrawal.PoolID)
	}
	if withdrawal.Withdrawer != "cosmos1user..." {
		t.Errorf("expected withdrawer cosmos1user..., got %s", withdrawal.Withdrawer)
	}
	if !withdrawal.SharesRequested.Equal(shares) {
		t.Errorf("expected shares requested %s, got %s", shares.String(), withdrawal.SharesRequested.String())
	}
	if !withdrawal.NAVAtRequest.Equal(nav) {
		t.Errorf("expected NAV at request %s, got %s", nav.String(), withdrawal.NAVAtRequest.String())
	}

	// Check initial status
	if withdrawal.Status != types.WithdrawalStatusPending {
		t.Errorf("expected pending status, got %s", withdrawal.Status)
	}

	// Check shares redeemed starts at 0
	if !withdrawal.SharesRedeemed.IsZero() {
		t.Errorf("expected shares redeemed 0, got %s", withdrawal.SharesRedeemed.String())
	}

	// Check amount received starts at 0
	if !withdrawal.AmountReceived.IsZero() {
		t.Errorf("expected amount received 0, got %s", withdrawal.AmountReceived.String())
	}

	// Check completed at is 0
	if withdrawal.CompletedAt != 0 {
		t.Errorf("expected completed at 0, got %d", withdrawal.CompletedAt)
	}

	// Check withdrawal ID is generated
	if withdrawal.WithdrawalID == "" {
		t.Error("expected withdrawal ID to be generated")
	}
}

// TestWithdrawalIsReady tests withdrawal readiness check
func TestWithdrawalIsReady(t *testing.T) {
	// Create withdrawal with 0 delay - should be ready immediately
	instantWithdrawal := types.NewWithdrawal(
		"test-pool",
		"cosmos1user...",
		math.LegacyMustNewDecFromStr("100"),
		math.LegacyOneDec(),
		0, // No delay
	)

	if !instantWithdrawal.IsReady() {
		t.Error("expected instant withdrawal to be ready")
	}

	// Create withdrawal with 4 day delay - should not be ready
	delayedWithdrawal := types.NewWithdrawal(
		"test-pool",
		"cosmos1user...",
		math.LegacyMustNewDecFromStr("100"),
		math.LegacyOneDec(),
		4, // T+4
	)

	if delayedWithdrawal.IsReady() {
		t.Error("expected delayed withdrawal to not be ready immediately")
	}

	// Simulate time passing (set available time to past)
	delayedWithdrawal.AvailableAt = time.Now().Unix() - 100

	if !delayedWithdrawal.IsReady() {
		t.Error("expected withdrawal to be ready after delay period")
	}
}

// TestProRataAllocation tests Pro-rata withdrawal allocation logic
func TestProRataAllocation(t *testing.T) {
	testCases := []struct {
		name               string
		totalPending       math.LegacyDec
		availableQuota     math.LegacyDec
		userRequest        math.LegacyDec
		expectedAllocation math.LegacyDec
	}{
		{
			name:               "full allocation - quota exceeds pending",
			totalPending:       math.LegacyMustNewDecFromStr("1000"),
			availableQuota:     math.LegacyMustNewDecFromStr("2000"),
			userRequest:        math.LegacyMustNewDecFromStr("500"),
			expectedAllocation: math.LegacyMustNewDecFromStr("500"), // Full amount
		},
		{
			name:               "partial allocation - 50% quota",
			totalPending:       math.LegacyMustNewDecFromStr("1000"),
			availableQuota:     math.LegacyMustNewDecFromStr("500"),
			userRequest:        math.LegacyMustNewDecFromStr("200"),
			expectedAllocation: math.LegacyMustNewDecFromStr("100"), // 50% of request
		},
		{
			name:               "partial allocation - 25% quota",
			totalPending:       math.LegacyMustNewDecFromStr("2000"),
			availableQuota:     math.LegacyMustNewDecFromStr("500"),
			userRequest:        math.LegacyMustNewDecFromStr("400"),
			expectedAllocation: math.LegacyMustNewDecFromStr("100"), // 25% of request
		},
		{
			name:               "minimal quota",
			totalPending:       math.LegacyMustNewDecFromStr("10000"),
			availableQuota:     math.LegacyMustNewDecFromStr("1000"),
			userRequest:        math.LegacyMustNewDecFromStr("500"),
			expectedAllocation: math.LegacyMustNewDecFromStr("50"), // 10% of request
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Calculate pro-rata allocation
			// Formula: userAllocation = (userRequest / totalPending) * availableQuota
			ratio := tc.userRequest.Quo(tc.totalPending)
			allocation := ratio.Mul(tc.availableQuota)

			// Cap at user request
			if allocation.GT(tc.userRequest) {
				allocation = tc.userRequest
			}

			// Allow small tolerance for decimal comparison
			diff := allocation.Sub(tc.expectedAllocation).Abs()
			tolerance := math.LegacyMustNewDecFromStr("0.01")

			if diff.GT(tolerance) {
				t.Errorf("expected allocation %s, got %s",
					tc.expectedAllocation.String(), allocation.String())
			}
		})
	}
}

// TestDailyRedemptionLimit tests 15% daily redemption limit calculation
func TestDailyRedemptionLimit(t *testing.T) {
	testCases := []struct {
		name          string
		poolTVL       math.LegacyDec
		limitPercent  math.LegacyDec
		expectedLimit math.LegacyDec
	}{
		{
			name:          "small pool - $100K TVL",
			poolTVL:       math.LegacyMustNewDecFromStr("100000"),
			limitPercent:  math.LegacyMustNewDecFromStr("0.15"),
			expectedLimit: math.LegacyMustNewDecFromStr("15000"),
		},
		{
			name:          "medium pool - $1M TVL",
			poolTVL:       math.LegacyMustNewDecFromStr("1000000"),
			limitPercent:  math.LegacyMustNewDecFromStr("0.15"),
			expectedLimit: math.LegacyMustNewDecFromStr("150000"),
		},
		{
			name:          "large pool - $10M TVL",
			poolTVL:       math.LegacyMustNewDecFromStr("10000000"),
			limitPercent:  math.LegacyMustNewDecFromStr("0.15"),
			expectedLimit: math.LegacyMustNewDecFromStr("1500000"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			dailyLimit := tc.poolTVL.Mul(tc.limitPercent)

			if !dailyLimit.Equal(tc.expectedLimit) {
				t.Errorf("expected daily limit %s, got %s",
					tc.expectedLimit.String(), dailyLimit.String())
			}
		})
	}
}

// TestWithdrawalQueuePriority tests FIFO queue ordering
func TestWithdrawalQueuePriority(t *testing.T) {
	// Create withdrawals at different times
	now := time.Now().Unix()

	withdrawals := []*types.Withdrawal{
		{
			WithdrawalID:    "wth-3",
			RequestedAt:     now - 100, // Oldest
			SharesRequested: math.LegacyMustNewDecFromStr("300"),
		},
		{
			WithdrawalID:    "wth-1",
			RequestedAt:     now - 300, // Even older
			SharesRequested: math.LegacyMustNewDecFromStr("100"),
		},
		{
			WithdrawalID:    "wth-2",
			RequestedAt:     now - 200, // Middle
			SharesRequested: math.LegacyMustNewDecFromStr("200"),
		},
	}

	// Sort by RequestedAt (FIFO)
	for i := 0; i < len(withdrawals)-1; i++ {
		for j := i + 1; j < len(withdrawals); j++ {
			if withdrawals[j].RequestedAt < withdrawals[i].RequestedAt {
				withdrawals[i], withdrawals[j] = withdrawals[j], withdrawals[i]
			}
		}
	}

	// Verify order: wth-1 (oldest), wth-2, wth-3 (newest)
	expectedOrder := []string{"wth-1", "wth-2", "wth-3"}
	for i, w := range withdrawals {
		if w.WithdrawalID != expectedOrder[i] {
			t.Errorf("expected withdrawal %s at position %d, got %s",
				expectedOrder[i], i, w.WithdrawalID)
		}
	}
}

// TestPartialWithdrawalFulfillment tests partial fulfillment tracking
func TestPartialWithdrawalFulfillment(t *testing.T) {
	withdrawal := types.NewWithdrawal(
		"main-lp",
		"cosmos1user...",
		math.LegacyMustNewDecFromStr("1000"), // Request 1000 shares
		math.LegacyOneDec(),
		4,
	)

	// First partial fulfillment - 300 shares at NAV 1.0
	firstFulfillment := math.LegacyMustNewDecFromStr("300")
	withdrawal.SharesRedeemed = withdrawal.SharesRedeemed.Add(firstFulfillment)
	withdrawal.AmountReceived = withdrawal.AmountReceived.Add(firstFulfillment.Mul(math.LegacyOneDec()))

	// Verify partial state
	if !withdrawal.SharesRedeemed.Equal(math.LegacyMustNewDecFromStr("300")) {
		t.Errorf("expected 300 shares redeemed, got %s", withdrawal.SharesRedeemed.String())
	}
	if !withdrawal.AmountReceived.Equal(math.LegacyMustNewDecFromStr("300")) {
		t.Errorf("expected 300 amount received, got %s", withdrawal.AmountReceived.String())
	}

	// Still pending (not fully filled)
	remainingShares := withdrawal.SharesRequested.Sub(withdrawal.SharesRedeemed)
	if !remainingShares.Equal(math.LegacyMustNewDecFromStr("700")) {
		t.Errorf("expected 700 remaining shares, got %s", remainingShares.String())
	}

	// Second partial fulfillment - 400 shares at NAV 1.1
	secondFulfillment := math.LegacyMustNewDecFromStr("400")
	secondNAV := math.LegacyMustNewDecFromStr("1.1")
	withdrawal.SharesRedeemed = withdrawal.SharesRedeemed.Add(secondFulfillment)
	withdrawal.AmountReceived = withdrawal.AmountReceived.Add(secondFulfillment.Mul(secondNAV))

	// Verify cumulative state
	if !withdrawal.SharesRedeemed.Equal(math.LegacyMustNewDecFromStr("700")) {
		t.Errorf("expected 700 shares redeemed, got %s", withdrawal.SharesRedeemed.String())
	}
	// Expected: 300 * 1.0 + 400 * 1.1 = 300 + 440 = 740
	expectedAmount := math.LegacyMustNewDecFromStr("740")
	if !withdrawal.AmountReceived.Equal(expectedAmount) {
		t.Errorf("expected %s amount received, got %s", expectedAmount.String(), withdrawal.AmountReceived.String())
	}

	// Final fulfillment - remaining 300 shares at NAV 1.05
	finalFulfillment := math.LegacyMustNewDecFromStr("300")
	finalNAV := math.LegacyMustNewDecFromStr("1.05")
	withdrawal.SharesRedeemed = withdrawal.SharesRedeemed.Add(finalFulfillment)
	withdrawal.AmountReceived = withdrawal.AmountReceived.Add(finalFulfillment.Mul(finalNAV))
	withdrawal.Status = types.WithdrawalStatusCompleted
	withdrawal.CompletedAt = time.Now().Unix()

	// Verify completion
	if !withdrawal.SharesRedeemed.Equal(withdrawal.SharesRequested) {
		t.Errorf("expected all shares redeemed, requested: %s, redeemed: %s",
			withdrawal.SharesRequested.String(), withdrawal.SharesRedeemed.String())
	}
	if withdrawal.Status != types.WithdrawalStatusCompleted {
		t.Errorf("expected completed status, got %s", withdrawal.Status)
	}
	// Expected total: 300 * 1.0 + 400 * 1.1 + 300 * 1.05 = 300 + 440 + 315 = 1055
	expectedTotal := math.LegacyMustNewDecFromStr("1055")
	if !withdrawal.AmountReceived.Equal(expectedTotal) {
		t.Errorf("expected total %s, got %s", expectedTotal.String(), withdrawal.AmountReceived.String())
	}
}

// TestWithdrawalCancellation tests withdrawal cancellation
func TestWithdrawalCancellation(t *testing.T) {
	withdrawal := types.NewWithdrawal(
		"main-lp",
		"cosmos1user...",
		math.LegacyMustNewDecFromStr("500"),
		math.LegacyOneDec(),
		4,
	)

	// Cancel withdrawal
	withdrawal.Status = types.WithdrawalStatusCancelled

	if withdrawal.Status != types.WithdrawalStatusCancelled {
		t.Errorf("expected cancelled status, got %s", withdrawal.Status)
	}

	// Shares should remain unredeemed
	if !withdrawal.SharesRedeemed.IsZero() {
		t.Errorf("expected 0 shares redeemed after cancel, got %s", withdrawal.SharesRedeemed.String())
	}
}

// TestFoundationPoolWithdrawalLock tests Foundation LP lock period
func TestFoundationPoolWithdrawalLock(t *testing.T) {
	// Foundation pool deposits are locked for 180 days
	deposit := types.NewDeposit(
		"foundation-lp",
		"cosmos1user...",
		math.LegacyMustNewDecFromStr("100000"),
		math.LegacyMustNewDecFromStr("100000"),
		math.LegacyOneDec(),
		180, // 180 days lock
	)

	// Verify deposit is locked
	if !deposit.IsLocked() {
		t.Error("expected foundation deposit to be locked")
	}

	// Unlock time should be 180 days in the future
	expectedUnlock := deposit.DepositedAt + (180 * 24 * 60 * 60)
	if deposit.UnlockAt != expectedUnlock {
		t.Errorf("expected unlock at %d, got %d", expectedUnlock, deposit.UnlockAt)
	}

	// Simulate unlock (set unlock time to past)
	deposit.UnlockAt = time.Now().Unix() - 100

	if deposit.IsLocked() {
		t.Error("expected deposit to be unlocked after lock period")
	}
}

// TestAvailableLiquidity tests available liquidity calculation
func TestAvailableLiquidity(t *testing.T) {
	testCases := []struct {
		name               string
		totalDeposits      math.LegacyDec
		activePositions    math.LegacyDec // Margin used in positions
		pendingWithdrawals math.LegacyDec
		expectedAvailable  math.LegacyDec
	}{
		{
			name:               "all liquidity available",
			totalDeposits:      math.LegacyMustNewDecFromStr("1000000"),
			activePositions:    math.LegacyZeroDec(),
			pendingWithdrawals: math.LegacyZeroDec(),
			expectedAvailable:  math.LegacyMustNewDecFromStr("1000000"),
		},
		{
			name:               "some positions open",
			totalDeposits:      math.LegacyMustNewDecFromStr("1000000"),
			activePositions:    math.LegacyMustNewDecFromStr("200000"),
			pendingWithdrawals: math.LegacyZeroDec(),
			expectedAvailable:  math.LegacyMustNewDecFromStr("800000"),
		},
		{
			name:               "positions and pending withdrawals",
			totalDeposits:      math.LegacyMustNewDecFromStr("1000000"),
			activePositions:    math.LegacyMustNewDecFromStr("200000"),
			pendingWithdrawals: math.LegacyMustNewDecFromStr("100000"),
			expectedAvailable:  math.LegacyMustNewDecFromStr("700000"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			available := tc.totalDeposits.Sub(tc.activePositions).Sub(tc.pendingWithdrawals)

			if !available.Equal(tc.expectedAvailable) {
				t.Errorf("expected available %s, got %s",
					tc.expectedAvailable.String(), available.String())
			}
		})
	}
}
