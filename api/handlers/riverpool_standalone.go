package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"cosmossdk.io/math"
	"github.com/openalpha/perp-dex/api/types"
)

// RiverpoolStandaloneHandler handles riverpool API requests in standalone mode
type RiverpoolStandaloneHandler struct {
	service types.RiverpoolService
}

// NewRiverpoolStandaloneHandler creates a new standalone RiverpoolHandler
func NewRiverpoolStandaloneHandler(svc types.RiverpoolService) *RiverpoolStandaloneHandler {
	return &RiverpoolStandaloneHandler{
		service: svc,
	}
}

// Helper to extract path parameters (since we're using http.ServeMux not gorilla/mux)
func extractPathParam(path, prefix, suffix string) string {
	path = strings.TrimPrefix(path, prefix)
	if suffix != "" {
		path = strings.TrimSuffix(path, suffix)
	}
	return path
}

// GetPools handles GET /v1/riverpool/pools
func (h *RiverpoolStandaloneHandler) GetPools(w http.ResponseWriter, r *http.Request) {
	pools, err := h.service.GetPools()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"pools": pools,
		"total": len(pools),
	})
}

// GetPool handles GET /v1/riverpool/pools/{poolId}
func (h *RiverpoolStandaloneHandler) GetPool(w http.ResponseWriter, r *http.Request) {
	poolID := r.Header.Get("X-Pool-ID")
	if poolID == "" {
		poolID = extractPathParam(r.URL.Path, "/v1/riverpool/pools/", "")
		// Check if it's a sub-route
		if strings.Contains(poolID, "/") {
			parts := strings.SplitN(poolID, "/", 2)
			poolID = parts[0]
		}
	}

	pool, err := h.service.GetPool(poolID)
	if err != nil {
		writeError(w, http.StatusNotFound, "pool_not_found", err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(pool)
}

// GetPoolsByType handles GET /v1/riverpool/pools/type/{poolType}
func (h *RiverpoolStandaloneHandler) GetPoolsByType(w http.ResponseWriter, r *http.Request) {
	poolType := extractPathParam(r.URL.Path, "/v1/riverpool/pools/type/", "")

	pools, err := h.service.GetPoolsByType(poolType)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"pools": pools,
		"total": len(pools),
		"type":  poolType,
	})
}

