package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"adedanha-golang/models"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for development
	},
}

// Client represents a single WebSocket connection
type Client struct {
	MatchID string
	UserID  string
	Conn    *websocket.Conn
	Send    chan []byte
	closed  bool
	mu      sync.Mutex
}

// SafeClose safely closes the Send channel only once
func (c *Client) SafeClose() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.closed {
		c.closed = true
		close(c.Send)
	}
}

// Hub maintains the set of active clients and broadcasts messages
type Hub struct {
	Matches          map[string]map[*Client]bool
	GlobalClients    map[*Client]bool
	Register         chan *Client
	Unregister       chan *Client
	GlobalRegister   chan *Client
	GlobalUnregister chan *Client
	mu               sync.RWMutex
}

var GameHub *Hub

func NewHub() *Hub {
	return &Hub{
		Matches:          make(map[string]map[*Client]bool),
		GlobalClients:    make(map[*Client]bool),
		Register:         make(chan *Client),
		Unregister:       make(chan *Client),
		GlobalRegister:   make(chan *Client),
		GlobalUnregister: make(chan *Client),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.Register:
			h.mu.Lock()
			if _, ok := h.Matches[client.MatchID]; !ok {
				h.Matches[client.MatchID] = make(map[*Client]bool)
			}
			h.Matches[client.MatchID][client] = true
			h.mu.Unlock()
			log.Printf("Client %s connected to match %s", client.UserID, client.MatchID)

		case client := <-h.Unregister:
			h.mu.Lock()
			if clients, ok := h.Matches[client.MatchID]; ok {
				if _, ok := clients[client]; ok {
					delete(clients, client)
					client.SafeClose()
					if len(clients) == 0 {
						delete(h.Matches, client.MatchID)
					}
				}
			}
			h.mu.Unlock()
			log.Printf("Client %s disconnected from match %s", client.UserID, client.MatchID)

		case client := <-h.GlobalRegister:
			h.mu.Lock()
			h.GlobalClients[client] = true
			h.mu.Unlock()
			log.Printf("Client %s connected globally", client.UserID)

		case client := <-h.GlobalUnregister:
			h.mu.Lock()
			if _, ok := h.GlobalClients[client]; ok {
				delete(h.GlobalClients, client)
				client.SafeClose()
			}
			h.mu.Unlock()
			log.Printf("Client %s disconnected globally", client.UserID)
		}
	}
}

// GetOnlineUserIDs returns a deduplicated list of user IDs currently connected via WebSocket
func (h *Hub) GetOnlineUserIDs() []string {
	h.mu.RLock()
	defer h.mu.RUnlock()

	userSet := make(map[string]bool)
	for _, clients := range h.Matches {
		for client := range clients {
			userSet[client.UserID] = true
		}
	}
	for client := range h.GlobalClients {
		userSet[client.UserID] = true
	}

	ids := make([]string, 0, len(userSet))
	for id := range userSet {
		ids = append(ids, id)
	}
	return ids
}

// BroadcastToMatch sends a message to all clients in a match
func BroadcastToMatch(matchID string, message models.WSMessage) {
	if GameHub == nil {
		return
	}

	data, err := json.Marshal(message)
	if err != nil {
		log.Printf("Error marshaling message: %v", err)
		return
	}

	GameHub.mu.RLock()
	clients, ok := GameHub.Matches[matchID]
	if !ok {
		GameHub.mu.RUnlock()
		return
	}

	// Collect clients to remove
	var toRemove []*Client
	for client := range clients {
		select {
		case client.Send <- data:
		default:
			toRemove = append(toRemove, client)
		}
	}
	GameHub.mu.RUnlock()

	// Remove dead clients outside the read lock
	if len(toRemove) > 0 {
		GameHub.mu.Lock()
		for _, client := range toRemove {
			if clients, ok := GameHub.Matches[matchID]; ok {
				if _, ok := clients[client]; ok {
					delete(clients, client)
					client.SafeClose()
				}
			}
		}
		GameHub.mu.Unlock()
	}
}

// BroadcastToGlobal sends a message to global presence clients
func BroadcastToGlobal(message models.WSMessage) {
	if GameHub == nil {
		return
	}

	data, err := json.Marshal(message)
	if err != nil {
		log.Printf("Error marshaling message: %v", err)
		return
	}

	GameHub.mu.RLock()
	defer GameHub.mu.RUnlock()

	for client := range GameHub.GlobalClients {
		if message.UserID != "" && client.UserID != message.UserID {
			continue
		}
		select {
		case client.Send <- data:
		default:
			// Skip if buffer is full — don't remove under read lock
		}
	}
}

// HandleWebSocket handles WebSocket connection requests
func HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	matchID := vars["matchId"]
	userID := vars["userId"]

	if matchID == "" || userID == "" {
		http.Error(w, "matchId and userId are required", http.StatusBadRequest)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}

	client := &Client{
		MatchID: matchID,
		UserID:  userID,
		Conn:    conn,
		Send:    make(chan []byte, 256),
	}

	GameHub.Register <- client

	go client.writePump()
	go client.readPump()
}

func (c *Client) writePump() {
	pingTicker := time.NewTicker(30 * time.Second)
	defer func() {
		pingTicker.Stop()
		c.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.Send:
			if !ok {
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.Conn.WriteMessage(websocket.TextMessage, message); err != nil {
				return
			}
		case <-pingTicker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func (c *Client) readPump() {
	defer func() {
		if c.MatchID != "" {
			GameHub.Unregister <- c
		} else {
			GameHub.GlobalUnregister <- c
		}
		c.Conn.Close()
	}()

	c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, _, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure, websocket.CloseNoStatusReceived) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}
		c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	}
}

// HandlePresenceWebSocket handles global presence WebSocket connections
func HandlePresenceWebSocket(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID := vars["userId"]

	if userID == "" {
		http.Error(w, "userId is required", http.StatusBadRequest)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}

	client := &Client{
		MatchID: "",
		UserID:  userID,
		Conn:    conn,
		Send:    make(chan []byte, 256),
	}

	GameHub.GlobalRegister <- client

	go client.writePump()
	go client.readPump()
}
