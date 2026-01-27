package api

// service_real_v2.go - Real E2E Service with actual Keeper implementations
// No mock data, full margin checking, position management, and price oracle

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"cosmossdk.io/log"
	"cosmossdk.io/math"
	"cosmossdk.io/store"
	"cosmossdk.io/store/metrics"
	storetypes "cosmossdk.io/store/types"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/openalpha/perp-dex/api/types"
	obkeeper "github.com/openalpha/perp-dex/x/orderbook/keeper"
	obtypes "github.com/openalpha/perp-dex/x/orderbook/types"
	perpkeeper "github.com/openalpha/perp-dex/x/perpetual/keeper"
	perptypes "github.com/openalpha/perp-dex/x/perpetual/types"
)

// RealServiceV2 implements all service interfaces with REAL Keeper implementations
// This is a true E2E service with:
// - Real perpetual.Keeper for market/account/position management
// - Real MarginChecker for margin validation
// - Real PositionManager for position operations
// - Hyperliquid Oracle for real-time prices
type RealServiceV2 struct {
	mu sync.RWMutex

	// Real Keepers
	perpKeeper      *perpkeeper.Keeper
	obKeeper        *obkeeper.Keeper
	marginChecker   *perpkeeper.MarginChecker
	positionManager *perpkeeper.PositionManager
	matchEngine     *obkeeper.MatchingEngineV2
	bankKeeper      *MemoryBankKeeper

	// Context and store
	sdkCtx   sdk.Context
	cms      storetypes.CommitMultiStore
	storeKey storetypes.StoreKey
	perpKey  storetypes.StoreKey

	// Oracle
	oracle *HyperliquidOracle

	// Logger
	logger log.Logger
}

// HyperliquidOracle fetches real-time prices from Hyperliquid API
type HyperliquidOracle struct {
	apiURL     string
	httpClient *http.Client
	cache      map[string]*PriceCache
	mu         sync.RWMutex
}

type PriceCache struct {
	Price     math.LegacyDec
	Timestamp time.Time
}

// OrderbookLevel represents a single price level in the orderbook
type OrderbookLevel struct {
	Price    string `json:"price"`
	Quantity string `json:"quantity"`
}

// OrderbookData represents L2 orderbook data
type OrderbookData struct {
	MarketID  string           `json:"market_id"`
	Bids      []OrderbookLevel `json:"bids"`
	Asks      []OrderbookLevel `json:"asks"`
	Timestamp int64            `json:"timestamp"`
}

// TradeData represents a single trade
type TradeData struct {
	TradeID   string `json:"trade_id"`
	MarketID  string `json:"market_id"`
	Price     string `json:"price"`
	Quantity  string `json:"quantity"`
	Side      string `json:"side"`
	Timestamp int64  `json:"timestamp"`
}

// KlineData represents a single candlestick
type KlineData struct {
	Time   int64   `json:"time"`
	Open   float64 `json:"open"`
	High   float64 `json:"high"`
	Low    float64 `json:"low"`
	Close  float64 `json:"close"`
	Volume float64 `json:"volume"`
}

// TickerData represents complete ticker information
type TickerData struct {
	MarketID    string `json:"market_id"`
	MarkPrice   string `json:"mark_price"`
	IndexPrice  string `json:"index_price"`
	LastPrice   string `json:"last_price"`
	High24h     string `json:"high_24h"`
	Low24h      string `json:"low_24h"`
	Volume24h   string `json:"volume_24h"`
	Change24h   string `json:"change_24h"`
	FundingRate string `json:"funding_rate"`
	NextFunding int64  `json:"next_funding"`
	Timestamp   int64  `json:"timestamp"`
}

// NewHyperliquidOracle creates a new oracle instance
func NewHyperliquidOracle() *HyperliquidOracle {
	return &HyperliquidOracle{
		apiURL: "https://api.hyperliquid.xyz/info",
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
		cache: make(map[string]*PriceCache),
	}
}

// assetToHL maps our market IDs to Hyperliquid asset names
var assetToHL = map[string]string{
	"BTC-USDC": "BTC",
	"ETH-USDC": "ETH",
	"SOL-USDC": "SOL",
}

