#!/bin/bash

# vWireguard Tunnel Routing Setup Script
# This script sets up the necessary iptables rules for tunnel routing

# Function to clean up a specific tunnel interface
cleanup_tunnel_interface() {
    local interface="$1"
    
    # Validate interface name (must be wg + exactly 3 chars and not wg0)
    if [ "$interface" = "wg0" ]; then
        echo "ERROR: Cannot clean up main server interface wg0!"
        return 1
    fi
    
    if ! [[ $interface =~ ^wg[a-zA-Z0-9]{3}$ ]]; then
        echo "ERROR: Invalid tunnel interface name: $interface"
        echo "Tunnel interfaces must be in format: wg + exactly 3 characters"
        return 1
    fi
    
    echo "Cleaning up tunnel interface: $interface"
    
    # Check if interface exists
    if ip link show "$interface" >/dev/null 2>&1; then
        echo "Stopping tunnel interface: $interface"
        wg-quick down "$interface" 2>/dev/null || true
        
        # Force remove interface if still exists
        if ip link show "$interface" >/dev/null 2>&1; then
            echo "Force removing interface: $interface"
            ip link delete "$interface" 2>/dev/null || true
        fi
    fi
    
    # Remove config file
    if [ -f "/etc/wireguard/${interface}.conf" ]; then
        echo "Removing config file: /etc/wireguard/${interface}.conf"
        rm -f "/etc/wireguard/${interface}.conf"
    fi
    
    echo "Successfully cleaned up tunnel interface: $interface"
}

# Usage function
show_usage() {
    echo "Usage: $0 [OPTIONS]"
    echo ""
    echo "Options:"
    echo "  --cleanup-only     Only cleanup old tunnel interfaces, don't setup routing"
    echo "  --cleanup IFACE    Cleanup specific tunnel interface (e.g. wg123)"
    echo "  --help            Show this help message"
    echo ""
    echo "Examples:"
    echo "  $0                 # Full setup with cleanup and routing"
    echo "  $0 --cleanup-only  # Only cleanup old tunnel interfaces"
    echo "  $0 --cleanup wg123 # Cleanup specific tunnel interface"
}

# Parse command line arguments
case "$1" in
    --help)
        show_usage
        exit 0
        ;;
    --cleanup)
        if [ -z "$2" ]; then
            echo "ERROR: Interface name required for --cleanup option"
            show_usage
            exit 1
        fi
        cleanup_tunnel_interface "$2"
        exit 0
        ;;
    --cleanup-only)
        echo "Cleanup mode: Only cleaning up old tunnel interfaces"
        ;;
    "")
        echo "Full setup mode: Cleanup + Routing setup"
        ;;
    *)
        echo "ERROR: Unknown option: $1"
        show_usage
        exit 1
        ;;
esac

echo "Setting up vWireguard tunnel routing..."

# Show current WireGuard interfaces
echo ""
echo "Current WireGuard interfaces:"
echo "============================="
ip link show | grep -E 'wg[a-zA-Z0-9]*:' | cut -d: -f2 | tr -d ' ' | while read iface; do
    if [ "$iface" = "wg0" ]; then
        echo "  $iface (MAIN SERVER INTERFACE - PROTECTED)"
    else
        echo "  $iface (tunnel interface - will be cleaned)"
    fi
done
echo ""

# Cleanup old tunnel interfaces first
echo "Cleaning up old tunnel interfaces..."
echo "WARNING: This will only clean tunnel interfaces (wg + 3 chars), NOT the main wg0 interface"

