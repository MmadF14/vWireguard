package service

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/MmadF14/vwireguard/model"
	"github.com/MmadF14/vwireguard/store"
)

// runSystemctl runs systemctl commands with proper privileges
func runSystemctl(args ...string) error {
	// Try with sudo first
	cmd := exec.Command("sudo", append([]string{"systemctl"}, args...)...)
	if err := cmd.Run(); err != nil {
		// If sudo fails, try without sudo (in case we're already root)
		cmd = exec.Command("systemctl", args...)
		return cmd.Run()
	}
	return nil
}

// GenerateXrayConfig builds an Xray config for WireGuard->V2Ray tunnels
func GenerateXrayConfig(tunnel *model.Tunnel) (string, error) {
	if tunnel == nil {
		return "", fmt.Errorf("tunnel is nil")
	}
	if tunnel.WGConfig == nil {
		return "", fmt.Errorf("WireGuard configuration is missing")
	}
	if tunnel.V2rayConfig == nil {
		return "", fmt.Errorf("V2Ray configuration is missing")
	}

	// Validate WireGuard configuration
	if tunnel.WGConfig.TunnelIP == "" {
		return "", fmt.Errorf("WireGuard tunnel IP is missing")
	}
	if tunnel.WGConfig.LocalPrivateKey == "" {
		return "", fmt.Errorf("WireGuard local private key is missing")
	}
	// For V2Ray tunnels, we don't need a remote public key since we're not connecting to a WireGuard peer
	// The WireGuard interface is just for local traffic routing

	// Validate V2Ray configuration
	vc := tunnel.V2rayConfig
	if vc.Protocol == "" {
		return "", fmt.Errorf("V2Ray protocol is missing")
	}
	if vc.RemoteAddress == "" {
		return "", fmt.Errorf("V2Ray remote address is missing")
	}
	if vc.RemotePort == 0 {
		return "", fmt.Errorf("V2Ray remote port is missing")
	}
	if vc.Security == "" {
		return "", fmt.Errorf("V2Ray security setting is missing")
	}
	if vc.Network == "" {
		return "", fmt.Errorf("V2Ray network type is missing")
	}

	// Protocol-specific validation
	switch vc.Protocol {
	case "vmess", "vless":
		if vc.UUID == "" {
			return "", fmt.Errorf("UUID is required for %s protocol", vc.Protocol)
		}
	case "trojan":
		if vc.Password == "" {
			return "", fmt.Errorf("Password is required for Trojan protocol")
		}
	default:
		return "", fmt.Errorf("Unsupported V2Ray protocol: %s", vc.Protocol)
	}

	inb := map[string]interface{}{
		"tag":      "wg-in",
		"protocol": "wireguard",
		"settings": map[string]interface{}{
			"address":    []string{fmt.Sprintf("%s/32", tunnel.WGConfig.TunnelIP)},
			"privateKey": tunnel.WGConfig.LocalPrivateKey,
			// For V2Ray tunnels, we don't need peers since we're not connecting to a WireGuard server
			// The WireGuard interface is just for local traffic routing
		},
	}

	ob := map[string]interface{}{
		"tag":      "v2-out",
		"protocol": vc.Protocol,
	}

	switch vc.Protocol {
	case "vmess", "vless":
		user := map[string]interface{}{"id": vc.UUID}
		if vc.Flow != "" {
			user["flow"] = vc.Flow
		}
		ob["settings"] = map[string]interface{}{
			"vnext": []map[string]interface{}{
				{
					"address": vc.RemoteAddress,
					"port":    vc.RemotePort,
					"users":   []map[string]interface{}{user},
				},
			},
		}
	case "trojan":
		ob["settings"] = map[string]interface{}{
			"servers": []map[string]interface{}{
				{
					"address":  vc.RemoteAddress,
					"port":     vc.RemotePort,
					"password": vc.Password,
				},
			},
		}
	}

	stream := map[string]interface{}{
		"network":  vc.Network,
		"security": vc.Security,
	}
	if vc.Security != "none" {
		tlsCfg := map[string]interface{}{}
		if vc.ServerName != "" {
			tlsCfg["serverName"] = vc.ServerName
		}
		if vc.SNI != "" {
			tlsCfg["serverName"] = vc.SNI
		}
		if len(vc.Alpn) > 0 {
			tlsCfg["alpn"] = vc.Alpn
		}
		if vc.Fingerprint != "" {
			tlsCfg["fingerprint"] = vc.Fingerprint
		}
		stream["tlsSettings"] = tlsCfg
	}
	if vc.Network == "ws" {
		stream["wsSettings"] = map[string]interface{}{"path": vc.Path}
	} else if vc.Network == "grpc" {
		stream["grpcSettings"] = map[string]interface{}{"serviceName": vc.Path}
	}
	ob["streamSettings"] = stream

	cfg := map[string]interface{}{
		"inbounds":  []interface{}{inb},
		"outbounds": []interface{}{ob, map[string]interface{}{"tag": "direct", "protocol": "freedom"}},
		"routing": map[string]interface{}{
			"rules": []interface{}{
				map[string]interface{}{"type": "field", "inboundTag": []string{"wg-in"}, "domain": []string{"geosite:ir"}, "outboundTag": "direct"},
				map[string]interface{}{"type": "field", "inboundTag": []string{"wg-in"}, "outboundTag": "v2-out"},
			},
		},
	}

	b, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// WriteConfigAndService writes the config file and systemd service
func WriteConfigAndService(tunnel *model.Tunnel, config string) error {
	cfgPath := filepath.Join("/etc/vwireguard/tunnels", fmt.Sprintf("%s.json", tunnel.ID))
	if err := os.MkdirAll(filepath.Dir(cfgPath), 0755); err != nil {
		return err
	}
	if err := os.WriteFile(cfgPath, []byte(config), 0644); err != nil {
		return err
	}

	servicePath := filepath.Join("/etc/systemd/system", fmt.Sprintf("vwireguard-tunnel-%s.service", tunnel.ID))
	serviceContent := fmt.Sprintf(`[Unit]
Description=vWireguard V2Ray Tunnel %s
After=network-online.target
[Service]
ExecStart=/usr/local/bin/xray -c /etc/vwireguard/tunnels/%s.json
Restart=on-failure
[Install]
WantedBy=multi-user.target
`, tunnel.ID, tunnel.ID)
	if err := os.WriteFile(servicePath, []byte(serviceContent), 0644); err != nil {
		return err
	}
	runSystemctl("daemon-reload")
	if err := runSystemctl("enable", "--now", fmt.Sprintf("vwireguard-tunnel-%s.service", tunnel.ID)); err != nil {
		return err
	}

	// Setup NAT rules for the tunnel network
	if tunnel.WGConfig != nil {
		subnet := tunnel.WGConfig.TunnelIP + "/24"
		outIface := "eth0"
		if tunnel.V2rayConfig != nil {
			remote := tunnel.V2rayConfig.RemoteAddress
			if idx := strings.Index(remote, ":"); idx > 0 {
				remote = remote[:idx]
			}
			if routeOut, err := exec.Command("sh", "-c", fmt.Sprintf("ip route get %s | awk '{for(i=1;i<NF;i++){if($i==\"dev\"){print $(i+1);exit}}}'", remote)).Output(); err == nil {
				outIface = strings.TrimSpace(string(routeOut))
			}
		}
		exec.Command("sysctl", "-w", "net.ipv4.ip_forward=1").Run()
		exec.Command("sysctl", "-w", "net.ipv6.conf.all.forwarding=1").Run()
		exec.Command("iptables", "-t", "nat", "-A", "POSTROUTING", "-s", subnet, "-o", outIface, "-j", "MASQUERADE").Run()
	}
	return nil
}

// StartTunnel starts the systemd service for the given tunnel ID. If the tunnel
// has RouteAll enabled, any other active RouteAll tunnels will be stopped first.
func StartTunnel(db store.IStore, id string) error {
	tunnel, err := db.GetTunnelByID(id)
	if err != nil {
		return err
	}

	if tunnel.RouteAll {
		tunnels, err := db.GetTunnels()
		if err == nil {
			for _, t := range tunnels {
				if t.ID != tunnel.ID && t.RouteAll && t.Status == model.TunnelStatusActive {
					runSystemctl("stop", fmt.Sprintf("vwireguard-tunnel-%s.service", t.ID))
					t.Status = model.TunnelStatusInactive
					db.SaveTunnel(t)
				}
			}
		}
	}

	// For V2Ray tunnels, ensure configuration and service files exist
	if tunnel.Type == model.TunnelTypeWireGuardToV2ray {
		// Check if config file exists
		cfgPath := filepath.Join("/etc/vwireguard/tunnels", fmt.Sprintf("%s.json", tunnel.ID))
		servicePath := filepath.Join("/etc/systemd/system", fmt.Sprintf("vwireguard-tunnel-%s.service", tunnel.ID))

		// If either file doesn't exist, recreate them
		if _, err := os.Stat(cfgPath); os.IsNotExist(err) {
			// Config file doesn't exist, recreate both files
			log.Printf("V2Ray tunnel configuration or service file missing, recreating for tunnel %s", tunnel.ID)

			// Generate V2Ray configuration
			cfg, err := GenerateXrayConfig(&tunnel)
			if err != nil {
				return fmt.Errorf("failed to generate V2Ray configuration: %v", err)
			}

			// Write configuration and service files
			if err := WriteConfigAndService(&tunnel, cfg); err != nil {
				return fmt.Errorf("failed to write V2Ray configuration and service: %v", err)
			}
		} else if _, err := os.Stat(servicePath); os.IsNotExist(err) {
			// Service file doesn't exist, recreate both files
			log.Printf("V2Ray tunnel service file missing, recreating for tunnel %s", tunnel.ID)

			// Generate V2Ray configuration
			cfg, err := GenerateXrayConfig(&tunnel)
			if err != nil {
				return fmt.Errorf("failed to generate V2Ray configuration: %v", err)
			}

			// Write configuration and service files
			if err := WriteConfigAndService(&tunnel, cfg); err != nil {
				return fmt.Errorf("failed to write V2Ray configuration and service: %v", err)
			}
		}
	}

	if os.Getenv("VWIREGUARD_TEST") != "1" {
		if err := runSystemctl("start", fmt.Sprintf("vwireguard-tunnel-%s.service", id)); err != nil {
			return err
		}
	}

	tunnel.Status = model.TunnelStatusActive
	return db.SaveTunnel(tunnel)
}

// StopTunnel stops the systemd service for the given tunnel ID and marks it inactive.
func StopTunnel(db store.IStore, id string) error {
	tunnel, err := db.GetTunnelByID(id)
	if err != nil {
		return err
	}

	if os.Getenv("VWIREGUARD_TEST") != "1" {
		runSystemctl("stop", fmt.Sprintf("vwireguard-tunnel-%s.service", id))
	}

	tunnel.Status = model.TunnelStatusInactive
	return db.SaveTunnel(tunnel)
}