// GetPrice fetches the current price from Hyperliquid
func (o *HyperliquidOracle) GetPrice(marketID string) (math.LegacyDec, error) {
	o.mu.RLock()
	cached, exists := o.cache[marketID]
	o.mu.RUnlock()

	// Use cache if less than 1 second old
	if exists && time.Since(cached.Timestamp) < time.Second {
		return cached.Price, nil
	}

	hlAsset, ok := assetToHL[marketID]
	if !ok {
		return math.LegacyZeroDec(), fmt.Errorf("unknown market: %s", marketID)
	}

	// Fetch from Hyperliquid API
	reqBody := fmt.Sprintf(`{"type": "metaAndAssetCtxs"}`)
	resp, err := o.httpClient.Post(o.apiURL, "application/json",
		io.NopCloser(strings.NewReader(reqBody)))
	if err != nil {
		// Return cached price on error
		if exists {
			return cached.Price, nil
		}
		return math.LegacyZeroDec(), err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		if exists {
			return cached.Price, nil
		}
		return math.LegacyZeroDec(), err
	}

	// Parse response
	var result []interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		if exists {
			return cached.Price, nil
		}
		return math.LegacyZeroDec(), err
	}

	// Find the asset in response by matching universe index
	if len(result) >= 2 {
		meta, ok := result[0].(map[string]interface{})
		if !ok {
			if exists {
				return cached.Price, nil
			}
			return math.LegacyZeroDec(), fmt.Errorf("invalid meta format")
		}

		universe, ok := meta["universe"].([]interface{})
		if !ok {
			if exists {
				return cached.Price, nil
			}
			return math.LegacyZeroDec(), fmt.Errorf("invalid universe format")
		}

		assetCtxs, ok := result[1].([]interface{})
		if !ok {
			if exists {
				return cached.Price, nil
			}
			return math.LegacyZeroDec(), fmt.Errorf("invalid assetCtxs format")
		}

		// Find the asset index in universe
		for i, u := range universe {
			uMap, ok := u.(map[string]interface{})
			if !ok {
				continue
			}
			name, _ := uMap["name"].(string)
			if name == hlAsset && i < len(assetCtxs) {
				ctxMap, ok := assetCtxs[i].(map[string]interface{})
				if !ok {
					continue
				}
				if markPx, ok := ctxMap["markPx"].(string); ok {
					price, err := math.LegacyNewDecFromStr(markPx)
					if err == nil {
						o.mu.Lock()
						o.cache[marketID] = &PriceCache{
							Price:     price,
							Timestamp: time.Now(),
						}
						o.mu.Unlock()
						return price, nil
					}
				}
			}
		}
	}

	// Fallback to cached price
	if exists {
		return cached.Price, nil
	}
	return math.LegacyZeroDec(), fmt.Errorf("price not found for %s", marketID)
}

// GetTicker fetches complete ticker data from Hyperliquid
func (o *HyperliquidOracle) GetTicker(marketID string) (*TickerData, error) {
	hlAsset, ok := assetToHL[marketID]
	if !ok {
		return nil, fmt.Errorf("unknown market: %s", marketID)
	}

	// Fetch metaAndAssetCtxs for comprehensive data
	reqBody := `{"type": "metaAndAssetCtxs"}`
	resp, err := o.httpClient.Post(o.apiURL, "application/json",
		io.NopCloser(strings.NewReader(reqBody)))
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var result []interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if len(result) < 2 {
		return nil, fmt.Errorf("invalid response format")
	}

	assetCtxs, ok := result[1].([]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid asset contexts format")
	}

	for _, ctx := range assetCtxs {
		ctxMap, ok := ctx.(map[string]interface{})
		if !ok {
			continue
		}
		// Note: Hyperliquid uses numeric index, not name
		if name, hasName := ctxMap["name"].(string); hasName && name == hlAsset {
			// Found our asset
			markPx := getStringValue(ctxMap, "markPx", "0")
			oraclePx := getStringValue(ctxMap, "oraclePx", markPx)
			midPx := getStringValue(ctxMap, "midPx", markPx)
			funding := getStringValue(ctxMap, "funding", "0")
			dayNtlVlm := getStringValue(ctxMap, "dayNtlVlm", "0")

			return &TickerData{
				MarketID:    marketID,
				MarkPrice:   markPx,
				IndexPrice:  oraclePx,
				LastPrice:   midPx,
				High24h:     markPx, // Will calculate from klines if needed
				Low24h:      markPx,
				Volume24h:   dayNtlVlm,
				Change24h:   "0.00", // Will calculate from klines if needed
				FundingRate: funding,
				NextFunding: time.Now().Truncate(time.Hour).Add(time.Hour).Unix(),
				Timestamp:   time.Now().UnixMilli(),
			}, nil
		}
	}

	// Try finding by index based on universe order
	meta, ok := result[0].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("price not found for %s", marketID)
	}
	universe, ok := meta["universe"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("price not found for %s", marketID)
	}

	for i, u := range universe {
		uMap, ok := u.(map[string]interface{})
		if !ok {
			continue
		}
		if name, _ := uMap["name"].(string); name == hlAsset {
			if i < len(assetCtxs) {
				ctxMap := assetCtxs[i].(map[string]interface{})
				markPx := getStringValue(ctxMap, "markPx", "0")
				oraclePx := getStringValue(ctxMap, "oraclePx", markPx)
				midPx := getStringValue(ctxMap, "midPx", markPx)
				funding := getStringValue(ctxMap, "funding", "0")
				dayNtlVlm := getStringValue(ctxMap, "dayNtlVlm", "0")

				return &TickerData{
					MarketID:    marketID,
					MarkPrice:   markPx,
					IndexPrice:  oraclePx,
					LastPrice:   midPx,
					High24h:     markPx,
					Low24h:      markPx,
					Volume24h:   dayNtlVlm,
					Change24h:   "0.00",
					FundingRate: funding,
					NextFunding: time.Now().Truncate(time.Hour).Add(time.Hour).Unix(),
					Timestamp:   time.Now().UnixMilli(),
				}, nil
			}
		}
	}

	return nil, fmt.Errorf("price not found for %s", marketID)
}

