package util

import (
	"bufio"
	"bytes"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/MmadF14/vwireguard/model"
	"github.com/labstack/gommon/log"
)

// WireGuardPeer represents a peer configuration for runtime operations
type WireGuardPeer struct {
	PublicKey           string
	AllowedIPs          []string
	PresharedKey        string
	PersistentKeepalive int
	Endpoint            string
}

// GetCurrentPeers retrieves the current peers from the WireGuard interface
func GetCurrentPeers(interfaceName string) (map[string]WireGuardPeer, error) {
	cmd := exec.Command("wg", "show", interfaceName, "peers")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to get current peers: %v", err)
	}

	peers := make(map[string]WireGuardPeer)
	scanner := bufio.NewScanner(bytes.NewReader(output))

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			// Parse peer line (format: <public_key>)
			peer := WireGuardPeer{
				PublicKey: line,
			}
			peers[line] = peer
		}
	}

	return peers, nil
}

// AddPeer adds a single peer to the WireGuard interface without disrupting others
func AddPeer(interfaceName string, peer WireGuardPeer) error {
	args := []string{"set", interfaceName, "peer", peer.PublicKey}

	if len(peer.AllowedIPs) > 0 {
		args = append(args, "allowed-ips", strings.Join(peer.AllowedIPs, ","))
	}

	if peer.PresharedKey != "" {
		args = append(args, "preshared-key", peer.PresharedKey)
	}

	if peer.PersistentKeepalive > 0 {
		args = append(args, "persistent-keepalive", fmt.Sprintf("%d", peer.PersistentKeepalive))
	}

	if peer.Endpoint != "" {
		args = append(args, "endpoint", peer.Endpoint)
	}

	cmd := exec.Command("wg", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to add peer %s: %v, output: %s", peer.PublicKey, err, string(output))
	}

	log.Printf("Successfully added peer %s to interface %s", peer.PublicKey, interfaceName)
	return nil
}

// RemovePeer removes a single peer from the WireGuard interface
func RemovePeer(interfaceName string, publicKey string) error {
	cmd := exec.Command("wg", "set", interfaceName, "peer", publicKey, "remove")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to remove peer %s: %v, output: %s", publicKey, err, string(output))
	}

	log.Printf("Successfully removed peer %s from interface %s", publicKey, interfaceName)
	return nil
}

// UpdatePeer updates an existing peer's configuration
func UpdatePeer(interfaceName string, peer WireGuardPeer) error {
	// First remove the peer, then add it back with new configuration
	if err := RemovePeer(interfaceName, peer.PublicKey); err != nil {
		return err
	}

	// Small delay to ensure the removal is processed
	time.Sleep(100 * time.Millisecond)

	return AddPeer(interfaceName, peer)
}

// SyncPeersFromConfig synchronizes peers from config file without restarting the interface
func SyncPeersFromConfig(interfaceName string, configPath string) error {
	// Try wg syncconf first (this is the most efficient method)
	cmd := exec.Command("wg", "syncconf", interfaceName, configPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("wg syncconf failed: %v, output: %s. This is normal for new interfaces.", err, string(output))
		return fmt.Errorf("wg syncconf failed: %v", err)
	}

	log.Printf("Successfully synced peers from config for interface %s", interfaceName)
	return nil
}

// ApplyConfigChanges applies configuration changes using the most appropriate method
func ApplyConfigChanges(interfaceName string, configPath string, clients []model.ClientData, settings model.GlobalSetting) error {
	// Get current peers to understand what's already configured
	currentPeers, err := GetCurrentPeers(interfaceName)
	if err != nil {
		log.Printf("Warning: Could not get current peers: %v", err)
		currentPeers = make(map[string]WireGuardPeer)
	}

	// Build new peer configurations
	newPeers := make(map[string]WireGuardPeer)
	for _, clientData := range clients {
		if !clientData.Client.Enabled {
			continue
		}

		peer := WireGuardPeer{
			PublicKey: clientData.Client.PublicKey,
		}

		// Set AllowedIPs
		if len(clientData.Client.AllocatedIPs) > 0 || len(clientData.Client.ExtraAllowedIPs) > 0 {
			allowedIPs := append(clientData.Client.AllocatedIPs, clientData.Client.ExtraAllowedIPs...)
			peer.AllowedIPs = allowedIPs
		} else {
			peer.AllowedIPs = []string{"0.0.0.0/0"}
		}

		// Set PresharedKey if available
		if clientData.Client.PresharedKey != "" {
			peer.PresharedKey = clientData.Client.PresharedKey
		}

		// Set PersistentKeepalive from global settings
		if settings.PersistentKeepalive > 0 {
			peer.PersistentKeepalive = settings.PersistentKeepalive
		}

		// Set Endpoint if available
		if clientData.Client.Endpoint != "" {
			peer.Endpoint = clientData.Client.Endpoint
		}

		newPeers[clientData.Client.PublicKey] = peer
	}

	// Strategy 1: Try to add only new peers (most efficient)
	addedCount := 0
	for pubKey, peer := range newPeers {
		if _, exists := currentPeers[pubKey]; !exists {
			if err := AddPeer(interfaceName, peer); err != nil {
				log.Printf("Failed to add peer %s: %v", pubKey, err)
				continue
			}
			addedCount++
		}
	}

	if addedCount > 0 {
		log.Printf("Successfully added %d new peers to interface %s", addedCount, interfaceName)
		return nil
	}

	// Strategy 2: If no new peers to add, try syncconf (efficient for updates)
	if err := SyncPeersFromConfig(interfaceName, configPath); err == nil {
		return nil
	}

	// Strategy 3: Fallback to service restart (only if absolutely necessary)
	log.Printf("Runtime peer management failed, falling back to service restart")
	return fmt.Errorf("runtime peer management failed, service restart required")
}

// IsInterfaceActive checks if the WireGuard interface is currently active
func IsInterfaceActive(interfaceName string) bool {
	cmd := exec.Command("wg", "show", interfaceName)
	return cmd.Run() == nil
}

// GetInterfaceStatus returns the current status of the WireGuard interface
func GetInterfaceStatus(interfaceName string) (string, error) {
	cmd := exec.Command("wg", "show", interfaceName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "inactive", nil
	}

	if len(output) > 0 {
		return "active", nil
	}
	return "inactive", nil
}
