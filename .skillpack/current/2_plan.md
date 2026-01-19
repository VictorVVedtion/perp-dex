# Phase 2: REST API 区块链集成 - 详细实施计划

## 目标: 实现 REST API 与区块链的完整集成

---

## 1. Server 结构改造

### 1.1 抽象数据访问层

在 `api/` 下新增 `Service` 接口：

```go
// api/service.go
type OrderService interface {
    PlaceOrder(ctx context.Context, req *PlaceOrderRequest) (*PlaceOrderResponse, error)
    CancelOrder(ctx context.Context, trader, orderID string) (*CancelOrderResponse, error)
    GetOrder(ctx context.Context, orderID string) (*Order, error)
    GetOrders(ctx context.Context, trader, marketID string, limit int) ([]*Order, error)
}

type PositionService interface {
    GetPositions(ctx context.Context, trader string) ([]*Position, error)
    ClosePosition(ctx context.Context, req *ClosePositionRequest) (*ClosePositionResponse, error)
}

type AccountService interface {
    GetAccount(ctx context.Context, trader string) (*Account, error)
    Deposit(ctx context.Context, trader string, amount string) (*Account, error)
    Withdraw(ctx context.Context, trader string, amount string) (*Account, error)
}
```

### 1.2 Server 结构扩展

```go
type Server struct {
    httpServer      *http.Server
    wsServer        *websocket.Server
    config          *Config
    mockMode        bool

    // 新增服务层
    orderService    OrderService
    positionService PositionService
    accountService  AccountService
    rateLimiter     *middleware.RateLimiter
}
```

### 1.3 Mock 模式切换

- `config.MockMode=true` 或 keeper 为空时走 mock
- 否则走 keeper
- 仅在 mock 模式启动 `startMockDataBroadcaster`

---

## 2. 新增端点实现

### 2.1 订单端点

#### POST /v1/orders - 提交订单

**请求:**
```json
{
  "market_id": "BTC-USDC",
  "side": "buy",
  "type": "limit",
  "price": "96000.00",
  "quantity": "0.05",
  "trader": "cosmos1..."
}
```

**响应:**
```json
{
  "order": {
    "order_id": "order-12",
    "trader": "cosmos1...",
    "market_id": "BTC-USDC",
    "side": "buy",
    "type": "limit",
    "price": "96000.00",
    "quantity": "0.05",
    "filled_qty": "0.00",
    "status": "open",
    "created_at": 1710000000000,
    "updated_at": 1710000000000
  },
  "match": {
    "filled_qty": "0.00",
    "avg_price": "0.00",
    "remaining_qty": "0.05",
    "trades": []
  }
}
```

#### DELETE /v1/orders/{id} - 取消订单

**响应:**
```json
{
  "order": {
    "order_id": "order-12",
    "status": "cancelled",
    "updated_at": 1710000100000
  },
  "cancelled": true
}
```

#### PUT /v1/orders/{id} - 修改订单 (cancel+replace)

**请求:**
```json
{
  "price": "96500.00",
  "quantity": "0.03"
}
```

**响应:**
```json
{
  "old_order_id": "order-12",
  "order": {
    "order_id": "order-13",
    "status": "open"
  },
  "match": {
    "filled_qty": "0.00",
    "remaining_qty": "0.03",
    "trades": []
  }
}
```

#### GET /v1/orders - 查询订单列表

**Query:** `trader`, `market_id`, `status`, `limit`, `cursor`

**响应:**
```json
{
  "orders": [...],
  "next_cursor": "order-13",
  "total": 1
}
```

### 2.2 仓位端点

#### POST /v1/positions/close - 平仓

**请求:**
```json
{
  "market_id": "BTC-USDC",
  "size": "0.05",
  "price": "97500.00"
}
```

**响应:**
```json
{
  "market_id": "BTC-USDC",
  "closed_size": "0.05",
  "close_price": "97500.00",
  "realized_pnl": "12.50",
  "account": {
    "trader": "cosmos1...",
    "balance": "9012.50",
    "available_balance": "9012.50"
  }
}
```

### 2.3 账户端点

#### POST /v1/account/deposit - 入金

**请求:**
```json
{
  "amount": "1000.00"
}
```

**响应:**
```json
{
  "account": {
    "trader": "cosmos1...",
    "balance": "1000.00",
    "available_balance": "1000.00"
  }
}
```

#### POST /v1/account/withdraw - 出金

**请求:**
```json
{
  "amount": "250.00"
}
```

**响应:**
```json
{
  "account": {
    "trader": "cosmos1...",
    "balance": "750.00",
    "available_balance": "750.00"
  }
}
```

---

## 3. 速率限制集成

### 3.1 中间件链

```go
// 1. CORS (最外层，处理 OPTIONS)
// 2. RateLimitMiddleware (IP/用户级限速)
// 3. OrderRateLimitMiddleware (仅对订单端点)
// 4. 实际处理器

mux := http.NewServeMux()
// ... 注册路由 ...

// 应用中间件
handler := corsMiddleware(
    middleware.RateLimitMiddleware(s.rateLimiter)(mux),
)
```

### 3.2 订单特殊限速

对 POST/PUT/DELETE /v1/orders 叠加 `OrderRateLimitMiddleware`

---

## 4. 文件清单

| 文件 | 操作 | 说明 |
|------|------|------|
| `api/service.go` | 新建 | 服务接口定义 |
| `api/service_keeper.go` | 新建 | Keeper 实现 |
| `api/service_mock.go` | 新建 | Mock 实现 |
| `api/server.go` | 修改 | 添加新端点，启用限速 |
| `api/handlers/orders.go` | 新建 | 订单处理器 |
| `api/handlers/positions.go` | 新建 | 仓位处理器 |
| `api/handlers/account.go` | 新建 | 账户处理器 |

---

## 5. 执行顺序

```
[1] 创建服务接口 (api/service.go)
     │
[2] 实现 Mock 服务 (api/service_mock.go)
     │
[3] 实现订单端点 (api/handlers/orders.go)
     │
[4] 实现仓位端点 (api/handlers/positions.go)
     │
[5] 实现账户端点 (api/handlers/account.go)
     │
[6] 修改 server.go - 注册路由 + 启用限速
     │
[7] 测试验证
```

---

## 6. 预计时间

| 子任务 | 时间 |
|--------|------|
| 服务接口定义 | 15 min |
| Mock 服务实现 | 20 min |
| 订单端点 | 30 min |
| 仓位端点 | 20 min |
| 账户端点 | 20 min |
| Server 改造 + 限速 | 15 min |
| 测试验证 | 20 min |
| **总计** | **~2.5 小时** |

---

## 7. 验收标准

- [ ] POST /v1/orders 能提交订单（mock 模式返回模拟数据）
- [ ] DELETE /v1/orders/{id} 能取消订单
- [ ] PUT /v1/orders/{id} 能修改订单
- [ ] GET /v1/orders 能查询订单列表
- [ ] POST /v1/positions/close 能平仓
- [ ] POST /v1/account/deposit 能入金
- [ ] POST /v1/account/withdraw 能出金
- [ ] 速率限制生效，返回 429 和 Retry-After 头
- [ ] 所有错误返回统一 JSON 格式

---

**准备进入 Phase 3: 执行实现**
