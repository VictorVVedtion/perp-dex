# PerpDEX E2E Real Tests

真正的端到端测试套件，测试从 HTTP 请求到 WebSocket 推送的完整链路。

## 测试覆盖

| 测试文件 | 测试类别 | 测试数量 | 说明 |
|---------|---------|---------|------|
| `trading_flow_test.go` | 交易流程 | 8 | 下单、撮合、取消、深度 |
| `websocket_test.go` | WebSocket | 10 | 连接、订阅、推送、重连 |
| `liquidation_test.go` | 清算流程 | 10 | 清算、保险基金、ADL |
| `concurrent_test.go` | 并发测试 | 6 | 多用户、竞态、负载 |

## 快速开始

### 1. 启动 API 服务器

```bash
# 在项目根目录
cd /path/to/perp-dex
go run ./cmd/api --port 8080 --mock
```

### 2. 运行测试

```bash
# 进入测试目录
cd tests/e2e_real

# 运行所有测试
go test -v ./...

# 或使用自动化脚本
./run_e2e.sh --start-server --report
```

## 测试详情

### 交易流程测试 (trading_flow_test.go)

- `TestTradingFlow_PlaceLimitOrder` - 限价单下单
- `TestTradingFlow_PlaceMarketOrder` - 市价单下单
- `TestTradingFlow_GetOrderBook` - 获取订单簿
- `TestTradingFlow_CancelOrder` - 取消订单
- `TestTradingFlow_OrderMatching` - 订单撮合
- `TestTradingFlow_GetMarkets` - 获取市场列表
- `TestTradingFlow_GetTicker` - 获取行情数据
- `TestTradingFlow_LatencyBenchmark` - 延迟基准测试

### WebSocket 测试 (websocket_test.go)

- `TestWebSocket_Connect` - WebSocket 连接
- `TestWebSocket_Subscribe` - 频道订阅
- `TestWebSocket_TickerUpdates` - Ticker 更新
- `TestWebSocket_OrderBookUpdates` - 订单簿更新
- `TestWebSocket_TradeUpdates` - 成交推送
- `TestWebSocket_MultipleChannels` - 多频道订阅
- `TestWebSocket_Unsubscribe` - 取消订阅
- `TestWebSocket_Reconnect` - 重连测试
- `TestWebSocket_MessageLatency` - 消息延迟
- `TestWebSocket_HighFrequency` - 高频消息

### 清算测试 (liquidation_test.go)

- `TestLiquidation_PositionAtRisk` - 风险仓位
- `TestLiquidation_GetLiquidablePositions` - 获取可清算仓位
- `TestLiquidation_ExecuteLiquidation` - 执行清算
- `TestLiquidation_GetInsuranceFund` - 保险基金状态
- `TestLiquidation_ADLQueue` - ADL 队列
- `TestLiquidation_MarginCall` - 追加保证金
- `TestLiquidation_PartialLiquidation` - 部分清算
- `TestLiquidation_LiquidationPriceCalculation` - 清算价计算
- `TestLiquidation_BatchLiquidation` - 批量清算
- `TestLiquidation_WebSocketNotifications` - 清算通知

### 并发测试 (concurrent_test.go)

- `TestConcurrent_MultipleTraders` - 多交易者并发
- `TestConcurrent_RaceCondition` - 竞态条件测试
- `TestConcurrent_WebSocketConnections` - 并发 WebSocket 连接
- `TestConcurrent_OrderCancellation` - 并发下单取消
- `TestConcurrent_LoadTest` - 负载测试
- `TestConcurrent_HighFrequencyTrading` - HFT 模拟

## 运行脚本选项

```bash
./run_e2e.sh [OPTIONS]

Options:
  --start-server    启动 API 服务器后再运行测试
  --stop-server     测试后停止服务器
  --report          生成详细报告
  --verbose         详细输出
  --help            显示帮助
```

## 与旧测试的对比

| 对比项 | 旧 "E2E" 测试 | 新 真正 E2E 测试 |
|--------|-------------|-----------------|
| 服务器 | 内存 Mock | 真实 HTTP 服务器 |
| HTTP 请求 | ❌ 不测试 | ✅ 完整测试 |
| WebSocket | ❌ 不测试 | ✅ 完整测试 |
| 多用户并发 | ❌ 有限 | ✅ 10-100 用户 |
| 实际延迟 | ❌ 内部操作 | ✅ 真实网络延迟 |
| 清算流程 | ❌ Mock | ✅ 完整流程 |

## 测试框架 (framework.go)

提供以下基础设施：

- `HTTPClient` - HTTP 请求客户端，带延迟统计
- `WSClient` - WebSocket 客户端
- `TestAccount` - 测试账户管理
- `LatencyReport` - 延迟报告生成

## 配置

默认配置：

```go
config := &TestConfig{
    APIBaseURL: "http://localhost:8080",
    WSBaseURL:  "ws://localhost:8080",
    Timeout:    10 * time.Second,
}
```

可通过环境变量覆盖：

```bash
export E2E_API_URL=http://localhost:8080
export E2E_WS_URL=ws://localhost:8080
```

## 注意事项

1. 测试需要运行中的 API 服务器
2. 如果服务器未运行，测试会自动跳过 (Skip)
3. 并发测试可能需要更长时间
4. 负载测试会生成大量请求

## 示例输出

```
=== RUN   TestTradingFlow_PlaceLimitOrder
    trading_flow_test.go:35: Order placed successfully, latency: 2.1ms
--- PASS: TestTradingFlow_PlaceLimitOrder (0.01s)

=== RUN   TestConcurrent_LoadTest
    concurrent_test.go:180: Load Test Results:
    concurrent_test.go:181:   Duration: 10s
    concurrent_test.go:182:   Workers: 10
    concurrent_test.go:183:   Total Requests: 15234
    concurrent_test.go:184:   Errors: 0
    concurrent_test.go:185:   Throughput: 1523.40 req/sec
========================================
E2E Test Report: Concurrent Load Test
========================================
Total Requests: 15234
Average Latency: 654µs
P50 Latency: 512µs
P95 Latency: 1.2ms
P99 Latency: 2.1ms
========================================
--- PASS: TestConcurrent_LoadTest (10.02s)
```
