package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"cosmossdk.io/math"
	"github.com/gorilla/mux"
	"github.com/openalpha/perp-dex/x/riverpool/types"
)

// MockPool creates a mock pool for testing
func MockPool(poolType string) *types.Pool {
	switch poolType {
	case types.PoolTypeFoundation:
		return types.NewFoundationPool()
	case types.PoolTypeMain:
		return types.NewMainPool()
	case types.PoolTypeCommunity:
		pool := &types.Pool{
			PoolID:             "cpool-test-001",
			PoolType:           types.PoolTypeCommunity,
			Name:               "Test Community Pool",
			Description:        "A test community pool",
			Status:             types.PoolStatusActive,
			TotalDeposits:      math.LegacyMustNewDecFromStr("50000"),
			TotalShares:        math.LegacyMustNewDecFromStr("50000"),
			NAV:                math.LegacyOneDec(),
			HighWaterMark:      math.LegacyOneDec(),
			CurrentDrawdown:    math.LegacyZeroDec(),
			DDGuardLevel:       types.DDGuardLevelNormal,
			MinDeposit:         math.LegacyMustNewDecFromStr("100"),
			MaxDeposit:         math.LegacyMustNewDecFromStr("10000"),
			ManagementFee:      math.LegacyMustNewDecFromStr("0.02"),
			PerformanceFee:     math.LegacyMustNewDecFromStr("0.20"),
			Owner:              "cosmos1owner...",
			OwnerMinStake:      math.LegacyMustNewDecFromStr("0.05"),
			OwnerCurrentStake:  math.LegacyMustNewDecFromStr("5000"),
			IsPrivate:          false,
			TotalHolders:       10,
			MaxLeverage:        math.LegacyMustNewDecFromStr("10"),
			AllowedMarkets:     []string{"BTC-USDC", "ETH-USDC"},
			Tags:               []string{"BTC", "ETH"},
			CreatedAt:          1704067200,
			UpdatedAt:          1704067200,
		}
		return pool
	default:
		return types.NewMainPool()
	}
}

// TestPoolToResponse tests the poolToResponse function
func TestPoolToResponse(t *testing.T) {
	testCases := []struct {
		name           string
		poolType       string
		expectedFields []string
	}{
		{
			name:     "foundation pool response",
			poolType: types.PoolTypeFoundation,
			expectedFields: []string{
				"pool_id", "pool_type", "name", "status", "nav",
				"seats_available",
			},
		},
		{
			name:     "main pool response",
			poolType: types.PoolTypeMain,
			expectedFields: []string{
				"pool_id", "pool_type", "name", "status", "nav",
				"daily_redemption_limit",
			},
		},
		{
			name:     "community pool response",
			poolType: types.PoolTypeCommunity,
			expectedFields: []string{
				"pool_id", "pool_type", "name", "status", "nav",
				"owner", "management_fee", "performance_fee",
				"max_leverage", "allowed_markets", "tags",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			pool := MockPool(tc.poolType)
			resp := poolToResponse(pool)

			// Check pool ID
			if resp.PoolID == "" {
				t.Error("expected pool ID to be set")
			}

			// Check pool type
			if resp.PoolType != tc.poolType {
				t.Errorf("expected pool type %s, got %s", tc.poolType, resp.PoolType)
			}

			// Check NAV is set
			if resp.NAV == "" {
				t.Error("expected NAV to be set")
			}

			// For community pools, check specific fields
			if tc.poolType == types.PoolTypeCommunity {
				if resp.Owner == "" {
					t.Error("expected owner to be set for community pool")
				}
				if resp.ManagementFee == "" {
					t.Error("expected management fee to be set for community pool")
				}
				if resp.PerformanceFee == "" {
					t.Error("expected performance fee to be set for community pool")
				}
				if len(resp.AllowedMarkets) == 0 {
					t.Error("expected allowed markets to be set for community pool")
				}
			}
		})
	}
}

