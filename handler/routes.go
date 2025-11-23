package handler

import (
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/sessions"
	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/v4"
	"github.com/labstack/gommon/log"
	"github.com/rs/xid"
	"github.com/skip2/go-qrcode"
	"golang.zx2c4.com/wireguard/wgctrl"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"

	"github.com/MmadF14/vwireguard/emailer"
	"github.com/MmadF14/vwireguard/model"
	"github.com/MmadF14/vwireguard/store"
	"github.com/MmadF14/vwireguard/telegram"
	"github.com/MmadF14/vwireguard/util"
)

var usernameRegexp = regexp.MustCompile("^[a-zA-Z0-9][a-zA-Z0-9-_.]*[a-zA-Z0-9]$")

// Route represents an internal API route
type Route struct {
	Method     string
	Path       string
	Handler    func(store.IStore) echo.HandlerFunc
	Middleware []echo.MiddlewareFunc
}

var internalRoutes []Route

// DeviceVM view model
type DeviceVM struct {
	Name  string
	Peers []PeerVM
}

// PeerVM view model
type PeerVM struct {
	PublicKey         string
	Name              string
	Email             string
	AllocatedIP       string
	Endpoint          string
	ReceivedBytes     int64
	TransmitBytes     int64
	LastHandshakeTime time.Time
	LastHandshakeRel  time.Duration
	Connected         bool
}

// Health check handler
func Health() echo.HandlerFunc {
	return func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	}
}

func Favicon() echo.HandlerFunc {
	return func(c echo.Context) error {
		if favicon, ok := os.LookupEnv(util.FaviconFilePathEnvVar); ok {
			return c.File(favicon)
		}
		return c.Redirect(http.StatusFound, util.BasePath+"/static/custom/img/favicon.ico")
	}
}

// LoginPage handler
func LoginPage() echo.HandlerFunc {
	return func(c echo.Context) error {
		return c.Render(http.StatusOK, "login.html", map[string]interface{}{})
	}
}

// Login for signing in handler
func Login(db store.IStore) echo.HandlerFunc {
	return func(c echo.Context) error {
		data := make(map[string]interface{})
		err := json.NewDecoder(c.Request().Body).Decode(&data)

		if err != nil {
			return c.JSON(http.StatusBadRequest, jsonHTTPResponse{false, "Bad post data"})
		}

		username := data["username"].(string)
		password := data["password"].(string)
		rememberMe := data["rememberMe"].(bool)

		if !usernameRegexp.MatchString(username) {
			return c.JSON(http.StatusBadRequest, jsonHTTPResponse{false, "Please provide a valid username"})
		}

		dbuser, err := db.GetUserByName(username)
		if err != nil {
			log.Infof("Cannot query user %s from DB", username)
			return c.JSON(http.StatusInternalServerError, jsonHTTPResponse{false, "Invalid credentials"})
		}

		userCorrect := subtle.ConstantTimeCompare([]byte(username), []byte(dbuser.Username)) == 1

		var passwordCorrect bool
		if dbuser.PasswordHash != "" {
			match, err := util.VerifyHash(dbuser.PasswordHash, password)
			if err != nil {
				return c.JSON(http.StatusInternalServerError, jsonHTTPResponse{false, "Cannot verify password"})
			}
			passwordCorrect = match
		} else {
			passwordCorrect = subtle.ConstantTimeCompare([]byte(password), []byte(dbuser.PasswordHash)) == 1
		}

		if userCorrect && passwordCorrect {
			ageMax := 0
			if rememberMe {
				ageMax = util.SessionMaxAge
			}

			cookiePath := util.GetCookiePath()

			sess, _ := session.Get("session", c)
			sess.Options = &sessions.Options{
				Path:     cookiePath,
				MaxAge:   ageMax,
				HttpOnly: true,
				SameSite: http.SameSiteLaxMode,
			}

			// set session_token
			tokenUID := xid.New().String()
			now := time.Now().UTC().Unix()
			sess.Values["username"] = dbuser.Username
			sess.Values["user_hash"] = util.GetDBUserCRC32(dbuser)
			sess.Values["admin"] = dbuser.Role == model.RoleAdmin
			sess.Values["session_token"] = tokenUID
			sess.Values["max_age"] = ageMax
			sess.Values["created_at"] = now
			sess.Values["updated_at"] = now
			sess.Save(c.Request(), c.Response())

			// set session_token in cookie
			cookie := new(http.Cookie)
			cookie.Name = "session_token"
			cookie.Path = cookiePath
			cookie.Value = tokenUID
			cookie.MaxAge = ageMax
			cookie.HttpOnly = true
			cookie.SameSite = http.SameSiteLaxMode
			c.SetCookie(cookie)

			return c.JSON(http.StatusOK, jsonHTTPResponse{true, "Logged in successfully"})
		}

		return c.JSON(http.StatusUnauthorized, jsonHTTPResponse{false, "Invalid credentials"})
	}
}

// GetUsers handler return a JSON list of all users
func GetUsers(db store.IStore) echo.HandlerFunc {
	return func(c echo.Context) error {
		usersList, err := db.GetUsers()
		if err != nil {
			return c.JSON(http.StatusInternalServerError, jsonHTTPResponse{
				false, fmt.Sprintf("Cannot get user list: %v", err),
			})
		}

		return c.JSON(http.StatusOK, usersList)
	}
}

// GetUser handler returns a JSON object of single user
func GetUser(db store.IStore) echo.HandlerFunc {
	return func(c echo.Context) error {
		username := c.Param("username")

		if !usernameRegexp.MatchString(username) {
			return c.JSON(http.StatusBadRequest, jsonHTTPResponse{false, "Please provide a valid username"})
		}

		if !isAdmin(c) && (username != currentUser(c)) {
			return c.JSON(http.StatusForbidden, jsonHTTPResponse{false, "Manager cannot access other user data"})
		}

		userData, err := db.GetUserByName(username)
		if err != nil {
			return c.JSON(http.StatusNotFound, jsonHTTPResponse{false, "User not found"})
		}

		return c.JSON(http.StatusOK, userData)
	}
}

// Logout to log a user out
func Logout() echo.HandlerFunc {
	return func(c echo.Context) error {
		clearSession(c)
		return c.Redirect(http.StatusTemporaryRedirect, util.BasePath+"/login")
	}
}

// LoadProfile to load user information
func LoadProfile(db store.IStore) echo.HandlerFunc {
	return func(c echo.Context) error {
		user, _ := db.GetUserByName(currentUser(c))
		return c.Render(http.StatusOK, "profile.html", map[string]interface{}{
			"baseData": model.BaseData{Active: "profile", CurrentUser: currentUser(c), Admin: user.Role == model.RoleAdmin},
		})
	}
}

// UsersSettings handler
func UsersSettings(db store.IStore) echo.HandlerFunc {
	return func(c echo.Context) error {
		user, _ := db.GetUserByName(currentUser(c))
		return c.Render(http.StatusOK, "users_settings.html", map[string]interface{}{
			"baseData": model.BaseData{Active: "users-settings", CurrentUser: currentUser(c), Admin: user.Role == model.RoleAdmin},
		})
	}
}