// GetPoolStats handles GET /v1/riverpool/pools/{poolId}/stats
func (h *RiverpoolStandaloneHandler) GetPoolStats(w http.ResponseWriter, r *http.Request) {
	poolID := r.Header.Get("X-Pool-ID")
	if poolID == "" {
		poolID = extractPathParam(r.URL.Path, "/v1/riverpool/pools/", "/stats")
	}

	stats, err := h.service.GetPoolStats(poolID)
	if err != nil {
		writeError(w, http.StatusNotFound, "pool_not_found", err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

// GetNAVHistory handles GET /v1/riverpool/pools/{poolId}/nav
func (h *RiverpoolStandaloneHandler) GetNAVHistory(w http.ResponseWriter, r *http.Request) {
	poolID := r.Header.Get("X-Pool-ID")
	if poolID == "" {
		poolID = extractPathParam(r.URL.Path, "/v1/riverpool/pools/", "/nav")
	}

	days := 30
	if d := r.URL.Query().Get("days"); d != "" {
		if parsed, err := strconv.Atoi(d); err == nil {
			days = parsed
		}
	}

	history, err := h.service.GetNAVHistory(poolID, days)
	if err != nil {
		writeError(w, http.StatusNotFound, "pool_not_found", err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"pool_id": poolID,
		"history": history,
		"days":    days,
	})
}

// GetDDGuardState handles GET /v1/riverpool/pools/{poolId}/ddguard
func (h *RiverpoolStandaloneHandler) GetDDGuardState(w http.ResponseWriter, r *http.Request) {
	poolID := r.Header.Get("X-Pool-ID")
	if poolID == "" {
		poolID = extractPathParam(r.URL.Path, "/v1/riverpool/pools/", "/ddguard")
	}

	state, err := h.service.GetDDGuardState(poolID)
	if err != nil {
		writeError(w, http.StatusNotFound, "pool_not_found", err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(state)
}

// GetUserDeposits handles GET /v1/riverpool/user/{user}/deposits
func (h *RiverpoolStandaloneHandler) GetUserDeposits(w http.ResponseWriter, r *http.Request) {
	user := r.Header.Get("X-User-Address")
	if user == "" {
		user = extractPathParam(r.URL.Path, "/v1/riverpool/user/", "/deposits")
		if strings.Contains(user, "/") {
			user = strings.Split(user, "/")[0]
		}
	}

	deposits, err := h.service.GetUserDeposits(user)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"user":     user,
		"deposits": deposits,
		"total":    len(deposits),
	})
}

// GetUserWithdrawals handles GET /v1/riverpool/user/{user}/withdrawals
func (h *RiverpoolStandaloneHandler) GetUserWithdrawals(w http.ResponseWriter, r *http.Request) {
	user := r.Header.Get("X-User-Address")
	if user == "" {
		user = extractPathParam(r.URL.Path, "/v1/riverpool/user/", "/withdrawals")
	}

	withdrawals, err := h.service.GetUserWithdrawals(user)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"user":        user,
		"withdrawals": withdrawals,
		"total":       len(withdrawals),
	})
}

// GetUserPoolBalance handles GET /v1/riverpool/pools/{poolId}/user/{user}/balance
func (h *RiverpoolStandaloneHandler) GetUserPoolBalance(w http.ResponseWriter, r *http.Request) {
	// Extract poolId and user from path: /v1/riverpool/pools/{poolId}/user/{user}/balance
	path := strings.TrimPrefix(r.URL.Path, "/v1/riverpool/pools/")
	parts := strings.Split(path, "/user/")
	if len(parts) != 2 {
		writeError(w, http.StatusBadRequest, "invalid_path", "Invalid path format")
		return
	}
	poolID := parts[0]
	user := strings.TrimSuffix(parts[1], "/balance")

	balance, err := h.service.GetUserPoolBalance(poolID, user)
	if err != nil {
		writeError(w, http.StatusNotFound, "not_found", err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(balance)
}

// GetUserOwnedPools handles GET /v1/riverpool/user/{user}/owned-pools
func (h *RiverpoolStandaloneHandler) GetUserOwnedPools(w http.ResponseWriter, r *http.Request) {
	user := extractPathParam(r.URL.Path, "/v1/riverpool/user/", "/owned-pools")

	pools, err := h.service.GetUserOwnedPools(user)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"owner": user,
		"pools": pools,
		"total": len(pools),
	})
}

// GetPoolDeposits handles GET /v1/riverpool/pools/{poolId}/deposits
func (h *RiverpoolStandaloneHandler) GetPoolDeposits(w http.ResponseWriter, r *http.Request) {
	poolID := r.Header.Get("X-Pool-ID")
	if poolID == "" {
		poolID = extractPathParam(r.URL.Path, "/v1/riverpool/pools/", "/deposits")
	}

	offset := 0
	limit := 100
	if o := r.URL.Query().Get("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil {
			offset = parsed
		}
	}
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil {
			limit = parsed
		}
	}

	deposits, total, err := h.service.GetPoolDeposits(poolID, offset, limit)
	if err != nil {
		writeError(w, http.StatusNotFound, "pool_not_found", err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"pool_id":  poolID,
		"deposits": deposits,
		"total":    total,
		"offset":   offset,
		"limit":    limit,
	})
}

// GetPendingWithdrawals handles GET /v1/riverpool/pools/{poolId}/withdrawals/pending
func (h *RiverpoolStandaloneHandler) GetPendingWithdrawals(w http.ResponseWriter, r *http.Request) {
	poolID := extractPathParam(r.URL.Path, "/v1/riverpool/pools/", "/withdrawals/pending")

	withdrawals, err := h.service.GetPendingWithdrawals(poolID)
	if err != nil {
		writeError(w, http.StatusNotFound, "pool_not_found", err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"pool_id":     poolID,
		"withdrawals": withdrawals,
		"total":       len(withdrawals),
	})
}