// GetOrderbook fetches L2 orderbook from Hyperliquid
func (o *HyperliquidOracle) GetOrderbook(marketID string, depth int) (*OrderbookData, error) {
	hlAsset, ok := assetToHL[marketID]
	if !ok {
		return nil, fmt.Errorf("unknown market: %s", marketID)
	}

	reqBody := fmt.Sprintf(`{"type":"l2Book","coin":"%s"}`, hlAsset)
	resp, err := o.httpClient.Post(o.apiURL, "application/json",
		io.NopCloser(strings.NewReader(reqBody)))
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	levels, ok := result["levels"].([]interface{})
	if !ok || len(levels) < 2 {
		return nil, fmt.Errorf("invalid orderbook format")
	}

	// levels[0] = bids, levels[1] = asks
	bidsRaw, _ := levels[0].([]interface{})
	asksRaw, _ := levels[1].([]interface{})

	bids := make([]OrderbookLevel, 0, depth)
	asks := make([]OrderbookLevel, 0, depth)

	for i, b := range bidsRaw {
		if i >= depth {
			break
		}
		bMap, ok := b.(map[string]interface{})
		if !ok {
			continue
		}
		bids = append(bids, OrderbookLevel{
			Price:    getStringValue(bMap, "px", "0"),
			Quantity: getStringValue(bMap, "sz", "0"),
		})
	}

	for i, a := range asksRaw {
		if i >= depth {
			break
		}
		aMap, ok := a.(map[string]interface{})
		if !ok {
			continue
		}
		asks = append(asks, OrderbookLevel{
			Price:    getStringValue(aMap, "px", "0"),
			Quantity: getStringValue(aMap, "sz", "0"),
		})
	}

	return &OrderbookData{
		MarketID:  marketID,
		Bids:      bids,
		Asks:      asks,
		Timestamp: time.Now().UnixMilli(),
	}, nil
}

// GetRecentTrades fetches recent trades from Hyperliquid
func (o *HyperliquidOracle) GetRecentTrades(marketID string, limit int) ([]TradeData, error) {
	hlAsset, ok := assetToHL[marketID]
	if !ok {
		return nil, fmt.Errorf("unknown market: %s", marketID)
	}

	reqBody := fmt.Sprintf(`{"type":"recentTrades","coin":"%s"}`, hlAsset)
	resp, err := o.httpClient.Post(o.apiURL, "application/json",
		io.NopCloser(strings.NewReader(reqBody)))
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var tradesRaw []interface{}
	if err := json.Unmarshal(body, &tradesRaw); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	trades := make([]TradeData, 0, limit)
	for i, t := range tradesRaw {
		if i >= limit {
			break
		}
		tMap, ok := t.(map[string]interface{})
		if !ok {
			continue
		}

		side := "buy"
		if s, ok := tMap["side"].(string); ok && s == "A" {
			side = "sell"
		}

		ts := time.Now().UnixMilli()
		if tsFloat, ok := tMap["time"].(float64); ok {
			ts = int64(tsFloat)
		}

		trades = append(trades, TradeData{
			TradeID:   fmt.Sprintf("T%d", ts),
			MarketID:  marketID,
			Price:     getStringValue(tMap, "px", "0"),
			Quantity:  getStringValue(tMap, "sz", "0"),
			Side:      side,
			Timestamp: ts,
		})
	}

	return trades, nil
}

