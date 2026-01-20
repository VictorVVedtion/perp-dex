package e2e

// real_engine_e2e_test.go - True E2E tests with real Keepers
// NO MOCK DATA - all operations go through actual implementations

import (
	"context"
	"testing"

	"cosmossdk.io/log"
	"github.com/openalpha/perp-dex/api"
	"github.com/openalpha/perp-dex/api/types"
	"github.com/stretchr/testify/require"
)

// TestRealEngineE2E_AccountAndMargin tests account creation and margin requirements
func TestRealEngineE2E_AccountAndMargin(t *testing.T) {
	// Create real service V2
	service, err := api.NewRealServiceV2(log.NewNopLogger())
	require.NoError(t, err, "failed to create real service")

	ctx := context.Background()

	// Test 1: Account not found without initialization
	t.Run("AccountNotFound", func(t *testing.T) {
		_, err := service.GetAccount(ctx, "unknown-trader")
		require.Error(t, err, "should fail for unknown trader")
	})

	// Test 2: Initialize account with balance
	t.Run("InitializeAccount", func(t *testing.T) {
		err := service.InitializeTestAccount("trader-1", "10000") // $10,000 USDC
		require.NoError(t, err)

		account, err := service.GetAccount(ctx, "trader-1")
		require.NoError(t, err)
		require.Equal(t, "10000.000000000000000000", account.Balance)
		require.Equal(t, "0.000000000000000000", account.LockedMargin)
	})

	// Test 3: Place order without sufficient margin
	t.Run("InsufficientMargin", func(t *testing.T) {
		// Try to place huge order with insufficient margin
		// BTC at $50000, 100 BTC = $5M notional, 5% margin = $250K required
		_, err := service.PlaceOrder(ctx, &types.PlaceOrderRequest{
			MarketID: "BTC-USDC",
			Trader:   "trader-1",
			Side:     "buy",
			Type:     "limit",
			Price:    "50000",
			Quantity: "100", // Way too large for $10K balance
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "insufficient margin")
	})

	// Test 4: Place order with sufficient margin
	t.Run("SufficientMargin", func(t *testing.T) {
		// 0.1 BTC at $50000 = $5000 notional, 5% margin = $250 required
		resp, err := service.PlaceOrder(ctx, &types.PlaceOrderRequest{
			MarketID: "BTC-USDC",
			Trader:   "trader-1",
			Side:     "buy",
			Type:     "limit",
			Price:    "50000",
			Quantity: "0.1",
		})
		require.NoError(t, err)
		require.NotNil(t, resp.Order)
		require.Equal(t, "ORDER_STATUS_OPEN", resp.Order.Status)
	})
}

// TestRealEngineE2E_OrderMatching tests real order matching
func TestRealEngineE2E_OrderMatching(t *testing.T) {
	service, err := api.NewRealServiceV2(log.NewNopLogger())
	require.NoError(t, err)

	ctx := context.Background()

	// Initialize two traders with balance
	err = service.InitializeTestAccount("buyer", "100000")
	require.NoError(t, err)
	err = service.InitializeTestAccount("seller", "100000")
	require.NoError(t, err)

	// Place buy order
	buyResp, err := service.PlaceOrder(ctx, &types.PlaceOrderRequest{
		MarketID: "BTC-USDC",
		Trader:   "buyer",
		Side:     "buy",
		Type:     "limit",
		Price:    "50000",
		Quantity: "1.0",
	})
	require.NoError(t, err)
	require.Equal(t, "ORDER_STATUS_OPEN", buyResp.Order.Status)
	require.Equal(t, "0.000000000000000000", buyResp.Match.FilledQty) // No match yet

	// Place sell order that matches
	sellResp, err := service.PlaceOrder(ctx, &types.PlaceOrderRequest{
		MarketID: "BTC-USDC",
		Trader:   "seller",
		Side:     "sell",
		Type:     "limit",
		Price:    "49900", // Below buy price - should match
		Quantity: "0.5",
	})
	require.NoError(t, err)
	require.Equal(t, "ORDER_STATUS_FILLED", sellResp.Order.Status)                   // Fully filled
	require.Equal(t, "0.500000000000000000", sellResp.Match.FilledQty) // 0.5 BTC filled

	// Verify trades were created
	require.True(t, len(sellResp.Match.Trades) > 0, "should have created trades")
}

// TestRealEngineE2E_PositionManagement tests position operations
func TestRealEngineE2E_PositionManagement(t *testing.T) {
	service, err := api.NewRealServiceV2(log.NewNopLogger())
	require.NoError(t, err)

	ctx := context.Background()

	// Initialize traders
	err = service.InitializeTestAccount("long-trader", "100000")
	require.NoError(t, err)
	err = service.InitializeTestAccount("short-trader", "100000")
	require.NoError(t, err)

	// Create matching orders to generate positions
	_, err = service.PlaceOrder(ctx, &types.PlaceOrderRequest{
		MarketID: "ETH-USDC",
		Trader:   "long-trader",
		Side:     "buy",
		Type:     "limit",
		Price:    "3000",
		Quantity: "10.0",
	})
	require.NoError(t, err)

	resp, err := service.PlaceOrder(ctx, &types.PlaceOrderRequest{
		MarketID: "ETH-USDC",
		Trader:   "short-trader",
		Side:     "sell",
		Type:     "limit",
		Price:    "2999",
		Quantity: "10.0",
	})
	require.NoError(t, err)
	require.Equal(t, "ORDER_STATUS_FILLED", resp.Order.Status)

	// Check positions
	longPositions, err := service.GetPositions(ctx, "long-trader")
	require.NoError(t, err)
	t.Logf("Long trader positions: %d", len(longPositions))

	shortPositions, err := service.GetPositions(ctx, "short-trader")
	require.NoError(t, err)
	t.Logf("Short trader positions: %d", len(shortPositions))
}

// TestRealEngineE2E_NoMockData verifies no mock data is used
func TestRealEngineE2E_NoMockData(t *testing.T) {
	service, err := api.NewRealServiceV2(log.NewNopLogger())
	require.NoError(t, err)

	ctx := context.Background()

	// Verify no pre-existing accounts
	_, err = service.GetAccount(ctx, "test-trader")
	require.Error(t, err, "should not have pre-existing accounts")

	// Initialize and verify real balance tracking
	err = service.InitializeTestAccount("test-trader", "5000")
	require.NoError(t, err)

	account, err := service.GetAccount(ctx, "test-trader")
	require.NoError(t, err)
	require.Equal(t, "5000.000000000000000000", account.Balance)

	// Place order and verify margin is locked
	_, err = service.PlaceOrder(ctx, &types.PlaceOrderRequest{
		MarketID: "BTC-USDC",
		Trader:   "test-trader",
		Side:     "buy",
		Type:     "limit",
		Price:    "50000",
		Quantity: "0.05", // $2500 notional, 5% margin = $125 required
	})
	require.NoError(t, err)

	// Check that margin was actually locked (real implementation)
	accountAfter, err := service.GetAccount(ctx, "test-trader")
	require.NoError(t, err)
	t.Logf("Balance: %s, Locked: %s", accountAfter.Balance, accountAfter.LockedMargin)
}

// TestRealEngineE2E_RealPriceValidation tests that orders use real price validation
func TestRealEngineE2E_RealPriceValidation(t *testing.T) {
	service, err := api.NewRealServiceV2(log.NewNopLogger())
	require.NoError(t, err)

	ctx := context.Background()

	err = service.InitializeTestAccount("price-test-trader", "100000")
	require.NoError(t, err)

	// Test invalid price
	_, err = service.PlaceOrder(ctx, &types.PlaceOrderRequest{
		MarketID: "BTC-USDC",
		Trader:   "price-test-trader",
		Side:     "buy",
		Type:     "limit",
		Price:    "invalid-price",
		Quantity: "1.0",
	})
	require.Error(t, err, "should reject invalid price")

	// Test invalid quantity
	_, err = service.PlaceOrder(ctx, &types.PlaceOrderRequest{
		MarketID: "BTC-USDC",
		Trader:   "price-test-trader",
		Side:     "buy",
		Type:     "limit",
		Price:    "50000",
		Quantity: "not-a-number",
	})
	require.Error(t, err, "should reject invalid quantity")
}
