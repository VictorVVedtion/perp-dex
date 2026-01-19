#!/bin/bash

set -e

CHAIN_ID="${CHAIN_ID:-perpdex-1}"
MONIKER="${MONIKER:-validator}"
HOME_DIR="${HOME_DIR:-$HOME/.perpdex}"
KEY_NAME="${KEY_NAME:-validator}"
DENOM="${DENOM:-usdc}"

echo "========================================"
echo "Initializing PerpDEX Chain"
echo "========================================"
echo "Chain ID: $CHAIN_ID"
echo "Moniker: $MONIKER"
echo "Home Dir: $HOME_DIR"
echo "Denom: $DENOM"
echo "========================================"

# Remove existing data
rm -rf "$HOME_DIR"

# Initialize the chain
perpdexd init "$MONIKER" --chain-id "$CHAIN_ID" --home "$HOME_DIR" --default-denom "$DENOM"

# Create validator key
echo "Creating validator key..."
perpdexd keys add "$KEY_NAME" --home "$HOME_DIR" --keyring-backend test

# Get validator address
VALIDATOR_ADDR=$(perpdexd keys show "$KEY_NAME" -a --home "$HOME_DIR" --keyring-backend test)
echo "Validator address: $VALIDATOR_ADDR"

# Add genesis account with tokens
echo "Adding genesis account..."
perpdexd genesis add-genesis-account "$VALIDATOR_ADDR" 1000000000000$DENOM,100000000000stake --home "$HOME_DIR"

# Create test accounts
echo "Creating test accounts..."
for i in {1..3}; do
    KEY="trader$i"
    perpdexd keys add "$KEY" --home "$HOME_DIR" --keyring-backend test
    ADDR=$(perpdexd keys show "$KEY" -a --home "$HOME_DIR" --keyring-backend test)
    perpdexd genesis add-genesis-account "$ADDR" 10000000000$DENOM --home "$HOME_DIR"
    echo "Created $KEY with address: $ADDR"
done

# Create gentx
echo "Creating gentx..."
perpdexd genesis gentx "$KEY_NAME" 100000000stake \
    --chain-id "$CHAIN_ID" \
    --home "$HOME_DIR" \
    --keyring-backend test \
    --moniker "$MONIKER"

# Collect gentxs
echo "Collecting gentxs..."
perpdexd genesis collect-gentxs --home "$HOME_DIR"

# Validate genesis
echo "Validating genesis..."
perpdexd genesis validate --home "$HOME_DIR"

echo "========================================"
echo "Chain initialized successfully!"
echo "========================================"
echo ""
echo "To start the node, run:"
echo "  perpdexd start --home $HOME_DIR"
echo ""
echo "Or use: ./scripts/start-node.sh"
echo "========================================"
