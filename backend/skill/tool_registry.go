package skill

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"client-monitor/database"
	"client-monitor/models"
)

// 函数类型定义 - 用于依赖注入解耦
type (
	// GetWeatherConfigFunc 获取天气配置
	GetWeatherConfigFunc func() (apiKey, apiHost string, ok bool)
	// TestWeatherFunc 测试天气配置
	TestWeatherFunc func(apiKey, apiHost, location string) (tempMax, tempMin, textDay, textNight, fxDate string, err error)
	// GetWeatherDaysFunc 获取多天天气
	GetWeatherDaysFunc func(apiKey, apiHost, location string) ([]map[string]string, error)
	// BuildWeatherMessageFunc 构建天气消息
	BuildWeatherMessageFunc func(location, tempMax, tempMin, textDay, textNight, fxDate string) string
	// SendWeatherFunc 发送天气通知
	SendWeatherFunc func(location, content string) error

	// ReminderCreateFunc 创建提醒
	ReminderCreateFunc func(userID uint, content, timeDesc string) (string, error)
	// ReminderListFunc 查看提醒列表
	ReminderListFunc func(userID uint) (string, error)
	// ReminderCancelFunc 取消提醒
	ReminderCancelFunc func(userID uint, keyword string) (string, error)

	// MemorySaveFunc 保存记忆
	MemorySaveFunc func(userID uint, content string) (string, error)
	// MemoryListFunc 查看记忆列表
	MemoryListFunc func(userID uint) (string, error)
	// MemoryDeleteFunc 删除记忆
	MemoryDeleteFunc func(userID uint, keyword string) (string, error)

	// LLMChatFunc LLM 聊天
	LLMChatFunc func(prompt string) (string, error)
)

// ToolRegistry 管理工具白名单
type ToolRegistry struct {
	tools  map[string]*Tool
	mu     sync.RWMutex

	// 注入的服务函数
	getWeatherConfig  GetWeatherConfigFunc
	testWeather       TestWeatherFunc
	getWeatherDays    GetWeatherDaysFunc
	buildWeatherMsg   BuildWeatherMessageFunc
	sendWeather       SendWeatherFunc

	reminderCreate ReminderCreateFunc
	reminderList    ReminderListFunc
	reminderCancel  ReminderCancelFunc

	memorySave   MemorySaveFunc
	memoryList    MemoryListFunc
	memoryDelete  MemoryDeleteFunc

	llmChat LLMChatFunc
}

// NewToolRegistry 创建工具注册表
func NewToolRegistry() *ToolRegistry {
	r := &ToolRegistry{
		tools: make(map[string]*Tool),
	}
	r.registerBuiltinTools()
	return r
}

// SetWeatherFunctions 设置天气相关函数（依赖注入）
func (r *ToolRegistry) SetWeatherFunctions(
	getConfig GetWeatherConfigFunc,
	test TestWeatherFunc,
	getDays GetWeatherDaysFunc,
	buildMsg BuildWeatherMessageFunc,
	send SendWeatherFunc,
) {
	r.getWeatherConfig = getConfig
	r.testWeather = test
	r.getWeatherDays = getDays
	r.buildWeatherMsg = buildMsg
	r.sendWeather = send
	log.Printf("[tool] 天气服务函数已注入")
}

// SetReminderFunctions 设置提醒相关函数（依赖注入）
func (r *ToolRegistry) SetReminderFunctions(
	create ReminderCreateFunc,
	list ReminderListFunc,
	cancel ReminderCancelFunc,
) {
	r.reminderCreate = create
	r.reminderList = list
	r.reminderCancel = cancel
	log.Printf("[tool] 提醒服务函数已注入")
}

// SetMemoryFunctions 设置记忆相关函数（依赖注入）
func (r *ToolRegistry) SetMemoryFunctions(
	save MemorySaveFunc,
	list MemoryListFunc,
	delete MemoryDeleteFunc,
) {
	r.memorySave = save
	r.memoryList = list
	r.memoryDelete = delete
	log.Printf("[tool] 记忆服务函数已注入")
}

