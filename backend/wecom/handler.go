package wecom

import (
	"context"
	"log"

	"client-monitor/agent"
	"client-monitor/ilink"
	"client-monitor/messaging"
)

// MessageHandler 消息处理器
type MessageHandler struct {
	handler *messaging.Handler
	router  *agent.AgentRouter
}

// NewMessageHandler 创建消息处理器
func NewMessageHandler() *MessageHandler {
	// 创建持久化器
	persister := NewPreferencePersister()

	// 创建路由器
	router := agent.NewAgentRouter(persister)

	// 注册适配器
	router.Register(agent.NewHTTPAgent())   // HTTP API 适配器（默认）
	router.Register(agent.NewClaudeAgent()) // Claude CLI 适配器

	// 创建 messaging handler
	h := messaging.NewHandler(
		func(ctx context.Context, name string) agent.Agent {
			return router.Get(name)
		},
		nil, // SaveDefaultFunc - 不需要持久化
	)

	// 设置路由器
	h.SetRouter(router)

	// 兼容旧代码：设置默认 agent
	httpAgent := agent.NewHTTPAgent()
	h.SetDefaultAgent("api", httpAgent)

	log.Println("[wecom] Message handler initialized with router (api, claude)")

	return &MessageHandler{
		handler: h,
		router:  router,
	}
}

// HandleMessage 处理消息入口
func (h *MessageHandler) HandleMessage(ctx context.Context, client *ilink.Client, msg ilink.WeixinMessage) {
	h.handler.HandleMessage(ctx, client, msg)
}
