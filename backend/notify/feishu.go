package notify

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"client-monitor/models"
)

// FeishuNotifier 飞书通知器
type FeishuNotifier struct {
	config models.FeishuConfig
}

// NewFeishuNotifier 创建飞书通知器
func NewFeishuNotifier(config models.FeishuConfig) *FeishuNotifier {
	return &FeishuNotifier{config: config}
}

// FeishuWebhookMessage 飞书 Webhook 消息
type FeishuWebhookMessage struct {
	MsgType string                 `json:"msg_type"`
	Content map[string]interface{} `json:"content"`
}

// FeishuCardMessage 飞书卡片消息
type FeishuCardMessage struct {
	MsgType string `json:"msg_type"`
	Card    struct {
		Config   map[string]interface{}   `json:"config"`
		Elements []map[string]interface{} `json:"elements"`
	} `json:"card"`
}

// SendWebhook 发送 Webhook 消息
func (f *FeishuNotifier) SendWebhook(title, content string, event models.Event) error {
	if f.config.WebhookURL == "" {
		return fmt.Errorf("feishu webhook URL is empty")
	}

	// 构建卡片消息
	card := map[string]interface{}{
		"config": map[string]interface{}{
			"wide_screen_mode": true,
		},
		"elements": []map[string]interface{}{
			{
				"tag": "div",
				"text": map[string]interface{}{
					"tag":     "lark_md",
					"content": fmt.Sprintf("**%s**\n\n%s", title, content),
				},
			},
			{
				"tag": "div",
				"fields": []map[string]interface{}{
					{
						"is_short": true,
						"text": map[string]interface{}{
							"tag":     "lark_md",
							"content": fmt.Sprintf("**客户端:**\n%s", event.ClientID),
						},
					},
					{
						"is_short": true,
						"text": map[string]interface{}{
							"tag":     "lark_md",
							"content": fmt.Sprintf("**事件类型:**\n%s", event.EventType),
						},
					},
					{
						"is_short": true,
						"text": map[string]interface{}{
							"tag":     "lark_md",
							"content": fmt.Sprintf("**状态:**\n%s", event.Status),
						},
					},
					{
						"is_short": true,
						"text": map[string]interface{}{
							"tag":     "lark_md",
							"content": fmt.Sprintf("**时间:**\n%s", event.CreatedAt.Format("2006-01-02 15:04:05")),
						},
					},
				},
			},
			{
				"tag": "note",
				"elements": []map[string]interface{}{
					{
						"tag":     "plain_text",
						"content": "来自 FluxPanel 监控系统",
					},
				},
			},
		},
	}

	// 根据状态设置颜色
	if event.Status == "error" {
		card["header"] = map[string]interface{}{
			"title": map[string]interface{}{
				"tag":     "plain_text",
				"content": "🚨 " + title,
			},
			"template": "red",
		}
	} else if event.Status == "warning" {
		card["header"] = map[string]interface{}{
			"title": map[string]interface{}{
				"tag":     "plain_text",
				"content": "⚠️ " + title,
			},
			"template": "yellow",
		}
	} else {
		card["header"] = map[string]interface{}{
			"title": map[string]interface{}{
				"tag":     "plain_text",
				"content": "✅ " + title,
			},
			"template": "green",
		}
	}

	msg := map[string]interface{}{
		"msg_type": "interactive",
		"card":     card,
	}

	return f.sendRequest(msg)
}

// SendWebhookWithSign 发送带签名的 Webhook 消息
func (f *FeishuNotifier) SendWebhookWithSign(title, content string, event models.Event, timestamp int64, secret string) error {
	if f.config.WebhookURL == "" {
		return fmt.Errorf("feishu webhook URL is empty")
	}

	// 生成签名
	stringToSign := fmt.Sprintf("%d\n%s", timestamp, secret)
	h := hmac.New(sha256.New, []byte(stringToSign))
	h.Write([]byte(""))
	signature := base64.StdEncoding.EncodeToString(h.Sum(nil))

	// 构建消息
	card := f.buildCard(title, content, event)
	msg := map[string]interface{}{
		"msg_type": "interactive",
		"card":     card,
		"timestamp": timestamp,
		"sign":      signature,
	}

	return f.sendRequest(msg)
}

