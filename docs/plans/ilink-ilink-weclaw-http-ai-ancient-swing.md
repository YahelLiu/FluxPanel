# AI 后端适配器切换 + Skill 系统计划

## Status: 📋 规划中

## Context

用户希望 iLink 消息处理支持：

### 已完成：AI 后端适配器切换
- ✅ 默认 HTTP API 模式（通义千问/OpenAI 等）
- ✅ Claude CLI 模式（`/claude` 切换）
- ✅ 用户偏好持久化

### 新需求：兼容 Claude Skill 格式

实现一个兼容 Claude Skill 格式但使用自己安全模型的 Skill 系统：

1. **Skill 格式兼容**：支持 SKILL.md + YAML frontmatter 格式
2. **渐进式加载**：先加载 metadata，命中后再加载完整内容
3. **安全边界**：不执行 scripts/，工具调用需要白名单验证
4. **用户隔离**：每用户独立启用/禁用 skill

---

## Skill 系统设计

### Skill 格式兼容

```
skills/
  web-search/
    SKILL.md          # 必需，YAML frontmatter
    scripts/          # 默认不执行
    references/       # 可读取
    templates/        # 可读取
```

**SKILL.md 示例**：
```markdown
---
name: web-search
description: search the web when the user needs current information
triggers:
  - 搜索
  - search
---

# Web Search
Use this skill when the user asks for latest information.
```

### 支持的 Skill 类型

| 类型 | 功能 | 示例 |
|------|------|------|
| instruction | 改变回答方式（人格、风格） | strict-coach, friendly-assistant |
| tool | 调用白名单工具 | web-search, reminder |
| resource | 读取 references/templates | translator |

### 不支持的类型

- ❌ script skill（执行代码）
- ❌ shell skill
- ❌ browser-control skill
- ❌ file-system-write skill

### 核心组件

```
SkillManager    - 导入、解析、启用/禁用 skills
SkillRouter     - 匹配 eligible skills，选择 active skills
PromptBuilder   - 构建 catalog prompt 和 active skill prompt
ToolRegistry    - 工具白名单，安全执行
```

### 每轮对话流程

```
用户消息
    ↓
┌─────────────────┐
│ Find Eligible   │ ← 检查: enabled + user_enabled + trusted
└────────┬────────┘
         ↓
┌─────────────────┐
│ Select Active   │ ← 匹配 triggers/keywords
└────────┬────────┘
         ↓
┌─────────────────┐
│ Build Prompt    │ ← 注入 SKILL.md 内容
└────────┬────────┘
         ↓
┌─────────────────┐
│ LLM Response    │ ← 可能包含 tool calls
└────────┬────────┘
         ↓
┌─────────────────┐
│ Tool Execution  │ ← 每次调用前检查白名单
└────────┬────────┘
         ↓
    响应用户
```

---

## 文件结构

### 新增文件

```
backend/
├── skill/                          # 新包
│   ├── types.go                    # 核心数据模型和接口
│   ├── parser.go                   # SKILL.md 解析器 (YAML frontmatter)
│   ├── loader.go                   # 文件系统加载器，支持懒加载
│   ├── manager.go                  # Skill 生命周期管理
│   ├── router.go                   # Skill 匹配和选择
│   ├── prompt_builder.go           # Prompt 构建
│   └── tool_registry.go            # 工具白名单和执行
│
├── models/
│   └── skill.go                    # 数据库模型 (新文件)
│
├── handlers/
│   └── skill.go                    # Skill 管理 HTTP handlers
│
├── services/
│   └── skill_executor.go           # Skill 执行集成
│
skills/                             # Skill 存储目录
├── reminder/
│   └── SKILL.md
├── translator/
│   └── SKILL.md
└── web-search/
    └── SKILL.md
```

---

## 数据库模型

