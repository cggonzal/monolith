package ws

import (
	"encoding/json"
	"log"
	"monolith/models"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"gorm.io/gorm"
)

// Hub manages channels, subscriptions, and broadcasts.
type Hub struct {
	// Map from channel name to set of clients subscribed.
	channels   map[string]map[*Client]bool
	register   chan Subscription
	unregister chan Subscription
	broadcast  chan BroadcastMessage
	db         *gorm.DB
	mu         sync.Mutex
}

// Subscription represents a client's subscription to a channel.
type Subscription struct {
	client  *Client
	channel string
}

// BroadcastMessage contains a message destined for a channel.
type BroadcastMessage struct {
	channel string
	data    []byte
}

// NewHub initializes a new Hub.
func NewHub(db *gorm.DB) *Hub {
	return &Hub{
		channels:   make(map[string]map[*Client]bool),
		register:   make(chan Subscription),
		unregister: make(chan Subscription),
		broadcast:  make(chan BroadcastMessage),
		db:         db,
	}
}

// Run starts the hub loop to process registrations, unregistrations, and broadcasts.
func (h *Hub) Run() {
	for {
		select {
		case sub := <-h.register:
			h.mu.Lock()
			if _, ok := h.channels[sub.channel]; !ok {
				h.channels[sub.channel] = make(map[*Client]bool)
			}
			h.channels[sub.channel][sub.client] = true
			sub.client.subscriptions[sub.channel] = true
			h.mu.Unlock()
			log.Printf("Client subscribed to channel %s", sub.channel)

		case sub := <-h.unregister:
			h.mu.Lock()
			if clients, ok := h.channels[sub.channel]; ok {
				if _, exists := clients[sub.client]; exists {
					delete(clients, sub.client)
					delete(sub.client.subscriptions, sub.channel)
					if len(clients) == 0 {
						delete(h.channels, sub.channel)
					}
				}
			}
			h.mu.Unlock()
			log.Printf("Client unsubscribed from channel %s", sub.channel)

		case msg := <-h.broadcast:
			h.mu.Lock()
			// Persist the message in the database.
			messageRecord := models.Message{
				Channel:   msg.channel,
				Content:   string(msg.data),
				CreatedAt: time.Now(),
			}
			if err := h.db.Create(&messageRecord).Error; err != nil {
				log.Printf("DB error: %v", err)
			}
			// Send the message to all clients subscribed to the channel.
			if clients, ok := h.channels[msg.channel]; ok {
				for client := range clients {
					select {
					case client.send <- msg.data:
					default:
						close(client.send)
						delete(clients, client)
					}
				}
			}
			h.mu.Unlock()
		}
	}
}

// Client represents a websocket client.
type Client struct {
	hub           *Hub
	conn          *websocket.Conn
	send          chan []byte
	subscriptions map[string]bool
}

// readPump pumps messages from the websocket connection to the hub.
func (c *Client) readPump() {
	defer func() {
		// Unregister all subscriptions on disconnect.
		for channel := range c.subscriptions {
			c.hub.unregister <- Subscription{
				client:  c,
				channel: channel,
			}
		}
		c.conn.Close()
	}()

	c.conn.SetReadLimit(512)
	c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err,
				websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("error: %v", err)
			}
			break
		}

		// The client is expected to send a JSON message with a command.
		// Example:
		//   {"command": "subscribe", "identifier": "ChatChannel"}
		//   {"command": "message", "identifier": "ChatChannel", "data": "Hello, World!"}
		var clientMsg struct {
			Command    string `json:"command"`
			Identifier string `json:"identifier"`
			Data       string `json:"data"`
		}
		if err := json.Unmarshal(message, &clientMsg); err != nil {
			log.Printf("Invalid message: %s", message)
			continue
		}

		switch clientMsg.Command {
		case "subscribe":
			c.hub.register <- Subscription{
				client:  c,
				channel: clientMsg.Identifier,
			}
		case "unsubscribe":
			c.hub.unregister <- Subscription{
				client:  c,
				channel: clientMsg.Identifier,
			}
		case "message":
			broadcastMsg := BroadcastMessage{
				channel: clientMsg.Identifier,
				data:    []byte(clientMsg.Data),
			}
			c.hub.broadcast <- broadcastMsg
		default:
			log.Printf("Unknown command: %s", clientMsg.Command)
		}
	}
}

// writePump pumps messages from the hub to the websocket connection.
func (c *Client) writePump() {
	ticker := time.NewTicker(54 * time.Second)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()
	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				// The hub closed the channel.
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				return
			}
		case <-ticker.C:
			// Send a ping to keep the connection alive.
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// upgrader is used to upgrade HTTP connections to WebSocket connections.
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	// Allow all origins for simplicity. In production, you should verify the origin.
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// serveWs handles websocket requests from clients.
func serveWs(hub *Hub, w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Upgrade error:", err)
		return
	}
	client := &Client{
		hub:           hub,
		conn:          conn,
		send:          make(chan []byte, 256),
		subscriptions: make(map[string]bool),
	}
	// Start writePump in a separate goroutine.
	go client.writePump()
	client.readPump()
}

// ServeWs is the handler for the /ws endpoint.
// It upgrades the HTTP connection to a WebSocket and registers the client with the shared Hub.
func ServeWs(hub *Hub, w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Upgrade error:", err)
		return
	}
	client := &Client{
		hub:           hub,
		conn:          conn,
		send:          make(chan []byte, 256),
		subscriptions: make(map[string]bool),
	}
	// Start writePump in a separate goroutine.
	go client.writePump()
	client.readPump()
}
