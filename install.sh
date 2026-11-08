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

# All logs go to stderr so command substitutions don't capture them.
log() { echo -e "${GREEN}[$(date '+%H:%M:%S')] $1${NC}" >&2; }
error() { echo -e "${RED}[$(date '+%H:%M:%S')] ❌ $1${NC}" >&2; exit 1; }
warn() { echo -e "${YELLOW}[$(date '+%H:%M:%S')] ⚠️  $1${NC}" >&2; }

# Check root access
if [ "$EUID" -ne 0 ]; then 
    error "Please run with root access: sudo bash install.sh"
fi

# Check prerequisites
check_prerequisites() {
    log "Checking prerequisites..."
    
    local missing=()
    for cmd in curl wget tar; do
        if ! command -v "$cmd" &> /dev/null; then
            missing+=("$cmd")
        fi
    done
    
    if ! command -v systemctl &> /dev/null; then
        error "systemd is required but not found. This script only supports systemd-based systems."
    fi
    
    if [ ${#missing[@]} -gt 0 ]; then
        warn "Missing required tools: ${missing[*]}"
        log "Installing missing basic packages..."
        
        if [ -f /etc/debian_version ]; then
            export DEBIAN_FRONTEND=noninteractive
            apt-get update -y
            apt-get install -y "${missing[@]}"
        elif command -v dnf &> /dev/null || command -v yum &> /dev/null; then
            local pm=yum
            command -v dnf &> /dev/null && pm=dnf
            $pm install -y "${missing[@]}"
        else
            error "Cannot auto-install prerequisites. Please install: ${missing[*]}"
        fi
    fi
    
    log "Basic prerequisites satisfied"
}

# Install WireGuard, iptables, iproute2, sudo and resolvconf
install_dependencies() {
    log "Installing WireGuard and network dependencies..."

    if [ -f /etc/debian_version ]; then
        export DEBIAN_FRONTEND=noninteractive
        apt-get update -y
        local pkgs=(wireguard wireguard-tools iptables iproute2 sudo curl wget tar ca-certificates)
        if ! command -v resolvconf &> /dev/null && ! command -v resolvectl &> /dev/null; then
            pkgs+=(resolvconf)
        fi
        apt-get install -y "${pkgs[@]}" || error "Failed to install required WireGuard packages"
    elif command -v dnf &> /dev/null || command -v yum &> /dev/null; then
        local pm=yum
        command -v dnf &> /dev/null && pm=dnf
        $pm install -y epel-release 2>/dev/null || true
        $pm install -y wireguard-tools iptables iproute sudo curl wget tar ca-certificates \
            || error "Failed to install required WireGuard packages"
        if ! modprobe wireguard 2>/dev/null; then
            $pm install -y kmod-wireguard 2>/dev/null || true
        fi
    else
        error "Unsupported distribution. Please install manually: wireguard, wireguard-tools, iptables, iproute2, sudo"
    fi

    for cmd in wg wg-quick iptables ip sudo; do
        command -v "$cmd" &> /dev/null || error "'$cmd' is still missing after installation"
    done

    if modprobe wireguard 2>/dev/null || ip link add wgtest type wireguard 2>/dev/null; then
        ip link del wgtest 2>/dev/null || true
    else
        warn "Kernel WireGuard support not detected (common in OpenVZ/LXC VPS). Kernel module might be needed."
    fi

    log "WireGuard dependencies installed ($(wg --version 2>/dev/null | head -n1 || echo 'wireguard-tools'))"
}

# Install AmneziaWG (DPI-resistant tunnels)
install_amneziawg() {
    log "Checking and installing AmneziaWG..."
    if command -v awg &> /dev/null && command -v awg-quick &> /dev/null; then
        log "AmneziaWG tools are already installed"
        mkdir -p /etc/amnezia/amneziawg
        chmod 700 /etc/amnezia/amneziawg
        return
    fi

    if grep -qi ubuntu /etc/os-release 2>/dev/null; then
        log "Installing AmneziaWG on Ubuntu..."
        export DEBIAN_FRONTEND=noninteractive
        apt-get install -y software-properties-common "linux-headers-$(uname -r)" 2>/dev/null || true
        add-apt-repository -y ppa:amnezia/ppa
        apt-get update -y
        apt-get install -y amneziawg amneziawg-tools || warn "AmneziaWG package install failed; plain WireGuard will still work."
    elif grep -qi debian /etc/os-release 2>/dev/null; then
        log "Installing AmneziaWG on Debian..."
        export DEBIAN_FRONTEND=noninteractive
        apt-get install -y curl gpg "linux-headers-$(uname -r)" 2>/dev/null || true
        curl -s --compressed "https://ppa.launchpadcontent.net/amnezia/ppa/ubuntu/dists/jammy/Release.gpg" | gpg --dearmor -o /etc/apt/trusted.gpg.d/amnezia.gpg 2>/dev/null || true
        echo "deb https://ppa.launchpadcontent.net/amnezia/ppa/ubuntu jammy main" > /etc/apt/sources.list.d/amnezia.list
        apt-get update -y
        apt-get install -y amneziawg amneziawg-tools || warn "AmneziaWG package install failed; plain WireGuard will still work."
    else
        warn "AmneziaWG auto-install currently supports Ubuntu/Debian. Plain WireGuard is ready."
    fi

    mkdir -p /etc/amnezia/amneziawg
    chmod 700 /etc/amnezia/amneziawg
}

# Enable and persist IP Forwarding
enable_ip_forwarding() {
    log "Enabling IP forwarding..."
    cat > /etc/sysctl.d/99-vwireguard.conf <<SYSCTL
net.ipv4.ip_forward=1
net.ipv6.conf.all.forwarding=1
SYSCTL
    sysctl --system > /dev/null 2>&1 || sysctl -w net.ipv4.ip_forward=1 > /dev/null
    mkdir -p /etc/wireguard
    chmod 700 /etc/wireguard
}

# Open firewall ports if ufw is active
open_firewall() {
    if command -v ufw &> /dev/null && ufw status 2>/dev/null | grep -q "Status: active"; then
        log "ufw is active, opening panel and WireGuard ports..."
        ufw allow 5000/tcp > /dev/null || true
        ufw allow 51820:51830/udp > /dev/null || true
    fi
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

# Get latest release tag from GitHub API or Git tags
get_latest_release() {
    local tag=""
    local api_url="https://api.github.com/repos/${REPO_OWNER}/${REPO_NAME}/releases/latest"
    
    # 1. Try GitHub API
    tag=$(curl -sL --max-time 10 -H "User-Agent: vWireguard-Installer" "$api_url" 2>/dev/null | grep -oP '"tag_name": "\K[^"]*' | head -n1 || true)
    
    # 2. Fallback to git ls-remote if API is rate-limited or fails
    if [ -z "$tag" ]; then
        tag=$(git ls-remote --tags --refs "https://github.com/${REPO_OWNER}/${REPO_NAME}.git" 2>/dev/null | cut -d/ -f3 | grep -E '^v[0-9]+' | sort -V | tail -n1 || true)
    fi

    # 3. Fallback to hardcoded known stable release if network blocks both
    if [ -z "$tag" ]; then
        tag="v1.0.5"
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
    mkdir -p "$INSTALL_DIR"/db/{clients,server,users,wake_on_lan_hosts,tunnels}
    
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
    echo -e "  ${YELLOW}Dependencies:${NC} vwg install-deps"
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
  vwg status       - Check service status
  vwg start        - Start service
  vwg stop         - Stop service
  vwg restart      - Restart service
  vwg log          - View logs
  vwg update       - Update to latest version
  vwg install-deps - Check and install dependencies (WireGuard & AmneziaWG)
EOF
}

# Main installation function
main() {
    log "Starting vWireguard installation..."
    
    check_prerequisites
    
    install_dependencies
    
    install_amneziawg
    
    enable_ip_forwarding
    
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
    
    open_firewall
    
    # Cleanup
    rm -f "$archive"
    
    show_summary
}

# Run main function
main "$@"
