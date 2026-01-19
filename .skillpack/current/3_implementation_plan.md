# Phase 3: 实施规划

## 1. 文件变更清单

### 1.1 新建文件 (15 个)

| 序号 | 文件路径 | 功能 | 优先级 | 依赖 |
|------|----------|------|--------|------|
| 1 | `x/perpetual/types/funding.go` | 资金费率类型定义 | P0 | - |
| 2 | `x/perpetual/types/market_status.go` | 市场状态枚举 | P0 | - |
| 3 | `x/perpetual/types/margin_mode.go` | 保证金模式枚举 | P1 | - |
| 4 | `x/perpetual/keeper/market.go` | 市场管理逻辑 | P0 | 1,2 |
| 5 | `x/perpetual/keeper/funding.go` | 资金费率计算与结算 | P0 | 1 |
| 6 | `x/perpetual/keeper/margin_mode.go` | 全仓/逐仓保证金 | P1 | 3 |
| 7 | `x/orderbook/types/order_extended.go` | 扩展订单类型 | P1 | - |
| 8 | `x/orderbook/keeper/conditional.go` | 条件单逻辑 | P1 | 7 |
| 9 | `x/orderbook/keeper/time_in_force.go` | 订单生命周期处理 | P1 | 7 |
| 10 | `proto/perpdex/perpetual/v1/funding.proto` | 资金费率 Protobuf | P0 | - |
| 11 | `proto/perpdex/perpetual/v1/market_msgs.proto` | 市场管理消息 | P0 | - |
| 12 | `x/perpetual/keeper/funding_test.go` | 资金费率测试 | P0 | 5 |
| 13 | `x/orderbook/keeper/conditional_test.go` | 条件单测试 | P1 | 8 |
| 14 | `x/perpetual/keeper/margin_mode_test.go` | 保证金模式测试 | P1 | 6 |
| 15 | `x/perpetual/keeper/market_test.go` | 市场管理测试 | P0 | 4 |

### 1.2 修改文件 (12 个)

| 序号 | 文件路径 | 修改内容 | 优先级 |
|------|----------|----------|--------|
| 1 | `x/perpetual/types/types.go` | 扩展 Market, Account, Position | P0 |
| 2 | `x/perpetual/types/errors.go` | 新增错误定义 | P0 |
| 3 | `x/perpetual/types/msgs.go` | 新增 MsgCreateMarket 等 | P0 |
| 4 | `x/perpetual/keeper/keeper.go` | 新增存储键前缀 | P0 |
| 5 | `x/orderbook/types/types.go` | 扩展 Order, 新增 TimeInForce | P1 |
| 6 | `x/orderbook/types/errors.go` | 新增错误定义 | P1 |
| 7 | `x/orderbook/keeper/keeper.go` | 条件单存储键 | P1 |
| 8 | `x/orderbook/keeper/matching_v2.go` | 支持高级订单匹配 | P1 |
| 9 | `x/clearinghouse/keeper/liquidation.go` | 适配两种保证金模式 | P1 |
| 10 | `app/app.go` | EndBlocker 顺序调整 | P0 |
| 11 | `proto/perpdex/perpetual/v1/types.proto` | 扩展类型定义 | P0 |
| 12 | `proto/perpdex/orderbook/v1/types.proto` | 扩展订单类型 | P1 |

---

## 2. 实施顺序

### 阶段 4.1: 多交易对支持 [P0] - Week 1-2

```
Day 1-2: 类型定义
├── x/perpetual/types/market_status.go
├── x/perpetual/types/types.go (扩展 Market)
└── x/perpetual/types/errors.go (新增错误)

Day 3-4: Keeper 实现
├── x/perpetual/keeper/keeper.go (存储键)
└── x/perpetual/keeper/market.go (市场管理)

Day 5-6: Protobuf 和消息
├── proto/perpdex/perpetual/v1/market_msgs.proto
├── x/perpetual/types/msgs.go (MsgCreateMarket)
└── make proto (生成代码)

Day 7-8: 测试和集成
├── x/perpetual/keeper/market_test.go
├── 初始化 4 个默认市场
└── 集成测试
```

### 阶段 4.2: 资金费率机制 [P0] - Week 3-4

```
Day 1-2: 类型定义
├── x/perpetual/types/funding.go
└── proto/perpdex/perpetual/v1/funding.proto

Day 3-5: Keeper 实现
├── x/perpetual/keeper/funding.go
│   ├── CalculateFundingRate()
│   ├── SettleFunding()
│   └── FundingEndBlocker()
└── x/perpetual/keeper/keeper.go (存储键)

Day 6-7: 集成
├── app/app.go (EndBlocker 添加资金费率)
└── 查询接口实现

Day 8-10: 测试
├── x/perpetual/keeper/funding_test.go
├── 24小时模拟测试
└── 边界条件测试
```

