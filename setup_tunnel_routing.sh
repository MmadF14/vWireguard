#!/bin/bash

# vWireguard Tunnel Routing Setup Script
# This script sets up the necessary iptables rules for tunnel routing

echo "Setting up vWireguard tunnel routing..."

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