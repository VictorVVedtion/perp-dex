package keeper

import (
	"testing"
	"time"

	"cosmossdk.io/math"
	"github.com/openalpha/perp-dex/x/clearinghouse/types"
)

// TestLiquidationConfig tests the default liquidation configuration
func TestLiquidationConfig(t *testing.T) {
	config := types.DefaultLiquidationConfig()

	// Test default values match Hyperliquid parameters
	tests := []struct {
		name     string
		got      interface{}
		expected interface{}
	}{
		{
			name:     "LargePositionThreshold",
			got:      config.LargePositionThreshold.String(),
			expected: "100000.000000000000000000",
		},
		{
			name:     "PartialLiquidationRate",
			got:      config.PartialLiquidationRate.String(),
			expected: "0.200000000000000000", // 20%
		},
		{
			name:     "CooldownPeriod",
			got:      config.CooldownPeriod,
			expected: 30 * time.Second,
		},
		{
			name:     "LiquidatorRewardRate",
			got:      config.LiquidatorRewardRate.String(),
			expected: "0.300000000000000000", // 30%
		},
		{
			name:     "InsuranceFundRate",
			got:      config.InsuranceFundRate.String(),
			expected: "0.700000000000000000", // 70%
		},
		{
			name:     "MinMaintenanceMarginRate",
			got:      config.MinMaintenanceMarginRate.String(),
			expected: "0.025000000000000000", // 2.5%
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.expected {
				t.Errorf("%s = %v, expected %v", tt.name, tt.got, tt.expected)
			}
		})
	}
}

// TestLiquidationState tests the liquidation state management
func TestLiquidationState(t *testing.T) {
	positionSize := math.LegacyNewDec(10)
	state := types.NewLiquidationState("test-position", "trader1", "BTC-USDC", positionSize)

	// Test initial state
	if state.CurrentTier != types.TierMarketOrder {
		t.Errorf("Initial tier = %v, expected TierMarketOrder", state.CurrentTier)
	}

	if !state.RemainingSize.Equal(positionSize) {
		t.Errorf("RemainingSize = %v, expected %v", state.RemainingSize, positionSize)
	}

	if state.IsInCooldown {
		t.Error("New state should not be in cooldown")
	}

	// Test cooldown
	state.StartCooldown(30 * time.Second)
	if !state.IsInCooldown {
		t.Error("State should be in cooldown after StartCooldown")
	}

	// Test CanLiquidate during cooldown
	now := time.Now()
	if state.CanLiquidate(now) {
		t.Error("Should not be able to liquidate during cooldown")
	}

	// Test CanLiquidate after cooldown
	afterCooldown := now.Add(31 * time.Second)
	if !state.CanLiquidate(afterCooldown) {
		t.Error("Should be able to liquidate after cooldown expires")
	}

	// Test EndCooldown
	state.EndCooldown()
	if state.IsInCooldown {
		t.Error("State should not be in cooldown after EndCooldown")
	}

	// Test UpdateAfterLiquidation
	liquidatedSize := math.LegacyNewDec(2)
	penalty := math.LegacyNewDec(100)
	state.UpdateAfterLiquidation(liquidatedSize, penalty, types.TierPartialLiquidation)

	expectedRemaining := math.LegacyNewDec(8)
	if !state.RemainingSize.Equal(expectedRemaining) {
		t.Errorf("RemainingSize after liquidation = %v, expected %v", state.RemainingSize, expectedRemaining)
	}

	if !state.TotalLiquidated.Equal(liquidatedSize) {
		t.Errorf("TotalLiquidated = %v, expected %v", state.TotalLiquidated, liquidatedSize)
	}

	if state.LiquidationCount != 1 {
		t.Errorf("LiquidationCount = %v, expected 1", state.LiquidationCount)
	}
}

