package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"client-monitor/database"
	"client-monitor/models"
)

// LLMService LLM 服务
type LLMService struct {
	client *http.Client
}

var (
	llmService *LLMService
)

// GetLLMService 获取 LLM 服务单例
func GetLLMService() *LLMService {
	if llmService == nil {
		llmService = &LLMService{
			client: &http.Client{Timeout: 60 * time.Second},
		}
	}
	return llmService
}

// ChatMessage 聊天消息
type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatRequest 聊天请求
type ChatRequest struct {
	Model    string        `json:"model"`
	Messages []ChatMessage `json:"messages"`
}

// ChatResponse 聊天响应
type ChatResponse struct {
	Choices []struct {
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
	Error struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    string `json:"code"`
	} `json:"error"`
}

// GetConfig 获取 LLM 配置
func (s *LLMService) GetConfig() *models.LLMConfig {
	var config models.LLMConfig
	if err := database.DB.Where("enabled = ?", true).First(&config).Error; err != nil {
		return nil
	}
	return &config
}

// Chat 调用 LLM 进行对话
func (s *LLMService) Chat(messages []ChatMessage) (string, error) {
	config := s.GetConfig()
	if config == nil {
		return "", fmt.Errorf("LLM 配置未设置")
	}

	return s.CallAPI(config, messages)
}

// CallAPI 调用 API
func (s *LLMService) CallAPI(config *models.LLMConfig, messages []ChatMessage) (string, error) {
	var baseURL string
	var model string

	// 根据提供商设置默认值
	switch config.Provider {
	case "qwen":
		baseURL = "https://dashscope.aliyuncs.com/compatible-mode/v1"
		if config.BaseURL != "" {
			baseURL = config.BaseURL
		}
		model = config.Model
		if model == "" {
			model = "qwen-plus"
		}
	case "openai":
		baseURL = "https://api.openai.com/v1"
		if config.BaseURL != "" {
			baseURL = config.BaseURL
		}
		model = config.Model
		if model == "" {
			model = "gpt-4o-mini"
		}
	default:
		// 自定义提供商
		if config.BaseURL == "" {
			return "", fmt.Errorf("自定义提供商需要设置 BaseURL")
		}
		baseURL = config.BaseURL
		model = config.Model
	}

	// 构建请求
	reqBody := ChatRequest{
		Model:    model,
		Messages: messages,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("序列化请求失败: %w", err)
	}

	url := fmt.Sprintf("%s/chat/completions", baseURL)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("创建请求失败: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", config.APIKey))

	// 发送请求
	resp, err := s.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("发送请求失败: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("读取响应失败: %w", err)
	}

	var chatResp ChatResponse
	if err := json.Unmarshal(body, &chatResp); err != nil {
		return "", fmt.Errorf("解析响应失败: %w, body: %s", err, string(body))
	}

	// 检查错误
	if chatResp.Error.Message != "" {
		return "", fmt.Errorf("API 错误: %s", chatResp.Error.Message)
	}

	// 检查响应
	if len(chatResp.Choices) == 0 {
		return "", fmt.Errorf("API 返回空响应")
	}

	log.Printf("LLM 调用成功, tokens: prompt=%d, completion=%d, total=%d",
		chatResp.Usage.PromptTokens,
		chatResp.Usage.CompletionTokens,
		chatResp.Usage.TotalTokens,
	)

	return chatResp.Choices[0].Message.Content, nil
}

// ChatWithSystem 带系统提示的对话
func (s *LLMService) ChatWithSystem(systemPrompt string, messages []ChatMessage) (string, error) {
	// 添加系统消息到开头
	allMessages := make([]ChatMessage, 0, len(messages)+1)
	allMessages = append(allMessages, ChatMessage{
		Role:    "system",
		Content: systemPrompt,
	})
	allMessages = append(allMessages, messages...)

	return s.Chat(allMessages)
}
