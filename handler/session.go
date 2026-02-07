package handler

import (
	"fmt"
	"net/http"
	"time"

	"github.com/MmadF14/vwireguard/util"
	"github.com/gorilla/sessions"
	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/v4"
)

func ValidSession(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		if !isValidSession(c) {
			nextURL := c.Request().URL
			if nextURL != nil && c.Request().Method == http.MethodGet {
				return c.Redirect(http.StatusTemporaryRedirect, fmt.Sprintf(util.BasePath+"/login?next=%s", c.Request().URL))
			} else {
				return c.Redirect(http.StatusTemporaryRedirect, util.BasePath+"/login")
			}
		}
		return next(c)
	}
}

func RefreshSession(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		doRefreshSession(c)
		return next(c)
	}
}

func NeedsAdmin(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		if !isAdmin(c) {
			return c.Redirect(http.StatusTemporaryRedirect, util.BasePath+"/")
		}
		return next(c)
	}
}

func isValidSession(c echo.Context) bool {
	if util.DisableLogin {
		return true
	}
	sess, _ := session.Get("session", c)
	cookie, err := c.Cookie("session_token")
	if err != nil || sess.Values["session_token"] != cookie.Value {
		return false
	}

	createdAt := getCreatedAt(sess)
	updatedAt := getUpdatedAt(sess)
	maxAge := getMaxAge(sess)

	if maxAge == 0 {
		maxAge = 86400
	}
	expiration := updatedAt + int64(maxAge)
	now := time.Now().UTC().Unix()
	if updatedAt > now || expiration < now || createdAt+util.SessionMaxDuration < now {
		return false
	}

	username := fmt.Sprintf("%s", sess.Values["username"])
	userHash := getUserHash(sess)
	if uHash, ok := util.DBUsersToCRC32[username]; !ok || userHash != uHash {
		return false
	}

	return true
}

func doRefreshSession(c echo.Context) {
	if util.DisableLogin {
		return
	}

	sess, _ := session.Get("session", c)
	maxAge := getMaxAge(sess)
	if maxAge <= 0 {
		return
	}

	oldCookie, err := c.Cookie("session_token")
	if err != nil || sess.Values["session_token"] != oldCookie.Value {
		return
	}

	createdAt := getCreatedAt(sess)
	updatedAt := getUpdatedAt(sess)
	expiration := updatedAt + int64(getMaxAge(sess))
	now := time.Now().UTC().Unix()
	if updatedAt > now || expiration < now || now-updatedAt < 86_400 || createdAt+util.SessionMaxDuration < now {
		return
	}

	cookiePath := util.GetCookiePath()

	sess.Values["updated_at"] = now
	sess.Options = &sessions.Options{
		Path:     cookiePath,
		MaxAge:   maxAge,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	}
	sess.Save(c.Request(), c.Response())

	cookie := new(http.Cookie)
	cookie.Name = "session_token"
	cookie.Path = cookiePath
	cookie.Value = oldCookie.Value
	cookie.MaxAge = maxAge
	cookie.HttpOnly = true
	cookie.SameSite = http.SameSiteLaxMode
	c.SetCookie(cookie)
}

func getMaxAge(sess *sessions.Session) int {
	if util.DisableLogin {
		return 0
	}

	maxAge := sess.Values["max_age"]

	switch typedMaxAge := maxAge.(type) {
	case int:
		return typedMaxAge
	default:
		return 0
	}
}

func getCreatedAt(sess *sessions.Session) int64 {
	if util.DisableLogin {
		return 0
	}

	createdAt := sess.Values["created_at"]

	switch typedCreatedAt := createdAt.(type) {
	case int64:
		return typedCreatedAt
	default:
		return 0
	}
}

func getUpdatedAt(sess *sessions.Session) int64 {
	if util.DisableLogin {
		return 0
	}

	lastUpdate := sess.Values["updated_at"]

	switch typedLastUpdate := lastUpdate.(type) {
	case int64:
		return typedLastUpdate
	default:
		return 0
	}
}

func getUserHash(sess *sessions.Session) uint32 {
	if util.DisableLogin {
		return 0
	}

	userHash := sess.Values["user_hash"]

	switch typedUserHash := userHash.(type) {
	case uint32:
		return typedUserHash
	default:
		return 0
	}
}

func currentUser(c echo.Context) string {
	if util.DisableLogin {
		return ""
	}

	sess, _ := session.Get("session", c)
	username := fmt.Sprintf("%s", sess.Values["username"])
	return username
}

func isAdmin(c echo.Context) bool {
	if util.DisableLogin {
		return true
	}

	sess, _ := session.Get("session", c)
	admin := fmt.Sprintf("%t", sess.Values["admin"])
	return admin == "true"
}

func setUser(c echo.Context, username string, admin bool, userCRC32 uint32) {
	sess, _ := session.Get("session", c)
	sess.Values["username"] = username
	sess.Values["user_hash"] = userCRC32
	sess.Values["admin"] = admin
	sess.Save(c.Request(), c.Response())
}

func clearSession(c echo.Context) {
	sess, _ := session.Get("session", c)
	sess.Values["username"] = ""
	sess.Values["user_hash"] = 0
	sess.Values["admin"] = false
	sess.Values["session_token"] = ""
	sess.Values["max_age"] = -1
	sess.Options.MaxAge = -1
	sess.Save(c.Request(), c.Response())

	cookiePath := util.GetCookiePath()

	cookie, err := c.Cookie("session_token")
	if err != nil {
		cookie = new(http.Cookie)
	}

	cookie.Name = "session_token"
	cookie.Path = cookiePath
	cookie.MaxAge = -1
	cookie.HttpOnly = true
	cookie.SameSite = http.SameSiteLaxMode
	c.SetCookie(cookie)
}