```go
// models/skill.go

// Skill 存储的 skill 定义
type Skill struct {
    ID           uint      `gorm:"primaryKey"`
    SkillID      string    `gorm:"uniqueIndex;size:100"` // 唯一标识
    Name         string    `gorm:"size:100"`
    Description  string    `gorm:"type:text"`
    Type         string    `gorm:"size:20"`         // instruction, tool, resource
    Source       string    `gorm:"size:20"`         // builtin, uploaded, claude-compatible
    Path         string    `gorm:"size:500"`        // 文件系统路径
    Trusted      bool      `gorm:"default:false"`
    AllowedTools string    `gorm:"type:text"`       // JSON array
    Enabled      bool      `gorm:"default:true"`
    ContentHash  string    `gorm:"size:64"`
    Triggers     string    `gorm:"type:text"`       // JSON array
    CreatedAt    time.Time
    UpdatedAt    time.Time
}

// UserSkillSetting 用户特定 skill 设置
type UserSkillSetting struct {
    ID        uint      `gorm:"primaryKey"`
    UserID    string    `gorm:"index;size:100"`  // wecom_user_id
    SkillID   uint      `gorm:"index"`           // FK to Skill
    Enabled   bool      `gorm:"default:true"`
    Priority  int       `gorm:"default:0"`
    Config    string    `gorm:"type:text"`       // JSON object
    CreatedAt time.Time
    UpdatedAt time.Time
}

// SkillExecutionLog 记录 skill 使用审计
type SkillExecutionLog struct {
    ID           uint      `gorm:"primaryKey"`
    UserID       string    `gorm:"index;size:100"`
    SkillID      uint      `gorm:"index"`
    ToolsCalled  string    `gorm:"type:text"`       // JSON array
    Success      bool      `gorm:"default:true"`
    Duration     int       // 毫秒
    CreatedAt    time.Time `gorm:"index"`
}
```

---

## 核心接口定义

```go
// skill/types.go

// Skill 表示一个解析后的 skill
type Skill struct {
    ID           string
    Name         string
    Description  string
    Type         SkillType  // instruction, tool, resource
    Source       string     // builtin, uploaded, claude-compatible
    Path         string
    Trusted      bool
    AllowedTools []string   // 系统控制的白名单
    Enabled      bool
    ContentHash  string
    
    // 懒加载内容
    content      *SkillContent
    contentLoaded bool
}

// SkillContent 包含完整的 skill 内容 (懒加载)
type SkillContent struct {
    SystemPrompt string            // SKILL.md body
    References   map[string]string // references/* 文件
    Templates    map[string]string // templates/* 文件
}

// SkillManager 管理 skill 生命周期
type SkillManager interface {
    Import(path string) (*Skill, error)
    Remove(skillID string) error
    Get(skillID string) (*Skill, error)
    List() ([]*Skill, error)
    SetEnabled(skillID string, enabled bool) error
    SetUserEnabled(userID, skillID string, enabled bool) error
}

// SkillRouter 匹配和选择 skills
type SkillRouter interface {
    FindEligible(userID, message string) ([]*Skill, error)
    SelectActive(eligible []*Skill, message string) ([]*Skill, error)
    Route(userID, message string) ([]*Skill, error)
}

// PromptBuilder 构建 skill prompts
type PromptBuilder interface {
    BuildCatalog(skills []*SkillMetadata) string
    BuildActivePrompt(skills []*Skill) string
    BuildSystemPrompt(basePrompt string, skills []*Skill) string
}

// ToolRegistry 管理工具白名单
type ToolRegistry interface {
    GetAllowed(skillID string) []Tool
    Execute(skillID, toolName string, params map[string]interface{}) (interface{}, error)
    IsAllowed(skillID, toolName string) bool
}
```

---

## 与现有代码集成

### 1. HTTPAgent 集成 (`backend/agent/http_adapter.go`)

```go
func (a *HTTPAgent) Chat(ctx context.Context, conversationID, message string) (string, error) {
    // 路由 skills
    activeSkills, _ := a.skillRouter.Route(conversationID, message)
    
    // 带 skills 处理
    return a.agentService.ProcessMessageWithSkills(user.ID, message, activeSkills)
}
```

### 2. ChatHandler 集成 (`backend/services/chat_handler.go`)

```go
func (h *ChatHandler) HandleWithSkills(userID uint, userMessage string, skills []*skill.Skill) (string, error) {
    basePrompt := h.buildSystemPrompt(context)
    skillPrompt := h.promptBuilder.BuildSystemPrompt(basePrompt, skills)
    
    return h.llm.ChatWithSystem(skillPrompt, messages)
}
```

### 3. Messaging Handler 集成 (`backend/messaging/handler.go`)

