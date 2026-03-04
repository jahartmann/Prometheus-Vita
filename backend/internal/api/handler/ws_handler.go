package handler

import (
	"log/slog"
	"net/http"
	"strings"

	"github.com/antigravity/prometheus/internal/service/auth"
	"github.com/antigravity/prometheus/internal/service/monitor"
	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type WSHandler struct {
	hub    *monitor.WSHub
	jwtSvc *auth.JWTService
}

func NewWSHandler(hub *monitor.WSHub, jwtSvc *auth.JWTService) *WSHandler {
	return &WSHandler{
		hub:    hub,
		jwtSvc: jwtSvc,
	}
}

func (h *WSHandler) HandleWS(c echo.Context) error {
	// Authenticate via query parameter or header
	token := c.QueryParam("token")
	if token == "" {
		authHeader := c.Request().Header.Get("Authorization")
		if authHeader != "" {
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) == 2 {
				token = parts[1]
			}
		}
	}

	if token != "" {
		if _, err := h.jwtSvc.ValidateAccessToken(token); err != nil {
			slog.Warn("ws auth failed", slog.Any("error", err))
			return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		}
	}

	conn, err := upgrader.Upgrade(c.Response(), c.Request(), nil)
	if err != nil {
		slog.Error("ws upgrade failed", slog.Any("error", err))
		return err
	}

	client := monitor.NewWSClient(h.hub, conn)
	h.hub.Register(client)

	go client.WritePump()
	go client.ReadPump()

	return nil
}
