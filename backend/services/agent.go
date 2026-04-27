package services

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"client-monitor/database"
	"client-monitor/models"
)

// AgentService Agent 服务
type AgentService struct {
	llm *LLMService
}

var (
	agentService *AgentService
)

// GetAgentService 获取 Agent 服务单例
func GetAgentService() *AgentService {
	if agentService == nil {
		agentService = &AgentService{
			llm: GetLLMService(),
		}
	}
	return agentService
}

// AnalyzeIntent 分析用户意图
func (a *AgentService) AnalyzeIntent(userMessage string) (*models.AgentResult, error) {
	prompt := fmt.Sprintf(`你是一个意图识别助手。分析用户消息，返回 JSON 格式的意图。

用户消息：%s

返回格式：
{
  "intent": "chat|memory|todo|reminder",
  "action": "create|list|complete|none",
  "content": "提取的内容",
  "time": "时间描述（如有）"
}

判断规则：
- intent=memory: 用户想让你记住某事（"记住..."、"以后..."、"我的xxx是..."）
- intent=todo: 用户想管理待办事项（"加个todo"、"记一下..."、"我有哪些todo"、"完成..."）
- intent=reminder: 用户想设置提醒（"X分钟后提醒我"、"明天X点提醒我"、"提醒我..."）
- intent=chat: 普通聊天

action 规则：
- todo: create（创建）、list（查看列表）、complete（完成）
- reminder: create（创建）
- memory: create（创建）
- chat: none

content 规则：
- 对于 memory：提取要记住的内容
- 对于 todo create：提取待办事项内容
- 对于 todo complete：提取要完成的待办事项关键词
- 对于 reminder：提取提醒内容
- 对于其他：可以留空

time 规则：
- 只在 reminder 时提取时间描述，如"30分钟后"、"明天10点"、"今晚8点"
- 其他情况可以留空

只返回 JSON，不要其他内容。`, userMessage)

	response, err := a.llm.Chat([]ChatMessage{
		{Role: "user", Content: prompt},
	})
	if err != nil {
		return nil, fmt.Errorf("调用 LLM 失败: %w", err)
	}

	// 清理响应，移除可能的 markdown 代码块标记
	response = strings.TrimSpace(response)
	response = strings.TrimPrefix(response, "```json")
	response = strings.TrimPrefix(response, "```")
	response = strings.TrimSuffix(response, "```")
	response = strings.TrimSpace(response)

	var result models.AgentResult
	if err := json.Unmarshal([]byte(response), &result); err != nil {
		log.Printf("解析意图失败: %v, response: %s", err, response)
		// 默认返回 chat
		return &models.AgentResult{
			Intent:  models.IntentChat,
			Action:  models.ActionNone,
			Content: "",
		}, nil
	}

	log.Printf("意图分析结果: intent=%s, action=%s, content=%s, time=%s",
		result.Intent, result.Action, result.Content, result.Time)

	return &result, nil
}

// ProcessMessage 处理用户消息
func (a *AgentService) ProcessMessage(userID uint, userMessage string) (string, error) {
	// 分析意图
	result, err := a.AnalyzeIntent(userMessage)
	if err != nil {
		return "", err
	}

	// 根据意图处理
	switch result.Intent {
	case models.IntentMemory:
		return a.handleMemory(userID, result)
	case models.IntentTodo:
		return a.handleTodo(userID, result)
	case models.IntentReminder:
		return a.handleReminder(userID, result)
	default:
		return a.handleChat(userID, userMessage)
	}
}

// handleMemory 处理记忆
func (a *AgentService) handleMemory(userID uint, result *models.AgentResult) (string, error) {
	memory := models.Memory{
		UserID:  userID,
		Content: result.Content,
	}
	if err := database.DB.Create(&memory).Error; err != nil {
		return "", fmt.Errorf("保存记忆失败: %w", err)
	}
	return "好的，我记住了。", nil
}

