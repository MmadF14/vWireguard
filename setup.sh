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

# Install dependencies
echo -e "${YELLOW}Installing system dependencies...${NC}"
apt update
apt install -y curl wget git unzip net-tools wireguard-tools build-essential

# Install Go if not installed
if ! command -v go &> /dev/null; then
    echo -e "${YELLOW}Installing Go...${NC}"
    wget https://go.dev/dl/go1.21.6.linux-amd64.tar.gz
    rm -rf /usr/local/go && tar -C /usr/local -xzf go1.21.6.linux-amd64.tar.gz
    export PATH=$PATH:/usr/local/go/bin
    echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
    source ~/.bashrc
    rm go1.21.6.linux-amd64.tar.gz
fi

# Clone the repository if not already done
if [ ! -d "vwireguard" ]; then
    echo -e "${YELLOW}Cloning repository...${NC}"
    git clone https://github.com/MmadF14/vwireguard.git
    cd vwireguard
else
    cd vwireguard
    echo -e "${YELLOW}Updating repository...${NC}"
    git pull
fi

# Fix dependencies
echo -e "${YELLOW}Fixing dependencies...${NC}"

# Update go.mod with correct replace directives
cat > go.mod << EOF
module github.com/MmadF14/vwireguard

go 1.21

require (
	github.com/NicoNex/echotron/v3 v3.27.0
	github.com/glendc/go-external-ip v0.1.0
	github.com/gorilla/sessions v1.2.2
	github.com/labstack/echo-contrib v0.15.0
	github.com/labstack/echo/v4 v4.11.4
	github.com/labstack/gommon v0.4.2
	github.com/rs/xid v1.5.0
	github.com/sabhiram/go-wol v0.0.0-20211224004021-c83b0c2f887d
	github.com/sdomino/scribble v0.0.0-20230717151034-b95d4df19aa8
	github.com/sendgrid/sendgrid-go v3.14.0+incompatible
	github.com/shirou/gopsutil/v3 v3.24.1
	github.com/skip2/go-qrcode v0.0.0-20200617195104-da1b6568686e
	github.com/tklauser/go-sysconf v0.3.12
	github.com/xhit/go-simple-mail/v2 v2.16.0
	golang.org/x/crypto v0.17.0
	golang.org/x/mod v0.14.0
	golang.org/x/sys v0.15.0
	golang.zx2c4.com/wireguard/wgctrl v0.0.0-20230429144221-925a1e7659e6
	gopkg.in/go-playground/validator.v9 v9.31.0
)

require (
	github.com/go-playground/locales v0.14.1 // indirect
	github.com/go-playground/universal-translator v0.18.1 // indirect
	github.com/go-test/deep v1.1.0 // indirect
	github.com/golang-jwt/jwt v3.2.2+incompatible // indirect
	github.com/google/go-cmp v0.6.0 // indirect
	github.com/gorilla/context v1.1.2 // indirect
	github.com/gorilla/securecookie v1.1.2 // indirect
	github.com/jcelliott/lumber v0.0.0-20160324203708-dd349441af25 // indirect
	github.com/josharian/native v1.1.0 // indirect
	github.com/leodido/go-urn v1.2.4 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/mdlayher/genetlink v1.3.2 // indirect
	github.com/mdlayher/netlink v1.7.2 // indirect
	github.com/mdlayher/socket v0.5.0 // indirect
	github.com/sendgrid/rest v2.6.9+incompatible // indirect
	github.com/toorop/go-dkim v0.0.0-20201103131630-e1cd1a0a5208 // indirect
	github.com/valyala/bytebufferpool v1.0.0 // indirect
	github.com/valyala/fasttemplate v1.2.2 // indirect
	golang.org/x/net v0.19.0 // indirect
	golang.org/x/sync v0.5.0 // indirect
	golang.org/x/text v0.14.0 // indirect
	golang.org/x/time v0.5.0 // indirect
	golang.zx2c4.com/wireguard v0.0.0-20230704135630-469159ecf7d1 // indirect
	gopkg.in/go-playground/assert.v1 v1.2.1 // indirect
)

replace (
	golang.zx2c4.com/wireguard => golang.zx2c4.com/wireguard v0.0.0-20230704135630-469159ecf7d1
	golang.zx2c4.com/wireguard/wgctrl => golang.zx2c4.com/wireguard/wgctrl v0.0.0-20230429144221-925a1e7659e6
)
EOF

# Update dependencies
go mod tidy

# Build
echo -e "${YELLOW}Building...${NC}"
go build -o vwireguard

# Create necessary directories
mkdir -p /etc/vwireguard
mkdir -p /var/lib/vwireguard

# Move binary and files
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
WorkingDirectory=/etc/vwireguard

[Install]
WantedBy=multi-user.target
EOF

# Create default config if it doesn't exist
if [ ! -f "/etc/vwireguard/config.toml" ]; then
    echo -e "${YELLOW}Creating default config...${NC}"
    cat > /etc/vwireguard/config.toml << EOF
listen_port = 5000
username = "admin"
password = "admin"
EOF
fi

# Reload systemd
systemctl daemon-reload

# Start service
echo -e "${YELLOW}Starting vWireguard...${NC}"
systemctl enable vwireguard
systemctl start vwireguard

# Check if service is running
if systemctl is-active --quiet vwireguard; then
    echo -e "${GREEN}vWireguard is running!${NC}"
    echo -e "${GREEN}You can access the panel at: http://YOUR_IP:5000${NC}"
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
    ufw allow 5000/tcp
    ufw allow 51820/udp
elif command -v firewall-cmd &> /dev/null; then
    firewall-cmd --permanent --add-port=5000/tcp
    firewall-cmd --permanent --add-port=51820/udp
    firewall-cmd --reload
fi

echo -e "${GREEN}Setup completed!${NC}"
echo -e "${YELLOW}Please change your password after first login${NC}" 