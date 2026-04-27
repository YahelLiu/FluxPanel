package handlers

import (
	"client-monitor/database"
	"client-monitor/models"
	"client-monitor/notify"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
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

	weather, err := notify.GetWeatherService().TestWeatherConfig(req.ApiKey, req.ApiHost, req.Location)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"weather": weather,
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
		// 如果解析失败，发送给所有启用的客户端
		config := notify.GetWeatherService().GetWeatherConfig()
		if config == nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "天气配置未设置"})
			return
		}

		go func() {
			notify.GetWeatherService().SendWeatherNotifications(*config)
		}()

		c.JSON(http.StatusOK, gin.H{"success": true, "message": "正在发送天气通知..."})
		return
	}

	// 发送给指定客户端
	if req.ClientID != "" {
		err := notify.GetWeatherService().SendWeatherToClient(req.ClientID)
		if err != nil {
			c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"success": true, "message": "天气通知已发送"})
		return
	}

	c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
}
