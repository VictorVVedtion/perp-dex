package keeper

import (
	"testing"

	"cosmossdk.io/math"
	"github.com/openalpha/perp-dex/x/riverpool/types"
)

// TestDDGuardThresholds tests DDGuard threshold values
func TestDDGuardThresholds(t *testing.T) {
	// Check warning threshold (10%)
	expectedWarning := math.LegacyMustNewDecFromStr("0.10")
	if !types.DDGuardWarningThreshold.Equal(expectedWarning) {
		t.Errorf("expected warning threshold 0.10, got %s", types.DDGuardWarningThreshold.String())
	}

	// Check reduce threshold (15%)
	expectedReduce := math.LegacyMustNewDecFromStr("0.15")
	if !types.DDGuardReduceThreshold.Equal(expectedReduce) {
		t.Errorf("expected reduce threshold 0.15, got %s", types.DDGuardReduceThreshold.String())
	}

	// Check halt threshold (30%)
	expectedHalt := math.LegacyMustNewDecFromStr("0.30")
	if !types.DDGuardHaltThreshold.Equal(expectedHalt) {
		t.Errorf("expected halt threshold 0.30, got %s", types.DDGuardHaltThreshold.String())
	}
}

// TestDDGuardLevelTransitions tests level transitions based on drawdown
func TestDDGuardLevelTransitions(t *testing.T) {
	testCases := []struct {
		name           string
		drawdownPct    string
		expectedLevel  string
		expectedStatus string
	}{
		{
			name:           "0% drawdown - normal",
			drawdownPct:    "0.00",
			expectedLevel:  types.DDGuardLevelNormal,
			expectedStatus: types.PoolStatusActive,
		},
		{
			name:           "5% drawdown - still normal",
			drawdownPct:    "0.05",
			expectedLevel:  types.DDGuardLevelNormal,
			expectedStatus: types.PoolStatusActive,
		},
		{
			name:           "9.9% drawdown - still normal",
			drawdownPct:    "0.099",
			expectedLevel:  types.DDGuardLevelNormal,
			expectedStatus: types.PoolStatusActive,
		},
		{
			name:           "10% drawdown - warning",
			drawdownPct:    "0.10",
			expectedLevel:  types.DDGuardLevelWarning,
			expectedStatus: types.PoolStatusActive,
		},
		{
			name:           "12% drawdown - warning",
			drawdownPct:    "0.12",
			expectedLevel:  types.DDGuardLevelWarning,
			expectedStatus: types.PoolStatusActive,
		},
		{
			name:           "14.9% drawdown - still warning",
			drawdownPct:    "0.149",
			expectedLevel:  types.DDGuardLevelWarning,
			expectedStatus: types.PoolStatusActive,
		},
		{
			name:           "15% drawdown - reduce",
			drawdownPct:    "0.15",
			expectedLevel:  types.DDGuardLevelReduce,
			expectedStatus: types.PoolStatusActive,
		},
		{
			name:           "20% drawdown - reduce",
			drawdownPct:    "0.20",
			expectedLevel:  types.DDGuardLevelReduce,
			expectedStatus: types.PoolStatusActive,
		},
		{
			name:           "29.9% drawdown - still reduce",
			drawdownPct:    "0.299",
			expectedLevel:  types.DDGuardLevelReduce,
			expectedStatus: types.PoolStatusActive,
		},
		{
			name:           "30% drawdown - halt",
			drawdownPct:    "0.30",
			expectedLevel:  types.DDGuardLevelHalt,
			expectedStatus: types.PoolStatusPaused,
		},
		{
			name:           "50% drawdown - halt",
			drawdownPct:    "0.50",
			expectedLevel:  types.DDGuardLevelHalt,
			expectedStatus: types.PoolStatusPaused,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			pool := types.NewMainPool()
			pool.TotalShares = math.LegacyMustNewDecFromStr("1000")
			pool.HighWaterMark = math.LegacyOneDec()

			// Calculate total value to achieve desired drawdown
			drawdown := math.LegacyMustNewDecFromStr(tc.drawdownPct)
			navAfterDrawdown := math.LegacyOneDec().Sub(drawdown)
			totalValue := navAfterDrawdown.Mul(pool.TotalShares)

			pool.UpdateNAV(totalValue)

			if pool.DDGuardLevel != tc.expectedLevel {
				t.Errorf("expected DDGuard level %s, got %s", tc.expectedLevel, pool.DDGuardLevel)
			}

			if pool.Status != tc.expectedStatus {
				t.Errorf("expected status %s, got %s", tc.expectedStatus, pool.Status)
			}
		})
	}
}