// UpdateUser to update user information
func UpdateUser(db store.IStore) echo.HandlerFunc {
	return func(c echo.Context) error {
		data := make(map[string]interface{})
		err := json.NewDecoder(c.Request().Body).Decode(&data)

		if err != nil {
			return c.JSON(http.StatusBadRequest, jsonHTTPResponse{false, "Bad post data"})
		}

		username := data["username"].(string)
		password := data["password"].(string)
		previousUsername := data["previous_username"].(string)
		role := model.UserRole(data["role"].(string))

		// اعتبارسنجی نام کاربری
		if !usernameRegexp.MatchString(username) {
			return c.JSON(http.StatusBadRequest, jsonHTTPResponse{false, "نام کاربری باید با حرف یا عدد شروع و تمام شود و فقط شامل حروف، اعداد، خط تیره، نقطه و زیرخط باشد"})
		}

		// اعتبارسنجی طول نام کاربری
		if len(username) < 3 || len(username) > 32 {
			return c.JSON(http.StatusBadRequest, jsonHTTPResponse{false, "نام کاربری باید بین 3 تا 32 کاراکتر باشد"})
		}

		// اعتبارسنجی نقش
		if role != model.RoleAdmin && role != model.RoleManager && role != model.RoleUser {
			return c.JSON(http.StatusBadRequest, jsonHTTPResponse{false, "نقش کاربر نامعتبر است"})
		}

		if !isAdmin(c) && (previousUsername != currentUser(c)) {
			return c.JSON(http.StatusForbidden, jsonHTTPResponse{false, "مدیر نمی‌تواند اطلاعات کاربران دیگر را تغییر دهد"})
		}

		if !isAdmin(c) {
			role = model.RoleUser
		}

		user, err := db.GetUserByName(previousUsername)
		if err != nil {
			return c.JSON(http.StatusNotFound, jsonHTTPResponse{false, err.Error()})
		}

		if username == "" || !usernameRegexp.MatchString(username) {
			return c.JSON(http.StatusBadRequest, jsonHTTPResponse{false, "Please provide a valid username"})
		} else {
			user.Username = username
		}

		if username != previousUsername {
			_, err := db.GetUserByName(username)
			if err == nil {
				return c.JSON(http.StatusBadRequest, jsonHTTPResponse{false, "This username is taken"})
			}
		}

		if password != "" {
			hash, err := util.HashPassword(password)
			if err != nil {
				return c.JSON(http.StatusInternalServerError, jsonHTTPResponse{false, err.Error()})
			}
			user.PasswordHash = hash
		}

		if previousUsername != currentUser(c) {
			user.Role = model.RoleUser
		}

		user.Role = role

		if err := db.DeleteUser(previousUsername); err != nil {
			return c.JSON(http.StatusInternalServerError, jsonHTTPResponse{false, err.Error()})
		}
		if err := db.SaveUser(user); err != nil {
			return c.JSON(http.StatusInternalServerError, jsonHTTPResponse{false, err.Error()})
		}
		log.Infof("Updated user information successfully")

		if previousUsername == currentUser(c) {
			setUser(c, user.Username, user.Role == model.RoleAdmin, util.GetDBUserCRC32(user))
		}

		return c.JSON(http.StatusOK, jsonHTTPResponse{true, "Updated user information successfully"})
	}
}

// CreateUser to create a new user
func CreateUser(db store.IStore) echo.HandlerFunc {
	return func(c echo.Context) error {
		var username, password, role string

		// سعی می‌کنیم اول داده‌ها را از JSON بخوانیم
		data := make(map[string]interface{})
		if err := json.NewDecoder(c.Request().Body).Decode(&data); err == nil {
			// اگر داده‌ها به صورت JSON ارسال شده‌اند
			if u, ok := data["username"].(string); ok {
				username = u
			}
			if p, ok := data["password"].(string); ok {
				password = p
			}
			if r, ok := data["role"].(string); ok {
				role = r
			}
		} else {
			// اگر داده‌ها به صورت فرم ارسال شده‌اند
			username = c.FormValue("username")
			password = c.FormValue("password")
			role = c.FormValue("role")
		}

		// اعتبارسنجی نام کاربری و رمز عبور
		if username == "" || password == "" {
			return c.JSON(http.StatusBadRequest, map[string]interface{}{
				"success": false,
				"error":   "نام کاربری و رمز عبور نمی‌توانند خالی باشند",
			})
		}

		// اعتبارسنجی نام کاربری
		if !usernameRegexp.MatchString(username) {
			return c.JSON(http.StatusBadRequest, map[string]interface{}{
				"success": false,
				"error":   "نام کاربری باید با حرف یا عدد شروع و تمام شود و فقط شامل حروف، اعداد، خط تیره، نقطه و زیرخط باشد",
			})
		}

		// اعتبارسنجی طول نام کاربری
		if len(username) < 3 || len(username) > 32 {
			return c.JSON(http.StatusBadRequest, map[string]interface{}{
				"success": false,
				"error":   "نام کاربری باید بین 3 تا 32 کاراکتر باشد",
			})
		}

		// بررسی وجود کاربر
		_, err := db.GetUserByName(username)
		if err == nil {
			return c.JSON(http.StatusBadRequest, map[string]interface{}{
				"success": false,
				"error":   "این نام کاربری قبلاً استفاده شده است",
			})
		}

		// تعیین نقش کاربر
		var userRole model.UserRole
		switch role {
		case "admin":
			userRole = model.RoleAdmin
		case "manager":
			userRole = model.RoleManager
		default:
			userRole = model.RoleUser
		}

		// ایجاد کاربر جدید
		user := model.User{
			Username: username,
			Role:     userRole,
		}

		// هش کردن رمز عبور
		hashedPassword, err := util.HashPassword(password)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]interface{}{
				"success": false,
				"error":   "خطا در پردازش رمز عبور",
			})
		}
		user.PasswordHash = hashedPassword

		// ذخیره کاربر
		if err := db.SaveUser(user); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]interface{}{
				"success": false,
				"error":   "خطا در ذخیره کاربر",
			})
		}

		return c.JSON(http.StatusOK, map[string]interface{}{
			"success": true,
			"message": "کاربر با موفقیت ایجاد شد",
		})
	}
}

// RemoveUser handler
func RemoveUser(db store.IStore) echo.HandlerFunc {
	return func(c echo.Context) error {
		data := make(map[string]interface{})
		err := json.NewDecoder(c.Request().Body).Decode(&data)

		if err != nil {
			return c.JSON(http.StatusBadRequest, jsonHTTPResponse{false, "Bad post data"})
		}

		username := data["username"].(string)

		if !usernameRegexp.MatchString(username) {
			return c.JSON(http.StatusBadRequest, jsonHTTPResponse{false, "Please provide a valid username"})
		}

		if username == currentUser(c) {
			return c.JSON(http.StatusForbidden, jsonHTTPResponse{false, "User cannot delete itself"})
		}
		// delete user from database

		if err := db.DeleteUser(username); err != nil {
			log.Error("Cannot delete user: ", err)
			return c.JSON(http.StatusInternalServerError, jsonHTTPResponse{false, "Cannot delete user from database"})
		}

		log.Infof("Removed user: %s", username)

		return c.JSON(http.StatusOK, jsonHTTPResponse{true, "User removed"})
	}
}

// WireGuardClients handler
func WireGuardClients(db store.IStore) echo.HandlerFunc {
	return func(c echo.Context) error {
		clientDataList, err := db.GetClients(true)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, jsonHTTPResponse{
				false, fmt.Sprintf("Cannot get client list: %v", err),
			})
		}

		return c.Render(http.StatusOK, "clients.html", map[string]interface{}{
			"baseData":       model.BaseData{Active: "", CurrentUser: currentUser(c), Admin: isAdmin(c)},
			"clientDataList": clientDataList,
		})
	}
}