```go
// 新增指令
case strings.HasPrefix(text, "/skill "):
    return h.handleSkillCommand(ctx, msg.FromUserID, strings.TrimPrefix(text, "/skill "))
    
case text == "/skills":
    return h.listSkills(ctx, msg.FromUserID)
```

---

## 安全模型

### 安全原则

1. **不信任 Skill 自身声明** - 所有权限由系统控制
2. **工具白名单** - 系统控制每个 skill 可用的工具
3. **执行沙箱** - 不执行脚本，无文件系统写权限

### 工具白名单示例

```go
var allowedTools = map[string]Tool{
    "web_search": {Handler: handleWebSearch},
    "set_reminder": {Handler: handleSetReminder},
    "get_memory": {Handler: handleGetMemory},
}
```

### 权限检查

```go
func Execute(skillID, toolName string, params map[string]interface{}) (interface{}, error) {
    // 每次调用前检查白名单
    if !IsAllowed(skillID, toolName) {
        return nil, fmt.Errorf("tool %s not allowed for skill %s", toolName, skillID)
    }
    return tool.Handler(params)
}
```

---

## 实现步骤

### Phase 1: 核心基础设施 (优先级: 高)

| 步骤 | 任务 | 文件 | 估计时间 |
|------|------|------|----------|
| 1.1 | 核心数据模型和接口 | `skill/types.go` | 2h |
| 1.2 | SKILL.md 解析器 | `skill/parser.go` | 2h |
| 1.3 | 数据库模型 | `models/skill.go` | 1h |
| 1.4 | 文件系统加载器 | `skill/loader.go` | 2h |

### Phase 2: 管理层 (优先级: 高)

| 步骤 | 任务 | 文件 | 估计时间 |
|------|------|------|----------|
| 2.1 | Skill Manager | `skill/manager.go` | 3h |
| 2.2 | Tool Registry | `skill/tool_registry.go` | 2h |

### Phase 3: 路由和 Prompts (优先级: 中)

| 步骤 | 任务 | 文件 | 估计时间 |
|------|------|------|----------|
| 3.1 | Skill Router | `skill/router.go` | 3h |
| 3.2 | Prompt Builder | `skill/prompt_builder.go` | 2h |

### Phase 4: 集成 (优先级: 中)

| 步骤 | 任务 | 文件 | 估计时间 |
|------|------|------|----------|
| 4.1 | Agent 集成 | `agent/http_adapter.go` | 3h |
| 4.2 | Messaging 集成 | `messaging/handler.go` | 2h |

### Phase 5: API 和内置 Skills (优先级: 低)

| 步骤 | 任务 | 文件 | 估计时间 |
|------|------|------|----------|
| 5.1 | HTTP API | `handlers/skill.go` | 2h |
| 5.2 | 内置 Skills | `skills/*/SKILL.md` | 1h |

---

## 验证方式

1. **单元测试**：解析器、路由器、权限检查
2. **集成测试**：完整消息处理流程
3. **端到端测试**：微信消息触发 skill

```bash
# 测试 skill 解析
go test ./skill/... -v

# 测试完整流程
curl -X POST http://localhost:8080/api/wecom/callback \
  -d '{"message": "帮我搜索一下天气"}'
```

---

## 已完成：AI 后端适配器切换

### 当前架构

```
iLink 消息 → messaging/handler.go → AgentRouter → [切换]
                                         │
                    ┌────────────────────┼────────────────────┐
                    ↓                    ↓                    ↓
              HTTPAgent            ClaudeAgent          其他Agent...
           (默认, HTTP API)     (Claude Code CLI)      (未来扩展)
```

---

## 设计方案

### 核心组件

#### 1. AgentAdapter 接口

```go
// agent/adapter.go
type AgentAdapter interface {
    // Name 返回适配器名称
    Name() string
    
    // Chat 发送消息并返回响应
    Chat(ctx context.Context, userID, message string) (string, error)
    
    // IsAvailable 检查适配器是否可用
    IsAvailable() bool
    
    // Info 返回适配器信息
    Info() AgentInfo
}

type AgentInfo struct {
    Name    string // "api", "claude", "acp"
    Type    string // "http", "cli", "acp"
    Model   string // "gpt-4o-mini", "claude-sonnet-4"
    Status  string // "available", "unavailable"
}
```

