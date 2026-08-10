package handler

import (
	"fmt"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/MmadF14/vwireguard/store"
	"github.com/labstack/echo/v4"
	"golang.zx2c4.com/wireguard/wgctrl"
)

// ---------------------------------------------------------------------------
//  Metrics endpoint
//
//  GET  <basePath>/api/v1/metrics          -> Prometheus text exposition
//  GET  <basePath>/api/v1/metrics?format=json -> the same numbers as JSON
//
//  Auth (any one of):
//    Authorization: Bearer <token>
//    X-Metrics-Token: <token>
//    ?token=<token>
//
//  The token is either the VWG_METRICS_TOKEN environment variable (a dedicated,
//  read-only scrape token - preferred, so you never hand Prometheus an admin
//  credential) or any valid admin API token.
//
//  Everything here is read-only.
// ---------------------------------------------------------------------------

// peerConnectedWindow is how recently a handshake must have happened for a peer
// to count as "connected". WireGuard rehandshakes about every 2 minutes, so 3
// minutes avoids flapping.
const peerConnectedWindow = 3 * time.Minute

type metricsPeer struct {
	Name          string `json:"name"`
	Email         string `json:"email"`
	PublicKey     string `json:"public_key"`
	Interface     string `json:"interface"`
	Enabled       bool   `json:"enabled"`
	Connected     bool   `json:"connected"`
	ReceiveBytes  int64  `json:"receive_bytes"`
	TransmitBytes int64  `json:"transmit_bytes"`
	// LastHandshakeAgeSeconds is -1 when the peer has never completed one.
	LastHandshakeAgeSeconds float64 `json:"last_handshake_age_seconds"`
	QuotaBytes              int64   `json:"quota_bytes"`
	QuotaUsedBytes          int64   `json:"quota_used_bytes"`
	Expired                 bool    `json:"expired"`
}

type metricsInterface struct {
	Name       string `json:"name"`
	ListenPort int    `json:"listen_port"`
	Peers      int    `json:"peers"`
	PeersUp    int    `json:"peers_up"`
}

type metricsSnapshot struct {
	Up                 int                `json:"up"`
	ScrapeDurationSecs float64            `json:"scrape_duration_seconds"`
	ClientsTotal       int                `json:"clients_total"`
	ClientsEnabled     int                `json:"clients_enabled"`
	ClientsConnected   int                `json:"clients_connected"`
	ClientsExpired     int                `json:"clients_expired"`
	ClientsOverQuota   int                `json:"clients_over_quota"`
	TotalReceiveBytes  int64              `json:"total_receive_bytes"`
	TotalTransmitBytes int64              `json:"total_transmit_bytes"`
	Interfaces         []metricsInterface `json:"interfaces"`
	Peers              []metricsPeer      `json:"peers"`
}

// metricsTokenFromRequest pulls the bearer/header/query token out of a request.
func metricsTokenFromRequest(c echo.Context) string {
	if h := c.Request().Header.Get("Authorization"); h != "" {
		if len(h) > 7 && strings.EqualFold(h[:7], "bearer ") {
			return strings.TrimSpace(h[7:])
		}
	}
	if h := c.Request().Header.Get("X-Metrics-Token"); h != "" {
		return strings.TrimSpace(h)
	}
	return strings.TrimSpace(c.QueryParam("token"))
}

// authorizeMetrics allows a dedicated scrape token or a valid admin token.
func authorizeMetrics(db store.IStore, c echo.Context) bool {
	token := metricsTokenFromRequest(c)
	if token == "" {
		return false
	}
	if envToken := strings.TrimSpace(os.Getenv("VWG_METRICS_TOKEN")); envToken != "" {
		if subtleEqual(token, envToken) {
			return true
		}
	}
	if _, err := verifyAdminToken(db, token); err == nil {
		return true
	}
	return false
}

// subtleEqual is a length-safe constant-time-ish comparison for the scrape token.
func subtleEqual(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	var diff byte
	for i := 0; i < len(a); i++ {
		diff |= a[i] ^ b[i]
	}
	return diff == 0
}

