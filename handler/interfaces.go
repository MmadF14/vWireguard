package handler

import (
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/MmadF14/vwireguard/model"
	"github.com/MmadF14/vwireguard/store"
	"github.com/MmadF14/vwireguard/util"
	"github.com/labstack/echo/v4"
	"github.com/labstack/gommon/log"
	"golang.zx2c4.com/wireguard/wgctrl"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

// ---------------------------------------------------------------------------
//  Interface management endpoints (multi-interface support)
//
//    GET    <base>/api/interfaces            list
//    GET    <base>/api/interfaces/:name      one
//    POST   <base>/api/interfaces            create / update
//    POST   <base>/api/interfaces/:name/delete
//
//  All require an admin session, like the rest of the panel's own API.
// ---------------------------------------------------------------------------

type interfaceRequest struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Enabled     bool     `json:"enabled"`
	Addresses   []string `json:"addresses"`
	ListenPort  int      `json:"listen_port"`
	MTU         int      `json:"mtu"`
	PostUp      string   `json:"post_up"`
	PreDown     string   `json:"pre_down"`
	PostDown    string   `json:"post_down"`
	Endpoint    string   `json:"endpoint"`
	DNSServers  []string `json:"dns_servers"`
	// PrivateKey may be left empty on create to have one generated.
	PrivateKey string `json:"private_key"`
	// RegenerateKeys forces a fresh keypair on update. Note this invalidates
	// every existing client config on that interface, so it is opt-in.
	RegenerateKeys bool `json:"regenerate_keys"`
}

// interfaceStatus is the list view: definition plus live runtime state.
type interfaceStatus struct {
	model.WGInterface
	Running     bool   `json:"running"`
	ClientCount int    `json:"client_count"`
	PeersUp     int    `json:"peers_up"`
	ConfigPath  string `json:"config_path"`
	ServiceName string `json:"service_name"`
	// EnabledAtBoot catches the "panel is up but the tunnel is gone after a
	// reboot" trap: the panel installer enables vwireguard but never
	// wg-quick@<name>, so the interface silently does not come back.
	EnabledAtBoot bool `json:"enabled_at_boot"`
	ConfigExists  bool `json:"config_exists"`
}

// InterfacesPage renders the interface management page.
func InterfacesPage() echo.HandlerFunc {
	return func(c echo.Context) error {
		return c.Render(http.StatusOK, "interfaces.html", map[string]interface{}{
			"baseData": model.BaseData{Active: "interfaces", CurrentUser: currentUser(c), Admin: isAdmin(c)},
		})
	}
}

// GetInterfaces handles GET /api/interfaces
func GetInterfaces(db store.IStore) echo.HandlerFunc {
	return func(c echo.Context) error {
		interfaces, err := db.GetInterfaces()
		if err != nil {
			log.Error("Cannot list interfaces: ", err)
			return c.JSON(http.StatusInternalServerError, jsonHTTPResponse{false, "Cannot list interfaces"})
		}

		// Count clients per interface.
		counts := map[string]int{}
		if clients, err := db.GetClients(false); err == nil {
			for _, cd := range clients {
				if cd.Client != nil {
					counts[model.InterfaceOf(cd.Client)]++
				}
			}
		}

		// Live peer state per device.
		running := map[string]int{}
		if wgClient, err := wgctrl.New(); err == nil {
			defer wgClient.Close()
			if devices, err := wgClient.Devices(); err == nil {
				for _, d := range devices {
					up := 0
					for _, p := range d.Peers {
						if !p.LastHandshakeTime.IsZero() && time.Since(p.LastHandshakeTime) <= peerConnectedWindow {
							up++
						}
					}
					running[d.Name] = up
				}
			}
		}

		out := make([]interfaceStatus, 0, len(interfaces))
		for _, i := range interfaces {
			up, isRunning := running[i.Name]
			_, statErr := os.Stat(i.ConfigFilePath())
			out = append(out, interfaceStatus{
				WGInterface:   i,
				Running:       isRunning,
				ClientCount:   counts[i.Name],
				PeersUp:       up,
				ConfigPath:    i.ConfigFilePath(),
				ServiceName:   i.ServiceName(),
				EnabledAtBoot: util.InterfaceIsEnabledAtBoot(i.Name),
				ConfigExists:  statErr == nil,
			})
		}
		return c.JSON(http.StatusOK, out)
	}
}

// GetInterface handles GET /api/interfaces/:name
func GetInterface(db store.IStore) echo.HandlerFunc {
	return func(c echo.Context) error {
		iface, err := db.GetInterface(c.Param("name"))
		if err != nil {
			return c.JSON(http.StatusNotFound, jsonHTTPResponse{false, err.Error()})
		}
		return c.JSON(http.StatusOK, iface)
	}
}

