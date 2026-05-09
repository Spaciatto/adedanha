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
}

// Hub maintains the set of active clients and broadcasts messages
type Hub struct {
	// Registered clients grouped by match ID
	Matches map[string]map[*Client]bool
	// Global presence clients (not in a specific match)
	GlobalClients map[*Client]bool
	// Register requests from clients
	Register chan *Client
	// Unregister requests from clients
	Unregister chan *Client
	// Global register/unregister
	GlobalRegister   chan *Client
	GlobalUnregister chan *Client
	// Mutex for thread-safe access
	mu sync.RWMutex
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
					close(client.Send)
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
				close(client.Send)
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

	// Also check the global presence connections
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
	GameHub.mu.RUnlock()

	if !ok {
		return
	}

	GameHub.mu.RLock()
	for client := range clients {
		select {
		case client.Send <- data:
		default:
			GameHub.mu.RUnlock()
			GameHub.mu.Lock()
			delete(clients, client)
			close(client.Send)
			GameHub.mu.Unlock()
			GameHub.mu.RLock()
		}
	}
	GameHub.mu.RUnlock()
}

// BroadcastToGlobal sends a message to all global presence clients
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
		// Only send to the target user if UserID is specified
		if message.UserID != "" && client.UserID != message.UserID {
			continue
		}
		select {
		case client.Send <- data:
		default:
			// Skip if buffer is full
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

	// Start goroutines for reading and writing
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
			err := c.Conn.WriteMessage(websocket.TextMessage, message)
			if err != nil {
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
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}
		// Reset read deadline on any message
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