// EstimateDeposit handles GET /v1/riverpool/pools/{poolId}/estimate/deposit
func (h *RiverpoolStandaloneHandler) EstimateDeposit(w http.ResponseWriter, r *http.Request) {
	poolID := extractPathParam(r.URL.Path, "/v1/riverpool/pools/", "/estimate/deposit")

	amountStr := r.URL.Query().Get("amount")
	if amountStr == "" {
		writeError(w, http.StatusBadRequest, "missing_amount", "amount query parameter is required")
		return
	}

	amount, err := math.LegacyNewDecFromStr(amountStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_amount", "invalid amount format")
		return
	}

	estimate, err := h.service.EstimateDeposit(poolID, amount)
	if err != nil {
		writeError(w, http.StatusNotFound, "pool_not_found", err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(estimate)
}

// EstimateWithdrawal handles GET /v1/riverpool/pools/{poolId}/estimate/withdrawal
func (h *RiverpoolStandaloneHandler) EstimateWithdrawal(w http.ResponseWriter, r *http.Request) {
	poolID := extractPathParam(r.URL.Path, "/v1/riverpool/pools/", "/estimate/withdrawal")

	sharesStr := r.URL.Query().Get("shares")
	if sharesStr == "" {
		writeError(w, http.StatusBadRequest, "missing_shares", "shares query parameter is required")
		return
	}

	shares, err := math.LegacyNewDecFromStr(sharesStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_shares", "invalid shares format")
		return
	}

	estimate, err := h.service.EstimateWithdrawal(poolID, shares)
	if err != nil {
		writeError(w, http.StatusNotFound, "pool_not_found", err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(estimate)
}

// Deposit handles POST /v1/riverpool/deposit
func (h *RiverpoolStandaloneHandler) Deposit(w http.ResponseWriter, r *http.Request) {
	var req struct {
		PoolID string `json:"pool_id"`
		User   string `json:"user"`
		Amount string `json:"amount"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "Invalid JSON body")
		return
	}

	if req.PoolID == "" || req.User == "" || req.Amount == "" {
		writeError(w, http.StatusBadRequest, "missing_fields", "pool_id, user, and amount are required")
		return
	}

	amount, err := math.LegacyNewDecFromStr(req.Amount)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_amount", "invalid amount format")
		return
	}

	result, err := h.service.Deposit(req.PoolID, req.User, amount)
	if err != nil {
		writeError(w, http.StatusBadRequest, "deposit_failed", err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(result)
}

// RequestWithdrawal handles POST /v1/riverpool/withdrawal/request
func (h *RiverpoolStandaloneHandler) RequestWithdrawal(w http.ResponseWriter, r *http.Request) {
	var req struct {
		PoolID string `json:"pool_id"`
		User   string `json:"user"`
		Shares string `json:"shares"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "Invalid JSON body")
		return
	}

	if req.PoolID == "" || req.User == "" || req.Shares == "" {
		writeError(w, http.StatusBadRequest, "missing_fields", "pool_id, user, and shares are required")
		return
	}

	shares, err := math.LegacyNewDecFromStr(req.Shares)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_shares", "invalid shares format")
		return
	}

	result, err := h.service.RequestWithdrawal(req.PoolID, req.User, shares)
	if err != nil {
		writeError(w, http.StatusBadRequest, "withdrawal_failed", err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(result)
}

// ClaimWithdrawal handles POST /v1/riverpool/withdrawal/claim
func (h *RiverpoolStandaloneHandler) ClaimWithdrawal(w http.ResponseWriter, r *http.Request) {
	var req struct {
		WithdrawalID string `json:"withdrawal_id"`
		User         string `json:"user"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "Invalid JSON body")
		return
	}

	if req.WithdrawalID == "" || req.User == "" {
		writeError(w, http.StatusBadRequest, "missing_fields", "withdrawal_id and user are required")
		return
	}

	result, err := h.service.ClaimWithdrawal(req.WithdrawalID, req.User)
	if err != nil {
		writeError(w, http.StatusBadRequest, "claim_failed", err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// CancelWithdrawal handles POST /v1/riverpool/withdrawal/cancel
func (h *RiverpoolStandaloneHandler) CancelWithdrawal(w http.ResponseWriter, r *http.Request) {
	var req struct {
		WithdrawalID string `json:"withdrawal_id"`
		User         string `json:"user"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "Invalid JSON body")
		return
	}

	if req.WithdrawalID == "" || req.User == "" {
		writeError(w, http.StatusBadRequest, "missing_fields", "withdrawal_id and user are required")
		return
	}

	if err := h.service.CancelWithdrawal(req.WithdrawalID, req.User); err != nil {
		writeError(w, http.StatusBadRequest, "cancel_failed", err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Withdrawal cancelled successfully",
	})
}

// GetPoolRevenue handles GET /v1/riverpool/pools/{poolId}/revenue
func (h *RiverpoolStandaloneHandler) GetPoolRevenue(w http.ResponseWriter, r *http.Request) {
	poolID := r.Header.Get("X-Pool-ID")
	if poolID == "" {
		poolID = extractPathParam(r.URL.Path, "/v1/riverpool/pools/", "/revenue")
		// Handle sub-routes
		if strings.Contains(poolID, "/") {
			parts := strings.SplitN(poolID, "/", 2)
			poolID = parts[0]
		}
	}

	revenue, err := h.service.GetPoolRevenue(poolID)
	if err != nil {
		writeError(w, http.StatusNotFound, "pool_not_found", err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(revenue)
}

// GetRevenueRecords handles GET /v1/riverpool/pools/{poolId}/revenue/records
func (h *RiverpoolStandaloneHandler) GetRevenueRecords(w http.ResponseWriter, r *http.Request) {
	poolID := extractPathParam(r.URL.Path, "/v1/riverpool/pools/", "/revenue/records")

	limit := 50
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil {
			limit = parsed
		}
	}

	records, err := h.service.GetRevenueRecords(poolID, limit)
	if err != nil {
		writeError(w, http.StatusNotFound, "pool_not_found", err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"pool_id": poolID,
		"records": records,
		"total":   len(records),
	})
}

// GetRevenueBreakdown handles GET /v1/riverpool/pools/{poolId}/revenue/breakdown
func (h *RiverpoolStandaloneHandler) GetRevenueBreakdown(w http.ResponseWriter, r *http.Request) {
	poolID := extractPathParam(r.URL.Path, "/v1/riverpool/pools/", "/revenue/breakdown")

	breakdown, err := h.service.GetRevenueBreakdown(poolID)
	if err != nil {
		writeError(w, http.StatusNotFound, "pool_not_found", err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(breakdown)
}

// CreateCommunityPool handles POST /v1/riverpool/community/create
func (h *RiverpoolStandaloneHandler) CreateCommunityPool(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Owner  string                    `json:"owner"`
		Params types.CommunityPoolParams `json:"params"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "Invalid JSON body")
		return
	}

	if req.Owner == "" || req.Params.Name == "" {
		writeError(w, http.StatusBadRequest, "missing_fields", "owner and params.name are required")
		return
	}

	pool, err := h.service.CreateCommunityPool(req.Owner, &req.Params)
	if err != nil {
		writeError(w, http.StatusBadRequest, "create_failed", err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(pool)
}

// GetPoolHolders handles GET /v1/riverpool/pools/{poolId}/holders
func (h *RiverpoolStandaloneHandler) GetPoolHolders(w http.ResponseWriter, r *http.Request) {
	poolID := r.Header.Get("X-Pool-ID")
	if poolID == "" {
		poolID = extractPathParam(r.URL.Path, "/v1/riverpool/pools/", "/holders")
	}

	holders, err := h.service.GetPoolHolders(poolID)
	if err != nil {
		writeError(w, http.StatusNotFound, "pool_not_found", err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"pool_id": poolID,
		"holders": holders,
		"total":   len(holders),
	})
}

// GetPoolPositions handles GET /v1/riverpool/pools/{poolId}/positions
func (h *RiverpoolStandaloneHandler) GetPoolPositions(w http.ResponseWriter, r *http.Request) {
	poolID := r.Header.Get("X-Pool-ID")
	if poolID == "" {
		poolID = extractPathParam(r.URL.Path, "/v1/riverpool/pools/", "/positions")
	}

	positions, err := h.service.GetPoolPositions(poolID)
	if err != nil {
		writeError(w, http.StatusNotFound, "pool_not_found", err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"pool_id":   poolID,
		"positions": positions,
		"total":     len(positions),
	})
}

// GetPoolTrades handles GET /v1/riverpool/pools/{poolId}/trades
func (h *RiverpoolStandaloneHandler) GetPoolTrades(w http.ResponseWriter, r *http.Request) {
	poolID := r.Header.Get("X-Pool-ID")
	if poolID == "" {
		poolID = extractPathParam(r.URL.Path, "/v1/riverpool/pools/", "/trades")
	}

	limit := 50
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil {
			limit = parsed
		}
	}

	trades, err := h.service.GetPoolTrades(poolID, limit)
	if err != nil {
		writeError(w, http.StatusNotFound, "pool_not_found", err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"pool_id": poolID,
		"trades":  trades,
		"total":   len(trades),
	})
}

// GetInviteCodes handles GET /v1/riverpool/community/{poolId}/invites
func (h *RiverpoolStandaloneHandler) GetInviteCodes(w http.ResponseWriter, r *http.Request) {
	poolID := extractPathParam(r.URL.Path, "/v1/riverpool/community/", "/invites")
	owner := r.Header.Get("X-Owner-Address")

	if owner == "" {
		writeError(w, http.StatusBadRequest, "missing_owner", "X-Owner-Address header is required")
		return
	}

	codes, err := h.service.GetInviteCodes(poolID, owner)
	if err != nil {
		writeError(w, http.StatusBadRequest, "get_codes_failed", err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"pool_id": poolID,
		"codes":   codes,
		"total":   len(codes),
	})
}

// GenerateInviteCode handles POST /v1/riverpool/community/{poolId}/invite
func (h *RiverpoolStandaloneHandler) GenerateInviteCode(w http.ResponseWriter, r *http.Request) {
	poolID := r.Header.Get("X-Pool-ID")
	if poolID == "" {
		poolID = extractPathParam(r.URL.Path, "/v1/riverpool/community/", "/invite")
	}

	var req struct {
		Owner string `json:"owner"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "Invalid JSON body")
		return
	}

	if req.Owner == "" {
		writeError(w, http.StatusBadRequest, "missing_owner", "owner is required")
		return
	}

	code, err := h.service.GenerateInviteCode(poolID, req.Owner)
	if err != nil {
		writeError(w, http.StatusBadRequest, "generate_failed", err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(code)
}

// PausePool handles POST /v1/riverpool/community/{poolId}/pause
func (h *RiverpoolStandaloneHandler) PausePool(w http.ResponseWriter, r *http.Request) {
	poolID := r.Header.Get("X-Pool-ID")
	if poolID == "" {
		poolID = extractPathParam(r.URL.Path, "/v1/riverpool/community/", "/pause")
	}

	var req struct {
		Owner string `json:"owner"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "Invalid JSON body")
		return
	}

	if req.Owner == "" {
		writeError(w, http.StatusBadRequest, "missing_owner", "owner is required")
		return
	}

	if err := h.service.PausePool(poolID, req.Owner); err != nil {
		writeError(w, http.StatusBadRequest, "pause_failed", err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Pool paused successfully",
	})
}

// ResumePool handles POST /v1/riverpool/community/{poolId}/resume
func (h *RiverpoolStandaloneHandler) ResumePool(w http.ResponseWriter, r *http.Request) {
	poolID := r.Header.Get("X-Pool-ID")
	if poolID == "" {
		poolID = extractPathParam(r.URL.Path, "/v1/riverpool/community/", "/resume")
	}

	var req struct {
		Owner string `json:"owner"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "Invalid JSON body")
		return
	}

	if req.Owner == "" {
		writeError(w, http.StatusBadRequest, "missing_owner", "owner is required")
		return
	}

	if err := h.service.ResumePool(poolID, req.Owner); err != nil {
		writeError(w, http.StatusBadRequest, "resume_failed", err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Pool resumed successfully",
	})
}

// ClosePool handles POST /v1/riverpool/community/{poolId}/close
func (h *RiverpoolStandaloneHandler) ClosePool(w http.ResponseWriter, r *http.Request) {
	poolID := r.Header.Get("X-Pool-ID")
	if poolID == "" {
		poolID = extractPathParam(r.URL.Path, "/v1/riverpool/community/", "/close")
	}

	var req struct {
		Owner string `json:"owner"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "Invalid JSON body")
		return
	}

	if req.Owner == "" {
		writeError(w, http.StatusBadRequest, "missing_owner", "owner is required")
		return
	}

	if err := h.service.ClosePool(poolID, req.Owner); err != nil {
		writeError(w, http.StatusBadRequest, "close_failed", err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Pool closed successfully",
	})
}

// GetPoolWithdrawals handles GET /v1/riverpool/pools/{poolId}/withdrawals
func (h *RiverpoolStandaloneHandler) GetPoolWithdrawals(w http.ResponseWriter, r *http.Request) {
	poolID := r.Header.Get("X-Pool-ID")
	if poolID == "" {
		poolID = extractPathParam(r.URL.Path, "/v1/riverpool/pools/", "/withdrawals")
	}

	offset := 0
	limit := 100
	if o := r.URL.Query().Get("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil {
			offset = parsed
		}
	}
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil {
			limit = parsed
		}
	}

	withdrawals, total, err := h.service.GetPoolWithdrawals(poolID, offset, limit)
	if err != nil {
		writeError(w, http.StatusNotFound, "pool_not_found", err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"pool_id":     poolID,
		"withdrawals": withdrawals,
		"total":       total,
		"offset":      offset,
		"limit":       limit,
	})
}

// GetUserPools handles GET /v1/riverpool/user/{address}/pools
func (h *RiverpoolStandaloneHandler) GetUserPools(w http.ResponseWriter, r *http.Request) {
	address := r.Header.Get("X-User-Address")
	if address == "" {
		address = extractPathParam(r.URL.Path, "/v1/riverpool/user/", "/pools")
	}

	deposits, err := h.service.GetUserDeposits(address)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}

	// Extract unique pool IDs
	poolMap := make(map[string]bool)
	for _, d := range deposits {
		poolMap[d.PoolID] = true
	}

	var poolIDs []string
	for pid := range poolMap {
		poolIDs = append(poolIDs, pid)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"user":     address,
		"pool_ids": poolIDs,
		"total":    len(poolIDs),
	})
}

// UpdateCommunityPool handles POST /v1/riverpool/community/{poolId}/update
func (h *RiverpoolStandaloneHandler) UpdateCommunityPool(w http.ResponseWriter, r *http.Request) {
	poolID := r.Header.Get("X-Pool-ID")
	if poolID == "" {
		poolID = extractPathParam(r.URL.Path, "/v1/riverpool/community/", "/update")
	}

	var req struct {
		Owner  string                    `json:"owner"`
		Params types.CommunityPoolParams `json:"params"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "Invalid JSON body")
		return
	}

	if req.Owner == "" {
		writeError(w, http.StatusBadRequest, "missing_owner", "owner is required")
		return
	}

	pool, err := h.service.UpdateCommunityPool(poolID, req.Owner, &req.Params)
	if err != nil {
		writeError(w, http.StatusBadRequest, "update_failed", err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(pool)
}

// PlacePoolOrder handles POST /v1/riverpool/community/{poolId}/order
func (h *RiverpoolStandaloneHandler) PlacePoolOrder(w http.ResponseWriter, r *http.Request) {
	poolID := r.Header.Get("X-Pool-ID")
	if poolID == "" {
		poolID = extractPathParam(r.URL.Path, "/v1/riverpool/community/", "/order")
	}

	var req struct {
		Owner    string `json:"owner"`
		MarketID string `json:"market_id"`
		Side     string `json:"side"`
		Size     string `json:"size"`
		Price    string `json:"price,omitempty"`
		Leverage string `json:"leverage"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "Invalid JSON body")
		return
	}

	if req.Owner == "" || req.MarketID == "" || req.Side == "" || req.Size == "" {
		writeError(w, http.StatusBadRequest, "missing_fields", "owner, market_id, side, and size are required")
		return
	}

	size, err := math.LegacyNewDecFromStr(req.Size)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_size", "invalid size format")
		return
	}

	var price math.LegacyDec
	if req.Price != "" {
		price, err = math.LegacyNewDecFromStr(req.Price)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_price", "invalid price format")
			return
		}
	}

	leverage := math.LegacyNewDec(10) // default
	if req.Leverage != "" {
		leverage, err = math.LegacyNewDecFromStr(req.Leverage)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_leverage", "invalid leverage format")
			return
		}
	}

	result, err := h.service.PlacePoolOrder(poolID, req.Owner, req.MarketID, req.Side, size, price, leverage)
	if err != nil {
		writeError(w, http.StatusBadRequest, "order_failed", err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(result)
}

// ClosePoolPosition handles POST /v1/riverpool/community/{poolId}/close
func (h *RiverpoolStandaloneHandler) ClosePoolPosition(w http.ResponseWriter, r *http.Request) {
	poolID := r.Header.Get("X-Pool-ID")
	if poolID == "" {
		poolID = extractPathParam(r.URL.Path, "/v1/riverpool/community/", "/close")
	}

	var req struct {
		Owner      string `json:"owner"`
		PositionID string `json:"position_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "Invalid JSON body")
		return
	}

	if req.Owner == "" || req.PositionID == "" {
		writeError(w, http.StatusBadRequest, "missing_fields", "owner and position_id are required")
		return
	}

	result, err := h.service.ClosePoolPosition(poolID, req.Owner, req.PositionID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "close_failed", err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}
