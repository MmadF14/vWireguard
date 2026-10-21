package util

import (
	"strings"
	"testing"

	"github.com/MmadF14/vwireguard/model"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

func TestInjectInterfaceLines(t *testing.T) {
	cfg := "# comment\n[Interface]\nAddress = 10.0.0.1/24\n\n[Peer]\nPublicKey = x\n"
	got := InjectInterfaceLines(cfg, "Jc = 4\n")
	want := "# comment\n[Interface]\nJc = 4\nAddress = 10.0.0.1/24\n\n[Peer]\nPublicKey = x\n"
	if got != want {
		t.Fatalf("got\n%q\nwant\n%q", got, want)
	}
	if InjectInterfaceLines(cfg, "") != cfg {
		t.Fatal("empty lines must leave the config untouched")
	}
}

func TestParseAmneziaDump(t *testing.T) {
	priv, _ := wgtypes.GeneratePrivateKey()
	peer, _ := wgtypes.GeneratePrivateKey()
	dump := strings.Join([]string{
		// interface line, with the AmneziaWG parameters appended
		"awg0\t" + priv.String() + "\t" + priv.PublicKey().String() + "\t51830\toff\t4\t40\t70\t20\t30\t5\t6\t7\t8",
		"awg0\t" + peer.PublicKey().String() + "\t(none)\t203.0.113.7:4321\t10.70.0.2/32\t1700000000\t1024\t2048\toff",
	}, "\n") + "\n"

	devices := parseAmneziaDump(dump)
	if len(devices) != 1 {
		t.Fatalf("expected 1 device, got %d", len(devices))
	}
	d := devices[0]
	if d.Name != "awg0" || d.ListenPort != 51830 || d.PublicKey != priv.PublicKey() {
		t.Fatalf("bad device: %+v", d)
	}
	if len(d.Peers) != 1 {
		t.Fatalf("expected 1 peer, got %d", len(d.Peers))
	}
	p := d.Peers[0]
	if p.PublicKey != peer.PublicKey() || p.ReceiveBytes != 1024 || p.TransmitBytes != 2048 {
		t.Fatalf("bad peer: %+v", p)
	}
	if p.LastHandshakeTime.Unix() != 1700000000 || p.Endpoint.String() != "203.0.113.7:4321" {
		t.Fatalf("bad peer handshake/endpoint: %+v", p)
	}
	if len(p.AllowedIPs) != 1 || p.AllowedIPs[0].String() != "10.70.0.2/32" {
		t.Fatalf("bad allowed IPs: %v", p.AllowedIPs)
	}
}

func TestBuildClientConfigForInterface(t *testing.T) {
	legacyKey, _ := wgtypes.GeneratePrivateKey()
	awgKey, _ := wgtypes.GeneratePrivateKey()
	clientKey, _ := wgtypes.GeneratePrivateKey()

	server := model.Server{
		KeyPair:   &model.ServerKeypair{PrivateKey: legacyKey.String(), PublicKey: legacyKey.PublicKey().String()},
		Interface: &model.ServerInterface{Addresses: []string{"10.252.1.0/24"}, ListenPort: 51820},
	}
	settings := model.GlobalSetting{
		EndpointAddress: "vpn.example.com:51820",
		RelayEndpoint:   "relay.example.com:443",
		MTU:             1450,
		ConfigFilePath:  "/etc/wireguard/wg0.conf",
	}
	client := model.Client{
		Name:         "alice",
		PrivateKey:   clientKey.String(),
		PublicKey:    clientKey.PublicKey().String(),
		AllocatedIPs: []string{"10.70.0.2/32"},
		Interface:    "awg0",
	}
	params := model.AmneziaWGParams{Jc: 4, Jmin: 40, Jmax: 70, S1: 20, S2: 30, H1: 5, H2: 6, H3: 7, H4: 8}
	iface := model.WGInterface{
		Name:       "awg0",
		Type:       model.InterfaceTypeAmneziaWG,
		AmneziaWG:  &params,
		PrivateKey: awgKey.String(),
		PublicKey:  awgKey.PublicKey().String(),
		Addresses:  []string{"10.70.0.1/24"},
		ListenPort: 51830,
		MTU:        1280,
	}

	lookup := func(name string) (model.WGInterface, error) { return iface, nil }
	resolved := ClientInterface(&client, settings, lookup)
	if resolved == nil || resolved.Name != "awg0" {
		t.Fatalf("expected the client to resolve to awg0, got %+v", resolved)
	}

	cfg := BuildClientConfigForInterface(client, server, settings, resolved)
	for _, want := range []string{
		"[Interface]\nJc = 4\n",
		"H4 = 8\n",
		"MTU = 1280\n",
		"PublicKey = " + awgKey.PublicKey().String() + "\n",
		// own port on the global host; the relay belongs to the legacy interface
		"Endpoint = vpn.example.com:51830\n",
	} {
		if !strings.Contains(cfg, want) {
			t.Errorf("config is missing %q:\n%s", want, cfg)
		}
	}

	// A client on the legacy interface is unchanged.
	client.Interface = ""
	if ClientInterface(&client, settings, lookup) != nil {
		t.Fatal("a client without an interface must use the legacy server")
	}
	if got, want := BuildClientConfigForInterface(client, server, settings, nil), BuildClientConfig(client, server, settings); got != want {
		t.Fatalf("legacy config changed:\n%s\nvs\n%s", got, want)
	}
}
