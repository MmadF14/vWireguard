package handler

import (
	"net/http"
	"os/exec"

	"github.com/MmadF14/vwireguard/model"
	"github.com/MmadF14/vwireguard/store"
	"github.com/labstack/echo/v4"
)

// UtilitiesPage handles the utilities page request
func UtilitiesPage(db store.IStore) echo.HandlerFunc {
	return func(c echo.Context) error {
		user, _ := db.GetUserByName(currentUser(c))
		data := map[string]interface{}{
			"baseData": model.BaseData{
				Active:      "utilities",
				CurrentUser: currentUser(c),
				Admin:       user.Role == model.RoleAdmin,
			},
		}

		return c.Render(http.StatusOK, "utilities.html", data)
	}
}

// RestartWireGuardService restarts the WireGuard service
func RestartWireGuardService(db store.IStore) echo.HandlerFunc {
	return func(c echo.Context) error {
		if !isAdmin(c) {
			return c.JSON(http.StatusForbidden, jsonHTTPResponse{false, "Only administrators can restart the service"})
		}

		// First try to restart wg-quick@wg0
		cmd := exec.Command("systemctl", "restart", "wg-quick@wg0")
		output, err := cmd.CombinedOutput()
		if err != nil {
			// If that fails, try restarting the main vWireguard service
			cmd = exec.Command("systemctl", "restart", "vwireguard")
			if output, err = cmd.CombinedOutput(); err != nil {
				return c.JSON(http.StatusInternalServerError, jsonHTTPResponse{false, "Failed to restart WireGuard service: " + string(output)})
			}
			return c.JSON(http.StatusOK, jsonHTTPResponse{true, "vWireguard service restarted successfully"})
		}

		return c.JSON(http.StatusOK, jsonHTTPResponse{true, "WireGuard service restarted successfully"})
	}
}

// FlushDNSCache flushes the system DNS cache
func FlushDNSCache(db store.IStore) echo.HandlerFunc {
	return func(c echo.Context) error {
		if !isAdmin(c) {
			return c.JSON(http.StatusForbidden, jsonHTTPResponse{false, "Only administrators can flush DNS cache"})
		}

		var err error
		var message string

		// Try different DNS cache flush methods
		if _, lookupErr := exec.LookPath("systemd-resolve"); lookupErr == nil {
			// systemd-resolved
			cmd := exec.Command("systemd-resolve", "--flush-caches")
			err = cmd.Run()
			message = "systemd DNS cache flushed successfully"
		} else if _, lookupErr := exec.LookPath("resolvectl"); lookupErr == nil {
			// newer systemd
			cmd := exec.Command("resolvectl", "flush-caches")
			err = cmd.Run()
			message = "systemd DNS cache flushed successfully"
		} else if _, lookupErr := exec.LookPath("systemctl"); lookupErr == nil {
			// Try restarting systemd-resolved
			cmd := exec.Command("systemctl", "restart", "systemd-resolved")
			err = cmd.Run()
			message = "systemd-resolved service restarted successfully"
		} else {
			// Fallback - just return success message
			message = "DNS cache flush attempted (no systemd-resolved found)"
		}

		if err != nil {
			return c.JSON(http.StatusInternalServerError, jsonHTTPResponse{false, "Failed to flush DNS cache: " + err.Error()})
		}

		return c.JSON(http.StatusOK, jsonHTTPResponse{true, message})
	}
}

// CheckForUpdates checks for system updates
func CheckForUpdates(db store.IStore) echo.HandlerFunc {
	return func(c echo.Context) error {
		if !isAdmin(c) {
			return c.JSON(http.StatusForbidden, jsonHTTPResponse{false, "Only administrators can check for updates"})
		}

		var output []byte
		var err error

		// Try different package managers
		if _, lookupErr := exec.LookPath("apt-get"); lookupErr == nil {
			// Debian/Ubuntu
			cmd := exec.Command("apt", "list", "--upgradable")
			if output, err = cmd.Output(); err != nil {
				return c.JSON(http.StatusInternalServerError, jsonHTTPResponse{false, "Failed to check for updates: " + err.Error()})
			}
		} else if _, lookupErr := exec.LookPath("yum"); lookupErr == nil {
			// RHEL/CentOS
			cmd := exec.Command("yum", "check-update")
			output, _ = cmd.Output() // yum check-update returns non-zero even when successful
		} else if _, lookupErr := exec.LookPath("dnf"); lookupErr == nil {
			// Fedora
			cmd := exec.Command("dnf", "check-update")
			output, _ = cmd.Output() // dnf check-update returns non-zero even when successful
		} else {
			// Generic system info
			cmd := exec.Command("uname", "-a")
			if output, err = cmd.Output(); err != nil {
				output = []byte("System information not available")
			}
		}

		return c.JSON(http.StatusOK, jsonHTTPResponse{true, string(output)})
	}
}

