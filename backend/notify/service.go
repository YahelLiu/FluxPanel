package notify

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"client-monitor/database"
	"client-monitor/models"
)

// Service 通知服务
type Service struct {
	feishuCache     map[uint]*FeishuNotifier
	wechatCache     map[uint]*WechatWorkNotifier
	accessTokenMux  sync.RWMutex
	feishuTokenMux  sync.RWMutex
}

var (
	service     *Service
	serviceOnce sync.Once
)

// GetService 获取通知服务单例
func GetService() *Service {
	serviceOnce.Do(func() {
		service = &Service{
			feishuCache: make(map[uint]*FeishuNotifier),
			wechatCache: make(map[uint]*WechatWorkNotifier),
		}
	})
	return service
}

// SendNotification 发送通知
func (s *Service) SendNotification(channel *models.NotificationChannel, event models.Event) error {
	if !channel.Enabled {
		return fmt.Errorf("channel %s is disabled", channel.Name)
	}

	// 检查触发条件
	if !s.shouldTrigger(channel.Trigger, event.Status) {
		return nil // 不满足触发条件，不发送
	}

	title := fmt.Sprintf("[%s] %s", event.Status, event.EventType)
	content := ""
	if event.Data != nil {
		var data map[string]interface{}
		if err := json.Unmarshal(event.Data, &data); err == nil {
			if msg, ok := data["message"].(string); ok {
				content = msg
			} else {
				contentBytes, _ := json.MarshalIndent(data, "", "  ")
				content = string(contentBytes)
			}
		}
	}

	var err error
	switch channel.Type {
	case models.NotificationTypeFeishu:
		err = s.sendFeishu(channel, title, content, event)
	case models.NotificationTypeWechatWork:
		err = s.sendWechatWork(channel, title, content, event)
	default:
		err = fmt.Errorf("unsupported notification type: %s", channel.Type)
	}

	// 记录日志
	s.logNotification(channel.ID, event.ID, err)

	return err
}

// SendAlertNotification 发送告警通知（使用自定义标题和内容）
func (s *Service) SendAlertNotification(channel *models.NotificationChannel, title, content string, event models.Event) error {
	if !channel.Enabled {
		return fmt.Errorf("channel %s is disabled", channel.Name)
	}

	var err error
	switch channel.Type {
	case models.NotificationTypeFeishu:
		err = s.sendFeishu(channel, title, content, event)
	case models.NotificationTypeWechatWork:
		err = s.sendWechatWork(channel, title, content, event)
	default:
		err = fmt.Errorf("unsupported notification type: %s", channel.Type)
	}

	// 记录日志
	s.logNotification(channel.ID, event.ID, err)

	return err
}

func (s *Service) shouldTrigger(trigger models.TriggerCondition, status string) bool {
	switch trigger {
	case models.TriggerOnAll:
		return true
	case models.TriggerOnError:
		return status == "error"
	case models.TriggerOnWarning:
		return status == "warning" || status == "error"
	case models.TriggerOnCustom:
		return true // 自定义规则在规则层面处理
	default:
		return status == "error"
	}
}

func (s *Service) sendFeishu(channel *models.NotificationChannel, title, content string, event models.Event) error {
	notifier := s.getFeishuNotifier(channel)

	switch channel.Mode {
	case models.NotificationModeWebhook:
		return notifier.SendWebhook(title, content, event)
	case models.NotificationModeApp:
		return notifier.SendToAllUsers(title, content, event)
	default:
		// 默认使用 Webhook
		if channel.Feishu.WebhookURL != "" {
			return notifier.SendWebhook(title, content, event)
		}
		return notifier.SendToAllUsers(title, content, event)
	}
}

func (s *Service) sendWechatWork(channel *models.NotificationChannel, title, content string, event models.Event) error {
	notifier := s.getWechatWorkNotifier(channel)

	switch channel.Mode {
	case models.NotificationModeWebhook:
		return notifier.SendWebhook(title, content, event)
	case models.NotificationModeApp:
		return notifier.SendToAllUsers(title, content, event)
	default:
		// 默认使用 Webhook
		if channel.WechatWork.WebhookURL != "" {
			return notifier.SendWebhook(title, content, event)
		}
		return notifier.SendToAllUsers(title, content, event)
	}
}

