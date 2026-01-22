package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"cosmossdk.io/math"
	"github.com/gorilla/mux"
	"github.com/openalpha/perp-dex/x/riverpool/keeper"
	"github.com/openalpha/perp-dex/x/riverpool/types"
)

// RiverpoolHandler handles riverpool API requests
type RiverpoolHandler struct {
	keeper      *keeper.Keeper
	queryServer *keeper.QueryServer
	msgServer   *keeper.MsgServer
}

// NewRiverpoolHandler creates a new RiverpoolHandler
func NewRiverpoolHandler(k *keeper.Keeper) *RiverpoolHandler {
	return &RiverpoolHandler{
		keeper:      k,
		queryServer: keeper.NewQueryServerImpl(k),
		msgServer:   keeper.NewMsgServerImpl(k),
	}
}

// RegisterRoutes registers riverpool API routes
func (h *RiverpoolHandler) RegisterRoutes(r *mux.Router) {
	// Pool routes
	r.HandleFunc("/v1/riverpool/pools", h.GetPools).Methods("GET")
	r.HandleFunc("/v1/riverpool/pools/{poolId}", h.GetPool).Methods("GET")
	r.HandleFunc("/v1/riverpool/pools/type/{poolType}", h.GetPoolsByType).Methods("GET")

	// Pool statistics
	r.HandleFunc("/v1/riverpool/pools/{poolId}/stats", h.GetPoolStats).Methods("GET")
	r.HandleFunc("/v1/riverpool/pools/{poolId}/nav/history", h.GetNAVHistory).Methods("GET")
	r.HandleFunc("/v1/riverpool/pools/{poolId}/ddguard", h.GetDDGuardState).Methods("GET")

	// User routes
	r.HandleFunc("/v1/riverpool/user/{user}/deposits", h.GetUserDeposits).Methods("GET")
	r.HandleFunc("/v1/riverpool/user/{user}/withdrawals", h.GetUserWithdrawals).Methods("GET")
	r.HandleFunc("/v1/riverpool/pools/{poolId}/user/{user}/balance", h.GetUserPoolBalance).Methods("GET")

	// Pool deposits and withdrawals
	r.HandleFunc("/v1/riverpool/pools/{poolId}/deposits", h.GetPoolDeposits).Methods("GET")
	r.HandleFunc("/v1/riverpool/pools/{poolId}/withdrawals/pending", h.GetPendingWithdrawals).Methods("GET")

	// Estimation routes
	r.HandleFunc("/v1/riverpool/pools/{poolId}/estimate/deposit", h.EstimateDeposit).Methods("GET")
	r.HandleFunc("/v1/riverpool/pools/{poolId}/estimate/withdrawal", h.EstimateWithdrawal).Methods("GET")

	// Transaction routes
	r.HandleFunc("/v1/riverpool/deposit", h.Deposit).Methods("POST")
	r.HandleFunc("/v1/riverpool/withdrawal/request", h.RequestWithdrawal).Methods("POST")
	r.HandleFunc("/v1/riverpool/withdrawal/claim", h.ClaimWithdrawal).Methods("POST")
	r.HandleFunc("/v1/riverpool/withdrawal/cancel", h.CancelWithdrawal).Methods("POST")

	// Revenue routes
	r.HandleFunc("/v1/riverpool/pools/{poolId}/revenue", h.GetPoolRevenue).Methods("GET")
	r.HandleFunc("/v1/riverpool/pools/{poolId}/revenue/records", h.GetRevenueRecords).Methods("GET")
	r.HandleFunc("/v1/riverpool/pools/{poolId}/revenue/breakdown", h.GetRevenueBreakdown).Methods("GET")

	// Community Pool routes
	r.HandleFunc("/v1/riverpool/community/create", h.CreateCommunityPool).Methods("POST")
	r.HandleFunc("/v1/riverpool/community/{poolId}/holders", h.GetPoolHolders).Methods("GET")
	r.HandleFunc("/v1/riverpool/community/{poolId}/positions", h.GetPoolPositions).Methods("GET")
	r.HandleFunc("/v1/riverpool/community/{poolId}/trades", h.GetPoolTrades).Methods("GET")
	r.HandleFunc("/v1/riverpool/community/{poolId}/stake", h.DepositOwnerStake).Methods("POST")
	r.HandleFunc("/v1/riverpool/community/{poolId}/invites", h.GetInviteCodes).Methods("GET")
	r.HandleFunc("/v1/riverpool/community/{poolId}/invites", h.GenerateInviteCode).Methods("POST")
	r.HandleFunc("/v1/riverpool/community/{poolId}/pause", h.PausePool).Methods("POST")
	r.HandleFunc("/v1/riverpool/community/{poolId}/resume", h.ResumePool).Methods("POST")
	r.HandleFunc("/v1/riverpool/community/{poolId}/close", h.ClosePool).Methods("POST")
	r.HandleFunc("/v1/riverpool/user/{user}/owned-pools", h.GetUserOwnedPools).Methods("GET")
}

