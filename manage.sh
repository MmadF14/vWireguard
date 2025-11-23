#!/bin/bash

# vWireguard Management CLI Tool (vwg)
# This script provides command-line management for vWireguard

set -e

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m'

SERVICE_NAME="vwireguard"
INSTALL_DIR="/usr/local/vwireguard"
REPO_OWNER="MmadF14"
REPO_NAME="vwireguard"

log() { echo -e "${GREEN}[$(date '+%H:%M:%S')] $1${NC}"; }
error() { echo -e "${RED}[$(date '+%H:%M:%S')] ❌ $1${NC}"; exit 1; }
warn() { echo -e "${YELLOW}[$(date '+%H:%M:%S')] ⚠️  $1${NC}"; }

# Check if running as root (for service commands)
check_root() {
    if [ "$EUID" -ne 0 ] && [[ "$1" =~ ^(start|stop|restart|update)$ ]]; then
        error "This command requires root access. Please run with: sudo vwg $1"
    fi
}

# Start service
cmd_start() {
    check_root "start"
    log "Starting ${SERVICE_NAME} service..."
    systemctl start "$SERVICE_NAME"
    sleep 1
    if systemctl is-active --quiet "$SERVICE_NAME"; then
        log "Service started successfully"
    else
        error "Failed to start service"
    fi
}

# Stop service
cmd_stop() {
    check_root "stop"
    log "Stopping ${SERVICE_NAME} service..."
    systemctl stop "$SERVICE_NAME"
    log "Service stopped"
}

# Restart service
cmd_restart() {
    check_root "restart"
    log "Restarting ${SERVICE_NAME} service..."
    systemctl restart "$SERVICE_NAME"
    sleep 1
    if systemctl is-active --quiet "$SERVICE_NAME"; then
        log "Service restarted successfully"
    else
        error "Failed to restart service"
    fi
}

# Check service status
cmd_status() {
    if systemctl is-active --quiet "$SERVICE_NAME" 2>/dev/null; then
        echo -e "${GREEN}Status: Running${NC}"
        systemctl status "$SERVICE_NAME" --no-pager -l
    elif systemctl is-enabled --quiet "$SERVICE_NAME" 2>/dev/null; then
        echo -e "${YELLOW}Status: Stopped${NC}"
        systemctl status "$SERVICE_NAME" --no-pager -l
    else
        echo -e "${RED}Status: Service not found${NC}"
        echo "vWireguard may not be installed. Run: bash <(curl -Ls https://raw.githubusercontent.com/${REPO_OWNER}/${REPO_NAME}/master/install.sh)"
    fi
}

# View logs
cmd_log() {
    local lines=${1:-50}
    if [ -n "$1" ] && [[ ! "$1" =~ ^[0-9]+$ ]]; then
        error "Invalid number of lines: $1"
    fi
    
    if systemctl list-unit-files | grep -q "^${SERVICE_NAME}.service"; then
        journalctl -u "$SERVICE_NAME" -n "$lines" --no-pager -f
    else
        error "Service not found. vWireguard may not be installed."
    fi
}