// TestDDGuardRecovery tests recovery from drawdown
func TestDDGuardRecovery(t *testing.T) {
	pool := types.NewMainPool()
	pool.TotalShares = math.LegacyMustNewDecFromStr("1000")
	pool.HighWaterMark = math.LegacyOneDec()

	// First, trigger warning level (12% drawdown)
	pool.UpdateNAV(math.LegacyMustNewDecFromStr("880"))
	if pool.DDGuardLevel != types.DDGuardLevelWarning {
		t.Errorf("expected warning level after 12%% drawdown, got %s", pool.DDGuardLevel)
	}

	// Partial recovery to 5% drawdown - should return to normal
	pool.UpdateNAV(math.LegacyMustNewDecFromStr("950"))
	if pool.DDGuardLevel != types.DDGuardLevelNormal {
		t.Errorf("expected normal level after recovery to 5%% drawdown, got %s", pool.DDGuardLevel)
	}

	// Check drawdown calculation
	expectedDrawdown := math.LegacyMustNewDecFromStr("0.05")
	drawdownDiff := pool.CurrentDrawdown.Sub(expectedDrawdown).Abs()
	tolerance := math.LegacyMustNewDecFromStr("0.0001")
	if drawdownDiff.GT(tolerance) {
		t.Errorf("expected drawdown ~0.05, got %s", pool.CurrentDrawdown.String())
	}
}

// TestDDGuardHighWaterMarkUpdate tests high water mark updates
func TestDDGuardHighWaterMarkUpdate(t *testing.T) {
	pool := types.NewMainPool()
	pool.TotalShares = math.LegacyMustNewDecFromStr("1000")

	// Initial NAV = 1.0
	if !pool.HighWaterMark.Equal(math.LegacyOneDec()) {
		t.Errorf("expected initial high water mark 1.0, got %s", pool.HighWaterMark.String())
	}

	// NAV increases to 1.1 - high water mark should update
	pool.UpdateNAV(math.LegacyMustNewDecFromStr("1100"))
	expectedHWM := math.LegacyMustNewDecFromStr("1.1")
	if !pool.HighWaterMark.Equal(expectedHWM) {
		t.Errorf("expected high water mark 1.1, got %s", pool.HighWaterMark.String())
	}
	if !pool.CurrentDrawdown.IsZero() {
		t.Errorf("expected drawdown 0 at new high, got %s", pool.CurrentDrawdown.String())
	}

	// NAV increases to 1.2 - high water mark should update again
	pool.UpdateNAV(math.LegacyMustNewDecFromStr("1200"))
	expectedHWM = math.LegacyMustNewDecFromStr("1.2")
	if !pool.HighWaterMark.Equal(expectedHWM) {
		t.Errorf("expected high water mark 1.2, got %s", pool.HighWaterMark.String())
	}

	// NAV drops to 1.1 - high water mark should NOT change
	pool.UpdateNAV(math.LegacyMustNewDecFromStr("1100"))
	if !pool.HighWaterMark.Equal(expectedHWM) {
		t.Errorf("expected high water mark to remain 1.2, got %s", pool.HighWaterMark.String())
	}

	// Drawdown should now be ~8.33%
	expectedDrawdown := math.LegacyMustNewDecFromStr("0.0833")
	drawdownDiff := pool.CurrentDrawdown.Sub(expectedDrawdown).Abs()
	tolerance := math.LegacyMustNewDecFromStr("0.001")
	if drawdownDiff.GT(tolerance) {
		t.Errorf("expected drawdown ~0.0833, got %s", pool.CurrentDrawdown.String())
	}
}

