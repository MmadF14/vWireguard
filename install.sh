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
echo -e "${CYAN}=== vWireguard Panel - نصب آسان ===${NC}"
echo ""
if [ "$EUID" -ne 0 ]; then 
    echo -e "${RED}❌ لطفاً با دسترسی root اجرا کنید:${NC}"
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
    log "نصب پکیج‌های مورد نیاز..."
    
    local os=$(detect_os)
    case $os in
        "debian")
            apt-get update -y
            apt-get install -y curl wget git build-essential wireguard wireguard-tools ufw nginx certbot python3-certbot-nginx
            ;;
        "rhel")
            yum update -y
            yum install -y curl wget git gcc make wireguard-tools firewalld nginx certbot python3-certbot-nginx
            ;;
        *)
            error "سیستم عامل پشتیبانی نمی‌شود"
            exit 1
            ;;
    esac
}
install_go() {
    if command -v go >/dev/null 2>&1; then
        log "Go قبلاً نصب شده است"
        return 0
    fi
    
    log "نصب Go..."
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
    echo -e "${CYAN}=== تنظیمات SSL ===${NC}"
    read -p "آیا دامنه دارید و می‌خواهید SSL فعال شود؟ (y/n): " -n 1 -r
    echo ""
    
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        read -p "نام دامنه خود را وارد کنید (مثال: vpn.example.com): " DOMAIN
        if [ -n "$DOMAIN" ]; then
            SSL_ENABLED=true
            log "SSL برای دامنه $DOMAIN فعال خواهد شد"
        else
            warn "دامنه وارد نشد، SSL غیرفعال می‌ماند"
        fi
    else
        log "SSL غیرفعال خواهد بود"
    fi
}
setup_vwireguard() {
    log "دانلود و نصب vWireguard..."
    mkdir -p $VWIREGUARD_DIR
    cd $VWIREGUARD_DIR
    local arch=$(uname -m | sed 's/x86_64/amd64/; s/aarch64/arm64/')
    local release_url="https://github.com/MmadF14/vwireguard/releases/latest/download/vwireguard-linux-${arch}.tar.gz"
    
    if wget -q --spider "$release_url" 2>/dev/null; then
        log "دانلود نسخه آماده..."
        wget -q "$release_url" -O vwireguard.tar.gz
        tar -xzf vwireguard.tar.gz
        rm vwireguard.tar.gz
    else
        log "ساخت از کد منبع..."
        git clone https://github.com/MmadF14/vwireguard.git temp
        mv temp/* .
        rm -rf temp
        
        export PATH=$PATH:/usr/local/go/bin
        go mod tidy
        go build -ldflags="-s -w" -o vwireguard
    fi
    
    chmod +x vwireguard
    mkdir -p db/{clients,server,users,wake_on_lan_hosts,tunnels}
}
setup_wireguard() {
    log "تنظیم WireGuard..."
    
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
setup_nginx() {
    log "تنظیم Nginx..."
    
    if [ "$SSL_ENABLED" = true ]; then
        cat > /etc/nginx/sites-available/vwireguard <<EOF
server {
    listen 80;
    server_name $DOMAIN;
    return 301 https://\$server_name\$request_uri;
}

server {
    listen 443 ssl http2;
    server_name $DOMAIN;
    
    location / {
        proxy_pass http://127.0.0.1:5000;
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;
    }
}
 EOF
    else
        cat > /etc/nginx/sites-available/vwireguard <<EOF
server {
    listen 80;
    server_name _;
    
    location / {
        proxy_pass http://127.0.0.1:5000;
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;
    }
}
EOF
    fi
    
    ln -sf /etc/nginx/sites-available/vwireguard /etc/nginx/sites-enabled/
    rm -f /etc/nginx/sites-enabled/default
    nginx -t && systemctl reload nginx
}
setup_ssl() {
    if [ "$SSL_ENABLED" = true ]; then
        log "دریافت گواهی SSL..."
        certbot --nginx -d "$DOMAIN" --non-interactive --agree-tos --email admin@"$DOMAIN" --redirect
        echo "0 12 * * * /usr/bin/certbot renew --quiet" | crontab -
    fi
}
create_service() {
    log "ایجاد سرویس systemd..."
    
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

[Install]
WantedBy=multi-user.target
EOF
    
    systemctl daemon-reload
    systemctl enable vwireguard
    systemctl enable wg-quick@wg0
}
setup_firewall() {
    log "تنظیم فایروال..."
    
    if command -v ufw >/dev/null 2>&1; then
        ufw --force enable
        ufw allow ssh
        ufw allow 80/tcp
        ufw allow 443/tcp
        ufw allow 51820/udp
    elif command -v firewall-cmd >/dev/null 2>&1; then
        systemctl start firewalld
        systemctl enable firewalld
        firewall-cmd --permanent --add-service=ssh
        firewall-cmd --permanent --add-service=http
        firewall-cmd --permanent --add-service=https
        firewall-cmd --permanent --add-port=51820/udp
        firewall-cmd --reload
    fi
}
start_services() {
    log "راه‌اندازی سرویس‌ها..."
    
    systemctl start nginx
    systemctl start wg-quick@wg0
    systemctl start vwireguard
    
    sleep 3
    
    if ! systemctl is-active --quiet vwireguard; then
        error "خطا در راه‌اندازی vWireguard"
        journalctl -u vwireguard --no-pager -n 10
        exit 1
    fi
}
show_summary() {
    local panel_url
    if [ "$SSL_ENABLED" = true ]; then
        panel_url="https://$DOMAIN"
    else
        panel_url="http://$PUBLIC_IP"
    fi
    
    echo ""
    echo -e "${GREEN}=====================================${NC}"
    echo -e "${GREEN}✅ نصب با موفقیت تکمیل شد!${NC}"
    echo -e "${GREEN}=====================================${NC}"
    echo ""
    echo -e "${CYAN}📋 اطلاعات پنل:${NC}"
    echo -e "  ${YELLOW}آدرس:${NC} $panel_url"
    echo -e "  ${YELLOW}نام کاربری:${NC} admin"
    echo -e "  ${YELLOW}رمز عبور:${NC} admin"
    echo ""
    echo -e "${CYAN}🔧 WireGuard:${NC}"
    echo -e "  ${YELLOW}پورت:${NC} 51820"
    echo -e "  ${YELLOW}تنظیمات:${NC} /etc/wireguard/wg0.conf"
    echo ""
    echo -e "${CYAN}⚙️ دستورات مفید:${NC}"
    echo -e "  ${YELLOW}وضعیت:${NC} systemctl status vwireguard"
    echo -e "  ${YELLOW}لاگ‌ها:${NC} journalctl -u vwireguard -f"
    echo -e "  ${YELLOW}ری‌استارت:${NC} systemctl restart vwireguard"
    echo ""
    echo -e "${GREEN}🎉 پنل آماده است!${NC}"
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
EOF
}
main() {
    log "شروع نصب vWireguard..."
    
    install_packages
    install_go
    ask_ssl
    setup_vwireguard
    setup_wireguard
    setup_nginx
    setup_ssl
    create_service
    setup_firewall
    start_services
    show_summary
}
main "$@" 