// handleTodo 处理 Todo
func (a *AgentService) handleTodo(userID uint, result *models.AgentResult) (string, error) {
	switch result.Action {
	case models.ActionCreate:
		todo := models.Todo{
			UserID:  userID,
			Content: result.Content,
		}
		// 如果有时间，尝试解析
		if result.Time != "" {
			if deadline, err := parseTimeDescription(result.Time); err == nil {
				todo.Deadline = &deadline
			}
		}
		if err := database.DB.Create(&todo).Error; err != nil {
			return "", fmt.Errorf("创建 Todo 失败: %w", err)
		}
		if todo.Deadline != nil {
			return fmt.Sprintf("已添加待办：%s（截止时间：%s）", result.Content, todo.Deadline.Format("2006-01-02 15:04")), nil
		}
		return fmt.Sprintf("已添加待办：%s", result.Content), nil

	case models.ActionList:
		var todos []models.Todo
		if err := database.DB.Where("user_id = ? AND completed = ?", userID, false).Order("created_at desc").Find(&todos).Error; err != nil {
			return "", fmt.Errorf("查询 Todo 失败: %w", err)
		}
		if len(todos) == 0 {
			return "你目前没有待办事项。", nil
		}
		var sb strings.Builder
		sb.WriteString("你的待办事项：\n")
		for i, todo := range todos {
			status := "⬜"
			if todo.Deadline != nil {
				sb.WriteString(fmt.Sprintf("%d. %s %s（截止：%s）\n", i+1, status, todo.Content, todo.Deadline.Format("2006-01-02 15:04")))
			} else {
				sb.WriteString(fmt.Sprintf("%d. %s %s\n", i+1, status, todo.Content))
			}
		}
		return sb.String(), nil

	case models.ActionComplete:
		// 模糊匹配 Todo
		var todos []models.Todo
		if err := database.DB.Where("user_id = ? AND completed = ? AND content ILIKE ?", userID, false, "%"+result.Content+"%").Find(&todos).Error; err != nil {
			return "", fmt.Errorf("查询 Todo 失败: %w", err)
		}
		if len(todos) == 0 {
			return "没有找到匹配的待办事项。", nil
		}
		if len(todos) > 1 {
			return "找到多个匹配的待办事项，请更具体一些。", nil
		}
		todo := todos[0]
		todo.Completed = true
		if err := database.DB.Save(&todo).Error; err != nil {
			return "", fmt.Errorf("更新 Todo 失败: %w", err)
		}
		return fmt.Sprintf("已完成：%s", todo.Content), nil
	}

	return "", fmt.Errorf("未知的 Todo 操作")
}

// handleReminder 处理提醒
func (a *AgentService) handleReminder(userID uint, result *models.AgentResult) (string, error) {
	// 解析时间
	remindAt, err := parseTimeDescription(result.Time)
	if err != nil {
		return "", fmt.Errorf("无法解析时间: %w", err)
	}

	reminder := models.Reminder{
		UserID:  userID,
		Content: result.Content,
		RemindAt: remindAt,
	}
	if err := database.DB.Create(&reminder).Error; err != nil {
		return "", fmt.Errorf("创建提醒失败: %w", err)
	}

	// 计算相对时间描述
	duration := time.Until(remindAt)
	var timeDesc string
	if duration < time.Hour {
		timeDesc = fmt.Sprintf("%d分钟后", int(duration.Minutes()))
	} else if duration < 24*time.Hour {
		timeDesc = fmt.Sprintf("%d小时后", int(duration.Hours()))
	} else {
		timeDesc = remindAt.Format("2006-01-02 15:04")
	}

	return fmt.Sprintf("好的，我会在%s提醒你%s。", timeDesc, result.Content), nil
}

