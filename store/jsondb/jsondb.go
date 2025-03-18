package jsondb

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"github.com/sdomino/scribble"
	"github.com/skip2/go-qrcode"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"

	"github.com/MmadF14/vwireguard/model"
	"github.com/MmadF14/vwireguard/store"
	"github.com/MmadF14/vwireguard/util"
)

type JSONDB struct {
	path string
	mu   sync.RWMutex
}

func New(path string) (*JSONDB, error) {
	return &JSONDB{
		path: path,
	}, nil
}

func (db *JSONDB) Init() error {
	db.mu.Lock()
	defer db.mu.Unlock()

	// Create database directory if it doesn't exist
	if err := os.MkdirAll(db.path, 0755); err != nil {
		return err
	}

	// Initialize default server if not exists
	serverPath := filepath.Join(db.path, "server.json")
	if _, err := os.Stat(serverPath); os.IsNotExist(err) {
		server := &model.Server{
			Interface: model.ServerInterface{
				Addresses: []string{"10.252.1.0/24"},
				ListenPort: 51820,
			},
			KeyPair: model.ServerKeypair{
				PrivateKey: "",
				PublicKey:  "",
			},
		}
		if err := db.saveJSON(serverPath, server); err != nil {
			return err
		}
	}

	// Initialize default admin user if not exists
	usersPath := filepath.Join(db.path, "users.json")
	if _, err := os.Stat(usersPath); os.IsNotExist(err) {
		users := []model.User{
			{
				ID:       1,
				Username: "admin",
				Password: "admin", // This should be hashed in production
				Role:     model.RoleAdmin,
			},
		}
		if err := db.saveJSON(usersPath, users); err != nil {
			return err
		}
	}

	return nil
}

func (db *JSONDB) Close() error {
	return nil
}

// User operations
func (db *JSONDB) GetUserByName(username string) (*model.User, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	usersPath := filepath.Join(db.path, "users.json")
	var users []model.User
	if err := db.loadJSON(usersPath, &users); err != nil {
		return nil, err
	}

	for _, user := range users {
		if user.Username == username {
			return &user, nil
		}
	}
	return nil, errors.New("user not found")
}

func (db *JSONDB) GetUserByID(id int) (*model.User, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	usersPath := filepath.Join(db.path, "users.json")
	var users []model.User
	if err := db.loadJSON(usersPath, &users); err != nil {
		return nil, err
	}

	for _, user := range users {
		if user.ID == id {
			return &user, nil
		}
	}
	return nil, errors.New("user not found")
}

func (db *JSONDB) CreateUser(user *model.User) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	usersPath := filepath.Join(db.path, "users.json")
	var users []model.User
	if err := db.loadJSON(usersPath, &users); err != nil {
		return err
	}

	users = append(users, *user)
	return db.saveJSON(usersPath, users)
}

func (db *JSONDB) UpdateUser(user *model.User) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	usersPath := filepath.Join(db.path, "users.json")
	var users []model.User
	if err := db.loadJSON(usersPath, &users); err != nil {
		return err
	}

	for i, u := range users {
		if u.ID == user.ID {
			users[i] = *user
			return db.saveJSON(usersPath, users)
		}
	}
	return errors.New("user not found")
}

func (db *JSONDB) DeleteUser(id int) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	usersPath := filepath.Join(db.path, "users.json")
	var users []model.User
	if err := db.loadJSON(usersPath, &users); err != nil {
		return err
	}

	for i, user := range users {
		if user.ID == id {
			users = append(users[:i], users[i+1:]...)
			return db.saveJSON(usersPath, users)
		}
	}
	return errors.New("user not found")
}

func (db *JSONDB) GetUsers() ([]model.User, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	usersPath := filepath.Join(db.path, "users.json")
	var users []model.User
	if err := db.loadJSON(usersPath, &users); err != nil {
		return nil, err
	}
	return users, nil
}

