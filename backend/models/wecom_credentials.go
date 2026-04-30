package models

import (
	"time"
)

// WeComCredentials iLink Bot 凭证（扫码登录后保存）
type WeComCredentials struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	BotToken    string    `gorm:"type:text" json:"bot_token"`      // iLink Bot Token
	ILinkBotID  string    `gorm:"size:100;uniqueIndex" json:"ilink_bot_id"` // Bot ID
	BaseURL     string    `gorm:"type:text" json:"base_url"`       // API Base URL
	ILinkUserID string    `gorm:"size:100" json:"ilink_user_id"`   // 用户 ID
	Enabled     bool      `gorm:"default:true" json:"enabled"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// TableName 指定表名
func (WeComCredentials) TableName() string {
	return "wecom_credentials"
}
