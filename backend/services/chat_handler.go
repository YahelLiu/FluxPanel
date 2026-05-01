package services

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"

	"client-monitor/database"
	"client-monitor/models"
	"client-monitor/skill"
)

// ChatHandler 聊天处理器
type ChatHandler struct {
	llm           *LLMService
	reminderHdl   *ReminderHandler
	memoryHdl     *MemoryHandler
	promptBuilder *skill.PromptBuilder
	toolRegistry  *skill.ToolRegistry
}

// NewChatHandler 创建聊天处理器
func NewChatHandler() *ChatHandler {
	return &ChatHandler{
		llm:           GetLLMService(),
		reminderHdl:   NewReminderHandler(),
		memoryHdl:     NewMemoryHandler(),
		promptBuilder: skill.NewPromptBuilder(),
		toolRegistry:  skill.GetToolRegistry(),
	}
}

// Handle 处理聊天消息
func (h *ChatHandler) Handle(userID uint, userMessage string) (string, error) {
	return h.HandleWithSkills(userID, userMessage, nil)
}

// HandleWithSkills 处理聊天消息（带 skills）
func (h *ChatHandler) HandleWithSkills(userID uint, userMessage string, skills []*skill.Skill) (string, error) {
	// 并行获取上下文
	context := h.getContext(userID)

	// 构建消息
	messages := h.buildMessages(context, userMessage)

	// 构建系统提示（包含 skills）
	systemPrompt := h.buildSystemPrompt(context, skills)

	// 如果有 skills，获取可用的工具定义
	var tools []ToolDefinition
	if len(skills) > 0 {
		tools = h.getToolDefinitions(skills)
	}

	// 如果有工具，使用工具调用流程
	if len(tools) > 0 {
		return h.handleWithTools(userID, systemPrompt, messages, tools, skills)
	}

	// 没有工具，普通聊天
	response, err := h.llm.ChatWithSystem(systemPrompt, messages)
	if err != nil {
		return "", err
	}

	// 保存对话记录
	h.saveConversation(userID, userMessage, response)

	return response, nil
}

// handleWithTools 处理带工具调用的对话
func (h *ChatHandler) handleWithTools(userID uint, systemPrompt string, messages []ChatMessage, tools []ToolDefinition, skills []*skill.Skill) (string, error) {
	const maxIterations = 5 // 最多 5 轮工具调用

	for i := 0; i < maxIterations; i++ {
		// 调用 LLM
		resp, err := h.llm.ChatWithTools(systemPrompt, messages, tools)
		if err != nil {
			return "", err
		}

		choice := resp.Choices[0]
		message := choice.Message

		// 检查是否有工具调用
		if len(message.ToolCalls) == 0 {
			// 没有工具调用，返回最终响应
			return message.Content, nil
		}

		log.Printf("[chat] LLM 请求调用 %d 个工具", len(message.ToolCalls))

		// 将助手消息（包含工具调用）添加到消息历史
		messages = append(messages, ChatMessage{
			Role:      "assistant",
			Content:   message.Content,
			ToolCalls: message.ToolCalls,
		})

		// 执行每个工具调用
		for _, toolCall := range message.ToolCalls {
			log.Printf("[chat] 执行工具: %s", toolCall.Function.Name)

			// 解析参数
			var args map[string]interface{}
			if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &args); err != nil {
				log.Printf("[chat] 解析工具参数失败: %v", err)
				messages = append(messages, ChatMessage{
					Role:       "tool",
					ToolCallID: toolCall.ID,
					Content:    fmt.Sprintf("参数解析失败: %v", err),
				})
				continue
			}

			// 添加 user_id 到参数
			args["user_id"] = fmt.Sprintf("%d", userID)

			// 创建 ToolCall 对象
			tc := &skill.ToolCall{
				ID:        toolCall.ID,
				Name:      toolCall.Function.Name,
				Arguments: args,
			}

			// 执行工具（使用第一个 skill 的权限）
			result, err := h.toolRegistry.Execute(skills[0], tc)
			var resultContent string
			if err != nil {
				resultContent = fmt.Sprintf("工具执行失败: %v", err)
				log.Printf("[chat] 工具执行失败: %v", err)
			} else {
				resultJSON, _ := json.Marshal(result)
				resultContent = string(resultJSON)
				log.Printf("[chat] 工具执行成功: %s", resultContent)
			}

			// 添加工具结果到消息历史
			messages = append(messages, ChatMessage{
				Role:       "tool",
				ToolCallID: toolCall.ID,
				Content:    resultContent,
			})
		}
	}

	// 超过最大迭代次数，强制返回
	return "", fmt.Errorf("工具调用次数超过限制")
}

// getToolDefinitions 从 skills 获取工具定义
func (h *ChatHandler) getToolDefinitions(skills []*skill.Skill) []ToolDefinition {
	var tools []ToolDefinition
	seenTools := make(map[string]bool)

	for _, s := range skills {
		for _, toolName := range s.AllowedTools {
			if seenTools[toolName] {
				continue
			}
			seenTools[toolName] = true

			tool := h.toolRegistry.Get(toolName)
			if tool != nil {
				tools = append(tools, ToolDefinition{
					Type: "function",
					Function: ToolFunction{
						Name:        tool.Name,
						Description: tool.Description,
						Parameters: map[string]interface{}{
							"type":       "object",
							"properties": h.buildParametersSchema(tool.Parameters),
							"required":   h.getRequiredParameters(tool.Parameters),
						},
					},
				})
			}
		}
	}

	return tools
}

// buildParametersSchema 构建参数 schema
func (h *ChatHandler) buildParametersSchema(params map[string]skill.Parameter) map[string]interface{} {
	schema := make(map[string]interface{})
	for name, param := range params {
		schema[name] = map[string]interface{}{
			"type":        param.Type,
			"description": param.Description,
		}
	}
	return schema
}

// getRequiredParameters 获取必需参数列表
func (h *ChatHandler) getRequiredParameters(params map[string]skill.Parameter) []string {
	var required []string
	for name, param := range params {
		if param.Required {
			required = append(required, name)
		}
	}
	return required
}

// StreamHandle 流式处理聊天消息
func (h *ChatHandler) StreamHandle(userID uint, userMessage string, onChunk func(string) error) error {
	// 并行获取上下文
	context := h.getContext(userID)

	// 构建消息
	messages := h.buildMessages(context, userMessage)

	// 流式调用 LLM
	var fullResponse strings.Builder
	err := h.llm.ChatWithSystemStream(h.buildSystemPrompt(context, nil), messages, func(chunk string) error {
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
func (h *ChatHandler) buildSystemPrompt(ctx *ChatContext, skills []*skill.Skill) string {
	prompt := "你是一个友好的 AI 助手。请用简洁、友好的方式回复用户。"

	if len(ctx.Memories) > 0 {
		prompt += "\n\n## 关于用户的重要信息（请记住并在对话中应用）："
		for _, m := range ctx.Memories {
			prompt += "\n- " + m.Content
		}
		prompt += "\n\n如果用户问到相关内容，请参考以上信息回答。"
	}

	// 注入 skills
	if len(skills) > 0 {
		skillPrompt := h.promptBuilder.BuildActivePrompt(skills)
		if skillPrompt != "" {
			prompt += "\n\n" + skillPrompt
		}
	}

	return prompt
}

// saveConversation 保存对话记录
func (h *ChatHandler) saveConversation(userID uint, userMessage, response string) {
	database.DB.Create(&models.Conversation{UserID: userID, Role: "user", Content: userMessage})
	database.DB.Create(&models.Conversation{UserID: userID, Role: "assistant", Content: response})
}
