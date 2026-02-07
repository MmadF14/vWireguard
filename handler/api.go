package handler

import (
	"crypto/subtle"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/gommon/log"
	"github.com/rs/xid"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"

	"github.com/MmadF14/vwireguard/model"
	"github.com/MmadF14/vwireguard/store"
	"github.com/MmadF14/vwireguard/util"
)

type APIRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Token    string `json:"token"`
}

type APILoginResponse struct {
	Status           string    `json:"status"`
	Token            string    `json:"token"`
	ExpireAt         time.Time `json:"expire_at"`
	TotalTraffic     int64     `json:"total_traffic"`
	RemainingTraffic int64     `json:"remaining_traffic"`
}

type APIConnectResponse struct {
	Config string `json:"config"`
}

type APIStatusResponse struct {
	Status           string    `json:"status"`
	TotalTraffic     int64     `json:"total_traffic"`
	UsedTraffic      int64     `json:"used_traffic"`
	RemainingTraffic int64     `json:"remaining_traffic"`
	ExpireAt         time.Time `json:"expire_at"`
	Expired          bool      `json:"expired"`
}

type AdminCreateClientRequest struct {
	Username   string `json:"username"`
	Email      string `json:"email"`
	Token      string `json:"token"`
	Expiration string `json:"expiration,omitempty"`
}

type AdminCreateClientResponse struct {
	Status string `json:"status"`
	Config string `json:"config"`
}

type AdminUpdateClientRequest struct {
	Username   string `json:"username"`
	AddDays    int    `json:"add_days"`
	ResetQuota bool   `json:"reset_quota"`
	Enable     *bool  `json:"enable,omitempty"`
	Token      string `json:"token"`
}

type AdminUpdateClientResponse struct {
	Status        string    `json:"status"`
	NewExpiration time.Time `json:"new_expiration"`
	Message       string    `json:"message"`
}

type AppUserInfoRequest struct {
	Username string `json:"username"`
}

type AppUserInfoResponse struct {
	Status         string    `json:"status"`
	PeerFound      bool      `json:"peer_found"`
	IsExpired      bool      `json:"is_expired"`
	IsOverQuota    bool      `json:"is_over_quota"`
	QuotaRemaining int64     `json:"quota_remaining"`
	ExpirationDate time.Time `json:"expiration_date"`
	Config         string    `json:"config,omitempty"`
	Message        string    `json:"message,omitempty"`
}

func APILogin(db store.IStore) echo.HandlerFunc {
	return func(c echo.Context) error {
		var req APIRequest
		if err := c.Bind(&req); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]interface{}{
				"status":  "error",
				"message": "Invalid request format",
			})
		}

		if req.Username == "" || req.Password == "" {
			return c.JSON(http.StatusBadRequest, map[string]interface{}{
				"status":  "error",
				"message": "Username and password are required",
			})
		}

		user, err := db.GetUserByName(req.Username)
		if err != nil {
			log.Infof("Cannot query user %s from DB: %v", req.Username, err)
			return c.JSON(http.StatusUnauthorized, map[string]interface{}{
				"status":  "error",
				"message": "Invalid credentials",
			})
		}

		var passwordCorrect bool
		if user.PasswordHash != "" {
			match, err := util.VerifyHash(user.PasswordHash, req.Password)
			if err != nil {
				return c.JSON(http.StatusInternalServerError, map[string]interface{}{
					"status":  "error",
					"message": "Cannot verify password",
				})
			}
			passwordCorrect = match
		} else {
			passwordCorrect = subtle.ConstantTimeCompare([]byte(req.Password), []byte(user.PasswordHash)) == 1
		}

		if !passwordCorrect {
			return c.JSON(http.StatusUnauthorized, map[string]interface{}{
				"status":  "error",
				"message": "Invalid credentials",
			})
		}

		token := xid.New().String()
		expireAt := time.Now().UTC().Add(30 * 24 * time.Hour)

		user.APIToken = token
		user.TokenExpire = expireAt
		if err := db.SaveUser(user); err != nil {
			log.Error("Cannot save user token: ", err)
			return c.JSON(http.StatusInternalServerError, map[string]interface{}{
				"status":  "error",
				"message": "Failed to generate token",
			})
		}

		clients, err := db.GetClients(false)
		if err != nil {
			log.Error("Cannot get clients: ", err)
		}

		var totalTraffic int64
		var usedTraffic int64
		var client *model.Client

		for _, clientData := range clients {
			if clientData.Client != nil {
				if clientData.Client.Name == req.Username || clientData.Client.Email == req.Username {
					client = clientData.Client
					break
				}
			}
		}

		if client != nil {
			totalTraffic = client.Quota
			usedTraffic = client.UsedQuota
		}

		remainingTraffic := totalTraffic - usedTraffic
		if remainingTraffic < 0 {
			remainingTraffic = 0
		}

		return c.JSON(http.StatusOK, APILoginResponse{
			Status:           "success",
			Token:            token,
			ExpireAt:         expireAt,
			TotalTraffic:     totalTraffic,
			RemainingTraffic: remainingTraffic,
		})
	}
}

