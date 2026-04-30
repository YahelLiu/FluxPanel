package notify

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"client-monitor/database"
	"client-monitor/models"
)

// WeatherService 天气服务
type WeatherService struct {
	stopChan chan struct{}
	mux      sync.RWMutex
}

var (
	weatherService     *WeatherService
	weatherServiceOnce sync.Once
)

// GetWeatherService 获取天气服务单例
func GetWeatherService() *WeatherService {
	weatherServiceOnce.Do(func() {
		weatherService = &WeatherService{
			stopChan: make(chan struct{}),
		}
	})
	return weatherService
}

// Start 启动天气服务
func (w *WeatherService) Start() {
	go w.scheduler()
	log.Println("Weather service started")
}

// Stop 停止天气服务
func (w *WeatherService) Stop() {
	close(w.stopChan)
	log.Println("Weather service stopped")
}

// scheduler 调度器
func (w *WeatherService) scheduler() {
	// 初始检查，延迟 1 分钟后开始
	time.Sleep(1 * time.Minute)

	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-w.stopChan:
			return
		case <-ticker.C:
			w.checkAndSchedule()
		}
	}
}

// checkAndSchedule 检查并调度天气推送
func (w *WeatherService) checkAndSchedule() {
	// 获取天气配置
	var config models.WeatherConfig
	if err := database.DB.First(&config).Error; err != nil {
		return // 未配置
	}

	if !config.Enabled || config.ApiKey == "" {
		return
	}

	// 获取时间段配置
	var schedules []models.WeatherSchedule
	database.DB.Where("enabled = ?", true).Find(&schedules)

	if len(schedules) == 0 {
		// 默认时间段
		schedules = []models.WeatherSchedule{
			{Name: "上午", StartHour: 8, EndHour: 12},
			{Name: "下午", StartHour: 18, EndHour: 22},
		}
	}

	now := time.Now()

	for _, schedule := range schedules {
		// 检查当前是否在时间段内
		if now.Hour() >= schedule.StartHour && now.Hour() < schedule.EndHour {
			// 检查是否已经设置过下次运行时间
			if schedule.NextRun != nil {
				// 检查是否到达运行时间
				if now.After(*schedule.NextRun) || now.Equal(*schedule.NextRun) {
					// 执行发送
					w.SendWeatherNotifications(config)
					// 清除下次运行时间，设置明天的随机时间
					w.scheduleNextRun(&schedule, now)
				}
			} else {
				// 设置今天的随机时间（如果还在时间段内）
				w.scheduleNextRun(&schedule, now)
			}
		}
	}
}

// scheduleNextRun 设置下次运行的随机时间
func (w *WeatherService) scheduleNextRun(schedule *models.WeatherSchedule, now time.Time) {
	var nextRun time.Time

	// 计算今天时间段的随机时间
	startTime := time.Date(now.Year(), now.Month(), now.Day(), schedule.StartHour, 0, 0, 0, now.Location())
	endTime := time.Date(now.Year(), now.Month(), now.Day(), schedule.EndHour, 0, 0, 0, now.Location())

	// 如果当前时间已过今天的时间段，则设置明天
	if now.Hour() >= schedule.EndHour {
		tomorrow := now.AddDate(0, 0, 1)
		startTime = time.Date(tomorrow.Year(), tomorrow.Month(), tomorrow.Day(), schedule.StartHour, 0, 0, 0, tomorrow.Location())
		endTime = time.Date(tomorrow.Year(), tomorrow.Month(), tomorrow.Day(), schedule.EndHour, 0, 0, 0, tomorrow.Location())
	}

	// 计算随机时间
	duration := endTime.Sub(startTime)
	if duration > 0 {
		randomDuration := time.Duration(rand.Int63n(int64(duration)))
		nextRun = startTime.Add(randomDuration)
	} else {
		nextRun = startTime
	}

	// 更新数据库
	database.DB.Model(schedule).Update("next_run", nextRun)
	log.Printf("Weather notification scheduled for %s at %s", schedule.Name, nextRun.Format("2006-01-02 15:04:05"))
}

