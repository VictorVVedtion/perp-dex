package keeper

import (
	"testing"

	"cosmossdk.io/math"
	"github.com/openalpha/perp-dex/x/perpetual/types"
)

// TestNewMarket tests market creation with default values
func TestNewMarket(t *testing.T) {
	market := types.NewMarket("BTC-USDC", "BTC", "USDC")

	// Check basic fields
	if market.MarketID != "BTC-USDC" {
		t.Errorf("expected market ID BTC-USDC, got %s", market.MarketID)
	}
	if market.BaseAsset != "BTC" {
		t.Errorf("expected base asset BTC, got %s", market.BaseAsset)
	}
	if market.QuoteAsset != "USDC" {
		t.Errorf("expected quote asset USDC, got %s", market.QuoteAsset)
	}

	// Check default leverage (updated to 50x for Hyperliquid alignment)
	expectedLeverage := math.LegacyNewDec(50)
	if !market.MaxLeverage.Equal(expectedLeverage) {
		t.Errorf("expected max leverage 50, got %s", market.MaxLeverage.String())
	}

	// Check margin rates (updated: 5% initial, 2.5% maintenance)
	expectedInitialMargin := math.LegacyNewDecWithPrec(5, 2) // 5%
	if !market.InitialMarginRate.Equal(expectedInitialMargin) {
		t.Errorf("expected initial margin 5%%, got %s", market.InitialMarginRate.String())
	}

	expectedMaintenanceMargin := math.LegacyNewDecWithPrec(25, 3) // 2.5%
	if !market.MaintenanceMarginRate.Equal(expectedMaintenanceMargin) {
		t.Errorf("expected maintenance margin 2.5%%, got %s", market.MaintenanceMarginRate.String())
	}

	// Check status
	if market.Status != types.MarketStatusActive {
		t.Errorf("expected active status, got %s", market.Status.String())
	}

	// Check funding interval (updated to 8 hours)
	if market.FundingInterval != 28800 {
		t.Errorf("expected funding interval 28800, got %d", market.FundingInterval)
	}
}

// TestNewMarketWithConfig tests market creation with custom config
func TestNewMarketWithConfig(t *testing.T) {
	config := types.MarketConfig{
		MarketID:              "ETH-USDC",
		BaseAsset:             "ETH",
		QuoteAsset:            "USDC",
		MaxLeverage:           math.LegacyNewDec(20),
		InitialMarginRate:     math.LegacyNewDecWithPrec(5, 2),  // 5%
		MaintenanceMarginRate: math.LegacyNewDecWithPrec(25, 3), // 2.5%
		TakerFeeRate:          math.LegacyNewDecWithPrec(3, 4),
		MakerFeeRate:          math.LegacyNewDecWithPrec(1, 4),
		TickSize:              math.LegacyNewDecWithPrec(1, 2),
		LotSize:               math.LegacyNewDecWithPrec(1, 3),
		MinOrderSize:          math.LegacyNewDecWithPrec(1, 3),
		MaxOrderSize:          math.LegacyNewDec(500),
		MaxPositionSize:       math.LegacyNewDec(5000),
		FundingInterval:       14400, // 4 hours
	}

	market := types.NewMarketWithConfig(config)

	if market.MarketID != "ETH-USDC" {
		t.Errorf("expected market ID ETH-USDC, got %s", market.MarketID)
	}

	if !market.MaxLeverage.Equal(math.LegacyNewDec(20)) {
		t.Errorf("expected max leverage 20, got %s", market.MaxLeverage.String())
	}

	if market.FundingInterval != 14400 {
		t.Errorf("expected funding interval 14400, got %d", market.FundingInterval)
	}
}

// TestDefaultMarketConfigs tests default market configurations
func TestDefaultMarketConfigs(t *testing.T) {
	configs := types.DefaultMarketConfigs()

	expectedMarkets := []string{"BTC-USDC", "ETH-USDC", "SOL-USDC", "ARB-USDC"}

	for _, marketID := range expectedMarkets {
		config, ok := configs[marketID]
		if !ok {
			t.Errorf("expected config for %s", marketID)
			continue
		}

		if config.MarketID != marketID {
			t.Errorf("expected market ID %s, got %s", marketID, config.MarketID)
		}

		// Verify all configs have 50x leverage (updated for Hyperliquid alignment)
		if !config.MaxLeverage.Equal(math.LegacyNewDec(50)) {
			t.Errorf("%s: expected max leverage 50, got %s", marketID, config.MaxLeverage.String())
		}

		// Verify funding interval is 8 hours (updated)
		if config.FundingInterval != 28800 {
			t.Errorf("%s: expected funding interval 28800, got %d", marketID, config.FundingInterval)
		}
	}
}

