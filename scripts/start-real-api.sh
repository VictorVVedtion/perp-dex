#!/bin/bash
# Start PerpDEX API Server with Real OrderBook Engine

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"

echo "╔══════════════════════════════════════════════════════════════╗"
echo "║           PerpDEX API Server - Real Engine Mode              ║"
echo "╚══════════════════════════════════════════════════════════════╝"
echo ""

# Parse arguments
HOST="${HOST:-0.0.0.0}"
PORT="${PORT:-8080}"

# Build if needed
if [ ! -f "$PROJECT_DIR/build/perpdexd" ]; then
    echo "Building API server..."
    cd "$PROJECT_DIR"
    go build -o build/api-server ./cmd/api
fi

echo "Configuration:"
echo "  Host:         $HOST"
echo "  Port:         $PORT"
echo "  Engine Mode:  REAL (MatchingEngineV2)"
echo ""

echo "Starting server with real orderbook engine..."
echo ""

# Start with real mode
cd "$PROJECT_DIR"
go run ./cmd/api --real --host="$HOST" --port="$PORT"
