package api

import (
	"fmt"
	"sync"
	"time"

	"cosmossdk.io/math"
	"github.com/openalpha/perp-dex/api/types"
)

// MockRiverpoolService implements types.RiverpoolService with mock data
type MockRiverpoolService struct {
	mu          sync.RWMutex
	pools       map[string]*types.PoolInfo
	deposits    map[string]*types.DepositInfo
	withdrawals map[string]*types.WithdrawalInfo
	navHistory  map[string][]*types.NAVPoint
}

// NewMockRiverpoolService creates a new mock RiverPool service
func NewMockRiverpoolService() *MockRiverpoolService {
	svc := &MockRiverpoolService{
		pools:       make(map[string]*types.PoolInfo),
		deposits:    make(map[string]*types.DepositInfo),
		withdrawals: make(map[string]*types.WithdrawalInfo),
		navHistory:  make(map[string][]*types.NAVPoint),
	}
	svc.initMockData()
	return svc
}

func (s *MockRiverpoolService) initMockData() {
	now := time.Now().Unix()

	// Foundation LP Pool
	s.pools["foundation-lp"] = &types.PoolInfo{
		PoolID:              "foundation-lp",
		PoolType:            "foundation",
		Name:                "Foundation LP",
		Description:         "Premier liquidity pool with guaranteed allocation",
		Status:              "active",
		TotalDeposits:       "5000000",
		TotalShares:         "5000000",
		NAV:                 "1.05",
		HighWaterMark:       "1.05",
		CurrentDrawdown:     "0",
		DDGuardLevel:        "normal",
		MinDeposit:          "100000",
		MaxDeposit:          "100000",
		LockPeriodDays:      180,
		RedemptionDelayDays: 7,
		DailyRedemptionLimit: "0",
		SeatsAvailable:      50,
		CreatedAt:           now - 86400*30,
		UpdatedAt:           now,
	}

	// Main LP Pool
	s.pools["main-lp"] = &types.PoolInfo{
		PoolID:              "main-lp",
		PoolType:            "main",
		Name:                "Main LP",
		Description:         "Open liquidity pool with flexible deposits",
		Status:              "active",
		TotalDeposits:       "10000000",
		TotalShares:         "9800000",
		NAV:                 "1.02",
		HighWaterMark:       "1.03",
		CurrentDrawdown:     "0.97",
		DDGuardLevel:        "normal",
		MinDeposit:          "100",
		MaxDeposit:          "0",
		LockPeriodDays:      0,
		RedemptionDelayDays: 4,
		DailyRedemptionLimit: "15",
		CreatedAt:           now - 86400*60,
		UpdatedAt:           now,
	}

	// Sample Community Pool
	s.pools["community-alpha"] = &types.PoolInfo{
		PoolID:              "community-alpha",
		PoolType:            "community",
		Name:                "Alpha Strategy",
		Description:         "High-frequency trading strategy pool",
		Status:              "active",
		TotalDeposits:       "500000",
		TotalShares:         "480000",
		NAV:                 "1.04",
		HighWaterMark:       "1.04",
		CurrentDrawdown:     "0",
		DDGuardLevel:        "normal",
		MinDeposit:          "1000",
		MaxDeposit:          "50000",
		LockPeriodDays:      30,
		RedemptionDelayDays: 7,
		DailyRedemptionLimit: "10",
		Owner:               "cosmos1owner123",
		CreatedAt:           now - 86400*14,
		UpdatedAt:           now,
	}

	// Generate NAV history for each pool
	for poolID := range s.pools {
		history := make([]*types.NAVPoint, 30)
		baseNAV := 1.0
		for i := 0; i < 30; i++ {
			baseNAV += (float64(i%3) - 1) * 0.001
			history[i] = &types.NAVPoint{
				Timestamp: now - int64((29-i)*86400),
				NAV:       fmt.Sprintf("%.4f", baseNAV),
			}
		}
		s.navHistory[poolID] = history
	}
}

