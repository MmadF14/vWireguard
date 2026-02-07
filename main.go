package main

import (
	"crypto/sha512"
	"embed"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/gommon/log"

	"github.com/MmadF14/vwireguard/emailer"
	"github.com/MmadF14/vwireguard/handler"
	"github.com/MmadF14/vwireguard/router"
	"github.com/MmadF14/vwireguard/store"
	"github.com/MmadF14/vwireguard/store/jsondb"
	"github.com/MmadF14/vwireguard/telegram"
	"github.com/MmadF14/vwireguard/util"
)

var (
	appVersion = "development"
	gitCommit  = "N/A"
	gitRef     = "N/A"
	buildTime  = fmt.Sprintf(time.Now().UTC().Format("01-02-2006 15:04:05"))

	disableLogin             = false
	bindAddress              = "0.0.0.0:5000"
	smtpHostname             = "127.0.0.1"
	smtpPort                 = 25
	smtpUsername             string
	smtpPassword             string
	smtpAuthType             = "NONE"
	smtpNoTLSCheck           = false
	smtpEncryption           = "STARTTLS"
	smtpHelo                 = "localhost"
	sendgridApiKey           string
	emailFrom                string
	emailFromName            = "vWireguard"
	telegramToken            string
	telegramAllowConfRequest = false
	telegramFloodWait        = 60
	sessionSecret            = util.RandomString(32)
	sessionMaxDuration       = 90
	sessionMaxAge            = 7
	wgConfTemplate           string
	basePath                 = "/"
	subnetRanges             string
)

const (
	defaultEmailSubject = "Your wireguard configuration"
	defaultEmailContent = `Hi,</br>
<p>In this email you can find your personal configuration for our wireguard server.</p>

<p>Best</p>
`
)

//go:embed templates/*
var embeddedTemplates embed.FS

//go:embed assets/*
var embeddedAssets embed.FS