// GetClients handler return a JSON list of Wireguard client data
func GetClients(db store.IStore) echo.HandlerFunc {
	return func(c echo.Context) error {
		clientDataList, err := db.GetClients(true)
		if err != nil {
			log.Error("Error getting clients: ", err)
			return c.JSON(http.StatusInternalServerError, jsonHTTPResponse{
				false, fmt.Sprintf("Cannot get client list: %v", err),
			})
		}

		// اگر لیست خالی باشد، یک آرایه خالی برگردانیم نه null
		if clientDataList == nil {
			clientDataList = make([]model.ClientData, 0)
		}

		// Get WireGuard usage data for online status and data usage
		usageMap, err := getWireGuardUsage()
		if err != nil {
			log.Error("Error getting WireGuard usage: ", err)
			// Continue without usage data
			usageMap = make(map[string]peerUsage)
		}

		// Process each client and fill subnet range
		processedList := make([]model.ClientData, 0, len(clientDataList))
		for _, clientData := range clientDataList {
			if clientData.Client != nil { // اطمینان از معتبر بودن داده
				// Add online status and data usage
				if usage, ok := usageMap[clientData.Client.PublicKey]; ok {
					// Check if client is online (last handshake within 3 minutes)
					clientData.Client.Status = "offline"
					if !usage.LastHandshake.IsZero() && time.Since(usage.LastHandshake).Minutes() < 3 {
						clientData.Client.Status = "online"
					}

					// Update used quota
					totalBytes := usage.Rx + usage.Tx
					clientData.Client.UsedQuota = int64(totalBytes)

					// Add last handshake time; preserve previous if no handshake yet
					lastSeen := usage.LastHandshake
					if lastSeen.IsZero() && clientData.Client.PersistentUsageData != nil {
						lastSeen = clientData.Client.PersistentUsageData.LastSeen
					}
					// Only set LastHandshake if it's not zero, otherwise leave it as zero (will be null in JSON)
					if !lastSeen.IsZero() {
						clientData.Client.LastHandshake = lastSeen
					}

					// Update persistent usage data
					if clientData.Client.PersistentUsageData == nil {
						clientData.Client.PersistentUsageData = &model.ClientUsageData{
							LastInterfaceBytesReceived: usage.Rx,
							LastInterfaceBytesSent:     usage.Tx,
						}
					}

					// Only update if we have new data
					if !usage.LastHandshake.IsZero() {
						clientData.Client.PersistentUsageData.LastSeen = usage.LastHandshake

						// Update first seen if not set
						if clientData.Client.PersistentUsageData.FirstSeen.IsZero() {
							clientData.Client.PersistentUsageData.FirstSeen = usage.LastHandshake
						}
					}

					// Compute usage delta based on last counters
					deltaRx := usage.Rx
					if usage.Rx >= clientData.Client.PersistentUsageData.LastInterfaceBytesReceived {
						deltaRx = usage.Rx - clientData.Client.PersistentUsageData.LastInterfaceBytesReceived
					}
					deltaTx := usage.Tx
					if usage.Tx >= clientData.Client.PersistentUsageData.LastInterfaceBytesSent {
						deltaTx = usage.Tx - clientData.Client.PersistentUsageData.LastInterfaceBytesSent
					}

					// Accumulate totals
					clientData.Client.PersistentUsageData.TotalBytesReceived += deltaRx
					clientData.Client.PersistentUsageData.TotalBytesSent += deltaTx

					// Update last interface counters
					clientData.Client.PersistentUsageData.LastInterfaceBytesReceived = usage.Rx
					clientData.Client.PersistentUsageData.LastInterfaceBytesSent = usage.Tx

					clientData.Client.PersistentUsageData.UpdatedAt = time.Now().UTC()

					// Update UsedQuota from persistent totals
					clientData.Client.UsedQuota = int64(clientData.Client.PersistentUsageData.TotalBytesReceived + clientData.Client.PersistentUsageData.TotalBytesSent)

					// Save the updated client data
					if err := db.SaveClient(*clientData.Client); err != nil {
						log.Error("Error saving client persistent data: ", err)
					}
				} else {
					clientData.Client.Status = "offline"
					// Use persistent data if available
					if clientData.Client.PersistentUsageData != nil {
						clientData.Client.UsedQuota = int64(clientData.Client.PersistentUsageData.TotalBytesReceived + clientData.Client.PersistentUsageData.TotalBytesSent)
						// Only set LastHandshake if LastSeen is not zero
						if !clientData.Client.PersistentUsageData.LastSeen.IsZero() {
							clientData.Client.LastHandshake = clientData.Client.PersistentUsageData.LastSeen
						}
					} else {
						clientData.Client.UsedQuota = 0
						// Ensure LastHandshake is zero (will be null in JSON)
						clientData.Client.LastHandshake = time.Time{}
					}
				}

				processedList = append(processedList, util.FillClientSubnetRange(clientData))
			}
		}

		// Return as a structured response
		return c.JSON(http.StatusOK, map[string]interface{}{
			"success": true,
			"clients": processedList,
		})
	}
}

// GetClient handler returns a JSON object of Wireguard client data
func GetClient(db store.IStore) echo.HandlerFunc {
	return func(c echo.Context) error {
		clientID := c.Param("id")

		if _, err := xid.FromString(clientID); err != nil {
			return c.JSON(http.StatusBadRequest, jsonHTTPResponse{false, "Please provide a valid client ID"})
		}

		qrCodeSettings := model.QRCodeSettings{
			Enabled:    true,
			IncludeDNS: true,
			IncludeMTU: true,
		}

		clientData, err := db.GetClientByID(clientID, qrCodeSettings)
		if err != nil {
			return c.JSON(http.StatusNotFound, jsonHTTPResponse{false, "Client not found"})
		}

		return c.JSON(http.StatusOK, util.FillClientSubnetRange(clientData))
	}
}

// GetClientQRCode handler returns QR code image for a client
func GetClientQRCode(db store.IStore) echo.HandlerFunc {
	return func(c echo.Context) error {
		clientID := c.Param("id")

		if _, err := xid.FromString(clientID); err != nil {
			return c.JSON(http.StatusBadRequest, jsonHTTPResponse{false, "Please provide a valid client ID"})
		}

		qrCodeSettings := model.QRCodeSettings{
			Enabled:    true,
			IncludeDNS: true,
			IncludeMTU: true,
		}

		clientData, err := db.GetClientByID(clientID, qrCodeSettings)
		if err != nil {
			return c.JSON(http.StatusNotFound, jsonHTTPResponse{false, "Client not found"})
		}

		if clientData.QRCode == "" {
			return c.JSON(http.StatusNotFound, jsonHTTPResponse{false, "QR code not available"})
		}

		// Remove data:image/png;base64, prefix if present
		qrData := strings.TrimPrefix(clientData.QRCode, "data:image/png;base64,")

		// Decode base64 to bytes
		qrBytes, err := base64.StdEncoding.DecodeString(qrData)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, jsonHTTPResponse{false, "Invalid QR code data"})
		}

		// Set response headers for image
		c.Response().Header().Set(echo.HeaderContentType, "image/png")
		c.Response().Header().Set(echo.HeaderCacheControl, "public, max-age=3600")

		return c.Blob(http.StatusOK, "image/png", qrBytes)
	}
}

// NewClient handler
func NewClient(db store.IStore) echo.HandlerFunc {
	return func(c echo.Context) error {
		var client model.Client
		c.Bind(&client)

		// اعتبارسنجی مقدار Quota
		if client.Quota < 0 {
			return c.JSON(http.StatusBadRequest, jsonHTTPResponse{false, "Quota cannot be negative"})
		}
		if client.ExpirationDays < 0 {
			return c.JSON(http.StatusBadRequest, jsonHTTPResponse{false, "Expiration days cannot be negative"})
		}
		client.Expiration = time.Time{}
		client.FirstConnectedAt = time.Time{}

		// Validate Telegram userid if provided
		if client.TgUserid != "" {
			idNum, err := strconv.ParseInt(client.TgUserid, 10, 64)
			if err != nil || idNum == 0 {
				return c.JSON(http.StatusBadRequest, jsonHTTPResponse{false, "Telegram userid must be a non-zero number"})
			}
		}

		// read server information
		server, err := db.GetServer()
		if err != nil {
			log.Error("Cannot fetch server from database: ", err)
			return c.JSON(http.StatusInternalServerError, jsonHTTPResponse{false, err.Error()})
		}

		// validate the input Allocation IPs
		allocatedIPs, err := util.GetAllocatedIPs("")
		check, err := util.ValidateIPAllocation(server.Interface.Addresses, allocatedIPs, client.AllocatedIPs)
		if !check {
			return c.JSON(http.StatusBadRequest, jsonHTTPResponse{false, fmt.Sprintf("%s", err)})
		}

		// validate the input AllowedIPs
		if util.ValidateAllowedIPs(client.AllowedIPs) == false {
			log.Warnf("Invalid Allowed IPs input from user: %v", client.AllowedIPs)
			return c.JSON(http.StatusBadRequest, jsonHTTPResponse{false, "Allowed IPs must be in CIDR format"})
		}

		// validate extra AllowedIPs
		if util.ValidateExtraAllowedIPs(client.ExtraAllowedIPs) == false {
			log.Warnf("Invalid Extra AllowedIPs input from user: %v", client.ExtraAllowedIPs)
			return c.JSON(http.StatusBadRequest, jsonHTTPResponse{false, "Extra AllowedIPs must be in CIDR format"})
		}

		// gen ID
		guid := xid.New()
		client.ID = guid.String()

		// gen Wireguard key pair
		if client.PublicKey == "" {
			key, err := wgtypes.GeneratePrivateKey()
			if err != nil {
				log.Error("Cannot generate wireguard key pair: ", err)
				return c.JSON(http.StatusInternalServerError, jsonHTTPResponse{false, "Cannot generate Wireguard key pair"})
			}
			client.PrivateKey = key.String()
			client.PublicKey = key.PublicKey().String()
		} else {
			_, err := wgtypes.ParseKey(client.PublicKey)
			if err != nil {
				log.Error("Cannot verify wireguard public key: ", err)
				return c.JSON(http.StatusInternalServerError, jsonHTTPResponse{false, "Cannot verify Wireguard public key"})
			}
			// check for duplicates
			clients, err := db.GetClients(false)
			if err != nil {
				log.Error("Cannot get clients for duplicate check")
				return c.JSON(http.StatusInternalServerError, jsonHTTPResponse{false, "Cannot get clients for duplicate check"})
			}
			for _, other := range clients {
				if other.Client.PublicKey == client.PublicKey {
					log.Error("Duplicate Public Key")
					return c.JSON(http.StatusInternalServerError, jsonHTTPResponse{false, "Duplicate Public Key"})
				}
			}
		}

		if client.PresharedKey == "" {
			presharedKey, err := wgtypes.GenerateKey()
			if err != nil {
				log.Error("Cannot generated preshared key: ", err)
				return c.JSON(http.StatusInternalServerError, jsonHTTPResponse{
					false, "Cannot generate Wireguard preshared key",
				})
			}
			client.PresharedKey = presharedKey.String()
		} else if client.PresharedKey == "-" {
			client.PresharedKey = ""
			log.Infof("skipped PresharedKey generation for user: %v", client.Name)
		} else {
			_, err := wgtypes.ParseKey(client.PresharedKey)
			if err != nil {
				log.Error("Cannot verify wireguard preshared key: ", err)
				return c.JSON(http.StatusInternalServerError, jsonHTTPResponse{false, "Cannot verify Wireguard preshared key"})
			}
		}

		client.CreatedBy = currentUser(c)
		client.CreatedAt = time.Now().UTC()
		client.UpdatedAt = client.CreatedAt

		// write client to the database
		if err := db.SaveClient(client); err != nil {
			return c.JSON(http.StatusInternalServerError, jsonHTTPResponse{false, err.Error()})
		}
		log.Infof("Created wireguard client: %v", client)

		// کانفیگ به صورت خودکار اعمال نمی‌شود
		return c.JSON(http.StatusOK, jsonHTTPResponse{true, "Created client successfully"})
	}
}

