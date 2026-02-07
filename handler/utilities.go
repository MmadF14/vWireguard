package handler

import (
	"net/http"
	"os/exec"

	"github.com/MmadF14/vwireguard/model"
	"github.com/MmadF14/vwireguard/store"
	"github.com/labstack/echo/v4"
)

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

func RestartWireGuardService(db store.IStore) echo.HandlerFunc {
	return func(c echo.Context) error {
		if !isAdmin(c) {
			return c.JSON(http.StatusForbidden, jsonHTTPResponse{false, "Only administrators can restart the service"})
		}

		cmd := exec.Command("systemctl", "restart", "wg-quick@wg0")
		output, err := cmd.CombinedOutput()
		if err != nil {
			cmd = exec.Command("systemctl", "restart", "vwireguard")
			if output, err = cmd.CombinedOutput(); err != nil {
				return c.JSON(http.StatusInternalServerError, jsonHTTPResponse{false, "Failed to restart WireGuard service: " + string(output)})
			}
			return c.JSON(http.StatusOK, jsonHTTPResponse{true, "vWireguard service restarted successfully"})
		}

		return c.JSON(http.StatusOK, jsonHTTPResponse{true, "WireGuard service restarted successfully"})
	}
}

func FlushDNSCache(db store.IStore) echo.HandlerFunc {
	return func(c echo.Context) error {
		if !isAdmin(c) {
			return c.JSON(http.StatusForbidden, jsonHTTPResponse{false, "Only administrators can flush DNS cache"})
		}

		var err error
		var message string

		if _, lookupErr := exec.LookPath("systemd-resolve"); lookupErr == nil {
			cmd := exec.Command("systemd-resolve", "--flush-caches")
			err = cmd.Run()
			message = "systemd DNS cache flushed successfully"
		} else if _, lookupErr := exec.LookPath("resolvectl"); lookupErr == nil {
			cmd := exec.Command("resolvectl", "flush-caches")
			err = cmd.Run()
			message = "systemd DNS cache flushed successfully"
		} else if _, lookupErr := exec.LookPath("systemctl"); lookupErr == nil {
			cmd := exec.Command("systemctl", "restart", "systemd-resolved")
			err = cmd.Run()
			message = "systemd-resolved service restarted successfully"
		} else {
			message = "DNS cache flush attempted (no systemd-resolved found)"
		}

		if err != nil {
			return c.JSON(http.StatusInternalServerError, jsonHTTPResponse{false, "Failed to flush DNS cache: " + err.Error()})
		}

		return c.JSON(http.StatusOK, jsonHTTPResponse{true, message})
	}
}

func CheckForUpdates(db store.IStore) echo.HandlerFunc {
	return func(c echo.Context) error {
		if !isAdmin(c) {
			return c.JSON(http.StatusForbidden, jsonHTTPResponse{false, "Only administrators can check for updates"})
		}

		var output []byte
		var err error

		var message string

		if _, lookupErr := exec.LookPath("apt"); lookupErr == nil {
			message = "Checking for available package updates on Debian/Ubuntu system...\n\n"
			cmd := exec.Command("apt", "list", "--upgradable")
			if output, err = cmd.Output(); err != nil {
				return c.JSON(http.StatusInternalServerError, jsonHTTPResponse{false, "Failed to check for updates: " + err.Error()})
			}
			if len(output) > 0 {
				message += string(output)
			} else {
				message += "No package updates available."
			}
		} else if _, lookupErr := exec.LookPath("yum"); lookupErr == nil {
			message = "Checking for available package updates on RHEL/CentOS system...\n\n"
			cmd := exec.Command("yum", "check-update")
			output, _ = cmd.Output()
			if len(output) > 0 {
				message += string(output)
			} else {
				message += "No package updates available."
			}
		} else if _, lookupErr := exec.LookPath("dnf"); lookupErr == nil {
			message = "Checking for available package updates on Fedora system...\n\n"
			cmd := exec.Command("dnf", "check-update")
			output, _ = cmd.Output()
			if len(output) > 0 {
				message += string(output)
			} else {
				message += "No package updates available."
			}
		} else {
			message = "Package manager not found. Showing system information instead:\n\n"
			cmd := exec.Command("uname", "-a")
			if output, err = cmd.Output(); err != nil {
				message += "System information not available"
			} else {
				message += string(output)
			}
		}

		return c.JSON(http.StatusOK, jsonHTTPResponse{true, message})
	}
}