// PoolResponse represents a pool in API responses
type PoolResponse struct {
	PoolID           string `json:"pool_id"`
	PoolType         string `json:"pool_type"`
	Name             string `json:"name"`
	Description      string `json:"description"`
	Status           string `json:"status"`
	TotalDeposits    string `json:"total_deposits"`
	TotalShares      string `json:"total_shares"`
	NAV              string `json:"nav"`
	HighWaterMark    string `json:"high_water_mark"`
	CurrentDrawdown  string `json:"current_drawdown"`
	DDGuardLevel     string `json:"dd_guard_level"`
	MinDeposit       string `json:"min_deposit"`
	MaxDeposit       string `json:"max_deposit"`
	LockPeriodDays   int64  `json:"lock_period_days"`
	RedemptionDelay  int64  `json:"redemption_delay_days"`
	DailyRedemptionLimit string `json:"daily_redemption_limit"`
	SeatsAvailable   int64  `json:"seats_available,omitempty"` // Foundation LP only
	CreatedAt        int64  `json:"created_at"`
	UpdatedAt        int64  `json:"updated_at"`
	// Community Pool specific fields
	Owner             string   `json:"owner,omitempty"`
	ManagementFee     string   `json:"management_fee,omitempty"`
	PerformanceFee    string   `json:"performance_fee,omitempty"`
	OwnerMinStake     string   `json:"owner_min_stake,omitempty"`
	OwnerCurrentStake string   `json:"owner_current_stake,omitempty"`
	IsPrivate         bool     `json:"is_private,omitempty"`
	RequiresInviteCode bool    `json:"requires_invite_code,omitempty"`
	TotalHolders      int64    `json:"total_holders,omitempty"`
	AllowedMarkets    []string `json:"allowed_markets,omitempty"`
	MaxLeverage       string   `json:"max_leverage,omitempty"`
	Tags              []string `json:"tags,omitempty"`
}

// poolToResponse converts a Pool to PoolResponse
func poolToResponse(pool *types.Pool) PoolResponse {
	resp := PoolResponse{
		PoolID:           pool.PoolID,
		PoolType:         pool.PoolType,
		Name:             pool.Name,
		Description:      pool.Description,
		Status:           pool.Status,
		TotalDeposits:    pool.TotalDeposits.String(),
		TotalShares:      pool.TotalShares.String(),
		NAV:              pool.NAV.String(),
		HighWaterMark:    pool.HighWaterMark.String(),
		CurrentDrawdown:  pool.CurrentDrawdown.String(),
		DDGuardLevel:     pool.DDGuardLevel,
		MinDeposit:       pool.MinDeposit.String(),
		MaxDeposit:       pool.MaxDeposit.String(),
		LockPeriodDays:   pool.LockPeriodDays,
		RedemptionDelay:  pool.RedemptionDelayDays,
		DailyRedemptionLimit: pool.DailyRedemptionLimit.String(),
		CreatedAt:        pool.CreatedAt,
		UpdatedAt:        pool.UpdatedAt,
	}

	// Add seats info for Foundation LP
	if pool.PoolType == types.PoolTypeFoundation {
		resp.SeatsAvailable = types.FoundationSeatCount - pool.GetSeatCount()
	}

	// Add Community Pool specific fields
	if pool.PoolType == types.PoolTypeCommunity {
		resp.Owner = pool.Owner
		resp.ManagementFee = pool.ManagementFee.String()
		resp.PerformanceFee = pool.PerformanceFee.String()
		resp.OwnerMinStake = pool.OwnerMinStake.String()
		resp.OwnerCurrentStake = pool.OwnerCurrentStake.String()
		resp.IsPrivate = pool.IsPrivate
		resp.RequiresInviteCode = pool.IsPrivate
		resp.TotalHolders = pool.TotalHolders
		resp.AllowedMarkets = pool.AllowedMarkets
		resp.MaxLeverage = pool.MaxLeverage.String()
		resp.Tags = pool.Tags
	}

	return resp
}