// EmailClient handler to send the configuration via email
func EmailClient(db store.IStore, mailer emailer.Emailer, emailSubject, emailContent string) echo.HandlerFunc {
	type clientIdEmailPayload struct {
		ID    string `json:"id"`
		Email string `json:"email"`
	}

	return func(c echo.Context) error {
		var payload clientIdEmailPayload
		c.Bind(&payload)
		// TODO validate email

		if _, err := xid.FromString(payload.ID); err != nil {
			return c.JSON(http.StatusBadRequest, jsonHTTPResponse{false, "Please provide a valid client ID"})
		}

		qrCodeSettings := model.QRCodeSettings{
			Enabled:    true,
			IncludeDNS: true,
			IncludeMTU: true,
		}
		clientData, err := db.GetClientByID(payload.ID, qrCodeSettings)
		if err != nil {
			log.Errorf("Cannot generate client id %s config file for downloading: %v", payload.ID, err)
			return c.JSON(http.StatusNotFound, jsonHTTPResponse{false, "Client not found"})
		}

		// build config
		server, _ := db.GetServer()
		globalSettings, _ := db.GetGlobalSettings()
		config := util.BuildClientConfig(*clientData.Client, server, globalSettings)

		cfgAtt := emailer.Attachment{Name: "wg0.conf", Data: []byte(config)}
		var attachments []emailer.Attachment
		if clientData.Client.PrivateKey != "" {
			qrdata, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(clientData.QRCode, "data:image/png;base64,"))
			if err != nil {
				return c.JSON(http.StatusInternalServerError, jsonHTTPResponse{false, "decoding: " + err.Error()})
			}
			qrAtt := emailer.Attachment{Name: "wg.png", Data: qrdata}
			attachments = []emailer.Attachment{cfgAtt, qrAtt}
		} else {
			attachments = []emailer.Attachment{cfgAtt}
		}
		err = mailer.Send(
			clientData.Client.Name,
			payload.Email,
			emailSubject,
			emailContent,
			attachments,
		)

		if err != nil {
			return c.JSON(http.StatusInternalServerError, jsonHTTPResponse{false, err.Error()})
		}

		return c.JSON(http.StatusOK, jsonHTTPResponse{true, "Email sent successfully"})
	}
}

// SendTelegramClient handler to send the configuration via Telegram
func SendTelegramClient(db store.IStore) echo.HandlerFunc {
	type clientIdUseridPayload struct {
		ID     string `json:"id"`
		Userid string `json:"userid"`
	}
	return func(c echo.Context) error {
		var payload clientIdUseridPayload
		c.Bind(&payload)

		clientData, err := db.GetClientByID(payload.ID, model.QRCodeSettings{Enabled: false})
		if err != nil {
			log.Errorf("Cannot generate client id %s config file for downloading: %v", payload.ID, err)
			return c.JSON(http.StatusNotFound, jsonHTTPResponse{false, "Client not found"})
		}

		// build config
		server, _ := db.GetServer()
		globalSettings, _ := db.GetGlobalSettings()
		config := util.BuildClientConfig(*clientData.Client, server, globalSettings)
		configData := []byte(config)
		var qrData []byte

		if clientData.Client.PrivateKey != "" {
			qrData, err = qrcode.Encode(config, qrcode.Medium, 512)
			if err != nil {
				return c.JSON(http.StatusInternalServerError, jsonHTTPResponse{false, "qr gen: " + err.Error()})
			}
		}

		userid, err := strconv.ParseInt(clientData.Client.TgUserid, 10, 64)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, jsonHTTPResponse{false, "userid: " + err.Error()})
		}

		err = telegram.SendConfig(userid, clientData.Client.Name, configData, qrData, false)

		if err != nil {
			return c.JSON(http.StatusInternalServerError, jsonHTTPResponse{false, err.Error()})
		}

		return c.JSON(http.StatusOK, jsonHTTPResponse{true, "Telegram message sent successfully"})
	}
}

// UpdateClient handler to update client information
func UpdateClient(db store.IStore) echo.HandlerFunc {
	return func(c echo.Context) error {
		var _client model.Client
		c.Bind(&_client)

		if _, err := xid.FromString(_client.ID); err != nil {
			return c.JSON(http.StatusBadRequest, jsonHTTPResponse{false, "Please provide a valid client ID"})
		}

		// validate client existence
		clientData, err := db.GetClientByID(_client.ID, model.QRCodeSettings{Enabled: false})
		if err != nil {
			return c.JSON(http.StatusNotFound, jsonHTTPResponse{false, "Client not found"})
		}

		// Validate Telegram userid if provided
		if _client.TgUserid != "" {
			idNum, err := strconv.ParseInt(_client.TgUserid, 10, 64)
			if err != nil || idNum == 0 {
				return c.JSON(http.StatusBadRequest, jsonHTTPResponse{false, "Telegram userid must be a non-zero number"})
			}
		}

		server, err := db.GetServer()
		if err != nil {
			return c.JSON(http.StatusBadRequest, jsonHTTPResponse{
				false, fmt.Sprintf("Cannot fetch server config: %s", err),
			})
		}

		client := *clientData.Client

		// validate the input Allocation IPs
		allocatedIPs, err := util.GetAllocatedIPs(client.ID)
		check, err := util.ValidateIPAllocation(server.Interface.Addresses, allocatedIPs, _client.AllocatedIPs)
		if !check {
			return c.JSON(http.StatusBadRequest, jsonHTTPResponse{false, fmt.Sprintf("%s", err)})
		}

		// validate the input AllowedIPs
		if util.ValidateAllowedIPs(_client.AllowedIPs) == false {
			log.Warnf("Invalid Allowed IPs input from user: %v", _client.AllowedIPs)
			return c.JSON(http.StatusBadRequest, jsonHTTPResponse{false, "Allowed IPs must be in CIDR format"})
		}

		if util.ValidateExtraAllowedIPs(_client.ExtraAllowedIPs) == false {
			log.Warnf("Invalid Extra AllowedIPs input from user: %v", _client.ExtraAllowedIPs)
			return c.JSON(http.StatusBadRequest, jsonHTTPResponse{false, "Extra Allowed IPs must be in CIDR format"})
		}

		// update Wireguard Client PublicKey
		if client.PublicKey != _client.PublicKey && _client.PublicKey != "" {
			_, err := wgtypes.ParseKey(_client.PublicKey)
			if err != nil {
				log.Error("Cannot verify provided Wireguard public key: ", err)
				return c.JSON(http.StatusInternalServerError, jsonHTTPResponse{false, "Cannot verify provided Wireguard public key"})
			}
			// check for duplicates
			clients, err := db.GetClients(false)
			if err != nil {
				log.Error("Cannot get client list for duplicate public key check")
				return c.JSON(http.StatusInternalServerError, jsonHTTPResponse{false, "Cannot get client list for duplicate public key check"})
			}
			for _, other := range clients {
				if other.Client.PublicKey == _client.PublicKey {
					log.Error("Duplicate Public Key")
					return c.JSON(http.StatusInternalServerError, jsonHTTPResponse{false, "Duplicate Public Key"})
				}
			}

			// When replacing any PublicKey, discard any locally stored Wireguard Client PrivateKey
			// Client PubKey no longer corresponds to locally stored PrivKey.
			if client.PrivateKey != "" {
				client.PrivateKey = ""
			}
		}

		// update Wireguard Client PresharedKey
		if client.PresharedKey != _client.PresharedKey && _client.PresharedKey != "" {
			_, err := wgtypes.ParseKey(_client.PresharedKey)
			if err != nil {
				log.Error("Cannot verify provided Wireguard preshared key: ", err)
				return c.JSON(http.StatusInternalServerError, jsonHTTPResponse{false, "Cannot verify provided Wireguard preshared key"})
			}
		}

		// حالا فیلدهای جدید را از _client به client منتقل می‌کنیم
		client.Quota = _client.Quota
		client.ExpirationDays = _client.ExpirationDays
		client.Expiration = _client.Expiration
		if !_client.FirstConnectedAt.IsZero() {
			client.FirstConnectedAt = _client.FirstConnectedAt
		}

		// اعتبارسنجی Quota و Expiration
		if client.Quota < 0 {
			return c.JSON(http.StatusBadRequest, jsonHTTPResponse{false, "Quota cannot be negative"})
		}
		if client.ExpirationDays < 0 {
			return c.JSON(http.StatusBadRequest, jsonHTTPResponse{false, "Expiration days cannot be negative"})
		}

		// map other data
		client.Name = _client.Name
		client.Email = _client.Email
		client.TgUserid = _client.TgUserid
		client.Enabled = _client.Enabled
		client.UseServerDNS = _client.UseServerDNS
		client.AllocatedIPs = _client.AllocatedIPs
		client.AllowedIPs = _client.AllowedIPs
		client.ExtraAllowedIPs = _client.ExtraAllowedIPs
		client.Endpoint = _client.Endpoint
		client.PublicKey = _client.PublicKey
		client.PresharedKey = _client.PresharedKey
		client.UpdatedAt = time.Now().UTC()
		client.AdditionalNotes = strings.ReplaceAll(strings.Trim(_client.AdditionalNotes, "\r\n"), "\r\n", "\n")

		// write to the database
		if err := db.SaveClient(client); err != nil {
			return c.JSON(http.StatusInternalServerError, jsonHTTPResponse{false, err.Error()})
		}
		log.Infof("Updated client information successfully => %v", client)

		// کانفیگ به صورت خودکار اعمال نمی‌شود
		return c.JSON(http.StatusOK, jsonHTTPResponse{true, "Updated client successfully"})
	}
}