# More specific pattern to match only tunnel interfaces (wg + exactly 3 characters)
for interface in $(ip link show | grep -E 'wg[a-zA-Z0-9]{3}:' | cut -d: -f2 | tr -d ' '); do
    # Double check it's not wg0 and has exactly 3 chars after 'wg'
    if [ "$interface" != "wg0" ] && [ ${#interface} -eq 5 ] && [[ $interface =~ ^wg[a-zA-Z0-9]{3}$ ]]; then
        echo "Found old tunnel interface: $interface"
        
        # Check if interface is actually up before trying to stop it
        if ip link show "$interface" >/dev/null 2>&1; then
            echo "Stopping tunnel interface: $interface"
            wg-quick down "$interface" 2>/dev/null || true
            
            # Force remove interface if still exists
            if ip link show "$interface" >/dev/null 2>&1; then
                echo "Force removing interface: $interface"
                ip link delete "$interface" 2>/dev/null || true
            fi
            
            # Remove config file
            if [ -f "/etc/wireguard/${interface}.conf" ]; then
                echo "Removing config file: /etc/wireguard/${interface}.conf"
                rm -f "/etc/wireguard/${interface}.conf"
            fi
            
            echo "Cleaned up interface: $interface"
        else
            echo "Interface $interface does not exist, skipping"
        fi
    else
        echo "Skipping interface: $interface (protected or invalid format)"
    fi
done

echo "Tunnel interface cleanup completed"
echo "Main WireGuard server interface (wg0) was NOT touched"

# Skip routing setup if cleanup-only mode
if [ "$1" = "--cleanup-only" ]; then
    echo ""
    echo "✅ Cleanup completed! Skipping routing setup as requested."
    echo "To set up routing later, run: sudo bash $0"
    exit 0
fi

# Get the main network interface
MAIN_INTERFACE=$(ip route show default | awk '/default/ { print $5 }')
if [ -z "$MAIN_INTERFACE" ]; then
    MAIN_INTERFACE="eth0"
    echo "Warning: Could not detect main interface, using default: $MAIN_INTERFACE"
else
    echo "Detected main interface: $MAIN_INTERFACE"
fi

# Enable IP forwarding
echo "Enabling IP forwarding..."
echo 'net.ipv4.ip_forward = 1' >> /etc/sysctl.conf
sysctl -p

# Set up basic iptables rules for WireGuard server
echo "Setting up iptables rules for WireGuard server (wg0)..."

# Remove any existing rules to avoid duplicates
iptables -D POSTROUTING -t nat -o wg0 -j MASQUERADE 2>/dev/null || true
iptables -D POSTROUTING -t nat -o $MAIN_INTERFACE -j MASQUERADE 2>/dev/null || true
iptables -D FORWARD -i wg0 -j ACCEPT 2>/dev/null || true
iptables -D FORWARD -o wg0 -j ACCEPT 2>/dev/null || true

# Add the correct rules
iptables -A POSTROUTING -t nat -o $MAIN_INTERFACE -j MASQUERADE
iptables -A FORWARD -i wg0 -j ACCEPT
iptables -A FORWARD -o wg0 -j ACCEPT

# Save iptables rules (method varies by distribution)
if command -v iptables-save >/dev/null 2>&1; then
    if [ -f /etc/iptables/rules.v4 ]; then
        iptables-save > /etc/iptables/rules.v4
        echo "Saved iptables rules to /etc/iptables/rules.v4"
    elif [ -f /etc/sysconfig/iptables ]; then
        iptables-save > /etc/sysconfig/iptables
        echo "Saved iptables rules to /etc/sysconfig/iptables"
    else
        echo "Warning: Could not determine where to save iptables rules"
        echo "Please save them manually using your distribution's method"
    fi
fi

echo ""
echo "✅ Tunnel routing setup completed!"
echo ""
echo "Current iptables rules:"
echo "======================"
iptables -L FORWARD -v --line-numbers
echo ""
echo "NAT rules:"
echo "=========="
iptables -t nat -L POSTROUTING -v --line-numbers
echo ""
echo "IP forwarding status:"
echo "===================="
sysctl net.ipv4.ip_forward
echo ""
echo "Main interface: $MAIN_INTERFACE"
echo ""
echo "🔧 If you need to remove these rules later, run:"
echo "   iptables -D POSTROUTING -t nat -o $MAIN_INTERFACE -j MASQUERADE"
echo "   iptables -D FORWARD -i wg0 -j ACCEPT"
echo "   iptables -D FORWARD -o wg0 -j ACCEPT" 