package handlers

import (
	"log"
	"net/http"
	"strconv"

	"client-monitor/database"
	"client-monitor/ilink"
	"client-monitor/models"
	"client-monitor/notify"
	"client-monitor/notify/types"
	"client-monitor/wecom"

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
	WechatILink models.WechatILinkConfig `json:"wechat_ilink"`
	Description string                   `json:"description"`
}

// CreateChannel POST /api/notifications/channels - 创建通知渠道
func CreateChannel(c *gin.Context) {
	var req CreateChannelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 微信 iLink 只能创建一个
	if req.Type == models.NotificationTypeWechatILink {
		var count int64
		database.DB.Model(&models.NotificationChannel{}).
			Where("type = ?", models.NotificationTypeWechatILink).
			Count(&count)
		if count > 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "微信 iLink 渠道已存在，请编辑现有渠道"})
			return
		}
	}

	channel := models.NotificationChannel{
		Name:        req.Name,
		Type:        req.Type,
		Mode:        req.Mode,
		Enabled:     req.Enabled,
		Trigger:     req.Trigger,
		Feishu:      req.Feishu,
		WechatWork:  req.WechatWork,
		WechatILink: req.WechatILink,
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
	WechatILink models.WechatILinkConfig `json:"wechat_ilink"`
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
	channel.WechatILink = req.WechatILink
	channel.Description = req.Description

	if err := database.DB.Save(&channel).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update channel"})
		return
	}

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

	title := "📢 测试通知"
	content := "这是一条测试消息，用于验证通知渠道配置是否正确。"

	// 根据渠道类型选择对应的驱动
	var driverName string
	switch channel.Type {
	case models.NotificationTypeWechatILink:
		driverName = "ilink"
	case models.NotificationTypeFeishu:
		driverName = "feishu"
	default:
		// 其他类型使用自动路由
		if err := notify.GetNotifyService().SendSystem(title, content); err != nil {
			c.JSON(http.StatusOK, gin.H{"success": false, "error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"success": true})
		return
	}

	// 指定驱动发送
	if err := notify.GetNotifyService().SendTo(driverName, types.NewNotifyMessage(types.MessageTypeSystem, title, content)); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "error": err.Error()})
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

// --- 微信 iLink 登录 ---

// GetWechatILinkQRCode GET /api/notifications/channels/wechat-ilink/qrcode - 获取登录二维码
func GetWechatILinkQRCode(c *gin.Context) {
	qr, err := ilink.FetchQRCode(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"qrcode_url": qr.QRCodeImgContent,
		"qrcode":     qr.QRCode,
	})
}

// GetWechatILinkStatus GET /api/notifications/channels/wechat-ilink/status - 检查登录状态
func GetWechatILinkStatus(c *gin.Context) {
	qrcode := c.Query("qrcode")
	if qrcode == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "qrcode required"})
		return
	}

	creds, err := ilink.PollQRStatus(c.Request.Context(), qrcode, nil)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"status": "waiting"})
		return
	}

	// 调试日志
	log.Printf("[ilink] Login success: BotToken=%s, ILinkBotID=%s, ILinkUserID=%s, BaseURL=%s",
		creds.BotToken, creds.ILinkBotID, creds.ILinkUserID, creds.BaseURL)

	// 登录成功，创建或更新微信 iLink 渠道
	var channel models.NotificationChannel
	result := database.DB.Where("type = ?", models.NotificationTypeWechatILink).First(&channel)

	if result.Error != nil {
		// 创建新渠道
		channel = models.NotificationChannel{
			Name:    "微信 iLink",
			Type:    models.NotificationTypeWechatILink,
			Mode:    models.NotificationModeApp,
			Enabled: true,
			WechatILink: models.WechatILinkConfig{
				BotToken:    creds.BotToken,
				ILinkBotID:  creds.ILinkBotID,
				BaseURL:     creds.BaseURL,
				ILinkUserID: creds.ILinkUserID,
				LoggedIn:    true,
			},
		}
		if err := database.DB.Create(&channel).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create channel"})
			return
		}
	} else {
		// 更新现有渠道
		channel.WechatILink = models.WechatILinkConfig{
			BotToken:    creds.BotToken,
			ILinkBotID:  creds.ILinkBotID,
			BaseURL:     creds.BaseURL,
			ILinkUserID: creds.ILinkUserID,
			LoggedIn:    true,
		}
		database.DB.Save(&channel)
	}

	// 重置 wecom 客户端，让它重新从数据库加载凭证
	wecom.ResetClient()

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"channel": channel,
	})
}

// GetWechatILinkChannel GET /api/notifications/channels/wechat-ilink - 获取微信 iLink 渠道
func GetWechatILinkChannel(c *gin.Context) {
	var channel models.NotificationChannel
	result := database.DB.Where("type = ?", models.NotificationTypeWechatILink).First(&channel)

	if result.Error != nil {
		c.JSON(http.StatusOK, gin.H{
			"exists":   false,
			"logged_in": false,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"exists":    true,
		"logged_in": channel.WechatILink.LoggedIn,
		"channel":   channel,
	})
}

// LogoutWechatILink POST /api/notifications/channels/wechat-ilink/logout - 微信登出
func LogoutWechatILink(c *gin.Context) {
	var channel models.NotificationChannel
	result := database.DB.Where("type = ?", models.NotificationTypeWechatILink).First(&channel)

	if result.Error != nil {
		c.JSON(http.StatusOK, gin.H{"success": true})
		return
	}

	// 清除登录状态
	channel.WechatILink.LoggedIn = false
	channel.WechatILink.BotToken = ""
	database.DB.Save(&channel)

	c.JSON(http.StatusOK, gin.H{"success": true})
}
