package handlers

import (
	"client-monitor/database"
	"client-monitor/models"
	"client-monitor/notify"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/datatypes"
)

// GetWeatherConfig GET /api/weather/config - 获取天气配置
func GetWeatherConfig(c *gin.Context) {
	var config models.WeatherConfig
	result := database.DB.First(&config)

	if result.Error != nil {
		// 返回默认配置
		c.JSON(http.StatusOK, models.WeatherConfig{
			Enabled: false,
			ApiHost: "devapi.qweather.com",
		})
		return
	}

	c.JSON(http.StatusOK, config)
}

// UpdateWeatherConfig PUT /api/weather/config - 更新天气配置
func UpdateWeatherConfig(c *gin.Context) {
	var req models.WeatherConfig
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 设置默认 API Host
	if req.ApiHost == "" {
		req.ApiHost = "devapi.qweather.com"
	}

	var config models.WeatherConfig
	result := database.DB.First(&config)

	if result.Error != nil {
		// 创建新配置
		if err := database.DB.Create(&req).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create config"})
			return
		}
		c.JSON(http.StatusOK, req)
	} else {
		// 更新现有配置
		config.Enabled = req.Enabled
		config.ApiKey = req.ApiKey
		config.ApiHost = req.ApiHost
		config.ChannelID = req.ChannelID

		if err := database.DB.Save(&config).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update config"})
			return
		}
		c.JSON(http.StatusOK, config)
	}
}

// TestWeatherConfig POST /api/weather/test - 测试天气配置
func TestWeatherConfig(c *gin.Context) {
	var req struct {
		ApiKey    string `json:"api_key"`
		ApiHost   string `json:"api_host"`
		Location  string `json:"location" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.ApiKey == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "API Key 不能为空"})
		return
	}

	if req.ApiHost == "" {
		req.ApiHost = "devapi.qweather.com"
	}

	tempMax, tempMin, textDay, textNight, fxDate, err := notify.GetWeatherService().TestWeatherConfig(req.ApiKey, req.ApiHost, req.Location)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"weather": map[string]string{
			"temp_max":   tempMax,
			"temp_min":   tempMin,
			"text_day":   textDay,
			"text_night": textNight,
			"fx_date":    fxDate,
		},
	})
}

// GetWeatherSchedules GET /api/weather/schedules - 获取天气推送时间配置
func GetWeatherSchedules(c *gin.Context) {
	var schedules []models.WeatherSchedule
	database.DB.Order("start_hour").Find(&schedules)

	if len(schedules) == 0 {
		// 返回默认配置
		schedules = []models.WeatherSchedule{
			{Name: "上午", StartHour: 8, EndHour: 12, Enabled: true},
			{Name: "下午", StartHour: 18, EndHour: 22, Enabled: true},
		}
	}

	c.JSON(http.StatusOK, schedules)
}

// UpdateWeatherSchedules PUT /api/weather/schedules - 更新天气推送时间配置
func UpdateWeatherSchedules(c *gin.Context) {
	var req []models.WeatherSchedule
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 删除现有配置
	database.DB.Where("1 = 1").Delete(&models.WeatherSchedule{})

	// 创建新配置
	for _, schedule := range req {
		schedule.ID = 0 // 确保创建新记录
		database.DB.Create(&schedule)
	}

	c.JSON(http.StatusOK, req)
}

// GetWeatherRecords GET /api/weather/records - 获取天气推送记录
func GetWeatherRecords(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	if pageSize > 100 {
		pageSize = 100
	}

	var total int64
	database.DB.Model(&models.WeatherRecord{}).Count(&total)

	var records []models.WeatherRecord
	offset := (page - 1) * pageSize
	database.DB.Order("created_at DESC").
		Offset(offset).
		Limit(pageSize).
		Find(&records)

	c.JSON(http.StatusOK, gin.H{
		"total":   total,
		"records": records,
	})
}

// SendWeatherNow POST /api/weather/send - 立即发送天气通知
func SendWeatherNow(c *gin.Context) {
	var req struct {
		ClientID string `json:"client_id"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.ClientID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "client_id required"})
		return
	}

	// 获取天气配置
	config := notify.GetWeatherService().GetWeatherConfig()
	if config == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "天气配置未设置"})
		return
	}

	// 获取客户端最新位置
	var event models.Event
	if err := database.DB.Where("client_id = ?", req.ClientID).
		Order("created_at desc").
		First(&event).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "error": "客户端没有数据"})
		return
	}

	// 解析位置
	var data map[string]interface{}
	if err := json.Unmarshal(event.Data, &data); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "error": "数据解析失败"})
		return
	}

	location, _ := data["location"].(map[string]interface{})
	city, _ := location["city"].(string)
	if city == "" {
		c.JSON(http.StatusOK, gin.H{"success": false, "error": "客户端没有位置信息"})
		return
	}

	// 获取天气
	tempMax, tempMin, textDay, textNight, fxDate, err := notify.GetWeatherService().TestWeatherConfig(config.ApiKey, config.ApiHost, city)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "error": err.Error()})
		return
	}

	// 构建消息
	message := FormatWeatherMessage(city, tempMax, tempMin, textDay, textNight, fxDate)

	// 获取客户端配置的通知渠道
	var order models.ClientOrder
	if err := database.DB.Where("client_id = ?", req.ClientID).First(&order).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "error": "客户端配置不存在"})
		return
	}

	if len(order.ChannelIDs) == 0 {
		c.JSON(http.StatusOK, gin.H{"success": false, "error": "未配置通知渠道"})
		return
	}

	// 发送通知到配置的渠道
	errs := notify.GetNotifyService().SendWeatherToChannels(city, message, order.ChannelIDs)
	if len(errs) > 0 {
		var errStrs []string
		for _, e := range errs {
			errStrs = append(errStrs, e.Error())
		}
		c.JSON(http.StatusOK, gin.H{"success": false, "error": strings.Join(errStrs, ", ")})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "message": fmt.Sprintf("天气通知已发送到 %d 个渠道", len(order.ChannelIDs))})
}

