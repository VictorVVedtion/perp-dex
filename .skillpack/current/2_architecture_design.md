# Phase 2: 架构设计

## 1. 多交易对支持设计

### 1.1 Market 结构扩展

```go
// x/perpetual/types/types.go (扩展)

// MarketStatus 市场状态
type MarketStatus int

const (
    MarketStatusInactive MarketStatus = iota
    MarketStatusActive
    MarketStatusSettling  // 资金费率结算中
    MarketStatusPaused    // 暂停交易
)

// Market 扩展字段
type Market struct {
    // 现有字段
    MarketID              string
    BaseAsset             string
    QuoteAsset            string
    MaxLeverage           math.LegacyDec
    InitialMarginRate     math.LegacyDec
    MaintenanceMarginRate math.LegacyDec
    TakerFeeRate          math.LegacyDec
    MakerFeeRate          math.LegacyDec
    TickSize              math.LegacyDec
    LotSize               math.LegacyDec
    IsActive              bool

    // 新增字段
    Status           MarketStatus   // 市场状态
    MinOrderSize     math.LegacyDec // 最小订单量
    MaxOrderSize     math.LegacyDec // 最大订单量
    MaxPositionSize  math.LegacyDec // 最大持仓量
    FundingInterval  int64          // 资金费率结算间隔(秒)
    InsuranceFundID  string         // 保险基金ID
    CreatedAt        time.Time
    UpdatedAt        time.Time
}

// 默认市场配置
var DefaultMarketConfigs = map[string]struct {
    Base, Quote    string
    MaxLeverage    int64
    TickSize       string
    LotSize        string
    MinOrderSize   string
    MaxOrderSize   string
    MaxPositionSize string
}{
    "BTC-USDC": {"BTC", "USDC", 10, "0.1", "0.0001", "0.0001", "100", "1000"},
    "ETH-USDC": {"ETH", "USDC", 10, "0.01", "0.001", "0.001", "1000", "10000"},
    "SOL-USDC": {"SOL", "USDC", 10, "0.001", "0.01", "0.01", "10000", "100000"},
    "ARB-USDC": {"ARB", "USDC", 10, "0.0001", "0.1", "0.1", "100000", "1000000"},
}
```

### 1.2 市场管理 Keeper

```go
// x/perpetual/keeper/market.go (新建)

// CreateMarket 创建新市场
func (k *Keeper) CreateMarket(ctx sdk.Context, msg *types.MsgCreateMarket) error {
    // 检查权限 (仅治理)
    if msg.Authority != k.GetAuthority() {
        return types.ErrUnauthorized
    }

    // 检查市场是否已存在
    if k.GetMarket(ctx, msg.MarketID) != nil {
        return types.ErrMarketExists
    }

    // 创建市场
    market := &types.Market{
        MarketID:              msg.MarketID,
        BaseAsset:             msg.BaseAsset,
        QuoteAsset:            msg.QuoteAsset,
        MaxLeverage:           msg.MaxLeverage,
        InitialMarginRate:     msg.InitialMarginRate,
        MaintenanceMarginRate: msg.MaintenanceMarginRate,
        TakerFeeRate:          msg.TakerFeeRate,
        MakerFeeRate:          msg.MakerFeeRate,
        TickSize:              msg.TickSize,
        LotSize:               msg.LotSize,
        MinOrderSize:          msg.MinOrderSize,
        MaxOrderSize:          msg.MaxOrderSize,
        MaxPositionSize:       msg.MaxPositionSize,
        FundingInterval:       28800, // 8小时
        Status:                types.MarketStatusActive,
        CreatedAt:             ctx.BlockTime(),
        UpdatedAt:             ctx.BlockTime(),
    }

    k.SetMarket(ctx, market)

    // 初始化订单簿
    k.orderbookKeeper.InitOrderBook(ctx, msg.MarketID)

    // 设置初始价格
    k.SetPrice(ctx, types.NewPriceInfo(msg.MarketID, msg.InitialPrice))

    // Emit event
    ctx.EventManager().EmitEvent(
        sdk.NewEvent(
            "market_created",
            sdk.NewAttribute("market_id", msg.MarketID),
            sdk.NewAttribute("base_asset", msg.BaseAsset),
            sdk.NewAttribute("quote_asset", msg.QuoteAsset),
        ),
    )

    return nil
}

// ListActiveMarkets 列出所有活跃市场
func (k *Keeper) ListActiveMarkets(ctx sdk.Context) []*types.Market {
    markets := k.GetAllMarkets(ctx)
    var active []*types.Market
    for _, m := range markets {
        if m.Status == types.MarketStatusActive {
            active = append(active, m)
        }
    }
    return active
}

// UpdateMarket 更新市场参数
func (k *Keeper) UpdateMarket(ctx sdk.Context, msg *types.MsgUpdateMarket) error {
    if msg.Authority != k.GetAuthority() {
        return types.ErrUnauthorized
    }

    market := k.GetMarket(ctx, msg.MarketID)
    if market == nil {
        return types.ErrMarketNotFound
    }

    // 更新允许修改的字段
    if !msg.MaxLeverage.IsNil() {
        market.MaxLeverage = msg.MaxLeverage
    }
    if !msg.TakerFeeRate.IsNil() {
        market.TakerFeeRate = msg.TakerFeeRate
    }
    if !msg.MakerFeeRate.IsNil() {
        market.MakerFeeRate = msg.MakerFeeRate
    }

    market.UpdatedAt = ctx.BlockTime()
    k.SetMarket(ctx, market)

    return nil
}
```