// SetClientStatus handler to enable / disable a client
func SetClientStatus(db store.IStore) echo.HandlerFunc {
	return func(c echo.Context) error {
		var clientID string
		var status bool
		var isAutomatic bool

		// پشتیبانی از هر دو متد GET و POST
		switch c.Request().Method {
		case "GET":
			clientID = c.Param("id")
			statusStr := c.Param("status")
			status = statusStr == "true"
			automaticStr := c.QueryParam("automatic")
			isAutomatic = automaticStr == "true"
		case "POST":
			clientID = c.Param("id")
			if clientID == "" {
				data := make(map[string]interface{})
				if err := json.NewDecoder(c.Request().Body).Decode(&data); err != nil {
					return c.JSON(http.StatusBadRequest, jsonHTTPResponse{false, "Invalid request data"})
				}
				if id, ok := data["id"].(string); ok {
					clientID = id
				}
				if s, ok := data["status"].(bool); ok {
					status = s
				}
				if a, ok := data["automatic"].(bool); ok {
					isAutomatic = a
				}
			} else {
				statusStr := c.Param("status")
				status = statusStr == "true"
				automaticStr := c.QueryParam("automatic")
				isAutomatic = automaticStr == "true"
			}
		default:
			return c.JSON(http.StatusMethodNotAllowed, jsonHTTPResponse{false, "Method not allowed"})
		}

		if clientID == "" {
			return c.JSON(http.StatusBadRequest, jsonHTTPResponse{false, "Client ID is required"})
		}

		if _, err := xid.FromString(clientID); err != nil {
			return c.JSON(http.StatusBadRequest, jsonHTTPResponse{false, "Please provide a valid client ID"})
		}

		// Get client data
		clientData, err := db.GetClientByID(clientID, model.QRCodeSettings{Enabled: false})
		if err != nil {
			log.Printf("Error getting client: %v", err)
			return c.JSON(http.StatusNotFound, jsonHTTPResponse{false, "Client not found"})
		}

		client := *clientData.Client

		// اگر وضعیت فعلی با وضعیت درخواستی یکسان است، نیازی به تغییر نیست
		if client.Enabled == status {
			return c.JSON(http.StatusOK, jsonHTTPResponse{true, "Client status already set"})
		}

		// اگر درخواست فعال‌سازی دستی است
		if status && !isAutomatic {
			// بررسی شرایط فعال‌سازی
			// if !client.Expiration.IsZero() && time.Now().After(client.Expiration) {
			// 	return c.JSON(http.StatusBadRequest, jsonHTTPResponse{false, "Cannot enable client: expiration date has passed"})
			// }

			// if client.Quota > 0 && client.UsedQuota >= client.Quota {
			// 	return c.JSON(http.StatusBadRequest, jsonHTTPResponse{false, "Cannot enable client: quota limit exceeded"})
			// }

			// فعال‌سازی کلاینت
			client.Enabled = true
			if err := db.SaveClient(client); err != nil {
				return c.JSON(http.StatusInternalServerError, jsonHTTPResponse{false, err.Error()})
			}

			return c.JSON(http.StatusOK, jsonHTTPResponse{true, "Client enabled successfully"})
		}

		// اگر درخواست غیرفعال‌سازی است (دستی یا خودکار)
		client.Enabled = false
		if err := db.SaveClient(client); err != nil {
			return c.JSON(http.StatusInternalServerError, jsonHTTPResponse{false, err.Error()})
		}

		// اگر غیرفعال‌سازی خودکار است، کانفیگ را اعمال می‌کنیم
		if isAutomatic {
			if err := applyWireGuardConfig(db); err != nil {
				log.Printf("Error applying WireGuard config after automatic disable: %v", err)
				// ادامه می‌دهیم چون کلاینت در هر صورت غیرفعال شده است
			} else {
				log.Printf("WireGuard config applied after automatic disable of client %s", client.Name)
			}
		}

		statusText := "disabled"
		if status {
			statusText = "enabled"
		}
		return c.JSON(http.StatusOK, jsonHTTPResponse{true, fmt.Sprintf("Client %s successfully", statusText)})
	}
}

// DownloadClient handler
func DownloadClient(db store.IStore) echo.HandlerFunc {
	return func(c echo.Context) error {
		clientID := c.QueryParam("clientid")
		if clientID == "" {
			return c.JSON(http.StatusNotFound, jsonHTTPResponse{false, "Missing clientid parameter"})
		}

		if _, err := xid.FromString(clientID); err != nil {
			return c.JSON(http.StatusBadRequest, jsonHTTPResponse{false, "Please provide a valid client ID"})
		}

		clientData, err := db.GetClientByID(clientID, model.QRCodeSettings{Enabled: false})
		if err != nil {
			log.Errorf("Cannot generate client id %s config file for downloading: %v", clientID, err)
			return c.JSON(http.StatusNotFound, jsonHTTPResponse{false, "Client not found"})
		}

		// build config
		server, err := db.GetServer()
		if err != nil {
			return c.JSON(http.StatusInternalServerError, jsonHTTPResponse{false, err.Error()})
		}
		globalSettings, err := db.GetGlobalSettings()
		if err != nil {
			return c.JSON(http.StatusInternalServerError, jsonHTTPResponse{false, err.Error()})
		}
		config := util.BuildClientConfig(*clientData.Client, server, globalSettings)

		// create io reader from string
		reader := strings.NewReader(config)

		// set response header for downloading
		c.Response().Header().Set(echo.HeaderContentDisposition, fmt.Sprintf("attachment; filename=%s.conf", clientData.Client.Name))
		return c.Stream(http.StatusOK, "text/conf", reader)
	}
}

