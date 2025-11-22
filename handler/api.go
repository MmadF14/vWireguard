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

		// If client doesn't exist, create one
		if client == nil {
			client, err = createClientForUser(db, user)
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
func createClientForUser(db store.IStore, user *model.User) (*model.Client, error) {
	// Get server configuration
	server, err := db.GetServer()
	if err != nil {
		return nil, fmt.Errorf("cannot get server config: %v", err)
	}

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

	log.Infof("Created WireGuard client for API user: %s (ID: %s)", user.Username, clientID)
	return &client, nil
}
