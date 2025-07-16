package model

import (
	"time"
)

// TunnelType represents different types of tunnels
type TunnelType string

const (
	TunnelTypeWireGuardToWireGuard TunnelType = "wg-to-wg"
	TunnelTypeWireGuardToDokodemo  TunnelType = "wg-to-dokodemo"
	TunnelTypeWireGuardToOpenVPN   TunnelType = "wg-to-openvpn"
	TunnelTypeWireGuardToL2TP      TunnelType = "wg-to-l2tp"
	TunnelTypeWireGuardToSOCKS     TunnelType = "wg-to-socks"
	TunnelTypeWireGuardToHTTP      TunnelType = "wg-to-http"
	TunnelTypeWireGuardToV2ray     TunnelType = "wg-to-v2ray"
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

// Tunnel model - for routing WireGuard clients through different tunnel types
type Tunnel struct {
	ID          string       `json:"id"`
	Name        string       `json:"name"`
	Type        TunnelType   `json:"type"`
	Status      TunnelStatus `json:"status"`
	StatusColor string       `json:"status_color"`
	Description string       `json:"description"`

	// Client routing - which clients use this tunnel
	ClientIDs    []string `json:"client_ids"`    // Specific client IDs
	ClientGroups []string `json:"client_groups"` // Or client groups/tags
	RouteAll     bool     `json:"route_all"`     // Route all clients through this tunnel

	// WireGuard-to-WireGuard specific fields
	WGConfig *WireGuardTunnelConfig `json:"wg_config,omitempty"`

	// Dokodemo Door specific fields
	DokodemoConfig *DokodemoTunnelConfig `json:"dokodemo_config,omitempty"`

	// Port forward specific fields
	PortForwardConfig *PortForwardConfig `json:"port_forward_config,omitempty"`

	// WireGuard to V2Ray specific fields
	V2rayConfig *V2rayTunnelConfig `json:"v2ray_config,omitempty"`

	// Traffic statistics
	BytesIn  int64 `json:"bytes_in"`
	BytesOut int64 `json:"bytes_out"`

	// Management
	Enabled   bool   `json:"enabled"`
	Priority  int    `json:"priority"` // Higher priority tunnels are preferred
	CreatedBy string `json:"created_by"`

	// Timestamps
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	LastSeen  time.Time `json:"last_seen"`
}

// WireGuard tunnel configuration
type WireGuardTunnelConfig struct {
	// Remote WireGuard server details
	RemoteEndpoint  string `json:"remote_endpoint"`         // IP:Port of remote WG server
	RemotePublicKey string `json:"remote_public_key"`       // Remote server's public key
	LocalPrivateKey string `json:"local_private_key"`       // Our private key for this tunnel
	LocalPublicKey  string `json:"local_public_key"`        // Our public key (auto-generated)
	PreSharedKey    string `json:"preshared_key,omitempty"` // Optional pre-shared key

	// Network configuration
	TunnelIP            string   `json:"tunnel_ip"`     // Our IP in the tunnel network
	AllowedIPs          []string `json:"allowed_ips"`   // IPs to route through tunnel
	DNS                 []string `json:"dns,omitempty"` // DNS servers to use
	MTU                 int      `json:"mtu,omitempty"` // MTU size
	PersistentKeepalive int      `json:"persistent_keepalive,omitempty"`
}

// Dokodemo Door tunnel configuration
type DokodemoTunnelConfig struct {
	// Target address (where traffic will be forwarded)
	Address string `json:"address"` // Target IP/hostname
	Port    int    `json:"port"`    // Target port
	Network string `json:"network"` // "tcp", "udp", or "tcp,udp"

	// Timeout settings
	Timeout        int  `json:"timeout,omitempty"`         // Connection timeout in seconds
	UserLevel      int  `json:"user_level,omitempty"`      // User level for traffic stats
	FollowRedirect bool `json:"follow_redirect,omitempty"` // Follow HTTP redirects

	// Advanced settings
	DomainStrategy string            `json:"domain_strategy,omitempty"` // "AsIs", "UseIP", "UseIPv4", "UseIPv6"
	UserAgent      string            `json:"user_agent,omitempty"`      // Custom user agent
	Headers        map[string]string `json:"headers,omitempty"`         // Custom headers
}

// Port forward configuration (Dokodemo Door style)
type PortForwardConfig struct {
	// Protocol
	Protocol string `json:"protocol"` // "tcp", "udp", "both"

	// Local binding
	LocalBindIP   string `json:"local_bind_ip"`   // IP to bind locally
	LocalBindPort int    `json:"local_bind_port"` // Port to bind locally

	// Remote target
	RemoteHost string `json:"remote_host"` // Target host
	RemotePort int    `json:"remote_port"` // Target port

	// Advanced options
	Transparent    bool   `json:"transparent"`          // Transparent proxy mode
	FollowRedirect bool   `json:"follow_redirect"`      // Follow redirects
	UserAgent      string `json:"user_agent,omitempty"` // Custom user agent for HTTP
}

type V2rayTunnelConfig struct {
	Protocol      string   `json:"protocol"`           // vmess|vless|trojan
	RemoteAddress string   `json:"remote_address"`     // IP or domain
	RemotePort    int      `json:"remote_port"`        // e.g. 443
	UUID          string   `json:"uuid,omitempty"`     // VMess/VLESS
	Flow          string   `json:"flow,omitempty"`     // optional
	Password      string   `json:"password,omitempty"` // Trojan
	Security      string   `json:"security"`           // tls|reality|none
	ServerName    string   `json:"server_name,omitempty"`
	Fingerprint   string   `json:"fingerprint,omitempty"`
	Alpn          []string `json:"alpn,omitempty"`
	Network       string   `json:"network"`        // tcp|ws|grpc
	Path          string   `json:"path,omitempty"` // for ws/grpc
	SNI           string   `json:"sni,omitempty"`  // TLS SNI
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