// RemoveClient handler
func RemoveClient(db store.IStore) echo.HandlerFunc {
	return func(c echo.Context) error {
		client := new(model.Client)
		c.Bind(client)

		if _, err := xid.FromString(client.ID); err != nil {
			return c.JSON(http.StatusBadRequest, jsonHTTPResponse{false, "Please provide a valid client ID"})
		}

		// delete client from database

		if err := db.DeleteClient(client.ID); err != nil {
			log.Error("Cannot delete wireguard client: ", err)
			return c.JSON(http.StatusInternalServerError, jsonHTTPResponse{false, "Cannot delete client from database"})
		}

		log.Infof("Removed wireguard client: %v", client)
		return c.JSON(http.StatusOK, jsonHTTPResponse{true, "Client removed"})
	}
}

// TerminateClient handler to terminate a client connection
func TerminateClient(db store.IStore, tmplDir fs.FS) echo.HandlerFunc {
	return func(c echo.Context) error {
		// Parse request body
		data := make(map[string]interface{})
		if err := json.NewDecoder(c.Request().Body).Decode(&data); err != nil {
			return c.JSON(http.StatusBadRequest, jsonHTTPResponse{false, "Bad post data"})
		}

		clientID := data["id"].(string)
		if _, err := xid.FromString(clientID); err != nil {
			return c.JSON(http.StatusBadRequest, jsonHTTPResponse{false, "Please provide a valid client ID"})
		}

		// Get client data
		clientData, err := db.GetClientByID(clientID, model.QRCodeSettings{Enabled: false})
		if err != nil {
			return c.JSON(http.StatusNotFound, jsonHTTPResponse{false, "Client not found"})
		}

		// Get settings for interface name
		settings, err := db.GetGlobalSettings()
		if err != nil {
			return c.JSON(http.StatusInternalServerError, jsonHTTPResponse{false, "Cannot get global settings"})
		}

		// Get interface name from config file path or use default
		interfaceName := "wg0"
		if settings.ConfigFilePath != "" {
			parts := strings.Split(settings.ConfigFilePath, "/")
			if len(parts) > 0 {
				baseName := parts[len(parts)-1]
				interfaceName = strings.TrimSuffix(baseName, ".conf")
			}
		}

		// Create WireGuard client
		wgClient, err := wgctrl.New()
		if err != nil {
			return c.JSON(http.StatusInternalServerError, jsonHTTPResponse{false, "Cannot create WireGuard client"})
		}
		defer wgClient.Close()

		// Parse public key
		pubKey, err := wgtypes.ParseKey(clientData.Client.PublicKey)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, jsonHTTPResponse{false, "Cannot parse client public key"})
		}

		// Remove peer from interface
		peerConfig := wgtypes.PeerConfig{
			PublicKey: pubKey,
			Remove:    true,
		}

		err = wgClient.ConfigureDevice(interfaceName, wgtypes.Config{
			Peers: []wgtypes.PeerConfig{peerConfig},
		})
		if err != nil {
			return c.JSON(http.StatusInternalServerError, jsonHTTPResponse{false, fmt.Sprintf("Cannot remove peer: %v", err)})
		}

		// Write new configuration
		server, err := db.GetServer()
		if err != nil {
			return c.JSON(http.StatusInternalServerError, jsonHTTPResponse{false, "Cannot get server config"})
		}

		clients, err := db.GetClients(false)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, jsonHTTPResponse{false, "Cannot get clients"})
		}

		users, err := db.GetUsers()
		if err != nil {
			return c.JSON(http.StatusInternalServerError, jsonHTTPResponse{false, "Cannot get users"})
		}

		err = util.WriteWireGuardServerConfig(tmplDir, server, clients, users, settings)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, jsonHTTPResponse{false, fmt.Sprintf("Cannot write config: %v", err)})
		}

		log.Infof("Terminated client %s (%s)", clientData.Client.Name, clientData.Client.ID)
		return c.JSON(http.StatusOK, jsonHTTPResponse{true, "Client terminated successfully"})
	}
}

// WireGuardServer handler
func WireGuardServer(db store.IStore) echo.HandlerFunc {
	return func(c echo.Context) error {
		server, err := db.GetServer()
		if err != nil {
			log.Error("Cannot get server config: ", err)
		}

		return c.Render(http.StatusOK, "server.html", map[string]interface{}{
			"baseData":        model.BaseData{Active: "wg-server", CurrentUser: currentUser(c), Admin: isAdmin(c)},
			"serverInterface": server.Interface,
			"serverKeyPair":   server.KeyPair,
		})
	}
}

// WireGuardServerInterfaces handler
func WireGuardServerInterfaces(db store.IStore) echo.HandlerFunc {
	return func(c echo.Context) error {
		var serverInterface model.ServerInterface
		if err := json.NewDecoder(c.Request().Body).Decode(&serverInterface); err != nil {
			log.Warnf("Cannot parse server interface request: %v", err)
			return c.JSON(http.StatusBadRequest, jsonHTTPResponse{false, "Invalid request"})
		}

		// validate the input addresses
		if util.ValidateServerAddresses(serverInterface.Addresses) == false {
			log.Warnf("Invalid server interface addresses input from user: %v", serverInterface.Addresses)
			return c.JSON(http.StatusBadRequest, jsonHTTPResponse{false, "Interface IP address must be in CIDR format"})
		}

		if serverInterface.ListenPort <= 0 || serverInterface.ListenPort > 65535 {
			log.Warnf("Invalid listen port: %v", serverInterface.ListenPort)
			return c.JSON(http.StatusBadRequest, jsonHTTPResponse{false, "Port must be in range 1..65535"})
		}

		serverInterface.UpdatedAt = time.Now().UTC()

		// write config to the database

		if err := db.SaveServerInterface(serverInterface); err != nil {
			return c.JSON(http.StatusInternalServerError, jsonHTTPResponse{false, "Interface IP address must be in CIDR format"})
		}
		log.Infof("Updated wireguard server interfaces settings: %v", serverInterface)

		return c.JSON(http.StatusOK, jsonHTTPResponse{true, "Updated interface addresses successfully"})
	}
}

// WireGuardServerKeyPair handler to generate private and public keys
func WireGuardServerKeyPair(db store.IStore) echo.HandlerFunc {
	return func(c echo.Context) error {
		// gen Wireguard key pair
		key, err := wgtypes.GeneratePrivateKey()
		if err != nil {
			log.Error("Cannot generate wireguard key pair: ", err)
			return c.JSON(http.StatusInternalServerError, jsonHTTPResponse{false, "Cannot generate Wireguard key pair"})
		}

		var serverKeyPair model.ServerKeypair
		serverKeyPair.PrivateKey = key.String()
		serverKeyPair.PublicKey = key.PublicKey().String()
		serverKeyPair.UpdatedAt = time.Now().UTC()

		if err := db.SaveServerKeyPair(serverKeyPair); err != nil {
			return c.JSON(http.StatusInternalServerError, jsonHTTPResponse{false, "Cannot generate Wireguard key pair"})
		}
		log.Infof("Updated wireguard server interfaces settings: %v", serverKeyPair)

		return c.JSON(http.StatusOK, serverKeyPair)
	}
}

// GlobalSettings handler
func GlobalSettings(db store.IStore) echo.HandlerFunc {
	return func(c echo.Context) error {
		globalSettings, err := db.GetGlobalSettings()
		if err != nil {
			log.Error("Cannot get global settings: ", err)
		}

		return c.Render(http.StatusOK, "global_settings.html", map[string]interface{}{
			"baseData":       model.BaseData{Active: "global-settings", CurrentUser: currentUser(c), Admin: isAdmin(c)},
			"globalSettings": globalSettings,
		})
	}
}

