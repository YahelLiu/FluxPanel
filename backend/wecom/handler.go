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
}

// NewMessageHandler 创建消息处理器
func NewMessageHandler() *MessageHandler {
	fluxPanelAgent := NewFluxPanelAgent()

	// 创建 weclaw messaging handler
	h := messaging.NewHandler(
		func(ctx context.Context, name string) agent.Agent {
			// 只支持 fluxpanel agent
			if name == "fluxpanel" {
				return fluxPanelAgent
			}
			return nil
		},
		nil, // SaveDefaultFunc - 不需要持久化
	)

	// 设置默认 agent
	h.SetDefaultAgent("fluxpanel", fluxPanelAgent)

	log.Println("[wecom] Message handler initialized with fluxpanel agent")

	return &MessageHandler{handler: h}
}

// HandleMessage 处理消息入口
func (h *MessageHandler) HandleMessage(ctx context.Context, client *ilink.Client, msg ilink.WeixinMessage) {
	h.handler.HandleMessage(ctx, client, msg)
}