// collectMetrics builds a full snapshot. It never returns an error for a missing
// WireGuard device - it just reports up=0 for that part, because a monitoring
// endpoint that 500s when the tunnel is down is useless exactly when you need it.
func collectMetrics(db store.IStore) metricsSnapshot {
	start := time.Now()
	snap := metricsSnapshot{Up: 1}

	// --- runtime peer state, keyed by public key -------------------------
	type live struct {
		iface         string
		rx, tx        int64
		lastHandshake time.Time
	}
	liveByKey := map[string]live{}

	if wgClient, err := wgctrl.New(); err == nil {
		defer wgClient.Close()
		if devices, err := wgClient.Devices(); err == nil {
			for _, d := range devices {
				mi := metricsInterface{
					Name:       d.Name,
					ListenPort: d.ListenPort,
					Peers:      len(d.Peers),
				}
				for _, p := range d.Peers {
					key := p.PublicKey.String()
					liveByKey[key] = live{
						iface:         d.Name,
						rx:            p.ReceiveBytes,
						tx:            p.TransmitBytes,
						lastHandshake: p.LastHandshakeTime,
					}
					snap.TotalReceiveBytes += p.ReceiveBytes
					snap.TotalTransmitBytes += p.TransmitBytes
					if !p.LastHandshakeTime.IsZero() && time.Since(p.LastHandshakeTime) <= peerConnectedWindow {
						mi.PeersUp++
					}
				}
				snap.Interfaces = append(snap.Interfaces, mi)
			}
		}
	} else {
		snap.Up = 0
	}

	sort.Slice(snap.Interfaces, func(i, j int) bool { return snap.Interfaces[i].Name < snap.Interfaces[j].Name })

	// --- configured clients ----------------------------------------------
	clients, err := db.GetClients(false)
	if err != nil {
		snap.Up = 0
		snap.ScrapeDurationSecs = time.Since(start).Seconds()
		return snap
	}

	now := time.Now()
	for _, cd := range clients {
		cl := cd.Client
		if cl == nil {
			continue
		}
		snap.ClientsTotal++
		if cl.Enabled {
			snap.ClientsEnabled++
		}

		mp := metricsPeer{
			Name:                    cl.Name,
			Email:                   cl.Email,
			PublicKey:               cl.PublicKey,
			Enabled:                 cl.Enabled,
			QuotaBytes:              cl.Quota,
			QuotaUsedBytes:          cl.UsedQuota,
			LastHandshakeAgeSeconds: -1,
		}

		if !cl.Expiration.IsZero() && now.After(cl.Expiration) {
			mp.Expired = true
			snap.ClientsExpired++
		}
		if cl.Quota > 0 && cl.UsedQuota >= cl.Quota {
			snap.ClientsOverQuota++
		}

		if lv, ok := liveByKey[cl.PublicKey]; ok {
			mp.Interface = lv.iface
			mp.ReceiveBytes = lv.rx
			mp.TransmitBytes = lv.tx
			if !lv.lastHandshake.IsZero() {
				mp.LastHandshakeAgeSeconds = time.Since(lv.lastHandshake).Seconds()
				if time.Since(lv.lastHandshake) <= peerConnectedWindow {
					mp.Connected = true
					snap.ClientsConnected++
				}
			}
		}
		snap.Peers = append(snap.Peers, mp)
	}

	sort.Slice(snap.Peers, func(i, j int) bool { return snap.Peers[i].Name < snap.Peers[j].Name })
	snap.ScrapeDurationSecs = time.Since(start).Seconds()
	return snap
}

