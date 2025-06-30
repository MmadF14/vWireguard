package jsondb

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
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
			return tunnels, fmt.Errorf("cannot decode tunnel JSON: %v", err)
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

	// Marshal and save
	peerData, err := json.Marshal(tunnel)
	if err != nil {
		return fmt.Errorf("cannot marshal tunnel: %v", err)
	}

	return db.conn.Write(tunnelCollectionName, tunnel.ID, string(peerData))
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