---

## 2. 资金费率机制设计

### 2.1 数据结构

```go
// x/perpetual/types/funding.go (新建)

// FundingRate 资金费率
type FundingRate struct {
    MarketID    string
    Rate        math.LegacyDec // 费率 (可正可负)
    MarkPrice   math.LegacyDec
    IndexPrice  math.LegacyDec
    Timestamp   time.Time
}

// FundingPayment 资金费用支付记录
type FundingPayment struct {
    PaymentID  string
    Trader     string
    MarketID   string
    PositionID string
    Amount     math.LegacyDec // 正数表示收到，负数表示支付
    Rate       math.LegacyDec
    Timestamp  time.Time
}

// FundingConfig 资金费率配置
type FundingConfig struct {
    Interval      int64          // 结算间隔 (秒)
    MaxRate       math.LegacyDec // 最大费率
    MinRate       math.LegacyDec // 最小费率
    DampingFactor math.LegacyDec // 阻尼系数 (0.03)
}

// DefaultFundingConfig 默认配置
func DefaultFundingConfig() FundingConfig {
    return FundingConfig{
        Interval:      28800, // 8小时
        MaxRate:       math.LegacyNewDecWithPrec(1, 3),  // 0.1%
        MinRate:       math.LegacyNewDecWithPrec(-1, 3), // -0.1%
        DampingFactor: math.LegacyNewDecWithPrec(3, 2),  // 0.03
    }
}
```

### 2.2 资金费率 Keeper