func APIConnect(db store.IStore) echo.HandlerFunc {
	return func(c echo.Context) error {
		var req APIRequest
		if err := c.Bind(&req); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]interface{}{
				"status":  "error",
				"message": "Invalid request format",
			})
		}

		if req.Token == "" {
			return c.JSON(http.StatusBadRequest, map[string]interface{}{
				"status":  "error",
				"message": "Token is required",
			})
		}

		users, err := db.GetUsers()
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]interface{}{
				"status":  "error",
				"message": "Cannot query users",
			})
		}

		var user *model.User
		for _, u := range users {
			if u.APIToken == req.Token {
				user = &u
				break
			}
		}

		if user == nil {
			return c.JSON(http.StatusUnauthorized, map[string]interface{}{
				"status":  "error",
				"message": "Invalid token",
			})
		}

		if !user.TokenExpire.IsZero() && time.Now().UTC().After(user.TokenExpire) {
			return c.JSON(http.StatusUnauthorized, map[string]interface{}{
				"status":  "error",
				"message": "Token expired",
			})
		}

		clients, err := db.GetClients(false)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]interface{}{
				"status":  "error",
				"message": "Cannot get clients",
			})
		}

		var client *model.Client
		for _, clientData := range clients {
			if clientData.Client != nil {
				if clientData.Client.Name == user.Username || clientData.Client.Email == user.Username {
					client = clientData.Client
					break
				}
			}
		}

		clientJustCreated := false

		if client == nil {
			client, err = createClientForUser(db, user)
			if err != nil {
				log.Error("Cannot create client for user: ", err)
				return c.JSON(http.StatusInternalServerError, map[string]interface{}{
					"status":  "error",
					"message": "Failed to create client configuration",
				})
			}
			clientJustCreated = true
		}

		if !client.Expiration.IsZero() && time.Now().UTC().After(client.Expiration) {
			return c.JSON(http.StatusForbidden, map[string]interface{}{
				"status":  "error",
				"message": "Account expired",
			})
		}

		if client.Quota > 0 && client.UsedQuota >= client.Quota {
			return c.JSON(http.StatusForbidden, map[string]interface{}{
				"status":  "error",
				"message": "Bandwidth quota exceeded",
			})
		}

		server, err := db.GetServer()
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]interface{}{
				"status":  "error",
				"message": "Cannot get server configuration",
			})
		}

		globalSettings, err := db.GetGlobalSettings()
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]interface{}{
				"status":  "error",
				"message": "Cannot get global settings",
			})
		}

		if clientJustCreated {
			interfaceName := util.GetInterfaceNameFromConfig(globalSettings.ConfigFilePath)
			if err := util.AddPeerToInterface(*client, server, globalSettings, interfaceName); err != nil {
				log.Warnf("Failed to add peer via hot reload for newly created client %s: %v", client.Name, err)
			} else {
				log.Infof("Newly created client %s added to interface", client.Name)
			}
		}

		config := util.BuildClientConfig(*client, server, globalSettings)

		return c.JSON(http.StatusOK, APIConnectResponse{
			Config: config,
		})
	}
}

