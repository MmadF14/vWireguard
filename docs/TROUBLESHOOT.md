# Troubleshooting

## No packets passing
If your clients connect but cannot access the Internet, make sure NAT rules are present and the MTU is set correctly.

1. Check that iptables has a MASQUERADE rule for your tunnel subnet:
   ```bash
   iptables -t nat -A POSTROUTING -s <WG_SUBNET> -o <OUT_IFACE> -j MASQUERADE
   ```
2. Enable IP forwarding:
   ```bash
   sysctl -w net.ipv4.ip_forward=1
   sysctl -w net.ipv6.conf.all.forwarding=1
   ```
3. On clients use MTU 1420 or lower to avoid fragmentation issues.
