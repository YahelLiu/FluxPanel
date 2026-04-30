package notify

import (
	"context"
	"log"
	"sync"

	"client-monitor/messaging"
	"client-monitor/models"
	"client-monitor/notify/drivers"
	"client-monitor/notify/types"
	"client-monitor/wecom"
)

// Dispatcher 通知分发器（保留兼容旧接口）
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
	case models.NotificationTypeWechatILink:
		if cfg, ok := config.(models.WechatILinkConfig); ok {
			return d.dispatchWechatILink(cfg, title, content, event)
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

func (d *Dispatcher) dispatchWechatILink(config models.WechatILinkConfig, title, content string, event models.Event) error {
	if !config.LoggedIn || config.BotToken == "" {
		log.Printf("[notify] WechatILink not logged in")
		return nil
	}

	client := wecom.GetClient()
	if client == nil {
		log.Printf("[notify] WechatILink client not available")
		return nil
	}

	// 如果没有配置 UserIDs，使用当前登录的 iLink 用户 ID
	userIDs := config.UserIDs
	if len(userIDs) == 0 && config.ILinkUserID != "" {
		userIDs = []string{config.ILinkUserID}
	}

	if len(userIDs) == 0 {
		log.Printf("[notify] WechatILink no target users configured")
		return nil
	}

	// 构建消息内容（包含标题）
	fullContent := title + "\n\n" + content

	// 发送给配置的用户列表
	for _, userID := range userIDs {
		if err := messaging.SendTextReply(context.Background(), client, userID, fullContent, "", ""); err != nil {
			log.Printf("[notify] WechatILink send to %s failed: %v", userID, err)
		} else {
			log.Printf("[notify] WechatILink sent to %s successfully", userID)
		}
	}

	return nil
}

// Service 通知服务（旧接口，保留兼容）
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
	case models.NotificationTypeWechatILink:
		return s.dispatcher.dispatchWechatILink(channel.WechatILink, title, content, event)
	}
	return nil
}

// ClearCache 清除缓存
func (s *Service) ClearCache() {
	s.dispatcher.factory.ClearCache()
}

// ============================================
// 新的统一通知服务（基于适配器架构）
// ============================================

// NotifyService 通知服务（对外暴露的统一入口）
type NotifyService struct {
	router *Router
}

var notifyService *NotifyService
var notifyServiceOnce sync.Once

// GetNotifyService 获取通知服务
func GetNotifyService() *NotifyService {
	notifyServiceOnce.Do(func() {
		notifyService = &NotifyService{
			router: GetRouter(),
		}
	})
	return notifyService
}

// Send 发送通知（自动路由到可用渠道）
func (s *NotifyService) Send(msg *types.NotifyMessage) error {
	log.Printf("[notify] Sending %s message: %s", msg.Type, msg.Title)
	return s.router.Route(msg)
}

// SendAll 发送通知到所有可用渠道
func (s *NotifyService) SendAll(msg *types.NotifyMessage) []error {
	log.Printf("[notify] Sending %s message to all channels: %s", msg.Type, msg.Title)
	return s.router.RouteAll(msg)
}

// SendTo 指定渠道发送
func (s *NotifyService) SendTo(driverName string, msg *types.NotifyMessage) error {
	return s.router.RouteTo(driverName, msg)
}

// 便捷方法

// SendAlert 发送告警
func (s *NotifyService) SendAlert(title, content string, priority types.Priority) error {
	msg := types.NewNotifyMessage(types.MessageTypeAlert, title, content).
		WithPriority(priority)
	return s.Send(msg)
}

// SendWeather 发送天气
func (s *NotifyService) SendWeather(location, content string) error {
	msg := types.NewNotifyMessage(types.MessageTypeWeather, "天气预报", content).
		WithSource(location, location)
	return s.Send(msg)
}

// SendReminder 发送提醒
func (s *NotifyService) SendReminder(content string) error {
	msg := types.NewNotifyMessage(types.MessageTypeReminder, "提醒", content).
		WithPriority(types.PriorityHigh)
	return s.Send(msg)
}

// SendSystem 发送系统消息
func (s *NotifyService) SendSystem(title, content string) error {
	msg := types.NewNotifyMessage(types.MessageTypeSystem, title, content)
	return s.Send(msg)
}

// SendChat 发送聊天消息
func (s *NotifyService) SendChat(content string) error {
	msg := types.NewNotifyMessage(types.MessageTypeChat, "", content)
	return s.Send(msg)
}

// GetAvailableDrivers 获取可用驱动列表
func (s *NotifyService) GetAvailableDrivers() []drivers.DriverInfo {
	return s.router.GetAvailableDrivers()
}
