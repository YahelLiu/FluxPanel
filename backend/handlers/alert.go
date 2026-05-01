package handlers

import (
	"client-monitor/database"
	"client-monitor/models"
	"client-monitor/notify"
	"client-monitor/notify/types"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// --- 告警阈值管理 ---

// ListAlertThresholds GET /api/alerts/thresholds - 获取所有告警阈值
func ListAlertThresholds(c *gin.Context) {
	var thresholds []models.AlertThreshold
	database.DB.Order("created_at DESC").Find(&thresholds)

	c.JSON(http.StatusOK, thresholds)
}

// CreateAlertThreshold POST /api/alerts/thresholds - 创建告警阈值
func CreateAlertThreshold(c *gin.Context) {
	var req models.AlertThresholdRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	threshold := models.AlertThreshold{
		Name:        req.Name,
		MetricType:  req.MetricType,
		Operator:    req.Operator,
		Threshold:   req.Threshold,
		Duration:    req.Duration,
		ChannelIDs:  req.ChannelIDs,
		Enabled:     req.Enabled,
		Description: req.Description,
	}

	if err := database.DB.Create(&threshold).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create threshold"})
		return
	}

	c.JSON(http.StatusCreated, threshold)
}

// UpdateAlertThreshold PUT /api/alerts/thresholds/:id - 更新告警阈值
func UpdateAlertThreshold(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var threshold models.AlertThreshold
	if err := database.DB.First(&threshold, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "threshold not found"})
		return
	}

	var req models.AlertThresholdRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	threshold.Name = req.Name
	threshold.MetricType = req.MetricType
	threshold.Operator = req.Operator
	threshold.Threshold = req.Threshold
	threshold.Duration = req.Duration
	threshold.ChannelIDs = req.ChannelIDs
	threshold.Enabled = req.Enabled
	threshold.Description = req.Description

	if err := database.DB.Save(&threshold).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update threshold"})
		return
	}

	c.JSON(http.StatusOK, threshold)
}

// DeleteAlertThreshold DELETE /api/alerts/thresholds/:id - 删除告警阈值
func DeleteAlertThreshold(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	result := database.DB.Delete(&models.AlertThreshold{}, id)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete threshold"})
		return
	}

	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "threshold not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// ToggleAlertThreshold PUT /api/alerts/thresholds/:id/toggle - 切换告警阈值启用状态
func ToggleAlertThreshold(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var threshold models.AlertThreshold
	if err := database.DB.First(&threshold, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "threshold not found"})
		return
	}

	threshold.Enabled = !threshold.Enabled
	if err := database.DB.Save(&threshold).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to toggle threshold"})
		return
	}

	c.JSON(http.StatusOK, threshold)
}

// --- 告警记录 ---

// ListAlertRecords GET /api/alerts/records - 获取告警记录
func ListAlertRecords(c *gin.Context) {
	status := c.Query("status")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	if pageSize > 100 {
		pageSize = 100
	}

	query := database.DB.Model(&models.AlertRecord{})

	if status != "" {
		query = query.Where("status = ?", status)
	}

	var total int64
	query.Count(&total)

	var records []models.AlertRecord
	offset := (page - 1) * pageSize
	query.Order("created_at DESC").
		Offset(offset).
		Limit(pageSize).
		Find(&records)

	c.JSON(http.StatusOK, gin.H{
		"total":   total,
		"records": records,
	})
}

// GetActiveAlerts GET /api/alerts/active - 获取活跃告警
func GetActiveAlerts(c *gin.Context) {
	alerts := notify.GetAlertService().GetActiveAlerts()
	c.JSON(http.StatusOK, alerts)
}

// ResolveAlert PUT /api/alerts/records/:id/resolve - 解决告警
func ResolveAlert(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	if err := notify.GetAlertService().ResolveAlert(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to resolve alert"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// DeleteAlertRecord DELETE /api/alerts/records/:id - 删除告警记录
func DeleteAlertRecord(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	result := database.DB.Delete(&models.AlertRecord{}, id)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete alert record"})
		return
	}

	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "alert record not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// TestAlertThreshold POST /api/alerts/thresholds/:id/test - 测试告警阈值
func TestAlertThreshold(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var threshold models.AlertThreshold
	if err := database.DB.First(&threshold, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "threshold not found"})
		return
	}

	// 获取关联的通知渠道
	if len(threshold.ChannelIDs) == 0 {
		c.JSON(http.StatusOK, gin.H{"success": false, "error": "未配置通知渠道"})
		return
	}

	// 转换为 []interface{} 用于 IN 查询
	channelIDInts := make([]int, len(threshold.ChannelIDs))
	for i, v := range threshold.ChannelIDs {
		channelIDInts[i] = v
	}

	var channels []models.NotificationChannel
	database.DB.Where("id IN ? AND enabled = ?", channelIDInts, true).Find(&channels)

	if len(channels) == 0 {
		c.JSON(http.StatusOK, gin.H{"success": false, "error": "没有可用的通知渠道"})
		return
	}

	// 构建测试告警消息
	metricName := getMetricName(threshold.MetricType)
	title := fmt.Sprintf("🧪 测试告警: %s", threshold.Name)
	content := fmt.Sprintf(
		"**%s** 阈值测试\n\n当前模拟值: %.1f%%\n阈值: %.1f%%\n这是一条测试消息",
		metricName, threshold.Threshold+5, threshold.Threshold,
	)

	// 发送通知到用户选择的渠道
	msg := types.NewNotifyMessage(types.MessageTypeAlert, title, content).
		WithPriority(types.PriorityHigh)

	success := false
	var lastErr error
	for _, channel := range channels {
		// 根据渠道类型发送
		var driverName string
		switch channel.Type {
		case models.NotificationTypeWechatILink:
			driverName = "ilink"
		case models.NotificationTypeFeishu:
			driverName = "feishu"
		default:
			continue
		}

		if err := notify.GetNotifyService().SendTo(driverName, msg); err != nil {
			lastErr = err
		} else {
			success = true
		}
	}

	if success {
		c.JSON(http.StatusOK, gin.H{"success": true})
	} else if lastErr != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "error": lastErr.Error()})
	} else {
		c.JSON(http.StatusOK, gin.H{"success": false, "error": "没有可发送的渠道"})
	}
}

func getMetricName(metricType string) string {
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
