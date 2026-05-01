package models

import "time"

// Skill 存储的 skill 定义
type Skill struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	SkillID      string    `gorm:"uniqueIndex;size:100" json:"skill_id"` // 唯一标识
	Name         string    `gorm:"size:100" json:"name"`
	Description  string    `gorm:"type:text" json:"description"`
	Type         string    `gorm:"size:20" json:"type"`         // instruction, tool, resource
	Source       string    `gorm:"size:20" json:"source"`       // builtin, uploaded, claude-compatible
	Path         string    `gorm:"size:500" json:"path"`        // 文件系统路径
	Version      string    `gorm:"size:20" json:"version"`
	Author       string    `gorm:"size:100" json:"author"`

	// 安全设置 (系统控制)
	Trusted      bool      `gorm:"default:false" json:"trusted"`
	AllowedTools string    `gorm:"type:text" json:"allowed_tools"` // JSON array
	Permissions  string    `gorm:"type:text" json:"permissions"`   // JSON array

	// 状态
	Enabled      bool      `gorm:"default:true" json:"enabled"`
	ContentHash  string    `gorm:"size:64" json:"content_hash"`

	// SKILL.md 元数据
	Triggers     string    `gorm:"type:text" json:"triggers"` // JSON array

	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// TableName
func (Skill) TableName() string {
	return "skills"
}

// UserSkillSetting 用户特定 skill 设置
type UserSkillSetting struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    string    `gorm:"index;size:100" json:"user_id"` // wecom_user_id
	SkillID   uint      `gorm:"index;column:skill_id" json:"skill_id"` // FK to Skill.ID
	Enabled   bool      `gorm:"default:true" json:"enabled"`
	Priority  int       `gorm:"default:0" json:"priority"`
	Config    string    `gorm:"type:text" json:"config"` // JSON object

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// TableName
func (UserSkillSetting) TableName() string {
	return "user_skill_settings"
}

// SkillExecutionLog 记录 skill 使用审计
type SkillExecutionLog struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	UserID       string    `gorm:"index;size:100" json:"user_id"`
	SkillID      uint      `gorm:"index;column:skill_id" json:"skill_id"`
	Message      string    `gorm:"type:text" json:"message"`     // 用户消息 (截断)
	ToolsCalled  string    `gorm:"type:text" json:"tools_called"` // JSON array
	Success      bool      `gorm:"default:true" json:"success"`
	ErrorMessage string    `gorm:"type:text" json:"error_message"`
	Duration     int       `json:"duration"` // 毫秒

	CreatedAt    time.Time `gorm:"index" json:"created_at"`
}

// TableName
func (SkillExecutionLog) TableName() string {
	return "skill_execution_logs"
}
