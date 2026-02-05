package model

import (
	"time"
)

// Client model
type Client struct {
	ID               string    `json:"id"`
	PrivateKey       string    `json:"private_key"`
	PublicKey        string    `json:"public_key"`
	PresharedKey     string    `json:"preshared_key"`
	Name             string    `json:"name"`
	TgUserid         string    `json:"telegram_userid"`
	Email            string    `json:"email"`
	SubnetRanges     []string  `json:"subnet_ranges,omitempty"`
	AllocatedIPs     []string  `json:"allocated_ips"`
	AllowedIPs       []string  `json:"allowed_ips"`
	ExtraAllowedIPs  []string  `json:"extra_allowed_ips"`
	Endpoint         string    `json:"endpoint"`
	AdditionalNotes  string    `json:"additional_notes"`
	UseServerDNS     bool      `json:"use_server_dns"`
	Enabled          bool      `json:"enabled"`
	CreatedBy        string    `json:"created_by,omitempty"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
	Quota            int64     `json:"quota,omitempty"`
	UsedQuota        int64     `json:"used_quota,omitempty"`
	Expiration       time.Time `json:"expiration,omitempty"`
	ExpirationDays   int       `json:"expiration_days,omitempty"`
	FirstConnectedAt time.Time `json:"first_connected_at,omitempty"`
	Status           string    `json:"status,omitempty"`
	LastHandshake    time.Time `json:"last_handshake,omitempty"`
	// Persistent storage for usage data
	PersistentUsageData *ClientUsageData `json:"persistent_usage_data,omitempty"`
}

// ClientUsageData stores persistent usage information
type ClientUsageData struct {
	TotalBytesReceived         uint64    `json:"total_bytes_received"`
	TotalBytesSent             uint64    `json:"total_bytes_sent"`
	LastInterfaceBytesReceived uint64    `json:"last_interface_bytes_received"`
	LastInterfaceBytesSent     uint64    `json:"last_interface_bytes_sent"`
	LastSeen                   time.Time `json:"last_seen"`
	FirstSeen                  time.Time `json:"first_seen"`
	UpdatedAt                  time.Time `json:"updated_at"`
}

// ClientData includes the Client and extra data
type ClientData struct {
	Client *Client
	QRCode string
}

type QRCodeSettings struct {
	Enabled    bool
	IncludeDNS bool
	IncludeMTU bool
}