// GetPools returns all pools
func (h *RiverpoolHandler) GetPools(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	offset, _ := strconv.ParseUint(r.URL.Query().Get("offset"), 10, 64)
	limit, _ := strconv.ParseUint(r.URL.Query().Get("limit"), 10, 64)
	if limit == 0 {
		limit = 20
	}

	pools, total, err := h.queryServer.Pools(ctx, offset, limit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response := make([]PoolResponse, len(pools))
	for i, pool := range pools {
		response[i] = poolToResponse(pool)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"pools": response,
		"total": total,
	})
}

// GetPool returns a single pool
func (h *RiverpoolHandler) GetPool(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	poolID := vars["poolId"]

	pool, err := h.queryServer.Pool(ctx, poolID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(poolToResponse(pool))
}

// GetPoolsByType returns pools filtered by type
func (h *RiverpoolHandler) GetPoolsByType(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	poolType := vars["poolType"]

	pools, err := h.queryServer.PoolsByType(ctx, poolType)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response := make([]PoolResponse, len(pools))
	for i, pool := range pools {
		response[i] = poolToResponse(pool)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"pools": response,
	})
}

// GetPoolStats returns pool statistics
func (h *RiverpoolHandler) GetPoolStats(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	poolID := vars["poolId"]

	stats, err := h.queryServer.PoolStats(ctx, poolID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"pool_id":                  stats.PoolID,
		"total_value_locked":       stats.TotalValueLocked.String(),
		"total_depositors":         stats.TotalDepositors,
		"total_pending_withdrawals": stats.TotalPendingWithdrawals.String(),
		"realized_pnl":             stats.RealizedPnL.String(),
		"unrealized_pnl":           stats.UnrealizedPnL.String(),
		"total_fees_collected":     stats.TotalFeesCollected.String(),
		"return_1d":                stats.Return1d.String(),
		"return_7d":                stats.Return7d.String(),
		"return_30d":               stats.Return30d.String(),
		"return_all_time":          stats.ReturnAllTime.String(),
		"updated_at":               stats.UpdatedAt,
	})
}

// GetNAVHistory returns NAV history for a pool
func (h *RiverpoolHandler) GetNAVHistory(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	poolID := vars["poolId"]

	fromTime, _ := strconv.ParseInt(r.URL.Query().Get("from"), 10, 64)
	toTime, _ := strconv.ParseInt(r.URL.Query().Get("to"), 10, 64)

	history, err := h.queryServer.NAVHistory(ctx, poolID, fromTime, toTime)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response := make([]map[string]interface{}, len(history))
	for i, h := range history {
		response[i] = map[string]interface{}{
			"pool_id":     h.PoolID,
			"nav":         h.NAV.String(),
			"total_value": h.TotalValue.String(),
			"timestamp":   h.Timestamp,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"history": response,
	})
}

// GetDDGuardState returns DDGuard state for a pool
func (h *RiverpoolHandler) GetDDGuardState(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	poolID := vars["poolId"]

	state, err := h.queryServer.DDGuardState(ctx, poolID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"pool_id":            state.PoolID,
		"level":              state.Level,
		"peak_nav":           state.PeakNAV.String(),
		"current_nav":        state.CurrentNAV.String(),
		"drawdown_percent":   state.DrawdownPercent.String(),
		"max_exposure_limit": state.MaxExposureLimit.String(),
		"triggered_at":       state.TriggeredAt,
		"last_checked_at":    state.LastCheckedAt,
	})
}