### 阶段 4.3: 高级订单类型 [P1] - Week 5-6

```
Day 1-2: 类型定义
├── x/orderbook/types/order_extended.go
│   ├── TimeInForce
│   ├── OrderFlags
│   └── ConditionalOrder
└── x/orderbook/types/errors.go

Day 3-5: Keeper 实现
├── x/orderbook/keeper/conditional.go
│   ├── PlaceConditionalOrder()
│   └── CheckAndTriggerConditionalOrders()
└── x/orderbook/keeper/time_in_force.go
    ├── ProcessIOC()
    ├── ProcessFOK()
    └── ProcessPostOnly()

Day 6-7: 匹配引擎适配
├── x/orderbook/keeper/matching_v2.go
│   ├── 支持 ReduceOnly
│   └── 支持 PostOnly 检查
└── x/orderbook/keeper/keeper.go (条件单存储)

Day 8-10: 测试
├── x/orderbook/keeper/conditional_test.go
└── 各订单类型单元测试
```

### 阶段 4.4: 全仓/逐仓保证金 [P1] - Week 7-8

```
Day 1-2: 类型定义
├── x/perpetual/types/margin_mode.go
└── x/perpetual/types/types.go (扩展 Account, Position)

Day 3-5: Keeper 实现
├── x/perpetual/keeper/margin_mode.go
│   ├── CalculateIsolatedMargin()
│   ├── CalculateCrossMargin()
│   └── SetMarginMode()
└── x/perpetual/keeper/margin.go (更新)

Day 6-7: 清算适配
└── x/clearinghouse/keeper/liquidation.go
    ├── 逐仓清算逻辑
    └── 全仓清算逻辑

Day 8-10: 测试
├── x/perpetual/keeper/margin_mode_test.go
└── 模式切换测试
```

---

## 3. 验证计划

### 3.1 单元测试

| 功能 | 测试文件 | 测试用例 |
|------|----------|----------|
| 多交易对 | `market_test.go` | 创建/更新/列表市场 |
| 资金费率 | `funding_test.go` | 费率计算/结算/上下限 |
| 条件单 | `conditional_test.go` | 止损/止盈触发 |
| 订单生命周期 | `time_in_force_test.go` | IOC/FOK/GTX |
| 保证金模式 | `margin_mode_test.go` | 逐仓/全仓切换和计算 |

### 3.2 集成测试

```bash
# 多市场测试
make test-markets
# 验证: 4个交易对独立运行

# 资金费率测试
make test-funding
# 验证: 24小时模拟，费率正确结算

# 高级订单测试
make test-orders
# 验证: 所有订单类型正确执行

# 保证金模式测试
make test-margin
# 验证: 模式切换，PnL计算正确
```

### 3.3 性能测试

| 测试项 | 目标 | 方法 |
|--------|------|------|
| 多市场撮合 | 4市场同时 >10K TPS | benchmark_test.go |
| 条件单触发 | <10ms 延迟 | 时间戳测量 |
| 资金费率结算 | 10K持仓 <1s | 批量测试 |

---

## 4. 依赖图

```
┌──────────────────────────────────────────────────────────┐
│                    Phase 4 实施依赖                       │
└──────────────────────────────────────────────────────────┘

4.1 多交易对支持 ─────┐
                      │
4.2 资金费率机制 ─────┼──→ 4.4 全仓/逐仓保证金
                      │         │
4.3 高级订单类型 ─────┘         │
                                │
                                ▼
                        Phase 5 验收审查
```

---

## 5. 风险控制

### 5.1 回滚策略

每个阶段完成后创建 Git Tag:
```bash
git tag v0.2.0-multi-market  # 4.1 完成
git tag v0.2.1-funding       # 4.2 完成
git tag v0.2.2-orders        # 4.3 完成
git tag v0.2.3-margin        # 4.4 完成
```

### 5.2 检查点

每阶段结束更新 `checkpoint.json`:
```json
{
  "current_phase": 4,
  "subtask": "4.1",
  "status": "completed",
  "files_created": [...],
  "files_modified": [...],
  "tests_passed": true
}
```

---

## 6. 开始实施

现在开始 Phase 4.1: 多交易对支持

→ 首先创建类型定义文件
