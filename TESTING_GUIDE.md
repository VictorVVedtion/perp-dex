# PerpDEX 测试与演示指南

## 目录
1. [环境准备](#1-环境准备)
2. [快速启动](#2-快速启动)
3. [单元测试](#3-单元测试)
4. [功能演示](#4-功能演示)
5. [API 测试](#5-api-测试)
6. [前端演示](#6-前端演示)
7. [Docker 部署测试](#7-docker-部署测试)

---

## 1. 环境准备

### 必需软件
```bash
# Go 1.22+
go version  # 需要 go1.22.11 或更高

# Node.js 18+ (前端)
node --version

# Docker (可选, 用于容器化部署)
docker --version
docker-compose --version
```

### 克隆和依赖安装
```bash
cd "/Users/vvedition/Desktop/dex mvp/perp-dex_副本"

# 后端依赖
go mod download

# 前端依赖
cd frontend
npm install
cd ..
```

---

## 2. 快速启动

### 方式 A: 使用 Makefile (推荐)
```bash
cd "/Users/vvedition/Desktop/dex mvp/perp-dex_副本"

# 编译
make build

# 初始化链 + 启动节点 (一键)
make run-local
```

### 方式 B: 手动启动
```bash
# 1. 编译二进制文件
go build -o ./build/perpdexd ./cmd/perpdexd

# 2. 初始化链
./scripts/init-chain.sh

# 3. 启动节点
./scripts/start-node.sh
```

### 验证节点运行
```bash
# 检查节点状态
curl http://localhost:26657/status | jq '.result.sync_info'

# 检查最新区块
curl http://localhost:26657/block | jq '.result.block.header.height'
```

---

## 3. 单元测试

### 运行所有测试
```bash
make test
# 或
go test -v ./...
```

### 运行特定模块测试
```bash
# 订单簿模块
go test -v ./x/orderbook/keeper/...

# 永续合约模块
go test -v ./x/perpetual/keeper/...

# 清算模块
go test -v ./x/clearinghouse/keeper/...
```

### 性能基准测试
```bash
# 撮合引擎性能测试
go test -bench=. ./x/orderbook/keeper/benchmark_test.go

# 并行撮合测试
go test -v ./x/orderbook/keeper/parallel_test.go
```

### 测试清单
| 测试文件 | 测试内容 |
|----------|----------|
| `parallel_test.go` | 并行撮合引擎 |
| `trailing_stop_test.go` | 追踪止损订单 |
| `oco_test.go` | OCO 订单 |
| `benchmark_test.go` | 性能基准 |
| `liquidation_v2_test.go` | 三层清算机制 |
| `market_test.go` | 市场管理 |
| `funding_test.go` | 资金费率 |

---

## 4. 功能演示

### 4.1 CLI 交易演示

**启动节点后, 打开新终端:**

```bash
cd "/Users/vvedition/Desktop/dex mvp/perp-dex_副本"
export HOME_DIR=$HOME/.perpdex

# 查看测试账户
perpdexd keys list --home $HOME_DIR --keyring-backend test

# 查看账户余额
TRADER1=$(perpdexd keys show trader1 -a --home $HOME_DIR --keyring-backend test)
echo "Trader1 地址: $TRADER1"
```

**查询市场:**
```bash
# 查询所有市场
perpdexd query perpetual markets --home $HOME_DIR

# 查询 BTC-USDC 市场
perpdexd query perpetual market BTC-USDC --home $HOME_DIR

# 查询当前价格
perpdexd query perpetual price BTC-USDC --home $HOME_DIR
```

**查询订单簿:**
```bash
# 查询订单簿深度
perpdexd query orderbook depth BTC-USDC --home $HOME_DIR

# 查询最近成交
perpdexd query orderbook trades BTC-USDC --limit 10 --home $HOME_DIR
```

**下单交易:**
```bash
# 下限价买单 (做多 0.1 BTC @ $50000, 10x 杠杆)
perpdexd tx orderbook place-order \
    BTC-USDC \
    buy \
    limit \
    50000 \
    0.1 \
    --leverage 10 \
    --from trader1 \
    --home $HOME_DIR \
    --keyring-backend test \
    --chain-id perpdex-1 \
    -y

# 下市价卖单 (做空)
perpdexd tx orderbook place-order \
    BTC-USDC \
    sell \
    market \
    0 \
    0.05 \
    --leverage 5 \
    --from trader2 \
    --home $HOME_DIR \
    --keyring-backend test \
    --chain-id perpdex-1 \
    -y
```

**查询仓位:**
```bash
# 查询 trader1 的仓位
perpdexd query perpetual position $TRADER1 BTC-USDC --home $HOME_DIR

# 查询所有仓位
perpdexd query perpetual positions --home $HOME_DIR
```

### 4.2 清算演示

```bash
# 查询仓位健康度
perpdexd query clearinghouse health $TRADER1 BTC-USDC --home $HOME_DIR

# 查询濒临清算的仓位
perpdexd query clearinghouse at-risk --home $HOME_DIR

# 查询保险基金状态
perpdexd query clearinghouse insurance-fund --home $HOME_DIR

# 查询 ADL 排名
perpdexd query clearinghouse adl-ranking BTC-USDC --home $HOME_DIR
```

### 4.3 资金费率演示

```bash
# 查询资金费率
perpdexd query perpetual funding BTC-USDC --home $HOME_DIR

# 查看资金结算时间 (8小时周期)
# 资金费率会在 00:00, 08:00, 16:00 UTC 自动结算
```

---

## 5. API 测试

### 启动 API 服务器
API 服务器已内置在节点中，启动节点后自动可用。

### REST API 测试

**市场数据:**
```bash
# 获取所有市场
curl http://localhost:1317/perpdex/perpetual/v1/markets | jq

# 获取 BTC-USDC 市场
curl http://localhost:1317/perpdex/perpetual/v1/markets/BTC-USDC | jq

# 获取价格
curl http://localhost:1317/perpdex/perpetual/v1/price/BTC-USDC | jq
```

**订单簿数据:**
```bash
# 获取订单簿
curl http://localhost:1317/perpdex/orderbook/v1/depth/BTC-USDC | jq

# 获取最近成交
curl http://localhost:1317/perpdex/orderbook/v1/trades/BTC-USDC | jq
```

**账户数据:**
```bash
TRADER1=$(perpdexd keys show trader1 -a --home $HOME/.perpdex --keyring-backend test)

# 获取账户信息
curl "http://localhost:1317/perpdex/perpetual/v1/account/$TRADER1" | jq

# 获取仓位
curl "http://localhost:1317/perpdex/perpetual/v1/positions/$TRADER1" | jq
```

### WebSocket 测试

使用 `websocat` 或浏览器控制台:

```bash
# 安装 websocat (macOS)
brew install websocat

# 连接 WebSocket
websocat ws://localhost:8080/ws
```

**订阅频道:**
```json
{"type": "subscribe", "channel": "ticker:BTC-USDC"}
{"type": "subscribe", "channel": "depth:BTC-USDC"}
{"type": "subscribe", "channel": "trades:BTC-USDC"}
```

**期望收到的消息:**
```json
{
  "type": "ticker",
  "channel": "ticker:BTC-USDC",
  "data": {
    "marketId": "BTC-USDC",
    "lastPrice": "50000.00",
    "markPrice": "50000.00",
    "indexPrice": "50000.00",
    "change24h": "+2.5%",
    "volume24h": "125000000"
  }
}
```

---

## 6. 前端演示

### 启动前端开发服务器
```bash
cd "/Users/vvedition/Desktop/dex mvp/perp-dex_副本/frontend"

# 开发模式启动
npm run dev
```

### 访问地址
- **前端**: http://localhost:3001
- **后端 RPC**: http://localhost:26657
- **后端 API**: http://localhost:1317

### 前端功能演示

1. **交易界面** (`/`)
   - 查看实时 K 线图表
   - 查看订单簿深度
   - 查看最近成交
   - 下限价/市价单

2. **钱包连接**
   - 点击 "Connect Wallet"
   - 如果没有 Keplr，会自动使用 Mock 模式
   - Mock 模式下会模拟交易

3. **仓位管理** (`/positions`)
   - 查看当前仓位
   - 查看未实现盈亏
   - 平仓操作

### Mock 模式测试

在 `.env.local` 中启用 Mock 模式:
```bash
cd frontend
echo "NEXT_PUBLIC_MOCK_MODE=true" >> .env.local
npm run dev
```

Mock 模式下:
- 无需启动后端节点
- 使用模拟数据
- 交易会生成模拟 TxHash

---

## 7. Docker 部署测试

### 单节点测试
```bash
cd "/Users/vvedition/Desktop/dex mvp/perp-dex_副本/deploy"

# 构建镜像
docker-compose build perpdex-node-0

# 启动单节点
docker-compose up perpdex-node-0
```

### 完整集群测试
```bash
# 启动 3 节点集群 + 监控
docker-compose up -d

# 查看服务状态
docker-compose ps

# 查看日志
docker-compose logs -f perpdex-node-0
```

### 访问服务
| 服务 | 端口 | URL |
|------|------|-----|
| Node 0 RPC | 26657 | http://localhost:26657 |
| Node 0 API | 1317 | http://localhost:1317 |
| WebSocket | 8080 | ws://localhost:8080/ws |
| Prometheus | 9099 | http://localhost:9099 |
| Grafana | 3000 | http://localhost:3000 |

### Grafana 登录
- URL: http://localhost:3000
- 用户名: `admin`
- 密码: `perpdex123`

### 清理
```bash
# 停止所有服务
docker-compose down

# 清理数据卷
docker-compose down -v
```

---

## 演示场景脚本

### 场景 1: 完整交易流程
```bash
#!/bin/bash
# demo_trading.sh

# 1. 查询初始状态
echo "=== 初始市场状态 ==="
perpdexd query perpetual market BTC-USDC --home $HOME/.perpdex

# 2. Trader1 下买单
echo "=== Trader1 下买单 ==="
perpdexd tx orderbook place-order BTC-USDC buy limit 50000 0.1 \
    --leverage 10 --from trader1 --home $HOME/.perpdex \
    --keyring-backend test --chain-id perpdex-1 -y

sleep 3

# 3. Trader2 下卖单 (匹配)
echo "=== Trader2 下卖单 ==="
perpdexd tx orderbook place-order BTC-USDC sell limit 50000 0.1 \
    --leverage 10 --from trader2 --home $HOME/.perpdex \
    --keyring-backend test --chain-id perpdex-1 -y

sleep 3

# 4. 查询成交
echo "=== 查询成交 ==="
perpdexd query orderbook trades BTC-USDC --limit 5 --home $HOME/.perpdex

# 5. 查询仓位
echo "=== 查询仓位 ==="
TRADER1=$(perpdexd keys show trader1 -a --home $HOME/.perpdex --keyring-backend test)
perpdexd query perpetual position $TRADER1 BTC-USDC --home $HOME/.perpdex
```

### 场景 2: 清算演示
```bash
#!/bin/bash
# demo_liquidation.sh

# 模拟价格剧烈波动导致清算
# (需要手动调整 Oracle 价格或等待 EndBlocker 自动处理)

echo "=== 查询濒临清算仓位 ==="
perpdexd query clearinghouse at-risk --home $HOME/.perpdex

echo "=== 保险基金状态 ==="
perpdexd query clearinghouse insurance-fund --home $HOME/.perpdex

echo "=== ADL 排名 ==="
perpdexd query clearinghouse adl-ranking BTC-USDC --home $HOME/.perpdex
```

---

## 常见问题

### Q: 节点启动失败?
```bash
# 检查端口占用
lsof -i :26656 -i :26657 -i :1317

# 重新初始化
rm -rf $HOME/.perpdex
./scripts/init-chain.sh
```

### Q: 前端连接后端失败?
```bash
# 确保后端 CORS 已启用
# start-node.sh 中已包含 --api.enabled-unsafe-cors

# 检查 API 是否可访问
curl http://localhost:1317/perpdex/perpetual/v1/markets
```

### Q: Docker 构建失败?
```bash
# 清理 Docker 缓存
docker system prune -f
docker-compose build --no-cache
```

---

*指南最后更新: 2026-01-19*
