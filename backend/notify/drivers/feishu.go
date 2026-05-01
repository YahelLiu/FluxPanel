package drivers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"client-monitor/database"
	"client-monitor/models"
	"client-monitor/notify/types"
)

// FeishuDriver 飞书通知驱动
type FeishuDriver struct {
	client *http.Client
}

// NewFeishuDriver 创建飞书驱动
func NewFeishuDriver() *FeishuDriver {
	return &FeishuDriver{
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

// Name 返回驱动名称
func (d *FeishuDriver) Name() string {
	return "feishu"
}

// Send 发送消息
func (d *FeishuDriver) Send(msg *types.NotifyMessage) error {
	log.Printf("[feishu] Send called, checking availability...")

	if !d.IsAvailable() {
		log.Printf("[feishu] Driver not available")
		return ErrNotAvailable
	}

	// 获取飞书配置
	var channel models.NotificationChannel
	if err := database.DB.Where("type = ? AND enabled = ?", models.NotificationTypeFeishu, true).First(&channel).Error; err != nil {
		log.Printf("[feishu] Channel query failed: %v", err)
		return fmt.Errorf("feishu channel not configured: %w", err)
	}

	config := channel.Feishu
	log.Printf("[feishu] WebhookURL: %s", config.WebhookURL)
	if config.WebhookURL == "" {
		return fmt.Errorf("feishu webhook URL is empty")
	}

	// 构建并发送消息
	return d.sendWebhook(&config, msg)
}

// IsAvailable 检查驱动是否可用
func (d *FeishuDriver) IsAvailable() bool {
	var channel models.NotificationChannel
	if err := database.DB.Where("type = ? AND enabled = ?", models.NotificationTypeFeishu, true).First(&channel).Error; err != nil {
		log.Printf("[feishu] IsAvailable: no enabled feishu channel found: %v", err)
		return false
	}
	available := channel.Feishu.WebhookURL != ""
	log.Printf("[feishu] IsAvailable: %v (WebhookURL length: %d)", available, len(channel.Feishu.WebhookURL))
	return available
}

// SupportedTypes 返回支持的消息类型（空表示支持所有）
func (d *FeishuDriver) SupportedTypes() []types.MessageType {
	return nil // 支持所有类型
}

// sendWebhook 通过 Webhook 发送消息
func (d *FeishuDriver) sendWebhook(config *models.FeishuConfig, msg *types.NotifyMessage) error {
	card := d.buildCard(msg)
	message := map[string]interface{}{
		"msg_type": "interactive",
		"card":     card,
	}

	body, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("marshal message failed: %w", err)
	}

	resp, err := d.client.Post(config.WebhookURL, "application/json", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("send request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("feishu api error: status=%d, body=%s", resp.StatusCode, string(respBody))
	}

	var result struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
	}
	if err := json.Unmarshal(respBody, &result); err == nil {
		if result.Code != 0 {
			return fmt.Errorf("feishu api error: code=%d, msg=%s", result.Code, result.Msg)
		}
	}

	log.Printf("[feishu] Sent %s message", msg.Type)
	return nil
}

// buildCard 构建飞书卡片消息
func (d *FeishuDriver) buildCard(msg *types.NotifyMessage) map[string]interface{} {
	// 根据消息类型和优先级确定颜色和图标
	template := "blue"
	emoji := d.getIcon(msg.Type)

	switch msg.Priority {
	case types.PriorityUrgent:
		template = "red"
	case types.PriorityHigh:
		template = "orange"
	case types.PriorityNormal:
		template = "blue"
	case types.PriorityLow:
		template = "green"
	}

	// 构建标题
	title := msg.Title
	if emoji != "" {
		title = emoji + " " + title
	}

	// 构建内容
	var contentBuilder strings.Builder
	contentBuilder.WriteString(msg.Content)

	// 添加来源信息
	if msg.SourceName != "" {
		contentBuilder.WriteString(fmt.Sprintf("\n\n**来源:** %s", msg.SourceName))
	}

	// 添加时间
	if !msg.Timestamp.IsZero() {
		contentBuilder.WriteString(fmt.Sprintf("\n**时间:** %s", msg.Timestamp.Format("2006-01-02 15:04:05")))
	}

	return map[string]interface{}{
		"config": map[string]interface{}{
			"wide_screen_mode": true,
		},
		"header": map[string]interface{}{
			"title": map[string]interface{}{
				"tag":     "plain_text",
				"content": title,
			},
			"template": template,
		},
		"elements": []map[string]interface{}{
			{
				"tag": "div",
				"text": map[string]interface{}{
					"tag":     "lark_md",
					"content": contentBuilder.String(),
				},
			},
		},
	}
}

// getIcon 获取消息类型对应的图标
func (d *FeishuDriver) getIcon(msgType types.MessageType) string {
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