func init() {
	flag.BoolVar(&disableLogin, "disable-login", util.LookupEnvOrBool("DISABLE_LOGIN", disableLogin), "Disable authentication on the app. This is potentially dangerous.")
	flag.StringVar(&bindAddress, "bind-address", util.LookupEnvOrString("BIND_ADDRESS", bindAddress), "Address:Port to which the app will be bound.")
	flag.StringVar(&smtpHostname, "smtp-hostname", util.LookupEnvOrString("SMTP_HOSTNAME", smtpHostname), "SMTP Hostname")
	flag.IntVar(&smtpPort, "smtp-port", util.LookupEnvOrInt("SMTP_PORT", smtpPort), "SMTP Port")
	flag.StringVar(&smtpHelo, "smtp-helo", util.LookupEnvOrString("SMTP_HELO", smtpHelo), "SMTP HELO Hostname")
	flag.StringVar(&smtpUsername, "smtp-username", util.LookupEnvOrString("SMTP_USERNAME", smtpUsername), "SMTP Username")
	flag.BoolVar(&smtpNoTLSCheck, "smtp-no-tls-check", util.LookupEnvOrBool("SMTP_NO_TLS_CHECK", smtpNoTLSCheck), "Disable TLS verification for SMTP. This is potentially dangerous.")
	flag.StringVar(&smtpEncryption, "smtp-encryption", util.LookupEnvOrString("SMTP_ENCRYPTION", smtpEncryption), "SMTP Encryption : NONE, SSL, SSLTLS, TLS or STARTTLS (by default)")
	flag.StringVar(&smtpAuthType, "smtp-auth-type", util.LookupEnvOrString("SMTP_AUTH_TYPE", smtpAuthType), "SMTP Auth Type : PLAIN, LOGIN or NONE.")
	flag.StringVar(&emailFrom, "email-from", util.LookupEnvOrString("EMAIL_FROM_ADDRESS", emailFrom), "'From' email address.")
	flag.StringVar(&emailFromName, "email-from-name", util.LookupEnvOrString("EMAIL_FROM_NAME", emailFromName), "'From' email name.")
	flag.StringVar(&telegramToken, "telegram-token", util.LookupEnvOrString("TELEGRAM_TOKEN", telegramToken), "Telegram bot token for distributing configs to clients.")
	flag.BoolVar(&telegramAllowConfRequest, "telegram-allow-conf-request", util.LookupEnvOrBool("TELEGRAM_ALLOW_CONF_REQUEST", telegramAllowConfRequest), "Allow users to get configs from the bot by sending a message.")
	flag.IntVar(&telegramFloodWait, "telegram-flood-wait", util.LookupEnvOrInt("TELEGRAM_FLOOD_WAIT", telegramFloodWait), "Time in minutes before the next conf request is processed.")
	flag.StringVar(&wgConfTemplate, "wg-conf-template", util.LookupEnvOrString("WG_CONF_TEMPLATE", wgConfTemplate), "Path to custom wg.conf template.")
	flag.StringVar(&basePath, "base-path", util.LookupEnvOrString("BASE_PATH", basePath), "The base path of the URL")
	flag.StringVar(&subnetRanges, "subnet-ranges", util.LookupEnvOrString("SUBNET_RANGES", subnetRanges), "IP ranges to choose from when assigning an IP for a client.")
	flag.IntVar(&sessionMaxDuration, "session-max-duration", util.LookupEnvOrInt("SESSION_MAX_DURATION", sessionMaxDuration), "Max time in days a remembered session is refreshed and valid.")
	flag.IntVar(&sessionMaxAge, "session-max-age", util.LookupEnvOrInt(util.SessionMaxAgeEnvVar, sessionMaxAge), "Duration in days for 'remember me' sessions.")

	var (
		smtpPasswordLookup   = util.LookupEnvOrString("SMTP_PASSWORD", smtpPassword)
		sendgridApiKeyLookup = util.LookupEnvOrString("SENDGRID_API_KEY", sendgridApiKey)
		sessionSecretLookup  = util.LookupEnvOrString("SESSION_SECRET", sessionSecret)
	)

	if smtpPasswordLookup != "" {
		flag.StringVar(&smtpPassword, "smtp-password", smtpPasswordLookup, "SMTP Password")
	} else {
		flag.StringVar(&smtpPassword, "smtp-password", util.LookupEnvOrFile("SMTP_PASSWORD_FILE", smtpPassword), "SMTP Password File")
	}

	if sendgridApiKeyLookup != "" {
		flag.StringVar(&sendgridApiKey, "sendgrid-api-key", sendgridApiKeyLookup, "Your sendgrid api key.")
	} else {
		flag.StringVar(&sendgridApiKey, "sendgrid-api-key", util.LookupEnvOrFile("SENDGRID_API_KEY_FILE", sendgridApiKey), "File containing your sendgrid api key.")
	}

	if sessionSecretLookup != "" {
		flag.StringVar(&sessionSecret, "session-secret", sessionSecretLookup, "The key used to encrypt session cookies.")
	} else {
		flag.StringVar(&sessionSecret, "session-secret", util.LookupEnvOrFile("SESSION_SECRET_FILE", sessionSecret), "File containing the key used to encrypt session cookies.")
	}

	flag.Parse()

	util.DisableLogin = disableLogin
	util.BindAddress = bindAddress
	util.SmtpHostname = smtpHostname
	util.SmtpPort = smtpPort
	util.SmtpHelo = smtpHelo
	util.SmtpUsername = smtpUsername
	util.SmtpPassword = smtpPassword
	util.SmtpAuthType = smtpAuthType
	util.SmtpNoTLSCheck = smtpNoTLSCheck
	util.SmtpEncryption = smtpEncryption
	util.SendgridApiKey = sendgridApiKey
	util.EmailFrom = emailFrom
	util.EmailFromName = emailFromName
	util.SessionSecret = sha512.Sum512([]byte(sessionSecret))
	util.SessionMaxDuration = int64(sessionMaxDuration) * 86_400
	util.SessionMaxAge = sessionMaxAge * 86_400
	util.WgConfTemplate = wgConfTemplate
	util.BasePath = util.ParseBasePath(basePath)
	util.SubnetRanges = util.ParseSubnetRanges(subnetRanges)

	lvl, _ := util.ParseLogLevel(util.LookupEnvOrString(util.LogLevel, "INFO"))

	telegram.Token = telegramToken
	telegram.AllowConfRequest = telegramAllowConfRequest
	telegram.FloodWait = telegramFloodWait
	telegram.LogLevel = lvl

	if lvl <= log.INFO {
		fmt.Println("vWireguard")
		fmt.Println("App Version\t:", appVersion)
		fmt.Println("Git Commit\t:", gitCommit)
		fmt.Println("Git Ref\t\t:", gitRef)
		fmt.Println("Build Time\t:", buildTime)
		fmt.Println("Git Repo\t:", "https://github.com/MmadF14/vWireguard")
		fmt.Println("Authentication\t:", !util.DisableLogin)
		fmt.Println("Bind address\t:", util.BindAddress)
		fmt.Println("Email from\t:", util.EmailFrom)
		fmt.Println("Email from name\t:", util.EmailFromName)
		fmt.Println("Custom wg.conf\t:", util.WgConfTemplate)
		fmt.Println("Base path\t:", util.BasePath+"/")
		fmt.Println("Subnet ranges\t:", util.GetSubnetRangesString())
	}
}

