package drivers

import "client-monitor/notify/types"

// Driver 通知渠道驱动接口
type Driver interface {
	// Name 返回驱动名称
	Name() string

	// Send 发送消息
	Send(msg *types.NotifyMessage) error

	// IsAvailable 检查驱动是否可用（如是否已登录、配置是否正确）
	IsAvailable() bool

	// SupportedTypes 返回支持的消息类型（空表示支持所有类型）
	SupportedTypes() []types.MessageType
}

// DriverInfo 驱动信息
type DriverInfo struct {
	Name        string
	Description string
	Available   bool
	Types       []types.MessageType
}