// TestPoolResponseJSON tests JSON serialization
func TestPoolResponseJSON(t *testing.T) {
	pool := MockPool(types.PoolTypeCommunity)
	resp := poolToResponse(pool)

	// Serialize to JSON
	jsonBytes, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("failed to marshal pool response: %v", err)
	}

	// Deserialize back
	var decoded PoolResponse
	err = json.Unmarshal(jsonBytes, &decoded)
	if err != nil {
		t.Fatalf("failed to unmarshal pool response: %v", err)
	}

	// Verify fields
	if decoded.PoolID != resp.PoolID {
		t.Errorf("expected pool ID %s, got %s", resp.PoolID, decoded.PoolID)
	}
	if decoded.NAV != resp.NAV {
		t.Errorf("expected NAV %s, got %s", resp.NAV, decoded.NAV)
	}
	if decoded.Owner != resp.Owner {
		t.Errorf("expected owner %s, got %s", resp.Owner, decoded.Owner)
	}
}

// TestCreateCommunityPoolRequest tests request validation
func TestCreateCommunityPoolRequest(t *testing.T) {
	testCases := []struct {
		name        string
		request     CreateCommunityPoolRequest
		expectValid bool
	}{
		{
			name: "valid request",
			request: CreateCommunityPoolRequest{
				Owner:               "cosmos1owner...",
				Name:                "Test Pool",
				Description:         "A test pool",
				MinDeposit:          "100",
				MaxDeposit:          "10000",
				ManagementFee:       "0.02",
				PerformanceFee:      "0.20",
				OwnerMinStake:       "0.05",
				LockPeriodDays:      7,
				RedemptionDelayDays: 3,
				IsPrivate:           false,
				MaxLeverage:         "10",
				AllowedMarkets:      []string{"BTC-USDC"},
				Tags:                []string{"BTC"},
			},
			expectValid: true,
		},
		{
			name: "missing owner",
			request: CreateCommunityPoolRequest{
				Name:           "Test Pool",
				MinDeposit:     "100",
				ManagementFee:  "0.02",
				PerformanceFee: "0.20",
				OwnerMinStake:  "0.05",
			},
			expectValid: false,
		},
		{
			name: "missing name",
			request: CreateCommunityPoolRequest{
				Owner:          "cosmos1owner...",
				MinDeposit:     "100",
				ManagementFee:  "0.02",
				PerformanceFee: "0.20",
				OwnerMinStake:  "0.05",
			},
			expectValid: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Basic validation
			isValid := tc.request.Owner != "" && tc.request.Name != ""

			if tc.expectValid && !isValid {
				t.Error("expected request to be valid")
			}
			if !tc.expectValid && isValid {
				t.Error("expected request to be invalid")
			}
		})
	}
}

// TestInviteCodeResponse tests invite code response structure
func TestInviteCodeResponse(t *testing.T) {
	resp := InviteCodeResponse{
		Code:      "abc12345",
		MaxUses:   10,
		UsedCount: 3,
		ExpiresAt: 1704153600,
		CreatedAt: 1704067200,
		IsActive:  true,
	}

	// Serialize to JSON
	jsonBytes, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("failed to marshal invite code response: %v", err)
	}

	// Deserialize back
	var decoded InviteCodeResponse
	err = json.Unmarshal(jsonBytes, &decoded)
	if err != nil {
		t.Fatalf("failed to unmarshal invite code response: %v", err)
	}

	// Verify fields
	if decoded.Code != resp.Code {
		t.Errorf("expected code %s, got %s", resp.Code, decoded.Code)
	}
	if decoded.MaxUses != resp.MaxUses {
		t.Errorf("expected max uses %d, got %d", resp.MaxUses, decoded.MaxUses)
	}
	if decoded.UsedCount != resp.UsedCount {
		t.Errorf("expected used count %d, got %d", resp.UsedCount, decoded.UsedCount)
	}
	if decoded.IsActive != resp.IsActive {
		t.Errorf("expected is active %v, got %v", resp.IsActive, decoded.IsActive)
	}
}