// SetLLMChat 设置 LLM 聊天函数（依赖注入）
func (r *ToolRegistry) SetLLMChat(chat LLMChatFunc) {
	r.llmChat = chat
	log.Printf("[tool] LLM 聊天函数已注入")
}

// registerBuiltinTools 注册内置工具
func (r *ToolRegistry) registerBuiltinTools() {
	// ========== 提醒工具 ==========
	r.Register(&Tool{
		Name:        "reminder_create",
		Description: "创建提醒，在指定时间提醒用户",
		Parameters: map[string]Parameter{
			"content": {
				Type:        "string",
				Description: "提醒内容",
				Required:    true,
			},
			"time": {
				Type:        "string",
				Description: "提醒时间 (如: 10分钟后, 明天9点, 2024-01-01 10:00)",
				Required:    true,
			},
		},
		Handler: r.handleReminderCreate,
	})

	r.Register(&Tool{
		Name:        "reminder_list",
		Description: "查看用户的所有待发送提醒",
		Parameters:  map[string]Parameter{},
		Handler:     r.handleReminderList,
	})

	r.Register(&Tool{
		Name:        "reminder_cancel",
		Description: "取消指定的提醒",
		Parameters: map[string]Parameter{
			"keyword": {
				Type:        "string",
				Description: "提醒内容关键词",
				Required:    true,
			},
		},
		Handler: r.handleReminderCancel,
	})

	// ========== 记忆工具 ==========
	r.Register(&Tool{
		Name:        "memory_save",
		Description: "保存用户记忆，记住用户告诉你的信息",
		Parameters: map[string]Parameter{
			"content": {
				Type:        "string",
				Description: "要记住的内容",
				Required:    true,
			},
		},
		Handler: r.handleMemorySave,
	})

	r.Register(&Tool{
		Name:        "memory_list",
		Description: "查看用户的所有记忆",
		Parameters:  map[string]Parameter{},
		Handler:     r.handleMemoryList,
	})

	r.Register(&Tool{
		Name:        "memory_delete",
		Description: "删除指定的记忆",
		Parameters: map[string]Parameter{
			"keyword": {
				Type:        "string",
				Description: "记忆内容关键词",
				Required:    true,
			},
		},
		Handler: r.handleMemoryDelete,
	})

	// ========== 天气工具 ==========
	r.Register(&Tool{
		Name:        "weather_get",
		Description: "获取主客户端位置的天气信息，支持查询今天、明天、后天的天气",
		Parameters: map[string]Parameter{
			"day": {
				Type:        "string",
				Description: "查询哪天的天气：今天、明天、后天。默认今天",
				Required:    false,
			},
		},
		Handler: r.handleWeatherGet,
	})

	r.Register(&Tool{
		Name:        "weather_send",
		Description: "发送天气通知到指定客户端绑定的通知渠道",
		Parameters: map[string]Parameter{
			"client_id": {
				Type:        "string",
				Description: "客户端ID，不传则发送给所有启用天气的客户端",
				Required:    false,
			},
		},
		Handler: r.handleWeatherSend,
	})

	// ========== 翻译工具 ==========
	r.Register(&Tool{
		Name:        "translator",
		Description: "多语言翻译",
		Parameters: map[string]Parameter{
			"text": {
				Type:        "string",
				Description: "要翻译的文本",
				Required:    true,
			},
			"target_lang": {
				Type:        "string",
				Description: "目标语言 (如: 英文, 中文, 日文, 韩文)",
				Required:    true,
			},
		},
		Handler: r.handleTranslator,
	})

	log.Printf("[tool] 已注册 %d 个内置工具", len(r.tools))
}

// Register 注册工具
func (r *ToolRegistry) Register(tool *Tool) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.tools[tool.Name]; exists {
		return fmt.Errorf("工具 %s 已存在", tool.Name)
	}

	r.tools[tool.Name] = tool
	return nil
}

