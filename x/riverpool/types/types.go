package types

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"time"

	"cosmossdk.io/math"
)

// Module name and store key
const (
	ModuleName = "riverpool"
	StoreKey   = ModuleName
)

// Pool types
const (
	PoolTypeFoundation = "foundation"
	PoolTypeMain       = "main"
	PoolTypeCommunity  = "community"
)

// Pool status
const (
	PoolStatusActive = "active"
	PoolStatusPaused = "paused"
	PoolStatusClosed = "closed"
)

// DDGuard levels
const (
	DDGuardLevelNormal  = "normal"
	DDGuardLevelWarning = "warning" // >= 10%
	DDGuardLevelReduce  = "reduce"  // >= 15%
	DDGuardLevelHalt    = "halt"    // >= 30%
)

// Withdrawal status
const (
	WithdrawalStatusPending    = "pending"
	WithdrawalStatusProcessing = "processing"
	WithdrawalStatusCompleted  = "completed"
	WithdrawalStatusCancelled  = "cancelled"
)

// DDGuard thresholds
var (
	DDGuardWarningThreshold = math.LegacyMustNewDecFromStr("0.10") // 10%
	DDGuardReduceThreshold  = math.LegacyMustNewDecFromStr("0.15") // 15%
	DDGuardHaltThreshold    = math.LegacyMustNewDecFromStr("0.30") // 30%
)

// Foundation LP constants
var (
	FoundationSeatCount    = int64(100)
	FoundationSeatSize     = math.LegacyMustNewDecFromStr("100000") // $100K
	FoundationLockDays     = int64(180)                              // 180 days
	FoundationPointsPerSeat = math.LegacyMustNewDecFromStr("5000000") // 5M points
)

// Main LP constants
var (
	MainMinDeposit          = math.LegacyMustNewDecFromStr("100")   // $100
	MainRedemptionDelayDays = int64(4)                               // T+4
	MainDailyRedemptionLimit = math.LegacyMustNewDecFromStr("0.15") // 15%
)

// Errors
var (
	ErrPoolNotFound           = errors.New("pool not found")
	ErrPoolNotActive          = errors.New("pool is not active")
	ErrPoolNotPaused          = errors.New("pool is not paused")
	ErrPoolAlreadyExists      = errors.New("pool already exists")
	ErrPoolHasDeposits        = errors.New("pool has deposits, cannot close")
	ErrDepositTooSmall        = errors.New("deposit amount below minimum")
	ErrDepositTooLarge        = errors.New("deposit amount exceeds maximum")
	ErrInsufficientShares     = errors.New("insufficient shares for withdrawal")
	ErrWithdrawalLocked       = errors.New("withdrawal is locked")
	ErrWithdrawalNotReady     = errors.New("withdrawal not yet available")
	ErrWithdrawalNotFound     = errors.New("withdrawal not found")
	ErrInvalidInviteCode      = errors.New("invalid invite code for private pool")
	ErrFoundationPoolFull     = errors.New("foundation pool is full")
	ErrOwnerStakeTooLow       = errors.New("owner stake must be at least 5%")
	ErrUnauthorized           = errors.New("unauthorized")
	ErrDDGuardHalt            = errors.New("pool trading halted due to DDGuard")
	ErrNotPoolOwner           = errors.New("not pool owner")
	ErrInvalidPoolName        = errors.New("invalid pool name")
	ErrInvalidOwner           = errors.New("invalid owner address")
	ErrInvalidMinDeposit      = errors.New("invalid minimum deposit amount")
	ErrInvalidOwnerStake      = errors.New("invalid owner minimum stake percentage")
	ErrInvalidManagementFee   = errors.New("invalid management fee (max 5%)")
	ErrInvalidPerformanceFee  = errors.New("invalid performance fee (max 50%)")
	ErrInvalidRedemptionLimit = errors.New("invalid daily redemption limit")
)