// TestPositionHealthV2 tests the position health assessment
func TestPositionHealthV2(t *testing.T) {
	tests := []struct {
		name                   string
		positionSize           math.LegacyDec
		entryPrice             math.LegacyDec
		markPrice              math.LegacyDec
		margin                 math.LegacyDec
		maintenanceMarginRate  math.LegacyDec
		largePositionThreshold math.LegacyDec
		expectedStatus         types.HealthStatus
		expectedIsLarge        bool
	}{
		{
			name:                   "Healthy small position",
			positionSize:           math.LegacyNewDec(1),
			entryPrice:             math.LegacyNewDec(50000),
			markPrice:              math.LegacyNewDec(50000),
			margin:                 math.LegacyNewDec(5000),    // 10% margin
			maintenanceMarginRate:  math.LegacyNewDecWithPrec(25, 3), // 2.5%
			largePositionThreshold: math.LegacyNewDec(100000),
			expectedStatus:         types.HealthStatusHealthy,
			expectedIsLarge:        false,
		},
		{
			name:                   "Liquidatable small position",
			positionSize:           math.LegacyNewDec(1),
			entryPrice:             math.LegacyNewDec(50000),
			markPrice:              math.LegacyNewDec(50000),   // No price change
			margin:                 math.LegacyNewDec(1000),    // Low margin, but above 2/3 of maint margin
			maintenanceMarginRate:  math.LegacyNewDecWithPrec(25, 3), // 2.5% = $1250 maint margin
			largePositionThreshold: math.LegacyNewDec(100000),
			expectedStatus:         types.HealthStatusLiquidatable, // $1000 < $1250 but > $833 (2/3)
			expectedIsLarge:        false,
		},
		{
			name:                   "Large position above threshold",
			positionSize:           math.LegacyNewDec(3),       // 3 BTC at $50k = $150k
			entryPrice:             math.LegacyNewDec(50000),
			markPrice:              math.LegacyNewDec(50000),
			margin:                 math.LegacyNewDec(3000),    // Low margin
			maintenanceMarginRate:  math.LegacyNewDecWithPrec(25, 3),
			largePositionThreshold: math.LegacyNewDec(100000),
			expectedStatus:         types.HealthStatusLiquidatable,
			expectedIsLarge:        true,
		},
		{
			name:                   "Backstop needed - very low equity",
			positionSize:           math.LegacyNewDec(1),
			entryPrice:             math.LegacyNewDec(50000),
			markPrice:              math.LegacyNewDec(45000),   // 10% loss
			margin:                 math.LegacyNewDec(500),     // Very low margin
			maintenanceMarginRate:  math.LegacyNewDecWithPrec(25, 3),
			largePositionThreshold: math.LegacyNewDec(100000),
			expectedStatus:         types.HealthStatusBackstop,
			expectedIsLarge:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			health := types.NewPositionHealthV2(
				"trader1",
				"BTC-USDC",
				tt.positionSize,
				tt.entryPrice,
				tt.markPrice,
				tt.margin,
				tt.maintenanceMarginRate,
				tt.largePositionThreshold,
			)

			if health.Status != tt.expectedStatus {
				t.Errorf("Status = %v, expected %v", health.Status, tt.expectedStatus)
			}

			if health.IsLargePosition != tt.expectedIsLarge {
				t.Errorf("IsLargePosition = %v, expected %v", health.IsLargePosition, tt.expectedIsLarge)
			}
		})
	}
}

// TestLiquidationTier tests the liquidation tier determination
func TestLiquidationTier(t *testing.T) {
	tests := []struct {
		name         string
		tier         types.LiquidationTier
		expectedStr  string
	}{
		{
			name:        "Market Order Tier",
			tier:        types.TierMarketOrder,
			expectedStr: "market_order",
		},
		{
			name:        "Partial Liquidation Tier",
			tier:        types.TierPartialLiquidation,
			expectedStr: "partial",
		},
		{
			name:        "Backstop Liquidation Tier",
			tier:        types.TierBackstopLiquidation,
			expectedStr: "backstop",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.tier.String() != tt.expectedStr {
				t.Errorf("String() = %v, expected %v", tt.tier.String(), tt.expectedStr)
			}
		})
	}
}

// TestHealthStatus tests the health status string representation
func TestHealthStatus(t *testing.T) {
	tests := []struct {
		status      types.HealthStatus
		expectedStr string
	}{
		{types.HealthStatusHealthy, "healthy"},
		{types.HealthStatusAtRisk, "at_risk"},
		{types.HealthStatusLiquidatable, "liquidatable"},
		{types.HealthStatusBackstop, "backstop"},
	}

	for _, tt := range tests {
		t.Run(tt.expectedStr, func(t *testing.T) {
			if tt.status.String() != tt.expectedStr {
				t.Errorf("String() = %v, expected %v", tt.status.String(), tt.expectedStr)
			}
		})
	}
}

// TestPartialLiquidationCalculation tests the partial liquidation size calculation
func TestPartialLiquidationCalculation(t *testing.T) {
	config := types.DefaultLiquidationConfig()

	positionSize := math.LegacyNewDec(10) // 10 BTC
	expectedPartialSize := positionSize.Mul(config.PartialLiquidationRate) // 20% = 2 BTC

	if !expectedPartialSize.Equal(math.LegacyNewDec(2)) {
		t.Errorf("Partial liquidation size = %v, expected 2", expectedPartialSize)
	}
}

// TestBackstopThreshold tests the backstop threshold calculation
func TestBackstopThreshold(t *testing.T) {
	config := types.DefaultLiquidationConfig()

	// Backstop threshold should be 2/3 (approximately 0.6667)
	expectedThreshold := math.LegacyNewDecWithPrec(6667, 4)
	if !config.BackstopThreshold.Equal(expectedThreshold) {
		t.Errorf("BackstopThreshold = %v, expected %v", config.BackstopThreshold, expectedThreshold)
	}
}