func GenerateSystemReport(db store.IStore) echo.HandlerFunc {
	return func(c echo.Context) error {
		if !isAdmin(c) {
			return c.JSON(http.StatusForbidden, jsonHTTPResponse{false, "Only administrators can generate system reports"})
		}

		report := make(map[string]interface{})

		cmd := exec.Command("uptime")
		if uptime, err := cmd.Output(); err == nil {
			report["uptime"] = string(uptime)
		} else {
			report["uptime"] = "Unable to get uptime"
		}

		cmd = exec.Command("free", "-h")
		if memory, err := cmd.Output(); err == nil {
			report["memory"] = string(memory)
		} else {
			report["memory"] = "Unable to get memory info"
		}

		cmd = exec.Command("df", "-h")
		if disk, err := cmd.Output(); err == nil {
			report["disk"] = string(disk)
		} else {
			report["disk"] = "Unable to get disk info"
		}

		cmd = exec.Command("systemctl", "status", "wg-quick@wg0")
		if wgStatus, err := cmd.Output(); err == nil {
			report["wireguard_status"] = string(wgStatus)
		} else {
			report["wireguard_status"] = "WireGuard service not running or not found"
		}

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

func GetSystemLogs(db store.IStore) echo.HandlerFunc {
	return func(c echo.Context) error {
		if !isAdmin(c) {
			return c.JSON(http.StatusForbidden, jsonHTTPResponse{false, "Only administrators can view system logs"})
		}

		level := c.QueryParam("level")
		if level == "" {
			level = "info"
		}

		var priority string
		switch level {
		case "error":
			priority = "3"
		case "info":
			priority = "6"
		default:
			priority = "6"
		}

		cmd := exec.Command("journalctl", "-n", "100", "--no-pager", "-p", priority, "--since", "1 hour ago")
		output, err := cmd.Output()
		if err != nil {
			cmd = exec.Command("journalctl", "-n", "100", "--no-pager", "--since", "1 hour ago")
			if output, err = cmd.Output(); err != nil {
				cmd = exec.Command("journalctl", "-n", "50", "--no-pager")
				if output, err = cmd.Output(); err != nil {
					return c.JSON(http.StatusInternalServerError, jsonHTTPResponse{false, "Failed to retrieve system logs: " + err.Error()})
				}
			}
		}

		return c.JSON(http.StatusOK, jsonHTTPResponse{true, string(output)})
	}
}

func ClearSystemLogs(db store.IStore) echo.HandlerFunc {
	return func(c echo.Context) error {
		if !isAdmin(c) {
			return c.JSON(http.StatusForbidden, jsonHTTPResponse{false, "Only administrators can clear system logs"})
		}

		var err error
		var message string

		cmd := exec.Command("journalctl", "--vacuum-time=1d")
		if err = cmd.Run(); err == nil {
			message = "System logs older than 1 day cleared successfully"
		} else {
			cmd = exec.Command("journalctl", "--vacuum-size=100M")
			if err = cmd.Run(); err == nil {
				message = "System logs trimmed to 100MB successfully"
			} else {
				cmd = exec.Command("systemctl", "kill", "--kill-who=main", "--signal=SIGUSR2", "systemd-journald.service")
				if err = cmd.Run(); err == nil {
					message = "System logs rotated successfully"
				} else {
					return c.JSON(http.StatusInternalServerError, jsonHTTPResponse{false, "Failed to clear system logs: " + err.Error()})
				}
			}
		}

		return c.JSON(http.StatusOK, jsonHTTPResponse{true, message})
	}
}