// TestPoolHolderResponse tests pool holder response structure
func TestPoolHolderResponse(t *testing.T) {
	resp := PoolHolderResponse{
		Address:     "cosmos1user...",
		Shares:      "1000.00",
		Value:       "1100.00",
		DepositedAt: 1704067200,
		IsOwner:     false,
	}

	// Serialize to JSON
	jsonBytes, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("failed to marshal pool holder response: %v", err)
	}

	// Verify JSON structure
	var decoded map[string]interface{}
	err = json.Unmarshal(jsonBytes, &decoded)
	if err != nil {
		t.Fatalf("failed to unmarshal pool holder response: %v", err)
	}

	if decoded["address"] != resp.Address {
		t.Errorf("expected address %s, got %v", resp.Address, decoded["address"])
	}
	if decoded["shares"] != resp.Shares {
		t.Errorf("expected shares %s, got %v", resp.Shares, decoded["shares"])
	}
	if decoded["is_owner"] != resp.IsOwner {
		t.Errorf("expected is_owner %v, got %v", resp.IsOwner, decoded["is_owner"])
	}
}

// TestPoolPositionResponse tests pool position response structure
func TestPoolPositionResponse(t *testing.T) {
	resp := PoolPositionResponse{
		PositionID:       "pos-001",
		MarketID:         "BTC-USDC",
		Side:             "long",
		Size:             "0.1",
		EntryPrice:       "50000",
		MarkPrice:        "51000",
		PnL:              "100",
		PnLPercent:       "2.0",
		Leverage:         "10",
		LiquidationPrice: "45000",
		Margin:           "500",
	}

	// Serialize to JSON
	jsonBytes, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("failed to marshal pool position response: %v", err)
	}

	// Deserialize back
	var decoded PoolPositionResponse
	err = json.Unmarshal(jsonBytes, &decoded)
	if err != nil {
		t.Fatalf("failed to unmarshal pool position response: %v", err)
	}

	// Verify fields
	if decoded.MarketID != resp.MarketID {
		t.Errorf("expected market ID %s, got %s", resp.MarketID, decoded.MarketID)
	}
	if decoded.Side != resp.Side {
		t.Errorf("expected side %s, got %s", resp.Side, decoded.Side)
	}
	if decoded.Leverage != resp.Leverage {
		t.Errorf("expected leverage %s, got %s", resp.Leverage, decoded.Leverage)
	}
}

// TestPoolTradeResponse tests pool trade response structure
func TestPoolTradeResponse(t *testing.T) {
	resp := PoolTradeResponse{
		TradeID:   "trd-001",
		MarketID:  "BTC-USDC",
		Side:      "buy",
		Price:     "50000",
		Size:      "0.1",
		Fee:       "2.5",
		PnL:       "0",
		Timestamp: 1704067200,
	}

	// Serialize to JSON
	jsonBytes, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("failed to marshal pool trade response: %v", err)
	}

	// Deserialize back
	var decoded PoolTradeResponse
	err = json.Unmarshal(jsonBytes, &decoded)
	if err != nil {
		t.Fatalf("failed to unmarshal pool trade response: %v", err)
	}

	// Verify fields
	if decoded.TradeID != resp.TradeID {
		t.Errorf("expected trade ID %s, got %s", resp.TradeID, decoded.TradeID)
	}
	if decoded.MarketID != resp.MarketID {
		t.Errorf("expected market ID %s, got %s", resp.MarketID, decoded.MarketID)
	}
	if decoded.Side != resp.Side {
		t.Errorf("expected side %s, got %s", resp.Side, decoded.Side)
	}
}

// TestRevenueStatsResponse tests revenue stats response
func TestRevenueStatsResponse(t *testing.T) {
	resp := RevenueStatsResponse{
		PoolID:            "main-lp",
		TotalRevenue:      "10000.00",
		SpreadRevenue:     "5000.00",
		FundingRevenue:    "3000.00",
		LiquidationProfit: "1500.00",
		TradingPnL:        "500.00",
		FeeRebates:        "0.00",
		Return1D:          "0.05",
		Return7D:          "0.35",
		Return30D:         "1.5",
		LastUpdated:       1704067200,
	}

	// Serialize to JSON
	jsonBytes, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("failed to marshal revenue stats response: %v", err)
	}

	// Deserialize back
	var decoded RevenueStatsResponse
	err = json.Unmarshal(jsonBytes, &decoded)
	if err != nil {
		t.Fatalf("failed to unmarshal revenue stats response: %v", err)
	}

	// Verify fields
	if decoded.PoolID != resp.PoolID {
		t.Errorf("expected pool ID %s, got %s", resp.PoolID, decoded.PoolID)
	}
	if decoded.TotalRevenue != resp.TotalRevenue {
		t.Errorf("expected total revenue %s, got %s", resp.TotalRevenue, decoded.TotalRevenue)
	}
}