// GetUserDeposits returns all deposits for a user
func (h *RiverpoolHandler) GetUserDeposits(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	user := vars["user"]

	deposits, totalValue, err := h.queryServer.UserDeposits(ctx, user)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response := make([]map[string]interface{}, len(deposits))
	for i, d := range deposits {
		response[i] = map[string]interface{}{
			"deposit_id":     d.DepositID,
			"pool_id":        d.PoolID,
			"depositor":      d.Depositor,
			"amount":         d.Amount.String(),
			"shares":         d.Shares.String(),
			"nav_at_deposit": d.NAVAtDeposit.String(),
			"deposited_at":   d.DepositedAt,
			"unlock_at":      d.UnlockAt,
			"points_earned":  d.PointsEarned.String(),
			"is_locked":      d.IsLocked(),
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"deposits":    response,
		"total_value": totalValue.String(),
	})
}

// GetUserWithdrawals returns all withdrawals for a user
func (h *RiverpoolHandler) GetUserWithdrawals(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	user := vars["user"]

	withdrawals, err := h.queryServer.UserWithdrawals(ctx, user)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response := make([]map[string]interface{}, len(withdrawals))
	for i, w := range withdrawals {
		response[i] = map[string]interface{}{
			"withdrawal_id":    w.WithdrawalID,
			"pool_id":          w.PoolID,
			"withdrawer":       w.Withdrawer,
			"shares_requested": w.SharesRequested.String(),
			"shares_redeemed":  w.SharesRedeemed.String(),
			"amount_received":  w.AmountReceived.String(),
			"nav_at_request":   w.NAVAtRequest.String(),
			"status":           w.Status,
			"requested_at":     w.RequestedAt,
			"available_at":     w.AvailableAt,
			"completed_at":     w.CompletedAt,
			"is_ready":         w.IsReady(),
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"withdrawals": response,
	})
}

// GetUserPoolBalance returns user's balance in a pool
func (h *RiverpoolHandler) GetUserPoolBalance(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	poolID := vars["poolId"]
	user := vars["user"]

	shares, value, costBasis, unrealizedPnL, pnlPercent, unlockAt, canWithdraw, err := h.queryServer.UserPoolBalance(ctx, poolID, user)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"shares":         shares.String(),
		"value":          value.String(),
		"cost_basis":     costBasis.String(),
		"unrealized_pnl": unrealizedPnL.String(),
		"pnl_percent":    pnlPercent.String(),
		"unlock_at":      unlockAt,
		"can_withdraw":   canWithdraw,
	})
}

