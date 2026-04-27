package services

import (
	"fmt"
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
		return "", fmt.Errorf("无法解析时间: %w", err)
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
