package notify

import (
	"log"
	"sync"

	"client-monitor/database"
	"client-monitor/models"
	"client-monitor/notify/drivers"
	"client-monitor/notify/types"
)

// ============================================
// 统一通知服务（基于适配器架构）
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

// SendWeatherToChannels 发送天气通知到指定渠道
func (s *NotifyService) SendWeatherToChannels(location, content string, channelIDs []int) []error {
	log.Printf("[notify] SendWeatherToChannels called, channelIDs: %v", channelIDs)

	msg := types.NewNotifyMessage(types.MessageTypeWeather, "天气预报", content).
		WithSource(location, location)

	var errs []error

	for _, channelID := range channelIDs {
		// 查询渠道信息
		var channel models.NotificationChannel
		if err := database.DB.First(&channel, channelID).Error; err != nil {
			log.Printf("[notify] Channel %d not found: %v", channelID, err)
			errs = append(errs, err)
			continue
		}

		log.Printf("[notify] Channel %d: type=%s, enabled=%v", channelID, channel.Type, channel.Enabled)

		if !channel.Enabled {
			log.Printf("[notify] Channel %d is disabled, skipping", channelID)
			continue
		}

		// 根据渠道类型确定驱动
		var driverName string
		switch channel.Type {
		case models.NotificationTypeWechatILink:
			driverName = "ilink"
		case models.NotificationTypeFeishu:
			driverName = "feishu"
		default:
			log.Printf("[notify] Channel %d has unknown type: %s", channelID, channel.Type)
			continue
		}

		log.Printf("[notify] Sending to driver: %s", driverName)
		if err := s.SendTo(driverName, msg); err != nil {
			log.Printf("[notify] Send to %s failed: %v", driverName, err)
			errs = append(errs, err)
		} else {
			log.Printf("[notify] Send to %s success", driverName)
		}
	}

	return errs
}
