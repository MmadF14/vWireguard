#!/bin/bash

# vWireguard Panel - One Click Installation Script
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

echo -e "${CYAN}=== vWireguard Panel - Easy Installation ===${NC}"
echo ""

if [ "$EUID" -ne 0 ]; then 
    echo -e "${RED}❌ Please run with root access:${NC}"
    echo -e "${GREEN}sudo bash install.sh${NC}"
    exit 1
fi

VWIREGUARD_DIR="/opt/vwireguard"
DOMAIN=""
SSL_ENABLED=false
DEFAULT_INTERFACE=$(ip route | awk '/default/ {print $5}' | head -n 1)
PUBLIC_IP=$(curl -s ifconfig.me 2>/dev/null || curl -s icanhazip.com 2>/dev/null || echo "localhost")

log() { echo -e "${GREEN}[$(date '+%H:%M:%S')] $1${NC}"; }
error() { echo -e "${RED}[$(date '+%H:%M:%S')] ❌ $1${NC}"; }
warn() { echo -e "${YELLOW}[$(date '+%H:%M:%S')] ⚠️  $1${NC}"; }

detect_os() {
    if [ -f /etc/debian_version ]; then
        echo "debian"
    elif [ -f /etc/redhat-release ]; then
        echo "rhel"
    else
        echo "unknown"
    fi
}

install_packages() {
    log "Installing required packages..."
    
    local os=$(detect_os)
    case $os in
        "debian")
            apt-get update -y
            apt-get install -y curl wget git build-essential wireguard wireguard-tools ufw openssl
            ;;
        "rhel")
            yum update -y
            yum install -y curl wget git gcc make wireguard-tools firewalld openssl
            ;;
        *)
            error "Operating system not supported"
            exit 1
            ;;
    esac
}

install_go() {
    if command -v go >/dev/null 2>&1; then
        log "Go is already installed"
        return 0
    fi
    
    log "Installing Go..."
    local go_version="go1.21.5"
    local arch=$(uname -m | sed 's/x86_64/amd64/; s/aarch64/arm64/')
    
    wget -q "https://go.dev/dl/${go_version}.linux-${arch}.tar.gz" -O /tmp/go.tar.gz
    rm -rf /usr/local/go
    tar -C /usr/local -xzf /tmp/go.tar.gz
    export PATH=$PATH:/usr/local/go/bin
    echo 'export PATH=$PATH:/usr/local/go/bin' >> /etc/profile
}

ask_ssl() {
    echo ""
    echo -e "${CYAN}=== SSL Configuration ===${NC}"
    read -p "Do you have a domain and want to enable SSL? (y/n): " -n 1 -r
    echo ""
    
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        read -p "Enter your domain name (example: vpn.example.com): " DOMAIN
        if [ -n "$DOMAIN" ]; then
            SSL_ENABLED=true
            log "SSL will be enabled for domain $DOMAIN"
        else
            warn "No domain entered, SSL will remain disabled"
        fi
    else
        log "SSL will be disabled"
    fi
}

