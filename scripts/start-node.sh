#!/bin/bash

set -e

HOME_DIR="${HOME_DIR:-$HOME/.perpdex}"

echo "========================================"
echo "Starting PerpDEX Node"
echo "========================================"
echo "Home Dir: $HOME_DIR"
echo "========================================"

# Check if chain is initialized
if [ ! -f "$HOME_DIR/config/genesis.json" ]; then
    echo "Error: Chain not initialized. Run ./scripts/init-chain.sh first."
    exit 1
fi

# Start the node
perpdexd start \
    --home "$HOME_DIR" \
    --api.enable \
    --api.enabled-unsafe-cors \
    --grpc.enable \
    --grpc-web.enable \
    --log_level info