// GetPoolDeposits returns all deposits in a pool
func (h *RiverpoolHandler) GetPoolDeposits(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	poolID := vars["poolId"]

	offset, _ := strconv.ParseUint(r.URL.Query().Get("offset"), 10, 64)
	limit, _ := strconv.ParseUint(r.URL.Query().Get("limit"), 10, 64)
	if limit == 0 {
		limit = 20
	}

	deposits, total, err := h.queryServer.PoolDeposits(ctx, poolID, offset, limit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response := make([]map[string]interface{}, len(deposits))
	for i, d := range deposits {
		response[i] = map[string]interface{}{
			"deposit_id":     d.DepositID,
			"pool_id":        d.PoolID,
			"depositor":      d.Depositor,
			"amount":         d.Amount.String(),
			"shares":         d.Shares.String(),
			"nav_at_deposit": d.NAVAtDeposit.String(),
			"deposited_at":   d.DepositedAt,
			"unlock_at":      d.UnlockAt,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"deposits": response,
		"total":    total,
	})
}

// GetPendingWithdrawals returns pending withdrawals for a pool
func (h *RiverpoolHandler) GetPendingWithdrawals(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	poolID := vars["poolId"]

	withdrawals, totalPendingShares, totalPendingValue, dailyLimitRemaining, err := h.queryServer.PendingWithdrawals(ctx, poolID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response := make([]map[string]interface{}, len(withdrawals))
	for i, w := range withdrawals {
		response[i] = map[string]interface{}{
			"withdrawal_id":    w.WithdrawalID,
			"pool_id":          w.PoolID,
			"withdrawer":       w.Withdrawer,
			"shares_requested": w.SharesRequested.String(),
			"shares_redeemed":  w.SharesRedeemed.String(),
			"nav_at_request":   w.NAVAtRequest.String(),
			"status":           w.Status,
			"requested_at":     w.RequestedAt,
			"available_at":     w.AvailableAt,
			"is_ready":         w.IsReady(),
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"withdrawals":             response,
		"total_pending_shares":    totalPendingShares.String(),
		"total_pending_value":     totalPendingValue.String(),
		"daily_limit_remaining":   dailyLimitRemaining.String(),
	})
}

// EstimateDeposit estimates shares for a deposit
func (h *RiverpoolHandler) EstimateDeposit(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	poolID := vars["poolId"]
	amountStr := r.URL.Query().Get("amount")

	amount, err := math.LegacyNewDecFromStr(amountStr)
	if err != nil {
		http.Error(w, "Invalid amount", http.StatusBadRequest)
		return
	}

	shares, nav, sharePrice, err := h.queryServer.EstimateDeposit(ctx, poolID, amount)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"shares":      shares.String(),
		"nav":         nav.String(),
		"share_price": sharePrice.String(),
	})
}

// EstimateWithdrawal estimates amount for a withdrawal
func (h *RiverpoolHandler) EstimateWithdrawal(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	poolID := vars["poolId"]
	sharesStr := r.URL.Query().Get("shares")

	shares, err := math.LegacyNewDecFromStr(sharesStr)
	if err != nil {
		http.Error(w, "Invalid shares", http.StatusBadRequest)
		return
	}

	amount, nav, availableAt, queuePosition, mayBeProrated, err := h.queryServer.EstimateWithdrawal(ctx, poolID, shares)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"amount":         amount.String(),
		"nav":            nav.String(),
		"available_at":   availableAt,
		"queue_position": queuePosition,
		"may_be_prorated": mayBeProrated,
	})
}

// DepositRequest represents a deposit request
type DepositRequest struct {
	Depositor  string `json:"depositor"`
	PoolID     string `json:"pool_id"`
	Amount     string `json:"amount"`
	InviteCode string `json:"invite_code,omitempty"`
}

