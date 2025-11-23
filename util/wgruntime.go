package util

import (
	"bufio"
	"bytes"
	"fmt"
	"os/exec"
	"strconv"
	"strings"

	"net"
	"time"

	"github.com/MmadF14/vwireguard/model"
	"golang.zx2c4.com/wireguard/wgctrl"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

// PeerState represents relevant runtime state of a WireGuard peer
type PeerState struct {
	PublicKey           string
	AllowedIPs          []string
	PresharedKey        string
	Endpoint            string
	PersistentKeepalive int
}

// PeerDiff represents an action required to sync peer state
type PeerDiff struct {
	Action string // add, remove, update
	Client *model.Client
	Key    string
}

// getCurrentPeers reads current peers from `wg show <iface> dump`
func getCurrentPeers(interfaceName string) (map[string]PeerState, error) {
	cmd := exec.Command("wg", "show", interfaceName, "dump")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("wg show failed: %v", err)
	}
	peers := make(map[string]PeerState)
	scanner := bufio.NewScanner(bytes.NewReader(out))
	if !scanner.Scan() { // interface line
		return peers, nil
	}
	for scanner.Scan() {
		f := strings.Fields(scanner.Text())
		if len(f) < 8 {
			continue
		}
		key := f[0]
		psk := f[1]
		endpoint := f[2]
		allowed := f[3]
		keep := f[7]
		allowedList := []string{}
		if allowed != "(none)" {
			allowedList = strings.Split(allowed, ",")
		}
		ka := 0
		if keep != "off" {
			if v, err := strconv.Atoi(keep); err == nil {
				ka = v
			}
		}
		if psk == "(none)" {
			psk = ""
		}
		if endpoint == "(none)" {
			endpoint = ""
		}
		peers[key] = PeerState{
			PublicKey:           key,
			AllowedIPs:          allowedList,
			PresharedKey:        psk,
			Endpoint:            endpoint,
			PersistentKeepalive: ka,
		}
	}
	return peers, nil
}

// buildTargetPeer builds a PeerState from a client
func buildTargetPeer(cl *model.Client, settings model.GlobalSetting) PeerState {
	allowed := append([]string{}, cl.AllocatedIPs...)
	allowed = append(allowed, cl.ExtraAllowedIPs...)
	if len(allowed) == 0 {
		allowed = []string{"0.0.0.0/0"}
	}
	keep := 0
	if settings.PersistentKeepalive > 0 {
		keep = settings.PersistentKeepalive
	}
	return PeerState{
		PublicKey:           cl.PublicKey,
		AllowedIPs:          allowed,
		PresharedKey:        cl.PresharedKey,
		Endpoint:            cl.Endpoint,
		PersistentKeepalive: keep,
	}
}

func equalPeer(a PeerState, b PeerState) bool {
	if a.PresharedKey != b.PresharedKey || a.Endpoint != b.Endpoint || a.PersistentKeepalive != b.PersistentKeepalive {
		return false
	}
	if len(a.AllowedIPs) != len(b.AllowedIPs) {
		return false
	}
	for i := range a.AllowedIPs {
		if a.AllowedIPs[i] != b.AllowedIPs[i] {
			return false
		}
	}
	return true
}

func findClientByKey(clients []model.ClientData, key string) *model.Client {
	for _, cd := range clients {
		if cd.Client != nil && cd.Client.PublicKey == key {
			return cd.Client
		}
	}
	return &model.Client{PublicKey: key}
}

// BuildPeerConfig converts a client entry into a wgtypes.PeerConfig for runtime updates
func BuildPeerConfig(cl *model.Client, settings model.GlobalSetting) (wgtypes.PeerConfig, error) {
	pubKey, err := wgtypes.ParseKey(cl.PublicKey)
	if err != nil {
		return wgtypes.PeerConfig{}, err
	}

	var psk *wgtypes.Key
	if cl.PresharedKey != "" {
		key, err := wgtypes.ParseKey(cl.PresharedKey)
		if err != nil {
			return wgtypes.PeerConfig{}, err
		}
		psk = &key
	}

	var allowedIPs []net.IPNet
	for _, ipStr := range append(append([]string{}, cl.AllocatedIPs...), cl.ExtraAllowedIPs...) {
		if ipStr == "" {
			continue
		}
		_, ipNet, err := net.ParseCIDR(ipStr)
		if err != nil {
			continue
		}
		allowedIPs = append(allowedIPs, *ipNet)
	}
	if len(allowedIPs) == 0 {
		if _, ipNet, err := net.ParseCIDR("0.0.0.0/0"); err == nil {
			allowedIPs = append(allowedIPs, *ipNet)
		}
	}

	var endpoint *net.UDPAddr
	if cl.Endpoint != "" {
		if ep, err := net.ResolveUDPAddr("udp", cl.Endpoint); err == nil {
			endpoint = ep
		}
	}

	var keepalive *time.Duration
	if settings.PersistentKeepalive > 0 {
		d := time.Duration(settings.PersistentKeepalive) * time.Second
		keepalive = &d
	}

	pc := wgtypes.PeerConfig{
		PublicKey:                   pubKey,
		PresharedKey:                psk,
		Endpoint:                    endpoint,
		PersistentKeepaliveInterval: keepalive,
		ReplaceAllowedIPs:           true,
		AllowedIPs:                  allowedIPs,
	}

	return pc, nil
}

