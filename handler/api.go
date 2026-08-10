package handler

import (
	"crypto/subtle"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/gommon/log"
	"github.com/rs/xid"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"

	"github.com/MmadF14/vwireguard/model"
	"github.com/MmadF14/vwireguard/store"
	"github.com/MmadF14/vwireguard/util"
)

// ---------------------------------------------------------------------------
// Login throttling.
//
// APILogin has no rate limit, and every node ships with admin/admin. Anyone who
// finds port 5000 can brute-force it. This is a small in-memory limiter keyed by
// source IP: after loginMaxFailures failed attempts inside loginWindow, further
// attempts from that IP are refused for loginLockout regardless of the password.
//
// In-memory is deliberate - a node runs a single process, and a limiter that
// survives a restart is not worth a datastore dependency here. A restart clears
// it, which at worst grants an attacker one more small window.
// ---------------------------------------------------------------------------
const (
	loginMaxFailures = 5
	loginWindow      = 5 * time.Minute
	loginLockout     = 15 * time.Minute
)

type loginAttempt struct {
	failures  int
	firstSeen time.Time
	blockedTo time.Time
}

var (
	loginAttempts   = make(map[string]*loginAttempt)
	loginAttemptsMu sync.Mutex
)

// loginBlocked reports whether this IP is currently locked out, and if so for
// how much longer.
func loginBlocked(ip string) (bool, time.Duration) {
	loginAttemptsMu.Lock()
	defer loginAttemptsMu.Unlock()
	a := loginAttempts[ip]
	if a == nil {
		return false, 0
	}
	if now := time.Now(); now.Before(a.blockedTo) {
		return true, time.Until(a.blockedTo)
	}
	return false, 0
}

// loginNoteFailure records a failed attempt and arms the lockout once the
// threshold is crossed inside the window.
func loginNoteFailure(ip string) {
	loginAttemptsMu.Lock()
	defer loginAttemptsMu.Unlock()
	now := time.Now()
	a := loginAttempts[ip]
	if a == nil || now.Sub(a.firstSeen) > loginWindow {
		a = &loginAttempt{firstSeen: now}
		loginAttempts[ip] = a
	}
	a.failures++
	if a.failures >= loginMaxFailures {
		a.blockedTo = now.Add(loginLockout)
	}
}

// loginNoteSuccess clears the counter for an IP after a good login.
func loginNoteSuccess(ip string) {
	loginAttemptsMu.Lock()
	defer loginAttemptsMu.Unlock()
	delete(loginAttempts, ip)
}

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

// AppUserInfoRequest represents the request for mobile app user info endpoint
type AppUserInfoRequest struct {
	Username string `json:"username"`
}

// AppUserInfoResponse represents the response for the mobile app user info endpoint
type AppUserInfoResponse struct {
	Status         string    `json:"status"`
	PeerFound      bool      `json:"peer_found"`
	IsExpired      bool      `json:"is_expired"`
	IsOverQuota    bool      `json:"is_over_quota"`
	QuotaRemaining int64     `json:"quota_remaining"`
	// QuotaTotal / QuotaUsed let the site sync real traffic usage. The site's
	// fetchUsageFromNode() already reads both keys and ignores them when absent,
	// so populating them here starts usage sync with no site-side change. Without
	// them, quota_remaining alone is uninformative (it floors to 0 whenever the
	// panel-created client has Quota = 0). See ZeroDelaySite PANEL-INTEGRATION.md.
	QuotaTotal     int64     `json:"quota_total"`
	QuotaUsed      int64     `json:"quota_used"`
	ExpirationDate time.Time `json:"expiration_date"`
	Config         string    `json:"config,omitempty"`
	Message        string    `json:"message,omitempty"`
}

