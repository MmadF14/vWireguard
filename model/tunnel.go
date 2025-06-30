package model

import (
	"time"
)

// TunnelType represents different types of tunnels
type TunnelType string

const (
	TunnelTypeWireGuardToWireGuard TunnelType = "wg-to-wg"
	TunnelTypeWireGuardToSSH       TunnelType = "wg-to-ssh"
	TunnelTypeWireGuardToOpenVPN   TunnelType = "wg-to-openvpn"
	TunnelTypeWireGuardToL2TP      TunnelType = "wg-to-l2tp"
	TunnelTypeWireGuardToSOCKS     TunnelType = "wg-to-socks"
	TunnelTypeWireGuardToHTTP      TunnelType = "wg-to-http"
	TunnelTypePortForward          TunnelType = "port-forward"
	TunnelTypeReverse              TunnelType = "reverse"
)

// TunnelStatus represents the status of a tunnel
type TunnelStatus string

const (
	TunnelStatusActive   TunnelStatus = "active"
	TunnelStatusInactive TunnelStatus = "inactive"
	TunnelStatusError    TunnelStatus = "error"
)

// Tunnel model
type Tunnel struct {
	ID          string       `json:"id"`
	Name        string       `json:"name"`
	Type        TunnelType   `json:"type"`
	Status      TunnelStatus `json:"status"`
	Description string       `json:"description"`

	// Source configuration
	SourceInterface string `json:"source_interface"`
	SourceIP        string `json:"source_ip"`
	SourcePort      int    `json:"source_port"`

	// Destination configuration
	DestinationIP   string `json:"destination_ip"`
	DestinationPort int    `json:"destination_port"`

	// Tunnel specific configurations
	Config map[string]interface{} `json:"config"`

	// Authentication for certain tunnel types
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
	KeyFile  string `json:"key_file,omitempty"`

	// Traffic statistics
	BytesIn  int64 `json:"bytes_in"`
	BytesOut int64 `json:"bytes_out"`

	// Management
	Enabled   bool   `json:"enabled"`
	CreatedBy string `json:"created_by"`

	// Timestamps
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	LastSeen  time.Time `json:"last_seen"`
}

// TunnelConfig represents configuration for different tunnel types
type TunnelConfig struct {
	// WireGuard specific
	WGPrivateKey string `json:"wg_private_key,omitempty"`
	WGPublicKey  string `json:"wg_public_key,omitempty"`
	WGEndpoint   string `json:"wg_endpoint,omitempty"`

	// SSH specific
	SSHUser    string `json:"ssh_user,omitempty"`
	SSHKeyPath string `json:"ssh_key_path,omitempty"`
	SSHHost    string `json:"ssh_host,omitempty"`
	SSHPort    int    `json:"ssh_port,omitempty"`

	// OpenVPN specific
	OpenVPNConfig string `json:"openvpn_config,omitempty"`
	OpenVPNCert   string `json:"openvpn_cert,omitempty"`
	OpenVPNKey    string `json:"openvpn_key,omitempty"`

	// SOCKS specific
	SOCKSVersion int  `json:"socks_version,omitempty"`
	SOCKSAuth    bool `json:"socks_auth,omitempty"`

	// HTTP specific
	HTTPSEnabled bool              `json:"https_enabled,omitempty"`
	HTTPHeaders  map[string]string `json:"http_headers,omitempty"`

	// Port forwarding specific
	Protocol string `json:"protocol,omitempty"` // tcp, udp, both

	// Additional options
	Persistent  bool   `json:"persistent,omitempty"`
	AutoRestart bool   `json:"auto_restart,omitempty"`
	HealthCheck string `json:"health_check,omitempty"`
	LogLevel    string `json:"log_level,omitempty"`
}