// FormatWeatherMessage 格式化天气消息（导出供其他包使用）
func FormatWeatherMessage(location, tempMax, tempMin, textDay, textNight, fxDate string) string {
	return "🌤️ 今日天气预报\n\n📍 " + location + " - " + fxDate + "\n\n🌡️ 温度: " + tempMin + "°C ~ " + tempMax + "°C\n☀️ 白天: " + textDay + "\n🌙 夜间: " + textNight
}

// PingWeather GET /api/weather/ping - 记录访问者位置（用于主设备定位）
func PingWeather(c *gin.Context) {
	clientID := c.Query("client_id")
	if clientID == "" {
		clientID = "Tab S9"
	}

	// 优先使用 URL 参数传入的 IP（适用于 frp 等代理环境）
	ip := c.Query("ip")

	// 如果没有传入 IP，尝试从请求头获取
	if ip == "" {
		ip = c.GetHeader("X-Forwarded-For")
	}
	if ip == "" {
		ip = c.GetHeader("X-Real-IP")
	}
	if ip == "" {
		ip = c.ClientIP()
	}
	// X-Forwarded-For 可能包含多个 IP，取第一个
	if strings.Contains(ip, ",") {
		ip = strings.TrimSpace(strings.Split(ip, ",")[0])
	}

	// 如果是内网 IP（Docker 网络），通过外部服务获取公网 IP
	if isPrivateIP(ip) {
		publicIP, err := getPublicIP()
		if err == nil {
			ip = publicIP
		}
	}

	log.Printf("[ping] X-Forwarded-For: %s, X-Real-IP: %s, ClientIP: %s", c.GetHeader("X-Forwarded-For"), c.GetHeader("X-Real-IP"), c.ClientIP())
	log.Printf("[ping] using IP: %s", ip)

	// 通过 IP 获取位置
	location, err := notify.GetWeatherService().LookupLocationByIP(ip)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"error":   "无法获取位置信息",
			"ip":      ip,
		})
		return
	}

	city, _ := location["city"].(string)
	if city == "" {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"error":   "无法解析城市",
			"ip":      ip,
		})
		return
	}

	// 创建事件记录（包含位置信息）
	data := map[string]interface{}{
		"location": location,
		"ip":       ip,
	}
	dataJSON, _ := json.Marshal(data)

	event := models.Event{
		ClientID:  clientID,
		EventType: "ping",
		Data:      datatypes.JSON(dataJSON),
		Status:    "success",
		CreatedAt: time.Now(),
	}
	database.DB.Create(&event)

	// 自动设置为主设备（先清除其他主设备）
	database.DB.Model(&models.ClientOrder{}).Where("is_primary = ?", true).Update("is_primary", false)

	var order models.ClientOrder
	result := database.DB.Where("client_id = ?", clientID).First(&order)
	if result.Error != nil {
		order = models.ClientOrder{
			ClientID:  clientID,
			IsPrimary: true,
		}
		database.DB.Create(&order)
	} else {
		order.IsPrimary = true
		database.DB.Save(&order)
	}

	c.JSON(http.StatusOK, gin.H{
		"success":   true,
		"client_id": clientID,
		"ip":        ip,
		"location":  location,
	})
}

// isPrivateIP 检查是否是内网 IP
func isPrivateIP(ip string) bool {
	// 10.0.0.0/8
	if strings.HasPrefix(ip, "10.") {
		return true
	}
	// 172.16.0.0/12
	if strings.HasPrefix(ip, "172.1") || strings.HasPrefix(ip, "172.2") || strings.HasPrefix(ip, "172.3") {
		// 更精确检查 172.16-31.x.x
		parts := strings.Split(ip, ".")
		if len(parts) >= 2 {
			second, _ := strconv.Atoi(parts[1])
			if second >= 16 && second <= 31 {
				return true
			}
		}
	}
	// 192.168.0.0/16
	if strings.HasPrefix(ip, "192.168.") {
		return true
	}
	// 127.0.0.0/8 (localhost)
	if strings.HasPrefix(ip, "127.") {
		return true
	}
	return false
}

// getPublicIP 通过外部服务获取公网 IP
func getPublicIP() (string, error) {
	// 使用 ipify.org 获取公网 IP
	resp, err := http.Get("https://api.ipify.org?format=text")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(body)), nil
}
