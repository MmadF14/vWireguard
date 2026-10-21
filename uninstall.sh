#!/bin/bash
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo -e "${YELLOW}Starting uninstallation...${NC}"
echo -e "${YELLOW}Stopping services...${NC}"
systemctl stop vwireguard
systemctl stop wg-quick@wg0
systemctl disable vwireguard
systemctl disable wg-quick@wg0
echo -e "${YELLOW}Removing service files...${NC}"
rm -f /etc/systemd/system/vwireguard.service
rm -f /etc/systemd/system/wg-quick@wg0.service
echo -e "${YELLOW}Removing WireGuard files...${NC}"
rm -rf /etc/wireguard
rm -rf /opt/vwireguard
echo -e "${YELLOW}Removing packages...${NC}"
apt-get remove -y wireguard wireguard-tools wireguard-dkms
apt-get remove -y nginx
apt-get remove -y python3-pip
apt-get autoremove -y
echo -e "${YELLOW}Removing configuration files...${NC}"
rm -f /etc/nginx/sites-enabled/vwireguard
rm -f /etc/nginx/sites-available/vwireguard
rm -f /root/vwireguard_credentials.txt
echo -e "${YELLOW}Reloading systemd...${NC}"
systemctl daemon-reload

echo -e "${GREEN}Uninstallation completed successfully!${NC}"
echo -e "${YELLOW}Please reboot your system to complete the cleanup.${NC}" 