// Pool represents a liquidity pool
type Pool struct {
	PoolID      string         `json:"pool_id"`
	PoolType    string         `json:"pool_type"`
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Status      string         `json:"status"`

	// Financial metrics
	TotalDeposits   math.LegacyDec `json:"total_deposits"`
	TotalShares     math.LegacyDec `json:"total_shares"`
	NAV             math.LegacyDec `json:"nav"`              // NAV per share
	HighWaterMark   math.LegacyDec `json:"high_water_mark"`
	CurrentDrawdown math.LegacyDec `json:"current_drawdown"`

	// DDGuard state
	DDGuardLevel string `json:"dd_guard_level"`

	// Configuration
	MinDeposit           math.LegacyDec `json:"min_deposit"`
	MaxDeposit           math.LegacyDec `json:"max_deposit"`
	LockPeriodDays       int64          `json:"lock_period_days"`
	RedemptionDelayDays  int64          `json:"redemption_delay_days"`
	DailyRedemptionLimit math.LegacyDec `json:"daily_redemption_limit"`

	// Fee structure
	ManagementFee  math.LegacyDec `json:"management_fee"`  // Annual % (e.g., 0.02 for 2%)
	PerformanceFee math.LegacyDec `json:"performance_fee"` // % of profits (e.g., 0.20 for 20%)

	// Community pool specific
	Owner              string         `json:"owner,omitempty"`
	OwnerMinStake      math.LegacyDec `json:"owner_min_stake,omitempty"`      // Min % owner must stake (e.g., 0.05 for 5%)
	OwnerCurrentStake  math.LegacyDec `json:"owner_current_stake,omitempty"`  // Current owner stake amount
	IsPrivate          bool           `json:"is_private,omitempty"`
	InviteCode         string         `json:"invite_code,omitempty"`
	TotalHolders       int64          `json:"total_holders,omitempty"`        // Number of unique depositors
	AllowedMarkets     []string       `json:"allowed_markets,omitempty"`      // Markets owner can trade
	MaxLeverage        math.LegacyDec `json:"max_leverage,omitempty"`         // Max leverage allowed (e.g., 10)
	Tags               []string       `json:"tags,omitempty"`                 // Pool tags for discovery

	// Foundation LP specific
	SeatsAvailable int64 `json:"seats_available,omitempty"`
	SeatsTotal     int64 `json:"seats_total,omitempty"`

	// Timestamps
	CreatedAt int64 `json:"created_at"`
	UpdatedAt int64 `json:"updated_at"`
}

// NewFoundationPool creates a new Foundation LP pool
func NewFoundationPool() *Pool {
	now := time.Now().Unix()
	return &Pool{
		PoolID:               "foundation-lp",
		PoolType:             PoolTypeFoundation,
		Name:                 "Foundation LP",
		Description:          "100 seats x $100K, 180-day lock, 5M Points per seat",
		Status:               PoolStatusActive,
		TotalDeposits:        math.LegacyZeroDec(),
		TotalShares:          math.LegacyZeroDec(),
		NAV:                  math.LegacyOneDec(), // Initial NAV = 1.0
		HighWaterMark:        math.LegacyOneDec(),
		CurrentDrawdown:      math.LegacyZeroDec(),
		DDGuardLevel:         DDGuardLevelNormal,
		MinDeposit:           FoundationSeatSize,
		MaxDeposit:           FoundationSeatSize,
		LockPeriodDays:       FoundationLockDays,
		RedemptionDelayDays:  0, // N/A during lock
		DailyRedemptionLimit: math.LegacyZeroDec(),
		ManagementFee:        math.LegacyZeroDec(),
		PerformanceFee:       math.LegacyZeroDec(),
		SeatsAvailable:       FoundationSeatCount,
		SeatsTotal:           FoundationSeatCount,
		CreatedAt:            now,
		UpdatedAt:            now,
	}
}

// NewMainPool creates a new Main LP pool
func NewMainPool() *Pool {
	now := time.Now().Unix()
	return &Pool{
		PoolID:               "main-lp",
		PoolType:             PoolTypeMain,
		Name:                 "Main LP",
		Description:          "$100 minimum, no lock, T+4 redemption, 15% daily limit",
		Status:               PoolStatusActive,
		TotalDeposits:        math.LegacyZeroDec(),
		TotalShares:          math.LegacyZeroDec(),
		NAV:                  math.LegacyOneDec(), // Initial NAV = 1.0
		HighWaterMark:        math.LegacyOneDec(),
		CurrentDrawdown:      math.LegacyZeroDec(),
		DDGuardLevel:         DDGuardLevelNormal,
		MinDeposit:           MainMinDeposit,
		MaxDeposit:           math.LegacyZeroDec(), // No maximum
		LockPeriodDays:       0,
		RedemptionDelayDays:  MainRedemptionDelayDays,
		DailyRedemptionLimit: MainDailyRedemptionLimit,
		ManagementFee:        math.LegacyZeroDec(),
		PerformanceFee:       math.LegacyZeroDec(),
		CreatedAt:            now,
		UpdatedAt:            now,
	}
}

