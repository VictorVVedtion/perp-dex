# Hyperliquid 功能对齐计划

## 📊 Executive Summary

本文档详细分析了当前 PerpDEX MVP 与 Hyperliquid 的功能差距，并提供了全面的对齐实施计划。

**目标**: 将 PerpDEX 对齐到 Hyperliquid 级别的功能和用户体验

**预估工作量**: 大型项目，涉及多个核心模块的重构和新增

---

## 🔍 差距分析

### 1. 订单类型对比

| 订单类型 | Hyperliquid | PerpDEX | 状态 | 优先级 |
|---------|-------------|---------|------|--------|
| Market | ✅ | ✅ | 已实现 | - |
| Limit | ✅ | ✅ | 已实现 | - |
| Stop Market | ✅ | ✅ | 已实现 (Stop Loss) | - |
| Stop Limit | ✅ | ✅ | 已实现 | - |
| Take Profit | ✅ | ✅ | 已实现 | - |
| Take Profit Limit | ✅ | ✅ | 已实现 | - |
| **Scale Orders** | ✅ | ❌ | **缺失** | P1 |
| **TWAP** | ✅ | ❌ | **缺失** | P1 |
| Trailing Stop | ✅ | ✅ | 已实现 | - |
| OCO | ✅ | ✅ | 已实现 | - |

### 2. 执行控制对比

| 执行控制 | Hyperliquid | PerpDEX | 状态 | 优先级 |
|---------|-------------|---------|------|--------|
| GTC | ✅ | ✅ | 已实现 | - |
| IOC | ✅ | ✅ | 已实现 | - |
| FOK | ✅ | ✅ | 已实现 | - |
| Post Only (ALO/GTX) | ✅ | ✅ | 已实现 | - |
| Reduce Only | ✅ | ✅ | 已实现 | - |
| **Trigger Price Mark/Last** | ✅ | ❌ | **缺失** | P2 |

### 3. 清算机制对比

| 功能 | Hyperliquid | PerpDEX | 状态 | 优先级 |
|------|-------------|---------|------|--------|
| 基础清算 | ✅ | ✅ | 已实现 | - |
| 清算奖励 | ✅ | ✅ | 已实现 (30%) | - |
| **部分清算 (>$100K)** | ✅ 20%先清算 | ❌ 全额清算 | **需升级** | P0 |
| **清算冷却期** | ✅ 30秒 | ❌ | **缺失** | P0 |
| **后备清算 (Vault)** | ✅ HLP | ❌ | **缺失** | P1 |
| **ADL 自动去杠杆** | ✅ | ❌ | **缺失** | P1 |
| **清算 Mark Price** | ✅ 外部+内部 | 部分 | **需升级** | P1 |

### 4. 保证金系统对比

| 功能 | Hyperliquid | PerpDEX | 状态 | 优先级 |
|------|-------------|---------|------|--------|
| 逐仓保证金 | ✅ | ✅ | 已实现 | - |
| 跨仓保证金 | ✅ | ✅ | 已实现 | - |
| **组合保证金** | ✅ | ❌ | **缺失** | P2 |
| **动态保证金调整** | ✅ | ❌ | **缺失** | P2 |
| 杠杆限制 (BTC/ETH) | ✅ 40x | ✅ 50x | 需调整 | P2 |

### 5. Vault 系统对比

| 功能 | Hyperliquid | PerpDEX | 状态 | 优先级 |
|------|-------------|---------|------|--------|
| 保险基金 | ✅ | ✅ 基础 | 需完善 | P1 |
| **HLP Vault** | ✅ | ❌ | **缺失** | P1 |
| **协议金库** | ✅ | ❌ | **缺失** | P2 |
| **用户金库** | ✅ | ❌ | **缺失** | P3 |
| **金库清算参与** | ✅ | ❌ | **缺失** | P1 |

### 6. API 系统对比

| 功能 | Hyperliquid | PerpDEX | 状态 | 优先级 |
|------|-------------|---------|------|--------|
| REST Info API | ✅ | ✅ 部分 | 需扩展 | P1 |
| REST Exchange API | ✅ | ✅ 部分 | 需扩展 | P1 |
| WebSocket 实时数据 | ✅ | ✅ 基础 | 需完善 | P1 |
| **API Wallet 委托** | ✅ | ❌ | **缺失** | P2 |
| **SDK (Python/TS)** | ✅ | ❌ | **缺失** | P2 |

### 7. UI/UX 对比

