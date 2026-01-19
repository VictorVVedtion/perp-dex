# Phase 1: 架构分析

## 1. 现有架构概览

### 1.1 模块依赖图

```
                    ┌─────────────────┐
                    │     app.go      │
                    │  (BaseApp入口)   │
                    └────────┬────────┘
                             │
         ┌───────────────────┼───────────────────┐
         │                   │                   │
         ▼                   ▼                   ▼
┌─────────────────┐ ┌─────────────────┐ ┌─────────────────┐
│  x/orderbook    │ │  x/perpetual    │ │ x/clearinghouse │
│                 │ │                 │ │                 │
│ • 订单管理      │ │ • 市场配置      │ │ • 清算检查      │
│ • 订单簿维护    │◄──• 持仓管理      │◄──• 强制平仓      │
│ • 撮合引擎     │ │ • 账户管理      │ │ • 健康检查      │
│ • 交易生成     │ │ • 价格预言机    │ │                 │
└────────┬────────┘ └────────┬────────┘ └─────────────────┘
         │                   │
         └───────────────────┘
              双向依赖
```

### 1.2 核心数据结构

| 模块 | 结构体 | 存储键 | 说明 |
|------|--------|--------|------|
| perpetual | Market | `0x01:{marketID}` | 市场配置 |
| perpetual | Position | `0x02:{trader}:{marketID}` | 持仓 |
| perpetual | Account | `0x03:{trader}` | 账户 |
| perpetual | PriceInfo | `0x04:{marketID}` | 价格 |
| orderbook | Order | `0x01:{orderID}` | 订单 |
| orderbook | OrderBook | `0x02:{marketID}` | 订单簿 |
| orderbook | Trade | `0x03:{tradeID}` | 交易 |
| clearinghouse | Liquidation | `0x01:{liquidationID}` | 清算记录 |

### 1.3 现有功能状态

| 功能 | 状态 | 位置 | 说明 |
|------|------|------|------|
| 撮合引擎 | ✅ 优化完成 | `x/orderbook/keeper/matching_v2.go` | 334x 性能提升 |
| 限价单/市价单 | ✅ 已实现 | `x/orderbook/types/types.go` | OrderType |
| 持仓管理 | ✅ 已实现 | `x/perpetual/keeper/position.go` | CRUD操作 |
| 清算检查 | ✅ 基础实现 | `x/clearinghouse/keeper/liquidation.go` | 健康检查 |
| 多交易对 | ❌ 未实现 | - | 需新增 |
| 资金费率 | ❌ 未实现 | - | 需新增 |
| 高级订单 | ❌ 未实现 | - | 需扩展 |
| 全仓/逐仓 | ❌ 未实现 | - | 需扩展 |

---

## 2. 升级需求分析

### 2.1 多交易对支持 [P0]

**当前限制：**
- `InitDefaultMarket` 硬编码 BTC-USDC
- 订单簿按 marketID 隔离，结构已支持
- 价格预言机按 marketID 存储

**需要修改：**
| 文件 | 修改内容 |
|------|----------|
| `x/perpetual/keeper/keeper.go` | 添加 `CreateMarket` 治理方法 |
| `x/perpetual/types/msgs.go` | 新增 `MsgCreateMarket` |
| `x/perpetual/keeper/market.go` (新建) | 市场管理逻辑 |
| `proto/perpdex/perpetual/v1/tx.proto` | 新增 CreateMarket RPC |

**验收标准：**
- 可通过治理提案创建新市场
- 4 个交易对独立运行 (BTC, ETH, SOL, ARB)
- 各市场参数可独立配置

### 2.2 资金费率机制 [P0]

**需求规格：**
- 8小时结算周期 (00:00, 08:00, 16:00 UTC)
- 费率公式: `R = 0.03 × (mark - index) / index`
- 费率上限: ±0.1% (每8小时)
- 多头付空头 (正费率) / 空头付多头 (负费率)

**需要新建：**
| 文件 | 内容 |
|------|------|
| `x/perpetual/keeper/funding.go` | 资金费率计算和结算 |
| `x/perpetual/types/funding.go` | FundingRate 结构体 |
| `proto/perpdex/perpetual/v1/funding.proto` | Protobuf 定义 |

**需要修改：**
| 文件 | 修改内容 |
|------|----------|
| `x/perpetual/keeper/keeper.go` | FundingKeyPrefix, 定时结算钩子 |
| `app/app.go` | EndBlocker 添加资金费率结算 |

### 2.3 高级订单类型 [P1]