func APIStatus(db store.IStore) echo.HandlerFunc {
	return func(c echo.Context) error {
		var req APIRequest
		if err := c.Bind(&req); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]interface{}{
				"status":  "error",
				"message": "Invalid request format",
			})
		}

		if req.Token == "" {
			return c.JSON(http.StatusBadRequest, map[string]interface{}{
				"status":  "error",
				"message": "Token is required",
			})
		}

		users, err := db.GetUsers()
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]interface{}{
				"status":  "error",
				"message": "Cannot query users",
			})
		}

		var user *model.User
		for _, u := range users {
			if u.APIToken == req.Token {
				user = &u
				break
			}
		}

		if user == nil {
			return c.JSON(http.StatusUnauthorized, map[string]interface{}{
				"status":  "error",
				"message": "Invalid token",
			})
		}

		clients, err := db.GetClients(false)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]interface{}{
				"status":  "error",
				"message": "Cannot get clients",
			})
		}

		var client *model.Client
		for _, clientData := range clients {
			if clientData.Client != nil {
				if clientData.Client.Name == user.Username || clientData.Client.Email == user.Username {
					client = clientData.Client
					break
				}
			}
		}

		if client == nil {
			return c.JSON(http.StatusOK, APIStatusResponse{
				Status:           "success",
				TotalTraffic:     0,
				UsedTraffic:      0,
				RemainingTraffic: 0,
				ExpireAt:         time.Time{},
				Expired:          false,
			})
		}

		totalTraffic := client.Quota
		usedTraffic := client.UsedQuota
		remainingTraffic := totalTraffic - usedTraffic
		if remainingTraffic < 0 {
			remainingTraffic = 0
		}

		expired := false
		if !client.Expiration.IsZero() && time.Now().UTC().After(client.Expiration) {
			expired = true
		}

		return c.JSON(http.StatusOK, APIStatusResponse{
			Status:           "success",
			TotalTraffic:     totalTraffic,
			UsedTraffic:      usedTraffic,
			RemainingTraffic: remainingTraffic,
			ExpireAt:         client.Expiration,
			Expired:          expired,
		})
	}
}

func APIAppUserInfo(db store.IStore) echo.HandlerFunc {
	return func(c echo.Context) error {
		var req AppUserInfoRequest
		if err := c.Bind(&req); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]interface{}{
				"status":  "error",
				"message": "Invalid request format",
			})
		}

		if req.Username == "" {
			return c.JSON(http.StatusBadRequest, map[string]interface{}{
				"status":  "error",
				"message": "Username is required",
			})
		}

		settings, err := db.GetGlobalSettings()
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]interface{}{
				"status":  "error",
				"message": "Cannot get global settings",
			})
		}

		appSecretHeader := c.Request().Header.Get("X-App-Secret")
		if settings.AppSecretToken == "" || subtle.ConstantTimeCompare([]byte(appSecretHeader), []byte(settings.AppSecretToken)) != 1 {
			return c.JSON(http.StatusUnauthorized, map[string]interface{}{
				"status":  "error",
				"message": "invalid app secret",
			})
		}

		clients, err := db.GetClients(false)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]interface{}{
				"status":  "error",
				"message": "Cannot get clients",
			})
		}

		var client *model.Client
		for _, clientData := range clients {
			if clientData.Client != nil && clientData.Client.Name == req.Username {
				client = clientData.Client
				break
			}
		}

		if client == nil {
			return c.JSON(http.StatusOK, AppUserInfoResponse{
				Status:    "success",
				PeerFound: false,
				Message:   "client not found",
			})
		}

		server, err := db.GetServer()
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]interface{}{
				"status":  "error",
				"message": "Cannot get server configuration",
			})
		}

		now := time.Now().UTC()
		isExpired := !client.Expiration.IsZero() && client.Expiration.Before(now)
		isOverQuota := client.Quota > 0 && client.UsedQuota >= client.Quota

		remaining := client.Quota - client.UsedQuota
		if remaining < 0 {
			remaining = 0
		}

		config := util.BuildClientConfig(*client, server, settings)

		return c.JSON(http.StatusOK, AppUserInfoResponse{
			Status:         "success",
			PeerFound:      true,
			IsExpired:      isExpired,
			IsOverQuota:    isOverQuota,
			QuotaRemaining: remaining,
			ExpirationDate: client.Expiration,
			Config:         config,
		})
	}
}

