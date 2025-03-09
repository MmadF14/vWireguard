package handler

import (
	"mime"
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
			ext := filepath.Ext(path)
			var contentType string

			// First try to get MIME type from standard library
			contentType = mime.TypeByExtension(ext)

			// If not found, use our custom mapping
			if contentType == "" {
				switch ext {
				case ".css":
					contentType = "text/css; charset=utf-8"
				case ".js":
					contentType = "application/javascript; charset=utf-8"
				case ".json":
					contentType = "application/json; charset=utf-8"
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
					contentType = "font/woff2"
				case ".ttf":
					contentType = "application/x-font-ttf"
				case ".eot":
					contentType = "application/vnd.ms-fontobject"
				case ".ico":
					contentType = "image/x-icon"
				default:
					contentType = "application/octet-stream"
				}
			}

			c.Response().Header().Set(echo.HeaderContentType, contentType)
		}

		return next(c)
	}
}