// GetKlines fetches candlestick data from Hyperliquid
func (o *HyperliquidOracle) GetKlines(marketID, interval string, limit int) ([]KlineData, error) {
	hlAsset, ok := assetToHL[marketID]
	if !ok {
		return nil, fmt.Errorf("unknown market: %s", marketID)
	}

	// Calculate time range
	endTime := time.Now()
	var duration time.Duration
	switch interval {
	case "1m":
		duration = time.Minute
	case "5m":
		duration = 5 * time.Minute
	case "15m":
		duration = 15 * time.Minute
	case "30m":
		duration = 30 * time.Minute
	case "1h":
		duration = time.Hour
	case "4h":
		duration = 4 * time.Hour
	case "1d":
		duration = 24 * time.Hour
	default:
		duration = time.Hour
		interval = "1h"
	}
	startTime := endTime.Add(-duration * time.Duration(limit))

	reqBody := fmt.Sprintf(`{"type":"candleSnapshot","req":{"coin":"%s","interval":"%s","startTime":%d,"endTime":%d}}`,
		hlAsset, interval, startTime.UnixMilli(), endTime.UnixMilli())

	resp, err := o.httpClient.Post(o.apiURL, "application/json",
		io.NopCloser(strings.NewReader(reqBody)))
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var candlesRaw []interface{}
	if err := json.Unmarshal(body, &candlesRaw); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	klines := make([]KlineData, 0, len(candlesRaw))
	for _, c := range candlesRaw {
		cMap, ok := c.(map[string]interface{})
		if !ok {
			continue
		}

		ts := int64(0)
		if t, ok := cMap["t"].(float64); ok {
			ts = int64(t) / 1000 // Convert to seconds
		}

		klines = append(klines, KlineData{
			Time:   ts,
			Open:   getFloatValue(cMap, "o", 0),
			High:   getFloatValue(cMap, "h", 0),
			Low:    getFloatValue(cMap, "l", 0),
			Close:  getFloatValue(cMap, "c", 0),
			Volume: getFloatValue(cMap, "v", 0),
		})
	}

	// Limit results
	if len(klines) > limit {
		klines = klines[len(klines)-limit:]
	}

	return klines, nil
}

// Helper function to get string value from map
func getStringValue(m map[string]interface{}, key, defaultVal string) string {
	if v, ok := m[key]; ok {
		switch val := v.(type) {
		case string:
			return val
		case float64:
			return fmt.Sprintf("%.8f", val)
		}
	}
	return defaultVal
}

// Helper function to get float value from map
func getFloatValue(m map[string]interface{}, key string, defaultVal float64) float64 {
	if v, ok := m[key]; ok {
		switch val := v.(type) {
		case float64:
			return val
		case string:
			if f, err := parseFloat(val); err == nil {
				return f
			}
		}
	}
	return defaultVal
}

// parseFloat parses a string to float64
func parseFloat(s string) (float64, error) {
	var f float64
	_, err := fmt.Sscanf(s, "%f", &f)
	return f, err
}

// MemoryBankKeeper implements a real in-memory bank keeper for standalone mode
// Tracks actual balances and enforces real transfers
type MemoryBankKeeper struct {
	balances map[string]map[string]math.LegacyDec // address -> denom -> amount
	modules  map[string]map[string]math.LegacyDec // module -> denom -> amount
	mu       sync.RWMutex
}

func NewMemoryBankKeeper() *MemoryBankKeeper {
	return &MemoryBankKeeper{
		balances: make(map[string]map[string]math.LegacyDec),
		modules:  make(map[string]map[string]math.LegacyDec),
	}
}

// InitializeAccount sets initial balance for an account
func (b *MemoryBankKeeper) InitializeAccount(addr string, denom string, amount math.LegacyDec) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.balances[addr] == nil {
		b.balances[addr] = make(map[string]math.LegacyDec)
	}
	b.balances[addr][denom] = amount
}

// GetBalance returns the balance for an address and denom
func (b *MemoryBankKeeper) GetBalance(addr string, denom string) math.LegacyDec {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if b.balances[addr] == nil {
		return math.LegacyZeroDec()
	}
	bal, ok := b.balances[addr][denom]
	if !ok {
		return math.LegacyZeroDec()
	}
	return bal
}

func (b *MemoryBankKeeper) SendCoinsFromAccountToModule(ctx context.Context, senderAddr sdk.AccAddress, recipientModule string, amt sdk.Coins) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	sender := senderAddr.String()
	if b.balances[sender] == nil {
		return fmt.Errorf("account %s not found", sender)
	}

	for _, coin := range amt {
		currentBal := b.balances[sender][coin.Denom]
		amtDec := math.LegacyNewDecFromInt(coin.Amount)
		if currentBal.LT(amtDec) {
			return fmt.Errorf("insufficient balance: have %s, need %s %s", currentBal.String(), amtDec.String(), coin.Denom)
		}
		b.balances[sender][coin.Denom] = currentBal.Sub(amtDec)

		// Add to module
		if b.modules[recipientModule] == nil {
			b.modules[recipientModule] = make(map[string]math.LegacyDec)
		}
		moduleBal := b.modules[recipientModule][coin.Denom]
		b.modules[recipientModule][coin.Denom] = moduleBal.Add(amtDec)
	}
	return nil
}

