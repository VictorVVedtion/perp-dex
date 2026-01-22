package types

import (
	"cosmossdk.io/math"
)

// RiverpoolService defines the interface for RiverPool operations
type RiverpoolService interface {
	// Pool queries
	GetPools() ([]*PoolInfo, error)
	GetPool(poolID string) (*PoolInfo, error)
	GetPoolsByType(poolType string) ([]*PoolInfo, error)
	GetPoolStats(poolID string) (*PoolStats, error)
	GetNAVHistory(poolID string, days int) ([]*NAVPoint, error)
	GetDDGuardState(poolID string) (*DDGuardState, error)

	// User queries
	GetUserDeposits(user string) ([]*DepositInfo, error)
	GetUserWithdrawals(user string) ([]*WithdrawalInfo, error)
	GetUserPoolBalance(poolID, user string) (*UserBalance, error)
	GetUserOwnedPools(user string) ([]*PoolInfo, error)

	// Pool deposits/withdrawals
	GetPoolDeposits(poolID string, offset, limit int) ([]*DepositInfo, int, error)
	GetPoolWithdrawals(poolID string, offset, limit int) ([]*WithdrawalInfo, int, error)
	GetPendingWithdrawals(poolID string) ([]*WithdrawalInfo, error)

	// Estimates
	EstimateDeposit(poolID string, amount math.LegacyDec) (*DepositEstimate, error)
	EstimateWithdrawal(poolID string, shares math.LegacyDec) (*WithdrawalEstimate, error)

	// Transactions
	Deposit(poolID, user string, amount math.LegacyDec) (*DepositResult, error)
	RequestWithdrawal(poolID, user string, shares math.LegacyDec) (*WithdrawalResult, error)
	ClaimWithdrawal(withdrawalID, user string) (*ClaimResult, error)
	CancelWithdrawal(withdrawalID, user string) error

	// Revenue
	GetPoolRevenue(poolID string) (*RevenueStats, error)
	GetRevenueRecords(poolID string, limit int) ([]*RevenueRecord, error)
	GetRevenueBreakdown(poolID string) (*RevenueBreakdown, error)

	// Community Pool
	CreateCommunityPool(owner string, params *CommunityPoolParams) (*PoolInfo, error)
	UpdateCommunityPool(poolID, owner string, params *CommunityPoolParams) (*PoolInfo, error)
	GetPoolHolders(poolID string) ([]*HolderInfo, error)
	GetPoolPositions(poolID string) ([]*PositionInfo, error)
	GetPoolTrades(poolID string, limit int) ([]*PoolTradeInfo, error)
	GetInviteCodes(poolID, owner string) ([]*InviteCode, error)
	GenerateInviteCode(poolID, owner string) (*InviteCode, error)
	PlacePoolOrder(poolID, owner, marketID, side string, size, price, leverage math.LegacyDec) (*PoolOrderResult, error)
	ClosePoolPosition(poolID, owner, positionID string) (*PoolCloseResult, error)
	PausePool(poolID, owner string) error
	ResumePool(poolID, owner string) error
	ClosePool(poolID, owner string) error
}

// Data types for RiverPool service

type PoolInfo struct {
	PoolID              string `json:"pool_id"`
	PoolType            string `json:"pool_type"` // "foundation", "main", "community"
	Name                string `json:"name"`
	Description         string `json:"description"`
	Status              string `json:"status"` // "active", "paused", "closed"
	TotalDeposits       string `json:"total_deposits"`
	TotalShares         string `json:"total_shares"`
	NAV                 string `json:"nav"`
	HighWaterMark       string `json:"high_water_mark"`
	CurrentDrawdown     string `json:"current_drawdown"`
	DDGuardLevel        string `json:"dd_guard_level"` // "normal", "warning", "critical"
	MinDeposit          string `json:"min_deposit"`
	MaxDeposit          string `json:"max_deposit"`
	LockPeriodDays      int64  `json:"lock_period_days"`
	RedemptionDelayDays int64  `json:"redemption_delay_days"`
	DailyRedemptionLimit string `json:"daily_redemption_limit"`
	SeatsAvailable      int64  `json:"seats_available,omitempty"`
	Owner               string `json:"owner,omitempty"` // Community pool only
	CreatedAt           int64  `json:"created_at"`
	UpdatedAt           int64  `json:"updated_at"`
}

