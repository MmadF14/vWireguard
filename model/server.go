package model

import (
	"time"
)

// Server represents the WireGuard server configuration
type Server struct {
	Interface ServerInterface
	KeyPair   ServerKeypair
}

// ServerKeypair represents the WireGuard server key pair
type ServerKeypair struct {
	PrivateKey string    `json:"private_key"`
	PublicKey  string    `json:"public_key"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// ServerInterface represents the WireGuard server interface configuration
type ServerInterface struct {
	Addresses   []string  `json:"addresses"`
	ListenPort  int       `json:"listen_port,string"` // ,string to get listen_port string input as int
	UpdatedAt   time.Time `json:"updated_at"`
	PostUp      string    `json:"post_up"`
	PreDown     string    `json:"pre_down"`
	PostDown    string    `json:"post_down"`
	CheckInterval int    `json:"check_interval"`
}

