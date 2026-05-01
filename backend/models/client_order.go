package models

// ClientOrder 客户端配置
type ClientOrder struct {
	ClientID       string    `gorm:"primaryKey" json:"client_id"`
	SortOrder      int       `gorm:"default:0" json:"sort_order"`

	// 天气相关
	WeatherEnabled bool      `gorm:"default:false" json:"weather_enabled"`

	// 通知渠道
	ChannelID      uint      `gorm:"default:0" json:"channel_id"`                // 单一渠道（兼容旧代码）
	ChannelIDs     IntArray  `gorm:"type:jsonb" json:"channel_ids"`              // 多渠道（新功能）

	// 主客户端标记（聊天查询天气时优先用这个客户端的位置）
	IsPrimary      bool      `gorm:"default:false" json:"is_primary"`

	// 隐藏标记（不在监控面板显示）
	Hidden         bool      `gorm:"default:false" json:"hidden"`
}

func (ClientOrder) TableName() string {
	return "client_orders"
}
