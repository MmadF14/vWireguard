#!/bin/bash

# Debug script for V2Ray tunnel issues
# This script will help identify the cause of "exit status 5" errors

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

echo -e "${BLUE}=== V2Ray Tunnel Debug Script ===${NC}"
echo ""

# Function to check if command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Function to print section header
print_section() {
    echo -e "${YELLOW}--- $1 ---${NC}"
}

# Function to print status
print_status() {
    if [ $1 -eq 0 ]; then
        echo -e "${GREEN}✓ $2${NC}"
    else
        echo -e "${RED}✗ $2${NC}"
    fi
}

# 1. Check system information
print_section "System Information"
echo "OS: $(cat /etc/os-release | grep PRETTY_NAME | cut -d'"' -f2)"
echo "Kernel: $(uname -r)"
echo "Architecture: $(uname -m)"
echo ""

# 2. Check xray installation
print_section "Xray Installation Check"

xray_found=false
xray_paths=("/usr/local/bin/xray" "/usr/bin/xray" "/opt/xray/xray")

for path in "${xray_paths[@]}"; do
    if [ -f "$path" ]; then
        echo -e "${GREEN}✓ Found xray at: $path${NC}"
        xray_found=true
        xray_path="$path"
        break
    else
        echo -e "${RED}✗ Not found: $path${NC}"
    fi
done

if [ "$xray_found" = false ]; then
    echo -e "${RED}❌ Xray binary not found in any expected location!${NC}"
    echo ""
    echo -e "${YELLOW}Installing xray-core...${NC}"
    bash install_xray.sh
    echo ""
    
    # Check again after installation
    for path in "${xray_paths[@]}"; do
        if [ -f "$path" ]; then
            echo -e "${GREEN}✓ Found xray at: $path (after installation)${NC}"
            xray_found=true
            xray_path="$path"
            break
        fi
    done
fi

if [ "$xray_found" = true ]; then
    echo "Xray version: $($xray_path version 2>/dev/null || echo 'Unable to get version')"
    echo "Xray permissions: $(ls -la $xray_path)"
fi
echo ""

# 3. Check vWireguard service status
print_section "vWireguard Service Status"
if systemctl is-active --quiet vwireguard; then
    echo -e "${GREEN}✓ vWireguard service is running${NC}"
else
    echo -e "${RED}✗ vWireguard service is not running${NC}"
    echo "Starting vWireguard service..."
    systemctl start vwireguard
    sleep 3
    if systemctl is-active --quiet vwireguard; then
        echo -e "${GREEN}✓ vWireguard service started successfully${NC}"
    else
        echo -e "${RED}✗ Failed to start vWireguard service${NC}"
        echo "Service logs:"
        journalctl -u vwireguard --no-pager -n 10
    fi
fi
echo ""