| 功能 | Hyperliquid | PerpDEX | 状态 | 优先级 |
|------|-------------|---------|------|--------|
| 交易界面 | ✅ 专业 | ✅ 基础 | 需升级 | P1 |
| 订单簿深度图 | ✅ | ✅ | 已实现 | - |
| K线图表 | ✅ | ✅ | 已实现 | - |
| **高级订单面板** | ✅ | 部分 | 需升级 | P1 |
| **Portfolio 视图** | ✅ | ❌ | **缺失** | P2 |
| **资金费率显示** | ✅ | 部分 | 需升级 | P2 |

---

## 🎯 优先级分类

### P0 - 关键 (必须立即实现)
风险管理核心功能，影响系统安全性

1. 三层清算机制
2. 部分清算 (大仓位)
3. 清算冷却期

### P1 - 高优先级 (核心功能)
用户体验和功能完整性的关键

1. Scale Orders 规模订单
2. TWAP 时间加权订单
3. HLP Vault / 后备清算
4. ADL 自动去杠杆
5. API 扩展
6. UI 高级订单面板

### P2 - 中优先级 (增强功能)
提升专业度和竞争力

1. 组合保证金 (Portfolio Margin)
2. 动态保证金调整
3. API Wallet 委托
4. SDK 开发
5. Portfolio 视图

### P3 - 低优先级 (后续迭代)
1. 用户金库系统
2. 更多市场对支持
3. 高级分析工具

---

## 📋 详细实施计划

### Phase 1: 清算机制升级 (P0) ⚠️ 最高优先级

#### 1.1 三层清算机制

**文件**: `x/clearinghouse/keeper/liquidation.go`

```go
// 新增清算层级
type LiquidationTier int

const (
    TierMarketOrder LiquidationTier = iota + 1  // 层级1: 市场订单清算
    TierPartialLiquidation                       // 层级2: 部分清算
    TierBackstopLiquidation                      // 层级3: 后备清算 (Vault)
)
```

**需要实现**:

1. **大仓位部分清算**
   - 仓位 > $100K USDC 时，首次只清算 20%
   - 实现清算冷却期 30 秒
   - 冷却期后可清算剩余部分

2. **后备清算机制**
   - 当 equity < 2/3 * maintenance margin 时
   - 触发 Liquidator Vault 接管仓位
   - 清算利润分配给 Vault 参与者

3. **ADL 自动去杠杆**
   - 当 HLP 无法覆盖损失时触发
   - 按盈利和杠杆排序对手方
   - 强制减仓盈利最多的交易者

**代码结构**:

```
x/clearinghouse/
├── keeper/
│   ├── liquidation.go          # 重构
│   ├── liquidation_tiers.go    # 新增: 三层清算
│   ├── partial_liquidation.go  # 新增: 部分清算
│   ├── backstop.go             # 新增: 后备清算
│   └── adl.go                  # 新增: ADL
├── types/
│   ├── liquidation_config.go   # 新增: 清算配置
│   └── adl.go                  # 新增: ADL 类型
```

---

### Phase 2: 高级订单类型 (P1)

#### 2.1 Scale Orders (规模订单)

**功能**: 在指定价格范围内创建多个限价订单

**文件**: `x/orderbook/keeper/scale_order.go`

```go
type ScaleOrder struct {
    OrderID       string
    Trader        string
    MarketID      string
    Side          Side
    TotalQuantity math.LegacyDec
    PriceStart    math.LegacyDec
    PriceEnd      math.LegacyDec
    OrderCount    int              // 订单数量 (通常 5-20)
    Distribution  string           // "linear" | "exponential"
    SubOrders     []*Order         // 生成的子订单
    Status        OrderStatus
    CreatedAt     time.Time
}
```

**实现要点**:
- 支持线性和指数分布
- 自动生成并管理子订单
- 取消时同时取消所有子订单

#### 2.2 TWAP 订单 (时间加权平均价格)

**功能**: 将大订单分散在时间段内执行，减少市场冲击

**文件**: `x/orderbook/keeper/twap.go`

```go
type TWAPOrder struct {
    OrderID        string
    Trader         string
    MarketID       string
    Side           Side
    TotalQuantity  math.LegacyDec
    Duration       time.Duration    // 执行时长
    Interval       time.Duration    // 子订单间隔 (30秒)
    SlippageTol    math.LegacyDec   // 滑点容忍度 (3%)
    ExecutedQty    math.LegacyDec
    SubOrdersTotal int
    SubOrdersExec  int
    Status         TWAPStatus
    StartTime      time.Time
    EndTime        time.Time
}

type TWAPStatus int

const (
    TWAPStatusPending TWAPStatus = iota
    TWAPStatusActive
    TWAPStatusCompleted
    TWAPStatusCancelled
)
```