// SendWeatherNotifications 发送天气通知（使用适配器）
func (w *WeatherService) SendWeatherNotifications(config models.WeatherConfig) {
	// 获取所有启用天气推送且有位置信息的客户端
	type ClientLocation struct {
		ClientID string
		Location string
	}
	var clientLocations []ClientLocation

	database.DB.Table("events").
		Select("DISTINCT events.client_id, events.data->>'location' as location").
		Joins("LEFT JOIN client_orders ON events.client_id = client_orders.client_id").
		Where("events.data->>'location' IS NOT NULL AND events.data->>'location' != ''").
		Where("client_orders.weather_enabled = ?", true).
		Scan(&clientLocations)

	if len(clientLocations) == 0 {
		log.Println("[weather] No clients with weather enabled and location information found")
		return
	}

	for _, cl := range clientLocations {
		weather, err := w.getWeather(config.ApiKey, config.ApiHost, cl.Location)
		if err != nil {
			log.Printf("[weather] Failed to get weather for %s: %v", cl.Location, err)
			continue
		}

		content := w.buildWeatherMessage(cl.Location, weather)

		// 使用统一的通知服务发送
		if err := GetNotifyService().SendWeather(cl.Location, content); err != nil {
			log.Printf("[weather] Failed to send weather for %s: %v", cl.Location, err)
		} else {
			log.Printf("[weather] Sent weather for location: %s", cl.Location)
		}

		w.saveWeatherRecord(cl.ClientID, cl.Location, weather)
	}
}

// SendWeatherToClient 发送天气通知给指定客户端（使用适配器）
func (w *WeatherService) SendWeatherToClient(clientID string) error {
	// 获取天气配置
	var config models.WeatherConfig
	if err := database.DB.First(&config).Error; err != nil {
		return fmt.Errorf("天气配置未设置")
	}

	if config.ApiKey == "" {
		return fmt.Errorf("API Key 未配置")
	}

	// 获取客户端最新位置
	type ClientInfo struct {
		Location string
	}
	var clientInfo ClientInfo

	database.DB.Table("events").
		Select("data->>'location' as location").
		Where("events.client_id = ? AND events.data->>'location' IS NOT NULL AND events.data->>'location' != ''", clientID).
		Order("events.created_at DESC").
		Limit(1).
		Scan(&clientInfo)

	if clientInfo.Location == "" {
		return fmt.Errorf("客户端没有位置信息")
	}

	// 获取天气信息
	weather, err := w.getWeather(config.ApiKey, config.ApiHost, clientInfo.Location)
	if err != nil {
		return fmt.Errorf("获取天气失败: %v", err)
	}

	content := w.buildWeatherMessage(clientInfo.Location, weather)

	// 使用统一的通知服务发送
	if err := GetNotifyService().SendWeather(clientInfo.Location, content); err != nil {
		return fmt.Errorf("发送失败: %v", err)
	}

	log.Printf("[weather] Sent weather for client: %s, location: %s", clientID, clientInfo.Location)

	// 记录
	w.saveWeatherRecord(clientID, clientInfo.Location, weather)

	return nil
}

// buildWeatherMessage 构建天气消息
func (w *WeatherService) buildWeatherMessage(location string, weather *models.WeatherAPIResponse) string {
	if len(weather.Daily) == 0 {
		return "暂无天气数据"
	}

	today := weather.Daily[0]
	var contentBuilder strings.Builder
	contentBuilder.WriteString(fmt.Sprintf("🌤️ 今日天气预报\n\n"))
	contentBuilder.WriteString(fmt.Sprintf("📍 %s - %s\n\n", location, today.FxDate))
	contentBuilder.WriteString(fmt.Sprintf("🌡️ 温度: %s°C ~ %s°C\n", today.TempMin, today.TempMax))
	contentBuilder.WriteString(fmt.Sprintf("☀️ 白天: %s\n", today.TextDay))
	contentBuilder.WriteString(fmt.Sprintf("🌙 夜间: %s\n", today.TextNight))

	// 如果有明天天气，也添加
	if len(weather.Daily) > 1 {
		tomorrow := weather.Daily[1]
		contentBuilder.WriteString(fmt.Sprintf("\n---\n\n📅 明天 (%s)\n", tomorrow.FxDate))
		contentBuilder.WriteString(fmt.Sprintf("🌡️ 温度: %s°C ~ %s°C\n", tomorrow.TempMin, tomorrow.TempMax))
		contentBuilder.WriteString(fmt.Sprintf("☀️ 白天: %s\n", tomorrow.TextDay))
		contentBuilder.WriteString(fmt.Sprintf("🌙 夜间: %s\n", tomorrow.TextNight))
	}

	return contentBuilder.String()
}

