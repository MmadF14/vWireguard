#!/bin/bash

# vWireguard Panel - One Click Installation Script
# این اسکریپت برای نصب کامل پنل vWireguard با یک کلیک طراحی شده است

# Text colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Print banner
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
echo -e "${CYAN}=== vWireguard Panel - One Click Installation ===${NC}"
echo -e "${YELLOW}این اسکریپت پنل vWireguard را به صورت کامل نصب می‌کند${NC}"
echo ""

# Check if running as root
if [ "$EUID" -ne 0 ]; then 
    echo -e "${RED}❌ این اسکریپت باید با دسترسی root اجرا شود${NC}"
    echo -e "${YELLOW}لطفاً دستور زیر را اجرا کنید:${NC}"
    echo -e "${GREEN}sudo bash one_click_install.sh${NC}"
    exit 1
fi

# Function to log messages
log_message() {
    echo -e "${GREEN}[$(date '+%Y-%m-%d %H:%M:%S')] $1${NC}"
}

# Function to log errors
log_error() {
    echo -e "${RED}[$(date '+%Y-%m-%d %H:%M:%S')] ERROR: $1${NC}"
}

# Function to log warnings
log_warning() {
    echo -e "${YELLOW}[$(date '+%Y-%m-%d %H:%M:%S')] WARNING: $1${NC}"
}

# Function to check if command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Function to detect OS
detect_os() {
    if [ -f /etc/os-release ]; then
        . /etc/os-release
        OS=$NAME
        VER=$VERSION_ID
    elif type lsb_release >/dev/null 2>&1; then
        OS=$(lsb_release -si)
        VER=$(lsb_release -sr)
    elif [ -f /etc/lsb-release ]; then
        . /etc/lsb-release
        OS=$DISTRIB_ID
        VER=$DISTRIB_RELEASE
    elif [ -f /etc/debian_version ]; then
        OS=Debian
        VER=$(cat /etc/debian_version)
    elif [ -f /etc/SuSe-release ]; then
        OS=SuSE
    elif [ -f /etc/redhat-release ]; then
        OS=RedHat
    else
        OS=$(uname -s)
        VER=$(uname -r)
    fi
    echo "$OS"
}

# Function to install dependencies based on OS
install_dependencies() {
    local os=$(detect_os)
    log_message "تشخیص سیستم عامل: $os"
    
    case "$os" in
        *"Ubuntu"*|*"Debian"*)
            log_message "نصب پکیج‌های مورد نیاز برای Ubuntu/Debian..."
            apt-get update
            apt-get install -y curl wget git build-essential ufw wireguard wireguard-tools
            ;;
        *"CentOS"*|*"Red Hat"*|*"Fedora"*)
            log_message "نصب پکیج‌های مورد نیاز برای CentOS/RHEL/Fedora..."
            if command_exists dnf; then
                dnf update -y
                dnf install -y curl wget git gcc make ufw wireguard-tools
            else
                yum update -y
                yum install -y curl wget git gcc make ufw wireguard-tools
            fi
            ;;
        *)
            log_warning "سیستم عامل شناسایی نشد. تلاش برای نصب عمومی..."
            if command_exists apt-get; then
                apt-get update && apt-get install -y curl wget git build-essential ufw wireguard wireguard-tools
            elif command_exists yum; then
                yum update -y && yum install -y curl wget git gcc make ufw wireguard-tools
            elif command_exists dnf; then
                dnf update -y && dnf install -y curl wget git gcc make ufw wireguard-tools
            else
                log_error "نمی‌توان پکیج‌های مورد نیاز را نصب کرد"
                exit 1
            fi
            ;;
    esac
}

