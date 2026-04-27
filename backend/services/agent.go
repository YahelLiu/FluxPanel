package services

import (
	"log"

	"client-monitor/models"
)

// AgentService Agent 服务 - 协调各处理器
type AgentService struct {
	recognizer   *IntentRecognizer
	chatHandler  *ChatHandler
	memoryHdl    *MemoryHandler
	reminderHdl  *ReminderHandler
}

var agentService *AgentService

// GetAgentService 获取 Agent 服务单例
func GetAgentService() *AgentService {
	if agentService == nil {
		agentService = &AgentService{
			recognizer:   NewIntentRecognizer(),
			chatHandler:  NewChatHandler(),
			memoryHdl:    NewMemoryHandler(),
			reminderHdl:  NewReminderHandler(),
		}
	}
	return agentService
}

// AnalyzeIntent 分析用户意图
func (a *AgentService) AnalyzeIntent(userMessage string) (*models.AgentResult, error) {
	result := a.recognizer.Recognize(userMessage)
	log.Printf("意图分析: intent=%s, action=%s", result.Intent, result.Action)
	return result, nil
}

// ProcessMessage 处理用户消息
func (a *AgentService) ProcessMessage(userID uint, userMessage string) (string, error) {
	result, err := a.AnalyzeIntent(userMessage)
	if err != nil {
		return "", err
	}

	switch result.Intent {
	case models.IntentMemory:
		return a.handleMemory(userID, result)
	case models.IntentReminder:
		return a.handleReminder(userID, result)
	default:
		return a.chatHandler.Handle(userID, userMessage)
	}
}

// StreamChat 流式聊天
func (a *AgentService) StreamChat(userID uint, userMessage string, onChunk func(string) error) error {
	return a.chatHandler.StreamHandle(userID, userMessage, onChunk)
}

// handleMemory 处理记忆意图
func (a *AgentService) handleMemory(userID uint, result *models.AgentResult) (string, error) {
	switch result.Action {
	case models.ActionCreate:
		return a.memoryHdl.Create(userID, result.Content)
	case models.ActionList:
		return a.memoryHdl.List(userID)
	case models.ActionCancel:
		return a.memoryHdl.Delete(userID, result.Content)
	}
	return "未知的记忆操作", nil
}

// handleReminder 处理提醒意图
func (a *AgentService) handleReminder(userID uint, result *models.AgentResult) (string, error) {
	switch result.Action {
	case models.ActionCreate:
		return a.reminderHdl.Create(userID, result.Content, result.Time)
	case models.ActionList:
		return a.reminderHdl.List(userID)
	case models.ActionCancel:
		return a.reminderHdl.Cancel(userID, result.Content)
	}
	return "未知的提醒操作", nil
}