// Implementation of types.RiverpoolService interface

func (s *MockRiverpoolService) GetPools() ([]*types.PoolInfo, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	pools := make([]*types.PoolInfo, 0, len(s.pools))
	for _, pool := range s.pools {
		pools = append(pools, pool)
	}
	return pools, nil
}

func (s *MockRiverpoolService) GetPool(poolID string) (*types.PoolInfo, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	pool, ok := s.pools[poolID]
	if !ok {
		return nil, fmt.Errorf("pool not found: %s", poolID)
	}
	return pool, nil
}

func (s *MockRiverpoolService) GetPoolsByType(poolType string) ([]*types.PoolInfo, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	pools := make([]*types.PoolInfo, 0)
	for _, pool := range s.pools {
		if pool.PoolType == poolType {
			pools = append(pools, pool)
		}
	}
	return pools, nil
}

func (s *MockRiverpoolService) GetPoolStats(poolID string) (*types.PoolStats, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	pool, ok := s.pools[poolID]
	if !ok {
		return nil, fmt.Errorf("pool not found: %s", poolID)
	}

	return &types.PoolStats{
		PoolID:           pool.PoolID,
		TotalDeposits:    pool.TotalDeposits,
		TotalWithdrawals: "1000000",
		TotalRevenue:     "50000",
		NAV:              pool.NAV,
		APY30d:           "12.5",
		APY7d:            "15.2",
		MaxDrawdown:      "3.5",
		SharpeRatio:      "2.1",
		HolderCount:      50,
	}, nil
}

func (s *MockRiverpoolService) GetNAVHistory(poolID string, days int) ([]*types.NAVPoint, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	history, ok := s.navHistory[poolID]
	if !ok {
		return nil, fmt.Errorf("pool not found: %s", poolID)
	}

	if days > len(history) {
		days = len(history)
	}
	return history[len(history)-days:], nil
}

func (s *MockRiverpoolService) GetDDGuardState(poolID string) (*types.DDGuardState, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	pool, ok := s.pools[poolID]
	if !ok {
		return nil, fmt.Errorf("pool not found: %s", poolID)
	}

	return &types.DDGuardState{
		PoolID:          pool.PoolID,
		Level:           pool.DDGuardLevel,
		CurrentDrawdown: pool.CurrentDrawdown,
		HighWaterMark:   pool.HighWaterMark,
		TriggerHistory:  []types.DDGuardTrigger{},
	}, nil
}

func (s *MockRiverpoolService) GetUserDeposits(user string) ([]*types.DepositInfo, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	deposits := make([]*types.DepositInfo, 0)
	for _, d := range s.deposits {
		if d.User == user {
			deposits = append(deposits, d)
		}
	}
	return deposits, nil
}

func (s *MockRiverpoolService) GetUserWithdrawals(user string) ([]*types.WithdrawalInfo, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	withdrawals := make([]*types.WithdrawalInfo, 0)
	for _, w := range s.withdrawals {
		if w.User == user {
			withdrawals = append(withdrawals, w)
		}
	}
	return withdrawals, nil
}

func (s *MockRiverpoolService) GetUserPoolBalance(poolID, user string) (*types.UserBalance, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	_, ok := s.pools[poolID]
	if !ok {
		return nil, fmt.Errorf("pool not found: %s", poolID)
	}

	return &types.UserBalance{
		PoolID:          poolID,
		User:            user,
		Shares:          "1000",
		Value:           "1020",
		UnrealizedPnL:   "20",
		DepositedAmount: "1000",
	}, nil
}

func (s *MockRiverpoolService) GetUserOwnedPools(user string) ([]*types.PoolInfo, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	pools := make([]*types.PoolInfo, 0)
	for _, pool := range s.pools {
		if pool.Owner == user {
			pools = append(pools, pool)
		}
	}
	return pools, nil
}