# 4. Check tunnel configurations
print_section "Tunnel Configurations"
if [ -d "/etc/vwireguard/tunnels" ]; then
    echo "Tunnel configs found:"
    ls -la /etc/vwireguard/tunnels/
    echo ""
    
    # Check each tunnel config
    for config in /etc/vwireguard/tunnels/*.json; do
        if [ -f "$config" ]; then
            tunnel_id=$(basename "$config" .json)
            echo "Tunnel ID: $tunnel_id"
            echo "Config file: $config"
            echo "File permissions: $(ls -la $config)"
            
            # Validate config with xray
            if [ "$xray_found" = true ]; then
                echo "Validating config with xray..."
                # Try test command first
                if $xray_path test -c "$config" 2>&1; then
                    echo -e "${GREEN}✓ Config validation passed${NC}"
                else
                    # If test command fails, try alternative validation
                    echo "Test command failed, trying alternative validation..."
                    if $xray_path -c "$config" -test 2>&1; then
                        echo -e "${GREEN}✓ Config validation passed (alternative method)${NC}"
                    else
                        # Check JSON syntax
                        if python3 -m json.tool "$config" >/dev/null 2>&1; then
                            echo -e "${GREEN}✓ JSON syntax is valid${NC}"
                            echo -e "${YELLOW}⚠ Config validation failed but JSON is valid - this may be a xray version compatibility issue${NC}"
                        else
                            echo -e "${RED}✗ Config validation failed - invalid JSON syntax${NC}"
                        fi
                    fi
                fi
            fi
            echo ""
        fi
    done
else
    echo -e "${YELLOW}No tunnel configurations found${NC}"
fi
echo ""

# 5. Check systemd services
print_section "Systemd Services"
echo "vWireguard tunnel services:"
systemctl list-units --type=service | grep vwireguard-tunnel || echo "No tunnel services found"
echo ""

# Check specific tunnel service if provided
if [ ! -z "$1" ]; then
    tunnel_id="$1"
    service_name="vwireguard-tunnel-$tunnel_id.service"
    
    print_section "Specific Tunnel Service: $service_name"
    
    if systemctl list-unit-files | grep -q "$service_name"; then
        echo "Service exists: ✓"
        echo "Service status:"
        systemctl status "$service_name" --no-pager -l
        
        echo ""
        echo "Service logs:"
        journalctl -u "$service_name" --no-pager -n 20
        
        echo ""
        echo "Service file content:"
        cat "/etc/systemd/system/$service_name" 2>/dev/null || echo "Service file not found"
        
        echo ""
        echo "Config file content:"
        cat "/etc/vwireguard/tunnels/$tunnel_id.json" 2>/dev/null || echo "Config file not found"
        
        echo ""
        echo "Testing service start:"
        systemctl start "$service_name" 2>&1
        start_exit_code=$?
        echo "Start exit code: $start_exit_code"
        
        if [ $start_exit_code -eq 0 ]; then
            echo -e "${GREEN}✓ Service started successfully${NC}"
        else
            echo -e "${RED}✗ Service failed to start (exit code: $start_exit_code)${NC}"
            echo "Recent logs:"
            journalctl -u "$service_name" --no-pager -n 10
        fi
    else
        echo -e "${RED}Service $service_name not found${NC}"
    fi
fi
echo ""

# 6. Check network configuration
print_section "Network Configuration"
echo "IP forwarding status:"
sysctl net.ipv4.ip_forward net.ipv6.conf.all.forwarding 2>/dev/null || echo "Unable to check IP forwarding"

echo ""
echo "Network interfaces:"
ip addr show | grep -E "^[0-9]+:|inet " | head -20

echo ""
echo "Routing table:"
ip route show | head -10
echo ""

# 7. Check firewall rules
print_section "Firewall Rules"
if command_exists ufw; then
    echo "UFW status:"
    ufw status
elif command_exists firewall-cmd; then
    echo "Firewalld status:"
    firewall-cmd --state
    echo "Active zones:"
    firewall-cmd --get-active-zones
else
    echo "No firewall detected"
fi
echo ""

# 8. Check system resources
print_section "System Resources"
echo "Memory usage:"
free -h

echo ""
echo "Disk usage:"
df -h /

echo ""
echo "Load average:"
uptime
echo ""

# 9. Check for common issues
print_section "Common Issues Check"

# Check if xray is executable
if [ "$xray_found" = true ]; then
    if [ -x "$xray_path" ]; then
        echo -e "${GREEN}✓ Xray is executable${NC}"
    else
        echo -e "${RED}✗ Xray is not executable${NC}"
        echo "Fixing permissions..."
        chmod +x "$xray_path"
    fi
fi

# Check if config directory exists and has proper permissions
if [ -d "/etc/vwireguard/tunnels" ]; then
    echo -e "${GREEN}✓ Tunnel config directory exists${NC}"
    echo "Directory permissions: $(ls -ld /etc/vwireguard/tunnels)"
else
    echo -e "${RED}✗ Tunnel config directory missing${NC}"
    echo "Creating directory..."
    mkdir -p /etc/vwireguard/tunnels
    chmod 755 /etc/vwireguard/tunnels
fi

# Check systemd daemon
echo ""
echo "Systemd daemon status:"
systemctl daemon-reload
if [ $? -eq 0 ]; then
    echo -e "${GREEN}✓ Systemd daemon reloaded successfully${NC}"
else
    echo -e "${RED}✗ Failed to reload systemd daemon${NC}"
fi
echo ""

# 10. Recommendations
print_section "Recommendations"

if [ "$xray_found" = false ]; then
    echo -e "${RED}1. Install xray-core:${NC}"
    echo "   bash install_xray.sh"
    echo ""
fi

echo -e "${YELLOW}2. If you're still getting 'exit status 5':${NC}"
echo "   - Check the specific tunnel service logs:"
echo "     journalctl -u vwireguard-tunnel-<TUNNEL_ID> -f"
echo "   - Verify the tunnel configuration in the web panel"
echo "   - Make sure all required fields are filled"
echo "   - Try stopping and starting the tunnel again"
echo ""

echo -e "${YELLOW}3. For more detailed debugging:${NC}"
echo "   - Check vWireguard logs: journalctl -u vwireguard -f"
echo "   - Verify xray configuration: $xray_path test -c /etc/vwireguard/tunnels/<TUNNEL_ID>.json"
echo "   - Check system resources and ensure sufficient memory/disk space"
echo ""

echo -e "${BLUE}=== Debug Complete ===${NC}"
echo ""
echo -e "${YELLOW}To debug a specific tunnel, run:${NC}"
echo "   bash debug_v2ray.sh <TUNNEL_ID>" 