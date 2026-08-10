package util

import (
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/MmadF14/vwireguard/model"
)

// ---------------------------------------------------------------------------
//  Per-interface config generation.
//
//  WriteWireGuardServerConfig writes exactly one file, whose path comes from
//  GlobalSetting.ConfigFilePath - fine when the panel owned a single interface.
//  With wg0/wg1/wg2 each interface needs its own /etc/wireguard/<name>.conf,
//  built from that interface's own keys, addresses, port, MTU and PostUp hooks,
//  and containing only the peers assigned to it.
//
//  The SAME wg.conf template is reused: WGInterface.ToLegacyServer() projects
//  an interface onto the model.Server shape the template already expects, and a
//  per-interface copy of GlobalSetting carries the MTU/path overrides. So a
//  custom template keeps working unchanged.
// ---------------------------------------------------------------------------

// resolveWgTemplate returns the wg.conf template text using the same precedence
// as WriteWireGuardServerConfig: explicit -wg-conf-template, then the embedded
// template directory, then the compiled-in default.
func resolveWgTemplate(tmplDir fs.FS) (string, error) {
	if len(WgConfTemplate) > 0 {
		b, err := os.ReadFile(WgConfTemplate)
		if err != nil {
			return "", err
		}
		return string(b), nil
	}
	if tmplDir != nil {
		return StringFromEmbedFile(tmplDir, "wg.conf")
	}
	b, err := GetWireGuardConfigTemplate()
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// ClientsForInterface filters a client list down to one interface.
// Clients with no explicit assignment belong to model.DefaultInterfaceName.
func ClientsForInterface(all []model.ClientData, ifaceName string) []model.ClientData {
	out := make([]model.ClientData, 0, len(all))
	for _, cd := range all {
		if cd.Client == nil {
			continue
		}
		if model.InterfaceOf(cd.Client) == ifaceName {
			out = append(out, cd)
		}
	}
	return out
}

// WriteWireGuardInterfaceConfig renders /etc/wireguard/<iface>.conf for a single
// interface. Only clients assigned to that interface are included.
//
// The file is written atomically (temp file + rename) so a crash mid-write
// cannot leave wg-quick with a truncated config.
func WriteWireGuardInterfaceConfig(
	tmplDir fs.FS,
	iface model.WGInterface,
	clientDataList []model.ClientData,
	usersList []model.User,
	globalSettings model.GlobalSetting,
) error {
	if err := iface.Validate(); err != nil {
		return err
	}

	tmplText, err := resolveWgTemplate(tmplDir)
	if err != nil {
		return err
	}

	// Per-interface overrides on top of the global settings.
	settings := globalSettings
	settings.ConfigFilePath = iface.ConfigFilePath()
	if iface.MTU > 0 {
		settings.MTU = iface.MTU
	}
	if len(iface.DNSServers) > 0 {
		settings.DNSServers = iface.DNSServers
	}
	if iface.Endpoint != "" {
		settings.EndpointAddress = iface.Endpoint
	}

	// Escape multiline notes so they stay valid comments.
	escaped := make([]model.ClientData, 0, len(clientDataList))
	for _, cd := range clientDataList {
		if cd.Client != nil && cd.Client.AdditionalNotes != "" {
			clientCopy := *cd.Client
			clientCopy.AdditionalNotes = strings.ReplaceAll(clientCopy.AdditionalNotes, "\n", "\n# ")
			cd.Client = &clientCopy
		}
		escaped = append(escaped, cd)
	}

	t, err := template.New("wg_config").Parse(tmplText)
	if err != nil {
		return err
	}

	dst := iface.ConfigFilePath()
	if err := os.MkdirAll(filepath.Dir(dst), 0700); err != nil {
		return err
	}

	tmp, err := os.CreateTemp(filepath.Dir(dst), ".wg-*.conf.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName) // no-op once the rename succeeds

	config := map[string]interface{}{
		"serverConfig":   iface.ToLegacyServer(),
		"clientDataList": escaped,
		"globalSettings": settings,
		"usersList":      usersList,
		"wgInterface":    iface,
	}

	if err := t.Execute(tmp, config); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}

	// wg-quick refuses world-readable configs; they hold private keys.
	if err := os.Chmod(tmpName, 0600); err != nil {
		return err
	}
	return os.Rename(tmpName, dst)
}

// WriteAllInterfaceConfigs renders every enabled interface in one pass.
// It returns the names it wrote plus the first error encountered, so a single
// broken interface does not silently stop the others from being written.
func WriteAllInterfaceConfigs(
	tmplDir fs.FS,
	interfaces []model.WGInterface,
	allClients []model.ClientData,
	usersList []model.User,
	globalSettings model.GlobalSetting,
) ([]string, error) {
	var written []string
	var firstErr error

	for _, iface := range interfaces {
		if !iface.Enabled {
			continue
		}
		peers := ClientsForInterface(allClients, iface.Name)
		if err := WriteWireGuardInterfaceConfig(tmplDir, iface, peers, usersList, globalSettings); err != nil {
			if firstErr == nil {
				firstErr = fmt.Errorf("interface %s: %v", iface.Name, err)
			}
			continue
		}
		written = append(written, iface.Name)
	}
	return written, firstErr
}

// ---------------------------------------------------------------------------
//  Service control
// ---------------------------------------------------------------------------

// runCmd runs a command and returns its combined output as a trimmed string.
func runCmd(name string, args ...string) (string, error) {
	out, err := exec.Command(name, args...).CombinedOutput()
	return strings.TrimSpace(string(out)), err
}

// SyncOrRestartInterface applies a freshly written config to a live interface.
//
// It prefers `wg syncconf`, which updates peers with ZERO downtime. syncconf
// cannot change the [Interface] block (Address/ListenPort/PrivateKey/MTU), so
// when that fails the interface is restarted instead. Restart drops every peer
// on THAT interface only - which is the whole point of splitting interfaces.
//
// Returns true when the zero-downtime path was used.
func SyncOrRestartInterface(iface model.WGInterface) (bool, error) {
	name := iface.Name
	cfg := iface.ConfigFilePath()

	// Is the device even up? If not, a sync is meaningless - bring it up.
	if _, err := runCmd("wg", "show", name); err != nil {
		if _, err := runCmd("sudo", "systemctl", "start", iface.ServiceName()); err != nil {
			return false, fmt.Errorf("interface %s is down and could not be started: %v", name, err)
		}
		return false, nil
	}

	// `wg syncconf` needs a stripped config (no wg-quick-only keys).
	stripped, err := runCmd("sudo", "wg-quick", "strip", name)
	if err == nil && stripped != "" {
		tmp, terr := os.CreateTemp("", "wg-sync-*.conf")
		if terr == nil {
			tmpName := tmp.Name()
			defer os.Remove(tmpName)
			_, werr := tmp.WriteString(stripped)
			tmp.Close()
			if werr == nil {
				if _, serr := runCmd("sudo", "wg", "syncconf", name, tmpName); serr == nil {
					return true, nil
				}
			}
		}
	} else {
		// Fall back to syncing straight from the file.
		if _, serr := runCmd("sudo", "wg", "syncconf", name, cfg); serr == nil {
			return true, nil
		}
	}

	// [Interface] changed (or syncconf is unavailable): restart this unit only.
	if out, rerr := runCmd("sudo", "systemctl", "restart", iface.ServiceName()); rerr != nil {
		return false, fmt.Errorf("restart of %s failed: %v (%s)", iface.ServiceName(), rerr, out)
	}
	return false, nil
}

// InterfaceServiceAction runs start/stop/restart/enable/disable on an interface's unit.
func InterfaceServiceAction(iface model.WGInterface, action string) error {
	switch action {
	case "start", "stop", "restart", "enable", "disable":
	default:
		return fmt.Errorf("unsupported action %q", action)
	}
	if out, err := runCmd("sudo", "systemctl", action, iface.ServiceName()); err != nil {
		return fmt.Errorf("systemctl %s %s failed: %v (%s)", action, iface.ServiceName(), err, out)
	}
	return nil
}

// InterfaceIsRunning reports whether the netdev currently exists.
func InterfaceIsRunning(name string) bool {
	_, err := runCmd("wg", "show", name)
	return err == nil
}

// InterfaceIsEnabledAtBoot reports whether wg-quick@<name> starts on boot.
// This is the check that catches "panel is up but the tunnel is not after a
// reboot", which is otherwise invisible until users complain.
func InterfaceIsEnabledAtBoot(name string) bool {
	out, _ := runCmd("systemctl", "is-enabled", "wg-quick@"+name)
	return out == "enabled" || out == "enabled-runtime"
}
