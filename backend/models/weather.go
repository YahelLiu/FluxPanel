package models

import (
	"time"
)

// WeatherConfig 天气配置
type WeatherConfig struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Enabled   bool      `gorm:"default:false" json:"enabled"`
	ApiKey    string    `gorm:"size:100" json:"api_key"`      // 和风天气 API Key
	ApiHost   string    `gorm:"size:100" json:"api_host"`     // API Host，默认 devapi.qweather.com
	ChannelID uint      `json:"channel_id"`                   // 通知渠道 ID
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// WeatherSchedule 天气推送时间配置
type WeatherSchedule struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Name      string    `gorm:"size:50" json:"name"`          // 时间段名称，如 "上午" "下午"
	StartHour int       `json:"start_hour"`                   // 开始小时，如 8
	EndHour   int       `json:"end_hour"`                     // 结束小时，如 12
	Enabled   bool      `gorm:"default:true" json:"enabled"`
	NextRun   *time.Time `json:"next_run"`                    // 下次运行时间
	CreatedAt time.Time `json:"created_at"`
}

// WeatherRecord 天气推送记录
type WeatherRecord struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	ClientID  string    `gorm:"index" json:"client_id"`
	Location  string    `gorm:"size:200" json:"location"`     // 城市/地址
	Date      string    `gorm:"size:20" json:"date"`          // 天气日期
	TempMax   string    `gorm:"size:10" json:"temp_max"`      // 最高温度
	TempMin   string    `gorm:"size:10" json:"temp_min"`      // 最低温度
	TextDay   string    `gorm:"size:50" json:"text_day"`      // 白天天气
	TextNight string    `gorm:"size:50" json:"text_night"`    // 夜间天气
	SentAt    time.Time `json:"sent_at"`                      // 发送时间
	CreatedAt time.Time `json:"created_at"`
}

// WeatherAPIResponse 和风天气 API 响应
type WeatherAPIResponse struct {
	Code  string `json:"code"`
	Daily []struct {
		FxDate    string `json:"fxDate"`
		TempMax   string `json:"tempMax"`
		TempMin   string `json:"tempMin"`
		TextDay   string `json:"textDay"`
		TextNight string `json:"textNight"`
	} `json:"daily"`
}

// CityLookupResponse 城市查询响应
type CityLookupResponse struct {
	Code string `json:"code"`
	Location []struct {
		Name    string `json:"name"`
		ID      string `json:"id"`
		Adm1    string `json:"adm1"` // 省/州
		Adm2    string `json:"adm2"` // 市
	} `json:"location"`
}
