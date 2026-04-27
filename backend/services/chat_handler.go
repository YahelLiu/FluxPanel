package services

import (
	"strings"
	"sync"

	"client-monitor/database"
	"client-monitor/models"
)

// ChatHandler 聊天处理器
type ChatHandler struct {
	llm         *LLMService
	reminderHdl *ReminderHandler
	memoryHdl   *MemoryHandler
}

// NewChatHandler 创建聊天处理器
func NewChatHandler() *ChatHandler {
	return &ChatHandler{
		llm:         GetLLMService(),
		reminderHdl: NewReminderHandler(),
		memoryHdl:   NewMemoryHandler(),
	}
}

// Handle 处理聊天消息
func (h *ChatHandler) Handle(userID uint, userMessage string) (string, error) {
	// 并行获取上下文
	context := h.getContext(userID)

	// 构建消息
	messages := h.buildMessages(context, userMessage)

	// 调用 LLM
	response, err := h.llm.ChatWithSystem(h.buildSystemPrompt(context), messages)
	if err != nil {
		return "", err
	}

	// 保存对话记录
	h.saveConversation(userID, userMessage, response)

	return response, nil
}

// StreamHandle 流式处理聊天消息
func (h *ChatHandler) StreamHandle(userID uint, userMessage string, onChunk func(string) error) error {
	// 并行获取上下文
	context := h.getContext(userID)

	// 构建消息
	messages := h.buildMessages(context, userMessage)

	// 流式调用 LLM
	var fullResponse strings.Builder
	err := h.llm.ChatWithSystemStream(h.buildSystemPrompt(context), messages, func(chunk string) error {
		fullResponse.WriteString(chunk)
		return onChunk(chunk)
	})

	if err != nil {
		return err
	}

	// 保存对话记录
	h.saveConversation(userID, userMessage, fullResponse.String())

	return nil
}

// ChatContext 聊天上下文
type ChatContext struct {
	Memories      []models.Memory
	Conversations []models.Conversation
}

// getContext 并行获取上下文
func (h *ChatHandler) getContext(userID uint) *ChatContext {
	ctx := &ChatContext{}
	var wg sync.WaitGroup

	wg.Add(2)
	go func() {
		defer wg.Done()
		database.DB.Where("user_id = ?", userID).Order("created_at desc").Limit(10).Find(&ctx.Memories)
	}()
	go func() {
		defer wg.Done()
		database.DB.Where("user_id = ?", userID).Order("created_at desc").Limit(10).Find(&ctx.Conversations)
	}()
	wg.Wait()

	return ctx
}

// buildMessages 构建消息列表
func (h *ChatHandler) buildMessages(ctx *ChatContext, userMessage string) []ChatMessage {
	messages := make([]ChatMessage, 0)

	// 添加历史对话（倒序变正序）
	for i := len(ctx.Conversations) - 1; i >= 0; i-- {
		c := ctx.Conversations[i]
		messages = append(messages, ChatMessage{
			Role:    c.Role,
			Content: c.Content,
		})
	}

	// 添加当前消息
	messages = append(messages, ChatMessage{
		Role:    "user",
		Content: userMessage,
	})

	return messages
}

// buildSystemPrompt 构建系统提示
func (h *ChatHandler) buildSystemPrompt(ctx *ChatContext) string {
	prompt := "你是一个友好的 AI 助手。请用简洁、友好的方式回复用户。"

	if len(ctx.Memories) > 0 {
		prompt += "\n\n## 关于用户的重要信息（请记住并在对话中应用）："
		for _, m := range ctx.Memories {
			prompt += "\n- " + m.Content
		}
		prompt += "\n\n如果用户问到相关内容，请参考以上信息回答。"
	}

	return prompt
}

// saveConversation 保存对话记录
func (h *ChatHandler) saveConversation(userID uint, userMessage, response string) {
	database.DB.Create(&models.Conversation{UserID: userID, Role: "user", Content: userMessage})
	database.DB.Create(&models.Conversation{UserID: userID, Role: "assistant", Content: response})
}
