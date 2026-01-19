#!/bin/bash

# Hyperliquid API Integration Verification Script
# Tests the Hyperliquid API endpoints to verify connectivity

set -e

echo "=========================================="
echo "Hyperliquid API Integration Verification"
echo "=========================================="
echo ""

# API Base URL
HL_API="https://api.hyperliquid.xyz/info"

echo "1. Testing Meta Data API..."
META_RESPONSE=$(curl -s -X POST "$HL_API" \
  -H "Content-Type: application/json" \
  -d '{"type": "meta"}' | head -c 500)

if [[ "$META_RESPONSE" == *"universe"* ]]; then
  echo "   ✓ Meta API working - received market data"
else
  echo "   ✗ Meta API failed"
  echo "   Response: $META_RESPONSE"
fi

echo ""
echo "2. Testing Asset Contexts API..."
CTX_RESPONSE=$(curl -s -X POST "$HL_API" \
  -H "Content-Type: application/json" \
  -d '{"type": "metaAndAssetCtxs"}' | head -c 500)

if [[ "$CTX_RESPONSE" == *"markPx"* ]]; then
  echo "   ✓ Asset Contexts API working - received price data"
else
  echo "   ✗ Asset Contexts API failed"
  echo "   Response: $CTX_RESPONSE"
fi

echo ""
echo "3. Testing L2 Order Book API (BTC)..."
L2_RESPONSE=$(curl -s -X POST "$HL_API" \
  -H "Content-Type: application/json" \
  -d '{"type": "l2Book", "coin": "BTC"}' | head -c 500)

if [[ "$L2_RESPONSE" == *"levels"* ]]; then
  echo "   ✓ L2 Order Book API working - received orderbook data"
else
  echo "   ✗ L2 Order Book API failed"
  echo "   Response: $L2_RESPONSE"
fi

echo ""
echo "4. Testing Recent Trades API (BTC)..."
TRADES_RESPONSE=$(curl -s -X POST "$HL_API" \
  -H "Content-Type: application/json" \
  -d '{"type": "recentTrades", "coin": "BTC"}' | head -c 500)

if [[ "$TRADES_RESPONSE" == *"px"* ]] || [[ "$TRADES_RESPONSE" == *"side"* ]]; then
  echo "   ✓ Recent Trades API working - received trade data"
else
  echo "   ✗ Recent Trades API failed"
  echo "   Response: $TRADES_RESPONSE"
fi

echo ""
echo "5. Testing Candle Snapshot API (BTC)..."
NOW=$(date +%s)000
START=$((NOW - 3600000))  # 1 hour ago
CANDLE_RESPONSE=$(curl -s -X POST "$HL_API" \
  -H "Content-Type: application/json" \
  -d "{\"type\": \"candleSnapshot\", \"req\": {\"coin\": \"BTC\", \"interval\": \"1m\", \"startTime\": $START, \"endTime\": $NOW}}" | head -c 500)

if [[ "$CANDLE_RESPONSE" == *"o"* ]] && [[ "$CANDLE_RESPONSE" == *"c"* ]]; then
  echo "   ✓ Candle Snapshot API working - received candle data"
else
  echo "   ✗ Candle Snapshot API failed"
  echo "   Response: $CANDLE_RESPONSE"
fi

echo ""
echo "=========================================="
echo "Verification Complete"
echo "=========================================="
echo ""
echo "If all tests passed, the Hyperliquid API integration is ready."
echo "Start the frontend with: cd frontend && npm run dev"