// Client operations
func (db *JSONDB) GetClientByID(id int) (*model.Client, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	clientsPath := filepath.Join(db.path, "clients.json")
	var clients []model.Client
	if err := db.loadJSON(clientsPath, &clients); err != nil {
		return nil, err
	}

	for _, client := range clients {
		if client.ID == id {
			return &client, nil
		}
	}
	return nil, errors.New("client not found")
}

func (db *JSONDB) GetClientByName(name string) (*model.Client, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	clientsPath := filepath.Join(db.path, "clients.json")
	var clients []model.Client
	if err := db.loadJSON(clientsPath, &clients); err != nil {
		return nil, err
	}

	for _, client := range clients {
		if client.Name == name {
			return &client, nil
		}
	}
	return nil, errors.New("client not found")
}

func (db *JSONDB) CreateClient(client *model.Client) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	clientsPath := filepath.Join(db.path, "clients.json")
	var clients []model.Client
	if err := db.loadJSON(clientsPath, &clients); err != nil {
		return err
	}

	clients = append(clients, *client)
	return db.saveJSON(clientsPath, clients)
}

func (db *JSONDB) UpdateClient(client *model.Client) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	clientsPath := filepath.Join(db.path, "clients.json")
	var clients []model.Client
	if err := db.loadJSON(clientsPath, &clients); err != nil {
		return err
	}

	for i, c := range clients {
		if c.ID == client.ID {
			clients[i] = *client
			return db.saveJSON(clientsPath, clients)
		}
	}
	return errors.New("client not found")
}

func (db *JSONDB) DeleteClient(id int) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	clientsPath := filepath.Join(db.path, "clients.json")
	var clients []model.Client
	if err := db.loadJSON(clientsPath, &clients); err != nil {
		return err
	}

	for i, client := range clients {
		if client.ID == id {
			clients = append(clients[:i], clients[i+1:]...)
			return db.saveJSON(clientsPath, clients)
		}
	}
	return errors.New("client not found")
}

func (db *JSONDB) GetClients() ([]model.Client, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	clientsPath := filepath.Join(db.path, "clients.json")
	var clients []model.Client
	if err := db.loadJSON(clientsPath, &clients); err != nil {
		return nil, err
	}
	return clients, nil
}

// Server operations
func (db *JSONDB) GetServer() (*model.Server, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	serverPath := filepath.Join(db.path, "server.json")
	var server model.Server
	if err := db.loadJSON(serverPath, &server); err != nil {
		return nil, err
	}
	return &server, nil
}

func (db *JSONDB) UpdateServer(server *model.Server) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	serverPath := filepath.Join(db.path, "server.json")
	return db.saveJSON(serverPath, server)
}

// Helper functions
func (db *JSONDB) loadJSON(path string, v interface{}) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, v)
}

func (db *JSONDB) saveJSON(path string, v interface{}) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func (o *JSONDB) GetGlobalSettings() (model.GlobalSetting, error) {
	settings := model.GlobalSetting{}
	return settings, o.loadJSON(filepath.Join(o.path, "server.json"), &settings)
}

func (o *JSONDB) GetServer() (model.Server, error) {
	server := model.Server{}
	if err := o.loadJSON(filepath.Join(o.path, "server.json"), &server); err != nil {
		return server, err
	}
	return server, nil
}

func (o *JSONDB) GetClients(hasQRCode bool) ([]model.ClientData, error) {
	var clients []model.ClientData

	// read all client json files in "clients" directory
	records, err := o.loadJSON(filepath.Join(o.path, "clients.json"), &clients)
	if err != nil {
		return clients, err
	}

	// build the ClientData list
	for _, f := range records {
		client := model.Client{}
		clientData := model.ClientData{}

		// get client info
		if err := json.Unmarshal(f, &client); err != nil {
			return clients, fmt.Errorf("cannot decode client json structure: %v", err)
		}

		// generate client qrcode image in base64
		if hasQRCode && client.PrivateKey != "" {
			server, _ := o.GetServer()
			globalSettings, _ := o.GetGlobalSettings()

			png, err := qrcode.Encode(util.BuildClientConfig(client, server, globalSettings), qrcode.Medium, 256)
			if err == nil {
				clientData.QRCode = "data:image/png;base64," + base64.StdEncoding.EncodeToString(png)
			} else {
				fmt.Print("Cannot generate QR code: ", err)
			}
		}

		// create the list of clients and their qrcode data
		clientData.Client = &client
		clients = append(clients, clientData)
	}

	return clients, nil
}