type PoolStats struct {
	PoolID           string `json:"pool_id"`
	TotalDeposits    string `json:"total_deposits"`
	TotalWithdrawals string `json:"total_withdrawals"`
	TotalRevenue     string `json:"total_revenue"`
	NAV              string `json:"nav"`
	APY30d           string `json:"apy_30d"`
	APY7d            string `json:"apy_7d"`
	MaxDrawdown      string `json:"max_drawdown"`
	SharpeRatio      string `json:"sharpe_ratio"`
	HolderCount      int    `json:"holder_count"`
}

type NAVPoint struct {
	Timestamp int64  `json:"timestamp"`
	NAV       string `json:"nav"`
}

type DDGuardState struct {
	PoolID          string `json:"pool_id"`
	Level           string `json:"level"` // "normal", "level1", "level2", "level3"
	CurrentDrawdown string `json:"current_drawdown"`
	HighWaterMark   string `json:"high_water_mark"`
	TriggerHistory  []DDGuardTrigger `json:"trigger_history"`
}

type DDGuardTrigger struct {
	Timestamp int64  `json:"timestamp"`
	Level     string `json:"level"`
	Drawdown  string `json:"drawdown"`
	Action    string `json:"action"`
}

type DepositInfo struct {
	DepositID   string `json:"deposit_id"`
	PoolID      string `json:"pool_id"`
	User        string `json:"user"`
	Amount      string `json:"amount"`
	Shares      string `json:"shares"`
	NAVAtDeposit string `json:"nav_at_deposit"`
	Status      string `json:"status"`
	LockedUntil int64  `json:"locked_until,omitempty"`
	CreatedAt   int64  `json:"created_at"`
}

type WithdrawalInfo struct {
	WithdrawalID    string `json:"withdrawal_id"`
	PoolID          string `json:"pool_id"`
	User            string `json:"user"`
	Shares          string `json:"shares"`
	EstimatedAmount string `json:"estimated_amount"`
	ActualAmount    string `json:"actual_amount,omitempty"`
	Status          string `json:"status"` // "pending", "claimable", "claimed", "cancelled"
	RequestedAt     int64  `json:"requested_at"`
	ClaimableAt     int64  `json:"claimable_at"`
	ClaimedAt       int64  `json:"claimed_at,omitempty"`
}

type UserBalance struct {
	PoolID      string `json:"pool_id"`
	User        string `json:"user"`
	Shares      string `json:"shares"`
	Value       string `json:"value"`
	UnrealizedPnL string `json:"unrealized_pnl"`
	DepositedAmount string `json:"deposited_amount"`
}

type DepositEstimate struct {
	PoolID        string `json:"pool_id"`
	Amount        string `json:"amount"`
	EstimatedShares string `json:"estimated_shares"`
	CurrentNAV    string `json:"current_nav"`
	MinDeposit    string `json:"min_deposit"`
	PointsReward  string `json:"points_reward,omitempty"`
}

type WithdrawalEstimate struct {
	PoolID          string `json:"pool_id"`
	Shares          string `json:"shares"`
	EstimatedAmount string `json:"estimated_amount"`
	CurrentNAV      string `json:"current_nav"`
	DelayDays       int    `json:"delay_days"`
	DailyLimit      string `json:"daily_limit"`
	QueuePosition   int    `json:"queue_position,omitempty"`
}

type DepositResult struct {
	DepositID   string `json:"deposit_id"`
	PoolID      string `json:"pool_id"`
	User        string `json:"user"`
	Amount      string `json:"amount"`
	Shares      string `json:"shares"`
	NAV         string `json:"nav"`
	LockedUntil int64  `json:"locked_until,omitempty"`
}

