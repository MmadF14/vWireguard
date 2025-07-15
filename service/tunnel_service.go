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
	if tunnel == nil || tunnel.WGConfig == nil || tunnel.V2rayConfig == nil {
		return "", fmt.Errorf("incomplete tunnel configuration")
	}

	inb := map[string]interface{}{
		"tag":      "wg-in",
		"protocol": "wireguard",
		"settings": map[string]interface{}{
			"address":    []string{fmt.Sprintf("%s/32", tunnel.WGConfig.TunnelIP)},
			"privateKey": tunnel.WGConfig.LocalPrivateKey,
			"peers": []map[string]interface{}{
				{
					"publicKey":  tunnel.WGConfig.RemotePublicKey,
					"allowedIPs": []string{"0.0.0.0/0", "::/0"},
				},
			},
		},
	}

	vc := tunnel.V2rayConfig
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