// APILogin handles POST /api/v1/login
func APILogin(db store.IStore) echo.HandlerFunc {
	return func(c echo.Context) error {
		// Refuse early if this source IP is locked out. Done before any DB work so
		// a brute-force attempt costs nothing.
		ip := c.RealIP()
		if blocked, wait := loginBlocked(ip); blocked {
			log.Warnf("Login blocked for %s (too many failures); %s remaining", ip, wait.Round(time.Second))
			return c.JSON(http.StatusTooManyRequests, map[string]interface{}{
				"status":  "error",
				"message": "Too many failed attempts. Try again later.",
			})
		}

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
			loginNoteFailure(ip)
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
			loginNoteFailure(ip)
			return c.JSON(http.StatusUnauthorized, map[string]interface{}{
				"status":  "error",
				"message": "Invalid credentials",
			})
		}

		// Good login: clear this IP's failure counter.
		loginNoteSuccess(ip)

		// Issue a NEW, INDEPENDENT token. This used to overwrite user.APIToken,
		// which invalidated every other logged-in device for the same account
		// (the site, the Windows client and a phone kept knocking each other
		// out). AddAPIToken appends instead, prunes expired entries and caps the
		// list, so each device keeps its own session with its own expiry.
		token := xid.New().String()
		expireAt := time.Now().UTC().Add(30 * 24 * time.Hour) // 30 days

		label := c.Request().UserAgent()
		if len(label) > 80 {
			label = label[:80]
		}
		user.AddAPIToken(token, expireAt, label)
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

		// ValidateAPIToken checks every token this user holds (and the legacy
		// single-token field), including each one's own expiry.
		var user *model.User
		for i := range users {
			if users[i].ValidateAPIToken(req.Token) {
				user = &users[i]
				break
			}
		}

		if user == nil {
			return c.JSON(http.StatusUnauthorized, map[string]interface{}{
				"status":  "error",
				"message": "Invalid or expired token",
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

		// Track if client was just created
		clientJustCreated := false

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
			clientJustCreated = true
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

		// Hot Reload: If client was just created, add peer to interface instantly
		if clientJustCreated {
			interfaceName := util.GetInterfaceNameFromConfig(globalSettings.ConfigFilePath)
			if err := util.AddPeerToInterface(*client, server, globalSettings, interfaceName); err != nil {
				log.Warnf("Failed to add peer via hot reload for newly created client %s: %v (client saved to DB)", client.Name, err)
				// Continue - client is saved in DB even if runtime update fails
			} else {
				log.Infof("Newly created client %s added to interface via Hot Reload", client.Name)
			}
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
		for i := range users {
			if users[i].ValidateAPIToken(req.Token) {
				user = &users[i]
				break
			}
		}

		if user == nil {
			return c.JSON(http.StatusUnauthorized, map[string]interface{}{
				"status":  "error",
				"message": "Invalid or expired token",
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

// APIAppUserInfo handles POST /api/v1/app/user-info
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
			QuotaTotal:     client.Quota,
			QuotaUsed:      client.UsedQuota,
			ExpirationDate: client.Expiration,
			Config:         config,
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
	for i := range users {
		if users[i].ValidateAPIToken(token) {
			user = &users[i]
			break
		}
	}

	if user == nil {
		return nil, fmt.Errorf("invalid or expired token")
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

		// Hot Reload: Add peer to interface instantly without restarting service
		interfaceName := util.GetInterfaceNameFromConfig(globalSettings.ConfigFilePath)
		if err := util.AddPeerToInterface(client, server, globalSettings, interfaceName); err != nil {
			log.Warnf("Failed to add peer via hot reload for client %s: %v (client saved to DB)", req.Username, err)
			// Continue - client is saved in DB even if runtime update fails
		} else {
			log.Infof("Client %s added to interface via Hot Reload", req.Username)
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
		wasEnabled := client.Enabled

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

		// Smart Renewal: Auto-enable if client becomes valid after update
		// Check if client is now valid (not expired and not over quota)
		if util.IsClientValid(*client) {
			// If client is valid after renewal, enable it
			client.Enabled = true
		} else {
			// Client is still not valid (shouldn't happen after renewal, but handle it)
			client.Enabled = false
		}
		client.UpdatedAt = now

		// Save updated client
		if err := db.SaveClient(*client); err != nil {
			log.Error("Cannot save updated client: ", err)
			return c.JSON(http.StatusInternalServerError, map[string]interface{}{
				"status":  "error",
				"message": "Failed to update client",
			})
		}

		// Log auto-enable if it happened
		if !wasEnabled && client.Enabled && util.IsClientValid(*client) {
			log.Infof("Client %s auto-enabled after renewal via API (expiration extended or quota reset)", req.Username)
		}

		// Get server and global settings for hot reload
		server, err := db.GetServer()
		if err != nil {
			log.Warnf("Cannot get server config for hot reload: %v", err)
		} else {
			globalSettings, err := db.GetGlobalSettings()
			if err != nil {
				log.Warnf("Cannot get global settings for hot reload: %v", err)
			} else {
				// Hot Reload: Update peer on interface instantly (adds back if re-enabled, removes if disabled/expired)
				interfaceName := util.GetInterfaceNameFromConfig(globalSettings.ConfigFilePath)
				if err := util.UpdatePeerOnInterface(*client, server, globalSettings, interfaceName); err != nil {
					log.Warnf("Failed to update peer via hot reload for client %s: %v (client saved to DB)", req.Username, err)
					// Continue - client is saved in DB even if runtime update fails
				} else {
					log.Infof("Client %s updated on interface via Hot Reload", req.Username)
				}
			}
		}

		log.Infof("Admin updated WireGuard client: %s (ID: %s, AddDays: %d, ResetQuota: %v)", req.Username, client.ID, req.AddDays, req.ResetQuota)
		return c.JSON(http.StatusOK, AdminUpdateClientResponse{
			Status:        "success",
			NewExpiration: client.Expiration,
			Message:       "Subscription extended successfully",
		})
	}
}