#### 2. HTTPAgentAdapter（现有逻辑）

```go
// agent/http_adapter.go
type HTTPAgentAdapter struct {
    agentService *services.AgentService
}

func (a *HTTPAgentAdapter) Name() string {
    return "api"
}

func (a *HTTPAgentAdapter) Chat(ctx context.Context, userID, message string) (string, error) {
    // 复用现有的 AgentService.ProcessMessage 逻辑
    return a.agentService.ProcessMessage(userID, message)
}

func (a *HTTPAgentAdapter) IsAvailable() bool {
    // 检查 LLM 配置是否存在且启用
    return services.GetLLMService().GetConfig() != nil
}
```

#### 3. ClaudeAdapter（新增）

```go
// agent/claude_adapter.go
type ClaudeAdapter struct {
    cliPath string
}

func (a *ClaudeAdapter) Name() string {
    return "claude"
}

func (a *ClaudeAdapter) Chat(ctx context.Context, userID, message string) (string, error) {
    // 调用 claude CLI
    // claude --print "message"
    cmd := exec.CommandContext(ctx, a.cliPath, "--print", message)
    output, err := cmd.Output()
    if err != nil {
        return "", fmt.Errorf("claude CLI error: %w", err)
    }
    return string(output), nil
}

func (a *ClaudeAdapter) IsAvailable() bool {
    // 检查 claude CLI 是否存在
    _, err := exec.LookPath("claude")
    return err == nil
}
```

#### 4. AgentRouter（路由器）

```go
// agent/router.go
type AgentRouter struct {
    adapters    map[string]AgentAdapter
    userSession sync.Map // userID -> adapterName
    defaultName string
}

func NewAgentRouter() *AgentRouter {
    r := &AgentRouter{
        adapters:    make(map[string]AgentAdapter),
        defaultName: "api",
    }
    
    // 注册适配器
    r.Register(NewHTTPAgentAdapter())
    r.Register(NewClaudeAdapter())
    
    return r
}

func (r *AgentRouter) Register(adapter AgentAdapter) {
    r.adapters[adapter.Name()] = adapter
}

// Switch 切换用户的 AI 后端
func (r *AgentRouter) Switch(userID, adapterName string) error {
    if _, ok := r.adapters[adapterName]; !ok {
        return fmt.Errorf("未知的适配器: %s", adapterName)
    }
    r.userSession.Store(userID, adapterName)
    return nil
}

// Get 获取用户当前的适配器
func (r *AgentRouter) Get(userID string) AgentAdapter {
    if name, ok := r.userSession.Load(userID); ok {
        if adapter, ok := r.adapters[name.(string)]; ok {
            return adapter
        }
    }
    return r.adapters[r.defaultName]
}

// Chat 使用用户当前的适配器处理消息
func (r *AgentRouter) Chat(ctx context.Context, userID, message string) (string, error) {
    adapter := r.Get(userID)
    if !adapter.IsAvailable() {
        return "", fmt.Errorf("%s 适配器不可用", adapter.Name())
    }
    return adapter.Chat(ctx, userID, message)
}
```

---

### 指令处理

在 `messaging/handler.go` 中添加指令解析：

```go
func (h *Handler) HandleMessage(ctx context.Context, client *ilink.Client, msg ilink.WeixinMessage) {
    text := extractText(msg)
    
    // 处理切换指令
    switch {
    case strings.HasPrefix(text, "/claude"):
        h.router.Switch(msg.FromUserID, "claude")
        SendTextReply(ctx, client, msg.FromUserID, "已切换到 Claude 模式 🤖", ...)
        return
        
    case strings.HasPrefix(text, "/api"):
        h.router.Switch(msg.FromUserID, "api")
        SendTextReply(ctx, client, msg.FromUserID, "已切换到 API 模式 🌐", ...)
        return
        
    case strings.HasPrefix(text, "/mode"):
        adapter := h.router.Get(msg.FromUserID)
        info := adapter.Info()
        reply := fmt.Sprintf("当前模式: %s\n模型: %s\n状态: %s", 
            info.Name, info.Model, info.Status)
        SendTextReply(ctx, client, msg.FromUserID, reply, ...)
        return
    }
    
    // 正常消息处理
    reply, err := h.router.Chat(ctx, msg.FromUserID, text)
    // ...
}
```