func (s *Service) getFeishuNotifier(channel *models.NotificationChannel) *FeishuNotifier {
	s.feishuTokenMux.RLock()
	notifier, ok := s.feishuCache[channel.ID]
	s.feishuTokenMux.RUnlock()

	if ok {
		return notifier
	}

	notifier = NewFeishuNotifier(channel.Feishu)
	s.feishuTokenMux.Lock()
	s.feishuCache[channel.ID] = notifier
	s.feishuTokenMux.Unlock()

	return notifier
}

func (s *Service) getWechatWorkNotifier(channel *models.NotificationChannel) *WechatWorkNotifier {
	s.accessTokenMux.RLock()
	notifier, ok := s.wechatCache[channel.ID]
	s.accessTokenMux.RUnlock()

	if ok {
		return notifier
	}

	notifier = NewWechatWorkNotifier(channel.WechatWork)
	s.accessTokenMux.Lock()
	s.wechatCache[channel.ID] = notifier
	s.accessTokenMux.Unlock()

	return notifier
}

func (s *Service) logNotification(channelID, eventID uint, err error) {
	logEntry := models.NotificationLog{
		ChannelID: channelID,
		EventID:   eventID,
		CreatedAt: time.Now(),
	}

	if err != nil {
		logEntry.Status = "failed"
		logEntry.Error = err.Error()
	} else {
		logEntry.Status = "success"
	}

	if dbErr := database.DB.Create(&logEntry).Error; dbErr != nil {
		log.Printf("Failed to save notification log: %v", dbErr)
	}
}

// NotifyEvent 根据规则发送事件通知
func (s *Service) NotifyEvent(event models.Event) {
	// 获取所有启用的通知渠道
	var channels []models.NotificationChannel
	database.DB.Where("enabled = ?", true).Find(&channels)

	// 获取所有启用的规则
	var rules []models.NotificationRule
	database.DB.Where("enabled = ?", true).Find(&rules)

	// 根据规则匹配并发送通知
	for _, rule := range rules {
		if !s.matchRule(rule, event) {
			continue
		}

		// 获取规则关联的渠道
		var ruleChannels []models.NotificationChannel
		if len(rule.ChannelIDs) > 0 {
			database.DB.Where("id IN ? AND enabled = ?", rule.ChannelIDs, true).Find(&ruleChannels)
		}

		for _, channel := range ruleChannels {
			go func(ch models.NotificationChannel, ev models.Event) {
				if err := s.SendNotification(&ch, ev); err != nil {
					log.Printf("Failed to send notification via %s: %v", ch.Name, err)
				}
			}(channel, event)
		}
	}

	// 如果没有规则，直接使用渠道的触发条件
	if len(rules) == 0 {
		for _, channel := range channels {
			go func(ch models.NotificationChannel, ev models.Event) {
				if err := s.SendNotification(&ch, ev); err != nil {
					log.Printf("Failed to send notification via %s: %v", ch.Name, err)
				}
			}(channel, event)
		}
	}
}

func (s *Service) matchRule(rule models.NotificationRule, event models.Event) bool {
	// 检查事件类型
	if len(rule.EventTypes) > 0 {
		matched := false
		for _, et := range rule.EventTypes {
			if et == event.EventType {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	// 检查状态
	if len(rule.StatusFilter) > 0 {
		matched := false
		for _, st := range rule.StatusFilter {
			if st == event.Status {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	// 检查客户端
	if len(rule.ClientIDs) > 0 {
		matched := false
		for _, cid := range rule.ClientIDs {
			if cid == event.ClientID {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	return true
}

// ClearCache 清除缓存（配置更新后调用）
func (s *Service) ClearCache(channelID uint) {
	s.feishuTokenMux.Lock()
	delete(s.feishuCache, channelID)
	s.feishuTokenMux.Unlock()

	s.accessTokenMux.Lock()
	delete(s.wechatCache, channelID)
	s.accessTokenMux.Unlock()
}
