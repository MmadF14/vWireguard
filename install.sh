#!/bin/bash

# Text colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
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

# Check if running as root
if [ "$EUID" -ne 0 ]; then 
    echo -e "${RED}Please run as root${NC}"
    exit 1
fi

# Update system
echo -e "${YELLOW}Updating system...${NC}"
apt-get update
apt-get upgrade -y

# Install required packages
echo -e "${YELLOW}Installing required packages...${NC}"
apt-get install -y wireguard wireguard-tools git curl wget build-essential ufw

# Install latest Go version
# نصب آخرین نسخه Go
echo -e "${YELLOW}Installing latest Go version...${NC}"

# دریافت نسخه آخر Go
GO_VERSION=$(curl -s https://go.dev/VERSION?m=text | head -n 1)
GO_TAR="${GO_VERSION}.linux-amd64.tar.gz"

# دانلود و نصب Go
wget "https://go.dev/dl/${GO_TAR}" -O /tmp/go.tar.gz
rm -rf /usr/local/go && tar -C /usr/local -xzf /tmp/go.tar.gz

# تنظیم متغیر PATH برای Go
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
echo 'export PATH=$PATH:/usr/local/go/bin' >> /etc/profile
export PATH=$PATH:/usr/local/go/bin

# بررسی نصب
if go version; then
    echo -e "${GREEN}Go installed successfully!${NC}"
else
    echo -e "${RED}Failed to install Go!${NC}"
    exit 1
fi


# Enable IP forwarding
echo -e "${YELLOW}Enabling IP forwarding...${NC}"
echo "net.ipv4.ip_forward=1" > /etc/sysctl.d/99-wireguard.conf
sysctl -p /etc/sysctl.d/99-wireguard.conf

# Create WireGuard configuration directory
echo -e "${YELLOW}Creating WireGuard configuration directory...${NC}"
mkdir -p /etc/wireguard

# Generate WireGuard server keys
echo -e "${YELLOW}Generating WireGuard server keys...${NC}"
wg genkey | tee /etc/wireguard/server_private.key | wg pubkey > /etc/wireguard/server_public.key
chmod 600 /etc/wireguard/server_private.key

# Detect default network interface
DEFAULT_INTERFACE=$(ip route | awk '/default/ {print $5}')
echo -e "${YELLOW}Detected default network interface: ${DEFAULT_INTERFACE}${NC}"

# Create WireGuard server configuration
echo -e "${YELLOW}Creating WireGuard server configuration...${NC}"
cat > /etc/wireguard/wg0.conf << EOL
[Interface]
PrivateKey = $(cat /etc/wireguard/server_private.key)
Address = 10.0.0.1/24
ListenPort = 51820
PostUp = iptables -A FORWARD -i wg0 -j ACCEPT; iptables -t nat -A POSTROUTING -o ${DEFAULT_INTERFACE} -j MASQUERADE
PostDown = iptables -D FORWARD -i wg0 -j ACCEPT; iptables -t nat -D POSTROUTING -o ${DEFAULT_INTERFACE} -j MASQUERADE

# Client configurations will be added here
EOL

# Setup firewall
echo -e "${YELLOW}Configuring firewall...${NC}"
ufw allow 5000/tcp
ufw allow 51820/udp
echo -e "y\n" | ufw enable

# Clone repository
echo -e "${YELLOW}Cloning vWireguard repository...${NC}"
rm -rf /opt/vwireguard
git clone https://github.com/MmadF14/vwireguard.git /opt/vwireguard

# Check and prepare assets
echo -e "${YELLOW}Preparing assets...${NC}"
cd /opt/vwireguard

# Try to find prepare_assets in common locations
ASSET_SCRIPT=$(find . -type f \( -name "prepare_assets" -o -name "prepare_assets.sh" \) | head -n 1)

if [ -n "$ASSET_SCRIPT" ]; then
    echo -e "${GREEN}Found asset script at: ${ASSET_SCRIPT}${NC}"
    chmod +x "$ASSET_SCRIPT"
    echo -e "${YELLOW}Executing asset preparation...${NC}"
    if "$ASSET_SCRIPT"; then
        echo -e "${GREEN}Assets prepared successfully!${NC}"
    else
        echo -e "${RED}Failed to execute asset script!${NC}"
        exit 1
    fi
else
    echo -e "${RED}No prepare_assets script found in repository!${NC}"
    echo -e "${YELLOW}Searching in all directories...${NC}"
    find . -type f -name "prepare_assets*"
    echo -e "${RED}Please ensure prepare_assets exists in the repository${NC}"
    exit 1
fi

# Build the application
echo -e "${YELLOW}Building vWireguard...${NC}"
export GOPATH=/go
export PATH=$PATH:/usr/local/go/bin:$GOPATH/bin
go mod download
go build -ldflags="-s -w" -o vwireguard

# Verify build
if [ ! -f "vwireguard" ]; then
    echo -e "${RED}Build failed! Check dependencies and try again.${NC}"
    exit 1
fi

# Create systemd service
echo -e "${YELLOW}Creating systemd service...${NC}"
cat > /etc/systemd/system/vwireguard.service << EOL
[Unit]
Description=vWireguard Web Interface
After=network.target

[Service]
Type=simple
User=root
WorkingDirectory=/opt/vwireguard
ExecStart=/opt/vwireguard/vwireguard
Restart=always
RestartSec=3

[Install]
WantedBy=multi-user.target
EOL

# Start services
echo -e "${YELLOW}Starting services...${NC}"
systemctl daemon-reload
systemctl enable wg-quick@wg0
systemctl start wg-quick@wg0

if ! systemctl is-active --quiet wg-quick@wg0; then
    echo -e "${RED}Failed to start WireGuard service!${NC}"
    exit 1
fi

systemctl enable vwireguard
systemctl start vwireguard

# Create default admin user
echo -e "${YELLOW}Creating default admin user...${NC}"
cat > /opt/vwireguard/config.json << EOL
{
    "users": [
        {
            "username": "admin",
            "password": "admin",
            "role": "admin"
        }
    ]
}
EOL

# Final checks
echo -e "${YELLOW}Verifying installation...${NC}"
if systemctl is-active --quiet vwireguard; then
    echo -e "${GREEN}vWireguard service is running!${NC}"
else
    echo -e "${RED}vWireguard service failed to start${NC}"
    journalctl -u vwireguard --no-pager -n 10
    exit 1
fi

echo -e "${GREEN}Installation completed successfully!${NC}"
echo -e "\n${YELLOW}=======================================================${NC}"
echo -e "${GREEN}Default Admin Credentials:${NC}"
echo -e "  ${YELLOW}Username: admin${NC}"
echo -e "  ${YELLOW}Password: admin${NC}"
echo -e "${GREEN}Access URL: http://$(curl -s ifconfig.me):5000${NC}"
echo -e "${YELLOW}=======================================================${NC}\n"