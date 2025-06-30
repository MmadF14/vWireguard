package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/rs/xid"

	"github.com/MmadF14/vwireguard/model"
	"github.com/MmadF14/vwireguard/store"
)

// TunnelsPage handler
func TunnelsPage(db store.IStore) echo.HandlerFunc {
	return func(c echo.Context) error {
		// Get tunnels
		tunnels, err := db.GetTunnels()
		if err != nil {
			return c.Render(http.StatusInternalServerError, "tunnels.html", map[string]interface{}{
				"baseData": map[string]interface{}{
					"Active": "tunnels",
				},
				"error": fmt.Sprintf("Failed to get tunnels: %v", err),
			})
		}

		return c.Render(http.StatusOK, "tunnels.html", map[string]interface{}{
			"baseData": map[string]interface{}{
				"Active": "tunnels",
			},
			"tunnels": tunnels,
		})
	}
}

// GetTunnels handler returns a JSON list of all tunnels
func GetTunnels(db store.IStore) echo.HandlerFunc {
	return func(c echo.Context) error {
		tunnels, err := db.GetTunnels()
		if err != nil {
			return c.JSON(http.StatusInternalServerError, jsonHTTPResponse{
				false, fmt.Sprintf("Cannot get tunnel list: %v", err),
			})
		}

		return c.JSON(http.StatusOK, tunnels)
	}
}

// GetTunnel handler returns a JSON object of single tunnel
func GetTunnel(db store.IStore) echo.HandlerFunc {
	return func(c echo.Context) error {
		tunnelID := c.Param("id")

		tunnel, err := db.GetTunnelByID(tunnelID)
		if err != nil {
			return c.JSON(http.StatusNotFound, jsonHTTPResponse{false, "Tunnel not found"})
		}

		return c.JSON(http.StatusOK, tunnel)
	}
}

// NewTunnel handler creates a new tunnel
func NewTunnel(db store.IStore) echo.HandlerFunc {
	return func(c echo.Context) error {
		var tunnel model.Tunnel

		// Bind JSON data
		if err := c.Bind(&tunnel); err != nil {
			return c.JSON(http.StatusBadRequest, jsonHTTPResponse{false, "Invalid tunnel data"})
		}

		// Validate required fields
		if tunnel.Name == "" {
			return c.JSON(http.StatusBadRequest, jsonHTTPResponse{false, "Tunnel name is required"})
		}

		if tunnel.Type == "" {
			return c.JSON(http.StatusBadRequest, jsonHTTPResponse{false, "Tunnel type is required"})
		}

		// Set default values
		tunnel.ID = xid.New().String()
		tunnel.Status = model.TunnelStatusInactive
		tunnel.Enabled = true
		tunnel.CreatedBy = currentUser(c)
		tunnel.CreatedAt = time.Now().UTC()
		tunnel.UpdatedAt = time.Now().UTC()

		// Initialize config if empty
		if tunnel.Config == nil {
			tunnel.Config = make(map[string]interface{})
		}

		// Save tunnel
		if err := db.SaveTunnel(tunnel); err != nil {
			return c.JSON(http.StatusInternalServerError, jsonHTTPResponse{false, fmt.Sprintf("Failed to save tunnel: %v", err)})
		}

		return c.JSON(http.StatusOK, jsonHTTPResponse{true, "Tunnel created successfully"})
	}
}

// UpdateTunnel handler updates an existing tunnel
func UpdateTunnel(db store.IStore) echo.HandlerFunc {
	return func(c echo.Context) error {
		tunnelID := c.Param("id")

		// Get existing tunnel
		existingTunnel, err := db.GetTunnelByID(tunnelID)
		if err != nil {
			return c.JSON(http.StatusNotFound, jsonHTTPResponse{false, "Tunnel not found"})
		}

		var updatedTunnel model.Tunnel
		if err := c.Bind(&updatedTunnel); err != nil {
			return c.JSON(http.StatusBadRequest, jsonHTTPResponse{false, "Invalid tunnel data"})
		}

		// Preserve certain fields
		updatedTunnel.ID = existingTunnel.ID
		updatedTunnel.CreatedAt = existingTunnel.CreatedAt
		updatedTunnel.CreatedBy = existingTunnel.CreatedBy
		updatedTunnel.UpdatedAt = time.Now().UTC()
		updatedTunnel.BytesIn = existingTunnel.BytesIn
		updatedTunnel.BytesOut = existingTunnel.BytesOut
		updatedTunnel.LastSeen = existingTunnel.LastSeen

		// Save updated tunnel
		if err := db.SaveTunnel(updatedTunnel); err != nil {
			return c.JSON(http.StatusInternalServerError, jsonHTTPResponse{false, fmt.Sprintf("Failed to update tunnel: %v", err)})
		}

		return c.JSON(http.StatusOK, jsonHTTPResponse{true, "Tunnel updated successfully"})
	}
}

// DeleteTunnel handler removes a tunnel
func DeleteTunnel(db store.IStore) echo.HandlerFunc {
	return func(c echo.Context) error {
		tunnelID := c.Param("id")

		// Check if tunnel exists
		_, err := db.GetTunnelByID(tunnelID)
		if err != nil {
			return c.JSON(http.StatusNotFound, jsonHTTPResponse{false, "Tunnel not found"})
		}

		// Delete tunnel
		if err := db.DeleteTunnel(tunnelID); err != nil {
			return c.JSON(http.StatusInternalServerError, jsonHTTPResponse{false, fmt.Sprintf("Failed to delete tunnel: %v", err)})
		}

		return c.JSON(http.StatusOK, jsonHTTPResponse{true, "Tunnel deleted successfully"})
	}
}