```go
// x/perpetual/keeper/funding.go (新建)

// Store key prefixes
var (
    FundingRateKeyPrefix     = []byte{0x05}
    FundingPaymentKeyPrefix  = []byte{0x06}
    NextFundingTimeKeyPrefix = []byte{0x07}
    FundingConfigKeyPrefix   = []byte{0x08}
)

// CalculateFundingRate 计算资金费率
// 公式: R = dampingFactor × (markPrice - indexPrice) / indexPrice
func (k *Keeper) CalculateFundingRate(ctx sdk.Context, marketID string) math.LegacyDec {
    priceInfo := k.GetPrice(ctx, marketID)
    if priceInfo == nil {
        return math.LegacyZeroDec()
    }

    config := k.GetFundingConfig(ctx, marketID)

    // R = 0.03 × (mark - index) / index
    if priceInfo.IndexPrice.IsZero() {
        return math.LegacyZeroDec()
    }

    priceDiff := priceInfo.MarkPrice.Sub(priceInfo.IndexPrice)
    rate := config.DampingFactor.Mul(priceDiff).Quo(priceInfo.IndexPrice)

    // 限制在 [minRate, maxRate] 范围
    if rate.GT(config.MaxRate) {
        rate = config.MaxRate
    } else if rate.LT(config.MinRate) {
        rate = config.MinRate
    }

    return rate
}

// SettleFunding 结算资金费率
func (k *Keeper) SettleFunding(ctx sdk.Context, marketID string) error {
    logger := k.Logger()

    // 计算费率
    rate := k.CalculateFundingRate(ctx, marketID)
    priceInfo := k.GetPrice(ctx, marketID)

    // 保存费率记录
    fundingRate := &types.FundingRate{
        MarketID:   marketID,
        Rate:       rate,
        MarkPrice:  priceInfo.MarkPrice,
        IndexPrice: priceInfo.IndexPrice,
        Timestamp:  ctx.BlockTime(),
    }
    k.SetFundingRate(ctx, fundingRate)

    // 获取该市场所有持仓
    positions := k.GetPositionsByMarket(ctx, marketID)

    var totalLongSize, totalShortSize math.LegacyDec = math.LegacyZeroDec(), math.LegacyZeroDec()

    for _, pos := range positions {
        if pos.Side == types.PositionSideLong {
            totalLongSize = totalLongSize.Add(pos.Size)
        } else {
            totalShortSize = totalShortSize.Add(pos.Size)
        }
    }

    // 计算并转移资金
    for _, pos := range positions {
        // 资金费用 = 持仓价值 × 费率
        notional := pos.Size.Mul(priceInfo.MarkPrice)
        payment := notional.Mul(rate)

        // 多头: 正费率支付，负费率收取
        // 空头: 正费率收取，负费率支付
        if pos.Side == types.PositionSideLong {
            payment = payment.Neg() // 多头支付
        }

        // 更新账户余额
        account := k.GetOrCreateAccount(ctx, pos.Trader)
        account.Balance = account.Balance.Add(payment)
        k.SetAccount(ctx, account)

        // 记录支付
        k.SaveFundingPayment(ctx, &types.FundingPayment{
            PaymentID:  k.generatePaymentID(ctx),
            Trader:     pos.Trader,
            MarketID:   marketID,
            PositionID: pos.Trader + ":" + marketID,
            Amount:     payment,
            Rate:       rate,
            Timestamp:  ctx.BlockTime(),
        })
    }

    // 更新下次结算时间
    config := k.GetFundingConfig(ctx, marketID)
    nextTime := ctx.BlockTime().Add(time.Duration(config.Interval) * time.Second)
    k.SetNextFundingTime(ctx, marketID, nextTime)

    logger.Info("funding settled",
        "market_id", marketID,
        "rate", rate.String(),
        "positions_affected", len(positions),
    )

    // Emit event
    ctx.EventManager().EmitEvent(
        sdk.NewEvent(
            "funding_settled",
            sdk.NewAttribute("market_id", marketID),
            sdk.NewAttribute("rate", rate.String()),
            sdk.NewAttribute("timestamp", ctx.BlockTime().String()),
        ),
    )

    return nil
}

// FundingEndBlocker 在 EndBlock 检查并执行资金费率结算
func (k *Keeper) FundingEndBlocker(ctx sdk.Context) error {
    markets := k.ListActiveMarkets(ctx)

    for _, market := range markets {
        nextTime := k.GetNextFundingTime(ctx, market.MarketID)
        if ctx.BlockTime().After(nextTime) || ctx.BlockTime().Equal(nextTime) {
            if err := k.SettleFunding(ctx, market.MarketID); err != nil {
                k.Logger().Error("failed to settle funding",
                    "market_id", market.MarketID,
                    "error", err,
                )
            }
        }
    }

    return nil
}
```

---

## 3. 高级订单类型设计

### 3.1 订单类型扩展

