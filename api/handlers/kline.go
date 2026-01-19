package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/openalpha/perp-dex/x/perpetual/keeper"
)

// KlineHandler handles K-line API requests
type KlineHandler struct {
	perpetualKeeper *keeper.Keeper
}

// NewKlineHandler creates a new KlineHandler
func NewKlineHandler(perpetualKeeper *keeper.Keeper) *KlineHandler {
	return &KlineHandler{
		perpetualKeeper: perpetualKeeper,
	}
}

// KlineResponse represents the API response for K-lines
type KlineResponse struct {
	MarketID string        `json:"market_id"`
	Interval string        `json:"interval"`
	Klines   []KlineData   `json:"klines"`
}

// KlineData represents a single K-line in API response
type KlineData struct {
	Time     int64   `json:"time"`     // Unix timestamp
	Open     float64 `json:"open"`
	High     float64 `json:"high"`
	Low      float64 `json:"low"`
	Close    float64 `json:"close"`
	Volume   float64 `json:"volume"`
	Turnover float64 `json:"turnover"`
}

// GetKlines handles GET /v1/markets/{market_id}/klines
// Query params: interval, from, to, limit
func (h *KlineHandler) GetKlines(w http.ResponseWriter, r *http.Request) {
	// Parse path parameter
	marketID := r.PathValue("market_id")
	if marketID == "" {
		marketID = r.URL.Query().Get("market_id")
	}
	if marketID == "" {
		http.Error(w, "market_id is required", http.StatusBadRequest)
		return
	}

	// Parse query parameters
	interval := r.URL.Query().Get("interval")
	if interval == "" {
		interval = "1m" // Default to 1 minute
	}

	fromStr := r.URL.Query().Get("from")
	toStr := r.URL.Query().Get("to")
	limitStr := r.URL.Query().Get("limit")

	// Parse timestamps
	var from, to int64
	var limit int = 200 // Default limit

	if fromStr != "" {
		from, _ = strconv.ParseInt(fromStr, 10, 64)
	}
	if toStr != "" {
		to, _ = strconv.ParseInt(toStr, 10, 64)
	}
	if limitStr != "" {
		limit, _ = strconv.Atoi(limitStr)
		if limit > 1000 {
			limit = 1000 // Max limit
		}
	}

	// Get context (in production, this would come from the request context)
	ctx := sdk.Context{}

	// Get K-lines from keeper
	klines := h.perpetualKeeper.GetKlines(
		ctx,
		marketID,
		keeper.KlineInterval(interval),
		from,
		to,
		limit,
	)

	// Convert to response format
	response := KlineResponse{
		MarketID: marketID,
		Interval: interval,
		Klines:   make([]KlineData, 0, len(klines)),
	}

	for _, k := range klines {
		response.Klines = append(response.Klines, KlineData{
			Time:     k.Timestamp,
			Open:     k.Open.MustFloat64(),
			High:     k.High.MustFloat64(),
			Low:      k.Low.MustFloat64(),
			Close:    k.Close.MustFloat64(),
			Volume:   k.Volume.MustFloat64(),
			Turnover: k.Turnover.MustFloat64(),
		})
	}

	// Send response
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	json.NewEncoder(w).Encode(response)
}

// GetLatestKlines handles GET /v1/markets/{market_id}/klines/latest
// Query params: interval, limit
func (h *KlineHandler) GetLatestKlines(w http.ResponseWriter, r *http.Request) {
	// Parse path parameter
	marketID := r.PathValue("market_id")
	if marketID == "" {
		marketID = r.URL.Query().Get("market_id")
	}
	if marketID == "" {
		http.Error(w, "market_id is required", http.StatusBadRequest)
		return
	}

	// Parse query parameters
	interval := r.URL.Query().Get("interval")
	if interval == "" {
		interval = "1m"
	}

	limitStr := r.URL.Query().Get("limit")
	limit := 200
	if limitStr != "" {
		limit, _ = strconv.Atoi(limitStr)
		if limit > 1000 {
			limit = 1000
		}
	}

	// Get context
	ctx := sdk.Context{}

	// Get latest K-lines
	klines := h.perpetualKeeper.GetLatestKlines(
		ctx,
		marketID,
		keeper.KlineInterval(interval),
		limit,
	)

	// Convert to response format
	response := KlineResponse{
		MarketID: marketID,
		Interval: interval,
		Klines:   make([]KlineData, 0, len(klines)),
	}

	for _, k := range klines {
		response.Klines = append(response.Klines, KlineData{
			Time:     k.Timestamp,
			Open:     k.Open.MustFloat64(),
			High:     k.High.MustFloat64(),
			Low:      k.Low.MustFloat64(),
			Close:    k.Close.MustFloat64(),
			Volume:   k.Volume.MustFloat64(),
			Turnover: k.Turnover.MustFloat64(),
		})
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	json.NewEncoder(w).Encode(response)
}

// RegisterRoutes registers K-line API routes
func (h *KlineHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /v1/markets/{market_id}/klines", h.GetKlines)
	mux.HandleFunc("GET /v1/markets/{market_id}/klines/latest", h.GetLatestKlines)
}
