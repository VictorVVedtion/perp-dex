# PerpDEX 链上 E2E 测试方案

## 一、测试概述

### 1.1 测试目标
- 验证链的基础功能（启动、出块、共识）
- 验证交易消息的完整生命周期
- 验证订单簿模块的核心功能
- 验证永续合约模块的保证金管理
- 验证多用户交互场景
- 验证错误处理和边界条件

### 1.2 测试环境
- 单节点测试网络
- Chain ID: `perpdex-test-1`
- 测试账户: validator, trader1, trader2, trader3
- 初始资金: 每个trader账户 1,000,000 usdc

### 1.3 测试框架
- Go testing + testify/require
- CLI 命令驱动（通过 perpdexd 二进制）
- 异步事件等待机制

---

## 二、测试分类

### 2.1 基础设施测试 (Infrastructure)
| 测试ID | 测试名称 | 测试描述 |
|--------|----------|----------|
| INF-001 | 节点连接性 | 验证能够连接到节点 RPC |
| INF-002 | 出块验证 | 验证链正常出块 |
| INF-003 | 区块高度递增 | 验证区块高度持续增长 |
| INF-004 | 状态同步 | 验证状态查询一致性 |

### 2.2 账户测试 (Account)
| 测试ID | 测试名称 | 测试描述 |
|--------|----------|----------|
| ACC-001 | 账户地址查询 | 验证账户地址格式正确 |
| ACC-002 | 初始余额验证 | 验证账户初始余额正确 |
| ACC-003 | 密钥管理 | 验证密钥导入导出功能 |
| ACC-004 | 多账户切换 | 验证不同账户操作隔离 |

### 2.3 订单簿测试 (Orderbook)
| 测试ID | 测试名称 | 测试描述 |
|--------|----------|----------|
| ORD-001 | 限价买单 | 提交限价买单并验证 |
| ORD-002 | 限价卖单 | 提交限价卖单并验证 |
| ORD-003 | 市价买单 | 提交市价买单并验证 |
| ORD-004 | 市价卖单 | 提交市价卖单并验证 |
| ORD-005 | 订单取消 | 取消未成交订单 |
| ORD-006 | 订单匹配 | 买卖订单自动撮合 |
| ORD-007 | 部分成交 | 大订单部分成交场景 |
| ORD-008 | 订单查询 | 查询订单状态和详情 |
| ORD-009 | 订单簿深度 | 查询市场深度数据 |
| ORD-010 | 重复订单ID | 验证订单ID唯一性 |

### 2.4 保证金测试 (Margin)
| 测试ID | 测试名称 | 测试描述 |
|--------|----------|----------|
| MAR-001 | 保证金存入 | 存入保证金到交易账户 |
| MAR-002 | 保证金提取 | 从交易账户提取保证金 |
| MAR-003 | 余额查询 | 查询保证金余额 |
| MAR-004 | 超额提取 | 尝试提取超过余额的金额 |
| MAR-005 | 负数金额 | 尝试存入/提取负数金额 |

### 2.5 仓位测试 (Position)
| 测试ID | 测试名称 | 测试描述 |
|--------|----------|----------|
| POS-001 | 开多仓 | 通过买单开多头仓位 |
| POS-002 | 开空仓 | 通过卖单开空头仓位 |
| POS-003 | 仓位查询 | 查询当前持仓信息 |
| POS-004 | 平仓操作 | 平掉现有仓位 |
| POS-005 | 仓位合并 | 同向订单合并仓位 |

### 2.6 清算测试 (Clearinghouse)
| 测试ID | 测试名称 | 测试描述 |
|--------|----------|----------|
| CLR-001 | 资金结算 | 验证交易资金结算 |
| CLR-002 | 手续费扣除 | 验证交易手续费计算 |
| CLR-003 | PnL 计算 | 验证盈亏计算正确性 |