// escapeLabel escapes a Prometheus label value.
func escapeLabel(s string) string {
	r := strings.NewReplacer(`\`, `\\`, `"`, `\"`, "\n", `\n`)
	return r.Replace(s)
}

func boolToF(b bool) int {
	if b {
		return 1
	}
	return 0
}

// renderPrometheus turns a snapshot into the text exposition format.
func renderPrometheus(s metricsSnapshot) string {
	var b strings.Builder

	write := func(name, help, typ string, body func()) {
		fmt.Fprintf(&b, "# HELP %s %s\n# TYPE %s %s\n", name, help, name, typ)
		body()
	}

	write("vwireguard_up", "1 if the panel could read WireGuard and its store.", "gauge", func() {
		fmt.Fprintf(&b, "vwireguard_up %d\n", s.Up)
	})
	write("vwireguard_scrape_duration_seconds", "Time spent collecting these metrics.", "gauge", func() {
		fmt.Fprintf(&b, "vwireguard_scrape_duration_seconds %f\n", s.ScrapeDurationSecs)
	})
	write("vwireguard_clients_total", "Number of configured clients.", "gauge", func() {
		fmt.Fprintf(&b, "vwireguard_clients_total %d\n", s.ClientsTotal)
	})
	write("vwireguard_clients_enabled", "Number of enabled clients.", "gauge", func() {
		fmt.Fprintf(&b, "vwireguard_clients_enabled %d\n", s.ClientsEnabled)
	})
	write("vwireguard_clients_connected", "Clients with a handshake in the last 3 minutes.", "gauge", func() {
		fmt.Fprintf(&b, "vwireguard_clients_connected %d\n", s.ClientsConnected)
	})
	write("vwireguard_clients_expired", "Clients past their expiration date.", "gauge", func() {
		fmt.Fprintf(&b, "vwireguard_clients_expired %d\n", s.ClientsExpired)
	})
	write("vwireguard_clients_over_quota", "Clients that have used their whole quota.", "gauge", func() {
		fmt.Fprintf(&b, "vwireguard_clients_over_quota %d\n", s.ClientsOverQuota)
	})
	write("vwireguard_receive_bytes_total", "Total bytes received from all peers.", "counter", func() {
		fmt.Fprintf(&b, "vwireguard_receive_bytes_total %d\n", s.TotalReceiveBytes)
	})
	write("vwireguard_transmit_bytes_total", "Total bytes transmitted to all peers.", "counter", func() {
		fmt.Fprintf(&b, "vwireguard_transmit_bytes_total %d\n", s.TotalTransmitBytes)
	})

	write("vwireguard_interface_peers", "Peers configured on a WireGuard interface.", "gauge", func() {
		for _, i := range s.Interfaces {
			fmt.Fprintf(&b, "vwireguard_interface_peers{interface=\"%s\",listen_port=\"%d\"} %d\n",
				escapeLabel(i.Name), i.ListenPort, i.Peers)
		}
	})
	write("vwireguard_interface_peers_up", "Peers on an interface with a recent handshake.", "gauge", func() {
		for _, i := range s.Interfaces {
			fmt.Fprintf(&b, "vwireguard_interface_peers_up{interface=\"%s\",listen_port=\"%d\"} %d\n",
				escapeLabel(i.Name), i.ListenPort, i.PeersUp)
		}
	})

	lbl := func(p metricsPeer) string {
		return fmt.Sprintf("client=\"%s\",email=\"%s\",interface=\"%s\"",
			escapeLabel(p.Name), escapeLabel(p.Email), escapeLabel(p.Interface))
	}

	write("vwireguard_peer_connected", "1 if this peer handshook in the last 3 minutes.", "gauge", func() {
		for _, p := range s.Peers {
			fmt.Fprintf(&b, "vwireguard_peer_connected{%s} %d\n", lbl(p), boolToF(p.Connected))
		}
	})
	write("vwireguard_peer_enabled", "1 if this peer is enabled in the panel.", "gauge", func() {
		for _, p := range s.Peers {
			fmt.Fprintf(&b, "vwireguard_peer_enabled{%s} %d\n", lbl(p), boolToF(p.Enabled))
		}
	})
	write("vwireguard_peer_receive_bytes_total", "Bytes received from this peer.", "counter", func() {
		for _, p := range s.Peers {
			fmt.Fprintf(&b, "vwireguard_peer_receive_bytes_total{%s} %d\n", lbl(p), p.ReceiveBytes)
		}
	})
	write("vwireguard_peer_transmit_bytes_total", "Bytes transmitted to this peer.", "counter", func() {
		for _, p := range s.Peers {
			fmt.Fprintf(&b, "vwireguard_peer_transmit_bytes_total{%s} %d\n", lbl(p), p.TransmitBytes)
		}
	})
	write("vwireguard_peer_last_handshake_age_seconds", "Seconds since this peer's last handshake (-1 = never).", "gauge", func() {
		for _, p := range s.Peers {
			fmt.Fprintf(&b, "vwireguard_peer_last_handshake_age_seconds{%s} %f\n", lbl(p), p.LastHandshakeAgeSeconds)
		}
	})
	write("vwireguard_peer_quota_bytes", "Quota assigned to this peer (0 = unlimited).", "gauge", func() {
		for _, p := range s.Peers {
			fmt.Fprintf(&b, "vwireguard_peer_quota_bytes{%s} %d\n", lbl(p), p.QuotaBytes)
		}
	})
	write("vwireguard_peer_quota_used_bytes", "Quota consumed by this peer.", "gauge", func() {
		for _, p := range s.Peers {
			fmt.Fprintf(&b, "vwireguard_peer_quota_used_bytes{%s} %d\n", lbl(p), p.QuotaUsedBytes)
		}
	})
	write("vwireguard_peer_expired", "1 if this peer is past its expiration date.", "gauge", func() {
		for _, p := range s.Peers {
			fmt.Fprintf(&b, "vwireguard_peer_expired{%s} %d\n", lbl(p), boolToF(p.Expired))
		}
	})

	return b.String()
}

// APIMetrics handles GET /api/v1/metrics
func APIMetrics(db store.IStore) echo.HandlerFunc {
	return func(c echo.Context) error {
		if !authorizeMetrics(db, c) {
			return c.JSON(http.StatusUnauthorized, map[string]interface{}{
				"status":  "error",
				"message": "Valid metrics or admin token required",
			})
		}

		snap := collectMetrics(db)

		if strings.EqualFold(c.QueryParam("format"), "json") {
			return c.JSON(http.StatusOK, snap)
		}
		return c.String(http.StatusOK, renderPrometheus(snap))
	}
}
