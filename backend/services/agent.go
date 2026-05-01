package services

import (
	"client-monitor/skill"
)

// AgentService Agent 服务 - 简化为委托层
type AgentService struct {
	chatHandler *ChatHandler
}

var agentService *AgentService

// GetAgentService 获取 Agent 服务单例
func GetAgentService() *AgentService {
	if agentService == nil {
		agentService = &AgentService{
			chatHandler: NewChatHandler(),
		}
	}
	return agentService
}

// ProcessMessage 处理用户消息（无 skills）
func (a *AgentService) ProcessMessage(userID uint, userMessage string) (string, error) {
	return a.chatHandler.HandleWithSkills(userID, userMessage, nil)
}

// ProcessMessageWithSkills 处理用户消息（带 skills）
func (a *AgentService) ProcessMessageWithSkills(userID uint, userMessage string, skills []*skill.Skill) (string, error) {
	return a.chatHandler.HandleWithSkills(userID, userMessage, skills)
}

// StreamChat 流式聊天
func (a *AgentService) StreamChat(userID uint, userMessage string, onChunk func(string) error) error {
	return a.chatHandler.StreamHandle(userID, userMessage, onChunk)
}