# Function to install Go
install_go() {
    log_message "نصب Go..."
    
    # Check if Go is already installed
    if command_exists go; then
        local go_version=$(go version | awk '{print $3}')
        log_message "Go در حال حاضر نصب است: $go_version"
        return 0
    fi
    
    # Download and install Go
    local go_version=$(curl -s https://go.dev/VERSION?m=text | head -n 1)
    local arch=$(uname -m)
    
    case "$arch" in
        x86_64) GOARCH=amd64 ;;
        aarch64|arm64) GOARCH=arm64 ;;
        armv7l|armv6l) GOARCH=arm ;;
        i386|i686) GOARCH=386 ;;
        *) GOARCH=amd64 ;;
    esac
    
    local go_tar="${go_version}.linux-${GOARCH}.tar.gz"
    local go_url="https://go.dev/dl/${go_tar}"
    
    log_message "دانلود Go $go_version..."
    if wget -qO /tmp/go.tar.gz "$go_url"; then
        rm -rf /usr/local/go
        tar -C /usr/local -xzf /tmp/go.tar.gz
        echo 'export PATH=$PATH:/usr/local/go/bin' >> /etc/profile
        echo 'export PATH=$PATH:/usr/local/go/bin' >> /root/.bashrc
        export PATH=$PATH:/usr/local/go/bin
        
        if go version >/dev/null 2>&1; then
            log_message "Go با موفقیت نصب شد"
            return 0
        else
            log_error "نصب Go ناموفق بود"
            return 1
        fi
    else
        log_error "دانلود Go ناموفق بود"
        return 1
    fi
}

# Function to install Node.js and Yarn
install_nodejs() {
    log_message "نصب Node.js و Yarn..."
    
    # Check if Node.js is already installed
    if command_exists node; then
        local node_version=$(node --version)
        log_message "Node.js در حال حاضر نصب است: $node_version"
    else
        # Install Node.js
        curl -fsSL https://deb.nodesource.com/setup_lts.x | bash -
        apt-get install -y nodejs
    fi
    
    # Install Yarn
    if ! command_exists yarn; then
        npm install -g yarn
    fi
    
    log_message "Node.js و Yarn نصب شدند"
}

