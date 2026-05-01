package messaging

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"client-monitor/agent"
	"client-monitor/ilink"
	"client-monitor/skill"

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
	router        *agent.AgentRouter // AI 后端路由器
	skillManager  *skill.Manager     // Skill 管理器
	contextTokens sync.Map           // map[userID]contextToken
	seenMsgs      sync.Map           // map[int64]time.Time — dedup by message_id
}

// NewHandler creates a new message handler.
func NewHandler(factory AgentFactory, saveDefault SaveDefaultFunc) *Handler {
	return &Handler{
		agents:      make(map[string]agent.Agent),
		factory:     factory,
		saveDefault: saveDefault,
	}
}

// SetRouter 设置 AI 后端路由器
func (h *Handler) SetRouter(router *agent.AgentRouter) {
	h.router = router
}

// SetSkillManager 设置 Skill 管理器
func (h *Handler) SetSkillManager(manager *skill.Manager) {
	h.skillManager = manager
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

	// 处理指令
	if reply, handled := h.handleCommand(ctx, client, msg, text, clientID); handled {
		if reply != "" {
			SendTextReply(ctx, client, msg.FromUserID, reply, msg.ContextToken, clientID)
		}
		return
	}

	// Send to agent
	h.sendToAgent(ctx, client, msg, text, clientID)
}

// handleCommand 处理切换指令，返回 (回复内容, 是否已处理)
func (h *Handler) handleCommand(ctx context.Context, client *ilink.Client, msg ilink.WeixinMessage, text, clientID string) (string, bool) {
	if h.router == nil {
		return "", false
	}

	text = strings.TrimSpace(text)

	switch {
	case text == "/claude":
		if err := h.router.Switch(msg.FromUserID, "claude"); err != nil {
			return fmt.Sprintf("切换失败: %v", err), true
		}
		return "已切换到 Claude 模式 🤖", true

	case text == "/api":
		if err := h.router.Switch(msg.FromUserID, "api"); err != nil {
			return fmt.Sprintf("切换失败: %v", err), true
		}
		return "已切换到 API 模式 🌐", true

	case text == "/mode":
		adapterName := h.router.GetCurrentAdapterName(msg.FromUserID)
		adapters := h.router.ListAdapters()

		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("当前模式: %s\n\n", adapterName))
		sb.WriteString("可用模式:\n")
		for _, info := range adapters {
			marker := " "
			if info.Name == adapterName {
				marker = "●"
			}
			sb.WriteString(fmt.Sprintf("%s %s (%s)\n", marker, info.Name, info.Model))
		}
		return sb.String(), true

	case text == "/models":
		adapters := h.router.ListAdapters()
		var sb strings.Builder
		sb.WriteString("可用模式:\n")
		for _, info := range adapters {
			sb.WriteString(fmt.Sprintf("• %s - %s (%s)\n", info.Name, info.Model, info.Type))
		}
		return sb.String(), true

	case text == "/skills":
		return h.handleListSkills(msg.FromUserID)

	case strings.HasPrefix(text, "/skill "):
		return h.handleSkillCommand(msg.FromUserID, strings.TrimPrefix(text, "/skill "))

	case text == "/help":
		return `可用指令:
/claude - 切换到 Claude 模式
/api - 切换到 API 模式
/mode - 查看当前模式
/models - 查看所有可用模式
/skills - 查看可用技能
/skill enable <id> - 启用技能
/skill disable <id> - 禁用技能
/help - 显示帮助`, true
	}

	return "", false
}

// handleListSkills 列出可用技能
func (h *Handler) handleListSkills(userID string) (string, bool) {
	if h.skillManager == nil {
		return "技能系统未启用", true
	}

	skills, err := h.skillManager.List()
	if err != nil {
		return fmt.Sprintf("获取技能列表失败: %v", err), true
	}

	if len(skills) == 0 {
		return "暂无可用技能", true
	}

	var sb strings.Builder
	sb.WriteString("可用技能:\n\n")
	for _, s := range skills {
		status := "✓"
		if !s.Enabled {
			status = "✗"
		}
		sb.WriteString(fmt.Sprintf("%s %s\n  %s\n\n", status, s.Name, s.Description))
	}
	sb.WriteString("使用 /skill enable/disable <id> 启用或禁用技能")
	return sb.String(), true
}

// handleSkillCommand 处理 skill 子命令
func (h *Handler) handleSkillCommand(userID, cmd string) (string, bool) {
	if h.skillManager == nil {
		return "技能系统未启用", true
	}

	parts := strings.Fields(cmd)
	if len(parts) < 2 {
		return "用法: /skill <enable|disable> <skill_id>", true
	}

	action := parts[0]
	skillID := parts[1]

	switch action {
	case "enable":
		if err := h.skillManager.SetUserEnabled(userID, skillID, true); err != nil {
			return fmt.Sprintf("启用失败: %v", err), true
		}
		return fmt.Sprintf("已启用技能: %s ✓", skillID), true

	case "disable":
		if err := h.skillManager.SetUserEnabled(userID, skillID, false); err != nil {
			return fmt.Sprintf("禁用失败: %v", err), true
		}
		return fmt.Sprintf("已禁用技能: %s ✗", skillID), true

	default:
		return "未知命令，可用: enable, disable", true
	}
}

// sendToAgent sends the message to the appropriate agent and replies.
func (h *Handler) sendToAgent(ctx context.Context, client *ilink.Client, msg ilink.WeixinMessage, text, clientID string) {
	log.Printf("[handler] processing message from %s: %q", msg.FromUserID, truncate(text, 100))

	// Send typing indicator
	go func() {
		if err := SendTypingState(ctx, client, msg.FromUserID, msg.ContextToken); err != nil {
			log.Printf("[handler] failed to send typing state: %v", err)
		}
	}()

	var reply string

	// 优先使用路由器
	if h.router != nil {
		var err error
		reply, err = h.router.Chat(ctx, msg.FromUserID, text)
		if err != nil {
			reply = fmt.Sprintf("抱歉，处理消息时出错了: %v", err)
			log.Printf("[handler] router error: %v", err)
		}
		log.Printf("[handler] router reply to %s via %s: %q",
			msg.FromUserID, h.router.GetCurrentAdapterName(msg.FromUserID), truncate(reply, 100))
	} else {
		// 回退到默认 agent
		ag := h.getDefaultAgent()
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