// SetTunnelStatus handler enables/disables a tunnel
func SetTunnelStatus(db store.IStore) echo.HandlerFunc {
	return func(c echo.Context) error {
		tunnelID := c.Param("id")

		data := make(map[string]interface{})
		if err := json.NewDecoder(c.Request().Body).Decode(&data); err != nil {
			return c.JSON(http.StatusBadRequest, jsonHTTPResponse{false, "Invalid request data"})
		}

		enabled, ok := data["enabled"].(bool)
		if !ok {
			return c.JSON(http.StatusBadRequest, jsonHTTPResponse{false, "Invalid enabled status"})
		}

		// Get existing tunnel
		tunnel, err := db.GetTunnelByID(tunnelID)
		if err != nil {
			return c.JSON(http.StatusNotFound, jsonHTTPResponse{false, "Tunnel not found"})
		}

		// Update status
		tunnel.Enabled = enabled
		if enabled {
			tunnel.Status = model.TunnelStatusActive
		} else {
			tunnel.Status = model.TunnelStatusInactive
		}
		tunnel.UpdatedAt = time.Now().UTC()

		// Save tunnel
		if err := db.SaveTunnel(tunnel); err != nil {
			return c.JSON(http.StatusInternalServerError, jsonHTTPResponse{false, fmt.Sprintf("Failed to update tunnel status: %v", err)})
		}

		return c.JSON(http.StatusOK, jsonHTTPResponse{true, "Tunnel status updated successfully"})
	}
}

// StartTunnel handler starts a tunnel
func StartTunnel(db store.IStore) echo.HandlerFunc {
	return func(c echo.Context) error {
		tunnelID := c.Param("id")

		// Get tunnel
		tunnel, err := db.GetTunnelByID(tunnelID)
		if err != nil {
			return c.JSON(http.StatusNotFound, jsonHTTPResponse{false, "Tunnel not found"})
		}

		if !tunnel.Enabled {
			return c.JSON(http.StatusBadRequest, jsonHTTPResponse{false, "Tunnel is disabled"})
		}

		// Here you would implement the actual tunnel starting logic
		// For now, just update the status
		err = db.UpdateTunnelStatus(tunnelID, model.TunnelStatusActive)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, jsonHTTPResponse{false, fmt.Sprintf("Failed to start tunnel: %v", err)})
		}

		return c.JSON(http.StatusOK, jsonHTTPResponse{true, "Tunnel started successfully"})
	}
}

// StopTunnel handler stops a tunnel
func StopTunnel(db store.IStore) echo.HandlerFunc {
	return func(c echo.Context) error {
		tunnelID := c.Param("id")

		// Get tunnel
		_, err := db.GetTunnelByID(tunnelID)
		if err != nil {
			return c.JSON(http.StatusNotFound, jsonHTTPResponse{false, "Tunnel not found"})
		}

		// Here you would implement the actual tunnel stopping logic
		// For now, just update the status
		err = db.UpdateTunnelStatus(tunnelID, model.TunnelStatusInactive)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, jsonHTTPResponse{false, fmt.Sprintf("Failed to stop tunnel: %v", err)})
		}

		return c.JSON(http.StatusOK, jsonHTTPResponse{true, "Tunnel stopped successfully"})
	}
}

// GetTunnelStats handler returns tunnel statistics
func GetTunnelStats(db store.IStore) echo.HandlerFunc {
	return func(c echo.Context) error {
		tunnelID := c.Param("id")

		tunnel, err := db.GetTunnelByID(tunnelID)
		if err != nil {
			return c.JSON(http.StatusNotFound, jsonHTTPResponse{false, "Tunnel not found"})
		}

		stats := map[string]interface{}{
			"id":         tunnel.ID,
			"name":       tunnel.Name,
			"status":     tunnel.Status,
			"bytes_in":   tunnel.BytesIn,
			"bytes_out":  tunnel.BytesOut,
			"last_seen":  tunnel.LastSeen,
			"created_at": tunnel.CreatedAt,
			"updated_at": tunnel.UpdatedAt,
		}

		return c.JSON(http.StatusOK, stats)
	}
}

// GetTunnelTypes handler returns available tunnel types
func GetTunnelTypes() echo.HandlerFunc {
	return func(c echo.Context) error {
		types := []map[string]interface{}{
			{
				"value":       string(model.TunnelTypeWireGuardToWireGuard),
				"label":       "WireGuard to WireGuard",
				"description": "Connect two WireGuard networks",
			},
			{
				"value":       string(model.TunnelTypeWireGuardToSSH),
				"label":       "WireGuard to SSH",
				"description": "SSH tunnel over WireGuard",
			},
			{
				"value":       string(model.TunnelTypeWireGuardToOpenVPN),
				"label":       "WireGuard to OpenVPN",
				"description": "Bridge WireGuard to OpenVPN",
			},
			{
				"value":       string(model.TunnelTypeWireGuardToL2TP),
				"label":       "WireGuard to L2TP",
				"description": "Bridge WireGuard to L2TP/IPSec",
			},
			{
				"value":       string(model.TunnelTypeWireGuardToSOCKS),
				"label":       "WireGuard to SOCKS",
				"description": "SOCKS proxy over WireGuard",
			},
			{
				"value":       string(model.TunnelTypeWireGuardToHTTP),
				"label":       "WireGuard to HTTP",
				"description": "HTTP proxy over WireGuard",
			},
			{
				"value":       string(model.TunnelTypePortForward),
				"label":       "Port Forward",
				"description": "Simple port forwarding",
			},
			{
				"value":       string(model.TunnelTypeReverse),
				"label":       "Reverse Tunnel",
				"description": "Reverse tunnel connection",
			},
		}

		return c.JSON(http.StatusOK, types)
	}
}
