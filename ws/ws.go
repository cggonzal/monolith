/*
Package ws implements a publish/subscribe WebSocket hub with persistent
messages stored in the database.
*/
package ws

import (
	"encoding/json"
	"log"
	"log/slog"
	"monolith/db"
	"monolith/models"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"gorm.io/gorm"
)

var HUB *Hub

// Hub manages channels, subscriptions, and broadcasts.
type Hub struct {
	// Map from channel name to set of clients subscribed.
	channels   map[string]map[*Client]bool
	register   chan Subscription
	unregister chan Subscription
	broadcast  chan BroadcastMessage
	db         *gorm.DB
	mu         sync.RWMutex
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

func InitPubSub() {
	// Initialize the Hub with an empty channels map and channels for register, unregister, and broadcast.
	// This function is called once at application startup.
	slog.Info("Initializing Pub/Sub")
	HUB = newHub(db.GetDB())
	go HUB.Run()
}

// NewHub initializes a new Hub.
func newHub(db *gorm.DB) *Hub {
	return &Hub{
		channels:   make(map[string]map[*Client]bool),
		register:   make(chan Subscription, 256),
		unregister: make(chan Subscription, 256),
		broadcast:  make(chan BroadcastMessage, 256),
		db:         db,
	}
}

// Broadcast enqueues a message to be sent to all clients subscribed to a channel.
// It can be called from any goroutine.
func (h *Hub) Broadcast(channel string, data []byte) {
	h.broadcast <- BroadcastMessage{channel: channel, data: data}
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
			// Persist the message asynchronously to avoid blocking broadcasts.
			go func(channel string, data []byte) {
				record := models.Message{
					Channel:   channel,
					Content:   string(data),
					CreatedAt: time.Now(),
				}
				if err := h.db.Create(&record).Error; err != nil {
					log.Printf("DB error: %v", err)
				}
			}(msg.channel, msg.data)

			// Snapshot clients subscribed to the channel.
			h.mu.RLock()
			var targets []*Client
			if clients, ok := h.channels[msg.channel]; ok {
				for c := range clients {
					targets = append(targets, c)
				}
			}
			h.mu.RUnlock()

			// Send the message outside the lock for scalability.
			for _, client := range targets {
				select {
				case client.send <- msg.data:
				default:
					close(client.send)
					h.mu.Lock()
					delete(h.channels[msg.channel], client)
					h.mu.Unlock()
				}
			}
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

// ServeWs is the handler for the /ws endpoint.
// It upgrades the HTTP connection to a WebSocket and registers the client with the shared Hub.
func ServeWs(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Upgrade error:", err)
		return
	}
	client := &Client{
		hub:           HUB,
		conn:          conn,
		send:          make(chan []byte, 256),
		subscriptions: make(map[string]bool),
	}
	// Start writePump in a separate goroutine.
	go client.writePump()
	client.readPump()
}
