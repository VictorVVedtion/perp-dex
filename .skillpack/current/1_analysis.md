# Phase 1: 深度分析报告 - REST API 区块链集成

## 任务: 实现 REST API 与区块链的完整集成

---

## 1. 问题诊断

### 1.1 当前 API 架构问题

```
┌──────────┐     ┌──────────┐     ┌──────────┐
│ Frontend │ --> │ REST API │ --> │ Mock Data│  ❌ 完全断开
└──────────┘     └──────────┘     └──────────┘
                      ✗
                 区块链未连接
```

**现状**:
- `api/server.go` 只有 GET 端点
- 所有数据来自 `mock_data.go` 中的随机生成函数
- 速率限制中间件完整实现但**从未使用**
- 没有任何写操作端点（POST/PUT/DELETE）

### 1.2 缺失的端点

| 端点 | 方法 | 用途 | 优先级 |
|------|------|------|--------|
| `/v1/orders` | POST | 提交订单 | P0 |
| `/v1/orders/{id}` | DELETE | 取消订单 | P0 |
| `/v1/orders/{id}` | PUT | 修改订单 | P1 |
| `/v1/orders` | GET | 查询订单列表 | P0 |
| `/v1/positions/close` | POST | 平仓 | P1 |
| `/v1/account/deposit` | POST | 入金 | P1 |
| `/v1/account/withdraw` | POST | 出金 | P1 |

### 1.3 可用的 Keeper 函数

从 `x/orderbook/keeper/keeper.go` 分析：

```go
// 已实现的关键函数
PlaceOrder(ctx, trader, marketID, side, orderType, price, quantity)
CancelOrder(ctx, trader, orderID)
GetOrder(ctx, orderID)
GetOrdersByTrader(ctx, trader)
GetOrderBook(ctx, marketID)
GetRecentTrades(ctx, marketID, limit)
```

### 1.4 速率限制状态

**已实现** (`api/middleware/ratelimit.go`):
- IP 速率限制: 100 req/s
- 用户速率限制: 200 req/s
- 订单速率限制: 10 orders/s, 10000/day
- Token bucket 算法
- 日限额计数器

**问题**: `server.go` 中从未调用 `RateLimitMiddleware`

---

## 2. 解决方案架构

### 2.1 目标架构

```
┌──────────┐     ┌──────────┐     ┌──────────┐     ┌─────────────┐
│ Frontend │ --> │ REST API │ --> │ Keepers  │ --> │ Cosmos Chain│
└──────────┘     └──────────┘     └──────────┘     └─────────────┘
                      │
                ┌─────┴─────┐
                │ RateLimit │
                │ Middleware│
                └───────────┘
```

### 2.2 需要修改的文件

| 文件 | 修改类型 | 说明 |
|------|----------|------|
| `api/server.go` | 重构 | 添加 keeper 引用，新增端点，启用中间件 |
| `api/handlers/orders.go` | 新建 | 订单相关处理器 |
| `api/handlers/positions.go` | 新建 | 仓位相关处理器 |
| `api/handlers/account.go` | 新建 | 账户相关处理器 |
| `cmd/api/main.go` | 修改 | 传入 keeper 引用 |

### 2.3 实现策略

1. **Server 结构改造**:
```go
type Server struct {
    httpServer      *http.Server
    wsServer        *websocket.Server
    config          *Config
    mockMode        bool
    // 新增
    orderbookKeeper *orderbookkeeper.Keeper
    perpetualKeeper *perpetualkeeper.Keeper
    rateLimiter     *middleware.RateLimiter
}
```

2. **端点路由**:
```go
// 启用速率限制
handler := middleware.RateLimitMiddleware(s.rateLimiter)(mux)

// 新增端点
mux.HandleFunc("/v1/orders", s.handleOrders)        // GET/POST
mux.HandleFunc("/v1/orders/", s.handleOrder)        // GET/PUT/DELETE
mux.HandleFunc("/v1/positions/close", s.handleClosePosition)
mux.HandleFunc("/v1/account/deposit", s.handleDeposit)
mux.HandleFunc("/v1/account/withdraw", s.handleWithdraw)
```

3. **Mock 模式保留**:
- 当 `mockMode=true` 时使用 mock 数据
- 当 `mockMode=false` 时查询链上数据

---

## 3. 子任务拆分

| 子任务 | 预计时间 | 依赖 |
|--------|----------|------|
| 3.1 Server 结构改造 | 15min | - |
| 3.2 订单端点实现 | 30min | 3.1 |
| 3.3 仓位端点实现 | 20min | 3.1 |
| 3.4 账户端点实现 | 20min | 3.1 |
| 3.5 启用速率限制 | 10min | 3.1 |
| 3.6 测试验证 | 15min | 3.2-3.5 |

---

## 4. 风险评估

| 风险 | 影响 | 缓解措施 |
|------|------|----------|
| Keeper 引用传递复杂 | 中 | 使用接口定义，支持 mock |
| Context 管理 | 低 | 创建合适的 SDK Context |
| 认证机制缺失 | 中 | 先实现基本功能，后续添加签名验证 |

---

## 5. 结论

问题明确，解决方案可行。主要工作量在于：
1. 改造 Server 结构以支持 keeper 注入
2. 实现 7 个新端点
3. 启用已实现的速率限制中间件

**准备进入 Phase 2: Codex 规划实施细节**