func (b *MemoryBankKeeper) SendCoinsFromModuleToAccount(ctx context.Context, senderModule string, recipientAddr sdk.AccAddress, amt sdk.Coins) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	recipient := recipientAddr.String()
	if b.modules[senderModule] == nil {
		return fmt.Errorf("module %s not found", senderModule)
	}

	for _, coin := range amt {
		moduleBal := b.modules[senderModule][coin.Denom]
		amtDec := math.LegacyNewDecFromInt(coin.Amount)
		if moduleBal.LT(amtDec) {
			return fmt.Errorf("insufficient module balance")
		}
		b.modules[senderModule][coin.Denom] = moduleBal.Sub(amtDec)

		// Add to account
		if b.balances[recipient] == nil {
			b.balances[recipient] = make(map[string]math.LegacyDec)
		}
		b.balances[recipient][coin.Denom] = b.balances[recipient][coin.Denom].Add(amtDec)
	}
	return nil
}

// RealPerpetualKeeper wraps perpetual.Keeper for the orderbook interface
type RealPerpetualKeeper struct {
	keeper        *perpkeeper.Keeper
	marginChecker *perpkeeper.MarginChecker
	posManager    *perpkeeper.PositionManager
	oracle        *HyperliquidOracle
	sdkCtx        sdk.Context
	mu            sync.RWMutex
}

func (rpk *RealPerpetualKeeper) GetMarket(ctx sdk.Context, marketID string) *obkeeper.Market {
	market := rpk.keeper.GetMarket(ctx, marketID)
	if market == nil {
		return nil
	}
	return &obkeeper.Market{
		MarketID:      market.MarketID,
		TakerFeeRate:  market.TakerFeeRate,
		MakerFeeRate:  market.MakerFeeRate,
		InitialMargin: market.InitialMarginRate,
	}
}

func (rpk *RealPerpetualKeeper) GetMarkPrice(ctx sdk.Context, marketID string) (math.LegacyDec, bool) {
	// First try oracle
	if rpk.oracle != nil {
		price, err := rpk.oracle.GetPrice(marketID)
		if err == nil && !price.IsZero() {
			return price, true
		}
	}
	// Fallback to stored price
	priceInfo := rpk.keeper.GetPrice(ctx, marketID)
	if priceInfo != nil {
		return priceInfo.MarkPrice, true
	}
	return math.LegacyZeroDec(), false
}

func (rpk *RealPerpetualKeeper) UpdatePosition(ctx sdk.Context, trader, marketID string, side obtypes.Side, qty, price, fee interface{}) error {
	rpk.mu.Lock()
	defer rpk.mu.Unlock()

	qtyDec := qty.(math.LegacyDec)
	priceDec := price.(math.LegacyDec)

	// Convert orderbook side to perpetual side
	posSide := perptypes.PositionSideLong
	if side == obtypes.SideSell {
		posSide = perptypes.PositionSideShort
	}

	// Use real PositionManager to open position
	_, err := rpk.posManager.OpenPosition(ctx, trader, marketID, posSide, qtyDec, priceDec)
	return err
}

func (rpk *RealPerpetualKeeper) CheckMarginRequirement(ctx sdk.Context, trader, marketID string, side obtypes.Side, qty, price interface{}) error {
	qtyDec := qty.(math.LegacyDec)
	priceDec := price.(math.LegacyDec)

	// Use real MarginChecker
	return rpk.marginChecker.CheckInitialMarginRequirement(ctx, trader, marketID, qtyDec, priceDec)
}

// NewRealServiceV2 creates a new real E2E service
func NewRealServiceV2(logger log.Logger) (*RealServiceV2, error) {
	// Create in-memory database
	db := dbm.NewMemDB()

	// Create store keys
	obStoreKey := storetypes.NewKVStoreKey("orderbook")
	perpStoreKey := storetypes.NewKVStoreKey("perpetual")

	// Create multi-store with proper metrics
	cms := store.NewCommitMultiStore(db, logger, metrics.NewNoOpMetrics())
	cms.MountStoreWithDB(obStoreKey, storetypes.StoreTypeIAVL, db)
	cms.MountStoreWithDB(perpStoreKey, storetypes.StoreTypeIAVL, db)
	if err := cms.LoadLatestVersion(); err != nil {
		return nil, fmt.Errorf("failed to load store: %w", err)
	}

	// Create codec
	interfaceRegistry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(interfaceRegistry)

	// Create bank keeper (real in-memory implementation)
	bankKeeper := NewMemoryBankKeeper()

	// Create REAL perpetual keeper
	perpKeeper := perpkeeper.NewKeeper(cdc, perpStoreKey, bankKeeper, "", logger)

	// Create margin checker and position manager
	marginChecker := perpkeeper.NewMarginChecker(perpKeeper)
	positionManager := perpkeeper.NewPositionManager(perpKeeper)

	// Create oracle
	oracle := NewHyperliquidOracle()

	// Create SDK context
	header := tmproto.Header{Height: 1, Time: time.Now()}
	sdkCtx := sdk.NewContext(cms, header, false, logger)

	// Initialize default markets in perpetual keeper
	initializeMarkets(perpKeeper, sdkCtx)

	// Create real perpetual keeper adapter for orderbook
	realPerpKeeper := &RealPerpetualKeeper{
		keeper:        perpKeeper,
		marginChecker: marginChecker,
		posManager:    positionManager,
		oracle:        oracle,
		sdkCtx:        sdkCtx,
	}

	// Create orderbook keeper with REAL perpetual keeper
	obKeeper := obkeeper.NewKeeper(cdc, obStoreKey, realPerpKeeper, logger)

	// Create matching engine
	matchEngine := obkeeper.NewMatchingEngineV2(obKeeper)

	service := &RealServiceV2{
		perpKeeper:      perpKeeper,
		obKeeper:        obKeeper,
		marginChecker:   marginChecker,
		positionManager: positionManager,
		matchEngine:     matchEngine,
		bankKeeper:      bankKeeper,
		sdkCtx:          sdkCtx,
		cms:             cms,
		storeKey:        obStoreKey,
		perpKey:         perpStoreKey,
		oracle:          oracle,
		logger:          logger,
	}

	return service, nil
}

