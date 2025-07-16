package service

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/MmadF14/vwireguard/model"
)

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
	// For V2Ray tunnels, we don't need a remote public key since we're creating a local WireGuard interface
	// that accepts client connections and routes traffic through V2Ray

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
			// For V2Ray tunnels, we need a peer to accept WireGuard client connections
			// The peer will accept any client (wildcard public key)
			"peers": []map[string]interface{}{
				{
					"publicKey":  "0000000000000000000000000000000000000000000000000000000000000000", // Accept any client
					"allowedIPs": []string{"0.0.0.0/0", "::/0"},
				},
			},
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
				// Route Iranian domains directly (bypass tunnel)
				map[string]interface{}{"type": "field", "inboundTag": []string{"wg-in"}, "domain": []string{"geosite:ir"}, "outboundTag": "direct"},
				// Route all other traffic through V2Ray
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
ExecStart=/usr/local/bin/xray -c /etc/vwireguard/tunnels/%%i.json
Restart=on-failure
[Install]
WantedBy=multi-user.target
`, tunnel.ID, tunnel.ID)
	if err := os.WriteFile(servicePath, []byte(serviceContent), 0644); err != nil {
		return err
	}
	exec.Command("systemctl", "daemon-reload").Run()
	return exec.Command("systemctl", "enable", "--now", fmt.Sprintf("vwireguard-tunnel-%s.service", tunnel.ID)).Run()
}
