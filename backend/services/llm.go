package services

import (
	"bufio"
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
			client: &http.Client{Timeout: 120 * time.Second},
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

// GetConfig 获取 LLM 配置（优先使用缓存）
func (s *LLMService) GetConfig() *models.LLMConfig {
	cache := GetCacheService()

	// 先尝试从缓存获取
	if config := cache.GetLLMConfig(); config != nil {
		return config
	}

	// 缓存未命中，从数据库查询
	var config models.LLMConfig
	if err := database.DB.Where("enabled = ?", true).First(&config).Error; err != nil {
		return nil
	}

	// 写入缓存
	cache.SetLLMConfig(&config)
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

// StreamChatRequest 流式请求结构
type StreamChatRequest struct {
	Model    string        `json:"model"`
	Messages []ChatMessage `json:"messages"`
	Stream   bool          `json:"stream"`
}

// StreamChatResponse 流式响应结构
type StreamChatResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index int `json:"index"`
		Delta struct {
			Role    string `json:"role,omitempty"`
			Content string `json:"content,omitempty"`
		} `json:"delta"`
		FinishReason *string `json:"finish_reason"`
	} `json:"choices"`
}

// ChatStream 流式调用 LLM，通过 callback 返回每个 chunk
func (s *LLMService) ChatStream(messages []ChatMessage, onChunk func(content string) error) error {
	config := s.GetConfig()
	if config == nil {
		return fmt.Errorf("LLM 配置未设置")
	}

	return s.CallAPIStream(config, messages, onChunk)
}

// CallAPIStream 流式调用 API
func (s *LLMService) CallAPIStream(config *models.LLMConfig, messages []ChatMessage, onChunk func(content string) error) error {
	var baseURL string
	var model string

	// 根据提供商设置默认值（与 CallAPI 相同逻辑）
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
		if config.BaseURL == "" {
			return fmt.Errorf("自定义提供商需要设置 BaseURL")
		}
		baseURL = config.BaseURL
		model = config.Model
	}

	// 构建流式请求
	reqBody := StreamChatRequest{
		Model:    model,
		Messages: messages,
		Stream:   true,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("序列化请求失败: %w", err)
	}

	url := fmt.Sprintf("%s/chat/completions", baseURL)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("创建请求失败: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", config.APIKey))
	req.Header.Set("Accept", "text/event-stream")

	// 发送请求
	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("发送请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API 返回错误: %s", string(body))
	}

	// 读取 SSE 流
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()

		// 跳过空行
		if line == "" {
			continue
		}

		// SSE 数据行以 "data: " 开头
		if len(line) > 6 && line[:6] == "data: " {
			data := line[6:]

			// 结束标记
			if data == "[DONE]" {
				break
			}

			var streamResp StreamChatResponse
			if err := json.Unmarshal([]byte(data), &streamResp); err != nil {
				log.Printf("解析流式响应失败: %v", err)
				continue
			}

			// 提取内容
			if len(streamResp.Choices) > 0 {
				content := streamResp.Choices[0].Delta.Content
				if content != "" {
					if err := onChunk(content); err != nil {
						return err
					}
				}
			}
		}
	}

	return scanner.Err()
}

// ChatWithSystemStream 带系统提示的流式对话
func (s *LLMService) ChatWithSystemStream(systemPrompt string, messages []ChatMessage, onChunk func(content string) error) error {
	allMessages := make([]ChatMessage, 0, len(messages)+1)
	allMessages = append(allMessages, ChatMessage{
		Role:    "system",
		Content: systemPrompt,
	})
	allMessages = append(allMessages, messages...)

	return s.ChatStream(allMessages, onChunk)
}