// initializeMarkets creates default markets with real parameters
func initializeMarkets(keeper *perpkeeper.Keeper, ctx sdk.Context) {
	markets := []struct {
		id            string
		takerFee      string
		makerFee      string
		initMargin    string
		maintMargin   string
		maxLeverage   string
	}{
		{"BTC-USDC", "0.0006", "0.0001", "0.05", "0.025", "20"},  // 5% init, 2.5% maint, 20x max
		{"ETH-USDC", "0.0006", "0.0001", "0.05", "0.025", "20"},
		{"SOL-USDC", "0.001", "0.0002", "0.10", "0.05", "10"},    // 10% init, 5% maint, 10x max
	}

	for _, m := range markets {
		takerFee, _ := math.LegacyNewDecFromStr(m.takerFee)
		makerFee, _ := math.LegacyNewDecFromStr(m.makerFee)
		initMargin, _ := math.LegacyNewDecFromStr(m.initMargin)
		maintMargin, _ := math.LegacyNewDecFromStr(m.maintMargin)
		maxLeverage, _ := math.LegacyNewDecFromStr(m.maxLeverage)

		market := &perptypes.Market{
			MarketID:              m.id,
			BaseAsset:             m.id[:3],
			QuoteAsset:            "USDC",
			TakerFeeRate:          takerFee,
			MakerFeeRate:          makerFee,
			InitialMarginRate:     initMargin,
			MaintenanceMarginRate: maintMargin,
			MaxLeverage:           maxLeverage,
			IsActive:              true,
		}
		keeper.SetMarket(ctx, market)
	}
}

// InitializeTestAccount creates an account with EXACT specified balance for testing
// This SETS the balance (not adds to it) to ensure deterministic test behavior
func (rs *RealServiceV2) InitializeTestAccount(trader string, balance string) error {
	rs.mu.Lock()
	defer rs.mu.Unlock()

	balanceDec, err := math.LegacyNewDecFromStr(balance)
	if err != nil {
		return err
	}

	// Get or create account, then SET the balance to exact value
	// (GetOrCreateAccount may give initial balance, we override it)
	account := rs.perpKeeper.GetOrCreateAccount(rs.sdkCtx, trader)
	account.Balance = balanceDec // SET to exact value, not deposit/add
	account.LockedMargin = math.LegacyZeroDec() // Reset locked margin
	rs.perpKeeper.SetAccount(rs.sdkCtx, account)

	// Also initialize in MemoryBankKeeper for real fund transfers
	rs.bankKeeper.InitializeAccount(trader, "uusdc", balanceDec)

	return nil
}

// ============ OrderService Implementation ============

func (rs *RealServiceV2) PlaceOrder(ctx context.Context, req *types.PlaceOrderRequest) (*types.PlaceOrderResponse, error) {
	rs.mu.Lock()
	defer rs.mu.Unlock()

	// Parse price and quantity
	price, err := math.LegacyNewDecFromStr(req.Price)
	if err != nil {
		return nil, fmt.Errorf("invalid price: %w", err)
	}
	qty, err := math.LegacyNewDecFromStr(req.Quantity)
	if err != nil {
		return nil, fmt.Errorf("invalid quantity: %w", err)
	}

	// Ensure account exists with balance
	account := rs.perpKeeper.GetAccount(rs.sdkCtx, req.Trader)
	if account == nil {
		return nil, fmt.Errorf("account not found: %s (use InitializeTestAccount first)", req.Trader)
	}

	// Check margin requirement BEFORE placing order
	requiredMargin := rs.marginChecker.CalculateInitialMargin(qty, price)
	if !account.CanAfford(requiredMargin) {
		return nil, fmt.Errorf("insufficient margin: required %s, available %s",
			requiredMargin.String(), account.AvailableBalance().String())
	}

	// Lock the margin for this order
	account.LockMargin(requiredMargin)
	rs.perpKeeper.SetAccount(rs.sdkCtx, account)

	// Convert side and type
	side := obtypes.SideBuy
	if req.Side == "sell" {
		side = obtypes.SideSell
	}
	orderType := obtypes.OrderTypeLimit
	if req.Type == "market" {
		orderType = obtypes.OrderTypeMarket
	}

	// Place order through real Keeper
	order, matchResult, err := rs.obKeeper.PlaceOrder(rs.sdkCtx, req.Trader, req.MarketID, side, orderType, price, qty)
	if err != nil {
		return nil, fmt.Errorf("failed to place order: %w", err)
	}

	// Flush cache to persist changes
	rs.matchEngine.Flush(rs.sdkCtx)

	return rs.convertPlaceOrderResponse(order, matchResult), nil
}

