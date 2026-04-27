package handlers

import (
	"encoding/xml"
	"log"
	"net/http"
	"strings"
	"time"

	"client-monitor/database"
	"client-monitor/models"
	"client-monitor/notify"
	"client-monitor/services"

	"github.com/gin-gonic/gin"
)

// WeComCallbackMsg 企业微信回调消息
type WeComCallbackMsg struct {
	ToUserName   string `xml:"ToUserName"`
	FromUserName string `xml:"FromUserName"`
	CreateTime   int64  `xml:"CreateTime"`
	MsgType      string `xml:"MsgType"`
	Content      string `xml:"Content"`
	MsgId       int64  `xml:"MsgId"`
	AgentID     int    `xml:"AgentID"`
}

// WeComReplyMsg 企业微信回复消息
type WeComReplyMsg struct {
	XMLName      xml.Name `xml:"xml"`
	ToUserName   string   `xml:"ToUserName"`
	FromUserName string   `xml:"FromUserName"`
	CreateTime   int64    `xml:"CreateTime"`
	MsgType      string   `xml:"MsgType"`
	Content      string   `xml:"Content"`
}

// WeComVerify 企业微信验证
type WeComVerify struct {
	MsgSignature string `form:"msg_signature"`
	Timestamp    string `form:"timestamp"`
	Nonce        string `form:"nonce"`
	EchoStr      string `form:"echostr"`
}

// HandleWeComCallback POST /api/wecom/callback - 处理企业微信回调
func HandleWeComCallback(c *gin.Context) {
	// 解析 XML 消息
	var msg WeComCallbackMsg
	if err := c.ShouldBindXML(&msg); err != nil {
		log.Printf("解析企业微信消息失败: %v", err)
		c.String(http.StatusBadRequest, "invalid message")
		return
	}

	log.Printf("收到企业微信消息: FromUser=%s, MsgType=%s, Content=%s",
		msg.FromUserName, msg.MsgType, msg.Content)

	// 只处理文本消息
	if msg.MsgType != "text" {
		c.String(http.StatusOK, "success")
		return
	}

	// 获取或创建用户
	userID := msg.FromUserName
	var user models.AIUser
	result := database.DB.Where("wecom_user_id = ?", userID).First(&user)
	if result.Error != nil {
		// 创建新用户
		user = models.AIUser{
			WecomUserID: userID,
			Name:        userID, // 默认使用 ID 作为名称
		}
		database.DB.Create(&user)
	}

	// 处理消息
	agent := services.GetAgentService()
	response, err := agent.ProcessMessage(user.ID, msg.Content)
	if err != nil {
		log.Printf("处理消息失败: %v", err)
		response = "抱歉，处理你的消息时出错了。"
	}

	// 回复消息
	reply := WeComReplyMsg{
		ToUserName:   msg.FromUserName,
		FromUserName: msg.ToUserName,
		CreateTime:   msg.CreateTime,
		MsgType:      "text",
		Content:      response,
	}

	c.XML(http.StatusOK, reply)
}

// HandleWeComVerify GET /api/wecom/callback - 企业微信验证
func HandleWeComVerify(c *gin.Context) {
	var verify WeComVerify
	if err := c.ShouldBindQuery(&verify); err != nil {
		c.String(http.StatusBadRequest, "invalid request")
		return
	}

	// TODO: 验证签名
	// 这里简化处理，直接返回 echostr
	c.String(http.StatusOK, verify.EchoStr)
}

// SendWeComMessage 主动发送企业微信消息
func SendWeComMessage(userID string, content string) error {
	// 获取企业微信配置
	var config models.WeComConfig
	if err := database.DB.Where("enabled = ?", true).First(&config).Error; err != nil {
		return err
	}

	// 使用现有的企业微信通知器
	wechatConfig := models.WechatWorkConfig{
		CorpID:  config.CorpID,
		AgentID: config.AgentID,
		Secret:  config.Secret,
	}

	notifier := notify.NewWechatWorkNotifier(wechatConfig)

	// 创建一个空事件
	event := models.Event{
		ClientID:  "assistant",
		EventType: "reminder",
		Status:    "info",
	}

	return notifier.SendAppMessage(userID, "提醒", content, event)
}

