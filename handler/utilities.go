package handler

import (
	"encoding/json"
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
				Admin:      user.Role == model.RoleAdmin,
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

		// Execute systemctl restart command
		cmd := exec.Command("systemctl", "restart", "wg-quick@wg0")
		if err := cmd.Run(); err != nil {
			return c.JSON(http.StatusInternalServerError, jsonHTTPResponse{false, "Failed to restart WireGuard service"})
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
		uptime, err := cmd.Output()
		if err == nil {
			report["uptime"] = string(uptime)
		}

		// Get memory usage
		cmd = exec.Command("free", "-h")
		memory, err := cmd.Output()
		if err == nil {
			report["memory"] = string(memory)
		}

		// Get disk usage
		cmd = exec.Command("df", "-h")
		disk, err := cmd.Output()
		if err == nil {
			report["disk"] = string(disk)
		}

		// Get WireGuard status
		cmd = exec.Command("systemctl", "status", "wg-quick@wg0")
		wgStatus, err := cmd.Output()
		if err == nil {
			report["wireguard_status"] = string(wgStatus)
		}

		// Get system logs
		cmd = exec.Command("journalctl", "-n", "100", "--no-pager")
		logs, err := cmd.Output()
		if err == nil {
			report["system_logs"] = string(logs)
		}

		// Convert report to JSON
		jsonReport, err := json.Marshal(report)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, jsonHTTPResponse{false, "Failed to generate report"})
		}

		return c.JSON(http.StatusOK, jsonHTTPResponse{true, string(jsonReport)})
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

		// Get logs based on level
		cmd := exec.Command("journalctl", "-n", "100", "--no-pager", "--priority="+level)
		output, err := cmd.Output()
		if err != nil {
			return c.JSON(http.StatusInternalServerError, jsonHTTPResponse{false, "Failed to retrieve system logs"})
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