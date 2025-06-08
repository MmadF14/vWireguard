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

show_menu() {
    echo -e "\n${GREEN}vWireguard Panel Management Script${NC}"
    echo -e "${YELLOW}0.${NC} Exit Script"
    echo -e "\n${BLUE}=== Installation ===${NC}"
    echo -e "${YELLOW}1.${NC} Install"
    echo -e "${YELLOW}2.${NC} Update"
    echo -e "${YELLOW}3.${NC} Uninstall"
    echo -e "\n${BLUE}=== Configuration ===${NC}"
    echo -e "${YELLOW}4.${NC} Reset Username & Password"
    echo -e "${YELLOW}5.${NC} Change Port"
    echo -e "${YELLOW}6.${NC} View Current Settings"
    echo -e "\n${BLUE}=== Service Management ===${NC}"
    echo -e "${YELLOW}7.${NC} Start"
    echo -e "${YELLOW}8.${NC} Stop"
    echo -e "${YELLOW}9.${NC} Restart"
    echo -e "${YELLOW}10.${NC} Check Status"
    echo -e "${YELLOW}11.${NC} View Logs"
    echo -e "\n${BLUE}=== Security ===${NC}"
    echo -e "${YELLOW}12.${NC} SSL Certificate Management"
    echo -e "${YELLOW}13.${NC} IP Limit Management"
    echo -e "${YELLOW}14.${NC} Firewall Management"
    echo -e "\n${BLUE}=== System ===${NC}"
    echo -e "${YELLOW}15.${NC} Backup Configuration"
    echo -e "${YELLOW}16.${NC} Restore Configuration"
    echo -e "${YELLOW}17.${NC} System Information"
}

