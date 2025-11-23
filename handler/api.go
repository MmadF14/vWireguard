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

// APIRequest represents a generic API request
type APIRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Token    string `json:"token"`
}

// APILoginResponse represents the response for login endpoint
type APILoginResponse struct {
	Status           string    `json:"status"`
	Token            string    `json:"token"`
	ExpireAt         time.Time `json:"expire_at"`
	TotalTraffic     int64     `json:"total_traffic"`
	RemainingTraffic int64     `json:"remaining_traffic"`
}

// APIConnectResponse represents the response for connect endpoint
type APIConnectResponse struct {
	Config string `json:"config"`
}

// APIStatusResponse represents the response for status endpoint
type APIStatusResponse struct {
	Status           string    `json:"status"`
	TotalTraffic     int64     `json:"total_traffic"`
	UsedTraffic      int64     `json:"used_traffic"`
	RemainingTraffic int64     `json:"remaining_traffic"`
	ExpireAt         time.Time `json:"expire_at"`
	Expired          bool      `json:"expired"`
}

// AdminCreateClientRequest represents the request for admin create client endpoint
type AdminCreateClientRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Token    string `json:"token"`
}

// AdminCreateClientResponse represents the response for admin create client endpoint
type AdminCreateClientResponse struct {
	Status string `json:"status"`
	Config string `json:"config"`
}

// AdminUpdateClientRequest represents the request for admin update client endpoint
type AdminUpdateClientRequest struct {
	Username   string `json:"username"`
	AddDays    int    `json:"add_days"`
	ResetQuota bool   `json:"reset_quota"`
	Token      string `json:"token"`
}

// AdminUpdateClientResponse represents the response for admin update client endpoint
type AdminUpdateClientResponse struct {
	Status        string    `json:"status"`
	NewExpiration time.Time `json:"new_expiration"`
	Message       string    `json:"message"`
}

// APILogin handles POST /api/v1/login
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

		// Get user from database
		user, err := db.GetUserByName(req.Username)
		if err != nil {
			log.Infof("Cannot query user %s from DB: %v", req.Username, err)
			return c.JSON(http.StatusUnauthorized, map[string]interface{}{
				"status":  "error",
				"message": "Invalid credentials",
			})
		}

		// Verify password
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

		// Generate API token
		token := xid.New().String()
		expireAt := time.Now().UTC().Add(30 * 24 * time.Hour) // 30 days

		// Update user with token
		user.APIToken = token
		user.TokenExpire = expireAt
		if err := db.SaveUser(user); err != nil {
			log.Error("Cannot save user token: ", err)
			return c.JSON(http.StatusInternalServerError, map[string]interface{}{
				"status":  "error",
				"message": "Failed to generate token",
			})
		}

		// Get user's client to calculate traffic
		clients, err := db.GetClients(false)
		if err != nil {
			log.Error("Cannot get clients: ", err)
		}

		var totalTraffic int64
		var usedTraffic int64
		var client *model.Client

		// Find client associated with this user (by username or email)
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

// APIConnect handles POST /api/v1/connect
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

		// Find user by token
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

		// Check token expiration
		if !user.TokenExpire.IsZero() && time.Now().UTC().After(user.TokenExpire) {
			return c.JSON(http.StatusUnauthorized, map[string]interface{}{
				"status":  "error",
				"message": "Token expired",
			})
		}

		// Get or create client for this user
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

		// Get server and global settings
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

		// If client doesn't exist, create one
		if client == nil {
			client, err = createClientForUser(db, user, server, globalSettings)
			if err != nil {
				log.Error("Cannot create client for user: ", err)
				return c.JSON(http.StatusInternalServerError, map[string]interface{}{
					"status":  "error",
					"message": "Failed to create client configuration",
				})
			}
		}

		// Check if user is expired or has no bandwidth left
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

		// Generate WireGuard config using the relay logic
		config := util.BuildClientConfig(*client, server, globalSettings)

		return c.JSON(http.StatusOK, APIConnectResponse{
			Config: config,
		})
	}
}

// APIStatus handles POST /api/v1/status
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

		// Find user by token
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

		// Get user's client
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