# Update to latest release
cmd_update() {
    check_root "update"
    
    if [ ! -d "$INSTALL_DIR" ] || [ ! -f "$INSTALL_DIR/vWireguard" ]; then
        error "vWireguard is not installed. Please run the installation script first."
    fi
    
    log "Starting vWireguard update..."
    
    # Detect architecture
    local arch=$(uname -m)
    case $arch in
        x86_64) arch="amd64" ;;
        aarch64|arm64) arch="arm64" ;;
        *) error "Unsupported architecture: $arch" ;;
    esac
    
    # Get latest release
    local api_url="https://api.github.com/repos/${REPO_OWNER}/${REPO_NAME}/releases/latest"
    local tag=$(curl -sL "$api_url" | grep -oP '"tag_name": "\K[^"]*' | head -n1)
    
    if [ -z "$tag" ]; then
        error "Failed to fetch latest release tag"
    fi
    
    log "Latest release: $tag"
    
    # Get current version (if available)
    local current_version=$("$INSTALL_DIR/vWireguard" -version 2>/dev/null || echo "unknown")
    log "Current version: $current_version"
    
    if [ "$current_version" = "$tag" ]; then
        warn "Already running the latest version: $tag"
        read -p "Force update anyway? (y/N): " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            log "Update cancelled"
            exit 0
        fi
    fi
    
    # Create backup
    local backup_dir="/tmp/vwireguard-backup-$(date +%Y%m%d_%H%M%S)"
    mkdir -p "$backup_dir"
    cp -r "$INSTALL_DIR/db" "$backup_dir/" 2>/dev/null || true
    cp "$INSTALL_DIR/vWireguard" "$backup_dir/vWireguard.old" 2>/dev/null || true
    log "Backup created at: $backup_dir"
    
    # Stop service
    if systemctl is-active --quiet "$SERVICE_NAME"; then
        log "Stopping service..."
        systemctl stop "$SERVICE_NAME"
    fi
    
    # Download new release
    local asset_name="vWireguard-linux-${arch}.tar.gz"
    local download_url="https://github.com/${REPO_OWNER}/${REPO_NAME}/releases/download/${tag}/${asset_name}"
    local temp_file="/tmp/${asset_name}"
    
    log "Downloading ${asset_name}..."
    if ! wget -q --show-progress -O "$temp_file" "$download_url"; then
        error "Failed to download release"
    fi
    
    # Extract to temporary directory
    local temp_extract="/tmp/vwireguard-extract-$$"
    mkdir -p "$temp_extract"
    tar -xzf "$temp_file" -C "$temp_extract"
    
    # Backup old binary
    mv "$INSTALL_DIR/vWireguard" "$INSTALL_DIR/vWireguard.old" 2>/dev/null || true
    
    # Install new files (preserve db)
    log "Installing new version..."
    cp "$temp_extract/vWireguard" "$INSTALL_DIR/vWireguard"
    chmod +x "$INSTALL_DIR/vWireguard"
    
    # Update static and templates if they exist
    if [ -d "$temp_extract/static" ]; then
        cp -r "$temp_extract/static" "$INSTALL_DIR/"
    fi
    if [ -d "$temp_extract/templates" ]; then
        cp -r "$temp_extract/templates" "$INSTALL_DIR/"
    fi
    if [ -d "$temp_extract/custom" ]; then
        cp -r "$temp_extract/custom" "$INSTALL_DIR/"
    fi
    
    # Update vwg script
    if [ -f "$temp_extract/vwg" ]; then
        cp "$temp_extract/vwg" "$INSTALL_DIR/vwg"
        chmod +x "$INSTALL_DIR/vwg"
    fi
    
    # Cleanup
    rm -rf "$temp_extract" "$temp_file"
    
    # Start service
    log "Starting service..."
    systemctl start "$SERVICE_NAME"
    
    sleep 2
    
    if systemctl is-active --quiet "$SERVICE_NAME"; then
        log "✅ Update completed successfully!"
        log "New version: $tag"
        log "Backup stored at: $backup_dir"
    else
        error "Failed to start service after update. Restoring from backup..."
        cp "$backup_dir/vWireguard.old" "$INSTALL_DIR/vWireguard"
        systemctl start "$SERVICE_NAME"
        if systemctl is-active --quiet "$SERVICE_NAME"; then
            warn "Service restored from backup. Update failed."
        else
            error "Failed to restore service. Manual intervention required."
        fi
    fi
}

# Show help
show_help() {
    cat << EOF
${BLUE}vWireguard Management CLI (vwg)${NC}

Usage: vwg <command> [options]

Commands:
  start          Start the vWireguard service (requires root)
  stop           Stop the vWireguard service (requires root)
  restart        Restart the vWireguard service (requires root)
  status         Show service status
  log [lines]    View service logs (default: 50 lines, use -f for follow)
  update         Update to the latest release (requires root)
  help           Show this help message

Examples:
  sudo vwg start
  vwg status
  vwg log 100
  vwg log -f      # Follow logs
  sudo vwg update

For installation:
  bash <(curl -Ls https://raw.githubusercontent.com/${REPO_OWNER}/${REPO_NAME}/master/install.sh)
EOF
}

# Main command dispatcher
main() {
    local command=${1:-help}
    
    case "$command" in
        start)
            cmd_start
            ;;
        stop)
            cmd_stop
            ;;
        restart)
            cmd_restart
            ;;
        status)
            cmd_status
            ;;
        log|logs)
            # Handle -f flag for follow
            if [ "$2" = "-f" ]; then
                journalctl -u "$SERVICE_NAME" -f
            else
                cmd_log "$2"
            fi
            ;;
        update|upgrade)
            cmd_update
            ;;
        help|--help|-h)
            show_help
            ;;
        *)
            error "Unknown command: $command\nRun 'vwg help' for usage information"
            ;;
    esac
}

# Run main function
main "$@"
