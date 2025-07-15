package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/rs/xid"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"

	"github.com/MmadF14/vwireguard/model"
	"github.com/MmadF14/vwireguard/service"
	"github.com/MmadF14/vwireguard/store"
	"github.com/MmadF14/vwireguard/store/jsondb"
)

// TunnelsPage handler
func TunnelsPage(db store.IStore) echo.HandlerFunc {
	return func(c echo.Context) error {
		// Get current user info
		username := currentUser(c)
		isAdmin := isAdmin(c)

		// Get tunnels
		tunnels := make([]model.Tunnel, 0)
		if db != nil {
			if dbTunnels, err := db.GetTunnels(); err == nil {
				tunnels = dbTunnels
				log.Printf("TunnelsPage: Found %d tunnels", len(tunnels))
				for i, tunnel := range tunnels {
					log.Printf("Tunnel %d: %s (%s) - Status: %s", i+1, tunnel.Name, tunnel.Type, tunnel.Status)
				}
			} else {
				log.Printf("TunnelsPage: Error getting tunnels: %v", err)
			}
		} else {
			log.Printf("TunnelsPage: Database is nil")
		}

		log.Printf("TunnelsPage: Rendering template with %d tunnels", len(tunnels))
		return c.Render(http.StatusOK, "tunnels.html", map[string]interface{}{
			"baseData": model.BaseData{Active: "tunnels", CurrentUser: username, Admin: isAdmin},
			"tunnels":  tunnels,
			"basePath": "/",
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
		log.Printf("NewTunnel: Handler called")

		var tunnelData struct {
			Name              string                       `json:"name"`
			Type              model.TunnelType             `json:"type"`
			Description       string                       `json:"description"`
			RouteAll          bool                         `json:"route_all"`
			ClientIDs         []string                     `json:"client_ids"`
			WGConfig          *model.WireGuardTunnelConfig `json:"wg_config,omitempty"`
			DokodemoConfig    *model.DokodemoTunnelConfig  `json:"dokodemo_config,omitempty"`
			PortForwardConfig *model.PortForwardConfig     `json:"port_forward_config,omitempty"`
			V2rayConfig       *model.V2rayTunnelConfig     `json:"v2ray_config,omitempty"`
		}

		// Bind JSON data
		if err := c.Bind(&tunnelData); err != nil {
			log.Printf("NewTunnel: Bind error: %v", err)
			return c.JSON(http.StatusBadRequest, jsonHTTPResponse{false, "Invalid tunnel data"})
		}

		log.Printf("NewTunnel: Received data - Name: %s, Type: %s", tunnelData.Name, tunnelData.Type)

		// Validate required fields
		if tunnelData.Name == "" {
			return c.JSON(http.StatusBadRequest, jsonHTTPResponse{false, "Tunnel name is required"})
		}

		if tunnelData.Type == "" {
			return c.JSON(http.StatusBadRequest, jsonHTTPResponse{false, "Tunnel type is required"})
		}

		// Validate type-specific configuration
		switch tunnelData.Type {
		case model.TunnelTypeWireGuardToWireGuard:
			if tunnelData.WGConfig == nil {
				return c.JSON(http.StatusBadRequest, jsonHTTPResponse{false, "WireGuard configuration is required"})
			}
			if tunnelData.WGConfig.RemoteEndpoint == "" || tunnelData.WGConfig.RemotePublicKey == "" {
				return c.JSON(http.StatusBadRequest, jsonHTTPResponse{false, "Remote endpoint and public key are required"})
			}
			// Handle keypair generation/validation
			if tunnelData.WGConfig.LocalPrivateKey == "" {
				// Generate new keypair if not provided
				privateKey, publicKey, err := generateWireGuardKeypair()
				if err != nil {
					return c.JSON(http.StatusInternalServerError, jsonHTTPResponse{false, "Failed to generate keypair"})
				}
				tunnelData.WGConfig.LocalPrivateKey = privateKey
				tunnelData.WGConfig.LocalPublicKey = publicKey
			} else {
				// Validate private key and derive public key
				privateKeyStr := strings.TrimSpace(tunnelData.WGConfig.LocalPrivateKey)
				privateKey, err := wgtypes.ParseKey(privateKeyStr)
				if err != nil {
					return c.JSON(http.StatusBadRequest, jsonHTTPResponse{false, "Invalid private key format: " + err.Error()})
				}

				// Ensure proper formatting and derive public key
				tunnelData.WGConfig.LocalPrivateKey = privateKey.String()
				tunnelData.WGConfig.LocalPublicKey = privateKey.PublicKey().String()
			}

		case model.TunnelTypeWireGuardToDokodemo:
			if tunnelData.DokodemoConfig == nil {
				return c.JSON(http.StatusBadRequest, jsonHTTPResponse{false, "Dokodemo configuration is required"})
			}
			if tunnelData.DokodemoConfig.Address == "" || tunnelData.DokodemoConfig.Port == 0 {
				return c.JSON(http.StatusBadRequest, jsonHTTPResponse{false, "Target address and port are required"})
			}
			if tunnelData.DokodemoConfig.Network == "" {
				tunnelData.DokodemoConfig.Network = "tcp" // Default to TCP
			}

		case model.TunnelTypePortForward:
			if tunnelData.PortForwardConfig == nil {
				return c.JSON(http.StatusBadRequest, jsonHTTPResponse{false, "Port forward configuration is required"})
			}
			if tunnelData.PortForwardConfig.RemoteHost == "" || tunnelData.PortForwardConfig.RemotePort == 0 {
				return c.JSON(http.StatusBadRequest, jsonHTTPResponse{false, "Remote host and port are required"})
			}
		case model.TunnelTypeWireGuardToV2ray:
			if tunnelData.V2rayConfig == nil {
				return c.JSON(http.StatusBadRequest, jsonHTTPResponse{false, "V2Ray configuration is required"})
			}

			// Basic required fields for all V2Ray protocols
			if tunnelData.V2rayConfig.Protocol == "" {
				return c.JSON(http.StatusBadRequest, jsonHTTPResponse{false, "V2Ray protocol is required"})
			}
			if tunnelData.V2rayConfig.RemoteAddress == "" {
				return c.JSON(http.StatusBadRequest, jsonHTTPResponse{false, "V2Ray remote address is required"})
			}
			if tunnelData.V2rayConfig.RemotePort == 0 {
				return c.JSON(http.StatusBadRequest, jsonHTTPResponse{false, "V2Ray remote port is required"})
			}
			if tunnelData.V2rayConfig.Security == "" {
				return c.JSON(http.StatusBadRequest, jsonHTTPResponse{false, "V2Ray security setting is required"})
			}
			if tunnelData.V2rayConfig.Network == "" {
				return c.JSON(http.StatusBadRequest, jsonHTTPResponse{false, "V2Ray network type is required"})
			}

			// Protocol-specific validation
			switch tunnelData.V2rayConfig.Protocol {
			case "vmess", "vless":
				if tunnelData.V2rayConfig.UUID == "" {
					return c.JSON(http.StatusBadRequest, jsonHTTPResponse{false, fmt.Sprintf("UUID is required for %s protocol", tunnelData.V2rayConfig.Protocol)})
				}
			case "trojan":
				if tunnelData.V2rayConfig.Password == "" {
					return c.JSON(http.StatusBadRequest, jsonHTTPResponse{false, "Password is required for Trojan protocol"})
				}
			default:
				return c.JSON(http.StatusBadRequest, jsonHTTPResponse{false, fmt.Sprintf("Unsupported V2Ray protocol: %s", tunnelData.V2rayConfig.Protocol)})
			}
		}

		// Create tunnel model
		tunnel := model.Tunnel{
			ID:                xid.New().String(),
			Name:              tunnelData.Name,
			Type:              tunnelData.Type,
			Description:       tunnelData.Description,
			Status:            model.TunnelStatusInactive,
			Enabled:           true,
			RouteAll:          tunnelData.RouteAll,
			ClientIDs:         tunnelData.ClientIDs,
			WGConfig:          tunnelData.WGConfig,
			DokodemoConfig:    tunnelData.DokodemoConfig,
			PortForwardConfig: tunnelData.PortForwardConfig,
			V2rayConfig:       tunnelData.V2rayConfig,
			Priority:          1,
			CreatedBy:         currentUser(c),
			CreatedAt:         time.Now().UTC(),
			UpdatedAt:         time.Now().UTC(),
		}

		// Save tunnel
		log.Printf("NewTunnel: Attempting to save tunnel - ID: %s, Name: %s", tunnel.ID, tunnel.Name)
		if err := db.SaveTunnel(tunnel); err != nil {
			log.Printf("NewTunnel: Save error: %v", err)
			return c.JSON(http.StatusInternalServerError, jsonHTTPResponse{false, fmt.Sprintf("Failed to save tunnel: %v", err)})
		}

		log.Printf("NewTunnel: Tunnel saved successfully - ID: %s", tunnel.ID)
		return c.JSON(http.StatusOK, jsonHTTPResponse{true, "Tunnel created successfully"})
	}
}

// CreateV2rayTunnel creates a WireGuard to V2Ray tunnel
func CreateV2rayTunnel(db store.IStore) echo.HandlerFunc {
	return func(c echo.Context) error {
		var data struct {
			Name        string                       `json:"name"`
			Description string                       `json:"description"`
			RouteAll    bool                         `json:"route_all"`
			ClientIDs   []string                     `json:"client_ids"`
			WGConfig    *model.WireGuardTunnelConfig `json:"wg_config"`
			V2rayConfig *model.V2rayTunnelConfig     `json:"v2ray_config"`
		}
		if err := c.Bind(&data); err != nil {
			return c.JSON(http.StatusBadRequest, jsonHTTPResponse{false, "Invalid tunnel data"})
		}
		if data.Name == "" || data.WGConfig == nil || data.V2rayConfig == nil {
			return c.JSON(http.StatusBadRequest, jsonHTTPResponse{false, "Missing required fields"})
		}
		t := model.Tunnel{
			ID:          xid.New().String(),
			Name:        data.Name,
			Type:        model.TunnelTypeWireGuardToV2ray,
			Description: data.Description,
			Status:      model.TunnelStatusInactive,
			Enabled:     true,
			RouteAll:    data.RouteAll,
			ClientIDs:   data.ClientIDs,
			WGConfig:    data.WGConfig,
			V2rayConfig: data.V2rayConfig,
			Priority:    1,
			CreatedBy:   currentUser(c),
			CreatedAt:   time.Now().UTC(),
			UpdatedAt:   time.Now().UTC(),
		}
		if t.WGConfig.LocalPrivateKey == "" {
			priv, pub, err := generateWireGuardKeypair()
			if err != nil {
				return c.JSON(http.StatusInternalServerError, jsonHTTPResponse{false, "Failed to generate keypair"})
			}
			t.WGConfig.LocalPrivateKey = priv
			t.WGConfig.LocalPublicKey = pub
		}
		if err := db.SaveTunnel(t); err != nil {
			return c.JSON(http.StatusInternalServerError, jsonHTTPResponse{false, fmt.Sprintf("Failed to save tunnel: %v", err)})
		}
		return c.JSON(http.StatusOK, jsonHTTPResponse{true, "Tunnel created successfully"})
	}
}

// UpdateV2rayTunnel updates WireGuard to V2Ray tunnel
func UpdateV2rayTunnel(db store.IStore) echo.HandlerFunc {
	return func(c echo.Context) error {
		id := c.Param("id")
		existing, err := db.GetTunnelByID(id)
		if err != nil {
			return c.JSON(http.StatusNotFound, jsonHTTPResponse{false, "Tunnel not found"})
		}
		if existing.Status == model.TunnelStatusActive {
			return c.JSON(http.StatusBadRequest, jsonHTTPResponse{false, "Cannot edit active tunnel. Please stop the tunnel first."})
		}
		var upd model.Tunnel
		if err := c.Bind(&upd); err != nil {
			return c.JSON(http.StatusBadRequest, jsonHTTPResponse{false, "Invalid tunnel data"})
		}
		upd.ID = existing.ID
		upd.Type = model.TunnelTypeWireGuardToV2ray
		upd.CreatedAt = existing.CreatedAt
		upd.CreatedBy = existing.CreatedBy
		upd.UpdatedAt = time.Now().UTC()
		upd.BytesIn = existing.BytesIn
		upd.BytesOut = existing.BytesOut
		upd.LastSeen = existing.LastSeen
		upd.Enabled = existing.Enabled
		upd.Status = existing.Status
		if err := db.SaveTunnel(upd); err != nil {
			return c.JSON(http.StatusInternalServerError, jsonHTTPResponse{false, fmt.Sprintf("Failed to update tunnel: %v", err)})
		}
		return c.JSON(http.StatusOK, jsonHTTPResponse{true, "Tunnel updated successfully"})
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

		// Prevent editing active tunnels - user must stop first
		if existingTunnel.Status == model.TunnelStatusActive {
			return c.JSON(http.StatusBadRequest, jsonHTTPResponse{false, "Cannot edit active tunnel. Please stop the tunnel first."})
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

		// Preserve enabled status and current status
		updatedTunnel.Enabled = existingTunnel.Enabled
		updatedTunnel.Status = existingTunnel.Status

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

// EnableTunnel sets a tunnel to enabled state
func EnableTunnel(db store.IStore) echo.HandlerFunc {
	return func(c echo.Context) error {
		tunnelID := c.Param("id")
		tunnel, err := db.GetTunnelByID(tunnelID)
		if err != nil {
			return c.JSON(http.StatusNotFound, jsonHTTPResponse{false, "Tunnel not found"})
		}
		tunnel.Enabled = true
		tunnel.Status = model.TunnelStatusActive
		tunnel.UpdatedAt = time.Now().UTC()
		if err := db.SaveTunnel(tunnel); err != nil {
			return c.JSON(http.StatusInternalServerError, jsonHTTPResponse{false, "Failed to enable tunnel"})
		}
		return c.JSON(http.StatusOK, jsonHTTPResponse{true, "Tunnel enabled"})
	}
}

// DisableTunnel sets a tunnel to disabled state
func DisableTunnel(db store.IStore) echo.HandlerFunc {
	return func(c echo.Context) error {
		tunnelID := c.Param("id")
		tunnel, err := db.GetTunnelByID(tunnelID)
		if err != nil {
			return c.JSON(http.StatusNotFound, jsonHTTPResponse{false, "Tunnel not found"})
		}
		tunnel.Enabled = false
		tunnel.Status = model.TunnelStatusInactive
		tunnel.UpdatedAt = time.Now().UTC()
		if err := db.SaveTunnel(tunnel); err != nil {
			return c.JSON(http.StatusInternalServerError, jsonHTTPResponse{false, "Failed to disable tunnel"})
		}
		return c.JSON(http.StatusOK, jsonHTTPResponse{true, "Tunnel disabled"})
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

		// Implement actual tunnel starting logic based on type
		switch tunnel.Type {
		case model.TunnelTypeWireGuardToWireGuard:
			if err := startWireGuardTunnel(tunnel); err != nil {
				log.Printf("Failed to start WireGuard tunnel: %v", err)
				return c.JSON(http.StatusInternalServerError, jsonHTTPResponse{false, "Failed to start WireGuard tunnel: " + err.Error()})
			}
		case model.TunnelTypeWireGuardToV2ray:
			if err := startV2rayTunnel(tunnel); err != nil {
				log.Printf("Failed to start V2Ray tunnel: %v", err)
				return c.JSON(http.StatusInternalServerError, jsonHTTPResponse{false, "Failed to start V2Ray tunnel: " + err.Error()})
			}
		default:
			log.Printf("Tunnel type %s not implemented yet", tunnel.Type)
			return c.JSON(http.StatusNotImplemented, jsonHTTPResponse{false, fmt.Sprintf("Tunnel type %s not implemented yet", tunnel.Type)})
		}

		// Update status
		err = db.UpdateTunnelStatus(tunnelID, model.TunnelStatusActive)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, jsonHTTPResponse{false, fmt.Sprintf("Failed to update tunnel status: %v", err)})
		}

		return c.JSON(http.StatusOK, jsonHTTPResponse{true, "Tunnel started successfully"})
	}
}

// StopTunnel handler stops a tunnel
func StopTunnel(db store.IStore) echo.HandlerFunc {
	return func(c echo.Context) error {
		tunnelID := c.Param("id")

		// Get tunnel
		tunnel, err := db.GetTunnelByID(tunnelID)
		if err != nil {
			return c.JSON(http.StatusNotFound, jsonHTTPResponse{false, "Tunnel not found"})
		}

		// Implement actual tunnel stopping logic based on type
		switch tunnel.Type {
		case model.TunnelTypeWireGuardToWireGuard:
			if err := stopWireGuardTunnel(tunnel); err != nil {
				log.Printf("Failed to stop WireGuard tunnel: %v", err)
				return c.JSON(http.StatusInternalServerError, jsonHTTPResponse{false, "Failed to stop WireGuard tunnel: " + err.Error()})
			}
		case model.TunnelTypeWireGuardToV2ray:
			if err := stopV2rayTunnel(tunnel); err != nil {
				log.Printf("Failed to stop V2Ray tunnel: %v", err)
				return c.JSON(http.StatusInternalServerError, jsonHTTPResponse{false, "Failed to stop V2Ray tunnel: " + err.Error()})
			}
		default:
			log.Printf("Tunnel type %s not implemented yet", tunnel.Type)
			return c.JSON(http.StatusNotImplemented, jsonHTTPResponse{false, fmt.Sprintf("Tunnel type %s not implemented yet", tunnel.Type)})
		}

		// Update status
		err = db.UpdateTunnelStatus(tunnelID, model.TunnelStatusInactive)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, jsonHTTPResponse{false, fmt.Sprintf("Failed to update tunnel status: %v", err)})
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

		// Update tunnel stats from WireGuard interface if tunnel is active
		if tunnel.Status == model.TunnelStatusActive {
			if err := updateTunnelStatsFromWG(db, tunnel); err != nil {
				log.Printf("Failed to update tunnel stats: %v", err)
			}
			// Reload tunnel data after update
			tunnel, _ = db.GetTunnelByID(tunnelID)
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
				"value":       string(model.TunnelTypeWireGuardToDokodemo),
				"label":       "WireGuard to Dokodemo Door",
				"description": "Transparent proxy using Dokodemo Door",
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
				"value":       string(model.TunnelTypeWireGuardToV2ray),
				"label":       "WireGuard to V2Ray",
				"description": "Route traffic via V2Ray outbound",
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

// generateWireGuardKeypair generates a new WireGuard keypair
func generateWireGuardKeypair() (privateKey, publicKey string, err error) {
	key, err := wgtypes.GeneratePrivateKey()
	if err != nil {
		return "", "", err
	}
	return key.String(), key.PublicKey().String(), nil
}

// GenerateKeypair generates a new WireGuard keypair or derives public key from private key
func GenerateKeypair() echo.HandlerFunc {
	return func(c echo.Context) error {
		var requestData struct {
			PrivateKey string `json:"private_key,omitempty"`
		}

		// Try to bind request data (might be empty for new generation)
		c.Bind(&requestData)

		var privateKeyStr, publicKeyStr string
		var err error

		if requestData.PrivateKey != "" {
			// Generate public key from provided private key
			privateKeyStr = strings.TrimSpace(requestData.PrivateKey)

			privateKey, err := wgtypes.ParseKey(privateKeyStr)
			if err != nil {
				return c.JSON(http.StatusBadRequest, jsonHTTPResponse{false, "Invalid private key format: " + err.Error()})
			}

			// Regenerate to ensure proper formatting
			privateKeyStr = privateKey.String()
			publicKeyStr = privateKey.PublicKey().String()
		} else {
			// Generate new keypair
			log.Printf("GenerateKeypair: Generating new keypair")
			privateKeyStr, publicKeyStr, err = generateWireGuardKeypair()
			if err != nil {
				log.Printf("GenerateKeypair: Failed to generate keypair: %v", err)
				return c.JSON(http.StatusInternalServerError, jsonHTTPResponse{false, "Failed to generate keypair"})
			}
		}

		log.Printf("GenerateKeypair: Success - Public key: %s", publicKeyStr)
		return c.JSON(http.StatusOK, map[string]interface{}{
			"success":     true,
			"private_key": privateKeyStr,
			"public_key":  publicKeyStr,
		})
	}
}

// GeneratePreSharedKey generates a new WireGuard PreShared Key
func GeneratePreSharedKey() echo.HandlerFunc {
	return func(c echo.Context) error {
		key, err := wgtypes.GenerateKey()
		if err != nil {
			log.Printf("GeneratePreSharedKey: Failed to generate key: %v", err)
			return c.JSON(http.StatusInternalServerError, jsonHTTPResponse{false, "Failed to generate PreShared Key"})
		}

		log.Printf("GeneratePreSharedKey: Success - Key generated")
		return c.JSON(http.StatusOK, map[string]interface{}{
			"success":       true,
			"preshared_key": key.String(),
		})
	}
}

// CleanupTunnels handler removes corrupted tunnel records
func CleanupTunnels(db store.IStore) echo.HandlerFunc {
	return func(c echo.Context) error {
		// Cast to JsonDB to access cleanup method
		if jsonDB, ok := db.(*jsondb.JsonDB); ok {
			if err := jsonDB.CleanupCorruptedTunnels(); err != nil {
				return c.JSON(http.StatusInternalServerError, jsonHTTPResponse{false, "Failed to cleanup tunnels: " + err.Error()})
			}
			return c.JSON(http.StatusOK, jsonHTTPResponse{true, "Tunnel cleanup completed successfully"})
		}
		return c.JSON(http.StatusInternalServerError, jsonHTTPResponse{false, "Cleanup not supported for this database type"})
	}
}

// DeleteAllTunnels handler removes all tunnel records (emergency cleanup)
func DeleteAllTunnels(db store.IStore) echo.HandlerFunc {
	return func(c echo.Context) error {
		// Get all tunnels first
		tunnels, err := db.GetTunnels()
		if err != nil {
			log.Printf("Error getting tunnels for cleanup: %v", err)
			return c.JSON(http.StatusInternalServerError, jsonHTTPResponse{false, "Failed to get tunnels"})
		}

		// Delete each tunnel
		deletedCount := 0
		for _, tunnel := range tunnels {
			if err := db.DeleteTunnel(tunnel.ID); err != nil {
				log.Printf("Error deleting tunnel %s: %v", tunnel.ID, err)
			} else {
				deletedCount++
			}
		}

		log.Printf("Deleted %d tunnels", deletedCount)
		return c.JSON(http.StatusOK, jsonHTTPResponse{true, fmt.Sprintf("Deleted %d tunnels successfully", deletedCount)})
	}
}

// startWireGuardTunnel starts a WireGuard tunnel
func startWireGuardTunnel(tunnel model.Tunnel) error {
	if tunnel.WGConfig == nil {
		return fmt.Errorf("WireGuard configuration is missing")
	}

	// Generate simple interface name (e.g., wg1, wg2, wg3)
	// Use last 3 chars of tunnel ID as number
	interfaceSuffix := tunnel.ID[len(tunnel.ID)-3:]
	interfaceName := fmt.Sprintf("wg%s", interfaceSuffix)

	log.Printf("Starting WireGuard tunnel: %s -> %s", tunnel.Name, interfaceName)

	// Check if wg-quick is available
	if _, err := exec.LookPath("wg-quick"); err != nil {
		log.Printf("wg-quick not found: %v", err)
		return fmt.Errorf("WireGuard tools not installed or not in PATH")
	}

	// Create WireGuard config file
	configPath := filepath.Join("/etc/wireguard", interfaceName+".conf")

	// Ensure /etc/wireguard directory exists
	if err := os.MkdirAll("/etc/wireguard", 0700); err != nil {
		log.Printf("Failed to create /etc/wireguard directory: %v", err)
		return fmt.Errorf("failed to create config directory: %v", err)
	}

	// WireGuard config content - safe routing to prevent server network loss
	safeAllowedIPs := "10.0.0.0/8,172.16.0.0/12,192.168.0.0/16" // Private networks only
	if len(tunnel.WGConfig.AllowedIPs) > 0 {
		// اگه user مقدار خاصی تنظیم کرده، اولش چک کن که خطرناک نباشه
		allowedIPsStr := strings.Join(tunnel.WGConfig.AllowedIPs, ", ")
		if strings.Contains(allowedIPsStr, "0.0.0.0/0") || strings.Contains(allowedIPsStr, "::/0") {
			log.Printf("ERROR: Dangerous routing detected! User tried to use 0.0.0.0/0 which would disconnect server")
			log.Printf("Automatically using safe private networks instead: %s", safeAllowedIPs)
			// برای امنیت کامل، فقط private networks استفاده میکنیم
			// اگه user واقعاً global routing میخواد، باید manual تنظیم کنه
		} else if strings.Contains(allowedIPsStr, "1.0.0.0") || strings.Contains(allowedIPsStr, "8.8.8.8") {
			// اگه public IP ranges داره، هشدار بده
			log.Printf("Warning: Public IP ranges detected in AllowedIPs - using private networks for safety")
			// Private networks فقط
		} else {
			// فقط اگه safe باشه، استفاده کن
			safeAllowedIPs = allowedIPsStr
		}
	}

	// Get main interface name for routing
	mainInterface := "eth0" // Default
	// Try to detect main interface
	routeCmd := exec.Command("ip", "route", "show", "default")
	if routeOutput, err := routeCmd.Output(); err == nil {
		routeStr := string(routeOutput)
		if strings.Contains(routeStr, "dev ") {
			parts := strings.Split(routeStr, "dev ")
			if len(parts) > 1 {
				interfaceParts := strings.Fields(parts[1])
				if len(interfaceParts) > 0 {
					mainInterface = interfaceParts[0]
					log.Printf("Detected main interface: %s", mainInterface)
				}
			}
		}
	}

	configContent := fmt.Sprintf(`[Interface]
PrivateKey = %s
Address = %s
PostUp = iptables -A FORWARD -i %s -j ACCEPT; iptables -A FORWARD -o %s -j ACCEPT; iptables -t nat -A POSTROUTING -o %s -j MASQUERADE
PostDown = iptables -D FORWARD -i %s -j ACCEPT; iptables -D FORWARD -o %s -j ACCEPT; iptables -t nat -D POSTROUTING -o %s -j MASQUERADE

[Peer]
PublicKey = %s
Endpoint = %s
AllowedIPs = %s`,
		tunnel.WGConfig.LocalPrivateKey,
		tunnel.WGConfig.TunnelIP,
		interfaceName, interfaceName, mainInterface, // PostUp rules
		interfaceName, interfaceName, mainInterface, // PostDown rules
		tunnel.WGConfig.RemotePublicKey,
		tunnel.WGConfig.RemoteEndpoint,
		safeAllowedIPs)

	// Add PreShared Key if available
	if tunnel.WGConfig.PreSharedKey != "" {
		configContent += fmt.Sprintf("\nPreSharedKey = %s", tunnel.WGConfig.PreSharedKey)
	}

	// Add PersistentKeepalive for better connectivity
	configContent += "\nPersistentKeepalive = 25"

	// Write config file with proper permissions
	if err := os.WriteFile(configPath, []byte(configContent), 0600); err != nil {
		return fmt.Errorf("failed to write config file: %v", err)
	}

	log.Printf("Created WireGuard config: %s", configPath)

	// Verify file exists and is readable
	if _, err := os.Stat(configPath); err != nil {
		return fmt.Errorf("config file not accessible: %v", err)
	}

	// Double check with ls command for debugging
	lsCmd := exec.Command("ls", "-la", configPath)
	lsOutput, _ := lsCmd.CombinedOutput()
	log.Printf("Config file details: %s", string(lsOutput))

	// Check if config file exists and is readable
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return fmt.Errorf("configuration file does not exist: %s", configPath)
	}

	// Validate config file syntax
	validateCmd := exec.Command("wg-quick", "strip", interfaceName)
	validateCmd.Dir = "/etc/wireguard"
	validateOutput, err := validateCmd.CombinedOutput()
	if err != nil {
		log.Printf("Config validation failed for %s: %v, Output: %s", interfaceName, err, string(validateOutput))
		return fmt.Errorf("invalid WireGuard configuration: %s", string(validateOutput))
	}

	log.Printf("Config validation passed for %s", interfaceName)

	// Check if interface is already active
	wgShowCmd := exec.Command("wg", "show", interfaceName)
	if wgShowCmd.Run() == nil {
		log.Printf("Interface %s is already active", interfaceName)
		return fmt.Errorf("tunnel interface %s is already active", interfaceName)
	}

	// Enable IP forwarding if not already enabled
	enableForwardingCmd := exec.Command("sysctl", "-w", "net.ipv4.ip_forward=1")
	if err := enableForwardingCmd.Run(); err != nil {
		log.Printf("Warning: Failed to enable IP forwarding: %v", err)
	}

	// Start the tunnel using wg-quick with interface name in /etc/wireguard directory
	cmd := exec.Command("wg-quick", "up", interfaceName)
	cmd.Dir = "/etc/wireguard" // Set working directory to /etc/wireguard

	// Add timeout to prevent hanging
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	// Use CommandContext for timeout
	cmdWithTimeout := exec.CommandContext(ctx, "wg-quick", "up", interfaceName)
	cmdWithTimeout.Dir = "/etc/wireguard"

	output, err := cmdWithTimeout.CombinedOutput()
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			log.Printf("Tunnel start timeout for %s", interfaceName)
			return fmt.Errorf("tunnel start timeout - check WireGuard configuration")
		}

		log.Printf("Failed to start tunnel %s: %v, Output: %s", interfaceName, err, string(output))

		// چک کنیم فایل واقعاً اونجا هست
		checkCmd := exec.Command("ls", "-la", "/etc/wireguard/")
		checkOutput, _ := checkCmd.CombinedOutput()
		log.Printf("Contents of /etc/wireguard/: %s", string(checkOutput))

		return fmt.Errorf("failed to start tunnel: %s", string(output))
	}

	log.Printf("Successfully started WireGuard tunnel: %s", interfaceName)
	log.Printf("Command output: %s", string(output))

	// Verify tunnel is up and running
	verifyCmd := exec.Command("wg", "show", interfaceName)
	if verifyOutput, err := verifyCmd.Output(); err == nil {
		log.Printf("Tunnel verification: %s", string(verifyOutput))
	}

	// Add additional routing for client traffic if needed
	if tunnel.RouteAll {
		log.Printf("Setting up routing for all clients through tunnel %s", interfaceName)
		// This would require integration with main WireGuard server routing
		// For now, just log the intention
	}

	return nil
}

// stopWireGuardTunnel stops a WireGuard tunnel
func stopWireGuardTunnel(tunnel model.Tunnel) error {
	if tunnel.WGConfig == nil {
		return fmt.Errorf("WireGuard configuration is missing")
	}

	// Generate simple interface name (e.g., wg1, wg2, wg3)
	// Use last 3 chars of tunnel ID as number
	interfaceSuffix := tunnel.ID[len(tunnel.ID)-3:]
	interfaceName := fmt.Sprintf("wg%s", interfaceSuffix)

	log.Printf("Stopping WireGuard tunnel: %s -> %s", tunnel.Name, interfaceName)

	// Check if wg-quick is available
	if _, err := exec.LookPath("wg-quick"); err != nil {
		log.Printf("wg-quick not found: %v", err)
		return fmt.Errorf("WireGuard tools not installed or not in PATH")
	}

	// Stop the tunnel using wg-quick with interface name in /etc/wireguard directory
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "wg-quick", "down", interfaceName)
	cmd.Dir = "/etc/wireguard" // Set working directory to /etc/wireguard
	output, err := cmd.CombinedOutput()
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			log.Printf("Tunnel stop timeout for %s", interfaceName)
			return fmt.Errorf("tunnel stop timeout")
		}

		log.Printf("Failed to stop tunnel %s: %v, Output: %s", interfaceName, err, string(output))

		// Check specific error conditions
		outputStr := string(output)
		if strings.Contains(outputStr, "is not a WireGuard interface") {
			log.Printf("Interface %s not found - it may already be stopped", interfaceName)
			// Continue with cleanup even if interface wasn't found
		} else {
			return fmt.Errorf("failed to stop tunnel: %s", outputStr)
		}
	} else {
		log.Printf("Successfully stopped WireGuard tunnel: %s", interfaceName)
		log.Printf("Command output: %s", string(output))
	}

	// Clean up config file
	configPath := filepath.Join("/etc/wireguard", interfaceName+".conf")
	if err := os.Remove(configPath); err != nil {
		if !os.IsNotExist(err) {
			log.Printf("Warning: Failed to remove config file %s: %v", configPath, err)
		}
	} else {
		log.Printf("Removed config file: %s", configPath)
	}

	// Verify interface is down
	wgShowCmd := exec.Command("wg", "show", interfaceName)
	if wgShowCmd.Run() == nil {
		log.Printf("Warning: Interface %s still appears to be active after stop", interfaceName)
	}

	return nil
}
func startV2rayTunnel(tunnel model.Tunnel) error {
	cfg, err := service.GenerateXrayConfig(&tunnel)
	if err != nil {
		return err
	}
	return service.WriteConfigAndService(&tunnel, cfg)
}

