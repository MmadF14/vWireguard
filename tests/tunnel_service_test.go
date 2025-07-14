package tests

import (
	"strings"
	"testing"

	"github.com/MmadF14/vwireguard/model"
	"github.com/MmadF14/vwireguard/service"
)

func TestGenerateXrayConfig(t *testing.T) {
	tunnel := &model.Tunnel{
		ID: "test",
		WGConfig: &model.WireGuardTunnelConfig{
			TunnelIP:        "10.0.0.2",
			LocalPrivateKey: "priv",
			RemotePublicKey: "pub",
		},
		V2rayConfig: &model.V2rayTunnelConfig{
			Protocol:      "vmess",
			RemoteAddress: "example.com",
			RemotePort:    443,
			UUID:          "abcd",
			Security:      "tls",
			Network:       "tcp",
		},
	}
	cfg, err := service.GenerateXrayConfig(tunnel)
	if err != nil {
		t.Fatalf("generate error: %v", err)
	}
	if !strings.Contains(cfg, "\"protocol\": \"vmess\"") {
		t.Fatalf("protocol not present in config: %s", cfg)
	}
	if !strings.Contains(cfg, "example.com") {
		t.Fatalf("remote address missing")
	}
}