func (s *MockRiverpoolService) GetPoolDeposits(poolID string, offset, limit int) ([]*types.DepositInfo, int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	_, ok := s.pools[poolID]
	if !ok {
		return nil, 0, fmt.Errorf("pool not found: %s", poolID)
	}

	deposits := make([]*types.DepositInfo, 0)
	for _, d := range s.deposits {
		if d.PoolID == poolID {
			deposits = append(deposits, d)
		}
	}
	return deposits, len(deposits), nil
}

func (s *MockRiverpoolService) GetPoolWithdrawals(poolID string, offset, limit int) ([]*types.WithdrawalInfo, int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	_, ok := s.pools[poolID]
	if !ok {
		return nil, 0, fmt.Errorf("pool not found: %s", poolID)
	}

	withdrawals := make([]*types.WithdrawalInfo, 0)
	for _, w := range s.withdrawals {
		if w.PoolID == poolID {
			withdrawals = append(withdrawals, w)
		}
	}
	return withdrawals, len(withdrawals), nil
}

func (s *MockRiverpoolService) GetPendingWithdrawals(poolID string) ([]*types.WithdrawalInfo, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	_, ok := s.pools[poolID]
	if !ok {
		return nil, fmt.Errorf("pool not found: %s", poolID)
	}

	withdrawals := make([]*types.WithdrawalInfo, 0)
	for _, w := range s.withdrawals {
		if w.PoolID == poolID && w.Status == "pending" {
			withdrawals = append(withdrawals, w)
		}
	}
	return withdrawals, nil
}

func (s *MockRiverpoolService) EstimateDeposit(poolID string, amount math.LegacyDec) (*types.DepositEstimate, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	pool, ok := s.pools[poolID]
	if !ok {
		return nil, fmt.Errorf("pool not found: %s", poolID)
	}

	nav, _ := math.LegacyNewDecFromStr(pool.NAV)
	shares := amount.Quo(nav)

	return &types.DepositEstimate{
		PoolID:          poolID,
		Amount:          amount.String(),
		EstimatedShares: shares.String(),
		CurrentNAV:      pool.NAV,
		MinDeposit:      pool.MinDeposit,
		PointsReward:    "5000000",
	}, nil
}

func (s *MockRiverpoolService) EstimateWithdrawal(poolID string, shares math.LegacyDec) (*types.WithdrawalEstimate, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	pool, ok := s.pools[poolID]
	if !ok {
		return nil, fmt.Errorf("pool not found: %s", poolID)
	}

	nav, _ := math.LegacyNewDecFromStr(pool.NAV)
	amount := shares.Mul(nav)

	return &types.WithdrawalEstimate{
		PoolID:          poolID,
		Shares:          shares.String(),
		EstimatedAmount: amount.String(),
		CurrentNAV:      pool.NAV,
		DelayDays:       int(pool.RedemptionDelayDays),
		DailyLimit:      pool.DailyRedemptionLimit,
		QueuePosition:   0,
	}, nil
}

func (s *MockRiverpoolService) Deposit(poolID, user string, amount math.LegacyDec) (*types.DepositResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	pool, ok := s.pools[poolID]
	if !ok {
		return nil, fmt.Errorf("pool not found: %s", poolID)
	}

	nav, _ := math.LegacyNewDecFromStr(pool.NAV)
	shares := amount.Quo(nav)
	now := time.Now().Unix()
	depositID := fmt.Sprintf("dep_%d", now)

	deposit := &types.DepositInfo{
		DepositID:    depositID,
		PoolID:       poolID,
		User:         user,
		Amount:       amount.String(),
		Shares:       shares.String(),
		NAVAtDeposit: pool.NAV,
		Status:       "confirmed",
		LockedUntil:  now + pool.LockPeriodDays*86400,
		CreatedAt:    now,
	}
	s.deposits[depositID] = deposit

	return &types.DepositResult{
		DepositID:   depositID,
		PoolID:      poolID,
		User:        user,
		Amount:      amount.String(),
		Shares:      shares.String(),
		NAV:         pool.NAV,
		LockedUntil: deposit.LockedUntil,
	}, nil
}

