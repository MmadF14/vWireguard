package jsondb

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/MmadF14/vwireguard/model"
	"github.com/MmadF14/vwireguard/util"
)

const tunnelCollectionName = "tunnels"

// GetTunnels implementation
func (db *JsonDB) GetTunnels() ([]model.Tunnel, error) {
	tunnels := make([]model.Tunnel, 0)

	records, err := db.conn.ReadAll(tunnelCollectionName)
	if err != nil {
		return tunnels, err
	}

	for _, f := range records {
		tunnel := model.Tunnel{}
		if err := json.Unmarshal([]byte(f), &tunnel); err != nil {
			// Log the error and try to identify the problematic record
			fmt.Printf("Warning: cannot decode tunnel JSON: %v\n", err)
			fmt.Printf("Problematic JSON data: %s\n", string(f))
			continue
		}
		tunnels = append(tunnels, tunnel)
	}

	return tunnels, nil
}

// GetTunnelByID implementation
func (db *JsonDB) GetTunnelByID(tunnelID string) (model.Tunnel, error) {
	tunnel := model.Tunnel{}

	err := db.conn.Read(tunnelCollectionName, tunnelID, &tunnel)
	if err != nil {
		return tunnel, err
	}

	return tunnel, nil
}

// SaveTunnel implementation
func (db *JsonDB) SaveTunnel(tunnel model.Tunnel) error {
	// Set timestamps
	now := time.Now().UTC()
	if tunnel.CreatedAt.IsZero() {
		tunnel.CreatedAt = now
	}
	tunnel.UpdatedAt = now

	// Set ID if empty
	if tunnel.ID == "" {
		tunnel.ID = util.RandomString(12)
	}

	// Set default status if empty
	if tunnel.Status == "" {
		tunnel.Status = model.TunnelStatusInactive
	}

	// Save directly - scribble will handle JSON marshaling
	return db.conn.Write(tunnelCollectionName, tunnel.ID, tunnel)
}

// DeleteTunnel implementation
func (db *JsonDB) DeleteTunnel(tunnelID string) error {
	return db.conn.Delete(tunnelCollectionName, tunnelID)
}

// UpdateTunnelStatus implementation
func (db *JsonDB) UpdateTunnelStatus(tunnelID string, status model.TunnelStatus) error {
	tunnel, err := db.GetTunnelByID(tunnelID)
	if err != nil {
		return err
	}

	tunnel.Status = status
	tunnel.UpdatedAt = time.Now().UTC()

	if status == model.TunnelStatusActive {
		tunnel.LastSeen = time.Now().UTC()
	}

	return db.SaveTunnel(tunnel)
}

// UpdateTunnelStats implementation
func (db *JsonDB) UpdateTunnelStats(tunnelID string, bytesIn, bytesOut int64) error {
	tunnel, err := db.GetTunnelByID(tunnelID)
	if err != nil {
		return err
	}

	tunnel.BytesIn = bytesIn
	tunnel.BytesOut = bytesOut
	tunnel.UpdatedAt = time.Now().UTC()
	tunnel.LastSeen = time.Now().UTC()

	return db.SaveTunnel(tunnel)
}

// ensureTunnelDir ensures the tunnel directory exists
func (db *JsonDB) ensureTunnelDir() error {
	tunnelDir := filepath.Join(db.dbPath, tunnelCollectionName)
	if _, err := os.Stat(tunnelDir); os.IsNotExist(err) {
		return os.MkdirAll(tunnelDir, os.ModePerm)
	}
	return nil
}

// CleanupCorruptedTunnels removes tunnel records that cannot be parsed
func (db *JsonDB) CleanupCorruptedTunnels() error {
	records, err := db.conn.ReadAll(tunnelCollectionName)
	if err != nil {
		return err
	}

	corruptedCount := 0
	for i, f := range records {
		tunnel := model.Tunnel{}
		if err := json.Unmarshal([]byte(f), &tunnel); err != nil {
			// This record is corrupted, try to delete it
			// We need to find the file name somehow
			tunnelDir := filepath.Join(db.dbPath, tunnelCollectionName)
			files, dirErr := os.ReadDir(tunnelDir)
			if dirErr == nil && i < len(files) {
				fileName := strings.TrimSuffix(files[i].Name(), ".json")
				if deleteErr := db.conn.Delete(tunnelCollectionName, fileName); deleteErr == nil {
					fmt.Printf("Deleted corrupted tunnel record: %s\n", fileName)
					corruptedCount++
				}
			}
		}
	}

	if corruptedCount > 0 {
		fmt.Printf("Cleaned up %d corrupted tunnel records\n", corruptedCount)
	}

	return nil
}