// Get 获取工具
func (r *ToolRegistry) Get(name string) *Tool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.tools[name]
}

// List 列出所有工具
func (r *ToolRegistry) List() []*Tool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var tools []*Tool
	for _, tool := range r.tools {
		tools = append(tools, tool)
	}
	return tools
}

// GetAllowed 返回 skill 允许的工具
func (r *ToolRegistry) GetAllowed(skill *Skill) []*Tool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var allowed []*Tool
	for _, toolName := range skill.AllowedTools {
		if tool, exists := r.tools[toolName]; exists {
			allowed = append(allowed, tool)
		}
	}
	return allowed
}

// IsAllowed 检查 skill 是否可以使用工具
func (r *ToolRegistry) IsAllowed(skill *Skill, toolName string) bool {
	for _, allowed := range skill.AllowedTools {
		if allowed == toolName {
			return true
		}
	}
	return false
}

// Execute 执行工具调用并检查权限
func (r *ToolRegistry) Execute(skill *Skill, toolCall *ToolCall) (*ToolResult, error) {
	tool := r.Get(toolCall.Name)
	if tool == nil {
		return nil, fmt.Errorf("工具 %s 不存在", toolCall.Name)
	}

	if !r.IsAllowed(skill, toolCall.Name) {
		return nil, fmt.Errorf("skill %s 不被允许使用工具 %s", skill.ID, toolCall.Name)
	}

	result, err := tool.Handler(toolCall.Arguments["user_id"].(string), toolCall.Arguments)
	if err != nil {
		return &ToolResult{
			ToolCallID: toolCall.ID,
			Error:      err.Error(),
		}, err
	}

	return &ToolResult{
		ToolCallID: toolCall.ID,
		Result:     result,
	}, nil
}

// ExecuteBatch 批量执行工具调用
func (r *ToolRegistry) ExecuteBatch(skill *Skill, calls []*ToolCall) []*ToolResult {
	var results []*ToolResult
	for _, call := range calls {
		result, _ := r.Execute(skill, call)
		results = append(results, result)
	}
	return results
}

// ========== 提醒工具处理器 ==========

func (r *ToolRegistry) handleReminderCreate(userID string, params map[string]interface{}) (interface{}, error) {
	if r.reminderCreate == nil {
		return nil, fmt.Errorf("提醒服务未配置")
	}

	dbUserID, err := r.parseUserID(userID)
	if err != nil {
		return nil, err
	}

	content, _ := params["content"].(string)
	timeDesc, _ := params["time"].(string)

	result, err := r.reminderCreate(dbUserID, content, timeDesc)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"success": true,
		"message": result,
	}, nil
}

func (r *ToolRegistry) handleReminderList(userID string, params map[string]interface{}) (interface{}, error) {
	if r.reminderList == nil {
		return nil, fmt.Errorf("提醒服务未配置")
	}

	// userID 已经是数据库 ID（从 chat_handler 传入）
	dbUserID, err := r.parseUserID(userID)
	if err != nil {
		return nil, err
	}

	result, err := r.reminderList(dbUserID)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"success": true,
		"message": result,
	}, nil
}

func (r *ToolRegistry) handleReminderCancel(userID string, params map[string]interface{}) (interface{}, error) {
	if r.reminderCancel == nil {
		return nil, fmt.Errorf("提醒服务未配置")
	}

	dbUserID, err := r.parseUserID(userID)
	if err != nil {
		return nil, err
	}

	keyword, _ := params["keyword"].(string)

	result, err := r.reminderCancel(dbUserID, keyword)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"success": true,
		"message": result,
	}, nil
}

// ========== 记忆工具处理器 ==========

func (r *ToolRegistry) handleMemorySave(userID string, params map[string]interface{}) (interface{}, error) {
	if r.memorySave == nil {
		return nil, fmt.Errorf("记忆服务未配置")
	}

	dbUserID, err := r.parseUserID(userID)
	if err != nil {
		return nil, err
	}

	content, _ := params["content"].(string)

	result, err := r.memorySave(dbUserID, content)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"success": true,
		"message": result,
	}, nil
}