### 2.7 错误处理测试 (Error Handling)
| 测试ID | 测试名称 | 测试描述 |
|--------|----------|----------|
| ERR-001 | 无效交易对 | 提交不存在的交易对订单 |
| ERR-002 | 无效价格 | 提交价格为0或负数的订单 |
| ERR-003 | 无效数量 | 提交数量为0或负数的订单 |
| ERR-004 | 未授权操作 | 尝试取消他人订单 |
| ERR-005 | 余额不足 | 余额不足时下单 |
| ERR-006 | 重复取消 | 重复取消同一订单 |

### 2.8 并发测试 (Concurrency)
| 测试ID | 测试名称 | 测试描述 |
|--------|----------|----------|
| CON-001 | 并发下单 | 多用户同时下单 |
| CON-002 | 高频交易 | 快速连续提交订单 |
| CON-003 | 并发取消 | 多用户同时取消订单 |

---

## 三、测试用例详细设计

### 3.1 INF-001: 节点连接性测试

```go
func TestNodeConnectivity(t *testing.T) {
    // 1. 调用 perpdexd status
    // 2. 验证返回节点信息
    // 3. 验证 chain_id = "perpdex-test-1"
}
```

**预期结果**: 成功返回节点状态，chain_id 正确

### 3.2 ORD-001: 限价买单测试

```go
func TestLimitBuyOrder(t *testing.T) {
    // 1. trader1 提交限价买单: BTC-USDC, buy, limit, 50000, 0.1
    // 2. 等待交易确认
    // 3. 查询订单状态
    // 4. 验证订单已提交
}
```

**预期结果**: 订单成功提交，状态为 pending

### 3.3 ORD-006: 订单匹配测试

```go
func TestOrderMatching(t *testing.T) {
    // 1. trader1 提交限价买单: price=50000, qty=0.1
    // 2. trader2 提交限价卖单: price=50000, qty=0.1
    // 3. 等待撮合
    // 4. 验证两订单都已成交
    // 5. 验证仓位变化
}
```

**预期结果**: 订单完全成交，双方仓位更新

### 3.4 MAR-001: 保证金存入测试

```go
func TestMarginDeposit(t *testing.T) {
    // 1. 查询初始保证金余额
    // 2. 存入 10000 usdc
    // 3. 查询新余额
    // 4. 验证余额增加 10000
}
```

**预期结果**: 保证金余额正确增加

---

## 四、测试执行流程

### 4.1 前置条件
```bash
# 1. 编译项目
make build

# 2. 初始化测试链
./scripts/init-chain.sh

# 3. 启动链（后台）
./build/perpdexd start --home .perpdex-test --minimum-gas-prices "0usdc" &

# 4. 等待链启动
sleep 5
```

### 4.2 运行测试
```bash
# 运行所有 E2E 测试
go test -v ./tests/e2e_chain/... -count=1

# 运行特定测试
go test -v ./tests/e2e_chain/... -run TestOrderbook

# 运行并生成覆盖率报告
go test -v ./tests/e2e_chain/... -coverprofile=coverage.out
```

### 4.3 清理环境
```bash
# 停止链
pkill perpdexd

# 清理数据
rm -rf .perpdex-test
```

---

## 五、测试报告模板

### 5.1 测试执行摘要
| 指标 | 数值 |
|------|------|
| 总测试数 | XX |
| 通过 | XX |
| 失败 | XX |
| 跳过 | XX |
| 执行时间 | XXs |

### 5.2 失败用例分析
| 测试ID | 失败原因 | 修复建议 |
|--------|----------|----------|
| XXX-00X | 描述 | 建议 |

---

## 六、持续集成配置

```yaml
# .github/workflows/e2e-test.yml
name: E2E Tests

on: [push, pull_request]

jobs:
  e2e-test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
        with:
          go-version: '1.21'
      - name: Build
        run: make build
      - name: Init Chain
        run: ./scripts/init-chain.sh
      - name: Start Chain
        run: |
          ./build/perpdexd start --home .perpdex-test --minimum-gas-prices "0usdc" &
          sleep 10
      - name: Run E2E Tests
        run: go test -v ./tests/e2e_chain/... -count=1
```