**实现要点**:
- 30 秒间隔执行子订单
- 3% 最大滑点保护
- 如果子订单未完全成交，后续订单增加最多 3x
- EndBlock 钩子触发子订单执行

---

### Phase 3: Vault 系统 (P1)

#### 3.1 HLP Vault (Hyperliquidity Provider)

**新模块**: `x/vault/`

**功能**:
1. 接收用户存款
2. 提供做市流动性
3. 参与清算获取收益
4. 利润分配给存款人

**代码结构**:

```
x/vault/
├── keeper/
│   ├── keeper.go               # 核心 Keeper
│   ├── deposit.go              # 存款/取款
│   ├── strategy.go             # 做市策略
│   ├── liquidation.go          # 清算参与
│   ├── pnl.go                  # 盈亏计算
│   └── distribution.go         # 收益分配
├── types/
│   ├── types.go                # 核心类型
│   ├── vault.go                # Vault 定义
│   ├── deposit.go              # 存款记录
│   ├── strategy.go             # 策略配置
│   └── msgs.go                 # 消息定义
├── client/
│   └── cli/                    # CLI 命令
└── module.go                   # 模块注册
```

**Vault 类型定义**:

```go
type Vault struct {
    VaultID          string
    Name             string
    Description      string
    TotalDeposits    math.LegacyDec
    TotalShares      math.LegacyDec
    UnrealizedPnL    math.LegacyDec
    RealizedPnL      math.LegacyDec
    LeaderAddress    string          // Vault 管理者
    LeaderFeeRate    math.LegacyDec  // 管理费率 (默认 10%)
    ProtocolFeeRate  math.LegacyDec  // 协议费率
    Status           VaultStatus
    Strategies       []StrategyType
    CreatedAt        time.Time
}

type VaultDeposit struct {
    DepositID    string
    VaultID      string
    Depositor    string
    Shares       math.LegacyDec
    DepositValue math.LegacyDec
    DepositTime  time.Time
}
```

---

### Phase 4: API 扩展 (P1)

#### 4.1 REST API 补充

**新增端点**:

```
# 市场数据
GET  /api/v1/markets                        # 所有市场列表
GET  /api/v1/markets/{market_id}/ticker     # 24h 行情
GET  /api/v1/markets/{market_id}/funding    # 资金费率

# 订单管理
POST /api/v1/orders/scale                   # 创建 Scale 订单
POST /api/v1/orders/twap                    # 创建 TWAP 订单
GET  /api/v1/orders/conditional             # 条件订单列表
DELETE /api/v1/orders/batch                 # 批量取消

# 账户
GET  /api/v1/account/portfolio              # 组合视图
GET  /api/v1/account/pnl/history            # 盈亏历史
GET  /api/v1/account/funding/history        # 资金费历史

# Vault
GET  /api/v1/vaults                         # Vault 列表
GET  /api/v1/vaults/{vault_id}              # Vault 详情
POST /api/v1/vaults/{vault_id}/deposit      # 存款
POST /api/v1/vaults/{vault_id}/withdraw     # 取款
```

#### 4.2 WebSocket 增强

**新增订阅频道**:

```javascript
// 订阅格式
{
  "op": "subscribe",
  "channel": "trades",
  "market": "BTC-USDC"
}

// 新增频道
- "fills"           // 个人成交
- "funding"         // 资金费率更新
- "liquidations"    // 清算事件
- "vault_pnl"       // Vault 盈亏
- "adl_warning"     // ADL 预警
```

---

### Phase 5: UI/UX 升级 (P1)

#### 5.1 高级订单面板

**文件**: `frontend/src/components/AdvancedOrderPanel.tsx`

**功能**:
- Scale Order 配置界面
- TWAP Order 配置界面
- 条件订单管理
- TP/SL 快捷设置

#### 5.2 Portfolio 视图

**文件**: `frontend/src/pages/portfolio.tsx`

**功能**:
- 所有仓位汇总
- 未实现盈亏可视化
- 保证金使用率
- 清算风险指示器

#### 5.3 Vault 界面

**文件**: `frontend/src/pages/vault.tsx`

**功能**:
- Vault 列表和详情
- 存款/取款操作
- 收益历史图表
- APY 显示

---

## 📁 文件变更汇总

### 新增文件