// SaveInterface handles POST /api/interfaces (create or update)
func SaveInterface(db store.IStore) echo.HandlerFunc {
	return func(c echo.Context) error {
		var req interfaceRequest
		if err := c.Bind(&req); err != nil {
			return c.JSON(http.StatusBadRequest, jsonHTTPResponse{false, "Invalid request"})
		}

		req.Name = strings.TrimSpace(req.Name)
		if req.Name == "" {
			return c.JSON(http.StatusBadRequest, jsonHTTPResponse{false, "Interface name is required"})
		}

		// Start from the existing record so a partial update does not wipe keys.
		iface, err := db.GetInterface(req.Name)
		isNew := err != nil
		if isNew {
			iface = model.WGInterface{Name: req.Name}
		}

		iface.Description = req.Description
		iface.Enabled = req.Enabled
		iface.Addresses = req.Addresses

		// Without a PostUp that adds FORWARD + MASQUERADE rules, peers on this
		// interface get an address and handshake fine but have NO internet -
		// the packets are never NATed out of the box. The original wg0 gets
		// these rules generated for it at first run; a hand-created interface
		// would silently miss them, so fill in the same defaults when the user
		// leaves the field empty.
		wan := util.GetDefaultInterfaceName()
		if strings.TrimSpace(req.PostUp) == "" {
			req.PostUp = fmt.Sprintf(
				"iptables -A FORWARD -i %%i -j ACCEPT; iptables -A FORWARD -o %%i -j ACCEPT; iptables -t nat -A POSTROUTING -o %s -j MASQUERADE",
				wan)
		}
		if strings.TrimSpace(req.PostDown) == "" {
			req.PostDown = fmt.Sprintf(
				"iptables -D FORWARD -i %%i -j ACCEPT; iptables -D FORWARD -o %%i -j ACCEPT; iptables -t nat -D POSTROUTING -o %s -j MASQUERADE",
				wan)
		}

		iface.ListenPort = req.ListenPort
		iface.MTU = req.MTU
		iface.PostUp = req.PostUp
		iface.PreDown = req.PreDown
		iface.PostDown = req.PostDown
		iface.Endpoint = strings.TrimSpace(req.Endpoint)
		iface.DNSServers = req.DNSServers

		switch {
		case strings.TrimSpace(req.PrivateKey) != "":
			key, err := wgtypes.ParseKey(strings.TrimSpace(req.PrivateKey))
			if err != nil {
				return c.JSON(http.StatusBadRequest, jsonHTTPResponse{false, "Invalid private key"})
			}
			iface.PrivateKey = key.String()
			iface.PublicKey = key.PublicKey().String()

		case req.RegenerateKeys || iface.PrivateKey == "":
			key, err := wgtypes.GeneratePrivateKey()
			if err != nil {
				return c.JSON(http.StatusInternalServerError, jsonHTTPResponse{false, "Cannot generate key"})
			}
			iface.PrivateKey = key.String()
			iface.PublicKey = key.PublicKey().String()
		}

		if err := db.SaveInterface(iface); err != nil {
			return c.JSON(http.StatusBadRequest, jsonHTTPResponse{false, err.Error()})
		}

		action := "updated"
		if isNew {
			action = "created"
		}
		log.Infof("Interface %s %s", iface.Name, action)
		return c.JSON(http.StatusOK, jsonHTTPResponse{true, fmt.Sprintf("Interface %s %s", iface.Name, action)})
	}
}