```go
// x/orderbook/types/types.go (扩展)

// OrderType 扩展
const (
    OrderTypeUnspecified OrderType = iota
    OrderTypeLimit                  // 限价单
    OrderTypeMarket                 // 市价单
    OrderTypeStopLoss              // 止损单
    OrderTypeTakeProfit            // 止盈单
    OrderTypeStopLimit             // 止损限价单
    OrderTypeTakeProfitLimit       // 止盈限价单
)

// TimeInForce 订单生命周期
type TimeInForce int

const (
    TimeInForceGTC TimeInForce = iota // Good Till Cancel
    TimeInForceIOC                     // Immediate Or Cancel
    TimeInForceFOK                     // Fill Or Kill
    TimeInForceGTX                     // Post Only (Good Till Crossing)
)

// OrderFlags 订单标志
type OrderFlags struct {
    ReduceOnly bool // 仅减仓
    PostOnly   bool // 仅做 Maker
}

// Order 扩展
type Order struct {
    // 现有字段...
    OrderID     string
    Trader      string
    MarketID    string
    Side        Side
    OrderType   OrderType
    Price       math.LegacyDec
    Quantity    math.LegacyDec
    FilledQty   math.LegacyDec
    Status      OrderStatus
    CreatedAt   time.Time
    UpdatedAt   time.Time

    // 新增字段
    TimeInForce   TimeInForce    // 订单生命周期
    TriggerPrice  math.LegacyDec // 触发价格 (条件单)
    Flags         OrderFlags     // 订单标志
    ClientOrderID string         // 客户端订单ID
    TriggeredAt   *time.Time     // 触发时间
}

// ConditionalOrder 条件单
type ConditionalOrder struct {
    OrderID        string
    Trader         string
    MarketID       string
    Side           Side
    OrderType      OrderType      // StopLoss / TakeProfit
    TriggerPrice   math.LegacyDec // 触发价格
    ExecutionPrice math.LegacyDec // 执行价格 (限价单时使用)
    Quantity       math.LegacyDec
    Flags          OrderFlags
    Status         OrderStatus
    CreatedAt      time.Time
    TriggeredAt    *time.Time
}
```

### 3.2 条件单 Keeper

```go
// x/orderbook/keeper/conditional.go (新建)

// Store key prefix
var ConditionalOrderKeyPrefix = []byte{0x06}

// PlaceConditionalOrder 创建条件单
func (k *Keeper) PlaceConditionalOrder(ctx sdk.Context, order *types.ConditionalOrder) error {
    // 验证触发价格
    if order.TriggerPrice.IsNil() || order.TriggerPrice.IsZero() {
        return types.ErrInvalidTriggerPrice
    }

    // 生成订单ID
    order.OrderID = k.generateOrderID(ctx)
    order.Status = types.OrderStatusOpen
    order.CreatedAt = ctx.BlockTime()

    // 保存条件单
    k.SetConditionalOrder(ctx, order)

    // Emit event
    ctx.EventManager().EmitEvent(
        sdk.NewEvent(
            "conditional_order_placed",
            sdk.NewAttribute("order_id", order.OrderID),
            sdk.NewAttribute("order_type", order.OrderType.String()),
            sdk.NewAttribute("trigger_price", order.TriggerPrice.String()),
        ),
    )

    return nil
}

// CheckAndTriggerConditionalOrders 检查并触发条件单
func (k *Keeper) CheckAndTriggerConditionalOrders(ctx sdk.Context, marketID string, markPrice math.LegacyDec) []*types.Order {
    conditionalOrders := k.GetActiveConditionalOrders(ctx, marketID)
    triggeredOrders := make([]*types.Order, 0)

    for _, condOrder := range conditionalOrders {
        if !condOrder.IsActive() {
            continue
        }

        shouldTrigger := false

        switch condOrder.OrderType {
        case types.OrderTypeStopLoss:
            // 多头止损: mark <= trigger
            // 空头止损: mark >= trigger
            if condOrder.Side == types.SideSell {
                shouldTrigger = markPrice.LTE(condOrder.TriggerPrice)
            } else {
                shouldTrigger = markPrice.GTE(condOrder.TriggerPrice)
            }

        case types.OrderTypeTakeProfit:
            // 多头止盈: mark >= trigger
            // 空头止盈: mark <= trigger
            if condOrder.Side == types.SideSell {
                shouldTrigger = markPrice.GTE(condOrder.TriggerPrice)
            } else {
                shouldTrigger = markPrice.LTE(condOrder.TriggerPrice)
            }
        }

        if shouldTrigger {
            // 转换为普通市价单
            now := ctx.BlockTime()
            order := &types.Order{
                OrderID:     k.generateOrderID(ctx),
                Trader:      condOrder.Trader,
                MarketID:    condOrder.MarketID,
                Side:        condOrder.Side,
                OrderType:   types.OrderTypeMarket,
                Price:       condOrder.ExecutionPrice,
                Quantity:    condOrder.Quantity,
                Flags:       condOrder.Flags,
                Status:      types.OrderStatusOpen,
                CreatedAt:   now,
                TriggeredAt: &now,
            }

            // 标记条件单为已触发
            condOrder.Status = types.OrderStatusFilled
            condOrder.TriggeredAt = &now
            k.SetConditionalOrder(ctx, condOrder)

            triggeredOrders = append(triggeredOrders, order)

            // Emit event
            ctx.EventManager().EmitEvent(
                sdk.NewEvent(
                    "conditional_order_triggered",
                    sdk.NewAttribute("conditional_order_id", condOrder.OrderID),
                    sdk.NewAttribute("new_order_id", order.OrderID),
                    sdk.NewAttribute("trigger_price", condOrder.TriggerPrice.String()),
                    sdk.NewAttribute("mark_price", markPrice.String()),
                ),
            )
        }
    }

    return triggeredOrders
}

// ProcessTimeInForce 处理订单生命周期
func (k *Keeper) ProcessTimeInForce(ctx sdk.Context, order *types.Order, result *MatchResult) error {
    switch order.TimeInForce {
    case types.TimeInForceIOC:
        // Immediate Or Cancel: 未完全成交部分取消
        if !order.IsFilled() {
            order.Cancel()
            k.SetOrder(ctx, order)
        }

    case types.TimeInForceFOK:
        // Fill Or Kill: 必须全部成交，否则全部取消
        if !order.IsFilled() {
            // 回滚已成交部分 (需要复杂处理)
            return types.ErrFOKNotFilled
        }

    case types.TimeInForceGTX:
        // Post Only: 如果会立即成交则取消
        if result != nil && len(result.Trades) > 0 {
            order.Cancel()
            k.SetOrder(ctx, order)
            return types.ErrPostOnlyWouldTake
        }
    }

    return nil
}
```

