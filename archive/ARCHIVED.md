# Archived Files

This directory contains archived test files that are redundant or superseded by more comprehensive tests.

## Archive Date: 2026-01-26

## Archived Tests

### `/archive/tests/e2e/`

| File | Size | Reason | Replaced By |
|------|------|--------|-------------|
| `orderbook_e2e_test.go` | 23.8 KB | Uses in-memory mock, doesn't test real network | `/tests/e2e_real/trading_flow_test.go` |
| `real_engine_e2e_test.go` | 7.2 KB | Functionality covered by chain and HTTP tests | `/tests/e2e_chain/` + `/tests/e2e_real/` |
| `tpsl_realtime_test.go` | 7.2 KB | Low coverage (only 3 markets), realtime price only | `/tests/e2e/tpsl_e2e_test.go` |

### `/archive/tests/benchmark/`

| File | Reason |
|------|--------|
| `engine_benchmark_test.go` | Scattered benchmark, should be consolidated |

### `/archive/tests/tps_benchmark/`

| File | Reason |
|------|--------|
| `tps_test.go` | Separate directory for single test file |

### `/archive/tools/`

| Directory | Reason |
|-----------|--------|
| `loadtest/` | Tool, not a test - should be in `/tools/` if needed |

## Active Test Directories (Kept)

| Directory | Tests | Purpose |
|-----------|-------|---------|
| `/tests/e2e/` | 1 file (tpsl_e2e_test.go) | Core TPSL E2E tests |
| `/tests/e2e_real/` | 9 files, 70+ tests | HTTP API + WebSocket comprehensive tests |
| `/tests/e2e_chain/` | 4 files, 11+ tests | On-chain transaction tests |
| `/tests/e2e_hyperliquid/` | 3 files, 27+ tests | External Hyperliquid API integration |
| `/x/*/keeper/*_test.go` | 10+ files | Module keeper unit tests |
| `/api/handlers/*_test.go` | 1 file | API handler tests |
| `/frontend/tests/` | 4 files | Frontend unit + E2E tests |

## Recovery

To restore any archived file:
```bash
mv archive/tests/e2e/orderbook_e2e_test.go tests/e2e/
```