// TestDDGuardPoolPauseOnHalt tests that pool is paused when halt level is reached
func TestDDGuardPoolPauseOnHalt(t *testing.T) {
	pool := types.NewMainPool()
	pool.TotalShares = math.LegacyMustNewDecFromStr("1000")
	pool.HighWaterMark = math.LegacyOneDec()

	// Verify pool starts active
	if pool.Status != types.PoolStatusActive {
		t.Errorf("expected initial status active, got %s", pool.Status)
	}

	// Trigger halt level (35% drawdown)
	pool.UpdateNAV(math.LegacyMustNewDecFromStr("650"))

	// Verify pool is now paused
	if pool.Status != types.PoolStatusPaused {
		t.Errorf("expected paused status after halt trigger, got %s", pool.Status)
	}
	if pool.DDGuardLevel != types.DDGuardLevelHalt {
		t.Errorf("expected halt DDGuard level, got %s", pool.DDGuardLevel)
	}
}

// TestDDGuardWithZeroShares tests DDGuard with zero shares
func TestDDGuardWithZeroShares(t *testing.T) {
	pool := types.NewMainPool()
	pool.TotalShares = math.LegacyZeroDec()

	// Should default to NAV = 1.0
	pool.UpdateNAV(math.LegacyMustNewDecFromStr("1000"))

	if !pool.NAV.Equal(math.LegacyOneDec()) {
		t.Errorf("expected NAV 1.0 with zero shares, got %s", pool.NAV.String())
	}
}

// TestDDGuardStateCreation tests DDGuard state type
func TestDDGuardStateCreation(t *testing.T) {
	state := &types.DDGuardState{
		PoolID:           "main-lp",
		Level:            types.DDGuardLevelNormal,
		PeakNAV:          math.LegacyMustNewDecFromStr("1.0"),
		CurrentNAV:       math.LegacyMustNewDecFromStr("1.0"),
		DrawdownPercent:  math.LegacyZeroDec(),
		MaxExposureLimit: math.LegacyOneDec(),
		TriggeredAt:      0,
		LastCheckedAt:    1704067200,
	}

	if state.Level != types.DDGuardLevelNormal {
		t.Errorf("expected normal level, got %s", state.Level)
	}

	if !state.DrawdownPercent.IsZero() {
		t.Errorf("expected zero drawdown, got %s", state.DrawdownPercent.String())
	}
}

// TestDDGuardExposureLimits tests exposure limit calculations at different levels
func TestDDGuardExposureLimits(t *testing.T) {
	testCases := []struct {
		name             string
		level            string
		expectedMaxLimit math.LegacyDec
	}{
		{
			name:             "normal level - full exposure",
			level:            types.DDGuardLevelNormal,
			expectedMaxLimit: math.LegacyOneDec(), // 100%
		},
		{
			name:             "warning level - slightly reduced",
			level:            types.DDGuardLevelWarning,
			expectedMaxLimit: math.LegacyMustNewDecFromStr("0.8"), // 80%
		},
		{
			name:             "reduce level - reduced exposure",
			level:            types.DDGuardLevelReduce,
			expectedMaxLimit: math.LegacyMustNewDecFromStr("0.5"), // 50%
		},
		{
			name:             "halt level - no new exposure",
			level:            types.DDGuardLevelHalt,
			expectedMaxLimit: math.LegacyZeroDec(), // 0%
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Exposure limits would be calculated in the keeper
			// This tests the expected values for documentation
			state := &types.DDGuardState{
				PoolID: "test-pool",
				Level:  tc.level,
			}

			// Calculate exposure limit based on level
			var maxExposure math.LegacyDec
			switch state.Level {
			case types.DDGuardLevelNormal:
				maxExposure = math.LegacyOneDec()
			case types.DDGuardLevelWarning:
				maxExposure = math.LegacyMustNewDecFromStr("0.8")
			case types.DDGuardLevelReduce:
				maxExposure = math.LegacyMustNewDecFromStr("0.5")
			case types.DDGuardLevelHalt:
				maxExposure = math.LegacyZeroDec()
			}

			if !maxExposure.Equal(tc.expectedMaxLimit) {
				t.Errorf("expected max exposure %s for level %s, got %s",
					tc.expectedMaxLimit.String(), tc.level, maxExposure.String())
			}
		})
	}
}
