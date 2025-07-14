package router

import (
	"github.com/labstack/echo/v4"

	"github.com/MmadF14/vwireguard/handler"
	"github.com/MmadF14/vwireguard/store"
)

func RegisterUtilsRoutes(g *echo.Group, db store.IStore) {
	g.POST("/parse_v2link", handler.ParseV2Link(), handler.ValidSession, handler.ContentTypeJson)
}
