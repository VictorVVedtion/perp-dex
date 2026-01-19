#!/bin/bash
#
# PerpDEX Fast Consensus Configuration Applier
#
# This script applies optimized CometBFT configuration for high-performance trading.
# It backs up the original configuration before applying changes.
#
# Usage:
#   ./apply_fast_config.sh [--home <perpdex-home>] [--backup] [--restore] [--dry-run]
#
# Options:
#   --home <path>   Path to PerpDEX home directory (default: ~/.perpdex)
#   --backup        Only create backup, don't apply changes
#   --restore       Restore from most recent backup
#   --dry-run       Show what would be changed without making changes
#   --help          Show this help message

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Default values
PERPDEX_HOME="${HOME}/.perpdex"
CONFIG_DIR=""
BACKUP_ONLY=false
RESTORE_MODE=false
DRY_RUN=false
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

# Print functions
print_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Show usage
show_help() {
    head -n 18 "$0" | tail -n 15 | sed 's/^# //'
    exit 0
}

# Parse command line arguments
parse_args() {
    while [[ $# -gt 0 ]]; do
        case $1 in
            --home)
                PERPDEX_HOME="$2"
                shift 2
                ;;
            --backup)
                BACKUP_ONLY=true
                shift
                ;;
            --restore)
                RESTORE_MODE=true
                shift
                ;;
            --dry-run)
                DRY_RUN=true
                shift
                ;;
            --help|-h)
                show_help
                ;;
            *)
                print_error "Unknown option: $1"
                show_help
                ;;
        esac
    done

    CONFIG_DIR="${PERPDEX_HOME}/config"
}

# Check prerequisites
check_prerequisites() {
    if [[ ! -d "$PERPDEX_HOME" ]]; then
        print_error "PerpDEX home directory not found: $PERPDEX_HOME"
        print_info "Run 'perpdexd init <moniker>' first to initialize the node."
        exit 1
    fi

    if [[ ! -d "$CONFIG_DIR" ]]; then
        print_error "Config directory not found: $CONFIG_DIR"
        exit 1
    fi

    if [[ ! -f "${CONFIG_DIR}/config.toml" ]]; then
        print_error "config.toml not found in $CONFIG_DIR"
        exit 1
    fi
}

# Create backup of current configuration
create_backup() {
    local timestamp=$(date +"%Y%m%d_%H%M%S")
    local backup_dir="${CONFIG_DIR}/backups"
    local backup_name="config_backup_${timestamp}"

    if [[ "$DRY_RUN" == true ]]; then
        print_info "[DRY-RUN] Would create backup at: ${backup_dir}/${backup_name}"
        return 0
    fi

    mkdir -p "$backup_dir"

    print_info "Creating backup of current configuration..."

    # Backup config.toml
    if [[ -f "${CONFIG_DIR}/config.toml" ]]; then
        cp "${CONFIG_DIR}/config.toml" "${backup_dir}/${backup_name}_config.toml"
    fi

    # Backup app.toml if exists
    if [[ -f "${CONFIG_DIR}/app.toml" ]]; then
        cp "${CONFIG_DIR}/app.toml" "${backup_dir}/${backup_name}_app.toml"
    fi

    # Create a manifest file
    cat > "${backup_dir}/${backup_name}_manifest.txt" <<EOF
PerpDEX Configuration Backup
Created: $(date)
Source: ${CONFIG_DIR}
Files:
  - ${backup_name}_config.toml
  - ${backup_name}_app.toml (if existed)
EOF

    print_success "Backup created at: ${backup_dir}/${backup_name}_*"
    echo "$backup_name" > "${backup_dir}/latest"
}

# Restore from backup
restore_backup() {
    local backup_dir="${CONFIG_DIR}/backups"

    if [[ ! -f "${backup_dir}/latest" ]]; then
        print_error "No backup found. Run with --backup first."
        exit 1
    fi

    local latest_backup=$(cat "${backup_dir}/latest")

    if [[ "$DRY_RUN" == true ]]; then
        print_info "[DRY-RUN] Would restore from: ${latest_backup}"
        return 0
    fi

    print_info "Restoring from backup: $latest_backup"

    if [[ -f "${backup_dir}/${latest_backup}_config.toml" ]]; then
        cp "${backup_dir}/${latest_backup}_config.toml" "${CONFIG_DIR}/config.toml"
        print_success "Restored config.toml"
    fi

    if [[ -f "${backup_dir}/${latest_backup}_app.toml" ]]; then
        cp "${backup_dir}/${latest_backup}_app.toml" "${CONFIG_DIR}/app.toml"
        print_success "Restored app.toml"
    fi

    print_success "Configuration restored successfully!"
}

