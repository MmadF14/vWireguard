package handler

import (
	"encoding/json"
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
	// Type is "wireguard" (default) or "amneziawg".
	Type string `json:"type"`
	// AmneziaWG may be left empty to keep the current parameters, or have a
	// random set generated when the interface becomes AmneziaWG.
	AmneziaWG *model.AmneziaWGParams `json:"amneziawg"`
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
		if devices, err := util.ListDevices(); err == nil {
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
				EnabledAtBoot: util.InterfaceIsEnabledAtBoot(i),
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
		previous := iface

		switch req.Type = strings.TrimSpace(req.Type); req.Type {
		case "", model.InterfaceTypeWireGuard:
			iface.Type = model.InterfaceTypeWireGuard
		case model.InterfaceTypeAmneziaWG:
			// The legacy interface's config is also rewritten by the server
			// page, the quota checker and startup through plain wg-quick, which
			// would bring a WireGuard device back up next to the AmneziaWG one.
			settings, err := db.GetGlobalSettings()
			if err != nil {
				return c.JSON(http.StatusInternalServerError, jsonHTTPResponse{false, "Cannot read global settings"})
			}
			if legacy := util.GetInterfaceNameFromConfig(settings.ConfigFilePath); req.Name == legacy {
				return c.JSON(http.StatusBadRequest, jsonHTTPResponse{false, fmt.Sprintf(
					"%s is the default interface and stays plain WireGuard. Create a separate interface (e.g. awg0) for AmneziaWG and move clients to it.",
					legacy)})
			}
			iface.Type = model.InterfaceTypeAmneziaWG
			if req.AmneziaWG != nil {
				iface.AmneziaWG = req.AmneziaWG
			} else if iface.AmneziaWG == nil {
				params, err := model.GenerateAmneziaWGParams()
				if err != nil {
					return c.JSON(http.StatusInternalServerError, jsonHTTPResponse{false, err.Error()})
				}
				iface.AmneziaWG = &params
			}
		default:
			return c.JSON(http.StatusBadRequest, jsonHTTPResponse{false, fmt.Sprintf("Unknown interface type %q", req.Type)})
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

		// If default interface (wg0) was updated, keep server record in sync
		if iface.Name == model.DefaultInterfaceName {
			if s, err := db.GetServer(); err == nil {
				if s.Interface != nil {
					s.Interface.ListenPort = iface.ListenPort
					s.Interface.Addresses = iface.Addresses
					s.Interface.PostUp = iface.PostUp
					s.Interface.PreDown = iface.PreDown
					s.Interface.PostDown = iface.PostDown
					s.Interface.UpdatedAt = time.Now().UTC()
					_ = db.SaveServerInterface(*s.Interface)
				}
				if s.KeyPair != nil && iface.PrivateKey != "" {
					s.KeyPair.PrivateKey = iface.PrivateKey
					s.KeyPair.PublicKey = iface.PublicKey
					s.KeyPair.UpdatedAt = time.Now().UTC()
					_ = db.SaveServerKeyPair(*s.KeyPair)
				}
			}
		}

		action := "updated"
		if isNew {
			action = "created"
		}
		msg := fmt.Sprintf("Interface %s %s", iface.Name, action)

		// Switching between WireGuard and AmneziaWG moves the interface to
		// another unit (wg-quick@ <-> awg-quick@). Stop the old one so the two
		// do not fight over the listen port, and keep it from returning at boot.
		if !isNew && previous.IsAmnezia() != iface.IsAmnezia() {
			for _, op := range []string{"stop", "disable"} {
				if err := util.InterfaceServiceAction(previous, op); err != nil {
					log.Warnf("Cannot %s %s after type change: %v", op, previous.ServiceName(), err)
				}
			}
			msg += fmt.Sprintf(". %s was stopped; apply the interface to start %s, then re-send client configs",
				previous.ServiceName(), iface.ServiceName())
		}

		log.Infof("Interface %s %s", iface.Name, action)
		return c.JSON(http.StatusOK, jsonHTTPResponse{true, msg})
	}
}

// GenerateAmneziaWGParams handles GET /api/amneziawg/params and returns a
// fresh random parameter set for the interface form.
func GenerateAmneziaWGParams() echo.HandlerFunc {
	return func(c echo.Context) error {
		params, err := model.GenerateAmneziaWGParams()
		if err != nil {
			return c.JSON(http.StatusInternalServerError, jsonHTTPResponse{false, err.Error()})
		}
		return c.JSON(http.StatusOK, params)
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

type migrateClientsRequest struct {
	FromInterface string   `json:"from_interface"`
	ToInterface   string   `json:"to_interface"`
	ClientIDs     []string `json:"client_ids,omitempty"`
}

// MigrateClientsInterface handles POST /api/interfaces/migrate
// Moves clients from one interface to another:
// 1. Removes peers from old interface runtime (RemovePeerFromInterface)
// 2. Assigns free IP from new interface subnet
// 3. Updates client JSON in database
// 4. Hot-reloads peer onto new interface (AmneziaWG/WireGuard)
// 5. Updates interface configuration files on disk
func MigrateClientsInterface(db store.IStore, tmplDir fs.FS) echo.HandlerFunc {
	return func(c echo.Context) error {
		var req migrateClientsRequest
		if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
			return c.JSON(http.StatusBadRequest, jsonHTTPResponse{false, "Invalid request format"})
		}

		req.FromInterface = strings.TrimSpace(req.FromInterface)
		req.ToInterface = strings.TrimSpace(req.ToInterface)

		if req.FromInterface == "" || req.ToInterface == "" {
			return c.JSON(http.StatusBadRequest, jsonHTTPResponse{false, "Both from_interface and to_interface are required"})
		}

		if req.FromInterface == req.ToInterface {
			return c.JSON(http.StatusBadRequest, jsonHTTPResponse{false, "Source and destination interfaces must be different"})
		}

		// Verify destination interface exists and has configured subnets
		toIface, err := db.GetInterface(req.ToInterface)
		if err != nil {
			return c.JSON(http.StatusBadRequest, jsonHTTPResponse{false, fmt.Sprintf("Destination interface %q not found", req.ToInterface)})
		}
		if len(toIface.Addresses) == 0 {
			return c.JSON(http.StatusBadRequest, jsonHTTPResponse{false, fmt.Sprintf("Destination interface %q has no addresses configured", req.ToInterface)})
		}

		// Fetch all clients
		clientsData, err := db.GetClients(false)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, jsonHTTPResponse{false, "Cannot read clients"})
		}

		server, err := db.GetServer()
		if err != nil {
			return c.JSON(http.StatusInternalServerError, jsonHTTPResponse{false, "Cannot read server config"})
		}

		settings, err := db.GetGlobalSettings()
		if err != nil {
			return c.JSON(http.StatusInternalServerError, jsonHTTPResponse{false, "Cannot read global settings"})
		}

		// Find target clients to migrate
		selectedMap := make(map[string]bool)
		for _, id := range req.ClientIDs {
			selectedMap[id] = true
		}

		var toMigrate []model.Client
		for _, cd := range clientsData {
			if cd.Client == nil {
				continue
			}
			if model.InterfaceOf(cd.Client) != req.FromInterface {
				continue
			}
			if len(req.ClientIDs) > 0 && !selectedMap[cd.Client.ID] {
				continue
			}
			toMigrate = append(toMigrate, *cd.Client)
		}

		if len(toMigrate) == 0 {
			return c.JSON(http.StatusOK, jsonHTTPResponse{true, fmt.Sprintf("No clients found on %s to migrate", req.FromInterface)})
		}

		// Get allocated IPs
		allocatedIPs, err := util.GetAllocatedIPs("")
		if err != nil {
			return c.JSON(http.StatusInternalServerError, jsonHTTPResponse{false, "Cannot read allocated IPs"})
		}

		allocatedMap := make(map[string]bool)
		for _, ip := range allocatedIPs {
			allocatedMap[ip] = true
		}

		// Ensure interface own addresses (server-side interface IPs) are not assigned to clients
		for _, ifAddr := range toIface.Addresses {
			if ipOnly, err := util.GetIPFromCIDR(ifAddr); err == nil {
				allocatedMap[ipOnly] = true
			}
		}

		migratedCount := 0
		var failedClients []string

		for _, client := range toMigrate {
			// 1. Remove from old interface runtime
			if client.PublicKey != "" {
				_ = util.RemovePeerFromInterface(client.PublicKey, req.FromInterface)
			}

			// Free old IP from allocated set
			for _, oldIP := range client.AllocatedIPs {
				if ipOnly, err := util.GetIPFromCIDR(oldIP); err == nil {
					delete(allocatedMap, ipOnly)
				}
				delete(allocatedMap, oldIP)
			}

			// Rebuild active allocatedIPs list (pure IP strings without CIDR mask)
			curAllocated := make([]string, 0, len(allocatedMap))
			for ip := range allocatedMap {
				curAllocated = append(curAllocated, ip)
			}

			// 2. Allocate free IP from new interface subnet
			newIP, err := util.GetAvailableIP(toIface.Addresses[0], curAllocated, toIface.Addresses)
			if err != nil {
				log.Errorf("Failed to allocate IP for client %s on %s: %v", client.Name, req.ToInterface, err)
				failedClients = append(failedClients, fmt.Sprintf("%s (no IP available)", client.Name))
				// re-add old IP back
				for _, oldIP := range client.AllocatedIPs {
					if ipOnly, err := util.GetIPFromCIDR(oldIP); err == nil {
						allocatedMap[ipOnly] = true
					}
					allocatedMap[oldIP] = true
				}
				continue
			}

			// Mark bare IP as allocated so next iteration in the batch sees it!
			allocatedMap[newIP] = true

			// Format CIDR
			var formattedIP string
			if strings.Contains(newIP, ":") {
				formattedIP = fmt.Sprintf("%s/128", newIP)
			} else {
				formattedIP = fmt.Sprintf("%s/32", newIP)
			}

			// 3. Update client model and save
			client.AllocatedIPs = []string{formattedIP}
			client.Interface = req.ToInterface
			client.UpdatedAt = time.Now().UTC()

			if err := db.SaveClient(client); err != nil {
				log.Errorf("Failed to save migrated client %s: %v", client.Name, err)
				failedClients = append(failedClients, fmt.Sprintf("%s (db save error)", client.Name))
				continue
			}

			// 4. Hot-reload to destination interface
			if client.Enabled && util.IsClientValid(client) {
				if err := util.AddPeerToInterface(client, server, settings, req.ToInterface); err != nil {
					log.Warnf("Failed to hot-reload peer %s to %s: %v", client.Name, req.ToInterface, err)
				}
			}

			migratedCount++
		}

		// 5. Update interface configuration files on disk
		if updatedClients, err := db.GetClients(false); err == nil {
			if users, err := db.GetUsers(); err == nil {
				// Re-render source interface config
				if fromIface, err := db.GetInterface(req.FromInterface); err == nil {
					fromPeers := util.ClientsForInterface(updatedClients, fromIface.Name)
					_ = util.WriteWireGuardInterfaceConfig(tmplDir, fromIface, fromPeers, users, settings)
				}
				// Re-render target interface config
				toPeers := util.ClientsForInterface(updatedClients, toIface.Name)
				_ = util.WriteWireGuardInterfaceConfig(tmplDir, toIface, toPeers, users, settings)
			}
		}

		msg := fmt.Sprintf("Successfully migrated %d client(s) from %s to %s", migratedCount, req.FromInterface, req.ToInterface)
		if len(failedClients) > 0 {
			msg += fmt.Sprintf(" (Failed: %s)", strings.Join(failedClients, ", "))
		}
		log.Infof(msg)

		return c.JSON(http.StatusOK, map[string]interface{}{
			"success":        len(failedClients) == 0 || migratedCount > 0,
			"message":        msg,
			"migrated_count": migratedCount,
		})
	}
}