// handleChat 处理普通聊天
func (a *AgentService) handleChat(userID uint, userMessage string) (string, error) {
	// 获取用户记忆
	var memories []models.Memory
	database.DB.Where("user_id = ?", userID).Order("created_at desc").Limit(10).Find(&memories)

	// 获取最近对话
	var conversations []models.Conversation
	database.DB.Where("user_id = ?", userID).Order("created_at desc").Limit(10).Find(&conversations)

	// 构建系统提示
	systemPrompt := "你是一个友好的 AI 助手。请用简洁、友好的方式回复用户。"

	// 添加记忆
	if len(memories) > 0 {
		systemPrompt += "\n\n用户记忆："
		for _, m := range memories {
			systemPrompt += "\n- " + m.Content
		}
	}

	// 构建消息
	messages := make([]ChatMessage, 0)

	// 添加历史对话（倒序变正序）
	for i := len(conversations) - 1; i >= 0; i-- {
		c := conversations[i]
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

	// 调用 LLM
	response, err := a.llm.ChatWithSystem(systemPrompt, messages)
	if err != nil {
		return "", err
	}

	// 保存对话记录
	database.DB.Create(&models.Conversation{UserID: userID, Role: "user", Content: userMessage})
	database.DB.Create(&models.Conversation{UserID: userID, Role: "assistant", Content: response})

	return response, nil
}

// parseTimeDescription 解析时间描述
func parseTimeDescription(desc string) (time.Time, error) {
	desc = strings.TrimSpace(desc)
	now := time.Now()

	// X分钟后
	var minutes int
	if _, err := fmt.Sscanf(desc, "%d分钟后", &minutes); err == nil {
		return now.Add(time.Duration(minutes) * time.Minute), nil
	}

	// X小时后
	var hours int
	if _, err := fmt.Sscanf(desc, "%d小时后", &hours); err == nil {
		return now.Add(time.Duration(hours) * time.Hour), nil
	}

	// 明天
	if strings.Contains(desc, "明天") {
		tomorrow := now.AddDate(0, 0, 1)
		// 尝试解析时间
		var hour, minute int
		if _, err := fmt.Sscanf(desc, "明天%d点%d分", &hour, &minute); err == nil {
			return time.Date(tomorrow.Year(), tomorrow.Month(), tomorrow.Day(), hour, minute, 0, 0, tomorrow.Location()), nil
		}
		if _, err := fmt.Sscanf(desc, "明天%d点", &hour); err == nil {
			return time.Date(tomorrow.Year(), tomorrow.Month(), tomorrow.Day(), hour, 0, 0, 0, tomorrow.Location()), nil
		}
		// 默认明天上午9点
		return time.Date(tomorrow.Year(), tomorrow.Month(), tomorrow.Day(), 9, 0, 0, 0, tomorrow.Location()), nil
	}

	// 今晚
	if strings.Contains(desc, "今晚") {
		var hour int
		if _, err := fmt.Sscanf(desc, "今晚%d点", &hour); err == nil {
			return time.Date(now.Year(), now.Month(), now.Day(), hour, 0, 0, 0, now.Location()), nil
		}
		// 默认今晚8点
		return time.Date(now.Year(), now.Month(), now.Day(), 20, 0, 0, 0, now.Location()), nil
	}

	// 今天
	if strings.Contains(desc, "今天") {
		var hour, minute int
		if _, err := fmt.Sscanf(desc, "今天%d点%d分", &hour, &minute); err == nil {
			return time.Date(now.Year(), now.Month(), now.Day(), hour, minute, 0, 0, now.Location()), nil
		}
		if _, err := fmt.Sscanf(desc, "今天%d点", &hour); err == nil {
			return time.Date(now.Year(), now.Month(), now.Day(), hour, 0, 0, 0, now.Location()), nil
		}
	}

	// 下周一/周二/等
	weekdayMap := map[string]time.Weekday{
		"周一": time.Monday, "周二": time.Tuesday, "周三": time.Wednesday,
		"周四": time.Thursday, "周五": time.Friday, "周六": time.Saturday, "周日": time.Sunday,
		"星期一": time.Monday, "星期二": time.Tuesday, "星期三": time.Wednesday,
		"星期四": time.Thursday, "星期五": time.Friday, "星期六": time.Saturday, "星期日": time.Sunday,
	}
	for name, weekday := range weekdayMap {
		if strings.Contains(desc, name) {
			daysUntil := int(weekday - now.Weekday())
			if daysUntil <= 0 {
				daysUntil += 7
			}
			target := now.AddDate(0, 0, daysUntil)
			var hour int
			if _, err := fmt.Sscanf(desc, "%s%d点", name, &hour); err == nil {
				return time.Date(target.Year(), target.Month(), target.Day(), hour, 0, 0, 0, target.Location()), nil
			}
			return time.Date(target.Year(), target.Month(), target.Day(), 9, 0, 0, 0, target.Location()), nil
		}
	}

	// 尝试标准时间格式
	t, err := time.Parse("2006-01-02 15:04", desc)
	if err == nil {
		return t, nil
	}
	t, err = time.Parse("2006-01-02", desc)
	if err == nil {
		return t, nil
	}

	return time.Time{}, fmt.Errorf("无法解析时间: %s", desc)
}
