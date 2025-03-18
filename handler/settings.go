package handler

import (
    "net/http"
    "github.com/labstack/echo/v4"
    "github.com/MmadF14/vwireguard/store"
)

type linkRequest struct {
    Link string `json:"link"`
}

// SaveGitHubLink saves the GitHub repository link
func SaveGitHubLink(db store.IStore) echo.HandlerFunc {
    return func(c echo.Context) error {
        var req linkRequest
        if err := c.Bind(&req); err != nil {
            return c.JSON(http.StatusBadRequest, jsonHTTPResponse{false, "Invalid request"})
        }

        settings, err := db.GetGlobalSettings()
        if err != nil {
            return c.JSON(http.StatusInternalServerError, jsonHTTPResponse{false, "Cannot get settings"})
        }

        settings.GitHubLink = req.Link
        if err := db.SaveGlobalSettings(settings); err != nil {
            return c.JSON(http.StatusInternalServerError, jsonHTTPResponse{false, "Cannot save settings"})
        }

        return c.JSON(http.StatusOK, jsonHTTPResponse{true, "GitHub link saved successfully"})
    }
}

// SaveTelegramLink saves the Telegram channel link
func SaveTelegramLink(db store.IStore) echo.HandlerFunc {
    return func(c echo.Context) error {
        var req linkRequest
        if err := c.Bind(&req); err != nil {
            return c.JSON(http.StatusBadRequest, jsonHTTPResponse{false, "Invalid request"})
        }

        settings, err := db.GetGlobalSettings()
        if err != nil {
            return c.JSON(http.StatusInternalServerError, jsonHTTPResponse{false, "Cannot get settings"})
        }

        settings.TelegramLink = req.Link
        if err := db.SaveGlobalSettings(settings); err != nil {
            return c.JSON(http.StatusInternalServerError, jsonHTTPResponse{false, "Cannot save settings"})
        }

        return c.JSON(http.StatusOK, jsonHTTPResponse{true, "Telegram link saved successfully"})
    }
} 