func (s *MockRiverpoolService) RequestWithdrawal(poolID, user string, shares math.LegacyDec) (*types.WithdrawalResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	pool, ok := s.pools[poolID]
	if !ok {
		return nil, fmt.Errorf("pool not found: %s", poolID)
	}

	now := time.Now().Unix()
	withdrawalID := fmt.Sprintf("wd_%d", now)
	claimableAt := now + pool.RedemptionDelayDays*86400

	nav, _ := math.LegacyNewDecFromStr(pool.NAV)
	amount := shares.Mul(nav)

	withdrawal := &types.WithdrawalInfo{
		WithdrawalID:    withdrawalID,
		PoolID:          poolID,
		User:            user,
		Shares:          shares.String(),
		EstimatedAmount: amount.String(),
		Status:          "pending",
		RequestedAt:     now,
		ClaimableAt:     claimableAt,
	}
	s.withdrawals[withdrawalID] = withdrawal

	return &types.WithdrawalResult{
		WithdrawalID: withdrawalID,
		PoolID:       poolID,
		User:         user,
		Shares:       shares.String(),
		ClaimableAt:  claimableAt,
	}, nil
}

func (s *MockRiverpoolService) ClaimWithdrawal(withdrawalID, user string) (*types.ClaimResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	withdrawal, ok := s.withdrawals[withdrawalID]
	if !ok {
		return nil, fmt.Errorf("withdrawal not found: %s", withdrawalID)
	}
	if withdrawal.User != user {
		return nil, fmt.Errorf("unauthorized")
	}
	if withdrawal.Status != "pending" && withdrawal.Status != "claimable" {
		return nil, fmt.Errorf("withdrawal not claimable")
	}

	now := time.Now().Unix()
	withdrawal.Status = "claimed"
	withdrawal.ActualAmount = withdrawal.EstimatedAmount
	withdrawal.ClaimedAt = now

	return &types.ClaimResult{
		WithdrawalID: withdrawalID,
		Amount:       withdrawal.ActualAmount,
		ClaimedAt:    now,
	}, nil
}

func (s *MockRiverpoolService) CancelWithdrawal(withdrawalID, user string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	withdrawal, ok := s.withdrawals[withdrawalID]
	if !ok {
		return fmt.Errorf("withdrawal not found: %s", withdrawalID)
	}
	if withdrawal.User != user {
		return fmt.Errorf("unauthorized")
	}
	if withdrawal.Status != "pending" {
		return fmt.Errorf("only pending withdrawals can be cancelled")
	}

	withdrawal.Status = "cancelled"
	return nil
}

func (s *MockRiverpoolService) GetPoolRevenue(poolID string) (*types.RevenueStats, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	_, ok := s.pools[poolID]
	if !ok {
		return nil, fmt.Errorf("pool not found: %s", poolID)
	}

	return &types.RevenueStats{
		PoolID:             poolID,
		TotalRevenue:       "50000",
		SpreadRevenue:      "30000",
		FundingRevenue:     "15000",
		LiquidationRevenue: "5000",
		Period:             "30d",
	}, nil
}

func (s *MockRiverpoolService) GetRevenueRecords(poolID string, limit int) ([]*types.RevenueRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	_, ok := s.pools[poolID]
	if !ok {
		return nil, fmt.Errorf("pool not found: %s", poolID)
	}

	now := time.Now().Unix()
	records := make([]*types.RevenueRecord, 0, limit)
	for i := 0; i < limit && i < 10; i++ {
		records = append(records, &types.RevenueRecord{
			Timestamp: now - int64(i*3600),
			Type:      []string{"spread", "funding", "liquidation"}[i%3],
			Amount:    fmt.Sprintf("%d", (i+1)*100),
		})
	}
	return records, nil
}

