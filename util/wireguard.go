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

// CurrentPeerState represents the current state of a peer in the interface
type CurrentPeerState struct {
	PublicKey           string
	AllowedIPs          []string
	PresharedKey        string
	PersistentKeepalive int
	Endpoint            string
	LastHandshake       time.Time
	ReceiveBytes        int64
	TransmitBytes       int64
}

// PeerDiff represents the difference between current and target peer state
type PeerDiff struct {
	Action    string // "add", "remove", "update", "none"
	PublicKey string
	OldState  *CurrentPeerState
	NewState  *WireGuardPeer
	Changes   []string // List of changed fields
}

// GetCurrentPeers retrieves the current peers from the WireGuard interface
func GetCurrentPeers(interfaceName string) (map[string]CurrentPeerState, error) {
	cmd := exec.Command("wg", "show", interfaceName, "dump")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to get current peers: %v", err)
	}

	peers := make(map[string]CurrentPeerState)
	scanner := bufio.NewScanner(bytes.NewReader(output))

	// Skip the first line (interface info)
	if scanner.Scan() {
		// Interface line - skip
	}

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		// Parse peer line: <public_key> <preshared_key> <endpoint> <allowed_ips> <last_handshake> <rx_bytes> <tx_bytes> <persistent_keepalive>
		fields := strings.Fields(line)
		if len(fields) < 8 {
			continue
		}

		publicKey := fields[0]
		presharedKey := fields[1]
		endpoint := fields[2]
		allowedIPsStr := fields[3]
		lastHandshakeStr := fields[4]
		rxBytesStr := fields[5]
		txBytesStr := fields[6]
		persistentKeepaliveStr := fields[7]

		// Parse allowed IPs
		var allowedIPs []string
		if allowedIPsStr != "(none)" {
			allowedIPs = strings.Split(allowedIPsStr, ",")
		}

		// Parse preshared key
		if presharedKey == "(none)" {
			presharedKey = ""
		}

		// Parse endpoint
		if endpoint == "(none)" {
			endpoint = ""
		}

		// Parse persistent keepalive
		persistentKeepalive := 0
		if persistentKeepaliveStr != "off" {
			if val, err := fmt.Sscanf(persistentKeepaliveStr, "%d", &persistentKeepalive); err != nil || val != 1 {
				persistentKeepalive = 0
			}
		}

		// Parse last handshake
		var lastHandshake time.Time
		if lastHandshakeStr != "0" {
			var timestamp int64
			if _, err := fmt.Sscanf(lastHandshakeStr, "%d", &timestamp); err != nil {
				lastHandshake = time.Unix(0, 0)
			} else {
				lastHandshake = time.Unix(timestamp, 0)
			}
		}

		// Parse bytes
		var rxBytes, txBytes int64
		fmt.Sscanf(rxBytesStr, "%d", &rxBytes)
		fmt.Sscanf(txBytesStr, "%d", &txBytes)

		peers[publicKey] = CurrentPeerState{
			PublicKey:           publicKey,
			AllowedIPs:          allowedIPs,
			PresharedKey:        presharedKey,
			PersistentKeepalive: persistentKeepalive,
			Endpoint:            endpoint,
			LastHandshake:       lastHandshake,
			ReceiveBytes:        rxBytes,
			TransmitBytes:       txBytes,
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

// UpdatePeer updates an existing peer's configuration without removing it
func UpdatePeer(interfaceName string, peer WireGuardPeer) error {
	args := []string{"set", interfaceName, "peer", peer.PublicKey}

	if len(peer.AllowedIPs) > 0 {
		args = append(args, "allowed-ips", strings.Join(peer.AllowedIPs, ","))
	}

	if peer.PresharedKey != "" {
		args = append(args, "preshared-key", peer.PresharedKey)
	} else {
		// If preshared key is empty, we need to remove it
		args = append(args, "preshared-key", "(none)")
	}

	if peer.PersistentKeepalive > 0 {
		args = append(args, "persistent-keepalive", fmt.Sprintf("%d", peer.PersistentKeepalive))
	} else {
		// If persistent keepalive is 0, we need to turn it off
		args = append(args, "persistent-keepalive", "off")
	}

	if peer.Endpoint != "" {
		args = append(args, "endpoint", peer.Endpoint)
	} else {
		// If endpoint is empty, we need to remove it
		args = append(args, "endpoint", "(none)")
	}

	cmd := exec.Command("wg", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to update peer %s: %v, output: %s", peer.PublicKey, err, string(output))
	}

	log.Printf("Successfully updated peer %s on interface %s", peer.PublicKey, interfaceName)
	return nil
}

// ComparePeerStates compares current and target peer states to determine what needs to be changed
func ComparePeerStates(current CurrentPeerState, target WireGuardPeer) PeerDiff {
	diff := PeerDiff{
		PublicKey: current.PublicKey,
		OldState:  &current,
		NewState:  &target,
		Changes:   []string{},
	}

	// Check if allowed IPs changed
	currentAllowedIPs := strings.Join(current.AllowedIPs, ",")
	targetAllowedIPs := strings.Join(target.AllowedIPs, ",")
	if currentAllowedIPs != targetAllowedIPs {
		diff.Changes = append(diff.Changes, "allowed-ips")
	}

	// Check if preshared key changed
	if current.PresharedKey != target.PresharedKey {
		diff.Changes = append(diff.Changes, "preshared-key")
	}

	// Check if persistent keepalive changed
	if current.PersistentKeepalive != target.PersistentKeepalive {
		diff.Changes = append(diff.Changes, "persistent-keepalive")
	}

	// Check if endpoint changed
	if current.Endpoint != target.Endpoint {
		diff.Changes = append(diff.Changes, "endpoint")
	}

	if len(diff.Changes) > 0 {
		diff.Action = "update"
	} else {
		diff.Action = "none"
	}

	return diff
}

// ComputePeerDiffs compares current interface state with target configuration
func ComputePeerDiffs(interfaceName string, clients []model.ClientData, settings model.GlobalSetting) ([]PeerDiff, error) {
	// Get current peers
	currentPeers, err := GetCurrentPeers(interfaceName)
	if err != nil {
		return nil, fmt.Errorf("failed to get current peers: %v", err)
	}

	// Build target peer configuration
	targetPeers := make(map[string]WireGuardPeer)
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

		targetPeers[clientData.Client.PublicKey] = peer
	}

	// Compute diffs
	var diffs []PeerDiff

	// Check for peers to add or update
	for pubKey, targetPeer := range targetPeers {
		if currentPeer, exists := currentPeers[pubKey]; exists {
			// Peer exists - check for updates
			diff := ComparePeerStates(currentPeer, targetPeer)
			if diff.Action == "update" {
				diffs = append(diffs, diff)
			}
		} else {
			// Peer doesn't exist - needs to be added
			diffs = append(diffs, PeerDiff{
				Action:    "add",
				PublicKey: pubKey,
				NewState:  &targetPeer,
			})
		}
	}

	// Check for peers to remove
	for pubKey, currentPeer := range currentPeers {
		if _, exists := targetPeers[pubKey]; !exists {
			// Peer exists in interface but not in target - needs to be removed
			diffs = append(diffs, PeerDiff{
				Action:    "remove",
				PublicKey: pubKey,
				OldState:  &currentPeer,
			})
		}
	}

	return diffs, nil
}

