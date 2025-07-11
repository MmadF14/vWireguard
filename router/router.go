package router

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"reflect"
	"strings"
	"text/template"

	"github.com/MmadF14/vwireguard/handler"
	"github.com/MmadF14/vwireguard/util"
	"github.com/gorilla/sessions"
	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/labstack/gommon/log"
)

// TemplateRegistry is a custom html/template renderer for Echo framework
type TemplateRegistry struct {
	templates map[string]*template.Template
	extraData map[string]interface{}
}

// Render e.Renderer interface
func (t *TemplateRegistry) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	tmpl, ok := t.templates[name]
	if !ok {
		err := errors.New("Template not found -> " + name)
		return err
	}

	// inject more app data information. E.g. appVersion
	if reflect.TypeOf(data).Kind() == reflect.Map {
		for k, v := range t.extraData {
			data.(map[string]interface{})[k] = v
		}

		data.(map[string]interface{})["client_defaults"] = util.ClientDefaultsFromEnv()
	}

	// login page does not need the base layout
	if name == "login.html" {
		return tmpl.Execute(w, data)
	}

	return tmpl.ExecuteTemplate(w, "base.html", data)
}

// formatBytes formats bytes into human readable format
func formatBytes(bytes int64) string {
	if bytes == 0 {
		return "0 B"
	}
	const unit = 1024
	sizes := []string{"B", "KB", "MB", "GB", "TB"}

	var i int
	absBytes := bytes
	if absBytes < 0 {
		absBytes = -absBytes
	}

	for absBytes >= unit && i < len(sizes)-1 {
		absBytes /= unit
		i++
	}

	divisor := int64(1)
	for j := 0; j < i; j++ {
		divisor *= unit
	}

	result := float64(bytes) / float64(divisor)
	if i == 0 {
		return fmt.Sprintf("%.0f %s", result, sizes[i])
	}
	return fmt.Sprintf("%.2f %s", result, sizes[i])
}