# Apply fast consensus configuration
apply_fast_config() {
    local config_file="${CONFIG_DIR}/config.toml"
    local fast_config="${PROJECT_ROOT}/config/fast_consensus.toml"

    if [[ ! -f "$fast_config" ]]; then
        print_warning "fast_consensus.toml not found at: $fast_config"
        print_info "Applying configuration via sed..."
    fi

    if [[ "$DRY_RUN" == true ]]; then
        print_info "[DRY-RUN] Would apply the following changes to $config_file:"
        echo ""
        echo "  Consensus Configuration:"
        echo "    timeout_propose = \"500ms\""
        echo "    timeout_propose_delta = \"100ms\""
        echo "    timeout_prevote = \"500ms\""
        echo "    timeout_prevote_delta = \"100ms\""
        echo "    timeout_precommit = \"500ms\""
        echo "    timeout_precommit_delta = \"100ms\""
        echo "    timeout_commit = \"500ms\""
        echo ""
        echo "  Mempool Configuration:"
        echo "    size = 10000"
        echo "    max_tx_bytes = 10485760"
        echo "    max_txs_bytes = 104857600"
        echo ""
        echo "  P2P Configuration:"
        echo "    flush_throttle_timeout = \"10ms\""
        echo "    send_rate = 20480000"
        echo "    recv_rate = 20480000"
        return 0
    fi

    print_info "Applying fast consensus configuration..."

    # Consensus timeouts
    sed -i.bak 's/timeout_propose = "[^"]*"/timeout_propose = "500ms"/g' "$config_file"
    sed -i.bak 's/timeout_propose_delta = "[^"]*"/timeout_propose_delta = "100ms"/g' "$config_file"
    sed -i.bak 's/timeout_prevote = "[^"]*"/timeout_prevote = "500ms"/g' "$config_file"
    sed -i.bak 's/timeout_prevote_delta = "[^"]*"/timeout_prevote_delta = "100ms"/g' "$config_file"
    sed -i.bak 's/timeout_precommit = "[^"]*"/timeout_precommit = "500ms"/g' "$config_file"
    sed -i.bak 's/timeout_precommit_delta = "[^"]*"/timeout_precommit_delta = "100ms"/g' "$config_file"
    sed -i.bak 's/timeout_commit = "[^"]*"/timeout_commit = "500ms"/g' "$config_file"

    # Mempool configuration
    sed -i.bak 's/^size = [0-9]*/size = 10000/g' "$config_file"
    sed -i.bak 's/max_tx_bytes = [0-9]*/max_tx_bytes = 10485760/g' "$config_file"
    sed -i.bak 's/max_txs_bytes = [0-9]*/max_txs_bytes = 104857600/g' "$config_file"

    # P2P configuration
    sed -i.bak 's/flush_throttle_timeout = "[^"]*"/flush_throttle_timeout = "10ms"/g' "$config_file"
    sed -i.bak 's/send_rate = [0-9]*/send_rate = 20480000/g' "$config_file"
    sed -i.bak 's/recv_rate = [0-9]*/recv_rate = 20480000/g' "$config_file"

    # Clean up backup files created by sed
    rm -f "${config_file}.bak"

    print_success "Fast consensus configuration applied!"
}

# Restart the node
restart_node() {
    if [[ "$DRY_RUN" == true ]]; then
        print_info "[DRY-RUN] Would attempt to restart perpdexd"
        return 0
    fi

    print_info "Checking for running perpdexd process..."

    if pgrep -x "perpdexd" > /dev/null; then
        print_warning "perpdexd is running. Please restart it manually to apply changes:"
        echo ""
        echo "  # Using systemd:"
        echo "  sudo systemctl restart perpdexd"
        echo ""
        echo "  # Or manually:"
        echo "  pkill perpdexd"
        echo "  perpdexd start"
    else
        print_info "perpdexd is not running. Start it with:"
        echo ""
        echo "  perpdexd start"
    fi
}

# Verify configuration
verify_config() {
    local config_file="${CONFIG_DIR}/config.toml"

    print_info "Verifying configuration..."

    # Check for key values
    local timeout_commit=$(grep "timeout_commit" "$config_file" | head -1 | grep -o '"[^"]*"' | tr -d '"')
    local mempool_size=$(grep "^size" "$config_file" | head -1 | awk '{print $3}')

    echo ""
    echo "Current Configuration Summary:"
    echo "=============================="
    echo "  timeout_commit: $timeout_commit"
    echo "  mempool_size:   $mempool_size"
    echo ""

    if [[ "$timeout_commit" == "500ms" ]]; then
        print_success "Fast consensus configuration is active"
    else
        print_warning "Configuration may not be fully applied"
    fi
}

# Main function
main() {
    echo ""
    echo "=========================================="
    echo "  PerpDEX Fast Consensus Configuration"
    echo "=========================================="
    echo ""

    parse_args "$@"

    print_info "PerpDEX Home: $PERPDEX_HOME"
    print_info "Config Dir:   $CONFIG_DIR"
    echo ""

    if [[ "$RESTORE_MODE" == true ]]; then
        check_prerequisites
        restore_backup
        verify_config
        restart_node
        exit 0
    fi

    check_prerequisites
    create_backup

    if [[ "$BACKUP_ONLY" == true ]]; then
        print_success "Backup completed. Use --restore to restore later."
        exit 0
    fi

    apply_fast_config
    verify_config
    restart_node

    echo ""
    print_success "Configuration update complete!"
    echo ""
    echo "For more information, see: docs/PERFORMANCE.md"
}

# Run main function
main "$@"