func (rs *RealServiceV2) CancelOrder(ctx context.Context, trader, orderID string) (*types.CancelOrderResponse, error) {
	rs.mu.Lock()
	defer rs.mu.Unlock()

	order, err := rs.obKeeper.CancelOrder(rs.sdkCtx, trader, orderID)
	if err != nil {
		return nil, err
	}

	rs.matchEngine.Flush(rs.sdkCtx)

	return &types.CancelOrderResponse{
		Order:     rs.convertOrder(order),
		Cancelled: true,
	}, nil
}

func (rs *RealServiceV2) ModifyOrder(ctx context.Context, trader, orderID string, req *types.ModifyOrderRequest) (*types.ModifyOrderResponse, error) {
	rs.mu.Lock()
	defer rs.mu.Unlock()

	oldOrder := rs.obKeeper.GetOrder(rs.sdkCtx, orderID)
	if oldOrder == nil {
		return nil, fmt.Errorf("order not found: %s", orderID)
	}

	if oldOrder.Trader != trader {
		return nil, fmt.Errorf("unauthorized: order belongs to different trader")
	}

	if !oldOrder.IsActive() {
		return nil, fmt.Errorf("order is not active")
	}

	// Cancel old order
	_, err := rs.obKeeper.CancelOrder(rs.sdkCtx, trader, orderID)
	if err != nil {
		return nil, fmt.Errorf("failed to cancel old order: %w", err)
	}

	// Create new order with modified values
	price := oldOrder.Price
	qty := oldOrder.Quantity
	if req.Price != "" {
		price, _ = math.LegacyNewDecFromStr(req.Price)
	}
	if req.Quantity != "" {
		qty, _ = math.LegacyNewDecFromStr(req.Quantity)
	}

	// Place new order
	newOrder, matchResult, err := rs.obKeeper.PlaceOrder(rs.sdkCtx, trader, oldOrder.MarketID, oldOrder.Side, oldOrder.OrderType, price, qty)
	if err != nil {
		return nil, fmt.Errorf("failed to place new order: %w", err)
	}

	rs.matchEngine.Flush(rs.sdkCtx)

	return &types.ModifyOrderResponse{
		OldOrderID: orderID,
		Order:      rs.convertOrder(newOrder),
		Match:      rs.convertMatchResult(matchResult),
	}, nil
}

func (rs *RealServiceV2) GetOrders(ctx context.Context, trader string) ([]*types.Order, error) {
	rs.mu.RLock()
	defer rs.mu.RUnlock()

	orders := rs.obKeeper.GetOrdersByTrader(rs.sdkCtx, trader)
	result := make([]*types.Order, 0, len(orders))
	for _, order := range orders {
		result = append(result, rs.convertOrder(order))
	}
	return result, nil
}

func (rs *RealServiceV2) GetOrder(ctx context.Context, orderID string) (*types.Order, error) {
	rs.mu.RLock()
	defer rs.mu.RUnlock()

	order := rs.obKeeper.GetOrder(rs.sdkCtx, orderID)
	if order == nil {
		return nil, fmt.Errorf("order not found: %s", orderID)
	}
	return rs.convertOrder(order), nil
}

// ============ PositionService Implementation ============

func (rs *RealServiceV2) GetPositions(ctx context.Context, trader string) ([]*types.Position, error) {
	rs.mu.RLock()
	defer rs.mu.RUnlock()

	positions := rs.perpKeeper.GetPositionsByTrader(rs.sdkCtx, trader)
	result := make([]*types.Position, 0, len(positions))
	for _, pos := range positions {
		result = append(result, rs.convertPosition(pos))
	}
	return result, nil
}

func (rs *RealServiceV2) GetPosition(ctx context.Context, trader, marketID string) (*types.Position, error) {
	rs.mu.RLock()
	defer rs.mu.RUnlock()

	pos := rs.perpKeeper.GetPosition(rs.sdkCtx, trader, marketID)
	if pos == nil {
		return nil, fmt.Errorf("position not found")
	}
	return rs.convertPosition(pos), nil
}