type WithdrawalResult struct {
	WithdrawalID string `json:"withdrawal_id"`
	PoolID       string `json:"pool_id"`
	User         string `json:"user"`
	Shares       string `json:"shares"`
	ClaimableAt  int64  `json:"claimable_at"`
}

type ClaimResult struct {
	WithdrawalID string `json:"withdrawal_id"`
	Amount       string `json:"amount"`
	ClaimedAt    int64  `json:"claimed_at"`
}

type RevenueStats struct {
	PoolID            string `json:"pool_id"`
	TotalRevenue      string `json:"total_revenue"`
	SpreadRevenue     string `json:"spread_revenue"`
	FundingRevenue    string `json:"funding_revenue"`
	LiquidationRevenue string `json:"liquidation_revenue"`
	Period            string `json:"period"`
}

type RevenueRecord struct {
	Timestamp int64  `json:"timestamp"`
	Type      string `json:"type"` // "spread", "funding", "liquidation"
	Amount    string `json:"amount"`
	TradeID   string `json:"trade_id,omitempty"`
}

type RevenueBreakdown struct {
	PoolID     string `json:"pool_id"`
	Spread     string `json:"spread"`
	Funding    string `json:"funding"`
	Liquidation string `json:"liquidation"`
	Total      string `json:"total"`
}

type CommunityPoolParams struct {
	Name             string `json:"name"`
	Description      string `json:"description"`
	MinDeposit       string `json:"min_deposit"`
	MaxDeposit       string `json:"max_deposit"`
	ManagementFee    string `json:"management_fee"`    // e.g., "0.02" for 2%
	PerformanceFee   string `json:"performance_fee"`   // e.g., "0.20" for 20%
	LockPeriodDays   int    `json:"lock_period_days"`
	RedemptionDelay  int    `json:"redemption_delay_days"`
	OwnerMinStake    string `json:"owner_min_stake"`   // e.g., "0.05" for 5%
	IsPrivate        bool   `json:"is_private"`
}

type HolderInfo struct {
	User           string `json:"user"`
	Shares         string `json:"shares"`
	SharePercent   string `json:"share_percent"`
	Value          string `json:"value"`
	DepositedAt    int64  `json:"deposited_at"`
}

type PositionInfo struct {
	PositionID string `json:"position_id"`
	MarketID   string `json:"market_id"`
	Side       string `json:"side"`
	Size       string `json:"size"`
	EntryPrice string `json:"entry_price"`
	MarkPrice  string `json:"mark_price"`
	PnL        string `json:"pnl"`
	Leverage   string `json:"leverage"`
}

type PoolTradeInfo struct {
	TradeID    string `json:"trade_id"`
	MarketID   string `json:"market_id"`
	Side       string `json:"side"`
	Size       string `json:"size"`
	Price      string `json:"price"`
	Fee        string `json:"fee"`
	PnL        string `json:"pnl"`
	ExecutedAt int64  `json:"executed_at"`
}

type InviteCode struct {
	Code       string `json:"code"`
	PoolID     string `json:"pool_id"`
	MaxUses    int    `json:"max_uses"`
	UsedCount  int    `json:"used_count"`
	ExpiresAt  int64  `json:"expires_at"`
	CreatedAt  int64  `json:"created_at"`
}

type PoolOrderResult struct {
	OrderID   string `json:"order_id"`
	PoolID    string `json:"pool_id"`
	MarketID  string `json:"market_id"`
	Side      string `json:"side"`
	Size      string `json:"size"`
	Price     string `json:"price"`
	Status    string `json:"status"`
	CreatedAt int64  `json:"created_at"`
}

type PoolCloseResult struct {
	PositionID string `json:"position_id"`
	PoolID     string `json:"pool_id"`
	RealizedPnL string `json:"realized_pnl"`
	ClosedAt   int64  `json:"closed_at"`
}
