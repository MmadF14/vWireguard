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

		// Execute system DNS flush command
		cmd := exec.Command("systemd-resolve", "--flush-caches")
		if err := cmd.Run(); err != nil {
			return c.JSON(http.StatusInternalServerError, jsonHTTPResponse{false, "Failed to flush DNS cache"})
		}

		return c.JSON(http.StatusOK, jsonHTTPResponse{true, "DNS cache flushed successfully"})
	}
}

// CheckForUpdates checks for system updates
func CheckForUpdates(db store.IStore) echo.HandlerFunc {
	return func(c echo.Context) error {
		if !isAdmin(c) {
			return c.JSON(http.StatusForbidden, jsonHTTPResponse{false, "Only administrators can check for updates"})
		}

		// Execute apt update and upgrade check
		cmd := exec.Command("apt-get", "update")
		if err := cmd.Run(); err != nil {
			return c.JSON(http.StatusInternalServerError, jsonHTTPResponse{false, "Failed to check for updates"})
		}

		cmd = exec.Command("apt-get", "upgrade", "-s")
		output, err := cmd.Output()
		if err != nil {
			return c.JSON(http.StatusInternalServerError, jsonHTTPResponse{false, "Failed to check for updates"})
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

		// Map level to journalctl priority
		var priority string
		switch level {
		case "error":
			priority = "err"
		case "warning":
			priority = "warning"
		case "info":
			priority = "info"
		case "debug":
			priority = "debug"
		default:
			priority = "info"
		}

		// Get logs based on level
		cmd := exec.Command("journalctl", "-n", "100", "--no-pager", "-p", priority)
		output, err := cmd.Output()
		if err != nil {
			// If journalctl fails, try alternative approach
			cmd = exec.Command("journalctl", "-n", "100", "--no-pager")
			if output, err = cmd.Output(); err != nil {
				return c.JSON(http.StatusInternalServerError, jsonHTTPResponse{false, "Failed to retrieve system logs: " + err.Error()})
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