setup_vwireguard() {
    log "Downloading and installing vWireguard..."
    mkdir -p $VWIREGUARD_DIR
    cd $VWIREGUARD_DIR
    
    local arch=$(uname -m | sed 's/x86_64/amd64/; s/aarch64/arm64/')
    local release_url="https://github.com/MmadF14/vwireguard/releases/latest/download/vwireguard-linux-${arch}.tar.gz"
    
    if wget -q --spider "$release_url" 2>/dev/null; then
        log "Downloading pre-built version..."
        wget -q "$release_url" -O vwireguard.tar.gz
        tar -xzf vwireguard.tar.gz
        rm vwireguard.tar.gz
    else
        log "Building from source..."
        git clone https://github.com/MmadF14/vwireguard.git temp
        mv temp/* .
        rm -rf temp
        
        export PATH=$PATH:/usr/local/go/bin
        go mod tidy
        go build -ldflags="-s -w" -o vwireguard
    fi
    
    chmod +x vwireguard
    mkdir -p db/{clients,server,users,wake_on_lan_hosts,tunnels}
    mkdir -p ssl
}

setup_ssl_certs() {
    if [ "$SSL_ENABLED" = true ]; then
        log "Setting up SSL certificates..."
        cd $VWIREGUARD_DIR/ssl
        
        # Generate self-signed certificate if user doesn't have custom certificates
        echo ""
        echo -e "${YELLOW}SSL Certificate Options:${NC}"
        echo "1. Generate self-signed certificate (quick start)"
        echo "2. Use existing certificate files"
        read -p "Choose option (1 or 2): " -n 1 -r
        echo ""
        
        if [[ $REPLY == "1" ]]; then
            log "Generating self-signed certificate..."
            openssl req -x509 -nodes -days 365 -newkey rsa:2048 \
                -keyout server.key \
                -out server.crt \
                -subj "/C=US/ST=State/L=City/O=Organization/CN=$DOMAIN"
            log "Self-signed certificate generated"
        else
            log "Please place your certificate files in $VWIREGUARD_DIR/ssl/"
            log "Required files: server.key (private key) and server.crt (certificate)"
            echo ""
            read -p "Press Enter after placing your certificate files..."
            
            if [ ! -f "server.key" ] || [ ! -f "server.crt" ]; then
                warn "Certificate files not found, generating self-signed certificate..."
                openssl req -x509 -nodes -days 365 -newkey rsa:2048 \
                    -keyout server.key \
                    -out server.crt \
                    -subj "/C=US/ST=State/L=City/O=Organization/CN=$DOMAIN"
            fi
        fi
        
        chmod 600 server.key
        chmod 644 server.crt
    fi
}

setup_wireguard() {
    log "Setting up WireGuard..."
    
    mkdir -p /etc/wireguard
    cd /etc/wireguard
    wg genkey | tee server_private.key | wg pubkey > server_public.key
    chmod 600 server_private.key
    
    cat > wg0.conf <<EOF
[Interface]
PrivateKey = $(cat server_private.key)
Address = 10.252.1.1/24
ListenPort = 51820
PostUp = iptables -A FORWARD -i wg0 -j ACCEPT; iptables -t nat -A POSTROUTING -o ${DEFAULT_INTERFACE} -j MASQUERADE
PostDown = iptables -D FORWARD -i wg0 -j ACCEPT; iptables -t nat -D POSTROUTING -o ${DEFAULT_INTERFACE} -j MASQUERADE

# Client configurations will be added here
EOF

    echo 'net.ipv4.ip_forward=1' > /etc/sysctl.d/99-wireguard.conf
    sysctl -p /etc/sysctl.d/99-wireguard.conf
}

create_service() {
    log "Creating systemd service..."
    
    cat > /etc/systemd/system/vwireguard.service <<EOF
[Unit]
Description=vWireguard Panel
After=network.target

[Service]
Type=simple
WorkingDirectory=$VWIREGUARD_DIR
ExecStart=$VWIREGUARD_DIR/vwireguard
Restart=always
RestartSec=3
User=root
Environment=PORT=5000

[Install]
WantedBy=multi-user.target
EOF
    
    systemctl daemon-reload
    systemctl enable vwireguard
    systemctl enable wg-quick@wg0
}

setup_firewall() {
    log "Setting up firewall..."
    
    if command -v ufw >/dev/null 2>&1; then
        ufw --force enable
        ufw allow ssh
        ufw allow 5000/tcp
        ufw allow 51820/udp
    elif command -v firewall-cmd >/dev/null 2>&1; then
        systemctl start firewalld
        systemctl enable firewalld
        firewall-cmd --permanent --add-service=ssh
        firewall-cmd --permanent --add-port=5000/tcp
        firewall-cmd --permanent --add-port=51820/udp
        firewall-cmd --reload
    fi
}

start_services() {
    log "Starting services..."
    
    systemctl start wg-quick@wg0
    systemctl start vwireguard
    
    sleep 3
    
    if ! systemctl is-active --quiet vwireguard; then
        error "Failed to start vWireguard"
        journalctl -u vwireguard --no-pager -n 10
        exit 1
    fi
}

show_summary() {
    local panel_url
    local protocol="http"
    local port="5000"
    
    if [ "$SSL_ENABLED" = true ]; then
        protocol="https"
        panel_url="${protocol}://$DOMAIN:$port"
    else
        panel_url="${protocol}://$PUBLIC_IP:$port"
    fi
    
    echo ""
    echo -e "${GREEN}=====================================${NC}"
    echo -e "${GREEN}✅ Installation completed successfully!${NC}"
    echo -e "${GREEN}=====================================${NC}"
    echo ""
    echo -e "${CYAN}📋 Panel Information:${NC}"
    echo -e "  ${YELLOW}URL:${NC} $panel_url"
    echo -e "  ${YELLOW}Username:${NC} admin"
    echo -e "  ${YELLOW}Password:${NC} admin"
    echo ""
    echo -e "${CYAN}🔧 WireGuard:${NC}"
    echo -e "  ${YELLOW}Port:${NC} 51820"
    echo -e "  ${YELLOW}Config:${NC} /etc/wireguard/wg0.conf"
    echo ""
    if [ "$SSL_ENABLED" = true ]; then
        echo -e "${CYAN}🔒 SSL Configuration:${NC}"
        echo -e "  ${YELLOW}Certificate:${NC} $VWIREGUARD_DIR/ssl/server.crt"
        echo -e "  ${YELLOW}Private Key:${NC} $VWIREGUARD_DIR/ssl/server.key"
        echo ""
    fi
    echo -e "${CYAN}⚙️ Useful Commands:${NC}"
    echo -e "  ${YELLOW}Status:${NC} systemctl status vwireguard"
    echo -e "  ${YELLOW}Logs:${NC} journalctl -u vwireguard -f"
    echo -e "  ${YELLOW}Restart:${NC} systemctl restart vwireguard"
    echo -e "  ${YELLOW}Update:${NC} bash install.sh update"
    echo ""
    echo -e "${GREEN}🎉 Panel is ready!${NC}"
    echo -e "${GREEN}=====================================${NC}"
    
    cat > /root/vwireguard-info.txt <<EOF
vWireguard Panel Information
===========================
Panel URL: $panel_url
Username: admin
Password: admin
WireGuard Port: 51820

Installation Directory: $VWIREGUARD_DIR
Config File: /etc/wireguard/wg0.conf
Service Status: systemctl status vwireguard

Update Command: bash install.sh update
EOF
}

# Update function
update_vwireguard() {
    log "Starting vWireguard update..."
    
    if [ ! -d "$VWIREGUARD_DIR" ] || [ ! -f "$VWIREGUARD_DIR/vwireguard" ]; then
        error "vWireguard is not installed. Please run script without parameters to install."
        exit 1
    fi
    
    # Create backup
    local backup_dir="/opt/vwireguard-backup-$(date +%Y%m%d_%H%M%S)"
    mkdir -p "$backup_dir"
    cp -r "$VWIREGUARD_DIR/db" "$backup_dir/" 2>/dev/null
    cp -r "$VWIREGUARD_DIR/ssl" "$backup_dir/" 2>/dev/null
    cp "$VWIREGUARD_DIR/vwireguard" "$backup_dir/vwireguard.old" 2>/dev/null
    log "Backup created at $backup_dir"
    
    # Stop service
    systemctl stop vwireguard
    
    # Update binary
    cd $VWIREGUARD_DIR
    local arch=$(uname -m | sed 's/x86_64/amd64/; s/aarch64/arm64/')
    local release_url="https://github.com/MmadF14/vwireguard/releases/latest/download/vwireguard-linux-${arch}.tar.gz"
    
    if wget -q --spider "$release_url" 2>/dev/null; then
        log "Downloading new version..."
        wget -q "$release_url" -O vwireguard-new.tar.gz
        mkdir -p temp_extract
        tar -xzf vwireguard-new.tar.gz -C temp_extract
        
        # Backup old binary and replace
        mv vwireguard vwireguard.old
        mv temp_extract/vwireguard ./vwireguard
        chmod +x vwireguard
        
        # Update templates and static files if they exist
        if [ -d "temp_extract/templates" ]; then
            cp -r temp_extract/templates ./
        fi
        if [ -d "temp_extract/static" ]; then
            cp -r temp_extract/static ./
        fi
        if [ -d "temp_extract/custom" ]; then
            cp -r temp_extract/custom ./
        fi
        
        rm -rf temp_extract vwireguard-new.tar.gz
    else
        log "Building from source..."
        install_go
        git clone https://github.com/MmadF14/vwireguard.git temp_build
        cd temp_build
        export PATH=$PATH:/usr/local/go/bin
        go mod tidy
        go build -ldflags="-s -w" -o ../vwireguard-new
        cd ..
        rm -rf temp_build
        
        mv vwireguard vwireguard.old
        mv vwireguard-new vwireguard
        chmod +x vwireguard
    fi
    
    # Start service
    systemctl start vwireguard
    
    if systemctl is-active --quiet vwireguard; then
        log "✅ Update completed successfully!"
        log "Old backup stored at: $backup_dir"
        log "Panel should be accessible shortly..."
    else
        error "Failed to start - restoring from backup..."
        cp "$backup_dir/vwireguard.old" ./vwireguard
        systemctl start vwireguard
        if systemctl is-active --quiet vwireguard; then
            log "Service restored from backup"
        else
            error "Failed to restore service. Please check logs: journalctl -u vwireguard"
        fi
    fi
}

# Main installation function
install_vwireguard() {
    log "Starting vWireguard installation..."
    
    install_packages
    install_go
    ask_ssl
    setup_vwireguard
    setup_ssl_certs
    setup_wireguard
    create_service
    setup_firewall
    start_services
    show_summary
}

# Main function
main() {
    case "${1:-}" in
        update|--update|-u)
            update_vwireguard
            ;;
        *)
            install_vwireguard
            ;;
    esac
}

main "$@" 