package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/openalpha/perp-dex/api/types"
)

// OrderHandler handles order-related HTTP requests
type OrderHandler struct {
	service types.OrderService
}

// NewOrderHandler creates a new order handler
func NewOrderHandler(service types.OrderService) *OrderHandler {
	return &OrderHandler{service: service}
}

// HandleOrders handles /v1/orders endpoint (GET for list, POST for create)
func (h *OrderHandler) HandleOrders(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.listOrders(w, r)
	case http.MethodPost:
		h.placeOrder(w, r)
	case http.MethodOptions:
		w.WriteHeader(http.StatusOK)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed")
	}
}

// HandleOrder handles /v1/orders/{id} endpoint (GET, PUT, DELETE)
func (h *OrderHandler) HandleOrder(w http.ResponseWriter, r *http.Request) {
	// Extract order ID from path
	path := r.URL.Path
	prefix := "/v1/orders/"
	if !strings.HasPrefix(path, prefix) {
		writeError(w, http.StatusBadRequest, "invalid_path", "Invalid path")
		return
	}
	orderID := strings.TrimPrefix(path, prefix)
	if orderID == "" {
		writeError(w, http.StatusBadRequest, "missing_order_id", "Order ID is required")
		return
	}

	switch r.Method {
	case http.MethodGet:
		h.getOrder(w, r, orderID)
	case http.MethodPut:
		h.modifyOrder(w, r, orderID)
	case http.MethodDelete:
		h.cancelOrder(w, r, orderID)
	case http.MethodOptions:
		w.WriteHeader(http.StatusOK)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed")
	}
}

// placeOrder handles POST /v1/orders
func (h *OrderHandler) placeOrder(w http.ResponseWriter, r *http.Request) {
	var req types.PlaceOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "Invalid JSON body")
		return
	}

	// Validate required fields
	if req.MarketID == "" {
		writeError(w, http.StatusBadRequest, "missing_market_id", "market_id is required")
		return
	}
	if req.Side == "" {
		writeError(w, http.StatusBadRequest, "missing_side", "side is required")
		return
	}
	if req.Type == "" {
		writeError(w, http.StatusBadRequest, "missing_type", "type is required")
		return
	}
	if req.Quantity == "" {
		writeError(w, http.StatusBadRequest, "missing_quantity", "quantity is required")
		return
	}
	if req.Type == "limit" && req.Price == "" {
		writeError(w, http.StatusBadRequest, "missing_price", "price is required for limit orders")
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

	resp, err := h.service.PlaceOrder(r.Context(), &req)
	if err != nil {
		writeError(w, http.StatusBadRequest, "place_order_failed", err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, resp)
}

// cancelOrder handles DELETE /v1/orders/{id}
func (h *OrderHandler) cancelOrder(w http.ResponseWriter, r *http.Request, orderID string) {
	trader := r.Header.Get("X-Trader-Address")
	if trader == "" {
		// Try to get from query param
		trader = r.URL.Query().Get("trader")
	}
	if trader == "" {
		writeError(w, http.StatusBadRequest, "missing_trader", "trader address is required")
		return
	}

	resp, err := h.service.CancelOrder(r.Context(), trader, orderID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			writeError(w, http.StatusNotFound, "order_not_found", err.Error())
		} else if strings.Contains(err.Error(), "unauthorized") {
			writeError(w, http.StatusForbidden, "unauthorized", err.Error())
		} else {
			writeError(w, http.StatusBadRequest, "cancel_order_failed", err.Error())
		}
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

// modifyOrder handles PUT /v1/orders/{id}
func (h *OrderHandler) modifyOrder(w http.ResponseWriter, r *http.Request, orderID string) {
	var req types.ModifyOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "Invalid JSON body")
		return
	}

	if req.Price == "" && req.Quantity == "" {
		writeError(w, http.StatusBadRequest, "missing_fields", "at least one of price or quantity is required")
		return
	}

	trader := r.Header.Get("X-Trader-Address")
	if trader == "" {
		writeError(w, http.StatusBadRequest, "missing_trader", "trader address is required")
		return
	}

	resp, err := h.service.ModifyOrder(r.Context(), trader, orderID, &req)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			writeError(w, http.StatusNotFound, "order_not_found", err.Error())
		} else if strings.Contains(err.Error(), "unauthorized") {
			writeError(w, http.StatusForbidden, "unauthorized", err.Error())
		} else {
			writeError(w, http.StatusBadRequest, "modify_order_failed", err.Error())
		}
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

// getOrder handles GET /v1/orders/{id}
func (h *OrderHandler) getOrder(w http.ResponseWriter, r *http.Request, orderID string) {
	order, err := h.service.GetOrder(r.Context(), orderID)
	if err != nil {
		writeError(w, http.StatusNotFound, "order_not_found", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{"order": order})
}

// listOrders handles GET /v1/orders
func (h *OrderHandler) listOrders(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()

	req := &types.ListOrdersRequest{
		Trader:   query.Get("trader"),
		MarketID: query.Get("market_id"),
		Status:   query.Get("status"),
		Cursor:   query.Get("cursor"),
	}

	// Parse limit
	limitStr := query.Get("limit")
	if limitStr != "" {
		var limit int
		if _, err := json.Number(limitStr).Int64(); err == nil {
			limit = 100 // default
		}
		req.Limit = limit
	}

	// Require trader for listing orders
	if req.Trader == "" {
		req.Trader = r.Header.Get("X-Trader-Address")
	}

	resp, err := h.service.ListOrders(r.Context(), req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "list_orders_failed", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

// Helper functions

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"error":   code,
		"message": message,
	})
}
