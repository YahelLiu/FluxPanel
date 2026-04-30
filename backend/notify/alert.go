package notify

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"client-monitor/database"
	"client-monitor/models"
	"client-monitor/notify/types"
)

// AlertService 告警服务
type AlertService struct {
	// 记录客户端最近一次的指标值，用于检测阈值
	metricCache map[string]map[string]float64 // clientID -> metricType -> value
	cacheMux    sync.RWMutex
}

var (
	alertService     *AlertService
	alertServiceOnce sync.Once
)

// GetAlertService 获取告警服务单例
func GetAlertService() *AlertService {
	alertServiceOnce.Do(func() {
		alertService = &AlertService{
			metricCache: make(map[string]map[string]float64),
		}
	})
	return alertService
}

// CheckEvent 检查事件是否触发告警
func (a *AlertService) CheckEvent(event models.Event) {
	if event.Data == nil {
		return
	}

	var data map[string]interface{}
	if err := json.Unmarshal(event.Data, &data); err != nil {
		return
	}

	// 获取所有启用的告警阈值
	var thresholds []models.AlertThreshold
	database.DB.Where("enabled = ?", true).Find(&thresholds)

	for _, threshold := range thresholds {
		value := a.extractMetricValue(data, threshold.MetricType)
		if value == nil {
			continue
		}

		// 检查是否触发阈值
		triggered := a.checkThreshold(*value, threshold.Operator, threshold.Threshold)

		// 记录指标值
		a.recordMetric(event.ClientID, threshold.MetricType, *value)

		// 检查是否需要发送告警
		if triggered {
			a.handleAlertTriggered(threshold, event, *value)
		}
	}
}

// extractMetricValue 从事件数据中提取指标值
func (a *AlertService) extractMetricValue(data map[string]interface{}, metricType string) *float64 {
	switch metricType {
	case "cpu":
		if cpu, ok := data["cpu"].(map[string]interface{}); ok {
			if load, ok := cpu["load_percent"].(float64); ok {
				return &load
			}
		}
	case "memory":
		if mem, ok := data["memory"].(map[string]interface{}); ok {
			if load, ok := mem["load_percent"].(float64); ok {
				return &load
			}
		}
	case "disk":
		// 检查所有硬盘，返回最高使用率
		if disks, ok := data["disks"].([]interface{}); ok {
			var maxLoad float64
			for _, d := range disks {
				if disk, ok := d.(map[string]interface{}); ok {
					if load, ok := disk["load_percent"].(float64); ok {
						if load > maxLoad {
							maxLoad = load
						}
					}
				}
			}
			if maxLoad > 0 {
				return &maxLoad
			}
		}
	}
	return nil
}

// checkThreshold 检查是否触发阈值
func (a *AlertService) checkThreshold(value float64, operator string, threshold float64) bool {
	switch operator {
	case ">":
		return value > threshold
	case ">=":
		return value >= threshold
	case "<":
		return value < threshold
	case "<=":
		return value <= threshold
	default:
		return false
	}
}

// handleAlertTriggered 处理告警触发
func (a *AlertService) handleAlertTriggered(threshold models.AlertThreshold, event models.Event, value float64) {
	// 检查是否已有未解决的告警
	var existingAlert models.AlertRecord
	err := database.DB.Where(
		"threshold_id = ? AND client_id = ? AND status = ?",
		threshold.ID, event.ClientID, "triggered",
	).First(&existingAlert).Error

	if err == nil {
		// 已存在未解决的告警，更新指标值
		existingAlert.MetricValue = value
		database.DB.Save(&existingAlert)
		return
	}

	// 创建新告警记录
	alertRecord := models.AlertRecord{
		ThresholdID: threshold.ID,
		ClientID:    event.ClientID,
		MetricType:  threshold.MetricType,
		MetricValue: value,
		Threshold:   threshold.Threshold,
		Status:      "triggered",
		Notified:    false,
		CreatedAt:   time.Now(),
	}

	if err := database.DB.Create(&alertRecord).Error; err != nil {
		log.Printf("Failed to create alert record: %v", err)
		return
	}

	// 发送通知
	if len(threshold.ChannelIDs) > 0 {
		a.sendAlertNotification(threshold, event, value)
		alertRecord.Notified = true
		database.DB.Save(&alertRecord)
	}
}

// sendAlertNotification 发送告警通知
func (a *AlertService) sendAlertNotification(threshold models.AlertThreshold, event models.Event, value float64) {
	metricName := a.getMetricName(threshold.MetricType)
	title := threshold.Name
	content := fmt.Sprintf(
		"%s 超过阈值\n\n当前值: %.1f%%\n阈值: %.1f%%",
		metricName, value, threshold.Threshold,
	)

	// 使用统一的通知服务发送告警
	if err := GetNotifyService().SendAlert(title, content, types.PriorityHigh); err != nil {
		log.Printf("[alert] Failed to send alert notification: %v", err)
	}
}

// getMetricName 获取指标名称
func (a *AlertService) getMetricName(metricType string) string {
	switch metricType {
	case "cpu":
		return "CPU 使用率"
	case "memory":
		return "内存使用率"
	case "disk":
		return "硬盘使用率"
	default:
		return metricType
	}
}

// recordMetric 记录指标值
func (a *AlertService) recordMetric(clientID, metricType string, value float64) {
	a.cacheMux.Lock()
	defer a.cacheMux.Unlock()

	if a.metricCache[clientID] == nil {
		a.metricCache[clientID] = make(map[string]float64)
	}
	a.metricCache[clientID][metricType] = value
}

// GetActiveAlerts 获取活跃告警
func (a *AlertService) GetActiveAlerts() []models.AlertRecord {
	var alerts []models.AlertRecord
	database.DB.Where("status = ?", "triggered").Order("created_at DESC").Find(&alerts)
	return alerts
}

// ResolveAlert 解决告警
func (a *AlertService) ResolveAlert(alertID uint) error {
	var alert models.AlertRecord
	if err := database.DB.First(&alert, alertID).Error; err != nil {
		return err
	}

	now := time.Now()
	alert.Status = "resolved"
	alert.ResolvedAt = &now
	return database.DB.Save(&alert).Error
}