func createClientForUser(db store.IStore, user *model.User) (*model.Client, error) {
	server, err := db.GetServer()
	if err != nil {
		return nil, fmt.Errorf("cannot get server config: %v", err)
	}

	clientID := xid.New().String()

	key, err := wgtypes.GeneratePrivateKey()
	if err != nil {
		return nil, fmt.Errorf("cannot generate wireguard key pair: %v", err)
	}

	presharedKey, err := wgtypes.GenerateKey()
	if err != nil {
		return nil, fmt.Errorf("cannot generate preshared key: %v", err)
	}

	allocatedIPs, err := util.GetAllocatedIPs("")
	if err != nil {
		return nil, fmt.Errorf("cannot get allocated IPs: %v", err)
	}

	var allocatedIP string
	if len(server.Interface.Addresses) > 0 {
		ip, err := util.GetAvailableIP(server.Interface.Addresses[0], allocatedIPs, server.Interface.Addresses)
		if err != nil {
			return nil, fmt.Errorf("cannot get available IP: %v", err)
		}
		if strings.Contains(ip, ":") {
			allocatedIP = fmt.Sprintf("%s/128", ip)
		} else {
			allocatedIP = fmt.Sprintf("%s/32", ip)
		}
	} else {
		return nil, fmt.Errorf("server has no interface addresses configured")
	}

	client := model.Client{
		ID:           clientID,
		PrivateKey:   key.String(),
		PublicKey:    key.PublicKey().String(),
		PresharedKey: presharedKey.String(),
		Name:         user.Username,
		Email:        user.Username,
		AllocatedIPs: []string{allocatedIP},
		AllowedIPs:   []string{"0.0.0.0/0"},
		UseServerDNS: true,
		Enabled:      true,
		CreatedBy:    "api",
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
		Quota:        0,
	}

	if err := db.SaveClient(client); err != nil {
		return nil, fmt.Errorf("cannot save client: %v", err)
	}

	log.Infof("Created WireGuard client for API user: %s (ID: %s)", user.Username, clientID)
	return &client, nil
}

func verifyAdminToken(db store.IStore, token string) (*model.User, error) {
	if token == "" {
		return nil, fmt.Errorf("token is required")
	}

	users, err := db.GetUsers()
	if err != nil {
		return nil, fmt.Errorf("cannot query users: %v", err)
	}

	var user *model.User
	for _, u := range users {
		if u.APIToken == token {
			user = &u
			break
		}
	}

	if user == nil {
		return nil, fmt.Errorf("invalid token")
	}

	if !user.TokenExpire.IsZero() && time.Now().UTC().After(user.TokenExpire) {
		return nil, fmt.Errorf("token expired")
	}

	if user.Role != model.RoleAdmin {
		return nil, fmt.Errorf("user is not an admin")
	}

	return user, nil
}

