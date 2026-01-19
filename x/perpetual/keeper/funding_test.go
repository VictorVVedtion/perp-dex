package keeper

import (
	"testing"
	"time"

	"cosmossdk.io/math"
	"github.com/openalpha/perp-dex/x/perpetual/types"
)

// TestCalculateFundingRate tests the funding rate calculation
func TestCalculateFundingRate(t *testing.T) {
	tests := []struct {
		name       string
		markPrice  math.LegacyDec
		indexPrice math.LegacyDec
		wantRate   math.LegacyDec
		wantSign   int // 1 for positive, -1 for negative, 0 for zero
	}{
		{
			name:       "mark equals index - zero rate",
			markPrice:  math.LegacyNewDec(50000),
			indexPrice: math.LegacyNewDec(50000),
			wantSign:   0,
		},
		{
			name:       "mark above index - positive rate",
			markPrice:  math.LegacyNewDec(51000),
			indexPrice: math.LegacyNewDec(50000),
			wantSign:   1,
		},
		{
			name:       "mark below index - negative rate",
			markPrice:  math.LegacyNewDec(49000),
			indexPrice: math.LegacyNewDec(50000),
			wantSign:   -1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Calculate rate using the formula
			config := types.DefaultFundingConfig()
			priceDiff := tt.markPrice.Sub(tt.indexPrice)
			rate := config.DampingFactor.Mul(priceDiff).Quo(tt.indexPrice)

			// Clamp to limits
			if rate.GT(config.MaxRate) {
				rate = config.MaxRate
			} else if rate.LT(config.MinRate) {
				rate = config.MinRate
			}

			// Verify sign
			if tt.wantSign > 0 && !rate.IsPositive() {
				t.Errorf("expected positive rate, got %s", rate.String())
			}
			if tt.wantSign < 0 && !rate.IsNegative() {
				t.Errorf("expected negative rate, got %s", rate.String())
			}
			if tt.wantSign == 0 && !rate.IsZero() {
				t.Errorf("expected zero rate, got %s", rate.String())
			}

			// Verify rate is within limits
			if rate.GT(config.MaxRate) || rate.LT(config.MinRate) {
				t.Errorf("rate %s outside limits [%s, %s]",
					rate.String(), config.MinRate.String(), config.MaxRate.String())
			}
		})
	}
}

// TestFundingRateClamp tests that funding rates are clamped correctly
// Updated for new parameters: max rate ±0.5%, damping factor 0.05
func TestFundingRateClamp(t *testing.T) {
	config := types.DefaultFundingConfig()

	tests := []struct {
		name       string
		priceDiff  math.LegacyDec // as percentage of index
		wantClamped bool
	}{
		{
			name:       "1% diff - not clamped",
			priceDiff:  math.LegacyNewDecWithPrec(1, 2),
			wantClamped: false,
		},
		{
			name:       "15% diff - should clamp to max",
			priceDiff:  math.LegacyNewDecWithPrec(15, 2), // 15% diff * 0.05 damping = 0.75% > 0.5%
			wantClamped: true,
		},
		{
			name:       "-15% diff - should clamp to min",
			priceDiff:  math.LegacyNewDecWithPrec(-15, 2),
			wantClamped: true,
		},
	}

	indexPrice := math.LegacyNewDec(50000)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			markPrice := indexPrice.Mul(math.LegacyOneDec().Add(tt.priceDiff))
			priceDiff := markPrice.Sub(indexPrice)
			rate := config.DampingFactor.Mul(priceDiff).Quo(indexPrice)

			// Check if clamping is needed
			needsClamp := rate.GT(config.MaxRate) || rate.LT(config.MinRate)
			if needsClamp != tt.wantClamped {
				t.Errorf("clamping mismatch: got %v, want %v (rate=%s, max=%s)", needsClamp, tt.wantClamped, rate.String(), config.MaxRate.String())
			}
		})
	}
}

