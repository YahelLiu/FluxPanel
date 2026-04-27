package notify

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"client-monitor/models"
)

// WechatWorkNotifier 企业微信通知器
type WechatWorkNotifier struct {
	config models.WechatWorkConfig
}

// NewWechatWorkNotifier 创建企业微信通知器
func NewWechatWorkNotifier(config models.WechatWorkConfig) *WechatWorkNotifier {
	return &WechatWorkNotifier{config: config}
}

// SendWebhook 发送 Webhook 消息
func (w *WechatWorkNotifier) SendWebhook(title, content string, event models.Event) error {
	if w.config.WebhookURL == "" {
		return fmt.Errorf("wechat work webhook URL is empty")
	}

	// 构建卡片消息
	msg := map[string]interface{}{
		"msgtype": "markdown",
		"markdown": map[string]interface{}{
			"content": w.formatMarkdown(title, content, event),
		},
	}

	return w.sendRequest(msg)
}

// SendWebhookCard 发送卡片消息
func (w *WechatWorkNotifier) SendWebhookCard(title, content string, event models.Event) error {
	if w.config.WebhookURL == "" {
		return fmt.Errorf("wechat work webhook URL is empty")
	}

	// 构建模板卡片消息
	cardContent := map[string]interface{}{
		"card_type": "text_notice",
		"main_title": map[string]interface{}{
			"title": title,
		},
		"sub_title_text": fmt.Sprintf("状态: %s | 时间: %s", event.Status, event.CreatedAt.Format("2006-01-02 15:04:05")),
		"horizontal_content_list": []map[string]interface{}{
			{
				"key":   "客户端",
				"value": event.ClientID,
			},
			{
				"key":   "事件类型",
				"value": event.EventType,
			},
		},
	}

	if content != "" {
		cardContent["card_image"] = map[string]interface{}{
			"url": "https://wework.qpic.cn/wwpic/2528_1176235270_1624065719_0",
		}
		cardContent["card_action"] = map[string]interface{}{
			"type": 1,
			"url":  "https://github.com/YahelLiu/FluxPanel",
		}
	}

	// 根据状态设置强调色
	if event.Status == "error" {
		cardContent["emphasis_content"] = map[string]interface{}{
			"title": "异常告警",
			"desc":  content,
		}
	} else if event.Status == "warning" {
		cardContent["emphasis_content"] = map[string]interface{}{
			"title": "警告",
			"desc":  content,
		}
	}

	msg := map[string]interface{}{
		"msgtype": "template_card",
		"template_card": cardContent,
	}

	return w.sendRequest(msg)
}

func (w *WechatWorkNotifier) formatMarkdown(title, content string, event models.Event) string {
	emoji := "✅"
	if event.Status == "error" {
		emoji = "🚨"
	} else if event.Status == "warning" {
		emoji = "⚠️"
	}

	return fmt.Sprintf(`%s **%s**

> 状态: <font color="%s">%s</font>
> 客户端: %s
> 事件类型: %s
> 时间: %s

%s
---
来自 FluxPanel 监控系统`,
		emoji,
		title,
		w.getStatusColor(event.Status),
		event.Status,
		event.ClientID,
		event.EventType,
		event.CreatedAt.Format("2006-01-02 15:04:05"),
		content,
	)
}

func (w *WechatWorkNotifier) getStatusColor(status string) string {
	switch status {
	case "error":
		return "warning"
	case "warning":
		return "comment"
	default:
		return "info"
	}
}

// sendRequest 发送 HTTP 请求
func (w *WechatWorkNotifier) sendRequest(msg interface{}) error {
	body, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal message failed: %w", err)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Post(w.config.WebhookURL, "application/json", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("send request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("wechat work api error: status=%d, body=%s", resp.StatusCode, string(respBody))
	}

	// 检查响应
	var result struct {
		ErrCode int    `json:"errcode"`
		ErrMsg  string `json:"errmsg"`
	}
	if err := json.Unmarshal(respBody, &result); err == nil {
		if result.ErrCode != 0 {
			return fmt.Errorf("wechat work api error: code=%d, msg=%s", result.ErrCode, result.ErrMsg)
		}
	}

	return nil
}

// --- 企业微信应用消息 ---

// GetAccessToken 获取企业微信访问令牌
func (w *WechatWorkNotifier) GetAccessToken() (string, error) {
	if w.config.CorpID == "" || w.config.Secret == "" {
		return "", fmt.Errorf("wechat work corp_id or secret is empty")
	}

	url := fmt.Sprintf("https://qyapi.weixin.qq.com/cgi-bin/gettoken?corpid=%s&corpsecret=%s", w.config.CorpID, w.config.Secret)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return "", fmt.Errorf("get access token failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	var result struct {
		ErrCode     int    `json:"errcode"`
		ErrMsg      string `json:"errmsg"`
		AccessToken string `json:"access_token"`
	}

	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("parse access token response failed: %w", err)
	}

	if result.ErrCode != 0 {
		return "", fmt.Errorf("get access token failed: code=%d, msg=%s", result.ErrCode, result.ErrMsg)
	}

	return result.AccessToken, nil
}

// SendAppMessage 发送应用消息
func (w *WechatWorkNotifier) SendAppMessage(userID, title, content string, event models.Event) error {
	accessToken, err := w.GetAccessToken()
	if err != nil {
		return err
	}

	// 构建消息
	msg := map[string]interface{}{
		"touser":  userID,
		"msgtype": "markdown",
		"agentid": w.config.AgentID,
		"markdown": map[string]interface{}{
			"content": w.formatMarkdown(title, content, event),
		},
	}

	url := fmt.Sprintf("https://qyapi.weixin.qq.com/cgi-bin/message/send?access_token=%s", accessToken)
	bodyBytes, _ := json.Marshal(msg)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Post(url, "application/json", bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("send app message failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	var result struct {
		ErrCode int    `json:"errcode"`
		ErrMsg  string `json:"errmsg"`
	}

	if err := json.Unmarshal(respBody, &result); err != nil {
		return fmt.Errorf("parse response failed: %w", err)
	}

	if result.ErrCode != 0 {
		return fmt.Errorf("send app message failed: code=%d, msg=%s", result.ErrCode, result.ErrMsg)
	}

	return nil
}

// SendToAllUsers 发送消息给所有配置的用户
func (w *WechatWorkNotifier) SendToAllUsers(title, content string, event models.Event) error {
	if len(w.config.UserIDs) == 0 {
		return fmt.Errorf("no user ids configured")
	}

	// 企业微信支持批量发送，用 | 分隔
	userIDList := ""
	for i, uid := range w.config.UserIDs {
		if i > 0 {
			userIDList += "|"
		}
		userIDList += uid
	}

	return w.SendAppMessage(userIDList, title, content, event)
}