func (r *ToolRegistry) handleMemoryList(userID string, params map[string]interface{}) (interface{}, error) {
	if r.memoryList == nil {
		return nil, fmt.Errorf("记忆服务未配置")
	}

	dbUserID, err := r.parseUserID(userID)
	if err != nil {
		return nil, err
	}

	result, err := r.memoryList(dbUserID)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"success": true,
		"message": result,
	}, nil
}

func (r *ToolRegistry) handleMemoryDelete(userID string, params map[string]interface{}) (interface{}, error) {
	if r.memoryDelete == nil {
		return nil, fmt.Errorf("记忆服务未配置")
	}

	dbUserID, err := r.parseUserID(userID)
	if err != nil {
		return nil, err
	}

	keyword, _ := params["keyword"].(string)

	result, err := r.memoryDelete(dbUserID, keyword)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"success": true,
		"message": result,
	}, nil
}

// ========== 天气工具处理器 ==========

// isClientOnline 检查客户端是否在线（1分钟内有数据上报）
func (r *ToolRegistry) isClientOnline(clientID string) bool {
	var event models.Event
	oneMinuteAgo := time.Now().Add(-1 * time.Minute)
	err := database.DB.Where("client_id = ? AND created_at > ?", clientID, oneMinuteAgo).
		First(&event).Error
	return err == nil
}

// isClientActive 检查客户端是否活跃（24小时内有数据）
func (r *ToolRegistry) isClientActive(clientID string) bool {
	var event models.Event
	oneDayAgo := time.Now().Add(-24 * time.Hour)
	err := database.DB.Where("client_id = ? AND created_at > ?", clientID, oneDayAgo).
		First(&event).Error
	return err == nil
}

// getClientLocation 获取客户端的位置
func (r *ToolRegistry) getClientLocation(clientID string) (string, error) {
	var event models.Event
	if err := database.DB.Where("client_id = ?", clientID).
		Order("created_at desc").
		First(&event).Error; err != nil {
		return "", err
	}

	var data map[string]interface{}
	if err := json.Unmarshal(event.Data, &data); err != nil {
		return "", err
	}

	if loc, ok := data["location"].(map[string]interface{}); ok {
		if city, ok := loc["city"].(string); ok && city != "" {
			return city, nil
		}
	}
	return "", fmt.Errorf("客户端没有位置信息")
}

// GetPrimaryClientLocation 获取主客户端的位置（支持主从降级）
func (r *ToolRegistry) getPrimaryClientLocation() (string, error) {
	// 先查找主客户端
	var primaryClient models.ClientOrder
	if err := database.DB.Where("is_primary = ?", true).First(&primaryClient).Error; err == nil {
		// 检查是否活跃（24小时内有数据）
		if r.isClientActive(primaryClient.ClientID) {
			location, err := r.getClientLocation(primaryClient.ClientID)
			if err == nil {
				return location, nil
			}
		}
	}

	// 主设备不活跃或没有，降级到其他客户端
	var clients []models.ClientOrder
	database.DB.Order("sort_order, client_id").Find(&clients)

	for _, client := range clients {
		if client.IsPrimary {
			continue // 已经检查过了
		}
		// 检查是否在线（1分钟内）
		if !r.isClientOnline(client.ClientID) {
			continue
		}
		location, err := r.getClientLocation(client.ClientID)
		if err == nil {
			return location, nil
		}
	}

	// 没有在线且有位置的客户端，尝试从最近的事件中获取位置
	var events []models.Event
	database.DB.Where("data->>'location' IS NOT NULL AND data->>'location' != ''").
		Order("created_at desc").
		Limit(10).
		Find(&events)

	for _, event := range events {
		var data map[string]interface{}
		if err := json.Unmarshal(event.Data, &data); err == nil {
			if loc, ok := data["location"].(map[string]interface{}); ok {
				if city, ok := loc["city"].(string); ok && city != "" {
					return city, nil
				}
			}
		}
	}

	return "", fmt.Errorf("未找到有位置信息的客户端")
}