func main() {
	db, err := jsondb.New("./db")
	if err != nil {
		panic(err)
	}
	if err := db.Init(); err != nil {
		panic(err)
	}
	extraData := make(map[string]interface{})
	extraData["appVersion"] = appVersion
	extraData["gitCommit"] = gitCommit
	extraData["basePath"] = util.BasePath
	extraData["loginDisabled"] = disableLogin

	tmplDir, _ := fs.Sub(fs.FS(embeddedTemplates), "templates")

	assetsDir, _ := fs.Sub(fs.FS(embeddedAssets), "assets")

	handler.StartQuotaChecker(db, tmplDir)

	initServerConfig(db, tmplDir)

	if err := util.ValidateAndFixSubnetRanges(db); err != nil {
		panic(err)
	}

	if lvl, _ := util.ParseLogLevel(util.LookupEnvOrString(util.LogLevel, "INFO")); lvl <= log.INFO {
		fmt.Println("Valid subnet ranges:", util.GetSubnetRangesString())
	}

	app := router.New(tmplDir, extraData, util.SessionSecret)

	app.Static(util.BasePath+"/assets", "assets")
	app.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if strings.HasPrefix(c.Path(), util.BasePath+"/assets/") {
				if strings.HasSuffix(c.Path(), ".js") {
					c.Response().Header().Set(echo.HeaderContentType, "application/javascript")
				}
				path := strings.TrimPrefix(c.Path(), util.BasePath+"/assets/")
				file, err := assetsDir.Open(path)
				if err != nil {
					return echo.NotFoundHandler(c)
				}
				defer file.Close()

				content, err := io.ReadAll(file)
				if err != nil {
					return err
				}

				return c.Blob(http.StatusOK, c.Response().Header().Get(echo.HeaderContentType), content)
			}
			return next(c)
		}
	})

	app.GET(util.BasePath, handler.WireGuardClients(db), handler.ValidSession, handler.RefreshSession)

	if !util.DisableLogin {
		app.GET(util.BasePath+"/login", handler.LoginPage())
		app.POST(util.BasePath+"/login", handler.Login(db), handler.ContentTypeJson)
		app.GET(util.BasePath+"/logout", handler.Logout(), handler.ValidSession)
		app.GET(util.BasePath+"/profile", handler.LoadProfile(db), handler.ValidSession, handler.RefreshSession)
		app.GET(util.BasePath+"/users-settings", handler.UsersSettings(db), handler.ValidSession, handler.RefreshSession, handler.NeedsAdmin)
		app.POST(util.BasePath+"/update-user", handler.UpdateUser(db), handler.ValidSession, handler.ContentTypeJson)
		app.POST(util.BasePath+"/create-user", handler.CreateUser(db), handler.ValidSession, handler.ContentTypeJson, handler.NeedsAdmin)
		app.POST(util.BasePath+"/remove-user", handler.RemoveUser(db), handler.ValidSession, handler.ContentTypeJson, handler.NeedsAdmin)
		app.GET(util.BasePath+"/get-users", handler.GetUsers(db), handler.ValidSession, handler.NeedsAdmin)
		app.GET(util.BasePath+"/api/user/:username", handler.GetUser(db), handler.ValidSession)
	}

	var sendmail emailer.Emailer
	if util.SendgridApiKey != "" {
		sendmail = emailer.NewSendgridApiMail(util.SendgridApiKey, util.EmailFromName, util.EmailFrom)
	} else {
		sendmail = emailer.NewSmtpMail(util.SmtpHostname, util.SmtpPort, util.SmtpUsername, util.SmtpPassword, util.SmtpHelo, util.SmtpNoTLSCheck, util.SmtpAuthType, util.EmailFromName, util.EmailFrom, util.SmtpEncryption)
	}

	app.GET(util.BasePath+"/test-hash", handler.GetHashesChanges(db), handler.ValidSession)
	app.GET(util.BasePath+"/about", handler.AboutPage())
	app.GET(util.BasePath+"/utilities", handler.UtilitiesPage(db), handler.ValidSession, handler.RefreshSession, handler.NeedsAdmin)

	app.POST(util.BasePath+"/api/utilities/restart-service", handler.RestartWireGuardService(db), handler.ValidSession, handler.ContentTypeJson, handler.NeedsAdmin)
	app.POST(util.BasePath+"/api/utilities/flush-dns", handler.FlushDNSCache(db), handler.ValidSession, handler.ContentTypeJson, handler.NeedsAdmin)
	app.POST(util.BasePath+"/api/utilities/check-updates", handler.CheckForUpdates(db), handler.ValidSession, handler.ContentTypeJson, handler.NeedsAdmin)
	app.POST(util.BasePath+"/api/utilities/generate-report", handler.GenerateSystemReport(db), handler.ValidSession, handler.ContentTypeJson, handler.NeedsAdmin)
	app.GET(util.BasePath+"/api/utilities/logs", handler.GetSystemLogs(db), handler.ValidSession, handler.NeedsAdmin)
	app.POST(util.BasePath+"/api/utilities/clear-logs", handler.ClearSystemLogs(db), handler.ValidSession, handler.ContentTypeJson, handler.NeedsAdmin)

	app.GET(util.BasePath+"/_health", handler.Health())
	app.GET(util.BasePath+"/favicon", handler.Favicon())
	app.POST(util.BasePath+"/new-client", handler.NewClient(db), handler.ValidSession, handler.ContentTypeJson)
	app.POST(util.BasePath+"/update-client", handler.UpdateClient(db), handler.ValidSession, handler.ContentTypeJson)
	app.POST(util.BasePath+"/email-client", handler.EmailClient(db, sendmail, defaultEmailSubject, defaultEmailContent), handler.ValidSession, handler.ContentTypeJson)
	app.POST(util.BasePath+"/send-telegram-client", handler.SendTelegramClient(db), handler.ValidSession, handler.ContentTypeJson)
	app.POST(util.BasePath+"/client/set-status", handler.SetClientStatus(db), handler.ValidSession, handler.ContentTypeJson)
	app.GET(util.BasePath+"/api/client/:id/status/:status", handler.SetClientStatus(db), handler.ValidSession)
	app.POST(util.BasePath+"/api/client/:id/status/:status", handler.SetClientStatus(db), handler.ValidSession)
	app.POST(util.BasePath+"/remove-client", handler.RemoveClient(db), handler.ValidSession, handler.ContentTypeJson)
	app.POST(util.BasePath+"/api/client/:id/remove", handler.RemoveClient(db), handler.ValidSession, handler.ContentTypeJson)
	app.PUT(util.BasePath+"/api/client/:id", handler.UpdateClient(db), handler.ValidSession, handler.ContentTypeJson)
	app.GET(util.BasePath+"/download", handler.DownloadClient(db), handler.ValidSession)
	app.GET(util.BasePath+"/wg-server", handler.WireGuardServer(db), handler.ValidSession, handler.RefreshSession, handler.NeedsAdmin)
	app.POST(util.BasePath+"/wg-server/interfaces", handler.WireGuardServerInterfaces(db), handler.ValidSession, handler.ContentTypeJson, handler.NeedsAdmin)
	app.POST(util.BasePath+"/wg-server/keypair", handler.WireGuardServerKeyPair(db), handler.ValidSession, handler.ContentTypeJson, handler.NeedsAdmin)
	app.GET(util.BasePath+"/global-settings", handler.GlobalSettings(db), handler.ValidSession, handler.RefreshSession, handler.NeedsAdmin)
	app.POST(util.BasePath+"/global-settings", handler.GlobalSettingSubmit(db), handler.ValidSession, handler.ContentTypeJson, handler.NeedsAdmin)
	app.POST(util.BasePath+"/display-settings", handler.DisplaySettingsSubmit(db), handler.ValidSession, handler.ContentTypeJson, handler.NeedsAdmin)
	app.GET(util.BasePath+"/status", handler.Status(db), handler.ValidSession, handler.RefreshSession)
	app.GET(util.BasePath+"/api/status-data", handler.StatusData(db), handler.ValidSession)
	app.GET(util.BasePath+"/api/clients", handler.GetClients(db), handler.ValidSession)
	app.GET(util.BasePath+"/api/client/:id", handler.GetClient(db), handler.ValidSession)
	app.GET(util.BasePath+"/api/client/:id/qr", handler.GetClientQRCode(db), handler.ValidSession)
	app.GET(util.BasePath+"/api/machine-ips", handler.MachineIPAddresses(), handler.ValidSession)
	app.GET(util.BasePath+"/api/subnet-ranges", handler.GetOrderedSubnetRanges(), handler.ValidSession)
	app.GET(util.BasePath+"/api/suggest-client-ips", handler.SuggestIPAllocation(db), handler.ValidSession)
	app.POST(util.BasePath+"/api/apply-wg-config", handler.ApplyServerConfig(db, tmplDir), handler.ValidSession, handler.ContentTypeJson)
	app.GET(util.BasePath+"/wake_on_lan_hosts", handler.GetWakeOnLanHosts(db), handler.ValidSession, handler.RefreshSession)
	app.POST(util.BasePath+"/wake_on_lan_host", handler.SaveWakeOnLanHost(db), handler.ValidSession, handler.ContentTypeJson)
	app.DELETE(util.BasePath+"/wake_on_lan_host/:mac_address", handler.DeleteWakeOnHost(db), handler.ValidSession, handler.ContentTypeJson)
	app.PUT(util.BasePath+"/wake_on_lan_host/:mac_address", handler.WakeOnHost(db), handler.ValidSession, handler.ContentTypeJson)
	app.POST(util.BasePath+"/api/terminate-client", handler.TerminateClient(db, tmplDir), handler.ValidSession, handler.ContentTypeJson)

	utilsGroup := app.Group(util.BasePath + "/api/utils")
	router.RegisterUtilsRoutes(utilsGroup, db)

	for _, route := range handler.GetInternalRoutes() {
		app.Add(route.Method, route.Path, route.Handler(db), route.Middleware...)
	}

	apiGroup := app.Group(util.BasePath + "/api/v1")
	apiGroup.POST("/login", handler.APILogin(db))
	apiGroup.POST("/connect", handler.APIConnect(db))
	apiGroup.POST("/status", handler.APIStatus(db))
	apiGroup.POST("/app/user-info", handler.APIAppUserInfo(db))

	apiGroup.POST("/admin/create-client", handler.APIAdminCreateClient(db))
	apiGroup.POST("/admin/update-client", handler.APIAdminUpdateClient(db))

	app.GET(util.BasePath+"/health", handler.Health())
	app.GET(util.BasePath+"/favicon.ico", handler.Favicon())

	app.GET(util.BasePath+"/system-monitor", handler.SystemMonitorPage(), handler.ValidSession, handler.RefreshSession, handler.NeedsAdmin)
	app.GET(util.BasePath+"/api/system-metrics", handler.GetSystemMetrics(), handler.ValidSession, handler.RefreshSession, handler.NeedsAdmin)
	app.GET(util.BasePath+"/api/backup", handler.BackupSystem(), handler.ValidSession, handler.RefreshSession, handler.NeedsAdmin)
	app.POST(util.BasePath+"/api/restore", handler.RestoreSystem(db), handler.ValidSession, handler.RefreshSession, handler.NeedsAdmin)

	app.Start(util.BindAddress)
}