// TestFundingPayment tests funding payment calculation
func TestFundingPayment(t *testing.T) {
	rate := math.LegacyNewDecWithPrec(1, 4) // 0.01% = 0.0001
	markPrice := math.LegacyNewDec(50000)

	tests := []struct {
		name     string
		side     types.PositionSide
		size     math.LegacyDec
		wantSign int // 1 for receive, -1 for pay
	}{
		{
			name:     "long position pays positive funding",
			side:     types.PositionSideLong,
			size:     math.LegacyNewDec(1),
			wantSign: -1, // Long pays
		},
		{
			name:     "short position receives positive funding",
			side:     types.PositionSideShort,
			size:     math.LegacyNewDec(1),
			wantSign: 1, // Short receives
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			notional := tt.size.Mul(markPrice)
			payment := notional.Mul(rate)

			// Long pays, short receives
			if tt.side == types.PositionSideLong {
				payment = payment.Neg()
			}

			if tt.wantSign > 0 && !payment.IsPositive() {
				t.Errorf("expected positive payment (receive), got %s", payment.String())
			}
			if tt.wantSign < 0 && !payment.IsNegative() {
				t.Errorf("expected negative payment (pay), got %s", payment.String())
			}
		})
	}
}

// TestFundingConfig tests default funding configuration
// Updated for Hyperliquid alignment: 1h interval, ±0.5% max rate, 0.05 damping
func TestFundingConfig(t *testing.T) {
	config := types.DefaultFundingConfig()

	// Verify interval is 1 hour (3600 seconds)
	if config.Interval != 3600 {
		t.Errorf("expected interval 3600, got %d", config.Interval)
	}

	// Verify damping factor is 0.05
	expectedDamping := math.LegacyNewDecWithPrec(5, 2)
	if !config.DampingFactor.Equal(expectedDamping) {
		t.Errorf("expected damping 0.05, got %s", config.DampingFactor.String())
	}

	// Verify max rate is 0.5%
	expectedMax := math.LegacyNewDecWithPrec(5, 3)
	if !config.MaxRate.Equal(expectedMax) {
		t.Errorf("expected max rate 0.005, got %s", config.MaxRate.String())
	}

	// Verify min rate is -0.5%
	expectedMin := math.LegacyNewDecWithPrec(-5, 3)
	if !config.MinRate.Equal(expectedMin) {
		t.Errorf("expected min rate -0.005, got %s", config.MinRate.String())
	}
}

// TestFundingRate_NewFundingRate tests FundingRate constructor
func TestFundingRate_NewFundingRate(t *testing.T) {
	marketID := "BTC-USDC"
	rate := math.LegacyNewDecWithPrec(5, 5)
	markPrice := math.LegacyNewDec(50000)
	indexPrice := math.LegacyNewDec(49500)

	fundingRate := types.NewFundingRate(marketID, rate, markPrice, indexPrice)

	if fundingRate.MarketID != marketID {
		t.Errorf("expected market ID %s, got %s", marketID, fundingRate.MarketID)
	}
	if !fundingRate.Rate.Equal(rate) {
		t.Errorf("expected rate %s, got %s", rate.String(), fundingRate.Rate.String())
	}
	if fundingRate.Timestamp.IsZero() {
		t.Error("expected non-zero timestamp")
	}
}

// TestFundingPayment_NewFundingPayment tests FundingPayment constructor
func TestFundingPayment_NewFundingPayment(t *testing.T) {
	paymentID := "funding-1"
	trader := "cosmos1abc..."
	marketID := "BTC-USDC"
	amount := math.LegacyNewDec(10)
	rate := math.LegacyNewDecWithPrec(5, 5)

	payment := types.NewFundingPayment(paymentID, trader, marketID, amount, rate)

	if payment.PaymentID != paymentID {
		t.Errorf("expected payment ID %s, got %s", paymentID, payment.PaymentID)
	}
	if payment.Trader != trader {
		t.Errorf("expected trader %s, got %s", trader, payment.Trader)
	}
	if !payment.Amount.Equal(amount) {
		t.Errorf("expected amount %s, got %s", amount.String(), payment.Amount.String())
	}
}

// TestFundingSettlementTiming tests funding settlement timing
// Updated for 1 hour interval
func TestFundingSettlementTiming(t *testing.T) {
	config := types.DefaultFundingConfig()

	// Set a base time
	baseTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	// Next funding should be 1 hour later
	nextFunding := baseTime.Add(time.Duration(config.Interval) * time.Second)
	expectedNext := time.Date(2024, 1, 1, 1, 0, 0, 0, time.UTC)

	if !nextFunding.Equal(expectedNext) {
		t.Errorf("expected next funding at %v, got %v", expectedNext, nextFunding)
	}
}