// TestLiquidationStateFullyLiquidated tests the IsFullyLiquidated check
func TestLiquidationStateFullyLiquidated(t *testing.T) {
	positionSize := math.LegacyNewDec(10)
	state := types.NewLiquidationState("test", "trader1", "BTC-USDC", positionSize)

	if state.IsFullyLiquidated() {
		t.Error("New state should not be fully liquidated")
	}

	// Liquidate fully
	state.UpdateAfterLiquidation(positionSize, math.LegacyNewDec(100), types.TierMarketOrder)

	if !state.IsFullyLiquidated() {
		t.Error("State should be fully liquidated after liquidating full size")
	}
}

// TestCooldownMechanism tests the cooldown timing
func TestCooldownMechanism(t *testing.T) {
	state := types.NewLiquidationState("test", "trader1", "BTC-USDC", math.LegacyNewDec(10))
	config := types.DefaultLiquidationConfig()

	// Start cooldown
	state.StartCooldown(config.CooldownPeriod)

	// Should not be able to liquidate immediately
	now := time.Now()
	if state.CanLiquidate(now) {
		t.Error("Should not be able to liquidate immediately after starting cooldown")
	}

	// Should still be in cooldown at 29 seconds
	after29s := now.Add(29 * time.Second)
	if state.CanLiquidate(after29s) {
		t.Error("Should still be in cooldown at 29 seconds")
	}

	// Should be able to liquidate after cooldown expires (31 seconds to be safe)
	after31s := now.Add(31 * time.Second)
	if !state.CanLiquidate(after31s) {
		t.Error("Should be able to liquidate after cooldown expires")
	}
}

// TestPositionHealthV2NeedsLiquidation tests the NeedsLiquidation helper
func TestPositionHealthV2NeedsLiquidation(t *testing.T) {
	// Healthy position
	healthyHealth := &types.PositionHealthV2{Status: types.HealthStatusHealthy}
	if healthyHealth.NeedsLiquidation() {
		t.Error("Healthy position should not need liquidation")
	}

	// At risk position
	atRiskHealth := &types.PositionHealthV2{Status: types.HealthStatusAtRisk}
	if atRiskHealth.NeedsLiquidation() {
		t.Error("At risk position should not need liquidation yet")
	}

	// Liquidatable position
	liquidatableHealth := &types.PositionHealthV2{Status: types.HealthStatusLiquidatable}
	if !liquidatableHealth.NeedsLiquidation() {
		t.Error("Liquidatable position should need liquidation")
	}

	// Backstop position
	backstopHealth := &types.PositionHealthV2{Status: types.HealthStatusBackstop}
	if !backstopHealth.NeedsLiquidation() {
		t.Error("Backstop position should need liquidation")
	}
}

// TestRewardDistribution tests the reward distribution calculation
func TestRewardDistribution(t *testing.T) {
	config := types.DefaultLiquidationConfig()

	penalty := math.LegacyNewDec(1000) // $1000 penalty

	// Calculate rewards
	liquidatorReward := penalty.Mul(config.LiquidatorRewardRate)
	insuranceFundFee := penalty.Mul(config.InsuranceFundRate)

	expectedLiquidatorReward := math.LegacyNewDec(300) // 30%
	expectedInsuranceFee := math.LegacyNewDec(700)     // 70%

	if !liquidatorReward.Equal(expectedLiquidatorReward) {
		t.Errorf("Liquidator reward = %v, expected %v", liquidatorReward, expectedLiquidatorReward)
	}

	if !insuranceFundFee.Equal(expectedInsuranceFee) {
		t.Errorf("Insurance fund fee = %v, expected %v", insuranceFundFee, expectedInsuranceFee)
	}

	// Total should equal original penalty
	total := liquidatorReward.Add(insuranceFundFee)
	if !total.Equal(penalty) {
		t.Errorf("Total distribution = %v, expected %v", total, penalty)
	}
}

// BenchmarkPositionHealthV2Assessment benchmarks the health assessment
func BenchmarkPositionHealthV2Assessment(b *testing.B) {
	config := types.DefaultLiquidationConfig()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = types.NewPositionHealthV2(
			"trader1",
			"BTC-USDC",
			math.LegacyNewDec(1),
			math.LegacyNewDec(50000),
			math.LegacyNewDec(48000),
			math.LegacyNewDec(1000),
			config.MinMaintenanceMarginRate,
			config.LargePositionThreshold,
		)
	}
}

// BenchmarkLiquidationStateUpdate benchmarks the state update
func BenchmarkLiquidationStateUpdate(b *testing.B) {
	state := types.NewLiquidationState("test", "trader1", "BTC-USDC", math.LegacyNewDec(1000))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		state.UpdateAfterLiquidation(
			math.LegacyNewDec(1),
			math.LegacyNewDec(100),
			types.TierPartialLiquidation,
		)
	}
}