// GetSeatCount returns the number of filled seats (Foundation LP only)
func (p *Pool) GetSeatCount() int64 {
	if p.PoolType != PoolTypeFoundation {
		return 0
	}
	if p.TotalDeposits.IsZero() {
		return 0
	}
	return p.TotalDeposits.Quo(FoundationSeatSize).TruncateInt64()
}

// HasAvailableSeats checks if Foundation LP has available seats
func (p *Pool) HasAvailableSeats() bool {
	if p.PoolType != PoolTypeFoundation {
		return true
	}
	return p.GetSeatCount() < FoundationSeatCount
}

// CalculateSharesForDeposit calculates shares for a given deposit amount
func (p *Pool) CalculateSharesForDeposit(amount math.LegacyDec) math.LegacyDec {
	if p.NAV.IsZero() || p.NAV.IsNegative() {
		return amount // 1:1 if NAV is invalid
	}
	return amount.Quo(p.NAV)
}

// CalculateValueForShares calculates value for a given number of shares
func (p *Pool) CalculateValueForShares(shares math.LegacyDec) math.LegacyDec {
	return shares.Mul(p.NAV)
}

// UpdateNAV updates the pool NAV based on total value
func (p *Pool) UpdateNAV(totalValue math.LegacyDec) {
	if p.TotalShares.IsZero() {
		p.NAV = math.LegacyOneDec()
		return
	}
	p.NAV = totalValue.Quo(p.TotalShares)
	p.UpdatedAt = time.Now().Unix()

	// Update high water mark and drawdown
	if p.NAV.GT(p.HighWaterMark) {
		p.HighWaterMark = p.NAV
		p.CurrentDrawdown = math.LegacyZeroDec()
	} else if p.HighWaterMark.IsPositive() {
		p.CurrentDrawdown = p.HighWaterMark.Sub(p.NAV).Quo(p.HighWaterMark)
	}

	// Update DDGuard level
	p.updateDDGuardLevel()
}

// updateDDGuardLevel updates the DDGuard level based on current drawdown
func (p *Pool) updateDDGuardLevel() {
	if p.CurrentDrawdown.GTE(DDGuardHaltThreshold) {
		p.DDGuardLevel = DDGuardLevelHalt
		p.Status = PoolStatusPaused
	} else if p.CurrentDrawdown.GTE(DDGuardReduceThreshold) {
		p.DDGuardLevel = DDGuardLevelReduce
	} else if p.CurrentDrawdown.GTE(DDGuardWarningThreshold) {
		p.DDGuardLevel = DDGuardLevelWarning
	} else {
		p.DDGuardLevel = DDGuardLevelNormal
	}
}

// Deposit represents a user's deposit in a pool
type Deposit struct {
	DepositID    string         `json:"deposit_id"`
	PoolID       string         `json:"pool_id"`
	Depositor    string         `json:"depositor"`
	Amount       math.LegacyDec `json:"amount"`
	Shares       math.LegacyDec `json:"shares"`
	NAVAtDeposit math.LegacyDec `json:"nav_at_deposit"`
	DepositedAt  int64          `json:"deposited_at"`
	UnlockAt     int64          `json:"unlock_at"` // 0 if no lock
	PointsEarned math.LegacyDec `json:"points_earned,omitempty"`
}

// NewDeposit creates a new deposit record
func NewDeposit(poolID, depositor string, amount, shares, nav math.LegacyDec, lockDays int64) *Deposit {
	now := time.Now().Unix()
	unlockAt := int64(0)
	if lockDays > 0 {
		unlockAt = now + lockDays*24*60*60
	}

	return &Deposit{
		DepositID:    generateID("dep"),
		PoolID:       poolID,
		Depositor:    depositor,
		Amount:       amount,
		Shares:       shares,
		NAVAtDeposit: nav,
		DepositedAt:  now,
		UnlockAt:     unlockAt,
		PointsEarned: math.LegacyZeroDec(),
	}
}

// IsLocked checks if the deposit is still locked
func (d *Deposit) IsLocked() bool {
	if d.UnlockAt == 0 {
		return false
	}
	return time.Now().Unix() < d.UnlockAt
}

