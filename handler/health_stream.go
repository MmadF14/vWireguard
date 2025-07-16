package handler

import (
	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"

	"github.com/MmadF14/vwireguard/monitor"
)

var upgrader = websocket.Upgrader{}

func TunnelHealthStream() echo.HandlerFunc {
	return func(c echo.Context) error {
		conn, err := upgrader.Upgrade(c.Response(), c.Request(), nil)
		if err != nil {
			return err
		}
		ch := monitor.Subscribe()
		defer monitor.Unsubscribe(ch)
		for {
			select {
			case up := <-ch:
				if err := conn.WriteJSON(up); err != nil {
					return nil
				}
			default:
				if _, _, err := conn.NextReader(); err != nil {
					return nil
				}
			}
		}
	}
}
