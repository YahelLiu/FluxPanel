package handlers

import (
	"client-monitor/database"
	"client-monitor/models"
	"client-monitor/notify"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// --- 通知渠道管理 ---

// ListChannels GET /api/notifications/channels - 获取所有通知渠道
func ListChannels(c *gin.Context) {
	var channels []models.NotificationChannel
	database.DB.Order("created_at DESC").Find(&channels)

	c.JSON(http.StatusOK, channels)
}

// GetChannel GET /api/notifications/channels/:id - 获取单个通知渠道
func GetChannel(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var channel models.NotificationChannel
	if err := database.DB.First(&channel, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "channel not found"})
		return
	}

	c.JSON(http.StatusOK, channel)
}

// CreateChannelRequest 创建渠道请求
type CreateChannelRequest struct {
	Name        string                   `json:"name" binding:"required"`
	Type        models.NotificationType  `json:"type" binding:"required"`
	Mode        models.NotificationMode  `json:"mode" binding:"required"`
	Enabled     bool                     `json:"enabled"`
	Trigger     models.TriggerCondition  `json:"trigger"`
	Feishu      models.FeishuConfig      `json:"feishu"`
	WechatWork  models.WechatWorkConfig  `json:"wechat_work"`
	Description string                   `json:"description"`
}

// CreateChannel POST /api/notifications/channels - 创建通知渠道
func CreateChannel(c *gin.Context) {
	var req CreateChannelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	channel := models.NotificationChannel{
		Name:        req.Name,
		Type:        req.Type,
		Mode:        req.Mode,
		Enabled:     req.Enabled,
		Trigger:     req.Trigger,
		Feishu:      req.Feishu,
		WechatWork:  req.WechatWork,
		Description: req.Description,
	}

	if channel.Trigger == "" {
		channel.Trigger = models.TriggerOnError
	}

	if err := database.DB.Create(&channel).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create channel"})
		return
	}

	c.JSON(http.StatusCreated, channel)
}

// UpdateChannelRequest 更新渠道请求
type UpdateChannelRequest struct {
	Name        string                   `json:"name"`
	Type        models.NotificationType  `json:"type"`
	Mode        models.NotificationMode  `json:"mode"`
	Enabled     *bool                    `json:"enabled"`
	Trigger     models.TriggerCondition  `json:"trigger"`
	Feishu      models.FeishuConfig      `json:"feishu"`
	WechatWork  models.WechatWorkConfig  `json:"wechat_work"`
	Description string                   `json:"description"`
}

// UpdateChannel PUT /api/notifications/channels/:id - 更新通知渠道
func UpdateChannel(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var channel models.NotificationChannel
	if err := database.DB.First(&channel, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "channel not found"})
		return
	}

	var req UpdateChannelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 更新字段
	if req.Name != "" {
		channel.Name = req.Name
	}
	if req.Type != "" {
		channel.Type = req.Type
	}
	if req.Mode != "" {
		channel.Mode = req.Mode
	}
	if req.Enabled != nil {
		channel.Enabled = *req.Enabled
	}
	if req.Trigger != "" {
		channel.Trigger = req.Trigger
	}
	channel.Feishu = req.Feishu
	channel.WechatWork = req.WechatWork
	channel.Description = req.Description

	if err := database.DB.Save(&channel).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update channel"})
		return
	}

	// 清除缓存
	notify.GetService().ClearCache()

	c.JSON(http.StatusOK, channel)
}

// DeleteChannel DELETE /api/notifications/channels/:id - 删除通知渠道
func DeleteChannel(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	result := database.DB.Delete(&models.NotificationChannel{}, id)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete channel"})
		return
	}

	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "channel not found"})
		return
	}

	// 清除缓存
	notify.GetService().ClearCache()

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// TestChannel POST /api/notifications/channels/:id/test - 测试通知渠道
func TestChannel(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var channel models.NotificationChannel
	if err := database.DB.First(&channel, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "channel not found"})
		return
	}

	// 创建测试事件
	testEvent := models.Event{
		ID:        0,
		ClientID:  "test-client",
		EventType: "test",
		Status:    "error",
		CreatedAt: channel.CreatedAt,
	}

	if err := notify.GetService().SendNotification(&channel, testEvent); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// --- 通知规则管理 ---

// ListRules GET /api/notifications/rules - 获取所有通知规则
func ListRules(c *gin.Context) {
	var rules []models.NotificationRule
	database.DB.Order("created_at DESC").Find(&rules)

	c.JSON(http.StatusOK, rules)
}

// CreateRuleRequest 创建规则请求
type CreateRuleRequest struct {
	Name         string           `json:"name" binding:"required"`
	ChannelIDs   models.IntArray  `json:"channel_ids"`
	EventTypes   models.StringArray `json:"event_types"`
	StatusFilter models.StringArray `json:"status_filter"`
	ClientIDs    models.StringArray `json:"client_ids"`
	Enabled      bool             `json:"enabled"`
}

// CreateRule POST /api/notifications/rules - 创建通知规则
func CreateRule(c *gin.Context) {
	var req CreateRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	rule := models.NotificationRule{
		Name:         req.Name,
		ChannelIDs:   req.ChannelIDs,
		EventTypes:   req.EventTypes,
		StatusFilter: req.StatusFilter,
		ClientIDs:    req.ClientIDs,
		Enabled:      req.Enabled,
	}

	if err := database.DB.Create(&rule).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create rule"})
		return
	}

	c.JSON(http.StatusCreated, rule)
}

// UpdateRuleRequest 更新规则请求
type UpdateRuleRequest struct {
	Name         string             `json:"name"`
	ChannelIDs   models.IntArray    `json:"channel_ids"`
	EventTypes   models.StringArray `json:"event_types"`
	StatusFilter models.StringArray `json:"status_filter"`
	ClientIDs    models.StringArray `json:"client_ids"`
	Enabled      *bool              `json:"enabled"`
}

// UpdateRule PUT /api/notifications/rules/:id - 更新通知规则
func UpdateRule(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var rule models.NotificationRule
	if err := database.DB.First(&rule, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "rule not found"})
		return
	}

	var req UpdateRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Name != "" {
		rule.Name = req.Name
	}
	rule.ChannelIDs = req.ChannelIDs
	rule.EventTypes = req.EventTypes
	rule.StatusFilter = req.StatusFilter
	rule.ClientIDs = req.ClientIDs
	if req.Enabled != nil {
		rule.Enabled = *req.Enabled
	}

	if err := database.DB.Save(&rule).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update rule"})
		return
	}

	c.JSON(http.StatusOK, rule)
}

// DeleteRule DELETE /api/notifications/rules/:id - 删除通知规则
func DeleteRule(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	result := database.DB.Delete(&models.NotificationRule{}, id)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete rule"})
		return
	}

	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "rule not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// --- 通知日志 ---

// ListLogs GET /api/notifications/logs - 获取通知日志
func ListLogs(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	channelID := c.Query("channel_id")

	if pageSize > 100 {
		pageSize = 100
	}

	query := database.DB.Model(&models.NotificationLog{})

	if channelID != "" {
		query = query.Where("channel_id = ?", channelID)
	}

	var total int64
	query.Count(&total)

	var logs []models.NotificationLog
	offset := (page - 1) * pageSize
	query.Order("created_at DESC").
		Offset(offset).
		Limit(pageSize).
		Find(&logs)

	c.JSON(http.StatusOK, gin.H{
		"total": total,
		"logs":  logs,
	})
}