// ComputePeerDiffs compares current interface state with target client list
func ComputePeerDiffs(interfaceName string, clients []model.ClientData, settings model.GlobalSetting) ([]PeerDiff, error) {
	current, err := getCurrentPeers(interfaceName)
	if err != nil {
		return nil, err
	}
	target := make(map[string]PeerState)
	for _, cd := range clients {
		if cd.Client == nil || !cd.Client.Enabled {
			continue
		}
		p := buildTargetPeer(cd.Client, settings)
		target[p.PublicKey] = p
	}

	var diffs []PeerDiff
	for key, t := range target {
		cState, ok := current[key]
		if !ok {
			cl := findClientByKey(clients, key)
			diffs = append(diffs, PeerDiff{Action: "add", Client: cl, Key: key})
		} else if !equalPeer(cState, t) {
			cl := findClientByKey(clients, key)
			diffs = append(diffs, PeerDiff{Action: "update", Client: cl, Key: key})
		}
	}
	for key := range current {
		if _, ok := target[key]; !ok {
			diffs = append(diffs, PeerDiff{Action: "remove", Key: key})
		}
	}
	return diffs, nil
}

// ApplyPeerDiffs applies the given diffs using wgctrl
func ApplyPeerDiffs(interfaceName string, diffs []PeerDiff, settings model.GlobalSetting) error {
	wgClient, err := wgctrl.New()
	if err != nil {
		return err
	}
	defer wgClient.Close()

	for _, diff := range diffs {
		switch diff.Action {
		case "add", "update":
			if diff.Client == nil {
				continue
			}
			pc, err := BuildPeerConfig(diff.Client, settings)
			if err != nil {
				return err
			}
			if diff.Action == "update" {
				pc.UpdateOnly = true
			}
			if err := wgClient.ConfigureDevice(interfaceName, wgtypes.Config{Peers: []wgtypes.PeerConfig{pc}}); err != nil {
				return err
			}
		case "remove":
			key, err := wgtypes.ParseKey(diff.Key)
			if err != nil {
				continue
			}
			pc := wgtypes.PeerConfig{PublicKey: key, Remove: true}
			if err := wgClient.ConfigureDevice(interfaceName, wgtypes.Config{Peers: []wgtypes.PeerConfig{pc}}); err != nil {
				return err
			}
		}
	}
	return nil
}

// GetInterfaceNameFromConfig extracts the interface name from the config file path
// Defaults to "wg0" if not found
func GetInterfaceNameFromConfig(configFilePath string) string {
	if configFilePath == "" {
		return "wg0"
	}
	parts := strings.Split(configFilePath, "/")
	if len(parts) > 0 {
		baseName := parts[len(parts)-1]
		return strings.TrimSuffix(baseName, ".conf")
	}
	return "wg0"
}

// AddPeerToInterface adds or updates a peer on the WireGuard interface using wgctrl
// This is a hot-reload operation that doesn't restart the service
func AddPeerToInterface(client model.Client, server model.Server, settings model.GlobalSetting, interfaceName string) error {
	if interfaceName == "" {
		interfaceName = GetInterfaceNameFromConfig(settings.ConfigFilePath)
	}

	wgClient, err := wgctrl.New()
	if err != nil {
		return fmt.Errorf("failed to create wgctrl client: %v", err)
	}
	defer wgClient.Close()

	pc, err := BuildPeerConfig(&client, settings)
	if err != nil {
		return fmt.Errorf("failed to build peer config: %v", err)
	}

	// Note: By default, ConfigureDevice only updates the specified peers without replacing all peers
	// This is the append mode behavior we want - it won't disrupt other active peers
	config := wgtypes.Config{
		ReplacePeers: false, // Don't replace all peers, just add/update this one
		Peers:        []wgtypes.PeerConfig{pc},
	}

	if err := wgClient.ConfigureDevice(interfaceName, config); err != nil {
		return fmt.Errorf("failed to configure device: %v", err)
	}

	return nil
}

// RemovePeerFromInterface removes a peer from the WireGuard interface using wgctrl
// This instantly disconnects the user without restarting the service
func RemovePeerFromInterface(publicKey string, interfaceName string) error {
	if interfaceName == "" {
		interfaceName = "wg0"
	}

	wgClient, err := wgctrl.New()
	if err != nil {
		return fmt.Errorf("failed to create wgctrl client: %v", err)
	}
	defer wgClient.Close()

	key, err := wgtypes.ParseKey(publicKey)
	if err != nil {
		return fmt.Errorf("failed to parse public key: %v", err)
	}

	// Crucial: Set Remove to true to instantly kick the user
	pc := wgtypes.PeerConfig{
		PublicKey: key,
		Remove:    true,
	}

	config := wgtypes.Config{
		Peers: []wgtypes.PeerConfig{pc},
	}

	if err := wgClient.ConfigureDevice(interfaceName, config); err != nil {
		return fmt.Errorf("failed to remove peer: %v", err)
	}

	return nil
}

// UpdatePeerOnInterface updates a peer on the WireGuard interface
// If client is disabled, expired, or over quota, it removes the peer
// Otherwise, it adds/updates the peer
func UpdatePeerOnInterface(client model.Client, server model.Server, settings model.GlobalSetting, interfaceName string) error {
	if interfaceName == "" {
		interfaceName = GetInterfaceNameFromConfig(settings.ConfigFilePath)
	}

	// Check if client should be removed
	shouldRemove := false

	// Check if disabled
	if !client.Enabled {
		shouldRemove = true
	}

	// Check if expired
	if !shouldRemove && !client.Expiration.IsZero() && time.Now().After(client.Expiration) {
		shouldRemove = true
	}

	// Check if over quota
	if !shouldRemove && client.Quota > 0 && client.UsedQuota >= client.Quota {
		shouldRemove = true
	}

	if shouldRemove {
		if client.PublicKey == "" {
			return fmt.Errorf("cannot remove peer: public key is empty")
		}
		return RemovePeerFromInterface(client.PublicKey, interfaceName)
	}

	// Client is valid, add/update peer
	return AddPeerToInterface(client, server, settings, interfaceName)
}
