#!/bin/bash

# vWireguard Panel - Update Script
set -e

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m'

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
echo -e "${CYAN}=== vWireguard Panel - Update ===${NC}"
echo ""

# Check if running as root
if [ "$EUID" -ne 0 ]; then 
    echo -e "${RED}❌ Please run with root privileges:${NC}"
    echo -e "${GREEN}sudo bash update.sh${NC}"
    exit 1
fi

REPO_OWNER="MmadF14"
REPO_NAME="vwireguard"
SERVICE_NAME="vwireguard"

# Detect installation directory
if [ -d "/usr/local/vwireguard" ]; then
    VWIREGUARD_DIR="/usr/local/vwireguard"
elif [ -d "/opt/vwireguard" ]; then
    VWIREGUARD_DIR="/opt/vwireguard"
else
    VWIREGUARD_DIR="/usr/local/vwireguard"
fi

BACKUP_DIR="/tmp/vwireguard-backup-$(date +%Y%m%d_%H%M%S)"

log() { echo -e "${GREEN}[$(date '+%H:%M:%S')] $1${NC}"; }
error() { echo -e "${RED}[$(date '+%H:%M:%S')] ❌ $1${NC}"; exit 1; }
warn() { echo -e "${YELLOW}[$(date '+%H:%M:%S')] ⚠️  $1${NC}"; }

# Check if vWireguard is installed
check_installation() {
    if [ ! -d "$VWIREGUARD_DIR" ]; then
        error "vWireguard is not installed at ${VWIREGUARD_DIR}. Please install first with install.sh"
    fi
    log "vWireguard installation found at ${VWIREGUARD_DIR}"
}