install_panel() {
    bash <(curl -Ls https://raw.githubusercontent.com/MmadF14/vwireguard/main/install.sh)
}

update_panel() {
    echo -e "${YELLOW}Updating vWireguard...${NC}"
    systemctl stop vwireguard
    bash <(curl -Ls https://raw.githubusercontent.com/MmadF14/vwireguard/main/install.sh)
    echo -e "${GREEN}Update completed!${NC}"
}

uninstall_panel() {
    echo -e "${RED}Are you sure you want to uninstall vWireguard? (y/N)${NC}"
    read -r confirm
    if [[ "$confirm" =~ ^[Yy]$ ]]; then
        systemctl stop vwireguard
        systemctl disable vwireguard
        rm -f /usr/local/bin/vwireguard
        rm -f /etc/systemd/system/vwireguard.service
        systemctl daemon-reload
        echo -e "${GREEN}vWireguard has been uninstalled${NC}"
    fi
}

reset_credentials() {
    echo -e "${YELLOW}Resetting credentials...${NC}"
    service_file="/etc/systemd/system/vwireguard.service"
    if [ ! -f "$service_file" ]; then
        echo -e "${RED}Service file not found${NC}"
        return
    fi

    new_user=$(tr -dc 'A-Za-z0-9' </dev/urandom | head -c 8)
    new_pass=$(tr -dc 'A-Za-z0-9' </dev/urandom | head -c 8)

    systemctl stop vwireguard

    if grep -q 'WGUI_USERNAME' "$service_file"; then
        sed -i "s|Environment=\"WGUI_USERNAME=.*\"|Environment=\"WGUI_USERNAME=${new_user}\"|" "$service_file"
    else
        sed -i "/^WorkingDirectory=/a Environment=\"WGUI_USERNAME=${new_user}\"" "$service_file"
    fi

    if grep -q 'WGUI_PASSWORD' "$service_file"; then
        sed -i "s|Environment=\"WGUI_PASSWORD=.*\"|Environment=\"WGUI_PASSWORD=${new_pass}\"|" "$service_file"
    else
        sed -i "/WGUI_USERNAME=/a Environment=\"WGUI_PASSWORD=${new_pass}\"" "$service_file"
    fi

    systemctl daemon-reload
    systemctl start vwireguard

    echo -e "${GREEN}Credentials have been reset to:${NC}"
    echo -e "Username: ${new_user}"
    echo -e "Password: ${new_pass}"
    echo "Username: ${new_user}" > /root/vwireguard_credentials.txt
    echo "Password: ${new_pass}" >> /root/vwireguard_credentials.txt
}

change_port() {
    echo -e "${YELLOW}Enter new port number:${NC}"
    read -r new_port
    
    if ! [[ "$new_port" =~ ^[0-9]+$ ]] || [ "$new_port" -lt 1 ] || [ "$new_port" -gt 65535 ]; then
        echo -e "${RED}Invalid port number${NC}"
        return
    fi
    
    if [ ! -f "/etc/vwireguard/config.toml" ]; then
        echo -e "${RED}Config file not found${NC}"
        return
    }
    
    # Stop the service
    systemctl stop vwireguard
    
    # Update port in config file
    sed -i "s/listen_port = .*/listen_port = $new_port/" /etc/vwireguard/config.toml
    
    # Update firewall rules
    if command -v ufw &> /dev/null; then
        ufw delete allow 5000/tcp 2>/dev/null
        ufw allow $new_port/tcp
    elif command -v firewall-cmd &> /dev/null; then
        firewall-cmd --permanent --remove-port=5000/tcp 2>/dev/null
        firewall-cmd --permanent --add-port=$new_port/tcp
        firewall-cmd --reload
    fi
    
    # Start the service
    systemctl start vwireguard
    
    echo -e "${GREEN}Port has been changed to $new_port${NC}"
    echo -e "${YELLOW}Please access the panel at: http://YOUR_IP:$new_port${NC}"
}

view_settings() {
    echo -e "${BLUE}Current Settings:${NC}"
    echo -e "Panel Port: $(grep -oP 'listen_port\s*=\s*\K\d+' /etc/vwireguard/config.toml)"
    echo -e "WireGuard Port: $(grep -oP 'ListenPort\s*=\s*\K\d+' /etc/wireguard/wg0.conf)"
    systemctl is-active --quiet vwireguard && echo -e "Panel Status: ${GREEN}Running${NC}" || echo -e "Panel Status: ${RED}Stopped${NC}"
}

manage_service() {
    case $1 in
        "start")
            systemctl start vwireguard
            ;;
        "stop")
            systemctl stop vwireguard
            ;;
        "restart")
            systemctl restart vwireguard
            ;;
        "status")
            systemctl status vwireguard
            ;;
        "enable-autostart")
            systemctl enable vwireguard
            echo -e "${GREEN}Autostart enabled for vWireguard${NC}"
            ;;
        "disable-autostart")
            systemctl disable vwireguard
            echo -e "${GREEN}Autostart disabled for vWireguard${NC}"
            ;;
    esac
}

view_logs() {
    journalctl -u vwireguard -n 100 --no-pager
}

manage_ssl() {
    echo -e "${YELLOW}SSL Certificate Management${NC}"
    echo "1. Install SSL Certificate"
    echo "2. Renew SSL Certificate"
    echo "3. View SSL Status"
    read -r ssl_choice
<<<<<<< HEAD
    case $ssl_choice in
        1)
            read -rp "Enter domain: " domain
            read -rp "Enter email (leave blank to skip): " email
            apt-get install -y nginx certbot python3-certbot-nginx
            cat > /etc/nginx/sites-available/vwireguard <<CONF
server {
    listen 80;
    server_name ${domain};
    location / {
        proxy_pass http://127.0.0.1:5000;
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;

        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;


    }
}
CONF
            ln -sf /etc/nginx/sites-available/vwireguard /etc/nginx/sites-enabled/vwireguard
            nginx -s reload || systemctl restart nginx
            if [ -n "$email" ]; then
                certbot --nginx --non-interactive --agree-tos -m "$email" -d "$domain"
            else
                certbot --nginx --register-unsafely-without-email --non-interactive --agree-tos -d "$domain"
            fi
            ;;
        2)
            certbot renew --quiet
            ;;
        3)
            certbot certificates
            ;;
        *)
            echo -e "${RED}Invalid option${NC}"
            ;;
    esac
=======
    # Add your SSL management logic here
>>>>>>> parent of 37fbd02 (Add optional SSL setup)
}

