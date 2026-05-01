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

// ReminderHandler 提醒处理器
type ReminderHandler struct{}

// NewReminderHandler 创建提醒处理器
func NewReminderHandler() *ReminderHandler {
	return &ReminderHandler{}
}

// Create 创建提醒
func (h *ReminderHandler) Create(userID uint, content string, timeDesc string) (string, error) {
	if timeDesc == "" {
		timeDesc = "1小时后"
	}

	remindAt, err := ParseTimeDescription(timeDesc)
	if err != nil {
		// 正则解析失败，尝试 LLM 解析
		log.Printf("[reminder] 正则解析时间失败: %v, 尝试 LLM 解析", err)
		remindAt, err = h.parseTimeWithLLM(content, timeDesc)
		if err != nil {
			return "", fmt.Errorf("无法解析时间: %w", err)
		}
	}

	reminder := models.Reminder{
		UserID:   userID,
		Content:  content,
		RemindAt: remindAt,
	}
	if err := database.DB.Create(&reminder).Error; err != nil {
		return "", fmt.Errorf("创建提醒失败: %w", err)
	}

	return h.formatResponse(content, remindAt), nil
}

// List 查看提醒列表
func (h *ReminderHandler) List(userID uint) (string, error) {
	var reminders []models.Reminder
	if err := database.DB.Where("user_id = ? AND sent = ?", userID, false).Order("remind_at asc").Find(&reminders).Error; err != nil {
		return "", fmt.Errorf("查询提醒失败: %w", err)
	}

	if len(reminders) == 0 {
		return "你目前没有待发送的提醒。", nil
	}

	var sb strings.Builder
	sb.WriteString("你的提醒列表：\n")
	for i, r := range reminders {
		status := "⏰"
		if r.RemindAt.Before(time.Now()) {
			status = "⏳"
		}
		sb.WriteString(fmt.Sprintf("%d. %s %s（时间：%s）\n", i+1, status, r.Content, r.RemindAt.Format("2006-01-02 15:04")))
	}
	return sb.String(), nil
}

// Cancel 取消提醒
func (h *ReminderHandler) Cancel(userID uint, keyword string) (string, error) {
	var reminders []models.Reminder
	if err := database.DB.Where("user_id = ? AND sent = ? AND content ILIKE ?", userID, false, "%"+keyword+"%").Find(&reminders).Error; err != nil {
		return "", fmt.Errorf("查询提醒失败: %w", err)
	}

	if len(reminders) == 0 {
		return "没有找到匹配的提醒。", nil
	}
	if len(reminders) > 1 {
		return "找到多个匹配的提醒，请更具体一些。", nil
	}

	if err := database.DB.Delete(&reminders[0]).Error; err != nil {
		return "", fmt.Errorf("取消提醒失败: %w", err)
	}
	return fmt.Sprintf("已取消提醒：%s", reminders[0].Content), nil
}

// formatResponse 格式化响应消息
func (h *ReminderHandler) formatResponse(content string, remindAt time.Time) string {
	duration := time.Until(remindAt)
	var displayTimeDesc string

	if duration < time.Minute {
		displayTimeDesc = "马上"
	} else if duration < time.Hour {
		displayTimeDesc = fmt.Sprintf("%d分钟后", int(duration.Minutes()+0.5))
	} else if duration < 24*time.Hour {
		displayTimeDesc = fmt.Sprintf("%d小时后", int(duration.Hours()+0.5))
	} else {
		displayTimeDesc = remindAt.Format("2006-01-02 15:04")
	}

	return fmt.Sprintf("好的，我会在%s提醒你%s。", displayTimeDesc, content)
}

// LLMTimeParseResult LLM 时间解析结果
type LLMTimeParseResult struct {
	Content  string `json:"content"`
	RemindAt string `json:"remind_at"`
	Error    string `json:"error,omitempty"`
}

// parseTimeWithLLM 使用 LLM 解析时间
func (h *ReminderHandler) parseTimeWithLLM(content string, timeDesc string) (time.Time, error) {
	llm := GetLLMService()
	if llm == nil {
		return time.Time{}, fmt.Errorf("LLM 服务不可用")
	}

	now := time.Now()
	prompt := fmt.Sprintf(`你是一个时间解析助手。从用户的消息中提取提醒时间。

当前时间：%s
用户提供的提醒内容：%s
用户提供的时间描述：%s

请返回 JSON 格式（只返回 JSON，不要其他内容）：
{
    "content": "提醒内容（如果需要调整）",
    "remind_at": "YYYY-MM-DD HH:MM",
    "error": "如果无法解析时间，在这里说明原因"
}

注意：
1. remind_at 必须是未来时间
2. 如果用户说"下周三"，计算具体日期
3. 如果用户说"月底"，计算当月最后一天
4. 如果时间描述不明确，在 error 中说明
5. 返回的时间格式必须是 YYYY-MM-DD HH:MM`, now.Format("2006-01-02 15:04:05"), content, timeDesc)

	resp, err := llm.Chat([]ChatMessage{
		{Role: "user", Content: prompt},
	})
	if err != nil {
		log.Printf("[reminder] LLM 调用失败: %v", err)
		return time.Time{}, fmt.Errorf("LLM 解析失败: %w", err)
	}

	// 清理响应（可能包含 markdown 代码块）
	resp = strings.TrimSpace(resp)
	if strings.HasPrefix(resp, "```") {
		// 移除 markdown 代码块
		lines := strings.Split(resp, "\n")
		var cleanLines []string
		for _, line := range lines {
			if strings.HasPrefix(line, "```") {
				continue
			}
			cleanLines = append(cleanLines, line)
		}
		resp = strings.Join(cleanLines, "\n")
	}

	var result LLMTimeParseResult
	if err := json.Unmarshal([]byte(resp), &result); err != nil {
		log.Printf("[reminder] 解析 LLM 响应失败: %v, 响应: %s", err, resp)
		return time.Time{}, fmt.Errorf("解析 LLM 响应失败: %w", err)
	}

	// 只有在没有返回时间的情况下，error 才是真正的错误
	if result.RemindAt == "" {
		if result.Error != "" {
			return time.Time{}, fmt.Errorf("LLM 解析错误: %s", result.Error)
		}
		return time.Time{}, fmt.Errorf("LLM 未返回时间")
	}

	// 解析时间
	remindAt, err := time.ParseInLocation("2006-01-02 15:04", result.RemindAt, now.Location())
	if err != nil {
		return time.Time{}, fmt.Errorf("解析时间格式失败: %w", err)
	}

	// 确保是未来时间
	if remindAt.Before(now) {
		return time.Time{}, fmt.Errorf("提醒时间已过")
	}

	// 如果有警告信息，记录日志但不阻断
	if result.Error != "" {
		log.Printf("[reminder] LLM 时间调整提示: %s", result.Error)
	}

	log.Printf("[reminder] LLM 解析成功: content=%s, remind_at=%s", result.Content, result.RemindAt)
	return remindAt, nil
}