# Install or verify required system dependencies (WireGuard & AmneziaWG)
install_dependencies() {
    log "Checking system dependencies (WireGuard & AmneziaWG)..."
    
    local pkgs_needed=()
    if ! command -v wg &> /dev/null || ! command -v wg-quick &> /dev/null; then
        pkgs_needed+=(wireguard wireguard-tools)
    fi
    if ! command -v iptables &> /dev/null; then
        pkgs_needed+=(iptables)
    fi
    if ! command -v ip &> /dev/null; then
        pkgs_needed+=(iproute2)
    fi
    if ! command -v sudo &> /dev/null; then
        pkgs_needed+=(sudo)
    fi

    if [ ${#pkgs_needed[@]} -gt 0 ]; then
        log "Installing missing base packages: ${pkgs_needed[*]}..."
        if [ -f /etc/debian_version ]; then
            export DEBIAN_FRONTEND=noninteractive
            apt-get update -y
            apt-get install -y "${pkgs_needed[@]}" || warn "Failed to install some base packages."
        elif command -v dnf &> /dev/null || command -v yum &> /dev/null; then
            local pm=yum
            command -v dnf &> /dev/null && pm=dnf
            $pm install -y epel-release 2>/dev/null || true
            $pm install -y wireguard-tools iptables iproute sudo || warn "Failed to install some base packages."
        fi
    fi

    # Check AmneziaWG tools
    if ! command -v awg &> /dev/null || ! command -v awg-quick &> /dev/null; then
        log "AmneziaWG tools not found. Installing AmneziaWG..."
        if grep -qi ubuntu /etc/os-release 2>/dev/null; then
            export DEBIAN_FRONTEND=noninteractive
            apt-get update -y
            apt-get install -y software-properties-common "linux-headers-$(uname -r)" 2>/dev/null || true
            add-apt-repository -y ppa:amnezia/ppa
            apt-get update -y
            apt-get install -y amneziawg amneziawg-tools 2>/dev/null || warn "AmneziaWG package install failed."
        elif grep -qi debian /etc/os-release 2>/dev/null; then
            export DEBIAN_FRONTEND=noninteractive
            apt-get update -y
            apt-get install -y curl gpg "linux-headers-$(uname -r)" 2>/dev/null || true
            curl -s --compressed "https://ppa.launchpadcontent.net/amnezia/ppa/ubuntu/dists/jammy/Release.gpg" | gpg --dearmor -o /etc/apt/trusted.gpg.d/amnezia.gpg 2>/dev/null || true
            echo "deb https://ppa.launchpadcontent.net/amnezia/ppa/ubuntu jammy main" > /etc/apt/sources.list.d/amnezia.list
            apt-get update -y
            apt-get install -y amneziawg amneziawg-tools 2>/dev/null || warn "AmneziaWG package install failed."
        fi
        mkdir -p /etc/amnezia/amneziawg
        chmod 700 /etc/amnezia/amneziawg
    else
        log "AmneziaWG is already installed."
    fi

    # Ensure IP forwarding
    log "Ensuring IP forwarding is enabled..."
    cat > /etc/sysctl.d/99-vwireguard.conf <<SYSCTL
net.ipv4.ip_forward=1
net.ipv6.conf.all.forwarding=1
SYSCTL
    sysctl --system > /dev/null 2>&1 || sysctl -w net.ipv4.ip_forward=1 > /dev/null
    mkdir -p /etc/wireguard
    chmod 700 /etc/wireguard
}

# Get current version
get_current_version() {
    cd "$VWIREGUARD_DIR"
    local bin=""
    [ -f "./vWireguard" ] && bin="./vWireguard"
    [ -z "$bin" ] && [ -f "./vwireguard" ] && bin="./vwireguard"

    if [ -n "$bin" ]; then
        CURRENT_VERSION=$($bin -version 2>/dev/null || echo "unknown")
    else
        CURRENT_VERSION="unknown"
    fi
    log "Current version: $CURRENT_VERSION"
}

# Create backup
create_backup() {
    log "Creating backup..."
    mkdir -p "$BACKUP_DIR"
    
    # Backup database
    if [ -d "$VWIREGUARD_DIR/db" ]; then
        cp -r "$VWIREGUARD_DIR/db" "$BACKUP_DIR/" 2>/dev/null || true
    fi
    
    # Backup binary
    if [ -f "$VWIREGUARD_DIR/vWireguard" ]; then
        cp "$VWIREGUARD_DIR/vWireguard" "$BACKUP_DIR/vWireguard.old" 2>/dev/null || true
    elif [ -f "$VWIREGUARD_DIR/vwireguard" ]; then
        cp "$VWIREGUARD_DIR/vwireguard" "$BACKUP_DIR/vWireguard.old" 2>/dev/null || true
    fi
    
    log "Backup saved to $BACKUP_DIR"
}

# Stop services
stop_services() {
    if systemctl is-active --quiet "$SERVICE_NAME" 2>/dev/null; then
        log "Stopping ${SERVICE_NAME} service..."
        systemctl stop "$SERVICE_NAME"
        sleep 1
    fi
}

# Download and install new version
update_binary() {
    log "Downloading new version..."
    
    cd "$VWIREGUARD_DIR"
    local arch=$(uname -m)
    case $arch in
        x86_64) arch="amd64" ;;
        aarch64|arm64) arch="arm64" ;;
        *) error "Unsupported architecture: $arch" ;;
    esac

    local tag=""
    local api_url="https://api.github.com/repos/${REPO_OWNER}/${REPO_NAME}/releases/latest"
    
    # 1. Try GitHub API
    tag=$(curl -sL --max-time 10 -H "User-Agent: vWireguard-Updater" "$api_url" 2>/dev/null | grep -oP '"tag_name": "\K[^"]*' | head -n1 || true)
    
    # 2. Fallback to git ls-remote if API is rate-limited or fails
    if [ -z "$tag" ]; then
        tag=$(git ls-remote --tags --refs "https://github.com/${REPO_OWNER}/${REPO_NAME}.git" 2>/dev/null | cut -d/ -f3 | grep -E '^v[0-9]+' | sort -V | tail -n1 || true)
    fi

    # 3. Fallback to hardcoded known stable release if network blocks both
    if [ -z "$tag" ]; then
        tag="v1.0.5"
    fi

    log "Target release: $tag"

    local asset_name="vWireguard-linux-${arch}.tar.gz"
    local download_url="https://github.com/${REPO_OWNER}/${REPO_NAME}/releases/download/${tag}/${asset_name}"
    local temp_file="/tmp/${asset_name}"
    
    log "Downloading ${asset_name}..."
    if ! wget -q --show-progress -O "$temp_file" "$download_url"; then
        error "Failed to download release from $download_url"
    fi
    
    local temp_extract="/tmp/vwireguard-extract-$$"
    mkdir -p "$temp_extract"
    tar -xzf "$temp_file" -C "$temp_extract"
    
    # Install new files
    log "Installing new files..."
    cp "$temp_extract/vWireguard" "$VWIREGUARD_DIR/vWireguard"
    chmod +x "$VWIREGUARD_DIR/vWireguard"

    if [ -d "$temp_extract/static" ]; then
        cp -r "$temp_extract/static" "$VWIREGUARD_DIR/"
    fi
    if [ -d "$temp_extract/templates" ]; then
        cp -r "$temp_extract/templates" "$VWIREGUARD_DIR/"
    fi
    if [ -d "$temp_extract/custom" ]; then
        cp -r "$temp_extract/custom" "$VWIREGUARD_DIR/"
    fi
    if [ -f "$temp_extract/vwg" ]; then
        cp "$temp_extract/vwg" "$VWIREGUARD_DIR/vwg"
        chmod +x "$VWIREGUARD_DIR/vwg"
        ln -sf "$VWIREGUARD_DIR/vwg" "/usr/bin/vwg" 2>/dev/null || true
    fi

    # Cleanup
    rm -rf "$temp_extract" "$temp_file"
    log "Files updated successfully"
}

