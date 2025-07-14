package util

import (
	"bufio"
	"bytes"
	"fmt"
	"os/exec"
	"strconv"
	"strings"

	"github.com/MmadF14/vwireguard/model"
	"golang.zx2c4.com/wireguard/wgctrl"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
	"net"
	"time"
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
