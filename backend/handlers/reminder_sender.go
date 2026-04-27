package handlers

import (
	"client-monitor/database"
	"client-monitor/models"
)

// ReminderSender 提醒发送器
type ReminderSender struct {
	hub *WebSocketHub
}

// NewReminderSender 创建提醒发送器
func NewReminderSender() *ReminderSender {
	return &ReminderSender{
		hub: GetWebSocketHub(),
	}
}

// SendReminder 发送提醒
func (s *ReminderSender) SendReminder(reminder *models.Reminder) error {
	// 获取用户
	var user models.AIUser
	if err := database.DB.First(&user, reminder.UserID).Error; err != nil {
		return err
	}

	// 发送 WebSocket 消息
	SendReminderToUser(user.WecomUserID, "⏰ "+reminder.Content)

	return nil
}

// SendReminderViaWebSocket 全局函数（兼容旧代码）
func SendReminderViaWebSocket(reminder *models.Reminder) error {
	sender := NewReminderSender()
	return sender.SendReminder(reminder)
}
