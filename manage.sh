#!/bin/bash

# vWireguard Management CLI Tool (vwg)
# Corrected & Optimized Version

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m'

# Config
SERVICE_NAME="vwireguard"
INSTALL_DIR="/usr/local/vwireguard"
REPO_OWNER="MmadF14"
REPO_NAME="vwireguard"

log() { echo -e "${GREEN}[$(date '+%H:%M:%S')] $1${NC}"; }
error() { echo -e "${RED}[$(date '+%H:%M:%S')] ❌ $1${NC}"; exit 1; }
warn() { echo -e "${YELLOW}[$(date '+%H:%M:%S')] ⚠️  $1${NC}"; }

# Check root access for modification commands
check_root() {
    if [ "$EUID" -ne 0 ]; then
        error "This command requires root access. Please run with: sudo vwg $1"
    fi
}

# --- Service Commands ---

cmd_start() {
    check_root "start"
    log "Starting ${SERVICE_NAME}..."
    systemctl start "$SERVICE_NAME"
    sleep 1
    if systemctl is-active --quiet "$SERVICE_NAME"; then
        log "Service started successfully ✅"
    else
        error "Failed to start service. Check logs: vwg log"
    fi
}

cmd_stop() {
    check_root "stop"
    log "Stopping ${SERVICE_NAME}..."
    systemctl stop "$SERVICE_NAME"
    log "Service stopped 🛑"
}

cmd_restart() {
    check_root "restart"
    log "Restarting ${SERVICE_NAME}..."
    systemctl restart "$SERVICE_NAME"
    sleep 1
    if systemctl is-active --quiet "$SERVICE_NAME"; then
        log "Service restarted successfully ♻️"
    else
        error "Failed to restart service. Check logs: vwg log"
    fi
}

cmd_status() {
    if systemctl is-active --quiet "$SERVICE_NAME" 2>/dev/null; then
        echo -e "${GREEN}● Service is RUNNING${NC}"
        echo -e "---------------------------------------------------"
        systemctl status "$SERVICE_NAME" --no-pager -n 5
    elif systemctl is-enabled --quiet "$SERVICE_NAME" 2>/dev/null; then
        echo -e "${YELLOW}● Service is STOPPED${NC}"
    else
        echo -e "${RED}❌ Service not found or not installed.${NC}"
    fi
}

cmd_log() {
    local lines=${1:-50}
    local follow=${2:-false}
    
    if ! systemctl list-unit-files | grep -q "^${SERVICE_NAME}.service"; then
        error "Service not found."
    fi

    if [ "$follow" = "true" ]; then
        journalctl -u "$SERVICE_NAME" -n "$lines" -f --no-pager
    else
        journalctl -u "$SERVICE_NAME" -n "$lines" --no-pager
    fi
}

# --- Update Command ---

cmd_update() {
    check_root "update"
    
    if [ ! -f "$INSTALL_DIR/vWireguard" ]; then
        error "vWireguard is not installed correctly."
    fi
    
    log "Checking for updates..."
    
    # 1. Detect Architecture
    local arch=$(uname -m)
    case $arch in
        x86_64) arch="amd64" ;;
        aarch64|arm64) arch="arm64" ;;
        *) error "Unsupported architecture: $arch" ;;
    esac
    
    # 2. Get Latest Tag from GitHub
    local api_url="https://api.github.com/repos/${REPO_OWNER}/${REPO_NAME}/releases/latest"
    local tag=$(curl -sL "$api_url" | grep -oP '"tag_name": "\K[^"]*' | head -n1)
    
    if [ -z "$tag" ]; then
        error "Failed to fetch latest version info."
    fi
    
    log "Latest version available: ${CYAN}$tag${NC}"
    log "Note: Assuming update is needed (Version check skipped)."

    read -p "Do you want to proceed with the update? (y/N): " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        log "Update cancelled."
        exit 0
    fi
    
    # 3. Backup
    local backup_dir="/tmp/vwireguard-backup-$(date +%Y%m%d_%H%M%S)"
    mkdir -p "$backup_dir"
    # Only backup DB and Config
    cp -r "$INSTALL_DIR/db" "$backup_dir/" 2>/dev/null || true
    if [ -f "$INSTALL_DIR/config.json" ]; then cp "$INSTALL_DIR/config.json" "$backup_dir/"; fi
    cp "$INSTALL_DIR/vWireguard" "$backup_dir/vWireguard.old"
    
    log "Backup created at: $backup_dir"
    
    # 4. Download & Install
    local asset_name="vWireguard-linux-${arch}.tar.gz"
    local download_url="https://github.com/${REPO_OWNER}/${REPO_NAME}/releases/download/${tag}/${asset_name}"
    local temp_file="/tmp/${asset_name}"
    
    log "Downloading update..."
    if ! wget -q --show-progress -O "$temp_file" "$download_url"; then
        error "Download failed."
    fi
    
    log "Installing..."
    systemctl stop "$SERVICE_NAME"
    
    # Extract
    tar -xzf "$temp_file" -C "/tmp/"
    
    # Replace Binary Only (Because assets are embedded)
    if [ -f "/tmp/vWireguard" ]; then
        mv "/tmp/vWireguard" "$INSTALL_DIR/vWireguard"
        chmod +x "$INSTALL_DIR/vWireguard"
    else
        # Restore backup if extraction failed
        cp "$backup_dir/vWireguard.old" "$INSTALL_DIR/vWireguard"
        error "New binary not found in archive. Update failed."
    fi

    # Update the CLI tool itself if present
    if [ -f "/tmp/vwg" ]; then
        mv "/tmp/vwg" "$INSTALL_DIR/vwg"
        chmod +x "$INSTALL_DIR/vwg"
    fi
    
    # Clean up
    rm -f "$temp_file" "/tmp/vWireguard" "/tmp/vwg"
    
    # Restart
    systemctl start "$SERVICE_NAME"
    log "✅ Updated to $tag successfully!"
}

# --- Help & Main ---

show_help() {
    cat << EOF
${BLUE}vWireguard Management CLI (vwg)${NC}

Usage: vwg <command> [options]

Commands:
  start          Start service
  stop           Stop service
  restart        Restart service
  status         Check status
  log [n] [-f]   View logs (n=lines, -f=follow)
                 Example: vwg log 100 -f
  update         Update to latest version
  help           Show this message
EOF
}

main() {
    local cmd=${1:-help}
    case "$cmd" in
        start)   cmd_start ;;
        stop)    cmd_stop ;;
        restart) cmd_restart ;;
        status)  cmd_status ;;
        log|logs)
            if [ "$2" == "-f" ]; then
                cmd_log 50 "true"
            elif [ "$3" == "-f" ]; then
                cmd_log "$2" "true"
            else
                cmd_log "$2" "false"
            fi
            ;;
        update|upgrade) cmd_update ;;
        *) show_help ;;
    esac
}

main "$@"
