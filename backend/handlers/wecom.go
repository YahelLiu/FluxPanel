package handlers

import (
	"log"
	"net/http"
	"strings"
	"time"

	"client-monitor/database"
	"client-monitor/models"
	"client-monitor/services"
	"client-monitor/skill"
	"client-monitor/wecom"

	"github.com/gin-gonic/gin"
)

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

	// 使用 iLink API 发送消息
	client := wecom.GetClient()
	if client == nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "error": "未登录企业微信，请先扫码登录"})
		return
	}

	// 直接发送消息
	if err := wecom.SendTestMessage(req.UserID, req.Message); err != nil {
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

	// 使用 skill router 获取 active skills
	var activeSkills []*skill.Skill
	skillManager := skill.GetManager()
	if skillManager != nil {
		router := skill.NewRouter(skillManager)
		var routeErr error
		activeSkills, routeErr = router.Route(req.UserID, req.Message)
		if routeErr != nil {
			log.Printf("[wecom] skill routing error: %v", routeErr)
		}
		if len(activeSkills) > 0 {
			skillNames := make([]string, len(activeSkills))
			for i, s := range activeSkills {
				skillNames[i] = s.Name
			}
			log.Printf("[wecom] activated skills: %v", skillNames)
		}
	}

	response, err := agent.ProcessMessageWithSkills(user.ID, req.Message, activeSkills)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "error": err.Error()})
		return
	}

	// 如果需要发送到企业微信
	if req.SendToWeCom {
		if err := wecom.SendTestMessage(req.UserID, response); err != nil {
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

	// 使用流式响应
	var fullResponse strings.Builder

	err := agent.StreamChat(userID, message, func(chunk string) error {
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

	// 发送结束标记
	c.SSEvent("done", gin.H{})
	flusher.Flush()

	// 如果需要发送到企业微信
	if sendToWeCom && fullResponse.Len() > 0 {
		if err := wecom.SendTestMessage(wecomUserID, fullResponse.String()); err != nil {
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
