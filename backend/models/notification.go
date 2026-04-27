package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"
)

// NotificationType 通知渠道类型
type NotificationType string

const (
	NotificationTypeFeishu     NotificationType = "feishu"
	NotificationTypeWechatWork NotificationType = "wechat_work"
)

// NotificationMode 通知模式
type NotificationMode string

const (
	NotificationModeWebhook NotificationMode = "webhook"
	NotificationModeApp     NotificationMode = "app"
)

// TriggerCondition 触发条件
type TriggerCondition string

const (
	TriggerOnError      TriggerCondition = "error"
	TriggerOnWarning    TriggerCondition = "warning"
	TriggerOnAll        TriggerCondition = "all"
	TriggerOnCustom     TriggerCondition = "custom"
)

// FeishuConfig 飞书配置
type FeishuConfig struct {
	WebhookURL string `json:"webhook_url,omitempty"`
	AppID      string `json:"app_id,omitempty"`
	AppSecret  string `json:"app_secret,omitempty"`
	// 单聊消息时指定接收用户
	UserIDs []string `json:"user_ids,omitempty"`
}

// Scan implements sql.Scanner for FeishuConfig
func (f *FeishuConfig) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}
	return json.Unmarshal(bytes, f)
}

// Value implements driver.Valuer for FeishuConfig
func (f FeishuConfig) Value() (driver.Value, error) {
	return json.Marshal(f)
}

// WechatWorkConfig 企业微信配置
type WechatWorkConfig struct {
	WebhookURL string `json:"webhook_url,omitempty"`
	CorpID     string `json:"corp_id,omitempty"`
	AgentID    string `json:"agent_id,omitempty"`
	Secret     string `json:"secret,omitempty"`
	// 单聊消息时指定接收用户
	UserIDs []string `json:"user_ids,omitempty"`
}

// Scan implements sql.Scanner for WechatWorkConfig
func (w *WechatWorkConfig) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}
	return json.Unmarshal(bytes, w)
}

// Value implements driver.Valuer for WechatWorkConfig
func (w WechatWorkConfig) Value() (driver.Value, error) {
	return json.Marshal(w)
}

// NotificationChannel 通知渠道配置
type NotificationChannel struct {
	ID          uint             `gorm:"primaryKey" json:"id"`
	Name        string           `gorm:"size:100;not null" json:"name"`
	Type        NotificationType `gorm:"size:20;not null" json:"type"`
	Mode        NotificationMode `gorm:"size:20;not null" json:"mode"`
	Enabled     bool             `gorm:"default:true" json:"enabled"`
	Trigger     TriggerCondition `gorm:"size:20;default:error" json:"trigger"`
	Feishu      FeishuConfig     `gorm:"type:jsonb" json:"feishu"`
	WechatWork  WechatWorkConfig `gorm:"type:jsonb" json:"wechat_work"`
	Description string           `gorm:"size:500" json:"description"`
	CreatedAt   time.Time        `json:"created_at"`
	UpdatedAt   time.Time        `json:"updated_at"`
}

// NotificationLog 通知日志
type NotificationLog struct {
	ID        uint             `gorm:"primaryKey" json:"id"`
	ChannelID uint             `gorm:"index" json:"channel_id"`
	EventID   uint             `gorm:"index" json:"event_id"`
	Status    string           `gorm:"size:20" json:"status"` // success, failed
	Message   string           `gorm:"type:text" json:"message"`
	Error     string           `gorm:"type:text" json:"error"`
	CreatedAt time.Time        `json:"created_at"`
}

// NotificationRule 通知规则
type NotificationRule struct {
	ID           uint             `gorm:"primaryKey" json:"id"`
	Name         string           `gorm:"size:100;not null" json:"name"`
	ChannelIDs   IntArray         `gorm:"type:jsonb" json:"channel_ids"`
	EventTypes   StringArray      `gorm:"type:jsonb" json:"event_types"`     // 为空表示所有类型
	StatusFilter StringArray      `gorm:"type:jsonb" json:"status_filter"`   // error, warning 等
	ClientIDs    StringArray      `gorm:"type:jsonb" json:"client_ids"`      // 为空表示所有客户端
	Enabled      bool             `gorm:"default:true" json:"enabled"`
	CreatedAt    time.Time        `json:"created_at"`
	UpdatedAt    time.Time        `json:"updated_at"`
}

// IntArray 用于存储整数数组，使用 JSON 格式
type IntArray []int

// Value implements driver.Valuer
func (a IntArray) Value() (driver.Value, error) {
	if a == nil {
		return nil, nil
	}
	return json.Marshal(a)
}

// Scan implements sql.Scanner
func (a *IntArray) Scan(value interface{}) error {
	if value == nil {
		*a = nil
		return nil
	}

	// Try []byte first (from pq driver)
	bytes, ok := value.([]byte)
	if ok {
		return json.Unmarshal(bytes, a)
	}

	// Try string
	str, ok := value.(string)
	if ok {
		return json.Unmarshal([]byte(str), a)
	}

	return errors.New("type assertion to []byte or string failed")
}

// StringArray 用于存储字符串数组，使用 JSON 格式
type StringArray []string

func (a StringArray) Value() (driver.Value, error) {
	return json.Marshal(a)
}

func (a *StringArray) Scan(value interface{}) error {
	if value == nil {
		*a = nil
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}
	return json.Unmarshal(bytes, a)
}
