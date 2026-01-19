# PerpDEX Order Book E2E Stress Test Report

**Generated**: 2026-01-19
**Platform**: macOS (Apple M4 Pro, 14 cores)
**Go Version**: 1.23+

---

## Executive Summary

This report presents comprehensive E2E stress testing results for four order book data structure implementations in PerpDEX:

1. **Skip List** (OrderBookV2) - Current production implementation
2. **HashMap + Heap** (dYdX style) - O(1) price lookup
3. **B+ Tree** (Bybit/CEX style) - Range query optimized
4. **ART** (Adaptive Radix Tree) - Memory efficient

### Key Findings

| Implementation | Throughput (ops/s) | P99 Latency | Memory Efficiency | Recommendation |
|----------------|-------------------|-------------|-------------------|----------------|
| **B+ Tree** | 4,457,453 | 542 ns | Best (6.1 MB) | **Production Ready** |
| **Skip List** | 2,422,715 | 1,667 ns | Good (9.2 MB) | Current Default |
| **HashMap** | 244,288 | 41,833 ns | Poor (20 MB) | Not Recommended |
| **ART** | 59,757 | 70,041 ns | Moderate (13.5 MB) | Not Recommended |

---

## Test Configuration

```yaml
Order Count: 50,000
Warmup Orders: 5,000
Price Levels: 100
Read/Write Ratio: 30% reads / 70% writes
Latency Sampling: Every 10th operation
```

---

## 1. Single-Threaded Performance

### 1.1 Throughput Comparison

| Implementation | Throughput (ops/s) | Relative |
|----------------|-------------------|----------|
| B+ Tree | 4,457,453 | 1.84x |
| Skip List | 2,422,715 | 1.00x (baseline) |
| HashMap | 244,288 | 0.10x |
| ART | 59,757 | 0.02x |

**Analysis**: B+ Tree achieves the highest throughput at 4.4M ops/sec, nearly 2x faster than Skip List. HashMap and ART show significantly lower performance due to heap maintenance overhead and prefix tree traversal costs respectively.

### 1.2 Latency Distribution

| Implementation | Avg (ns) | P50 (ns) | P95 (ns) | P99 (ns) | Max (ns) |
|----------------|----------|----------|----------|----------|----------|
| B+ Tree | 191 | 208 | 333 | 542 | 6,625 |
| Skip List | 377 | 417 | 625 | 1,667 | 13,083 |
| HashMap | 4,186 | 333 | 35,042 | 41,833 | 236,125 |
| ART | 16,770 | 792 | 64,209 | 70,041 | 207,667 |

**Analysis**: B+ Tree shows the most consistent latency profile with tight P99 at 542ns. HashMap and ART exhibit high tail latencies due to heap reorganization and tree rebalancing respectively.

### 1.3 Memory Efficiency

| Implementation | Memory Alloc | Total Alloc | GC Pauses |
|----------------|--------------|-------------|-----------|
| B+ Tree | 6.1 MB | 6.1 MB | 0 |
| Skip List | 9.2 MB | 9.2 MB | 0 |
| ART | 13.5 MB | 93.1 MB | 4 |
| HashMap | 20.0 MB | 188.6 MB | 8 |

**Analysis**: B+ Tree is the most memory-efficient with minimal allocations. HashMap generates significant garbage due to map resizing, causing multiple GC pauses.

---

## 2. Concurrent Performance (14 goroutines)

### 2.1 Concurrent Throughput

| Implementation | Throughput (ops/s) | Scaling Factor |
|----------------|-------------------|----------------|
| B+ Tree | 3,037,398 | 0.68x |
| Skip List | 1,497,790 | 0.62x |
| HashMap | 222,640 | 0.91x |
| ART | 61,821 | 1.03x |

**Analysis**: B+ Tree and Skip List show ~60-70% scaling efficiency under concurrent load due to lock contention. HashMap maintains good scaling but low absolute throughput. ART scales linearly but remains slowest.

### 2.2 Concurrent P99 Latency