// Deposit handles deposit requests
func (h *RiverpoolHandler) Deposit(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req DepositRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	msg := &types.MsgDeposit{
		Depositor:  req.Depositor,
		PoolID:     req.PoolID,
		Amount:     req.Amount,
		InviteCode: req.InviteCode,
	}

	resp, err := h.msgServer.Deposit(ctx, msg)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// WithdrawalRequest represents a withdrawal request
type WithdrawalRequest struct {
	Withdrawer string `json:"withdrawer"`
	PoolID     string `json:"pool_id"`
	Shares     string `json:"shares"`
}

// RequestWithdrawal handles withdrawal requests
func (h *RiverpoolHandler) RequestWithdrawal(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req WithdrawalRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	msg := &types.MsgRequestWithdrawal{
		Withdrawer: req.Withdrawer,
		PoolID:     req.PoolID,
		Shares:     req.Shares,
	}

	resp, err := h.msgServer.RequestWithdrawal(ctx, msg)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// ClaimWithdrawalRequest represents a claim withdrawal request
type ClaimWithdrawalRequest struct {
	Withdrawer   string `json:"withdrawer"`
	WithdrawalID string `json:"withdrawal_id"`
}

// ClaimWithdrawal handles claim withdrawal requests
func (h *RiverpoolHandler) ClaimWithdrawal(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req ClaimWithdrawalRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	msg := &types.MsgClaimWithdrawal{
		Withdrawer:   req.Withdrawer,
		WithdrawalID: req.WithdrawalID,
	}

	resp, err := h.msgServer.ClaimWithdrawal(ctx, msg)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// CancelWithdrawalRequest represents a cancel withdrawal request
type CancelWithdrawalRequest struct {
	Withdrawer   string `json:"withdrawer"`
	WithdrawalID string `json:"withdrawal_id"`
}

// CancelWithdrawal handles cancel withdrawal requests
func (h *RiverpoolHandler) CancelWithdrawal(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req CancelWithdrawalRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	msg := &types.MsgCancelWithdrawal{
		Withdrawer:   req.Withdrawer,
		WithdrawalID: req.WithdrawalID,
	}

	resp, err := h.msgServer.CancelWithdrawal(ctx, msg)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// RevenueStatsResponse represents pool revenue statistics
type RevenueStatsResponse struct {
	PoolID            string `json:"pool_id"`
	TotalRevenue      string `json:"total_revenue"`
	SpreadRevenue     string `json:"spread_revenue"`
	FundingRevenue    string `json:"funding_revenue"`
	LiquidationProfit string `json:"liquidation_profit"`
	TradingPnL        string `json:"trading_pnl"`
	FeeRebates        string `json:"fee_rebates"`
	Return1D          string `json:"return_1d"`
	Return7D          string `json:"return_7d"`
	Return30D         string `json:"return_30d"`
	LastUpdated       int64  `json:"last_updated"`
}

// RevenueRecordResponse represents a single revenue record
type RevenueRecordResponse struct {
	RecordID    string `json:"record_id"`
	PoolID      string `json:"pool_id"`
	Source      string `json:"source"`
	Amount      string `json:"amount"`
	NAVImpact   string `json:"nav_impact"`
	Timestamp   int64  `json:"timestamp"`
	BlockHeight int64  `json:"block_height"`
	MarketID    string `json:"market_id,omitempty"`
	Details     string `json:"details,omitempty"`
}

// RevenueBreakdownResponse represents revenue breakdown by source
type RevenueBreakdownResponse struct {
	PoolID      string            `json:"pool_id"`
	Period      string            `json:"period"`
	TotalAmount string            `json:"total_amount"`
	Breakdown   map[string]string `json:"breakdown"`
}

// GetPoolRevenue handles GET /v1/riverpool/pools/{poolId}/revenue
func (h *RiverpoolHandler) GetPoolRevenue(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	poolID := vars["poolId"]

	// In standalone API mode, return default values
	// Keeper methods require sdk.Context which is not available in HTTP context
	resp := RevenueStatsResponse{
		PoolID:            poolID,
		TotalRevenue:      "0",
		SpreadRevenue:     "0",
		FundingRevenue:    "0",
		LiquidationProfit: "0",
		TradingPnL:        "0",
		FeeRebates:        "0",
		Return1D:          "0",
		Return7D:          "0",
		Return30D:         "0",
		LastUpdated:       0,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// GetRevenueRecords handles GET /v1/riverpool/pools/{poolId}/revenue/records
func (h *RiverpoolHandler) GetRevenueRecords(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	poolID := vars["poolId"]

	// In standalone API mode, return empty records
	// Keeper methods require sdk.Context which is not available in HTTP context
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"pool_id": poolID,
		"records": []RevenueRecordResponse{},
		"total":   0,
	})
}

// GetRevenueBreakdown handles GET /v1/riverpool/pools/{poolId}/revenue/breakdown
func (h *RiverpoolHandler) GetRevenueBreakdown(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	poolID := vars["poolId"]

	// Parse period param
	periodStr := r.URL.Query().Get("period")
	period := "7d"

	switch periodStr {
	case "1d", "24h":
		period = "1d"
	case "7d":
		period = "7d"
	case "30d":
		period = "30d"
	case "all":
		period = "all"
	}

	// In standalone API mode, return empty breakdown
	// Keeper methods require sdk.Context which is not available in HTTP context
	resp := RevenueBreakdownResponse{
		PoolID:      poolID,
		Period:      period,
		TotalAmount: "0",
		Breakdown:   map[string]string{},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// ============ Community Pool Handlers ============

// CreateCommunityPoolRequest represents the request body for creating a community pool
type CreateCommunityPoolRequest struct {
	Owner              string   `json:"owner"`
	Name               string   `json:"name"`
	Description        string   `json:"description"`
	MinDeposit         string   `json:"min_deposit"`
	MaxDeposit         string   `json:"max_deposit"`
	ManagementFee      string   `json:"management_fee"`
	PerformanceFee     string   `json:"performance_fee"`
	OwnerMinStake      string   `json:"owner_min_stake"`
	LockPeriodDays     int64    `json:"lock_period_days"`
	RedemptionDelayDays int64   `json:"redemption_delay_days"`
	IsPrivate          bool     `json:"is_private"`
	MaxLeverage        string   `json:"max_leverage"`
	AllowedMarkets     []string `json:"allowed_markets"`
	Tags               []string `json:"tags"`
}

// CreateCommunityPool handles POST /v1/riverpool/community/create
func (h *RiverpoolHandler) CreateCommunityPool(w http.ResponseWriter, r *http.Request) {
	var req CreateCommunityPoolRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	// In standalone API mode, return error
	// Creating community pools requires sdk.Context and blockchain state
	http.Error(w, "community pool creation not available in standalone API mode", http.StatusServiceUnavailable)
}

// PoolHolderResponse represents a holder in API responses
type PoolHolderResponse struct {
	Address     string `json:"address"`
	Shares      string `json:"shares"`
	Value       string `json:"value"`
	DepositedAt int64  `json:"deposited_at"`
	IsOwner     bool   `json:"is_owner"`
}

// GetPoolHolders handles GET /v1/riverpool/community/{poolId}/holders
func (h *RiverpoolHandler) GetPoolHolders(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	poolID := vars["poolId"]
	ctx := r.Context()

	pool, err := h.queryServer.Pool(ctx, poolID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	// Get all deposits for this pool
	deposits, _, err := h.queryServer.PoolDeposits(ctx, poolID, 0, 1000)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Aggregate by depositor
	holderMap := make(map[string]*PoolHolderResponse)
	for _, dep := range deposits {
		if holder, exists := holderMap[dep.Depositor]; exists {
			shares, _ := math.LegacyNewDecFromStr(holder.Shares)
			depShares := dep.Shares
			holder.Shares = shares.Add(depShares).String()
		} else {
			value := dep.Shares.Mul(pool.NAV)
			holderMap[dep.Depositor] = &PoolHolderResponse{
				Address:     dep.Depositor,
				Shares:      dep.Shares.String(),
				Value:       value.String(),
				DepositedAt: dep.DepositedAt,
				IsOwner:     dep.Depositor == pool.Owner,
			}
		}
	}

	holders := make([]PoolHolderResponse, 0, len(holderMap))
	for _, h := range holderMap {
		holders = append(holders, *h)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"pool_id": poolID,
		"holders": holders,
		"total":   len(holders),
	})
}

// PoolPositionResponse represents a pool position in API responses
type PoolPositionResponse struct {
	PositionID       string `json:"position_id"`
	MarketID         string `json:"market_id"`
	Side             string `json:"side"`
	Size             string `json:"size"`
	EntryPrice       string `json:"entry_price"`
	MarkPrice        string `json:"mark_price"`
	PnL              string `json:"pnl"`
	PnLPercent       string `json:"pnl_percent"`
	Leverage         string `json:"leverage"`
	LiquidationPrice string `json:"liquidation_price"`
	Margin           string `json:"margin"`
}

// GetPoolPositions handles GET /v1/riverpool/community/{poolId}/positions
func (h *RiverpoolHandler) GetPoolPositions(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	poolID := vars["poolId"]

	// Placeholder - In production, query actual positions from perpetual keeper
	positions := []PoolPositionResponse{}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"pool_id":   poolID,
		"positions": positions,
		"total":     len(positions),
	})
}

// PoolTradeResponse represents a pool trade in API responses
type PoolTradeResponse struct {
	TradeID   string `json:"trade_id"`
	MarketID  string `json:"market_id"`
	Side      string `json:"side"`
	Price     string `json:"price"`
	Size      string `json:"size"`
	Fee       string `json:"fee"`
	PnL       string `json:"pnl"`
	Timestamp int64  `json:"timestamp"`
}

// GetPoolTrades handles GET /v1/riverpool/community/{poolId}/trades
func (h *RiverpoolHandler) GetPoolTrades(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	poolID := vars["poolId"]

	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit == 0 {
		limit = 50
	}

	// Placeholder - In production, query actual trades
	trades := []PoolTradeResponse{}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"pool_id": poolID,
		"trades":  trades,
		"total":   len(trades),
	})
}

// DepositOwnerStakeRequest represents the request body
type DepositOwnerStakeRequest struct {
	Owner  string `json:"owner"`
	Amount string `json:"amount"`
}

// DepositOwnerStake handles POST /v1/riverpool/community/{poolId}/stake
func (h *RiverpoolHandler) DepositOwnerStake(w http.ResponseWriter, r *http.Request) {
	// In standalone API mode, return error
	// Owner stake operations require sdk.Context and blockchain state
	http.Error(w, "owner stake deposit not available in standalone API mode", http.StatusServiceUnavailable)
}

// InviteCodeResponse represents an invite code in API responses
type InviteCodeResponse struct {
	Code      string `json:"code"`
	MaxUses   int64  `json:"max_uses"`
	UsedCount int64  `json:"used_count"`
	ExpiresAt int64  `json:"expires_at"`
	CreatedAt int64  `json:"created_at"`
	IsActive  bool   `json:"is_active"`
}

// GetInviteCodes handles GET /v1/riverpool/community/{poolId}/invites
func (h *RiverpoolHandler) GetInviteCodes(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	poolID := vars["poolId"]

	// In standalone API mode, return empty codes
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"pool_id": poolID,
		"codes":   []InviteCodeResponse{},
		"total":   0,
	})
}

