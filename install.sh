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
    exit 1
fi

# Update system
echo -e "${YELLOW}Updating system...${NC}"
apt-get update
apt-get upgrade -y

# Install required packages
echo -e "${YELLOW}Installing required packages...${NC}"
apt-get install -y wireguard wireguard-tools git curl wget build-essential

# Install latest Go version
echo -e "${YELLOW}Installing latest Go version...${NC}"
wget https://go.dev/dl/go1.21.6.linux-amd64.tar.gz
rm -rf /usr/local/go && tar -C /usr/local -xzf go1.21.6.linux-amd64.tar.gz
export PATH=$PATH:/usr/local/go/bin
rm go1.21.6.linux-amd64.tar.gz

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

# Create WireGuard server configuration
echo -e "${YELLOW}Creating WireGuard server configuration...${NC}"
cat > /etc/wireguard/wg0.conf << EOL
[Interface]
PrivateKey = $(cat /etc/wireguard/server_private.key)
Address = 10.0.0.1/24
ListenPort = 51820
PostUp = iptables -A FORWARD -i wg0 -j ACCEPT; iptables -t nat -A POSTROUTING -o eth0 -j MASQUERADE
PostDown = iptables -D FORWARD -i wg0 -j ACCEPT; iptables -t nat -D POSTROUTING -o eth0 -j MASQUERADE

# Client configurations will be added here
EOL

# Create directory for vWireguard
echo -e "${YELLOW}Creating vWireguard directory...${NC}"
rm -rf /opt/vwireguard
mkdir -p /opt/vwireguard
cd /opt/vwireguard

# Clone repository
echo -e "${YELLOW}Cloning vWireguard repository...${NC}"
git clone https://github.com/MmadF14/vwireguard.git .

# Build the application
echo -e "${YELLOW}Building vWireguard...${NC}"
export GOPATH=/opt/vwireguard
export PATH=$PATH:/usr/local/go/bin

# Check if Go is installed correctly
if ! command -v go &> /dev/null; then
    echo -e "${RED}Go is not installed correctly${NC}"
    exit 1
fi

# Initialize Go module if not exists
echo -e "${YELLOW}Checking for Go module...${NC}"
if [ ! -f "go.mod" ]; then
    echo -e "${YELLOW}Creating go.mod file...${NC}"
    go mod init github.com/MmadF14/vwireguard
fi

# Download dependencies
echo -e "${YELLOW}Downloading dependencies...${NC}"
go mod tidy

# Build the application
echo -e "${YELLOW}Building vWireguard...${NC}"
go build -o vwireguard

# Check if build was successful
if [ ! -f "vwireguard" ]; then
    echo -e "${RED}Build failed!${NC}"
    exit 1
fi

# Make the binary executable
chmod +x vwireguard

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

# Create WireGuard service
echo -e "${YELLOW}Creating WireGuard service...${NC}"
systemctl enable wg-quick@wg0
systemctl start wg-quick@wg0

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

# Stop and disable existing service if it exists
echo -e "${YELLOW}Stopping existing service...${NC}"
systemctl stop vwireguard || true
systemctl disable vwireguard || true

# Enable and start vWireguard service
echo -e "${YELLOW}Starting vWireguard service...${NC}"
systemctl daemon-reload
systemctl enable vwireguard
systemctl start vwireguard

# Check if service is running
echo -e "${YELLOW}Checking service status...${NC}"
if systemctl is-active --quiet vwireguard; then
    echo -e "${GREEN}vWireguard service is running!${NC}"
else
    echo -e "${RED}vWireguard service failed to start${NC}"
    echo -e "${RED}Please check logs with: journalctl -u vwireguard${NC}"
    exit 1
fi

echo -e "${GREEN}Installation completed!${NC}"
echo -e "${GREEN}Default credentials:${NC}"
echo -e "${GREEN}Username: admin${NC}"
echo -e "${GREEN}Password: admin${NC}"
echo -e "${GREEN}Please change the default password after first login!${NC}"
echo -e "${GREEN}Access the web interface at http://YOUR_SERVER_IP:5000${NC}"
echo -e "${GREEN}WireGuard server is running on port 51820${NC}"
echo -e "${YELLOW}Note: Please configure your firewall to allow ports 5000 (TCP) and 51820 (UDP)${NC}" 