# PerpDEX REST API Specification

## Overview

PerpDEX REST API 提供完整的永续合约交易接口，支持订单管理、仓位操作和账户管理。

**Base URL:** `http://localhost:8080`

**认证方式:** `X-Trader-Address` Header 或请求体中的 `trader` 字段

---

## 端点一览

| 方法 | 路径 | 描述 |
|------|------|------|
| GET | `/health` | 健康检查 |
| GET | `/v1/markets` | 获取市场列表 |
| GET | `/v1/markets/{id}` | 获取单个市场 |
| GET | `/v1/markets/{id}/ticker` | 获取行情 |
| GET | `/v1/markets/{id}/orderbook` | 获取订单簿 |
| GET | `/v1/markets/{id}/trades` | 获取成交记录 |
| **POST** | `/v1/orders` | **提交订单** |
| **GET** | `/v1/orders` | **查询订单列表** |
| **GET** | `/v1/orders/{id}` | **查询单个订单** |
| **PUT** | `/v1/orders/{id}` | **修改订单** |
| **DELETE** | `/v1/orders/{id}` | **取消订单** |
| GET | `/v1/positions` | 查询仓位列表 |
| GET | `/v1/positions/{marketID}` | 查询单个仓位 |
| **POST** | `/v1/positions/close` | **平仓** |
| GET | `/v1/account` | 查询账户信息 |
| **POST** | `/v1/account/deposit` | **入金** |
| **POST** | `/v1/account/withdraw` | **出金** |

---

## 订单接口

### POST /v1/orders - 提交订单

**Request:**
```json
{
  "market_id": "BTC-USDC",
  "side": "buy",           // "buy" | "sell"
  "type": "limit",         // "limit" | "market"
  "price": "96000.00",     // 限价单必填
  "quantity": "0.05",
  "trader": "cosmos1..."   // 可选，也可通过 Header 传入
}
```

**Response (201 Created):**
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

### GET /v1/orders - 查询订单列表

**Query Parameters:**
| 参数 | 类型 | 必填 | 描述 |
|------|------|------|------|
| trader | string | 否 | 交易者地址 |
| market_id | string | 否 | 市场 ID |
| status | string | 否 | 订单状态 (open/filled/cancelled) |
| limit | int | 否 | 返回数量限制 (默认 100) |
| cursor | string | 否 | 分页游标 |

**Response (200 OK):**
```json
{
  "orders": [...],
  "next_cursor": "order-13",
  "total": 1
}
```

### DELETE /v1/orders/{id} - 取消订单

**Headers:**
```
X-Trader-Address: cosmos1...
```

**Response (200 OK):**
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

### PUT /v1/orders/{id} - 修改订单

采用 Cancel-Replace 机制：取消旧订单，创建新订单。

**Request:**
```json
{
  "price": "96500.00",     // 可选
  "quantity": "0.03"       // 可选，至少填一个
}
```

**Response (200 OK):**
```json
{
  "old_order_id": "order-12",
  "order": {
    "order_id": "order-13",
    "status": "open",
    ...
  },
  "match": {...}
}
```

---

## 仓位接口

### GET /v1/positions - 查询仓位列表

**Query Parameters:**
| 参数 | 类型 | 必填 | 描述 |
|------|------|------|------|
| trader | string | 否 | 交易者地址 |

**Response (200 OK):**
```json
{
  "positions": [
    {
      "market_id": "BTC-USDC",
      "trader": "cosmos1...",
      "side": "long",
      "size": "0.1",
      "entry_price": "97200.00",
      "mark_price": "97500.00",
      "margin": "1944.00",
      "leverage": "5",
      "unrealized_pnl": "30.00",
      "liquidation_price": "88560.00",
      "margin_mode": "isolated"
    }
  ],
  "total": 1
}
```

### POST /v1/positions/close - 平仓

**Request:**
```json
{
  "market_id": "BTC-USDC",
  "size": "0.05",          // 可选，默认全部平仓
  "price": "97500.00",     // 可选，默认使用标记价格
  "trader": "cosmos1..."
}
```

**Response (200 OK):**
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

---

## 账户接口

### GET /v1/account - 查询账户

**Query Parameters:**
| 参数 | 类型 | 必填 | 描述 |
|------|------|------|------|
| trader | string | 是 | 交易者地址 |

**Response (200 OK):**
```json
{
  "account": {
    "trader": "cosmos1...",
    "balance": "12500.00",
    "locked_margin": "4000.00",
    "available_balance": "8500.00",
    "margin_mode": "isolated",
    "updated_at": 1710000000000
  }
}
```

### POST /v1/account/deposit - 入金

**Request:**
```json
{
  "amount": "1000.00",
  "trader": "cosmos1..."
}
```

**Response (200 OK):**
```json
{
  "account": {
    "trader": "cosmos1...",
    "balance": "1000.00",
    "available_balance": "1000.00",
    ...
  }
}
```

### POST /v1/account/withdraw - 出金

**Request:**
```json
{
  "amount": "250.00",
  "trader": "cosmos1..."
}
```

**Response (200 OK):**
```json
{
  "account": {
    "trader": "cosmos1...",
    "balance": "750.00",
    "available_balance": "750.00",
    ...
  }
}
```

---

## 错误响应

所有错误返回统一 JSON 格式：

```json
{
  "error": "error_code",
  "message": "Human readable message"
}
```

**常见错误码：**

| HTTP Status | Error Code | 描述 |
|-------------|------------|------|
| 400 | invalid_json | 请求体 JSON 格式错误 |
| 400 | missing_market_id | 缺少 market_id |
| 400 | missing_trader | 缺少 trader 地址 |
| 400 | missing_price | 限价单缺少 price |
| 401 | unauthorized | 未授权 |
| 403 | unauthorized | 订单不属于该交易者 |
| 404 | order_not_found | 订单不存在 |
| 404 | position_not_found | 仓位不存在 |
| 405 | method_not_allowed | HTTP 方法不允许 |
| 429 | rate_limit_exceeded | 请求频率超限 |

---

## 速率限制

| 类型 | 限制 |
|------|------|
| IP 请求 | 100 req/s |
| 用户请求 | 200 req/s |
| 订单提交 | 10 orders/s |
| 日订单量 | 10,000 orders/day |

超限时返回 `429 Too Many Requests`，包含 `Retry-After` Header。

---

## WebSocket

**Endpoint:** `ws://localhost:8080/ws`

订阅格式：
```json
{
  "action": "subscribe",
  "channel": "ticker",
  "market_id": "BTC-USDC"
}
```

可用频道：
- `ticker` - 行情推送
- `orderbook` - 订单簿更新
- `trades` - 成交推送
- `klines` - K 线数据

---

## 示例

### cURL 提交订单

```bash
curl -X POST http://localhost:8080/v1/orders \
  -H "Content-Type: application/json" \
  -H "X-Trader-Address: cosmos1abc..." \
  -d '{
    "market_id": "BTC-USDC",
    "side": "buy",
    "type": "limit",
    "price": "96000.00",
    "quantity": "0.05"
  }'
```

### cURL 查询账户

```bash
curl "http://localhost:8080/v1/account?trader=cosmos1abc..."
```

### cURL 平仓

```bash
curl -X POST http://localhost:8080/v1/positions/close \
  -H "Content-Type: application/json" \
  -H "X-Trader-Address: cosmos1abc..." \
  -d '{
    "market_id": "BTC-USDC"
  }'
```
