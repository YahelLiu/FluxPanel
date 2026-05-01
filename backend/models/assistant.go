package models

import (
	"time"
)

// AIUser AI 助手用户
type AIUser struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	WecomUserID  string    `gorm:"size:100;uniqueIndex" json:"wecom_user_id"` // 企业微信用户ID
	Name         string    `gorm:"size:100" json:"name"`                       // 用户名称
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// Conversation 对话记录
type Conversation struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `gorm:"index" json:"user_id"`
	Role      string    `gorm:"size:20" json:"role"`       // user 或 assistant
	Content   string    `gorm:"type:text" json:"content"`  // 消息内容
	CreatedAt time.Time `gorm:"index" json:"created_at"`
}

// Memory 记忆
type Memory struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `gorm:"index" json:"user_id"`
	Content   string    `gorm:"type:text" json:"content"` // 记忆内容
	CreatedAt time.Time `json:"created_at"`
}

// Todo 待办事项
type Todo struct {
	ID        uint       `gorm:"primaryKey" json:"id"`
	UserID    uint       `gorm:"index" json:"user_id"`
	Content   string     `gorm:"type:text" json:"content"` // Todo内容
	Deadline  *time.Time `json:"deadline"`                 // 截止时间（可选）
	Completed bool       `gorm:"default:false" json:"completed"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

// Reminder 提醒
type Reminder struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `gorm:"index" json:"user_id"`
	Content   string    `gorm:"type:text" json:"content"` // 提醒内容
	RemindAt  time.Time `gorm:"index" json:"remind_at"`   // 提醒时间
	Sent      bool      `gorm:"default:false" json:"sent"`
	CreatedAt time.Time `json:"created_at"`
}

// Intent 意图类型
type Intent string

const (
	IntentChat     Intent = "chat"
	IntentMemory   Intent = "memory"
	IntentTodo     Intent = "todo"
	IntentReminder Intent = "reminder"
)

// Action 操作类型
type Action string

const (
	ActionCreate  Action = "create"
	ActionList    Action = "list"
	ActionComplete Action = "complete"
	ActionCancel  Action = "cancel"
	ActionNone    Action = "none"
)

// AgentResult Agent 决策结果
type AgentResult struct {
	Intent  Intent  `json:"intent"`
	Action  Action  `json:"action"`
	Content string  `json:"content"`
	Time    string  `json:"time,omitempty"` // 时间描述（可选）
}

// LLMConfig LLM 配置
type LLMConfig struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Provider  string    `gorm:"size:50" json:"provider"`   // openai, qwen
	APIKey    string    `gorm:"type:text" json:"api_key"`  // API Key
	BaseURL   string    `gorm:"type:text" json:"base_url"` // API Base URL（可选）
	Model     string    `gorm:"size:100" json:"model"`     // 模型名称
	Enabled   bool      `gorm:"default:true" json:"enabled"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// WeComConfig 企业微信配置
type WeComConfig struct {
	ID              uint      `gorm:"primaryKey" json:"id"`
	CorpID          string    `gorm:"size:100" json:"corp_id"`
	AgentID         string    `gorm:"size:100" json:"agent_id"`
	Secret          string    `gorm:"type:text" json:"secret"`
	Token           string    `gorm:"type:text" json:"token"`            // 回调验证用
	EncodingAESKey  string    `gorm:"type:text" json:"encoding_aes_key"` // 消息加解密用
	Enabled         bool      `gorm:"default:true" json:"enabled"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// UserAIPreference 用户 AI 偏好设置
type UserAIPreference struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	WecomUserID string    `gorm:"uniqueIndex;size:100" json:"wecom_user_id"` // 企业微信用户ID
	AdapterName string    `gorm:"size:50" json:"adapter_name"`               // 当前使用的适配器
	UpdatedAt   time.Time `json:"updated_at"`
}

// TableName 指定表名
func (UserAIPreference) TableName() string {
	return "user_ai_preferences"
}