// createClientForUser creates a new WireGuard client for a user
func createClientForUser(db store.IStore, user *model.User, server *model.Server, globalSettings *model.GlobalSetting) (*model.Client, error) {
	// Generate client ID
	clientID := xid.New().String()

	// Generate WireGuard key pair
	key, err := wgtypes.GeneratePrivateKey()
	if err != nil {
		return nil, fmt.Errorf("cannot generate wireguard key pair: %v", err)
	}

	// Generate preshared key
	presharedKey, err := wgtypes.GenerateKey()
	if err != nil {
		return nil, fmt.Errorf("cannot generate preshared key: %v", err)
	}

	// Get available IP
	allocatedIPs, err := util.GetAllocatedIPs("")
	if err != nil {
		return nil, fmt.Errorf("cannot get allocated IPs: %v", err)
	}

	// Suggest an IP from the first available subnet
	var allocatedIP string
	if len(server.Interface.Addresses) > 0 {
		ip, err := util.GetAvailableIP(server.Interface.Addresses[0], allocatedIPs, server.Interface.Addresses)
		if err != nil {
			return nil, fmt.Errorf("cannot get available IP: %v", err)
		}
		// Format as CIDR
		if strings.Contains(ip, ":") {
			allocatedIP = fmt.Sprintf("%s/128", ip)
		} else {
			allocatedIP = fmt.Sprintf("%s/32", ip)
		}
	} else {
		return nil, fmt.Errorf("server has no interface addresses configured")
	}

	// Create client
	client := model.Client{
		ID:           clientID,
		PrivateKey:   key.String(),
		PublicKey:    key.PublicKey().String(),
		PresharedKey: presharedKey.String(),
		Name:         user.Username,
		Email:        user.Username, // Use username as email if not set
		AllocatedIPs: []string{allocatedIP},
		AllowedIPs:   []string{"0.0.0.0/0"}, // Default: route all traffic
		UseServerDNS: true,
		Enabled:      true,
		CreatedBy:    "api",
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
		Quota:        0, // Unlimited by default, can be set via admin panel
	}

	// Save client
	if err := db.SaveClient(client); err != nil {
		return nil, fmt.Errorf("cannot save client: %v", err)
	}

	if err := util.AddPeerToInterface(client, *server); err != nil {
		return nil, fmt.Errorf("cannot hot-add client: %v", err)
	}

	clients, err := db.GetClients(false)
	if err != nil {
		return nil, fmt.Errorf("cannot refresh clients: %v", err)
	}

	users, err := db.GetUsers()
	if err != nil {
		return nil, fmt.Errorf("cannot refresh users: %v", err)
	}

	if err := util.WriteWireGuardServerConfig(nil, *server, clients, users, *globalSettings); err != nil {
		return nil, fmt.Errorf("cannot persist wireguard config: %v", err)
	}

	log.Infof("Created WireGuard client for API user: %s (ID: %s)", user.Username, clientID)
	return &client, nil
}

// verifyAdminToken verifies that the token belongs to an admin user
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

	// Check token expiration
	if !user.TokenExpire.IsZero() && time.Now().UTC().After(user.TokenExpire) {
		return nil, fmt.Errorf("token expired")
	}

	// Check if user is admin
	if user.Role != model.RoleAdmin {
		return nil, fmt.Errorf("user is not an admin")
	}

	return user, nil
}

// APIAdminCreateClient handles POST /api/v1/admin/create-client
func APIAdminCreateClient(db store.IStore) echo.HandlerFunc {
	return func(c echo.Context) error {
		var req AdminCreateClientRequest
		if err := c.Bind(&req); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]interface{}{
				"status":  "error",
				"message": "Invalid request format",
			})
		}

		// Validate required fields
		if req.Username == "" || req.Email == "" || req.Token == "" {
			return c.JSON(http.StatusBadRequest, map[string]interface{}{
				"status":  "error",
				"message": "Username, email, and token are required",
			})
		}

		// Verify admin token
		adminUser, err := verifyAdminToken(db, req.Token)
		if err != nil {
			return c.JSON(http.StatusForbidden, map[string]interface{}{
				"status":  "error",
				"message": err.Error(),
			})
		}
		_ = adminUser // Admin user verified

		// Check if client already exists
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
			// Return existing client config
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

		// Get server configuration
		server, err := db.GetServer()
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]interface{}{
				"status":  "error",
				"message": "Cannot get server configuration",
			})
		}

		// Generate client ID
		clientID := xid.New().String()

		// Generate WireGuard key pair
		key, err := wgtypes.GeneratePrivateKey()
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]interface{}{
				"status":  "error",
				"message": "Cannot generate WireGuard key pair",
			})
		}

		// Generate preshared key
		presharedKey, err := wgtypes.GenerateKey()
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]interface{}{
				"status":  "error",
				"message": "Cannot generate preshared key",
			})
		}

		// Get available IP
		allocatedIPs, err := util.GetAllocatedIPs("")
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]interface{}{
				"status":  "error",
				"message": "Cannot get allocated IPs",
			})
		}

		// Suggest an IP from the first available subnet
		var allocatedIP string
		if len(server.Interface.Addresses) > 0 {
			ip, err := util.GetAvailableIP(server.Interface.Addresses[0], allocatedIPs, server.Interface.Addresses)
			if err != nil {
				return c.JSON(http.StatusInternalServerError, map[string]interface{}{
					"status":  "error",
					"message": fmt.Sprintf("Cannot get available IP: %v", err),
				})
			}
			// Format as CIDR
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

		// Create client with 1-day trial
		now := time.Now().UTC()
		client := model.Client{
			ID:           clientID,
			PrivateKey:   key.String(),
			PublicKey:    key.PublicKey().String(),
			PresharedKey: presharedKey.String(),
			Name:         req.Username,
			Email:        req.Email,
			AllocatedIPs: []string{allocatedIP},
			AllowedIPs:   []string{"0.0.0.0/0"}, // Default: route all traffic
			UseServerDNS: true,
			Enabled:      true,
			CreatedBy:    "admin-api",
			CreatedAt:    now,
			UpdatedAt:    now,
			Expiration:   now.Add(24 * time.Hour), // 1 Day trial
			Quota:        0,                       // Unlimited
		}

		// Save client
		if err := db.SaveClient(client); err != nil {
			log.Error("Cannot save client: ", err)
			return c.JSON(http.StatusInternalServerError, map[string]interface{}{
				"status":  "error",
				"message": "Failed to save client",
			})
		}

		// Get global settings for config generation
		globalSettings, err := db.GetGlobalSettings()
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]interface{}{
				"status":  "error",
				"message": "Cannot get global settings",
			})
		}

		if err := util.AddPeerToInterface(client, *server); err != nil {
			log.Error("Cannot apply runtime peer: ", err)
		}

		clients, err := db.GetClients(false)
		if err != nil {
			log.Error("Cannot get clients for config persistence: ", err)
		}

		users, err := db.GetUsers()
		if err != nil {
			log.Error("Cannot get users for config persistence: ", err)
		}

		if err := util.WriteWireGuardServerConfig(nil, *server, clients, users, globalSettings); err != nil {
			log.Error("Cannot persist WireGuard config: ", err)
		}

		// Generate WireGuard config
		config := util.BuildClientConfig(client, server, globalSettings)

		log.Infof("Admin created WireGuard client: %s (ID: %s, Trial: 1 day)", req.Username, clientID)
		return c.JSON(http.StatusOK, AdminCreateClientResponse{
			Status: "success",
			Config: config,
		})
	}
}

