package store

import (
	"github.com/MmadF14/vwireguard/model"
)

// IStore defines the interface for data storage operations
type IStore interface {
	// User operations
	GetUserByName(username string) (*model.User, error)
	GetUserByID(id int) (*model.User, error)
	CreateUser(user *model.User) error
	UpdateUser(user *model.User) error
	DeleteUser(id int) error
	GetUsers() ([]model.User, error)

	// Client operations
	GetClientByID(id int) (*model.Client, error)
	GetClientByName(name string) (*model.Client, error)
	CreateClient(client *model.Client) error
	UpdateClient(client *model.Client) error
	DeleteClient(id int) error
	GetClients() ([]model.Client, error)

	// Server operations
	GetServer() (*model.Server, error)
	UpdateServer(server *model.Server) error

	// System operations
	Init() error
	Close() error
}