// Status handler to show wireguard connection status
func Status(db store.IStore) echo.HandlerFunc {
	return func(c echo.Context) error {
		wgClient, err := wgctrl.New()
		if err != nil {
			return c.Render(http.StatusInternalServerError, "status.html", map[string]interface{}{
				"baseData": model.BaseData{Active: "status", CurrentUser: currentUser(c), Admin: isAdmin(c)},
				"error":    err.Error(),
				"devices":  nil,
			})
		}

		devices, err := wgClient.Devices()
		if err != nil {
			return c.Render(http.StatusInternalServerError, "status.html", map[string]interface{}{
				"baseData": model.BaseData{Active: "status", CurrentUser: currentUser(c), Admin: isAdmin(c)},
				"error":    err.Error(),
				"devices":  nil,
			})
		}

		devicesVm := make([]DeviceVM, 0, len(devices))
		if len(devices) > 0 {
			m := make(map[string]*model.Client)
			clients, err := db.GetClients(false)
			if err != nil {
				return c.Render(http.StatusInternalServerError, "status.html", map[string]interface{}{
					"baseData": model.BaseData{Active: "status", CurrentUser: currentUser(c), Admin: isAdmin(c)},
					"error":    err.Error(),
					"devices":  nil,
				})
			}
			for i := range clients {
				if clients[i].Client != nil {
					m[clients[i].Client.PublicKey] = clients[i].Client
				}
			}

			conv := map[bool]int{true: 1, false: 0}
			for i := range devices {
				devVm := DeviceVM{Name: devices[i].Name}
				for j := range devices[i].Peers {
					var allocatedIPs string
					for _, ip := range devices[i].Peers[j].AllowedIPs {
						if len(allocatedIPs) > 0 {
							allocatedIPs += "</br>"
						}
						allocatedIPs += ip.String()
					}
					pVm := PeerVM{
						PublicKey:         devices[i].Peers[j].PublicKey.String(),
						ReceivedBytes:     devices[i].Peers[j].ReceiveBytes,
						TransmitBytes:     devices[i].Peers[j].TransmitBytes,
						LastHandshakeTime: devices[i].Peers[j].LastHandshakeTime,
						LastHandshakeRel:  time.Since(devices[i].Peers[j].LastHandshakeTime),
						AllocatedIP:       allocatedIPs,
					}
					pVm.Connected = pVm.LastHandshakeRel.Minutes() < 3.

					if isAdmin(c) {
						pVm.Endpoint = devices[i].Peers[j].Endpoint.String()
					}

					if _client, ok := m[pVm.PublicKey]; ok {
						pVm.Name = _client.Name
						pVm.Email = _client.Email
					}
					devVm.Peers = append(devVm.Peers, pVm)
				}
				sort.SliceStable(devVm.Peers, func(i, j int) bool { return devVm.Peers[i].Name < devVm.Peers[j].Name })
				sort.SliceStable(devVm.Peers, func(i, j int) bool { return conv[devVm.Peers[i].Connected] > conv[devVm.Peers[j].Connected] })
				devicesVm = append(devicesVm, devVm)
			}
		}

		return c.Render(http.StatusOK, "status.html", map[string]interface{}{
			"baseData": model.BaseData{Active: "status", CurrentUser: currentUser(c), Admin: isAdmin(c)},
			"devices":  devicesVm,
			"error":    "",
		})
	}
}

// StatusData handler to return JSON status data for clients
func StatusData(db store.IStore) echo.HandlerFunc {
	return func(c echo.Context) error {
		log.Debug("Starting StatusData handler")
		wgClient, err := wgctrl.New()
		if err != nil {
			log.Error("Failed to create WireGuard client:", err)
			return c.JSON(http.StatusInternalServerError, jsonHTTPResponse{false, err.Error()})
		}

		devices, err := wgClient.Devices()
		if err != nil {
			log.Error("Failed to get WireGuard devices:", err)
			return c.JSON(http.StatusInternalServerError, jsonHTTPResponse{false, err.Error()})
		}
		log.Debugf("Found %d WireGuard devices", len(devices))

		devicesVm := make([]DeviceVM, 0, len(devices))
		if len(devices) > 0 {
			m := make(map[string]*model.Client)
			clients, err := db.GetClients(false)
			if err != nil {
				log.Error("Failed to get clients from database:", err)
				return c.JSON(http.StatusInternalServerError, jsonHTTPResponse{false, err.Error()})
			}
			log.Debugf("Found %d clients in database", len(clients))

			for i := range clients {
				if clients[i].Client != nil {
					m[clients[i].Client.PublicKey] = clients[i].Client
				}
			}

			conv := map[bool]int{true: 1, false: 0}
			for i := range devices {
				devVm := DeviceVM{Name: devices[i].Name}
				log.Debugf("Processing device %s with %d peers", devices[i].Name, len(devices[i].Peers))

				for j := range devices[i].Peers {
					var allocatedIPs string
					for _, ip := range devices[i].Peers[j].AllowedIPs {
						if len(allocatedIPs) > 0 {
							allocatedIPs += "</br>"
						}
						allocatedIPs += ip.String()
					}
					pVm := PeerVM{
						PublicKey:         devices[i].Peers[j].PublicKey.String(),
						ReceivedBytes:     devices[i].Peers[j].ReceiveBytes,
						TransmitBytes:     devices[i].Peers[j].TransmitBytes,
						LastHandshakeTime: devices[i].Peers[j].LastHandshakeTime,
						LastHandshakeRel:  time.Since(devices[i].Peers[j].LastHandshakeTime),
						AllocatedIP:       allocatedIPs,
					}
					pVm.Connected = pVm.LastHandshakeRel.Minutes() < 3.
					log.Debugf("Peer %s: last handshake %v ago, connected: %v",
						pVm.PublicKey[:8], pVm.LastHandshakeRel, pVm.Connected)

					if isAdmin(c) {
						pVm.Endpoint = devices[i].Peers[j].Endpoint.String()
					}

					if _client, ok := m[pVm.PublicKey]; ok {
						pVm.Name = _client.Name
						pVm.Email = _client.Email
						log.Debugf("Found client info for peer %s: name=%s", pVm.PublicKey[:8], pVm.Name)
					} else {
						log.Debugf("No client info found for peer %s", pVm.PublicKey[:8])
					}
					devVm.Peers = append(devVm.Peers, pVm)
				}
				sort.SliceStable(devVm.Peers, func(i, j int) bool { return devVm.Peers[i].Name < devVm.Peers[j].Name })
				sort.SliceStable(devVm.Peers, func(i, j int) bool { return conv[devVm.Peers[i].Connected] > conv[devVm.Peers[j].Connected] })
				devicesVm = append(devicesVm, devVm)
			}
		}

		log.Debug("StatusData handler completed successfully")
		return c.JSON(http.StatusOK, map[string]interface{}{
			"success": true,
			"devices": devicesVm,
		})
	}
}

// GlobalSettingSubmit handler to update the global settings
func GlobalSettingSubmit(db store.IStore) echo.HandlerFunc {
	return func(c echo.Context) error {
		var globalSettings model.GlobalSetting
		c.Bind(&globalSettings)

		// validate the input dns server list
		if util.ValidateIPAddressList(globalSettings.DNSServers) == false {
			log.Warnf("Invalid DNS server list input from user: %v", globalSettings.DNSServers)
			return c.JSON(http.StatusBadRequest, jsonHTTPResponse{false, "Invalid DNS server address"})
		}

		globalSettings.UpdatedAt = time.Now().UTC()

		// write config to the database
		if err := db.SaveGlobalSettings(globalSettings); err != nil {
			return c.JSON(http.StatusInternalServerError, jsonHTTPResponse{false, "Cannot save global settings"})
		}

		log.Infof("Updated global settings: %v", globalSettings)

		return c.JSON(http.StatusOK, jsonHTTPResponse{true, "Updated global settings successfully"})
	}
}

// DisplaySettingsSubmit handler to update display settings
func DisplaySettingsSubmit(db store.IStore) echo.HandlerFunc {
	return func(c echo.Context) error {
		var displaySettings struct {
			Timezone string `json:"timezone"`
			Language string `json:"language"`
		}
		c.Bind(&displaySettings)

		// Validate timezone
		if displaySettings.Timezone == "" {
			return c.JSON(http.StatusBadRequest, jsonHTTPResponse{false, "Timezone is required"})
		}

		// Validate language
		if displaySettings.Language == "" {
			return c.JSON(http.StatusBadRequest, jsonHTTPResponse{false, "Language is required"})
		}

		// Get current global settings
		currentSettings, err := db.GetGlobalSettings()
		if err != nil {
			return c.JSON(http.StatusInternalServerError, jsonHTTPResponse{false, "Cannot get current settings"})
		}

		// Update display settings
		currentSettings.Timezone = displaySettings.Timezone
		currentSettings.Language = displaySettings.Language
		currentSettings.UpdatedAt = time.Now().UTC()

		// Save updated settings
		if err := db.SaveGlobalSettings(currentSettings); err != nil {
			return c.JSON(http.StatusInternalServerError, jsonHTTPResponse{false, "Cannot save display settings"})
		}

		log.Infof("Updated display settings: timezone=%s, language=%s", displaySettings.Timezone, displaySettings.Language)

		return c.JSON(http.StatusOK, jsonHTTPResponse{true, "Updated display settings successfully"})
	}
}