---

## 4. 全仓/逐仓保证金设计

### 4.1 数据结构扩展

```go
// x/perpetual/types/types.go (扩展)

// MarginMode 保证金模式
type MarginMode int

const (
    MarginModeIsolated MarginMode = iota // 逐仓
    MarginModeCross                       // 全仓
)

// Account 扩展
type Account struct {
    // 现有字段
    Trader       string
    Balance      math.LegacyDec
    LockedMargin math.LegacyDec

    // 新增字段
    MarginMode     MarginMode     // 保证金模式
    CrossMarginPnL math.LegacyDec // 全仓模式未实现PnL
}

// Position 扩展
type Position struct {
    // 现有字段...
    Trader           string
    MarketID         string
    Side             PositionSide
    Size             math.LegacyDec
    EntryPrice       math.LegacyDec
    Margin           math.LegacyDec
    Leverage         math.LegacyDec
    LiquidationPrice math.LegacyDec
    OpenedAt         time.Time
    UpdatedAt        time.Time

    // 新增字段
    MarginMode       MarginMode     // 该持仓的保证金模式
    IsolatedMargin   math.LegacyDec // 逐仓模式下的独立保证金
}
```

### 4.2 保证金计算逻辑

```go
// x/perpetual/keeper/margin.go (新建/扩展)

// CalculateIsolatedMargin 计算逐仓保证金需求
func (k *Keeper) CalculateIsolatedMargin(position *types.Position, markPrice math.LegacyDec) *MarginInfo {
    notional := position.Size.Mul(markPrice)
    unrealizedPnL := position.CalculateUnrealizedPnL(markPrice)
    equity := position.Margin.Add(unrealizedPnL)

    market := k.GetMarket(ctx, position.MarketID)
    maintenanceMargin := notional.Mul(market.MaintenanceMarginRate)
    marginRatio := equity.Quo(notional)

    return &MarginInfo{
        Equity:            equity,
        MaintenanceMargin: maintenanceMargin,
        MarginRatio:       marginRatio,
        IsHealthy:         marginRatio.GTE(market.MaintenanceMarginRate),
    }
}

// CalculateCrossMargin 计算全仓保证金
func (k *Keeper) CalculateCrossMargin(ctx sdk.Context, trader string) *CrossMarginInfo {
    account := k.GetAccount(ctx, trader)
    positions := k.GetPositionsByTrader(ctx, trader)

    var totalNotional, totalUnrealizedPnL, totalMaintenanceMargin math.LegacyDec
    totalNotional = math.LegacyZeroDec()
    totalUnrealizedPnL = math.LegacyZeroDec()
    totalMaintenanceMargin = math.LegacyZeroDec()

    for _, pos := range positions {
        if pos.MarginMode != types.MarginModeCross {
            continue // 跳过逐仓持仓
        }

        priceInfo := k.GetPrice(ctx, pos.MarketID)
        market := k.GetMarket(ctx, pos.MarketID)

        notional := pos.Size.Mul(priceInfo.MarkPrice)
        pnl := pos.CalculateUnrealizedPnL(priceInfo.MarkPrice)
        maintenance := notional.Mul(market.MaintenanceMarginRate)

        totalNotional = totalNotional.Add(notional)
        totalUnrealizedPnL = totalUnrealizedPnL.Add(pnl)
        totalMaintenanceMargin = totalMaintenanceMargin.Add(maintenance)
    }

    // 全仓权益 = 账户余额 + 全部未实现PnL
    crossEquity := account.Balance.Add(totalUnrealizedPnL)

    // 全仓保证金率
    var crossMarginRatio math.LegacyDec
    if totalNotional.IsPositive() {
        crossMarginRatio = crossEquity.Quo(totalNotional)
    } else {
        crossMarginRatio = math.LegacyNewDec(1) // 无持仓时100%
    }

    return &CrossMarginInfo{
        Equity:               crossEquity,
        TotalNotional:        totalNotional,
        TotalUnrealizedPnL:   totalUnrealizedPnL,
        TotalMaintenanceMargin: totalMaintenanceMargin,
        MarginRatio:          crossMarginRatio,
        IsHealthy:            crossMarginRatio.GTE(math.LegacyNewDecWithPrec(5, 2)),
    }
}

// SetMarginMode 切换保证金模式
func (k *Keeper) SetMarginMode(ctx sdk.Context, trader string, mode types.MarginMode) error {
    account := k.GetAccount(ctx, trader)
    if account == nil {
        return types.ErrAccountNotFound
    }

    // 检查是否有未平仓持仓
    positions := k.GetPositionsByTrader(ctx, trader)
    if len(positions) > 0 {
        return types.ErrCannotChangeMarginModeWithPositions
    }

    account.MarginMode = mode
    k.SetAccount(ctx, account)

    ctx.EventManager().EmitEvent(
        sdk.NewEvent(
            "margin_mode_changed",
            sdk.NewAttribute("trader", trader),
            sdk.NewAttribute("mode", mode.String()),
        ),
    )

    return nil
}

// MarginInfo 保证金信息
type MarginInfo struct {
    Equity            math.LegacyDec
    MaintenanceMargin math.LegacyDec
    MarginRatio       math.LegacyDec
    IsHealthy         bool
}

// CrossMarginInfo 全仓保证金信息
type CrossMarginInfo struct {
    Equity                 math.LegacyDec
    TotalNotional          math.LegacyDec
    TotalUnrealizedPnL     math.LegacyDec
    TotalMaintenanceMargin math.LegacyDec
    MarginRatio            math.LegacyDec
    IsHealthy              bool
}
```