# Start services
start_services() {
    log "Starting ${SERVICE_NAME} service..."
    systemctl start "$SERVICE_NAME"
    sleep 2
    
    if systemctl is-active --quiet "$SERVICE_NAME"; then
        log "Service started successfully"
    else
        error "Failed to start service"
        log "Restoring from backup..."
        restore_backup
        exit 1
    fi
}

# Restore backup if update fails
restore_backup() {
    log "Restoring from backup..."
    systemctl stop "$SERVICE_NAME" || true
    
    if [ -f "$BACKUP_DIR/vWireguard.old" ]; then
        cp "$BACKUP_DIR/vWireguard.old" "$VWIREGUARD_DIR/vWireguard"
        chmod +x "$VWIREGUARD_DIR/vWireguard"
    fi
    
    systemctl start "$SERVICE_NAME"
    if systemctl is-active --quiet "$SERVICE_NAME"; then
        log "Restore successful"
    else
        error "Failed to restore - please check logs: journalctl -u ${SERVICE_NAME}"
    fi
}

# Get new version
get_new_version() {
    cd "$VWIREGUARD_DIR"
    NEW_VERSION=$("./vWireguard" -version 2>/dev/null || echo "latest")
    log "New version: $NEW_VERSION"
}

# Show update summary
show_summary() {
    echo ""
    echo -e "${GREEN}=====================================${NC}"
    echo -e "${GREEN}✅ Update completed successfully!${NC}"
    echo -e "${GREEN}=====================================${NC}"
    echo ""
    echo -e "${CYAN}📋 Update Information:${NC}"
    echo -e "  ${YELLOW}Previous version:${NC} $CURRENT_VERSION"
    echo -e "  ${YELLOW}New version:${NC} $NEW_VERSION"
    echo -e "  ${YELLOW}Backup location:${NC} $BACKUP_DIR"
    echo ""
    echo -e "${CYAN}⚙️ Useful commands:${NC}"
    echo -e "  ${YELLOW}Status:${NC} vwg status"
    echo -e "  ${YELLOW}Logs:${NC} vwg log"
    echo -e "  ${YELLOW}Dependencies:${NC} vwg install-deps"
    echo -e "  ${YELLOW}Restart:${NC} vwg restart"
    echo ""
    echo -e "${GREEN}🎉 Panel updated successfully!${NC}"
    echo -e "${GREEN}=====================================${NC}"
}

# Main function
main() {
    log "Starting vWireguard update..."
    
    check_installation
    get_current_version
    
    # Verify and install any new system dependencies
    install_dependencies
    
    create_backup
    stop_services
    update_binary
    start_services
    get_new_version
    show_summary
}

# Run main function
main "$@"
