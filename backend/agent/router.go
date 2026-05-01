package agent

import (
	"context"
	"fmt"
	"log"
	"sync"
)

// AgentRouter 路由不同用户的消息到不同的 AI 后端
type AgentRouter struct {
	adapters    map[string]Agent
	userSession sync.Map // userID -> adapterName
	defaultName string
	persister   UserPreferencePersister
}

// UserPreferencePersister 用户偏好持久化接口
type UserPreferencePersister interface {
	Get(userID string) (adapterName string, err error)
	Set(userID, adapterName string) error
}

// NewAgentRouter 创建路由器
func NewAgentRouter(persister UserPreferencePersister) *AgentRouter {
	return &AgentRouter{
		adapters:    make(map[string]Agent),
		defaultName: "api",
		persister:   persister,
	}
}

// Register 注册适配器
func (r *AgentRouter) Register(adapter Agent) {
	r.adapters[adapter.Info().Name] = adapter
	log.Printf("[router] 注册适配器: %s", adapter.Info().Name)
}

// Switch 切换用户的 AI 后端
func (r *AgentRouter) Switch(userID, adapterName string) error {
	if _, ok := r.adapters[adapterName]; !ok {
		return fmt.Errorf("未知的适配器: %s", adapterName)
	}

	// 1. 内存缓存
	r.userSession.Store(userID, adapterName)

	// 2. 持久化
	if r.persister != nil {
		if err := r.persister.Set(userID, adapterName); err != nil {
			log.Printf("[router] 持久化失败: %v", err)
		}
	}

	log.Printf("[router] 用户 %s 切换到 %s", userID, adapterName)
	return nil
}

// Get 获取用户当前的适配器
func (r *AgentRouter) Get(userID string) Agent {
	// 1. 先查内存缓存
	if name, ok := r.userSession.Load(userID); ok {
		if adapter, ok := r.adapters[name.(string)]; ok {
			return adapter
		}
	}

	// 2. 查持久化存储
	if r.persister != nil {
		if name, err := r.persister.Get(userID); err == nil && name != "" {
			if adapter, ok := r.adapters[name]; ok {
				// 回写内存缓存
				r.userSession.Store(userID, name)
				return adapter
			}
		}
	}

	// 3. 返回默认
	return r.adapters[r.defaultName]
}

// Chat 使用用户当前的适配器处理消息
func (r *AgentRouter) Chat(ctx context.Context, userID, message string) (string, error) {
	adapter := r.Get(userID)
	if adapter == nil {
		return "", fmt.Errorf("没有可用的适配器")
	}
	return adapter.Chat(ctx, userID, message)
}

// ResetSession 重置用户会话
func (r *AgentRouter) ResetSession(ctx context.Context, userID string) (string, error) {
	adapter := r.Get(userID)
	if adapter == nil {
		return "", fmt.Errorf("没有可用的适配器")
	}
	return adapter.ResetSession(ctx, userID)
}

// ListAdapters 列出所有适配器
func (r *AgentRouter) ListAdapters() []AgentInfo {
	var list []AgentInfo
	for _, adapter := range r.adapters {
		list = append(list, adapter.Info())
	}
	return list
}

// GetCurrentAdapterName 获取用户当前的适配器名称
func (r *AgentRouter) GetCurrentAdapterName(userID string) string {
	adapter := r.Get(userID)
	if adapter != nil {
		return adapter.Info().Name
	}
	return r.defaultName
}
