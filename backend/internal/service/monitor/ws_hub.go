package monitor

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
)

type WSMessage struct {
	Type string `json:"type"`
	Data any    `json:"data"`
}

type WSClient struct {
	hub       *WSHub
	conn      *websocket.Conn
	send      chan []byte
	closeOnce sync.Once
}

// closeSend safely closes the send channel exactly once, even if invoked from
// multiple paths (unregister + broadcast-drop). Without this guard a stale
// client could be closed twice and panic.
func (c *WSClient) closeSend() {
	c.closeOnce.Do(func() {
		close(c.send)
	})
}

type WSHub struct {
	clients    map[*WSClient]bool
	broadcast  chan []byte
	register   chan *WSClient
	unregister chan *WSClient
	done       chan struct{}
	doneOnce   sync.Once
	stopped    atomic.Bool
	dropped    atomic.Uint64
	mu         sync.RWMutex
}

func NewWSHub() *WSHub {
	return &WSHub{
		clients:    make(map[*WSClient]bool),
		broadcast:  make(chan []byte, 1024),
		register:   make(chan *WSClient),
		unregister: make(chan *WSClient),
		done:       make(chan struct{}),
	}
}

func (h *WSHub) Run() {
	for {
		select {
		case <-h.done:
			h.drainOnShutdown()
			return

		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			count := len(h.clients)
			h.mu.Unlock()
			slog.Debug("ws client connected", slog.Int("total", count))

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				client.closeSend()
			}
			count := len(h.clients)
			h.mu.Unlock()
			slog.Debug("ws client disconnected", slog.Int("total", count))

		case message := <-h.broadcast:
			h.mu.Lock()
			for client := range h.clients {
				select {
				case client.send <- message:
				default:
					// Client is slow or stale — drop it. Doing this with the
					// hub lock held is safe because no other handler in this
					// loop runs concurrently.
					delete(h.clients, client)
					client.closeSend()
					h.dropped.Add(1)
				}
			}
			h.mu.Unlock()
		}
	}
}

// drainOnShutdown closes every connected client's send channel so their write
// pumps exit. Called exactly once when the hub is shutting down.
func (h *WSHub) drainOnShutdown() {
	h.mu.Lock()
	defer h.mu.Unlock()
	for client := range h.clients {
		client.closeSend()
	}
	h.clients = map[*WSClient]bool{}
}

// Shutdown stops the hub goroutine and closes every active client connection.
// Safe to call multiple times.
func (h *WSHub) Shutdown(ctx context.Context) error {
	if !h.stopped.CompareAndSwap(false, true) {
		return nil
	}
	h.doneOnce.Do(func() { close(h.done) })
	// Give Run() a moment to drain. We don't strictly need to block on it,
	// but giving it a window lets in-flight broadcasts settle.
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(100 * time.Millisecond):
		return nil
	}
}

func (h *WSHub) BroadcastMessage(msg WSMessage) {
	if h.stopped.Load() {
		return
	}
	data, err := json.Marshal(msg)
	if err != nil {
		slog.Error("failed to marshal ws message", slog.Any("error", err))
		return
	}
	select {
	case h.broadcast <- data:
	default:
		h.dropped.Add(1)
		slog.Warn("ws broadcast channel full, dropping message",
			slog.String("type", msg.Type),
			slog.Uint64("dropped_total", h.dropped.Load()),
		)
	}
}

func (h *WSHub) ClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

// DroppedCount returns the cumulative count of broadcast messages that were
// dropped because either the hub channel was full or a client was too slow.
// Useful for monitoring/alerting.
func (h *WSHub) DroppedCount() uint64 {
	return h.dropped.Load()
}

func (h *WSHub) Register(client *WSClient) {
	if h.stopped.Load() {
		return
	}
	select {
	case h.register <- client:
	case <-h.done:
	}
}

func (h *WSHub) Unregister(client *WSClient) {
	if h.stopped.Load() {
		// Hub is gone; close the client's send channel directly so its
		// write pump can exit.
		client.closeSend()
		return
	}
	select {
	case h.unregister <- client:
	case <-h.done:
		client.closeSend()
	}
}

func NewWSClient(hub *WSHub, conn *websocket.Conn) *WSClient {
	return &WSClient{
		hub:  hub,
		conn: conn,
		send: make(chan []byte, 256),
	}
}

func (c *WSClient) WritePump() {
	ticker := time.NewTicker(30 * time.Second)
	defer func() {
		ticker.Stop()
		_ = c.conn.Close()
	}()
	for {
		select {
		case message, ok := <-c.send:
			if !ok {
				_ = c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			_ = c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				return
			}
		case <-ticker.C:
			_ = c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func (c *WSClient) ReadPump() {
	defer func() {
		c.hub.Unregister(c)
		_ = c.conn.Close()
	}()
	_ = c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.conn.SetPongHandler(func(string) error {
		_ = c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})
	for {
		_, _, err := c.conn.ReadMessage()
		if err != nil {
			return
		}
	}
}