// New function
func New(tmplDir fs.FS, extraData map[string]interface{}, secret [64]byte) *echo.Echo {
	e := echo.New()

	cookiePath := util.GetCookiePath()

	cookieStore := sessions.NewCookieStore(secret[:32], secret[32:])
	cookieStore.Options.Path = cookiePath
	cookieStore.Options.HttpOnly = true
	cookieStore.MaxAge(util.SessionMaxAge)

	e.Use(session.Middleware(cookieStore))

	// read html template file to string
	tmplBaseString, err := util.StringFromEmbedFile(tmplDir, "base.html")
	if err != nil {
		log.Fatal(err)
	}

	tmplLoginString, err := util.StringFromEmbedFile(tmplDir, "login.html")
	if err != nil {
		log.Fatal(err)
	}

	tmplProfileString, err := util.StringFromEmbedFile(tmplDir, "profile.html")
	if err != nil {
		log.Fatal(err)
	}

	tmplClientsString, err := util.StringFromEmbedFile(tmplDir, "clients.html")
	if err != nil {
		log.Fatal(err)
	}

	tmplServerString, err := util.StringFromEmbedFile(tmplDir, "server.html")
	if err != nil {
		log.Fatal(err)
	}

	tmplGlobalSettingsString, err := util.StringFromEmbedFile(tmplDir, "global_settings.html")
	if err != nil {
		log.Fatal(err)
	}

	tmplUsersSettingsString, err := util.StringFromEmbedFile(tmplDir, "users_settings.html")
	if err != nil {
		log.Fatal(err)
	}

	tmplStatusString, err := util.StringFromEmbedFile(tmplDir, "status.html")
	if err != nil {
		log.Fatal(err)
	}

	tmplWakeOnLanHostsString, err := util.StringFromEmbedFile(tmplDir, "wake_on_lan_hosts.html")
	if err != nil {
		log.Fatal(err)
	}

	aboutPageString, err := util.StringFromEmbedFile(tmplDir, "about.html")
	if err != nil {
		log.Fatal(err)
	}

	tmplSystemMonitorString, err := util.StringFromEmbedFile(tmplDir, "system_monitor.html")
	if err != nil {
		log.Fatal(err)
	}

	tmplUtilitiesString, err := util.StringFromEmbedFile(tmplDir, "utilities.html")
	if err != nil {
		log.Fatal(err)
	}

	tmplTunnelsString, err := util.StringFromEmbedFile(tmplDir, "tunnels.html")
	if err != nil {
		log.Fatal(err)
	}

	// create template list
	funcs := template.FuncMap{
		"StringsJoin": strings.Join,
		"formatBytes": formatBytes,
		"substr": func(s string, start, length int) string {
			if start < 0 {
				start = 0
			}
			if start >= len(s) {
				return ""
			}
			end := start + length
			if end > len(s) {
				end = len(s)
			}
			return s[start:end]
		},
		"upper": strings.ToUpper,
		"len": func(v interface{}) int {
			switch v := v.(type) {
			case string:
				return len(v)
			case []interface{}:
				return len(v)
			case map[string]interface{}:
				return len(v)
			default:
				return 0
			}
		},
		"eq": func(a, b interface{}) bool {
			return a == b
		},
		"not": func(v interface{}) bool {
			switch v := v.(type) {
			case bool:
				return !v
			case nil:
				return true
			default:
				return false
			}
		},
	}
	templates := make(map[string]*template.Template)
	templates["login.html"] = template.Must(template.New("login").Funcs(funcs).Parse(tmplLoginString))
	templates["profile.html"] = template.Must(template.New("profile").Funcs(funcs).Parse(tmplBaseString + tmplProfileString))
	templates["clients.html"] = template.Must(template.New("clients").Funcs(funcs).Parse(tmplBaseString + tmplClientsString))
	templates["server.html"] = template.Must(template.New("server").Funcs(funcs).Parse(tmplBaseString + tmplServerString))
	templates["global_settings.html"] = template.Must(template.New("global_settings").Funcs(funcs).Parse(tmplBaseString + tmplGlobalSettingsString))
	templates["users_settings.html"] = template.Must(template.New("users_settings").Funcs(funcs).Parse(tmplBaseString + tmplUsersSettingsString))
	templates["status.html"] = template.Must(template.New("status").Funcs(funcs).Parse(tmplBaseString + tmplStatusString))
	templates["wake_on_lan_hosts.html"] = template.Must(template.New("wake_on_lan_hosts").Funcs(funcs).Parse(tmplBaseString + tmplWakeOnLanHostsString))
	templates["about.html"] = template.Must(template.New("about").Funcs(funcs).Parse(tmplBaseString + aboutPageString))
	templates["system_monitor.html"] = template.Must(template.New("system_monitor").Funcs(funcs).Parse(tmplBaseString + tmplSystemMonitorString))
	templates["utilities.html"] = template.Must(template.New("utilities").Funcs(funcs).Parse(tmplBaseString + tmplUtilitiesString))
	templates["tunnels.html"] = template.Must(template.New("tunnels").Funcs(funcs).Parse(tmplBaseString + tmplTunnelsString))

	lvl, err := util.ParseLogLevel(util.LookupEnvOrString(util.LogLevel, "INFO"))
	if err != nil {
		log.Fatal(err)
	}
	logConfig := middleware.DefaultLoggerConfig
	logConfig.Skipper = func(c echo.Context) bool {
		resp := c.Response()
		if resp.Status >= 500 && lvl > log.ERROR { // do not log if response is 5XX but log level is higher than ERROR
			return true
		} else if resp.Status >= 400 && lvl > log.WARN { // do not log if response is 4XX but log level is higher than WARN
			return true
		} else if lvl > log.DEBUG { // do not log if log level is higher than DEBUG
			return true
		}
		return false
	}

	e.Logger.SetLevel(lvl)
	e.Pre(middleware.RemoveTrailingSlash())
	e.Use(middleware.LoggerWithConfig(logConfig))
	e.HideBanner = true
	e.HidePort = lvl > log.INFO // hide the port output if the log level is higher than INFO
	e.Validator = NewValidator()
	e.Renderer = &TemplateRegistry{
		templates: templates,
		extraData: extraData,
	}

	// Middleware
	e.Use(middleware.Recover())
	e.Use(handler.StaticHandler)

	// Static files
	e.Static("/static", "static")

	return e
}
