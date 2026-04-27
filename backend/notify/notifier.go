package notify

import "client-monitor/models"

// Notifier 通知器接口
type Notifier interface {
	Send(title, content string, event models.Event) error
}
