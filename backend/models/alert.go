package models

import (
	"time"
)

// AlertThreshold 告警阈值配置
type AlertThreshold struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	Name        string    `gorm:"size:100;not null" json:"name"`
	MetricType  string    `gorm:"size:50;not null" json:"metric_type"` // cpu, memory, disk
	Operator    string    `gorm:"size:10;not null" json:"operator"`    // >, >=, <, <=
	Threshold   float64   `gorm:"not null" json:"threshold"`           // 阈值百分比
	Duration    int       `gorm:"default:0" json:"duration"`           // 持续时间(秒)，0表示立即触发
	ChannelIDs  IntArray  `gorm:"type:jsonb" json:"channel_ids"`     // 通知渠道
	Enabled     bool      `gorm:"default:true" json:"enabled"`
	Description string    `gorm:"size:500" json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// AlertRecord 告警记录
type AlertRecord struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	ThresholdID  uint      `gorm:"index" json:"threshold_id"`
	ClientID     string    `gorm:"index" json:"client_id"`
	MetricType   string    `gorm:"size:50" json:"metric_type"`
	MetricValue  float64   `json:"metric_value"`
	Threshold    float64   `json:"threshold"`
	Status       string    `gorm:"size:20" json:"status"` // triggered, resolved
	Notified     bool      `json:"notified"`
	ResolvedAt   *time.Time `json:"resolved_at"`
	CreatedAt    time.Time `json:"created_at"`
}

// AlertThresholdRequest 创建/更新阈值请求
type AlertThresholdRequest struct {
	Name        string  `json:"name" binding:"required"`
	MetricType  string  `json:"metric_type" binding:"required"`
	Operator    string  `json:"operator" binding:"required"`
	Threshold   float64 `json:"threshold" binding:"required"`
	Duration    int     `json:"duration"`
	ChannelIDs  IntArray `json:"channel_ids"`
	Enabled     bool    `json:"enabled"`
	Description string  `json:"description"`
}
