package handlers

import (
	"client-monitor/models"
	"log"
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for development
	},
}

// WebSocketClient represents a connected WebSocket client
type WebSocketClient struct {
	Conn *websocket.Conn
	Send chan models.Event
}

var (
	clients     = make(map[*WebSocketClient]bool)
	clientsMux  sync.RWMutex
	broadcastCh = make(chan models.Event, 100)
)

// InitWebSocket starts the broadcast goroutine
func InitWebSocket() {
	go handleBroadcast()
}

func handleBroadcast() {
	for event := range broadcastCh {
		clientsMux.RLock()
		for client := range clients {
			select {
			case client.Send <- event:
			default:
				// Client buffer full, skip
			}
		}
		clientsMux.RUnlock()
	}
}

// BroadcastEvent sends an event to all connected WebSocket clients
func BroadcastEvent(event models.Event) {
	select {
	case broadcastCh <- event:
	default:
		log.Println("Broadcast channel full, dropping event")
	}
}

// HandleWebSocket handles WebSocket connections at /ws
func HandleWebSocket(c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}

	client := &WebSocketClient{
		Conn: conn,
		Send: make(chan models.Event, 10),
	}

	clientsMux.Lock()
	clients[client] = true
	clientsMux.Unlock()

	// Send initial connection message
	conn.WriteJSON(map[string]string{"type": "connected"})

	// Start write goroutine
	go func() {
		defer func() {
			conn.Close()
			clientsMux.Lock()
			delete(clients, client)
			clientsMux.Unlock()
		}()

		for {
			select {
			case event := <-client.Send:
				if err := conn.WriteJSON(map[string]interface{}{
					"type":  "event",
					"event": event,
				}); err != nil {
					log.Printf("WebSocket write error: %v", err)
					return
				}
			}
		}
	}()

	// Read loop (keep connection alive, handle close)
	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			break
		}
	}
}

// GetConnectedClients returns the number of connected WebSocket clients
func GetConnectedClients() int {
	clientsMux.RLock()
	defer clientsMux.RUnlock()
	return len(clients)
}
