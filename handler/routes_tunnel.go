package handler

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/rs/xid"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"

	"github.com/MmadF14/vwireguard/model"
	"github.com/MmadF14/vwireguard/store"
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
			// Generate local keypair if not provided
			if tunnelData.WGConfig.LocalPrivateKey == "" {
				privateKey, publicKey, err := generateWireGuardKeypair()
				if err != nil {
					return c.JSON(http.StatusInternalServerError, jsonHTTPResponse{false, "Failed to generate keypair"})
				}
				tunnelData.WGConfig.LocalPrivateKey = privateKey
				tunnelData.WGConfig.LocalPublicKey = publicKey
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
			log.Printf("GenerateKeypair: Deriving public key from provided private key")

			privateKey, err := wgtypes.ParseKey(requestData.PrivateKey)
			if err != nil {
				log.Printf("GenerateKeypair: Invalid private key: %v", err)
				return c.JSON(http.StatusBadRequest, jsonHTTPResponse{false, "Invalid private key format"})
			}

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