// Withdrawal represents a withdrawal request
type Withdrawal struct {
	WithdrawalID    string         `json:"withdrawal_id"`
	PoolID          string         `json:"pool_id"`
	Withdrawer      string         `json:"withdrawer"`
	SharesRequested math.LegacyDec `json:"shares_requested"`
	SharesRedeemed  math.LegacyDec `json:"shares_redeemed"`
	AmountReceived  math.LegacyDec `json:"amount_received"`
	NAVAtRequest    math.LegacyDec `json:"nav_at_request"`
	Status          string         `json:"status"`
	RequestedAt     int64          `json:"requested_at"`
	AvailableAt     int64          `json:"available_at"` // T+N timestamp
	CompletedAt     int64          `json:"completed_at"`
}

// NewWithdrawal creates a new withdrawal request
func NewWithdrawal(poolID, withdrawer string, shares, nav math.LegacyDec, delayDays int64) *Withdrawal {
	now := time.Now().Unix()
	availableAt := now + delayDays*24*60*60

	return &Withdrawal{
		WithdrawalID:    generateID("wth"),
		PoolID:          poolID,
		Withdrawer:      withdrawer,
		SharesRequested: shares,
		SharesRedeemed:  math.LegacyZeroDec(),
		AmountReceived:  math.LegacyZeroDec(),
		NAVAtRequest:    nav,
		Status:          WithdrawalStatusPending,
		RequestedAt:     now,
		AvailableAt:     availableAt,
		CompletedAt:     0,
	}
}

// IsReady checks if the withdrawal is ready to be claimed
func (w *Withdrawal) IsReady() bool {
	return time.Now().Unix() >= w.AvailableAt
}

// DDGuardState tracks the drawdown guard state for a pool
type DDGuardState struct {
	PoolID           string         `json:"pool_id"`
	Level            string         `json:"level"`
	PeakNAV          math.LegacyDec `json:"peak_nav"`
	CurrentNAV       math.LegacyDec `json:"current_nav"`
	DrawdownPercent  math.LegacyDec `json:"drawdown_percent"`
	MaxExposureLimit math.LegacyDec `json:"max_exposure_limit"`
	TriggeredAt      int64          `json:"triggered_at"`
	LastCheckedAt    int64          `json:"last_checked_at"`
}

// PoolStats aggregates pool statistics
type PoolStats struct {
	PoolID                  string         `json:"pool_id"`
	TotalValueLocked        math.LegacyDec `json:"total_value_locked"`
	TotalDepositors         int64          `json:"total_depositors"`
	TotalPendingWithdrawals math.LegacyDec `json:"total_pending_withdrawals"`
	RealizedPnL             math.LegacyDec `json:"realized_pnl"`
	UnrealizedPnL           math.LegacyDec `json:"unrealized_pnl"`
	TotalFeesCollected      math.LegacyDec `json:"total_fees_collected"`
	Return1d                math.LegacyDec `json:"return_1d"`
	Return7d                math.LegacyDec `json:"return_7d"`
	Return30d               math.LegacyDec `json:"return_30d"`
	ReturnAllTime           math.LegacyDec `json:"return_all_time"`
	UpdatedAt               int64          `json:"updated_at"`
}

// NewPoolStats creates a new pool stats record
func NewPoolStats(poolID string) *PoolStats {
	return &PoolStats{
		PoolID:                  poolID,
		TotalValueLocked:        math.LegacyZeroDec(),
		TotalDepositors:         0,
		TotalPendingWithdrawals: math.LegacyZeroDec(),
		RealizedPnL:             math.LegacyZeroDec(),
		UnrealizedPnL:           math.LegacyZeroDec(),
		TotalFeesCollected:      math.LegacyZeroDec(),
		Return1d:                math.LegacyZeroDec(),
		Return7d:                math.LegacyZeroDec(),
		Return30d:               math.LegacyZeroDec(),
		ReturnAllTime:           math.LegacyZeroDec(),
		UpdatedAt:               time.Now().Unix(),
	}
}

// NAVHistory stores historical NAV data points
type NAVHistory struct {
	PoolID     string         `json:"pool_id"`
	NAV        math.LegacyDec `json:"nav"`
	TotalValue math.LegacyDec `json:"total_value"`
	Timestamp  int64          `json:"timestamp"`
}

// RevenueRecord tracks revenue sources for a pool
type RevenueRecord struct {
	RecordID    string         `json:"record_id"`
	PoolID      string         `json:"pool_id"`
	Source      string         `json:"source"` // "spread", "funding", "liquidation"
	Amount      math.LegacyDec `json:"amount"`
	Description string         `json:"description"`
	Timestamp   int64          `json:"timestamp"`
}

// generateID generates a unique ID with a prefix
func generateID(prefix string) string {
	b := make([]byte, 8)
	rand.Read(b)
	return prefix + "-" + hex.EncodeToString(b)
}

