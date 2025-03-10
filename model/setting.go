package model

import (
	"time"
)

// GlobalSetting represents global settings
type GlobalSetting struct {
	EndpointAddress     string    `json:"endpoint_address"`
	DNSServers          []string  `json:"dns_servers"`
	MTU                 int       `json:"mtu,string"`
	PersistentKeepalive int       `json:"persistent_keepalive,string"`
	FirewallMark        string    `json:"firewall_mark"`
	Table               string    `json:"table"`
	ConfigFilePath      string    `json:"config_file_path"`
	UpdatedAt           time.Time `json:"updated_at"`

	// WARP settings
	WARPEnabled bool     `json:"warp_enabled"`
	WARPDomains []string `json:"warp_domains"`
	WARPExclude []string `json:"warp_exclude"` // Domains to exclude from WARP

	// Internal use
	Hash string `json:"-"`
}
