package router

import (
	"github.com/labstack/echo/v4"

	"github.com/MmadF14/vwireguard/handler"
	"github.com/MmadF14/vwireguard/store"
)

func RegisterTunnelRoutes(g *echo.Group, db store.IStore) {
	g.GET("", handler.GetTunnels(db), handler.ValidSession)
	g.GET("/:id", handler.GetTunnel(db), handler.ValidSession)
	g.POST("/v2ray", handler.CreateV2rayTunnel(db), handler.ValidSession, handler.ContentTypeJson)
	g.PUT("/v2ray/:id", handler.UpdateV2rayTunnel(db), handler.ValidSession, handler.ContentTypeJson)
	g.POST("/:id/enable", handler.EnableTunnel(db), handler.ValidSession, handler.ContentTypeJson)
	g.POST("/:id/disable", handler.DisableTunnel(db), handler.ValidSession, handler.ContentTypeJson)
}
