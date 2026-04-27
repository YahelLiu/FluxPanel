package notify

import (
	"log"
	"sync"

	"client-monitor/models"
)

// Dispatcher 通知分发器
type Dispatcher struct {
	factory *NotifierFactory
	mu      sync.RWMutex
}

var dispatcher *Dispatcher
var dispatcherOnce sync.Once

// GetDispatcher 获取分发器单例
func GetDispatcher() *Dispatcher {
	dispatcherOnce.Do(func() {
		dispatcher = &Dispatcher{
			factory: GetFactory(),
		}
	})
	return dispatcher
}

// Dispatch 分发通知
func (d *Dispatcher) Dispatch(channelType models.NotificationType, config interface{}, title, content string, event models.Event) error {
	switch channelType {
	case models.NotificationTypeFeishu:
		if cfg, ok := config.(models.FeishuConfig); ok {
			return d.dispatchFeishu(cfg, title, content, event)
		}
	case models.NotificationTypeWechatWork:
		if cfg, ok := config.(models.WechatWorkConfig); ok {
			return d.dispatchWechatWork(cfg, title, content, event)
		}
	}
	return nil
}

func (d *Dispatcher) dispatchFeishu(config models.FeishuConfig, title, content string, event models.Event) error {
	notifier := d.factory.GetFeishu(config)

	if config.WebhookURL != "" {
		return notifier.Send(title, content, event)
	}
	if len(config.UserIDs) > 0 {
		return notifier.SendToAllUsers(title, content, event)
	}

	return nil
}

func (d *Dispatcher) dispatchWechatWork(config models.WechatWorkConfig, title, content string, event models.Event) error {
	notifier := d.factory.GetWechatWork(config)

	if config.WebhookURL != "" {
		return notifier.Send(title, content, event)
	}
	if len(config.UserIDs) > 0 {
		return notifier.SendToAllUsers(title, content, event)
	}

	return nil
}

// Service 通知服务
type Service struct {
	dispatcher *Dispatcher
}

var service *Service
var serviceOnce sync.Once

// GetService 获取服务单例
func GetService() *Service {
	serviceOnce.Do(func() {
		service = &Service{
			dispatcher: GetDispatcher(),
		}
	})
	return service
}

// SendNotification 发送通知（兼容旧接口）
func (s *Service) SendNotification(channel *models.NotificationChannel, event models.Event) error {
	log.Printf("SendNotification: client=%s type=%s status=%s", event.ClientID, event.EventType, event.Status)
	return nil
}

// SendAlertNotification 发送告警通知
func (s *Service) SendAlertNotification(channel *models.NotificationChannel, title, content string, event models.Event) error {
	switch channel.Type {
	case models.NotificationTypeFeishu:
		return s.dispatcher.dispatchFeishu(channel.Feishu, title, content, event)
	case models.NotificationTypeWechatWork:
		return s.dispatcher.dispatchWechatWork(channel.WechatWork, title, content, event)
	}
	return nil
}

// ClearCache 清除缓存
func (s *Service) ClearCache() {
	s.dispatcher.factory.ClearCache()
}
