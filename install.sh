#!/bin/bash

# Text colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Print banner
echo -e "${BLUE}"
echo "██╗   ██╗██╗    ██╗██╗██████╗ ███████╗ ██████╗ ██╗   ██╗ █████╗ ██████╗ ██████╗ "
echo "██║   ██║██║    ██║██║██╔══██╗██╔════╝██╔════╝ ██║   ██║██╔══██╗██╔══██╗██╔══██╗"
echo "██║   ██║██║ █╗ ██║██║██████╔╝█████╗  ██║  ███╗██║   ██║███████║██████╔╝██║  ██║"
echo "╚██╗ ██╔╝██║███╗██║██║██╔══██╗██╔══╝  ██║   ██║██║   ██║██╔══██║██╔══██╗██║  ██║"
echo " ╚████╔╝ ╚███╔███╔╝██║██║  ██║███████╗╚██████╔╝╚██████╔╝██║  ██║██║  ██║██████╔╝"
echo "  ╚═══╝   ╚══╝╚══╝ ╚═╝╚═╝  ╚═╝╚══════╝ ╚═════╝  ╚═════╝ ╚═╝  ╚═╝╚═╝  ╚═╝╚═════╝ "
echo -e "${NC}"

# Check if running as root
if [ "$EUID" -ne 0 ]; then 
    echo -e "${RED}Please run as root${NC}"
    exit
fi

# Detect OS
if [ -f /etc/os-release ]; then
    . /etc/os-release
    OS=$NAME
    VER=$VERSION_ID
else
    echo -e "${RED}Cannot detect OS${NC}"
    exit
fi

echo -e "${GREEN}Installing on $OS $VER${NC}"

# Install dependencies
echo -e "${YELLOW}Installing dependencies...${NC}"
if [[ "$OS" == "Ubuntu" ]] || [[ "$OS" == "Debian" ]]; then
    apt update
    apt install -y curl wget unzip net-tools wireguard-tools
elif [[ "$OS" == "CentOS" ]]; then
    yum update -y
    yum install -y curl wget unzip net-tools wireguard-tools
else
    echo -e "${RED}Unsupported OS${NC}"
    exit
fi

# Install Go if not installed
if ! command -v go &> /dev/null; then
    echo -e "${YELLOW}Installing Go...${NC}"
    wget https://go.dev/dl/go1.21.6.linux-amd64.tar.gz
    rm -rf /usr/local/go && tar -C /usr/local -xzf go1.21.6.linux-amd64.tar.gz
    export PATH=$PATH:/usr/local/go/bin
    rm go1.21.6.linux-amd64.tar.gz
fi

# Get latest release version
echo -e "${YELLOW}Getting latest version...${NC}"
LATEST_VERSION=$(curl --silent "https://api.github.com/repos/MmadF14/vwireguard/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')

if [ -z "$LATEST_VERSION" ]; then
    echo -e "${RED}Could not get latest version${NC}"
    exit
fi

echo -e "${GREEN}Latest version: $LATEST_VERSION${NC}"

# Download latest release
echo -e "${YELLOW}Downloading vWireguard...${NC}"
wget "https://github.com/MmadF14/vwireguard/releases/download/$LATEST_VERSION/vwireguard_${LATEST_VERSION}_linux_amd64.tar.gz"
tar -xzf "vwireguard_${LATEST_VERSION}_linux_amd64.tar.gz"
rm "vwireguard_${LATEST_VERSION}_linux_amd64.tar.gz"

# Move binary to /usr/local/bin
mv vwireguard /usr/local/bin/
chmod +x /usr/local/bin/vwireguard

# Create service file
echo -e "${YELLOW}Creating service file...${NC}"
cat > /etc/systemd/system/vwireguard.service << EOF
[Unit]
Description=vWireguard Panel
After=network.target

[Service]
Type=simple
User=root
ExecStart=/usr/local/bin/vwireguard
Restart=on-failure
RestartSec=5s

[Install]
WantedBy=multi-user.target
EOF

# Reload systemd
systemctl daemon-reload

# Start service
echo -e "${YELLOW}Starting vWireguard...${NC}"
systemctl enable vwireguard
systemctl start vwireguard

# Check if service is running
if systemctl is-active --quiet vwireguard; then
    echo -e "${GREEN}vWireguard is running!${NC}"
    echo -e "${GREEN}You can access the panel at: http://YOUR_IP:8080${NC}"
    echo -e "${GREEN}Default credentials:${NC}"
    echo -e "${GREEN}Username: admin${NC}"
    echo -e "${GREEN}Password: admin${NC}"
else
    echo -e "${RED}vWireguard failed to start${NC}"
    echo -e "${RED}Please check logs with: journalctl -u vwireguard${NC}"
fi

# Setup firewall rules
echo -e "${YELLOW}Setting up firewall rules...${NC}"
if command -v ufw &> /dev/null; then
    ufw allow 8080/tcp
    ufw allow 51820/udp
elif command -v firewall-cmd &> /dev/null; then
    firewall-cmd --permanent --add-port=8080/tcp
    firewall-cmd --permanent --add-port=51820/udp
    firewall-cmd --reload
fi

echo -e "${GREEN}Installation completed!${NC}"
echo -e "${YELLOW}Please change your password after first login${NC}" 