func (o *JSONDB) GetClientByID(clientID string, qrCodeSettings model.QRCodeSettings) (model.ClientData, error) {
	client := model.Client{}
	clientData := model.ClientData{}

	// read client information
	if err := o.loadJSON(filepath.Join(o.path, "clients.json"), &client); err != nil {
		return clientData, err
	}

	// generate client qrcode image in base64
	if qrCodeSettings.Enabled && client.PrivateKey != "" {
		server, _ := o.GetServer()
		globalSettings, _ := o.GetGlobalSettings()
		client := client
		if !qrCodeSettings.IncludeDNS {
			globalSettings.DNSServers = []string{}
		}
		if !qrCodeSettings.IncludeMTU {
			globalSettings.MTU = 0
		}

		png, err := qrcode.Encode(util.BuildClientConfig(client, server, globalSettings), qrcode.Medium, 256)
		if err == nil {
			clientData.QRCode = "data:image/png;base64," + base64.StdEncoding.EncodeToString(png)
		} else {
			fmt.Print("Cannot generate QR code: ", err)
		}
	}

	clientData.Client = &client

	return clientData, nil
}

func (o *JSONDB) SaveClient(client model.Client) error {
	clientPath := filepath.Join(o.path, "clients.json")
	output := o.saveJSON(clientPath, client)
	if output == nil {
		if client.Enabled && len(client.TgUserid) > 0 {
			if userid, err := strconv.ParseInt(client.TgUserid, 10, 64); err == nil {
				util.UpdateTgToClientID(userid, client.ID)
			}
		} else {
			util.RemoveTgToClientID(client.ID)
		}
	} else {
		util.RemoveTgToClientID(client.ID)
	}
	err := o.saveJSON(clientPath, client)
	if err != nil {
		return err
	}
	return output
}

func (o *JSONDB) SaveServerInterface(serverInterface model.ServerInterface) error {
	serverInterfacePath := filepath.Join(o.path, "server.json")
	output := o.saveJSON(serverInterfacePath, serverInterface)
	err := util.ManagePerms(serverInterfacePath)
	if err != nil {
		return err
	}
	return output
}

func (o *JSONDB) SaveServerKeyPair(serverKeyPair model.ServerKeypair) error {
	serverKeyPairPath := filepath.Join(o.path, "server.json")
	output := o.saveJSON(serverKeyPairPath, serverKeyPair)
	err := util.ManagePerms(serverKeyPairPath)
	if err != nil {
		return err
	}
	return output
}

func (o *JSONDB) SaveGlobalSettings(globalSettings model.GlobalSetting) error {
	globalSettingsPath := filepath.Join(o.path, "server.json")
	output := o.saveJSON(globalSettingsPath, globalSettings)
	err := util.ManagePerms(globalSettingsPath)
	if err != nil {
		return err
	}
	return output
}

func (o *JSONDB) GetPath() string {
	return o.path
}

func (o *JSONDB) GetHashes() (model.ClientServerHashes, error) {
	hashes := model.ClientServerHashes{}
	return hashes, o.loadJSON(filepath.Join(o.path, "server.json"), &hashes)
}

func (o *JSONDB) SaveHashes(hashes model.ClientServerHashes) error {
	hashesPath := filepath.Join(o.path, "server.json")
	output := o.saveJSON(hashesPath, hashes)
	err := util.ManagePerms(hashesPath)
	if err != nil {
		return err
	}
	return output
}
