package jsondb

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/MmadF14/vwireguard/model"
	"github.com/MmadF14/vwireguard/util"
)

// ---------------------------------------------------------------------------
//  Multi-interface storage
//
//  Each interface is one JSON document under db/interfaces/<name>.json.
//
//  A pre-multi-interface install has no such directory. Rather than making the
//  caller deal with that, GetInterfaces migrates the legacy single-server
//  record (server/interfaces.json + server/keypair.json) into an interface
//  named model.DefaultInterfaceName the first time it runs. The legacy records
//  are left in place untouched, so downgrading to an older binary still works.
// ---------------------------------------------------------------------------

const interfacesCollection = "interfaces"

// migrateOnce guards the legacy import so concurrent requests cannot race it.
var migrateOnce sync.Mutex

func (o *JsonDB) interfacesDir() string {
	return path.Join(o.dbPath, interfacesCollection)
}

// ensureInterfacesMigrated creates the collection and imports the legacy record.
func (o *JsonDB) ensureInterfacesMigrated() error {
	migrateOnce.Lock()
	defer migrateOnce.Unlock()

	dir := o.interfacesDir()
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		if err := os.MkdirAll(dir, os.ModePerm); err != nil {
			return err
		}
	}

	records, err := o.conn.ReadAll(interfacesCollection)
	if err == nil && len(records) > 0 {
		return nil // already migrated
	}

	// Pull the old single-server record forward.
	server, err := o.GetServer()
	if err != nil {
		// Nothing to migrate (fresh install); the caller will create interfaces.
		return nil
	}

	iface := model.WGInterfaceFromLegacy(server, model.DefaultInterfaceName)

	// The legacy config path may name the interface something other than wg0.
	if settings, err := o.GetGlobalSettings(); err == nil && settings.ConfigFilePath != "" {
		if name := util.GetInterfaceNameFromConfig(settings.ConfigFilePath); name != "" {
			iface.Name = name
		}
	}

	if iface.PrivateKey == "" || len(iface.Addresses) == 0 {
		// Incomplete legacy record - do not write a half-formed interface.
		return nil
	}

	return o.SaveInterface(*iface)
}

// GetInterfaces returns every configured interface, sorted by name.
func (o *JsonDB) GetInterfaces() ([]model.WGInterface, error) {
	if err := o.ensureInterfacesMigrated(); err != nil {
		return nil, err
	}

	var interfaces []model.WGInterface
	records, err := o.conn.ReadAll(interfacesCollection)
	if err != nil {
		// An empty/missing collection is not an error.
		return interfaces, nil
	}

	for _, rec := range records {
		var iface model.WGInterface
		if err := json.Unmarshal([]byte(rec), &iface); err != nil {
			continue // skip a corrupt record rather than failing the whole list
		}
		interfaces = append(interfaces, iface)
	}

	sort.Slice(interfaces, func(i, j int) bool { return interfaces[i].Name < interfaces[j].Name })
	return interfaces, nil
}

// GetInterface returns one interface by name.
func (o *JsonDB) GetInterface(name string) (model.WGInterface, error) {
	var iface model.WGInterface
	name = strings.TrimSpace(name)
	if name == "" {
		name = model.DefaultInterfaceName
	}
	if err := o.ensureInterfacesMigrated(); err != nil {
		return iface, err
	}
	if err := o.conn.Read(interfacesCollection, name, &iface); err != nil {
		return iface, fmt.Errorf("interface %q not found: %v", name, err)
	}
	return iface, nil
}

// SaveInterface validates and persists an interface.
func (o *JsonDB) SaveInterface(iface model.WGInterface) error {
	if err := iface.Validate(); err != nil {
		return err
	}

	// Two interfaces sharing a listen port would silently fight over the socket.
	existing, _ := o.conn.ReadAll(interfacesCollection)
	for _, rec := range existing {
		var other model.WGInterface
		if err := json.Unmarshal([]byte(rec), &other); err != nil {
			continue
		}
		if other.Name == iface.Name {
			continue
		}
		if other.ListenPort == iface.ListenPort {
			return fmt.Errorf("listen port %d is already used by interface %q", iface.ListenPort, other.Name)
		}
	}

	now := time.Now().UTC()
	if iface.CreatedAt.IsZero() {
		iface.CreatedAt = now
	}
	iface.UpdatedAt = now

	if err := o.conn.Write(interfacesCollection, iface.Name, iface); err != nil {
		return err
	}
	return util.ManagePerms(path.Join(o.interfacesDir(), iface.Name+".json"))
}

// DeleteInterface removes an interface. It refuses while clients still
// reference it, so you cannot orphan peers by accident.
func (o *JsonDB) DeleteInterface(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("interface name is required")
	}

	clients, err := o.GetClients(false)
	if err != nil {
		return err
	}
	var inUse []string
	for _, cd := range clients {
		if cd.Client == nil {
			continue
		}
		if model.InterfaceOf(cd.Client) == name {
			inUse = append(inUse, cd.Client.Name)
		}
	}
	if len(inUse) > 0 {
		preview := inUse
		if len(preview) > 5 {
			preview = preview[:5]
		}
		return fmt.Errorf("interface %q still has %d client(s) assigned (e.g. %s) - move or delete them first",
			name, len(inUse), strings.Join(preview, ", "))
	}

	return o.conn.Delete(interfacesCollection, name)
}

// ClientsByInterface groups clients by the interface they belong to.
func (o *JsonDB) ClientsByInterface() (map[string][]model.ClientData, error) {
	clients, err := o.GetClients(false)
	if err != nil {
		return nil, err
	}
	grouped := map[string][]model.ClientData{}
	for _, cd := range clients {
		if cd.Client == nil {
			continue
		}
		key := model.InterfaceOf(cd.Client)
		grouped[key] = append(grouped[key], cd)
	}
	return grouped, nil
}