// GenerateInviteCodeRequest represents the request body
type GenerateInviteCodeRequest struct {
	Owner         string `json:"owner"`
	MaxUses       int    `json:"max_uses"`
	ExpiresInDays int    `json:"expires_in_days"`
}

// GenerateInviteCode handles POST /v1/riverpool/community/{poolId}/invites
func (h *RiverpoolHandler) GenerateInviteCode(w http.ResponseWriter, r *http.Request) {
	// In standalone API mode, return error
	http.Error(w, "invite code generation not available in standalone API mode", http.StatusServiceUnavailable)
}

// PoolOwnerRequest represents a request with just owner field
type PoolOwnerRequest struct {
	Owner string `json:"owner"`
}

// PausePool handles POST /v1/riverpool/community/{poolId}/pause
func (h *RiverpoolHandler) PausePool(w http.ResponseWriter, r *http.Request) {
	// In standalone API mode, return error
	http.Error(w, "pool pause not available in standalone API mode", http.StatusServiceUnavailable)
}

// ResumePool handles POST /v1/riverpool/community/{poolId}/resume
// Note: In standalone API mode, pool state modifications require blockchain transactions
func (h *RiverpoolHandler) ResumePool(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "pool resume not available in standalone API mode", http.StatusServiceUnavailable)
}

// ClosePool handles POST /v1/riverpool/community/{poolId}/close
// Note: In standalone API mode, pool state modifications require blockchain transactions
func (h *RiverpoolHandler) ClosePool(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "pool close not available in standalone API mode", http.StatusServiceUnavailable)
}

// GetUserOwnedPools handles GET /v1/riverpool/user/{user}/owned-pools
func (h *RiverpoolHandler) GetUserOwnedPools(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	user := vars["user"]
	ctx := r.Context()

	// Get all community pools and filter by owner
	pools, _, err := h.queryServer.Pools(ctx, 0, 1000)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	ownedPools := make([]PoolResponse, 0)
	for _, pool := range pools {
		if pool.PoolType == types.PoolTypeCommunity && pool.Owner == user {
			ownedPools = append(ownedPools, poolToResponse(pool))
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"owner": user,
		"pools": ownedPools,
		"total": len(ownedPools),
	})
}
