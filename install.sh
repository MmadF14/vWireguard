#!/bin/bash

# vWireguard Panel - Binary Release Installation Script
# This script downloads and installs pre-built binaries from GitHub releases
# NO compilation required on the user's server

set -e

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m'

# Repository information
REPO_OWNER="MmadF14"
REPO_NAME="vwireguard"
INSTALL_DIR="/usr/local/vwireguard"
SERVICE_NAME="vwireguard"

echo -e "${BLUE}"
cat << "EOF"
██╗   ██╗██╗    ██╗██╗██████╗ ███████╗ ██████╗ ██╗   ██╗ █████╗ ██████╗ ██████╗ 
██║   ██║██║    ██║██║██╔══██╗██╔════╝██╔════╝ ██║   ██║██╔══██╗██╔══██╗██╔══██╗
██║   ██║██║ █╗ ██║██║██████╔╝█████╗  ██║  ███╗██║   ██║███████║██████╔╝██║  ██║
╚██╗ ██╔╝██║███╗██║██║██╔══██╗██╔══╝  ██║   ██║██║   ██║██╔══██║██╔══██╗██║  ██║
 ╚████╔╝ ╚███╔███╔╝██║██║  ██║███████╗╚██████╔╝╚██████╔╝██║  ██║██║  ██║██████╔╝
  ╚═══╝   ╚══╝╚══╝ ╚═╝╚═╝  ╚═╝╚══════╝ ╚═════╝  ╚═════╝ ╚═╝  ╚═╝╚═╝  ╚═╝╚═════╝ 
EOF
echo -e "${NC}"

log() { echo -e "${GREEN}[$(date '+%H:%M:%S')] $1${NC}"; }
error() { echo -e "${RED}[$(date '+%H:%M:%S')] ❌ $1${NC}"; exit 1; }
warn() { echo -e "${YELLOW}[$(date '+%H:%M:%S')] ⚠️  $1${NC}"; }

# Check root access
if [ "$EUID" -ne 0 ]; then 
    error "Please run with root access: sudo bash install.sh"
fi

# Check prerequisites
check_prerequisites() {
    log "Checking prerequisites..."
    
    local missing=()
    
    if ! command -v curl &> /dev/null; then
        missing+=("curl")
    fi
    
    if ! command -v wget &> /dev/null; then
        missing+=("wget")
    fi
    
    if ! command -v tar &> /dev/null; then
        missing+=("tar")
    fi
    
    if ! command -v systemctl &> /dev/null; then
        error "systemd is required but not found. This script only supports systemd-based systems."
    fi
    
    if [ ${#missing[@]} -gt 0 ]; then
        warn "Missing required tools: ${missing[*]}"
        log "Installing missing packages..."
        
        if [ -f /etc/debian_version ]; then
            apt-get update -y
            apt-get install -y "${missing[@]}"
        elif [ -f /etc/redhat-release ]; then
            yum install -y "${missing[@]}"
        else
            error "Cannot auto-install prerequisites. Please install: ${missing[*]}"
        fi
    fi
    
    log "All prerequisites satisfied"
}

# Detect architecture
detect_arch() {
    local arch=$(uname -m)
    case $arch in
        x86_64)
            echo "amd64"
            ;;
        aarch64|arm64)
            echo "arm64"
            ;;
        *)
            error "Unsupported architecture: $arch. Supported: amd64, arm64"
            ;;
    esac
}

# Get latest release tag from GitHub API
get_latest_release() {
    local api_url="https://api.github.com/repos/${REPO_OWNER}/${REPO_NAME}/releases/latest"
    local tag=$(curl -sL "$api_url" | grep -oP '"tag_name": "\K[^"]*' | head -n1)
    
    if [ -z "$tag" ]; then
        error "Failed to fetch latest release tag from GitHub"
    fi
    
    echo "$tag"
}

# Download release asset
download_release() {
    local tag=$1
    local arch=$2
    local asset_name="vWireguard-linux-${arch}.tar.gz"
    local download_url="https://github.com/${REPO_OWNER}/${REPO_NAME}/releases/download/${tag}/${asset_name}"
    local temp_file="/tmp/${asset_name}"
    
    log "Downloading ${asset_name} from release ${tag}..."
    
    if ! wget -q --show-progress -O "$temp_file" "$download_url"; then
        error "Failed to download release asset. URL: $download_url"
    fi
    
    if [ ! -f "$temp_file" ] || [ ! -s "$temp_file" ]; then
        error "Downloaded file is empty or missing"
    fi
    
    echo "$temp_file"
}

# Stop service if running
stop_service() {
    if systemctl is-active --quiet "$SERVICE_NAME" 2>/dev/null; then
        log "Stopping ${SERVICE_NAME} service..."
        systemctl stop "$SERVICE_NAME" || warn "Failed to stop service (may not exist yet)"
    fi
}

