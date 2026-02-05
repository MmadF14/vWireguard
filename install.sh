#!/bin/bash

# vWireguard Panel - Binary Release Installation Script
# This script downloads and installs pre-built binaries from GitHub releases
# NO compilation required on the user's server
# Full One-Click Setup (Dependencies + WireGuard + IP Forwarding + Panel)

set -e

# Colors
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

# Display Logo
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

# Logging functions
log() { echo -e "${GREEN}[$(date '+%H:%M:%S')] $1${NC}"; }
error() { echo -e "${RED}[$(date '+%H:%M:%S')] ❌ $1${NC}"; exit 1; }
warn() { echo -e "${YELLOW}[$(date '+%H:%M:%S')] ⚠️  $1${NC}"; }

# 1. Check root access
if [ "$EUID" -ne 0 ]; then 
    error "Please run with root access: sudo bash install.sh"
fi

# 2. Check and Install Prerequisites (Modified to include WireGuard)
check_and_install_prerequisites() {
    log "Checking system prerequisites..."
    
    local missing=()
    local install_cmd=""
    local package_manager=""
    
    # Detect package manager
    if [ -f /etc/debian_version ]; then
        package_manager="apt"
        install_cmd="apt-get install -y"
        # Update repo info first
        apt-get update -y > /dev/null 2>&1
    elif [ -f /etc/redhat-release ]; then
        package_manager="yum"
        install_cmd="yum install -y"
        # Install EPEL for WireGuard on CentOS/RHEL
        if ! rpm -qa | grep -q epel-release; then
            log "Installing EPEL release..."
            yum install -y epel-release > /dev/null 2>&1
        fi
    else
        warn "Unknown OS. Automatic dependency installation might fail."
    fi

    # List of required tools
    local required_tools=("curl" "wget" "tar")
    
    # Check for basic tools
    for tool in "${required_tools[@]}"; do
        if ! command -v "$tool" &> /dev/null; then
            missing+=("$tool")
        fi
    done

    # Check for WireGuard tools (wg command)
    if ! command -v wg &> /dev/null; then
        if [ "$package_manager" == "apt" ]; then
            missing+=("wireguard" "iptables")
        elif [ "$package_manager" == "yum" ]; then
            missing+=("wireguard-tools" "iptables")
        fi
    fi

    # Install missing packages
    if [ ${#missing[@]} -gt 0 ]; then
        log "Installing missing packages: ${missing[*]}..."
        if [ -n "$install_cmd" ]; then
            $install_cmd "${missing[@]}"
        else
            error "Cannot auto-install: ${missing[*]}. Please install them manually."
        fi
    else
        log "All prerequisites (curl, wget, tar, wireguard) are satisfied."
    fi

    # Verify systemd
    if ! command -v systemctl &> /dev/null; then
        error "systemd is required but not found. This script only supports systemd-based systems."
    fi
}

# 3. Enable IP Forwarding (Crucial for VPN)
enable_ip_forwarding() {
    log "Configuring IP forwarding..."
    
    local sysctl_file="/etc/sysctl.d/99-vwireguard.conf"
    
    # Create a persistent config file
    cat > "$sysctl_file" <<EOF
net.ipv4.ip_forward=1
net.ipv6.conf.all.forwarding=1
EOF

    # Apply changes
    sysctl -p "$sysctl_file" > /dev/null 2>&1
    
    log "IP forwarding enabled."
}

# 4. Detect architecture
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

# 5. Get latest release tag from GitHub API
get_latest_release() {
    log "Fetching latest release version..."
    local api_url="https://api.github.com/repos/${REPO_OWNER}/${REPO_NAME}/releases/latest"
    
    # Use sed instead of grep -P for better compatibility
    local tag=$(curl -sL "$api_url" | sed -n 's/.*"tag_name": *"\([^"]*\)".*/\1/p')
    
    if [ -z "$tag" ]; then
        # Fallback method if API fails or limit reached (optional)
        warn "GitHub API failed, trying to guess from redirects..."
        tag=$(curl -sL -o /dev/null -w %{url_effective} "https://github.com/${REPO_OWNER}/${REPO_NAME}/releases/latest" | rev | cut -d/ -f1 | rev)
    fi

    if [ -z "$tag" ] || [ "$tag" == "releases" ]; then
        error "Failed to fetch latest release tag from GitHub."
    fi
    
    echo "$tag"
}

# 6. Download release asset
download_release() {
    local tag=$1
    local arch=$2
    local asset_name="vWireguard-linux-${arch}.tar.gz"
    local download_url="https://github.com/${REPO_OWNER}/${REPO_NAME}/releases/download/${tag}/${asset_name}"
    local temp_file="/tmp/${asset_name}"
    
    log "Downloading ${asset_name} from release ${tag}..."
    
    if ! wget -q --show-progress -O "$temp_file" "$download_url"; then
        error "Failed to download release asset. Please check if the asset '${asset_name}' exists in release '${tag}'."
    fi
    
    if [ ! -f "$temp_file" ] || [ ! -s "$temp_file" ]; then
        error "Downloaded file is empty or missing"
    fi
    
    echo "$temp_file"
}

# 7. Stop service if running
stop_service() {
    if systemctl is-active --quiet "$SERVICE_NAME" 2>/dev/null; then
        log "Stopping existing ${SERVICE_NAME} service..."
        systemctl stop "$SERVICE_NAME" || warn "Failed to stop service"
    fi
}

# 8. Preserve existing data
preserve_data() {
    if [ -d "$INSTALL_DIR/db" ]; then
        log "Backing up existing database..."
        local backup_dir="/tmp/vwireguard-db-backup-$(date +%Y%m%d_%H%M%S)"
        mkdir -p "$backup_dir"
        
        # Backup db folder
        cp -r "$INSTALL_DIR/db" "$backup_dir/" 2>/dev/null || true
        
        # Also backup config file if it exists
        if [ -f "$INSTALL_DIR/config.json" ]; then
             cp "$INSTALL_DIR/config.json" "$backup_dir/" 2>/dev/null || true
        fi
        
        echo "$backup_dir"
    fi
}

# 9. Extract and install
install_files() {
    local archive=$1
    local backup_dir=$2
    
    log "Extracting release package to ${INSTALL_DIR}..."
    
    # Create installation directory
    mkdir -p "$INSTALL_DIR"
    
    # Extract archive
    # We use -C to extract directly into the dir
    tar -xzf "$archive" -C "$INSTALL_DIR" || error "Failed to extract archive"
    
    # Restore database if it existed
    if [ -n "$backup_dir" ] && [ -d "$backup_dir/db" ]; then
        log "Restoring database from backup..."
        cp -r "$backup_dir/db"/* "$INSTALL_DIR/db/" 2>/dev/null || true
        
        if [ -f "$backup_dir/config.json" ]; then
            cp "$backup_dir/config.json" "$INSTALL_DIR/" 2>/dev/null || true
        fi
        
        rm -rf "$backup_dir"
    fi
    
    # Set permissions
    chmod +x "$INSTALL_DIR/vWireguard"
    if [ -f "$INSTALL_DIR/vwg" ]; then
        chmod +x "$INSTALL_DIR/vwg"
    fi
    
    # Ensure db directory structure exists
    mkdir -p "$INSTALL_DIR/db/{clients,server,users,wake_on_lan_hosts,tunnels}"
    
    log "Files installed successfully"
}

# 10. Create systemd service
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

# 11. Create symlink for vwg command
create_symlink() {
    if [ -f "$INSTALL_DIR/vwg" ]; then
        if [ -L "/usr/bin/vwg" ] || [ -f "/usr/bin/vwg" ]; then
            rm -f "/usr/bin/vwg"
        fi
        ln -s "$INSTALL_DIR/vwg" "/usr/bin/vwg"
        chmod +x "/usr/bin/vwg"
        log "Management command 'vwg' installed to /usr/bin/vwg"
    else
        warn "'vwg' management script not found in archive. Skipping symlink."
    fi
}

# 12. Start service
start_service() {
    log "Starting ${SERVICE_NAME} service..."
    systemctl start "$SERVICE_NAME"
    
    sleep 2
    
    if systemctl is-active --quiet "$SERVICE_NAME"; then
        log "Service started successfully"
    else
        # Try to show logs if failed
        warn "Service failed to start instantly. Checking logs..."
        journalctl -u "$SERVICE_NAME" --no-pager -n 10
        error "Failed to start service. Please check logs above."
    fi
}

# 13. Show installation summary
show_summary() {
    # Try to get public IP
    local public_ip=$(curl -s --connect-timeout 5 ifconfig.me 2>/dev/null || curl -s --connect-timeout 5 icanhazip.com 2>/dev/null || echo "YOUR_SERVER_IP")
    
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
    echo -e "  ${YELLOW}Logs:${NC}   vwg log"
    echo -e "  ${YELLOW}Restart:${NC} vwg restart"
    echo -e "  ${YELLOW}Update:${NC}  vwg update"
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
    
    # Step 1: Install Dependencies (Curl, Wget, WireGuard, IPtables)
    check_and_install_prerequisites
    
    # Step 2: Enable IP Forwarding
    enable_ip_forwarding
    
    # Step 3: Architecture Check
    local arch=$(detect_arch)
    log "Detected architecture: $arch"
    
    # Step 4: Get Version
    local tag=$(get_latest_release)
    log "Latest release: $tag"
    
    # Step 5: Download
    local archive=$(download_release "$tag" "$arch")
    
    # Step 6: Stop old service
    stop_service
    
    # Step 7: Backup Data
    local backup_dir=$(preserve_data)
    
    # Step 8: Install Files
    install_files "$archive" "$backup_dir"
    
    # Step 9: Create Service
    create_service
    
    # Step 10: Create CLI symlink
    create_symlink
    
    # Step 11: Start
    start_service
    
    # Cleanup
    rm -f "$archive"
    
    # Done
    show_summary
}

# Run main function
main "$@"