// ApplyPeerDiffs applies only the computed differences to the interface
func ApplyPeerDiffs(interfaceName string, diffs []PeerDiff) error {
	var addedCount, removedCount, updatedCount int

	for _, diff := range diffs {
		switch diff.Action {
		case "add":
			if err := AddPeer(interfaceName, *diff.NewState); err != nil {
				log.Printf("Failed to add peer %s: %v", diff.PublicKey, err)
				continue
			}
			addedCount++
			log.Printf("Added peer %s", diff.PublicKey)

		case "remove":
			if err := RemovePeer(interfaceName, diff.PublicKey); err != nil {
				log.Printf("Failed to remove peer %s: %v", diff.PublicKey, err)
				continue
			}
			removedCount++
			log.Printf("Removed peer %s", diff.PublicKey)

		case "update":
			if err := UpdatePeer(interfaceName, *diff.NewState); err != nil {
				log.Printf("Failed to update peer %s: %v", diff.PublicKey, err)
				continue
			}
			updatedCount++
			log.Printf("Updated peer %s (changes: %s)", diff.PublicKey, strings.Join(diff.Changes, ", "))

		case "none":
			// No changes needed
			continue
		}
	}

	if addedCount > 0 || removedCount > 0 || updatedCount > 0 {
		log.Printf("Applied peer changes: %d added, %d removed, %d updated", addedCount, removedCount, updatedCount)
	} else {
		log.Printf("No peer changes needed - interface is already in sync")
	}

	return nil
}

// ApplyConfigChanges applies configuration changes using pure runtime commands with zero disruption
func ApplyConfigChanges(interfaceName string, configPath string, clients []model.ClientData, settings model.GlobalSetting) error {
	// Step 1: Compute exact differences between current and target state
	diffs, err := ComputePeerDiffs(interfaceName, clients, settings)
	if err != nil {
		return fmt.Errorf("failed to compute peer diffs: %v", err)
	}

	// Step 2: Apply only the differences using precise wg set commands
	if err := ApplyPeerDiffs(interfaceName, diffs); err != nil {
		return fmt.Errorf("failed to apply peer diffs: %v", err)
	}

	// Step 3: Update the config file for persistence (but don't load it)
	// This is handled by the calling function

	log.Printf("Configuration applied successfully using pure runtime commands - zero disruption to existing connections")
	return nil
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

// GetActivePeerCount returns the number of currently active peers
func GetActivePeerCount(interfaceName string) (int, error) {
	peers, err := GetCurrentPeers(interfaceName)
	if err != nil {
		return 0, err
	}
	return len(peers), nil
}

// ValidatePeerConfiguration validates that a peer configuration is correct
func ValidatePeerConfiguration(peer WireGuardPeer) error {
	if peer.PublicKey == "" {
		return fmt.Errorf("public key is required")
	}

	if len(peer.AllowedIPs) == 0 {
		return fmt.Errorf("at least one allowed IP is required")
	}

	return nil
}
