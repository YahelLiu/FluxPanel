package notify

import (
	"log"
	"sync"
	"time"

	"client-monitor/database"
	"client-monitor/models"
)

// ReminderService 提醒服务
type ReminderService struct {
	stopChan chan struct{}
	mux      sync.RWMutex
}

var (
	reminderService     *ReminderService
	reminderServiceOnce sync.Once
)

// GetReminderService 获取提醒服务单例
func GetReminderService() *ReminderService {
	reminderServiceOnce.Do(func() {
		reminderService = &ReminderService{
			stopChan: make(chan struct{}),
		}
	})
	return reminderService
}

// Start 启动提醒服务
func (r *ReminderService) Start() {
	go r.scheduler()
	log.Println("Reminder service started")
}

// Stop 停止提醒服务
func (r *ReminderService) Stop() {
	close(r.stopChan)
	log.Println("Reminder service stopped")
}

// scheduler 调度器
func (r *ReminderService) scheduler() {
	// 初始检查，延迟 5 秒后开始
	time.Sleep(5 * time.Second)
	r.checkAndSend() // 立即检查一次

	ticker := time.NewTicker(10 * time.Second) // 每10秒检查一次
	defer ticker.Stop()

	for {
		select {
		case <-r.stopChan:
			return
		case <-ticker.C:
			r.checkAndSend()
		}
	}
}

// checkAndSend 检查并发送提醒
func (r *ReminderService) checkAndSend() {
	now := time.Now()

	// 查找到期的提醒
	var reminders []models.Reminder
	if err := database.DB.Where("remind_at <= ? AND sent = ?", now, false).Find(&reminders).Error; err != nil {
		log.Printf("查询提醒失败: %v", err)
		return
	}

	if len(reminders) == 0 {
		return
	}

	log.Printf("发现 %d 条待发送提醒", len(reminders))

	for _, reminder := range reminders {
		if err := r.sendReminder(&reminder); err != nil {
			log.Printf("发送提醒失败 (ID=%d): %v", reminder.ID, err)
		} else {
			// 标记为已发送
			database.DB.Model(&reminder).Update("sent", true)
			log.Printf("提醒已发送 (ID=%d): %s", reminder.ID, reminder.Content)
		}
	}
}

// sendReminder 发送提醒
func (r *ReminderService) sendReminder(reminder *models.Reminder) error {
	// 先尝试通过 WebSocket 发送（用于测试环境）
	if sendReminderCallback != nil {
		sendReminderCallback(reminder)
	}

	// 使用统一的通知服务发送提醒
	if err := GetNotifyService().SendReminder(reminder.Content); err != nil {
		log.Printf("[reminder] Failed to send reminder: %v", err)
		return err
	}

	return nil
}

// sendReminderCallback 发送提醒的回调函数（用于 WebSocket 推送）
var sendReminderCallback func(*models.Reminder)

// SetSendReminderCallback 设置发送提醒的回调
func SetSendReminderCallback(cb func(*models.Reminder)) {
	sendReminderCallback = cb
}

// SendReminderNow 立即发送提醒（用于测试）
func (r *ReminderService) SendReminderNow(reminderID uint) error {
	var reminder models.Reminder
	if err := database.DB.First(&reminder, reminderID).Error; err != nil {
		return err
	}

	if err := r.sendReminder(&reminder); err != nil {
		return err
	}

	// 标记为已发送
	database.DB.Model(&reminder).Update("sent", true)
	return nil
}
