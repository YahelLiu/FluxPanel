package skill

import "time"

// SkillType 定义 skill 类型
type SkillType string

const (
	SkillTypeInstruction SkillType = "instruction" // 改变回答方式（人格、风格）
	SkillTypeTool        SkillType = "tool"        // 调用白名单工具
	SkillTypeResource    SkillType = "resource"    // 读取 references/templates
)

// Skill 表示一个解析后的 skill
type Skill struct {
	// 元数据 (优先加载)
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Type        SkillType `json:"type"`
	Source      string    `json:"source"` // "builtin", "uploaded", "claude-compatible"
	Path        string    `json:"path"`
	Version     string    `json:"version"`
	Author      string    `json:"author"`

	// 安全设置 (系统控制，不从 SKILL.md 读取)
	Trusted      bool     `json:"trusted"`
	AllowedTools []string `json:"allowed_tools"` // 系统控制的白名单
	Permissions  []string `json:"permissions"`   // 授予的权限

	// 状态
	Enabled     bool   `json:"enabled"`
	ContentHash string `json:"content_hash"`

	// 匹配关键词
	Triggers []string `json:"triggers"`

	// 内容 (懒加载)
	content      *SkillContent
	contentLoaded bool

	// 时间戳
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// SkillContent 包含完整的 skill 内容 (懒加载)
type SkillContent struct {
	SystemPrompt string            // SKILL.md body
	Instructions string            // 详细指令
	References   map[string]string // references/* 文件
	Templates    map[string]string // templates/* 文件
}

// SkillMetadata 是 catalog 的轻量版本
type SkillMetadata struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Type        SkillType `json:"type"`
	Triggers    []string  `json:"triggers"`
}

// SkillConfig 从 SKILL.md YAML frontmatter 解析
type SkillConfig struct {
	Name        string   `yaml:"name"`
	Description string   `yaml:"description"`
	Version     string   `yaml:"version,omitempty"`
	Author      string   `yaml:"author,omitempty"`
	Type        string   `yaml:"type,omitempty"`
	Triggers    []string `yaml:"triggers,omitempty"`
}

// UserSkillSetting 用户级 skill 配置
type UserSkillSetting struct {
	UserID    string                 `json:"user_id"`
	SkillID   string                 `json:"skill_id"`
	Enabled   bool                   `json:"enabled"`
	Priority  int                    `json:"priority"`
	Config    map[string]interface{} `json:"config"`
	CreatedAt time.Time              `json:"created_at"`
	UpdatedAt time.Time              `json:"updated_at"`
}

// Tool 表示一个可用工具
type Tool struct {
	Name        string
	Description string
	Parameters  map[string]Parameter
	Handler     func(userID string, params map[string]interface{}) (interface{}, error)
}

// Parameter 定义工具参数
type Parameter struct {
	Type        string `json:"type"`
	Description string `json:"description"`
	Required    bool   `json:"required"`
}

// ToolCall 表示工具调用请求
type ToolCall struct {
	ID        string                 `json:"id"`
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments"`
}

// ToolResult 表示工具执行结果
type ToolResult struct {
	ToolCallID string      `json:"tool_call_id"`
	Result     interface{} `json:"result"`
	Error      string      `json:"error,omitempty"`
}