---

### 文件结构

```
backend/
├── agent/
│   ├── agent.go           # AgentAdapter 接口定义
│   ├── router.go          # AgentRouter 路由器
│   ├── http_adapter.go    # HTTP API 适配器
│   └── claude_adapter.go  # Claude CLI 适配器
├── messaging/
│   └── handler.go         # 添加指令解析
└── wecom/
    └── agent_adapter.go   # 可删除或保留兼容
```

---

### 支持的指令

| 指令 | 功能 |
|------|------|
| `/claude` | 切换到 Claude 模式 |
| `/api` | 切换到 API 模式 |
| `/mode` | 查看当前模式 |
| `/skills` | 查看可用 skills |
| `/skill enable <id>` | 启用 skill |
| `/skill disable <id>` | 禁用 skill |

---

## 设计决策

### 存储方式：数据库 + 文件系统

- **数据库**：存储元数据（name, description, triggers, enabled, trusted 等）
- **文件系统**：存储 SKILL.md 完整内容和 references/templates
- **优势**：支持热重载，适合开发调试

### 内置工具

| 工具 | 功能 | 说明 |
|------|------|------|
| `reminder` | 提醒管理 | 创建、查看、取消提醒 |
| `memory` | 记忆管理 | 保存、查看用户记忆 |
| `web_search` | 网络搜索 | 搜索互联网信息（需接入搜索 API） |
| `translator` | 翻译 | 多语言翻译能力 |

### 路由方式：关键词匹配

根据 SKILL.md 中的 `triggers` 字段进行关键词匹配，简单高效。

---

## 待解决：循环依赖问题

### 问题

```
notify → messaging → agent → skill → notify (循环)
```

`skill/tool_registry.go` 直接调用 `notify.GetWeatherService()` 导致循环依赖。

### 解决方案：函数注入

在 `skill` 包定义函数类型，在初始化时注入 `notify` 的具体实现：

```go
// skill/tool_registry.go
type GetWeatherConfigFunc func() (apiKey, apiHost string, ok bool)
type TestWeatherFunc func(apiKey, apiHost, location string) (tempMax, tempMin, textDay, textNight, fxDate string, err error)

type ToolRegistry struct {
    tools  map[string]*Tool
    mu     sync.RWMutex
    // 注入的函数
    getWeatherConfig GetWeatherConfigFunc
    testWeather      TestWeatherFunc
    // ...
}

func (r *ToolRegistry) SetWeatherFunctions(getConfig GetWeatherConfigFunc, test TestWeatherFunc, ...) {
    r.getWeatherConfig = getConfig
    r.testWeather = test
}
```

```go
// wecom/handler.go 初始化时注入
toolRegistry.SetWeatherFunctions(
    func() (string, string, bool) {
        cfg := notify.GetWeatherService().GetWeatherConfig()
        return cfg.ApiKey, cfg.ApiHost, cfg != nil
    },
    func(apiKey, apiHost, location string) (string, string, string, string, string, error) {
        weather, err := notify.GetWeatherService().TestWeatherConfig(apiKey, apiHost, location)
        // ...
    },
)
```

### 改动文件

| 文件 | 改动 |
|------|------|
| `skill/tool_registry.go` | 移除 notify 导入，添加函数类型和注入方法 |
| `wecom/handler.go` | 初始化时注入天气服务函数 |

---

## 内置工具列表

| 工具 | 功能 | 对应服务 |
|------|------|----------|
| `reminder_create` | 创建提醒 | services.ReminderHandler.Create |
| `reminder_list` | 查看提醒 | services.ReminderHandler.List |
| `reminder_cancel` | 取消提醒 | services.ReminderHandler.Cancel |
| `memory_save` | 保存记忆 | services.MemoryHandler.Create |
| `memory_list` | 查看记忆 | services.MemoryHandler.List |
| `memory_delete` | 删除记忆 | services.MemoryHandler.Delete |
| `weather_get` | 获取天气 | notify.WeatherService |
| `weather_send` | 发送天气通知 | notify.WeatherService + notify.NotifyService |
| `translator` | 翻译 | services.LLMService |

---

## 下一步

1. 解决循环依赖
2. 测试编译
3. 提交代码
