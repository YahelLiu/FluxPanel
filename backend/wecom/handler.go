package wecom

import (
	"context"
	"log"
	"os"
	"path/filepath"

	"client-monitor/agent"
	"client-monitor/ilink"
	"client-monitor/messaging"
	"client-monitor/models"
	"client-monitor/skill"
)

// MessageHandler 消息处理器
type MessageHandler struct {
	handler      *messaging.Handler
	router       *agent.AgentRouter
	skillManager *skill.Manager
}

// WeatherFuncs 天气服务函数集合
type WeatherFuncs struct {
	GetConfig              func() (apiKey, apiHost string, ok bool)
	TestWeather            func(apiKey, apiHost, location string) (tempMax, tempMin, textDay, textNight, fxDate string, err error)
	GetWeatherDays         func(apiKey, apiHost, location string) ([]map[string]string, error)
	BuildMessage           func(location, tempMax, tempMin, textDay, textNight, fxDate string) string
	SendWeather            func(location, content string) error
	SendWeatherToChannels  func(location, content string, channelIDs []int) []error
}

// ServiceFuncs 服务函数集合（用于依赖注入）
type ServiceFuncs struct {
	Weather *WeatherFuncs

	// Reminder 服务
	ReminderCreate func(userID uint, content, timeDesc string) (string, error)
	ReminderList   func(userID uint) (string, error)
	ReminderCancel func(userID uint, keyword string) (string, error)

	// Memory 服务（兼容旧接口）
	MemorySave   func(userID uint, content string) (string, error)
	MemoryList   func(userID uint) (string, error)
	MemoryDelete func(userID uint, keyword string) (string, error)

	// Memory V2 服务
	MemoryCreate      func(userID uint, content, category string, importance int, source string) (*models.Memory, error)
	MemorySearch      func(userID uint, query string, limit int) ([]models.Memory, error)
	MemoryUpdate      func(userID uint, memoryID uint, content string) (*models.Memory, error)
	MemoryDeleteByID  func(userID uint, memoryID uint) error
	MemoryListByCat   func(userID uint, category string) ([]models.Memory, error)

	// LLM 服务
	LLMChat func(prompt string) (string, error)
}

// 全局消息处理器
var globalMessageHandler *MessageHandler

// InitMessageHandler 初始化消息处理器
func InitMessageHandler(weatherFuncs *WeatherFuncs) {
	globalMessageHandler = NewMessageHandlerWithWeather(weatherFuncs)
}

// InitMessageHandlerWithServices 初始化消息处理器（带完整服务注入）
func InitMessageHandlerWithServices(serviceFuncs *ServiceFuncs) {
	globalMessageHandler = NewMessageHandlerWithServices(serviceFuncs)
}

// GetMessageHandler 获取消息处理器
func GetMessageHandler() *MessageHandler {
	if globalMessageHandler == nil {
		globalMessageHandler = NewMessageHandler()
	}
	return globalMessageHandler
}

// NewMessageHandler 创建消息处理器
func NewMessageHandler() *MessageHandler {
	return NewMessageHandlerWithServices(nil)
}

// NewMessageHandlerWithWeather 创建消息处理器（带天气服务注入）
// Deprecated: 使用 NewMessageHandlerWithServices 替代
func NewMessageHandlerWithWeather(weatherFuncs *WeatherFuncs) *MessageHandler {
	var svcFuncs *ServiceFuncs
	if weatherFuncs != nil {
		svcFuncs = &ServiceFuncs{Weather: weatherFuncs}
	}
	return NewMessageHandlerWithServices(svcFuncs)
}

// NewMessageHandlerWithServices 创建消息处理器（带完整服务注入）
func NewMessageHandlerWithServices(serviceFuncs *ServiceFuncs) *MessageHandler {
	// 创建持久化器
	persister := NewPreferencePersister()

	// 创建路由器
	router := agent.NewAgentRouter(persister)

	// 注册适配器
	httpAgent := agent.NewHTTPAgent()
	router.Register(httpAgent)                  // HTTP API 适配器（默认）
	router.Register(agent.NewClaudeAgent())     // Claude CLI 适配器

	// 创建 Skill Manager
	skillsDir := getSkillsDir()
	skillManager := skill.NewManager(skillsDir)

	// 扫描并导入 skills
	if _, err := skillManager.ScanDirectory(skillsDir); err != nil {
		log.Printf("[wecom] 扫描 skills 失败: %v", err)
	}

	// 获取 ToolRegistry 并注入服务依赖
	toolRegistry := skill.GetToolRegistry()

	// 注入天气服务
	if serviceFuncs != nil && serviceFuncs.Weather != nil {
		toolRegistry.SetWeatherFunctions(
			serviceFuncs.Weather.GetConfig,
			serviceFuncs.Weather.TestWeather,
			serviceFuncs.Weather.GetWeatherDays,
			serviceFuncs.Weather.BuildMessage,
			serviceFuncs.Weather.SendWeather,
			serviceFuncs.Weather.SendWeatherToChannels,
		)
	}

	// 注入 Reminder 服务
	if serviceFuncs != nil && serviceFuncs.ReminderCreate != nil {
		toolRegistry.SetReminderFunctions(
			serviceFuncs.ReminderCreate,
			serviceFuncs.ReminderList,
			serviceFuncs.ReminderCancel,
		)
	}

	// 注入 Memory 服务
	if serviceFuncs != nil && serviceFuncs.MemorySave != nil {
		toolRegistry.SetMemoryFunctions(
			serviceFuncs.MemorySave,
			serviceFuncs.MemoryList,
			serviceFuncs.MemoryDelete,
		)
	}

	// 注入 Memory V2 服务
	if serviceFuncs != nil && serviceFuncs.MemoryCreate != nil {
		toolRegistry.SetMemoryFunctionsV2(
			serviceFuncs.MemoryCreate,
			serviceFuncs.MemorySearch,
			serviceFuncs.MemoryUpdate,
			serviceFuncs.MemoryDeleteByID,
			serviceFuncs.MemoryListByCat,
		)
	}

	// 注入 LLM 服务
	if serviceFuncs != nil && serviceFuncs.LLMChat != nil {
		toolRegistry.SetLLMChat(serviceFuncs.LLMChat)
	}

	// 为 HTTP Agent 设置 skill manager
	httpAgent.SetSkillManager(skillManager)

	// 创建 messaging handler
	h := messaging.NewHandler(
		func(ctx context.Context, name string) agent.Agent {
			return router.Get(name)
		},
		nil, // SaveDefaultFunc - 不需要持久化
	)

	// 设置路由器
	h.SetRouter(router)

	// 设置 Skill Manager
	h.SetSkillManager(skillManager)

	// 设置默认 agent
	h.SetDefaultAgent("api", httpAgent)

	log.Println("[wecom] Message handler initialized with router (api, claude) and skills")

	return &MessageHandler{
		handler:      h,
		router:       router,
		skillManager: skillManager,
	}
}

// getSkillsDir 获取 skills 目录路径
func getSkillsDir() string {
	// 优先使用环境变量
	if dir := os.Getenv("SKILLS_DIR"); dir != "" {
		return dir
	}

	// 默认使用 ./skills
	execPath, _ := os.Executable()
	baseDir := filepath.Dir(execPath)
	return filepath.Join(baseDir, "skills")
}

// HandleMessage 处理消息入口
func (h *MessageHandler) HandleMessage(ctx context.Context, client *ilink.Client, msg ilink.WeixinMessage) {
	h.handler.HandleMessage(ctx, client, msg)
}