// HandleWeComTest POST /api/wecom/test - 测试发送企业微信消息
func HandleWeComTest(c *gin.Context) {
	var req struct {
		UserID  string `json:"user_id" binding:"required"`
		Message string `json:"message" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 发送消息
	if err := SendWeComMessage(req.UserID, req.Message); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "消息已发送"})
}

// HandleWeComChat POST /api/wecom/chat - 直接聊天接口（用于测试）
func HandleWeComChat(c *gin.Context) {
	var req struct {
		UserID      string `json:"user_id" binding:"required"`
		Message     string `json:"message" binding:"required"`
		SendToWeCom bool   `json:"send_to_wecom"` // 是否发送到企业微信
		Stream      bool   `json:"stream"`        // 是否使用流式响应
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 获取或创建用户
	var user models.AIUser
	result := database.DB.Where("wecom_user_id = ?", req.UserID).First(&user)
	if result.Error != nil {
		user = models.AIUser{
			WecomUserID: req.UserID,
			Name:        req.UserID,
		}
		database.DB.Create(&user)
	}

	// 如果请求流式响应
	if req.Stream {
		handleWeComChatStream(c, user.ID, req.Message, req.UserID, req.SendToWeCom)
		return
	}

	// 非流式处理
	agent := services.GetAgentService()
	response, err := agent.ProcessMessage(user.ID, req.Message)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "error": err.Error()})
		return
	}

	// 如果需要发送到企业微信
	if req.SendToWeCom {
		if err := SendWeComMessage(req.UserID, response); err != nil {
			log.Printf("发送到企业微信失败: %v", err)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success":  true,
		"response": response,
	})
}

// handleWeComChatStream 流式聊天处理
func handleWeComChatStream(c *gin.Context, userID uint, message, wecomUserID string, sendToWeCom bool) {
	agent := services.GetAgentService()

	// 设置 SSE 响应头
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Transfer-Encoding", "chunked")

	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		c.JSON(http.StatusOK, gin.H{"success": false, "error": "不支持流式响应"})
		return
	}

	// 先分析意图（这部分很快，不需要流式）
	result, err := agent.AnalyzeIntent(message)
	if err != nil {
		c.SSEvent("error", gin.H{"error": err.Error()})
		flusher.Flush()
		return
	}

	// 非聊天意图，直接处理（记忆、提醒等）
	if result.Intent != models.IntentChat {
		response, err := agent.ProcessMessage(userID, message)
		if err != nil {
			c.SSEvent("error", gin.H{"error": err.Error()})
			flusher.Flush()
			return
		}
		// 直接发送完整响应，一次性
		c.SSEvent("done", gin.H{"content": response})
		flusher.Flush()
		return
	}

	// 聊天意图，使用流式响应
	var fullResponse strings.Builder

	err = agent.StreamChat(userID, message, func(chunk string) error {
		fullResponse.WriteString(chunk)
		c.SSEvent("message", gin.H{"content": chunk})
		flusher.Flush()
		return nil
	})

	if err != nil {
		c.SSEvent("error", gin.H{"error": err.Error()})
		flusher.Flush()
		return
	}

	// 发送结束标记（不包含内容，只是标记结束）
	c.SSEvent("done", gin.H{})
	flusher.Flush()

	// 如果需要发送到企业微信
	if sendToWeCom && fullResponse.Len() > 0 {
		if err := SendWeComMessage(wecomUserID, fullResponse.String()); err != nil {
			log.Printf("发送到企业微信失败: %v", err)
		}
	}
}

// GetWeComConfig GET /api/wecom/config - 获取企业微信配置
func GetWeComConfig(c *gin.Context) {
	var config models.WeComConfig
	result := database.DB.First(&config)

	if result.Error != nil {
		c.JSON(http.StatusOK, models.WeComConfig{
			Enabled: false,
		})
		return
	}

	// 隐藏敏感信息
	config.Secret = maskString(config.Secret)
	config.Token = maskString(config.Token)
	config.EncodingAESKey = maskString(config.EncodingAESKey)

	c.JSON(http.StatusOK, config)
}

// UpdateWeComConfig PUT /api/wecom/config - 更新企业微信配置
func UpdateWeComConfig(c *gin.Context) {
	var req models.WeComConfig
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var config models.WeComConfig
	result := database.DB.First(&config)

	if result.Error != nil {
		// 创建新配置
		if err := database.DB.Create(&req).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "创建配置失败"})
			return
		}
		c.JSON(http.StatusOK, req)
	} else {
		// 更新配置（不更新空值）
		updates := make(map[string]interface{})
		if req.CorpID != "" {
			updates["corp_id"] = req.CorpID
		}
		if req.AgentID != "" {
			updates["agent_id"] = req.AgentID
		}
		if req.Secret != "" {
			updates["secret"] = req.Secret
		}
		if req.Token != "" {
			updates["token"] = req.Token
		}
		if req.EncodingAESKey != "" {
			updates["encoding_aes_key"] = req.EncodingAESKey
		}
		updates["enabled"] = req.Enabled

		database.DB.Model(&config).Updates(updates)
		c.JSON(http.StatusOK, config)
	}
}

// GetPendingReminders GET /api/wecom/reminders/pending - 获取待发送的提醒（用于前端轮询）
func GetPendingReminders(c *gin.Context) {
	userID := c.Query("user_id")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user_id required"})
		return
	}

	// 获取用户
	var user models.AIUser
	if err := database.DB.Where("wecom_user_id = ?", userID).First(&user).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"reminders": []interface{}{}})
		return
	}

	// 查找到期但未发送的提醒
	var reminders []models.Reminder
	database.DB.Where("user_id = ? AND remind_at <= ? AND sent = ?", user.ID, time.Now(), false).
		Order("remind_at asc").
		Find(&reminders)

	c.JSON(http.StatusOK, gin.H{"reminders": reminders})
}

// MarkReminderSent POST /api/wecom/reminders/:id/sent - 标记提醒已发送（前端确认）
func MarkReminderSent(c *gin.Context) {
	id := c.Param("id")
	database.DB.Model(&models.Reminder{}).Where("id = ?", id).Update("sent", true)
	c.JSON(http.StatusOK, gin.H{"success": true})
}

// maskString 隐藏字符串中间部分
func maskString(s string) string {
	if len(s) <= 8 {
		return strings.Repeat("*", len(s))
	}
	return s[:4] + strings.Repeat("*", len(s)-8) + s[len(s)-4:]
}
