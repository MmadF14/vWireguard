package model

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

// ---------------------------------------------------------------------------
//  Multi-interface support
//
//  The panel originally modelled exactly ONE WireGuard interface: model.Server
//  was a single record (server/interfaces.json + server/keypair.json), so you
//  could not run wg0 for direct users, wg1 for relayed users and wg2 for free
//  users on the same node.
//
//  WGInterface is a full, self-contained description of one interface. The
//  legacy Server record is migrated into an interface named DefaultInterfaceName
//  on first run, so existing installs keep working untouched.
// ---------------------------------------------------------------------------

// DefaultInterfaceName is the interface a legacy install is migrated into, and
// the one a client falls back to when it has no explicit assignment.
const DefaultInterfaceName = "wg0"

// interfaceNameRe matches names Linux will actually accept for a netdev.
var interfaceNameRe = regexp.MustCompile(`^[a-zA-Z0-9_-]{1,15}$`)

// WGInterface describes a single WireGuard interface managed by the panel.
type WGInterface struct {
	// Name is the netdev name AND the config file stem: /etc/wireguard/<Name>.conf
	Name string `json:"name"`

	// Description is free text shown in the UI.
	Description string `json:"description,omitempty"`

	// Enabled interfaces are brought up and written out; disabled ones are kept
	// in the database but skipped.
	Enabled bool `json:"enabled"`

	PrivateKey string `json:"private_key"`
	PublicKey  string `json:"public_key"`

	// Addresses are the interface's own CIDRs, e.g. ["10.66.0.1/24"].
	Addresses  []string `json:"addresses"`
	ListenPort int      `json:"listen_port"`

	// MTU of 0 means "let the template decide".
	MTU int `json:"mtu,omitempty"`

	PostUp   string `json:"post_up,omitempty"`
	PreDown  string `json:"pre_down,omitempty"`
	PostDown string `json:"post_down,omitempty"`

	// Endpoint that clients on this interface should dial. Empty falls back to
	// the global setting. This is what makes per-interface routing useful: wg1
	// can hand out a relay endpoint while wg0 hands out the direct one.
	Endpoint string `json:"endpoint,omitempty"`

	// DNSServers overrides the global DNS for clients on this interface.
	DNSServers []string `json:"dns_servers,omitempty"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Validate checks an interface definition before it is persisted.
func (w *WGInterface) Validate() error {
	name := strings.TrimSpace(w.Name)
	if !interfaceNameRe.MatchString(name) {
		return fmt.Errorf("interface name %q is invalid: use 1-15 chars of a-z, A-Z, 0-9, - or _", w.Name)
	}
	if w.ListenPort <= 0 || w.ListenPort > 65535 {
		return fmt.Errorf("listen port %d is out of range", w.ListenPort)
	}
	if len(w.Addresses) == 0 {
		return fmt.Errorf("interface %s needs at least one address", name)
	}
	if w.PrivateKey == "" {
		return fmt.Errorf("interface %s has no private key", name)
	}
	return nil
}

// ConfigFilePath is where wg-quick expects this interface's config.
func (w *WGInterface) ConfigFilePath() string {
	return "/etc/wireguard/" + w.Name + ".conf"
}

// ServiceName is the systemd unit that carries this interface.
func (w *WGInterface) ServiceName() string {
	return "wg-quick@" + w.Name
}

// InterfaceOf returns the interface a client belongs to, falling back to the
// default when the client predates multi-interface support.
func InterfaceOf(c *Client) string {
	if c == nil || strings.TrimSpace(c.Interface) == "" {
		return DefaultInterfaceName
	}
	return strings.TrimSpace(c.Interface)
}

// ToLegacyServer projects an interface back onto the old single-server shape.
// Everything that still expects model.Server keeps working through this.
func (w *WGInterface) ToLegacyServer() Server {
	return Server{
		KeyPair: &ServerKeypair{
			PrivateKey: w.PrivateKey,
			PublicKey:  w.PublicKey,
			UpdatedAt:  w.UpdatedAt,
		},
		Interface: &ServerInterface{
			Addresses:  w.Addresses,
			ListenPort: w.ListenPort,
			UpdatedAt:  w.UpdatedAt,
			PostUp:     w.PostUp,
			PreDown:    w.PreDown,
			PostDown:   w.PostDown,
		},
	}
}

// WGInterfaceFromLegacy builds the default interface from the old Server record.
// Used once, during migration.
func WGInterfaceFromLegacy(s Server, name string) *WGInterface {
	if name == "" {
		name = DefaultInterfaceName
	}
	w := &WGInterface{
		Name:        name,
		Description: "Migrated from the original single-interface configuration",
		Enabled:     true,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}
	if s.KeyPair != nil {
		w.PrivateKey = s.KeyPair.PrivateKey
		w.PublicKey = s.KeyPair.PublicKey
	}
	if s.Interface != nil {
		w.Addresses = s.Interface.Addresses
		w.ListenPort = s.Interface.ListenPort
		w.PostUp = s.Interface.PostUp
		w.PreDown = s.Interface.PreDown
		w.PostDown = s.Interface.PostDown
	}
	return w
}
