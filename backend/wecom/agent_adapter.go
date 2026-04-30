package wecom

import (
	"context"

	"client-monitor/agent"
	"client-monitor/database"
	"client-monitor/models"
	"client-monitor/services"
)

// FluxPanelAgent 适配 FluxPanel AgentService 到 weclaw Agent 接口
type FluxPanelAgent struct {
	agentService *services.AgentService
	model        string
}

// NewFluxPanelAgent 创建 FluxPanel Agent 适配器
func NewFluxPanelAgent() *FluxPanelAgent {
	// 从数据库获取模型配置
	var llmConfig models.LLMConfig
	database.DB.Where("enabled = ?", true).First(&llmConfig)

	model := llmConfig.Model
	if model == "" {
		model = "gpt-4o-mini"
	}

	return &FluxPanelAgent{
		agentService: services.GetAgentService(),
		model:        model,
	}
}

// Chat 实现 agent.Agent 接口
// conversationID 是 wecom_user_id (如 "wmXYZ123")
func (a *FluxPanelAgent) Chat(ctx context.Context, conversationID string, message string) (string, error) {
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
func (a *FluxPanelAgent) ResetSession(ctx context.Context, conversationID string) (string, error) {
	var user models.AIUser
	if err := database.DB.Where("wecom_user_id = ?", conversationID).First(&user).Error; err != nil {
		return "", err
	}

	// 删除对话历史
	database.DB.Where("user_id = ?", user.ID).Delete(&models.Conversation{})
	return "会话已重置", nil
}

// Info 返回 Agent 信息
func (a *FluxPanelAgent) Info() agent.AgentInfo {
	return agent.AgentInfo{
		Name:  "fluxpanel",
		Type:  "internal",
		Model: a.model,
	}
}

// SetCwd no-op (FluxPanel agent 不需要工作目录)
func (a *FluxPanelAgent) SetCwd(cwd string) {}
