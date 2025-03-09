package handler

import (
	"path/filepath"
	"strings"

	"github.com/labstack/echo/v4"
)

// StaticHandler handles serving static files with proper MIME types
func StaticHandler(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		path := c.Request().URL.Path

		// Handle static files
		if strings.HasPrefix(path, "/static/") {
			// Set proper MIME types
			ext := filepath.Ext(path)
			var contentType string
			switch ext {
			case ".css":
				contentType = "text/css"
			case ".js":
				contentType = "application/javascript"
			case ".png":
				contentType = "image/png"
			case ".jpg", ".jpeg":
				contentType = "image/jpeg"
			case ".gif":
				contentType = "image/gif"
			case ".svg":
				contentType = "image/svg+xml"
			case ".woff":
				contentType = "application/font-woff"
			case ".woff2":
				contentType = "application/font-woff2"
			case ".ttf":
				contentType = "application/x-font-ttf"
			case ".eot":
				contentType = "application/vnd.ms-fontobject"
			default:
				contentType = "application/octet-stream"
			}

			c.Response().Header().Set("Content-Type", contentType)
		}

		return next(c)
	}
}