// GenerateSystemReport generates a system report
func GenerateSystemReport(db store.IStore) echo.HandlerFunc {
	return func(c echo.Context) error {
		if !isAdmin(c) {
			return c.JSON(http.StatusForbidden, jsonHTTPResponse{false, "Only administrators can generate system reports"})
		}

		// Collect system information
		report := make(map[string]interface{})

		// Get system uptime
		cmd := exec.Command("uptime")
		if uptime, err := cmd.Output(); err == nil {
			report["uptime"] = string(uptime)
		} else {
			report["uptime"] = "Unable to get uptime"
		}

		// Get memory usage
		cmd = exec.Command("free", "-h")
		if memory, err := cmd.Output(); err == nil {
			report["memory"] = string(memory)
		} else {
			report["memory"] = "Unable to get memory info"
		}

		// Get disk usage
		cmd = exec.Command("df", "-h")
		if disk, err := cmd.Output(); err == nil {
			report["disk"] = string(disk)
		} else {
			report["disk"] = "Unable to get disk info"
		}

		// Get WireGuard status
		cmd = exec.Command("systemctl", "status", "wg-quick@wg0")
		if wgStatus, err := cmd.Output(); err == nil {
			report["wireguard_status"] = string(wgStatus)
		} else {
			report["wireguard_status"] = "WireGuard service not running or not found"
		}

		// Get system logs
		cmd = exec.Command("journalctl", "-n", "50", "--no-pager")
		if logs, err := cmd.Output(); err == nil {
			report["system_logs"] = string(logs)
		} else {
			report["system_logs"] = "Unable to get system logs"
		}

		return c.JSON(http.StatusOK, map[string]interface{}{
			"success": true,
			"message": report,
		})
	}
}

// GetSystemLogs retrieves system logs
func GetSystemLogs(db store.IStore) echo.HandlerFunc {
	return func(c echo.Context) error {
		if !isAdmin(c) {
			return c.JSON(http.StatusForbidden, jsonHTTPResponse{false, "Only administrators can view system logs"})
		}

		level := c.QueryParam("level")
		if level == "" {
			level = "info"
		}

		// Map level to journalctl priority numbers
		var priority string
		switch level {
		case "error":
			priority = "3" // err
		case "warning":
			priority = "4" // warning
		case "info":
			priority = "6" // info
		case "debug":
			priority = "7" // debug
		default:
			priority = "6" // info
		}

		// Get logs based on level
		cmd := exec.Command("journalctl", "-n", "100", "--no-pager", "-p", priority, "--since", "1 hour ago")
		output, err := cmd.Output()
		if err != nil {
			// If journalctl fails, try alternative approach without priority filter
			cmd = exec.Command("journalctl", "-n", "100", "--no-pager", "--since", "1 hour ago")
			if output, err = cmd.Output(); err != nil {
				// Last resort - just get recent logs without any filters
				cmd = exec.Command("journalctl", "-n", "50", "--no-pager")
				if output, err = cmd.Output(); err != nil {
					return c.JSON(http.StatusInternalServerError, jsonHTTPResponse{false, "Failed to retrieve system logs: " + err.Error()})
				}
			}
		}

		return c.JSON(http.StatusOK, jsonHTTPResponse{true, string(output)})
	}
}

// ClearSystemLogs clears system logs
func ClearSystemLogs(db store.IStore) echo.HandlerFunc {
	return func(c echo.Context) error {
		if !isAdmin(c) {
			return c.JSON(http.StatusForbidden, jsonHTTPResponse{false, "Only administrators can clear system logs"})
		}

		// Clear system logs
		cmd := exec.Command("journalctl", "--vacuum-time=1s")
		if err := cmd.Run(); err != nil {
			return c.JSON(http.StatusInternalServerError, jsonHTTPResponse{false, "Failed to clear system logs"})
		}

		return c.JSON(http.StatusOK, jsonHTTPResponse{true, "System logs cleared successfully"})
	}
}
