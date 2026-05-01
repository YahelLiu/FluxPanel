package notify

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"log"
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
	mux sync.RWMutex
}

var (
	weatherService     *WeatherService
	weatherServiceOnce sync.Once
)

// GetWeatherService 获取天气服务单例
func GetWeatherService() *WeatherService {
	weatherServiceOnce.Do(func() {
		weatherService = &WeatherService{}
	})
	return weatherService
}

// buildWeatherMessage 构建天气消息
func (w *WeatherService) BuildWeatherMessage(location string, weather *models.WeatherAPIResponse) string {
	return w.buildWeatherMessage(location, weather)
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

// GetWeather 获取天气信息
func (w *WeatherService) GetWeather(apiKey, apiHost, location string) (*models.WeatherAPIResponse, error) {
	return w.getWeather(apiKey, apiHost, location)
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
func (w *WeatherService) TestWeatherConfig(apiKey, apiHost, testLocation string) (string, string, string, string, string, error) {
	weather, err := w.getWeather(apiKey, apiHost, testLocation)
	if err != nil {
		return "", "", "", "", "", err
	}
	if len(weather.Daily) == 0 {
		return "", "", "", "", "", fmt.Errorf("no weather data")
	}
	today := weather.Daily[0]
	return today.TempMax, today.TempMin, today.TextDay, today.TextNight, today.FxDate, nil
}

// GetWeatherDays 获取多天天气数据
func (w *WeatherService) GetWeatherDays(apiKey, apiHost, location string) ([]map[string]string, error) {
	weather, err := w.getWeather(apiKey, apiHost, location)
	if err != nil {
		return nil, err
	}
	if len(weather.Daily) == 0 {
		return nil, fmt.Errorf("no weather data")
	}

	var days []map[string]string
	for _, d := range weather.Daily {
		days = append(days, map[string]string{
			"date":      d.FxDate,
			"temp_max":  d.TempMax,
			"temp_min":  d.TempMin,
			"text_day":  d.TextDay,
			"text_night": d.TextNight,
		})
	}
	return days, nil
}

// LookupLocationByIP 通过 IP 查询位置
func (w *WeatherService) LookupLocationByIP(ip string) (map[string]interface{}, error) {
	// 使用百度 IP 定位 API（更准确）
	url := fmt.Sprintf("https://qifu.baidu.com/api/v1/ip-portrait/brief-info?ip=%s", ip)

	client := &http.Client{Timeout: 10 * time.Second}
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Referer", "https://qifu.baidu.com/")
	req.Header.Set("User-Agent", "Mozilla/5.0")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("IP lookup failed: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	var result struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    struct {
			Country  string `json:"country"`
			Province string `json:"province"`
			City     string `json:"city"`
			ISP      string `json:"isp"`
			QueryIP  string `json:"query_ip"`
		} `json:"data"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("parse response failed: %w", err)
	}

	if result.Code != 200 {
		return nil, fmt.Errorf("IP lookup failed: code=%d, message=%s", result.Code, result.Message)
	}

	// 优先使用城市，如果没有则用省份
	location := result.Data.City
	if location == "" {
		location = result.Data.Province
	}

	return map[string]interface{}{
		"city":     location,
		"province": result.Data.Province,
		"country":  result.Data.Country,
		"isp":      result.Data.ISP,
		"ip":       result.Data.QueryIP,
	}, nil
}