func (s *MockRiverpoolService) GetRevenueBreakdown(poolID string) (*types.RevenueBreakdown, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	_, ok := s.pools[poolID]
	if !ok {
		return nil, fmt.Errorf("pool not found: %s", poolID)
	}

	return &types.RevenueBreakdown{
		PoolID:      poolID,
		Spread:      "30000",
		Funding:     "15000",
		Liquidation: "5000",
		Total:       "50000",
	}, nil
}

func (s *MockRiverpoolService) CreateCommunityPool(owner string, params *types.CommunityPoolParams) (*types.PoolInfo, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().Unix()
	poolID := fmt.Sprintf("community-%d", now)

	pool := &types.PoolInfo{
		PoolID:              poolID,
		PoolType:            "community",
		Name:                params.Name,
		Description:         params.Description,
		Status:              "active",
		TotalDeposits:       "0",
		TotalShares:         "0",
		NAV:                 "1.0",
		HighWaterMark:       "1.0",
		CurrentDrawdown:     "0",
		DDGuardLevel:        "normal",
		MinDeposit:          params.MinDeposit,
		MaxDeposit:          params.MaxDeposit,
		LockPeriodDays:      int64(params.LockPeriodDays),
		RedemptionDelayDays: int64(params.RedemptionDelay),
		DailyRedemptionLimit: "10",
		Owner:               owner,
		CreatedAt:           now,
		UpdatedAt:           now,
	}
	s.pools[poolID] = pool

	return pool, nil
}

func (s *MockRiverpoolService) UpdateCommunityPool(poolID, owner string, params *types.CommunityPoolParams) (*types.PoolInfo, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	pool, ok := s.pools[poolID]
	if !ok {
		return nil, fmt.Errorf("pool not found: %s", poolID)
	}

	if pool.Owner != owner {
		return nil, fmt.Errorf("unauthorized: not pool owner")
	}

	if params.Name != "" {
		pool.Name = params.Name
	}
	if params.Description != "" {
		pool.Description = params.Description
	}
	pool.UpdatedAt = time.Now().Unix()

	return pool, nil
}

func (s *MockRiverpoolService) GetPoolHolders(poolID string) ([]*types.HolderInfo, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	_, ok := s.pools[poolID]
	if !ok {
		return nil, fmt.Errorf("pool not found: %s", poolID)
	}

	return []*types.HolderInfo{
		{User: "cosmos1holder1", Shares: "10000", SharePercent: "20", Value: "10200", DepositedAt: time.Now().Unix() - 86400*7},
		{User: "cosmos1holder2", Shares: "5000", SharePercent: "10", Value: "5100", DepositedAt: time.Now().Unix() - 86400*3},
	}, nil
}

func (s *MockRiverpoolService) GetPoolPositions(poolID string) ([]*types.PositionInfo, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	pool, ok := s.pools[poolID]
	if !ok {
		return nil, fmt.Errorf("pool not found: %s", poolID)
	}

	if pool.PoolType != "community" {
		return []*types.PositionInfo{}, nil
	}

	return []*types.PositionInfo{
		{PositionID: "pos_1", MarketID: "BTC-USDC", Side: "long", Size: "0.5", EntryPrice: "50000", MarkPrice: "51000", PnL: "500", Leverage: "5"},
	}, nil
}

func (s *MockRiverpoolService) GetPoolTrades(poolID string, limit int) ([]*types.PoolTradeInfo, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	pool, ok := s.pools[poolID]
	if !ok {
		return nil, fmt.Errorf("pool not found: %s", poolID)
	}

	if pool.PoolType != "community" {
		return []*types.PoolTradeInfo{}, nil
	}

	now := time.Now().Unix()
	trades := make([]*types.PoolTradeInfo, 0, limit)
	for i := 0; i < limit && i < 5; i++ {
		trades = append(trades, &types.PoolTradeInfo{
			TradeID:    fmt.Sprintf("trade_%d", i),
			MarketID:   "BTC-USDC",
			Side:       []string{"buy", "sell"}[i%2],
			Size:       "0.1",
			Price:      "50000",
			Fee:        "5",
			PnL:        "100",
			ExecutedAt: now - int64(i*3600),
		})
	}
	return trades, nil
}

