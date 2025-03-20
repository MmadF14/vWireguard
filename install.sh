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
apt-get install -y wireguard wireguard-tools golang-go git

# Create directory for vWireguard
echo -e "${YELLOW}Creating vWireguard directory...${NC}"
mkdir -p /opt/vwireguard
cd /opt/vwireguard

# Clone repository
echo -e "${YELLOW}Cloning vWireguard repository...${NC}"
git clone https://github.com/MmadF14/vwireguard.git .

# Build the application
echo -e "${YELLOW}Building vWireguard...${NC}"
go build -o vwireguard

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

# Enable and start service
echo -e "${YELLOW}Starting vWireguard service...${NC}"
systemctl daemon-reload
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

echo -e "${GREEN}Installation completed!${NC}"
echo -e "${GREEN}Default credentials:${NC}"
echo -e "${GREEN}Username: admin${NC}"
echo -e "${GREEN}Password: admin${NC}"
echo -e "${GREEN}Please change the default password after first login!${NC}"
echo -e "${GREEN}Access the web interface at http://localhost:8080${NC}"

# Setup firewall rules
echo -e "${YELLOW}Setting up firewall rules...${NC}"
if command -v ufw &> /dev/null; then
    ufw allow 5000/tcp
    ufw allow 51820/udp
elif command -v firewall-cmd &> /dev/null; then
    firewall-cmd --permanent --add-port=5000/tcp
    firewall-cmd --permanent --add-port=51820/udp
    firewall-cmd --reload
fi 