| Implementation | P99 Latency (ns) | vs Single-Threaded |
|----------------|-----------------|---------------------|
| B+ Tree | 151,000 | 278x |
| Skip List | 363,250 | 218x |
| HashMap | 795,334 | 19x |
| ART | 1,147,375 | 16x |

**Analysis**: Concurrent access significantly increases tail latencies across all implementations due to lock contention. B+ Tree remains the fastest in absolute terms.

---

## 3. High Read Ratio (80% reads)

### 3.1 Read-Heavy Throughput

| Implementation | Throughput (ops/s) | vs Balanced |
|----------------|-------------------| ------------|
| B+ Tree | 6,474,296 | +45% |
| Skip List | 5,455,082 | +125% |
| HashMap | 96,721 | -60% |
| ART | 23,031 | -61% |

**Analysis**: Skip List and B+ Tree excel in read-heavy workloads. HashMap and ART performance degrades due to heap traversal overhead during GetBest/GetTop operations.

### 3.2 Read Operation Latency

| Implementation | P95 (ns) | P99 (ns) |
|----------------|----------|----------|
| B+ Tree | 292 | 375 |
| Skip List | 583 | 917 |
| HashMap | 37,500 | 43,875 |
| ART | 68,166 | 74,625 |

---

## 4. Benchmark Results (Go Benchmark)

### 4.1 AddOrder Performance

```
BenchmarkAddOrder_SkipList-14    1,600,159    904.8 ns/op    270 B/op    8 allocs/op
BenchmarkAddOrder_HashMap-14     2,618,499    442.9 ns/op    265 B/op    8 allocs/op
BenchmarkAddOrder_BTree-14       2,120,984    616.1 ns/op    157 B/op    5 allocs/op
BenchmarkAddOrder_ART-14         1,327,800    909.9 ns/op    251 B/op    7 allocs/op
```

### 4.2 RemoveOrder Performance

```
BenchmarkRemoveOrder_SkipList-14    1,451,354    848.2 ns/op    239 B/op    8 allocs/op
BenchmarkRemoveOrder_HashMap-14     2,559,091    494.9 ns/op    245 B/op    8 allocs/op
BenchmarkRemoveOrder_BTree-14       2,254,375    611.8 ns/op    134 B/op    5 allocs/op
BenchmarkRemoveOrder_ART-14         1,338,801    888.7 ns/op    199 B/op    7 allocs/op
```

### 4.3 GetBestLevels Performance

```
BenchmarkGetBest_SkipList-14    312,793,345    3.828 ns/op    0 B/op    0 allocs/op
BenchmarkGetBest_HashMap-14       4,780,621    253.2 ns/op  240 B/op    8 allocs/op
BenchmarkGetBest_BTree-14       217,716,152    5.489 ns/op    0 B/op    0 allocs/op
BenchmarkGetBest_ART-14               3,600  323,303 ns/op   32 KB/op   8 allocs/op
```

### 4.4 GetTop10 Performance

```
BenchmarkGetTop10_SkipList-14    16,676,847    72.17 ns/op    160 B/op    2 allocs/op
BenchmarkGetTop10_HashMap-14          5,091   226,776 ns/op  248 KB/op 6645 allocs/op
BenchmarkGetTop10_BTree-14        5,336,055    224.7 ns/op    320 B/op   10 allocs/op
BenchmarkGetTop10_ART-14              3,715   321,287 ns/op   33 KB/op   10 allocs/op
```

### 4.5 MixedOperations Performance

```
BenchmarkMixedOps_SkipList-14    7,819    158,047 ns/op    117 KB/op    3,309 allocs/op
BenchmarkMixedOps_HashMap-14     4,918    243,396 ns/op    234 KB/op    7,330 allocs/op
BenchmarkMixedOps_BTree-14      21,106     56,693 ns/op     68 KB/op    2,226 allocs/op
BenchmarkMixedOps_ART-14         2,353    501,681 ns/op    307 KB/op    5,285 allocs/op
```

---

## 5. Production Recommendations