// APIAdminUpdateClient handles POST /api/v1/admin/update-client
func APIAdminUpdateClient(db store.IStore) echo.HandlerFunc {
	return func(c echo.Context) error {
		var req AdminUpdateClientRequest
		if err := c.Bind(&req); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]interface{}{
				"status":  "error",
				"message": "Invalid request format",
			})
		}

		// Validate required fields
		if req.Username == "" || req.Token == "" {
			return c.JSON(http.StatusBadRequest, map[string]interface{}{
				"status":  "error",
				"message": "Username and token are required",
			})
		}

		// Verify admin token
		adminUser, err := verifyAdminToken(db, req.Token)
		if err != nil {
			return c.JSON(http.StatusForbidden, map[string]interface{}{
				"status":  "error",
				"message": err.Error(),
			})
		}
		_ = adminUser // Admin user verified

		// Find client by username
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

		// Update expiration if AddDays > 0
		if req.AddDays > 0 {
			// Determine base time: if expired, use now; otherwise use current expiration
			baseTime := client.Expiration
			if client.Expiration.IsZero() || now.After(client.Expiration) {
				baseTime = now
			}
			client.Expiration = baseTime.Add(time.Duration(req.AddDays) * 24 * time.Hour)
		}

		// Reset quota if requested
		if req.ResetQuota {
			client.UsedQuota = 0
		}

		// Ensure client is enabled
		client.Enabled = true
		client.UpdatedAt = now

		// Save updated client
		if err := db.SaveClient(*client); err != nil {
			log.Error("Cannot save updated client: ", err)
			return c.JSON(http.StatusInternalServerError, map[string]interface{}{
				"status":  "error",
				"message": "Failed to update client",
			})
		}

		if err := util.UpdatePeerOnInterface(*client); err != nil {
			log.Error("Cannot hot-update peer: ", err)
		}

		if server, err := db.GetServer(); err == nil {
			if globalSettings, err := db.GetGlobalSettings(); err == nil {
				clients, err := db.GetClients(false)
				if err != nil {
					log.Error("Cannot refresh clients for config persistence: ", err)
				}
				users, err := db.GetUsers()
				if err != nil {
					log.Error("Cannot refresh users for config persistence: ", err)
				}
				if err := util.WriteWireGuardServerConfig(nil, *server, clients, users, globalSettings); err != nil {
					log.Error("Cannot persist WireGuard config after update: ", err)
				}
			} else {
				log.Error("Cannot get settings for config persistence: ", err)
			}
		} else {
			log.Error("Cannot get server for config persistence: ", err)
		}

		log.Infof("Admin updated WireGuard client: %s (ID: %s, AddDays: %d, ResetQuota: %v)", req.Username, client.ID, req.AddDays, req.ResetQuota)
		return c.JSON(http.StatusOK, AdminUpdateClientResponse{
			Status:        "success",
			NewExpiration: client.Expiration,
			Message:       "Subscription extended successfully",
		})
	}
}
