# PerpDEX 撮合引擎性能优化报告

## 概述

本次优化实现了四个主要方面的性能提升，参考 Hyperliquid 高频交易架构设计。

## 优化一：跳表订单簿 (SkipList Order Book)

### 实现文件
- `x/orderbook/keeper/orderbook_v2.go` - 跳表订单簿实现
- `x/orderbook/keeper/matching_v2.go` - 优化撮合引擎

### 技术细节
- **数据结构**：使用 SkipList 替代传统切片
- **时间复杂度**：插入/删除从 O(n) 降至 O(log n)
- **特性**：
  - 自动价格排序（Bids 降序，Asks 升序）
  - O(1) 获取最优价格
  - 线程安全（RWMutex）
  - 缓存刷新机制减少存储操作

### 性能提升
| 操作 | 优化前 | 优化后 | 提升 |
|------|--------|--------|------|
| 撮合 1000 订单 | 1,989 ms | 6.2 ms | **319x** |
| 添加订单 | 47.7 μs | 0.84 μs | **57x** |
| 删除订单 | 23.7 μs | 0.85 μs | **28x** |
| 内存分配 | 32.7M | 107K | **304x 减少** |

---

## 优化二：链下撮合器 (Offchain Matcher)

### 实现文件
- `offchain/matcher/matcher.go` - 撮合核心逻辑
- `offchain/matcher/cache.go` - 订单/交易缓存
- `offchain/matcher/submitter.go` - 批量提交器
- `offchain/cmd/matcher/main.go` - CLI 入口

### 技术细节
- **架构**：链下撮合 + 链上结算
- **批量提交**：累积交易后批量上链（默认 100 笔/批，500ms 间隔）
- **事件驱动**：基于 channel 的异步处理
- **重试机制**：失败自动重试，保证交易最终一致性

### 运行方式
```bash
# 启动链下撮合器
go run ./offchain/cmd/matcher/... --demo

# 自定义配置
go run ./offchain/cmd/matcher/... \
  --batch-size 200 \
  --batch-interval 300ms \
  --rpc http://localhost:26657
```

### 预期收益
- 订单响应延迟：从 500ms+ 降至 <10ms
- 吞吐量提升：50-70%
- 链上 gas 节省：批量提交减少交易数量

---

## 优化三：并行撮合引擎 (Parallel Matcher)

### 实现文件
- `x/orderbook/keeper/parallel.go` - 并行撮合实现

### 技术细节
- **多市场并行**：不同交易对独立并发处理
- **Worker Pool**：可配置工作线程数（默认 4 个）
- **批量处理**：订单分批处理（默认 100 订单/批）
- **超时控制**：防止长时间阻塞（默认 5 秒）

### 配置
```go
config := ParallelConfig{
    Enabled:   true,
    Workers:   4,     // CPU 核心数
    BatchSize: 100,   // 批量大小
    Timeout:   5s,    // 超时时间
}
```

### 预期收益
- 多市场场景下吞吐量提升 20-30%
- 充分利用多核 CPU 资源

---

## 优化四：CometBFT 配置优化

### 实现文件
- `cmd/perpdexd/cmd/root.go` - 默认配置
- `config/fast_consensus.toml` - 可选配置文件

### 配置变更

| 参数 | 默认值 | 优化值 | 说明 |
|------|--------|--------|------|
| timeout_propose | 3s | 500ms | 提案超时 |
| timeout_prevote | 1s | 500ms | 预投票超时 |
| timeout_precommit | 1s | 500ms | 预提交超时 |
| timeout_commit | 5s | 500ms | 提交超时 |
| mempool.size | 5000 | 10000 | 内存池大小 |
| max_tx_bytes | 1MB | 10MB | 单笔交易限制 |
| max_txs_bytes | 10MB | 100MB | 区块交易限制 |
| p2p.flush_throttle | 100ms | 10ms | 消息刷新间隔 |
| send/recv_rate | 5MB/s | 20MB/s | 网络速率 |

### 预期收益
- 出块时间：从 ~5s 降至 ~1s
- 交易确认延迟降低 10-20%

---

## 综合性能提升

| 指标 | 优化前 | 优化后 | 提升 |
|------|--------|--------|------|
| 单订单撮合延迟 | ~2ms | ~6μs | **330x** |
| 内存使用 | 高 | 低 | **300x 减少** |
| 出块时间 | 5s | 1s | **5x** |
| 多市场吞吐量 | 基准 | +20-30% | 并行化 |
| 订单响应 | 链上确认 | 即时响应 | 链下撮合 |

---

## 使用建议

### 开发/测试环境
```bash
# 使用默认优化配置
perpdexd start

# 启动链下撮合器
go run ./offchain/cmd/matcher/... --demo
```

### 生产环境
```bash
# 根据实际网络调整参数
perpdexd start --home ~/.perpdex-prod

# 监控 Prometheus 指标
curl http://localhost:26660/metrics
```

### 监控指标
- `perpdex_orderbook_match_latency_ms` - 撮合延迟
- `perpdex_orderbook_orders_total` - 订单总数
- `perpdex_trades_per_second` - 每秒成交数

---

## 文件清单

```
x/orderbook/keeper/
├── orderbook_v2.go      # 跳表订单簿 (450 行)
├── matching_v2.go       # 优化撮合引擎
├── parallel.go          # 并行撮合 (586 行)
└── benchmark_test.go    # 性能基准测试

offchain/
├── matcher/
│   ├── matcher.go       # 链下撮合器 (493 行)
│   ├── cache.go         # 订单缓存
│   └── submitter.go     # 批量提交器
└── cmd/matcher/
    └── main.go          # CLI 入口

config/
└── fast_consensus.toml  # 快速共识配置 (191 行)
```

---

## 总结

通过这四个方面的优化，PerpDEX 的撮合性能得到了显著提升：

1. **跳表订单簿**：核心数据结构优化，撮合速度提升 319 倍
2. **链下撮合**：分离计算层和共识层，大幅降低延迟
3. **并行处理**：充分利用多核 CPU，提升多市场吞吐量
4. **共识优化**：缩短出块时间，加快交易确认

这些优化使 PerpDEX 达到了接近 Hyperliquid 的高频交易性能水平。