// generateInviteCode generates a random invite code
func GenerateInviteCode() string {
	b := make([]byte, 4)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// ============================================================
// Community Pool Types
// ============================================================

// InviteCode represents an invite code for private pools
type InviteCode struct {
	Code       string `json:"code"`
	PoolID     string `json:"pool_id"`
	MaxUses    int64  `json:"max_uses"`    // 0 = unlimited
	UsedCount  int64  `json:"used_count"`
	CreatedBy  string `json:"created_by"`
	CreatedAt  int64  `json:"created_at"`
	ExpiresAt  int64  `json:"expires_at"`  // 0 = never expires
	IsActive   bool   `json:"is_active"`
}

// NewInviteCode creates a new invite code
func NewInviteCode(poolID, createdBy string, maxUses int64, expiresIn int64) *InviteCode {
	now := time.Now().Unix()
	expiresAt := int64(0)
	if expiresIn > 0 {
		expiresAt = now + expiresIn
	}

	return &InviteCode{
		Code:      GenerateInviteCode(),
		PoolID:    poolID,
		MaxUses:   maxUses,
		UsedCount: 0,
		CreatedBy: createdBy,
		CreatedAt: now,
		ExpiresAt: expiresAt,
		IsActive:  true,
	}
}

// IsValid checks if the invite code is valid for use
func (ic *InviteCode) IsValid() bool {
	if !ic.IsActive {
		return false
	}
	if ic.ExpiresAt > 0 && time.Now().Unix() > ic.ExpiresAt {
		return false
	}
	if ic.MaxUses > 0 && ic.UsedCount >= ic.MaxUses {
		return false
	}
	return true
}

// PoolHolder represents a depositor in a pool
type PoolHolder struct {
	PoolID      string         `json:"pool_id"`
	Address     string         `json:"address"`
	Shares      math.LegacyDec `json:"shares"`
	Value       math.LegacyDec `json:"value"`        // Current value (shares * NAV)
	DepositedAt int64          `json:"deposited_at"` // First deposit timestamp
	IsOwner     bool           `json:"is_owner"`
}

// PoolPosition represents a trading position opened by pool owner
type PoolPosition struct {
	PositionID    string         `json:"position_id"`
	PoolID        string         `json:"pool_id"`
	MarketID      string         `json:"market_id"`
	Side          string         `json:"side"` // "long" or "short"
	Size          math.LegacyDec `json:"size"`
	EntryPrice    math.LegacyDec `json:"entry_price"`
	CurrentPrice  math.LegacyDec `json:"current_price"`
	Leverage      math.LegacyDec `json:"leverage"`
	Margin        math.LegacyDec `json:"margin"`
	UnrealizedPnL math.LegacyDec `json:"unrealized_pnl"`
	RealizedPnL   math.LegacyDec `json:"realized_pnl"`
	LiqPrice      math.LegacyDec `json:"liq_price"`
	OpenedAt      int64          `json:"opened_at"`
	UpdatedAt     int64          `json:"updated_at"`
}

// NewPoolPosition creates a new pool position
func NewPoolPosition(poolID, marketID, side string, size, entryPrice, leverage, margin math.LegacyDec) *PoolPosition {
	now := time.Now().Unix()
	return &PoolPosition{
		PositionID:    generateID("pos"),
		PoolID:        poolID,
		MarketID:      marketID,
		Side:          side,
		Size:          size,
		EntryPrice:    entryPrice,
		CurrentPrice:  entryPrice,
		Leverage:      leverage,
		Margin:        margin,
		UnrealizedPnL: math.LegacyZeroDec(),
		RealizedPnL:   math.LegacyZeroDec(),
		LiqPrice:      math.LegacyZeroDec(), // Will be calculated
		OpenedAt:      now,
		UpdatedAt:     now,
	}
}

// PoolTrade represents a trade executed by pool owner
type PoolTrade struct {
	TradeID    string         `json:"trade_id"`
	PoolID     string         `json:"pool_id"`
	MarketID   string         `json:"market_id"`
	Side       string         `json:"side"` // "buy" or "sell"
	Size       math.LegacyDec `json:"size"`
	Price      math.LegacyDec `json:"price"`
	Fee        math.LegacyDec `json:"fee"`
	PnL        math.LegacyDec `json:"pnl"` // For close trades
	PositionID string         `json:"position_id,omitempty"`
	ExecutedAt int64          `json:"executed_at"`
}

// NewPoolTrade creates a new pool trade record
func NewPoolTrade(poolID, marketID, side string, size, price, fee math.LegacyDec) *PoolTrade {
	return &PoolTrade{
		TradeID:    generateID("trd"),
		PoolID:     poolID,
		MarketID:   marketID,
		Side:       side,
		Size:       size,
		Price:      price,
		Fee:        fee,
		PnL:        math.LegacyZeroDec(),
		ExecutedAt: time.Now().Unix(),
	}
}

// CommunityPoolConfig represents the configuration for creating a community pool
type CommunityPoolConfig struct {
	Name                 string         `json:"name"`
	Description          string         `json:"description"`
	Owner                string         `json:"owner"`
	MinDeposit           math.LegacyDec `json:"min_deposit"`
	MaxDeposit           math.LegacyDec `json:"max_deposit"`
	ManagementFee        math.LegacyDec `json:"management_fee"`  // Annual %
	PerformanceFee       math.LegacyDec `json:"performance_fee"` // % of profits
	OwnerMinStake        math.LegacyDec `json:"owner_min_stake"` // Min % owner must stake
	LockPeriodDays       int64          `json:"lock_period_days"`
	RedemptionDelayDays  int64          `json:"redemption_delay_days"`
	DailyRedemptionLimit math.LegacyDec `json:"daily_redemption_limit"`
	IsPrivate            bool           `json:"is_private"`
	MaxLeverage          math.LegacyDec `json:"max_leverage"`
	AllowedMarkets       []string       `json:"allowed_markets"`
	Tags                 []string       `json:"tags"`
}

// Validate validates the community pool configuration
func (c *CommunityPoolConfig) Validate() error {
	if len(c.Name) == 0 || len(c.Name) > 50 {
		return ErrInvalidPoolName
	}
	if len(c.Owner) == 0 {
		return ErrInvalidOwner
	}
	// Check MinDeposit only if initialized
	if !c.MinDeposit.IsNil() && c.MinDeposit.IsNegative() {
		return ErrInvalidMinDeposit
	}
	// Check OwnerMinStake only if initialized
	if !c.OwnerMinStake.IsNil() && c.OwnerMinStake.LT(math.LegacyMustNewDecFromStr("0.05")) {
		return ErrInvalidOwnerStake
	}
	// Check ManagementFee only if initialized
	if !c.ManagementFee.IsNil() && c.ManagementFee.GT(math.LegacyMustNewDecFromStr("0.05")) {
		return ErrInvalidManagementFee
	}
	// Check PerformanceFee only if initialized
	if !c.PerformanceFee.IsNil() && c.PerformanceFee.GT(math.LegacyMustNewDecFromStr("0.50")) {
		return ErrInvalidPerformanceFee
	}
	return nil
}

// NewCommunityPool creates a new community pool from config
func NewCommunityPool(config *CommunityPoolConfig) (*Pool, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	now := time.Now().Unix()
	poolID := generateID("cpool")

	pool := &Pool{
		PoolID:               poolID,
		PoolType:             PoolTypeCommunity,
		Name:                 config.Name,
		Description:          config.Description,
		Status:               PoolStatusActive,
		TotalDeposits:        math.LegacyZeroDec(),
		TotalShares:          math.LegacyZeroDec(),
		NAV:                  math.LegacyOneDec(),
		HighWaterMark:        math.LegacyOneDec(),
		CurrentDrawdown:      math.LegacyZeroDec(),
		DDGuardLevel:         DDGuardLevelNormal,
		MinDeposit:           config.MinDeposit,
		MaxDeposit:           config.MaxDeposit,
		LockPeriodDays:       config.LockPeriodDays,
		RedemptionDelayDays:  config.RedemptionDelayDays,
		DailyRedemptionLimit: config.DailyRedemptionLimit,
		ManagementFee:        config.ManagementFee,
		PerformanceFee:       config.PerformanceFee,
		Owner:                config.Owner,
		OwnerMinStake:        config.OwnerMinStake,
		OwnerCurrentStake:    math.LegacyZeroDec(),
		IsPrivate:            config.IsPrivate,
		TotalHolders:         0,
		AllowedMarkets:       config.AllowedMarkets,
		MaxLeverage:          config.MaxLeverage,
		Tags:                 config.Tags,
		CreatedAt:            now,
		UpdatedAt:            now,
	}

	// Generate invite code for private pools
	if config.IsPrivate {
		pool.InviteCode = GenerateInviteCode()
	}

	return pool, nil
}