### 5.1 Recommended Implementation: B+ Tree

**Rationale**:
1. **Highest Throughput**: 4.4M ops/sec (1.8x faster than Skip List)
2. **Best Latency Profile**: P99 < 1μs, Max < 10μs
3. **Memory Efficient**: Lowest memory footprint, zero GC pauses
4. **Allocation Efficient**: Fewest allocations per operation (5 allocs/op)
5. **Range Query Optimized**: GetTop10 at 224ns vs 72ns (3x slower than Skip List, but still sub-microsecond)

### 5.2 Implementation Trade-offs

| Use Case | Recommended | Reason |
|----------|-------------|--------|
| High Throughput Trading | B+ Tree | Best overall performance |
| Memory Constrained | B+ Tree | Lowest memory footprint |
| Read-Heavy Workload | Skip List | Fastest GetBest (3.8ns) |
| Simple Implementation | HashMap | Conceptually simplest |

### 5.3 Migration Path

1. **Phase 1**: Deploy B+ Tree as alternative engine (current PR)
2. **Phase 2**: A/B test in testnet environment
3. **Phase 3**: Gradual production rollout with monitoring
4. **Phase 4**: Full migration after stability confirmation

---

## 6. Test Files

| File | Description |
|------|-------------|
| `x/orderbook/keeper/e2e_stress_test.go` | E2E stress test suite |
| `x/orderbook/keeper/benchmark_test.go` | Unit benchmarks |
| `x/orderbook/keeper/benchmark_comparison_test.go` | Implementation comparison |
| `x/orderbook/keeper/orderbook_interface.go` | Unified interface |
| `x/orderbook/keeper/orderbook_btree.go` | B+ Tree implementation |
| `x/orderbook/keeper/orderbook_hashmap.go` | HashMap implementation |
| `x/orderbook/keeper/orderbook_art.go` | ART implementation |
| `x/orderbook/keeper/orderbook_v2.go` | Skip List implementation |

---

## 7. Running Tests

```bash
# Run all E2E stress tests
go test -v -run "TestE2E" ./x/orderbook/keeper/ -timeout 600s

# Run specific stress test
go test -v -run "TestE2EStressAllImplementations" ./x/orderbook/keeper/

# Run benchmarks
go test -bench="." -benchmem ./x/orderbook/keeper/

# Run comparison benchmarks only
go test -bench="BenchmarkAddOrder|BenchmarkGetBest" -benchmem ./x/orderbook/keeper/
```

---

## Appendix: Raw JSON Results

```json
{
  "generated_at": "2026-01-19T12:00:00Z",
  "cpus": 14,
  "goos": "darwin",
  "goarch": "arm64",
  "results": [
    {
      "implementation": "SkipList",
      "throughput_ops_per_sec": 2422715.38,
      "avg_latency_ns": 376.87,
      "p50_latency_ns": 417,
      "p95_latency_ns": 625,
      "p99_latency_ns": 1667,
      "memory_alloc_bytes": 9680000
    },
    {
      "implementation": "HashMap",
      "throughput_ops_per_sec": 244288.83,
      "avg_latency_ns": 4185.89,
      "p50_latency_ns": 333,
      "p95_latency_ns": 35042,
      "p99_latency_ns": 41833,
      "memory_alloc_bytes": 20937200
    },
    {
      "implementation": "BTree",
      "throughput_ops_per_sec": 4457453.47,
      "avg_latency_ns": 191.12,
      "p50_latency_ns": 208,
      "p95_latency_ns": 333,
      "p99_latency_ns": 542,
      "memory_alloc_bytes": 6430200
    },
    {
      "implementation": "ART",
      "throughput_ops_per_sec": 59757.66,
      "avg_latency_ns": 16770.50,
      "p50_latency_ns": 792,
      "p95_latency_ns": 64209,
      "p99_latency_ns": 70041,
      "memory_alloc_bytes": 14204560
    }
  ]
}
```

---

**Report Generated by PerpDEX E2E Stress Test Suite**
