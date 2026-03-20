package monitor

import (
	"encoding/json"
	"log/slog"
	"sync"

	"github.com/gorilla/websocket"
)

type WSMessage struct {
	Type string `json:"type"`
	Data any    `json:"data"`
}

type WSClient struct {
	hub  *WSHub
	conn *websocket.Conn
	send chan []byte
}

type WSHub struct {
	clients    map[*WSClient]bool
	broadcast  chan []byte
	register   chan *WSClient
	unregister chan *WSClient
	mu         sync.RWMutex
}

func NewWSHub() *WSHub {
	return &WSHub{
		clients:    make(map[*WSClient]bool),
		broadcast:  make(chan []byte, 256),
		register:   make(chan *WSClient),
		unregister: make(chan *WSClient),
	}
}

func (h *WSHub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()
			slog.Debug("ws client connected", slog.Int("total", len(h.clients)))

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}
			h.mu.Unlock()
			slog.Debug("ws client disconnected", slog.Int("total", len(h.clients)))

		case message := <-h.broadcast:
			h.mu.RLock()
			var stale []*WSClient
			for client := range h.clients {
				select {
				case client.send <- message:
				default:
					stale = append(stale, client)
				}
			}
			h.mu.RUnlock()

			if len(stale) > 0 {
				h.mu.Lock()
				for _, client := range stale {
					if _, ok := h.clients[client]; ok {
						delete(h.clients, client)
						close(client.send)
					}
				}
				h.mu.Unlock()
			}
		}
	}
}

func (h *WSHub) BroadcastMessage(msg WSMessage) {
	data, err := json.Marshal(msg)
	if err != nil {
		slog.Error("failed to marshal ws message", slog.Any("error", err))
		return
	}
	select {
	case h.broadcast <- data:
	default:
		slog.Warn("ws broadcast channel full, dropping message")
	}
}

func (h *WSHub) ClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

func (h *WSHub) Register(client *WSClient) {
	h.register <- client
}

func (h *WSHub) Unregister(client *WSClient) {
	h.unregister <- client
}

func NewWSClient(hub *WSHub, conn *websocket.Conn) *WSClient {
	return &WSClient{
		hub:  hub,
		conn: conn,
		send: make(chan []byte, 256),
	}
}

func (c *WSClient) WritePump() {
	defer c.conn.Close()

	for message := range c.send {
		if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
			return
		}
	}
}

func (c *WSClient) ReadPump() {
	defer func() {
		c.hub.Unregister(c)
		c.conn.Close()
	}()

	for {
		_, _, err := c.conn.ReadMessage()
		if err != nil {
			return
		}
	}
}