// stopV2rayTunnel stops and removes the systemd service
func stopV2rayTunnel(tunnel model.Tunnel) error {
	serviceName := fmt.Sprintf("vwireguard-tunnel-%s.service", tunnel.ID)
	exec.Command("systemctl", "disable", "--now", serviceName).Run()
	os.Remove(filepath.Join("/etc/systemd/system", serviceName))
	os.Remove(filepath.Join("/etc/vwireguard/tunnels", tunnel.ID+".json"))
	exec.Command("systemctl", "daemon-reload").Run()
	return nil
}

// updateTunnelStatsFromWG updates tunnel statistics from WireGuard interface
func updateTunnelStatsFromWG(db store.IStore, tunnel model.Tunnel) error {
	if tunnel.WGConfig == nil {
		return fmt.Errorf("WireGuard configuration is missing")
	}

	// Generate simple interface name (e.g., wg1, wg2, wg3)
	// Use last 3 chars of tunnel ID as number
	interfaceSuffix := tunnel.ID[len(tunnel.ID)-3:]
	interfaceName := fmt.Sprintf("wg%s", interfaceSuffix)

	log.Printf("Updating tunnel stats from WireGuard interface: %s", interfaceName)

	// Check if wg-quick is available
	if _, err := exec.LookPath("wg-quick"); err != nil {
		log.Printf("wg-quick not found: %v", err)
		return fmt.Errorf("WireGuard tools not installed or not in PATH")
	}

	// Get tunnel stats from WireGuard interface
	statsCmd := exec.Command("wg", "show", interfaceName)
	statsOutput, err := statsCmd.Output()
	if err != nil {
		log.Printf("Failed to get tunnel stats from WireGuard interface: %v", err)
		return fmt.Errorf("failed to get tunnel stats from WireGuard interface")
	}

	// Parse stats output
	stats := string(statsOutput)
	lines := strings.Split(stats, "\n")
	for _, line := range lines {
		if strings.Contains(line, "transfer") {
			parts := strings.Fields(line)
			if len(parts) >= 3 {
				bytesIn, err := strconv.ParseUint(parts[1], 10, 64)
				if err != nil {
					log.Printf("Failed to parse bytes_in: %v", err)
				}
				bytesOut, err := strconv.ParseUint(parts[2], 10, 64)
				if err != nil {
					log.Printf("Failed to parse bytes_out: %v", err)
				}
				tunnel.BytesIn = int64(bytesIn)
				tunnel.BytesOut = int64(bytesOut)
			}
		}
	}

	// Update tunnel stats in database
	if err := db.SaveTunnel(tunnel); err != nil {
		log.Printf("Failed to update tunnel stats in database: %v", err)
		return fmt.Errorf("failed to update tunnel stats in database")
	}

	log.Printf("Tunnel stats updated successfully")
	return nil
}