func (rs *RealServiceV2) ClosePosition(ctx context.Context, trader, marketID string) (*types.ClosePositionResponse, error) {
	rs.mu.Lock()
	defer rs.mu.Unlock()

	// Get current mark price from oracle
	markPrice, err := rs.oracle.GetPrice(marketID)
	if err != nil {
		return nil, fmt.Errorf("failed to get mark price for %s: %w", marketID, err)
	}

	// Close position using real PositionManager
	realizedPnL, err := rs.positionManager.ClosePosition(rs.sdkCtx, trader, marketID, markPrice)
	if err != nil {
		return nil, err
	}

	return &types.ClosePositionResponse{
		MarketID:    marketID,
		RealizedPnl: realizedPnL.String(),
	}, nil
}

// ============ AccountService Implementation ============

func (rs *RealServiceV2) GetAccount(ctx context.Context, trader string) (*types.Account, error) {
	rs.mu.RLock()
	defer rs.mu.RUnlock()

	account := rs.perpKeeper.GetAccount(rs.sdkCtx, trader)
	if account == nil {
		return nil, fmt.Errorf("account not found: %s", trader)
	}
	return rs.convertAccount(account), nil
}

// GetAccountEquity returns equity information (uses Account type)
func (rs *RealServiceV2) GetAccountEquity(ctx context.Context, trader string) (*types.Account, error) {
	rs.mu.RLock()
	defer rs.mu.RUnlock()

	equity := rs.marginChecker.CalculateAccountEquity(rs.sdkCtx, trader)
	account := rs.perpKeeper.GetAccount(rs.sdkCtx, trader)
	if account == nil {
		return nil, fmt.Errorf("account not found")
	}

	return &types.Account{
		Trader:           trader,
		Balance:          account.Balance.String(),
		LockedMargin:     account.LockedMargin.String(),
		AvailableBalance: equity.String(),
		MarginMode:       account.MarginMode.String(),
	}, nil
}

// ============ Conversion Helpers ============

func (rs *RealServiceV2) convertOrder(order *obtypes.Order) *types.Order {
	return &types.Order{
		OrderID:   order.OrderID,
		Trader:    order.Trader,
		MarketID:  order.MarketID,
		Side:      order.Side.String(),
		Type:      order.OrderType.String(),
		Price:     order.Price.String(),
		Quantity:  order.Quantity.String(),
		FilledQty: order.FilledQty.String(),
		Status:    order.Status.String(),
		CreatedAt: order.CreatedAt.UnixMilli(),
		UpdatedAt: order.UpdatedAt.UnixMilli(),
	}
}

func (rs *RealServiceV2) convertPosition(pos *perptypes.Position) *types.Position {
	markPrice, _ := rs.oracle.GetPrice(pos.MarketID)
	unrealizedPnL := pos.CalculateUnrealizedPnL(markPrice)

	return &types.Position{
		Trader:        pos.Trader,
		MarketID:      pos.MarketID,
		Side:          pos.Side.String(),
		Size:          pos.Size.String(),
		EntryPrice:    pos.EntryPrice.String(),
		MarkPrice:     markPrice.String(),
		Margin:        pos.Margin.String(),
		UnrealizedPnl: unrealizedPnL.String(),
		MarginMode:    "isolated",
	}
}

func (rs *RealServiceV2) convertAccount(account *perptypes.Account) *types.Account {
	return &types.Account{
		Trader:       account.Trader,
		Balance:      account.Balance.String(),
		LockedMargin: account.LockedMargin.String(),
		MarginMode:   account.MarginMode.String(),
	}
}

func (rs *RealServiceV2) convertPlaceOrderResponse(order *obtypes.Order, match *obkeeper.MatchResult) *types.PlaceOrderResponse {
	return &types.PlaceOrderResponse{
		Order: rs.convertOrder(order),
		Match: rs.convertMatchResult(match),
	}
}

func (rs *RealServiceV2) convertMatchResult(match *obkeeper.MatchResult) *types.MatchResult {
	if match == nil {
		return &types.MatchResult{}
	}
	trades := make([]types.TradeInfo, 0, len(match.Trades))
	for _, t := range match.Trades {
		trades = append(trades, types.TradeInfo{
			TradeID:   t.TradeID,
			Price:     t.Price.String(),
			Quantity:  t.Quantity.String(),
			Timestamp: t.Timestamp.UnixMilli(),
		})
	}
	return &types.MatchResult{
		FilledQty:    match.FilledQty.String(),
		AvgPrice:     match.AvgPrice.String(),
		RemainingQty: match.RemainingQty.String(),
		Trades:       trades,
	}
}
