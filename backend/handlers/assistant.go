package handlers

import (
	"net/http"
	"strconv"

	"client-monitor/database"
	"client-monitor/models"
	"client-monitor/services"

	"github.com/gin-gonic/gin"
)

// TestLLMRequest LLM 测试请求
type TestLLMRequest struct {
	Message string `json:"message" binding:"required"`
}

// TestLLM POST /api/assistant/llm/test - 测试 LLM
func TestLLM(c *gin.Context) {
	var req TestLLMRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 调用 LLM
	llm := services.GetLLMService()
	response, err := llm.Chat([]services.ChatMessage{
		{Role: "user", Content: req.Message},
	})
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":  true,
		"response": response,
	})
}

// GetLLMConfig GET /api/assistant/llm - 获取 LLM 配置
func GetLLMConfig(c *gin.Context) {
	var config models.LLMConfig
	result := database.DB.First(&config)

	if result.Error != nil {
		c.JSON(http.StatusOK, models.LLMConfig{
			Provider: "qwen",
			Model:    "qwen-plus",
			Enabled:  false,
		})
		return
	}

	// 隐藏 API Key
	if config.APIKey != "" {
		config.APIKey = config.APIKey[:8] + "..." + config.APIKey[len(config.APIKey)-4:]
	}

	c.JSON(http.StatusOK, config)
}

// UpdateLLMConfig PUT /api/assistant/llm - 更新 LLM 配置
func UpdateLLMConfig(c *gin.Context) {
	var req models.LLMConfig
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var config models.LLMConfig
	result := database.DB.First(&config)

	if result.Error != nil {
		// 创建新配置
		if err := database.DB.Create(&req).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "创建配置失败"})
			return
		}
		// 清除缓存
		services.GetCacheService().ClearLLMConfig()
		c.JSON(http.StatusOK, req)
	} else {
		// 更新配置
		updates := make(map[string]interface{})
		if req.Provider != "" {
			updates["provider"] = req.Provider
		}
		if req.Model != "" {
			updates["model"] = req.Model
		}
		if req.APIKey != "" && !isMaskedAPIKey(req.APIKey) {
			updates["api_key"] = req.APIKey
		}
		if req.BaseURL != "" {
			updates["base_url"] = req.BaseURL
		}
		updates["enabled"] = req.Enabled

		database.DB.Model(&config).Updates(updates)
		// 清除缓存
		services.GetCacheService().ClearLLMConfig()
		c.JSON(http.StatusOK, config)
	}
}

// isMaskedAPIKey 检查是否是隐藏的 API Key
func isMaskedAPIKey(key string) bool {
	return len(key) > 12 && key[8:11] == "..."
}

// GetTodos GET /api/assistant/todos - 获取 Todo 列表
func GetTodos(c *gin.Context) {
	userIDStr := c.Query("user_id")
	if userIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user_id is required"})
		return
	}

	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user_id"})
		return
	}

	completed := c.Query("completed")
	var todos []models.Todo
	query := database.DB.Where("user_id = ?", userID)
	if completed != "" {
		if completed == "true" {
			query = query.Where("completed = ?", true)
		} else {
			query = query.Where("completed = ?", false)
		}
	}
	query.Order("created_at desc").Find(&todos)

	c.JSON(http.StatusOK, todos)
}

// CreateTodo POST /api/assistant/todos - 创建 Todo
func CreateTodo(c *gin.Context) {
	var req models.Todo
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := database.DB.Create(&req).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建失败"})
		return
	}

	c.JSON(http.StatusOK, req)
}

// UpdateTodo PUT /api/assistant/todos/:id - 更新 Todo
func UpdateTodo(c *gin.Context) {
	id := c.Param("id")

	var req models.Todo
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var todo models.Todo
	if err := database.DB.First(&todo, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Todo 不存在"})
		return
	}

	updates := make(map[string]interface{})
	if req.Content != "" {
		updates["content"] = req.Content
	}
	if req.Deadline != nil {
		updates["deadline"] = req.Deadline
	}
	updates["completed"] = req.Completed

	database.DB.Model(&todo).Updates(updates)
	c.JSON(http.StatusOK, todo)
}

// DeleteTodo DELETE /api/assistant/todos/:id - 删除 Todo
func DeleteTodo(c *gin.Context) {
	id := c.Param("id")

	if err := database.DB.Delete(&models.Todo{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// GetMemories GET /api/assistant/memories - 获取记忆列表
func GetMemories(c *gin.Context) {
	userIDStr := c.Query("user_id")
	if userIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user_id is required"})
		return
	}

	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user_id"})
		return
	}

	var memories []models.Memory
	database.DB.Where("user_id = ?", userID).Order("created_at desc").Find(&memories)

	c.JSON(http.StatusOK, memories)
}

// DeleteMemory DELETE /api/assistant/memories/:id - 删除记忆
func DeleteMemory(c *gin.Context) {
	id := c.Param("id")

	if err := database.DB.Delete(&models.Memory{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// GetReminders GET /api/assistant/reminders - 获取提醒列表
func GetReminders(c *gin.Context) {
	userIDStr := c.Query("user_id")
	if userIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user_id is required"})
		return
	}

	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user_id"})
		return
	}

	var reminders []models.Reminder
	database.DB.Where("user_id = ?", userID).Order("remind_at asc").Find(&reminders)

	c.JSON(http.StatusOK, reminders)
}

// GetAIUsers GET /api/assistant/users - 获取 AI 用户列表
func GetAIUsers(c *gin.Context) {
	var users []models.AIUser
	database.DB.Order("created_at desc").Find(&users)

	c.JSON(http.StatusOK, users)
}

// GetConversations GET /api/assistant/conversations - 获取对话记录
func GetConversations(c *gin.Context) {
	userIDStr := c.Query("user_id")
	if userIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user_id is required"})
		return
	}

	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user_id"})
		return
	}

	var conversations []models.Conversation
	database.DB.Where("user_id = ?", userID).Order("created_at desc").Limit(50).Find(&conversations)

	c.JSON(http.StatusOK, conversations)
}
