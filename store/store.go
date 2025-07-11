package store

import (
	"github.com/MmadF14/vwireguard/model"
)

type IStore interface {
	Init() error
	GetUsers() ([]model.User, error)
	GetUserByName(username string) (model.User, error)
	SaveUser(user model.User) error
	DeleteUser(username string) error
	GetGlobalSettings() (model.GlobalSetting, error)
	GetServer() (model.Server, error)
	GetClients(hasQRCode bool) ([]model.ClientData, error)
	GetClientByID(clientID string, qrCode model.QRCodeSettings) (model.ClientData, error)
	SaveClient(client model.Client) error
	DeleteClient(clientID string) error
	SaveServerInterface(serverInterface model.ServerInterface) error
	SaveServerKeyPair(serverKeyPair model.ServerKeypair) error
	SaveGlobalSettings(globalSettings model.GlobalSetting) error
	GetWakeOnLanHosts() ([]model.WakeOnLanHost, error)
	GetWakeOnLanHost(macAddress string) (*model.WakeOnLanHost, error)
	DeleteWakeOnHostLanHost(macAddress string) error
	SaveWakeOnLanHost(host model.WakeOnLanHost) error
	DeleteWakeOnHost(host model.WakeOnLanHost) error
	GetPath() string
	SaveHashes(hashes model.ClientServerHashes) error
	GetHashes() (model.ClientServerHashes, error)

	// Tunnel methods
	GetTunnels() ([]model.Tunnel, error)
	GetTunnelByID(tunnelID string) (model.Tunnel, error)
	SaveTunnel(tunnel model.Tunnel) error
	DeleteTunnel(tunnelID string) error
	UpdateTunnelStatus(tunnelID string, status model.TunnelStatus) error
	UpdateTunnelStats(tunnelID string, bytesIn, bytesOut int64) error
}
