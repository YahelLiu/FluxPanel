package notify

import (
	"context"
	"log"

	"client-monitor/messaging"
	"client-monitor/wecom"
)

// WeComiLinkNotifier 使用 iLink API 发送通知
type WeComiLinkNotifier struct{}

// NewWeComiLinkNotifier 创建 iLink 通知器
func NewWeComiLinkNotifier() *WeComiLinkNotifier {
	return &WeComiLinkNotifier{}
}

// Send 发送通知给指定用户
func (n *WeComiLinkNotifier) Send(userID, content string) error {
	client := wecom.GetClient()
	if client == nil {
		return wecom.ErrNotLoggedIn
	}

	return messaging.SendTextReply(context.Background(), client, userID, content, "", "")
}

// SendToAll 发送给所有配置的用户
func (n *WeComiLinkNotifier) SendToAll(userIDs []string, content string) error {
	client := wecom.GetClient()
	if client == nil {
		return wecom.ErrNotLoggedIn
	}

	for _, userID := range userIDs {
		if err := messaging.SendTextReply(context.Background(), client, userID, content, "", ""); err != nil {
			log.Printf("[wecom] Send to %s failed: %v", userID, err)
		}
	}
	return nil
}
