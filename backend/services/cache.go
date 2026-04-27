package services

import (
	"sync"

	"client-monitor/models"
)

// CacheService 缓存服务
type CacheService struct {
	users     sync.Map // 用户缓存: wecomUserID -> *AIUser
	llmConfig *models.LLMConfig
	llmMux    sync.RWMutex
}

var cacheService *CacheService
var cacheOnce sync.Once

// GetCacheService 获取缓存服务单例
func GetCacheService() *CacheService {
	cacheOnce.Do(func() {
		cacheService = &CacheService{}
	})
	return cacheService
}

// GetUser 获取缓存的用户信息
func (c *CacheService) GetUser(wecomUserID string) *models.AIUser {
	if v, ok := c.users.Load(wecomUserID); ok {
		return v.(*models.AIUser)
	}
	return nil
}

// SetUser 设置用户缓存
func (c *CacheService) SetUser(user *models.AIUser) {
	c.users.Store(user.WecomUserID, user)
}

// DeleteUser 删除用户缓存
func (c *CacheService) DeleteUser(wecomUserID string) {
	c.users.Delete(wecomUserID)
}

// GetLLMConfig 获取缓存的 LLM 配置
func (c *CacheService) GetLLMConfig() *models.LLMConfig {
	c.llmMux.RLock()
	defer c.llmMux.RUnlock()
	return c.llmConfig
}

// SetLLMConfig 设置 LLM 配置缓存
func (c *CacheService) SetLLMConfig(config *models.LLMConfig) {
	c.llmMux.Lock()
	defer c.llmMux.Unlock()
	c.llmConfig = config
}

// ClearLLMConfig 清除 LLM 配置缓存
func (c *CacheService) ClearLLMConfig() {
	c.llmMux.Lock()
	defer c.llmMux.Unlock()
	c.llmConfig = nil
}
