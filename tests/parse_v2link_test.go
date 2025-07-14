package tests

import (
	"encoding/base64"
	"testing"

	"github.com/MmadF14/vwireguard/handler"
)

func TestParseV2Link_VMess(t *testing.T) {
	data := `{"add":"example.com","port":"443","id":"abcd","net":"tcp"}`
	link := "vmess://" + base64.StdEncoding.EncodeToString([]byte(data))
	cfg, err := handler.ParseV2LinkString(link)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Protocol != "vmess" || cfg.UUID != "abcd" || cfg.RemoteAddress != "example.com" || cfg.RemotePort != 443 {
		t.Fatalf("parsed data incorrect: %+v", cfg)
	}
}

func TestParseV2Link_VLESS(t *testing.T) {
	link := "vless://abcd@example.com:8443?type=ws"
	cfg, err := handler.ParseV2LinkString(link)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Protocol != "vless" || cfg.UUID != "abcd" || cfg.RemoteAddress != "example.com" || cfg.RemotePort != 8443 {
		t.Fatalf("parsed data incorrect: %+v", cfg)
	}
}

func TestParseV2Link_Trojan(t *testing.T) {
	link := "trojan://pass@example.com:443"
	cfg, err := handler.ParseV2LinkString(link)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Protocol != "trojan" || cfg.Password != "pass" || cfg.RemoteAddress != "example.com" || cfg.RemotePort != 443 {
		t.Fatalf("parsed data incorrect: %+v", cfg)
	}
}
