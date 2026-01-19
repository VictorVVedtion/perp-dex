package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/openalpha/perp-dex/api/types"
)

// AccountHandler handles account-related HTTP requests
type AccountHandler struct {
	service types.AccountService
}

// NewAccountHandler creates a new account handler
func NewAccountHandler(service types.AccountService) *AccountHandler {
	return &AccountHandler{service: service}
}

// HandleAccount handles /v1/account endpoint (GET for account info)
func (h *AccountHandler) HandleAccount(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.getAccount(w, r)
	case http.MethodOptions:
		w.WriteHeader(http.StatusOK)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed")
	}
}

// HandleDeposit handles POST /v1/account/deposit
func (h *AccountHandler) HandleDeposit(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed")
		return
	}

	var req types.DepositRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "Invalid JSON body")
		return
	}

	// Validate required fields
	if req.Amount == "" {
		writeError(w, http.StatusBadRequest, "missing_amount", "amount is required")
		return
	}

	// Get trader from header or body
	if req.Trader == "" {
		req.Trader = r.Header.Get("X-Trader-Address")
	}
	if req.Trader == "" {
		writeError(w, http.StatusBadRequest, "missing_trader", "trader address is required")
		return
	}

	resp, err := h.service.Deposit(r.Context(), &req)
	if err != nil {
		writeError(w, http.StatusBadRequest, "deposit_failed", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

// HandleWithdraw handles POST /v1/account/withdraw
func (h *AccountHandler) HandleWithdraw(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed")
		return
	}

	var req types.WithdrawRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "Invalid JSON body")
		return
	}

	// Validate required fields
	if req.Amount == "" {
		writeError(w, http.StatusBadRequest, "missing_amount", "amount is required")
		return
	}

	// Get trader from header or body
	if req.Trader == "" {
		req.Trader = r.Header.Get("X-Trader-Address")
	}
	if req.Trader == "" {
		writeError(w, http.StatusBadRequest, "missing_trader", "trader address is required")
		return
	}

	resp, err := h.service.Withdraw(r.Context(), &req)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			writeError(w, http.StatusNotFound, "account_not_found", err.Error())
		} else if strings.Contains(err.Error(), "insufficient") {
			writeError(w, http.StatusBadRequest, "insufficient_balance", err.Error())
		} else {
			writeError(w, http.StatusBadRequest, "withdraw_failed", err.Error())
		}
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

// getAccount handles GET /v1/account
func (h *AccountHandler) getAccount(w http.ResponseWriter, r *http.Request) {
	trader := r.URL.Query().Get("trader")
	if trader == "" {
		trader = r.Header.Get("X-Trader-Address")
	}
	if trader == "" {
		writeError(w, http.StatusBadRequest, "missing_trader", "trader address is required")
		return
	}

	account, err := h.service.GetAccount(r.Context(), trader)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "get_account_failed", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{"account": account})
}
