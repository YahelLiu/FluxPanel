package handlers

import (
	"client-monitor/ilink"
	"client-monitor/wecom"
	"net/http"

	"github.com/gin-gonic/gin"
)

// GetLoginQRCode GET /api/wecom/login/qrcode - 获取登录二维码
func GetLoginQRCode(c *gin.Context) {
	qr, err := ilink.FetchQRCode(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"qrcode_url": qr.QRCodeImgContent, // 图片 URL
		"qrcode":     qr.QRCode,           // 用于轮询状态
	})
}

// GetLoginStatus GET /api/wecom/login/status - 轮询登录状态
func GetLoginStatus(c *gin.Context) {
	qrcode := c.Query("qrcode")
	if qrcode == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "qrcode required"})
		return
	}

	creds, err := ilink.PollQRStatus(c.Request.Context(), qrcode, nil)
	if err != nil {
		// 未确认或过期，返回等待状态
		c.JSON(http.StatusOK, gin.H{"status": "waiting"})
		return
	}

	// 登录成功 - 这里不再保存凭证，由 notification.go 中的新 API 处理
	c.JSON(http.StatusOK, gin.H{
		"status":     "success",
		"bot_id":     creds.ILinkBotID,
		"bot_token":  creds.BotToken,
		"ilink_bot_id": creds.ILinkBotID,
		"base_url":   creds.BaseURL,
		"ilink_user_id": creds.ILinkUserID,
	})
}

// GetWeComStatus GET /api/wecom/status - 获取连接状态
func GetWeComStatus(c *gin.Context) {
	if !wecom.HasWechatILinkChannel() {
		c.JSON(http.StatusOK, gin.H{
			"connected": false,
			"message":   "请先扫码登录",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"connected": true,
		"bot_id":    wecom.GetBotID(),
	})
}

// Logout DELETE /api/wecom/session - 登出
func Logout(c *gin.Context) {
	// 注：现在登出逻辑已在 notification.go 的 LogoutWechatILink 中处理
	// 这个 handler 保留是为了向后兼容
	wecom.ResetClient()
	c.JSON(http.StatusOK, gin.H{"success": true})
}

// GetMonitorStatus GET /api/wecom/monitor/status - 获取 Monitor 运行状态
func GetMonitorStatus(c *gin.Context) {
	running, startTime := wecom.GetMonitorStatus()
	c.JSON(http.StatusOK, gin.H{
		"running":    running,
		"start_time": startTime,
		"logged_in":  wecom.HasWechatILinkChannel(),
		"bot_id":     wecom.GetBotID(),
	})
}