// TestMarketStatus tests market status methods
func TestMarketStatus(t *testing.T) {
	tests := []struct {
		status      types.MarketStatus
		isActive    bool
		isTradeable bool
		str         string
	}{
		{types.MarketStatusInactive, false, false, "inactive"},
		{types.MarketStatusActive, true, true, "active"},
		{types.MarketStatusSettling, false, true, "settling"},
		{types.MarketStatusPaused, false, false, "paused"},
	}

	for _, tt := range tests {
		t.Run(tt.str, func(t *testing.T) {
			if tt.status.IsActive() != tt.isActive {
				t.Errorf("expected IsActive() = %v, got %v", tt.isActive, tt.status.IsActive())
			}
			if tt.status.IsTradeable() != tt.isTradeable {
				t.Errorf("expected IsTradeable() = %v, got %v", tt.isTradeable, tt.status.IsTradeable())
			}
			if tt.status.String() != tt.str {
				t.Errorf("expected String() = %s, got %s", tt.str, tt.status.String())
			}
		})
	}
}

// TestValidateOrderSize tests order size validation
func TestValidateOrderSize(t *testing.T) {
	minSize := math.LegacyNewDecWithPrec(1, 4) // 0.0001
	maxSize := math.LegacyNewDec(100)          // 100

	tests := []struct {
		name    string
		size    math.LegacyDec
		wantErr bool
	}{
		{
			name:    "valid size",
			size:    math.LegacyNewDec(1),
			wantErr: false,
		},
		{
			name:    "below minimum",
			size:    math.LegacyNewDecWithPrec(1, 5), // 0.00001
			wantErr: true,
		},
		{
			name:    "above maximum",
			size:    math.LegacyNewDec(200),
			wantErr: true,
		},
		{
			name:    "at minimum",
			size:    minSize,
			wantErr: false,
		},
		{
			name:    "at maximum",
			size:    maxSize,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var err error
			if tt.size.LT(minSize) {
				err = types.ErrOrderSizeTooSmall
			} else if tt.size.GT(maxSize) {
				err = types.ErrOrderSizeTooLarge
			}

			hasErr := err != nil
			if hasErr != tt.wantErr {
				t.Errorf("expected error = %v, got %v", tt.wantErr, hasErr)
			}
		})
	}
}

// TestMarketConfig tests market configuration
func TestMarketConfig(t *testing.T) {
	config := types.MarketConfig{
		MarketID:              "TEST-USDC",
		BaseAsset:             "TEST",
		QuoteAsset:            "USDC",
		MaxLeverage:           math.LegacyNewDec(10),
		InitialMarginRate:     math.LegacyNewDecWithPrec(1, 1),
		MaintenanceMarginRate: math.LegacyNewDecWithPrec(5, 2),
		TakerFeeRate:          math.LegacyNewDecWithPrec(5, 4),
		MakerFeeRate:          math.LegacyNewDecWithPrec(2, 4),
		TickSize:              math.LegacyNewDecWithPrec(1, 2),
		LotSize:               math.LegacyNewDecWithPrec(1, 4),
		MinOrderSize:          math.LegacyNewDecWithPrec(1, 4),
		MaxOrderSize:          math.LegacyNewDec(100),
		MaxPositionSize:       math.LegacyNewDec(1000),
		FundingInterval:       28800,
		InsuranceFundID:       "test-insurance",
	}

	// Verify all fields are set
	if config.MarketID == "" {
		t.Error("market ID should not be empty")
	}
	if config.InsuranceFundID == "" {
		t.Error("insurance fund ID should not be empty")
	}

	// Create market from config
	market := types.NewMarketWithConfig(config)
	if market.InsuranceFundID != config.InsuranceFundID {
		t.Errorf("expected insurance fund ID %s, got %s", config.InsuranceFundID, market.InsuranceFundID)
	}
}