func APIAdminCreateClient(db store.IStore) echo.HandlerFunc {
	return func(c echo.Context) error {
		var req AdminCreateClientRequest
		if err := c.Bind(&req); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]interface{}{
				"status":  "error",
				"message": "Invalid request format",
			})
		}

		if req.Username == "" || req.Email == "" || req.Token == "" {
			return c.JSON(http.StatusBadRequest, map[string]interface{}{
				"status":  "error",
				"message": "Username, email, and token are required",
			})
		}

		_, err := verifyAdminToken(db, req.Token)
		if err != nil {
			return c.JSON(http.StatusForbidden, map[string]interface{}{
				"status":  "error",
				"message": err.Error(),
			})
		}

		clients, err := db.GetClients(false)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]interface{}{
				"status":  "error",
				"message": "Cannot get clients",
			})
		}

		var existingClient *model.Client
		for _, clientData := range clients {
			if clientData.Client != nil && clientData.Client.Name == req.Username {
				existingClient = clientData.Client
				break
			}
		}

		if existingClient != nil {
			server, err := db.GetServer()
			if err != nil {
				return c.JSON(http.StatusInternalServerError, map[string]interface{}{
					"status":  "error",
					"message": "Cannot get server configuration",
				})
			}

			globalSettings, err := db.GetGlobalSettings()
			if err != nil {
				return c.JSON(http.StatusInternalServerError, map[string]interface{}{
					"status":  "error",
					"message": "Cannot get global settings",
				})
			}

			config := util.BuildClientConfig(*existingClient, server, globalSettings)
			return c.JSON(http.StatusOK, AdminCreateClientResponse{
				Status: "success",
				Config: config,
			})
		}

		server, err := db.GetServer()
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]interface{}{
				"status":  "error",
				"message": "Cannot get server configuration",
			})
		}

		clientID := xid.New().String()

		key, err := wgtypes.GeneratePrivateKey()
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]interface{}{
				"status":  "error",
				"message": "Cannot generate WireGuard key pair",
			})
		}

		presharedKey, err := wgtypes.GenerateKey()
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]interface{}{
				"status":  "error",
				"message": "Cannot generate preshared key",
			})
		}

		allocatedIPs, err := util.GetAllocatedIPs("")
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]interface{}{
				"status":  "error",
				"message": "Cannot get allocated IPs",
			})
		}

		var allocatedIP string
		if len(server.Interface.Addresses) > 0 {
			ip, err := util.GetAvailableIP(server.Interface.Addresses[0], allocatedIPs, server.Interface.Addresses)
			if err != nil {
				return c.JSON(http.StatusInternalServerError, map[string]interface{}{
					"status":  "error",
					"message": fmt.Sprintf("Cannot get available IP: %v", err),
				})
			}
			if strings.Contains(ip, ":") {
				allocatedIP = fmt.Sprintf("%s/128", ip)
			} else {
				allocatedIP = fmt.Sprintf("%s/32", ip)
			}
		} else {
			return c.JSON(http.StatusInternalServerError, map[string]interface{}{
				"status":  "error",
				"message": "Server has no interface addresses configured",
			})
		}

		now := time.Now().UTC()
		var expirationTime time.Time

		if req.Expiration != "" {
			parsedExpiration, err := time.Parse(time.RFC3339, req.Expiration)
			if err != nil {
				return c.JSON(http.StatusBadRequest, map[string]interface{}{
					"status":  "error",
					"message": fmt.Sprintf("Invalid expiration format. Expected RFC3339: %v", err),
				})
			}
			expirationTime = parsedExpiration.UTC()
		} else {
			expirationTime = now.Add(24 * time.Hour)
		}

		client := model.Client{
			ID:           clientID,
			PrivateKey:   key.String(),
			PublicKey:    key.PublicKey().String(),
			PresharedKey: presharedKey.String(),
			Name:         req.Username,
			Email:        req.Email,
			AllocatedIPs: []string{allocatedIP},
			AllowedIPs:   []string{"0.0.0.0/0"},
			UseServerDNS: true,
			Enabled:      true,
			CreatedBy:    "admin-api",
			CreatedAt:    now,
			UpdatedAt:    now,
			Expiration:   expirationTime,
			Quota:        0,
		}

		if err := db.SaveClient(client); err != nil {
			log.Error("Cannot save client: ", err)
			return c.JSON(http.StatusInternalServerError, map[string]interface{}{
				"status":  "error",
				"message": "Failed to save client",
			})
		}

		globalSettings, err := db.GetGlobalSettings()
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]interface{}{
				"status":  "error",
				"message": "Cannot get global settings",
			})
		}

		interfaceName := util.GetInterfaceNameFromConfig(globalSettings.ConfigFilePath)
		if err := util.AddPeerToInterface(client, server, globalSettings, interfaceName); err != nil {
			log.Warnf("Failed to add peer via hot reload for client %s: %v", req.Username, err)
		} else {
			log.Infof("Client %s added to interface", req.Username)
		}

		config := util.BuildClientConfig(client, server, globalSettings)

		log.Infof("Admin created WireGuard client: %s (ID: %s)", req.Username, clientID)
		return c.JSON(http.StatusOK, AdminCreateClientResponse{
			Status: "success",
			Config: config,
		})
	}
}