---

## 5. EndBlocker 执行顺序

```go
// app/app.go (修改)

func (app *App) EndBlocker(ctx sdk.Context) (sdk.EndBlock, error) {
    // 1. 价格预言机更新
    if err := app.PerpetualKeeper.UpdatePrices(ctx); err != nil {
        app.Logger().Error("failed to update prices", "error", err)
    }

    // 2. 条件单触发检查
    markets := app.PerpetualKeeper.ListActiveMarkets(ctx)
    for _, market := range markets {
        priceInfo := app.PerpetualKeeper.GetPrice(ctx, market.MarketID)
        if priceInfo != nil {
            triggeredOrders := app.OrderbookKeeper.CheckAndTriggerConditionalOrders(
                ctx, market.MarketID, priceInfo.MarkPrice,
            )
            // 执行触发的订单
            for _, order := range triggeredOrders {
                app.OrderbookKeeper.ProcessTriggeredOrder(ctx, order)
            }
        }
    }

    // 3. 订单匹配
    if err := app.OrderbookKeeper.EndBlocker(ctx); err != nil {
        app.Logger().Error("failed to match orders", "error", err)
    }

    // 4. 资金费率结算
    if err := app.PerpetualKeeper.FundingEndBlocker(ctx); err != nil {
        app.Logger().Error("failed to settle funding", "error", err)
    }

    // 5. 清算检查
    if err := app.ClearinghouseKeeper.LiquidationEndBlocker(ctx); err != nil {
        app.Logger().Error("failed to check liquidations", "error", err)
    }

    return sdk.EndBlock{}, nil
}
```

---

## 下一步

→ Phase 3: 实施规划
