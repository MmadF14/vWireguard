#!/bin/bash

set -e

REPO_OWNER="MmadF14"
REPO_NAME="vwireguard"
INSTALL_DIR="/usr/local/vwireguard"
SERVICE_NAME="vwireguard"

echo -e "\033[0;34m"
cat << "EOF"
██╗   ██╗██╗    ██╗██╗██████╗ ███████╗ ██████╗ ██╗   ██╗ █████╗ ██████╗ ██████╗ 
██║   ██║██║    ██║██║██╔══██╗██╔════╝██╔════╝ ██║   ██║██╔══██╗██╔══██╗██╔══██╗
██║   ██║██║ █╗ ██║██║██████╔╝█████╗  ██║  ███╗██║   ██║███████║██████╔╝██║  ██║
╚██╗ ██╔╝██║███╗██║██║██╔══██╗██╔══╝  ██║   ██║██║   ██║██╔══██║██╔══██╗██║  ██║
 ╚████╔╝ ╚███╔███╔╝██║██║  ██║███████╗╚██████╔╝╚██████╔╝██║  ██║██║  ██║██████╔╝
  ╚═══╝   ╚══╝╚══╝ ╚═╝╚═╝  ╚═╝╚══════╝ ╚═════╝  ╚═════╝ ╚═╝  ╚═╝╚═╝  ╚═╝╚═════╝ 
EOF
echo -e "\033[0m"

log() { echo -e "\033[0;32m[$(date '+%H:%M:%S')] $1\033[0m" >&2; }
error() { echo -e "\033[0;31m[$(date '+%H:%M:%S')] ❌ $1\033[0m" >&2; exit 1; }
warn() { echo -e "\033[1;33m[$(date '+%H:%M:%S')] ⚠️  $1\033[0m" >&2; }

if [ "$EUID" -ne 0 ]; then 
    error "Please run with root access: sudo bash install.sh"
fi

check_and_install_prerequisites() {
    log "Checking system prerequisites..."
    
    local missing=()
    local install_cmd=""
    local package_manager=""
    
    if [ -f /etc/debian_version ]; then
        package_manager="apt"
        install_cmd="apt-get install -y"
        apt-get update -y > /dev/null 2>&1
    elif [ -f /etc/redhat-release ]; then
        package_manager="yum"
        install_cmd="yum install -y"
        if ! rpm -qa | grep -q epel-release; then
            yum install -y epel-release > /dev/null 2>&1
        fi
    else
        warn "Unknown OS. Automatic dependency installation might fail."
    fi

    local required_tools=("curl" "wget" "tar")
    for tool in "${required_tools[@]}"; do
        if ! command -v "$tool" &> /dev/null; then
            missing+=("$tool")
        fi
    done

    if ! command -v wg &> /dev/null; then
        if [ "$package_manager" == "apt" ]; then
            missing+=("wireguard" "iptables")
        elif [ "$package_manager" == "yum" ]; then
            missing+=("wireguard-tools" "iptables")
        fi
    fi

    if [ ${#missing[@]} -gt 0 ]; then
        log "Installing missing packages: ${missing[*]}..."
        if [ -n "$install_cmd" ]; then
            $install_cmd "${missing[@]}" > /dev/null 2>&1
        else
            error "Cannot auto-install: ${missing[*]}. Please install them manually."
        fi
    else
        log "All prerequisites are satisfied."
    fi

    if ! command -v systemctl &> /dev/null; then
        error "systemd is required but not found."
    fi
}

enable_ip_forwarding() {
    log "Configuring IP forwarding..."
    cat > "/etc/sysctl.d/99-vwireguard.conf" <<EOF
net.ipv4.ip_forward=1
net.ipv6.conf.all.forwarding=1
EOF
    sysctl -p "/etc/sysctl.d/99-vwireguard.conf" > /dev/null 2>&1
}

detect_arch() {
    local arch=$(uname -m)
    case $arch in
        x86_64) echo "amd64" ;;
        aarch64|arm64) echo "arm64" ;;
        *) error "Unsupported architecture: $arch" ;;
    esac
}

get_latest_release() {
    log "Fetching latest release version..."
    local api_url="https://api.github.com/repos/${REPO_OWNER}/${REPO_NAME}/releases/latest"
    local tag=$(curl -sL "$api_url" | sed -n 's/.*"tag_name": *"\([^"]*\)".*/\1/p')
    
    if [ -z "$tag" ]; then
        tag=$(curl -sL -o /dev/null -w %{url_effective} "https://github.com/${REPO_OWNER}/${REPO_NAME}/releases/latest" | rev | cut -d/ -f1 | rev)
    fi

    if [ -z "$tag" ] || [ "$tag" == "releases" ]; then
        error "Failed to fetch latest release tag from GitHub."
    fi
    
    echo "$tag"
}