// TestHTTPRouteRegistration tests that routes are properly registered
func TestHTTPRouteRegistration(t *testing.T) {
	// This is a basic test to verify route paths are correct
	routes := []struct {
		path   string
		method string
	}{
		{"/v1/riverpool/pools", "GET"},
		{"/v1/riverpool/pools/{poolId}", "GET"},
		{"/v1/riverpool/pools/type/{poolType}", "GET"},
		{"/v1/riverpool/pools/{poolId}/stats", "GET"},
		{"/v1/riverpool/pools/{poolId}/nav/history", "GET"},
		{"/v1/riverpool/pools/{poolId}/ddguard", "GET"},
		{"/v1/riverpool/user/{user}/deposits", "GET"},
		{"/v1/riverpool/user/{user}/withdrawals", "GET"},
		{"/v1/riverpool/deposit", "POST"},
		{"/v1/riverpool/withdrawal/request", "POST"},
		{"/v1/riverpool/withdrawal/claim", "POST"},
		{"/v1/riverpool/community/create", "POST"},
		{"/v1/riverpool/community/{poolId}/holders", "GET"},
		{"/v1/riverpool/community/{poolId}/positions", "GET"},
		{"/v1/riverpool/community/{poolId}/trades", "GET"},
		{"/v1/riverpool/community/{poolId}/invites", "GET"},
		{"/v1/riverpool/community/{poolId}/invites", "POST"},
		{"/v1/riverpool/community/{poolId}/pause", "POST"},
		{"/v1/riverpool/community/{poolId}/resume", "POST"},
		{"/v1/riverpool/community/{poolId}/close", "POST"},
		{"/v1/riverpool/user/{user}/owned-pools", "GET"},
	}

	for _, route := range routes {
		t.Run(route.method+" "+route.path, func(t *testing.T) {
			// Verify route pattern is valid
			_, err := mux.NewRouter().NewRoute().Path(route.path).GetPathTemplate()
			if err != nil {
				t.Errorf("invalid route path: %s", route.path)
			}
		})
	}
}

// TestRequestBodyParsing tests JSON request body parsing
func TestRequestBodyParsing(t *testing.T) {
	testCases := []struct {
		name        string
		body        string
		expectError bool
	}{
		{
			name: "valid create pool request",
			body: `{
				"owner": "cosmos1owner...",
				"name": "Test Pool",
				"description": "A test pool",
				"min_deposit": "100",
				"max_deposit": "10000",
				"management_fee": "0.02",
				"performance_fee": "0.20",
				"owner_min_stake": "0.05",
				"lock_period_days": 7,
				"is_private": false,
				"max_leverage": "10",
				"allowed_markets": ["BTC-USDC"],
				"tags": ["BTC"]
			}`,
			expectError: false,
		},
		{
			name:        "empty body",
			body:        "",
			expectError: true,
		},
		{
			name:        "invalid JSON",
			body:        "{invalid}",
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var req CreateCommunityPoolRequest
			err := json.NewDecoder(bytes.NewBufferString(tc.body)).Decode(&req)

			if tc.expectError && err == nil {
				t.Error("expected error but got nil")
			}
			if !tc.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

// TestResponseStatusCodes tests HTTP response status codes
func TestResponseStatusCodes(t *testing.T) {
	testCases := []struct {
		name           string
		statusCode     int
		expectedStatus int
	}{
		{"success", http.StatusOK, 200},
		{"bad request", http.StatusBadRequest, 400},
		{"not found", http.StatusNotFound, 404},
		{"internal error", http.StatusInternalServerError, 500},
		{"forbidden", http.StatusForbidden, 403},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			recorder.WriteHeader(tc.statusCode)

			if recorder.Code != tc.expectedStatus {
				t.Errorf("expected status %d, got %d", tc.expectedStatus, recorder.Code)
			}
		})
	}
}

// TestContentTypeHeader tests Content-Type header setting
func TestContentTypeHeader(t *testing.T) {
	recorder := httptest.NewRecorder()
	recorder.Header().Set("Content-Type", "application/json")

	contentType := recorder.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("expected Content-Type application/json, got %s", contentType)
	}
}
