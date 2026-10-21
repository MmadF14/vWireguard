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

# All three MUST print to stderr (>&2), never stdout. Several functions below
# are called as `x=$(some_func ...)`, and some_func itself calls log/warn to
# report progress. If those went to stdout, their text gets glued onto the
# function's real return value inside the $(...) capture - e.g.
# download_release() used to return "\033[0;32m[..] Downloading ...\033[0m\n/tmp/foo.tar.gz"
# instead of just "/tmp/foo.tar.gz", and the subsequent `tar -xzf "$archive"`
# failed with a confusing "Cannot open" error that had nothing to do with tar.
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

# Install everything the panel shells out to: WireGuard itself, iptables (NAT
# rules in PostUp), iproute2, sudo (the panel runs `sudo wg ...`) and a DNS
# helper for wg-quick's DNS= line. Without these the panel starts fine but
# no tunnel can ever come up.
install_dependencies() {
    log "Installing WireGuard and required packages..."

    if [ -f /etc/debian_version ]; then
        export DEBIAN_FRONTEND=noninteractive
        apt-get update -y
        local pkgs=(wireguard wireguard-tools iptables iproute2 sudo curl wget tar ca-certificates)
        # wg-quick needs a resolvconf provider to apply DNS=; systemd-resolved
        # already supplies one on recent Ubuntu/Debian.
        if ! command -v resolvconf &> /dev/null && ! command -v resolvectl &> /dev/null; then
            pkgs+=(resolvconf)
        fi
        apt-get install -y "${pkgs[@]}" || error "Failed to install required packages"
    elif command -v dnf &> /dev/null || command -v yum &> /dev/null; then
        local pm=yum
        command -v dnf &> /dev/null && pm=dnf
        $pm install -y epel-release 2>/dev/null || true
        $pm install -y wireguard-tools iptables iproute sudo curl wget tar ca-certificates \
            || error "Failed to install required packages"
        # Kernels without built-in WireGuard need the module package.
        if ! modprobe wireguard 2>/dev/null; then
            $pm install -y kmod-wireguard 2>/dev/null || true
        fi
    else
        error "Unsupported distribution. Install manually: wireguard, wireguard-tools, iptables, iproute2, sudo, curl, wget, tar"
    fi

    for cmd in wg wg-quick iptables ip sudo; do
        command -v "$cmd" &> /dev/null || error "'$cmd' is still missing after installation"
    done

    if modprobe wireguard 2>/dev/null || ip link add wgtest type wireguard 2>/dev/null; then
        ip link del wgtest 2>/dev/null || true
    else
        warn "The kernel has no WireGuard support (common on OpenVZ/LXC VPS). Tunnels will not start."
    fi

    log "Dependencies installed ($(wg --version 2>/dev/null | head -n1))"
}

# Packet forwarding must survive reboots; the panel only sets it at runtime.
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

# Optional AmneziaWG (DPI-resistant interfaces): INSTALL_AMNEZIAWG=1 or --amneziawg
install_amneziawg() {
    if ! grep -qi ubuntu /etc/os-release 2>/dev/null; then
        warn "AmneziaWG auto-install supports Ubuntu only. See the README for manual steps."
        return
    fi
    log "Installing AmneziaWG..."
    apt-get install -y software-properties-common "linux-headers-$(uname -r)" \
        || warn "Kernel headers for $(uname -r) not found; the AmneziaWG module may fail to build"
    add-apt-repository -y ppa:amnezia/ppa
    apt-get update -y
    apt-get install -y amneziawg || { warn "AmneziaWG installation failed; plain WireGuard still works"; return; }
    mkdir -p /etc/amnezia/amneziawg
    chmod 700 /etc/amnezia/amneziawg
}

# The panel writes /etc/wireguard/wg0.conf on first start. wg-quick@wg0 has to
# be running for the panel to add peers, and enabled so it returns after reboot.
enable_wireguard_service() {
    log "Waiting for the panel to create its WireGuard config..."
    local i
    for i in $(seq 1 20); do
        [ -f /etc/wireguard/wg0.conf ] && break
        sleep 1
    done

    if [ ! -f /etc/wireguard/wg0.conf ]; then
        warn "No /etc/wireguard/wg0.conf yet. Open the panel, then run: systemctl enable --now wg-quick@wg0"
        return
    fi

    if systemctl enable --now wg-quick@wg0; then
        log "wg-quick@wg0 is running and enabled at boot"
    else
        warn "Could not start wg-quick@wg0. Check: journalctl -u wg-quick@wg0"
    fi
}

# Open the ports if ufw is in use (it silently drops them otherwise)
open_firewall() {
    if command -v ufw &> /dev/null && ufw status 2>/dev/null | grep -q "Status: active"; then
        log "ufw is active, opening panel and WireGuard ports..."
        ufw allow 5000/tcp > /dev/null
        ufw allow 51820/udp > /dev/null
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

    install_dependencies

    enable_ip_forwarding

    if [ "$INSTALL_AMNEZIAWG" = "1" ] || [[ " $* " == *" --amneziawg "* ]]; then
        install_amneziawg
    fi

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

    enable_wireguard_service

    open_firewall
    
    # Cleanup
    rm -f "$archive"
    
    show_summary
}

# Run main function
main "$@"