download_release() {
    local tag=$1
    local arch=$2
    local asset_name="vWireguard-linux-${arch}.tar.gz"
    local download_url="https://github.com/${REPO_OWNER}/${REPO_NAME}/releases/download/${tag}/${asset_name}"
    local temp_file="/tmp/${asset_name}"
    
    log "Downloading ${asset_name} from release ${tag}..."
    
    if ! wget -q --show-progress -O "$temp_file" "$download_url"; then
        error "Failed to download release asset."
    fi
    
    if [ ! -f "$temp_file" ] || [ ! -s "$temp_file" ]; then
        error "Downloaded file is empty or missing"
    fi
    
    echo "$temp_file"
}

stop_service() {
    if systemctl is-active --quiet "$SERVICE_NAME" 2>/dev/null; then
        log "Stopping existing service..."
        systemctl stop "$SERVICE_NAME" || true
    fi
}

preserve_data() {
    if [ -d "$INSTALL_DIR/db" ]; then
        log "Backing up existing database..."
        local backup_dir="/tmp/vwireguard-db-backup-$(date +%Y%m%d_%H%M%S)"
        mkdir -p "$backup_dir"
        cp -r "$INSTALL_DIR/db" "$backup_dir/" 2>/dev/null || true
        if [ -f "$INSTALL_DIR/config.json" ]; then
             cp "$INSTALL_DIR/config.json" "$backup_dir/" 2>/dev/null || true
        fi
        echo "$backup_dir"
    fi
}

install_files() {
    local archive=$1
    local backup_dir=$2
    
    log "Extracting release package to ${INSTALL_DIR}..."
    mkdir -p "$INSTALL_DIR"
    
    tar -xzf "$archive" -C "$INSTALL_DIR" || error "Failed to extract archive"
    
    if [ -n "$backup_dir" ] && [ -d "$backup_dir/db" ]; then
        log "Restoring database..."
        cp -r "$backup_dir/db"/* "$INSTALL_DIR/db/" 2>/dev/null || true
        if [ -f "$backup_dir/config.json" ]; then
            cp "$backup_dir/config.json" "$INSTALL_DIR/" 2>/dev/null || true
        fi
        rm -rf "$backup_dir"
    fi
    
    chmod +x "$INSTALL_DIR/vWireguard"
    if [ -f "$INSTALL_DIR/vwg" ]; then
        chmod +x "$INSTALL_DIR/vwg"
    fi
    
    mkdir -p "$INSTALL_DIR/db/{clients,server,users,wake_on_lan_hosts,tunnels}"
}

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
    systemctl enable "$SERVICE_NAME" > /dev/null 2>&1
}

create_symlink() {
    if [ -f "$INSTALL_DIR/vwg" ]; then
        rm -f "/usr/bin/vwg"
        ln -s "$INSTALL_DIR/vwg" "/usr/bin/vwg"
        chmod +x "/usr/bin/vwg"
    fi
}

start_service() {
    log "Starting service..."
    systemctl start "$SERVICE_NAME"
    sleep 2
    if ! systemctl is-active --quiet "$SERVICE_NAME"; then
        warn "Service failed to start. Logs:"
        journalctl -u "$SERVICE_NAME" --no-pager -n 10
        error "Installation finished but service is not running."
    fi
}

show_summary() {
    local public_ip=$(curl -s --connect-timeout 5 ifconfig.me 2>/dev/null || echo "YOUR_IP")
    echo ""
    echo -e "\033[0;32m=====================================\033[0m"
    echo -e "\033[0;32m✅ Installation completed successfully!\033[0m"
    echo -e "\033[0;32m=====================================\033[0m"
    echo -e "\033[0;36mPanel URL:\033[0m http://${public_ip}:5000"
    echo -e "\033[0;36mDefault:\033[0m   admin / admin"
    echo -e "\033[0;36mCommand:\033[0m   vwg"
    echo ""
}

main() {
    check_and_install_prerequisites
    enable_ip_forwarding
    
    local arch=$(detect_arch)
    local tag=$(get_latest_release)
    
    log "Detected: $arch | Version: $tag"
    
    local archive=$(download_release "$tag" "$arch")
    
    stop_service
    local backup_dir=$(preserve_data)
    
    install_files "$archive" "$backup_dir"
    create_service
    create_symlink
    start_service
    
    rm -f "$archive"
    show_summary
}

main "$@"
