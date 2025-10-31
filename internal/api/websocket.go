package api

import (
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"kubeforge/internal/db"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for development
	},
}

// WebSocketHub manages all active WebSocket connections
type WebSocketHub struct {
	clients    map[uint]map[*websocket.Conn]bool
	register   chan *Client
	unregister chan *Client
	broadcast  chan *BroadcastMessage
	mu         sync.RWMutex
}

type Client struct {
	conn      *websocket.Conn
	clusterID uint
	hub       *WebSocketHub
}

type BroadcastMessage struct {
	clusterID uint
	data      interface{}
}

var Hub = &WebSocketHub{
	clients:    make(map[uint]map[*websocket.Conn]bool),
	register:   make(chan *Client),
	unregister: make(chan *Client),
	broadcast:  make(chan *BroadcastMessage, 256),
}

func (h *WebSocketHub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			if h.clients[client.clusterID] == nil {
				h.clients[client.clusterID] = make(map[*websocket.Conn]bool)
			}
			h.clients[client.clusterID][client.conn] = true
			h.mu.Unlock()
			log.Printf("Client registered for cluster %d", client.clusterID)

		case client := <-h.unregister:
			h.mu.Lock()
			if clients, ok := h.clients[client.clusterID]; ok {
				if _, ok := clients[client.conn]; ok {
					delete(clients, client.conn)
					client.conn.Close()
					if len(clients) == 0 {
						delete(h.clients, client.clusterID)
					}
				}
			}
			h.mu.Unlock()
			log.Printf("Client unregistered from cluster %d", client.clusterID)

		case message := <-h.broadcast:
			h.mu.RLock()
			clients := h.clients[message.clusterID]
			h.mu.RUnlock()

			for conn := range clients {
				err := conn.WriteJSON(message.data)
				if err != nil {
					log.Printf("WebSocket write error: %v", err)
					h.unregister <- &Client{conn: conn, clusterID: message.clusterID, hub: h}
				}
			}
		}
	}
}

// BroadcastEvent sends an event to all clients watching a cluster
func (h *WebSocketHub) BroadcastEvent(clusterID uint, event db.Event) {
	h.broadcast <- &BroadcastMessage{
		clusterID: clusterID,
		data:      event,
	}
}

// HandleWebSocket handles WebSocket connections for cluster events
func HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.ParseUint(vars["id"], 10, 32)
	if err != nil {
		WriteError(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid cluster ID")
		return
	}
	clusterID := uint(id)

	// Upgrade HTTP connection to WebSocket
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Failed to upgrade connection: %v", err)
		return
	}

	client := &Client{
		conn:      conn,
		clusterID: clusterID,
		hub:       Hub,
	}

	Hub.register <- client

	// Send recent events immediately
	go func() {
		var events []db.Event
		if err := db.DB.Where("cluster_id = ?", clusterID).
			Order("timestamp desc").
			Limit(50).
			Find(&events).Error; err == nil {
			// Reverse to get chronological order
			for i := len(events) - 1; i >= 0; i-- {
				conn.WriteJSON(events[i])
				time.Sleep(10 * time.Millisecond) // Small delay for better UX
			}
		}
	}()

	// Keep connection alive with ping/pong
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
					Hub.unregister <- client
					return
				}
			}
		}
	}()

	// Read messages from client (if any)
	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			Hub.unregister <- client
			break
		}
	}
}
