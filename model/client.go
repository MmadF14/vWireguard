package model

import (
	"time"
)

// Client represents a WireGuard client configuration
type Client struct {
	ID            int      `json:"id"`
	Name          string   `json:"name"`
	Email         string   `json:"email"`
	PrivateKey    string   `json:"private_key"`
	PublicKey     string   `json:"public_key"`
	Address       string   `json:"address"`
	AllowedIPs    []string `json:"allowed_ips"`
	ExtraAllowedIPs []string `json:"extra_allowed_ips"`
	UseServerDNS  bool     `json:"use_server_dns"`
	Enabled       bool     `json:"enabled"`
	TgUserid      string   `json:"tg_userid"`
}

// ClientData represents client data with additional information
type ClientData struct {
	Client     Client
	QRCode     string
	ConfigFile string
}

type QRCodeSettings struct {
	Enabled    bool
	IncludeDNS bool
	IncludeMTU bool
}
