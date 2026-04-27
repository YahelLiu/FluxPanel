package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

// WebSocketHub WebSocket 连接管理中心
type WebSocketHub struct {
	clients    map[*WebSocketClient]bool
	register   chan *WebSocketClient
	unregister chan *WebSocketClient
	broadcast  chan []byte
	mu         sync.RWMutex
}

// WebSocketClient WebSocket 客户端
type WebSocketClient struct {
	hub    *WebSocketHub
	conn   *websocket.Conn
	send   chan []byte
	userID string
}

// NewWebSocketHub 创建 WebSocket 中心
func NewWebSocketHub() *WebSocketHub {
	return &WebSocketHub{
		clients:    make(map[*WebSocketClient]bool),
		register:   make(chan *WebSocketClient, 10),
		unregister: make(chan *WebSocketClient, 10),
		broadcast:  make(chan []byte, 100),
	}
}

// Run 运行 WebSocket 中心
func (h *WebSocketHub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()
			log.Printf("WebSocket connected, user: %s, total: %d", client.userID, len(h.clients))

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}
			h.mu.Unlock()
			log.Printf("WebSocket disconnected, user: %s", client.userID)

		case message := <-h.broadcast:
			h.mu.RLock()
			for client := range h.clients {
				select {
				case client.send <- message:
				default:
					close(client.send)
					delete(h.clients, client)
				}
			}
			h.mu.RUnlock()
		}
	}
}

// Broadcast 广播消息
func (h *WebSocketHub) Broadcast(message []byte) {
	select {
	case h.broadcast <- message:
	default:
		log.Println("Broadcast channel full")
	}
}

// SendToUser 发送消息给特定用户
func (h *WebSocketHub) SendToUser(userID string, message []byte) int {
	h.mu.RLock()
	defer h.mu.RUnlock()

	sent := 0
	for client := range h.clients {
		if client.userID == userID {
			select {
			case client.send <- message:
				sent++
			default:
				// 缓冲区满，跳过
			}
		}
	}
	return sent
}

// ClientCount 获取客户端数量
func (h *WebSocketHub) ClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

// --- 全局实例 ---

var hub *WebSocketHub
var hubOnce sync.Once

// GetWebSocketHub 获取 WebSocket 中心单例
func GetWebSocketHub() *WebSocketHub {
	hubOnce.Do(func() {
		hub = NewWebSocketHub()
		go hub.Run()
	})
	return hub
}

// InitWebSocket 初始化 WebSocket
func InitWebSocket() {
	GetWebSocketHub()
}

// --- HTTP Handler ---

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// HandleWebSocket 处理 WebSocket 连接
func HandleWebSocket(c *gin.Context) {
	userID := c.Query("user_id")

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}

	hub := GetWebSocketHub()
	client := &WebSocketClient{
		hub:    hub,
		conn:   conn,
		send:   make(chan []byte, 10),
		userID: userID,
	}

	hub.register <- client

	// 发送连接成功消息
	conn.WriteJSON(map[string]string{"type": "connected"})

	// 写协程
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer func() {
			ticker.Stop()
			hub.unregister <- client
			conn.Close()
		}()

		for {
			select {
			case message, ok := <-client.send:
				if !ok {
					return
				}
				if err := conn.WriteMessage(websocket.TextMessage, message); err != nil {
					return
				}
			case <-ticker.C:
				if err := conn.WriteJSON(map[string]string{"type": "ping"}); err != nil {
					return
				}
			}
		}
	}()

	// 读循环
	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			break
		}
	}
}

// SendReminderToUser 发送提醒给用户
func SendReminderToUser(userID string, content string) {
	hub := GetWebSocketHub()
	msg := map[string]interface{}{
		"type":    "reminder",
		"content": content,
		"time":    time.Now().Format("15:04:05"),
	}

	data := jsonMarshal(msg)
	sent := hub.SendToUser(userID, data)
	log.Printf("Sent reminder to %d WebSocket connections for user %s", sent, userID)
}

// GetConnectedClients 获取连接数
func GetConnectedClients() int {
	return GetWebSocketHub().ClientCount()
}

func jsonMarshal(v interface{}) []byte {
	data, _ := json.Marshal(v)
	return data
}

// BroadcastMessage 广播消息到所有客户端
func BroadcastMessage(data []byte) {
	GetWebSocketHub().Broadcast(data)
}