func (s *MockRiverpoolService) GetInviteCodes(poolID, owner string) ([]*types.InviteCode, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	pool, ok := s.pools[poolID]
	if !ok {
		return nil, fmt.Errorf("pool not found: %s", poolID)
	}

	if pool.Owner != owner {
		return nil, fmt.Errorf("unauthorized: not pool owner")
	}

	now := time.Now().Unix()
	return []*types.InviteCode{
		{Code: "ALPHA123", PoolID: poolID, MaxUses: 10, UsedCount: 3, ExpiresAt: now + 86400*30, CreatedAt: now - 86400*7},
	}, nil
}

func (s *MockRiverpoolService) GenerateInviteCode(poolID, owner string) (*types.InviteCode, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	pool, ok := s.pools[poolID]
	if !ok {
		return nil, fmt.Errorf("pool not found: %s", poolID)
	}

	if pool.Owner != owner {
		return nil, fmt.Errorf("unauthorized: not pool owner")
	}

	now := time.Now().Unix()
	return &types.InviteCode{
		Code:      fmt.Sprintf("INV%d", now),
		PoolID:    poolID,
		MaxUses:   10,
		UsedCount: 0,
		ExpiresAt: now + 86400*30,
		CreatedAt: now,
	}, nil
}

func (s *MockRiverpoolService) PlacePoolOrder(poolID, owner, marketID, side string, size, price, leverage math.LegacyDec) (*types.PoolOrderResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	pool, ok := s.pools[poolID]
	if !ok {
		return nil, fmt.Errorf("pool not found: %s", poolID)
	}

	if pool.Owner != owner {
		return nil, fmt.Errorf("unauthorized: not pool owner")
	}

	if pool.PoolType != "community" {
		return nil, fmt.Errorf("orders only allowed for community pools")
	}

	now := time.Now().Unix()
	return &types.PoolOrderResult{
		OrderID:   fmt.Sprintf("order_%d", now),
		PoolID:    poolID,
		MarketID:  marketID,
		Side:      side,
		Size:      size.String(),
		Price:     price.String(),
		Status:    "filled",
		CreatedAt: now,
	}, nil
}

func (s *MockRiverpoolService) ClosePoolPosition(poolID, owner, positionID string) (*types.PoolCloseResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	pool, ok := s.pools[poolID]
	if !ok {
		return nil, fmt.Errorf("pool not found: %s", poolID)
	}

	if pool.Owner != owner {
		return nil, fmt.Errorf("unauthorized: not pool owner")
	}

	now := time.Now().Unix()
	return &types.PoolCloseResult{
		PositionID:  positionID,
		PoolID:      poolID,
		RealizedPnL: "500",
		ClosedAt:    now,
	}, nil
}

func (s *MockRiverpoolService) PausePool(poolID, owner string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	pool, ok := s.pools[poolID]
	if !ok {
		return fmt.Errorf("pool not found: %s", poolID)
	}

	if pool.Owner != owner {
		return fmt.Errorf("unauthorized: not pool owner")
	}

	pool.Status = "paused"
	pool.UpdatedAt = time.Now().Unix()
	return nil
}

func (s *MockRiverpoolService) ResumePool(poolID, owner string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	pool, ok := s.pools[poolID]
	if !ok {
		return fmt.Errorf("pool not found: %s", poolID)
	}

	if pool.Owner != owner {
		return fmt.Errorf("unauthorized: not pool owner")
	}

	pool.Status = "active"
	pool.UpdatedAt = time.Now().Unix()
	return nil
}

func (s *MockRiverpoolService) ClosePool(poolID, owner string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	pool, ok := s.pools[poolID]
	if !ok {
		return fmt.Errorf("pool not found: %s", poolID)
	}

	if pool.Owner != owner {
		return fmt.Errorf("unauthorized: not pool owner")
	}

	pool.Status = "closed"
	pool.UpdatedAt = time.Now().Unix()
	return nil
}