```
x/clearinghouse/keeper/
├── liquidation_tiers.go        # 三层清算
├── partial_liquidation.go      # 部分清算
├── backstop.go                 # 后备清算
└── adl.go                      # ADL

x/orderbook/keeper/
├── scale_order.go              # Scale 订单
└── twap.go                     # TWAP 订单

x/vault/                        # 新模块
├── keeper/
├── types/
├── client/
└── module.go

api/handlers/
├── scale.go
├── twap.go
├── vault.go
└── portfolio.go

frontend/src/
├── pages/
│   ├── portfolio.tsx
│   └── vault.tsx
├── components/
│   ├── AdvancedOrderPanel.tsx
│   ├── ScaleOrderForm.tsx
│   ├── TWAPOrderForm.tsx
│   ├── VaultCard.tsx
│   ├── PortfolioSummary.tsx
│   └── ADLWarning.tsx
```

### 修改文件

```
x/clearinghouse/keeper/liquidation.go    # 重构清算逻辑
x/perpetual/keeper/margin.go             # 添加组合保证金支持
x/orderbook/types/types.go               # 添加新订单类型
api/server.go                            # 添加新路由
api/websocket/server.go                  # 添加新频道
frontend/src/components/TradeForm.tsx    # 添加高级订单入口
frontend/src/stores/tradingStore.ts      # 状态管理扩展
```

---

## 🚀 实施时间线

### 第一阶段: 清算机制 (Week 1-2)
- [ ] 三层清算架构设计
- [ ] 部分清算实现
- [ ] 清算冷却期
- [ ] 后备清算基础
- [ ] 单元测试

### 第二阶段: 高级订单 (Week 3-4)
- [ ] Scale Order 实现
- [ ] TWAP Order 实现
- [ ] 触发价格 Mark/Last 选项
- [ ] 集成测试

### 第三阶段: Vault 系统 (Week 5-6)
- [ ] Vault 模块基础
- [ ] 存款/取款逻辑
- [ ] 清算参与机制
- [ ] 收益分配

### 第四阶段: ADL (Week 7)
- [ ] ADL 算法实现
- [ ] 排序和选择逻辑
- [ ] 前端警告界面

### 第五阶段: API 和 UI (Week 8-9)
- [ ] REST API 扩展
- [ ] WebSocket 增强
- [ ] 前端高级订单面板
- [ ] Portfolio 页面
- [ ] Vault 页面

### 第六阶段: 测试和优化 (Week 10)
- [ ] 端到端测试
- [ ] 性能基准测试
- [ ] Bug 修复
- [ ] 文档更新

---

## 📊 验收标准

### 功能验收
- [ ] 三层清算正常工作
- [ ] Scale/TWAP 订单可正常下单和执行
- [ ] Vault 存取款和收益分配正确
- [ ] ADL 在极端情况下正确触发
- [ ] 所有 API 端点可用
- [ ] WebSocket 实时推送正常

### 性能验收
- [ ] 清算延迟 < 500ms
- [ ] TWAP 子订单准时执行 (±1秒)
- [ ] API 响应时间 < 100ms
- [ ] WebSocket 延迟 < 50ms

### 测试覆盖
- [ ] 单元测试覆盖率 > 80%
- [ ] 清算边界情况测试
- [ ] 压力测试通过

---

## 📚 参考资源

### Hyperliquid 官方文档
- [Order Types](https://hyperliquid.gitbook.io/hyperliquid-docs/trading/order-types)
- [Liquidations](https://hyperliquid.gitbook.io/hyperliquid-docs/trading/liquidations)
- [Portfolio Margin](https://hyperliquid.gitbook.io/hyperliquid-docs/trading/portfolio-margin)
- [Vaults](https://hyperliquid.gitbook.io/hyperliquid-docs/hypercore/vaults/protocol-vaults)
- [API Documentation](https://hyperliquid.gitbook.io/hyperliquid-docs/for-developers/api)

### 技术参考
- [Bitcoin News - Hyperliquid Explained](https://news.bitcoin.com/hyperliquid-explained-a-deep-dive-into-the-perp-dex-that-reshaped-crypto-in-2025/)
- [QuickNode - Protocol Analysis](https://blog.quicknode.com/hyperliquid-protocol-analysis-2025/)
- [Cointelegraph - HLP Explained](https://cointelegraph.com/explained/what-is-hyperliquid-hlp-and-how-does-it-work)

---

**文档版本**: 1.0.0
**创建日期**: 2026-01-19
**最后更新**: 2026-01-19