# Preserve existing data
preserve_data() {
    if [ -d "$INSTALL_DIR/db" ]; then
        log "Backing up existing database..."
        local backup_dir="/tmp/vwireguard-db-backup-$(date +%Y%m%d_%H%M%S)"
        mkdir -p "$backup_dir"
        cp -r "$INSTALL_DIR/db" "$backup_dir/" 2>/dev/null || true
        echo "$backup_dir"
    fi
}

# Extract and install
install_files() {
    local archive=$1
    local backup_dir=$2
    
    log "Extracting release package..."
    
    # Create installation directory
    mkdir -p "$INSTALL_DIR"
    cd "$INSTALL_DIR"
    
    # Extract archive
    tar -xzf "$archive" || error "Failed to extract archive"
    
    # Restore database if it existed
    if [ -n "$backup_dir" ] && [ -d "$backup_dir/db" ]; then
        log "Restoring database from backup..."
        cp -r "$backup_dir/db"/* "$INSTALL_DIR/db/" 2>/dev/null || true
        rm -rf "$backup_dir"
    fi
    
    # Set permissions
    chmod +x "$INSTALL_DIR/vWireguard"
    chmod +x "$INSTALL_DIR/vwg"
    
    # Ensure db directory structure exists
    mkdir -p "$INSTALL_DIR/db/{clients,server,users,wake_on_lan_hosts,tunnels}"
    
    log "Files installed successfully"
}

# Create systemd service
create_service() {
    log "Creating systemd service..."
    
    cat > "/etc/systemd/system/${SERVICE_NAME}.service" <<EOF
[Unit]
Description=vWireguard Panel
After=network.target

[Service]
Type=simple
WorkingDirectory=${INSTALL_DIR}
ExecStart=${INSTALL_DIR}/vWireguard
Restart=always
RestartSec=3
User=root
Environment=PORT=5000

[Install]
WantedBy=multi-user.target
EOF
    
    systemctl daemon-reload
    systemctl enable "$SERVICE_NAME"
    log "Service created and enabled"
}

# Create symlink for vwg command
create_symlink() {
    if [ -L "/usr/bin/vwg" ]; then
        rm -f "/usr/bin/vwg"
    fi
    ln -s "$INSTALL_DIR/vwg" "/usr/bin/vwg"
    chmod +x "/usr/bin/vwg"
    log "Management command 'vwg' installed to /usr/bin/vwg"
}

# Start service
start_service() {
    log "Starting ${SERVICE_NAME} service..."
    systemctl start "$SERVICE_NAME"
    
    sleep 2
    
    if systemctl is-active --quiet "$SERVICE_NAME"; then
        log "Service started successfully"
    else
        error "Failed to start service. Check logs: journalctl -u ${SERVICE_NAME}"
    fi
}

# Show installation summary
show_summary() {
    local public_ip=$(curl -s ifconfig.me 2>/dev/null || curl -s icanhazip.com 2>/dev/null || echo "localhost")
    
    echo ""
    echo -e "${GREEN}=====================================${NC}"
    echo -e "${GREEN}✅ Installation completed successfully!${NC}"
    echo -e "${GREEN}=====================================${NC}"
    echo ""
    echo -e "${CYAN}📋 Panel Information:${NC}"
    echo -e "  ${YELLOW}URL:${NC} http://${public_ip}:5000"
    echo -e "  ${YELLOW}Username:${NC} admin"
    echo -e "  ${YELLOW}Password:${NC} admin"
    echo ""
    echo -e "${CYAN}📁 Installation Directory:${NC}"
    echo -e "  ${YELLOW}Path:${NC} ${INSTALL_DIR}"
    echo ""
    echo -e "${CYAN}⚙️  Useful Commands:${NC}"
    echo -e "  ${YELLOW}Status:${NC} vwg status"
    echo -e "  ${YELLOW}Logs:${NC} vwg log"
    echo -e "  ${YELLOW}Restart:${NC} vwg restart"
    echo -e "  ${YELLOW}Update:${NC} vwg update"
    echo ""
    echo -e "${GREEN}🎉 Panel is ready!${NC}"
    echo -e "${GREEN}=====================================${NC}"
    
    # Save info to file
    cat > /root/vwireguard-info.txt <<EOF
vWireguard Panel Information
===========================
Panel URL: http://${public_ip}:5000
Username: admin
Password: admin

Installation Directory: ${INSTALL_DIR}
Service Name: ${SERVICE_NAME}

Management Commands:
  vwg status   - Check service status
  vwg start    - Start service
  vwg stop     - Stop service
  vwg restart  - Restart service
  vwg log      - View logs
  vwg update   - Update to latest version
EOF
}

# Main installation function
main() {
    log "Starting vWireguard installation..."
    
    check_prerequisites
    
    local arch=$(detect_arch)
    log "Detected architecture: $arch"
    
    local tag=$(get_latest_release)
    log "Latest release: $tag"
    
    local archive=$(download_release "$tag" "$arch")
    
    stop_service
    
    local backup_dir=$(preserve_data)
    
    install_files "$archive" "$backup_dir"
    
    create_service
    
    create_symlink
    
    start_service
    
    # Cleanup
    rm -f "$archive"
    
    show_summary
}

# Run main function
main "$@"
