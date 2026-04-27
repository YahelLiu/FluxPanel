package models

type ClientOrder struct {
	ClientID       string `gorm:"primaryKey" json:"client_id"`
	SortOrder      int    `gorm:"default:0" json:"sort_order"`
	WeatherEnabled bool   `gorm:"default:false" json:"weather_enabled"`           // 是否启用天气推送
	ChannelID      uint   `gorm:"default:0" json:"channel_id"`                    // 通知渠道ID，0表示使用默认渠道
}

func (ClientOrder) TableName() string {
	return "client_orders"
}
