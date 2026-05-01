package agent

import (
	"context"

	"client-monitor/database"
	"client-monitor/models"
	"client-monitor/services"
)

// HTTPAgent HTTP API 适配器（通义千问/OpenAI等）
type HTTPAgent struct {
	agentService *services.AgentService
	model        string
}

// NewHTTPAgent 创建 HTTP API 适配器
func NewHTTPAgent() *HTTPAgent {
	// 从数据库获取模型配置
	var llmConfig models.LLMConfig
	database.DB.Where("enabled = ?", true).First(&llmConfig)

	model := llmConfig.Model
	if model == "" {
		model = "gpt-4o-mini"
	}

	return &HTTPAgent{
		agentService: services.GetAgentService(),
		model:        model,
	}
}

// Chat 实现 Agent 接口
func (a *HTTPAgent) Chat(ctx context.Context, conversationID string, message string) (string, error) {
	// 1. 根据 wecom_user_id 获取或创建 AIUser
	var user models.AIUser
	result := database.DB.Where("wecom_user_id = ?", conversationID).First(&user)
	if result.Error != nil {
		user = models.AIUser{
			WecomUserID: conversationID,
			Name:        conversationID,
		}
		database.DB.Create(&user)
	}

	// 2. 调用现有的 AgentService
	return a.agentService.ProcessMessage(user.ID, message)
}

// ResetSession 清除对话历史
func (a *HTTPAgent) ResetSession(ctx context.Context, conversationID string) (string, error) {
	var user models.AIUser
	if err := database.DB.Where("wecom_user_id = ?", conversationID).First(&user).Error; err != nil {
		return "", err
	}

	// 删除对话历史
	database.DB.Where("user_id = ?", user.ID).Delete(&models.Conversation{})
	return "会话已重置", nil
}

// Info 返回 Agent 信息
func (a *HTTPAgent) Info() AgentInfo {
	// 尝试获取当前配置的模型
	var llmConfig models.LLMConfig
	if err := database.DB.Where("enabled = ?", true).First(&llmConfig).Error; err == nil {
		a.model = llmConfig.Model
	}

	status := "available"
	if a.agentService == nil {
		status = "unavailable"
	}

	return AgentInfo{
		Name:   "api",
		Type:   "http",
		Model:  a.model,
		Status: status,
	}
}

// SetCwd no-op
func (a *HTTPAgent) SetCwd(cwd string) {}