func APIAdminUpdateClient(db store.IStore) echo.HandlerFunc {
	return func(c echo.Context) error {
		var req AdminUpdateClientRequest
		if err := c.Bind(&req); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]interface{}{
				"status":  "error",
				"message": "Invalid request format",
			})
		}

		if req.Username == "" || req.Token == "" {
			return c.JSON(http.StatusBadRequest, map[string]interface{}{
				"status":  "error",
				"message": "Username and token are required",
			})
		}

		_, err := verifyAdminToken(db, req.Token)
		if err != nil {
			return c.JSON(http.StatusForbidden, map[string]interface{}{
				"status":  "error",
				"message": err.Error(),
			})
		}

		clients, err := db.GetClients(false)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]interface{}{
				"status":  "error",
				"message": "Cannot get clients",
			})
		}

		var client *model.Client
		for _, clientData := range clients {
			if clientData.Client != nil && clientData.Client.Name == req.Username {
				client = clientData.Client
				break
			}
		}

		if client == nil {
			return c.JSON(http.StatusNotFound, map[string]interface{}{
				"status":  "error",
				"message": fmt.Sprintf("Client with username %s not found", req.Username),
			})
		}

		now := time.Now().UTC()
		wasEnabled := client.Enabled

		if req.AddDays > 0 {
			baseTime := client.Expiration
			if client.Expiration.IsZero() || now.After(client.Expiration) {
				baseTime = now
			}
			client.Expiration = baseTime.Add(time.Duration(req.AddDays) * 24 * time.Hour)
		}

		if req.ResetQuota {
			client.UsedQuota = 0
		}

		if req.Enable != nil {
			client.Enabled = *req.Enable
		} else {
			if util.IsClientValid(*client) {
				client.Enabled = true
			} else {
				client.Enabled = false
			}
		}
		client.UpdatedAt = now

		if err := db.SaveClient(*client); err != nil {
			log.Error("Cannot save updated client: ", err)
			return c.JSON(http.StatusInternalServerError, map[string]interface{}{
				"status":  "error",
				"message": "Failed to update client",
			})
		}

		if !wasEnabled && client.Enabled && util.IsClientValid(*client) {
			log.Infof("Client %s auto-enabled after renewal", req.Username)
		}

		server, err := db.GetServer()
		if err != nil {
			log.Warnf("Cannot get server config: %v", err)
		} else {
			globalSettings, err := db.GetGlobalSettings()
			if err != nil {
				log.Warnf("Cannot get global settings: %v", err)
			} else {
				interfaceName := util.GetInterfaceNameFromConfig(globalSettings.ConfigFilePath)

				if req.Enable != nil {
					if *req.Enable {
						if err := util.AddPeerToInterface(*client, server, globalSettings, interfaceName); err != nil {
							log.Warnf("Failed to add peer via hot reload for client %s: %v", req.Username, err)
						} else {
							log.Infof("Client %s enabled and added to interface", req.Username)
						}
					} else {
						if err := util.RemovePeerFromInterface(client.PublicKey, interfaceName); err != nil {
							log.Warnf("Failed to remove peer via hot reload for client %s: %v", req.Username, err)
						} else {
							log.Infof("Client %s disabled and removed from interface", req.Username)
						}
					}
				} else {
					if err := util.UpdatePeerOnInterface(*client, server, globalSettings, interfaceName); err != nil {
						log.Warnf("Failed to update peer via hot reload for client %s: %v", req.Username, err)
					} else {
						log.Infof("Client %s updated on interface", req.Username)
					}
				}
			}
		}

		log.Infof("Admin updated WireGuard client: %s (ID: %s)", req.Username, client.ID)
		return c.JSON(http.StatusOK, AdminUpdateClientResponse{
			Status:        "success",
			NewExpiration: client.Expiration,
			Message:       "Subscription extended successfully",
		})
	}
}