manage_ip_limit() {
    echo -e "${YELLOW}IP Limit Management${NC}"
    echo "1. Set Connection Limit"
    echo "2. View Current Limits"
    echo "3. Reset Limits"
    read -r ip_choice
    # Add your IP limit management logic here
}

manage_firewall() {
    echo -e "${YELLOW}Firewall Management${NC}"
    echo "1. Install Firewall"
    echo "2. View Firewall Rules"
    echo "3. Allow Port"
    echo "4. Block Port"
    echo "5. Enable Firewall"
    echo "6. Disable Firewall"
    echo "7. View Firewall Status"
    read -r fw_choice

    case $fw_choice in
        1)
            # Install firewall logic here
            echo -e "${GREEN}Firewall installed!${NC}"
            ;;
        2)
            # View firewall rules logic here
            echo -e "${GREEN}Current firewall rules:${NC}"
            ;;
        3)
            echo -e "${YELLOW}Enter port to allow:${NC}"
            read -r port
            ufw allow "$port"
            echo -e "${GREEN}Port $port allowed!${NC}"
            ;;
        4)
            echo -e "${YELLOW}Enter port to block:${NC}"
            read -r port
            ufw deny "$port"
            echo -e "${GREEN}Port $port blocked!${NC}"
            ;;
        5)
            ufw enable
            echo -e "${GREEN}Firewall enabled!${NC}"
            ;;
        6)
            ufw disable
            echo -e "${GREEN}Firewall disabled!${NC}"
            ;;
        7)
            ufw status
            ;;
        *)
            echo -e "${RED}Invalid option${NC}"
            ;;
    esac
}

backup_config() {
    echo -e "${YELLOW}Creating backup...${NC}"
    backup_dir="/root/vwireguard_backup_$(date +%Y%m%d_%H%M%S)"
    mkdir -p "$backup_dir"
    cp -r /etc/vwireguard "$backup_dir/"
    cp -r /etc/wireguard "$backup_dir/"
    tar -czf "${backup_dir}.tar.gz" "$backup_dir"
    rm -rf "$backup_dir"
    echo -e "${GREEN}Backup created at ${backup_dir}.tar.gz${NC}"
}
 
restore_config() {
    echo -e "${YELLOW}Enter backup file path:${NC}"
    read -r backup_file
    if [ -f "$backup_file" ]; then
        systemctl stop vwireguard
        tar -xzf "$backup_file" -C /
        systemctl start vwireguard
        echo -e "${GREEN}Configuration restored!${NC}"
    else
        echo -e "${RED}Backup file not found${NC}"
    fi
}

show_system_info() {
    echo -e "${BLUE}=== System Information ===${NC}"
    echo -e "OS: $(cat /etc/os-release | grep PRETTY_NAME | cut -d'"' -f2)"
    echo -e "Kernel: $(uname -r)"
    echo -e "CPU: $(grep 'model name' /proc/cpuinfo | head -n1 | cut -d':' -f2)"
    echo -e "Memory: $(free -h | awk '/^Mem:/ {print $2}')"
    echo -e "Disk Usage: $(df -h / | awk 'NR==2 {print $5}')"
    echo -e "Panel Version: $(vwireguard -v 2>/dev/null || echo 'Unknown')"
}

# Main loop
while true; do
    show_menu
    echo -e "\n${GREEN}Please enter your selection [0-17]:${NC}"
    read -r choice

    case $choice in
        0) exit 0 ;;
        1) install_panel ;;
        2) update_panel ;;
        3) uninstall_panel ;;
        4) reset_credentials ;;
        5) change_port ;;
        6) view_settings ;;
        7) manage_service "start" ;;
        8) manage_service "stop" ;;
        9) manage_service "restart" ;;
        10) manage_service "status" ;;
        11) view_logs ;;
        12) manage_ssl ;;
        13) manage_ip_limit ;;
        14) manage_firewall ;;
        15) manage_service "enable-autostart" ;;
        16) manage_service "disable-autostart" ;;
        17) show_system_info ;;
        *) echo -e "${RED}Invalid option${NC}" ;;
    esac

    echo -e "\n${YELLOW}Press Enter to continue...${NC}"
    read -r
done 
