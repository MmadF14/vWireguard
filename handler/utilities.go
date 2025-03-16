package handler

import (
	"net/http"

	"github.com/MmadF14/vwireguard/model"
	"github.com/MmadF14/vwireguard/store"
	"github.com/labstack/echo/v4"
)

// UtilitiesPage handles the utilities page request
func UtilitiesPage(db store.IStore) echo.HandlerFunc {
	return func(c echo.Context) error {
		user, _ := db.GetUserByName(currentUser(c))
		data := map[string]interface{}{
			"baseData": model.BaseData{
				Active:      "utilities",
				CurrentUser: currentUser(c),
				Admin:      user.Role == model.RoleAdmin,
			},
		}

		return c.Render(http.StatusOK, "utilities.html", data)
	}
} 