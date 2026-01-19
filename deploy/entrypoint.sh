#!/bin/bash
set -e

# PerpDEX Node Entrypoint Script
# Handles initialization and startup

# Configuration from environment
MONIKER="${PERPDEX_MONIKER:-perpdex-node}"
CHAIN_ID="${PERPDEX_CHAIN_ID:-perpdex-mainnet-1}"
HOME_DIR="${PERPDEX_HOME:-/root/.perpdex}"
P2P_PORT="${PERPDEX_P2P_PORT:-26656}"
RPC_PORT="${PERPDEX_RPC_PORT:-26657}"
GRPC_PORT="${PERPDEX_GRPC_PORT:-9090}"
API_PORT="${PERPDEX_API_PORT:-1317}"
MODE="${PERPDEX_MODE:-validator}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Initialize node if not already initialized
initialize_node() {
    if [ ! -f "$HOME_DIR/config/genesis.json" ]; then
        log_info "Initializing node: $MONIKER"
        perpdexd init "$MONIKER" --chain-id "$CHAIN_ID" --home "$HOME_DIR"

        # Copy genesis if provided
        if [ -f "/config/genesis.json" ]; then
            log_info "Copying genesis file"
            cp /config/genesis.json "$HOME_DIR/config/genesis.json"
        fi

        # Copy private validator key if provided
        if [ -f "/config/priv_validator_key.json" ]; then
            log_info "Copying validator key"
            cp /config/priv_validator_key.json "$HOME_DIR/config/priv_validator_key.json"
        fi

        # Copy node key if provided
        if [ -f "/config/node_key.json" ]; then
            log_info "Copying node key"
            cp /config/node_key.json "$HOME_DIR/config/node_key.json"
        fi
    else
        log_info "Node already initialized"
    fi
}

# Configure CometBFT
configure_cometbft() {
    CONFIG_FILE="$HOME_DIR/config/config.toml"

    log_info "Configuring CometBFT"

    # P2P settings
    sed -i "s/laddr = \"tcp:\/\/0.0.0.0:26656\"/laddr = \"tcp:\/\/0.0.0.0:$P2P_PORT\"/" "$CONFIG_FILE"
    sed -i "s/laddr = \"tcp:\/\/127.0.0.1:26657\"/laddr = \"tcp:\/\/0.0.0.0:$RPC_PORT\"/" "$CONFIG_FILE"

    # Enable Prometheus metrics
    sed -i 's/prometheus = false/prometheus = true/' "$CONFIG_FILE"
    sed -i 's/prometheus_listen_addr = ":26660"/prometheus_listen_addr = "0.0.0.0:26660"/' "$CONFIG_FILE"

    # Performance tuning
    sed -i 's/timeout_propose = "3s"/timeout_propose = "1s"/' "$CONFIG_FILE"
    sed -i 's/timeout_prevote = "1s"/timeout_prevote = "500ms"/' "$CONFIG_FILE"
    sed -i 's/timeout_precommit = "1s"/timeout_precommit = "500ms"/' "$CONFIG_FILE"
    sed -i 's/timeout_commit = "5s"/timeout_commit = "1s"/' "$CONFIG_FILE"

    # Increase mempool size
    sed -i 's/size = 5000/size = 10000/' "$CONFIG_FILE"
    sed -i 's/max_txs_bytes = 1073741824/max_txs_bytes = 2147483648/' "$CONFIG_FILE"

    # Configure seeds and persistent peers
    if [ -n "$PERPDEX_SEEDS" ]; then
        log_info "Configuring seeds: $PERPDEX_SEEDS"
        sed -i "s/seeds = \"\"/seeds = \"$PERPDEX_SEEDS\"/" "$CONFIG_FILE"
    fi

    if [ -n "$PERPDEX_PERSISTENT_PEERS" ]; then
        log_info "Configuring persistent peers: $PERPDEX_PERSISTENT_PEERS"
        sed -i "s/persistent_peers = \"\"/persistent_peers = \"$PERPDEX_PERSISTENT_PEERS\"/" "$CONFIG_FILE"
    fi

    # Sentry node configuration
    if [ "$MODE" = "sentry" ]; then
        log_info "Configuring as sentry node"
        sed -i 's/pex = true/pex = false/' "$CONFIG_FILE"
        sed -i 's/addr_book_strict = true/addr_book_strict = false/' "$CONFIG_FILE"
    fi
}

# Configure app.toml
configure_app() {
    APP_FILE="$HOME_DIR/config/app.toml"

    log_info "Configuring application"

    # Enable API
    sed -i 's/enable = false/enable = true/' "$APP_FILE"
    sed -i "s/address = \"tcp:\/\/localhost:1317\"/address = \"tcp:\/\/0.0.0.0:$API_PORT\"/" "$APP_FILE"

    # Enable gRPC
    sed -i "s/address = \"0.0.0.0:9090\"/address = \"0.0.0.0:$GRPC_PORT\"/" "$APP_FILE"

    # Enable Swagger
    sed -i 's/swagger = false/swagger = true/' "$APP_FILE"

    # Pruning (keep last 100 states, prune every 10 blocks)
    sed -i 's/pruning = "default"/pruning = "custom"/' "$APP_FILE"
    sed -i 's/pruning-keep-recent = "0"/pruning-keep-recent = "100"/' "$APP_FILE"
    sed -i 's/pruning-interval = "0"/pruning-interval = "10"/' "$APP_FILE"

    # State sync (for fast sync)
    if [ "$PERPDEX_STATE_SYNC" = "true" ]; then
        log_info "Enabling state sync"
        sed -i 's/enable = false/enable = true/' "$HOME_DIR/config/config.toml"
    fi
}

# Wait for other nodes (for cluster startup)
wait_for_peers() {
    if [ -n "$PERPDEX_WAIT_FOR" ]; then
        log_info "Waiting for peers: $PERPDEX_WAIT_FOR"
        IFS=',' read -ra PEERS <<< "$PERPDEX_WAIT_FOR"
        for peer in "${PEERS[@]}"; do
            host=$(echo "$peer" | cut -d':' -f1)
            port=$(echo "$peer" | cut -d':' -f2)

            log_info "Waiting for $host:$port..."
            while ! nc -z "$host" "$port" 2>/dev/null; do
                sleep 1
            done
            log_info "$host:$port is available"
        done
    fi
}

# Start the node
start_node() {
    log_info "Starting PerpDEX node: $MONIKER"
    log_info "Chain ID: $CHAIN_ID"
    log_info "Mode: $MODE"

    exec perpdexd start \
        --home "$HOME_DIR" \
        --log_level "${PERPDEX_LOG_LEVEL:-info}" \
        --log_format "${PERPDEX_LOG_FORMAT:-json}" \
        "$@"
}

# Main entrypoint
main() {
    case "$1" in
        init)
            initialize_node
            configure_cometbft
            configure_app
            ;;
        start)
            initialize_node
            configure_cometbft
            configure_app
            wait_for_peers
            shift
            start_node "$@"
            ;;
        tendermint)
            shift
            exec perpdexd tendermint "$@"
            ;;
        keys)
            shift
            exec perpdexd keys "$@"
            ;;
        query)
            shift
            exec perpdexd query "$@"
            ;;
        tx)
            shift
            exec perpdexd tx "$@"
            ;;
        *)
            exec "$@"
            ;;
    esac
}

main "$@"
