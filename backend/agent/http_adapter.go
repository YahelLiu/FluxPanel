package agent

import (
	"context"
	"log"

	"client-monitor/database"
	"client-monitor/models"
	"client-monitor/services"
	"client-monitor/skill"
)

// HTTPAgent HTTP API 适配器（通义千问/OpenAI等）
type HTTPAgent struct {
	chatHandler  *services.ChatHandler
	model        string
	skillManager *skill.Manager
	skillRouter  *skill.Router
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
		chatHandler: services.NewChatHandler(),
		model:       model,
	}
}

// SetSkillManager 设置 skill manager
func (a *HTTPAgent) SetSkillManager(manager *skill.Manager) {
	a.skillManager = manager
	a.skillRouter = skill.NewRouter(manager)
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

	// 2. 路由 skills (如果有)
	var activeSkills []*skill.Skill
	if a.skillRouter != nil {
		var err error
		activeSkills, err = a.skillRouter.Route(conversationID, message)
		if err != nil {
			log.Printf("[http_agent] skill routing error: %v", err)
		}
		if len(activeSkills) > 0 {
			skillNames := make([]string, len(activeSkills))
			for i, s := range activeSkills {
				skillNames[i] = s.Name
			}
			log.Printf("[http_agent] activated skills: %v", skillNames)
		}
	}

	// 3. 直接调用 ChatHandler（统一使用 HandleWithSkills）
	return a.chatHandler.HandleWithSkills(user.ID, message, activeSkills)
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
	if a.chatHandler == nil {
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