# Function to download and build vWireguard
setup_vwireguard() {
    log_message "تنظیم vWireguard..."
    
    # Create installation directory
    mkdir -p /opt/vwireguard
    cd /opt/vwireguard
    
    # Try to download latest release first
    log_message "تلاش برای دانلود آخرین نسخه..."
    local arch=$(uname -m)
    case "$arch" in
        x86_64) GOARCH=amd64 ;;
        aarch64|arm64) GOARCH=arm64 ;;
        armv7l|armv6l) GOARCH=arm ;;
        i386|i686) GOARCH=386 ;;
        *) GOARCH=amd64 ;;
    esac
    
    local release_url=$(curl -s https://api.github.com/repos/MmadF14/vwireguard/releases/latest \
        | grep browser_download_url \
        | grep "linux" \
        | grep "${GOARCH}" \
        | head -n 1 \
        | cut -d '"' -f 4)
    
    if [ -n "$release_url" ]; then
        log_message "دانلود نسخه آماده..."
        if wget -qO /tmp/vwireguard.tar.gz "$release_url"; then
            tar -xzf /tmp/vwireguard.tar.gz -C /opt/vwireguard
            chmod +x /opt/vwireguard/vwireguard
            log_message "نسخه آماده با موفقیت دانلود شد"
            return 0
        fi
    fi
    
    # If release download failed, build from source
    log_message "ساخت از کد منبع..."
    rm -rf /opt/vwireguard/*
    git clone https://github.com/MmadF14/vwireguard.git /tmp/vwireguard_src
    cp -r /tmp/vwireguard_src/* /opt/vwireguard/
    rm -rf /tmp/vwireguard_src
    
    # Create database directories
    mkdir -p /opt/vwireguard/db/{clients,server,users,wake_on_lan_hosts}
    
    # Prepare assets
    cd /opt/vwireguard
    if [ -f "prepare_assets.sh" ]; then
        chmod +x prepare_assets.sh
        ./prepare_assets.sh
    elif [ -f "prepare_assets" ]; then
        chmod +x prepare_assets
        ./prepare_assets
    else
        log_warning "فایل prepare_assets یافت نشد"
    fi
    
    # Build the application
    export GOPATH=/go
    export PATH=$PATH:/usr/local/go/bin:$GOPATH/bin
    go mod tidy
    go build -ldflags="-s -w" -o vwireguard
    
    if [ ! -f "vwireguard" ]; then
        log_error "ساخت برنامه ناموفق بود"
        return 1
    fi
    
    chmod +x vwireguard
    log_message "vWireguard با موفقیت ساخته شد"
}

# Function to configure WireGuard
configure_wireguard() {
    log_message "تنظیم WireGuard..."
    
    # Create WireGuard directory
    mkdir -p /etc/wireguard
    
    # Generate server keys
    wg genkey | tee /etc/wireguard/server_private.key | wg pubkey > /etc/wireguard/server_public.key
    chmod 600 /etc/wireguard/server_private.key
    
    # Detect default network interface
    local default_interface=$(ip route | awk '/default/ {print $5}' | head -n 1)
    if [ -z "$default_interface" ]; then
        default_interface="eth0"
    fi
    
    log_message "رابط شبکه پیش‌فرض: $default_interface"
    
    # Create WireGuard configuration
    local server_private_key=$(cat /etc/wireguard/server_private.key)
    cat > /etc/wireguard/wg0.conf <<EOL
[Interface]
PrivateKey = ${server_private_key}
Address = 10.252.1.1/24
ListenPort = 51820
PostUp = iptables -A FORWARD -i wg0 -j ACCEPT; iptables -t nat -A POSTROUTING -o ${default_interface} -j MASQUERADE
PostDown = iptables -D FORWARD -i wg0 -j ACCEPT; iptables -t nat -D POSTROUTING -o ${default_interface} -j MASQUERADE

# Client configurations will be added here
EOL
    
    # Enable IP forwarding
    echo "net.ipv4.ip_forward=1" > /etc/sysctl.d/99-wireguard.conf
    sysctl -p /etc/sysctl.d/99-wireguard.conf
    
    log_message "WireGuard تنظیم شد"
}

# Function to create systemd service
create_service() {
    log_message "ایجاد سرویس systemd..."
    
    cat > /etc/systemd/system/vwireguard.service <<EOF
[Unit]
Description=vWireguard Web Interface
After=network.target

[Service]
Type=simple
WorkingDirectory=/opt/vwireguard
ExecStart=/opt/vwireguard/vwireguard
Restart=always
RestartSec=3
User=root
Group=root

[Install]
WantedBy=multi-user.target
EOF
    
    systemctl daemon-reload
    systemctl enable vwireguard
    systemctl enable wg-quick@wg0
    
    log_message "سرویس systemd ایجاد شد"
}

# Function to configure firewall
configure_firewall() {
    log_message "تنظیم فایروال..."
    
    # Configure UFW
    if command_exists ufw; then
        ufw --force enable
        ufw allow ssh
        ufw allow 5000/tcp  # vWireguard panel
        ufw allow 51820/udp # WireGuard
        ufw allow 80/tcp    # HTTP (for SSL)
        ufw allow 443/tcp   # HTTPS
        log_message "فایروال UFW تنظیم شد"
    elif command_exists firewall-cmd; then
        firewall-cmd --permanent --add-service=ssh
        firewall-cmd --permanent --add-port=5000/tcp
        firewall-cmd --permanent --add-port=51820/udp
        firewall-cmd --permanent --add-port=80/tcp
        firewall-cmd --permanent --add-port=443/tcp
        firewall-cmd --reload
        log_message "فایروال firewalld تنظیم شد"
    else
        log_warning "فایروال یافت نشد"
    fi
}

# Function to start services
start_services() {
    log_message "راه‌اندازی سرویس‌ها..."
    
    # Start WireGuard
    systemctl start wg-quick@wg0
    if ! systemctl is-active --quiet wg-quick@wg0; then
        log_error "راه‌اندازی WireGuard ناموفق بود"
        return 1
    fi
    
    # Start vWireguard panel
    systemctl start vwireguard
    sleep 3
    
    if ! systemctl is-active --quiet vwireguard; then
        log_error "راه‌اندازی پنل vWireguard ناموفق بود"
        journalctl -u vwireguard --no-pager -n 10
        return 1
    fi
    
    log_message "سرویس‌ها با موفقیت راه‌اندازی شدند"
}

# Function to create credentials file
create_credentials() {
    local username="admin"
    local password="admin"
    
    cat > /root/vwireguard_credentials.txt <<EOF
=== vWireguard Panel Credentials ===
Username: ${username}
Password: ${password}
Panel URL: http://$(curl -s ifconfig.me):5000
WireGuard Port: 51820

=== دسترسی‌های پنل vWireguard ===
نام کاربری: ${username}
رمز عبور: ${password}
آدرس پنل: http://$(curl -s ifconfig.me):5000
پورت WireGuard: 51820
EOF
    
    log_message "فایل اطلاعات ورود در /root/vwireguard_credentials.txt ذخیره شد"
}

# Function to show installation summary
show_summary() {
    echo ""
    echo -e "${GREEN}=======================================================${NC}"
    echo -e "${GREEN}✅ نصب vWireguard با موفقیت تکمیل شد!${NC}"
    echo -e "${GREEN}=======================================================${NC}"
    echo ""
    echo -e "${CYAN}📋 اطلاعات نصب:${NC}"
    echo -e "  ${YELLOW}نام کاربری:${NC} admin"
    echo -e "  ${YELLOW}رمز عبور:${NC} admin"
    echo -e "  ${YELLOW}آدرس پنل:${NC} http://$(curl -s ifconfig.me):5000"
    echo -e "  ${YELLOW}پورت WireGuard:${NC} 51820"
    echo ""
    echo -e "${CYAN}📁 فایل‌های مهم:${NC}"
    echo -e "  ${YELLOW}پوشه نصب:${NC} /opt/vwireguard"
    echo -e "  ${YELLOW}تنظیمات WireGuard:${NC} /etc/wireguard/wg0.conf"
    echo -e "  ${YELLOW}اطلاعات ورود:${NC} /root/vwireguard_credentials.txt"
    echo ""
    echo -e "${CYAN}🔧 دستورات مفید:${NC}"
    echo -e "  ${YELLOW}مشاهده وضعیت:${NC} systemctl status vwireguard"
    echo -e "  ${YELLOW}مشاهده لاگ‌ها:${NC} journalctl -u vwireguard -f"
    echo -e "  ${YELLOW}راه‌اندازی مجدد:${NC} systemctl restart vwireguard"
    echo -e "  ${YELLOW}توقف سرویس:${NC} systemctl stop vwireguard"
    echo ""
    echo -e "${GREEN}=======================================================${NC}"
    echo -e "${GREEN}🎉 پنل vWireguard آماده استفاده است!${NC}"
    echo -e "${GREEN}=======================================================${NC}"
}

# Main installation function
main() {
    log_message "شروع نصب vWireguard..."
    
    # Update system
    log_message "به‌روزرسانی سیستم..."
    apt-get update && apt-get upgrade -y
    
    # Install dependencies
    install_dependencies
    
    # Install Go
    if ! install_go; then
        log_error "نصب Go ناموفق بود"
        exit 1
    fi
    
    # Install Node.js and Yarn
    install_nodejs
    
    # Setup vWireguard
    if ! setup_vwireguard; then
        log_error "تنظیم vWireguard ناموفق بود"
        exit 1
    fi
    
    # Configure WireGuard
    configure_wireguard
    
    # Create systemd service
    create_service
    
    # Configure firewall
    configure_firewall
    
    # Start services
    if ! start_services; then
        log_error "راه‌اندازی سرویس‌ها ناموفق بود"
        exit 1
    fi
    
    # Create credentials file
    create_credentials
    
    # Show summary
    show_summary
}

# Run main function
main "$@" 