func initServerConfig(db store.IStore, tmplDir fs.FS) {
	settings, err := db.GetGlobalSettings()
	if err != nil {
		log.Fatalf("Cannot get global settings: %v", err)
	}

	if _, err := os.Stat(settings.ConfigFilePath); err == nil {
		return
	}

	server, err := db.GetServer()
	if err != nil {
		log.Fatalf("Cannot get server config: %v", err)
	}

	clients, err := db.GetClients(false)
	if err != nil {
		log.Fatalf("Cannot get client config: %v", err)
	}

	users, err := db.GetUsers()
	if err != nil {
		log.Fatalf("Cannot get user config: %v", err)
	}

	err = util.WriteWireGuardServerConfig(tmplDir, server, clients, users, settings)
	if err != nil {
		log.Fatalf("Cannot create server config: %v", err)
	}
}

func enableIPForwarding() {
	cmd := exec.Command("sysctl", "-w", "net.ipv4.ip_forward=1")
	if err := cmd.Run(); err != nil {
		log.Warnf("Failed to enable IPv4 forwarding (may require root): %v", err)
	} else {
		log.Info("IPv4 forwarding enabled")
	}

	cmd = exec.Command("sysctl", "-w", "net.ipv6.conf.all.forwarding=1")
	if err := cmd.Run(); err != nil {
		log.Warnf("Failed to enable IPv6 forwarding (may require root): %v", err)
	} else {
		log.Info("IPv6 forwarding enabled")
	}
}

func initTelegram(initDeps telegram.TgBotInitDependencies) {
	go func() {
		for {
			err := telegram.Start(initDeps)
			if err == nil {
				break
			}
		}
	}()
}