func (f *FeishuNotifier) buildCard(title, content string, event models.Event) map[string]interface{} {
	card := map[string]interface{}{
		"config": map[string]interface{}{
			"wide_screen_mode": true,
		},
		"elements": []map[string]interface{}{
			{
				"tag": "div",
				"text": map[string]interface{}{
					"tag":     "lark_md",
					"content": fmt.Sprintf("**%s**\n\n%s", title, content),
				},
			},
			{
				"tag": "div",
				"fields": []map[string]interface{}{
					{
						"is_short": true,
						"text": map[string]interface{}{
							"tag":     "lark_md",
							"content": fmt.Sprintf("**客户端:**\n%s", event.ClientID),
						},
					},
					{
						"is_short": true,
						"text": map[string]interface{}{
							"tag":     "lark_md",
							"content": fmt.Sprintf("**事件类型:**\n%s", event.EventType),
						},
					},
				},
			},
		},
	}

	// 设置颜色
	template := "green"
	emoji := "✅"
	if event.Status == "error" {
		template = "red"
		emoji = "🚨"
	} else if event.Status == "warning" {
		template = "yellow"
		emoji = "⚠️"
	}

	card["header"] = map[string]interface{}{
		"title": map[string]interface{}{
			"tag":     "plain_text",
			"content": emoji + " " + title,
		},
		"template": template,
	}

	return card
}

// sendRequest 发送 HTTP 请求
func (f *FeishuNotifier) sendRequest(msg interface{}) error {
	body, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal message failed: %w", err)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Post(f.config.WebhookURL, "application/json", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("send request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("feishu api error: status=%d, body=%s", resp.StatusCode, string(respBody))
	}

	// 检查响应
	var result struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
	}
	if err := json.Unmarshal(respBody, &result); err == nil {
		if result.Code != 0 {
			return fmt.Errorf("feishu api error: code=%d, msg=%s", result.Code, result.Msg)
		}
	}

	return nil
}

// --- 飞书应用消息 (需要 App ID 和 App Secret) ---

// FeishuAccessToken 飞书访问令牌响应
type FeishuAccessToken struct {
	AccessToken string `json:"app_access_token"`
	ExpireIn    int    `json:"expire"`
}

// GetAccessToken 获取飞书应用访问令牌
func (f *FeishuNotifier) GetAccessToken() (string, error) {
	if f.config.AppID == "" || f.config.AppSecret == "" {
		return "", fmt.Errorf("feishu app_id or app_secret is empty")
	}

	url := "https://open.feishu.cn/open-apis/auth/v3/app_access_token/internal"
	body := map[string]string{
		"app_id":     f.config.AppID,
		"app_secret": f.config.AppSecret,
	}

	bodyBytes, _ := json.Marshal(body)
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Post(url, "application/json", bytes.NewReader(bodyBytes))
	if err != nil {
		return "", fmt.Errorf("get access token failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	var result struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
		Data struct {
			AccessToken string `json:"app_access_token"`
		} `json:"data"`
	}

	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("parse access token response failed: %w", err)
	}

	if result.Code != 0 {
		return "", fmt.Errorf("get access token failed: code=%d, msg=%s", result.Code, result.Msg)
	}

	return result.Data.AccessToken, nil
}

// SendAppMessage 发送应用消息（单聊）
func (f *FeishuNotifier) SendAppMessage(userID, title, content string, event models.Event) error {
	accessToken, err := f.GetAccessToken()
	if err != nil {
		return err
	}

	// 构建消息
	card := f.buildCard(title, content, event)
	msg := map[string]interface{}{
		"receive_id_type": "user_id",
		"content":         string(mustJSON(card)),
		"msg_type":        "interactive",
	}

	url := fmt.Sprintf("https://open.feishu.cn/open-apis/im/v1/messages?receive_id_type=user_id&receive_id=%s", userID)
	bodyBytes, _ := json.Marshal(msg)

	req, _ := http.NewRequest("POST", url, bytes.NewReader(bodyBytes))
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("send app message failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	var result struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
	}

	if err := json.Unmarshal(respBody, &result); err != nil {
		return fmt.Errorf("parse response failed: %w", err)
	}

	if result.Code != 0 {
		return fmt.Errorf("send app message failed: code=%d, msg=%s", result.Code, result.Msg)
	}

	return nil
}

// SendToAllUsers 发送消息给所有配置的用户
func (f *FeishuNotifier) SendToAllUsers(title, content string, event models.Event) error {
	if len(f.config.UserIDs) == 0 {
		return fmt.Errorf("no user ids configured")
	}

	var lastErr error
	for _, userID := range f.config.UserIDs {
		if err := f.SendAppMessage(userID, title, content, event); err != nil {
			lastErr = err
		}
	}

	return lastErr
}

func mustJSON(v interface{}) []byte {
	b, _ := json.Marshal(v)
	return b
}
