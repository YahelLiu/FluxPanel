package wecom

import (
	"context"
	"errors"
	"sync"

	"client-monitor/database"
	"client-monitor/ilink"
	"client-monitor/messaging"
	"client-monitor/models"
)

var (
	client     *ilink.Client
	clientOnce sync.Once
	clientMu   sync.RWMutex
)

// ErrNotLoggedIn 未登录错误
var ErrNotLoggedIn = errors.New("wecom: not logged in")

// GetClient 获取 iLink 客户端单例
func GetClient() *ilink.Client {
	clientMu.RLock()
	if client != nil {
		clientMu.RUnlock()
		return client
	}
	clientMu.RUnlock()

	clientMu.Lock()
	defer clientMu.Unlock()

	// 双重检查
	if client != nil {
		return client
	}

	creds, err := loadCredentials()
	if err != nil {
		return nil
	}

	client = ilink.NewClient(creds)
	return client
}

// ResetClient 重置客户端（用于重新登录后）
func ResetClient() {
	clientMu.Lock()
	defer clientMu.Unlock()
	client = nil
}

// loadCredentials 从通知渠道加载凭证
func loadCredentials() (*ilink.Credentials, error) {
	var channel models.NotificationChannel
	err := database.DB.Where("type = ? AND enabled = ?", models.NotificationTypeWechatILink, true).First(&channel).Error
	if err != nil {
		return nil, err
	}
	if !channel.WechatILink.LoggedIn {
		return nil, errors.New("wechat ilink not logged in")
	}
	return &ilink.Credentials{
		BotToken:    channel.WechatILink.BotToken,
		ILinkBotID:  channel.WechatILink.ILinkBotID,
		BaseURL:     channel.WechatILink.BaseURL,
		ILinkUserID: channel.WechatILink.ILinkUserID,
	}, nil
}

// SendTestMessage 发送测试消息
func SendTestMessage(userID, content string) error {
	c := GetClient()
	if c == nil {
		return ErrNotLoggedIn
	}
	return messaging.SendTextReply(context.Background(), c, userID, content, "", "")
}

// GetILinkUserID 获取当前登录用户的 iLink UserID
func GetILinkUserID() string {
	// 先尝试从内存中的 client 获取
	clientMu.RLock()
	if client != nil {
		userID := client.ILinkUserID()
		clientMu.RUnlock()
		if userID != "" {
			return userID
		}
	} else {
		clientMu.RUnlock()
	}

	// 回退到数据库查询
	var channel models.NotificationChannel
	err := database.DB.Where("type = ? AND enabled = ?", models.NotificationTypeWechatILink, true).First(&channel).Error
	if err != nil {
		return ""
	}
	return channel.WechatILink.ILinkUserID
}

// HasWechatILinkChannel 检查是否存在已登录的微信 iLink 通知渠道
func HasWechatILinkChannel() bool {
	var channel models.NotificationChannel
	err := database.DB.Where("type = ? AND enabled = ?", models.NotificationTypeWechatILink, true).First(&channel).Error
	if err != nil {
		return false
	}
	return channel.WechatILink.LoggedIn
}

// HasCredentials 检查是否已登录（兼容旧代码）
func HasCredentials() bool {
	return HasWechatILinkChannel()
}

// GetBotID 获取当前 Bot ID
func GetBotID() string {
	c := GetClient()
	if c == nil {
		return ""
	}
	return c.BotID()
}