func (r *ToolRegistry) handleWeatherGet(userID string, params map[string]interface{}) (interface{}, error) {
	if r.getWeatherConfig == nil || r.getWeatherDays == nil {
		return nil, fmt.Errorf("天气服务未配置")
	}

	// 获取主客户端位置
	location, err := r.getPrimaryClientLocation()
	if err != nil {
		return nil, err
	}

	apiKey, apiHost, ok := r.getWeatherConfig()
	if !ok {
		return nil, fmt.Errorf("天气服务未配置")
	}

	// 获取多天天气
	days, err := r.getWeatherDays(apiKey, apiHost, location)
	if err != nil {
		return nil, err
	}

	// 解析 day 参数
	dayParam, _ := params["day"].(string)
	dayIndex := 0 // 默认今天
	if dayParam == "明天" || dayParam == "tomorrow" {
		dayIndex = 1
	} else if dayParam == "后天" {
		dayIndex = 2
	}

	// 如果请求的天数超出范围
	if dayIndex >= len(days) {
		return nil, fmt.Errorf("没有%s的天气数据", dayParam)
	}

	// 返回指定天的天气
	day := days[dayIndex]
	dayLabel := "今天"
	if dayIndex == 1 {
		dayLabel = "明天"
	} else if dayIndex == 2 {
		dayLabel = "后天"
	}

	return map[string]interface{}{
		"success":    true,
		"location":   location,
		"day_label":  dayLabel,
		"date":       day["date"],
		"temp_max":   day["temp_max"],
		"temp_min":   day["temp_min"],
		"text_day":   day["text_day"],
		"text_night": day["text_night"],
		"message":    fmt.Sprintf("%s %s %s: %s°C~%s°C, 白天%s, 夜间%s", location, dayLabel, day["date"], day["temp_min"], day["temp_max"], day["text_day"], day["text_night"]),
		"all_days":   days, // 也返回所有天的数据
	}, nil
}

func (r *ToolRegistry) handleWeatherSend(userID string, params map[string]interface{}) (interface{}, error) {
	if r.getWeatherConfig == nil || r.testWeather == nil || r.buildWeatherMsg == nil || r.sendWeather == nil {
		return nil, fmt.Errorf("天气服务未配置")
	}

	clientID, _ := params["client_id"].(string)

	// 获取要推送的客户端列表
	var clients []models.ClientOrder
	if clientID != "" {
		// 指定客户端
		var client models.ClientOrder
		if err := database.DB.Where("client_id = ?", clientID).First(&client).Error; err != nil {
			return nil, fmt.Errorf("客户端不存在: %s", clientID)
		}
		clients = append(clients, client)
	} else {
		// 所有启用天气的客户端
		database.DB.Where("weather_enabled = ?", true).Find(&clients)
	}

	if len(clients) == 0 {
		return nil, fmt.Errorf("没有启用天气推送的客户端")
	}

	apiKey, apiHost, ok := r.getWeatherConfig()
	if !ok {
		return nil, fmt.Errorf("天气服务未配置")
	}

	var results []string
	for _, client := range clients {
		// 获取客户端位置
		var event models.Event
		if err := database.DB.Where("client_id = ?", client.ClientID).
			Order("created_at desc").
			First(&event).Error; err != nil {
			results = append(results, fmt.Sprintf("%s: 无位置信息", client.ClientID))
			continue
		}

		var location string
		var data map[string]interface{}
		if err := json.Unmarshal(event.Data, &data); err == nil {
			if loc, ok := data["location"].(map[string]interface{}); ok {
				if city, ok := loc["city"].(string); ok {
					location = city
				}
			}
		}

		if location == "" {
			results = append(results, fmt.Sprintf("%s: 无位置信息", client.ClientID))
			continue
		}

		tempMax, tempMin, textDay, textNight, fxDate, err := r.testWeather(apiKey, apiHost, location)
		if err != nil {
			results = append(results, fmt.Sprintf("%s: 天气查询失败 - %v", client.ClientID, err))
			continue
		}

		content := r.buildWeatherMsg(location, tempMax, tempMin, textDay, textNight, fxDate)

		if err := r.sendWeather(location, content); err != nil {
			results = append(results, fmt.Sprintf("%s: 发送失败 - %v", client.ClientID, err))
		} else {
			results = append(results, fmt.Sprintf("%s: 已发送 %s 天气", client.ClientID, location))
		}
	}

	return map[string]interface{}{
		"success": true,
		"message": strings.Join(results, "\n"),
	}, nil
}