// ApplyInterfaceConfig handles POST /api/interfaces/:name/apply
//
// Renders /etc/wireguard/<name>.conf from this interface's own definition and
// only the clients assigned to it, then applies it with `wg syncconf` when
// possible (zero downtime) or a restart of that ONE unit when the [Interface]
// block changed. Other interfaces are never touched.
func ApplyInterfaceConfig(db store.IStore, tmplDir fs.FS) echo.HandlerFunc {
	return func(c echo.Context) error {
		iface, err := db.GetInterface(c.Param("name"))
		if err != nil {
			return c.JSON(http.StatusNotFound, jsonHTTPResponse{false, err.Error()})
		}

		clients, err := db.GetClients(false)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, jsonHTTPResponse{false, "Cannot read clients"})
		}
		users, err := db.GetUsers()
		if err != nil {
			return c.JSON(http.StatusInternalServerError, jsonHTTPResponse{false, "Cannot read users"})
		}
		settings, err := db.GetGlobalSettings()
		if err != nil {
			return c.JSON(http.StatusInternalServerError, jsonHTTPResponse{false, "Cannot read global settings"})
		}

		peers := util.ClientsForInterface(clients, iface.Name)
		if err := util.WriteWireGuardInterfaceConfig(tmplDir, iface, peers, users, settings); err != nil {
			log.Errorf("Cannot write config for %s: %v", iface.Name, err)
			return c.JSON(http.StatusInternalServerError,
				jsonHTTPResponse{false, fmt.Sprintf("Cannot write %s: %v", iface.ConfigFilePath(), err)})
		}

		if !iface.Enabled {
			return c.JSON(http.StatusOK, jsonHTTPResponse{true,
				fmt.Sprintf("Wrote %s (%d peers). Interface is disabled, so it was not started.",
					iface.ConfigFilePath(), len(peers))})
		}

		zeroDowntime, err := util.SyncOrRestartInterface(iface)
		if err != nil {
			log.Errorf("Cannot apply %s: %v", iface.Name, err)
			return c.JSON(http.StatusInternalServerError,
				jsonHTTPResponse{false, fmt.Sprintf("Config written but apply failed: %v", err)})
		}

		how := "restarted (interface settings changed)"
		if zeroDowntime {
			how = "synced with zero downtime"
		}
		return c.JSON(http.StatusOK, jsonHTTPResponse{true,
			fmt.Sprintf("%s: %d peers written and %s", iface.Name, len(peers), how)})
	}
}

// ApplyAllInterfaceConfigs handles POST /api/interfaces/apply-all
func ApplyAllInterfaceConfigs(db store.IStore, tmplDir fs.FS) echo.HandlerFunc {
	return func(c echo.Context) error {
		interfaces, err := db.GetInterfaces()
		if err != nil {
			return c.JSON(http.StatusInternalServerError, jsonHTTPResponse{false, "Cannot list interfaces"})
		}
		clients, err := db.GetClients(false)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, jsonHTTPResponse{false, "Cannot read clients"})
		}
		users, err := db.GetUsers()
		if err != nil {
			return c.JSON(http.StatusInternalServerError, jsonHTTPResponse{false, "Cannot read users"})
		}
		settings, err := db.GetGlobalSettings()
		if err != nil {
			return c.JSON(http.StatusInternalServerError, jsonHTTPResponse{false, "Cannot read global settings"})
		}

		written, writeErr := util.WriteAllInterfaceConfigs(tmplDir, interfaces, clients, users, settings)

		var applied, failed []string
		for _, iface := range interfaces {
			if !iface.Enabled {
				continue
			}
			if !contains(written, iface.Name) {
				continue
			}
			if _, err := util.SyncOrRestartInterface(iface); err != nil {
				failed = append(failed, fmt.Sprintf("%s (%v)", iface.Name, err))
				continue
			}
			applied = append(applied, iface.Name)
		}

		msg := fmt.Sprintf("Applied: %s", strings.Join(applied, ", "))
		if len(applied) == 0 {
			msg = "No interface was applied"
		}
		if len(failed) > 0 {
			msg += " | Failed: " + strings.Join(failed, "; ")
		}
		if writeErr != nil {
			msg += " | Write error: " + writeErr.Error()
		}
		return c.JSON(http.StatusOK, jsonHTTPResponse{len(failed) == 0 && writeErr == nil, msg})
	}
}

// InterfaceService handles POST /api/interfaces/:name/service/:action
// action is one of start|stop|restart|enable|disable.
func InterfaceService(db store.IStore) echo.HandlerFunc {
	return func(c echo.Context) error {
		iface, err := db.GetInterface(c.Param("name"))
		if err != nil {
			return c.JSON(http.StatusNotFound, jsonHTTPResponse{false, err.Error()})
		}
		action := c.Param("action")
		if err := util.InterfaceServiceAction(iface, action); err != nil {
			return c.JSON(http.StatusInternalServerError, jsonHTTPResponse{false, err.Error()})
		}
		return c.JSON(http.StatusOK, jsonHTTPResponse{true,
			fmt.Sprintf("%s %sd", iface.ServiceName(), strings.TrimSuffix(action, "e"))})
	}
}

func contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

// DeleteInterface handles POST /api/interfaces/:name/delete
func DeleteInterface(db store.IStore) echo.HandlerFunc {
	return func(c echo.Context) error {
		name := c.Param("name")
		if name == model.DefaultInterfaceName {
			return c.JSON(http.StatusBadRequest,
				jsonHTTPResponse{false, "The default interface cannot be deleted"})
		}
		if err := db.DeleteInterface(name); err != nil {
			return c.JSON(http.StatusBadRequest, jsonHTTPResponse{false, err.Error()})
		}
		log.Infof("Interface %s deleted", name)
		return c.JSON(http.StatusOK, jsonHTTPResponse{true, "Interface deleted"})
	}
}
