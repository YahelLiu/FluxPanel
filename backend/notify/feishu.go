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
	client *http.Client
}

// NewFeishuNotifier 创建飞书通知器
func NewFeishuNotifier(config models.FeishuConfig) *FeishuNotifier {
	return &FeishuNotifier{
		config: config,
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

// Send 发送通知
func (f *FeishuNotifier) Send(title, content string, event models.Event) error {
	if f.config.WebhookURL == "" {
		return fmt.Errorf("feishu webhook URL is empty")
	}
	return f.sendWebhook(title, content, event)
}

func (f *FeishuNotifier) sendWebhook(title, content string, event models.Event) error {
	card := f.buildCard(title, content, event)
	msg := map[string]interface{}{
		"msg_type": "interactive",
		"card":     card,
	}
	return f.sendRequest(msg)
}

// SendWithSign 发送带签名的消息
func (f *FeishuNotifier) SendWithSign(title, content string, event models.Event, timestamp int64, secret string) error {
	stringToSign := fmt.Sprintf("%d\n%s", timestamp, secret)
	h := hmac.New(sha256.New, []byte(stringToSign))
	h.Write([]byte(""))
	signature := base64.StdEncoding.EncodeToString(h.Sum(nil))

	card := f.buildCard(title, content, event)
	msg := map[string]interface{}{
		"msg_type":  "interactive",
		"card":      card,
		"timestamp": timestamp,
		"sign":      signature,
	}
	return f.sendRequest(msg)
}

func (f *FeishuNotifier) buildCard(title, content string, event models.Event) map[string]interface{} {
	template := "green"
	emoji := "✅"
	if event.Status == "error" {
		template = "red"
		emoji = "🚨"
	} else if event.Status == "warning" {
		template = "yellow"
		emoji = "⚠️"
	}

	return map[string]interface{}{
		"config": map[string]interface{}{
			"wide_screen_mode": true,
		},
		"header": map[string]interface{}{
			"title": map[string]interface{}{
				"tag":     "plain_text",
				"content": emoji + " " + title,
			},
			"template": template,
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
}

func (f *FeishuNotifier) sendRequest(msg interface{}) error {
	body, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal message failed: %w", err)
	}

	resp, err := f.client.Post(f.config.WebhookURL, "application/json", jsonReader(body))
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

	return nil
}

// GetAccessToken 获取访问令牌
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
	resp, err := f.client.Post(url, "application/json", jsonReader(bodyBytes))
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

// SendAppMessage 发送应用消息
func (f *FeishuNotifier) SendAppMessage(userID, title, content string, event models.Event) error {
	accessToken, err := f.GetAccessToken()
	if err != nil {
		return err
	}

	card := f.buildCard(title, content, event)
	msg := map[string]interface{}{
		"receive_id_type": "user_id",
		"content":         string(mustJSON(card)),
		"msg_type":        "interactive",
	}

	url := fmt.Sprintf("https://open.feishu.cn/open-apis/im/v1/messages?receive_id_type=user_id&receive_id=%s", userID)
	bodyBytes, _ := json.Marshal(msg)

	req, _ := http.NewRequest("POST", url, jsonReader(bodyBytes))
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := f.client.Do(req)
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

// SendToAllUsers 发送消息给所有用户
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

func jsonReader(data []byte) *bytes.Reader {
	return bytes.NewReader(data)
}

func mustJSON(v interface{}) []byte {
	b, _ := json.Marshal(v)
	return b
}