**需要支持：**
| 订单类型 | 触发条件 | 说明 |
|----------|----------|------|
| Stop Loss | `mark_price <= stop_price` (多) | 止损单 |
| Take Profit | `mark_price >= take_price` (多) | 止盈单 |
| Post-Only | 不吃单 | 仅做 Maker |
| Reduce-Only | 仅减仓 | 不增加持仓 |
| IOC | 立即成交或取消 | Immediate Or Cancel |
| FOK | 全部成交或取消 | Fill Or Kill |
| GTC | 一直有效 | Good Till Cancel |

**需要修改：**
| 文件 | 修改内容 |
|------|----------|
| `x/orderbook/types/types.go` | 扩展 OrderType, 添加 TimeInForce |
| `x/orderbook/keeper/matching_v2.go` | 支持条件单匹配逻辑 |
| `x/orderbook/keeper/conditional.go` (新建) | 条件单触发逻辑 |

### 2.4 全仓/逐仓保证金 [P1]

**模式区别：**
| 模式 | 保证金计算 | 清算影响 |
|------|------------|----------|
| 逐仓 (Isolated) | 每仓独立 | 单仓清算 |
| 全仓 (Cross) | 账户共享 | 全仓清算 |

**需要修改：**
| 文件 | 修改内容 |
|------|----------|
| `x/perpetual/types/types.go` | Account 添加 MarginMode |
| `x/perpetual/keeper/margin.go` | 全仓保证金计算逻辑 |
| `x/clearinghouse/keeper/liquidation.go` | 适配两种模式清算 |

---

## 3. 架构决策

### 3.1 设计原则

1. **向后兼容**: 现有 API 不变，新功能通过扩展实现
2. **最小侵入**: 优先新建文件，减少现有代码修改
3. **模块解耦**: 资金费率、条件单等独立模块
4. **配置驱动**: 市场参数通过配置而非硬编码

### 3.2 存储键规划

```go
// perpetual 模块新增
FundingKeyPrefix       = []byte{0x05}  // 资金费率记录
FundingHistoryPrefix   = []byte{0x06}  // 资金费率历史
NextFundingTimeKey     = []byte{0x07}  // 下次结算时间

// orderbook 模块新增
ConditionalOrderPrefix = []byte{0x06}  // 条件单
OrderTimeInForceKey    = []byte{0x07}  // 订单生命周期
```

### 3.3 模块接口扩展

```go
// PerpetualKeeper 新增方法
type PerpetualKeeper interface {
    // 现有方法...

    // 新增: 多市场
    CreateMarket(ctx sdk.Context, market *Market) error
    ListMarkets(ctx sdk.Context) []*Market

    // 新增: 资金费率
    CalculateFundingRate(ctx sdk.Context, marketID string) math.LegacyDec
    SettleFunding(ctx sdk.Context, marketID string) error
    GetFundingHistory(ctx sdk.Context, marketID string, limit int) []*FundingRecord

    // 新增: 保证金模式
    SetMarginMode(ctx sdk.Context, trader string, mode MarginMode) error
    GetMarginMode(ctx sdk.Context, trader string) MarginMode
    CalculateCrossMargin(ctx sdk.Context, trader string) math.LegacyDec
}

// OrderbookKeeper 新增方法
type OrderbookKeeper interface {
    // 现有方法...

    // 新增: 高级订单
    PlaceConditionalOrder(ctx sdk.Context, order *ConditionalOrder) error
    CheckConditionalOrders(ctx sdk.Context, marketID string, markPrice math.LegacyDec) []*Order
    CancelConditionalOrder(ctx sdk.Context, orderID string) error
}
```

---

## 4. 风险评估

### 4.1 技术风险

| 风险 | 等级 | 缓解措施 |
|------|------|----------|
| 资金费率计算精度 | 高 | 使用 LegacyDec，单元测试验证 |
| 条件单触发时序 | 中 | EndBlocker 顺序保证 |
| 全仓清算连锁 | 高 | 严格测试边界条件 |
| 存储迁移 | 低 | 新增存储键，无需迁移 |

### 4.2 依赖风险

| 依赖 | 风险 | 说明 |
|------|------|------|
| Cosmos SDK v0.50.6 | 低 | 稳定版本 |
| CometBFT v0.38.11 | 低 | 稳定版本 |
| cosmossdk.io/math | 低 | 标准精度库 |

---

## 5. 实施优先级

```
Week 1-2:  [P0] 多交易对支持
Week 3-4:  [P0] 资金费率机制
Week 5-6:  [P1] 高级订单类型
Week 7-8:  [P1] 全仓/逐仓保证金
```

---

## 下一步

→ Phase 2: 详细架构设计
