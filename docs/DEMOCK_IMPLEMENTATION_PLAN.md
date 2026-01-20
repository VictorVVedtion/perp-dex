# 去 Mock 化实施计划

## 概述

本文档描述如何将 PerpDEX 项目从 Mock 数据模式迁移到真实引擎模式，实现真实性能验证。

## 当前架构问题

```
API Layer (MockService)  ←── 断层 ──→  Cosmos Modules (OrderBookV2, MatchingEngineV2)
```

- `api/service_mock.go`: 所有订单操作返回模拟数据
- `x/orderbook/keeper/`: 高性能引擎从未被 API 层调用

---

## 阶段 1: 接入真实订单簿引擎

### 1.1 新建 `api/service_real.go`

```go
package api

import (
    "context"
    "cosmossdk.io/math"
    sdk "github.com/cosmos/cosmos-sdk/types"
    "github.com/openalpha/perp-dex/api/types"
    obkeeper "github.com/openalpha/perp-dex/x/orderbook/keeper"
    obtypes "github.com/openalpha/perp-dex/x/orderbook/types"
    perpkeeper "github.com/openalpha/perp-dex/x/perpetual/keeper"
)

// RealService implements service interfaces with real orderbook engine
type RealService struct {
    obKeeper      *obkeeper.Keeper
    perpKeeper    *perpkeeper.Keeper
    matchEngine   *obkeeper.MatchingEngineV2
    sdkCtx        sdk.Context
}

func NewRealService(obk *obkeeper.Keeper, pk *perpkeeper.Keeper, ctx sdk.Context) *RealService {
    return &RealService{
        obKeeper:    obk,
        perpKeeper:  pk,
        matchEngine: obkeeper.NewMatchingEngineV2(obk),
        sdkCtx:      ctx,
    }
}

// PlaceOrder 使用真实匹配引擎
func (rs *RealService) PlaceOrder(ctx context.Context, req *types.PlaceOrderRequest) (*types.PlaceOrderResponse, error) {
    // 转换参数
    price, _ := math.LegacyNewDecFromStr(req.Price)
    qty, _ := math.LegacyNewDecFromStr(req.Quantity)
    side := obtypes.SideBuy
    if req.Side == "sell" { side = obtypes.SideSell }
    orderType := obtypes.OrderTypeLimit
    if req.Type == "market" { orderType = obtypes.OrderTypeMarket }

    // 调用真实 Keeper
    order, result, err := rs.obKeeper.PlaceOrder(ctx, req.Trader, req.MarketID, side, orderType, price, qty)
    if err != nil {
        return nil, err
    }

    // 转换响应
    return convertToAPIResponse(order, result), nil
}
```

### 1.2 修改 `api/server.go`

添加 `NewServerWithRealService()` 构造函数:

```go
func NewServerWithRealService(config *Config, obk *obkeeper.Keeper, pk *perpkeeper.Keeper, ctx sdk.Context) *Server {
    realService := NewRealService(obk, pk, ctx)
    return &Server{
        config:          config,
        orderService:    realService,
        positionService: realService,
        accountService:  realService,
        mockMode:        false,
    }
}
```

### 1.3 修改 `cmd/api/main.go`

添加 `--real` 启动模式:

```go
realMode := flag.Bool("real", false, "Use real orderbook engine")

if *realMode {
    // 初始化 Cosmos SDK Context 和 Keepers
    obKeeper, perpKeeper, sdkCtx := initializeKeepers()
    server = api.NewServerWithRealService(config, obKeeper, perpKeeper, sdkCtx)
} else {
    server = api.NewServer(config)
}
```

---

## 阶段 2: 性能基准测试改造

### 2.1 新建 `tests/benchmark/engine_benchmark_test.go`

直接测试 `MatchingEngineV2`，绕过 HTTP 层:

```go
func BenchmarkMatchingEngineV2_PlaceOrder(b *testing.B) {
    keeper := setupTestKeeper()
    engine := obkeeper.NewMatchingEngineV2(keeper)

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        order := createTestOrder(i)
        engine.ProcessOrderOptimized(ctx, order)
    }
}

func BenchmarkMatchingEngineV2_MatchBatch(b *testing.B) {
    // 测试批量匹配性能
}

func BenchmarkOrderBookV2_AddRemove(b *testing.B) {
    // 测试订单簿增删性能
}
```

### 2.2 测试场景

| 场景 | 目标指标 | 当前 Mock 情况 |
|------|---------|---------------|
| 10K 订单匹配 | < 10ms | N/A (未测试) |
| 单订单添加 | < 1μs | N/A |
| 高并发(100) | 锁竞争 < 5% | N/A |

---

## 阶段 3: 本地链集成

### 3.1 启动流程

```bash
# 1. 编译
make build

# 2. 初始化链
make init-chain

# 3. 启动节点
make start

# 4. 启动 API 服务器 (真实模式)
./cmd/api/main.go --real --chain-rpc=localhost:26657
```

### 3.2 配置文件 `config/localnet.yaml`

```yaml
chain:
  id: perpdex-local-1
  rpc: "http://localhost:26657"
  rest: "http://localhost:1317"

api:
  real_mode: true
  mock_fallback: false

performance:
  parallel_matching: true
  workers: 4
  batch_size: 100
```

---

## 阶段 4: 混合模式架构

### 4.1 数据源分离

| 数据类型 | 数据源 | 理由 |
|---------|-------|------|
| 市场价格/K线/订单簿(展示) | Hyperliquid API | 真实市场数据 |
| 订单下单/取消/修改 | 本地 MatchingEngineV2 | 真实匹配逻辑 |
| 持仓/账户 | 本地 perpetual.Keeper | 真实状态管理 |

### 4.2 前端配置

```typescript
// frontend/src/lib/config.ts
export const config = {
  api: {
    marketData: process.env.NEXT_PUBLIC_HL_API_URL,  // Hyperliquid
    trading: process.env.NEXT_PUBLIC_API_URL,        // 本地真实引擎
  },
  features: {
    mockMode: false,
    useHyperliquidForMarketData: true,
    useRealEngineForTrading: true,
  }
}
```

---

## 实施顺序

```
Week 1: 阶段 1 (RealService)
├── Day 1-2: 实现 service_real.go
├── Day 3: 修改 server.go 和 main.go
└── Day 4-5: 单元测试和集成验证

Week 2: 阶段 2 (基准测试)
├── Day 1-2: 实现 engine_benchmark_test.go
├── Day 3: 运行基准测试，收集基线数据
└── Day 4-5: 性能优化迭代

Week 3: 阶段 3 (本地链)
├── Day 1-2: 配置本地链启动脚本
├── Day 3-4: 集成测试
└── Day 5: E2E 测试

Week 4: 阶段 4 (混合模式)
├── Day 1-2: 前端配置更新
├── Day 3-4: 全链路测试
└── Day 5: 文档和清理
```

---

## 验收标准

1. **阶段 1**: `go run cmd/api/main.go --real` 可正常启动并处理订单
2. **阶段 2**: 基准测试显示 10K 订单匹配 < 10ms
3. **阶段 3**: 完整的链上交易流程可验证
4. **阶段 4**: 前端可同时使用 Hyperliquid 数据和真实引擎交易

---

## 风险与缓解

| 风险 | 缓解措施 |
|------|---------|
| SDK Context 生命周期管理 | 使用 Context Pool |
| 并发安全 | 已有锁机制 (OrderBookV2.mu) |
| 数据一致性 | 定期 Flush + EndBlocker |
