# PerpDEX Performance Tuning Guide

This document describes the performance optimizations implemented in PerpDEX for high-frequency trading with sub-second block times.

## Overview

PerpDEX is optimized for:
- **Sub-second block times**: ~500ms target block time
- **High throughput**: 10,000+ transactions per block
- **Low latency matching**: <100ms EndBlocker execution
- **Fast finality**: Single-block finality

## CometBFT Configuration

### Consensus Timeouts

The consensus timing has been optimized for fast block production:

| Parameter | Default | Optimized | Description |
|-----------|---------|-----------|-------------|
| `timeout_propose` | 3s | 500ms | Time to wait for block proposal |
| `timeout_propose_delta` | 500ms | 100ms | Increase per round |
| `timeout_prevote` | 1s | 500ms | Time to wait for prevotes |
| `timeout_prevote_delta` | 500ms | 100ms | Increase per round |
| `timeout_precommit` | 1s | 500ms | Time to wait for precommits |
| `timeout_precommit_delta` | 500ms | 100ms | Increase per round |
| `timeout_commit` | 5s | 500ms | Time between blocks |

### Mempool Configuration

| Parameter | Default | Optimized | Description |
|-----------|---------|-----------|-------------|
| `size` | 5000 | 10000 | Max transactions in mempool |
| `max_tx_bytes` | 1MB | 10MB | Max single transaction size |
| `max_txs_bytes` | 1GB | 100MB | Max total mempool size |
| `recheck` | true | true | Recheck transactions after commit |

### P2P Configuration

| Parameter | Default | Optimized | Description |
|-----------|---------|-----------|-------------|
| `flush_throttle_timeout` | 100ms | 10ms | Message flush interval |
| `send_rate` | 5MB/s | 20MB/s | Outbound rate limit |
| `recv_rate` | 5MB/s | 20MB/s | Inbound rate limit |
| `max_packet_msg_payload_size` | 1024 | 10240 | Max packet payload |

## Applying Configuration

### Method 1: Code-level (Default)

The optimized configuration is applied automatically via `initCometBFTConfig()` in `cmd/perpdexd/cmd/root.go`. This takes effect when initializing a new node.

### Method 2: Configuration Script

For existing nodes, use the provided script:

```bash
# Dry run - see what changes would be made
./scripts/apply_fast_config.sh --dry-run

# Create backup only
./scripts/apply_fast_config.sh --backup

# Apply fast configuration
./scripts/apply_fast_config.sh

# Restore previous configuration
./scripts/apply_fast_config.sh --restore
```

### Method 3: Manual Configuration

Edit `~/.perpdex/config/config.toml` directly using the template in `config/fast_consensus.toml`.

## Performance Monitoring

### EndBlocker Metrics

The EndBlocker logs performance metrics for each block:

```
INFO EndBlocker performance block=1234 total_ms=45 oracle_ms=5 matching_ms=35 liquidation_ms=5
```

### Matching Engine Statistics

When orders are processed:

```
INFO Matching engine stats block=1234 orders_processed=150 trades_executed=75 volume=1500000.00 avg_latency_us=234
```

### Liquidation Statistics

When liquidations occur:

```
INFO Liquidation stats block=1234 liquidations=3 volume=450000.00
```

### Performance Alerts

A warning is logged if EndBlocker exceeds 100ms:

```
WARN EndBlocker exceeded latency threshold block=1234 duration_ms=150 threshold_ms=100
```

## Benchmark Results

### Single Validator (Local)

| Metric | Value |
|--------|-------|
| Block Time | ~500ms |
| Orders/Block | 1,000+ |
| Matching Latency | <50ms |
| EndBlocker Time | <100ms |

### Multi-Validator (Testnet)

| Validators | Block Time | Orders/Block | Notes |
|------------|------------|--------------|-------|
| 1 | ~500ms | 1,000+ | Single node, fastest |
| 4 | ~800ms | 800+ | Low network latency |
| 7 | ~1.2s | 500+ | Geographically distributed |

## Tuning Recommendations

### Development / Single Node

Use the aggressive defaults:
- `timeout_commit`: 500ms
- Best for testing and development

### Testnet / Low-latency Network

Slightly more conservative:
- `timeout_commit`: 1s
- `timeout_propose`: 1s

### Production / Mainnet

Depends on validator distribution:
- `timeout_commit`: 1-2s
- `timeout_propose`: 1-2s
- Monitor block times and adjust

## Troubleshooting

### Slow Block Times

1. Check network latency between validators
2. Increase timeouts if validators are geographically distributed
3. Monitor EndBlocker execution time

### Mempool Congestion

1. Increase `mempool.size` if transactions are being dropped
2. Check transaction size limits
3. Monitor mempool size via RPC

### High CPU Usage

1. Reduce P2P rates if bandwidth is limited
2. Increase `flush_throttle_timeout`
3. Consider reducing `mempool.recheck` frequency

## Architecture Notes

### Order Matching Pipeline

```
BeginBlocker
    |
    v
DeliverTx (validate & queue orders)
    |
    v
EndBlocker
    |
    +-> Oracle Price Update (~5ms)
    |
    +-> Order Matching (~50ms)
    |       |
    |       +-> Get pending orders
    |       +-> Match against order book
    |       +-> Update positions
    |       +-> Record trades
    |
    +-> Liquidation Check (~20ms)
    |       |
    |       +-> Scan unhealthy positions
    |       +-> Execute liquidations
    |
    v
Commit
```

### Performance Considerations

1. **Order Book Structure**: Uses price-time priority with O(log n) lookups
2. **State Access**: Minimizes KVStore reads/writes
3. **Batch Processing**: Orders processed in batches per market
4. **Event Emission**: Async event emission for trades

## Future Optimizations

1. **Parallel Matching**: Process multiple markets concurrently
2. **State Caching**: In-memory order book caching
3. **Compact Encoding**: More efficient order serialization
4. **Merkle Tree Optimization**: Faster state commitment
