package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"runtime/debug"
	"strings"
	"sync"

	"github.com/antigravity/prometheus/internal/model"
	"github.com/antigravity/prometheus/internal/service/auth"
	"github.com/antigravity/prometheus/internal/service/logstream"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
)

type logStreamSubscribeMsg struct {
	NodeIDs        []string `json:"node_ids"`
	Sources        []string `json:"sources"`
	SeverityFilter []string `json:"severity_filter"`
}

type logStreamMsg struct {
	Type string         `json:"type"`
	Data model.LogEntry `json:"data"`
}

type LogStreamHandler struct {
	streamMgr *logstream.StreamManager
	jwtSvc    *auth.JWTService
	upgrader  websocket.Upgrader
}

func NewLogStreamHandler(
	streamMgr *logstream.StreamManager,
	jwtSvc *auth.JWTService,
	origins []string,
) *LogStreamHandler {
	h := &LogStreamHandler{
		streamMgr: streamMgr,
		jwtSvc:    jwtSvc,
	}
	h.upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 4096,
		CheckOrigin: func(r *http.Request) bool {
			origin := r.Header.Get("Origin")
			if origin == "" {
				return true
			}
			if len(origins) == 0 {
				return true
			}
			for _, allowed := range origins {
				if allowed == "*" || allowed == origin {
					return true
				}
			}
			slog.Warn("log stream ws origin rejected", slog.String("origin", origin))
			return false
		},
	}
	return h
}

func (h *LogStreamHandler) HandleWS(c echo.Context) error {
	token := c.QueryParam("token")
	if token == "" {
		// Try Authorization header as fallback
		authHeader := c.Request().Header.Get("Authorization")
		if strings.HasPrefix(authHeader, "Bearer ") {
			token = strings.TrimPrefix(authHeader, "Bearer ")
		}
	}
	if token == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "token required"})
	}

	if _, err := h.jwtSvc.ValidateAccessToken(token); err != nil {
		slog.Warn("log stream ws auth failed", slog.Any("error", err))
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}

	conn, err := h.upgrader.Upgrade(c.Response(), c.Request(), nil)
	if err != nil {
		slog.Error("log stream ws upgrade failed", slog.Any("error", err))
		return err
	}
	defer conn.Close()

	// Read subscription message from client.
	var sub logStreamSubscribeMsg
	if err := conn.ReadJSON(&sub); err != nil {
		slog.Warn("log stream ws: failed to read subscription", slog.Any("error", err))
		return nil
	}

	// Build source and severity filter sets for fast lookup.
	sourceSet := make(map[string]struct{}, len(sub.Sources))
	for _, s := range sub.Sources {
		sourceSet[s] = struct{}{}
	}
	severitySet := make(map[string]struct{}, len(sub.SeverityFilter))
	for _, s := range sub.SeverityFilter {
		severitySet[s] = struct{}{}
	}

	// Subscribe to each requested node.
	type subscription struct {
		nodeID string
		ch     <-chan model.LogEntry
	}

	subs := make([]subscription, 0, len(sub.NodeIDs))
	for _, nid := range sub.NodeIDs {
		// Validate UUID format but subscribe using string key.
		if _, err := uuid.Parse(nid); err != nil {
			continue
		}
		ch := h.streamMgr.Subscribe(nid)
		subs = append(subs, subscription{nodeID: nid, ch: ch})
	}

	defer func() {
		for _, s := range subs {
			h.streamMgr.Unsubscribe(s.nodeID, s.ch)
		}
	}()

	// Fan-in all subscription channels into a single merged channel.
	merged := make(chan model.LogEntry, 256)
	done := make(chan struct{})
	var fanWg sync.WaitGroup

	for _, s := range subs {
		s := s
		fanWg.Add(1)
		go func() {
			defer fanWg.Done()
			defer func() {
				if r := recover(); r != nil {
					slog.Error("log stream fan-in goroutine panicked",
						slog.String("node_id", s.nodeID),
						slog.Any("panic", r),
						slog.String("stack", string(debug.Stack())),
					)
				}
			}()
			for {
				select {
				case <-done:
					return
				case entry, ok := <-s.ch:
					if !ok {
						return
					}
					select {
					case merged <- entry:
					default:
						// Drop if merged is full
					}
				}
			}
		}()
	}

	// Close merged after all fan-in goroutines exit.
	go func() {
		fanWg.Wait()
		close(merged)
	}()

	defer close(done)

	// Write pump: send entries from merged channel to WebSocket.
	for entry := range merged {
		// Apply source filter.
		if len(sourceSet) > 0 {
			if _, ok := sourceSet[entry.Source]; !ok {
				continue
			}
		}
		// Apply severity filter.
		if len(severitySet) > 0 {
			if _, ok := severitySet[entry.Severity]; !ok {
				continue
			}
		}

		msg := logStreamMsg{Type: "log", Data: entry}
		data, err := json.Marshal(msg)
		if err != nil {
			continue
		}
		if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
			slog.Debug("log stream ws: write error, closing", slog.Any("error", err))
			return nil
		}
	}
	return nil
}
