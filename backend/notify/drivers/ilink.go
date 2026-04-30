package drivers

import (
	"context"
	"errors"
	"log"
	"strings"

	"client-monitor/messaging"
	"client-monitor/notify/types"
	"client-monitor/wecom"
)

// Errors
var (
	ErrNotAvailable  = errors.New("driver not available")
	ErrUserNotFound  = errors.New("user not found")
)

// ILinkDriver iLink 通知驱动
type ILinkDriver struct{}

// NewILinkDriver 创建 iLink 驱动
func NewILinkDriver() *ILinkDriver {
	return &ILinkDriver{}
}

// Name 返回驱动名称
func (d *ILinkDriver) Name() string {
	return "ilink"
}

// Send 发送消息
func (d *ILinkDriver) Send(msg *types.NotifyMessage) error {
	if !d.IsAvailable() {
		log.Printf("[ilink] Send: driver not available")
		return ErrNotAvailable
	}

	client := wecom.GetClient()
	if client == nil {
		log.Printf("[ilink] Send: client is nil")
		return ErrNotAvailable
	}

	userID := wecom.GetILinkUserID()
	log.Printf("[ilink] Send: userID=%s, msgType=%s, title=%s", userID, msg.Type, msg.Title)

	if userID == "" {
		return ErrUserNotFound
	}

	// 格式化消息
	content := d.formatMessage(msg)

	// 发送消息
	if err := messaging.SendTextReply(context.Background(), client, userID, content, "", ""); err != nil {
		log.Printf("[ilink] Send failed: %v", err)
		return err
	}

	log.Printf("[ilink] Sent %s message to %s: %s", msg.Type, userID, msg.Title)
	return nil
}

// IsAvailable 检查驱动是否可用
func (d *ILinkDriver) IsAvailable() bool {
	// 直接检查内存中的 client 是否存在
	client := wecom.GetClient()
	if client == nil {
		log.Printf("[ilink] IsAvailable: client is nil")
		return false
	}
	// 还需要检查 userID
	userID := wecom.GetILinkUserID()
	log.Printf("[ilink] IsAvailable: userID=%s", userID)
	return userID != ""
}

// SupportedTypes 返回支持的消息类型（空表示支持所有）
func (d *ILinkDriver) SupportedTypes() []types.MessageType {
	return nil // 支持所有类型
}

// formatMessage 格式化消息内容
func (d *ILinkDriver) formatMessage(msg *types.NotifyMessage) string {
	var sb strings.Builder

	// 根据消息类型添加图标
	icon := d.getIcon(msg.Type)
	if icon != "" {
		sb.WriteString(icon)
		sb.WriteString(" ")
	}

	// 添加标题
	if msg.Title != "" {
		sb.WriteString(msg.Title)
		sb.WriteString("\n\n")
	}

	// 添加内容
	sb.WriteString(msg.Content)

	return sb.String()
}

// getIcon 获取消息类型对应的图标
func (d *ILinkDriver) getIcon(msgType types.MessageType) string {
	switch msgType {
	case types.MessageTypeAlert:
		return "🚨"
	case types.MessageTypeWeather:
		return "🌤️"
	case types.MessageTypeReminder:
		return "⏰"
	case types.MessageTypeSystem:
		return "📢"
	case types.MessageTypeChat:
		return "💬"
	default:
		return ""
	}
}
