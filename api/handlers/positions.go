package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/openalpha/perp-dex/api/types"
)

// PositionHandler handles position-related HTTP requests
type PositionHandler struct {
	service types.PositionService
}

// NewPositionHandler creates a new position handler
func NewPositionHandler(service types.PositionService) *PositionHandler {
	return &PositionHandler{service: service}
}

// HandlePositions handles /v1/positions endpoint (GET for list)
func (h *PositionHandler) HandlePositions(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.listPositions(w, r)
	case http.MethodOptions:
		w.WriteHeader(http.StatusOK)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed")
	}
}

// HandlePosition handles /v1/positions/{marketID} endpoint (GET)
func (h *PositionHandler) HandlePosition(w http.ResponseWriter, r *http.Request) {
	// Extract market ID from path
	path := r.URL.Path
	prefix := "/v1/positions/"
	if !strings.HasPrefix(path, prefix) {
		writeError(w, http.StatusBadRequest, "invalid_path", "Invalid path")
		return
	}
	marketID := strings.TrimPrefix(path, prefix)
	if marketID == "" {
		writeError(w, http.StatusBadRequest, "missing_market_id", "Market ID is required")
		return
	}

	// Handle /v1/positions/close separately
	if marketID == "close" {
		h.closePosition(w, r)
		return
	}

	switch r.Method {
	case http.MethodGet:
		h.getPosition(w, r, marketID)
	case http.MethodOptions:
		w.WriteHeader(http.StatusOK)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed")
	}
}

// HandleClosePosition handles POST /v1/positions/close
func (h *PositionHandler) HandleClosePosition(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed")
		return
	}
	h.closePosition(w, r)
}

// listPositions handles GET /v1/positions
func (h *PositionHandler) listPositions(w http.ResponseWriter, r *http.Request) {
	trader := r.URL.Query().Get("trader")
	if trader == "" {
		trader = r.Header.Get("X-Trader-Address")
	}

	positions, err := h.service.GetPositions(r.Context(), trader)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "list_positions_failed", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"positions": positions,
		"total":     len(positions),
	})
}

// getPosition handles GET /v1/positions/{marketID}
func (h *PositionHandler) getPosition(w http.ResponseWriter, r *http.Request, marketID string) {
	trader := r.URL.Query().Get("trader")
	if trader == "" {
		trader = r.Header.Get("X-Trader-Address")
	}
	if trader == "" {
		writeError(w, http.StatusBadRequest, "missing_trader", "trader address is required")
		return
	}

	position, err := h.service.GetPosition(r.Context(), trader, marketID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			writeError(w, http.StatusNotFound, "position_not_found", err.Error())
		} else {
			writeError(w, http.StatusInternalServerError, "get_position_failed", err.Error())
		}
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{"position": position})
}

// closePosition handles POST /v1/positions/close
func (h *PositionHandler) closePosition(w http.ResponseWriter, r *http.Request) {
	var req types.ClosePositionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "Invalid JSON body")
		return
	}

	// Validate required fields
	if req.MarketID == "" {
		writeError(w, http.StatusBadRequest, "missing_market_id", "market_id is required")
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

	resp, err := h.service.ClosePosition(r.Context(), &req)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			writeError(w, http.StatusNotFound, "position_not_found", err.Error())
		} else {
			writeError(w, http.StatusBadRequest, "close_position_failed", err.Error())
		}
		return
	}

	writeJSON(w, http.StatusOK, resp)
}
