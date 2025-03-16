package handler

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

// UtilitiesPage handles the utilities page request
func UtilitiesPage(c echo.Context) error {
	baseData := GetBaseData(c)
	baseData.Active = "utilities"

	data := map[string]interface{}{
		"baseData": baseData,
	}

	return c.Render(http.StatusOK, "utilities.html", data)
} 