package messaging

import (
	"context"
	"log"
	"sync"
	"time"

	"client-monitor/agent"
	"client-monitor/ilink"

	"github.com/google/uuid"
)

// AgentFactory creates an agent by config name. Returns nil if the name is unknown.
type AgentFactory func(ctx context.Context, name string) agent.Agent

// SaveDefaultFunc persists the default agent name to config file.
type SaveDefaultFunc func(name string) error

// Handler processes incoming WeChat messages and dispatches replies.
type Handler struct {
	mu            sync.RWMutex
	defaultName   string
	agents        map[string]agent.Agent // name -> running agent
	factory       AgentFactory
	saveDefault   SaveDefaultFunc
	contextTokens sync.Map // map[userID]contextToken
	seenMsgs      sync.Map // map[int64]time.Time — dedup by message_id
}

// NewHandler creates a new message handler.
func NewHandler(factory AgentFactory, saveDefault SaveDefaultFunc) *Handler {
	return &Handler{
		agents:      make(map[string]agent.Agent),
		factory:     factory,
		saveDefault: saveDefault,
	}
}

// SetDefaultAgent sets the default agent (already started).
func (h *Handler) SetDefaultAgent(name string, ag agent.Agent) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.defaultName = name
	h.agents[name] = ag
	log.Printf("[handler] default agent ready: %s (%s)", name, ag.Info())
}

// getDefaultAgent returns the default agent (may be nil if not ready yet).
func (h *Handler) getDefaultAgent() agent.Agent {
	h.mu.RLock()
	defer h.mu.RUnlock()
	if h.defaultName == "" {
		return nil
	}
	return h.agents[h.defaultName]
}

// HandleMessage processes a single incoming message.
func (h *Handler) HandleMessage(ctx context.Context, client *ilink.Client, msg ilink.WeixinMessage) {
	// 诊断日志：记录收到的消息
	log.Printf("[handler] received message: type=%d state=%d from=%s msg_id=%d",
		msg.MessageType, msg.MessageState, msg.FromUserID, msg.MessageID)

	// Only process user messages that are finished
	if msg.MessageType != ilink.MessageTypeUser {
		log.Printf("[handler] skipping non-user message: type=%d", msg.MessageType)
		return
	}
	if msg.MessageState != ilink.MessageStateFinish {
		log.Printf("[handler] skipping non-finish message: state=%d", msg.MessageState)
		return
	}

	// Deduplicate by message_id
	if msg.MessageID != 0 {
		if _, loaded := h.seenMsgs.LoadOrStore(msg.MessageID, time.Now()); loaded {
			return
		}
		go h.cleanSeenMsgs()
	}

	// Extract text from item list
	text := extractText(msg)
	if text == "" {
		log.Printf("[handler] received non-text message from %s, skipping", msg.FromUserID)
		return
	}

	log.Printf("[handler] received from %s: %q", msg.FromUserID, truncate(text, 80))

	// Store context token for this user
	h.contextTokens.Store(msg.FromUserID, msg.ContextToken)

	// Generate a clientID for this reply
	clientID := uuid.New().String()

	// Send to default agent
	h.sendToDefaultAgent(ctx, client, msg, text, clientID)
}

// sendToDefaultAgent sends the message to the default agent and replies.
func (h *Handler) sendToDefaultAgent(ctx context.Context, client *ilink.Client, msg ilink.WeixinMessage, text, clientID string) {
	log.Printf("[handler] processing message from %s: %q", msg.FromUserID, truncate(text, 100))

	// Send typing indicator
	go func() {
		if err := SendTypingState(ctx, client, msg.FromUserID, msg.ContextToken); err != nil {
			log.Printf("[handler] failed to send typing state: %v", err)
		}
	}()

	ag := h.getDefaultAgent()
	var reply string
	if ag != nil {
		var err error
		reply, err = ag.Chat(ctx, msg.FromUserID, text)
		if err != nil {
			reply = "抱歉，处理消息时出错了。"
			log.Printf("[handler] agent error: %v", err)
		}
		log.Printf("[handler] agent reply to %s: %q", msg.FromUserID, truncate(reply, 100))
	} else {
		log.Printf("[handler] WARNING: agent not ready for %s", msg.FromUserID)
		reply = "[系统] AI 服务暂时不可用，请稍后重试。"
	}

	if err := SendTextReply(ctx, client, msg.FromUserID, reply, msg.ContextToken, clientID); err != nil {
		log.Printf("[handler] failed to send reply to %s: %v", msg.FromUserID, err)
	} else {
		log.Printf("[handler] reply sent to %s successfully", msg.FromUserID)
	}
}

// cleanSeenMsgs removes entries older than 5 minutes from the dedup cache.
func (h *Handler) cleanSeenMsgs() {
	cutoff := time.Now().Add(-5 * time.Minute)
	h.seenMsgs.Range(func(key, value any) bool {
		if t, ok := value.(time.Time); ok && t.Before(cutoff) {
			h.seenMsgs.Delete(key)
		}
		return true
	})
}

func extractText(msg ilink.WeixinMessage) string {
	for _, item := range msg.ItemList {
		if item.Type == ilink.ItemTypeText && item.TextItem != nil {
			return item.TextItem.Text
		}
	}
	return ""
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
