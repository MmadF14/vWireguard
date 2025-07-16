package monitor

import (
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/MmadF14/vwireguard/model"
	"github.com/MmadF14/vwireguard/store"
)

type Update struct {
	ID    string `json:"id"`
	Color string `json:"color"`
}

var (
	subsMu      sync.Mutex
	subscribers = map[chan Update]struct{}{}
)

func Subscribe() chan Update {
	ch := make(chan Update, 1)
	subsMu.Lock()
	subscribers[ch] = struct{}{}
	subsMu.Unlock()
	return ch
}

func Unsubscribe(ch chan Update) {
	subsMu.Lock()
	delete(subscribers, ch)
	close(ch)
	subsMu.Unlock()
}

func publish(u Update) {
	subsMu.Lock()
	for ch := range subscribers {
		select {
		case ch <- u:
		default:
		}
	}
	subsMu.Unlock()
}

func Start(db store.IStore) {
	go func() {
		for {
			tunnels, err := db.GetTunnels()
			if err == nil {
				for _, t := range tunnels {
					color := "gray"
					if t.Status == model.TunnelStatusActive {
						if t.Type == model.TunnelTypeWireGuardToWireGuard {
							iface := "wg" + t.ID[len(t.ID)-3:]
							out, err := exec.Command("wg", "show", iface, "latest-handshakes").Output()
							if err == nil && len(out) > 0 {
								parts := strings.Fields(string(out))
								if len(parts) >= 2 {
									ts, _ := time.ParseDuration(parts[1] + "ms")
									if ts.Seconds() <= 90 {
										color = "green"
									} else {
										color = "red"
									}
								}
							}
						} else if t.WGConfig != nil {
							ip := t.WGConfig.TunnelIP
							if ip != "" {
								if err := exec.Command("ping", "-c", "1", "-W", "2", "-I", ip, "1.1.1.1").Run(); err == nil {
									color = "green"
								} else {
									color = "red"
								}
							}
						}
					}
					if t.StatusColor != color {
						t.StatusColor = color
						db.SaveTunnel(t)
						publish(Update{ID: t.ID, Color: color})
					}
				}
			}
			time.Sleep(30 * time.Second)
		}
	}()
}
