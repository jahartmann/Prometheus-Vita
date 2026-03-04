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

type WSHandler struct {
	hub      *monitor.WSHub
	jwtSvc   *auth.JWTService
	upgrader websocket.Upgrader
}

func NewWSHandler(hub *monitor.WSHub, jwtSvc *auth.JWTService, allowedOrigins []string) *WSHandler {
	h := &WSHandler{
		hub:    hub,
		jwtSvc: jwtSvc,
	}
	h.upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			origin := r.Header.Get("Origin")
			if origin == "" {
				return true
			}
			if len(allowedOrigins) == 0 {
				return true
			}
			for _, allowed := range allowedOrigins {
				if allowed == "*" || allowed == origin {
					return true
				}
			}
			slog.Warn("ws origin rejected", slog.String("origin", origin))
			return false
		},
	}
	return h
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

	// Require authentication - reject if no token provided
	if token == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "token required"})
	}

	if _, err := h.jwtSvc.ValidateAccessToken(token); err != nil {
		slog.Warn("ws auth failed", slog.Any("error", err))
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}

	conn, err := h.upgrader.Upgrade(c.Response(), c.Request(), nil)
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
