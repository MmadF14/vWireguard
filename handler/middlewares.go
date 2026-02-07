package handler

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

func ContentTypeJson(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		contentType := c.Request().Header.Get("Content-Type")
		if contentType != "application/json" {
			return c.JSON(http.StatusBadRequest, jsonHTTPResponse{false, "Only JSON allowed"})
		}

		return next(c)
	}
}