// getWeather 获取天气信息
func (w *WeatherService) getWeather(apiKey, apiHost, location string) (*models.WeatherAPIResponse, error) {
	if apiHost == "" {
		apiHost = "devapi.qweather.com"
	}

	// 先查询城市 ID
	cityID, err := w.lookupCity(apiKey, apiHost, location)
	if err != nil {
		return nil, fmt.Errorf("city lookup failed: %w", err)
	}

	// 获取天气预报（3天）- 使用 key 参数认证
	weatherURL := fmt.Sprintf("https://%s/v7/weather/3d?location=%s&key=%s", apiHost, cityID, apiKey)

	log.Printf("Weather URL: %s", weatherURL)

	client := &http.Client{Timeout: 10 * time.Second}
	req, _ := http.NewRequest("GET", weatherURL, nil)
	req.Header.Set("Accept-Encoding", "gzip")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// 处理 gzip 压缩响应
	var reader io.Reader = resp.Body
	if resp.Header.Get("Content-Encoding") == "gzip" {
		gzReader, err := gzip.NewReader(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("gzip reader error: %w", err)
		}
		defer gzReader.Close()
		reader = gzReader
	}

	body, _ := io.ReadAll(reader)
	log.Printf("Weather response: %s", string(body))

	var weatherResp models.WeatherAPIResponse
	if err := json.Unmarshal(body, &weatherResp); err != nil {
		return nil, err
	}

	if weatherResp.Code != "200" {
		return nil, fmt.Errorf("weather API error: code=%s", weatherResp.Code)
	}

	return &weatherResp, nil
}

// lookupCity 查询城市 ID
func (w *WeatherService) lookupCity(apiKey, apiHost, location string) (string, error) {
	// 清理地址，提取城市名
	cityName := w.extractCityName(location)

	// 使用 key 参数认证，路径为 /geo/v2/city/lookup
	lookupURL := fmt.Sprintf("https://%s/geo/v2/city/lookup?location=%s&key=%s", apiHost, url.QueryEscape(cityName), apiKey)

	log.Printf("City lookup URL: %s", lookupURL)

	client := &http.Client{Timeout: 10 * time.Second}
	req, _ := http.NewRequest("GET", lookupURL, nil)
	req.Header.Set("Accept-Encoding", "gzip")
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	log.Printf("City lookup response status: %d", resp.StatusCode)

	// 处理 gzip 压缩响应
	var reader io.Reader = resp.Body
	if resp.Header.Get("Content-Encoding") == "gzip" {
		gzReader, err := gzip.NewReader(resp.Body)
		if err != nil {
			return "", fmt.Errorf("gzip reader error: %w", err)
		}
		defer gzReader.Close()
		reader = gzReader
	}

	body, _ := io.ReadAll(reader)
	log.Printf("City lookup response body: %s", string(body))

	var cityResp models.CityLookupResponse
	if err := json.Unmarshal(body, &cityResp); err != nil {
		return "", fmt.Errorf("parse response failed: %w, body: %s", err, string(body))
	}

	if cityResp.Code != "200" || len(cityResp.Location) == 0 {
		return "", fmt.Errorf("city not found: %s, code: %s", cityName, cityResp.Code)
	}

	// 返回第一个匹配的城市 ID
	return cityResp.Location[0].ID, nil
}

// extractCityName 从地址中提取城市名
func (w *WeatherService) extractCityName(address string) string {
	// 地址格式可能是: "城市名, 国家" 或 "省 市 区" 等
	parts := strings.Split(address, ",")
	if len(parts) > 0 {
		// 取最后一部分（城市名）
		city := strings.TrimSpace(parts[len(parts)-1])
		// 如果包含中文省市区格式
		if strings.Contains(city, "省") || strings.Contains(city, "市") {
			// 尝试提取市
			if idx := strings.Index(city, "市"); idx > 0 {
				return city[:idx+3]
			}
		}
		return city
	}
	return address
}

// saveWeatherRecord 保存天气记录
func (w *WeatherService) saveWeatherRecord(clientID, location string, weather *models.WeatherAPIResponse) {
	if len(weather.Daily) == 0 {
		return
	}

	today := weather.Daily[0]
	record := models.WeatherRecord{
		ClientID:  clientID,
		Location:  location,
		Date:      today.FxDate,
		TempMax:   today.TempMax,
		TempMin:   today.TempMin,
		TextDay:   today.TextDay,
		TextNight: today.TextNight,
		SentAt:    time.Now(),
		CreatedAt: time.Now(),
	}

	database.DB.Create(&record)
}

// GetWeatherConfig 获取天气配置
func (w *WeatherService) GetWeatherConfig() *models.WeatherConfig {
	var config models.WeatherConfig
	if err := database.DB.First(&config).Error; err != nil {
		return nil
	}
	return &config
}

// UpdateWeatherConfig 更新天气配置
func (w *WeatherService) UpdateWeatherConfig(config *models.WeatherConfig) error {
	if config.ID == 0 {
		return database.DB.Create(config).Error
	}
	return database.DB.Save(config).Error
}

// TestWeatherConfig 测试天气配置
func (w *WeatherService) TestWeatherConfig(apiKey, apiHost, testLocation string) (*models.WeatherAPIResponse, error) {
	return w.getWeather(apiKey, apiHost, testLocation)
}
