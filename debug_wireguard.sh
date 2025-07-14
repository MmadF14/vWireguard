#!/bin/bash

# WireGuard Debug Script
# This script helps debug peer management issues

echo "🔍 WireGuard Debug Script"
echo "=========================="

# Check if running as root
if [ "$EUID" -ne 0 ]; then
    echo "❌ This script must be run as root (use sudo)"
    exit 1
fi

# Function to check interface status
check_interface() {
    local interface="$1"
    echo "📡 Checking interface: $interface"
    
    if wg show "$interface" >/dev/null 2>&1; then
        echo "✅ Interface $interface is active"
        echo "📊 Current peers:"
        wg show "$interface" | grep -E "^peer" || echo "   No peers found"
    else
        echo "❌ Interface $interface is not active"
        return 1
    fi
}

# Function to show detailed peer information
show_peers() {
    local interface="$1"
    echo "🔍 Detailed peer information for $interface:"
    wg show "$interface" dump | while IFS=$'\t' read -r pubkey preshared endpoint allowedips lasthandshake rxbytes txbytes persistentkeepalive; do
        if [ "$pubkey" != "wg0" ]; then
            echo "   Peer: $pubkey"
            echo "     Allowed IPs: $allowedips"
            echo "     Endpoint: $endpoint"
            echo "     Last Handshake: $lasthandshake"
            echo "     RX/TX: $rxbytes/$txbytes bytes"
            echo "     Persistent Keepalive: $persistentkeepalive"
            echo ""
        fi
    done
}

# Function to test wg commands
test_wg_commands() {
    local interface="$1"
    echo "🧪 Testing wg commands for $interface:"
    
    # Test basic wg show
    echo "   Testing: wg show $interface"
    if wg show "$interface" >/dev/null 2>&1; then
        echo "   ✅ wg show works"
    else
        echo "   ❌ wg show failed"
    fi
    
    # Test wg show dump
    echo "   Testing: wg show $interface dump"
    if wg show "$interface" dump >/dev/null 2>&1; then
        echo "   ✅ wg show dump works"
    else
        echo "   ❌ wg show dump failed"
    fi
}

# Function to check API endpoints
check_api() {
    echo "🌐 Checking API endpoints:"
    
    # Get base URL from environment or default
    BASE_URL="${BASE_URL:-http://localhost:5000}"
    
    # Check status endpoint
    echo "   Testing: GET $BASE_URL/api/wg/status"
    if curl -s "$BASE_URL/api/wg/status" >/dev/null 2>&1; then
        echo "   ✅ Status endpoint accessible"
        curl -s "$BASE_URL/api/wg/status" | jq . 2>/dev/null || echo "   ⚠️  Status response (not JSON):"
        curl -s "$BASE_URL/api/wg/status"
    else
        echo "   ❌ Status endpoint failed"
    fi
    
    # Check diffs endpoint
    echo "   Testing: GET $BASE_URL/api/wg/diffs"
    if curl -s "$BASE_URL/api/wg/diffs" >/dev/null 2>&1; then
        echo "   ✅ Diffs endpoint accessible"
        curl -s "$BASE_URL/api/wg/diffs" | jq . 2>/dev/null || echo "   ⚠️  Diffs response (not JSON):"
        curl -s "$BASE_URL/api/wg/diffs"
    else
        echo "   ❌ Diffs endpoint failed"
    fi
}

# Function to check logs
check_logs() {
    echo "📋 Checking recent logs:"
    
    # Check systemd logs
    echo "   Recent vwireguard service logs:"
    journalctl -u vwireguard --no-pager -n 20 | grep -E "(DEBUG|ERROR|SUCCESS|INFO)" || echo "   No relevant logs found"
    
    # Check for WireGuard related logs
    echo "   Recent WireGuard related logs:"
    journalctl --no-pager -n 50 | grep -i wireguard || echo "   No WireGuard logs found"
}

# Function to test peer operations
test_peer_operations() {
    local interface="$1"
    echo "🧪 Testing peer operations for $interface:"
    
    # Get a sample peer key (first one found)
    local peer_key=$(wg show "$interface" dump | tail -n +2 | head -n 1 | cut -f1)
    
    if [ -n "$peer_key" ]; then
        echo "   Found test peer: $peer_key"
        
        # Test removing and re-adding a peer
        echo "   Testing: Remove and re-add peer"
        if wg set "$interface" peer "$peer_key" remove; then
            echo "   ✅ Peer removed successfully"
            
            # Try to re-add with basic config
            if wg set "$interface" peer "$peer_key" allowed-ips 0.0.0.0/0; then
                echo "   ✅ Peer re-added successfully"
            else
                echo "   ❌ Failed to re-add peer"
            fi
        else
            echo "   ❌ Failed to remove peer"
        fi
    else
        echo "   ⚠️  No peers found to test with"
    fi
}

# Main execution
echo ""
echo "1️⃣  Checking WireGuard interface status..."
check_interface "wg0"

echo ""
echo "2️⃣  Showing current peers..."
show_peers "wg0"

echo ""
echo "3️⃣  Testing wg commands..."
test_wg_commands "wg0"

echo ""
echo "4️⃣  Testing peer operations..."
test_peer_operations "wg0"

echo ""
echo "5️⃣  Checking API endpoints..."
check_api

echo ""
echo "6️⃣  Checking logs..."
check_logs

echo ""
echo "🔍 Debug Summary:"
echo "=================="
echo "• Interface active: $(wg show wg0 >/dev/null 2>&1 && echo "Yes" || echo "No")"
echo "• Peer count: $(wg show wg0 dump | tail -n +2 | wc -l)"
echo "• Service status: $(systemctl is-active vwireguard 2>/dev/null || echo "Unknown")"
echo "• API accessible: $(curl -s http://localhost:5000/api/wg/status >/dev/null 2>&1 && echo "Yes" || echo "No")"

echo ""
echo "💡 Next steps:"
echo "1. Create a new client in the web interface"
echo "2. Click 'Apply Config'"
echo "3. Check logs: journalctl -u vwireguard -f"
echo "4. Verify peer appears: wg show wg0"
echo "5. Test connection from client" 