// ========== 翻译工具处理器 ==========

func (r *ToolRegistry) handleTranslator(userID string, params map[string]interface{}) (interface{}, error) {
	if r.llmChat == nil {
		return nil, fmt.Errorf("LLM 服务未配置")
	}

	text, _ := params["text"].(string)
	targetLang, _ := params["target_lang"].(string)

	prompt := fmt.Sprintf("请将以下文本翻译成%s，只返回翻译结果，不要解释：\n\n%s", targetLang, text)

	result, err := r.llmChat(prompt)
	if err != nil {
		return nil, fmt.Errorf("翻译失败: %v", err)
	}

	return map[string]interface{}{
		"success":     true,
		"original":    text,
		"translated":  result,
		"target_lang": targetLang,
		"message":     result,
	}, nil
}

// ========== 辅助函数 ==========

// parseUserID 解析用户 ID（支持数据库 ID 或 wecom_user_id）
func (r *ToolRegistry) parseUserID(userID string) (uint, error) {
	// 尝试解析为数字（数据库 ID）
	var dbID uint
	if _, err := fmt.Sscanf(userID, "%d", &dbID); err == nil && dbID > 0 {
		return dbID, nil
	}

	// 否则作为 wecom_user_id 查询
	return r.getUserDBID(userID)
}

// getUserDBID 根据 wecom_user_id 获取数据库 user ID
func (r *ToolRegistry) getUserDBID(wecomUserID string) (uint, error) {
	var user models.AIUser
	if err := database.DB.Where("wecom_user_id = ?", wecomUserID).First(&user).Error; err != nil {
		user = models.AIUser{
			WecomUserID: wecomUserID,
			Name:        wecomUserID,
		}
		if err := database.DB.Create(&user).Error; err != nil {
			return 0, fmt.Errorf("创建用户失败: %w", err)
		}
	}
	return user.ID, nil
}

// GetToolsForLLM 返回 LLM 可用的工具定义
func (r *ToolRegistry) GetToolsForLLM(skill *Skill) []map[string]interface{} {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var tools []map[string]interface{}
	for _, toolName := range skill.AllowedTools {
		if tool, exists := r.tools[toolName]; exists {
			tools = append(tools, map[string]interface{}{
				"type": "function",
				"function": map[string]interface{}{
					"name":        tool.Name,
					"description": tool.Description,
					"parameters": map[string]interface{}{
						"type":       "object",
						"properties": r.buildParametersSchema(tool.Parameters),
						"required":   r.getRequiredParameters(tool.Parameters),
					},
				},
			})
		}
	}
	return tools
}

func (r *ToolRegistry) buildParametersSchema(params map[string]Parameter) map[string]interface{} {
	schema := make(map[string]interface{})
	for name, param := range params {
		schema[name] = map[string]interface{}{
			"type":        param.Type,
			"description": param.Description,
		}
	}
	return schema
}

func (r *ToolRegistry) getRequiredParameters(params map[string]Parameter) []string {
	var required []string
	for name, param := range params {
		if param.Required {
			required = append(required, name)
		}
	}
	return required
}

// 全局 ToolRegistry 实例
var globalToolRegistry *ToolRegistry

// GetToolRegistry 获取全局 ToolRegistry
func GetToolRegistry() *ToolRegistry {
	if globalToolRegistry == nil {
		globalToolRegistry = NewToolRegistry()
	}
	return globalToolRegistry
}