// MachineIPAddresses handler to get local interface ip addresses
func MachineIPAddresses() echo.HandlerFunc {
	return func(c echo.Context) error {
		// get private ip addresses
		interfaceList, err := util.GetInterfaceIPs()
		if err != nil {
			return c.JSON(http.StatusInternalServerError, jsonHTTPResponse{false, "Cannot get machine ip addresses"})
		}

		// get public ip address
		// TODO: Remove the go-external-ip dependency
		publicInterface, err := util.GetPublicIP()
		if err != nil {
			log.Warn("Cannot get machine public ip address: ", err)
		} else {
			// prepend public ip to the list
			interfaceList = append([]model.Interface{publicInterface}, interfaceList...)
		}

		return c.JSON(http.StatusOK, interfaceList)
	}
}

// GetOrderedSubnetRanges handler to get the ordered list of subnet ranges
func GetOrderedSubnetRanges() echo.HandlerFunc {
	return func(c echo.Context) error {
		return c.JSON(http.StatusOK, util.SubnetRangesOrder)
	}
}

// SuggestIPAllocation handler to get the list of ip address for client
func SuggestIPAllocation(db store.IStore) echo.HandlerFunc {
	return func(c echo.Context) error {
		server, err := db.GetServer()
		if err != nil {
			log.Error("Cannot fetch server config from database: ", err)
			return c.JSON(http.StatusBadRequest, jsonHTTPResponse{false, err.Error()})
		}

		// return the list of suggestedIPs
		// we take the first available ip address from
		// each server's network addresses.
		suggestedIPs := make([]string, 0)
		allocatedIPs, err := util.GetAllocatedIPs("")
		if err != nil {
			log.Error("Cannot suggest ip allocation. Failed to get list of allocated ip addresses: ", err)
			return c.JSON(http.StatusInternalServerError, jsonHTTPResponse{
				false, "Cannot suggest ip allocation: failed to get list of allocated ip addresses",
			})
		}

		sr := c.QueryParam("sr")
		searchCIDRList := make([]string, 0)
		found := false

		// Use subnet range or default to interface addresses
		if util.SubnetRanges[sr] != nil {
			for _, cidr := range util.SubnetRanges[sr] {
				searchCIDRList = append(searchCIDRList, cidr.String())
			}
		} else {
			searchCIDRList = append(searchCIDRList, server.Interface.Addresses...)
		}

		// Save only unique IPs
		ipSet := make(map[string]struct{})

		for _, cidr := range searchCIDRList {
			ip, err := util.GetAvailableIP(cidr, allocatedIPs, server.Interface.Addresses)
			if err != nil {
				log.Error("Failed to get available ip from a CIDR: ", err)
				continue
			}
			found = true
			if strings.Contains(ip, ":") {
				ipSet[fmt.Sprintf("%s/128", ip)] = struct{}{}
			} else {
				ipSet[fmt.Sprintf("%s/32", ip)] = struct{}{}
			}
		}

		if !found {
			return c.JSON(http.StatusInternalServerError, jsonHTTPResponse{
				false,
				"Cannot suggest ip allocation: failed to get available ip. Try a different subnet or deallocate some ips.",
			})
		}

		for ip := range ipSet {
			suggestedIPs = append(suggestedIPs, ip)
		}

		return c.JSON(http.StatusOK, suggestedIPs)
	}
}

// ApplyServerConfig handler to write config file and restart Wireguard server
func ApplyServerConfig(db store.IStore, tmplDir fs.FS) echo.HandlerFunc {
	return func(c echo.Context) error {
		server, err := db.GetServer()
		if err != nil {
			log.Error("Cannot get server config: ", err)
			return c.JSON(http.StatusInternalServerError, jsonHTTPResponse{false, "Cannot get server config"})
		}

		clients, err := db.GetClients(false)
		if err != nil {
			log.Error("Cannot get client config: ", err)
			return c.JSON(http.StatusInternalServerError, jsonHTTPResponse{false, "Cannot get client config"})
		}

		users, err := db.GetUsers()
		if err != nil {
			log.Error("Cannot get users config: ", err)
			return c.JSON(http.StatusInternalServerError, jsonHTTPResponse{false, "Cannot get users config"})
		}

		settings, err := db.GetGlobalSettings()
		if err != nil {
			log.Error("Cannot get global settings: ", err)
			return c.JSON(http.StatusInternalServerError, jsonHTTPResponse{false, "Cannot get global settings"})
		}

		// Write config file
		err = util.WriteWireGuardServerConfig(tmplDir, server, clients, users, settings)
		if err != nil {
			log.Error("Cannot apply server config: ", err)
			return c.JSON(http.StatusInternalServerError, jsonHTTPResponse{
				false, fmt.Sprintf("Cannot apply server config: %v", err),
			})
		}

		// Get interface name from config file path
		interfaceName := "wg0"
		if settings.ConfigFilePath != "" {
			parts := strings.Split(settings.ConfigFilePath, "/")
			if len(parts) > 0 {
				baseName := parts[len(parts)-1]
				interfaceName = strings.TrimSuffix(baseName, ".conf")
			}
		}

		syncCmd := exec.Command("sudo", "wg", "syncconf", interfaceName, settings.ConfigFilePath)
		syncOutput, syncErr := syncCmd.CombinedOutput()
		if syncErr != nil {
			log.Errorf("wg syncconf failed: %v, output: %s. Falling back to service restart", syncErr, string(syncOutput))

			// Restart WireGuard service as a fallback
			serviceName := fmt.Sprintf("wg-quick@%s", interfaceName)

			// Try different service names if the first one fails
			serviceNames := []string{
				serviceName,
				"wg-quick@" + interfaceName,
				"wireguard@" + interfaceName,
				"wg-quick",
				"wireguard",
			}

			var restartSuccess bool
			var lastError error
			var lastOutput string

			for _, svcName := range serviceNames {
				cmd := exec.Command("sudo", "systemctl", "restart", svcName)
				output, err := cmd.CombinedOutput()
				if err == nil {
					// Check if service is active
					checkCmd := exec.Command("sudo", "systemctl", "is-active", svcName)
					status, err := checkCmd.CombinedOutput()
					if err == nil && strings.TrimSpace(string(status)) == "active" {
						restartSuccess = true
						break
					}
				}
				lastError = err
				lastOutput = string(output)
			}

			if !restartSuccess {
				log.Error("Cannot restart WireGuard service: ", lastError, ", Output: ", lastOutput)
				return c.JSON(http.StatusInternalServerError, jsonHTTPResponse{
					false, fmt.Sprintf("Cannot restart WireGuard service: %v. Please check if WireGuard is installed and running.", lastError),
				})
			}
		}
		err = util.UpdateHashes(db)
		if err != nil {
			log.Error("Cannot update hashes: ", err)
			return c.JSON(http.StatusInternalServerError, jsonHTTPResponse{
				false, fmt.Sprintf("Cannot update hashes: %v", err),
			})
		}

		return c.JSON(http.StatusOK, jsonHTTPResponse{true, "Applied server config successfully"})
	}
}

// GetHashesChanges handler returns if database hashes have changed
func GetHashesChanges(db store.IStore) echo.HandlerFunc {
	return func(c echo.Context) error {
		if util.HashesChanged(db) {
			return c.JSON(http.StatusOK, jsonHTTPResponse{true, "Hashes changed"})
		} else {
			return c.JSON(http.StatusOK, jsonHTTPResponse{false, "Hashes not changed"})
		}
	}
}

// AboutPage handler
func AboutPage() echo.HandlerFunc {
	return func(c echo.Context) error {
		return c.Render(http.StatusOK, "about.html", map[string]interface{}{
			"baseData": model.BaseData{Active: "about", CurrentUser: currentUser(c), Admin: isAdmin(c)},
		})
	}
}

func init() {
	internalRoutes = make([]Route, 0)
	// اضافه کردن روت داخلی برای غیرفعال‌سازی خودکار
	internalRoutes = append(internalRoutes, Route{
		Method:     "POST",
		Path:       "/internal/client/:id/status/:status",
		Handler:    SetClientStatus,
		Middleware: []echo.MiddlewareFunc{InternalOnly},
	})
}

// InternalOnly middleware to ensure request is from localhost
func InternalOnly(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		if c.Request().RemoteAddr != "127.0.0.1" && c.Request().RemoteAddr != "::1" {
			return c.JSON(http.StatusForbidden, jsonHTTPResponse{false, "Internal endpoints can only be accessed from localhost"})
		}
		return next(c)
	}
}

// GetInternalRoutes returns the list of internal routes
func GetInternalRoutes() []Route {
	return internalRoutes
}
