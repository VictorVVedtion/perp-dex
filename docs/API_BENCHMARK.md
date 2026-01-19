# PerpDEX API 压测指南

## 快速开始

### 1. 启动服务器

```bash
# 编译
go build -o ./build/perpdex-api ./cmd/api

# 启动 (--keeper 连接真实订单簿引擎)
./build/perpdex-api --keeper=true
```

服务启动后:
- API: `http://localhost:8080`
- 订单簿查询: `http://localhost:8081/orderbook/BTC-USDC`

### 2. 运行压测

```bash
# 编译压测工具
go build -o ./build/benchmark_http ./cmd/benchmark/main.go

# 运行压测
./build/benchmark_http -n 5000 -c 8
```

参数:
| 参数 | 说明 | 默认值 |
|------|------|--------|
| `-n` | 订单数量 | 1000 |
| `-c` | 并发数 | 8 |
| `-url` | API 地址 | http://localhost:8080 |
| `-market` | 交易对 | BTC-USDC |

## 压测结果

| 指标 | 结果 |
|------|------|
| 订单数 | 5,000 |
| 成功率 | 100% |
| 吞吐量 | 231 orders/sec |
| P50 延迟 | 34.5 ms |
| P99 延迟 | 67.6 ms |

## API 端点

### 下单
```bash
curl -X POST http://localhost:8080/v1/orders \
  -H "Content-Type: application/json" \
  -H "X-Trader-Address: alice" \
  -d '{"market_id":"BTC-USDC","side":"buy","type":"limit","price":"95000","quantity":"1.0"}'
```

### 查询订单
```bash
curl http://localhost:8080/v1/orders -H "X-Trader-Address: alice"
```

### 查询订单簿
```bash
curl http://localhost:8081/orderbook/BTC-USDC
```

### 取消订单
```bash
curl -X DELETE http://localhost:8080/v1/orders/order-1 -H "X-Trader-Address: alice"
```

## E2E 测试

```bash
# 运行所有 E2E 测试
go test -v ./tests/e2e_api/...

# 单独运行
go test -v -run "TestOrderMatchingHTTP" ./tests/e2e_api/
go test -v -run "TestHighThroughputHTTP" ./tests/e2e_api/
go test -v -run "TestConcurrentHTTP" ./tests/e2e_api/
```
