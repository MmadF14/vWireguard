package util

import (
	"bufio"
	"fmt"
	"net"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"golang.zx2c4.com/wireguard/wgctrl"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

// ---------------------------------------------------------------------------
//  AmneziaWG runtime access.
//
//  wgctrl only speaks to the "wireguard" generic-netlink family and to
//  /var/run/wireguard. AmneziaWG registers its own family and socket directory,
//  so its devices are invisible to wgctrl: listing them, reading their traffic
//  counters and hot-adding peers all have to go through the `awg` tool.
//  ListDevices hides that split so status pages, metrics and the quota checker
//  see AmneziaWG peers exactly like WireGuard ones.
// ---------------------------------------------------------------------------

// ListDevices returns every WireGuard device known to wgctrl plus every
// AmneziaWG device reported by `awg show all dump`. A missing awg tool is not
// an error: the system simply has no AmneziaWG devices.
func ListDevices() ([]*wgtypes.Device, error) {
	var devices []*wgtypes.Device
	wgClient, wgErr := wgctrl.New()
	if wgErr == nil {
		defer wgClient.Close()
		devices, wgErr = wgClient.Devices()
	}

	awgDevices, awgErr := listAmneziaDevices()
	if wgErr != nil && awgErr != nil {
		return nil, wgErr
	}

	seen := map[string]bool{}
	for _, d := range devices {
		seen[d.Name] = true
	}
	for _, d := range awgDevices {
		if !seen[d.Name] {
			devices = append(devices, d)
		}
	}
	return devices, nil
}

// listAmneziaDevices parses `awg show all dump`. Every line starts with the
// device name; the first line of a device describes the interface itself
// (its field count depends on the AmneziaWG version), the rest are peers:
//
//	<dev> <public-key> <preshared-key> <endpoint> <allowed-ips> <latest-handshake> <rx> <tx> <keepalive>
func listAmneziaDevices() ([]*wgtypes.Device, error) {
	out, err := exec.Command("awg", "show", "all", "dump").Output()
	if err != nil {
		return nil, err
	}
	return parseAmneziaDump(string(out)), nil
}

func parseAmneziaDump(dump string) []*wgtypes.Device {
	var devices []*wgtypes.Device
	byName := map[string]*wgtypes.Device{}

	scanner := bufio.NewScanner(strings.NewReader(dump))
	for scanner.Scan() {
		f := strings.Split(scanner.Text(), "\t")
		if len(f) < 2 {
			continue
		}
		dev, ok := byName[f[0]]
		if !ok {
			// Interface line: <dev> <private-key> <public-key> <listen-port> <fwmark> [awg params...]
			dev = &wgtypes.Device{Name: f[0]}
			if len(f) >= 4 {
				if k, err := wgtypes.ParseKey(f[2]); err == nil {
					dev.PublicKey = k
				}
				dev.ListenPort, _ = strconv.Atoi(f[3])
			}
			byName[f[0]] = dev
			devices = append(devices, dev)
			continue
		}
		if len(f) < 9 {
			continue
		}
		key, err := wgtypes.ParseKey(f[1])
		if err != nil {
			continue
		}
		peer := wgtypes.Peer{PublicKey: key}
		if psk, err := wgtypes.ParseKey(f[2]); err == nil {
			peer.PresharedKey = psk
		}
		if f[3] != "(none)" {
			if ep, err := net.ResolveUDPAddr("udp", f[3]); err == nil {
				peer.Endpoint = ep
			}
		}
		if f[4] != "(none)" {
			for _, cidr := range strings.Split(f[4], ",") {
				if _, ipNet, err := net.ParseCIDR(strings.TrimSpace(cidr)); err == nil {
					peer.AllowedIPs = append(peer.AllowedIPs, *ipNet)
				}
			}
		}
		if ts, err := strconv.ParseInt(f[5], 10, 64); err == nil && ts > 0 {
			peer.LastHandshakeTime = time.Unix(ts, 0)
		}
		peer.ReceiveBytes, _ = strconv.ParseInt(f[6], 10, 64)
		peer.TransmitBytes, _ = strconv.ParseInt(f[7], 10, 64)
		if ka, err := strconv.Atoi(f[8]); err == nil {
			peer.PersistentKeepaliveInterval = time.Duration(ka) * time.Second
		}
		dev.Peers = append(dev.Peers, peer)
	}
	return devices
}

// isAmneziaDevice reports whether name is a live AmneziaWG device rather than
// a WireGuard one.
func isAmneziaDevice(name string) bool {
	if wgClient, err := wgctrl.New(); err == nil {
		_, devErr := wgClient.Device(name)
		wgClient.Close()
		if devErr == nil {
			return false
		}
	}
	return exec.Command("awg", "show", name, "public-key").Run() == nil
}

// awgConfigurePeer is wgctrl's ConfigureDevice for a single peer, done with
// `awg set` because wgctrl cannot reach AmneziaWG devices.
func awgConfigurePeer(name string, pc wgtypes.PeerConfig) error {
	args := []string{"set", name, "peer", pc.PublicKey.String()}
	var stdin string

	if pc.Remove {
		args = append(args, "remove")
	} else {
		if pc.PresharedKey != nil {
			// Read from stdin so the key never shows up in the process list.
			args = append(args, "preshared-key", "/dev/stdin")
			stdin = pc.PresharedKey.String()
		}
		if pc.Endpoint != nil {
			args = append(args, "endpoint", pc.Endpoint.String())
		}
		if pc.PersistentKeepaliveInterval != nil {
			args = append(args, "persistent-keepalive", strconv.Itoa(int(pc.PersistentKeepaliveInterval.Seconds())))
		}
		allowed := make([]string, 0, len(pc.AllowedIPs))
		for _, ipNet := range pc.AllowedIPs {
			allowed = append(allowed, ipNet.String())
		}
		// `awg set` replaces the allowed-ips list, matching ReplaceAllowedIPs.
		args = append(args, "allowed-ips", strings.Join(allowed, ","))
	}

	cmd := exec.Command("awg", args...)
	if stdin != "" {
		cmd.Stdin = strings.NewReader(stdin)
	}
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("awg set failed: %v (%s)", err, strings.TrimSpace(string(out)))
	}
	return nil
}
